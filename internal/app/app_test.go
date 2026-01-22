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
				CurrentContext: "test",
				UseMockClient:  true,
				Contexts: []config.ContextConfig{
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
		CurrentContext: "test",
		UseMockClient:  true, // Use mock client for testing
		Contexts: []config.ContextConfig{
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
	assert.False(t, app.isRunning)

	// Give time for initialization
	time.Sleep(100 * time.Millisecond)

	// App should be properly initialized without panic
}

func TestStop(t *testing.T) {
	cfg := &config.Config{
		RefreshRate:    "2s",
		CurrentContext: "test",
		UseMockClient:  true, // Use mock client for testing
		Contexts: []config.ContextConfig{
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
	app.isRunning = true

	// Stop should complete without error
	err = app.Stop()
	assert.NoError(t, err)
	assert.False(t, app.isRunning)
}
