// Package api provides external HTTP API endpoints for accessing observability data.
// The API supports Prometheus queries, historical data analysis, resource efficiency metrics,
// and subscription management. All endpoints are protected by configurable security layers
// including authentication, rate limiting, request validation, and audit logging.
package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/jontk/s9s/internal/plugin"
	"github.com/jontk/s9s/plugins/observability/analysis"
	"github.com/jontk/s9s/plugins/observability/historical"
	"github.com/jontk/s9s/plugins/observability/prometheus"
	"github.com/jontk/s9s/plugins/observability/security"
	"github.com/jontk/s9s/plugins/observability/subscription"
)

// ExternalAPI provides HTTP API endpoints for external access to observability data
type ExternalAPI struct {
	client              *prometheus.CachedClient
	subscriptionMgr     *subscription.SubscriptionManager
	historicalCollector *historical.HistoricalDataCollector
	historicalAnalyzer  *historical.HistoricalAnalyzer
	efficiencyAnalyzer  *analysis.ResourceEfficiencyAnalyzer
	rateLimiter         *security.RateLimiter
	rateLimitConfig     security.RateLimitConfig
	validator           *security.RequestValidator
	validationConfig    security.ValidationConfig
	auditLogger         *security.AuditLogger
	auditConfig         security.AuditConfig
	server              *http.Server
	enabled             bool
	port                int
	authToken           string
}

// Config for external API
type Config struct {
	Enabled    bool                      `json:"enabled" yaml:"enabled"`
	Port       int                       `json:"port" yaml:"port"`
	AuthToken  string                    `json:"auth_token" yaml:"authToken"`
	RateLimit  security.RateLimitConfig  `json:"rate_limit" yaml:"rateLimit"`
	Validation security.ValidationConfig `json:"validation" yaml:"validation"`
	Audit      security.AuditConfig      `json:"audit" yaml:"audit"`
}

// DefaultConfig returns default API configuration
func DefaultConfig() Config {
	return Config{
		Enabled:    false,
		Port:       8080,
		AuthToken:  "",
		RateLimit:  security.DefaultRateLimitConfig(),
		Validation: security.DefaultValidationConfig(),
		Audit:      security.DefaultAuditConfig(),
	}
}

// NewExternalAPI creates a new external API instance
func NewExternalAPI(
	client *prometheus.CachedClient,
	subscriptionMgr *subscription.SubscriptionManager,
	historicalCollector *historical.HistoricalDataCollector,
	historicalAnalyzer *historical.HistoricalAnalyzer,
	efficiencyAnalyzer *analysis.ResourceEfficiencyAnalyzer,
	config Config,
) *ExternalAPI {
	var rateLimiter *security.RateLimiter
	if config.Enabled && (config.RateLimit.RequestsPerMinute > 0 || config.RateLimit.EnableGlobalLimit) {
		rateLimiter = security.NewRateLimiter(config.RateLimit)
	}

	var validator *security.RequestValidator
	if config.Enabled && config.Validation.Enabled {
		var err error
		validator, err = security.NewRequestValidator(config.Validation)
		if err != nil {
			// Log error but don't fail initialization
			validator = nil
		}
	}

	var auditLogger *security.AuditLogger
	if config.Enabled && config.Audit.Enabled {
		var err error
		auditLogger, err = security.NewAuditLogger(config.Audit)
		if err != nil {
			// Log error but don't fail initialization
			auditLogger = nil
		}
	}

	return &ExternalAPI{
		client:              client,
		subscriptionMgr:     subscriptionMgr,
		historicalCollector: historicalCollector,
		historicalAnalyzer:  historicalAnalyzer,
		efficiencyAnalyzer:  efficiencyAnalyzer,
		rateLimiter:         rateLimiter,
		rateLimitConfig:     config.RateLimit,
		validator:           validator,
		validationConfig:    config.Validation,
		auditLogger:         auditLogger,
		auditConfig:         config.Audit,
		enabled:             config.Enabled,
		port:                config.Port,
		authToken:           config.AuthToken,
	}
}

// Start starts the external API server
func (api *ExternalAPI) Start(ctx context.Context) error {
	if !api.enabled {
		return nil
	}

	mux := http.NewServeMux()

	// Create middleware chain (rate limiting + authentication)
	middleware := api.createMiddleware()

	// Register API routes with middleware
	mux.HandleFunc("/api/v1/metrics/query", middleware(api.handleMetricsQuery))
	mux.HandleFunc("/api/v1/metrics/query_range", middleware(api.handleMetricsQueryRange))
	mux.HandleFunc("/api/v1/historical/data", middleware(api.handleHistoricalData))
	mux.HandleFunc("/api/v1/historical/statistics", middleware(api.handleHistoricalStatistics))
	mux.HandleFunc("/api/v1/analysis/trend", middleware(api.handleTrendAnalysis))
	mux.HandleFunc("/api/v1/analysis/anomaly", middleware(api.handleAnomalyDetection))
	mux.HandleFunc("/api/v1/analysis/seasonal", middleware(api.handleSeasonalAnalysis))
	mux.HandleFunc("/api/v1/efficiency/resource", middleware(api.handleResourceEfficiency))
	mux.HandleFunc("/api/v1/efficiency/cluster", middleware(api.handleClusterEfficiency))
	mux.HandleFunc("/api/v1/subscriptions", middleware(api.handleSubscriptions))
	mux.HandleFunc("/api/v1/subscriptions/create", middleware(api.handleCreateSubscription))
	mux.HandleFunc("/api/v1/subscriptions/delete", middleware(api.handleDeleteSubscription))
	mux.HandleFunc("/api/v1/status", middleware(api.handleStatus))
	mux.HandleFunc("/health", api.handleHealth) // Health endpoint doesn't require rate limiting

	api.server = &http.Server{
		Addr:              fmt.Sprintf(":%d", api.port),
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	go func() {
		if err := api.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			// Log error
		}
	}()

	return nil
}

// Stop stops the external API server
func (api *ExternalAPI) Stop(ctx context.Context) error {
	if api.rateLimiter != nil {
		api.rateLimiter.Stop()
	}

	if api.auditLogger != nil {
		_ = api.auditLogger.Close()
	}

	if api.server == nil {
		return nil
	}

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	return api.server.Shutdown(ctx)
}

// createMiddleware creates the middleware chain (audit + validation + rate limiting + authentication)
func (api *ExternalAPI) createMiddleware() func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		// Apply audit logging middleware first (outermost)
		handler := next
		if api.auditLogger != nil {
			handler = security.AuditMiddleware(api.auditLogger)(handler)
		}

		// Apply validation middleware
		if api.validator != nil {
			handler = security.ValidationMiddleware(api.validator)(handler)
		}

		// Apply rate limiting middleware
		if api.rateLimiter != nil {
			handler = security.RateLimitMiddleware(api.rateLimiter, api.rateLimitConfig)(handler)
		}

		// Apply authentication middleware last (innermost)
		return api.authenticateMiddleware(handler)
	}
}

// authenticateMiddleware provides authentication for API endpoints
func (api *ExternalAPI) authenticateMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if api.authToken == "" {
			// No authentication required
			next(w, r)
			return
		}

		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			api.writeError(w, http.StatusUnauthorized, "Authorization header required")
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			api.writeError(w, http.StatusUnauthorized, "Invalid authorization header format")
			return
		}

		if parts[1] != api.authToken {
			api.writeError(w, http.StatusUnauthorized, "Invalid token")
			return
		}

		next(w, r)
	}
}

// handleMetricsQuery handles Prometheus query requests
func (api *ExternalAPI) handleMetricsQuery(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		api.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	query := r.URL.Query().Get("query")
	if query == "" {
		api.writeError(w, http.StatusBadRequest, "Query parameter is required")
		return
	}

	timeStr := r.URL.Query().Get("time")
	var queryTime time.Time
	if timeStr != "" {
		if parsed, err := time.Parse(time.RFC3339, timeStr); err == nil {
			queryTime = parsed
		} else if timestamp, err := strconv.ParseInt(timeStr, 10, 64); err == nil {
			queryTime = time.Unix(timestamp, 0)
		} else {
			api.writeError(w, http.StatusBadRequest, "Invalid time format")
			return
		}
	} else {
		queryTime = time.Now()
	}

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	result, err := api.client.Query(ctx, query, queryTime)
	if err != nil {
		api.writeError(w, http.StatusInternalServerError, fmt.Sprintf("Query failed: %v", err))
		return
	}

	api.writeJSON(w, map[string]interface{}{
		"status": "success",
		"data":   result,
	})
}

// handleMetricsQueryRange handles Prometheus range query requests
func (api *ExternalAPI) handleMetricsQueryRange(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		api.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	query := r.URL.Query().Get("query")
	if query == "" {
		api.writeError(w, http.StatusBadRequest, "Query parameter is required")
		return
	}

	start, err := api.parseTime(r.URL.Query().Get("start"))
	if err != nil {
		api.writeError(w, http.StatusBadRequest, fmt.Sprintf("Invalid start time: %v", err))
		return
	}

	end, err := api.parseTime(r.URL.Query().Get("end"))
	if err != nil {
		api.writeError(w, http.StatusBadRequest, fmt.Sprintf("Invalid end time: %v", err))
		return
	}

	stepStr := r.URL.Query().Get("step")
	if stepStr == "" {
		stepStr = "15s"
	}
	step, err := time.ParseDuration(stepStr)
	if err != nil {
		api.writeError(w, http.StatusBadRequest, fmt.Sprintf("Invalid step duration: %v", err))
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 60*time.Second)
	defer cancel()

	result, err := api.client.QueryRange(ctx, query, start, end, step)
	if err != nil {
		api.writeError(w, http.StatusInternalServerError, fmt.Sprintf("Range query failed: %v", err))
		return
	}

	api.writeJSON(w, map[string]interface{}{
		"status": "success",
		"data":   result,
	})
}

// handleHistoricalData handles historical data requests
func (api *ExternalAPI) handleHistoricalData(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		api.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	metricName := r.URL.Query().Get("metric")
	if metricName == "" {
		api.writeError(w, http.StatusBadRequest, "Metric parameter is required")
		return
	}

	start, err := api.parseTime(r.URL.Query().Get("start"))
	if err != nil {
		api.writeError(w, http.StatusBadRequest, fmt.Sprintf("Invalid start time: %v", err))
		return
	}

	end, err := api.parseTime(r.URL.Query().Get("end"))
	if err != nil {
		api.writeError(w, http.StatusBadRequest, fmt.Sprintf("Invalid end time: %v", err))
		return
	}

	series, err := api.historicalCollector.GetHistoricalData(metricName, start, end)
	if err != nil {
		api.writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to get historical data: %v", err))
		return
	}

	api.writeJSON(w, map[string]interface{}{
		"status": "success",
		"data":   series,
	})
}

// handleHistoricalStatistics handles historical statistics requests
func (api *ExternalAPI) handleHistoricalStatistics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		api.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	metricName := r.URL.Query().Get("metric")
	if metricName == "" {
		api.writeError(w, http.StatusBadRequest, "Metric parameter is required")
		return
	}

	durationStr := r.URL.Query().Get("duration")
	if durationStr == "" {
		durationStr = "24h"
	}
	duration, err := time.ParseDuration(durationStr)
	if err != nil {
		api.writeError(w, http.StatusBadRequest, fmt.Sprintf("Invalid duration: %v", err))
		return
	}

	stats, err := api.historicalCollector.GetMetricStatistics(metricName, duration)
	if err != nil {
		api.writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to get statistics: %v", err))
		return
	}

	api.writeJSON(w, map[string]interface{}{
		"status": "success",
		"data":   stats,
	})
}

// handleTrendAnalysis handles trend analysis requests
func (api *ExternalAPI) handleTrendAnalysis(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		api.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	metricName := r.URL.Query().Get("metric")
	if metricName == "" {
		api.writeError(w, http.StatusBadRequest, "Metric parameter is required")
		return
	}

	durationStr := r.URL.Query().Get("duration")
	if durationStr == "" {
		durationStr = "24h"
	}
	duration, err := time.ParseDuration(durationStr)
	if err != nil {
		api.writeError(w, http.StatusBadRequest, fmt.Sprintf("Invalid duration: %v", err))
		return
	}

	analysis, err := api.historicalAnalyzer.AnalyzeTrend(metricName, duration)
	if err != nil {
		api.writeError(w, http.StatusInternalServerError, fmt.Sprintf("Trend analysis failed: %v", err))
		return
	}

	api.writeJSON(w, map[string]interface{}{
		"status": "success",
		"data":   analysis,
	})
}

// handleAnomalyDetection handles anomaly detection requests
func (api *ExternalAPI) handleAnomalyDetection(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		api.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	metricName := r.URL.Query().Get("metric")
	if metricName == "" {
		api.writeError(w, http.StatusBadRequest, "Metric parameter is required")
		return
	}

	durationStr := r.URL.Query().Get("duration")
	if durationStr == "" {
		durationStr = "24h"
	}
	duration, err := time.ParseDuration(durationStr)
	if err != nil {
		api.writeError(w, http.StatusBadRequest, fmt.Sprintf("Invalid duration: %v", err))
		return
	}

	sensitivityStr := r.URL.Query().Get("sensitivity")
	sensitivity := 2.0
	if sensitivityStr != "" {
		if parsed, err := strconv.ParseFloat(sensitivityStr, 64); err == nil {
			sensitivity = parsed
		}
	}

	analysis, err := api.historicalAnalyzer.DetectAnomalies(metricName, duration, sensitivity)
	if err != nil {
		api.writeError(w, http.StatusInternalServerError, fmt.Sprintf("Anomaly detection failed: %v", err))
		return
	}

	api.writeJSON(w, map[string]interface{}{
		"status": "success",
		"data":   analysis,
	})
}

// handleSeasonalAnalysis handles seasonal analysis requests
func (api *ExternalAPI) handleSeasonalAnalysis(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		api.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	metricName := r.URL.Query().Get("metric")
	if metricName == "" {
		api.writeError(w, http.StatusBadRequest, "Metric parameter is required")
		return
	}

	durationStr := r.URL.Query().Get("duration")
	if durationStr == "" {
		durationStr = "168h" // 1 week
	}
	duration, err := time.ParseDuration(durationStr)
	if err != nil {
		api.writeError(w, http.StatusBadRequest, fmt.Sprintf("Invalid duration: %v", err))
		return
	}

	analysis, err := api.historicalAnalyzer.AnalyzeSeasonality(metricName, duration)
	if err != nil {
		api.writeError(w, http.StatusInternalServerError, fmt.Sprintf("Seasonal analysis failed: %v", err))
		return
	}

	api.writeJSON(w, map[string]interface{}{
		"status": "success",
		"data":   analysis,
	})
}

// handleResourceEfficiency handles resource efficiency analysis requests
func (api *ExternalAPI) handleResourceEfficiency(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		api.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	resourceType := r.URL.Query().Get("type")
	if resourceType == "" {
		api.writeError(w, http.StatusBadRequest, "Resource type parameter is required")
		return
	}

	var resType analysis.ResourceType
	switch resourceType {
	case "cpu":
		resType = analysis.ResourceCPU
	case "memory":
		resType = analysis.ResourceMemory
	case "storage":
		resType = analysis.ResourceStorage
	case "network":
		resType = analysis.ResourceNetwork
	case "gpu":
		resType = analysis.ResourceGPU
	default:
		api.writeError(w, http.StatusBadRequest, "Invalid resource type")
		return
	}

	durationStr := r.URL.Query().Get("duration")
	if durationStr == "" {
		durationStr = "168h" // 1 week
	}
	duration, err := time.ParseDuration(durationStr)
	if err != nil {
		api.writeError(w, http.StatusBadRequest, fmt.Sprintf("Invalid duration: %v", err))
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	analysis, err := api.efficiencyAnalyzer.AnalyzeResourceEfficiency(ctx, resType, duration)
	if err != nil {
		api.writeError(w, http.StatusInternalServerError, fmt.Sprintf("Resource efficiency analysis failed: %v", err))
		return
	}

	api.writeJSON(w, map[string]interface{}{
		"status": "success",
		"data":   analysis,
	})
}

// handleClusterEfficiency handles cluster efficiency analysis requests
func (api *ExternalAPI) handleClusterEfficiency(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		api.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	durationStr := r.URL.Query().Get("duration")
	if durationStr == "" {
		durationStr = "168h" // 1 week
	}
	duration, err := time.ParseDuration(durationStr)
	if err != nil {
		api.writeError(w, http.StatusBadRequest, fmt.Sprintf("Invalid duration: %v", err))
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 60*time.Second)
	defer cancel()

	analysis, err := api.efficiencyAnalyzer.AnalyzeClusterEfficiency(ctx, duration)
	if err != nil {
		api.writeError(w, http.StatusInternalServerError, fmt.Sprintf("Cluster efficiency analysis failed: %v", err))
		return
	}

	api.writeJSON(w, map[string]interface{}{
		"status": "success",
		"data":   analysis,
	})
}

// handleSubscriptions handles subscription listing
func (api *ExternalAPI) handleSubscriptions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		api.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	subscriptions := api.subscriptionMgr.ListSubscriptions()

	api.writeJSON(w, map[string]interface{}{
		"status": "success",
		"data":   subscriptions,
	})
}

// handleCreateSubscription handles subscription creation
func (api *ExternalAPI) handleCreateSubscription(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		api.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var request struct {
		ProviderID string                 `json:"provider_id"`
		Params     map[string]interface{} `json:"params"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		api.writeError(w, http.StatusBadRequest, fmt.Sprintf("Invalid JSON: %v", err))
		return
	}

	// Create a dummy callback for API subscriptions
	callback := func(data interface{}, err error) {
		// API subscriptions don't use callbacks directly
	}

	subscriptionID, err := api.subscriptionMgr.Subscribe(request.ProviderID, request.Params, callback)
	if err != nil {
		api.writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to create subscription: %v", err))
		return
	}

	api.writeJSON(w, map[string]interface{}{
		"status": "success",
		"data": map[string]interface{}{
			"subscription_id": subscriptionID,
		},
	})
}

// handleDeleteSubscription handles subscription deletion
func (api *ExternalAPI) handleDeleteSubscription(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		api.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	subscriptionID := r.URL.Query().Get("id")
	if subscriptionID == "" {
		api.writeError(w, http.StatusBadRequest, "Subscription ID is required")
		return
	}

	err := api.subscriptionMgr.Unsubscribe(plugin.SubscriptionID(subscriptionID))
	if err != nil {
		api.writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to delete subscription: %v", err))
		return
	}

	api.writeJSON(w, map[string]interface{}{
		"status": "success",
	})
}

// handleStatus handles status requests
func (api *ExternalAPI) handleStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		api.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	status := map[string]interface{}{
		"api_enabled":       api.enabled,
		"api_port":          api.port,
		"subscriptions":     api.subscriptionMgr.GetStats(),
		"historical":        api.historicalCollector.GetCollectorStats(),
		"cache":             api.client.CacheStats(),
		"available_metrics": api.historicalCollector.GetAvailableMetrics(),
	}

	// Add rate limiting stats if available
	if api.rateLimiter != nil {
		status["rate_limiting"] = api.rateLimiter.GetStats()
	}

	// Add validation stats if available
	if api.validator != nil {
		status["validation"] = api.validator.GetStats()
	}

	// Add audit stats if available
	if api.auditLogger != nil {
		status["audit"] = api.auditLogger.GetStats()
	}

	api.writeJSON(w, map[string]interface{}{
		"status": "success",
		"data":   status,
	})
}

// handleHealth handles health check requests
func (api *ExternalAPI) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "healthy",
		"time":   time.Now(),
	})
}

// Helper methods

func (api *ExternalAPI) parseTime(timeStr string) (time.Time, error) {
	if timeStr == "" {
		return time.Now(), nil
	}

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

func (api *ExternalAPI) writeJSON(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(data)
}

func (api *ExternalAPI) writeError(w http.ResponseWriter, code int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "error",
		"error":  message,
	})
}
