package config

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/jontk/s9s/plugins/observability/endpoints"
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

// Field binding helpers to reduce parsing complexity

// parseStringField parses a string field from config and assigns it if present
func (p *Parser) parseStringField(key string, target *string) {
	if val, ok := p.getValue(key); ok {
		if str, ok := val.(string); ok {
			*target = str
		}
	}
}

// parseBoolField parses a boolean field from config and assigns it if present
func (p *Parser) parseBoolField(key string, target *bool) {
	if val, ok := p.getValue(key); ok {
		if b, err := p.parseBool(val); err == nil {
			*target = b
		}
	}
}

// parseIntField parses an integer field from config and assigns it if present
func (p *Parser) parseIntField(key string, target *int) {
	if val, ok := p.getValue(key); ok {
		if i, err := p.parseInt(val); err == nil {
			*target = i
		}
	}
}

// parseDurationField parses a duration field from config and assigns it if present
func (p *Parser) parseDurationField(key string, target *time.Duration) {
	if val, ok := p.getValue(key); ok {
		if dur, err := p.parseDuration(val); err == nil {
			*target = dur
		}
	}
}

// parseFloatField parses a float field from config and assigns it if present
func (p *Parser) parseFloatField(key string, target *float64) {
	if val, ok := p.getValue(key); ok {
		if f, err := p.parseFloat(val); err == nil {
			*target = f
		}
	}
}

// parseStringArrayField parses a string array field from config and assigns it if present
func (p *Parser) parseStringArrayField(key string, target *[]string) {
	if val, ok := p.getValue(key); ok {
		if arr, err := p.parseStringArray(val); err == nil {
			*target = arr
		}
	}
}

// parsePrometheusConfig parses Prometheus-specific configuration
	//nolint:unparam // Designed for future extensibility; currently always returns nil
func (p *Parser) parsePrometheusConfig(config *PrometheusConfig) error {
	p.parseStringField("prometheus.endpoint", &config.Endpoint)
	p.parseDurationField("prometheus.timeout", &config.Timeout)

	// Parse auth configuration
	p.parseStringField("prometheus.auth.type", &config.Auth.Type)
	p.parseStringField("prometheus.auth.username", &config.Auth.Username)
	p.parseStringField("prometheus.auth.password", &config.Auth.Password)
	p.parseStringField("prometheus.auth.token", &config.Auth.Token)

	// Parse TLS configuration
	p.parseBoolField("prometheus.tls.enabled", &config.TLS.Enabled)
	p.parseBoolField("prometheus.tls.insecureSkipVerify", &config.TLS.InsecureSkipVerify)
	p.parseStringField("prometheus.tls.caFile", &config.TLS.CAFile)
	p.parseStringField("prometheus.tls.certFile", &config.TLS.CertFile)
	p.parseStringField("prometheus.tls.keyFile", &config.TLS.KeyFile)

	// Parse retry configuration
	p.parseIntField("prometheus.retry.maxRetries", &config.Retry.MaxRetries)
	p.parseDurationField("prometheus.retry.initialDelay", &config.Retry.InitialDelay)
	p.parseDurationField("prometheus.retry.maxDelay", &config.Retry.MaxDelay)
	p.parseFloatField("prometheus.retry.multiplier", &config.Retry.Multiplier)

	return nil
}

// parseDisplayConfig parses Display-specific configuration
	//nolint:unparam // Designed for future extensibility; currently always returns nil
func (p *Parser) parseDisplayConfig(config *DisplayConfig) error {
	p.parseRefreshInterval("display.refreshInterval", config)
	p.parseShowOverlays("display.showOverlays", config)
	p.parseShowSparklines("display.showSparklines", config)
	p.parseSparklinePoints("display.sparklinePoints", config)
	p.parseColorScheme("display.colorScheme", config)
	p.parseDecimalPrecision("display.decimalPrecision", config)
	return nil
}

// parseRefreshInterval parses the display refresh interval
func (p *Parser) parseRefreshInterval(key string, config *DisplayConfig) {
	if val, ok := p.getValue(key); ok {
		if duration, err := p.parseDuration(val); err == nil {
			config.RefreshInterval = duration
		}
	}
}

// parseShowOverlays parses the show overlays flag
func (p *Parser) parseShowOverlays(key string, config *DisplayConfig) {
	if val, ok := p.getValue(key); ok {
		if b, err := p.parseBool(val); err == nil {
			config.ShowOverlays = b
		}
	}
}

// parseShowSparklines parses the show sparklines flag
func (p *Parser) parseShowSparklines(key string, config *DisplayConfig) {
	if val, ok := p.getValue(key); ok {
		if b, err := p.parseBool(val); err == nil {
			config.ShowSparklines = b
		}
	}
}

// parseSparklinePoints parses the sparkline points configuration
func (p *Parser) parseSparklinePoints(key string, config *DisplayConfig) {
	if val, ok := p.getValue(key); ok {
		if i, err := p.parseInt(val); err == nil {
			config.SparklinePoints = i
		}
	}
}

// parseColorScheme parses the color scheme configuration
func (p *Parser) parseColorScheme(key string, config *DisplayConfig) {
	if val, ok := p.getValue(key); ok {
		if str, ok := val.(string); ok {
			config.ColorScheme = str
		}
	}
}

// parseDecimalPrecision parses the decimal precision configuration
func (p *Parser) parseDecimalPrecision(key string, config *DisplayConfig) {
	if val, ok := p.getValue(key); ok {
		if i, err := p.parseInt(val); err == nil {
			config.DecimalPrecision = i
		}
	}
}

// parseAlertsConfig parses Alerts-specific configuration
	//nolint:unparam // Designed for future extensibility; currently always returns nil
func (p *Parser) parseAlertsConfig(config *AlertConfig) error {
	p.parseAlertsEnabled("alerts.enabled", config)
	p.parseCheckInterval("alerts.checkInterval", config)
	p.parseShowNotifications("alerts.showNotifications", config)
	p.parseHistoryRetention("alerts.historyRetention", config)
	return nil
}

// parseAlertsEnabled parses the alerts enabled flag
func (p *Parser) parseAlertsEnabled(key string, config *AlertConfig) {
	if val, ok := p.getValue(key); ok {
		if b, err := p.parseBool(val); err == nil {
			config.Enabled = b
		}
	}
}

// parseCheckInterval parses the alerts check interval
func (p *Parser) parseCheckInterval(key string, config *AlertConfig) {
	if val, ok := p.getValue(key); ok {
		if duration, err := p.parseDuration(val); err == nil {
			config.CheckInterval = duration
		}
	}
}

// parseShowNotifications parses the show notifications flag
func (p *Parser) parseShowNotifications(key string, config *AlertConfig) {
	if val, ok := p.getValue(key); ok {
		if b, err := p.parseBool(val); err == nil {
			config.ShowNotifications = b
		}
	}
}

// parseHistoryRetention parses the history retention duration
func (p *Parser) parseHistoryRetention(key string, config *AlertConfig) {
	if val, ok := p.getValue(key); ok {
		if duration, err := p.parseDuration(val); err == nil {
			config.HistoryRetention = duration
		}
	}
}

// parseCacheConfig parses Cache-specific configuration
	//nolint:unparam // Designed for future extensibility; currently always returns nil
func (p *Parser) parseCacheConfig(config *CacheConfig) error {
	p.parseCacheEnabled("cache.enabled", config)
	p.parseDefaultTTL("cache.defaultTTL", config)
	p.parseMaxSize("cache.maxSize", config)
	p.parseCleanupInterval("cache.cleanupInterval", config)
	return nil
}

// parseCacheEnabled parses the cache enabled flag
func (p *Parser) parseCacheEnabled(key string, config *CacheConfig) {
	if val, ok := p.getValue(key); ok {
		if b, err := p.parseBool(val); err == nil {
			config.Enabled = b
		}
	}
}

// parseDefaultTTL parses the default cache TTL
func (p *Parser) parseDefaultTTL(key string, config *CacheConfig) {
	if val, ok := p.getValue(key); ok {
		if duration, err := p.parseDuration(val); err == nil {
			config.DefaultTTL = duration
		}
	}
}

// parseMaxSize parses the maximum cache size
func (p *Parser) parseMaxSize(key string, config *CacheConfig) {
	if val, ok := p.getValue(key); ok {
		if i, err := p.parseInt(val); err == nil {
			config.MaxSize = i
		}
	}
}

// parseCleanupInterval parses the cache cleanup interval
func (p *Parser) parseCleanupInterval(key string, config *CacheConfig) {
	if val, ok := p.getValue(key); ok {
		if duration, err := p.parseDuration(val); err == nil {
			config.CleanupInterval = duration
		}
	}
}

// parseMetricsConfig parses Metrics-specific configuration
	//nolint:unparam // Designed for future extensibility; currently always returns nil
func (p *Parser) parseMetricsConfig(config *MetricsConfig) error {
	p.parseNodeMetrics(config)
	p.parseJobMetrics(config)
	return nil
}

// parseNodeMetrics parses node-specific metrics configuration
func (p *Parser) parseNodeMetrics(config *MetricsConfig) {
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
}

// parseJobMetrics parses job-specific metrics configuration
func (p *Parser) parseJobMetrics(config *MetricsConfig) {
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
}

// parseSecurityConfig parses Security-specific configuration
	//nolint:unparam // Designed for future extensibility; currently always returns nil
func (p *Parser) parseSecurityConfig(config *SecurityConfig) error {
	// Parse secrets configuration
	p.parseStringField("security.secrets.storageDir", &config.Secrets.StorageDir)
	p.parseBoolField("security.secrets.encryptAtRest", &config.Secrets.EncryptAtRest)

	// Parse master key source (requires special handling for type conversion)
	if val, ok := p.getValue("security.secrets.masterKeySource"); ok {
		if str, ok := val.(string); ok {
			config.Secrets.MasterKeySource = security.SecretSource(str)
		}
	}

	p.parseStringField("security.secrets.masterKeyEnv", &config.Secrets.MasterKeyEnv)

	// Parse API security configuration
	p.parseBoolField("security.api.enableAuth", &config.API.EnableAuth)

	// Parse rate limit configuration
	p.parseIntField("security.api.rateLimit.requestsPerMinute", &config.API.RateLimit.RequestsPerMinute)
	p.parseBoolField("security.api.rateLimit.enableGlobalLimit", &config.API.RateLimit.EnableGlobalLimit)
	p.parseIntField("security.api.rateLimit.globalRequestsPerMinute", &config.API.RateLimit.GlobalRequestsPerMinute)

	// Parse validation configuration
	p.parseBoolField("security.api.validation.enabled", &config.API.Validation.Enabled)
	p.parseIntField("security.api.validation.maxQueryLength", &config.API.Validation.MaxQueryLength)

	// Parse audit configuration
	p.parseBoolField("security.api.audit.enabled", &config.API.Audit.Enabled)
	p.parseStringField("security.api.audit.logFile", &config.API.Audit.LogFile)

	return nil
}

// parseExternalAPIConfig parses ExternalAPI-specific configuration
	//nolint:unparam // Designed for future extensibility; currently always returns nil
func (p *Parser) parseExternalAPIConfig(config *endpoints.Config) error {
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
