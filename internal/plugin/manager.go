package plugin

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/jontk/s9s/internal/debug"
)

// Manager manages the lifecycle of all plugins
type Manager struct {
	mu              sync.RWMutex
	registry        *Registry
	plugins         map[string]Plugin
	states          map[string]PluginState
	config          map[string]map[string]interface{}
	ctx             context.Context
	cancel          context.CancelFunc
	healthCheckTime time.Duration
}

// State represents the current state of a plugin
type State struct {
	Enabled      bool
	Running      bool
	Health       HealthStatus
	LastError    error
	StartTime    time.Time
	RestartCount int
}

// PluginState is an alias for backward compatibility
type PluginState = State

// NewManager creates a new plugin manager
func NewManager() *Manager {
	ctx, cancel := context.WithCancel(context.Background())

	return &Manager{
		registry:        NewRegistry(),
		plugins:         make(map[string]Plugin),
		states:          make(map[string]State),
		config:          make(map[string]map[string]interface{}),
		ctx:             ctx,
		cancel:          cancel,
		healthCheckTime: 30 * time.Second,
	}
}

// RegisterPlugin registers a plugin with the manager
func (m *Manager) RegisterPlugin(plugin Plugin) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	info := plugin.GetInfo()

	// Check if plugin already registered
	if _, exists := m.plugins[info.Name]; exists {
		return fmt.Errorf("plugin %s already registered", info.Name)
	}

	// Register in registry
	if err := m.registry.Register(plugin); err != nil {
		return fmt.Errorf("failed to register plugin %s: %w", info.Name, err)
	}

	// Store plugin
	m.plugins[info.Name] = plugin
	m.states[info.Name] = PluginState{
		Enabled: false,
		Running: false,
		Health: HealthStatus{
			Healthy: true,
			Status:  "initialized",
			Message: "Plugin registered",
		},
	}

	debug.Logger.Printf("Registered plugin: %s v%s", info.Name, info.Version)
	return nil
}

// EnablePlugin enables and starts a plugin
func (m *Manager) EnablePlugin(name string, config map[string]interface{}) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	plugin, exists := m.plugins[name]
	if !exists {
		return fmt.Errorf("plugin %s not found", name)
	}

	state := m.states[name]
	if state.Enabled {
		return fmt.Errorf("plugin %s already enabled", name)
	}

	// Check dependencies
	info := plugin.GetInfo()
	for _, req := range info.Requires {
		if depState, exists := m.states[req]; !exists || !depState.Running {
			return fmt.Errorf("required plugin %s is not running", req)
		}
	}

	// Store configuration
	m.config[name] = config

	// Initialize plugin
	if err := plugin.Init(m.ctx, config); err != nil {
		state.LastError = err
		m.states[name] = state
		return fmt.Errorf("failed to initialize plugin %s: %w", name, err)
	}

	// Start plugin
	if err := plugin.Start(m.ctx); err != nil {
		state.LastError = err
		m.states[name] = state
		return fmt.Errorf("failed to start plugin %s: %w", name, err)
	}

	// Update state
	state.Enabled = true
	state.Running = true
	state.StartTime = time.Now()
	state.Health = plugin.Health()
	m.states[name] = state

	// Call lifecycle hook if implemented
	if lifecycle, ok := plugin.(LifecycleAware); ok {
		if err := lifecycle.OnEnable(m.ctx); err != nil {
			debug.Logger.Printf("Plugin %s OnEnable hook failed: %v", name, err)
		}
	}

	debug.Logger.Printf("Enabled plugin: %s", name)
	return nil
}

// DisablePlugin disables and stops a plugin
func (m *Manager) DisablePlugin(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	plugin, exists := m.plugins[name]
	if !exists {
		return fmt.Errorf("plugin %s not found", name)
	}

	state := m.states[name]
	if !state.Enabled {
		return fmt.Errorf("plugin %s not enabled", name)
	}

	// Check if other plugins depend on this one
	for otherName, otherPlugin := range m.plugins {
		if otherName == name {
			continue
		}
		otherState := m.states[otherName]
		if !otherState.Enabled {
			continue
		}

		otherInfo := otherPlugin.GetInfo()
		for _, req := range otherInfo.Requires {
			if req == name {
				return fmt.Errorf("plugin %s is required by %s", name, otherName)
			}
		}
	}

	// Call lifecycle hook if implemented
	if lifecycle, ok := plugin.(LifecycleAware); ok {
		if err := lifecycle.OnDisable(m.ctx); err != nil {
			debug.Logger.Printf("Plugin %s OnDisable hook failed: %v", name, err)
		}
	}

	// Stop plugin
	if err := plugin.Stop(m.ctx); err != nil {
		debug.Logger.Printf("Error stopping plugin %s: %v", name, err)
		// Continue with disable even if stop fails
	}

	// Update state
	state.Enabled = false
	state.Running = false
	m.states[name] = state

	debug.Logger.Printf("Disabled plugin: %s", name)
	return nil
}

// GetPlugin returns a plugin by name
func (m *Manager) GetPlugin(name string) (Plugin, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	plugin, exists := m.plugins[name]
	if !exists {
		return nil, fmt.Errorf("plugin %s not found", name)
	}

	return plugin, nil
}

// GetPluginState returns the state of a plugin
func (m *Manager) GetPluginState(name string) (PluginState, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	state, exists := m.states[name]
	if !exists {
		return PluginState{}, fmt.Errorf("plugin %s not found", name)
	}

	return state, nil
}

// ListPlugins returns all registered plugins
func (m *Manager) ListPlugins() []PluginInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()

	plugins := make([]PluginInfo, 0, len(m.plugins))
	for name, plugin := range m.plugins {
		info := plugin.GetInfo()
		state := m.states[name]

		plugins = append(plugins, PluginInfo{
			Info:    info,
			State:   state,
			Enabled: state.Enabled,
			Running: state.Running,
		})
	}

	return plugins
}

// GetViewPlugins returns all plugins that provide views
func (m *Manager) GetViewPlugins() []ViewPlugin {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var viewPlugins []ViewPlugin
	for name, plugin := range m.plugins {
		state := m.states[name]
		if !state.Running {
			continue
		}

		if viewPlugin, ok := plugin.(ViewPlugin); ok {
			viewPlugins = append(viewPlugins, viewPlugin)
		}
	}

	return viewPlugins
}

// GetOverlayPlugins returns all plugins that provide overlays
func (m *Manager) GetOverlayPlugins() []OverlayPlugin {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var overlayPlugins []OverlayPlugin
	for name, plugin := range m.plugins {
		state := m.states[name]
		if !state.Running {
			continue
		}

		if overlayPlugin, ok := plugin.(OverlayPlugin); ok {
			overlayPlugins = append(overlayPlugins, overlayPlugin)
		}
	}

	return overlayPlugins
}

// UpdatePluginConfig updates a plugin's configuration
func (m *Manager) UpdatePluginConfig(name string, config map[string]interface{}) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	plugin, exists := m.plugins[name]
	if !exists {
		return fmt.Errorf("plugin %s not found", name)
	}

	// Validate configuration if plugin supports it
	if configurable, ok := plugin.(ConfigurablePlugin); ok {
		if err := configurable.ValidateConfig(config); err != nil {
			return fmt.Errorf("invalid configuration: %w", err)
		}

		oldConfig := m.config[name]

		// Apply configuration
		if err := configurable.SetConfig(config); err != nil {
			return fmt.Errorf("failed to apply configuration: %w", err)
		}

		// Call lifecycle hook if implemented
		if lifecycle, ok := plugin.(LifecycleAware); ok {
			if err := lifecycle.OnConfigChange(m.ctx, oldConfig, config); err != nil {
				// Rollback on error
				_ = configurable.SetConfig(oldConfig)
				return fmt.Errorf("configuration change failed: %w", err)
			}
		}
	}

	// Store new configuration
	m.config[name] = config

	debug.Logger.Printf("Updated configuration for plugin: %s", name)
	return nil
}

// StartHealthChecks starts periodic health checks for all plugins
func (m *Manager) StartHealthChecks() {
	go func() {
		ticker := time.NewTicker(m.healthCheckTime)
		defer ticker.Stop()

		for {
			select {
			case <-m.ctx.Done():
				return
			case <-ticker.C:
				m.checkHealth()
			}
		}
	}()
}

// checkHealth performs health checks on all running plugins
func (m *Manager) checkHealth() {
	m.mu.Lock()
	defer m.mu.Unlock()

	for name, plugin := range m.plugins {
		state := m.states[name]
		if !state.Running {
			continue
		}

		health := plugin.Health()
		state.Health = health

		// Handle unhealthy plugins
		if !health.Healthy && state.Running {
			debug.Logger.Printf("Plugin %s unhealthy: %s", name, health.Message)

			// Attempt restart if configured
			if state.RestartCount < 3 {
				debug.Logger.Printf("Attempting to restart plugin %s (attempt %d/3)",
					name, state.RestartCount+1)

				// Stop plugin
				if err := plugin.Stop(m.ctx); err != nil {
					debug.Logger.Printf("Error stopping unhealthy plugin %s: %v", name, err)
				}

				// Restart plugin
				if err := plugin.Start(m.ctx); err != nil {
					debug.Logger.Printf("Failed to restart plugin %s: %v", name, err)
					state.Running = false
					state.LastError = err
				} else {
					state.RestartCount++
					state.StartTime = time.Now()
				}
			} else {
				// Too many restarts, disable plugin
				debug.Logger.Printf("Plugin %s exceeded restart limit, disabling", name)
				state.Running = false
				state.Enabled = false
			}
		}

		m.states[name] = state
	}
}

// Stop stops all plugins and the manager
func (m *Manager) Stop() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Cancel context to signal shutdown
	m.cancel()

	// Stop all plugins in reverse dependency order
	stopped := make(map[string]bool)

	for {
		stoppedAny := false

		for name, plugin := range m.plugins {
			if stopped[name] {
				continue
			}

			state := m.states[name]
			if !state.Running {
				stopped[name] = true
				continue
			}

			// Check if any plugins depend on this one
			canStop := true
			for otherName, otherPlugin := range m.plugins {
				if stopped[otherName] || otherName == name {
					continue
				}

				otherInfo := otherPlugin.GetInfo()
				for _, req := range otherInfo.Requires {
					if req == name {
						canStop = false
						break
					}
				}
				if !canStop {
					break
				}
			}

			if canStop {
				debug.Logger.Printf("Stopping plugin: %s", name)
				if err := plugin.Stop(context.Background()); err != nil {
					debug.Logger.Printf("Error stopping plugin %s: %v", name, err)
				}
				stopped[name] = true
				stoppedAny = true
			}
		}

		if !stoppedAny {
			break
		}
	}

	debug.Logger.Printf("Plugin manager stopped")
	return nil
}

// PluginInfo combines plugin info with runtime state
type PluginInfo struct {
	Info    Info
	State   State
	Enabled bool
	Running bool
}
