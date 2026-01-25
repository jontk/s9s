package security

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// RequestValidator handles request validation for API endpoints
type RequestValidator struct {
	config ValidationConfig
}

// ValidationConfig contains validation settings
type ValidationConfig struct {
	// Enable request validation
	Enabled bool `yaml:"enabled" json:"enabled"`

	// Maximum query length
	MaxQueryLength int `yaml:"maxQueryLength" json:"maxQueryLength"`

	// Maximum time range for queries
	MaxTimeRange time.Duration `yaml:"maxTimeRange" json:"maxTimeRange"`

	// Allowed metric patterns (regex)
	AllowedMetricPatterns []string `yaml:"allowedMetricPatterns" json:"allowedMetricPatterns"`

	// Blocked metric patterns (regex)
	BlockedMetricPatterns []string `yaml:"blockedMetricPatterns" json:"blockedMetricPatterns"`

	// Maximum number of time series that can be returned
	MaxTimeSeries int `yaml:"maxTimeSeries" json:"maxTimeSeries"`

	// Enable query complexity validation
	EnableComplexityValidation bool `yaml:"enableComplexityValidation" json:"enableComplexityValidation"`

	// Maximum query complexity score
	MaxComplexityScore int `yaml:"maxComplexityScore" json:"maxComplexityScore"`
}

// DefaultValidationConfig returns default validation configuration
func DefaultValidationConfig() ValidationConfig {
	return ValidationConfig{
		Enabled:                    true,
		MaxQueryLength:             10000,
		MaxTimeRange:               24 * time.Hour,
		AllowedMetricPatterns:      []string{".*"}, // Allow all by default
		BlockedMetricPatterns:      []string{},     // Block none by default
		MaxTimeSeries:              10000,
		EnableComplexityValidation: true,
		MaxComplexityScore:         100,
	}
}

// NewRequestValidator creates a new request validator
func NewRequestValidator(config *ValidationConfig) (*RequestValidator, error) {
	// Validate configuration
	if config.MaxQueryLength <= 0 {
		config.MaxQueryLength = 10000
	}
	if config.MaxTimeRange <= 0 {
		config.MaxTimeRange = 24 * time.Hour
	}
	if config.MaxTimeSeries <= 0 {
		config.MaxTimeSeries = 10000
	}
	if config.MaxComplexityScore <= 0 {
		config.MaxComplexityScore = 100
	}

	// Compile regex patterns for validation
	for _, pattern := range append(config.AllowedMetricPatterns, config.BlockedMetricPatterns...) {
		if _, err := regexp.Compile(pattern); err != nil {
			return nil, fmt.Errorf("invalid regex pattern '%s': %w", pattern, err)
		}
	}

	return &RequestValidator{
		config: *config,
	}, nil
}

// ValidationMiddleware creates HTTP middleware for request validation
func ValidationMiddleware(validator *RequestValidator) func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			if validator == nil || !validator.config.Enabled {
				next(w, r)
				return
			}

			// Validate request based on endpoint
			if err := validator.ValidateRequest(r); err != nil {
				writeValidationError(w, err)
				return
			}

			next(w, r)
		}
	}
}

// ValidateRequest validates an HTTP request
func (v *RequestValidator) ValidateRequest(r *http.Request) error {
	// Validate based on endpoint
	switch {
	case strings.Contains(r.URL.Path, "/query"):
		return v.validateQueryRequest(r)
	case strings.Contains(r.URL.Path, "/historical"):
		return v.validateHistoricalRequest(r)
	case strings.Contains(r.URL.Path, "/analysis"):
		return v.validateAnalysisRequest(r)
	default:
		return v.validateGenericRequest(r)
	}
}

// validateQueryRequest validates Prometheus query requests
func (v *RequestValidator) validateQueryRequest(r *http.Request) error {
	query := r.URL.Query().Get("query")
	if query == "" {
		return fmt.Errorf("query parameter is required")
	}

	// Check query length
	if len(query) > v.config.MaxQueryLength {
		return fmt.Errorf("query length exceeds maximum (%d characters)", v.config.MaxQueryLength)
	}

	// Validate metric patterns
	if err := v.validateMetricPatterns(query); err != nil {
		return err
	}

	// Validate query complexity if enabled
	if v.config.EnableComplexityValidation {
		if err := v.validateQueryComplexity(query); err != nil {
			return err
		}
	}

	// Validate time range for range queries
	if strings.Contains(r.URL.Path, "query_range") {
		if err := v.validateTimeRange(r); err != nil {
			return err
		}
	}

	return nil
}

// validateHistoricalRequest validates historical data requests
func (v *RequestValidator) validateHistoricalRequest(r *http.Request) error {
	metricName := r.URL.Query().Get("metric")
	if metricName == "" {
		return fmt.Errorf("metric parameter is required")
	}

	// Validate metric patterns
	if err := v.validateMetricPatterns(metricName); err != nil {
		return err
	}

	// Validate time range
	if err := v.validateTimeRange(r); err != nil {
		return err
	}

	return nil
}

// validateAnalysisRequest validates analysis requests
func (v *RequestValidator) validateAnalysisRequest(r *http.Request) error {
	metricName := r.URL.Query().Get("metric")
	if metricName == "" {
		return fmt.Errorf("metric parameter is required")
	}

	// Validate metric patterns
	if err := v.validateMetricPatterns(metricName); err != nil {
		return err
	}

	// Validate analysis-specific parameters
	if strings.Contains(r.URL.Path, "/anomaly") {
		if err := v.validateAnomalyParameters(r); err != nil {
			return err
		}
	}

	return nil
}

// validateGenericRequest validates generic requests
func (v *RequestValidator) validateGenericRequest(r *http.Request) error {
	// Basic validation for all requests
	if r.Method != http.MethodGet && r.Method != http.MethodPost && r.Method != http.MethodDelete {
		return fmt.Errorf("unsupported HTTP method: %s", r.Method)
	}

	// Validate content length for POST requests
	if r.Method == http.MethodPost {
		if r.ContentLength > 1024*1024 { // 1MB limit
			return fmt.Errorf("request body too large (max 1MB)")
		}
	}

	return nil
}

// validateMetricPatterns validates metric names against allowed/blocked patterns
func (v *RequestValidator) validateMetricPatterns(metricName string) error {
	// Check blocked patterns first
	for _, pattern := range v.config.BlockedMetricPatterns {
		matched, err := regexp.MatchString(pattern, metricName)
		if err != nil {
			continue // Skip invalid patterns
		}
		if matched {
			return fmt.Errorf("metric '%s' matches blocked pattern '%s'", metricName, pattern)
		}
	}

	// Check allowed patterns
	if len(v.config.AllowedMetricPatterns) == 0 {
		return nil // No restrictions if no patterns specified
	}

	for _, pattern := range v.config.AllowedMetricPatterns {
		matched, err := regexp.MatchString(pattern, metricName)
		if err != nil {
			continue // Skip invalid patterns
		}
		if matched {
			return nil // Found matching allowed pattern
		}
	}

	return fmt.Errorf("metric '%s' does not match any allowed patterns", metricName)
}

// validateQueryComplexity validates Prometheus query complexity
func (v *RequestValidator) validateQueryComplexity(query string) error {
	score := calculateComplexityScore(query)
	if score > v.config.MaxComplexityScore {
		return fmt.Errorf("query complexity score (%d) exceeds maximum (%d)", score, v.config.MaxComplexityScore)
	}
	return nil
}

// calculateComplexityScore calculates a simple complexity score for a query
func calculateComplexityScore(query string) int {
	score := 0

	// Base score for query length
	score += len(query) / 10

	// Add score for various operations
	complexOperations := map[string]int{
		"rate(":        10,
		"irate(":       8,
		"increase(":    8,
		"histogram_":   15,
		"avg_over_":    12,
		"max_over_":    12,
		"min_over_":    12,
		"sum_over_":    12,
		"stddev_over_": 15,
		"quantile":     20,
		"topk(":        15,
		"bottomk(":     15,
		"group_":       10,
		"join":         25,
		"on(":          15,
		"by(":          10,
		"without(":     10,
		"and":          5,
		"or":           5,
		"unless":       8,
	}

	for operation, opScore := range complexOperations {
		score += strings.Count(strings.ToLower(query), operation) * opScore
	}

	// Add score for nested operations (count parentheses)
	score += strings.Count(query, "(") * 2

	// Add score for label matchers
	score += strings.Count(query, "=~") * 5 // Regex matching
	score += strings.Count(query, "!~") * 5 // Negative regex matching
	score += strings.Count(query, "!=") * 3 // Not equal
	score += strings.Count(query, "=") * 1  // Equal matching

	return score
}

// validateTimeRange validates time range parameters
func (v *RequestValidator) validateTimeRange(r *http.Request) error {
	startStr := r.URL.Query().Get("start")
	endStr := r.URL.Query().Get("end")
	durationStr := r.URL.Query().Get("duration")

	var timeRange time.Duration

	if startStr != "" && endStr != "" {
		// Parse start and end times
		start, err := parseTimeParameter(startStr)
		if err != nil {
			return fmt.Errorf("invalid start time: %w", err)
		}

		end, err := parseTimeParameter(endStr)
		if err != nil {
			return fmt.Errorf("invalid end time: %w", err)
		}

		if end.Before(start) {
			return fmt.Errorf("end time must be after start time")
		}

		timeRange = end.Sub(start)
	} else if durationStr != "" {
		// Parse duration
		var err error
		timeRange, err = time.ParseDuration(durationStr)
		if err != nil {
			return fmt.Errorf("invalid duration: %w", err)
		}
	}

	// Check if time range exceeds maximum
	if timeRange > v.config.MaxTimeRange {
		return fmt.Errorf("time range (%s) exceeds maximum allowed (%s)", timeRange, v.config.MaxTimeRange)
	}

	return nil
}

// validateAnomalyParameters validates anomaly detection specific parameters
func (v *RequestValidator) validateAnomalyParameters(r *http.Request) error {
	sensitivityStr := r.URL.Query().Get("sensitivity")
	if sensitivityStr != "" {
		sensitivity, err := strconv.ParseFloat(sensitivityStr, 64)
		if err != nil {
			return fmt.Errorf("invalid sensitivity value: %w", err)
		}
		if sensitivity < 0.1 || sensitivity > 10.0 {
			return fmt.Errorf("sensitivity must be between 0.1 and 10.0")
		}
	}

	return nil
}

// parseTimeParameter parses time parameters from request
func parseTimeParameter(timeStr string) (time.Time, error) {
	// Try RFC3339 format first
	if t, err := time.Parse(time.RFC3339, timeStr); err == nil {
		return t, nil
	}

	// Try Unix timestamp
	if timestamp, err := strconv.ParseInt(timeStr, 10, 64); err == nil {
		return time.Unix(timestamp, 0), nil
	}

	return time.Time{}, fmt.Errorf("invalid time format: %s", timeStr)
}

// writeValidationError writes a validation error response
func writeValidationError(w http.ResponseWriter, err error) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusBadRequest)

	response := map[string]interface{}{
		"status": "error",
		"error":  "Validation failed",
		"detail": err.Error(),
		"code":   "VALIDATION_ERROR",
	}

	_ = json.NewEncoder(w).Encode(response)
}

// GetStats returns validation statistics
func (v *RequestValidator) GetStats() map[string]interface{} {
	return map[string]interface{}{
		"validation_enabled":      v.config.Enabled,
		"max_query_length":        v.config.MaxQueryLength,
		"max_time_range":          v.config.MaxTimeRange.String(),
		"allowed_metric_patterns": v.config.AllowedMetricPatterns,
		"blocked_metric_patterns": v.config.BlockedMetricPatterns,
		"max_time_series":         v.config.MaxTimeSeries,
		"complexity_validation":   v.config.EnableComplexityValidation,
		"max_complexity_score":    v.config.MaxComplexityScore,
	}
}
