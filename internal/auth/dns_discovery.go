package auth

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/jontk/s9s/internal/debug"
)

// DNSEndpointDiscoverer implements DNS-based service discovery
type DNSEndpointDiscoverer struct {
	config   DiscoveryConfig
	balancer LoadBalancer
	resolver *net.Resolver
	cache    *dnsCache
	// TODO(lint): Review unused code - field mutex is unused
	// mutex     sync.RWMutex
}

// dnsCache caches DNS resolution results
type dnsCache struct {
	mutex      sync.RWMutex
	entries    map[string]*dnsCacheEntry
	defaultTTL time.Duration
}

// dnsCacheEntry represents a cached DNS result
type dnsCacheEntry struct {
	endpoints []Endpoint
	expires   time.Time
}

// NewDNSEndpointDiscoverer creates a new DNS-based endpoint discoverer
func NewDNSEndpointDiscoverer() EndpointDiscoverer {
	return &DNSEndpointDiscoverer{
		balancer: NewRoundRobinLoadBalancer(),
		resolver: &net.Resolver{
			PreferGo: true,
			Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
				d := net.Dialer{
					Timeout: time.Second * 5,
				}
				return d.DialContext(ctx, network, address)
			},
		},
		cache: &dnsCache{
			entries:    make(map[string]*dnsCacheEntry),
			defaultTTL: 5 * time.Minute,
		},
	}
}

// GetInfo returns information about this discoverer
func (d *DNSEndpointDiscoverer) GetInfo() DiscovererInfo {
	return DiscovererInfo{
		Name:        "dns",
		Version:     "1.0.0",
		Description: "DNS-based service discovery with SRV record support",
		Author:      "s9s Team",
		Supported:   []string{"dns", "srv", "txt", "a", "aaaa"},
	}
}

// Initialize initializes the DNS endpoint discoverer
func (d *DNSEndpointDiscoverer) Initialize(ctx context.Context, config DiscoveryConfig) error {
	d.config = config

	// Validate required configuration
	serviceName := config.GetString("service_name")
	if serviceName == "" {
		return fmt.Errorf("service_name is required for DNS discovery")
	}

	// Set custom DNS servers if provided
	if dnsServers := config.Get("dns_servers"); dnsServers != nil {
		if servers, ok := dnsServers.([]interface{}); ok {
			var serverList []string
			for _, server := range servers {
				if serverStr, ok := server.(string); ok {
					serverList = append(serverList, serverStr)
				}
			}
			if len(serverList) > 0 {
				d.resolver = &net.Resolver{
					PreferGo: true,
					Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
						// Use first DNS server from the list
						dnsAddr := serverList[0]
						if !strings.Contains(dnsAddr, ":") {
							dnsAddr += ":53"
						}

						dialer := net.Dialer{
							Timeout: time.Second * 5,
						}
						return dialer.DialContext(ctx, network, dnsAddr)
					},
				}
			}
		}
	}

	// Set cache TTL if provided
	if ttl := config.GetInt("cache_ttl"); ttl > 0 {
		d.cache.defaultTTL = time.Duration(ttl) * time.Second
	}

	debug.Logger.Printf("Initialized DNS endpoint discoverer for service: %s", serviceName)
	return nil
}

// DiscoverEndpoints discovers endpoints using DNS resolution
func (d *DNSEndpointDiscoverer) DiscoverEndpoints(ctx context.Context, clusterID string) ([]Endpoint, error) {
	serviceName := d.config.GetString("service_name")

	// Check cache first
	if endpoints := d.getFromCache(serviceName); endpoints != nil {
		debug.Logger.Printf("Retrieved %d endpoints from DNS cache for cluster %s", len(endpoints), clusterID)
		return endpoints, nil
	}

	debug.Logger.Printf("Discovering endpoints via DNS for service: %s", serviceName)

	var endpoints []Endpoint
	var err error

	// Determine discovery method based on service name format
	if strings.HasPrefix(serviceName, "_") {
		// SRV record discovery
		endpoints, err = d.discoverViaSRV(ctx, serviceName, clusterID)
	} else {
		// A/AAAA record discovery
		endpoints, err = d.discoverViaA(ctx, serviceName, clusterID)
	}

	if err != nil {
		return nil, fmt.Errorf("DNS discovery failed: %w", err)
	}

	if len(endpoints) == 0 {
		return nil, fmt.Errorf("no endpoints discovered for service: %s", serviceName)
	}

	// Cache the results
	d.putInCache(serviceName, endpoints)

	debug.Logger.Printf("Discovered %d endpoints via DNS for cluster %s", len(endpoints), clusterID)
	return endpoints, nil
}

// discoverViaSRV discovers endpoints using SRV records
func (d *DNSEndpointDiscoverer) discoverViaSRV(ctx context.Context, serviceName, clusterID string) ([]Endpoint, error) {
	debug.Logger.Printf("Performing SRV lookup for: %s", serviceName)

	_, srvRecords, err := d.resolver.LookupSRV(ctx, "", "", serviceName)
	if err != nil {
		return nil, fmt.Errorf("SRV lookup failed: %w", err)
	}

	var endpoints []Endpoint
	defaultPort := d.config.GetInt("port")
	scheme := d.config.GetString("scheme")
	if scheme == "" {
		scheme = "https"
	}

	for _, srv := range srvRecords {
		// Remove trailing dot from target
		target := strings.TrimSuffix(srv.Target, ".")

		port := int(srv.Port)
		if port == 0 && defaultPort > 0 {
			port = defaultPort
		}

		url := fmt.Sprintf("%s://%s", scheme, target)
		if port > 0 && port != 80 && port != 443 {
			url = fmt.Sprintf("%s://%s:%d", scheme, target, port)
		}

		endpoint := Endpoint{
			URL:       url,
			ClusterID: clusterID,
			Weight:    int(srv.Weight),
			Status:    EndpointStatusUnknown,
			Metadata: map[string]string{
				"discovery_method": "dns-srv",
				"target":           target,
				"port":             strconv.Itoa(port),
				"priority":         strconv.Itoa(int(srv.Priority)),
				"weight":           strconv.Itoa(int(srv.Weight)),
			},
			LastCheck: time.Time{},
		}

		// Default weight if SRV record has 0 weight
		if endpoint.Weight == 0 {
			endpoint.Weight = 100
		}

		endpoints = append(endpoints, endpoint)
	}

	// Sort by priority (lower values = higher priority)
	d.sortEndpointsByPriority(endpoints)

	return endpoints, nil
}

// discoverViaA discovers endpoints using A/AAAA records
func (d *DNSEndpointDiscoverer) discoverViaA(ctx context.Context, hostname, clusterID string) ([]Endpoint, error) {
	debug.Logger.Printf("Performing A/AAAA lookup for: %s", hostname)

	// Look up A records (IPv4)
	ipAddrs, err := d.resolver.LookupIPAddr(ctx, hostname)
	if err != nil {
		return nil, fmt.Errorf("A/AAAA lookup failed: %w", err)
	}

	var endpoints []Endpoint
	port := d.config.GetInt("port")
	scheme := d.config.GetString("scheme")
	if scheme == "" {
		scheme = "https"
	}
	if port == 0 {
		port = 6820 // Default SLURM REST API port
	}

	for i, ipAddr := range ipAddrs {
		url := fmt.Sprintf("%s://%s", scheme, ipAddr.IP.String())
		if port > 0 && port != 80 && port != 443 {
			url = fmt.Sprintf("%s://%s:%d", scheme, ipAddr.IP.String(), port)
		}

		endpoint := Endpoint{
			URL:       url,
			ClusterID: clusterID,
			Weight:    100, // Default weight for A/AAAA records
			Status:    EndpointStatusUnknown,
			Metadata: map[string]string{
				"discovery_method": "dns-a",
				"hostname":         hostname,
				"ip_address":       ipAddr.IP.String(),
				"ip_version":       getIPVersion(ipAddr.IP),
				"index":            strconv.Itoa(i),
			},
			LastCheck: time.Time{},
		}

		endpoints = append(endpoints, endpoint)
	}

	return endpoints, nil
}

// getIPVersion returns the IP version string
func getIPVersion(ip net.IP) string {
	if ip.To4() != nil {
		return "ipv4"
	}
	return "ipv6"
}

// sortEndpointsByPriority sorts endpoints by SRV priority
func (d *DNSEndpointDiscoverer) sortEndpointsByPriority(endpoints []Endpoint) {
	// Simple bubble sort by priority (from metadata)
	n := len(endpoints)
	for i := 0; i < n-1; i++ {
		for j := 0; j < n-i-1; j++ {
			priority1, _ := strconv.Atoi(endpoints[j].Metadata["priority"])
			priority2, _ := strconv.Atoi(endpoints[j+1].Metadata["priority"])

			if priority1 > priority2 {
				endpoints[j], endpoints[j+1] = endpoints[j+1], endpoints[j]
			}
		}
	}
}

// HealthCheck performs a health check on an endpoint
func (d *DNSEndpointDiscoverer) HealthCheck(ctx context.Context, endpoint Endpoint) error {
	debug.Logger.Printf("Health checking endpoint: %s", endpoint.URL)

	// Use the same health check logic as static discovery
	healthPath := d.config.GetString("health_check_path")
	if healthPath == "" {
		healthPath = "/slurm/v0.0.40/ping"
	}

	healthURL := endpoint.URL + healthPath

	// Create HTTP client with timeout
	client := d.getHTTPClient()

	// Create health check request
	req, err := http.NewRequestWithContext(ctx, "GET", healthURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create health check request: %w", err)
	}

	req.Header.Set("User-Agent", "s9s-dns-health-check/1.0")

	// Execute health check
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("health check request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	// Check response status
	if resp.StatusCode < 200 || resp.StatusCode >= 400 {
		return fmt.Errorf("health check failed with status %d", resp.StatusCode)
	}

	debug.Logger.Printf("Health check passed for endpoint: %s", endpoint.URL)
	return nil
}

// getHTTPClient returns an HTTP client for health checks
func (d *DNSEndpointDiscoverer) getHTTPClient() *http.Client {
	timeout := 10 * time.Second
	if customTimeout := d.config.GetInt("health_check_timeout"); customTimeout > 0 {
		timeout = time.Duration(customTimeout) * time.Second
	}

	return &http.Client{
		Timeout: timeout,
	}
}

// GetLoadBalancer returns the load balancer for endpoint selection
func (d *DNSEndpointDiscoverer) GetLoadBalancer() LoadBalancer {
	return d.balancer
}

// Cache management

// getFromCache retrieves endpoints from cache if not expired
func (d *DNSEndpointDiscoverer) getFromCache(serviceName string) []Endpoint {
	d.cache.mutex.RLock()
	defer d.cache.mutex.RUnlock()

	entry, exists := d.cache.entries[serviceName]
	if !exists || time.Now().After(entry.expires) {
		return nil
	}

	// Return a copy to avoid external modifications
	result := make([]Endpoint, len(entry.endpoints))
	copy(result, entry.endpoints)
	return result
}

// putInCache stores endpoints in cache
func (d *DNSEndpointDiscoverer) putInCache(serviceName string, endpoints []Endpoint) {
	d.cache.mutex.Lock()
	defer d.cache.mutex.Unlock()

	ttl := d.cache.defaultTTL
	if customTTL := d.config.GetInt("cache_ttl"); customTTL > 0 {
		ttl = time.Duration(customTTL) * time.Second
	}

	d.cache.entries[serviceName] = &dnsCacheEntry{
		endpoints: endpoints,
		expires:   time.Now().Add(ttl),
	}

	debug.Logger.Printf("Cached %d endpoints for service %s (TTL: %v)", len(endpoints), serviceName, ttl)
}

// clearExpiredCache removes expired entries from cache
func (d *DNSEndpointDiscoverer) clearExpiredCache() {
	d.cache.mutex.Lock()
	defer d.cache.mutex.Unlock()

	now := time.Now()
	for serviceName, entry := range d.cache.entries {
		if now.After(entry.expires) {
			delete(d.cache.entries, serviceName)
			debug.Logger.Printf("Removed expired cache entry for service: %s", serviceName)
		}
	}
}

// StartPeriodicDiscovery starts periodic DNS discovery and cache cleanup
func (d *DNSEndpointDiscoverer) StartPeriodicDiscovery(ctx context.Context) {
	refreshInterval := 5 * time.Minute
	if interval := d.config.GetInt("refresh_interval"); interval > 0 {
		refreshInterval = time.Duration(interval) * time.Second
	}

	ticker := time.NewTicker(refreshInterval)
	defer ticker.Stop()

	// Start cache cleanup ticker
	cacheCleanupTicker := time.NewTicker(1 * time.Minute)
	defer cacheCleanupTicker.Stop()

	debug.Logger.Printf("Started DNS discovery with refresh interval: %v", refreshInterval)

	for {
		select {
		case <-ticker.C:
			// Clear cache to force fresh discovery on next request
			d.clearCache()
			debug.Logger.Printf("Cleared DNS cache for periodic refresh")

		case <-cacheCleanupTicker.C:
			// Clean up expired cache entries
			d.clearExpiredCache()

		case <-ctx.Done():
			debug.Logger.Printf("Stopping DNS periodic discovery")
			return
		}
	}
}

// clearCache clears all cache entries
func (d *DNSEndpointDiscoverer) clearCache() {
	d.cache.mutex.Lock()
	defer d.cache.mutex.Unlock()

	d.cache.entries = make(map[string]*dnsCacheEntry)
}

// Cleanup performs any necessary cleanup
func (d *DNSEndpointDiscoverer) Cleanup() error {
	d.clearCache()
	debug.Logger.Printf("DNS endpoint discoverer cleanup completed")
	return nil
}

// DNSLoadBalancer extends weighted round-robin with DNS-specific features
type DNSLoadBalancer struct {
	*WeightedRoundRobinLoadBalancer
	preferIPv4 bool
}

// NewDNSLoadBalancer creates a DNS-aware load balancer
func NewDNSLoadBalancer() LoadBalancer {
	return &DNSLoadBalancer{
		WeightedRoundRobinLoadBalancer: &WeightedRoundRobinLoadBalancer{
			weights: make(map[string]int),
			current: make(map[string]int),
		},
		preferIPv4: true, // Default to IPv4 preference
	}
}

// SelectEndpoint selects an endpoint with DNS-specific logic
func (d *DNSLoadBalancer) SelectEndpoint(endpoints []Endpoint) (*Endpoint, error) {
	if len(endpoints) == 0 {
		return nil, fmt.Errorf("no endpoints available")
	}

	// Filter by health status
	healthy := make([]Endpoint, 0)
	for _, ep := range endpoints {
		if ep.Status != EndpointStatusUnhealthy {
			healthy = append(healthy, ep)
		}
	}

	if len(healthy) == 0 {
		healthy = endpoints // Fallback to all endpoints
	}

	// Apply IP version preference if multiple IPs for same service
	if d.preferIPv4 && len(healthy) > 1 {
		ipv4Endpoints := make([]Endpoint, 0)
		for _, ep := range healthy {
			if ep.Metadata["ip_version"] == "ipv4" {
				ipv4Endpoints = append(ipv4Endpoints, ep)
			}
		}
		if len(ipv4Endpoints) > 0 {
			healthy = ipv4Endpoints
		}
	}

	// Use weighted round-robin selection
	return d.WeightedRoundRobinLoadBalancer.SelectEndpoint(healthy)
}

// GetStrategy returns the load balancing strategy name
func (d *DNSLoadBalancer) GetStrategy() string {
	return "dns-weighted-round-robin"
}
