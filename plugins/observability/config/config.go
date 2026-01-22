// Package config provides configuration management for the observability plugin.
// It handles parsing, validation, and merging of configuration from various sources
// including YAML files, environment variables, and command-line arguments.
// The package supports comprehensive configuration for Prometheus, caching, security,
// alerts, and display preferences.
package config

import (
	"fmt"
	"net/url"
	"time"

	"github.com/jontk/s9s/plugins/observability/api"
	"github.com/jontk/s9s/plugins/observability/security"
)

// Config represents the observability plugin configuration
type Config struct {
	// Prometheus configuration
	Prometheus PrometheusConfig `yaml:"prometheus" json:"prometheus"`

	// Display preferences
	Display DisplayConfig `yaml:"display" json:"display"`

	// Alert configuration
	Alerts AlertConfig `yaml:"alerts" json:"alerts"`

	// Cache configuration
	Cache CacheConfig `yaml:"cache" json:"cache"`

	// Metrics configuration
	Metrics MetricsConfig `yaml:"metrics" json:"metrics"`

	// Security configuration
	Security SecurityConfig `yaml:"security" json:"security"`

	// External API configuration
	ExternalAPI api.Config `yaml:"externalAPI" json:"externalAPI"`

	// Logging configuration
	Logging LoggingConfig `yaml:"logging" json:"logging"`
}

// PrometheusConfig contains Prometheus connection settings
type PrometheusConfig struct {
	// Endpoint is the Prometheus server URL
	Endpoint string `yaml:"endpoint" json:"endpoint"`

	// Timeout for Prometheus queries
	Timeout time.Duration `yaml:"timeout" json:"timeout"`

	// Authentication settings
	Auth AuthConfig `yaml:"auth" json:"auth"`

	// TLS configuration
	TLS TLSConfig `yaml:"tls" json:"tls"`

	// Retry configuration
	Retry RetryConfig `yaml:"retry" json:"retry"`
}

// AuthConfig contains authentication settings
type AuthConfig struct {
	// Type of authentication: "none", "basic", "bearer"
	Type string `yaml:"type" json:"type"`

	// Username for basic auth
	Username string `yaml:"username,omitempty" json:"username,omitempty"`

	// Password for basic auth
	Password string `yaml:"password,omitempty" json:"password,omitempty"`

	// Token for bearer auth
	Token string `yaml:"token,omitempty" json:"token,omitempty"`

	// Secret references for secure token storage
	TokenSecretRef    string `yaml:"tokenSecretRef,omitempty" json:"tokenSecretRef,omitempty"`
	PasswordSecretRef string `yaml:"passwordSecretRef,omitempty" json:"passwordSecretRef,omitempty"`
}

// TLSConfig contains TLS settings
type TLSConfig struct {
	// Enable TLS
	Enabled bool `yaml:"enabled" json:"enabled"`

	// Skip certificate verification (insecure)
	InsecureSkipVerify bool `yaml:"insecureSkipVerify" json:"insecureSkipVerify"`

	// CA certificate file path
	CAFile string `yaml:"caFile,omitempty" json:"caFile,omitempty"`

	// Client certificate file path
	CertFile string `yaml:"certFile,omitempty" json:"certFile,omitempty"`

	// Client key file path
	KeyFile string `yaml:"keyFile,omitempty" json:"keyFile,omitempty"`
}

// RetryConfig contains retry settings
type RetryConfig struct {
	// Maximum number of retries
	MaxRetries int `yaml:"maxRetries" json:"maxRetries"`

	// Initial retry delay
	InitialDelay time.Duration `yaml:"initialDelay" json:"initialDelay"`

	// Maximum retry delay
	MaxDelay time.Duration `yaml:"maxDelay" json:"maxDelay"`

	// Backoff multiplier
	Multiplier float64 `yaml:"multiplier" json:"multiplier"`
}

// DisplayConfig contains display preferences
type DisplayConfig struct {
	// Refresh interval for metrics
	RefreshInterval time.Duration `yaml:"refreshInterval" json:"refreshInterval"`

	// Show metric overlays on existing views
	ShowOverlays bool `yaml:"showOverlays" json:"showOverlays"`

	// Enable sparklines in tables
	ShowSparklines bool `yaml:"showSparklines" json:"showSparklines"`

	// Number of historical points for sparklines
	SparklinePoints int `yaml:"sparklinePoints" json:"sparklinePoints"`

	// Color scheme: "default", "colorblind", "monochrome"
	ColorScheme string `yaml:"colorScheme" json:"colorScheme"`

	// Decimal precision for values
	DecimalPrecision int `yaml:"decimalPrecision" json:"decimalPrecision"`
}

// AlertConfig contains alert settings
type AlertConfig struct {
	// Enable alerting
	Enabled bool `yaml:"enabled" json:"enabled"`

	// Check interval for alerts
	CheckInterval time.Duration `yaml:"checkInterval" json:"checkInterval"`

	// Load predefined alert rules
	LoadPredefinedRules bool `yaml:"loadPredefinedRules" json:"loadPredefinedRules"`

	// Show notifications for new alerts
	ShowNotifications bool `yaml:"showNotifications" json:"showNotifications"`

	// Default alert rules
	Rules []AlertRule `yaml:"rules" json:"rules"`

	// Alert history retention
	HistoryRetention time.Duration `yaml:"historyRetention" json:"historyRetention"`
}

// AlertRule defines an alert condition
type AlertRule struct {
	// Rule name
	Name string `yaml:"name" json:"name"`

	// Metric to monitor
	Metric string `yaml:"metric" json:"metric"`

	// Comparison operator: ">", "<", ">=", "<=", "==", "!="
	Operator string `yaml:"operator" json:"operator"`

	// Threshold value
	Threshold float64 `yaml:"threshold" json:"threshold"`

	// Duration before triggering
	Duration time.Duration `yaml:"duration" json:"duration"`

	// Severity: "info", "warning", "critical"
	Severity string `yaml:"severity" json:"severity"`

	// Rule enabled
	Enabled bool `yaml:"enabled" json:"enabled"`
}

// CacheConfig contains cache settings
type CacheConfig struct {
	// Enable caching
	Enabled bool `yaml:"enabled" json:"enabled"`

	// Default TTL for cached metrics
	DefaultTTL time.Duration `yaml:"defaultTTL" json:"defaultTTL"`

	// Maximum cache size in entries
	MaxSize int `yaml:"maxSize" json:"maxSize"`

	// Cleanup interval
	CleanupInterval time.Duration `yaml:"cleanupInterval" json:"cleanupInterval"`
}

// MetricsConfig contains metric collection settings
type MetricsConfig struct {
	// Node metrics configuration
	Node NodeMetricsConfig `yaml:"node" json:"node"`

	// Job metrics configuration
	Job JobMetricsConfig `yaml:"job" json:"job"`

	// Custom queries
	CustomQueries map[string]string `yaml:"customQueries" json:"customQueries"`
}

// NodeMetricsConfig contains node-specific metric settings
type NodeMetricsConfig struct {
	// Label used to identify nodes in Prometheus
	NodeLabel string `yaml:"nodeLabel" json:"nodeLabel"`

	// Metrics to collect
	EnabledMetrics []string `yaml:"enabledMetrics" json:"enabledMetrics"`

	// Query range for rate calculations
	RateRange string `yaml:"rateRange" json:"rateRange"`
}

// JobMetricsConfig contains job-specific metric settings
type JobMetricsConfig struct {
	// Enable job metrics collection
	Enabled bool `yaml:"enabled" json:"enabled"`

	// cgroup path pattern
	CgroupPattern string `yaml:"cgroupPattern" json:"cgroupPattern"`

	// Metrics to collect
	EnabledMetrics []string `yaml:"enabledMetrics" json:"enabledMetrics"`
}

// SecurityConfig contains security settings
type SecurityConfig struct {
	// Secrets management configuration
	Secrets security.SecretConfig `yaml:"secrets" json:"secrets"`

	// API security settings
	API APISecurityConfig `yaml:"api" json:"api"`
}

// APISecurityConfig contains API security settings
type APISecurityConfig struct {
	// Enable API authentication
	EnableAuth bool `yaml:"enableAuth" json:"enableAuth"`

	// API token (can reference a secret)
	AuthToken string `yaml:"authToken,omitempty" json:"authToken,omitempty"`

	// Reference to secret containing auth token
	AuthTokenSecretRef string `yaml:"authTokenSecretRef,omitempty" json:"authTokenSecretRef,omitempty"`

	// Rate limiting configuration
	RateLimit security.RateLimitConfig `yaml:"rateLimit" json:"rateLimit"`

	// Request validation configuration
	Validation security.ValidationConfig `yaml:"validation" json:"validation"`

	// Audit logging configuration
	Audit security.AuditConfig `yaml:"audit" json:"audit"`
}

// LoggingConfig contains logging settings
type LoggingConfig struct {
	// Enable debug logging
	Enabled bool `yaml:"enabled" json:"enabled"`

	// Log level: "DEBUG", "INFO", "WARN", "ERROR"
	Level string `yaml:"level" json:"level"`

	// Log file path
	LogFile string `yaml:"logFile" json:"logFile"`

	// Component name for logging
	Component string `yaml:"component" json:"component"`

	// Log to console as well as file
	LogToConsole bool `yaml:"logToConsole" json:"logToConsole"`
}

// DefaultConfig returns the default configuration
func DefaultConfig() *Config {
	return &Config{
		Prometheus: PrometheusConfig{
			Endpoint: "http://localhost:9090",
			Timeout:  10 * time.Second,
			Auth: AuthConfig{
				Type: "none",
			},
			TLS: TLSConfig{
				Enabled:            false,
				InsecureSkipVerify: false,
			},
			Retry: RetryConfig{
				MaxRetries:   3,
				InitialDelay: 1 * time.Second,
				MaxDelay:     10 * time.Second,
				Multiplier:   2.0,
			},
		},
		Display: DisplayConfig{
			RefreshInterval:  30 * time.Second,
			ShowOverlays:     true,
			ShowSparklines:   true,
			SparklinePoints:  20,
			ColorScheme:      "default",
			DecimalPrecision: 1,
		},
		Alerts: AlertConfig{
			Enabled:             true,
			CheckInterval:       1 * time.Minute,
			LoadPredefinedRules: true,
			ShowNotifications:   true,
			HistoryRetention:    24 * time.Hour,
			Rules: []AlertRule{
				{
					Name:      "High CPU Usage",
					Metric:    "cpu_usage",
					Operator:  ">",
					Threshold: 90.0,
					Duration:  5 * time.Minute,
					Severity:  "warning",
					Enabled:   true,
				},
				{
					Name:      "High Memory Usage",
					Metric:    "memory_usage",
					Operator:  ">",
					Threshold: 95.0,
					Duration:  5 * time.Minute,
					Severity:  "critical",
					Enabled:   true,
				},
			},
		},
		Cache: CacheConfig{
			Enabled:         true,
			DefaultTTL:      30 * time.Second,
			MaxSize:         1000,
			CleanupInterval: 5 * time.Minute,
		},
		Metrics: MetricsConfig{
			Node: NodeMetricsConfig{
				NodeLabel: "instance",
				EnabledMetrics: []string{
					"cpu", "memory", "load", "disk", "network", "filesystem",
				},
				RateRange: "5m",
			},
			Job: JobMetricsConfig{
				Enabled:       true,
				CgroupPattern: "/slurm/uid_.*/job_%s",
				EnabledMetrics: []string{
					"cpu", "memory", "throttle",
				},
			},
			CustomQueries: make(map[string]string),
		},
		Security: SecurityConfig{
			Secrets: security.SecretConfig{
				StorageDir:         "./data/secrets",
				EncryptAtRest:      true,
				MasterKeySource:    security.SecretSourceEnvironment,
				MasterKeyEnv:       "OBSERVABILITY_MASTER_KEY",
				EnableRotation:     true,
				RotationInterval:   24 * time.Hour,
				RequireEncryption:  true,
				AllowInlineSecrets: false, // Don't allow inline secrets by default
			},
			API: APISecurityConfig{
				EnableAuth: false,
				RateLimit:  security.DefaultRateLimitConfig(),
				Validation: security.DefaultValidationConfig(),
				Audit:      security.DefaultAuditConfig(),
			},
		},
		ExternalAPI: api.DefaultConfig(),
		Logging: LoggingConfig{
			Enabled:      true,
			Level:        "DEBUG",
			LogFile:      "data/observability/debug.log",
			Component:    "observability",
			LogToConsole: true,
		},
	}
}

// Validate validates the configuration
func (c *Config) Validate() error {
	// Validate Prometheus endpoint
	if c.Prometheus.Endpoint == "" {
		return fmt.Errorf("prometheus endpoint is required")
	}

	// Parse and validate URL
	u, err := url.Parse(c.Prometheus.Endpoint)
	if err != nil {
		return fmt.Errorf("invalid prometheus endpoint URL: %w", err)
	}

	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("prometheus endpoint must use http or https scheme")
	}

	// Validate timeouts
	if c.Prometheus.Timeout <= 0 {
		return fmt.Errorf("prometheus timeout must be positive")
	}

	// Validate auth configuration
	switch c.Prometheus.Auth.Type {
	case "none":
		// No validation needed
	case "basic":
		hasDirectAuth := c.Prometheus.Auth.Username != "" && c.Prometheus.Auth.Password != ""
		hasSecretRef := c.Prometheus.Auth.PasswordSecretRef != ""
		if !hasDirectAuth && !hasSecretRef {
			return fmt.Errorf("basic auth requires username and either password or passwordSecretRef")
		}
	case "bearer":
		hasDirectToken := c.Prometheus.Auth.Token != ""
		hasSecretRef := c.Prometheus.Auth.TokenSecretRef != ""
		if !hasDirectToken && !hasSecretRef {
			return fmt.Errorf("bearer auth requires either token or tokenSecretRef")
		}
	default:
		return fmt.Errorf("invalid auth type: %s", c.Prometheus.Auth.Type)
	}

	// Validate TLS configuration
	if c.Prometheus.TLS.Enabled {
		if c.Prometheus.TLS.CertFile != "" && c.Prometheus.TLS.KeyFile == "" {
			return fmt.Errorf("TLS cert file requires key file")
		}
		if c.Prometheus.TLS.KeyFile != "" && c.Prometheus.TLS.CertFile == "" {
			return fmt.Errorf("TLS key file requires cert file")
		}
	}

	// Validate retry configuration
	if c.Prometheus.Retry.MaxRetries < 0 {
		return fmt.Errorf("max retries must be non-negative")
	}
	if c.Prometheus.Retry.Multiplier <= 0 {
		return fmt.Errorf("retry multiplier must be positive")
	}

	// Validate display configuration
	if c.Display.RefreshInterval <= 0 {
		return fmt.Errorf("refresh interval must be positive")
	}
	if c.Display.SparklinePoints < 0 {
		return fmt.Errorf("sparkline points must be non-negative")
	}
	if c.Display.DecimalPrecision < 0 {
		return fmt.Errorf("decimal precision must be non-negative")
	}

	// Validate color scheme
	validColorSchemes := map[string]bool{
		"default":    true,
		"colorblind": true,
		"monochrome": true,
	}
	if !validColorSchemes[c.Display.ColorScheme] {
		return fmt.Errorf("invalid color scheme: %s", c.Display.ColorScheme)
	}

	// Validate alert configuration
	if c.Alerts.Enabled {
		if c.Alerts.CheckInterval <= 0 {
			return fmt.Errorf("alert check interval must be positive")
		}

		// Validate alert rules
		for i, rule := range c.Alerts.Rules {
			if rule.Name == "" {
				return fmt.Errorf("alert rule %d: name is required", i)
			}
			if rule.Metric == "" {
				return fmt.Errorf("alert rule %s: metric is required", rule.Name)
			}

			// Validate operator
			validOperators := map[string]bool{
				">": true, "<": true, ">=": true, "<=": true, "==": true, "!=": true,
			}
			if !validOperators[rule.Operator] {
				return fmt.Errorf("alert rule %s: invalid operator %s", rule.Name, rule.Operator)
			}

			// Validate severity
			validSeverities := map[string]bool{
				"info": true, "warning": true, "critical": true,
			}
			if !validSeverities[rule.Severity] {
				return fmt.Errorf("alert rule %s: invalid severity %s", rule.Name, rule.Severity)
			}

			if rule.Duration <= 0 {
				return fmt.Errorf("alert rule %s: duration must be positive", rule.Name)
			}
		}
	}

	// Validate cache configuration
	if c.Cache.Enabled {
		if c.Cache.DefaultTTL <= 0 {
			return fmt.Errorf("cache default TTL must be positive")
		}
		if c.Cache.MaxSize <= 0 {
			return fmt.Errorf("cache max size must be positive")
		}
		if c.Cache.CleanupInterval <= 0 {
			return fmt.Errorf("cache cleanup interval must be positive")
		}
	}

	// Validate metrics configuration
	if c.Metrics.Node.NodeLabel == "" {
		return fmt.Errorf("node label is required")
	}
	if c.Metrics.Node.RateRange == "" {
		return fmt.Errorf("node rate range is required")
	}

	// Validate rate range format
	if _, err := time.ParseDuration(c.Metrics.Node.RateRange); err != nil {
		// Try Prometheus format (e.g., "5m")
		if len(c.Metrics.Node.RateRange) < 2 {
			return fmt.Errorf("invalid rate range format: %s", c.Metrics.Node.RateRange)
		}
	}

	if c.Metrics.Job.Enabled && c.Metrics.Job.CgroupPattern == "" {
		return fmt.Errorf("job cgroup pattern is required when job metrics are enabled")
	}

	return nil
}

// MergeWithDefaults merges the configuration with defaults
func (c *Config) MergeWithDefaults() {
	def := DefaultConfig()

	// Merge Prometheus config
	if c.Prometheus.Endpoint == "" {
		c.Prometheus.Endpoint = def.Prometheus.Endpoint
	}
	if c.Prometheus.Timeout == 0 {
		c.Prometheus.Timeout = def.Prometheus.Timeout
	}
	if c.Prometheus.Auth.Type == "" {
		c.Prometheus.Auth.Type = def.Prometheus.Auth.Type
	}
	if c.Prometheus.Retry.MaxRetries == 0 {
		c.Prometheus.Retry.MaxRetries = def.Prometheus.Retry.MaxRetries
	}
	if c.Prometheus.Retry.InitialDelay == 0 {
		c.Prometheus.Retry.InitialDelay = def.Prometheus.Retry.InitialDelay
	}
	if c.Prometheus.Retry.MaxDelay == 0 {
		c.Prometheus.Retry.MaxDelay = def.Prometheus.Retry.MaxDelay
	}
	if c.Prometheus.Retry.Multiplier == 0 {
		c.Prometheus.Retry.Multiplier = def.Prometheus.Retry.Multiplier
	}

	// Merge Display config
	if c.Display.RefreshInterval == 0 {
		c.Display.RefreshInterval = def.Display.RefreshInterval
	}
	if c.Display.SparklinePoints == 0 {
		c.Display.SparklinePoints = def.Display.SparklinePoints
	}
	if c.Display.ColorScheme == "" {
		c.Display.ColorScheme = def.Display.ColorScheme
	}
	if c.Display.DecimalPrecision == 0 {
		c.Display.DecimalPrecision = def.Display.DecimalPrecision
	}

	// Merge Alert config
	if c.Alerts.CheckInterval == 0 {
		c.Alerts.CheckInterval = def.Alerts.CheckInterval
	}
	if c.Alerts.HistoryRetention == 0 {
		c.Alerts.HistoryRetention = def.Alerts.HistoryRetention
	}
	if len(c.Alerts.Rules) == 0 {
		c.Alerts.Rules = def.Alerts.Rules
	}

	// Merge Cache config
	if c.Cache.DefaultTTL == 0 {
		c.Cache.DefaultTTL = def.Cache.DefaultTTL
	}
	if c.Cache.MaxSize == 0 {
		c.Cache.MaxSize = def.Cache.MaxSize
	}
	if c.Cache.CleanupInterval == 0 {
		c.Cache.CleanupInterval = def.Cache.CleanupInterval
	}

	// Merge Metrics config
	if c.Metrics.Node.NodeLabel == "" {
		c.Metrics.Node.NodeLabel = def.Metrics.Node.NodeLabel
	}
	if c.Metrics.Node.RateRange == "" {
		c.Metrics.Node.RateRange = def.Metrics.Node.RateRange
	}
	if len(c.Metrics.Node.EnabledMetrics) == 0 {
		c.Metrics.Node.EnabledMetrics = def.Metrics.Node.EnabledMetrics
	}
	if c.Metrics.Job.CgroupPattern == "" {
		c.Metrics.Job.CgroupPattern = def.Metrics.Job.CgroupPattern
	}
	if len(c.Metrics.Job.EnabledMetrics) == 0 {
		c.Metrics.Job.EnabledMetrics = def.Metrics.Job.EnabledMetrics
	}

	// Merge Logging config
	if c.Logging.Level == "" {
		c.Logging.Level = def.Logging.Level
	}
	if c.Logging.LogFile == "" {
		c.Logging.LogFile = def.Logging.LogFile
	}
	if c.Logging.Component == "" {
		c.Logging.Component = def.Logging.Component
	}
}
