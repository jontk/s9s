// Package discovery provides automatic discovery of SLURM cluster endpoints.
package discovery

import (
	"context"
	"fmt"
	"net"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/jontk/s9s/internal/debug"
)

// DiscoveredEndpoint represents a discovered slurmrestd endpoint
type DiscoveredEndpoint struct {
	URL        string            // Full URL (e.g., "http://hostname:6820")
	Host       string            // Hostname or IP
	Port       int               // Port number
	Source     string            // Discovery source (env, config, srv, scontrol)
	Confidence float64           // Confidence score (0-1)
	Metadata   map[string]string // Additional metadata
}

// AutoDiscovery provides automatic discovery of slurmrestd endpoints
type AutoDiscovery struct {
	enabled       bool
	timeout       time.Duration
	defaultPort   int
	scontrolPath  string
	cacheDuration time.Duration

	// Cached result
	mu             sync.RWMutex
	cachedEndpoint *DiscoveredEndpoint
	cacheExpiry    time.Time
}

// AutoDiscoveryConfig holds configuration for auto-discovery
type AutoDiscoveryConfig struct {
	Enabled        bool          `mapstructure:"enabled"`
	EnableEndpoint bool          `mapstructure:"enableEndpoint"`
	EnableToken    bool          `mapstructure:"enableToken"`
	Timeout        time.Duration `mapstructure:"timeout"`
	DefaultPort    int           `mapstructure:"defaultPort"`
	ScontrolPath   string        `mapstructure:"scontrolPath"`
	CacheDuration  time.Duration `mapstructure:"cacheDuration"`
}

// DefaultAutoDiscoveryConfig returns the default configuration
func DefaultAutoDiscoveryConfig() AutoDiscoveryConfig {
	return AutoDiscoveryConfig{
		Enabled:        true,
		EnableEndpoint: true,
		EnableToken:    true,
		Timeout:        10 * time.Second,
		DefaultPort:    6820,
		ScontrolPath:   "scontrol",
		CacheDuration:  5 * time.Minute,
	}
}

// NewAutoDiscovery creates a new AutoDiscovery instance with default configuration
func NewAutoDiscovery() *AutoDiscovery {
	cfg := DefaultAutoDiscoveryConfig()
	return NewAutoDiscoveryWithConfig(cfg)
}

// NewAutoDiscoveryWithConfig creates a new AutoDiscovery instance with custom configuration
func NewAutoDiscoveryWithConfig(cfg AutoDiscoveryConfig) *AutoDiscovery {
	return &AutoDiscovery{
		enabled:       cfg.Enabled,
		timeout:       cfg.Timeout,
		defaultPort:   cfg.DefaultPort,
		scontrolPath:  cfg.ScontrolPath,
		cacheDuration: cfg.CacheDuration,
	}
}

// DiscoverEndpoint discovers a slurmrestd endpoint using the priority chain
// Priority order:
// 1. SRV record _slurmrestd._tcp
// 2. SRV record _slurmctld._tcp (derive with default port)
// 3. scontrol ping output
func (ad *AutoDiscovery) DiscoverEndpoint(ctx context.Context) (*DiscoveredEndpoint, error) {
	if !ad.enabled {
		return nil, fmt.Errorf("auto-discovery is disabled")
	}

	// Check cache first
	ad.mu.RLock()
	if ad.cachedEndpoint != nil && time.Now().Before(ad.cacheExpiry) {
		endpoint := ad.cachedEndpoint
		ad.mu.RUnlock()
		debug.Logger.Printf("Returning cached endpoint: %s", endpoint.URL)
		return endpoint, nil
	}
	ad.mu.RUnlock()

	debug.Logger.Printf("Starting endpoint auto-discovery")

	// Create context with timeout
	ctxWithTimeout, cancel := context.WithTimeout(ctx, ad.timeout)
	defer cancel()

	var endpoint *DiscoveredEndpoint
	var err error

	// Priority 1: Try SRV record _slurmrestd._tcp
	endpoint, err = ad.discoverViaSRV(ctxWithTimeout, "_slurmrestd._tcp")
	if err == nil && endpoint != nil {
		debug.Logger.Printf("Found endpoint via _slurmrestd._tcp SRV record: %s", endpoint.URL)
		ad.cacheEndpoint(endpoint)
		return endpoint, nil
	}
	debug.Logger.Printf("SRV record _slurmrestd._tcp not found: %v", err)

	// Priority 2: Try SRV record _slurmctld._tcp
	endpoint, err = ad.discoverViaSRV(ctxWithTimeout, "_slurmctld._tcp")
	if err == nil && endpoint != nil {
		// Use default port for slurmrestd when using slurmctld SRV
		endpoint.Port = ad.defaultPort
		endpoint.URL = fmt.Sprintf("http://%s:%d", endpoint.Host, endpoint.Port)
		debug.Logger.Printf("Found endpoint via _slurmctld._tcp SRV record (using default port): %s", endpoint.URL)
		ad.cacheEndpoint(endpoint)
		return endpoint, nil
	}
	debug.Logger.Printf("SRV record _slurmctld._tcp not found: %v", err)

	// Priority 3: Try scontrol ping
	endpoint, err = ad.discoverViaScontrol(ctxWithTimeout)
	if err == nil && endpoint != nil {
		debug.Logger.Printf("Found endpoint via scontrol ping: %s", endpoint.URL)
		ad.cacheEndpoint(endpoint)
		return endpoint, nil
	}
	debug.Logger.Printf("scontrol ping discovery failed: %v", err)

	return nil, fmt.Errorf("unable to discover slurmrestd endpoint: all discovery methods failed")
}

// discoverViaSRV attempts to discover endpoint using DNS SRV records
func (ad *AutoDiscovery) discoverViaSRV(ctx context.Context, srvName string) (*DiscoveredEndpoint, error) {
	// Get domain from hostname
	hostname, err := os.Hostname()
	if err != nil {
		return nil, fmt.Errorf("failed to get hostname: %w", err)
	}

	domain := ""
	if strings.Contains(hostname, ".") {
		parts := strings.Split(hostname, ".")
		if len(parts) > 1 {
			domain = strings.Join(parts[1:], ".")
		}
	}

	if domain == "" {
		return nil, fmt.Errorf("unable to determine domain from hostname")
	}

	fullSrvName := srvName + "." + domain

	// Create a resolver with custom timeout
	resolver := &net.Resolver{
		PreferGo: true,
		Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
			d := net.Dialer{
				Timeout: 5 * time.Second,
			}
			return d.DialContext(ctx, network, address)
		},
	}

	_, srvRecords, err := resolver.LookupSRV(ctx, "", "", fullSrvName)
	if err != nil {
		return nil, fmt.Errorf("SRV lookup failed for %s: %w", fullSrvName, err)
	}

	if len(srvRecords) == 0 {
		return nil, fmt.Errorf("no SRV records found for %s", fullSrvName)
	}

	// Use the first (highest priority) SRV record
	srv := srvRecords[0]
	host := strings.TrimSuffix(srv.Target, ".")
	port := int(srv.Port)

	if port == 0 {
		port = ad.defaultPort
	}

	return &DiscoveredEndpoint{
		URL:        fmt.Sprintf("http://%s:%d", host, port),
		Host:       host,
		Port:       port,
		Source:     "srv-" + srvName,
		Confidence: 0.9,
		Metadata: map[string]string{
			"srv_record": fullSrvName,
			"priority":   fmt.Sprintf("%d", srv.Priority),
			"weight":     fmt.Sprintf("%d", srv.Weight),
		},
	}, nil
}

// discoverViaScontrol attempts to discover endpoint using scontrol ping
func (ad *AutoDiscovery) discoverViaScontrol(ctx context.Context) (*DiscoveredEndpoint, error) {
	sd := NewScontrolDiscoveryWithConfig(ad.scontrolPath, ad.timeout, ad.defaultPort)

	clusters, err := sd.Discover(ctx)
	if err != nil {
		return nil, err
	}

	if len(clusters) == 0 {
		return nil, fmt.Errorf("no clusters discovered via scontrol")
	}

	// Use the first cluster (highest confidence)
	cluster := clusters[0]
	if len(cluster.RestEndpoints) == 0 {
		return nil, fmt.Errorf("no REST endpoints in discovered cluster")
	}

	return &DiscoveredEndpoint{
		URL:        cluster.RestEndpoints[0],
		Host:       cluster.Host,
		Port:       cluster.Port,
		Source:     "scontrol",
		Confidence: cluster.Confidence,
		Metadata:   cluster.Metadata,
	}, nil
}

// cacheEndpoint caches the discovered endpoint
func (ad *AutoDiscovery) cacheEndpoint(endpoint *DiscoveredEndpoint) {
	ad.mu.Lock()
	defer ad.mu.Unlock()
	ad.cachedEndpoint = endpoint
	ad.cacheExpiry = time.Now().Add(ad.cacheDuration)
}

// ClearCache clears the cached endpoint
func (ad *AutoDiscovery) ClearCache() {
	ad.mu.Lock()
	defer ad.mu.Unlock()
	ad.cachedEndpoint = nil
	ad.cacheExpiry = time.Time{}
}

// SetEnabled enables or disables auto-discovery
func (ad *AutoDiscovery) SetEnabled(enabled bool) {
	ad.enabled = enabled
}

// IsEnabled returns whether auto-discovery is enabled
func (ad *AutoDiscovery) IsEnabled() bool {
	return ad.enabled
}

// GetCachedEndpoint returns the cached endpoint without triggering discovery
func (ad *AutoDiscovery) GetCachedEndpoint() *DiscoveredEndpoint {
	ad.mu.RLock()
	defer ad.mu.RUnlock()
	if ad.cachedEndpoint != nil && time.Now().Before(ad.cacheExpiry) {
		return ad.cachedEndpoint
	}
	return nil
}

// Result represents the result of endpoint discovery with source information
type Result struct {
	Endpoint *DiscoveredEndpoint
	Source   string
	Error    error
}

//nolint:revive // type alias for backward compatibility
type DiscoveryResult = Result

// DiscoverEndpointWithFallback tries multiple discovery methods and returns all results
func (ad *AutoDiscovery) DiscoverEndpointWithFallback(ctx context.Context) []Result {
	results := make([]Result, 0, 3)

	ctxWithTimeout, cancel := context.WithTimeout(ctx, ad.timeout)
	defer cancel()

	// Try SRV _slurmrestd._tcp
	endpoint, err := ad.discoverViaSRV(ctxWithTimeout, "_slurmrestd._tcp")
	results = append(results, Result{
		Endpoint: endpoint,
		Source:   "srv-_slurmrestd._tcp",
		Error:    err,
	})

	// Try SRV _slurmctld._tcp
	endpoint, err = ad.discoverViaSRV(ctxWithTimeout, "_slurmctld._tcp")
	if endpoint != nil {
		endpoint.Port = ad.defaultPort
		endpoint.URL = fmt.Sprintf("http://%s:%d", endpoint.Host, endpoint.Port)
	}
	results = append(results, Result{
		Endpoint: endpoint,
		Source:   "srv-_slurmctld._tcp",
		Error:    err,
	})

	// Try scontrol ping
	endpoint, err = ad.discoverViaScontrol(ctxWithTimeout)
	results = append(results, Result{
		Endpoint: endpoint,
		Source:   "scontrol",
		Error:    err,
	})

	return results
}
