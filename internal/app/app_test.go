package app

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/jontk/s9s/internal/config"
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
				RefreshRate: "2s",
				Cluster: config.ClusterConfig{
					Endpoint: "http://localhost:6820",
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
				assert.NotNil(t, app.flash)
				assert.NotNil(t, app.cmdLine)
				assert.NotNil(t, app.views)
				assert.NotNil(t, app.errorChan)
				assert.NotNil(t, app.flashChan)
				
				// Clean up
				app.Stop(context.Background())
			}
		})
	}
}

func TestFlashMessages(t *testing.T) {
	cfg := &config.Config{
		RefreshRate: "2s",
		Cluster: config.ClusterConfig{
			Endpoint: "http://localhost:6820",
		},
	}
	
	ctx := context.Background()
	app, err := New(ctx, cfg)
	require.NoError(t, err)
	defer app.Stop(context.Background())
	
	// Start flash message handler
	go app.handleFlashMessages()
	
	// Test different flash levels
	app.Flash(flashInfo, "Info message")
	app.Flash(flashSuccess, "Success message")
	app.Flash(flashWarning, "Warning message")
	app.Flash(flashError, "Error message")
	
	// Give time for messages to be processed
	time.Sleep(100 * time.Millisecond)
	
	// Messages should have been processed without panic
	// (In a real test, we'd check the UI output)
}

func TestStop(t *testing.T) {
	cfg := &config.Config{
		RefreshRate: "2s",
		Cluster: config.ClusterConfig{
			Endpoint: "http://localhost:6820",
		},
	}
	
	ctx := context.Background()
	app, err := New(ctx, cfg)
	require.NoError(t, err)
	
	// Start the app in a goroutine
	app.isRunning = true
	
	// Create a stop context with timeout
	stopCtx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	
	// Stop should complete without error
	err = app.Stop(stopCtx)
	assert.NoError(t, err)
	assert.False(t, app.isRunning)
}