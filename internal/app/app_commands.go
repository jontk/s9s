package app

import (
	"fmt"
	"time"

	"github.com/gdamore/tcell/v2"
)

// showCommandLine shows the command input field
func (s *S9s) showCommandLine() {
	s.cmdVisible = true
	s.cmdLine.SetText("")
	s.mainLayout.ResizeItem(s.cmdLine, 1, 0)
	s.app.SetFocus(s.cmdLine)
}

// hideCommandLine hides the command input field
func (s *S9s) hideCommandLine() {
	s.cmdVisible = false
	s.mainLayout.ResizeItem(s.cmdLine, 0, 0)

	// Return focus to current view
	if currentView, err := s.viewMgr.GetCurrentView(); err == nil {
		s.app.SetFocus(currentView.Render())
	}
}

// onCommandDone handles command line completion
func (s *S9s) onCommandDone(key tcell.Key) {
	defer s.hideCommandLine()

	if key != tcell.KeyEnter {
		return
	}

	command := s.cmdLine.GetText()
	if command == "" {
		return
	}

	s.executeCommand(command)
}

// executeCommand executes a command
func (s *S9s) executeCommand(command string) {
	if handler, ok := s.commandHandlers()[command]; ok {
		handler()
		return
	}
	s.statusBar.Error(fmt.Sprintf("Unknown command: %s", command))
}

// commandHandlers returns a map of command strings to handler functions
func (s *S9s) commandHandlers() map[string]func() {
	return map[string]func(){
		"q":             func() { _ = s.Stop() },
		"quit":          func() { _ = s.Stop() },
		"jobs":          func() { s.switchToView("jobs") },
		"j":             func() { s.switchToView("jobs") },
		"nodes":         func() { s.switchToView("nodes") },
		"n":             func() { s.switchToView("nodes") },
		"partitions":    func() { s.switchToView("partitions") },
		"p":             func() { s.switchToView("partitions") },
		"reservations":  func() { s.switchToView("reservations") },
		"qos":           func() { s.switchToView("qos") },
		"accounts":      func() { s.switchToView("accounts") },
		"users":         func() { s.switchToView("users") },
		"dashboard":     func() { s.switchToView("dashboard") },
		"health":        func() { s.switchToView("health") },
		"performance":   func() { s.switchToView("performance") },
		"refresh":       s.handleRefreshCommand,
		"r":             s.handleRefreshCommand,
		"help":          func() { s.showHelp() },
		"h":             func() { s.showHelp() },
		"prefs":         func() { s.showPreferences() },
		"preferences":   func() { s.showPreferences() },
		"layout":        func() { s.showLayoutSwitcher() },
		"layouts":       func() { s.showLayoutSwitcher() },
		"config":        func() { s.showConfiguration() },
		"configuration": func() { s.showConfiguration() },
		"settings":      func() { s.showConfiguration() },
	}
}

// handleRefreshCommand handles the refresh command
func (s *S9s) handleRefreshCommand() {
	currentView, err := s.viewMgr.GetCurrentView()
	if err != nil {
		return
	}

	go func() {
		if err := currentView.Refresh(); err != nil {
			s.statusBar.Error(fmt.Sprintf("Refresh failed: %v", err))
		} else {
			s.statusBar.Success("Refreshed")
			// Restore hints after success message expires
			time.Sleep(3500 * time.Millisecond)
			if cv, err := s.viewMgr.GetCurrentView(); err == nil {
				s.statusBar.SetHints(cv.Hints())
			}
		}
	}()
}
