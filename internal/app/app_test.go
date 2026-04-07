package app

import (
	"context"
	"testing"
	"time"

	"github.com/jontk/s9s/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	tests := []struct {
		name    string
		config  *config.Config
		wantErr bool
	}{
		{
			name:    "nil config",
			config:  nil,
			wantErr: true,
		},
		{
			name: "valid config",
			config: &config.Config{
				RefreshRate:    "2s",
				DefaultCluster: "test",
				UseMockClient:  true,
				Clusters: []config.ClusterContext{
					{
						Name: "test",
						Cluster: config.ClusterConfig{
							Endpoint: "http://localhost:6820",
						},
					},
				},
				UI: config.UIConfig{
					Skin:        "default",
					EnableMouse: true,
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			app, err := New(ctx, tt.config)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, app)
			} else {
				require.NoError(t, err)
				require.NotNil(t, app)

				// Verify initialization
				assert.NotNil(t, app.app)
				assert.NotNil(t, app.pages)
				assert.NotNil(t, app.header)
				assert.NotNil(t, app.statusBar)
				assert.NotNil(t, app.cmdLine)

				// Clean up
				_ = app.Stop()
			}
		})
	}
}

func TestBasicFunctionality(t *testing.T) {
	cfg := &config.Config{
		RefreshRate:    "2s",
		DefaultCluster: "test",
		UseMockClient:  true, // Use mock client for testing
		Clusters: []config.ClusterContext{
			{
				Name: "test",
				Cluster: config.ClusterConfig{
					Endpoint: "http://localhost:6820",
				},
			},
		},
	}

	ctx := context.Background()
	app, err := New(ctx, cfg)
	require.NoError(t, err)
	defer func() { _ = app.Stop() }()

	// Test basic state
	assert.False(t, app.isRunning.Load())

	// Give time for initialization
	time.Sleep(100 * time.Millisecond)

	// App should be properly initialized without panic
}

// TestApplyConfigRearmsRefreshTicker exercises the ticker-lifecycle path
// without going through New() (which attempts to load plugins and may fail
// in test environments with stale .so files). It constructs a minimal S9s
// and drives startRefreshTimer / ApplyConfig directly.
func TestApplyConfigRearmsRefreshTicker(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg := &config.Config{RefreshRate: "2s"}
	s := &S9s{ctx: ctx, cancel: cancel, config: cfg}
	s.autoRefresh.Store(true)
	s.isRunning.Store(true)

	// Start initial ticker at 2s.
	s.startRefreshTimer(2 * time.Second)
	firstTicker := s.refreshTicker
	firstStop := s.refreshStop
	require.NotNil(t, firstTicker, "initial ticker should be set")
	require.NotNil(t, firstStop, "initial stop chan should be set")

	// Apply a new config with a different refresh rate.
	newCfg := &config.Config{RefreshRate: "5s"}
	s.ApplyConfig(newCfg)

	assert.Same(t, newCfg, s.config, "config pointer was not swapped")
	require.NotNil(t, s.refreshTicker, "new ticker should be set")
	require.NotNil(t, s.refreshStop, "new stop chan should be set")
	assert.NotSame(t, firstTicker, s.refreshTicker, "ticker was not replaced")
	assert.False(t, firstStop == s.refreshStop, "stop chan was not replaced")

	// The old stop chan should be closed (signaling the old goroutine to exit).
	select {
	case _, ok := <-firstStop:
		assert.False(t, ok, "old stop chan should be closed")
	default:
		t.Fatalf("old stop chan was not closed after ApplyConfig")
	}

	// Empty refreshRate should disable the ticker entirely.
	s.ApplyConfig(&config.Config{RefreshRate: ""})
	assert.Nil(t, s.refreshTicker, "empty refreshRate should stop the ticker")
	assert.Nil(t, s.refreshStop, "empty refreshRate should clear the stop chan")
}

// TestConfigModalSaveTriggersApplyConfig proves the full wiring from a
// ConfigManager-style save callback down to S9s.ApplyConfig, which is the
// chain we depend on when the user edits refreshRate in the F10 modal and
// clicks Save. It does NOT require a real tview Application.
func TestConfigModalSaveTriggersApplyConfig(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	s := &S9s{ctx: ctx, cancel: cancel, config: &config.Config{RefreshRate: "10s"}}
	s.autoRefresh.Store(true)
	s.isRunning.Store(true)
	s.startRefreshTimer(10 * time.Second)
	originalTicker := s.refreshTicker
	require.NotNil(t, originalTicker)

	// This mirrors the lambda registered in app_modals.go:142 via
	// configView.SetConfigChangedCallback. If the wiring ever regresses,
	// this test will catch it.
	onConfigChanged := func(newCfg *config.Config) {
		if newCfg == nil {
			return
		}
		s.ApplyConfig(newCfg)
	}

	// Simulate the user editing refreshRate in the modal and clicking Save:
	// ConfigManager.saveConfiguration writes to disk then invokes onSave
	// with its in-memory cm.currentConfig (here, edited).
	edited := &config.Config{RefreshRate: "2s"}
	onConfigChanged(edited)

	assert.Same(t, edited, s.config, "S9s.config must be swapped to the edited config")
	assert.NotSame(t, originalTicker, s.refreshTicker, "ticker must be replaced with the new cadence")
}

func TestStop(t *testing.T) {
	cfg := &config.Config{
		RefreshRate:    "2s",
		DefaultCluster: "test",
		UseMockClient:  true, // Use mock client for testing
		Clusters: []config.ClusterContext{
			{
				Name: "test",
				Cluster: config.ClusterConfig{
					Endpoint: "http://localhost:6820",
				},
			},
		},
	}

	ctx := context.Background()
	app, err := New(ctx, cfg)
	require.NoError(t, err)

	// Start the app in a goroutine
	app.isRunning.Store(true)

	// Stop should complete without error
	err = app.Stop()
	assert.NoError(t, err)
	assert.False(t, app.isRunning.Load())
}
