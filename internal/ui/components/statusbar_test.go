package components

import (
	"fmt"
	"testing"
	"time"
)

const helpNudge = "  [gray]?:All shortcuts[white]"

func TestStatusBar_SetHints(t *testing.T) {
	statusBar := NewStatusBar()

	hints := []string{"F1 Help", "F2 Alerts", "Tab Switch"}
	statusBar.SetHints(hints)

	// Should display hints when no message is present
	text := statusBar.GetText(false)
	expected := "F1 Help  F2 Alerts  Tab Switch" + helpNudge
	if text != expected {
		t.Errorf("Expected hints '%s', got '%s'", expected, text)
	}
}

func TestStatusBar_MessageOverridesHints(t *testing.T) {
	statusBar := NewStatusBar()

	// Set hints first
	hints := []string{"F1 Help", "F2 Alerts"}
	statusBar.SetHints(hints)

	// Add a temporary message
	statusBar.Success("Operation completed")

	// Should show message, not hints
	text := statusBar.GetText(false)
	expected := "[green]✓ Operation completed[white]"
	if text != expected {
		t.Errorf("Expected success message '%s', got '%s'", expected, text)
	}
}

func TestStatusBar_HintsReturnAfterMessageExpires(t *testing.T) {
	statusBar := NewStatusBar()

	// Set hints
	hints := []string{"F1 Help", "F2 Alerts"}
	statusBar.SetHints(hints)

	// Add a very short message
	statusBar.SetMessage("Temporary", 1*time.Millisecond)

	// Wait for message to expire
	time.Sleep(10 * time.Millisecond)

	// Clear message manually (simulates the goroutine)
	statusBar.ClearMessage()

	// Should show hints again
	text := statusBar.GetText(false)
	expected := "F1 Help  F2 Alerts" + helpNudge
	if text != expected {
		t.Errorf("Expected hints to return after message expires, got '%s'", text)
	}
}

func TestStatusBar_NoConflictWithMultipleCalls(t *testing.T) {
	statusBar := NewStatusBar()

	// Simulate rapid view switching
	for i := 0; i < 10; i++ {
		hints := []string{fmt.Sprintf("View %d", i), "F1 Help"}
		statusBar.SetHints(hints)
	}

	// Should show the last set of hints
	text := statusBar.GetText(false)
	expected := "View 9  F1 Help" + helpNudge
	if text != expected {
		t.Errorf("Expected latest hints '%s', got '%s'", expected, text)
	}
}

func TestStatusBar_EmptyHintsShowNudgeOnly(t *testing.T) {
	statusBar := NewStatusBar()

	// Set hints then clear by showing empty view
	statusBar.SetHints([]string{"something"})

	// formatHints with empty should show just the nudge
	text := statusBar.formatHints(nil)
	expected := "[gray]?:All shortcuts[white]"
	if text != expected {
		t.Errorf("Expected nudge only, got '%s'", text)
	}
}
