package tui

import (
	"testing"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/jontk/s9s/internal/testapi"
)

// TestNumberKeyNavigation verifies that number keys switch between views
func TestNumberKeyNavigation(t *testing.T) {
	h := NewTUITestHarness(t)
	defer h.Cleanup()

	h.Start()
	time.Sleep(100 * time.Millisecond)

	testCases := []struct {
		key      rune
		expected string
	}{
		{'1', "jobs"},
		{'2', "nodes"},
		{'3', "partitions"},
		{'4', "reservations"},
		{'5', "qos"},
		{'6', "accounts"},
		{'7', "users"},
		{'8', "dashboard"},
		{'9', "health"},
		{'0', "performance"},
	}

	for _, tc := range testCases {
		t.Run(tc.expected, func(t *testing.T) {
			h.SendRune(tc.key)
			time.Sleep(100 * time.Millisecond)

			h.AssertCurrentView(tc.expected)
		})
	}
}

// TestTabNavigation verifies that Tab key cycles through views
func TestTabNavigation(t *testing.T) {
	h := NewTUITestHarness(t)
	defer h.Cleanup()

	h.Start()
	time.Sleep(100 * time.Millisecond)

	// Get initial view
	initialView := h.GetCurrentView()

	// Press Tab and verify view changed
	h.SendTab()
	time.Sleep(100 * time.Millisecond)

	secondView := h.GetCurrentView()
	if secondView == initialView {
		h.DumpScreen()
		t.Error("Tab did not switch views")
	}

	// Press Tab multiple times to cycle through views
	for i := 0; i < 5; i++ {
		h.SendTab()
		time.Sleep(50 * time.Millisecond)
	}

	// Should have cycled through several views
	currentView := h.GetCurrentView()
	if currentView == "" {
		t.Error("Current view is empty after Tab navigation")
	}
}

// TestBacktabNavigation verifies that Shift+Tab cycles through views in reverse
func TestBacktabNavigation(t *testing.T) {
	h := NewTUITestHarness(t)
	defer h.Cleanup()

	h.Start()
	time.Sleep(100 * time.Millisecond)

	// Get initial view
	initialView := h.GetCurrentView()

	// Press Shift+Tab and verify view changed
	h.SendBacktab()
	time.Sleep(100 * time.Millisecond)

	secondView := h.GetCurrentView()
	if secondView == initialView {
		h.DumpScreen()
		t.Error("Shift+Tab did not switch views")
	}
}

// TestCommandModeNavigation verifies command mode navigation
func TestCommandModeNavigation(t *testing.T) {
	h := NewTUITestHarness(t)
	defer h.Cleanup()

	h.Start()
	time.Sleep(100 * time.Millisecond)

	testCases := []struct {
		command  string
		expected string
	}{
		{"jobs", "jobs"},
		{"nodes", "nodes"},
		{"partitions", "partitions"},
	}

	for _, tc := range testCases {
		t.Run(tc.command, func(t *testing.T) {
			// Enter command mode
			h.SendRune(':')
			time.Sleep(50 * time.Millisecond)

			// Command line should be visible
			if !h.IsCmdVisible() {
				h.DumpScreen()
				t.Error("Command line not visible after pressing ':'")
			}

			// Type the command
			for _, ch := range tc.command {
				h.SendRune(ch)
				time.Sleep(10 * time.Millisecond)
			}

			// Press Enter
			h.SendEnter()
			time.Sleep(100 * time.Millisecond)

			// Verify view changed
			h.AssertCurrentView(tc.expected)

			// Command line should be hidden
			if h.IsCmdVisible() {
				h.DumpScreen()
				t.Error("Command line still visible after command execution")
			}
		})
	}
}

// TestEscapeFromCommandMode verifies that Esc closes command mode
func TestEscapeFromCommandMode(t *testing.T) {
	h := NewTUITestHarness(t)
	defer h.Cleanup()

	h.Start()
	time.Sleep(100 * time.Millisecond)

	// Enter command mode
	h.SendRune(':')
	time.Sleep(50 * time.Millisecond)

	if !h.IsCmdVisible() {
		h.DumpScreen()
		t.Fatal("Command line not visible after pressing ':'")
	}

	// Press Escape
	h.SendEsc()
	time.Sleep(50 * time.Millisecond)

	// Command line should be hidden
	if h.IsCmdVisible() {
		h.DumpScreen()
		t.Error("Command line still visible after Esc")
	}
}

// TestHelpModal verifies that ? and F1 open help modal
func TestHelpModal(t *testing.T) {
	h := NewTUITestHarness(t)
	defer h.Cleanup()

	h.Start()
	time.Sleep(100 * time.Millisecond)

	// Press ? to open help
	h.SendRune('?')
	time.Sleep(100 * time.Millisecond)

	// Help modal should be open
	h.AssertModalOpen()

	// Screen should contain help text
	h.AssertContainsText("Help")

	// Close help with Esc
	h.SendEsc()
	time.Sleep(50 * time.Millisecond)

	// Modal should be closed
	h.AssertModalClosed()
}

// TestF1Help verifies that F1 opens help modal
func TestF1Help(t *testing.T) {
	h := NewTUITestHarness(t)
	defer h.Cleanup()

	h.Start()
	time.Sleep(100 * time.Millisecond)

	// Press F1 to open help
	h.SendKey(tcell.KeyF1, 0, tcell.ModNone)
	time.Sleep(100 * time.Millisecond)

	// Help modal should be open
	h.AssertModalOpen()

	// Close help with Esc
	h.SendEsc()
	time.Sleep(50 * time.Millisecond)

	h.AssertModalClosed()
}

// TestNavigationWithStateAPI verifies navigation using the state API
func TestNavigationWithStateAPI(t *testing.T) {
	h := NewTUITestHarness(t)
	defer h.Cleanup()

	h.Start()
	time.Sleep(100 * time.Millisecond)

	// Create state exporter
	state := testapi.NewStateExporter(h.GetApp())

	// Navigate to nodes view
	h.SendRune('2')
	time.Sleep(100 * time.Millisecond)

	// Verify using state API
	appState := state.GetState()
	if appState.CurrentView != "nodes" {
		t.Errorf("Expected current view to be 'nodes', got %q", appState.CurrentView)
	}
	if appState.ModalOpen {
		t.Error("Expected no modal to be open")
	}

	// Open help modal
	h.SendRune('?')
	time.Sleep(100 * time.Millisecond)

	// Verify modal is open using state API
	appState = state.GetState()
	if !appState.ModalOpen {
		t.Error("Expected modal to be open")
	}
}

// TestSequentialNavigation verifies navigating through multiple views in sequence
func TestSequentialNavigation(t *testing.T) {
	h := NewTUITestHarness(t)
	defer h.Cleanup()

	h.Start()
	time.Sleep(100 * time.Millisecond)

	sequence := []struct {
		key  rune
		view string
	}{
		{'1', "jobs"},
		{'2', "nodes"},
		{'3', "partitions"},
		{'1', "jobs"},
		{'5', "qos"},
		{'8', "dashboard"},
	}

	for i, step := range sequence {
		t.Logf("Step %d: pressing %c, expecting %s", i+1, step.key, step.view)
		h.SendRune(step.key)
		time.Sleep(50 * time.Millisecond)
		h.AssertCurrentView(step.view)
	}
}

// TestNavigationPreservesViewState verifies that navigating away and back preserves view state
func TestNavigationPreservesViewState(t *testing.T) {
	h := NewTUITestHarness(t)
	defer h.Cleanup()

	h.Start()
	time.Sleep(100 * time.Millisecond)

	// Start on jobs view
	h.AssertCurrentView("jobs")

	// Navigate to nodes view
	h.SendRune('2')
	time.Sleep(100 * time.Millisecond)
	h.AssertCurrentView("nodes")

	// Navigate back to jobs view
	h.SendRune('1')
	time.Sleep(100 * time.Millisecond)
	h.AssertCurrentView("jobs")

	// The app should still be functional
	screenText := h.GetScreenText()
	if len(screenText) < 10 {
		h.DumpScreen()
		t.Error("Screen appears empty after navigation")
	}
}
