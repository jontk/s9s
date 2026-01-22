package config

import (
	"testing"
	"time"
)

func TestParserBasicFunctionality(t *testing.T) {
	configMap := map[string]interface{}{
		"prometheus.endpoint":         "http://test:9090",
		"prometheus.timeout":          "30s",
		"display.refreshInterval":     "60s",
		"display.showOverlays":        true,
		"cache.enabled":               true,
		"cache.maxSize":               500,
		"metrics.node.enabledMetrics": []interface{}{"cpu", "memory"},
		"metrics.job.enabledMetrics":  "cpu,memory,disk",
	}

	parser := NewParser(configMap)
	config, err := parser.ParseConfig()
	if err != nil {
		t.Fatalf("ParseConfig failed: %v", err)
	}

	// Test parsed values
	if config.Prometheus.Endpoint != "http://test:9090" {
		t.Errorf("Expected endpoint 'http://test:9090', got: %s", config.Prometheus.Endpoint)
	}

	if config.Prometheus.Timeout != 30*time.Second {
		t.Errorf("Expected timeout 30s, got: %v", config.Prometheus.Timeout)
	}

	if !config.Display.ShowOverlays {
		t.Errorf("Expected showOverlays to be true")
	}

	if len(config.Metrics.Node.EnabledMetrics) != 2 {
		t.Errorf("Expected 2 enabled metrics, got: %d", len(config.Metrics.Node.EnabledMetrics))
	}

	if len(config.Metrics.Job.EnabledMetrics) != 3 {
		t.Errorf("Expected 3 job metrics, got: %d", len(config.Metrics.Job.EnabledMetrics))
	}
}

func TestParserDurationErrors(t *testing.T) {
	tests := []struct {
		name      string
		value     interface{}
		shouldErr bool
	}{
		{"valid string", "30s", false},
		{"valid int", 30, false},
		{"valid float", 30.5, false},
		{"valid duration", 30 * time.Second, false},
		{"invalid string", "invalid", true},
		{"invalid type", []int{1, 2, 3}, true},
		{"nil value", nil, true},
	}

	parser := NewParser(map[string]interface{}{})

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parser.parseDuration(tt.value)
			if tt.shouldErr && err == nil {
				t.Errorf("Expected error for %v, but got none", tt.value)
			}
			if !tt.shouldErr && err != nil {
				t.Errorf("Expected no error for %v, but got: %v", tt.value, err)
			}
		})
	}
}

func TestParserBooleanErrors(t *testing.T) {
	tests := []struct {
		name      string
		value     interface{}
		expected  bool
		shouldErr bool
	}{
		{"bool true", true, true, false},
		{"bool false", false, false, false},
		{"string true", "true", true, false},
		{"string false", "false", false, false},
		{"string 1", "1", true, false},
		{"string 0", "0", false, false},
		{"invalid string", "maybe", false, true},
		{"invalid type", 123, false, true},
		{"nil value", nil, false, true},
	}

	parser := NewParser(map[string]interface{}{})

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parser.parseBool(tt.value)
			if tt.shouldErr && err == nil {
				t.Errorf("Expected error for %v, but got none", tt.value)
			}
			if !tt.shouldErr && err != nil {
				t.Errorf("Expected no error for %v, but got: %v", tt.value, err)
			}
			if !tt.shouldErr && result != tt.expected {
				t.Errorf("Expected %v for %v, got: %v", tt.expected, tt.value, result)
			}
		})
	}
}

func TestParserIntegerErrors(t *testing.T) {
	tests := []struct {
		name      string
		value     interface{}
		expected  int
		shouldErr bool
	}{
		{"int value", 42, 42, false},
		{"int64 value", int64(42), 42, false},
		{"float64 value", float64(42.0), 42, false},
		{"string value", "42", 42, false},
		{"invalid string", "not_a_number", 0, true},
		{"float with decimal", 42.7, 42, false},
		{"invalid type", []int{1, 2, 3}, 0, true},
		{"nil value", nil, 0, true},
	}

	parser := NewParser(map[string]interface{}{})

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parser.parseInt(tt.value)
			if tt.shouldErr && err == nil {
				t.Errorf("Expected error for %v, but got none", tt.value)
			}
			if !tt.shouldErr && err != nil {
				t.Errorf("Expected no error for %v, but got: %v", tt.value, err)
			}
			if !tt.shouldErr && result != tt.expected {
				t.Errorf("Expected %d for %v, got: %d", tt.expected, tt.value, result)
			}
		})
	}
}

func TestParserStringArrayErrors(t *testing.T) {
	tests := []struct {
		name      string
		value     interface{}
		expected  []string
		shouldErr bool
	}{
		{"string slice", []interface{}{"a", "b", "c"}, []string{"a", "b", "c"}, false},
		{"mixed slice with strings", []interface{}{"a", 123, "c"}, []string{"a", "c"}, false},
		{"string array", []string{"a", "b"}, []string{"a", "b"}, false},
		{"comma separated", "a,b,c", []string{"a", "b", "c"}, false},
		{"comma separated with spaces", " a , b , c ", []string{"a", "b", "c"}, false},
		{"single string", "single", []string{"single"}, false},
		{"empty string", "", []string{""}, false},
		{"invalid type", 123, nil, true},
		{"nil value", nil, nil, true},
	}

	parser := NewParser(map[string]interface{}{})

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parser.parseStringArray(tt.value)
			if tt.shouldErr && err == nil {
				t.Errorf("Expected error for %v, but got none", tt.value)
			}
			if !tt.shouldErr && err != nil {
				t.Errorf("Expected no error for %v, but got: %v", tt.value, err)
			}
			if !tt.shouldErr {
				if len(result) != len(tt.expected) {
					t.Errorf("Expected length %d for %v, got: %d", len(tt.expected), tt.value, len(result))
				} else {
					for i, expected := range tt.expected {
						if result[i] != expected {
							t.Errorf("Expected %s at index %d for %v, got: %s", expected, i, tt.value, result[i])
						}
					}
				}
			}
		})
	}
}

func TestParserEmptyConfig(t *testing.T) {
	parser := NewParser(map[string]interface{}{})
	config, err := parser.ParseConfig()
	if err != nil {
		t.Fatalf("ParseConfig with empty map should not fail: %v", err)
	}

	// Should return default configuration
	defaultConfig := DefaultConfig()
	if config.Prometheus.Endpoint != defaultConfig.Prometheus.Endpoint {
		t.Errorf("Expected default endpoint, got: %s", config.Prometheus.Endpoint)
	}
}

func TestParserMissingKeys(t *testing.T) {
	configMap := map[string]interface{}{
		"prometheus.endpoint": "http://custom:9090",
		// Missing other keys - should use defaults
	}

	parser := NewParser(configMap)
	config, err := parser.ParseConfig()
	if err != nil {
		t.Fatalf("ParseConfig should handle missing keys: %v", err)
	}

	// Custom value should be set
	if config.Prometheus.Endpoint != "http://custom:9090" {
		t.Errorf("Expected custom endpoint, got: %s", config.Prometheus.Endpoint)
	}

	// Default values should be used for missing keys
	defaultConfig := DefaultConfig()
	if config.Display.RefreshInterval != defaultConfig.Display.RefreshInterval {
		t.Errorf("Expected default refresh interval, got: %v", config.Display.RefreshInterval)
	}
}

func TestParserInvalidConfigTypes(t *testing.T) {
	tests := []struct {
		name      string
		configMap map[string]interface{}
		expectErr bool
	}{
		{
			name: "invalid prometheus timeout",
			configMap: map[string]interface{}{
				"prometheus.timeout": []int{1, 2, 3},
			},
			expectErr: false, // Parser should ignore invalid values and use defaults
		},
		{
			name: "invalid boolean",
			configMap: map[string]interface{}{
				"display.showOverlays": "maybe",
			},
			expectErr: false, // Parser should ignore invalid values
		},
		{
			name: "invalid integer",
			configMap: map[string]interface{}{
				"cache.maxSize": "not_a_number",
			},
			expectErr: false, // Parser should ignore invalid values
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := NewParser(tt.configMap)
			_, err := parser.ParseConfig()
			if tt.expectErr && err == nil {
				t.Errorf("Expected error but got none")
			}
			if !tt.expectErr && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
		})
	}
}

func TestParserNilConfigMap(t *testing.T) {
	parser := NewParser(nil)
	config, err := parser.ParseConfig()
	if err != nil {
		t.Fatalf("ParseConfig with nil map should not fail: %v", err)
	}

	// Should return default configuration
	defaultConfig := DefaultConfig()
	if config.Prometheus.Endpoint != defaultConfig.Prometheus.Endpoint {
		t.Errorf("Expected default endpoint, got: %s", config.Prometheus.Endpoint)
	}
}

func TestParserComplexNestedConfig(t *testing.T) {
	configMap := map[string]interface{}{
		"prometheus.auth.type":      "bearer",
		"prometheus.auth.token":     "secret-token",
		"prometheus.tls.enabled":    true,
		"prometheus.tls.caFile":     "/path/to/ca.pem",
		"alerts.enabled":            false,
		"alerts.checkInterval":      "2m",
		"cache.defaultTTL":          "45s",
		"metrics.node.nodeLabel":    "custom_instance",
		"metrics.job.cgroupPattern": "/custom/path/%s",
	}

	parser := NewParser(configMap)
	config, err := parser.ParseConfig()
	if err != nil {
		t.Fatalf("ParseConfig failed: %v", err)
	}

	// Test nested values
	if config.Prometheus.Auth.Type != "bearer" {
		t.Errorf("Expected auth type 'bearer', got: %s", config.Prometheus.Auth.Type)
	}

	if config.Prometheus.Auth.Token != "secret-token" {
		t.Errorf("Expected token 'secret-token', got: %s", config.Prometheus.Auth.Token)
	}

	if !config.Prometheus.TLS.Enabled {
		t.Errorf("Expected TLS to be enabled")
	}

	if config.Prometheus.TLS.CAFile != "/path/to/ca.pem" {
		t.Errorf("Expected CA file '/path/to/ca.pem', got: %s", config.Prometheus.TLS.CAFile)
	}

	if config.Alerts.Enabled {
		t.Errorf("Expected alerts to be disabled")
	}

	if config.Cache.DefaultTTL != 45*time.Second {
		t.Errorf("Expected cache TTL 45s, got: %v", config.Cache.DefaultTTL)
	}
}
