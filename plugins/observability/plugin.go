package observability

import (
	"context"
	"fmt"

	"github.com/rivo/tview"
	
	"github.com/jontk/s9s/internal/plugin"
	"github.com/jontk/s9s/plugins/observability/overlays"
	"github.com/jontk/s9s/plugins/observability/prometheus"
	"github.com/jontk/s9s/plugins/observability/views"
)

// ObservabilityPlugin implements the observability plugin
type ObservabilityPlugin struct {
	config       *Config
	client       *prometheus.Client
	cachedClient *prometheus.CachedClient
	app          *tview.Application
	view         *views.ObservabilityView
	running      bool
}

// New creates a new observability plugin instance
func New() *ObservabilityPlugin {
	return &ObservabilityPlugin{
		config: DefaultConfig(),
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
		Type:        "view,overlay,data",
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
		},
	}
}

// Init initializes the plugin with configuration
func (p *ObservabilityPlugin) Init(ctx context.Context, config map[string]interface{}) error {
	// TODO: Parse configuration from map into Config struct
	// For now, merge with defaults
	p.config.MergeWithDefaults()
	
	// Validate configuration
	if err := p.config.Validate(); err != nil {
		return fmt.Errorf("configuration validation failed: %w", err)
	}
	
	// Create Prometheus client
	clientConfig := prometheus.Config{
		Endpoint: p.config.Prometheus.Endpoint,
		Timeout:  p.config.Prometheus.Timeout,
		// TODO: Add auth configuration
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
	
	return nil
}

// Start starts the plugin
func (p *ObservabilityPlugin) Start(ctx context.Context) error {
	p.running = true
	
	// Test Prometheus connection
	if err := p.client.Health(ctx); err != nil {
		return fmt.Errorf("Prometheus health check failed: %w", err)
	}
	
	return nil
}

// Stop stops the plugin
func (p *ObservabilityPlugin) Stop(ctx context.Context) error {
	p.running = false
	
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
	
	if err := p.client.Health(ctx); err != nil {
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
			"endpoint":     p.config.Prometheus.Endpoint,
			"cache_stats":  p.cachedClient.CacheStats(),
			"view_active":  p.view != nil,
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
	switch overlayID {
	case "jobs-metrics":
		overlay := overlays.NewJobsOverlay(p.cachedClient, p.config.Metrics.Job.CgroupPattern)
		if err := overlay.Initialize(ctx); err != nil {
			return nil, fmt.Errorf("failed to initialize jobs overlay: %w", err)
		}
		return overlay, nil
	case "nodes-metrics":
		overlay := overlays.NewNodesOverlay(p.cachedClient, p.config.Metrics.Node.NodeLabel)
		if err := overlay.Initialize(ctx); err != nil {
			return nil, fmt.Errorf("failed to initialize nodes overlay: %w", err)
		}
		return overlay, nil
	default:
		return nil, fmt.Errorf("unknown overlay: %s", overlayID)
	}
}

// DataPlugin interface implementation

// GetDataSources returns the data sources provided by this plugin
func (p *ObservabilityPlugin) GetDataSources() []plugin.DataSourceInfo {
	return []plugin.DataSourceInfo{
		{
			ID:          "prometheus-metrics",
			Name:        "Prometheus Metrics",
			Description: "Real-time metrics from Prometheus",
			Type:        "metrics",
		},
		{
			ID:          "alerts",
			Name:        "Active Alerts",
			Description: "Active monitoring alerts",
			Type:        "alerts",
		},
	}
}

// QueryData queries data from a data source
func (p *ObservabilityPlugin) QueryData(ctx context.Context, sourceID string, query plugin.DataQuery) (interface{}, error) {
	switch sourceID {
	case "prometheus-metrics":
		// TODO: Implement metrics query
		return nil, fmt.Errorf("metrics query not yet implemented")
	case "alerts":
		// TODO: Implement alerts query
		return nil, fmt.Errorf("alerts query not yet implemented")
	default:
		return nil, fmt.Errorf("unknown data source: %s", sourceID)
	}
}

// SubscribeData subscribes to data updates
func (p *ObservabilityPlugin) SubscribeData(ctx context.Context, sourceID string, query plugin.DataQuery) (<-chan plugin.DataUpdate, error) {
	// TODO: Implement data subscription
	return nil, fmt.Errorf("data subscription not yet implemented")
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
	// TODO: Convert Config struct to map
	return map[string]interface{}{
		"prometheus.endpoint": p.config.Prometheus.Endpoint,
		"display.refreshInterval": p.config.Display.RefreshInterval.String(),
		"display.showOverlays": p.config.Display.ShowOverlays,
		"alerts.enabled": p.config.Alerts.Enabled,
	}
}