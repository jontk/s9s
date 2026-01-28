package config

import (
	"testing"
	"time"
)

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	if config == nil {
		t.Fatal("DefaultConfig() returned nil")
	}

	// Test Prometheus defaults
	if config.Prometheus.Endpoint != "http://localhost:9090" {
		t.Errorf("Expected default endpoint 'http://localhost:9090', got %s", config.Prometheus.Endpoint)
	}

	if config.Prometheus.Timeout != 10*time.Second {
		t.Errorf("Expected default timeout 10s, got %s", config.Prometheus.Timeout)
	}

	if config.Prometheus.Auth.Type != "none" {
		t.Errorf("Expected default auth type 'none', got %s", config.Prometheus.Auth.Type)
	}

	// Test Display defaults
	if config.Display.RefreshInterval != 30*time.Second {
		t.Errorf("Expected default refresh interval 30s, got %s", config.Display.RefreshInterval)
	}

	if !config.Display.ShowOverlays {
		t.Error("Expected default ShowOverlays true")
	}

	if config.Display.ColorScheme != "default" {
		t.Errorf("Expected default color scheme 'default', got %s", config.Display.ColorScheme)
	}

	// Test Alert defaults
	if !config.Alerts.Enabled {
		t.Error("Expected default alerts enabled")
	}

	if config.Alerts.CheckInterval != 1*time.Minute {
		t.Errorf("Expected default check interval 1m, got %s", config.Alerts.CheckInterval)
	}

	if len(config.Alerts.Rules) == 0 {
		t.Error("Expected default alert rules")
	}

	// Test Cache defaults
	if !config.Cache.Enabled {
		t.Error("Expected default cache enabled")
	}

	if config.Cache.DefaultTTL != 30*time.Second {
		t.Errorf("Expected default cache TTL 30s, got %s", config.Cache.DefaultTTL)
	}

	if config.Cache.MaxSize != 1000 {
		t.Errorf("Expected default cache max size 1000, got %d", config.Cache.MaxSize)
	}

	// Test Metrics defaults
	if config.Metrics.Node.NodeLabel != "instance" {
		t.Errorf("Expected default node label 'instance', got %s", config.Metrics.Node.NodeLabel)
	}

	if config.Metrics.Node.RateRange != "5m" {
		t.Errorf("Expected default rate range '5m', got %s", config.Metrics.Node.RateRange)
	}

	if !config.Metrics.Job.Enabled {
		t.Error("Expected default job metrics enabled")
	}

	expectedMetrics := []string{"cpu", "memory", "load", "disk", "network", "filesystem"}
	if len(config.Metrics.Node.EnabledMetrics) != len(expectedMetrics) {
		t.Errorf("Expected %d default node metrics, got %d", len(expectedMetrics), len(config.Metrics.Node.EnabledMetrics))
	}
}

func TestValidateConfig(t *testing.T) {
	tests := []struct {
		name    string
		config  func() *Config
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid default config",
			config:  DefaultConfig,
			wantErr: false,
		},
		{
			name: "empty endpoint",
			config: func() *Config {
				cfg := DefaultConfig()
				cfg.Prometheus.Endpoint = ""
				return cfg
			},
			wantErr: true,
			errMsg:  "prometheus endpoint is required",
		},
		{
			name: "invalid endpoint URL",
			config: func() *Config {
				cfg := DefaultConfig()
				cfg.Prometheus.Endpoint = "not-a-url"
				return cfg
			},
			wantErr: true,
		},
		{
			name: "invalid URL scheme",
			config: func() *Config {
				cfg := DefaultConfig()
				cfg.Prometheus.Endpoint = "ftp://localhost:9090"
				return cfg
			},
			wantErr: true,
			errMsg:  "prometheus endpoint must use http or https scheme",
		},
		{
			name: "zero timeout",
			config: func() *Config {
				cfg := DefaultConfig()
				cfg.Prometheus.Timeout = 0
				return cfg
			},
			wantErr: true,
			errMsg:  "prometheus timeout must be positive",
		},
		{
			name: "basic auth missing username",
			config: func() *Config {
				cfg := DefaultConfig()
				cfg.Prometheus.Auth.Type = "basic"
				cfg.Prometheus.Auth.Password = "pass"
				return cfg
			},
			wantErr: true,
			errMsg:  "basic auth requires username and either password or passwordSecretRef",
		},
		{
			name: "basic auth missing password",
			config: func() *Config {
				cfg := DefaultConfig()
				cfg.Prometheus.Auth.Type = "basic"
				cfg.Prometheus.Auth.Username = "user"
				return cfg
			},
			wantErr: true,
			errMsg:  "basic auth requires username and either password or passwordSecretRef",
		},
		{
			name: "bearer auth missing token",
			config: func() *Config {
				cfg := DefaultConfig()
				cfg.Prometheus.Auth.Type = "bearer"
				return cfg
			},
			wantErr: true,
			errMsg:  "bearer auth requires either token or tokenSecretRef",
		},
		{
			name: "invalid auth type",
			config: func() *Config {
				cfg := DefaultConfig()
				cfg.Prometheus.Auth.Type = "invalid"
				return cfg
			},
			wantErr: true,
			errMsg:  "invalid auth type: invalid",
		},
		{
			name: "TLS cert without key",
			config: func() *Config {
				cfg := DefaultConfig()
				cfg.Prometheus.TLS.Enabled = true
				cfg.Prometheus.TLS.CertFile = "/path/to/cert.pem"
				return cfg
			},
			wantErr: true,
			errMsg:  "TLS cert file requires key file",
		},
		{
			name: "TLS key without cert",
			config: func() *Config {
				cfg := DefaultConfig()
				cfg.Prometheus.TLS.Enabled = true
				cfg.Prometheus.TLS.KeyFile = "/path/to/key.pem"
				return cfg
			},
			wantErr: true,
			errMsg:  "TLS key file requires cert file",
		},
		{
			name: "negative max retries",
			config: func() *Config {
				cfg := DefaultConfig()
				cfg.Prometheus.Retry.MaxRetries = -1
				return cfg
			},
			wantErr: true,
			errMsg:  "max retries must be non-negative",
		},
		{
			name: "zero multiplier",
			config: func() *Config {
				cfg := DefaultConfig()
				cfg.Prometheus.Retry.Multiplier = 0
				return cfg
			},
			wantErr: true,
			errMsg:  "retry multiplier must be positive",
		},
		{
			name: "zero refresh interval",
			config: func() *Config {
				cfg := DefaultConfig()
				cfg.Display.RefreshInterval = 0
				return cfg
			},
			wantErr: true,
			errMsg:  "refresh interval must be positive",
		},
		{
			name: "negative sparkline points",
			config: func() *Config {
				cfg := DefaultConfig()
				cfg.Display.SparklinePoints = -1
				return cfg
			},
			wantErr: true,
			errMsg:  "sparkline points must be non-negative",
		},
		{
			name: "invalid color scheme",
			config: func() *Config {
				cfg := DefaultConfig()
				cfg.Display.ColorScheme = "invalid"
				return cfg
			},
			wantErr: true,
			errMsg:  "invalid color scheme: invalid",
		},
		{
			name: "zero alert check interval",
			config: func() *Config {
				cfg := DefaultConfig()
				cfg.Alerts.Enabled = true
				cfg.Alerts.CheckInterval = 0
				return cfg
			},
			wantErr: true,
			errMsg:  "alert check interval must be positive",
		},
		{
			name: "alert rule missing name",
			config: func() *Config {
				cfg := DefaultConfig()
				cfg.Alerts.Rules = []AlertRule{
					{
						Metric:    "cpu_usage",
						Operator:  ">",
						Threshold: 90,
						Duration:  time.Minute,
						Severity:  "warning",
					},
				}
				return cfg
			},
			wantErr: true,
			errMsg:  "alert rule 0: name is required",
		},
		{
			name: "alert rule invalid operator",
			config: func() *Config {
				cfg := DefaultConfig()
				cfg.Alerts.Rules = []AlertRule{
					{
						Name:      "Test Rule",
						Metric:    "cpu_usage",
						Operator:  "invalid",
						Threshold: 90,
						Duration:  time.Minute,
						Severity:  "warning",
					},
				}
				return cfg
			},
			wantErr: true,
		},
		{
			name: "alert rule invalid severity",
			config: func() *Config {
				cfg := DefaultConfig()
				cfg.Alerts.Rules = []AlertRule{
					{
						Name:      "Test Rule",
						Metric:    "cpu_usage",
						Operator:  ">",
						Threshold: 90,
						Duration:  time.Minute,
						Severity:  "invalid",
					},
				}
				return cfg
			},
			wantErr: true,
		},
		{
			name: "zero cache TTL",
			config: func() *Config {
				cfg := DefaultConfig()
				cfg.Cache.Enabled = true
				cfg.Cache.DefaultTTL = 0
				return cfg
			},
			wantErr: true,
			errMsg:  "cache default TTL must be positive",
		},
		{
			name: "empty node label",
			config: func() *Config {
				cfg := DefaultConfig()
				cfg.Metrics.Node.NodeLabel = ""
				return cfg
			},
			wantErr: true,
			errMsg:  "node label is required",
		},
		{
			name: "empty rate range",
			config: func() *Config {
				cfg := DefaultConfig()
				cfg.Metrics.Node.RateRange = ""
				return cfg
			},
			wantErr: true,
			errMsg:  "node rate range is required",
		},
		{
			name: "job metrics enabled without cgroup pattern",
			config: func() *Config {
				cfg := DefaultConfig()
				cfg.Metrics.Job.Enabled = true
				cfg.Metrics.Job.CgroupPattern = ""
				return cfg
			},
			wantErr: true,
			errMsg:  "job cgroup pattern is required when job metrics are enabled",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := tt.config()
			err := config.Validate()

			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && tt.errMsg != "" {
				if err == nil || err.Error() != tt.errMsg {
					t.Errorf("Expected error message '%s', got '%v'", tt.errMsg, err)
				}
			}
		})
	}
}

func TestMergeWithDefaults(t *testing.T) {
	// Create a minimal config
	config := &Config{
		Prometheus: PrometheusConfig{
			Endpoint: "http://custom:9090",
			// Other fields will be defaults
		},
		Display: DisplayConfig{
			RefreshInterval: 60 * time.Second,
			// Other fields will be defaults
		},
	}

	config.MergeWithDefaults()

	// Check that custom values are preserved
	if config.Prometheus.Endpoint != "http://custom:9090" {
		t.Error("Custom endpoint was overridden")
	}

	if config.Display.RefreshInterval != 60*time.Second {
		t.Error("Custom refresh interval was overridden")
	}

	// Check that defaults were applied
	if config.Prometheus.Timeout == 0 {
		t.Error("Default timeout was not applied")
	}

	if config.Prometheus.Auth.Type != "none" {
		t.Error("Default auth type was not applied")
	}

	if config.Display.ShowOverlays == false && config.Display.ColorScheme == "" {
		t.Error("Default display settings were not applied")
	}

	if config.Cache.DefaultTTL == 0 {
		t.Error("Default cache TTL was not applied")
	}
}

func TestAlertRuleValidation(t *testing.T) {
	validOperators := []string{">", "<", ">=", "<=", "==", "!="}
	validSeverities := []string{"info", "warning", "critical"}

	for _, op := range validOperators {
		rule := AlertRule{
			Name:      "Test Rule",
			Metric:    "test_metric",
			Operator:  op,
			Threshold: 50,
			Duration:  time.Minute,
			Severity:  "warning",
			Enabled:   true,
		}

		config := &Config{
			Prometheus: PrometheusConfig{
				Endpoint: "http://localhost:9090",
				Timeout:  time.Second,
				Auth:     AuthConfig{Type: "none"},
				Retry: RetryConfig{
					MaxRetries:   3,
					InitialDelay: time.Second,
					MaxDelay:     10 * time.Second,
					Multiplier:   2.0,
				},
			},
			Display: DisplayConfig{RefreshInterval: time.Minute, ColorScheme: "default"},
			Alerts:  AlertConfig{Enabled: true, CheckInterval: time.Minute, Rules: []AlertRule{rule}},
			Cache:   CacheConfig{Enabled: true, DefaultTTL: time.Minute, MaxSize: 100, CleanupInterval: time.Minute},
			Metrics: MetricsConfig{Node: NodeMetricsConfig{NodeLabel: "instance", RateRange: "5m"}},
		}

		if err := config.Validate(); err != nil {
			t.Errorf("Valid operator %s was rejected: %v", op, err)
		}
	}

	for _, severity := range validSeverities {
		rule := AlertRule{
			Name:      "Test Rule",
			Metric:    "test_metric",
			Operator:  ">",
			Threshold: 50,
			Duration:  time.Minute,
			Severity:  severity,
			Enabled:   true,
		}

		config := &Config{
			Prometheus: PrometheusConfig{
				Endpoint: "http://localhost:9090",
				Timeout:  time.Second,
				Auth:     AuthConfig{Type: "none"},
				Retry: RetryConfig{
					MaxRetries:   3,
					InitialDelay: time.Second,
					MaxDelay:     10 * time.Second,
					Multiplier:   2.0,
				},
			},
			Display: DisplayConfig{RefreshInterval: time.Minute, ColorScheme: "default"},
			Alerts:  AlertConfig{Enabled: true, CheckInterval: time.Minute, Rules: []AlertRule{rule}},
			Cache:   CacheConfig{Enabled: true, DefaultTTL: time.Minute, MaxSize: 100, CleanupInterval: time.Minute},
			Metrics: MetricsConfig{Node: NodeMetricsConfig{NodeLabel: "instance", RateRange: "5m"}},
		}

		if err := config.Validate(); err != nil {
			t.Errorf("Valid severity %s was rejected: %v", severity, err)
		}
	}
}

func TestColorSchemeValidation(t *testing.T) {
	validSchemes := []string{"default", "colorblind", "monochrome"}

	for _, scheme := range validSchemes {
		config := DefaultConfig()
		config.Display.ColorScheme = scheme

		if err := config.Validate(); err != nil {
			t.Errorf("Valid color scheme %s was rejected: %v", scheme, err)
		}
	}
}

func TestAuthConfigValidation(t *testing.T) {
	// Test valid basic auth
	config := DefaultConfig()
	config.Prometheus.Auth.Type = "basic"
	config.Prometheus.Auth.Username = "testuser"
	config.Prometheus.Auth.Password = "testpass"

	if err := config.Validate(); err != nil {
		t.Errorf("Valid basic auth was rejected: %v", err)
	}

	// Test valid bearer auth
	config = DefaultConfig()
	config.Prometheus.Auth.Type = "bearer"
	config.Prometheus.Auth.Token = "testtoken"

	if err := config.Validate(); err != nil {
		t.Errorf("Valid bearer auth was rejected: %v", err)
	}

	// Test valid none auth
	config = DefaultConfig()
	config.Prometheus.Auth.Type = "none"

	if err := config.Validate(); err != nil {
		t.Errorf("Valid none auth was rejected: %v", err)
	}
}

func TestTLSConfigValidation(t *testing.T) {
	// Test valid TLS config with client certs
	config := DefaultConfig()
	config.Prometheus.TLS.Enabled = true
	config.Prometheus.TLS.CertFile = "/path/to/cert.pem"
	config.Prometheus.TLS.KeyFile = "/path/to/key.pem"
	config.Prometheus.TLS.CAFile = "/path/to/ca.pem"

	if err := config.Validate(); err != nil {
		t.Errorf("Valid TLS config was rejected: %v", err)
	}

	// Test valid TLS config without client certs
	config = DefaultConfig()
	config.Prometheus.TLS.Enabled = true
	config.Prometheus.TLS.InsecureSkipVerify = true

	if err := config.Validate(); err != nil {
		t.Errorf("Valid TLS config without client certs was rejected: %v", err)
	}
}

func TestRateRangeValidation(t *testing.T) {
	validRanges := []string{"1m", "5m", "1h", "24h"}

	for _, rateRange := range validRanges {
		config := DefaultConfig()
		config.Metrics.Node.RateRange = rateRange

		if err := config.Validate(); err != nil {
			t.Errorf("Valid rate range %s was rejected: %v", rateRange, err)
		}
	}

	// Test invalid rate range
	config := DefaultConfig()
	config.Metrics.Node.RateRange = "x"

	if err := config.Validate(); err == nil {
		t.Error("Invalid rate range was accepted")
	}
}
