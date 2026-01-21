package plugin

import (
	"context"
	"sync"
	"testing"
	"time"
)

// mockPlugin implements a basic plugin for testing
type mockPlugin struct {
	name     string
	version  string
	started  bool
	stopped  bool
	requires []string
	healthy  bool
	mu       sync.RWMutex
}

func (m *mockPlugin) GetInfo() Info {
	return Info{
		Name:        m.name,
		Version:     m.version,
		Description: "Mock plugin for testing",
		Author:      "Test",
		License:     "MIT",
		Requires:    m.requires,
		Provides:    []string{"mock"},
	}
}

func (m *mockPlugin) Init(ctx context.Context, config map[string]interface{}) error {
	return nil
}

func (m *mockPlugin) Start(ctx context.Context) error {
	m.started = true
	return nil
}

func (m *mockPlugin) Stop(ctx context.Context) error {
	m.stopped = true
	return nil
}

func (m *mockPlugin) Health() HealthStatus {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.healthy {
		return HealthStatus{
			Healthy: true,
			Status:  "healthy",
			Message: "Mock plugin is healthy",
		}
	}
	return HealthStatus{
		Healthy: false,
		Status:  "unhealthy",
		Message: "Plugin is unhealthy",
	}
}

func TestPluginManager(t *testing.T) {
	manager := NewManager()

	// Test plugin registration
	plugin := &mockPlugin{
		name:    "test-plugin",
		version: "1.0.0",
	}

	err := manager.RegisterPlugin(plugin)
	if err != nil {
		t.Fatalf("Failed to register plugin: %v", err)
	}

	// Test getting plugin
	retrieved, err := manager.GetPlugin("test-plugin")
	if err != nil {
		t.Fatalf("Failed to get plugin: %v", err)
	}

	if retrieved.GetInfo().Name != "test-plugin" {
		t.Errorf("Retrieved wrong plugin")
	}

	// Test enabling plugin
	err = manager.EnablePlugin("test-plugin", nil)
	if err != nil {
		t.Fatalf("Failed to enable plugin: %v", err)
	}

	if !plugin.started {
		t.Error("Plugin was not started")
	}

	// Test plugin state
	state, err := manager.GetPluginState("test-plugin")
	if err != nil {
		t.Fatalf("Failed to get plugin state: %v", err)
	}

	if !state.Enabled || !state.Running {
		t.Error("Plugin should be enabled and running")
	}

	// Test disabling plugin
	err = manager.DisablePlugin("test-plugin")
	if err != nil {
		t.Fatalf("Failed to disable plugin: %v", err)
	}

	if !plugin.stopped {
		t.Error("Plugin was not stopped")
	}

	// Test manager shutdown
	err = manager.Stop()
	if err != nil {
		t.Fatalf("Failed to stop manager: %v", err)
	}
}

func TestPluginRegistry(t *testing.T) {
	registry := NewRegistry()

	// Test registration
	plugin1 := &mockPlugin{name: "plugin1", version: "1.0.0"}
	plugin2 := &mockPlugin{name: "plugin2", version: "1.0.0"}

	err := registry.Register(plugin1)
	if err != nil {
		t.Fatalf("Failed to register plugin1: %v", err)
	}

	err = registry.Register(plugin2)
	if err != nil {
		t.Fatalf("Failed to register plugin2: %v", err)
	}

	// Test duplicate registration
	err = registry.Register(plugin1)
	if err == nil {
		t.Error("Should not allow duplicate registration")
	}

	// Test listing
	plugins := registry.List()
	if len(plugins) != 2 {
		t.Errorf("Expected 2 plugins, got %d", len(plugins))
	}

	// Test get by name
	retrieved, err := registry.Get("plugin1")
	if err != nil {
		t.Fatalf("Failed to get plugin1: %v", err)
	}

	if retrieved.GetInfo().Name != "plugin1" {
		t.Error("Retrieved wrong plugin")
	}

	// Test unregister
	err = registry.Unregister("plugin1")
	if err != nil {
		t.Fatalf("Failed to unregister plugin1: %v", err)
	}

	plugins = registry.List()
	if len(plugins) != 1 {
		t.Errorf("Expected 1 plugin after unregister, got %d", len(plugins))
	}
}

func TestPluginDependencies(t *testing.T) {
	manager := NewManager()

	// Create plugins with dependencies
	pluginA := &mockPlugin{name: "pluginA", version: "1.0.0"}
	pluginB := &mockPlugin{
		name:     "pluginB",
		version:  "1.0.0",
		requires: []string{"pluginA"},
	}

	// Register plugins
	_ = manager.RegisterPlugin(pluginA)
	_ = manager.RegisterPlugin(pluginB)

	// Try to enable pluginB without pluginA - should fail
	err := manager.EnablePlugin("pluginB", nil)
	if err == nil {
		t.Error("Should not allow enabling plugin with unmet dependencies")
	}

	// Enable pluginA first
	err = manager.EnablePlugin("pluginA", nil)
	if err != nil {
		t.Fatalf("Failed to enable pluginA: %v", err)
	}

	// Now enable pluginB - should succeed
	err = manager.EnablePlugin("pluginB", nil)
	if err != nil {
		t.Fatalf("Failed to enable pluginB with dependencies met: %v", err)
	}

	// Try to disable pluginA while pluginB depends on it - should fail
	err = manager.DisablePlugin("pluginA")
	if err == nil {
		t.Error("Should not allow disabling plugin that others depend on")
	}

	// Disable pluginB first
	err = manager.DisablePlugin("pluginB")
	if err != nil {
		t.Fatalf("Failed to disable pluginB: %v", err)
	}

	// Now disable pluginA - should succeed
	err = manager.DisablePlugin("pluginA")
	if err != nil {
		t.Fatalf("Failed to disable pluginA: %v", err)
	}
}

func TestHealthChecks(t *testing.T) {
	manager := NewManager()
	manager.healthCheckTime = 100 * time.Millisecond // Fast health checks for testing

	plugin := &mockPlugin{
		name:    "health-test",
		version: "1.0.0",
		healthy: true,
	}

	_ = manager.RegisterPlugin(plugin)
	_ = manager.EnablePlugin("health-test", nil)

	// Start health checks
	manager.StartHealthChecks()

	// Wait for initial health check
	time.Sleep(150 * time.Millisecond)

	// Make plugin unhealthy (with proper locking)
	plugin.mu.Lock()
	plugin.healthy = false
	plugin.mu.Unlock()

	// Wait for health check to detect unhealthy state
	time.Sleep(200 * time.Millisecond)

	state, _ := manager.GetPluginState("health-test")
	if state.Health.Healthy {
		t.Error("Health check should have detected unhealthy state")
	}

	_ = manager.Stop()
}