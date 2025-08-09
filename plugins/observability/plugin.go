package observability

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/rivo/tview"

	"github.com/jontk/s9s/internal/plugin"
	"github.com/jontk/s9s/plugins/observability/analysis"
	"github.com/jontk/s9s/plugins/observability/api"
	"github.com/jontk/s9s/plugins/observability/config"
	"github.com/jontk/s9s/plugins/observability/historical"
	"github.com/jontk/s9s/plugins/observability/overlays"
	"github.com/jontk/s9s/plugins/observability/prometheus"
	"github.com/jontk/s9s/plugins/observability/subscription"
	"github.com/jontk/s9s/plugins/observability/views"
)

// ObservabilityPlugin implements the observability plugin
type ObservabilityPlugin struct {
	config        *config.Config
	client        *prometheus.Client
	cachedClient  *prometheus.CachedClient
	overlayMgr    *overlays.OverlayManager
	subscriptionMgr *subscription.SubscriptionManager
	notificationMgr *subscription.NotificationManager
	persistence     *subscription.SubscriptionPersistence
	historicalCollector *historical.HistoricalDataCollector
	historicalAnalyzer  *historical.HistoricalAnalyzer
	efficiencyAnalyzer  *analysis.ResourceEfficiencyAnalyzer
	externalAPI   *api.ExternalAPI
	app           *tview.Application
	view          *views.ObservabilityView
	running       bool
}

// New creates a new observability plugin instance
func New() *ObservabilityPlugin {
	return &ObservabilityPlugin{
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
func (p *ObservabilityPlugin) Init(ctx context.Context, config map[string]interface{}) error {
	// Parse configuration from map into Config struct
	if err := p.parseConfig(config); err != nil {
		return fmt.Errorf("configuration parsing failed: %w", err)
	}

	// Merge with defaults for any missing values
	p.config.MergeWithDefaults()

	// Validate configuration
	if err := p.config.Validate(); err != nil {
		return fmt.Errorf("configuration validation failed: %w", err)
	}

	// Create Prometheus client with full configuration
	clientConfig := prometheus.ClientConfig{
		Endpoint:      p.config.Prometheus.Endpoint,
		Timeout:       p.config.Prometheus.Timeout,
		TLSSkipVerify: p.config.Prometheus.TLS.InsecureSkipVerify,
		TLSCertFile:   p.config.Prometheus.TLS.CertFile,
		TLSKeyFile:    p.config.Prometheus.TLS.KeyFile,
		TLSCAFile:     p.config.Prometheus.TLS.CAFile,
	}

	// Add authentication configuration
	switch p.config.Prometheus.Auth.Type {
	case "basic":
		clientConfig.Username = p.config.Prometheus.Auth.Username
		clientConfig.Password = p.config.Prometheus.Auth.Password
	case "bearer":
		clientConfig.BearerToken = p.config.Prometheus.Auth.Token
	}

	client, err := prometheus.NewClient(clientConfig)
	if err != nil {
		return fmt.Errorf("failed to create Prometheus client: %w", err)
	}
	p.client = client

	// Create cached client
	p.cachedClient = prometheus.NewCachedClient(
		client,
		p.config.Cache.DefaultTTL,
		p.config.Cache.MaxSize,
	)

	// Create overlay manager
	p.overlayMgr = overlays.NewOverlayManager(
		p.cachedClient,
		p.config.Display.RefreshInterval,
	)

	// Create subscription manager
	p.subscriptionMgr = subscription.NewSubscriptionManager(p.cachedClient)

	// Create notification manager
	p.notificationMgr = subscription.NewNotificationManager(1000)

	// Create persistence manager
	persistenceConfig := subscription.PersistenceConfig{
		DataDir:      "./data/observability",
		AutoSave:     true,
		SaveInterval: 5 * time.Minute,
	}

	var err error
	p.persistence, err = subscription.NewSubscriptionPersistence(persistenceConfig, p.subscriptionMgr)
	if err != nil {
		return fmt.Errorf("failed to create subscription persistence: %w", err)
	}

	// Load persisted subscriptions
	if err := p.persistence.LoadSubscriptions(); err != nil {
		// Log error but don't fail initialization
	}

	// Create historical data collector
	historicalConfig := historical.CollectorConfig{
		DataDir:         "./data/historical",
		Retention:       30 * 24 * time.Hour, // 30 days
		CollectInterval: 5 * time.Minute,
		MaxDataPoints:   10000,
		Queries:         historical.DefaultCollectorConfig().Queries,
	}

	p.historicalCollector, err = historical.NewHistoricalDataCollector(p.cachedClient, historicalConfig)
	if err != nil {
		return fmt.Errorf("failed to create historical data collector: %w", err)
	}

	// Create historical analyzer
	p.historicalAnalyzer = historical.NewHistoricalAnalyzer(p.historicalCollector)

	// Create efficiency analyzer
	p.efficiencyAnalyzer = analysis.NewResourceEfficiencyAnalyzer(p.cachedClient, p.historicalCollector, p.historicalAnalyzer)

	// Create external API
	apiConfig := api.Config{
		Enabled:   false, // Disabled by default, can be configured
		Port:      8080,
		AuthToken: "",
	}
	
	// Check for API configuration
	if val, ok := config["api.enabled"]; ok {
		if b, ok := val.(bool); ok {
			apiConfig.Enabled = b
		}
	}
	if val, ok := config["api.port"]; ok {
		if port, ok := val.(int); ok {
			apiConfig.Port = port
		} else if portStr, ok := val.(string); ok {
			if port, err := strconv.Atoi(portStr); err == nil {
				apiConfig.Port = port
			}
		}
	}
	if val, ok := config["api.auth_token"]; ok {
		if token, ok := val.(string); ok {
			apiConfig.AuthToken = token
		}
	}

	p.externalAPI = api.NewExternalAPI(
		p.cachedClient,
		p.subscriptionMgr,
		p.historicalCollector,
		p.historicalAnalyzer,
		p.efficiencyAnalyzer,
		apiConfig,
	)

	return nil
}

// Start starts the plugin
func (p *ObservabilityPlugin) Start(ctx context.Context) error {
	p.running = true

	// Test Prometheus connection
	if err := p.client.TestConnection(ctx); err != nil {
		return fmt.Errorf("Prometheus health check failed: %w", err)
	}

	// Start overlay manager
	if err := p.overlayMgr.Start(ctx); err != nil {
		return fmt.Errorf("failed to start overlay manager: %w", err)
	}

	// Start subscription manager
	if err := p.subscriptionMgr.Start(ctx); err != nil {
		return fmt.Errorf("failed to start subscription manager: %w", err)
	}

	// Start historical data collector
	if err := p.historicalCollector.Start(ctx); err != nil {
		return fmt.Errorf("failed to start historical data collector: %w", err)
	}

	// Start external API if enabled
	if err := p.externalAPI.Start(ctx); err != nil {
		return fmt.Errorf("failed to start external API: %w", err)
	}

	return nil
}

// Stop stops the plugin
func (p *ObservabilityPlugin) Stop(ctx context.Context) error {
	p.running = false

	// Stop external API
	if p.externalAPI != nil {
		if err := p.externalAPI.Stop(ctx); err != nil {
			// Log error but don't fail the stop operation
		}
	}

	// Stop historical data collector
	if p.historicalCollector != nil {
		if err := p.historicalCollector.Stop(); err != nil {
			// Log error but don't fail the stop operation
		}
	}

	// Stop subscription manager
	if p.subscriptionMgr != nil {
		if err := p.subscriptionMgr.Stop(); err != nil {
			// Log error but don't fail the stop operation
		}
	}

	// Stop persistence (save final state)
	if p.persistence != nil {
		p.persistence.SaveSubscriptions()
		p.persistence.Stop()
	}

	// Stop overlay manager
	if p.overlayMgr != nil {
		if err := p.overlayMgr.Stop(); err != nil {
			// Log error but don't fail the stop operation
		}
	}

	if p.view != nil {
		if err := p.view.Stop(ctx); err != nil {
			return fmt.Errorf("failed to stop view: %w", err)
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

	if err := p.client.TestConnection(ctx); err != nil {
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

	return plugin.HealthStatus{
		Healthy: true,
		Status:  "healthy",
		Message: "Plugin is running and connected to Prometheus",
		Details: map[string]interface{}{
			"endpoint":           p.config.Prometheus.Endpoint,
			"cache_stats":        p.cachedClient.CacheStats(),
			"view_active":        p.view != nil,
			"subscription_stats": p.subscriptionMgr.GetStats(),
			"notification_stats": p.notificationMgr.GetStats(),
			"historical_stats":   p.historicalCollector.GetCollectorStats(),
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
	if viewID != "observability" {
		return nil, fmt.Errorf("unknown view: %s", viewID)
	}

	// Get the tview app from context
	app, ok := ctx.Value("app").(*tview.Application)
	if !ok {
		return nil, fmt.Errorf("tview application not found in context")
	}
	p.app = app

	// Create the observability view
	p.view = views.NewObservabilityView(app, p.cachedClient, p.config)

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
		jobsOverlay := overlays.NewJobsOverlay(p.cachedClient, p.config.Metrics.Job.CgroupPattern)
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
		nodesOverlay := overlays.NewNodesOverlay(p.cachedClient, p.config.Metrics.Node.NodeLabel)
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
	if p.overlayMgr != nil {
		if err := p.overlayMgr.RegisterOverlay(overlayInfo, overlay); err != nil {
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
	}
}

// Query performs a one-time data query
func (p *ObservabilityPlugin) Query(ctx context.Context, providerID string, params map[string]interface{}) (interface{}, error) {
	switch providerID {
	case "prometheus-metrics", "alerts", "node-metrics", "job-metrics":
		if p.subscriptionMgr == nil {
			return nil, fmt.Errorf("subscription manager not initialized")
		}
		return p.subscriptionMgr.GetData(ctx, providerID, params)
	
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
	
	default:
		return nil, fmt.Errorf("unknown data provider: %s", providerID)
	}
}

// Subscribe allows other plugins to subscribe to data updates
func (p *ObservabilityPlugin) Subscribe(ctx context.Context, providerID string, callback plugin.DataCallback) (plugin.SubscriptionID, error) {
	if p.subscriptionMgr == nil {
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
		p.notificationMgr,
	)

	subscriptionID, err := p.subscriptionMgr.Subscribe(providerID, params, enhancedCallback.Call)
	if err != nil {
		return "", fmt.Errorf("failed to create subscription: %w", err)
	}

	// Update the enhanced callback with the subscription ID
	enhancedCallback.subscriptionID = string(subscriptionID)

	return subscriptionID, nil
}

// Unsubscribe removes a data subscription
func (p *ObservabilityPlugin) Unsubscribe(ctx context.Context, subscriptionID plugin.SubscriptionID) error {
	if p.subscriptionMgr == nil {
		return fmt.Errorf("subscription manager not initialized")
	}

	if err := p.subscriptionMgr.Unsubscribe(subscriptionID); err != nil {
		return fmt.Errorf("failed to unsubscribe: %w", err)
	}

	// Remove from persistence
	if p.persistence != nil {
		p.persistence.DeleteSubscription(string(subscriptionID))
	}

	return nil
}

// ConfigurablePlugin interface implementation

// ValidateConfig validates configuration changes
func (p *ObservabilityPlugin) ValidateConfig(config map[string]interface{}) error {
	// TODO: Implement configuration validation
	return nil
}

// UpdateConfig updates the plugin configuration at runtime
func (p *ObservabilityPlugin) UpdateConfig(ctx context.Context, config map[string]interface{}) error {
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
		"prometheus.endpoint": p.config.Prometheus.Endpoint,
		"prometheus.timeout": p.config.Prometheus.Timeout.String(),
		"prometheus.auth.type": p.config.Prometheus.Auth.Type,
		"prometheus.tls.insecureSkipVerify": p.config.Prometheus.TLS.InsecureSkipVerify,
		"display.refreshInterval": p.config.Display.RefreshInterval.String(),
		"display.showOverlays": p.config.Display.ShowOverlays,
		"display.showSparklines": p.config.Display.ShowSparklines,
		"display.colorScheme": p.config.Display.ColorScheme,
		"alerts.enabled": p.config.Alerts.Enabled,
		"alerts.checkInterval": p.config.Alerts.CheckInterval.String(),
		"alerts.showNotifications": p.config.Alerts.ShowNotifications,
		"cache.enabled": p.config.Cache.Enabled,
		"cache.defaultTTL": p.config.Cache.DefaultTTL.String(),
		"cache.maxSize": p.config.Cache.MaxSize,
	}
}

// parseConfig parses configuration from map into Config struct
func (p *ObservabilityPlugin) parseConfig(configMap map[string]interface{}) error {
	// Initialize with default config if not already initialized
	if p.config == nil {
		p.config = config.DefaultConfig()
	}

	// Helper function to get nested values using dot notation
	getValue := func(key string) (interface{}, bool) {
		if val, exists := configMap[key]; exists {
			return val, true
		}
		return nil, false
	}

	// Helper function to parse duration strings
	parseDuration := func(val interface{}) (time.Duration, error) {
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

	// Helper function to parse boolean values
	parseBool := func(val interface{}) (bool, error) {
		switch v := val.(type) {
		case bool:
			return v, nil
		case string:
			return strconv.ParseBool(v)
		default:
			return false, fmt.Errorf("invalid boolean type: %T", val)
		}
	}

	// Helper function to parse integer values
	parseInt := func(val interface{}) (int, error) {
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

	// Parse Prometheus configuration
	if val, ok := getValue("prometheus.endpoint"); ok {
		if str, ok := val.(string); ok {
			p.config.Prometheus.Endpoint = str
		}
	}

	if val, ok := getValue("prometheus.timeout"); ok {
		if duration, err := parseDuration(val); err == nil {
			p.config.Prometheus.Timeout = duration
		}
	}

	// Parse auth configuration
	if val, ok := getValue("prometheus.auth.type"); ok {
		if str, ok := val.(string); ok {
			p.config.Prometheus.Auth.Type = str
		}
	}

	if val, ok := getValue("prometheus.auth.username"); ok {
		if str, ok := val.(string); ok {
			p.config.Prometheus.Auth.Username = str
		}
	}

	if val, ok := getValue("prometheus.auth.password"); ok {
		if str, ok := val.(string); ok {
			p.config.Prometheus.Auth.Password = str
		}
	}

	if val, ok := getValue("prometheus.auth.token"); ok {
		if str, ok := val.(string); ok {
			p.config.Prometheus.Auth.Token = str
		}
	}

	// Parse TLS configuration
	if val, ok := getValue("prometheus.tls.enabled"); ok {
		if b, err := parseBool(val); err == nil {
			p.config.Prometheus.TLS.Enabled = b
		}
	}

	if val, ok := getValue("prometheus.tls.insecureSkipVerify"); ok {
		if b, err := parseBool(val); err == nil {
			p.config.Prometheus.TLS.InsecureSkipVerify = b
		}
	}

	if val, ok := getValue("prometheus.tls.caFile"); ok {
		if str, ok := val.(string); ok {
			p.config.Prometheus.TLS.CAFile = str
		}
	}

	if val, ok := getValue("prometheus.tls.certFile"); ok {
		if str, ok := val.(string); ok {
			p.config.Prometheus.TLS.CertFile = str
		}
	}

	if val, ok := getValue("prometheus.tls.keyFile"); ok {
		if str, ok := val.(string); ok {
			p.config.Prometheus.TLS.KeyFile = str
		}
	}

	// Parse Display configuration
	if val, ok := getValue("display.refreshInterval"); ok {
		if duration, err := parseDuration(val); err == nil {
			p.config.Display.RefreshInterval = duration
		}
	}

	if val, ok := getValue("display.showOverlays"); ok {
		if b, err := parseBool(val); err == nil {
			p.config.Display.ShowOverlays = b
		}
	}

	if val, ok := getValue("display.showSparklines"); ok {
		if b, err := parseBool(val); err == nil {
			p.config.Display.ShowSparklines = b
		}
	}

	if val, ok := getValue("display.sparklinePoints"); ok {
		if i, err := parseInt(val); err == nil {
			p.config.Display.SparklinePoints = i
		}
	}

	if val, ok := getValue("display.colorScheme"); ok {
		if str, ok := val.(string); ok {
			p.config.Display.ColorScheme = str
		}
	}

	if val, ok := getValue("display.decimalPrecision"); ok {
		if i, err := parseInt(val); err == nil {
			p.config.Display.DecimalPrecision = i
		}
	}

	// Parse Alerts configuration
	if val, ok := getValue("alerts.enabled"); ok {
		if b, err := parseBool(val); err == nil {
			p.config.Alerts.Enabled = b
		}
	}

	if val, ok := getValue("alerts.checkInterval"); ok {
		if duration, err := parseDuration(val); err == nil {
			p.config.Alerts.CheckInterval = duration
		}
	}

	if val, ok := getValue("alerts.loadPredefinedRules"); ok {
		if b, err := parseBool(val); err == nil {
			p.config.Alerts.LoadPredefinedRules = b
		}
	}

	if val, ok := getValue("alerts.showNotifications"); ok {
		if b, err := parseBool(val); err == nil {
			p.config.Alerts.ShowNotifications = b
		}
	}

	// Parse Cache configuration
	if val, ok := getValue("cache.enabled"); ok {
		if b, err := parseBool(val); err == nil {
			p.config.Cache.Enabled = b
		}
	}

	if val, ok := getValue("cache.defaultTTL"); ok {
		if duration, err := parseDuration(val); err == nil {
			p.config.Cache.DefaultTTL = duration
		}
	}

	if val, ok := getValue("cache.maxSize"); ok {
		if i, err := parseInt(val); err == nil {
			p.config.Cache.MaxSize = i
		}
	}

	if val, ok := getValue("cache.cleanupInterval"); ok {
		if duration, err := parseDuration(val); err == nil {
			p.config.Cache.CleanupInterval = duration
		}
	}

	// Parse Metrics configuration
	if val, ok := getValue("metrics.node.nodeLabel"); ok {
		if str, ok := val.(string); ok {
			p.config.Metrics.Node.NodeLabel = str
		}
	}

	if val, ok := getValue("metrics.node.rateRange"); ok {
		if str, ok := val.(string); ok {
			p.config.Metrics.Node.RateRange = str
		}
	}

	if val, ok := getValue("metrics.job.enabled"); ok {
		if b, err := parseBool(val); err == nil {
			p.config.Metrics.Job.Enabled = b
		}
	}

	if val, ok := getValue("metrics.job.cgroupPattern"); ok {
		if str, ok := val.(string); ok {
			p.config.Metrics.Job.CgroupPattern = str
		}
	}

	// Parse array configurations
	if val, ok := getValue("metrics.node.enabledMetrics"); ok {
		if arr, ok := val.([]interface{}); ok {
			metrics := make([]string, 0, len(arr))
			for _, item := range arr {
				if str, ok := item.(string); ok {
					metrics = append(metrics, str)
				}
			}
			p.config.Metrics.Node.EnabledMetrics = metrics
		} else if str, ok := val.(string); ok {
			// Support comma-separated string format
			p.config.Metrics.Node.EnabledMetrics = strings.Split(str, ",")
			for i, metric := range p.config.Metrics.Node.EnabledMetrics {
				p.config.Metrics.Node.EnabledMetrics[i] = strings.TrimSpace(metric)
			}
		}
	}

	if val, ok := getValue("metrics.job.enabledMetrics"); ok {
		if arr, ok := val.([]interface{}); ok {
			metrics := make([]string, 0, len(arr))
			for _, item := range arr {
				if str, ok := item.(string); ok {
					metrics = append(metrics, str)
				}
			}
			p.config.Metrics.Job.EnabledMetrics = metrics
		} else if str, ok := val.(string); ok {
			// Support comma-separated string format
			p.config.Metrics.Job.EnabledMetrics = strings.Split(str, ",")
			for i, metric := range p.config.Metrics.Job.EnabledMetrics {
				p.config.Metrics.Job.EnabledMetrics[i] = strings.TrimSpace(metric)
			}
		}
	}

	return nil
}

// Historical data query methods

// queryHistoricalData handles historical data queries
func (p *ObservabilityPlugin) queryHistoricalData(params map[string]interface{}) (interface{}, error) {
	if p.historicalCollector == nil {
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

	return p.historicalCollector.GetHistoricalData(metricName, start, end)
}

// queryTrendAnalysis handles trend analysis queries
func (p *ObservabilityPlugin) queryTrendAnalysis(params map[string]interface{}) (interface{}, error) {
	if p.historicalAnalyzer == nil {
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

	return p.historicalAnalyzer.AnalyzeTrend(metricName, duration)
}

// queryAnomalyDetection handles anomaly detection queries
func (p *ObservabilityPlugin) queryAnomalyDetection(params map[string]interface{}) (interface{}, error) {
	if p.historicalAnalyzer == nil {
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

	return p.historicalAnalyzer.DetectAnomalies(metricName, duration, sensitivity)
}

// querySeasonalAnalysis handles seasonal analysis queries
func (p *ObservabilityPlugin) querySeasonalAnalysis(params map[string]interface{}) (interface{}, error) {
	if p.historicalAnalyzer == nil {
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

	return p.historicalAnalyzer.AnalyzeSeasonality(metricName, duration)
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
	if p.efficiencyAnalyzer == nil {
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
	return p.efficiencyAnalyzer.AnalyzeResourceEfficiency(ctx, resType, duration)
}

// queryClusterEfficiency handles cluster-wide efficiency analysis queries
func (p *ObservabilityPlugin) queryClusterEfficiency(params map[string]interface{}) (interface{}, error) {
	if p.efficiencyAnalyzer == nil {
		return nil, fmt.Errorf("efficiency analyzer not initialized")
	}

	duration, err := p.parseDurationParam(params, "analysis_period", 7*24*time.Hour) // Default to 1 week
	if err != nil {
		return nil, fmt.Errorf("invalid analysis_period parameter: %w", err)
	}

	ctx := context.Background()
	return p.efficiencyAnalyzer.AnalyzeClusterEfficiency(ctx, duration)
}