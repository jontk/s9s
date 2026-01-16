package observability

import (
	"context"
	"testing"
	"time"

	"github.com/jontk/s9s/plugins/observability/config"
)

func TestNew(t *testing.T) {
	plugin := New()
	if plugin == nil {
		t.Fatal("New() returned nil")
	}

	if plugin.config == nil {
		t.Error("Default config should be initialized")
	}
}

func TestParseConfig(t *testing.T) {
	plugin := New()

	tests := []struct {
		name     string
		config   map[string]interface{}
		wantErr  bool
		validate func(*config.Config) error
	}{
		{
			name: "valid prometheus endpoint",
			config: map[string]interface{}{
				"prometheus.endpoint": "http://localhost:9090",
				"prometheus.timeout":  "30s",
			},
			wantErr: false,
			validate: func(cfg *config.Config) error {
				if cfg.Prometheus.Endpoint != "http://localhost:9090" {
					t.Errorf("Expected endpoint http://localhost:9090, got %s", cfg.Prometheus.Endpoint)
				}
				if cfg.Prometheus.Timeout != 30*time.Second {
					t.Errorf("Expected timeout 30s, got %s", cfg.Prometheus.Timeout)
				}
				return nil
			},
		},
		{
			name: "auth configuration",
			config: map[string]interface{}{
				"prometheus.auth.type":     "basic",
				"prometheus.auth.username": "testuser",
				"prometheus.auth.password": "testpass",
			},
			wantErr: false,
			validate: func(cfg *config.Config) error {
				if cfg.Prometheus.Auth.Type != "basic" {
					t.Errorf("Expected auth type basic, got %s", cfg.Prometheus.Auth.Type)
				}
				if cfg.Prometheus.Auth.Username != "testuser" {
					t.Errorf("Expected username testuser, got %s", cfg.Prometheus.Auth.Username)
				}
				if cfg.Prometheus.Auth.Password != "testpass" {
					t.Errorf("Expected password testpass, got %s", cfg.Prometheus.Auth.Password)
				}
				return nil
			},
		},
		{
			name: "TLS configuration",
			config: map[string]interface{}{
				"prometheus.tls.enabled":            true,
				"prometheus.tls.insecureSkipVerify": false,
				"prometheus.tls.certFile":           "/path/to/cert.pem",
				"prometheus.tls.keyFile":            "/path/to/key.pem",
				"prometheus.tls.caFile":             "/path/to/ca.pem",
			},
			wantErr: true, // Expect error because TLS files don't exist
			validate: func(cfg *config.Config) error {
				if !cfg.Prometheus.TLS.Enabled {
					t.Error("Expected TLS enabled")
				}
				if cfg.Prometheus.TLS.InsecureSkipVerify {
					t.Error("Expected TLS verification enabled")
				}
				if cfg.Prometheus.TLS.CertFile != "/path/to/cert.pem" {
					t.Errorf("Expected cert file /path/to/cert.pem, got %s", cfg.Prometheus.TLS.CertFile)
				}
				return nil
			},
		},
		{
			name: "display configuration",
			config: map[string]interface{}{
				"display.refreshInterval":  "60s",
				"display.showOverlays":     false,
				"display.showSparklines":   false,
				"display.sparklinePoints":  50,
				"display.colorScheme":      "colorblind",
				"display.decimalPrecision": 2,
			},
			wantErr: false,
			validate: func(cfg *config.Config) error {
				if cfg.Display.RefreshInterval != 60*time.Second {
					t.Errorf("Expected refresh interval 60s, got %s", cfg.Display.RefreshInterval)
				}
				if cfg.Display.ShowOverlays {
					t.Error("Expected ShowOverlays false")
				}
				if cfg.Display.ColorScheme != "colorblind" {
					t.Errorf("Expected colorblind scheme, got %s", cfg.Display.ColorScheme)
				}
				return nil
			},
		},
		{
			name: "cache configuration",
			config: map[string]interface{}{
				"cache.enabled":         true,
				"cache.defaultTTL":      "120s",
				"cache.maxSize":         2000,
				"cache.cleanupInterval": "10m",
			},
			wantErr: false,
			validate: func(cfg *config.Config) error {
				if !cfg.Cache.Enabled {
					t.Error("Expected cache enabled")
				}
				if cfg.Cache.DefaultTTL != 120*time.Second {
					t.Errorf("Expected cache TTL 120s, got %s", cfg.Cache.DefaultTTL)
				}
				if cfg.Cache.MaxSize != 2000 {
					t.Errorf("Expected cache max size 2000, got %d", cfg.Cache.MaxSize)
				}
				return nil
			},
		},
		{
			name: "array configuration - comma separated",
			config: map[string]interface{}{
				"metrics.node.enabledMetrics": "cpu,memory,disk",
				"metrics.job.enabledMetrics":  "cpu,memory",
			},
			wantErr: false,
			validate: func(cfg *config.Config) error {
				expected := []string{"cpu", "memory", "disk"}
				if len(cfg.Metrics.Node.EnabledMetrics) != len(expected) {
					t.Errorf("Expected %d node metrics, got %d", len(expected), len(cfg.Metrics.Node.EnabledMetrics))
				}
				for i, metric := range expected {
					if i >= len(cfg.Metrics.Node.EnabledMetrics) || cfg.Metrics.Node.EnabledMetrics[i] != metric {
						t.Errorf("Expected node metric %s at index %d, got %v", metric, i, cfg.Metrics.Node.EnabledMetrics)
					}
				}
				return nil
			},
		},
		{
			name: "array configuration - slice",
			config: map[string]interface{}{
				"metrics.node.enabledMetrics": []interface{}{"cpu", "memory", "network"},
			},
			wantErr: false,
			validate: func(cfg *config.Config) error {
				expected := []string{"cpu", "memory", "network"}
				if len(cfg.Metrics.Node.EnabledMetrics) != len(expected) {
					t.Errorf("Expected %d node metrics, got %d", len(expected), len(cfg.Metrics.Node.EnabledMetrics))
				}
				for i, metric := range expected {
					if i >= len(cfg.Metrics.Node.EnabledMetrics) || cfg.Metrics.Node.EnabledMetrics[i] != metric {
						t.Errorf("Expected node metric %s at index %d", metric, i)
					}
				}
				return nil
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := plugin.Init(context.Background(), tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("Init() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && tt.validate != nil {
				plugin.config.MergeWithDefaults()
				if err := tt.validate(plugin.config); err != nil {
					t.Errorf("Validation failed: %v", err)
				}
			}
		})
	}
}

func TestInitialization(t *testing.T) {
	plugin := New()
	ctx := context.Background()

	// Test with minimal valid config
	config := map[string]interface{}{
		"prometheus.endpoint": "http://localhost:9090",
		"prometheus.timeout":  "10s",
	}

	err := plugin.Init(ctx, config)
	if err != nil {
		t.Fatalf("Init() failed: %v", err)
	}

	// Verify clients are created
	if plugin.components == nil {
		t.Fatal("Components should be initialized")
	}
	
	if plugin.components.Client == nil {
		t.Error("Prometheus client should be initialized")
	}

	if plugin.components.CachedClient == nil {
		t.Error("Cached Prometheus client should be initialized")
	}

	if plugin.components.OverlayMgr == nil {
		t.Error("Overlay manager should be initialized")
	}
}

func TestGetInfo(t *testing.T) {
	plugin := New()
	info := plugin.GetInfo()

	if info.Name != "observability" {
		t.Errorf("Expected name 'observability', got %s", info.Name)
	}

	if info.Version == "" {
		t.Error("Version should not be empty")
	}

	if len(info.Provides) == 0 {
		t.Error("Should provide at least one capability")
	}

	// Check for required config fields
	requiredFields := []string{
		"prometheus.endpoint",
		"prometheus.timeout",
		"display.refreshInterval",
		"display.showOverlays",
		"alerts.enabled",
	}

	for _, field := range requiredFields {
		if _, exists := info.ConfigSchema[field]; !exists {
			t.Errorf("Missing required config field: %s", field)
		}
	}
}

func TestGetViews(t *testing.T) {
	plugin := New()
	views := plugin.GetViews()

	if len(views) == 0 {
		t.Error("Should provide at least one view")
	}

	// Check for observability view
	found := false
	for _, view := range views {
		if view.ID == "observability" {
			found = true
			if view.Name == "" {
				t.Error("View name should not be empty")
			}
			if view.Description == "" {
				t.Error("View description should not be empty")
			}
			break
		}
	}

	if !found {
		t.Error("Should provide observability view")
	}
}

func TestGetOverlays(t *testing.T) {
	plugin := New()

	// Test with overlays enabled
	plugin.config = config.DefaultConfig()
	plugin.config.Display.ShowOverlays = true

	overlays := plugin.GetOverlays()
	if len(overlays) == 0 {
		t.Error("Should provide overlays when enabled")
	}

	// Check for expected overlays
	expectedOverlays := []string{"jobs-metrics", "nodes-metrics"}
	for _, expectedID := range expectedOverlays {
		found := false
		for _, overlay := range overlays {
			if overlay.ID == expectedID {
				found = true
				if overlay.Name == "" {
					t.Errorf("Overlay %s name should not be empty", expectedID)
				}
				if len(overlay.TargetViews) == 0 {
					t.Errorf("Overlay %s should have target views", expectedID)
				}
				break
			}
		}
		if !found {
			t.Errorf("Should provide overlay %s", expectedID)
		}
	}

	// Test with overlays disabled
	plugin.config.Display.ShowOverlays = false
	overlays = plugin.GetOverlays()
	if len(overlays) != 0 {
		t.Error("Should not provide overlays when disabled")
	}
}

func TestHealthCheck(t *testing.T) {
	plugin := New()

	// Test when not running
	health := plugin.Health()
	if health.Healthy {
		t.Error("Should not be healthy when not running")
	}
	if health.Status != "stopped" {
		t.Errorf("Expected status 'stopped', got %s", health.Status)
	}

	// Test when running but without proper initialization
	plugin.running = true
	if plugin.components != nil && plugin.components.Client != nil {
		// This would require a real Prometheus server to test properly
		// For now, we just verify the structure
		health = plugin.Health()
		if health.Message == "" {
			t.Error("Health message should not be empty")
		}
		if health.Details == nil {
			t.Error("Health details should not be nil")
		}
	}
}

func TestConfigurationMethods(t *testing.T) {
	plugin := New()
	plugin.config = config.DefaultConfig()

	// Test GetConfigSchema
	schema := plugin.GetConfigSchema()
	if len(schema) == 0 {
		t.Error("Config schema should not be empty")
	}

	// Test GetCurrentConfig
	currentConfig := plugin.GetCurrentConfig()
	if len(currentConfig) == 0 {
		t.Error("Current config should not be empty")
	}

	// Verify some expected fields
	expectedFields := []string{
		"prometheus.endpoint",
		"display.refreshInterval",
		"alerts.enabled",
		"cache.enabled",
	}

	for _, field := range expectedFields {
		if _, exists := currentConfig[field]; !exists {
			t.Errorf("Missing field in current config: %s", field)
		}
	}
}

func TestDataProviders(t *testing.T) {
	plugin := New()
	providers := plugin.GetDataProviders()

	if len(providers) == 0 {
		t.Error("Should provide at least one data provider")
	}

	// Check for expected providers
	expectedProviders := []string{"prometheus-metrics", "alerts"}
	for _, expectedID := range expectedProviders {
		found := false
		for _, provider := range providers {
			if provider.ID == expectedID {
				found = true
				if provider.Name == "" {
					t.Errorf("Provider %s name should not be empty", expectedID)
				}
				if provider.Description == "" {
					t.Errorf("Provider %s description should not be empty", expectedID)
				}
				break
			}
		}
		if !found {
			t.Errorf("Should provide data provider %s", expectedID)
		}
	}
}

// Integration test helpers
func createTestPlugin() *ObservabilityPlugin {
	plugin := New()
	config := map[string]interface{}{
		"prometheus.endpoint":     "http://localhost:9090",
		"prometheus.timeout":      "10s",
		"display.refreshInterval": "30s",
		"alerts.enabled":          true,
		"cache.enabled":           true,
	}

	ctx := context.Background()
	if err := plugin.Init(ctx, config); err != nil {
		panic(err)
	}

	return plugin
}

func TestIntegrationLifecycle(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	plugin := createTestPlugin()
	ctx := context.Background()

	// Start the plugin (this will fail without a real Prometheus server)
	// But we can test the startup logic
	err := plugin.Start(ctx)
	if err != nil {
		// Expected to fail without real Prometheus server
		t.Logf("Start failed as expected: %v", err)
	}

	// Test stopping
	err = plugin.Stop(ctx)
	if err != nil {
		t.Errorf("Stop failed: %v", err)
	}

	if plugin.running {
		t.Error("Plugin should not be running after stop")
	}
}