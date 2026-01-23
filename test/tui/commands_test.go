package tui

import (
	"testing"
	"time"
)

// TestCommandExecution verifies that various commands execute correctly
func TestCommandExecution(t *testing.T) {
	commands := []struct {
		name     string
		command  string
		expected string // Expected view or state after command
	}{
		{"jobs_short", "j", "jobs"},
		{"nodes_short", "n", "nodes"},
		{"partitions_short", "p", "partitions"},
		{"help", "help", ""},   // May show modal
		{"refresh", "refresh", "jobs"}, // Should stay on current view
	}

	for _, tc := range commands {
		t.Run(tc.name, func(t *testing.T) {
			h := NewTUITestHarness(t)
			defer h.Cleanup()

			h.Start()
			time.Sleep(100 * time.Millisecond)

			// Enter command mode
			h.SendRune(':')
			time.Sleep(50 * time.Millisecond)

			// Type command
			for _, ch := range tc.command {
				h.SendRune(ch)
				time.Sleep(10 * time.Millisecond)
			}

			// Execute
			h.SendEnter()
			time.Sleep(100 * time.Millisecond)

			// Verify expected state (if not quitting)
			if tc.expected != "" {
				h.AssertCurrentView(tc.expected)
			}

			// Close any modal that opened
			if h.IsModalOpen() {
				h.SendEsc()
				time.Sleep(50 * time.Millisecond)
			}
		})
	}
}

// TestInvalidCommands verifies that invalid commands are handled gracefully
func TestInvalidCommands(t *testing.T) {
	h := NewTUITestHarness(t)
	defer h.Cleanup()

	h.Start()
	time.Sleep(100 * time.Millisecond)

	invalidCommands := []string{
		"invalid",
		"foobar",
		"123abc",
		"!@#$",
	}

	for _, cmd := range invalidCommands {
		t.Run(cmd, func(t *testing.T) {
			// Enter command mode
			h.SendRune(':')
			time.Sleep(50 * time.Millisecond)

			// Type invalid command
			for _, ch := range cmd {
				h.SendRune(ch)
				time.Sleep(10 * time.Millisecond)
			}

			// Execute
			h.SendEnter()
			time.Sleep(100 * time.Millisecond)

			// App should still be functional and show error
			h.AssertCurrentView("jobs")
			screenText := h.GetScreenText()
			if len(screenText) < 10 {
				h.DumpScreen()
				t.Error("Screen appears empty after invalid command")
			}
		})
	}
}

// TestEmptyCommand verifies that empty command doesn't cause issues
func TestEmptyCommand(t *testing.T) {
	h := NewTUITestHarness(t)
	defer h.Cleanup()

	h.Start()
	time.Sleep(100 * time.Millisecond)

	// Enter command mode
	h.SendRune(':')
	time.Sleep(50 * time.Millisecond)

	// Press Enter without typing anything
	h.SendEnter()
	time.Sleep(50 * time.Millisecond)

	// Should close command line and stay on current view
	if h.IsCmdVisible() {
		t.Error("Command line still visible after empty command")
	}
	h.AssertCurrentView("jobs")
}

// TestCommandWithSpaces verifies commands with spaces work
func TestCommandWithSpaces(t *testing.T) {
	h := NewTUITestHarness(t)
	defer h.Cleanup()

	h.Start()
	time.Sleep(100 * time.Millisecond)

	// Commands with spaces should be handled gracefully
	// (even if they're invalid)
	h.SendRune(':')
	time.Sleep(50 * time.Millisecond)

	for _, ch := range "foo bar" {
		h.SendRune(ch)
		time.Sleep(10 * time.Millisecond)
	}

	h.SendEnter()
	time.Sleep(100 * time.Millisecond)

	// App should still be functional
	h.AssertCurrentView("jobs")
}

// TestRapidCommandExecution verifies rapid command execution is stable
func TestRapidCommandExecution(t *testing.T) {
	h := NewTUITestHarness(t)
	defer h.Cleanup()

	h.Start()
	time.Sleep(100 * time.Millisecond)

	// Execute multiple commands rapidly
	commands := []string{"jobs", "nodes", "partitions", "jobs"}

	for _, cmd := range commands {
		h.SendRune(':')
		time.Sleep(20 * time.Millisecond)

		for _, ch := range cmd {
			h.SendRune(ch)
			time.Sleep(5 * time.Millisecond)
		}

		h.SendEnter()
		time.Sleep(30 * time.Millisecond)
	}

	// Should end on jobs view and be stable
	h.AssertCurrentView("jobs")
	screenText := h.GetScreenText()
	if len(screenText) < 10 {
		h.DumpScreen()
		t.Error("Screen appears empty after rapid commands")
	}
}

// TestCommandCancellation verifies Esc cancels command entry
func TestCommandCancellation(t *testing.T) {
	h := NewTUITestHarness(t)
	defer h.Cleanup()

	h.Start()
	time.Sleep(100 * time.Millisecond)

	initialView := h.GetCurrentView()

	// Enter command mode
	h.SendRune(':')
	time.Sleep(50 * time.Millisecond)

	// Type partial command
	h.SendRune('q')
	h.SendRune('u')
	time.Sleep(20 * time.Millisecond)

	// Cancel with Esc
	h.SendEsc()
	time.Sleep(50 * time.Millisecond)

	// Should close command line without executing
	if h.IsCmdVisible() {
		t.Error("Command line still visible after Esc")
	}

	// Should stay on same view
	if h.GetCurrentView() != initialView {
		t.Errorf("View changed after canceling command: expected %s, got %s",
			initialView, h.GetCurrentView())
	}
}

// TestCommandHistory verifies command line behavior
func TestCommandLineBehavior(t *testing.T) {
	h := NewTUITestHarness(t)
	defer h.Cleanup()

	h.Start()
	time.Sleep(100 * time.Millisecond)

	testCases := []struct {
		name   string
		action func()
		verify func()
	}{
		{
			name: "command_line_shows",
			action: func() {
				h.SendRune(':')
				time.Sleep(50 * time.Millisecond)
			},
			verify: func() {
				if !h.IsCmdVisible() {
					t.Error("Command line not visible")
				}
			},
		},
		{
			name: "command_line_hides_after_enter",
			action: func() {
				h.SendRune(':')
				time.Sleep(50 * time.Millisecond)
				h.SendEnter()
				time.Sleep(50 * time.Millisecond)
			},
			verify: func() {
				if h.IsCmdVisible() {
					t.Error("Command line still visible after Enter")
				}
			},
		},
		{
			name: "command_line_hides_after_esc",
			action: func() {
				h.SendRune(':')
				time.Sleep(50 * time.Millisecond)
				h.SendEsc()
				time.Sleep(50 * time.Millisecond)
			},
			verify: func() {
				if h.IsCmdVisible() {
					t.Error("Command line still visible after Esc")
				}
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tc.action()
			tc.verify()
		})
	}
}
