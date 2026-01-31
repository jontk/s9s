# Plugin Development Guide

This guide provides comprehensive instructions for developing plugins for the s9s SLURM management interface. Plugins allow you to extend s9s with custom views, commands, and integrations.

## Table of Contents

- [Overview](#overview)
- [Plugin Architecture](#plugin-architecture)
- [Project Structure](#project-structure)
- [Basic Plugin Implementation](#basic-plugin-implementation)
- [Adding Custom Commands](#adding-custom-commands)
- [Creating Custom Views](#creating-custom-views)
- [Custom Key Bindings](#custom-key-bindings)
- [Event Handling](#event-handling)
- [Building Plugins](#building-plugins)
- [Installation](#installation)
- [Testing](#testing)
- [Advanced Features](#advanced-features)
- [Best Practices](#best-practices)
- [Example Plugins](#example-plugins)
- [Troubleshooting](#troubleshooting)

## Overview

The s9s plugin system allows you to:
- Add custom views to the TUI
- Implement new commands
- Define custom key bindings
- React to application events
- Integrate with external systems

## Plugin Architecture

Plugins are implemented as Go shared libraries (`.so` files) that implement the `Plugin` interface. They are loaded dynamically at runtime.

### Plugin Interface

```go
type Plugin interface {
    GetInfo() PluginInfo
    Initialize(ctx context.Context, client dao.SlurmClient) error
    GetCommands() []Command
    GetViews() []View
    GetKeyBindings() []KeyBinding
    OnEvent(event Event) error
    Cleanup() error
}
```

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
    "github.com/jontk/s9s/internal/dao"
    "github.com/jontk/s9s/internal/plugins"
)

type MyPlugin struct {
    client dao.SlurmClient
}

// Entry point - must be exported
func NewPlugin() plugins.Plugin {
    return &MyPlugin{}
}

func (p *MyPlugin) GetInfo() plugins.PluginInfo {
    return plugins.PluginInfo{
        Name:        "my-plugin",
        Version:     "1.0.0",
        Description: "My custom s9s plugin",
        Author:      "Your Name",
        Website:     "https://example.com",
    }
}

func (p *MyPlugin) Initialize(ctx context.Context, client dao.SlurmClient) error {
    p.client = client
    return nil
}

func (p *MyPlugin) GetCommands() []plugins.Command {
    return []plugins.Command{}
}

func (p *MyPlugin) GetViews() []plugins.View {
    return []plugins.View{}
}

func (p *MyPlugin) GetKeyBindings() []plugins.KeyBinding {
    return []plugins.KeyBinding{}
}

func (p *MyPlugin) OnEvent(event plugins.Event) error {
    return nil
}

func (p *MyPlugin) Cleanup() error {
    return nil
}
```

## Adding Custom Commands

Extend the `GetCommands()` method to provide custom commands:

```go
func (p *MyPlugin) GetCommands() []plugins.Command {
    return []plugins.Command{
        {
            Name:        "status",
            Description: "Show custom status information",
            Usage:       "status [options]",
            Handler: func(args []string) error {
                // Your command logic here
                fmt.Println("Custom status command executed")
                return nil
            },
        },
    }
}
```

## Creating Custom Views

### View Structure

```go
type MyView struct {
    content *tview.TextView
}

func (v *MyView) GetName() string {
    return "myview"
}

func (v *MyView) GetTitle() string {
    return "My Custom View"
}

func (v *MyView) Render() tview.Primitive {
    if v.content == nil {
        v.content = tview.NewTextView()
        v.content.SetBorder(true).SetTitle("My Plugin View")
        v.content.SetText("This is my custom view content")
    }
    return v.content
}

func (v *MyView) OnKey(event *tcell.EventKey) *tcell.EventKey {
    switch event.Rune() {
    case 'q':
        return nil // Close view
    case 'r':
        v.Refresh()
    }
    return event
}

func (v *MyView) Refresh() error {
    // Update view content
    return nil
}

func (v *MyView) Init(ctx context.Context) error {
    return nil
}
```

### Registering Views

```go
func (p *MyPlugin) GetViews() []plugins.View {
    return []plugins.View{
        &MyView{},
    }
}
```

## Custom Key Bindings

Define custom key bindings for your plugin:

```go
func (p *MyPlugin) GetKeyBindings() []plugins.KeyBinding {
    return []plugins.KeyBinding{
        {
            Key:         'M',
            Modifiers:   tcell.ModCtrl,
            Description: "My custom action",
            Handler: func() error {
                // Your key binding logic
                return nil
            },
        },
    }
}
```

## Event Handling

Respond to application events:

```go
func (p *MyPlugin) OnEvent(event plugins.Event) error {
    switch event.Type {
    case plugins.EventJobSubmitted:
        // React to job submission
        jobData := event.Data.(JobData)
        fmt.Printf("Job %s submitted\n", jobData.ID)

    case plugins.EventViewChanged:
        // React to view changes
        viewName := event.Data.(string)
        fmt.Printf("Switched to view: %s\n", viewName)
    }
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

```bash
# Enable plugins
s9s --plugins

# Load specific plugin
s9s --plugin my-plugin
```

## Testing

### Unit Testing

```go
// +build plugin

func TestMyPlugin(t *testing.T) {
    plugin := &MyPlugin{}

    // Test plugin info
    info := plugin.GetInfo()
    assert.Equal(t, "my-plugin", info.Name)

    // Test initialization
    ctx := context.Background()
    err := plugin.Initialize(ctx, nil)
    assert.NoError(t, err)

    // Test commands
    commands := plugin.GetCommands()
    assert.Len(t, commands, 1)
}
```

### Integration Testing

```bash
# Build and test with s9s
make build
s9s --plugin ./my-plugin.so --mock
```

## Advanced Features

### SLURM Client Integration

Access SLURM cluster information through the client:

```go
func (p *MyPlugin) Initialize(ctx context.Context, client dao.SlurmClient) error {
    p.client = client

    // Access SLURM cluster information
    info, err := client.ClusterInfo()
    if err != nil {
        return err
    }

    // Get job information
    jobs, err := client.GetJobs()
    if err != nil {
        return err
    }

    return nil
}
```

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

- Clean up resources in the `Cleanup()` method
- Cancel goroutines when plugin is unloaded
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

This plugin displays custom metrics collected periodically:

```go
// Monitoring plugin that displays custom metrics
type MonitoringPlugin struct {
    client dao.SlurmClient
    ticker *time.Ticker
    metrics map[string]interface{}
}

func (p *MonitoringPlugin) Initialize(ctx context.Context, client dao.SlurmClient) error {
    p.client = client
    p.metrics = make(map[string]interface{})

    // Start background metrics collection
    p.ticker = time.NewTicker(30 * time.Second)
    go p.collectMetrics(ctx)

    return nil
}

func (p *MonitoringPlugin) collectMetrics(ctx context.Context) {
    for {
        select {
        case <-ctx.Done():
            return
        case <-p.ticker.C:
            // Collect custom metrics
            p.updateMetrics()
        }
    }
}
```

### Notification Plugin

This plugin sends notifications for job events:

```go
// Plugin that sends notifications for job events
type NotificationPlugin struct {
    webhookURL string
}

func (p *NotificationPlugin) OnEvent(event plugins.Event) error {
    switch event.Type {
    case plugins.EventJobCompleted:
        return p.sendNotification("Job completed", event.Data)
    case plugins.EventNodeStateChanged:
        return p.sendNotification("Node state changed", event.Data)
    }
    return nil
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
- Check goroutine cleanup in Cleanup()

**Memory leaks**
- Implement proper cleanup in `Cleanup()` method
- Cancel all goroutines on cleanup
- Close all file handles and connections

### Debugging

Enable debug logging and validation:

```bash
# Enable debug logging
s9s --debug --plugin ./my-plugin.so

# Check plugin loading
s9s --list-plugins

# Validate plugin
s9s --validate-plugin ./my-plugin.so
```

## API Reference

For complete API documentation, see:
- Plugin Interface: `/internal/plugins/interface.go`
- DAO Interface: `/internal/dao/interface.go`
- Example Plugins: `/internal/plugins/examples/`

## Contributing

To contribute plugin examples or improvements to the plugin system:

1. Fork the repository
2. Create a feature branch
3. Add your plugin example or improvement
4. Write tests and documentation
5. Submit a pull request

See `CONTRIBUTING.md` for detailed guidelines.
