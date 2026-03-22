package update

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

const stateFileName = "update-state.json"

// State persists update check metadata.
type State struct {
	LastChecked    time.Time `json:"lastChecked"`
	LatestVersion  string    `json:"latestVersion"`
	LatestURL      string    `json:"latestURL"`
	CurrentVersion string    `json:"currentVersion"`
}

// LoadState reads the update state from ~/.s9s/update-state.json.
// Returns an empty state (not an error) if the file is missing or corrupt.
func LoadState() *State {
	return LoadStateFrom(defaultStateDir())
}

// LoadStateFrom reads the update state from a specific directory.
func LoadStateFrom(dir string) *State {
	data, err := os.ReadFile(filepath.Join(dir, stateFileName))
	if err != nil {
		return &State{}
	}

	var s State
	if err := json.Unmarshal(data, &s); err != nil {
		return &State{}
	}

	return &s
}

// Save writes the state to ~/.s9s/update-state.json atomically.
func (s *State) Save() error {
	return s.SaveTo(defaultStateDir())
}

// SaveTo writes the state to a specific directory atomically.
func (s *State) SaveTo(dir string) error {
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return err
	}

	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}

	tmp := filepath.Join(dir, stateFileName+".tmp")
	if err := os.WriteFile(tmp, data, 0o640); err != nil {
		return err
	}

	return os.Rename(tmp, filepath.Join(dir, stateFileName))
}

// NeedsCheck returns true if enough time has passed since the last check.
func (s *State) NeedsCheck(interval time.Duration) bool {
	if s.LastChecked.IsZero() {
		return true
	}
	return time.Since(s.LastChecked) >= interval
}

func defaultStateDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".s9s")
}
