package app

import (
	"context"
	"os"
	"path/filepath"

	"github.com/gdamore/tcell/v2"
	"github.com/jontk/s9s/internal/plugins"
	"github.com/rivo/tview"
)

// loadPlugins loads plugins from the configured plugin directories
// Returns error for extensibility, currently always returns nil
//
//nolint:unparam // error parameter kept for future extensibility
func (s *S9s) loadPlugins() error {
	// Load plugins from the standard plugins directory
	homeDir, err := os.UserHomeDir()
	if err == nil {
		s9sConfigDir := filepath.Join(homeDir, ".s9s")
		pluginDir := filepath.Join(s9sConfigDir, "plugins")
		_ = s.pluginManager.LoadPluginsFromDirectory(pluginDir)
	}

	// Load plugins from system directory
	systemPluginDir := "/usr/share/s9s/plugins"
	_ = s.pluginManager.LoadPluginsFromDirectory(systemPluginDir)

	// Load plugins from local directory (for development)
	localPluginDir := filepath.Join(".", "plugins")
	_ = s.pluginManager.LoadPluginsFromDirectory(localPluginDir)

	return nil
}

// registerPluginViews registers all views from loaded plugins
// Returns error for extensibility, currently always returns nil
//
//nolint:unparam // error parameter kept for future extensibility
func (s *S9s) registerPluginViews() error {
	// Get all plugin views
	pluginViews := s.pluginManager.GetViews()

	s.logger.Info().Int("count", len(pluginViews)).Msg("Registering plugin views")

	for _, pluginView := range pluginViews {
		s.logger.Info().Str("view", pluginView.GetName()).Msg("Registering plugin view")

		// Create context with tview application for plugin initialization
		ctx := context.WithValue(s.ctx, appContextKey, s.app)

		// Initialize the plugin view
		if err := pluginView.Init(ctx); err != nil {
			s.logger.Warn().Err(err).Str("view", pluginView.GetName()).Msg("Failed to initialize plugin view")
			continue
		}

		// Create adapter to bridge plugin view to s9s view interface
		viewAdapter := &PluginViewAdapter{
			pluginView: pluginView,
		}

		// Add to view manager
		if err := s.viewMgr.AddView(viewAdapter); err != nil {
			s.logger.Warn().Err(err).Str("view", pluginView.GetName()).Msg("Failed to add plugin view")
			continue
		}

		// Add to content pages
		s.contentPages.AddPage(pluginView.GetName(), pluginView.Render(), true, false)

		s.logger.Info().Str("view", pluginView.GetName()).Msg("Successfully registered plugin view")
	}

	// Update header with new view names (including plugin views)
	s.header.SetViews(s.viewMgr.GetViewNames())

	return nil
}

// PluginViewAdapter adapts a plugin view to the s9s view interface.
type PluginViewAdapter struct {
	pluginView plugins.View
}

// Name returns the name of the plugin view.
func (p *PluginViewAdapter) Name() string {
	return p.pluginView.GetName()
}

// Title returns the title of the plugin view.
func (p *PluginViewAdapter) Title() string {
	return p.pluginView.GetTitle()
}

// Hints returns keyboard hints for the plugin view.
func (p *PluginViewAdapter) Hints() []string {
	// Default hints for plugin views
	return []string{"Tab=Switch", "F5=Refresh", "?=Help", "q=Quit"}
}

// Init initializes the plugin view with the provided context.
func (p *PluginViewAdapter) Init(ctx context.Context) error {
	return p.pluginView.Init(ctx)
}

// Render returns the tview.Primitive representation of the plugin view.
func (p *PluginViewAdapter) Render() tview.Primitive {
	return p.pluginView.Render()
}

// Refresh refreshes the plugin view.
func (p *PluginViewAdapter) Refresh() error {
	return p.pluginView.Refresh()
}

// OnKey handles keyboard events for the plugin view.
func (p *PluginViewAdapter) OnKey(event *tcell.EventKey) *tcell.EventKey {
	return p.pluginView.OnKey(event)
}

// OnFocus handles focus events for the plugin view.
func (p *PluginViewAdapter) OnFocus() error {
	// Plugin views don't have OnFocus, so this is a no-op
	return nil
}

// OnLoseFocus handles loss of focus events for the plugin view.
func (p *PluginViewAdapter) OnLoseFocus() error {
	// Plugin views don't have OnLoseFocus, so this is a no-op
	return nil
}

// Stop stops the plugin view.
func (p *PluginViewAdapter) Stop() error {
	// Plugin views don't have Stop, so this is a no-op
	return nil
}

// SetSwitchViewFn sets the callback function to switch to another view
func (p *PluginViewAdapter) SetSwitchViewFn(fn func(string)) {
	// Plugin views don't support view switching, so this is a no-op
}

// SwitchToView switches to another view
func (p *PluginViewAdapter) SwitchToView(viewName string) {
	// Plugin views don't support view switching, so this is a no-op
}
