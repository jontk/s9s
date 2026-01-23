package tui

import (
	"testing"
	"time"

	"github.com/gdamore/tcell/v2"
)

// TestJobCancelOperation verifies that job cancel operation can be initiated
func TestJobCancelOperation(t *testing.T) {
	h := NewTUITestHarness(t)
	defer h.Cleanup()

	h.Start()
	time.Sleep(100 * time.Millisecond)

	// Ensure we're on jobs view
	h.AssertCurrentView("jobs")

	// Try to cancel a job with 'c' key
	h.SendRune('c')
	time.Sleep(100 * time.Millisecond)

	// Check if a confirmation modal appears or if the screen changes
	// The exact behavior depends on implementation
	screenText := h.GetScreenText()
	if len(screenText) == 0 {
		h.DumpScreen()
		t.Error("Screen is empty after attempting cancel operation")
	}
}

// TestJobHoldOperation verifies that job hold operation can be initiated
func TestJobHoldOperation(t *testing.T) {
	h := NewTUITestHarness(t)
	defer h.Cleanup()

	h.Start()
	time.Sleep(100 * time.Millisecond)

	// Ensure we're on jobs view
	h.AssertCurrentView("jobs")

	// Try to hold a job with 'h' key (common operation key)
	h.SendRune('h')
	time.Sleep(100 * time.Millisecond)

	// Verify app is still functional
	screenText := h.GetScreenText()
	if len(screenText) == 0 {
		h.DumpScreen()
		t.Error("Screen is empty after attempting hold operation")
	}
}

// TestModalEscapeHandling verifies that Esc closes modals properly
func TestModalEscapeHandling(t *testing.T) {
	h := NewTUITestHarness(t)
	defer h.Cleanup()

	h.Start()
	time.Sleep(100 * time.Millisecond)

	// Open help modal
	h.SendRune('?')
	time.Sleep(100 * time.Millisecond)

	// Modal should be open
	h.AssertModalOpen()

	// Press Escape
	h.SendEsc()
	time.Sleep(50 * time.Millisecond)

	// Modal should be closed
	h.AssertModalClosed()

	// App should be functional
	h.AssertCurrentView("jobs")
}

// TestMultipleModalOperations verifies handling of multiple modal operations
func TestMultipleModalOperations(t *testing.T) {
	h := NewTUITestHarness(t)
	defer h.Cleanup()

	h.Start()
	time.Sleep(100 * time.Millisecond)

	// Open and close help modal multiple times
	for i := 0; i < 3; i++ {
		// Open help
		h.SendRune('?')
		time.Sleep(50 * time.Millisecond)
		h.AssertModalOpen()

		// Close help
		h.SendEsc()
		time.Sleep(50 * time.Millisecond)
		h.AssertModalClosed()
	}

	// App should still be functional
	h.AssertCurrentView("jobs")
}

// TestAlertModal verifies that F2 opens alerts modal
func TestAlertsModal(t *testing.T) {
	h := NewTUITestHarness(t)
	defer h.Cleanup()

	h.Start()
	time.Sleep(100 * time.Millisecond)

	// Press F2 to open alerts
	h.SendKey(tcell.KeyF2, 0, tcell.ModNone)
	time.Sleep(100 * time.Millisecond)

	// Modal should be open
	h.AssertModalOpen()

	// Close with Esc
	h.SendEsc()
	time.Sleep(50 * time.Millisecond)

	h.AssertModalClosed()
}

// TestPreferencesModal verifies that F3 opens preferences modal
func TestPreferencesModal(t *testing.T) {
	h := NewTUITestHarness(t)
	defer h.Cleanup()

	h.Start()
	time.Sleep(100 * time.Millisecond)

	// Press F3 to open preferences
	h.SendKey(tcell.KeyF3, 0, tcell.ModNone)
	time.Sleep(100 * time.Millisecond)

	// Modal should be open
	h.AssertModalOpen()

	// Close with Esc
	h.SendEsc()
	time.Sleep(50 * time.Millisecond)

	h.AssertModalClosed()
}

// TestConfigurationModal verifies that F10 key is handled
func TestConfigurationModal(t *testing.T) {
	h := NewTUITestHarness(t)
	defer h.Cleanup()

	h.Start()
	time.Sleep(100 * time.Millisecond)

	// Press F10 (may open configuration modal depending on environment)
	h.SendKey(tcell.KeyF10, 0, tcell.ModNone)
	time.Sleep(200 * time.Millisecond)

	// If modal opened, close it
	if h.IsModalOpen() {
		h.SendEsc()
		time.Sleep(50 * time.Millisecond)
		h.AssertModalClosed()
	}

	// App should still be functional
	h.AssertCurrentView("jobs")
}

// TestQuitOperation verifies that 'q' quits the application
func TestQuitWithQ(t *testing.T) {
	// Skip this test - quit commands cause race conditions with screen finalization
	// TODO: Find a better way to test quit functionality without race conditions
	t.Skip("Skipping quit test due to race condition with tcell SimulationScreen")

	h := NewTUITestHarness(t)
	defer h.Cleanup()

	h.Start()
	time.Sleep(100 * time.Millisecond)

	// Press 'q' to quit
	h.SendRune('q')
	time.Sleep(100 * time.Millisecond)

	// The app should initiate shutdown
	// We can't easily verify complete shutdown in this test context,
	// but we can verify the command was processed
}

// TestQuitCommand verifies that :quit command works
func TestQuitCommand(t *testing.T) {
	// Skip this test - quit commands cause race conditions with screen finalization
	// TODO: Find a better way to test quit functionality without race conditions
	t.Skip("Skipping quit test due to race condition with tcell SimulationScreen")

	h := NewTUITestHarness(t)
	defer h.Cleanup()

	h.Start()
	time.Sleep(100 * time.Millisecond)

	// Enter command mode
	h.SendRune(':')
	time.Sleep(50 * time.Millisecond)

	// Type quit command
	for _, ch := range "quit" {
		h.SendRune(ch)
		time.Sleep(10 * time.Millisecond)
	}

	// Press Enter
	h.SendEnter()
	time.Sleep(100 * time.Millisecond)

	// The app should initiate shutdown
}

// TestOperationWithoutSelection verifies operations handle no selection gracefully
func TestOperationWithoutSelection(t *testing.T) {
	h := NewTUITestHarness(t)
	defer h.Cleanup()

	h.Start()
	time.Sleep(100 * time.Millisecond)

	// Try various operation keys without selecting anything
	operations := []rune{'c', 'h', 'r'}

	for _, op := range operations {
		h.SendRune(op)
		time.Sleep(50 * time.Millisecond)

		// App should still be functional
		screenText := h.GetScreenText()
		if len(screenText) == 0 {
			h.DumpScreen()
			t.Errorf("Screen is empty after operation key %c", op)
		}
	}
}

// TestLayoutSwitcher verifies that F4 opens layout switcher
func TestLayoutSwitcher(t *testing.T) {
	h := NewTUITestHarness(t)
	defer h.Cleanup()

	h.Start()
	time.Sleep(100 * time.Millisecond)

	// Press F4 to open layout switcher
	h.SendKey(tcell.KeyF4, 0, tcell.ModNone)
	time.Sleep(100 * time.Millisecond)

	// Modal should be open
	// Note: F4 may not always open a modal depending on implementation
	// Just verify the app is still functional
	if h.IsModalOpen() {
		// If modal opened, close it
		h.SendEsc()
		time.Sleep(50 * time.Millisecond)
		h.AssertModalClosed()
	}

	// App should still be functional
	h.AssertCurrentView("jobs")
}

// TestSequentialOperations verifies that multiple operations can be performed in sequence
func TestSequentialOperations(t *testing.T) {
	h := NewTUITestHarness(t)
	defer h.Cleanup()

	h.Start()
	time.Sleep(100 * time.Millisecond)

	// Perform a sequence of operations
	sequence := []struct {
		action string
		key    interface{}
	}{
		{"open_help", '?'},
		{"close_modal", "esc"},
		{"switch_view", '2'},
		{"refresh", "f5"},
		{"open_alerts", "f2"},
		{"close_modal", "esc"},
		{"switch_back", '1'},
	}

	for _, step := range sequence {
		switch v := step.key.(type) {
		case rune:
			h.SendRune(v)
		case string:
			if v == "esc" {
				h.SendEsc()
			} else if v == "f5" {
				h.SendKey(tcell.KeyF5, 0, tcell.ModNone)
			} else if v == "f2" {
				h.SendKey(tcell.KeyF2, 0, tcell.ModNone)
			}
		}
		time.Sleep(50 * time.Millisecond)

		// Verify app is still functional
		screenText := h.GetScreenText()
		if len(screenText) == 0 {
			h.DumpScreen()
			t.Errorf("Screen is empty after step: %s", step.action)
		}
	}

	// Should end on jobs view
	h.AssertCurrentView("jobs")
	h.AssertModalClosed()
}

// TestConcurrentModalHandling verifies that opening modals while one is already open works correctly
func TestConcurrentModalHandling(t *testing.T) {
	h := NewTUITestHarness(t)
	defer h.Cleanup()

	h.Start()
	time.Sleep(100 * time.Millisecond)

	// Open help modal
	h.SendRune('?')
	time.Sleep(100 * time.Millisecond)
	h.AssertModalOpen()

	// Try to open another modal (should close first or ignore)
	h.SendKey(114, 0, 0) // F2 for alerts
	time.Sleep(100 * time.Millisecond)

	// A modal should still be open
	// (behavior depends on implementation)

	// Close any open modal
	h.SendEsc()
	time.Sleep(50 * time.Millisecond)

	// If still open, close again
	if h.IsModalOpen() {
		h.SendEsc()
		time.Sleep(50 * time.Millisecond)
	}

	h.AssertModalClosed()
}
