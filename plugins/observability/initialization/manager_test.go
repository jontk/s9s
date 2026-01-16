package initialization

import (
	"testing"
	"time"

	"github.com/jontk/s9s/plugins/observability/config"
	"github.com/jontk/s9s/plugins/observability/security"
)

func TestManagerBasicInitialization(t *testing.T) {
	cfg := config.DefaultConfig()
	// Override security config for testing
	cfg.Security.Secrets.StorageDir = t.TempDir()
	cfg.Security.Secrets.EncryptAtRest = false
	cfg.Security.Secrets.RequireEncryption = false
	manager := NewManager(cfg)

	components, err := manager.InitializeComponents()
	if err != nil {
		t.Fatalf("InitializeComponents failed: %v", err)
	}

	// Check that all components were initialized
	if components.Client == nil {
		t.Error("Expected Client to be initialized")
	}
	if components.CachedClient == nil {
		t.Error("Expected CachedClient to be initialized")
	}
	if components.OverlayMgr == nil {
		t.Error("Expected OverlayMgr to be initialized")
	}
	if components.SubscriptionMgr == nil {
		t.Error("Expected SubscriptionMgr to be initialized")
	}
	if components.NotificationMgr == nil {
		t.Error("Expected NotificationMgr to be initialized")
	}
	if components.Persistence == nil {
		t.Error("Expected Persistence to be initialized")
	}
	if components.HistoricalCollector == nil {
		t.Error("Expected HistoricalCollector to be initialized")
	}
	if components.HistoricalAnalyzer == nil {
		t.Error("Expected HistoricalAnalyzer to be initialized")
	}
	if components.EfficiencyAnalyzer == nil {
		t.Error("Expected EfficiencyAnalyzer to be initialized")
	}
	if components.ExternalAPI == nil {
		t.Error("Expected ExternalAPI to be initialized")
	}
}

func TestManagerInvalidPrometheusConfig(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Prometheus.Endpoint = "://invalid-url-format"  // This will fail url.Parse
	
	manager := NewManager(cfg)
	_, err := manager.InitializeComponents()
	if err == nil {
		t.Error("Expected error with invalid Prometheus endpoint")
	}
}

func TestManagerEmptyPrometheusEndpoint(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Prometheus.Endpoint = ""
	
	manager := NewManager(cfg)
	_, err := manager.InitializeComponents()
	if err == nil {
		t.Error("Expected error with empty Prometheus endpoint")
	}
}

func TestManagerZeroTimeout(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Prometheus.Timeout = 0
	
	manager := NewManager(cfg)
	components, err := manager.InitializeComponents()
	if err != nil {
		t.Fatalf("InitializeComponents should handle zero timeout: %v", err)
	}
	
	// Should still initialize components
	if components.Client == nil {
		t.Error("Expected Client to be initialized even with zero timeout")
	}
}

func TestManagerInvalidCacheConfig(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Cache.MaxSize = 0
	cfg.Cache.DefaultTTL = 0
	
	manager := NewManager(cfg)
	components, err := manager.InitializeComponents()
	if err != nil {
		t.Fatalf("InitializeComponents should handle invalid cache config: %v", err)
	}
	
	// Should still initialize with defaults
	if components.CachedClient == nil {
		t.Error("Expected CachedClient to be initialized with default values")
	}
}

func TestManagerNilConfig(t *testing.T) {
	manager := NewManager(nil)
	_, err := manager.InitializeComponents()
	if err == nil {
		t.Error("Expected error with nil config")
	}
}

func TestManagerInitializationOrder(t *testing.T) {
	cfg := config.DefaultConfig()
	manager := NewManager(cfg)

	// Test individual initialization methods
	components := &Components{}

	// Client must be initialized first
	err := manager.initPrometheusClient(components)
	if err != nil {
		t.Fatalf("initPrometheusClient failed: %v", err)
	}

	// Cache depends on client
	err = manager.initCaching(components)
	if err != nil {
		t.Fatalf("initCaching failed: %v", err)
	}

	// Overlays depend on cached client
	err = manager.initOverlays(components)
	if err != nil {
		t.Fatalf("initOverlays failed: %v", err)
	}

	// Subscriptions depend on cached client
	err = manager.initSubscriptions(components)
	if err != nil {
		t.Fatalf("initSubscriptions failed: %v", err)
	}

	// Historical data depends on cached client
	err = manager.initHistoricalData(components)
	if err != nil {
		t.Fatalf("initHistoricalData failed: %v", err)
	}

	// Analysis depends on historical components
	err = manager.initAnalysis(components)
	if err != nil {
		t.Fatalf("initAnalysis failed: %v", err)
	}

	// API depends on all other components
	err = manager.initExternalAPI(components)
	if err != nil {
		t.Fatalf("initExternalAPI failed: %v", err)
	}
}

func TestManagerInitCachingWithoutClient(t *testing.T) {
	cfg := config.DefaultConfig()
	manager := NewManager(cfg)
	components := &Components{} // No client initialized

	err := manager.initCaching(components)
	if err == nil {
		t.Error("Expected error when initializing cache without client")
	}
	if err != nil && err.Error() != "Prometheus client not initialized" {
		t.Errorf("Expected 'Prometheus client not initialized' error, got: %s", err.Error())
	}
}

func TestManagerInitOverlaysWithoutCachedClient(t *testing.T) {
	cfg := config.DefaultConfig()
	manager := NewManager(cfg)
	components := &Components{} // No cached client initialized

	err := manager.initOverlays(components)
	if err == nil {
		t.Error("Expected error when initializing overlays without cached client")
	}
}

func TestManagerInitSubscriptionsWithoutCachedClient(t *testing.T) {
	cfg := config.DefaultConfig()
	manager := NewManager(cfg)
	components := &Components{} // No cached client initialized

	err := manager.initSubscriptions(components)
	if err == nil {
		t.Error("Expected error when initializing subscriptions without cached client")
	}
}

func TestManagerInitHistoricalDataWithoutCachedClient(t *testing.T) {
	cfg := config.DefaultConfig()
	manager := NewManager(cfg)
	components := &Components{} // No cached client initialized

	err := manager.initHistoricalData(components)
	if err == nil {
		t.Error("Expected error when initializing historical data without cached client")
	}
}

func TestManagerInitAnalysisWithoutCachedClient(t *testing.T) {
	cfg := config.DefaultConfig()
	manager := NewManager(cfg)
	components := &Components{} // No cached client initialized

	err := manager.initAnalysis(components)
	if err == nil {
		t.Error("Expected error when initializing analysis without cached client")
	}
}

func TestManagerInitExternalAPIWithoutCachedClient(t *testing.T) {
	cfg := config.DefaultConfig()
	manager := NewManager(cfg)
	components := &Components{} // No cached client initialized

	err := manager.initExternalAPI(components)
	if err == nil {
		t.Error("Expected error when initializing external API without cached client")
	}
}

func TestComponentsStopMethod(t *testing.T) {
	cfg := config.DefaultConfig()
	manager := NewManager(cfg)

	components, err := manager.InitializeComponents()
	if err != nil {
		t.Fatalf("InitializeComponents failed: %v", err)
	}

	// Should not panic, errors are expected since some components may not be running
	err = components.Stop()
	// We don't check for error since some components may report they're not running
	_ = err
}

func TestComponentsStopWithNilComponents(t *testing.T) {
	components := &Components{} // All fields nil

	// Should not panic
	err := components.Stop()
	if err != nil {
		t.Errorf("Components.Stop() with nil components failed: %v", err)
	}
}

func TestManagerWithCustomConfig(t *testing.T) {
	cfg := &config.Config{
		Prometheus: config.PrometheusConfig{
			Endpoint: "http://custom-prometheus:9090",
			Timeout:  5 * time.Second,
			Auth: config.AuthConfig{
				Type:     "bearer",
				Token:    "test-token",
			},
		},
		Cache: config.CacheConfig{
			Enabled:    true,
			DefaultTTL: 15 * time.Second,
			MaxSize:    2000,
		},
		Display: config.DisplayConfig{
			RefreshInterval: 45 * time.Second,
		},
		Security: config.SecurityConfig{
			Secrets: security.SecretConfig{
				StorageDir:         t.TempDir(),
				EncryptAtRest:      false, // Disable encryption for test
				MasterKeySource:    security.SecretSourceEnvironment,
				MasterKeyEnv:       "TEST_MASTER_KEY",
				EnableRotation:     false,
				RequireEncryption:  false,
				AllowInlineSecrets: true,
			},
		},
	}

	manager := NewManager(cfg)
	components, err := manager.InitializeComponents()
	if err != nil {
		t.Fatalf("InitializeComponents with custom config failed: %v", err)
	}

	// Verify components are initialized
	if components.Client == nil {
		t.Error("Expected Client to be initialized with custom config")
	}
	if components.CachedClient == nil {
		t.Error("Expected CachedClient to be initialized with custom config")
	}
}

func TestManagerInitializationErrors(t *testing.T) {
	tests := []struct {
		name   string
		config *config.Config
	}{
		{
			name: "invalid prometheus scheme",
			config: &config.Config{
				Prometheus: config.PrometheusConfig{
					Endpoint: " ://invalid-url",  // Invalid URL with space
					Timeout:  10 * time.Second,
				},
				Cache: config.CacheConfig{
					Enabled:    true,
					DefaultTTL: 30 * time.Second,
					MaxSize:    1000,
				},
				Display: config.DisplayConfig{
					RefreshInterval: 30 * time.Second,
				},
			},
		},
		{
			name: "malformed URL",
			config: &config.Config{
				Prometheus: config.PrometheusConfig{
					Endpoint: "://malformed",
					Timeout:  10 * time.Second,
				},
				Cache: config.CacheConfig{
					Enabled:    true,
					DefaultTTL: 30 * time.Second,
					MaxSize:    1000,
				},
				Display: config.DisplayConfig{
					RefreshInterval: 30 * time.Second,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager := NewManager(tt.config)
			_, err := manager.InitializeComponents()
			if err == nil {
				t.Errorf("Expected error for test case: %s", tt.name)
			}
		})
	}
}