package streaming

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
	"github.com/jontk/s9s/internal/fileperms"
)

// FilterManager manages filters and presets for streaming
type FilterManager struct {
	filters     map[string]*StreamFilter
	chains      map[string]*FilterChain
	presets     map[string]*FilterPreset
	activeChain *FilterChain
	presetsPath string
	mu          sync.RWMutex
}

// NewFilterManager creates a new filter manager
func NewFilterManager(configPath string) *FilterManager {
	fm := &FilterManager{
		filters:     make(map[string]*StreamFilter),
		chains:      make(map[string]*FilterChain),
		presets:     make(map[string]*FilterPreset),
		presetsPath: filepath.Join(configPath, "filter_presets.json"),
	}

	// Load saved presets
	_ = fm.loadPresets()

	// Add common presets if none exist
	if len(fm.presets) == 0 {
		for _, preset := range GetCommonPresets() {
			fm.presets[preset.ID] = preset
		}
		_ = fm.savePresets()
	}

	return fm
}

// AddFilter adds a new filter
func (fm *FilterManager) AddFilter(filter *StreamFilter) error {
	fm.mu.Lock()
	defer fm.mu.Unlock()

	if filter.ID == "" {
		filter.ID = GenerateFilterID()
	}

	fm.filters[filter.ID] = filter
	return nil
}

// RemoveFilter removes a filter by ID
func (fm *FilterManager) RemoveFilter(filterID string) error {
	fm.mu.Lock()
	defer fm.mu.Unlock()

	if _, exists := fm.filters[filterID]; !exists {
		return fmt.Errorf("filter %s not found", filterID)
	}

	delete(fm.filters, filterID)

	// Remove from active chain if present
	if fm.activeChain != nil {
		newFilters := make([]*StreamFilter, 0)
		for _, f := range fm.activeChain.Filters {
			if f.ID != filterID {
				newFilters = append(newFilters, f)
			}
		}
		fm.activeChain.Filters = newFilters
	}

	return nil
}

// GetFilter retrieves a filter by ID
func (fm *FilterManager) GetFilter(filterID string) (*StreamFilter, error) {
	fm.mu.RLock()
	defer fm.mu.RUnlock()

	filter, exists := fm.filters[filterID]
	if !exists {
		return nil, fmt.Errorf("filter %s not found", filterID)
	}

	return filter, nil
}

// CreateChain creates a new filter chain
func (fm *FilterManager) CreateChain(name string, mode ChainMode, filterIDs []string) (*FilterChain, error) {
	fm.mu.Lock()
	defer fm.mu.Unlock()

	chain := &FilterChain{
		ID:      GenerateFilterID(),
		Name:    name,
		Mode:    mode,
		Active:  true,
		Filters: make([]*StreamFilter, 0),
	}

	// Add filters to chain
	for _, id := range filterIDs {
		if filter, exists := fm.filters[id]; exists {
			chain.Filters = append(chain.Filters, filter)
		}
	}

	fm.chains[chain.ID] = chain
	return chain, nil
}

// SetActiveChain sets the active filter chain
func (fm *FilterManager) SetActiveChain(chainID string) error {
	fm.mu.Lock()
	defer fm.mu.Unlock()

	if chainID == "" {
		fm.activeChain = nil
		return nil
	}

	chain, exists := fm.chains[chainID]
	if !exists {
		return fmt.Errorf("chain %s not found", chainID)
	}

	fm.activeChain = chain
	return nil
}

// ApplyActiveFilters applies the active filter chain to a line
func (fm *FilterManager) ApplyActiveFilters(line string, timestamp time.Time) (bool, map[string][]int) {
	fm.mu.RLock()
	defer fm.mu.RUnlock()

	if fm.activeChain == nil {
		return true, nil
	}

	return fm.activeChain.Apply(line, timestamp)
}

// QuickFilter creates and activates a simple keyword filter
func (fm *FilterManager) QuickFilter(pattern string, filterType FilterType) error {
	filter, err := NewStreamFilter(filterType, pattern)
	if err != nil {
		return err
	}

	fm.mu.Lock()
	defer fm.mu.Unlock()

	// Add filter
	fm.filters[filter.ID] = filter

	// Create or update quick filter chain
	quickChain := &FilterChain{
		ID:      "quick_filter",
		Name:    "Quick Filter",
		Mode:    ChainModeAND,
		Active:  true,
		Filters: []*StreamFilter{filter},
	}

	fm.chains[quickChain.ID] = quickChain
	fm.activeChain = quickChain

	return nil
}

// ClearFilters removes all active filters
func (fm *FilterManager) ClearFilters() {
	fm.mu.Lock()
	defer fm.mu.Unlock()

	fm.activeChain = nil
	// Keep filters in memory for potential reuse
}

// SavePreset saves the current filter configuration as a preset
func (fm *FilterManager) SavePreset(name, description, category string) (*FilterPreset, error) {
	fm.mu.Lock()
	defer fm.mu.Unlock()

	if fm.activeChain == nil || len(fm.activeChain.Filters) == 0 {
		return nil, fmt.Errorf("no active filters to save")
	}

	preset := &FilterPreset{
		ID:          GenerateFilterID(),
		Name:        name,
		Description: description,
		Category:    category,
		Filters:     make([]*StreamFilter, len(fm.activeChain.Filters)),
		Created:     time.Now(),
		Tags:        []string{},
	}

	// Deep copy filters
	for i, f := range fm.activeChain.Filters {
		filterCopy := *f
		preset.Filters[i] = &filterCopy
	}

	fm.presets[preset.ID] = preset
	_ = fm.savePresets()

	return preset, nil
}

// LoadPreset loads and activates a filter preset
func (fm *FilterManager) LoadPreset(presetID string) error {
	fm.mu.Lock()
	defer fm.mu.Unlock()

	preset, exists := fm.presets[presetID]
	if !exists {
		return fmt.Errorf("preset %s not found", presetID)
	}

	// Create chain from preset
	chain := &FilterChain{
		ID:          GenerateFilterID(),
		Name:        preset.Name,
		Description: preset.Description,
		Mode:        ChainModeAND,
		Active:      true,
		Filters:     make([]*StreamFilter, 0),
	}

	// Add filters to manager and chain
	for _, f := range preset.Filters {
		filterCopy := *f
		filterCopy.ID = GenerateFilterID()
		filterCopy.Stats = FilterStats{Created: time.Now()}

		fm.filters[filterCopy.ID] = &filterCopy
		chain.Filters = append(chain.Filters, &filterCopy)
	}

	fm.chains[chain.ID] = chain
	fm.activeChain = chain

	// Update preset usage
	preset.LastUsed = time.Now()
	preset.UseCount++
	_ = fm.savePresets()

	return nil
}

// GetPresets returns all available presets
func (fm *FilterManager) GetPresets() []*FilterPreset {
	fm.mu.RLock()
	defer fm.mu.RUnlock()

	presets := make([]*FilterPreset, 0, len(fm.presets))
	for _, preset := range fm.presets {
		presets = append(presets, preset)
	}

	return presets
}

// GetFilterStats returns statistics for all active filters
func (fm *FilterManager) GetFilterStats() map[string]FilterStats {
	fm.mu.RLock()
	defer fm.mu.RUnlock()

	stats := make(map[string]FilterStats)

	if fm.activeChain != nil {
		for _, filter := range fm.activeChain.Filters {
			stats[filter.ID] = filter.Stats
		}
	}

	return stats
}

// loadPresets loads saved presets from disk
func (fm *FilterManager) loadPresets() error {
	data, err := os.ReadFile(fm.presetsPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // No presets file yet
		}
		return err
	}

	var presets []*FilterPreset
	if err := json.Unmarshal(data, &presets); err != nil {
		return err
	}

	for _, preset := range presets {
		fm.presets[preset.ID] = preset
	}

	return nil
}

// savePresets saves presets to disk
func (fm *FilterManager) savePresets() error {
	presets := make([]*FilterPreset, 0, len(fm.presets))
	for _, preset := range fm.presets {
		presets = append(presets, preset)
	}

	data, err := json.MarshalIndent(presets, "", "  ")
	if err != nil {
		return err
	}

	// Ensure directory exists
	dir := filepath.Dir(fm.presetsPath)
	if err := os.MkdirAll(dir, fileperms.ConfigDir); err != nil {
		return err
	}

	return os.WriteFile(fm.presetsPath, data, fileperms.ConfigFile)
}
