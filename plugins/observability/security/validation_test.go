package security

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestNewRequestValidator(t *testing.T) {
	config := DefaultValidationConfig()
	validator, err := NewRequestValidator(config)

	if err != nil {
		t.Fatalf("NewRequestValidator failed: %v", err)
	}

	if validator == nil {
		t.Fatal("NewRequestValidator returned nil")
	}

	if !validator.config.Enabled {
		t.Error("Expected validation to be enabled by default")
	}
}

func TestNewRequestValidatorInvalidRegex(t *testing.T) {
	config := ValidationConfig{
		AllowedMetricPatterns: []string{"[invalid"},
	}

	_, err := NewRequestValidator(config)
	if err == nil {
		t.Error("Expected error for invalid regex pattern")
	}
}

func TestValidateQueryRequest(t *testing.T) {
	config := ValidationConfig{
		Enabled:                    true,
		MaxQueryLength:             500, // Increased to allow complex query testing
		MaxTimeRange:               time.Hour,
		AllowedMetricPatterns:      []string{"cpu_.*", "memory_.*", "disk_.*", "request_duration"},
		BlockedMetricPatterns:      []string{".*_secret.*"},
		MaxTimeSeries:              1000,
		EnableComplexityValidation: true,
		MaxComplexityScore:         50,
	}

	validator, err := NewRequestValidator(config)
	if err != nil {
		t.Fatalf("Failed to create validator: %v", err)
	}

	tests := []struct {
		name    string
		query   string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid simple query",
			query:   "cpu_usage",
			wantErr: false,
		},
		{
			name:    "valid memory query",
			query:   "memory_total",
			wantErr: false,
		},
		{
			name:    "blocked secret metric",
			query:   "api_secret_total",
			wantErr: true,
			errMsg:  "matches blocked pattern",
		},
		{
			name:    "disallowed metric pattern",
			query:   "network_usage",
			wantErr: true,
			errMsg:  "does not match any allowed patterns",
		},
		{
			name:    "query too long",
			query:   strings.Repeat("x", 501), // Exceeds MaxQueryLength of 500
			wantErr: true,
			errMsg:  "query length exceeds maximum",
		},
		{
			name:    "complex query exceeds complexity limit",
			query:   "rate(cpu_usage[5m]) + rate(memory_usage[5m]) + rate(disk_usage[5m]) + histogram_quantile(0.95, rate(request_duration[5m]))",
			wantErr: true,
			errMsg:  "complexity score",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/api/v1/metrics/query?query="+url.QueryEscape(tt.query), nil)
			err := validator.ValidateRequest(req)

			if (err != nil) != tt.wantErr {
				t.Errorf("validateQueryRequest() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && tt.errMsg != "" {
				if err == nil || !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("Expected error message to contain '%s', got '%v'", tt.errMsg, err)
				}
			}
		})
	}
}

func TestValidateTimeRange(t *testing.T) {
	config := ValidationConfig{
		MaxTimeRange: 24 * time.Hour,
	}

	validator, _ := NewRequestValidator(config)

	tests := []struct {
		name     string
		start    string
		end      string
		duration string
		wantErr  bool
		errMsg   string
	}{
		{
			name:    "valid time range",
			start:   "2023-01-01T00:00:00Z",
			end:     "2023-01-01T12:00:00Z", // 12 hours
			wantErr: false,
		},
		{
			name:    "valid duration",
			duration: "12h",
			wantErr: false,
		},
		{
			name:    "time range too large",
			start:   "2023-01-01T00:00:00Z",
			end:     "2023-01-03T00:00:00Z", // 48 hours
			wantErr: true,
			errMsg:  "exceeds maximum allowed",
		},
		{
			name:     "duration too large",
			duration: "48h",
			wantErr:  true,
			errMsg:   "exceeds maximum allowed",
		},
		{
			name:    "end before start",
			start:   "2023-01-02T00:00:00Z",
			end:     "2023-01-01T00:00:00Z",
			wantErr: true,
			errMsg:  "end time must be after start time",
		},
		{
			name:    "invalid start time format",
			start:   "invalid-time",
			end:     "2023-01-01T12:00:00Z",
			wantErr: true,
			errMsg:  "invalid start time",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url := "/api/v1/metrics/query_range?query=cpu_usage"
			if tt.start != "" {
				url += "&start=" + tt.start
			}
			if tt.end != "" {
				url += "&end=" + tt.end
			}
			if tt.duration != "" {
				url += "&duration=" + tt.duration
			}

			req := httptest.NewRequest("GET", url, nil)
			err := validator.validateTimeRange(req)

			if (err != nil) != tt.wantErr {
				t.Errorf("validateTimeRange() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && tt.errMsg != "" {
				if err == nil || !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("Expected error message to contain '%s', got '%v'", tt.errMsg, err)
				}
			}
		})
	}
}

func TestComplexityScoreCalculation(t *testing.T) {
	tests := []struct {
		name     string
		query    string
		maxScore int
	}{
		{
			name:     "simple metric",
			query:    "cpu_usage",
			maxScore: 10,
		},
		{
			name:     "rate function",
			query:    "rate(cpu_usage[5m])",
			maxScore: 25,
		},
		{
			name:     "complex aggregation",
			query:    "sum by (instance) (rate(cpu_usage[5m]))",
			maxScore: 40,
		},
		{
			name:     "very complex query",
			query:    "histogram_quantile(0.95, sum by (le) (rate(http_request_duration_seconds_bucket[5m])))",
			maxScore: 80,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := calculateComplexityScore(tt.query)
			if score > tt.maxScore {
				t.Errorf("Query '%s' complexity score %d exceeds expected maximum %d", tt.query, score, tt.maxScore)
			}
			t.Logf("Query complexity score: %d", score)
		})
	}
}

func TestValidationMiddleware(t *testing.T) {
	config := ValidationConfig{
		Enabled:               true,
		MaxQueryLength:        50,
		AllowedMetricPatterns: []string{"cpu_.*"},
	}

	validator, err := NewRequestValidator(config)
	if err != nil {
		t.Fatalf("Failed to create validator: %v", err)
	}

	// Create test handler
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	}

	// Wrap with validation middleware
	middleware := ValidationMiddleware(validator)
	wrappedHandler := middleware(handler)

	tests := []struct {
		name           string
		url            string
		expectedStatus int
	}{
		{
			name:           "valid query",
			url:            "/api/v1/metrics/query?query=cpu_usage",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "invalid metric pattern",
			url:            "/api/v1/metrics/query?query=memory_usage",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "query too long",
			url:            "/api/v1/metrics/query?query=" + strings.Repeat("a", 51),
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "missing query parameter",
			url:            "/api/v1/metrics/query",
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tt.url, nil)
			w := httptest.NewRecorder()
			wrappedHandler(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d. Response: %s", tt.expectedStatus, w.Code, w.Body.String())
			}
		})
	}
}

func TestValidationMiddlewareDisabled(t *testing.T) {
	config := ValidationConfig{
		Enabled: false,
	}

	validator, err := NewRequestValidator(config)
	if err != nil {
		t.Fatalf("Failed to create validator: %v", err)
	}

	// Create test handler
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	}

	// Wrap with validation middleware
	middleware := ValidationMiddleware(validator)
	wrappedHandler := middleware(handler)

	// Even invalid requests should pass when validation is disabled
	req := httptest.NewRequest("GET", "/api/v1/metrics/query", nil) // Missing query
	w := httptest.NewRecorder()
	wrappedHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected request to pass when validation disabled, got status %d", w.Code)
	}
}

func TestParseTimeParameter(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{
			name:    "RFC3339 format",
			input:   "2023-01-01T12:00:00Z",
			wantErr: false,
		},
		{
			name:    "Unix timestamp",
			input:   "1672574400", // 2023-01-01T12:00:00Z
			wantErr: false,
		},
		{
			name:    "invalid format",
			input:   "not-a-time",
			wantErr: true,
		},
		{
			name:    "empty string",
			input:   "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parseTimeParameter(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseTimeParameter() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateAnomalyParameters(t *testing.T) {
	validator, _ := NewRequestValidator(DefaultValidationConfig())

	tests := []struct {
		name        string
		sensitivity string
		wantErr     bool
	}{
		{
			name:        "valid sensitivity",
			sensitivity: "2.0",
			wantErr:     false,
		},
		{
			name:        "minimum sensitivity",
			sensitivity: "0.1",
			wantErr:     false,
		},
		{
			name:        "maximum sensitivity",
			sensitivity: "10.0",
			wantErr:     false,
		},
		{
			name:        "sensitivity too low",
			sensitivity: "0.05",
			wantErr:     true,
		},
		{
			name:        "sensitivity too high",
			sensitivity: "15.0",
			wantErr:     true,
		},
		{
			name:        "invalid sensitivity format",
			sensitivity: "not-a-number",
			wantErr:     true,
		},
		{
			name:        "no sensitivity parameter",
			sensitivity: "",
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url := "/api/v1/analysis/anomaly?metric=cpu_usage"
			if tt.sensitivity != "" {
				url += "&sensitivity=" + tt.sensitivity
			}

			req := httptest.NewRequest("GET", url, nil)
			err := validator.validateAnomalyParameters(req)

			if (err != nil) != tt.wantErr {
				t.Errorf("validateAnomalyParameters() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateMetricPatterns(t *testing.T) {
	config := ValidationConfig{
		AllowedMetricPatterns: []string{"cpu_.*", "memory_.*"},
		BlockedMetricPatterns: []string{".*_secret.*", ".*_password.*"},
	}

	validator, _ := NewRequestValidator(config)

	tests := []struct {
		name       string
		metricName string
		wantErr    bool
		errType    string
	}{
		{
			name:       "allowed cpu metric",
			metricName: "cpu_usage",
			wantErr:    false,
		},
		{
			name:       "allowed memory metric",
			metricName: "memory_total",
			wantErr:    false,
		},
		{
			name:       "blocked secret metric",
			metricName: "api_secret_total",
			wantErr:    true,
			errType:    "blocked pattern",
		},
		{
			name:       "blocked password metric",
			metricName: "db_password_count",
			wantErr:    true,
			errType:    "blocked pattern",
		},
		{
			name:       "disallowed disk metric",
			metricName: "disk_usage",
			wantErr:    true,
			errType:    "allowed patterns",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.validateMetricPatterns(tt.metricName)

			if (err != nil) != tt.wantErr {
				t.Errorf("validateMetricPatterns() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && tt.errType != "" {
				if err == nil || !strings.Contains(err.Error(), tt.errType) {
					t.Errorf("Expected error message to contain '%s', got '%v'", tt.errType, err)
				}
			}
		})
	}
}

func TestValidateGenericRequest(t *testing.T) {
	validator, _ := NewRequestValidator(DefaultValidationConfig())

	tests := []struct {
		name   string
		method string
		body   string
		wantErr bool
	}{
		{
			name:    "valid GET request",
			method:  "GET",
			wantErr: false,
		},
		{
			name:    "valid POST request",
			method:  "POST",
			body:    "small body",
			wantErr: false,
		},
		{
			name:    "valid DELETE request",
			method:  "DELETE",
			wantErr: false,
		},
		{
			name:    "invalid method",
			method:  "PATCH",
			wantErr: true,
		},
		{
			name:    "POST body too large",
			method:  "POST",
			body:    strings.Repeat("a", 1024*1024+1), // > 1MB
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var body *strings.Reader
			if tt.body != "" {
				body = strings.NewReader(tt.body)
			}

			var req *http.Request
			if body != nil {
				req = httptest.NewRequest(tt.method, "/api/v1/test", body)
			} else {
				req = httptest.NewRequest(tt.method, "/api/v1/test", nil)
			}
			if body != nil {
				req.ContentLength = int64(len(tt.body))
			}

			err := validator.validateGenericRequest(req)

			if (err != nil) != tt.wantErr {
				t.Errorf("validateGenericRequest() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}


func TestValidationGetStats(t *testing.T) {
	config := DefaultValidationConfig()
	validator, _ := NewRequestValidator(config)

	stats := validator.GetStats()

	if stats == nil {
		t.Fatal("GetStats returned nil")
	}

	expectedFields := []string{
		"validation_enabled",
		"max_query_length",
		"max_time_range",
		"allowed_metric_patterns",
		"blocked_metric_patterns",
		"max_time_series",
		"complexity_validation",
		"max_complexity_score",
	}

	for _, field := range expectedFields {
		if _, exists := stats[field]; !exists {
			t.Errorf("Missing expected field in stats: %s", field)
		}
	}

	if enabled, ok := stats["validation_enabled"].(bool); !ok || !enabled {
		t.Error("Expected validation_enabled to be true")
	}
}