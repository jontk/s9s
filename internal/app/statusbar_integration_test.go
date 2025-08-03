package app

import (
	"context"
	"testing"

	"github.com/jontk/s9s/internal/config"
)

func TestStatusBarHintsAfterViewSwitching(t *testing.T) {
	// Create a test app with mock client
	cfg := &config.Config{
		UseMockClient: true,
		RefreshRate:   "", // Disable auto-refresh for testing
	}

	ctx := context.Background()
	app, err := New(ctx, cfg)
	if err != nil {
		t.Fatalf("Failed to create app: %v", err)
	}
	defer app.Stop()

	// Switch to jobs view
	err = app.viewMgr.SetCurrentView("jobs")
	if err != nil {
		t.Fatalf("Failed to switch to jobs view: %v", err)
	}
	app.updateCurrentView()

	// Get the current view hints
	currentView, err := app.viewMgr.GetCurrentView()
	if err != nil {
		t.Fatalf("Failed to get current view: %v", err)
	}

	expectedHints := currentView.Hints()
	if len(expectedHints) == 0 {
		t.Fatal("Expected hints from jobs view, got none")
	}

	// Switch to nodes view and back
	err = app.viewMgr.SetCurrentView("nodes")
	if err != nil {
		t.Fatalf("Failed to switch to nodes view: %v", err)
	}
	app.updateCurrentView()

	err = app.viewMgr.SetCurrentView("jobs")
	if err != nil {
		t.Fatalf("Failed to switch back to jobs view: %v", err)
	}
	app.updateCurrentView()

	// Verify hints are still displayed (no message should be present)
	text := app.statusBar.TextView.GetText(false)

	// The text should contain the hints, not be empty
	if text == "" {
		t.Error("Status bar is empty after view switching - hints should be displayed")
	}

	// Verify it contains expected hint keywords from the view
	if !contains(text, "F1") && !contains(text, "Help") {
		t.Error("Status bar should contain hint keywords after view switching")
	}

	t.Logf("Status bar shows: %s", text)
}

func TestStatusBarHintsAfterRefresh(t *testing.T) {
	// Create a test app with mock client
	cfg := &config.Config{
		UseMockClient: true,
		RefreshRate:   "", // Disable auto-refresh for testing
	}

	ctx := context.Background()
	app, err := New(ctx, cfg)
	if err != nil {
		t.Fatalf("Failed to create app: %v", err)
	}
	defer app.Stop()

	// Switch to jobs view
	err = app.viewMgr.SetCurrentView("jobs")
	if err != nil {
		t.Fatalf("Failed to switch to jobs view: %v", err)
	}
	app.updateCurrentView()

	// Simulate manual refresh (like F5 key)
	err = app.viewMgr.RefreshCurrentView()
	if err != nil {
		t.Fatalf("Failed to refresh view: %v", err)
	}

	// The refresh doesn't show a message in RefreshCurrentView(),
	// so hints should still be visible
	text := app.statusBar.TextView.GetText(false)
	if text == "" {
		t.Error("Status bar is empty after refresh - hints should be displayed")
	}

	t.Logf("Status bar after refresh: %s", text)
}

func TestStatusBarHintsWithTemporaryMessage(t *testing.T) {
	// Create a test app with mock client
	cfg := &config.Config{
		UseMockClient: true,
		RefreshRate:   "", // Disable auto-refresh for testing
	}

	ctx := context.Background()
	app, err := New(ctx, cfg)
	if err != nil {
		t.Fatalf("Failed to create app: %v", err)
	}
	defer app.Stop()

	// Switch to jobs view
	err = app.viewMgr.SetCurrentView("jobs")
	if err != nil {
		t.Fatalf("Failed to switch to jobs view: %v", err)
	}
	app.updateCurrentView()

	// Show a temporary success message
	app.statusBar.Success("Test message")

	// Should show the message, not hints
	text := app.statusBar.TextView.GetText(false)
	if !contains(text, "Test message") {
		t.Errorf("Expected success message to be displayed, got: %s", text)
	}

	// Clear the message manually (simulating expiry)
	app.statusBar.ClearMessage()

	// Should show hints again
	text = app.statusBar.TextView.GetText(false)
	if text == "" {
		t.Error("Status bar is empty after clearing message - hints should be restored")
	}

	t.Logf("Status bar after clearing message: %s", text)
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) &&
		   (s == substr ||
		    (len(s) > len(substr) &&
		     (s[:len(substr)] == substr ||
		      s[len(s)-len(substr):] == substr ||
		      containsSubstring(s, substr))))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}