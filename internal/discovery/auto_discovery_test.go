package discovery

import (
	"testing"
	"time"
)

func TestDefaultAutoDiscoveryConfig(t *testing.T) {
	cfg := DefaultAutoDiscoveryConfig()

	if !cfg.Enabled {
		t.Errorf("expected Enabled=true, got false")
	}

	if !cfg.EnableEndpoint {
		t.Errorf("expected EnableEndpoint=true, got false")
	}

	if !cfg.EnableToken {
		t.Errorf("expected EnableToken=true, got false")
	}

	if cfg.Timeout != 10*time.Second {
		t.Errorf("expected Timeout=10s, got %v", cfg.Timeout)
	}

	if cfg.DefaultPort != 6820 {
		t.Errorf("expected DefaultPort=6820, got %d", cfg.DefaultPort)
	}

	if cfg.ScontrolPath != "scontrol" {
		t.Errorf("expected ScontrolPath='scontrol', got %q", cfg.ScontrolPath)
	}

	if cfg.CacheDuration != 5*time.Minute {
		t.Errorf("expected CacheDuration=5m, got %v", cfg.CacheDuration)
	}
}

func TestNewAutoDiscovery(t *testing.T) {
	ad := NewAutoDiscovery()

	if ad == nil {
		t.Fatalf("expected non-nil AutoDiscovery")
	}

	if !ad.IsEnabled() {
		t.Errorf("expected auto-discovery to be enabled by default")
	}
}

func TestNewAutoDiscoveryWithConfig(t *testing.T) {
	cfg := AutoDiscoveryConfig{
		Enabled:       false,
		Timeout:       30 * time.Second,
		DefaultPort:   8080,
		ScontrolPath:  "/usr/bin/scontrol",
		CacheDuration: 10 * time.Minute,
	}

	ad := NewAutoDiscoveryWithConfig(cfg)

	if ad.IsEnabled() {
		t.Errorf("expected auto-discovery to be disabled")
	}

	if ad.timeout != 30*time.Second {
		t.Errorf("expected timeout=30s, got %v", ad.timeout)
	}

	if ad.defaultPort != 8080 {
		t.Errorf("expected defaultPort=8080, got %d", ad.defaultPort)
	}

	if ad.scontrolPath != "/usr/bin/scontrol" {
		t.Errorf("expected scontrolPath='/usr/bin/scontrol', got %q", ad.scontrolPath)
	}

	if ad.cacheDuration != 10*time.Minute {
		t.Errorf("expected cacheDuration=10m, got %v", ad.cacheDuration)
	}
}

func TestSetEnabled(t *testing.T) {
	ad := NewAutoDiscovery()

	// Default should be enabled
	if !ad.IsEnabled() {
		t.Errorf("expected enabled=true by default")
	}

	// Disable
	ad.SetEnabled(false)
	if ad.IsEnabled() {
		t.Errorf("expected enabled=false after SetEnabled(false)")
	}

	// Re-enable
	ad.SetEnabled(true)
	if !ad.IsEnabled() {
		t.Errorf("expected enabled=true after SetEnabled(true)")
	}
}

func TestClearCache(t *testing.T) {
	ad := NewAutoDiscovery()

	// Manually cache an endpoint
	endpoint := &DiscoveredEndpoint{
		URL:    "http://localhost:6820",
		Host:   "localhost",
		Port:   6820,
		Source: "test",
	}
	ad.cacheEndpoint(endpoint)

	// Verify it's cached
	if cached := ad.GetCachedEndpoint(); cached == nil {
		t.Errorf("expected cached endpoint, got nil")
	}

	// Clear cache
	ad.ClearCache()

	// Verify it's cleared
	if cached := ad.GetCachedEndpoint(); cached != nil {
		t.Errorf("expected nil after ClearCache, got %+v", cached)
	}
}

func TestCacheExpiry(t *testing.T) {
	cfg := AutoDiscoveryConfig{
		Enabled:       true,
		CacheDuration: 1 * time.Millisecond, // Very short for testing
	}

	ad := NewAutoDiscoveryWithConfig(cfg)

	// Cache an endpoint
	endpoint := &DiscoveredEndpoint{
		URL:    "http://localhost:6820",
		Host:   "localhost",
		Port:   6820,
		Source: "test",
	}
	ad.cacheEndpoint(endpoint)

	// Wait for cache to expire
	time.Sleep(5 * time.Millisecond)

	// Verify it's expired
	if cached := ad.GetCachedEndpoint(); cached != nil {
		t.Errorf("expected nil after cache expiry, got %+v", cached)
	}
}

func TestDiscoveredEndpoint(t *testing.T) {
	endpoint := DiscoveredEndpoint{
		URL:        "http://slurm-controller:6820",
		Host:       "slurm-controller",
		Port:       6820,
		Source:     "scontrol",
		Confidence: 0.9,
		Metadata: map[string]string{
			"role": "primary",
		},
	}

	if endpoint.URL != "http://slurm-controller:6820" {
		t.Errorf("unexpected URL: %s", endpoint.URL)
	}

	if endpoint.Host != "slurm-controller" {
		t.Errorf("unexpected Host: %s", endpoint.Host)
	}

	if endpoint.Port != 6820 {
		t.Errorf("unexpected Port: %d", endpoint.Port)
	}

	if endpoint.Source != "scontrol" {
		t.Errorf("unexpected Source: %s", endpoint.Source)
	}

	if endpoint.Confidence != 0.9 {
		t.Errorf("unexpected Confidence: %f", endpoint.Confidence)
	}

	if endpoint.Metadata["role"] != "primary" {
		t.Errorf("unexpected Metadata: %v", endpoint.Metadata)
	}
}

func TestDiscoveryResult(t *testing.T) {
	result := DiscoveryResult{
		Endpoint: &DiscoveredEndpoint{
			URL:  "http://localhost:6820",
			Host: "localhost",
			Port: 6820,
		},
		Source: "scontrol",
		Error:  nil,
	}

	if result.Endpoint == nil {
		t.Errorf("expected non-nil endpoint")
	}

	if result.Source != "scontrol" {
		t.Errorf("expected source 'scontrol', got %q", result.Source)
	}

	if result.Error != nil {
		t.Errorf("expected nil error, got %v", result.Error)
	}
}
