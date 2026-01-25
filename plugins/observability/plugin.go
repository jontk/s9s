package observability

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/rivo/tview"

	"github.com/jontk/s9s/internal/plugin"
	"github.com/jontk/s9s/plugins/observability/analysis"
	"github.com/jontk/s9s/plugins/observability/config"
	"github.com/jontk/s9s/plugins/observability/initialization"
	"github.com/jontk/s9s/plugins/observability/logging"
	"github.com/jontk/s9s/plugins/observability/overlays"
	"github.com/jontk/s9s/plugins/observability/subscription"
	"github.com/jontk/s9s/plugins/observability/views"
)

// Plugin implements the observability plugin
type Plugin struct {
	config      *config.Config
	components  *initialization.Components
	app         *tview.Application
	view        *views.ObservabilityView
	logger      *logging.Logger
	running     bool
	slurmClient interface{} // Store SLURM client for job queries
}

//nolint:revive // type alias for backward compatibility
type ObservabilityPlugin = Plugin

// New creates a new observability plugin instance
func New() *Plugin {
	return &Plugin{
		config: config.DefaultConfig(),
	}
}

// GetInfo returns plugin information
func (p *ObservabilityPlugin) GetInfo() plugin.Info {
	return plugin.Info{
		Name:        "observability",
		Version:     "1.0.0",
		Description: "Prometheus integration for real-time metrics monitoring",
		Author:      "s9s team",
		License:     "MIT",
		Requires:    []string{},
		Provides:    []string{"metrics", "alerts", "observability-view"},
		ConfigSchema: map[string]plugin.ConfigField{
			"prometheus.endpoint": {
				Type:        "string",
				Description: "Prometheus server endpoint",
				Default:     "http://localhost:9090",
				Required:    true,
				Validation:  "^https?://",
			},
			"prometheus.timeout": {
				Type:        "duration",
				Description: "Query timeout",
				Default:     "10s",
				Required:    false,
			},
			"display.refreshInterval": {
				Type:        "duration",
				Description: "Metrics refresh interval",
				Default:     "30s",
				Required:    false,
			},
			"display.showOverlays": {
				Type:        "bool",
				Description: "Show metric overlays on existing views",
				Default:     true,
				Required:    false,
			},
			"alerts.enabled": {
				Type:        "bool",
				Description: "Enable alert monitoring",
				Default:     true,
				Required:    false,
			},
			"api.enabled": {
				Type:        "bool",
				Description: "Enable external HTTP API",
				Default:     false,
				Required:    false,
			},
			"api.port": {
				Type:        "int",
				Description: "HTTP API server port",
				Default:     8080,
				Required:    false,
			},
			"api.auth_token": {
				Type:        "string",
				Description: "Authentication token for API access (optional)",
				Default:     "",
				Required:    false,
			},
		},
	}
}

// Init initializes the plugin with configuration
func (p *ObservabilityPlugin) Init(_ context.Context, configMap map[string]interface{}) error {
	// Parse configuration from map into Config struct
	parser := config.NewParser(configMap)
	parsedConfig, err := parser.ParseConfig()
	if err != nil {
		return fmt.Errorf("configuration parsing failed: %w", err)
	}

	// Merge with defaults for any missing values
	parsedConfig.MergeWithDefaults()

	// Validate configuration
	if err := parsedConfig.Validate(); err != nil {
		return fmt.Errorf("configuration validation failed: %w", err)
	}

	p.config = parsedConfig

	// Initialize logger first so we can log everything else
	if p.config.Logging.Enabled {
		loggerConfig := logging.Config{
			LogFile:   p.config.Logging.LogFile,
			Level:     p.config.Logging.Level,
			Component: p.config.Logging.Component,
		}
		p.logger, err = logging.NewLogger(loggerConfig)
		if err != nil {
			return fmt.Errorf("failed to initialize logger: %w", err)
		}
		// Set as global logger
		logging.SetGlobalLogger(p.logger)
		p.logger.Info("plugin", "Observability plugin logger initialized")
		p.logger.Debug("plugin", "Logger config: file=%s, level=%s", loggerConfig.LogFile, loggerConfig.Level)
	} else {
		// Use a fallback logger that logs to stdout only
		p.logger = logging.GetGlobalLogger()
		p.logger.Info("plugin", "Using fallback logger (logging disabled in config)")
	}

	p.logger.Info("plugin", "Starting observability plugin initialization")
	p.logger.Debug("plugin", "Configuration: Prometheus endpoint=%s, timeout=%v",
		p.config.Prometheus.Endpoint, p.config.Prometheus.Timeout)

	// Initialize all plugin components
	initManager := initialization.NewManager(p.config)
	p.logger.Debug("plugin", "Initializing plugin components with manager")
	components, err := initManager.InitializeComponents()
	if err != nil {
		p.logger.Error("plugin", "Component initialization failed: %v", err)
		return fmt.Errorf("component initialization failed: %w", err)
	}

	p.components = components
	p.logger.Info("plugin", "Plugin initialization completed successfully")
	return nil
}

// Start starts the plugin
func (p *ObservabilityPlugin) Start(ctx context.Context) error {
	p.logger.Info("plugin", "Starting observability plugin")
	p.running = true

	// Test Prometheus connection
	p.logger.Debug("plugin", "Testing Prometheus connection to %s", p.config.Prometheus.Endpoint)
	if err := p.components.Client.TestConnection(ctx); err != nil {
		p.logger.Error("plugin", "Prometheus health check failed: %v", err)
		return fmt.Errorf("prometheus health check failed: %w", err)
	}
	p.logger.Info("plugin", "Prometheus connection test successful")

	// Start metrics collector
	p.logger.Debug("plugin", "Starting metrics collector")
	if err := p.components.MetricsCollector.Start(); err != nil {
		p.logger.Error("plugin", "Failed to start metrics collector: %v", err)
		return fmt.Errorf("failed to start metrics collector: %w", err)
	}
	p.logger.Debug("plugin", "Metrics collector started successfully")

	// Start overlay manager
	p.logger.Debug("plugin", "Starting overlay manager")
	if err := p.components.OverlayMgr.Start(ctx); err != nil {
		p.logger.Error("plugin", "Failed to start overlay manager: %v", err)
		return fmt.Errorf("failed to start overlay manager: %w", err)
	}
	p.logger.Debug("plugin", "Overlay manager started successfully")

	// Start subscription manager
	p.logger.Debug("plugin", "Starting subscription manager")
	if err := p.components.SubscriptionMgr.Start(ctx); err != nil {
		p.logger.Error("plugin", "Failed to start subscription manager: %v", err)
		return fmt.Errorf("failed to start subscription manager: %w", err)
	}
	p.logger.Debug("plugin", "Subscription manager started successfully")

	// Start historical data collector
	p.logger.Debug("plugin", "Starting historical data collector")
	if err := p.components.HistoricalCollector.Start(ctx); err != nil {
		p.logger.Error("plugin", "Failed to start historical data collector: %v", err)
		return fmt.Errorf("failed to start historical data collector: %w", err)
	}
	p.logger.Debug("plugin", "Historical data collector started successfully")

	// Start external API if enabled and initialized
	if p.components.ExternalAPI != nil {
		p.logger.Debug("plugin", "Starting external API")
		if err := p.components.ExternalAPI.Start(ctx); err != nil {
			p.logger.Error("plugin", "Failed to start external API: %v", err)
			return fmt.Errorf("failed to start external API: %w", err)
		}
		p.logger.Debug("plugin", "External API started successfully")
	} else {
		p.logger.Debug("plugin", "External API not configured, skipping")
	}

	p.logger.Info("plugin", "Observability plugin started successfully")
	return nil
}

// SetSlurmClient sets the SLURM client for job queries
func (p *ObservabilityPlugin) SetSlurmClient(client interface{}) {
	p.slurmClient = client
	// Pass to view if it exists
	if p.view != nil {
		p.view.SetSlurmClient(client)
	}
}

// Stop stops the plugin
func (p *ObservabilityPlugin) Stop(ctx context.Context) error {
	p.logger.Info("plugin", "Stopping observability plugin")
	p.running = false

	// Stop external API
	if p.components.ExternalAPI != nil {
		p.logger.Debug("plugin", "Stopping external API")
		if err := p.components.ExternalAPI.Stop(ctx); err != nil {
			p.logger.Error("plugin", "Error stopping external API: %v", err)
		} else {
			p.logger.Debug("plugin", "External API stopped successfully")
		}
	}

	// Stop all other components using the Components.Stop method
	if p.components != nil {
		p.logger.Debug("plugin", "Stopping plugin components")
		if err := p.components.Stop(); err != nil {
			p.logger.Error("plugin", "Error stopping components: %v", err)
		} else {
			p.logger.Debug("plugin", "Components stopped successfully")
		}
	}

	if p.view != nil {
		p.logger.Debug("plugin", "Stopping observability view")
		if err := p.view.Stop(ctx); err != nil {
			p.logger.Error("plugin", "Failed to stop view: %v", err)
			return fmt.Errorf("failed to stop view: %w", err)
		}
		p.logger.Debug("plugin", "View stopped successfully")
	}

	// Close logger
	if p.logger != nil {
		p.logger.Info("plugin", "Observability plugin stopped successfully")
		if p.config.Logging.Enabled {
			_ = p.logger.Close()
		}
	}

	return nil
}

// Health returns the plugin health status
func (p *ObservabilityPlugin) Health() plugin.HealthStatus {
	if !p.running {
		return plugin.HealthStatus{
			Healthy: false,
			Status:  "stopped",
			Message: "Plugin is not running",
		}
	}

	// Check Prometheus connectivity
	ctx, cancel := context.WithTimeout(context.Background(), p.config.Prometheus.Timeout)
	defer cancel()

	if err := p.components.Client.TestConnection(ctx); err != nil {
		return plugin.HealthStatus{
			Healthy: false,
			Status:  "unhealthy",
			Message: fmt.Sprintf("Prometheus connection failed: %v", err),
			Details: map[string]interface{}{
				"endpoint": p.config.Prometheus.Endpoint,
				"error":    err.Error(),
			},
		}
	}

	// Get plugin internal metrics
	var pluginMetrics map[string]interface{}
	if p.components.MetricsCollector != nil {
		pluginMetrics = p.components.MetricsCollector.GetMetrics().GetAllStats()
	}

	// Get secrets manager health
	var secretsHealth map[string]interface{}
	if p.components.SecretsManager != nil {
		secretsHealth = p.components.SecretsManager.Health()
	}

	return plugin.HealthStatus{
		Healthy: true,
		Status:  "healthy",
		Message: "Plugin is running and connected to Prometheus",
		Details: map[string]interface{}{
			"endpoint":           p.config.Prometheus.Endpoint,
			"cache_stats":        p.components.CachedClient.CacheStats(),
			"view_active":        p.view != nil,
			"subscription_stats": p.components.SubscriptionMgr.GetStats(),
			"notification_stats": p.components.NotificationMgr.GetStats(),
			"historical_stats":   p.components.HistoricalCollector.GetCollectorStats(),
			"plugin_metrics":     pluginMetrics,
			"secrets_health":     secretsHealth,
		},
	}
}

// ViewPlugin interface implementation

// GetViews returns the views provided by this plugin
func (p *ObservabilityPlugin) GetViews() []plugin.ViewInfo {
	return []plugin.ViewInfo{
		{
			ID:          "observability",
			Name:        "Observability",
			Description: "Real-time metrics and monitoring dashboard",
			Icon:        "📊",
			Shortcut:    "o",
			Category:    "monitoring",
		},
	}
}

// CreateView creates a view instance
func (p *ObservabilityPlugin) CreateView(ctx context.Context, viewID string) (plugin.View, error) {
	p.logger.Debug("plugin", "CreateView called with viewID: %s", viewID)

	if viewID != "observability" {
		p.logger.Error("plugin", "Unknown view requested: %s", viewID)
		return nil, fmt.Errorf("unknown view: %s", viewID)
	}

	// Get the tview app from context
	app, ok := ctx.Value("app").(*tview.Application)
	if !ok {
		p.logger.Error("plugin", "tview application not found in context")
		return nil, fmt.Errorf("tview application not found in context")
	}
	p.app = app
	p.logger.Debug("plugin", "Retrieved tview application from context")

	// Create the observability view
	p.logger.Debug("plugin", "Creating observability view with cached client and config")
	p.view = views.NewObservabilityView(app, p.components.CachedClient, p.config)

	// Pass SLURM client if available
	if p.slurmClient != nil {
		p.logger.Debug("plugin", "Setting SLURM client in view")
		p.view.SetSlurmClient(p.slurmClient)
	}

	p.logger.Info("plugin", "Observability view created successfully")

	return p.view, nil
}

// OverlayPlugin interface implementation

// GetOverlays returns the overlays provided by this plugin
func (p *ObservabilityPlugin) GetOverlays() []plugin.OverlayInfo {
	if !p.config.Display.ShowOverlays {
		return []plugin.OverlayInfo{}
	}

	return []plugin.OverlayInfo{
		{
			ID:          "jobs-metrics",
			Name:        "Job Metrics",
			Description: "Adds real-time CPU and memory usage to jobs view",
			TargetViews: []string{"jobs"},
			Priority:    100,
		},
		{
			ID:          "nodes-metrics",
			Name:        "Node Metrics",
			Description: "Adds real-time resource utilization to nodes view",
			TargetViews: []string{"nodes"},
			Priority:    100,
		},
	}
}

// CreateOverlay creates an overlay instance
func (p *ObservabilityPlugin) CreateOverlay(ctx context.Context, overlayID string) (plugin.Overlay, error) {
	var overlay plugin.Overlay
	var overlayInfo plugin.OverlayInfo
	var err error

	switch overlayID {
	case "jobs-metrics":
		jobsOverlay := overlays.NewJobsOverlay(p.components.CachedClient, p.config.Metrics.Job.CgroupPattern)
		if err = jobsOverlay.Initialize(ctx); err != nil {
			return nil, fmt.Errorf("failed to initialize jobs overlay: %w", err)
		}
		overlay = jobsOverlay
		overlayInfo = plugin.OverlayInfo{
			ID:          "jobs-metrics",
			Name:        "Job Metrics",
			Description: "Adds real-time CPU and memory usage to jobs view",
			TargetViews: []string{"jobs"},
			Priority:    100,
		}
	case "nodes-metrics":
		nodesOverlay := overlays.NewNodesOverlay(p.components.CachedClient, p.config.Metrics.Node.NodeLabel)
		if err = nodesOverlay.Initialize(ctx); err != nil {
			return nil, fmt.Errorf("failed to initialize nodes overlay: %w", err)
		}
		overlay = nodesOverlay
		overlayInfo = plugin.OverlayInfo{
			ID:          "nodes-metrics",
			Name:        "Node Metrics",
			Description: "Adds real-time resource utilization to nodes view",
			TargetViews: []string{"nodes"},
			Priority:    100,
		}
	default:
		return nil, fmt.Errorf("unknown overlay: %s", overlayID)
	}

	// Register with overlay manager
	if p.components.OverlayMgr != nil {
		if err := p.components.OverlayMgr.RegisterOverlay(overlayInfo, overlay); err != nil {
			return nil, fmt.Errorf("failed to register overlay with manager: %w", err)
		}
	}

	return overlay, nil
}

// DataPlugin interface implementation

// GetDataProviders returns the data providers offered
func (p *ObservabilityPlugin) GetDataProviders() []plugin.DataProviderInfo {
	return []plugin.DataProviderInfo{
		{
			ID:          "prometheus-metrics",
			Name:        "Prometheus Metrics",
			Description: "Real-time metrics from Prometheus",
		},
		{
			ID:          "alerts",
			Name:        "Active Alerts",
			Description: "Active monitoring alerts",
		},
		{
			ID:          "historical-data",
			Name:        "Historical Data",
			Description: "Historical metric data with time series analysis",
		},
		{
			ID:          "trend-analysis",
			Name:        "Trend Analysis",
			Description: "Statistical trend analysis of historical data",
		},
		{
			ID:          "anomaly-detection",
			Name:        "Anomaly Detection",
			Description: "Automated anomaly detection in metric data",
		},
		{
			ID:          "seasonal-analysis",
			Name:        "Seasonal Analysis",
			Description: "Seasonal pattern analysis for metric data",
		},
		{
			ID:          "resource-efficiency",
			Name:        "Resource Efficiency Analysis",
			Description: "Comprehensive resource utilization and efficiency analysis with optimization recommendations",
		},
		{
			ID:          "cluster-efficiency",
			Name:        "Cluster Efficiency Analysis",
			Description: "Overall cluster efficiency analysis with cost optimization insights",
		},
		{
			ID:          "plugin-metrics",
			Name:        "Plugin Internal Metrics",
			Description: "Internal metrics and performance data for the observability plugin itself",
		},
		{
			ID:          "secrets-manager",
			Name:        "Secrets Manager Status",
			Description: "Status and health information for the secrets management system",
		},
	}
}

// Query performs a one-time data query
func (p *ObservabilityPlugin) Query(ctx context.Context, providerID string, params map[string]interface{}) (interface{}, error) {
	switch providerID {
	case "prometheus-metrics", "alerts", "node-metrics", "job-metrics":
		if p.components.SubscriptionMgr == nil {
			return nil, fmt.Errorf("subscription manager not initialized")
		}
		return p.components.SubscriptionMgr.GetData(ctx, providerID, params)

	case "historical-data":
		return p.queryHistoricalData(params)

	case "trend-analysis":
		return p.queryTrendAnalysis(params)

	case "anomaly-detection":
		return p.queryAnomalyDetection(params)

	case "seasonal-analysis":
		return p.querySeasonalAnalysis(params)

	case "resource-efficiency":
		return p.queryResourceEfficiency(params)

	case "cluster-efficiency":
		return p.queryClusterEfficiency(params)

	case "plugin-metrics":
		return p.queryPluginMetrics(params)

	case "secrets-manager":
		return p.querySecretsManager(params)

	default:
		return nil, fmt.Errorf("unknown data provider: %s", providerID)
	}
}

// Subscribe allows other plugins to subscribe to data updates
func (p *ObservabilityPlugin) Subscribe(_ context.Context, providerID string, callback plugin.DataCallback) (plugin.SubscriptionID, error) {
	if p.components.SubscriptionMgr == nil {
		return "", fmt.Errorf("subscription manager not initialized")
	}

	// Default parameters for subscription
	params := map[string]interface{}{
		"update_interval": p.config.Display.RefreshInterval.String(),
	}

	// Create enhanced callback with notifications
	enhancedCallback := subscription.NewEnhancedSubscriptionCallback(
		callback,
		"", // Will be set after subscription is created
		providerID,
		p.components.NotificationMgr,
	)

	subscriptionID, err := p.components.SubscriptionMgr.Subscribe(providerID, params, enhancedCallback.Call)
	if err != nil {
		return "", fmt.Errorf("failed to create subscription: %w", err)
	}

	// Update the enhanced callback with the subscription ID
	enhancedCallback.SetSubscriptionID(string(subscriptionID))

	return subscriptionID, nil
}

// Unsubscribe removes a data subscription
func (p *ObservabilityPlugin) Unsubscribe(_ context.Context, subscriptionID plugin.SubscriptionID) error {
	if p.components.SubscriptionMgr == nil {
		return fmt.Errorf("subscription manager not initialized")
	}

	if err := p.components.SubscriptionMgr.Unsubscribe(subscriptionID); err != nil {
		return fmt.Errorf("failed to unsubscribe: %w", err)
	}

	// Remove from persistence
	if p.components.Persistence != nil {
		_ = p.components.Persistence.DeleteSubscription(string(subscriptionID))
	}

	return nil
}

// ConfigurablePlugin interface implementation

// ValidateConfig validates configuration changes
func (p *ObservabilityPlugin) ValidateConfig(_ map[string]interface{}) error {
	// TODO: Implement configuration validation
	return nil
}

// UpdateConfig updates the plugin configuration at runtime
func (p *ObservabilityPlugin) UpdateConfig(_ context.Context, _ map[string]interface{}) error {
	// TODO: Implement configuration update
	return fmt.Errorf("configuration update not yet implemented")
}

// GetConfigSchema returns the configuration schema
func (p *ObservabilityPlugin) GetConfigSchema() map[string]plugin.ConfigField {
	return p.GetInfo().ConfigSchema
}

// GetCurrentConfig returns the current configuration
func (p *ObservabilityPlugin) GetCurrentConfig() map[string]interface{} {
	return map[string]interface{}{
		"prometheus.endpoint":               p.config.Prometheus.Endpoint,
		"prometheus.timeout":                p.config.Prometheus.Timeout.String(),
		"prometheus.auth.type":              p.config.Prometheus.Auth.Type,
		"prometheus.tls.insecureSkipVerify": p.config.Prometheus.TLS.InsecureSkipVerify,
		"display.refreshInterval":           p.config.Display.RefreshInterval.String(),
		"display.showOverlays":              p.config.Display.ShowOverlays,
		"display.showSparklines":            p.config.Display.ShowSparklines,
		"display.colorScheme":               p.config.Display.ColorScheme,
		"alerts.enabled":                    p.config.Alerts.Enabled,
		"alerts.checkInterval":              p.config.Alerts.CheckInterval.String(),
		"alerts.showNotifications":          p.config.Alerts.ShowNotifications,
		"cache.enabled":                     p.config.Cache.Enabled,
		"cache.defaultTTL":                  p.config.Cache.DefaultTTL.String(),
		"cache.maxSize":                     p.config.Cache.MaxSize,
	}
}

// Historical data query methods

// queryHistoricalData handles historical data queries
func (p *ObservabilityPlugin) queryHistoricalData(params map[string]interface{}) (interface{}, error) {
	if p.components.HistoricalCollector == nil {
		return nil, fmt.Errorf("historical collector not initialized")
	}

	metricName, ok := params["metric_name"].(string)
	if !ok {
		return nil, fmt.Errorf("metric_name parameter is required")
	}

	// Parse time range parameters
	duration, err := p.parseDurationParam(params, "duration", 24*time.Hour)
	if err != nil {
		return nil, fmt.Errorf("invalid duration parameter: %w", err)
	}

	end := time.Now()
	start := end.Add(-duration)

	// Override with explicit start/end if provided
	if startStr, ok := params["start"].(string); ok {
		if parsedStart, err := time.Parse(time.RFC3339, startStr); err == nil {
			start = parsedStart
		}
	}

	if endStr, ok := params["end"].(string); ok {
		if parsedEnd, err := time.Parse(time.RFC3339, endStr); err == nil {
			end = parsedEnd
		}
	}

	return p.components.HistoricalCollector.GetHistoricalData(metricName, start, end)
}

// queryTrendAnalysis handles trend analysis queries
func (p *ObservabilityPlugin) queryTrendAnalysis(params map[string]interface{}) (interface{}, error) {
	if p.components.HistoricalAnalyzer == nil {
		return nil, fmt.Errorf("historical analyzer not initialized")
	}

	metricName, ok := params["metric_name"].(string)
	if !ok {
		return nil, fmt.Errorf("metric_name parameter is required")
	}

	duration, err := p.parseDurationParam(params, "duration", 24*time.Hour)
	if err != nil {
		return nil, fmt.Errorf("invalid duration parameter: %w", err)
	}

	return p.components.HistoricalAnalyzer.AnalyzeTrend(metricName, duration)
}

// queryAnomalyDetection handles anomaly detection queries
func (p *ObservabilityPlugin) queryAnomalyDetection(params map[string]interface{}) (interface{}, error) {
	if p.components.HistoricalAnalyzer == nil {
		return nil, fmt.Errorf("historical analyzer not initialized")
	}

	metricName, ok := params["metric_name"].(string)
	if !ok {
		return nil, fmt.Errorf("metric_name parameter is required")
	}

	duration, err := p.parseDurationParam(params, "duration", 24*time.Hour)
	if err != nil {
		return nil, fmt.Errorf("invalid duration parameter: %w", err)
	}

	// Parse sensitivity parameter
	sensitivity := 2.0 // Default
	if s, ok := params["sensitivity"].(float64); ok {
		sensitivity = s
	} else if s, ok := params["sensitivity"].(string); ok {
		if parsed, err := strconv.ParseFloat(s, 64); err == nil {
			sensitivity = parsed
		}
	}

	return p.components.HistoricalAnalyzer.DetectAnomalies(metricName, duration, sensitivity)
}

// querySeasonalAnalysis handles seasonal analysis queries
func (p *ObservabilityPlugin) querySeasonalAnalysis(params map[string]interface{}) (interface{}, error) {
	if p.components.HistoricalAnalyzer == nil {
		return nil, fmt.Errorf("historical analyzer not initialized")
	}

	metricName, ok := params["metric_name"].(string)
	if !ok {
		return nil, fmt.Errorf("metric_name parameter is required")
	}

	duration, err := p.parseDurationParam(params, "duration", 7*24*time.Hour) // Default to 1 week
	if err != nil {
		return nil, fmt.Errorf("invalid duration parameter: %w", err)
	}

	return p.components.HistoricalAnalyzer.AnalyzeSeasonality(metricName, duration)
}

// parseDurationParam parses a duration parameter from the params map
func (p *ObservabilityPlugin) parseDurationParam(params map[string]interface{}, key string, defaultValue time.Duration) (time.Duration, error) {
	if val, ok := params[key]; ok {
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
		}
	}
	return defaultValue, nil
}

// queryResourceEfficiency handles resource efficiency analysis queries
func (p *ObservabilityPlugin) queryResourceEfficiency(params map[string]interface{}) (interface{}, error) {
	if p.components.EfficiencyAnalyzer == nil {
		return nil, fmt.Errorf("efficiency analyzer not initialized")
	}

	resourceType, ok := params["resource_type"].(string)
	if !ok {
		return nil, fmt.Errorf("resource_type parameter is required")
	}

	duration, err := p.parseDurationParam(params, "analysis_period", 7*24*time.Hour) // Default to 1 week
	if err != nil {
		return nil, fmt.Errorf("invalid analysis_period parameter: %w", err)
	}

	// Convert string to ResourceType
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
		return nil, fmt.Errorf("unsupported resource type: %s", resourceType)
	}

	ctx := context.Background()
	return p.components.EfficiencyAnalyzer.AnalyzeResourceEfficiency(ctx, resType, duration)
}

// queryClusterEfficiency handles cluster-wide efficiency analysis queries
func (p *ObservabilityPlugin) queryClusterEfficiency(params map[string]interface{}) (interface{}, error) {
	if p.components.EfficiencyAnalyzer == nil {
		return nil, fmt.Errorf("efficiency analyzer not initialized")
	}

	duration, err := p.parseDurationParam(params, "analysis_period", 7*24*time.Hour) // Default to 1 week
	if err != nil {
		return nil, fmt.Errorf("invalid analysis_period parameter: %w", err)
	}

	ctx := context.Background()
	return p.components.EfficiencyAnalyzer.AnalyzeClusterEfficiency(ctx, duration)
}

// queryPluginMetrics handles plugin internal metrics queries
func (p *ObservabilityPlugin) queryPluginMetrics(params map[string]interface{}) (interface{}, error) {
	if p.components.MetricsCollector == nil {
		return nil, fmt.Errorf("metrics collector not initialized")
	}

	// Check if specific metric category is requested
	category, ok := params["category"].(string)
	if ok {
		switch category {
		case "connections":
			return p.components.MetricsCollector.GetMetrics().GetConnectionStats(), nil
		case "queries":
			return p.components.MetricsCollector.GetMetrics().GetQueryStats(), nil
		case "components":
			return p.components.MetricsCollector.GetMetrics().GetComponentStats(), nil
		case "resources":
			return p.components.MetricsCollector.GetMetrics().GetResourceStats(), nil
		case "performance":
			return p.components.MetricsCollector.GetMetrics().GetPerformanceStats(), nil
		case "health":
			return p.components.MetricsCollector.GetMetrics().GetHealthStats(), nil
		default:
			return nil, fmt.Errorf("unknown metrics category: %s", category)
		}
	}

	// Return all metrics if no category specified
	return p.components.MetricsCollector.GetMetrics().GetAllStats(), nil
}

// querySecretsManager handles secrets manager status queries
func (p *ObservabilityPlugin) querySecretsManager(params map[string]interface{}) (interface{}, error) {
	if p.components.SecretsManager == nil {
		return nil, fmt.Errorf("secrets manager not initialized")
	}

	// Check if specific operation is requested
	operation, ok := params["operation"].(string)
	if ok {
		switch operation {
		case "health":
			return p.components.SecretsManager.Health(), nil
		case "list":
			return p.components.SecretsManager.ListSecrets(), nil
		case "audit":
			return p.components.SecretsManager.GetAuditLog(), nil
		default:
			return nil, fmt.Errorf("unknown secrets manager operation: %s", operation)
		}
	}

	// Return comprehensive status if no operation specified
	return map[string]interface{}{
		"health":    p.components.SecretsManager.Health(),
		"secrets":   p.components.SecretsManager.ListSecrets(),
		"audit_log": p.components.SecretsManager.GetAuditLog(),
	}, nil
}
