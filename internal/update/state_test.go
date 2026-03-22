package update

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestState_SaveAndLoad(t *testing.T) {
	dir := t.TempDir()

	state := &State{
		LastChecked:    time.Date(2026, 3, 22, 10, 0, 0, 0, time.UTC),
		LatestVersion:  "0.8.0",
		LatestURL:      "https://github.com/jontk/s9s/releases/tag/v0.8.0",
		CurrentVersion: "0.7.1",
	}

	if err := state.SaveTo(dir); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(filepath.Join(dir, stateFileName)); err != nil {
		t.Fatalf("State file not created: %v", err)
	}

	loaded := LoadStateFrom(dir)
	if loaded.LatestVersion != "0.8.0" {
		t.Errorf("LatestVersion = %q, want %q", loaded.LatestVersion, "0.8.0")
	}
	if loaded.CurrentVersion != "0.7.1" {
		t.Errorf("CurrentVersion = %q, want %q", loaded.CurrentVersion, "0.7.1")
	}
	if loaded.LatestURL != state.LatestURL {
		t.Errorf("LatestURL = %q, want %q", loaded.LatestURL, state.LatestURL)
	}
}

func TestState_LoadMissingFile(t *testing.T) {
	dir := t.TempDir()
	state := LoadStateFrom(dir)

	if !state.LastChecked.IsZero() {
		t.Errorf("expected zero LastChecked for missing file")
	}
	if state.LatestVersion != "" {
		t.Errorf("expected empty LatestVersion for missing file")
	}
}

func TestState_LoadCorruptFile(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, stateFileName), []byte("{invalid json"), 0o640); err != nil {
		t.Fatal(err)
	}

	state := LoadStateFrom(dir)
	if !state.LastChecked.IsZero() {
		t.Errorf("expected zero LastChecked for corrupt file")
	}
}

func TestState_NeedsCheck(t *testing.T) {
	tests := []struct {
		name        string
		lastChecked time.Time
		interval    time.Duration
		want        bool
	}{
		{"zero time", time.Time{}, 24 * time.Hour, true},
		{"recently checked", time.Now().Add(-1 * time.Hour), 24 * time.Hour, false},
		{"interval elapsed", time.Now().Add(-25 * time.Hour), 24 * time.Hour, true},
		{"exactly at boundary", time.Now().Add(-24 * time.Hour), 24 * time.Hour, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &State{LastChecked: tt.lastChecked}
			got := s.NeedsCheck(tt.interval)
			if got != tt.want {
				t.Errorf("NeedsCheck() = %v, want %v", got, tt.want)
			}
		})
	}
}
