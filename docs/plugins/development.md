# Plugin Development Guide

This guide provides comprehensive instructions for developing plugins for the s9s SLURM management interface. Plugins allow you to extend s9s with custom views, commands, and integrations.

## Table of Contents

- [Overview](#overview)
- [Plugin Architecture](#plugin-architecture)
- [Project Structure](#project-structure)
- [Basic Plugin Implementation](#basic-plugin-implementation)
- [Creating Custom Views](#creating-custom-views)
- [Creating Overlay Plugins](#creating-overlay-plugins)
- [Event Handling with Hooks](#event-handling-with-hooks)
- [Building Plugins](#building-plugins)
- [Installation](#installation)
- [Testing](#testing)
- [Advanced Features](#advanced-features)
- [Best Practices](#best-practices)
- [Example Plugins](#example-plugins)
- [Troubleshooting](#troubleshooting)

## Overview

The s9s plugin system allows you to:
- Add custom views to the TUI (via `ViewPlugin`)
- Overlay additional data on existing views (via `OverlayPlugin`)
- Provide data to other plugins (via `DataPlugin`)
- React to lifecycle and configuration events (via `LifecycleAware`)
- Integrate with external systems through hooks (via `HookablePlugin`)

## Plugin Architecture

Plugins are implemented as Go shared libraries (`.so` files) that implement the `Plugin` interface. They are loaded dynamically at runtime.

### Plugin Interface

Every plugin must implement the base `Plugin` interface defined in `internal/plugin/interface.go`:

```go
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
```

Plugins can also implement additional interfaces for extended functionality:

- **`ViewPlugin`** -- provides custom TUI views (`GetViews()`, `CreateView()`)
- **`OverlayPlugin`** -- adds data overlays to existing views (`GetOverlays()`, `CreateOverlay()`)
- **`DataPlugin`** -- provides data to other plugins via pub/sub (`Subscribe()`, `Query()`)
- **`ConfigurablePlugin`** -- supports runtime configuration changes (`GetConfig()`, `SetConfig()`, `ValidateConfig()`)
- **`HookablePlugin`** -- provides hooks for event-driven integration (`GetHooks()`, `RegisterHook()`)
- **`LifecycleAware`** -- receives lifecycle events (`OnEnable()`, `OnDisable()`, `OnConfigChange()`)
- **`Prioritizable`** -- controls initialization order (`GetPriority()`)

See `internal/plugin/interface.go` for the full interface definitions.

## Project Structure

```
my-plugin/
├── main.go          # Plugin implementation
├── Makefile         # Build configuration
├── go.mod           # Go module
└── README.md        # Plugin documentation
```

## Basic Plugin Implementation

### Step 1: Create the main module

```go
// +build plugin

package main

import (
    "context"
    "github.com/jontk/s9s/internal/plugin"
)

type MyPlugin struct {
    config map[string]interface{}
}

// Entry point - must be exported
func NewPlugin() plugin.Plugin {
    return &MyPlugin{}
}

func (p *MyPlugin) GetInfo() plugin.Info {
    return plugin.Info{
        Name:        "my-plugin",
        Version:     "1.0.0",
        Description: "My custom s9s plugin",
        Author:      "Your Name",
        License:     "MIT",
        Provides:    []string{"my-capability"},
        ConfigSchema: map[string]plugin.ConfigField{
            "api_key": {
                Type:        "string",
                Description: "API key for external service",
                Required:    true,
            },
        },
    }
}

func (p *MyPlugin) Init(ctx context.Context, config map[string]interface{}) error {
    p.config = config
    return nil
}

func (p *MyPlugin) Start(ctx context.Context) error {
    // Start background processes
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
        Message: "Plugin is running normally",
    }
}
```

## Creating Custom Views

Implement the `ViewPlugin` interface to add custom views to the TUI. Your plugin must implement both `Plugin` and `ViewPlugin`.

### ViewPlugin Interface

```go
type ViewPlugin interface {
    Plugin

    // GetViews returns the views provided by this plugin
    GetViews() []ViewInfo

    // CreateView creates a specific view instance
    CreateView(ctx context.Context, viewID string) (View, error)
}
```

### View Structure

Each view must implement the `View` interface:

```go
type MyView struct {
    content *tview.TextView
}

func (v *MyView) GetName() string {
    return "My Custom View"
}

func (v *MyView) GetID() string {
    return "my-view"
}

func (v *MyView) GetPrimitive() tview.Primitive {
    if v.content == nil {
        v.content = tview.NewTextView()
        v.content.SetBorder(true).SetTitle("My Plugin View")
        v.content.SetText("This is my custom view content")
    }
    return v.content
}

func (v *MyView) Update(ctx context.Context) error {
    // Refresh the view data
    return nil
}

func (v *MyView) HandleKey(event *tcell.EventKey) bool {
    switch event.Rune() {
    case 'r':
        v.Update(context.Background())
        return true
    }
    return false
}

func (v *MyView) SetFocus(app *tview.Application) {
    app.SetFocus(v.content)
}

func (v *MyView) GetHelp() string {
    return "r=Refresh"
}
```

### Registering Views

```go
func (p *MyPlugin) GetViews() []plugin.ViewInfo {
    return []plugin.ViewInfo{
        {
            ID:          "my-view",
            Name:        "My Custom View",
            Description: "Displays custom information",
            Icon:        "M",
            Shortcut:    "Ctrl+M",
            Category:    "monitoring",
        },
    }
}

func (p *MyPlugin) CreateView(ctx context.Context, viewID string) (plugin.View, error) {
    if viewID == "my-view" {
        return &MyView{}, nil
    }
    return nil, fmt.Errorf("unknown view: %s", viewID)
}
```

## Creating Overlay Plugins

Overlays add columns or modify data in existing views without replacing them. Implement the `OverlayPlugin` interface:

```go
type OverlayPlugin interface {
    Plugin

    GetOverlays() []OverlayInfo
    CreateOverlay(ctx context.Context, overlayID string) (Overlay, error)
}
```

Each overlay can define additional columns and provide cell data and styling for rows in target views.

## Event Handling with Hooks

Use the `HookablePlugin` interface to provide hooks that other plugins or the application can subscribe to:

```go
func (p *MyPlugin) GetHooks() []plugin.HookInfo {
    return []plugin.HookInfo{
        {
            ID:          "on-metric-update",
            Name:        "Metric Update",
            Description: "Triggered when new metrics are collected",
        },
    }
}

func (p *MyPlugin) RegisterHook(hookID string, callback plugin.HookCallback) error {
    // Store and call the callback when the hook fires
    return nil
}
```

## Building Plugins

### Makefile Example

```makefile
PLUGIN_NAME = my-plugin
PLUGIN_SO = $(PLUGIN_NAME).so

.PHONY: build clean install

build:
	go build -buildmode=plugin -tags=plugin -o $(PLUGIN_SO) .

clean:
	rm -f $(PLUGIN_SO)

install: build
	mkdir -p ~/.s9s/plugins
	cp $(PLUGIN_SO) ~/.s9s/plugins/

test:
	go test -tags=plugin ./...
```

### Build Command

```bash
make build
# Or manually:
go build -buildmode=plugin -tags=plugin -o my-plugin.so .
```

## Installation

### Manual Installation

```bash
# Copy plugin to plugins directory
mkdir -p ~/.s9s/plugins
cp my-plugin.so ~/.s9s/plugins/
```

### Configuration

Add to your s9s config file:

```yaml
plugins:
  enabled: true
  directory: ~/.s9s/plugins
  autoload:
    - my-plugin
```

### Loading at Runtime

Plugins are loaded automatically from the configured directory when s9s starts with `plugins.enabled: true`. There are no CLI flags for loading individual plugins at this time.

> **Note:** See [#119](https://github.com/jontk/s9s/issues/119) for planned plugin CLI commands.

## Testing

### Unit Testing

```go
// +build plugin

func TestMyPlugin(t *testing.T) {
    p := &MyPlugin{}

    // Test plugin info
    info := p.GetInfo()
    assert.Equal(t, "my-plugin", info.Name)
    assert.NotEmpty(t, info.Version)
    assert.NotEmpty(t, info.Description)

    // Test initialization
    ctx := context.Background()
    err := p.Init(ctx, map[string]interface{}{"api_key": "test"})
    assert.NoError(t, err)

    // Test start/stop lifecycle
    err = p.Start(ctx)
    assert.NoError(t, err)

    health := p.Health()
    assert.True(t, health.Healthy)

    err = p.Stop(ctx)
    assert.NoError(t, err)
}
```

### Integration Testing

```bash
# Build and install the plugin, then run s9s in mock mode
make install
s9s --mock
```

## Advanced Features

### SLURM Client Integration

If your plugin needs access to SLURM data, accept a `dao.SlurmClient` via configuration or dependency injection during `Init()`:

```go
func (p *MyPlugin) Init(ctx context.Context, config map[string]interface{}) error {
    // Configuration is passed as a map; extract what you need
    p.config = config
    return nil
}
```

The plugin manager handles dependency resolution. Declare dependencies in your `Info.Requires` field to ensure required plugins are started first.

### SSH Integration

Execute commands on remote nodes:

```go
func (p *MyPlugin) connectToNode(nodeID string) error {
    // Access SSH functionality if available
    if p.sshClient != nil {
        session, err := p.sshClient.NewSession(nodeID)
        if err != nil {
            return err
        }
        defer session.Close()

        // Execute commands on remote node
        output, err := session.CombinedOutput("hostname")
        return err
    }
    return nil
}
```

### Plugin Configuration

Load and manage plugin configuration:

```go
type PluginConfig struct {
    APIKey    string `yaml:"api_key"`
    Endpoint  string `yaml:"endpoint"`
    Timeout   int    `yaml:"timeout"`
}

func (p *MyPlugin) loadConfig() (*PluginConfig, error) {
    configPath := "~/.s9s/plugins/my-plugin.yaml"
    // Load and parse configuration
    return config, nil
}
```

## Best Practices

### 1. Error Handling

Always handle errors gracefully:
- Return meaningful error messages
- Don't panic in plugin code
- Handle resource cleanup on errors

### 2. Resource Management

- Clean up resources in the `Stop()` method
- Cancel goroutines when plugin is stopped
- Close file handles and network connections

### 3. Performance

- Cache expensive operations
- Use goroutines for background work
- Avoid blocking the main UI thread
- Monitor memory usage in long-running plugins

### 4. User Experience

- Provide clear command descriptions
- Use consistent key bindings
- Follow s9s UI conventions
- Document plugin behavior and usage

### 5. Testing

- Write comprehensive unit tests
- Test error conditions
- Validate with different SLURM configurations
- Include integration tests

## Example Plugins

### Monitoring Plugin

This plugin collects custom metrics periodically and exposes a view:

```go
// Monitoring plugin that displays custom metrics
type MonitoringPlugin struct {
    ticker  *time.Ticker
    metrics map[string]interface{}
    cancel  context.CancelFunc
}

func (p *MonitoringPlugin) GetInfo() plugin.Info {
    return plugin.Info{
        Name:        "monitoring",
        Version:     "1.0.0",
        Description: "Collects and displays custom cluster metrics",
        Author:      "Your Name",
        Provides:    []string{"metrics"},
    }
}

func (p *MonitoringPlugin) Init(ctx context.Context, config map[string]interface{}) error {
    p.metrics = make(map[string]interface{})
    return nil
}

func (p *MonitoringPlugin) Start(ctx context.Context) error {
    ctx, p.cancel = context.WithCancel(ctx)
    p.ticker = time.NewTicker(30 * time.Second)
    go p.collectMetrics(ctx)
    return nil
}

func (p *MonitoringPlugin) Stop(ctx context.Context) error {
    p.cancel()
    if p.ticker != nil {
        p.ticker.Stop()
    }
    return nil
}

func (p *MonitoringPlugin) Health() plugin.HealthStatus {
    return plugin.HealthStatus{Healthy: true, Status: "healthy", Message: "OK"}
}

func (p *MonitoringPlugin) collectMetrics(ctx context.Context) {
    for {
        select {
        case <-ctx.Done():
            return
        case <-p.ticker.C:
            p.updateMetrics()
        }
    }
}
```

## Troubleshooting

### Common Issues

**Plugin not loading**
- Check build tags and shared library format
- Verify plugin filename matches configuration
- Check plugin directory permissions

**Missing symbols**
- Ensure `NewPlugin()` function is exported
- Verify correct build flags are used
- Check function signature matches interface

**Runtime panics**
- Add proper error handling and validation
- Test with mock data before integration
- Check goroutine cleanup in Stop()

**Memory leaks**
- Implement proper cleanup in `Stop()` method
- Cancel all goroutines on stop
- Close all file handles and connections

### Debugging

Enable debug logging to troubleshoot plugin loading issues:

```bash
# Enable debug logging
s9s --debug
```

The plugin manager logs registration, initialization, start, stop, and health check events. Look for lines containing the plugin name in the debug output.

> **Note:** See [#119](https://github.com/jontk/s9s/issues/119) for planned plugin CLI commands such as listing and validating plugins.

## API Reference

For complete API documentation, see:
- Plugin Interface: `/internal/plugin/interface.go`
- Plugin Manager: `/internal/plugin/manager.go`
- Plugin Registry: `/internal/plugin/registry.go`
- DAO Interface: `/internal/dao/interface.go`

## Contributing

To contribute plugin examples or improvements to the plugin system:

1. Fork the repository
2. Create a feature branch
3. Add your plugin example or improvement
4. Write tests and documentation
5. Submit a pull request

See `CONTRIBUTING.md` for detailed guidelines.
