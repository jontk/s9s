package plugin

import (
	"fmt"
	"sort"
	"sync"
	"time"
)

// Registry manages plugin registration and metadata
type Registry struct {
	mu       sync.RWMutex
	plugins  map[string]Plugin
	metadata map[string]PluginMetadata
	provides map[string][]string // capability -> plugin names
	requires map[string][]string // plugin -> required plugins
}

// PluginMetadata contains additional metadata about a plugin
type PluginMetadata struct {
	RegistrationTime int64
	LoadOrder        int
	Source           string // "builtin", "external", "dynamic"
	Path             string // File path for external plugins
	Checksum         string // For verification
}

// NewRegistry creates a new plugin registry
func NewRegistry() *Registry {
	return &Registry{
		plugins:  make(map[string]Plugin),
		metadata: make(map[string]PluginMetadata),
		provides: make(map[string][]string),
		requires: make(map[string][]string),
	}
}

// Register registers a plugin in the registry
func (r *Registry) Register(plugin Plugin) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	info := plugin.GetInfo()

	// Validate plugin info
	if err := r.validatePluginInfo(info); err != nil {
		return fmt.Errorf("invalid plugin info: %w", err)
	}

	// Check if already registered
	if _, exists := r.plugins[info.Name]; exists {
		return fmt.Errorf("plugin %s already registered", info.Name)
	}

	// Register plugin
	r.plugins[info.Name] = plugin

	// Store metadata
	r.metadata[info.Name] = PluginMetadata{
		RegistrationTime: getCurrentTimestamp(),
		LoadOrder:        len(r.plugins),
		Source:           "builtin", // Default, can be overridden
	}

	// Index capabilities
	for _, capability := range info.Provides {
		r.provides[capability] = append(r.provides[capability], info.Name)
	}

	// Index requirements
	if len(info.Requires) > 0 {
		r.requires[info.Name] = info.Requires
	}

	return nil
}

// Unregister removes a plugin from the registry
func (r *Registry) Unregister(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	plugin, exists := r.plugins[name]
	if !exists {
		return fmt.Errorf("plugin %s not found", name)
	}

	info := plugin.GetInfo()

	// Remove from plugins
	delete(r.plugins, name)
	delete(r.metadata, name)
	delete(r.requires, name)

	// Remove from capability index
	for _, capability := range info.Provides {
		providers := r.provides[capability]
		for i, provider := range providers {
			if provider == name {
				r.provides[capability] = append(providers[:i], providers[i+1:]...)
				break
			}
		}
		if len(r.provides[capability]) == 0 {
			delete(r.provides, capability)
		}
	}

	return nil
}

// Get retrieves a plugin by name
func (r *Registry) Get(name string) (Plugin, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	plugin, exists := r.plugins[name]
	if !exists {
		return nil, fmt.Errorf("plugin %s not found", name)
	}

	return plugin, nil
}

// GetByCapability returns all plugins that provide a specific capability
func (r *Registry) GetByCapability(capability string) []Plugin {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var plugins []Plugin

	providers, exists := r.provides[capability]
	if !exists {
		return plugins
	}

	for _, name := range providers {
		if plugin, exists := r.plugins[name]; exists {
			plugins = append(plugins, plugin)
		}
	}

	return plugins
}

// List returns all registered plugins
func (r *Registry) List() []Plugin {
	r.mu.RLock()
	defer r.mu.RUnlock()

	plugins := make([]Plugin, 0, len(r.plugins))
	for _, plugin := range r.plugins {
		plugins = append(plugins, plugin)
	}

	// Sort by load order
	sort.Slice(plugins, func(i, j int) bool {
		infoI := plugins[i].GetInfo()
		infoJ := plugins[j].GetInfo()
		metaI := r.metadata[infoI.Name]
		metaJ := r.metadata[infoJ.Name]
		return metaI.LoadOrder < metaJ.LoadOrder
	})

	return plugins
}

// GetDependencyOrder returns plugins in dependency order
func (r *Registry) GetDependencyOrder() ([]string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Build dependency graph
	graph := make(map[string][]string)
	inDegree := make(map[string]int)

	// Initialize graph
	for name := range r.plugins {
		graph[name] = []string{}
		inDegree[name] = 0
	}

	// Build edges
	for name, deps := range r.requires {
		for _, dep := range deps {
			if _, exists := r.plugins[dep]; !exists {
				return nil, fmt.Errorf("plugin %s requires non-existent plugin %s", name, dep)
			}
			graph[dep] = append(graph[dep], name)
			inDegree[name]++
		}
	}

	// Topological sort using Kahn's algorithm
	var result []string
	queue := []string{}

	// Find all nodes with no incoming edges
	for name, degree := range inDegree {
		if degree == 0 {
			queue = append(queue, name)
		}
	}

	// Sort initial queue by priority if plugins implement Prioritizable
	sort.Slice(queue, func(i, j int) bool {
		return r.comparePriority(queue[i], queue[j])
	})

	// Process queue
	for len(queue) > 0 {
		// Dequeue
		current := queue[0]
		queue = queue[1:]
		result = append(result, current)

		// Process neighbors
		neighbors := graph[current]
		sort.Slice(neighbors, func(i, j int) bool {
			return r.comparePriority(neighbors[i], neighbors[j])
		})

		for _, neighbor := range neighbors {
			inDegree[neighbor]--
			if inDegree[neighbor] == 0 {
				queue = append(queue, neighbor)
			}
		}
	}

	// Check for cycles
	if len(result) != len(r.plugins) {
		return nil, fmt.Errorf("circular dependency detected in plugins")
	}

	return result, nil
}

// GetMetadata returns metadata for a plugin
func (r *Registry) GetMetadata(name string) (PluginMetadata, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	metadata, exists := r.metadata[name]
	if !exists {
		return PluginMetadata{}, fmt.Errorf("plugin %s not found", name)
	}

	return metadata, nil
}

// SetMetadata updates metadata for a plugin
func (r *Registry) SetMetadata(name string, metadata PluginMetadata) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.plugins[name]; !exists {
		return fmt.Errorf("plugin %s not found", name)
	}

	r.metadata[name] = metadata
	return nil
}

// ValidateDependencies checks if all plugin dependencies are satisfied
func (r *Registry) ValidateDependencies() error {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for name, deps := range r.requires {
		for _, dep := range deps {
			if _, exists := r.plugins[dep]; !exists {
				return fmt.Errorf("plugin %s requires non-existent plugin %s", name, dep)
			}
		}
	}

	// Check for circular dependencies
	_, err := r.GetDependencyOrder()
	return err
}

// GetCapabilities returns all available capabilities
func (r *Registry) GetCapabilities() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	capabilities := make([]string, 0, len(r.provides))
	for capability := range r.provides {
		capabilities = append(capabilities, capability)
	}

	sort.Strings(capabilities)
	return capabilities
}

// validatePluginInfo validates plugin information
func (r *Registry) validatePluginInfo(info Info) error {
	if info.Name == "" {
		return fmt.Errorf("plugin name is required")
	}

	if info.Version == "" {
		return fmt.Errorf("plugin version is required")
	}

	if info.Description == "" {
		return fmt.Errorf("plugin description is required")
	}

	// Validate configuration schema
	for fieldName, field := range info.ConfigSchema {
		if field.Type == "" {
			return fmt.Errorf("config field %s has no type", fieldName)
		}

		validTypes := []string{"string", "int", "bool", "float", "array", "object"}
		validType := false
		for _, t := range validTypes {
			if field.Type == t {
				validType = true
				break
			}
		}
		if !validType {
			return fmt.Errorf("config field %s has invalid type %s", fieldName, field.Type)
		}
	}

	return nil
}

// comparePriority compares plugin priorities
func (r *Registry) comparePriority(a, b string) bool {
	pluginA, _ := r.plugins[a]
	pluginB, _ := r.plugins[b]

	priorityA := 0
	priorityB := 0

	if p, ok := pluginA.(Prioritizable); ok {
		priorityA = p.GetPriority()
	}

	if p, ok := pluginB.(Prioritizable); ok {
		priorityB = p.GetPriority()
	}

	return priorityA < priorityB
}

// getCurrentTimestamp returns current Unix timestamp
func getCurrentTimestamp() int64 {
	return time.Now().Unix()
}

