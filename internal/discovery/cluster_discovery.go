package discovery

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/jontk/s9s/internal/config"
	"github.com/jontk/s9s/internal/debug"
)

// ClusterDiscovery handles automatic SLURM cluster discovery
type ClusterDiscovery struct {
	timeout time.Duration
	client  *http.Client
}

// DiscoveredCluster represents a discovered SLURM cluster
type DiscoveredCluster struct {
	Name           string
	Host           string
	Port           int
	RestEndpoints  []string
	ConfigPath     string
	Version        string
	Confidence     float64
	DetectionMethods []string
	Metadata       map[string]string
}

// DiscoveryMethod represents a cluster discovery method
type DiscoveryMethod interface {
	Name() string
	Discover(ctx context.Context) ([]*DiscoveredCluster, error)
	Priority() int
}

// NewClusterDiscovery creates a new cluster discovery instance
func NewClusterDiscovery() *ClusterDiscovery {
	return &ClusterDiscovery{
		timeout: 10 * time.Second,
		client: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

// DiscoverClusters discovers SLURM clusters using multiple methods
func (cd *ClusterDiscovery) DiscoverClusters(ctx context.Context) ([]*DiscoveredCluster, error) {
	debug.Logger.Printf("Starting cluster discovery")

	// Initialize discovery methods
	methods := []DiscoveryMethod{
		NewEnvironmentDiscovery(),
		NewConfigFileDiscovery(),
		NewNetworkDiscovery(cd.client),
		NewDNSDiscovery(),
		NewProcessDiscovery(),
	}

	// Channel to collect results
	resultsChan := make(chan []*DiscoveredCluster, len(methods))
	var wg sync.WaitGroup

	// Run discovery methods in parallel
	for _, method := range methods {
		wg.Add(1)
		go func(m DiscoveryMethod) {
			defer wg.Done()
			
			debug.Logger.Printf("Running discovery method: %s", m.Name())
			clusters, err := m.Discover(ctx)
			if err != nil {
				debug.Logger.Printf("Discovery method %s failed: %v", m.Name(), err)
				resultsChan <- []*DiscoveredCluster{}
				return
			}
			
			debug.Logger.Printf("Discovery method %s found %d clusters", m.Name(), len(clusters))
			resultsChan <- clusters
		}(method)
	}

	// Wait for all methods to complete
	go func() {
		wg.Wait()
		close(resultsChan)
	}()

	// Collect and merge results
	var allClusters []*DiscoveredCluster
	for clusters := range resultsChan {
		allClusters = append(allClusters, clusters...)
	}

	// Merge and deduplicate clusters
	mergedClusters := cd.mergeClusters(allClusters)
	
	// Sort by confidence
	cd.sortByConfidence(mergedClusters)

	debug.Logger.Printf("Cluster discovery completed: found %d unique clusters", len(mergedClusters))
	return mergedClusters, nil
}

// Environment Variable Discovery
type EnvironmentDiscovery struct{}

func NewEnvironmentDiscovery() DiscoveryMethod {
	return &EnvironmentDiscovery{}
}

func (ed *EnvironmentDiscovery) Name() string {
	return "environment"
}

func (ed *EnvironmentDiscovery) Priority() int {
	return 10
}

func (ed *EnvironmentDiscovery) Discover(ctx context.Context) ([]*DiscoveredCluster, error) {
	var clusters []*DiscoveredCluster

	// Check SLURM environment variables
	slurmCtldHost := os.Getenv("SLURM_CONTROLLER_HOST")
	slurmConf := os.Getenv("SLURM_CONF")
	slurmConfDir := os.Getenv("SLURM_CONF_DIR")

	if slurmCtldHost != "" {
		cluster := &DiscoveredCluster{
			Name:             "environment-cluster",
			Host:             slurmCtldHost,
			Port:             6820, // Default port
			RestEndpoints:    []string{fmt.Sprintf("https://%s:6820", slurmCtldHost)},
			Confidence:       0.8,
			DetectionMethods: []string{"SLURM_CONTROLLER_HOST"},
			Metadata:         map[string]string{
				"source": "environment",
				"host_env": slurmCtldHost,
			},
		}
		clusters = append(clusters, cluster)
	}

	if slurmConf != "" {
		if cluster := ed.parseConfigFile(slurmConf); cluster != nil {
			cluster.DetectionMethods = append(cluster.DetectionMethods, "SLURM_CONF")
			clusters = append(clusters, cluster)
		}
	}

	if slurmConfDir != "" {
		confPath := filepath.Join(slurmConfDir, "slurm.conf")
		if cluster := ed.parseConfigFile(confPath); cluster != nil {
			cluster.DetectionMethods = append(cluster.DetectionMethods, "SLURM_CONF_DIR")
			clusters = append(clusters, cluster)
		}
	}

	return clusters, nil
}

func (ed *EnvironmentDiscovery) parseConfigFile(path string) *DiscoveredCluster {
	file, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer file.Close()

	cluster := &DiscoveredCluster{
		ConfigPath:       path,
		Confidence:       0.9,
		DetectionMethods: []string{"config-file"},
		Metadata:         map[string]string{
			"source": "config-file",
			"config_path": path,
		},
	}

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		
		if strings.HasPrefix(line, "ClusterName=") {
			cluster.Name = strings.TrimPrefix(line, "ClusterName=")
		} else if strings.HasPrefix(line, "ControlMachine=") {
			cluster.Host = strings.TrimPrefix(line, "ControlMachine=")
		} else if strings.HasPrefix(line, "SlurmctldHost=") {
			// Parse SlurmctldHost which might include port
			hostLine := strings.TrimPrefix(line, "SlurmctldHost=")
			if strings.Contains(hostLine, "(") {
				// Format: SlurmctldHost=hostname(addr:port)
				re := regexp.MustCompile(`^([^(]+)\(([^:]+):?(\d+)?\)`)
				matches := re.FindStringSubmatch(hostLine)
				if len(matches) >= 3 {
					cluster.Host = matches[2]
					if len(matches) > 3 && matches[3] != "" {
						if port, err := strconv.Atoi(matches[3]); err == nil {
							cluster.Port = port
						}
					}
				}
			} else {
				cluster.Host = hostLine
			}
		}
	}

	if cluster.Host != "" && cluster.Port == 0 {
		cluster.Port = 6820 // Default SLURM REST port
	}

	if cluster.Host != "" {
		cluster.RestEndpoints = []string{
			fmt.Sprintf("https://%s:%d", cluster.Host, cluster.Port),
		}
	}

	if cluster.Name == "" {
		cluster.Name = "config-cluster"
	}

	return cluster
}

// Config File Discovery
type ConfigFileDiscovery struct{}

func NewConfigFileDiscovery() DiscoveryMethod {
	return &ConfigFileDiscovery{}
}

func (cfd *ConfigFileDiscovery) Name() string {
	return "config-files"
}

func (cfd *ConfigFileDiscovery) Priority() int {
	return 8
}

func (cfd *ConfigFileDiscovery) Discover(ctx context.Context) ([]*DiscoveredCluster, error) {
	var clusters []*DiscoveredCluster

	// Common SLURM configuration paths
	configPaths := []string{
		"/etc/slurm/slurm.conf",
		"/usr/local/etc/slurm.conf",
		"/opt/slurm/etc/slurm.conf",
		"/usr/local/etc/slurm/slurm.conf",
		filepath.Join(os.Getenv("HOME"), ".slurm", "slurm.conf"),
	}

	ed := &EnvironmentDiscovery{}
	for _, path := range configPaths {
		if cluster := ed.parseConfigFile(path); cluster != nil {
			cluster.DetectionMethods = []string{"config-file-scan"}
			clusters = append(clusters, cluster)
		}
	}

	return clusters, nil
}

// Network Discovery
type NetworkDiscovery struct {
	client *http.Client
}

func NewNetworkDiscovery(client *http.Client) DiscoveryMethod {
	return &NetworkDiscovery{client: client}
}

func (nd *NetworkDiscovery) Name() string {
	return "network"
}

func (nd *NetworkDiscovery) Priority() int {
	return 6
}

func (nd *NetworkDiscovery) Discover(ctx context.Context) ([]*DiscoveredCluster, error) {
	var clusters []*DiscoveredCluster

	// Try common local endpoints
	localEndpoints := []string{
		"http://localhost:6820",
		"https://localhost:6820",
		"http://127.0.0.1:6820",
		"https://127.0.0.1:6820",
	}

	for _, endpoint := range localEndpoints {
		if nd.testEndpoint(ctx, endpoint) {
			cluster := &DiscoveredCluster{
				Name:             "local-cluster",
				Host:             "localhost",
				Port:             6820,
				RestEndpoints:    []string{endpoint},
				Confidence:       0.7,
				DetectionMethods: []string{"network-scan"},
				Metadata:         map[string]string{
					"source": "network",
					"endpoint": endpoint,
				},
			}
			clusters = append(clusters, cluster)
		}
	}

	// Scan local network for common SLURM ports
	clusters = append(clusters, nd.scanLocalNetwork(ctx)...)

	return clusters, nil
}

func (nd *NetworkDiscovery) testEndpoint(ctx context.Context, endpoint string) bool {
	// Test if SLURM REST API is available at endpoint
	testURL := endpoint + "/slurm/v0.0.40/ping"
	
	req, err := http.NewRequestWithContext(ctx, "GET", testURL, nil)
	if err != nil {
		return false
	}

	resp, err := nd.client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	// Check if it looks like a SLURM REST API response
	return resp.StatusCode == 200 || resp.StatusCode == 401 || resp.StatusCode == 403
}

func (nd *NetworkDiscovery) scanLocalNetwork(ctx context.Context) []*DiscoveredCluster {
	var clusters []*DiscoveredCluster

	// Get local network interfaces
	interfaces, err := net.Interfaces()
	if err != nil {
		return clusters
	}

	for _, iface := range interfaces {
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}

		for _, addr := range addrs {
			ipNet, ok := addr.(*net.IPNet)
			if !ok || ipNet.IP.IsLoopback() {
				continue
			}

			if ipNet.IP.To4() != nil {
				// Scan a few IPs in the subnet (limit to avoid being aggressive)
				clusters = append(clusters, nd.scanSubnet(ctx, ipNet, 5)...)
			}
		}
	}

	return clusters
}

func (nd *NetworkDiscovery) scanSubnet(ctx context.Context, ipNet *net.IPNet, maxHosts int) []*DiscoveredCluster {
	var clusters []*DiscoveredCluster
	
	// Simple scan of a few IPs (in production, would be more sophisticated)
	baseIP := ipNet.IP.To4()
	if baseIP == nil {
		return clusters
	}

	scanned := 0
	for i := 1; i < 255 && scanned < maxHosts; i++ {
		ip := net.IPv4(baseIP[0], baseIP[1], baseIP[2], byte(i))
		endpoint := fmt.Sprintf("https://%s:6820", ip.String())
		
		if nd.testEndpoint(ctx, endpoint) {
			cluster := &DiscoveredCluster{
				Name:             fmt.Sprintf("network-cluster-%s", ip.String()),
				Host:             ip.String(),
				Port:             6820,
				RestEndpoints:    []string{endpoint},
				Confidence:       0.5,
				DetectionMethods: []string{"network-scan"},
				Metadata:         map[string]string{
					"source": "network-scan",
					"scanned_ip": ip.String(),
				},
			}
			clusters = append(clusters, cluster)
		}
		scanned++
	}

	return clusters
}

// DNS Discovery
type DNSDiscovery struct{}

func NewDNSDiscovery() DiscoveryMethod {
	return &DNSDiscovery{}
}

func (dd *DNSDiscovery) Name() string {
	return "dns"
}

func (dd *DNSDiscovery) Priority() int {
	return 7
}

func (dd *DNSDiscovery) Discover(ctx context.Context) ([]*DiscoveredCluster, error) {
	var clusters []*DiscoveredCluster

	// Try common SLURM SRV records
	srvNames := []string{
		"_slurmrestd._tcp",
		"_slurm._tcp", 
		"_slurmctld._tcp",
	}

	// Get domain from hostname
	hostname, err := os.Hostname()
	if err != nil {
		return clusters, nil
	}

	domain := ""
	if strings.Contains(hostname, ".") {
		parts := strings.Split(hostname, ".")
		if len(parts) > 1 {
			domain = strings.Join(parts[1:], ".")
		}
	}

	if domain == "" {
		return clusters, nil
	}

	for _, srvName := range srvNames {
		fullSrvName := srvName + "." + domain
		
		_, srvRecords, err := net.LookupSRV("", "", fullSrvName)
		if err != nil {
			continue
		}

		for _, srv := range srvRecords {
			cluster := &DiscoveredCluster{
				Name:             fmt.Sprintf("dns-cluster-%s", strings.TrimSuffix(srv.Target, ".")),
				Host:             strings.TrimSuffix(srv.Target, "."),
				Port:             int(srv.Port),
				RestEndpoints:    []string{fmt.Sprintf("https://%s:%d", strings.TrimSuffix(srv.Target, "."), srv.Port)},
				Confidence:       0.8,
				DetectionMethods: []string{"dns-srv"},
				Metadata:         map[string]string{
					"source": "dns",
					"srv_record": fullSrvName,
					"priority": strconv.Itoa(int(srv.Priority)),
					"weight": strconv.Itoa(int(srv.Weight)),
				},
			}
			clusters = append(clusters, cluster)
		}
	}

	return clusters, nil
}

// Process Discovery
type ProcessDiscovery struct{}

func NewProcessDiscovery() DiscoveryMethod {
	return &ProcessDiscovery{}
}

func (pd *ProcessDiscovery) Name() string {
	return "process"
}

func (pd *ProcessDiscovery) Priority() int {
	return 4
}

func (pd *ProcessDiscovery) Discover(ctx context.Context) ([]*DiscoveredCluster, error) {
	var clusters []*DiscoveredCluster

	// Check for running SLURM processes
	if pd.isProcessRunning("slurmctld") {
		cluster := &DiscoveredCluster{
			Name:             "local-slurmctld",
			Host:             "localhost",
			Port:             6820,
			RestEndpoints:    []string{"https://localhost:6820"},
			Confidence:       0.6,
			DetectionMethods: []string{"process-scan"},
			Metadata:         map[string]string{
				"source": "process",
				"process": "slurmctld",
			},
		}
		clusters = append(clusters, cluster)
	}

	if pd.isProcessRunning("slurmrestd") {
		cluster := &DiscoveredCluster{
			Name:             "local-slurmrestd",
			Host:             "localhost", 
			Port:             6820,
			RestEndpoints:    []string{"https://localhost:6820"},
			Confidence:       0.9,
			DetectionMethods: []string{"process-scan"},
			Metadata:         map[string]string{
				"source": "process",
				"process": "slurmrestd",
			},
		}
		clusters = append(clusters, cluster)
	}

	return clusters, nil
}

func (pd *ProcessDiscovery) isProcessRunning(processName string) bool {
	// Simple process check using pgrep (Linux/Unix)
	// cmd := fmt.Sprintf("pgrep %s", processName) // Not used in this implementation
	
	// This is a simplified check - in production would use proper process scanning
	_, err := os.Stat("/proc")
	if err != nil {
		return false // Not a /proc filesystem
	}

	// Check /proc for process directories
	procEntries, err := os.ReadDir("/proc")
	if err != nil {
		return false
	}

	for _, entry := range procEntries {
		if !entry.IsDir() {
			continue
		}

		// Check if directory name is numeric (PID)
		if _, err := strconv.Atoi(entry.Name()); err != nil {
			continue
		}

		// Read command line
		cmdlinePath := filepath.Join("/proc", entry.Name(), "cmdline")
		if cmdlineData, err := os.ReadFile(cmdlinePath); err == nil {
			cmdline := string(cmdlineData)
			if strings.Contains(cmdline, processName) {
				return true
			}
		}
	}

	return false
}

// Helper methods for ClusterDiscovery

// mergeClusters merges and deduplicates discovered clusters
func (cd *ClusterDiscovery) mergeClusters(clusters []*DiscoveredCluster) []*DiscoveredCluster {
	clusterMap := make(map[string]*DiscoveredCluster)

	for _, cluster := range clusters {
		key := cd.getClusterKey(cluster)
		
		if existing, exists := clusterMap[key]; exists {
			// Merge clusters with same key
			existing.Confidence = (existing.Confidence + cluster.Confidence) / 2
			existing.DetectionMethods = append(existing.DetectionMethods, cluster.DetectionMethods...)
			
			// Merge metadata
			for k, v := range cluster.Metadata {
				existing.Metadata[k] = v
			}
			
			// Merge REST endpoints
			for _, endpoint := range cluster.RestEndpoints {
				found := false
				for _, existingEndpoint := range existing.RestEndpoints {
					if existingEndpoint == endpoint {
						found = true
						break
					}
				}
				if !found {
					existing.RestEndpoints = append(existing.RestEndpoints, endpoint)
				}
			}
		} else {
			clusterMap[key] = cluster
		}
	}

	var result []*DiscoveredCluster
	for _, cluster := range clusterMap {
		result = append(result, cluster)
	}

	return result
}

// getClusterKey generates a unique key for a cluster
func (cd *ClusterDiscovery) getClusterKey(cluster *DiscoveredCluster) string {
	if cluster.Host != "" {
		return fmt.Sprintf("%s:%d", cluster.Host, cluster.Port)
	}
	if len(cluster.RestEndpoints) > 0 {
		return cluster.RestEndpoints[0]
	}
	return cluster.Name
}

// sortByConfidence sorts clusters by confidence score (highest first)
func (cd *ClusterDiscovery) sortByConfidence(clusters []*DiscoveredCluster) {
	for i := 0; i < len(clusters)-1; i++ {
		for j := i + 1; j < len(clusters); j++ {
			if clusters[i].Confidence < clusters[j].Confidence {
				clusters[i], clusters[j] = clusters[j], clusters[i]
			}
		}
	}
}

// ToClusterConfig converts a DiscoveredCluster to config.ClusterConfig
func (dc *DiscoveredCluster) ToClusterConfig() config.ClusterConfig {
	clusterConfig := config.ClusterConfig{
		APIVersion: "v0.0.43",
		Timeout:    "30s",
	}

	if len(dc.RestEndpoints) > 0 {
		clusterConfig.Endpoint = dc.RestEndpoints[0]
	}

	return clusterConfig
}