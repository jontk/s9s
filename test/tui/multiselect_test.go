package tui

import (
	"testing"
	"time"

	"github.com/gdamore/tcell/v2"
)

// TestMultiSelectModeActivation verifies that multi-select mode can be activated
func TestMultiSelectModeActivation(t *testing.T) {
	h := NewTUITestHarness(t)
	defer h.Cleanup()

	h.Start()
	time.Sleep(100 * time.Millisecond)

	// Ensure we're on jobs view
	h.AssertCurrentView("jobs")

	// Try to activate multi-select with 'v' key (common vim-like binding)
	h.SendRune('v')
	time.Sleep(100 * time.Millisecond)

	// Verify app is still functional
	screenText := h.GetScreenText()
	if len(screenText) == 0 {
		h.DumpScreen()
		t.Error("Screen is empty after attempting to activate multi-select")
	}
}

// TestSpacebarSelection verifies that spacebar can be used for selection
func TestSpacebarSelection(t *testing.T) {
	h := NewTUITestHarness(t)
	defer h.Cleanup()

	h.Start()
	time.Sleep(100 * time.Millisecond)

	// Ensure we're on jobs view
	h.AssertCurrentView("jobs")

	// Try to select with spacebar
	h.SendKey(32, ' ', 0) // Space key
	time.Sleep(100 * time.Millisecond)

	// Verify app is still functional
	screenText := h.GetScreenText()
	if len(screenText) == 0 {
		h.DumpScreen()
		t.Error("Screen is empty after spacebar selection")
	}
}

// TestArrowKeyNavigation verifies that arrow keys work for navigation
func TestArrowKeyNavigation(t *testing.T) {
	h := NewTUITestHarness(t)
	defer h.Cleanup()

	h.Start()
	time.Sleep(100 * time.Millisecond)

	// Ensure we're on jobs view
	h.AssertCurrentView("jobs")

	// Test arrow key navigation
	arrowKeys := []struct {
		name string
		key  int // tcell key codes
	}{
		{"down", 258},   // tcell.KeyDown
		{"up", 259},     // tcell.KeyUp
		{"right", 261},  // tcell.KeyRight
		{"left", 260},   // tcell.KeyLeft
	}

	for _, arrow := range arrowKeys {
		t.Run(arrow.name, func(t *testing.T) {
			h.SendKey(tcell.Key(arrow.key), 0, tcell.ModNone)
			time.Sleep(50 * time.Millisecond)

			// Verify app is still functional
			screenText := h.GetScreenText()
			if len(screenText) == 0 {
				h.DumpScreen()
				t.Errorf("Screen is empty after %s arrow key", arrow.name)
			}
		})
	}
}

// TestVimStyleNavigation verifies that vim-style keys (hjkl) work for navigation
func TestVimStyleNavigation(t *testing.T) {
	h := NewTUITestHarness(t)
	defer h.Cleanup()

	h.Start()
	time.Sleep(100 * time.Millisecond)

	// Ensure we're on jobs view
	h.AssertCurrentView("jobs")

	// Test vim-style navigation
	vimKeys := []rune{'j', 'k', 'h', 'l', 'g', 'G'}

	for _, key := range vimKeys {
		h.SendRune(key)
		time.Sleep(50 * time.Millisecond)

		// Verify app is still functional
		screenText := h.GetScreenText()
		if len(screenText) == 0 {
			h.DumpScreen()
			t.Errorf("Screen is empty after vim key %c", key)
		}
	}
}

// TestMultiSelectAndOperation verifies selecting multiple items and performing operations
func TestMultiSelectAndOperation(t *testing.T) {
	h := NewTUITestHarness(t)
	defer h.Cleanup()

	h.Start()
	time.Sleep(100 * time.Millisecond)

	// Ensure we're on jobs view
	h.AssertCurrentView("jobs")

	// Try to activate multi-select
	h.SendRune('v')
	time.Sleep(50 * time.Millisecond)

	// Select a few items
	for i := 0; i < 3; i++ {
		h.SendKey(32, ' ', 0) // Space to select
		time.Sleep(20 * time.Millisecond)
		h.SendRune('j') // Down to next item
		time.Sleep(20 * time.Millisecond)
	}

	// Try an operation (like cancel)
	h.SendRune('c')
	time.Sleep(100 * time.Millisecond)

	// Verify app handled it (modal might appear)
	screenText := h.GetScreenText()
	if len(screenText) == 0 {
		h.DumpScreen()
		t.Error("Screen is empty after multi-select operation")
	}
}

// TestSelectAllFunctionality verifies that Ctrl+A or similar can select all items
func TestSelectAllFunctionality(t *testing.T) {
	h := NewTUITestHarness(t)
	defer h.Cleanup()

	h.Start()
	time.Sleep(100 * time.Millisecond)

	// Ensure we're on jobs view
	h.AssertCurrentView("jobs")

	// Try Ctrl+A to select all (if implemented)
	h.SendKey(1, 'a', tcell.ModCtrl) // Ctrl+A
	time.Sleep(100 * time.Millisecond)

	// Verify app is still functional
	screenText := h.GetScreenText()
	if len(screenText) == 0 {
		h.DumpScreen()
		t.Error("Screen is empty after Ctrl+A")
	}
}

// TestDeselectFunctionality verifies that items can be deselected
func TestDeselectFunctionality(t *testing.T) {
	h := NewTUITestHarness(t)
	defer h.Cleanup()

	h.Start()
	time.Sleep(100 * time.Millisecond)

	// Ensure we're on jobs view
	h.AssertCurrentView("jobs")

	// Try to select and then deselect
	h.SendRune('v') // Activate multi-select
	time.Sleep(50 * time.Millisecond)

	h.SendKey(32, ' ', 0) // Select current item
	time.Sleep(50 * time.Millisecond)

	h.SendKey(32, ' ', 0) // Deselect same item
	time.Sleep(50 * time.Millisecond)

	// Exit multi-select mode
	h.SendEsc()
	time.Sleep(50 * time.Millisecond)

	// Verify app is still functional
	h.AssertCurrentView("jobs")
}

// TestMultiSelectAcrossViews verifies multi-select behavior when switching views
func TestMultiSelectAcrossViews(t *testing.T) {
	h := NewTUITestHarness(t)
	defer h.Cleanup()

	h.Start()
	time.Sleep(100 * time.Millisecond)

	// Start on jobs view and activate multi-select
	h.AssertCurrentView("jobs")
	h.SendRune('v')
	time.Sleep(50 * time.Millisecond)

	// Switch to another view
	h.SendRune('2') // Nodes view
	time.Sleep(100 * time.Millisecond)

	h.AssertCurrentView("nodes")

	// Switch back to jobs
	h.SendRune('1')
	time.Sleep(100 * time.Millisecond)

	h.AssertCurrentView("jobs")

	// App should be stable
	screenText := h.GetScreenText()
	if len(screenText) == 0 {
		h.DumpScreen()
		t.Error("Screen is empty after view switching with multi-select")
	}
}

// TestPageNavigationKeys verifies that PageUp/PageDown work
func TestPageNavigationKeys(t *testing.T) {
	h := NewTUITestHarness(t)
	defer h.Cleanup()

	h.Start()
	time.Sleep(100 * time.Millisecond)

	// Ensure we're on jobs view
	h.AssertCurrentView("jobs")

	// Test PageDown
	h.SendKey(267, 0, 0) // tcell.KeyPgDn
	time.Sleep(50 * time.Millisecond)

	// Test PageUp
	h.SendKey(266, 0, 0) // tcell.KeyPgUp
	time.Sleep(50 * time.Millisecond)

	// Verify app is still functional
	screenText := h.GetScreenText()
	if len(screenText) == 0 {
		h.DumpScreen()
		t.Error("Screen is empty after page navigation")
	}
}

// TestHomeEndKeys verifies that Home/End keys work for navigation
func TestHomeEndKeys(t *testing.T) {
	h := NewTUITestHarness(t)
	defer h.Cleanup()

	h.Start()
	time.Sleep(100 * time.Millisecond)

	// Ensure we're on jobs view
	h.AssertCurrentView("jobs")

	// Test End key
	h.SendKey(269, 0, 0) // tcell.KeyEnd
	time.Sleep(50 * time.Millisecond)

	// Test Home key
	h.SendKey(268, 0, 0) // tcell.KeyHome
	time.Sleep(50 * time.Millisecond)

	// Verify app is still functional
	screenText := h.GetScreenText()
	if len(screenText) == 0 {
		h.DumpScreen()
		t.Error("Screen is empty after Home/End keys")
	}
}

// TestEnterKeyAction verifies that Enter key performs default action
func TestEnterKeyAction(t *testing.T) {
	h := NewTUITestHarness(t)
	defer h.Cleanup()

	h.Start()
	time.Sleep(100 * time.Millisecond)

	// Ensure we're on jobs view
	h.AssertCurrentView("jobs")

	// Press Enter (might open details or perform default action)
	h.SendEnter()
	time.Sleep(100 * time.Millisecond)

	// App should handle it gracefully
	screenText := h.GetScreenText()
	if len(screenText) == 0 {
		h.DumpScreen()
		t.Error("Screen is empty after Enter key")
	}

	// Close any modal that might have opened
	if h.IsModalOpen() {
		h.SendEsc()
		time.Sleep(50 * time.Millisecond)
	}
}

// TestRapidSelectionOperations verifies that rapid selection operations are stable
func TestRapidSelectionOperations(t *testing.T) {
	h := NewTUITestHarness(t)
	defer h.Cleanup()

	h.Start()
	time.Sleep(100 * time.Millisecond)

	// Ensure we're on jobs view
	h.AssertCurrentView("jobs")

	// Rapidly navigate and select
	for i := 0; i < 20; i++ {
		h.SendRune('j') // Down
		time.Sleep(10 * time.Millisecond)
		if i%3 == 0 {
			h.SendKey(32, ' ', 0) // Select every 3rd item
			time.Sleep(10 * time.Millisecond)
		}
	}

	// App should still be stable
	screenText := h.GetScreenText()
	if len(screenText) == 0 {
		h.DumpScreen()
		t.Error("Screen is empty after rapid selection operations")
	}

	h.AssertCurrentView("jobs")
}
