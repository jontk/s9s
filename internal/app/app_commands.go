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
	switch command {
	case "q", "quit":
		_ = s.Stop()
	case "jobs", "j":
		s.switchToView("jobs")
	case "nodes", "n":
		s.switchToView("nodes")
	case "partitions", "p":
		s.switchToView("partitions")
	case "reservations":
		s.switchToView("reservations")
	case "qos":
		s.switchToView("qos")
	case "accounts":
		s.switchToView("accounts")
	case "users":
		s.switchToView("users")
	case "dashboard":
		s.switchToView("dashboard")
	case "health":
		s.switchToView("health")
	case "performance":
		s.switchToView("performance")
	case "refresh", "r":
		if currentView, err := s.viewMgr.GetCurrentView(); err == nil {
			go func() {
				if err := currentView.Refresh(); err != nil {
					s.statusBar.Error(fmt.Sprintf("Refresh failed: %v", err))
				} else {
					s.statusBar.Success("Refreshed")
					// Restore hints after success message expires
					time.Sleep(3500 * time.Millisecond) // Wait slightly longer than success message
					if cv, err := s.viewMgr.GetCurrentView(); err == nil {
						s.statusBar.SetHints(cv.Hints())
					}
				}
			}()
		}
	case "help", "h":
		s.showHelp()
	case "prefs", "preferences":
		s.showPreferences()
	case "layout", "layouts":
		s.showLayoutSwitcher()
	case "config", "configuration", "settings":
		s.showConfiguration()
	default:
		s.statusBar.Error(fmt.Sprintf("Unknown command: %s", command))
	}
}
