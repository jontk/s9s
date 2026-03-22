package app

import (
	"context"
	"testing"

	"github.com/jontk/s9s/internal/config"
	"github.com/jontk/s9s/internal/version"
	"github.com/stretchr/testify/require"
)

func newTestAppForUpdate(t *testing.T) *S9s {
	t.Helper()
	cfg := &config.Config{
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
		Update: config.UpdateConfig{
			Enabled:       true,
			CheckInterval: "24h",
		},
	}

	app, err := New(context.Background(), cfg)
	require.NoError(t, err)
	return app
}

func TestCheckForUpdates_DevBuildSkips(t *testing.T) {
	orig := version.Version
	defer func() { version.Version = orig }()
	version.Version = "dev"

	app := newTestAppForUpdate(t)
	app.checkForUpdates()
}

func TestCheckForUpdates_UpToDateSkips(t *testing.T) {
	orig := version.Version
	defer func() { version.Version = orig }()
	// Version higher than any release — exercises interval parsing,
	// state loading, and the "not newer" exit path without hitting
	// QueueUpdateDraw (which blocks without a running tview loop).
	version.Version = "999.999.999"

	app := newTestAppForUpdate(t)
	app.checkForUpdates()
}

func TestCheckForUpdates_CancelledContextSkips(t *testing.T) {
	orig := version.Version
	defer func() { version.Version = orig }()
	version.Version = "0.0.1"

	app := newTestAppForUpdate(t)
	app.config.Update.CheckInterval = "0s" // force check

	// Cancel context so the HTTP call fails immediately.
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	app.ctx = ctx

	// Exercises: interval parse, NeedsCheck(0s)=true, context timeout,
	// debug log on failure, and clean return.
	app.checkForUpdates()
}
