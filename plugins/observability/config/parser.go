package config

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// Parser handles configuration parsing from generic maps
type Parser struct {
	configMap map[string]interface{}
}

// NewParser creates a new configuration parser
func NewParser(configMap map[string]interface{}) *Parser {
	return &Parser{
		configMap: configMap,
	}
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

	if val, ok := p.getValue("alerts.loadPredefinedRules"); ok {
		if b, err := p.parseBool(val); err == nil {
			config.LoadPredefinedRules = b
		}
	}

	if val, ok := p.getValue("alerts.showNotifications"); ok {
		if b, err := p.parseBool(val); err == nil {
			config.ShowNotifications = b
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

	// Parse array configurations
	if val, ok := p.getValue("metrics.node.enabledMetrics"); ok {
		if metrics, err := p.parseStringArray(val); err == nil {
			config.Node.EnabledMetrics = metrics
		}
	}

	if val, ok := p.getValue("metrics.job.enabledMetrics"); ok {
		if metrics, err := p.parseStringArray(val); err == nil {
			config.Job.EnabledMetrics = metrics
		}
	}

	return nil
}

// Helper methods

// getValue gets a nested value using dot notation
func (p *Parser) getValue(key string) (interface{}, bool) {
	if val, exists := p.configMap[key]; exists {
		return val, true
	}
	return nil, false
}

// parseDuration parses duration values from various types
func (p *Parser) parseDuration(val interface{}) (time.Duration, error) {
	switch v := val.(type) {
	case string:
		return time.ParseDuration(v)
	case time.Duration:
		return v, nil
	case int:
		return time.Duration(v) * time.Second, nil
	case int64:
		return time.Duration(v) * time.Second, nil
	case float64:
		return time.Duration(v) * time.Second, nil
	default:
		return 0, fmt.Errorf("invalid duration type: %T", val)
	}
}

// parseBool parses boolean values from various types
func (p *Parser) parseBool(val interface{}) (bool, error) {
	switch v := val.(type) {
	case bool:
		return v, nil
	case string:
		return strconv.ParseBool(v)
	default:
		return false, fmt.Errorf("invalid boolean type: %T", val)
	}
}

// parseInt parses integer values from various types
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
		return 0, fmt.Errorf("invalid integer type: %T", val)
	}
}

// parseStringArray parses string arrays from various types
func (p *Parser) parseStringArray(val interface{}) ([]string, error) {
	switch v := val.(type) {
	case []interface{}:
		result := make([]string, 0, len(v))
		for _, item := range v {
			if str, ok := item.(string); ok {
				result = append(result, str)
			}
		}
		return result, nil
	case []string:
		return v, nil
	case string:
		// Support comma-separated string format
		parts := strings.Split(v, ",")
		for i, part := range parts {
			parts[i] = strings.TrimSpace(part)
		}
		return parts, nil
	default:
		return nil, fmt.Errorf("invalid string array type: %T", val)
	}
}