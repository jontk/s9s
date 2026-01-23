package preferences

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/jontk/s9s/internal/streaming"
	"github.com/jontk/s9s/internal/fileperms"
)

// StreamingPreferences represents user preferences for streaming functionality
type StreamingPreferences struct {
	// General streaming settings
	AutoStartForRunningJobs bool   `json:"auto_start_for_running_jobs"`
	DefaultAutoScroll       bool   `json:"default_auto_scroll"`
	ShowTimestamps          bool   `json:"show_timestamps"`
	ExportFormat            string `json:"export_format"`

	// Performance settings
	MaxConcurrentStreams int `json:"max_concurrent_streams"`
	BufferSizeLines      int `json:"buffer_size_lines"`
	PollIntervalSeconds  int `json:"poll_interval_seconds"`
	MaxMemoryMB          int `json:"max_memory_mb"`
	FileCheckIntervalMs  int `json:"file_check_interval_ms"`

	// Remote streaming settings
	EnableRemoteStreaming bool `json:"enable_remote_streaming"`
	SSHTimeout            int  `json:"ssh_timeout_seconds"`
	RemoteBufferSize      int  `json:"remote_buffer_size"`

	// Display settings
	MultiStreamGridSize string   `json:"multi_stream_grid_size"` // "2x2", "3x3", "2x3"
	StreamPanelHeight   int      `json:"stream_panel_height"`
	ShowBufferStats     bool     `json:"show_buffer_stats"`
	HighlightPatterns   []string `json:"highlight_patterns"`

	// Advanced settings
	EnableCompression      bool `json:"enable_compression"`
	StreamHistoryDays      int  `json:"stream_history_days"`
	AutoCleanupInactive    bool `json:"auto_cleanup_inactive"`
	InactiveTimeoutMinutes int  `json:"inactive_timeout_minutes"`
}

// StreamingPreferencesManager manages streaming preferences
type StreamingPreferencesManager struct {
	preferences *StreamingPreferences
	configPath  string
	mu          sync.RWMutex
}

// DefaultStreamingPreferences returns default streaming preferences
func DefaultStreamingPreferences() *StreamingPreferences {
	return &StreamingPreferences{
		// General settings
		AutoStartForRunningJobs: true,
		DefaultAutoScroll:       true,
		ShowTimestamps:          true,
		ExportFormat:            "txt",

		// Performance settings
		MaxConcurrentStreams: 4,
		BufferSizeLines:      10000,
		PollIntervalSeconds:  2,
		MaxMemoryMB:          50,
		FileCheckIntervalMs:  1000,

		// Remote streaming settings
		EnableRemoteStreaming: true,
		SSHTimeout:            30,
		RemoteBufferSize:      5000,

		// Display settings
		MultiStreamGridSize: "2x2",
		StreamPanelHeight:   20,
		ShowBufferStats:     true,
		HighlightPatterns:   []string{"ERROR", "WARNING", "FAILED", "SUCCESS"},

		// Advanced settings
		EnableCompression:      false,
		StreamHistoryDays:      7,
		AutoCleanupInactive:    true,
		InactiveTimeoutMinutes: 30,
	}
}

// NewStreamingPreferencesManager creates a new streaming preferences manager
func NewStreamingPreferencesManager(configDir string) (*StreamingPreferencesManager, error) {
	if configDir == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return nil, err
		}
		configDir = filepath.Join(homeDir, ".s9s")
	}

	configPath := filepath.Join(configDir, "streaming_preferences.json")

	manager := &StreamingPreferencesManager{
		configPath:  configPath,
		preferences: DefaultStreamingPreferences(),
	}

	// Load existing preferences if available
	if err := manager.Load(); err != nil && !os.IsNotExist(err) {
		return nil, err
	}

	return manager, nil
}

// Load loads preferences from disk
func (m *StreamingPreferencesManager) Load() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	data, err := os.ReadFile(m.configPath)
	if err != nil {
		return err
	}

	var prefs StreamingPreferences
	if err := json.Unmarshal(data, &prefs); err != nil {
		return err
	}

	m.preferences = &prefs
	return nil
}

// Save saves preferences to disk
func (m *StreamingPreferencesManager) Save() error {
	m.mu.RLock()
	data, err := json.MarshalIndent(m.preferences, "", "  ")
	m.mu.RUnlock()

	if err != nil {
		return err
	}

	// Ensure directory exists
	dir := filepath.Dir(m.configPath)
	if err := os.MkdirAll(dir, fileperms.ConfigDir); err != nil {
		return err
	}

	return os.WriteFile(m.configPath, data, fileperms.ConfigFile)
}

// GetPreferences returns a copy of the current preferences
func (m *StreamingPreferencesManager) GetPreferences() StreamingPreferences {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Return a copy to prevent external modifications
	return *m.preferences
}

// UpdatePreferences updates preferences with validation
func (m *StreamingPreferencesManager) UpdatePreferences(update func(*StreamingPreferences)) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Create a copy for validation
	newPrefs := *m.preferences
	update(&newPrefs)

	// Validate preferences
	if err := m.validatePreferences(&newPrefs); err != nil {
		return err
	}

	m.preferences = &newPrefs
	return m.Save()
}

// validatePreferences validates preference values
func (m *StreamingPreferencesManager) validatePreferences(prefs *StreamingPreferences) error {
	// Validate numeric ranges
	if prefs.MaxConcurrentStreams < 1 || prefs.MaxConcurrentStreams > 16 {
		prefs.MaxConcurrentStreams = 4
	}

	if prefs.BufferSizeLines < 100 || prefs.BufferSizeLines > 100000 {
		prefs.BufferSizeLines = 10000
	}

	if prefs.PollIntervalSeconds < 1 || prefs.PollIntervalSeconds > 60 {
		prefs.PollIntervalSeconds = 2
	}

	if prefs.MaxMemoryMB < 10 || prefs.MaxMemoryMB > 1000 {
		prefs.MaxMemoryMB = 50
	}

	if prefs.FileCheckIntervalMs < 100 || prefs.FileCheckIntervalMs > 10000 {
		prefs.FileCheckIntervalMs = 1000
	}

	// Validate grid size
	validGridSizes := map[string]bool{
		"2x2": true, "3x3": true, "2x3": true, "3x2": true, "4x4": true,
	}
	if !validGridSizes[prefs.MultiStreamGridSize] {
		prefs.MultiStreamGridSize = "2x2"
	}

	// Validate export format
	validFormats := map[string]bool{
		"txt": true, "json": true, "csv": true, "md": true,
	}
	if !validFormats[prefs.ExportFormat] {
		prefs.ExportFormat = "txt"
	}

	return nil
}

// ToStreamConfig converts preferences to streaming.StreamConfig
func (m *StreamingPreferencesManager) ToStreamConfig() *streaming.StreamConfig {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return &streaming.StreamConfig{
		MaxConcurrentStreams: m.preferences.MaxConcurrentStreams,
		BufferSize:           m.preferences.BufferSizeLines,
		PollInterval:         time.Duration(m.preferences.PollIntervalSeconds) * time.Second,
		MaxMemoryMB:          m.preferences.MaxMemoryMB,
		AutoScroll:           m.preferences.DefaultAutoScroll,
		ShowTimestamps:       m.preferences.ShowTimestamps,
		ExportFormat:         m.preferences.ExportFormat,
	}
}

// ToSlurmConfig converts preferences to streaming.SlurmConfig
func (m *StreamingPreferencesManager) ToSlurmConfig() *streaming.SlurmConfig {
	m.mu.RLock()
	defer m.mu.RUnlock()

	config := streaming.DefaultSlurmConfig()
	config.FileCheckInterval = time.Duration(m.preferences.FileCheckIntervalMs) * time.Millisecond
	config.BufferSize = m.preferences.BufferSizeLines
	config.RemoteAccess = m.preferences.EnableRemoteStreaming

	return config
}

// Reset resets preferences to defaults
func (m *StreamingPreferencesManager) Reset() error {
	m.mu.Lock()
	m.preferences = DefaultStreamingPreferences()
	m.mu.Unlock()

	return m.Save()
}

// GetConfigPath returns the configuration file path
func (m *StreamingPreferencesManager) GetConfigPath() string {
	return m.configPath
}

// SetHighlightPatterns sets the highlight patterns
func (m *StreamingPreferencesManager) SetHighlightPatterns(patterns []string) error {
	return m.UpdatePreferences(func(p *StreamingPreferences) {
		p.HighlightPatterns = patterns
	})
}

// GetHighlightPatterns returns the current highlight patterns
func (m *StreamingPreferencesManager) GetHighlightPatterns() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Return a copy of the slice
	patterns := make([]string, len(m.preferences.HighlightPatterns))
	copy(patterns, m.preferences.HighlightPatterns)
	return patterns
}

// GetGridDimensions returns the grid dimensions from the grid size string
func (m *StreamingPreferencesManager) GetGridDimensions() (rows, cols int) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	switch m.preferences.MultiStreamGridSize {
	case "2x2":
		return 2, 2
	case "3x3":
		return 3, 3
	case "2x3":
		return 2, 3
	case "3x2":
		return 3, 2
	case "4x4":
		return 4, 4
	default:
		return 2, 2
	}
}

// GetMaxStreams returns the maximum number of concurrent streams
func (m *StreamingPreferencesManager) GetMaxStreams() int {
	rows, cols := m.GetGridDimensions()
	maxFromGrid := rows * cols

	m.mu.RLock()
	maxFromPrefs := m.preferences.MaxConcurrentStreams
	m.mu.RUnlock()

	// Return the minimum of grid capacity and preference setting
	if maxFromGrid < maxFromPrefs {
		return maxFromGrid
	}
	return maxFromPrefs
}
