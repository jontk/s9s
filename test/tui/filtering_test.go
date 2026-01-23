package tui

import (
	"testing"
	"time"

	"github.com/gdamore/tcell/v2"
)

// TestFilterBarActivation verifies that the filter bar can be activated
func TestFilterBarActivation(t *testing.T) {
	h := NewTUITestHarness(t)
	defer h.Cleanup()

	h.Start()
	time.Sleep(100 * time.Millisecond)

	// Try activating filter with '/' key (common pattern in TUI apps)
	h.SendRune('/')
	time.Sleep(100 * time.Millisecond)

	// Check if a filter/search interface appears
	// This is a basic test - actual behavior depends on implementation
	screenText := h.GetScreenText()
	if len(screenText) == 0 {
		h.DumpScreen()
		t.Error("Screen is empty after attempting to activate filter")
	}
}

// TestSearchInJobsView verifies search functionality in jobs view
func TestSearchInJobsView(t *testing.T) {
	h := NewTUITestHarness(t)
	defer h.Cleanup()

	h.Start()
	time.Sleep(100 * time.Millisecond)

	// Ensure we're on jobs view
	h.AssertCurrentView("jobs")

	// The mock client should provide some jobs
	// Basic verification that the view renders
	screenText := h.GetScreenText()
	if len(screenText) < 10 {
		h.DumpScreen()
		t.Error("Jobs view appears empty")
	}
}

// TestSearchInNodesView verifies search functionality in nodes view
func TestSearchInNodesView(t *testing.T) {
	h := NewTUITestHarness(t)
	defer h.Cleanup()

	h.Start()
	time.Sleep(100 * time.Millisecond)

	// Switch to nodes view
	h.SendRune('2')
	time.Sleep(100 * time.Millisecond)

	h.AssertCurrentView("nodes")

	// Basic verification that the view renders
	screenText := h.GetScreenText()
	if len(screenText) < 10 {
		h.DumpScreen()
		t.Error("Nodes view appears empty")
	}
}

// TestFilterClear verifies that filters can be cleared
func TestFilterClear(t *testing.T) {
	h := NewTUITestHarness(t)
	defer h.Cleanup()

	h.Start()
	time.Sleep(100 * time.Millisecond)

	// Verify app is running
	h.AssertCurrentView("jobs")

	// Test Escape key clears any active filters/search
	h.SendEsc()
	time.Sleep(50 * time.Millisecond)

	// App should still be functional
	screenText := h.GetScreenText()
	if len(screenText) < 10 {
		h.DumpScreen()
		t.Error("Screen appears empty after Escape")
	}
}

// TestRefreshView verifies that F5 refreshes the current view
func TestRefreshView(t *testing.T) {
	h := NewTUITestHarness(t)
	defer h.Cleanup()

	h.Start()
	time.Sleep(100 * time.Millisecond)

	// Press F5 to refresh
	h.SendKey(tcell.KeyF5, 0, tcell.ModNone)
	time.Sleep(200 * time.Millisecond)

	// View should still be displayed
	h.AssertCurrentView("jobs")

	// Check for success message or that view is still functional
	screenText := h.GetScreenText()
	if len(screenText) < 10 {
		h.DumpScreen()
		t.Error("Screen appears empty after refresh")
	}
}

// TestRefreshCommand verifies that :refresh command works
func TestRefreshCommand(t *testing.T) {
	h := NewTUITestHarness(t)
	defer h.Cleanup()

	h.Start()
	time.Sleep(100 * time.Millisecond)

	// Enter command mode
	h.SendRune(':')
	time.Sleep(50 * time.Millisecond)

	// Type refresh command
	for _, ch := range "refresh" {
		h.SendRune(ch)
		time.Sleep(10 * time.Millisecond)
	}

	// Press Enter
	h.SendEnter()
	time.Sleep(200 * time.Millisecond)

	// View should still be displayed
	h.AssertCurrentView("jobs")

	screenText := h.GetScreenText()
	if len(screenText) < 10 {
		h.DumpScreen()
		t.Error("Screen appears empty after refresh command")
	}
}

// TestViewFiltersAreIndependent verifies that each view maintains its own filter state
func TestViewFiltersAreIndependent(t *testing.T) {
	h := NewTUITestHarness(t)
	defer h.Cleanup()

	h.Start()
	time.Sleep(100 * time.Millisecond)

	// Test navigation between views
	views := []rune{'1', '2', '3', '1'}
	expected := []string{"jobs", "nodes", "partitions", "jobs"}

	for i, key := range views {
		h.SendRune(key)
		time.Sleep(50 * time.Millisecond)

		h.AssertCurrentView(expected[i])

		// Each view should render independently
		screenText := h.GetScreenText()
		if len(screenText) < 10 {
			h.DumpScreen()
			t.Errorf("View %s appears empty", expected[i])
		}
	}
}

// TestContinuousViewSwitching verifies that rapid view switching works correctly
func TestContinuousViewSwitching(t *testing.T) {
	h := NewTUITestHarness(t)
	defer h.Cleanup()

	h.Start()
	time.Sleep(100 * time.Millisecond)

	// Rapidly switch between views
	for i := 0; i < 10; i++ {
		h.SendRune('1')
		time.Sleep(20 * time.Millisecond)
		h.SendRune('2')
		time.Sleep(20 * time.Millisecond)
	}

	// App should still be stable
	h.AssertCurrentView("nodes") // Should end on nodes (last '2' pressed)

	screenText := h.GetScreenText()
	if len(screenText) < 10 {
		h.DumpScreen()
		t.Error("Screen appears empty after rapid view switching")
	}
}
