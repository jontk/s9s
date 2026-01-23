package tui

import (
	"testing"
	"time"
)

// TestCompleteUserWorkflow simulates a realistic user session
func TestCompleteUserWorkflow(t *testing.T) {
	h := NewTUITestHarness(t)
	defer h.Cleanup()

	h.Start()
	time.Sleep(100 * time.Millisecond)

	// Workflow: User opens app, checks jobs, switches to nodes, opens help, then quits

	// 1. Start on jobs view
	h.AssertCurrentView("jobs")
	time.Sleep(100 * time.Millisecond)

	// 2. Navigate with arrow keys (simulating browsing)
	h.SendRune('j') // Down (vim style)
	time.Sleep(30 * time.Millisecond)
	h.SendRune('k') // Up
	time.Sleep(30 * time.Millisecond)

	// 3. Switch to nodes view to check cluster
	h.SendRune('2')
	time.Sleep(100 * time.Millisecond)
	h.AssertCurrentView("nodes")

	// 4. Open help to learn keyboard shortcuts
	h.SendRune('?')
	time.Sleep(100 * time.Millisecond)
	h.AssertModalOpen()

	// 5. Close help
	h.SendEsc()
	time.Sleep(50 * time.Millisecond)
	h.AssertModalClosed()

	// 6. Go back to jobs
	h.SendRune('1')
	time.Sleep(100 * time.Millisecond)
	h.AssertCurrentView("jobs")

	// Verify app is still stable after complete workflow
	screenText := h.GetScreenText()
	if len(screenText) < 10 {
		h.DumpScreen()
		t.Error("Screen appears empty after workflow")
	}
}

// TestPowerUserWorkflow simulates an experienced user with commands
func TestPowerUserWorkflow(t *testing.T) {
	h := NewTUITestHarness(t)
	defer h.Cleanup()

	h.Start()
	time.Sleep(100 * time.Millisecond)

	// Power user uses command mode for everything

	// 1. Check jobs with command
	h.SendRune(':')
	time.Sleep(20 * time.Millisecond)
	for _, ch := range "jobs" {
		h.SendRune(ch)
		time.Sleep(5 * time.Millisecond)
	}
	h.SendEnter()
	time.Sleep(100 * time.Millisecond)
	h.AssertCurrentView("jobs")

	// 2. Check nodes with command
	h.SendRune(':')
	time.Sleep(20 * time.Millisecond)
	for _, ch := range "nodes" {
		h.SendRune(ch)
		time.Sleep(5 * time.Millisecond)
	}
	h.SendEnter()
	time.Sleep(100 * time.Millisecond)
	h.AssertCurrentView("nodes")

	// 3. Refresh with command
	h.SendRune(':')
	time.Sleep(20 * time.Millisecond)
	for _, ch := range "refresh" {
		h.SendRune(ch)
		time.Sleep(5 * time.Millisecond)
	}
	h.SendEnter()
	time.Sleep(200 * time.Millisecond)

	// Should still be on nodes view
	h.AssertCurrentView("nodes")
}

// TestMultiViewComparisonWorkflow simulates comparing data across views
func TestMultiViewComparisonWorkflow(t *testing.T) {
	h := NewTUITestHarness(t)
	defer h.Cleanup()

	h.Start()
	time.Sleep(100 * time.Millisecond)

	// User rapidly switches between views to compare data

	// Check all views in sequence
	views := []struct {
		key  rune
		name string
	}{
		{'1', "jobs"},
		{'2', "nodes"},
		{'3', "partitions"},
		{'4', "reservations"},
		{'5', "qos"},
		{'1', "jobs"}, // Back to jobs
	}

	for _, v := range views {
		h.SendRune(v.key)
		time.Sleep(80 * time.Millisecond)
		h.AssertCurrentView(v.name)

		// Verify view rendered
		screenText := h.GetScreenText()
		if len(screenText) < 10 {
			h.DumpScreen()
			t.Errorf("View %s appears empty", v.name)
		}
	}
}

// TestExploratoryWorkflow simulates user exploring the application
func TestExploratoryWorkflow(t *testing.T) {
	h := NewTUITestHarness(t)
	defer h.Cleanup()

	h.Start()
	time.Sleep(100 * time.Millisecond)

	// New user explores by trying different keys

	// 1. Try Tab navigation
	h.SendTab()
	time.Sleep(100 * time.Millisecond)
	currentView := h.GetCurrentView()
	if currentView == "" {
		t.Error("Tab didn't switch views")
	}

	// 2. Try number keys
	h.SendRune('1')
	time.Sleep(100 * time.Millisecond)
	h.AssertCurrentView("jobs")

	// 3. Try help
	h.SendRune('?')
	time.Sleep(100 * time.Millisecond)
	h.AssertModalOpen()
	h.SendEsc()
	time.Sleep(50 * time.Millisecond)

	// 4. Try F keys
	h.SendKey(281, 0, 0) // F1 equivalent
	time.Sleep(100 * time.Millisecond)

	// Close any modal that opened
	if h.IsModalOpen() {
		h.SendEsc()
		time.Sleep(50 * time.Millisecond)
	}

	// Should still be functional
	h.AssertModalClosed()
}

// TestTabBasedNavigationWorkflow simulates user preferring Tab key
func TestTabBasedNavigationWorkflow(t *testing.T) {
	h := NewTUITestHarness(t)
	defer h.Cleanup()

	h.Start()
	time.Sleep(100 * time.Millisecond)

	initialView := h.GetCurrentView()

	// Navigate forward through all views using Tab
	visitedViews := []string{initialView}
	for i := 0; i < 5; i++ {
		h.SendTab()
		time.Sleep(80 * time.Millisecond)

		currentView := h.GetCurrentView()
		visitedViews = append(visitedViews, currentView)

		// Verify view is functional
		screenText := h.GetScreenText()
		if len(screenText) < 10 {
			h.DumpScreen()
			t.Errorf("View %s appears empty", currentView)
		}
	}

	// Should have visited multiple different views
	uniqueViews := make(map[string]bool)
	for _, v := range visitedViews {
		uniqueViews[v] = true
	}

	if len(uniqueViews) < 2 {
		t.Errorf("Tab navigation didn't visit multiple views: %v", visitedViews)
	}
}

// TestKeyboardShortcutDiscovery simulates learning shortcuts
func TestKeyboardShortcutDiscovery(t *testing.T) {
	h := NewTUITestHarness(t)
	defer h.Cleanup()

	h.Start()
	time.Sleep(100 * time.Millisecond)

	// User tries various keys to discover shortcuts
	keyTests := []struct {
		name string
		key  rune
	}{
		{"help", '?'},
		{"jobs", '1'},
		{"nodes", '2'},
		{"command", ':'},
	}

	for _, kt := range keyTests {
		t.Run(kt.name, func(t *testing.T) {
			h.SendRune(kt.key)
			time.Sleep(80 * time.Millisecond)

			// Should do something and not crash
			screenText := h.GetScreenText()
			if len(screenText) < 10 {
				h.DumpScreen()
				t.Errorf("Screen appears empty after key %c", kt.key)
			}

			// Clean up (close modals, command line, etc.)
			if h.IsModalOpen() {
				h.SendEsc()
				time.Sleep(50 * time.Millisecond)
			}
			if h.IsCmdVisible() {
				h.SendEsc()
				time.Sleep(50 * time.Millisecond)
			}
		})
	}
}

// TestMonitoringWorkflow simulates user monitoring job status
func TestMonitoringWorkflow(t *testing.T) {
	h := NewTUITestHarness(t)
	defer h.Cleanup()

	h.Start()
	time.Sleep(100 * time.Millisecond)

	// User repeatedly checks jobs and refreshes

	for i := 0; i < 3; i++ {
		// Check jobs
		h.SendRune('1')
		time.Sleep(100 * time.Millisecond)
		h.AssertCurrentView("jobs")

		// Refresh to see updates
		h.SendKey(281, 0, 0) // F5 equivalent
		time.Sleep(150 * time.Millisecond)

		// Still on jobs view
		h.AssertCurrentView("jobs")

		// Verify data displayed
		screenText := h.GetScreenText()
		if len(screenText) < 10 {
			h.DumpScreen()
			t.Error("Screen appears empty during monitoring")
		}
	}
}

// TestQuickCheckWorkflow simulates user doing quick status check
func TestQuickCheckWorkflow(t *testing.T) {
	h := NewTUITestHarness(t)
	defer h.Cleanup()

	h.Start()
	time.Sleep(100 * time.Millisecond)

	// User quickly checks each view and exits

	// Quick scan of all main views
	h.SendRune('1') // Jobs
	time.Sleep(50 * time.Millisecond)

	h.SendRune('2') // Nodes
	time.Sleep(50 * time.Millisecond)

	h.SendRune('3') // Partitions
	time.Sleep(50 * time.Millisecond)

	// All views should have rendered without crashing
	h.AssertCurrentView("partitions")
	screenText := h.GetScreenText()
	if len(screenText) < 10 {
		h.DumpScreen()
		t.Error("Screen appears empty after quick check")
	}
}

// TestConfigurationChangeWorkflow simulates changing settings
func TestConfigurationChangeWorkflow(t *testing.T) {
	h := NewTUITestHarness(t)
	defer h.Cleanup()

	h.Start()
	time.Sleep(100 * time.Millisecond)

	// User opens configuration modal
	h.SendKey(288, 0, 0) // F10
	time.Sleep(150 * time.Millisecond)

	if h.IsModalOpen() {
		// Configuration modal opened successfully
		// User looks around then closes without changing
		time.Sleep(100 * time.Millisecond)
		h.SendEsc()
		time.Sleep(50 * time.Millisecond)

		// Should be back to normal view
		h.AssertModalClosed()
		h.AssertCurrentView("jobs")
	}

	// App should be stable after opening/closing config
	screenText := h.GetScreenText()
	if len(screenText) < 10 {
		h.DumpScreen()
		t.Error("Screen appears empty after config workflow")
	}
}

// TestErrorRecoveryWorkflow simulates user recovering from mistakes
func TestErrorRecoveryWorkflow(t *testing.T) {
	h := NewTUITestHarness(t)
	defer h.Cleanup()

	h.Start()
	time.Sleep(100 * time.Millisecond)

	// User makes several mistakes but recovers

	// 1. Starts typing a command but cancels
	h.SendRune(':')
	time.Sleep(20 * time.Millisecond)
	h.SendRune('x')
	h.SendRune('y')
	h.SendRune('z')
	h.SendEsc() // Cancel
	time.Sleep(50 * time.Millisecond)

	// 2. Opens modal accidentally and closes
	h.SendRune('?')
	time.Sleep(50 * time.Millisecond)
	h.SendEsc()
	time.Sleep(50 * time.Millisecond)

	// 3. Types invalid command
	h.SendRune(':')
	time.Sleep(20 * time.Millisecond)
	for _, ch := range "invalid" {
		h.SendRune(ch)
		time.Sleep(5 * time.Millisecond)
	}
	h.SendEnter()
	time.Sleep(50 * time.Millisecond)

	// Should recover and be functional
	h.AssertCurrentView("jobs")
	h.AssertModalClosed()

	screenText := h.GetScreenText()
	if len(screenText) < 10 {
		h.DumpScreen()
		t.Error("Screen appears empty after error recovery")
	}
}
