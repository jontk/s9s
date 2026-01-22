package plugins

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"plugin"
	"sync"

	"github.com/jontk/s9s/internal/dao"
	"github.com/jontk/s9s/internal/debug"
)

// Manager implements the PluginManager interface
type Manager struct {
	plugins map[string]Plugin
	mu      sync.RWMutex
	ctx     context.Context
	client  dao.SlurmClient
}

// NewManager creates a new plugin manager
func NewManager(ctx context.Context, client dao.SlurmClient) *Manager {
	return &Manager{
		plugins: make(map[string]Plugin),
		ctx:     ctx,
		client:  client,
	}
}

// LoadPlugin loads a plugin from the given path
func (m *Manager) LoadPlugin(path string) error {
	debug.Logger.Printf("Loading plugin from: %s", path)

	// Load the plugin as a Go plugin
	p, err := plugin.Open(path)
	if err != nil {
		return fmt.Errorf("failed to load plugin %s: %w", path, err)
	}

	// Look for the NewPlugin function
	newPluginSym, err := p.Lookup("NewPlugin")
	if err != nil {
		return fmt.Errorf("plugin %s does not export NewPlugin function: %w", path, err)
	}

	// Call the NewPlugin function
	newPluginFunc, ok := newPluginSym.(func() Plugin)
	if !ok {
		return fmt.Errorf("plugin %s: NewPlugin has invalid signature", path)
	}

	pluginInstance := newPluginFunc()
	if pluginInstance == nil {
		return fmt.Errorf("plugin %s: NewPlugin returned nil", path)
	}

	// Initialize the plugin
	if err := pluginInstance.Initialize(m.ctx, m.client); err != nil {
		return fmt.Errorf("failed to initialize plugin %s: %w", path, err)
	}

	info := pluginInstance.GetInfo()
	debug.Logger.Printf("Loaded plugin: %s v%s", info.Name, info.Version)

	m.mu.Lock()
	m.plugins[info.Name] = pluginInstance
	m.mu.Unlock()

	return nil
}

// LoadPluginsFromDirectory loads all plugins from a directory
func (m *Manager) LoadPluginsFromDirectory(dir string) error {
	debug.Logger.Printf("Loading plugins from directory: %s", dir)

	if _, err := os.Stat(dir); os.IsNotExist(err) {
		debug.Logger.Printf("Plugin directory does not exist: %s", dir)
		return nil // Not an error, just no plugins to load
	}

	return filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip non-plugin files
		if info.IsDir() || filepath.Ext(path) != ".so" {
			return nil
		}

		if err := m.LoadPlugin(path); err != nil {
			debug.Logger.Printf("Failed to load plugin %s: %v", path, err)
			// Don't fail the entire loading process for one bad plugin
		}

		return nil
	})
}

// GetPlugin returns a plugin by name
func (m *Manager) GetPlugin(name string) Plugin {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.plugins[name]
}

// GetAllPlugins returns all loaded plugins
func (m *Manager) GetAllPlugins() []Plugin {
	m.mu.RLock()
	defer m.mu.RUnlock()

	plugins := make([]Plugin, 0, len(m.plugins))
	for _, p := range m.plugins {
		plugins = append(plugins, p)
	}
	return plugins
}

// GetCommands returns all commands from all plugins
func (m *Manager) GetCommands() []Command {
	var commands []Command

	for _, p := range m.GetAllPlugins() {
		commands = append(commands, p.GetCommands()...)
	}

	return commands
}

// GetViews returns all views from all plugins
func (m *Manager) GetViews() []View {
	var views []View

	for _, p := range m.GetAllPlugins() {
		views = append(views, p.GetViews()...)
	}

	return views
}

// GetKeyBindings returns all key bindings from all plugins
func (m *Manager) GetKeyBindings() []KeyBinding {
	var bindings []KeyBinding

	for _, p := range m.GetAllPlugins() {
		bindings = append(bindings, p.GetKeyBindings()...)
	}

	return bindings
}

// SendEvent sends an event to all plugins
func (m *Manager) SendEvent(event Event) error {
	for _, p := range m.GetAllPlugins() {
		if err := p.OnEvent(event); err != nil {
			debug.Logger.Printf("Plugin %s failed to handle event %v: %v", p.GetInfo().Name, event.Type, err)
		}
	}
	return nil
}

// UnloadPlugin unloads a plugin
func (m *Manager) UnloadPlugin(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	plugin, exists := m.plugins[name]
	if !exists {
		return fmt.Errorf("plugin %s not found", name)
	}

	if err := plugin.Cleanup(); err != nil {
		debug.Logger.Printf("Plugin %s cleanup failed: %v", name, err)
	}

	delete(m.plugins, name)
	debug.Logger.Printf("Unloaded plugin: %s", name)
	return nil
}

// UnloadAllPlugins unloads all plugins
func (m *Manager) UnloadAllPlugins() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for name, plugin := range m.plugins {
		if err := plugin.Cleanup(); err != nil {
			debug.Logger.Printf("Plugin %s cleanup failed: %v", name, err)
		}
	}

	m.plugins = make(map[string]Plugin)
	debug.Logger.Printf("Unloaded all plugins")
	return nil
}
