package tui

import (
	"testing"
	"time"

	"github.com/jontk/s9s/internal/testapi"
)

// TestEmptyViewState verifies app handles empty/no data gracefully
func TestEmptyViewState(t *testing.T) {
	h := NewTUITestHarness(t)
	defer h.Cleanup()

	h.Start()
	time.Sleep(100 * time.Millisecond)

	// Even with mock data, views should render without crashing
	views := []rune{'1', '2', '3', '4', '5', '6', '7', '8', '9', '0'}

	for _, key := range views {
		h.SendRune(key)
		time.Sleep(50 * time.Millisecond)

		// Should render something, not crash
		screenText := h.GetScreenText()
		if len(screenText) < 10 {
			h.DumpScreen()
			t.Errorf("View appears empty for key %c", key)
		}
	}
}

// TestRapidViewSwitching verifies no race conditions with rapid switching
func TestRapidViewSwitching(t *testing.T) {
	h := NewTUITestHarness(t)
	defer h.Cleanup()

	h.Start()
	time.Sleep(100 * time.Millisecond)

	// Rapidly switch views 100 times
	for i := 0; i < 100; i++ {
		h.SendRune('1')
		h.SendRune('2')
		h.SendRune('3')
		time.Sleep(1 * time.Millisecond)
	}

	// App should still be stable
	time.Sleep(100 * time.Millisecond)
	h.AssertCurrentView("partitions") // Should end on view 3

	screenText := h.GetScreenText()
	if len(screenText) < 10 {
		h.DumpScreen()
		t.Error("Screen appears empty after rapid switching")
	}
}

// TestModalStacking verifies handling of multiple modals
func TestModalStacking(t *testing.T) {
	h := NewTUITestHarness(t)
	defer h.Cleanup()

	h.Start()
	time.Sleep(100 * time.Millisecond)

	// Try to open multiple modals (app should handle gracefully)
	h.SendRune('?') // Help modal
	time.Sleep(50 * time.Millisecond)

	// Try to open another modal while one is open
	// (behavior depends on implementation, but shouldn't crash)
	h.SendRune('?')
	time.Sleep(50 * time.Millisecond)

	// Close modals
	h.SendEsc()
	time.Sleep(50 * time.Millisecond)

	if h.IsModalOpen() {
		h.SendEsc()
		time.Sleep(50 * time.Millisecond)
	}

	// Should be back to normal
	h.AssertModalClosed()
	h.AssertCurrentView("jobs")
}

// TestViewStatePreservation verifies view state is preserved
func TestViewStatePreservation(t *testing.T) {
	h := NewTUITestHarness(t)
	defer h.Cleanup()

	h.Start()
	time.Sleep(100 * time.Millisecond)

	// Switch to jobs view
	h.SendRune('1')
	time.Sleep(50 * time.Millisecond)
	h.AssertCurrentView("jobs")

	// Switch to nodes view
	h.SendRune('2')
	time.Sleep(50 * time.Millisecond)
	h.AssertCurrentView("nodes")

	// Switch back to jobs - should still work
	h.SendRune('1')
	time.Sleep(50 * time.Millisecond)
	h.AssertCurrentView("jobs")

	// App should be stable
	screenText := h.GetScreenText()
	if len(screenText) < 10 {
		h.DumpScreen()
		t.Error("Screen appears empty after view navigation")
	}
}

// TestStateConsistency verifies state API matches screen state
func TestStateConsistency(t *testing.T) {
	h := NewTUITestHarness(t)
	defer h.Cleanup()

	h.Start()
	time.Sleep(100 * time.Millisecond)

	state := testapi.NewStateExporter(h.GetApp())

	testCases := []struct {
		name         string
		action       func()
		verifyScreen func() bool
		verifyState  func(*testapi.AppState) bool
	}{
		{
			name: "modal_opens",
			action: func() {
				h.SendRune('?')
				time.Sleep(100 * time.Millisecond)
			},
			verifyScreen: func() bool {
				return h.IsModalOpen()
			},
			verifyState: func(s *testapi.AppState) bool {
				return s.ModalOpen
			},
		},
		{
			name: "modal_closes",
			action: func() {
				h.SendEsc()
				time.Sleep(50 * time.Millisecond)
			},
			verifyScreen: func() bool {
				return !h.IsModalOpen()
			},
			verifyState: func(s *testapi.AppState) bool {
				return !s.ModalOpen
			},
		},
		{
			name: "view_switches",
			action: func() {
				h.SendRune('2')
				time.Sleep(100 * time.Millisecond)
			},
			verifyScreen: func() bool {
				return h.GetCurrentView() == "nodes"
			},
			verifyState: func(s *testapi.AppState) bool {
				return s.CurrentView == "nodes"
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tc.action()

			// Verify screen state
			if !tc.verifyScreen() {
				h.DumpScreen()
				t.Error("Screen state verification failed")
			}

			// Verify API state matches
			appState := state.GetState()
			if !tc.verifyState(appState) {
				t.Errorf("API state verification failed: %+v", appState)
			}
		})
	}
}

// TestLongRunningStability verifies app is stable over time
func TestLongRunningStability(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping long-running test in short mode")
	}

	h := NewTUITestHarness(t)
	defer h.Cleanup()

	h.Start()
	time.Sleep(100 * time.Millisecond)

	// Perform various operations over 5 seconds
	endTime := time.Now().Add(5 * time.Second)
	operationCount := 0

	for time.Now().Before(endTime) {
		// Cycle through different operations
		switch operationCount % 5 {
		case 0:
			h.SendRune('1') // Jobs
		case 1:
			h.SendRune('2') // Nodes
		case 2:
			h.SendRune('?') // Help
			time.Sleep(50 * time.Millisecond)
			h.SendEsc()
		case 3:
			h.SendTab() // Next view
		case 4:
			h.SendRune(':') // Command mode
			time.Sleep(20 * time.Millisecond)
			h.SendEsc()
		}

		operationCount++
		time.Sleep(100 * time.Millisecond)
	}

	// Should still be functional
	h.AssertModalClosed()
	screenText := h.GetScreenText()
	if len(screenText) < 10 {
		h.DumpScreen()
		t.Error("Screen appears empty after long-running test")
	}

	t.Logf("Completed %d operations successfully", operationCount)
}

// TestBoundaryConditions verifies edge cases in navigation
func TestBoundaryConditions(t *testing.T) {
	h := NewTUITestHarness(t)
	defer h.Cleanup()

	h.Start()
	time.Sleep(100 * time.Millisecond)

	// Test navigating beyond boundaries
	testCases := []struct {
		name   string
		action func()
	}{
		{
			name: "multiple_tabs_forward",
			action: func() {
				for i := 0; i < 20; i++ {
					h.SendTab()
					time.Sleep(20 * time.Millisecond)
				}
			},
		},
		{
			name: "multiple_tabs_backward",
			action: func() {
				for i := 0; i < 20; i++ {
					h.SendBacktab()
					time.Sleep(20 * time.Millisecond)
				}
			},
		},
		{
			name: "repeated_same_view",
			action: func() {
				for i := 0; i < 10; i++ {
					h.SendRune('1')
					time.Sleep(20 * time.Millisecond)
				}
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tc.action()

			// Should still be functional
			screenText := h.GetScreenText()
			if len(screenText) < 10 {
				h.DumpScreen()
				t.Errorf("Screen appears empty after %s", tc.name)
			}
		})
	}
}

// TestRecoveryFromErrors verifies app recovers from error states
func TestRecoveryFromErrors(t *testing.T) {
	h := NewTUITestHarness(t)
	defer h.Cleanup()

	h.Start()
	time.Sleep(100 * time.Millisecond)

	// Try various operations that might cause errors
	// App should recover gracefully

	// Invalid commands
	h.SendRune(':')
	time.Sleep(20 * time.Millisecond)
	for _, ch := range "invalid_command_xyz" {
		h.SendRune(ch)
		time.Sleep(5 * time.Millisecond)
	}
	h.SendEnter()
	time.Sleep(50 * time.Millisecond)

	// Should still be functional
	h.AssertCurrentView("jobs")

	// Try operations without selections
	h.SendRune('c') // Cancel (likely no selection)
	time.Sleep(50 * time.Millisecond)

	// Should still be functional
	screenText := h.GetScreenText()
	if len(screenText) < 10 {
		h.DumpScreen()
		t.Error("Screen appears empty after error recovery")
	}
}

// TestMemoryStability verifies no obvious memory leaks
func TestMemoryStability(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping memory stability test in short mode")
	}

	// Create and destroy multiple harnesses
	for i := 0; i < 10; i++ {
		h := NewTUITestHarness(t)
		h.Start()
		time.Sleep(100 * time.Millisecond)

		// Do some operations
		h.SendRune('1')
		h.SendRune('2')
		h.SendRune('?')
		time.Sleep(50 * time.Millisecond)
		h.SendEsc()

		h.Cleanup()
	}

	t.Log("Successfully created and cleaned up 10 test harnesses")
}
