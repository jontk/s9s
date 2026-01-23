package tui

import (
	"testing"
	"time"
)

// TestInitialDisplay verifies that the application displays the initial view correctly
func TestInitialDisplay(t *testing.T) {
	h := NewTUITestHarness(t)
	defer h.Cleanup()

	h.Start()

	// Wait for initial rendering
	time.Sleep(100 * time.Millisecond)

	// Verify that the jobs view is displayed by default
	h.AssertCurrentView("jobs")

	// The screen should contain typical UI elements
	h.AssertContainsText("S9S")
}

// TestHeaderDisplay verifies the header is displayed
func TestHeaderDisplay(t *testing.T) {
	h := NewTUITestHarness(t)
	defer h.Cleanup()

	h.Start()
	time.Sleep(100 * time.Millisecond)

	// The header should be visible
	screenText := h.GetScreenText()
	if len(screenText) < 10 {
		t.Error("Screen appears empty")
		h.DumpScreen()
	}
}

// TestStatusBarDisplay verifies the status bar is displayed
func TestStatusBarDisplay(t *testing.T) {
	h := NewTUITestHarness(t)
	defer h.Cleanup()

	h.Start()
	time.Sleep(100 * time.Millisecond)

	// Get the bottom lines of the screen (status bar is at the bottom)
	lines := h.GetScreenLines()
	if len(lines) < 2 {
		t.Fatal("Not enough lines on screen")
	}

	// Status bar should contain hints
	// Note: The exact content depends on the current view
	lastLine := lines[len(lines)-2] // Second to last line (last is empty)
	if len(lastLine) == 0 {
		h.DumpScreen()
		t.Error("Status bar appears empty")
	}
}

// TestJobsViewDisplay verifies the jobs view displays correctly
func TestJobsViewDisplay(t *testing.T) {
	h := NewTUITestHarness(t)
	defer h.Cleanup()

	h.Start()
	time.Sleep(100 * time.Millisecond)

	// Should be on jobs view by default
	h.AssertCurrentView("jobs")

	// The mock client should return some jobs
	// Check for table headers (common job view elements)
	if !h.ContainsTextCaseInsensitive("job") {
		h.DumpScreen()
		t.Error("Jobs view does not contain expected content")
	}
}

// TestNodesViewDisplay verifies the nodes view displays correctly
func TestNodesViewDisplay(t *testing.T) {
	h := NewTUITestHarness(t)
	defer h.Cleanup()

	h.Start()
	time.Sleep(100 * time.Millisecond)

	// Switch to nodes view
	h.SendRune('2')
	time.Sleep(100 * time.Millisecond)

	// Verify we're on the nodes view
	h.AssertCurrentView("nodes")

	// Check for nodes view content
	if !h.ContainsTextCaseInsensitive("node") {
		h.DumpScreen()
		t.Error("Nodes view does not contain expected content")
	}
}

// TestPartitionsViewDisplay verifies the partitions view displays correctly
func TestPartitionsViewDisplay(t *testing.T) {
	h := NewTUITestHarness(t)
	defer h.Cleanup()

	h.Start()
	time.Sleep(100 * time.Millisecond)

	// Switch to partitions view
	h.SendRune('3')
	time.Sleep(100 * time.Millisecond)

	// Verify we're on the partitions view
	h.AssertCurrentView("partitions")

	// Check for partitions view content
	if !h.ContainsTextCaseInsensitive("partition") {
		h.DumpScreen()
		t.Error("Partitions view does not contain expected content")
	}
}

// TestScreenSizeHandling verifies the app handles different screen sizes
func TestScreenSizeHandling(t *testing.T) {
	sizes := []struct {
		width  int
		height int
		name   string
	}{
		{80, 24, "small"},
		{120, 40, "medium"},
		{160, 50, "large"},
	}

	for _, size := range sizes {
		t.Run(size.name, func(t *testing.T) {
			h := NewTUITestHarnessWithSize(t, size.width, size.height)
			defer h.Cleanup()

			h.Start()
			time.Sleep(100 * time.Millisecond)

			// Should render without crashing
			h.AssertCurrentView("jobs")

			// Screen should not be blank
			screenText := h.GetScreenText()
			if len(screenText) < 10 {
				t.Errorf("Screen appears empty for size %dx%d", size.width, size.height)
				h.DumpScreen()
			}
		})
	}
}

// TestMultipleViewsRendering tests that all views can be rendered
func TestMultipleViewsRendering(t *testing.T) {
	viewKeys := []struct {
		key  rune
		name string
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

	h := NewTUITestHarness(t)
	defer h.Cleanup()

	h.Start()
	time.Sleep(100 * time.Millisecond)

	for _, view := range viewKeys {
		t.Run(view.name, func(t *testing.T) {
			h.SendRune(view.key)
			time.Sleep(100 * time.Millisecond)

			h.AssertCurrentView(view.name)

			// Screen should not be blank
			screenText := h.GetScreenText()
			if len(screenText) < 10 {
				t.Errorf("View %s appears empty", view.name)
				h.DumpScreen()
			}
		})
	}
}
