// Package preferences provides user preference management and persistence.
package preferences

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/jontk/s9s/internal/fileperms"
)

// UserPreferences represents all user-configurable preferences
type UserPreferences struct {
	mu         *sync.RWMutex
	configPath string
	Layouts    LayoutPrefs `json:"layouts"`
	lastSaved  time.Time
	onChange   []func()
}

// LayoutPrefs contains dashboard layout preferences
type LayoutPrefs struct {
	CurrentLayout string `json:"current_layout"`
}

// NewUserPreferences creates a new preferences manager
func NewUserPreferences(configPath string) (*UserPreferences, error) {
	up := &UserPreferences{
		mu:         &sync.RWMutex{},
		configPath: configPath,
		onChange:   make([]func(), 0),
	}

	// Set defaults
	up.setDefaults()

	// Try to load existing preferences
	if err := up.Load(); err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("failed to load preferences: %w", err)
	}

	return up, nil
}

// setDefaults sets default preference values
func (up *UserPreferences) setDefaults() {
	up.Layouts = LayoutPrefs{
		CurrentLayout: "default",
	}
}

// Load loads preferences from disk
func (up *UserPreferences) Load() error {
	up.mu.Lock()
	defer up.mu.Unlock()

	data, err := os.ReadFile(up.configPath)
	if err != nil {
		return err
	}

	// Parse JSON (unknown fields are silently ignored)
	var prefs UserPreferences
	if err := json.Unmarshal(data, &prefs); err != nil {
		return fmt.Errorf("failed to parse preferences: %w", err)
	}

	up.Layouts = prefs.Layouts

	return nil
}

// Save saves preferences to disk
func (up *UserPreferences) Save() error {
	up.mu.RLock()
	defer up.mu.RUnlock()

	// Create directory if needed
	dir := filepath.Dir(up.configPath)
	if err := os.MkdirAll(dir, fileperms.ConfigDir); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Marshal to JSON
	data, err := json.MarshalIndent(up, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal preferences: %w", err)
	}

	// Write to temp file first
	tempFile := up.configPath + ".tmp"
	if err := os.WriteFile(tempFile, data, fileperms.ConfigFile); err != nil {
		return fmt.Errorf("failed to write preferences: %w", err)
	}

	// Rename to actual file
	if err := os.Rename(tempFile, up.configPath); err != nil {
		_ = os.Remove(tempFile)
		return fmt.Errorf("failed to save preferences: %w", err)
	}

	up.lastSaved = time.Now()
	return nil
}

// Get returns a copy of current preferences
func (up *UserPreferences) Get() UserPreferences {
	up.mu.RLock()
	defer up.mu.RUnlock()

	return UserPreferences{
		configPath: up.configPath,
		Layouts:    up.Layouts,
		lastSaved:  up.lastSaved,
	}
}

// Update updates preferences with validation
func (up *UserPreferences) Update(update func(*UserPreferences) error) error {
	up.mu.Lock()
	defer up.mu.Unlock()

	// Create a copy for validation (mutex will be nil but that's ok for temp)
	temp := UserPreferences{
		configPath: up.configPath,
		Layouts:    up.Layouts,
		lastSaved:  up.lastSaved,
	}

	// Apply updates
	if err := update(&temp); err != nil {
		return err
	}

	// Apply validated changes (excluding mutex and callbacks)
	up.configPath = temp.configPath
	up.Layouts = temp.Layouts
	up.lastSaved = temp.lastSaved

	// Notify listeners
	for _, fn := range up.onChange {
		go fn()
	}

	// Save without auto-locking since we already hold the lock
	return up.saveWithoutLock()
}

// OnChange registers a callback for preference changes
func (up *UserPreferences) OnChange(fn func()) {
	up.mu.Lock()
	defer up.mu.Unlock()

	up.onChange = append(up.onChange, fn)
}

// Reset resets preferences to defaults
func (up *UserPreferences) Reset() error {
	up.mu.Lock()
	defer up.mu.Unlock()

	up.setDefaults()

	// Notify listeners
	for _, fn := range up.onChange {
		go fn()
	}

	return up.saveWithoutLock()
}

// saveWithoutLock saves preferences to disk without acquiring locks
func (up *UserPreferences) saveWithoutLock() error {
	// Create directory if needed
	dir := filepath.Dir(up.configPath)
	if err := os.MkdirAll(dir, fileperms.ConfigDir); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Marshal to JSON
	data, err := json.MarshalIndent(up, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal preferences: %w", err)
	}

	// Write to temp file first
	tempFile := up.configPath + ".tmp"
	if err := os.WriteFile(tempFile, data, fileperms.ConfigFile); err != nil {
		return fmt.Errorf("failed to write preferences: %w", err)
	}

	// Rename to actual file
	if err := os.Rename(tempFile, up.configPath); err != nil {
		_ = os.Remove(tempFile)
		return fmt.Errorf("failed to save preferences: %w", err)
	}

	up.lastSaved = time.Now()
	return nil
}
