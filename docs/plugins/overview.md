# Plugin System Overview

The s9s plugin system allows extending the functionality of s9s through modular plugins. Plugins can provide new views, overlay data on existing views, provide data to other plugins, and more.

## Table of Contents

- [Plugin Architecture](#plugin-architecture)
- [Core Components](#core-components)
- [Creating a Plugin](#creating-a-plugin)
- [Available Plugins](#available-plugins)
- [Plugin Development Guidelines](#plugin-development-guidelines)
- [Testing Plugins](#testing-plugins)

## Plugin Architecture

s9s has two plugin systems:

### 1. Compile-time Plugin System (`internal/plugin/`)

The primary plugin system uses compile-time registration. Plugins implement interfaces from `internal/plugin/interface.go` and are registered at build time.

**Base `Plugin` interface** -- all plugins must implement:
- `GetInfo() Info` -- metadata (name, version, author, dependencies, config schema)
- `Init(ctx, config)` -- initialize with configuration
- `Start(ctx)` / `Stop(ctx)` -- lifecycle management
- `Health() HealthStatus` -- health reporting

**Specialized interfaces:**
- `ViewPlugin` -- provides custom TUI views (`GetViews()`, `CreateView()`)
- `OverlayPlugin` -- overlays data on existing views (`GetOverlays()`, `CreateOverlay()`)
- `DataPlugin` -- provides data to other plugins (`GetDataProviders()`, `Subscribe()`, `Unsubscribe()`, `Query()`)
- `ConfigurablePlugin` -- runtime configuration support (`GetConfig()`, `SetConfig()`, `ValidateConfig()`, `GetConfigUI()`)
- `HookablePlugin` -- event hooks (`GetHooks()`, `RegisterHook()`)
- `LifecycleAware` -- lifecycle events (`OnEnable()`, `OnDisable()`, `OnConfigChange()`)
- `Prioritizable` -- controls initialization order (`GetPriority()`)

**Plugin Manager** (`internal/plugin/manager.go`) -- manages plugin lifecycle, dependencies, and health checks.

**Plugin Registry** (`internal/plugin/registry.go`) -- registers and indexes plugins, resolves dependencies.

### 2. Shared Library Plugin System (`internal/plugins/`)

The secondary system loads `.so` shared libraries at runtime. Plugins implement interfaces from `internal/plugins/interface.go`:

**Base `Plugin` interface:**
- `GetInfo() PluginInfo` -- basic plugin info (name, version, description, author, website)
- `Initialize(ctx, client dao.SlurmClient)` -- initialize with SLURM client access
- `GetCommands() []Command` -- provide CLI commands (Name, Description, Usage, Handler)
- `GetViews() []View` -- provide TUI views (GetName, GetTitle, Render, OnKey, Refresh, Init)
- `GetKeyBindings() []KeyBinding` -- custom key bindings
- `OnEvent(event Event)` -- react to application events (ViewChanged, JobSubmitted, etc.)
- `Cleanup()` -- cleanup on unload

**Plugin Manager** (`internal/plugins/interface.go: PluginManager`) -- loads/unloads `.so` plugins from directories, sends events to all plugins.

## Creating a Plugin

### Basic Plugin Structure

```go
package myplugin

import (
    "context"
    "github.com/jontk/s9s/internal/plugin"
)

type MyPlugin struct {
    // Plugin state
}

func New() *MyPlugin {
    return &MyPlugin{}
}

func (p *MyPlugin) GetInfo() plugin.Info {
    return plugin.Info{
        Name:        "my-plugin",
        Version:     "1.0.0",
        Description: "My awesome plugin",
        Author:      "Your Name",
        License:     "MIT",
        Requires:    []string{}, // Dependencies
        Provides:    []string{"feature1", "feature2"},
    }
}

func (p *MyPlugin) Init(ctx context.Context, config map[string]interface{}) error {
    // Initialize plugin with configuration
    return nil
}

func (p *MyPlugin) Start(ctx context.Context) error {
    // Start plugin services
    return nil
}

func (p *MyPlugin) Stop(ctx context.Context) error {
    // Clean up resources
    return nil
}

func (p *MyPlugin) Health() plugin.HealthStatus {
    return plugin.HealthStatus{
        Healthy: true,
        Status:  "healthy",
        Message: "Plugin is running",
    }
}
```

### View Plugin Example

```go
type MyViewPlugin struct {
    MyPlugin // Embed base plugin
}

func (p *MyViewPlugin) GetViews() []plugin.ViewInfo {
    return []plugin.ViewInfo{
        {
            ID:          "my-view",
            Name:        "My View",
            Description: "Custom view for my data",
            Icon:        "chart",
            Shortcut:    "m",
            Category:    "monitoring",
        },
    }
}

func (p *MyViewPlugin) CreateView(ctx context.Context, viewID string) (plugin.View, error) {
    if viewID == "my-view" {
        return NewMyView(), nil
    }
    return nil, fmt.Errorf("unknown view: %s", viewID)
}
```

### Overlay Plugin Example

```go
type MyOverlayPlugin struct {
    MyPlugin // Embed base plugin
}

func (p *MyOverlayPlugin) GetOverlays() []plugin.OverlayInfo {
    return []plugin.OverlayInfo{
        {
            ID:          "cpu-usage",
            Name:        "CPU Usage",
            Description: "Adds CPU usage to job view",
            TargetViews: []string{"jobs"},
            Priority:    100,
        },
    }
}

func (p *MyOverlayPlugin) CreateOverlay(ctx context.Context, overlayID string) (plugin.Overlay, error) {
    if overlayID == "cpu-usage" {
        return NewCPUOverlay(), nil
    }
    return nil, fmt.Errorf("unknown overlay: %s", overlayID)
}
```

## Plugin Configuration

Plugins can define their configuration schema:

```go
ConfigSchema: map[string]plugin.ConfigField{
    "endpoint": {
        Type:        "string",
        Description: "API endpoint URL",
        Default:     "http://localhost:8080",
        Required:    true,
        Validation:  "^https?://",
    },
    "refresh_rate": {
        Type:        "int",
        Description: "Refresh rate in seconds",
        Default:     30,
        Required:    false,
    },
}
```

Configuration is passed to the plugin during initialization and can be updated at runtime if the plugin implements `ConfigurablePlugin`.

## Plugin Lifecycle

1. **Registration**: Plugin is registered with the manager
2. **Initialization**: `Init()` is called with configuration
3. **Start**: `Start()` is called to begin operations
4. **Running**: Plugin performs its functions
5. **Health Checks**: Manager periodically calls `Health()`
6. **Stop**: `Stop()` is called for graceful shutdown

## Available Plugins

### Observability Plugin

Provides Prometheus integration for real-time metrics monitoring.

**Features:**
- Real-time CPU, memory, disk, and network metrics
- Job-level resource usage from cgroup-exporter
- Metric overlays on existing views
- Alert monitoring
- Historical data analysis

**Configuration:**
```yaml
plugins:
  - name: observability
    enabled: true
    config:
      prometheus:
        endpoint: "http://prometheus:9090"
        timeout: "10s"
        refreshInterval: "30s"
      display:
        showOverlays: true
        enableAlerts: true
```

For detailed documentation on the observability plugin, see [Observability Plugin](./observability.md).

## Plugin Development Guidelines

1. **Error Handling**: Always return meaningful errors
2. **Context Awareness**: Respect context cancellation
3. **Resource Management**: Clean up resources in `Stop()`
4. **Health Reporting**: Provide accurate health status
5. **Configuration Validation**: Validate config in `Init()`
6. **Logging**: Use the debug logger for diagnostics
7. **Testing**: Write unit tests for your plugin

## Testing Plugins

```go
func TestMyPlugin(t *testing.T) {
    plugin := New()

    // Test initialization
    err := plugin.Init(context.Background(), map[string]interface{}{
        "endpoint": "http://test:8080",
    })
    if err != nil {
        t.Fatalf("Init failed: %v", err)
    }

    // Test start
    err = plugin.Start(context.Background())
    if err != nil {
        t.Fatalf("Start failed: %v", err)
    }

    // Test health
    health := plugin.Health()
    if !health.Healthy {
        t.Errorf("Plugin unhealthy: %s", health.Message)
    }

    // Test stop
    err = plugin.Stop(context.Background())
    if err != nil {
        t.Fatalf("Stop failed: %v", err)
    }
}
```

## Future Enhancements

- Plugin marketplace/registry
- Plugin sandboxing
- Inter-plugin communication
- Plugin versioning and updates
- Resource limits enforcement

## Next Steps

To get started developing plugins, see the [Plugin Development Guide](./development.md).
