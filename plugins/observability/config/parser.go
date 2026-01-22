package config

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/jontk/s9s/plugins/observability/api"
	"github.com/jontk/s9s/plugins/observability/security"
)

// Parser handles configuration parsing from generic maps
type Parser struct {
	data map[string]interface{}
}

// NewParser creates a new configuration parser
func NewParser(data map[string]interface{}) *Parser {
	return &Parser{data: data}
}

// ParseConfig parses configuration from map into Config struct
func (p *Parser) ParseConfig() (*Config, error) {
	config := DefaultConfig()

	// Parse Prometheus configuration
	if err := p.parsePrometheusConfig(&config.Prometheus); err != nil {
		return nil, fmt.Errorf("failed to parse prometheus config: %w", err)
	}

	// Parse Display configuration
	if err := p.parseDisplayConfig(&config.Display); err != nil {
		return nil, fmt.Errorf("failed to parse display config: %w", err)
	}

	// Parse Alerts configuration
	if err := p.parseAlertsConfig(&config.Alerts); err != nil {
		return nil, fmt.Errorf("failed to parse alerts config: %w", err)
	}

	// Parse Cache configuration
	if err := p.parseCacheConfig(&config.Cache); err != nil {
		return nil, fmt.Errorf("failed to parse cache config: %w", err)
	}

	// Parse Metrics configuration
	if err := p.parseMetricsConfig(&config.Metrics); err != nil {
		return nil, fmt.Errorf("failed to parse metrics config: %w", err)
	}

	// Parse Security configuration
	if err := p.parseSecurityConfig(&config.Security); err != nil {
		return nil, fmt.Errorf("failed to parse security config: %w", err)
	}

	// Parse External API configuration
	if err := p.parseExternalAPIConfig(&config.ExternalAPI); err != nil {
		return nil, fmt.Errorf("failed to parse external API config: %w", err)
	}

	return config, nil
}

// parsePrometheusConfig parses Prometheus-specific configuration
func (p *Parser) parsePrometheusConfig(config *PrometheusConfig) error {
	if val, ok := p.getValue("prometheus.endpoint"); ok {
		if str, ok := val.(string); ok {
			config.Endpoint = str
		}
	}

	if val, ok := p.getValue("prometheus.timeout"); ok {
		if duration, err := p.parseDuration(val); err == nil {
			config.Timeout = duration
		}
	}

	// Parse auth configuration
	if val, ok := p.getValue("prometheus.auth.type"); ok {
		if str, ok := val.(string); ok {
			config.Auth.Type = str
		}
	}

	if val, ok := p.getValue("prometheus.auth.username"); ok {
		if str, ok := val.(string); ok {
			config.Auth.Username = str
		}
	}

	if val, ok := p.getValue("prometheus.auth.password"); ok {
		if str, ok := val.(string); ok {
			config.Auth.Password = str
		}
	}

	if val, ok := p.getValue("prometheus.auth.token"); ok {
		if str, ok := val.(string); ok {
			config.Auth.Token = str
		}
	}

	// Parse TLS configuration
	if val, ok := p.getValue("prometheus.tls.enabled"); ok {
		if b, err := p.parseBool(val); err == nil {
			config.TLS.Enabled = b
		}
	}

	if val, ok := p.getValue("prometheus.tls.insecureSkipVerify"); ok {
		if b, err := p.parseBool(val); err == nil {
			config.TLS.InsecureSkipVerify = b
		}
	}

	if val, ok := p.getValue("prometheus.tls.caFile"); ok {
		if str, ok := val.(string); ok {
			config.TLS.CAFile = str
		}
	}

	if val, ok := p.getValue("prometheus.tls.certFile"); ok {
		if str, ok := val.(string); ok {
			config.TLS.CertFile = str
		}
	}

	if val, ok := p.getValue("prometheus.tls.keyFile"); ok {
		if str, ok := val.(string); ok {
			config.TLS.KeyFile = str
		}
	}

	// Parse retry configuration
	if val, ok := p.getValue("prometheus.retry.maxRetries"); ok {
		if i, err := p.parseInt(val); err == nil {
			config.Retry.MaxRetries = i
		}
	}

	if val, ok := p.getValue("prometheus.retry.initialDelay"); ok {
		if duration, err := p.parseDuration(val); err == nil {
			config.Retry.InitialDelay = duration
		}
	}

	if val, ok := p.getValue("prometheus.retry.maxDelay"); ok {
		if duration, err := p.parseDuration(val); err == nil {
			config.Retry.MaxDelay = duration
		}
	}

	if val, ok := p.getValue("prometheus.retry.multiplier"); ok {
		if f, err := p.parseFloat(val); err == nil {
			config.Retry.Multiplier = f
		}
	}

	return nil
}

// parseDisplayConfig parses Display-specific configuration
func (p *Parser) parseDisplayConfig(config *DisplayConfig) error {
	if val, ok := p.getValue("display.refreshInterval"); ok {
		if duration, err := p.parseDuration(val); err == nil {
			config.RefreshInterval = duration
		}
	}

	if val, ok := p.getValue("display.showOverlays"); ok {
		if b, err := p.parseBool(val); err == nil {
			config.ShowOverlays = b
		}
	}

	if val, ok := p.getValue("display.showSparklines"); ok {
		if b, err := p.parseBool(val); err == nil {
			config.ShowSparklines = b
		}
	}

	if val, ok := p.getValue("display.sparklinePoints"); ok {
		if i, err := p.parseInt(val); err == nil {
			config.SparklinePoints = i
		}
	}

	if val, ok := p.getValue("display.colorScheme"); ok {
		if str, ok := val.(string); ok {
			config.ColorScheme = str
		}
	}

	if val, ok := p.getValue("display.decimalPrecision"); ok {
		if i, err := p.parseInt(val); err == nil {
			config.DecimalPrecision = i
		}
	}

	return nil
}

// parseAlertsConfig parses Alerts-specific configuration
func (p *Parser) parseAlertsConfig(config *AlertConfig) error {
	if val, ok := p.getValue("alerts.enabled"); ok {
		if b, err := p.parseBool(val); err == nil {
			config.Enabled = b
		}
	}

	if val, ok := p.getValue("alerts.checkInterval"); ok {
		if duration, err := p.parseDuration(val); err == nil {
			config.CheckInterval = duration
		}
	}

	if val, ok := p.getValue("alerts.showNotifications"); ok {
		if b, err := p.parseBool(val); err == nil {
			config.ShowNotifications = b
		}
	}

	if val, ok := p.getValue("alerts.historyRetention"); ok {
		if duration, err := p.parseDuration(val); err == nil {
			config.HistoryRetention = duration
		}
	}

	return nil
}

// parseCacheConfig parses Cache-specific configuration
func (p *Parser) parseCacheConfig(config *CacheConfig) error {
	if val, ok := p.getValue("cache.enabled"); ok {
		if b, err := p.parseBool(val); err == nil {
			config.Enabled = b
		}
	}

	if val, ok := p.getValue("cache.defaultTTL"); ok {
		if duration, err := p.parseDuration(val); err == nil {
			config.DefaultTTL = duration
		}
	}

	if val, ok := p.getValue("cache.maxSize"); ok {
		if i, err := p.parseInt(val); err == nil {
			config.MaxSize = i
		}
	}

	if val, ok := p.getValue("cache.cleanupInterval"); ok {
		if duration, err := p.parseDuration(val); err == nil {
			config.CleanupInterval = duration
		}
	}

	return nil
}

// parseMetricsConfig parses Metrics-specific configuration
func (p *Parser) parseMetricsConfig(config *MetricsConfig) error {
	if val, ok := p.getValue("metrics.node.nodeLabel"); ok {
		if str, ok := val.(string); ok {
			config.Node.NodeLabel = str
		}
	}

	if val, ok := p.getValue("metrics.node.rateRange"); ok {
		if str, ok := val.(string); ok {
			config.Node.RateRange = str
		}
	}

	if val, ok := p.getValue("metrics.node.enabledMetrics"); ok {
		if arr, err := p.parseStringArray(val); err == nil {
			config.Node.EnabledMetrics = arr
		}
	}

	if val, ok := p.getValue("metrics.job.enabled"); ok {
		if b, err := p.parseBool(val); err == nil {
			config.Job.Enabled = b
		}
	}

	if val, ok := p.getValue("metrics.job.cgroupPattern"); ok {
		if str, ok := val.(string); ok {
			config.Job.CgroupPattern = str
		}
	}

	if val, ok := p.getValue("metrics.job.enabledMetrics"); ok {
		if arr, err := p.parseStringArray(val); err == nil {
			config.Job.EnabledMetrics = arr
		}
	}

	return nil
}

// parseSecurityConfig parses Security-specific configuration
func (p *Parser) parseSecurityConfig(config *SecurityConfig) error {
	// Parse secrets configuration
	if val, ok := p.getValue("security.secrets.storageDir"); ok {
		if str, ok := val.(string); ok {
			config.Secrets.StorageDir = str
		}
	}

	if val, ok := p.getValue("security.secrets.encryptAtRest"); ok {
		if b, err := p.parseBool(val); err == nil {
			config.Secrets.EncryptAtRest = b
		}
	}

	if val, ok := p.getValue("security.secrets.masterKeySource"); ok {
		if str, ok := val.(string); ok {
			config.Secrets.MasterKeySource = security.SecretSource(str)
		}
	}

	if val, ok := p.getValue("security.secrets.masterKeyEnv"); ok {
		if str, ok := val.(string); ok {
			config.Secrets.MasterKeyEnv = str
		}
	}

	// Parse API security configuration
	if val, ok := p.getValue("security.api.enableAuth"); ok {
		if b, err := p.parseBool(val); err == nil {
			config.API.EnableAuth = b
		}
	}

	// Parse rate limit configuration
	if val, ok := p.getValue("security.api.rateLimit.requestsPerMinute"); ok {
		if i, err := p.parseInt(val); err == nil {
			config.API.RateLimit.RequestsPerMinute = i
		}
	}

	if val, ok := p.getValue("security.api.rateLimit.enableGlobalLimit"); ok {
		if b, err := p.parseBool(val); err == nil {
			config.API.RateLimit.EnableGlobalLimit = b
		}
	}

	if val, ok := p.getValue("security.api.rateLimit.globalRequestsPerMinute"); ok {
		if i, err := p.parseInt(val); err == nil {
			config.API.RateLimit.GlobalRequestsPerMinute = i
		}
	}

	// Parse validation configuration
	if val, ok := p.getValue("security.api.validation.enabled"); ok {
		if b, err := p.parseBool(val); err == nil {
			config.API.Validation.Enabled = b
		}
	}

	if val, ok := p.getValue("security.api.validation.maxQueryLength"); ok {
		if i, err := p.parseInt(val); err == nil {
			config.API.Validation.MaxQueryLength = i
		}
	}

	// Parse audit configuration
	if val, ok := p.getValue("security.api.audit.enabled"); ok {
		if b, err := p.parseBool(val); err == nil {
			config.API.Audit.Enabled = b
		}
	}

	if val, ok := p.getValue("security.api.audit.logFile"); ok {
		if str, ok := val.(string); ok {
			config.API.Audit.LogFile = str
		}
	}

	return nil
}

// parseExternalAPIConfig parses ExternalAPI-specific configuration
func (p *Parser) parseExternalAPIConfig(config *api.Config) error {
	if val, ok := p.getValue("externalAPI.enabled"); ok {
		if b, err := p.parseBool(val); err == nil {
			config.Enabled = b
		}
	}

	if val, ok := p.getValue("externalAPI.port"); ok {
		if i, err := p.parseInt(val); err == nil {
			config.Port = i
		}
	}

	return nil
}

// Helper methods

// getValue retrieves a value from the configuration map using dot notation
func (p *Parser) getValue(key string) (interface{}, bool) {
	// First, try to get the value directly with the full dotted key (flat map style)
	if val, ok := p.data[key]; ok {
		return val, true
	}

	// If not found, try nested map traversal
	parts := strings.Split(key, ".")
	current := p.data

	for i, part := range parts {
		if i == len(parts)-1 {
			val, ok := current[part]
			return val, ok
		}

		next, ok := current[part].(map[string]interface{})
		if !ok {
			// Try map[interface{}]interface{} (common with YAML)
			if nextAlt, ok := current[part].(map[interface{}]interface{}); ok {
				next = make(map[string]interface{})
				for k, v := range nextAlt {
					if str, ok := k.(string); ok {
						next[str] = v
					}
				}
			} else {
				return nil, false
			}
		}
		current = next
	}

	return nil, false
}

// parseBool parses various boolean representations
func (p *Parser) parseBool(val interface{}) (bool, error) {
	switch v := val.(type) {
	case bool:
		return v, nil
	case string:
		return strconv.ParseBool(v)
	default:
		return false, fmt.Errorf("cannot parse %T as bool", val)
	}
}

// parseInt parses various integer representations
func (p *Parser) parseInt(val interface{}) (int, error) {
	switch v := val.(type) {
	case int:
		return v, nil
	case int64:
		return int(v), nil
	case float64:
		return int(v), nil
	case string:
		return strconv.Atoi(v)
	default:
		return 0, fmt.Errorf("cannot parse %T as int", val)
	}
}

// parseFloat parses various float representations
func (p *Parser) parseFloat(val interface{}) (float64, error) {
	switch v := val.(type) {
	case float64:
		return v, nil
	case float32:
		return float64(v), nil
	case int:
		return float64(v), nil
	case int64:
		return float64(v), nil
	case string:
		return strconv.ParseFloat(v, 64)
	default:
		return 0, fmt.Errorf("cannot parse %T as float", val)
	}
}

// parseDuration parses various duration representations
func (p *Parser) parseDuration(val interface{}) (time.Duration, error) {
	switch v := val.(type) {
	case time.Duration:
		return v, nil
	case string:
		return time.ParseDuration(v)
	case int:
		return time.Duration(v) * time.Second, nil
	case int64:
		return time.Duration(v) * time.Second, nil
	case float64:
		return time.Duration(v * float64(time.Second)), nil
	default:
		return 0, fmt.Errorf("cannot parse %T as duration", val)
	}
}

// parseStringArray parses various string array representations
func (p *Parser) parseStringArray(val interface{}) ([]string, error) {
	if val == nil {
		return nil, fmt.Errorf("cannot parse nil as string array")
	}

	switch v := val.(type) {
	case []string:
		// Trim spaces from each string
		result := make([]string, len(v))
		for i, s := range v {
			result[i] = strings.TrimSpace(s)
		}
		return result, nil
	case []interface{}:
		// Extract strings from interface slice
		var result []string
		for _, item := range v {
			if str, ok := item.(string); ok {
				result = append(result, strings.TrimSpace(str))
			}
		}
		return result, nil
	case string:
		// Handle comma-separated string
		parts := strings.Split(v, ",")
		result := make([]string, len(parts))
		for i, part := range parts {
			result[i] = strings.TrimSpace(part)
		}
		return result, nil
	default:
		return nil, fmt.Errorf("cannot parse %T as string array", val)
	}
}
