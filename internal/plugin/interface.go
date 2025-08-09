package plugin

import (
	"context"
	
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// Plugin represents the base interface that all plugins must implement
type Plugin interface {
	// GetInfo returns metadata about the plugin
	GetInfo() Info
	
	// Init initializes the plugin with configuration
	Init(ctx context.Context, config map[string]interface{}) error
	
	// Start starts the plugin's background processes
	Start(ctx context.Context) error
	
	// Stop gracefully stops the plugin
	Stop(ctx context.Context) error
	
	// Health returns the current health status of the plugin
	Health() HealthStatus
}

// Info contains metadata about a plugin
type Info struct {
	Name        string   `json:"name"`
	Version     string   `json:"version"`
	Description string   `json:"description"`
	Author      string   `json:"author"`
	License     string   `json:"license"`
	Requires    []string `json:"requires"`    // Required plugins
	Provides    []string `json:"provides"`    // Capabilities provided
	ConfigSchema map[string]ConfigField `json:"config_schema"`
}

// ConfigField describes a configuration field
type ConfigField struct {
	Type        string      `json:"type"`        // string, int, bool, float, array, object
	Description string      `json:"description"`
	Default     interface{} `json:"default"`
	Required    bool        `json:"required"`
	Validation  string      `json:"validation"`  // Regex or validation rule
}

// HealthStatus represents the health of a plugin
type HealthStatus struct {
	Healthy bool   `json:"healthy"`
	Status  string `json:"status"`  // "healthy", "degraded", "unhealthy"
	Message string `json:"message"`
	Details map[string]interface{} `json:"details,omitempty"`
}

// ViewPlugin represents a plugin that provides custom views
type ViewPlugin interface {
	Plugin
	
	// GetViews returns the views provided by this plugin
	GetViews() []ViewInfo
	
	// CreateView creates a specific view instance
	CreateView(ctx context.Context, viewID string) (View, error)
}

// ViewInfo describes a view provided by a plugin
type ViewInfo struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Icon        string   `json:"icon"`        // Icon character or emoji
	Shortcut    string   `json:"shortcut"`    // Keyboard shortcut
	Category    string   `json:"category"`    // View category (monitoring, management, etc.)
}

// View represents a custom view interface
type View interface {
	// GetName returns the display name
	GetName() string
	
	// GetID returns the unique identifier
	GetID() string
	
	// GetPrimitive returns the tview primitive for rendering
	GetPrimitive() tview.Primitive
	
	// Update refreshes the view data
	Update(ctx context.Context) error
	
	// HandleKey processes keyboard input
	HandleKey(event *tcell.EventKey) bool
	
	// SetFocus sets focus to this view
	SetFocus(app *tview.Application)
	
	// GetHelp returns help text for this view
	GetHelp() string
}

// OverlayPlugin represents a plugin that overlays data on existing views
type OverlayPlugin interface {
	Plugin
	
	// GetOverlays returns the overlays provided by this plugin
	GetOverlays() []OverlayInfo
	
	// CreateOverlay creates a specific overlay instance
	CreateOverlay(ctx context.Context, overlayID string) (Overlay, error)
}

// OverlayInfo describes an overlay provided by a plugin
type OverlayInfo struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	TargetViews []string `json:"target_views"` // Views this overlay applies to
	Priority    int      `json:"priority"`     // Higher priority overlays render last
}

// Overlay represents a data overlay for existing views
type Overlay interface {
	// GetID returns the unique identifier
	GetID() string
	
	// GetColumns returns additional columns to add
	GetColumns() []ColumnDefinition
	
	// GetCellData returns data for a specific cell
	GetCellData(ctx context.Context, viewID string, rowID interface{}, columnID string) (string, error)
	
	// GetCellStyle returns styling for a specific cell
	GetCellStyle(ctx context.Context, viewID string, rowID interface{}, columnID string) CellStyle
	
	// ShouldRefresh indicates if the overlay needs refresh
	ShouldRefresh() bool
}

// ColumnDefinition defines a column to be added by an overlay
type ColumnDefinition struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Width    int    `json:"width"`
	Priority int    `json:"priority"` // Column display priority
	Align    string `json:"align"`    // left, center, right
}

// CellStyle defines styling for a cell
type CellStyle struct {
	Foreground string `json:"foreground"` // Color name or hex
	Background string `json:"background"` // Color name or hex
	Bold       bool   `json:"bold"`
	Italic     bool   `json:"italic"`
	Underline  bool   `json:"underline"`
}

// DataPlugin represents a plugin that provides data to other plugins
type DataPlugin interface {
	Plugin
	
	// GetDataProviders returns the data providers offered
	GetDataProviders() []DataProviderInfo
	
	// Subscribe allows other plugins to subscribe to data updates
	Subscribe(ctx context.Context, providerID string, callback DataCallback) (SubscriptionID, error)
	
	// Unsubscribe removes a data subscription
	Unsubscribe(ctx context.Context, subscriptionID SubscriptionID) error
	
	// Query performs a one-time data query
	Query(ctx context.Context, providerID string, params map[string]interface{}) (interface{}, error)
}

// DataProviderInfo describes a data provider
type DataProviderInfo struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Schema      map[string]interface{} `json:"schema"`      // Data schema
	QueryParams map[string]ConfigField `json:"query_params"` // Available query parameters
}

// DataCallback is called when subscribed data updates
type DataCallback func(data interface{}, error error)

// SubscriptionID represents a data subscription
type SubscriptionID string

// ConfigurablePlugin represents a plugin that can be configured at runtime
type ConfigurablePlugin interface {
	Plugin
	
	// GetConfig returns the current configuration
	GetConfig() map[string]interface{}
	
	// SetConfig updates the configuration
	SetConfig(config map[string]interface{}) error
	
	// ValidateConfig validates a configuration without applying it
	ValidateConfig(config map[string]interface{}) error
	
	// GetConfigUI returns a UI for configuration (optional)
	GetConfigUI() tview.Primitive
}

// HookablePlugin represents a plugin that provides hooks for events
type HookablePlugin interface {
	Plugin
	
	// GetHooks returns the hooks provided by this plugin
	GetHooks() []HookInfo
	
	// RegisterHook registers a callback for a hook
	RegisterHook(hookID string, callback HookCallback) error
}

// HookInfo describes a hook provided by a plugin
type HookInfo struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]ConfigField `json:"parameters"`
}

// HookCallback is called when a hook is triggered
type HookCallback func(ctx context.Context, params map[string]interface{}) error

// LifecycleAware represents a plugin that needs lifecycle events
type LifecycleAware interface {
	// OnEnable is called when the plugin is enabled
	OnEnable(ctx context.Context) error
	
	// OnDisable is called when the plugin is disabled
	OnDisable(ctx context.Context) error
	
	// OnConfigChange is called when configuration changes
	OnConfigChange(ctx context.Context, oldConfig, newConfig map[string]interface{}) error
}

// Prioritizable represents a plugin that has priority ordering
type Prioritizable interface {
	// GetPriority returns the plugin priority (higher = later initialization)
	GetPriority() int
}

// ResourceManager represents a plugin that manages resources
type ResourceManager interface {
	// GetResourceUsage returns current resource usage
	GetResourceUsage() ResourceUsage
	
	// GetResourceLimits returns resource limits
	GetResourceLimits() ResourceLimits
	
	// SetResourceLimits sets resource limits
	SetResourceLimits(limits ResourceLimits) error
}

// ResourceUsage represents current resource usage
type ResourceUsage struct {
	MemoryBytes   int64   `json:"memory_bytes"`
	CPUPercent    float64 `json:"cpu_percent"`
	Goroutines    int     `json:"goroutines"`
	Connections   int     `json:"connections"`
	CacheSize     int64   `json:"cache_size"`
}

// ResourceLimits represents resource limits
type ResourceLimits struct {
	MaxMemoryBytes   int64   `json:"max_memory_bytes"`
	MaxCPUPercent    float64 `json:"max_cpu_percent"`
	MaxGoroutines    int     `json:"max_goroutines"`
	MaxConnections   int     `json:"max_connections"`
	MaxCacheSize     int64   `json:"max_cache_size"`
}