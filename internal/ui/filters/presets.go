package filters

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"github.com/jontk/s9s/internal/fileperms"
)

// FilterPreset represents a saved filter configuration
type FilterPreset struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	ViewType    string `json:"view_type"` // "jobs", "nodes", "all", etc.
	FilterStr   string `json:"filter_str"`
	IsGlobal    bool   `json:"is_global"`
}

// PresetManager manages filter presets
type PresetManager struct {
	presetsDir string
	presets    map[string][]FilterPreset // Keyed by view type
}

// NewPresetManager creates a new preset manager
func NewPresetManager() *PresetManager {
	homeDir, _ := os.UserHomeDir()
	presetsDir := filepath.Join(homeDir, ".s9s", "filters")

	// Create presets directory if it doesn't exist
	_ = os.MkdirAll(presetsDir, fileperms.ConfigDir)

	manager := &PresetManager{
		presetsDir: presetsDir,
		presets:    make(map[string][]FilterPreset),
	}

	// Load existing presets
	_ = manager.loadPresets()

	// Add default presets if none exist
	if len(manager.presets) == 0 {
		manager.createDefaultPresets()
	}

	return manager
}

// loadPresets loads all saved presets from disk
func (m *PresetManager) loadPresets() error {
	presetsFile := filepath.Join(m.presetsDir, "presets.json")

	data, err := os.ReadFile(presetsFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // No presets file yet
		}
		return err
	}

	var allPresets []FilterPreset
	if err := json.Unmarshal(data, &allPresets); err != nil {
		return err
	}

	// Organize presets by view type
	for _, preset := range allPresets {
		m.presets[preset.ViewType] = append(m.presets[preset.ViewType], preset)
		if preset.IsGlobal {
			m.presets["all"] = append(m.presets["all"], preset)
		}
	}

	return nil
}

// savePresets saves all presets to disk
func (m *PresetManager) savePresets() error {
	presetsFile := filepath.Join(m.presetsDir, "presets.json")

	// Flatten all presets
	var allPresets []FilterPreset
	seen := make(map[string]bool)

	for _, presets := range m.presets {
		for _, preset := range presets {
			key := fmt.Sprintf("%s-%s", preset.ViewType, preset.Name)
			if !seen[key] {
				allPresets = append(allPresets, preset)
				seen[key] = true
			}
		}
	}

	data, err := json.MarshalIndent(allPresets, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(presetsFile, data, fileperms.ConfigFile)
}

// createDefaultPresets creates default filter presets
func (m *PresetManager) createDefaultPresets() {
	// Job presets
	jobPresets := []FilterPreset{
		{
			Name:        "My Jobs",
			Description: "Show only my jobs",
			ViewType:    "jobs",
			FilterStr:   "user=$USER",
		},
		{
			Name:        "Running Jobs",
			Description: "Show only running jobs",
			ViewType:    "jobs",
			FilterStr:   "state=RUNNING",
		},
		{
			Name:        "Pending Jobs",
			Description: "Show only pending jobs",
			ViewType:    "jobs",
			FilterStr:   "state=PENDING",
		},
		{
			Name:        "Failed Jobs",
			Description: "Show failed and timeout jobs",
			ViewType:    "jobs",
			FilterStr:   "state in (FAILED,TIMEOUT)",
		},
		{
			Name:        "GPU Jobs",
			Description: "Show jobs on GPU partition",
			ViewType:    "jobs",
			FilterStr:   "partition=gpu",
		},
		{
			Name:        "High Priority",
			Description: "Show high priority jobs",
			ViewType:    "jobs",
			FilterStr:   "priority>1000",
		},
		{
			Name:        "Large Memory Jobs",
			Description: "Jobs requiring more than 32GB RAM",
			ViewType:    "jobs",
			FilterStr:   "memory>32G",
		},
		{
			Name:        "Long Running Jobs",
			Description: "Jobs running longer than 4 hours",
			ViewType:    "jobs",
			FilterStr:   "time>4:00:00",
		},
		{
			Name:        "Today's Jobs",
			Description: "Jobs submitted today",
			ViewType:    "jobs",
			FilterStr:   "submittime=today",
		},
		{
			Name:        "Recent Failures",
			Description: "Jobs that failed in last 24 hours",
			ViewType:    "jobs",
			FilterStr:   "state=FAILED endtime=\"last 24h\"",
		},
	}

	// Node presets
	nodePresets := []FilterPreset{
		{
			Name:        "Available Nodes",
			Description: "Show idle and mixed nodes",
			ViewType:    "nodes",
			FilterStr:   "state in (IDLE,MIXED)",
		},
		{
			Name:        "Down Nodes",
			Description: "Show down and drain nodes",
			ViewType:    "nodes",
			FilterStr:   "state in (DOWN,DRAIN,DRAINING)",
		},
		{
			Name:        "GPU Nodes",
			Description: "Show nodes with GPU resources",
			ViewType:    "nodes",
			FilterStr:   "features~gpu",
		},
		{
			Name:        "High Memory",
			Description: "Show nodes with >256GB memory",
			ViewType:    "nodes",
			FilterStr:   "memory>256000",
		},
		{
			Name:        "Compute Partition",
			Description: "Show nodes in compute partition",
			ViewType:    "nodes",
			FilterStr:   "partition=compute",
		},
		{
			Name:        "Super High Memory",
			Description: "Nodes with more than 512GB RAM",
			ViewType:    "nodes",
			FilterStr:   "memory>512G",
		},
		{
			Name:        "Multi-GPU Nodes",
			Description: "Nodes with multiple GPU features",
			ViewType:    "nodes",
			FilterStr:   "features=~gpu.*gpu|features~v100|features~a100",
		},
		{
			Name:        "Specific Node Pattern",
			Description: "Nodes matching compute node naming pattern",
			ViewType:    "nodes",
			FilterStr:   "name=~^compute[0-9]{3}$",
		},
	}

	// Global presets (work across views)
	globalPresets := []FilterPreset{
		{
			Name:        "Production",
			Description: "Filter for production resources",
			ViewType:    "all",
			FilterStr:   "partition=production",
			IsGlobal:    true,
		},
		{
			Name:        "Development",
			Description: "Filter for development resources",
			ViewType:    "all",
			FilterStr:   "partition in (dev,debug)",
			IsGlobal:    true,
		},
	}

	// Add presets
	for _, preset := range jobPresets {
		_ = m.AddPreset(preset)
	}
	for _, preset := range nodePresets {
		_ = m.AddPreset(preset)
	}
	for _, preset := range globalPresets {
		_ = m.AddPreset(preset)
	}

	// Save to disk
	_ = m.savePresets()
}

// GetPresets returns presets for a specific view type
func (m *PresetManager) GetPresets(viewType string) []FilterPreset {
	var result []FilterPreset

	// Add view-specific presets
	if presets, ok := m.presets[viewType]; ok {
		result = append(result, presets...)
	}

	// Add global presets
	if globalPresets, ok := m.presets["all"]; ok {
		result = append(result, globalPresets...)
	}

	return result
}

// AddPreset adds a new preset
func (m *PresetManager) AddPreset(preset FilterPreset) error {
	// Validate preset
	if preset.Name == "" {
		return fmt.Errorf("preset name is required")
	}
	if preset.ViewType == "" {
		preset.ViewType = "all"
		preset.IsGlobal = true
	}

	// Add to appropriate list
	m.presets[preset.ViewType] = append(m.presets[preset.ViewType], preset)
	if preset.IsGlobal {
		m.presets["all"] = append(m.presets["all"], preset)
	}

	// Save to disk
	return m.savePresets()
}

// RemovePreset removes a preset by name and view type
func (m *PresetManager) RemovePreset(name, viewType string) error {
	if presets, ok := m.presets[viewType]; ok {
		for i, preset := range presets {
			if preset.Name == name {
				m.presets[viewType] = append(presets[:i], presets[i+1:]...)
				return m.savePresets()
			}
		}
	}
	return fmt.Errorf("preset not found")
}

// UpdatePreset updates an existing preset
func (m *PresetManager) UpdatePreset(oldName, viewType string, newPreset FilterPreset) error {
	if presets, ok := m.presets[viewType]; ok {
		for i, preset := range presets {
			if preset.Name == oldName {
				m.presets[viewType][i] = newPreset
				return m.savePresets()
			}
		}
	}
	return fmt.Errorf("preset not found")
}
