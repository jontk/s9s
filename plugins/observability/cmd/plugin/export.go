package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"gopkg.in/yaml.v3"

	"github.com/jontk/s9s/internal/dao"
	"github.com/jontk/s9s/internal/plugin"
	"github.com/jontk/s9s/internal/plugins"
	observability "github.com/jontk/s9s/plugins/observability"
	"github.com/jontk/s9s/plugins/observability/logging"
)

// PluginAdapter adapts the observability plugin to the s9s plugin interface
type PluginAdapter struct {
	plugin *observability.ObservabilityPlugin
}

// NewPlugin is the exported function that s9s looks for when loading plugins
func NewPlugin() plugins.Plugin {
	return &PluginAdapter{
		plugin: observability.New(),
	}
}

// GetInfo returns plugin information
func (a *PluginAdapter) GetInfo() plugins.PluginInfo {
	info := a.plugin.GetInfo()
	return plugins.PluginInfo{
		Name:        info.Name,
		Version:     info.Version,
		Description: info.Description,
		Author:      info.Author,
		Website:     info.License, // Using License field for website since it's available
	}
}

// Initialize initializes the plugin
func (a *PluginAdapter) Initialize(ctx context.Context, client dao.SlurmClient) error {
	// Convert the config from context if available
	configMap := make(map[string]interface{})

	// The s9s plugin manager might pass config through context
	if config, ok := ctx.Value("plugin_config").(map[string]interface{}); ok {
		configMap = config
	}

	// If no config from context, try to read from s9s config file
	if len(configMap) == 0 {
		// Try to load config from the standard s9s config location
		if yamlConfig, err := loadS9sConfig(); err == nil {
			configMap = yamlConfig
			logging.Debug("adapter", "Loaded config from s9s config.yaml")
		} else {
			logging.Debug("adapter", "Could not load s9s config: %v, using defaults", err)
		}
	}

	// Log debug info about config
	logging.Debug("adapter", "Plugin config map: %+v", configMap)

	// Initialize the observability plugin
	logging.Debug("adapter", "Initializing observability plugin")
	if err := a.plugin.Init(ctx, configMap); err != nil {
		logging.Error("adapter", "Plugin Init failed: %v", err)
		return err
	}
	logging.Debug("adapter", "Plugin initialized successfully")

	// Pass the SLURM client to the plugin
	if client != nil {
		logging.Debug("adapter", "Setting SLURM client in plugin")
		a.plugin.SetSlurmClient(client)
	}

	// Start the plugin
	logging.Debug("adapter", "Starting observability plugin")
	if err := a.plugin.Start(ctx); err != nil {
		logging.Error("adapter", "Plugin Start failed: %v", err)
		return err
	}
	logging.Debug("adapter", "Plugin started successfully")

	return nil
}

// GetCommands returns the commands this plugin provides
func (a *PluginAdapter) GetCommands() []plugins.Command {
	// The observability plugin doesn't provide commands currently
	return []plugins.Command{}
}

// GetViews returns the views this plugin provides
func (a *PluginAdapter) GetViews() []plugins.View {
	views := a.plugin.GetViews()
	result := make([]plugins.View, 0, len(views))

	for _, v := range views {
		// Create view adapter
		viewAdapter := &ViewAdapter{
			plugin: a.plugin,
			info:   v,
		}
		result = append(result, viewAdapter)
	}

	return result
}

// GetKeyBindings returns custom key bindings
func (a *PluginAdapter) GetKeyBindings() []plugins.KeyBinding {
	// The observability plugin doesn't provide custom key bindings currently
	return []plugins.KeyBinding{}
}

// OnEvent handles events
func (a *PluginAdapter) OnEvent(event plugins.Event) error {
	// The observability plugin doesn't handle events currently
	return nil
}

// Cleanup cleans up the plugin
func (a *PluginAdapter) Cleanup() error {
	ctx := context.Background()
	return a.plugin.Stop(ctx)
}

// ViewAdapter adapts the observability view to the s9s view interface
type ViewAdapter struct {
	plugin *observability.ObservabilityPlugin
	info   plugin.ViewInfo
	view   plugin.View
}

// GetName returns the view name
func (v *ViewAdapter) GetName() string {
	return v.info.ID
}

// GetTitle returns the view title
func (v *ViewAdapter) GetTitle() string {
	return v.info.Name
}

// Render returns the tview primitive
func (v *ViewAdapter) Render() tview.Primitive {
	if v.view != nil {
		return v.view.GetPrimitive()
	}
	return nil
}

// OnKey handles key events
func (v *ViewAdapter) OnKey(event *tcell.EventKey) *tcell.EventKey {
	if v.view != nil {
		handled := v.view.HandleKey(event)
		if handled {
			return nil // Event was handled
		}
	}
	return event
}

// Refresh updates the view data
func (v *ViewAdapter) Refresh() error {
	logging.Debug("view-adapter", "Refreshing view %s", v.info.ID)
	if v.view != nil {
		ctx := context.Background()
		err := v.view.Update(ctx)
		if err != nil {
			logging.Error("view-adapter", "Failed to refresh view %s: %v", v.info.ID, err)
			return err
		}
		logging.Debug("view-adapter", "View %s refreshed successfully", v.info.ID)
	} else {
		logging.Warn("view-adapter", "Attempted to refresh view %s but view is nil", v.info.ID)
	}
	return nil
}

// Init initializes the view
func (v *ViewAdapter) Init(ctx context.Context) error {
	logging.Debug("view-adapter", "Initializing view %s", v.info.ID)
	// Create the actual view
	view, err := v.plugin.CreateView(ctx, v.info.ID)
	if err != nil {
		logging.Error("view-adapter", "Failed to create view %s: %v", v.info.ID, err)
		return err
	}
	v.view = view
	logging.Debug("view-adapter", "View %s initialized successfully", v.info.ID)
	return nil
}

// loadS9sConfig loads the observability plugin configuration from s9s config.yaml
func loadS9sConfig() (map[string]interface{}, error) {
	// Try to find the s9s config file
	var configPath string

	// Check current directory first
	if _, err := os.Stat("config.yaml"); err == nil {
		configPath = "config.yaml"
	} else {
		// Check home directory
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("could not get home directory: %w", err)
		}
		configPath = filepath.Join(home, ".s9s", "config.yaml")
	}

	// Read the config file
	// nolint:gosec // G304: configPath constructed from app config, not user input
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("could not read config file %s: %w", configPath, err)
	}

	// Parse the YAML
	var config map[string]interface{}
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("could not parse YAML: %w", err)
	}

	// Extract the observability plugin config
	plugins, ok := config["plugins"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("no plugins section in config")
	}

	// Find the observability plugin
	for _, p := range plugins {
		plugin, ok := p.(map[string]interface{})
		if !ok {
			continue
		}

		name, ok := plugin["name"].(string)
		if !ok || name != "observability" {
			continue
		}

		// Return the plugin's config section
		pluginConfig, ok := plugin["config"].(map[string]interface{})
		if !ok {
			return make(map[string]interface{}), nil
		}

		return pluginConfig, nil
	}

	return nil, fmt.Errorf("observability plugin not found in config")
}
