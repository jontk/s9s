package app

import (
	"fmt"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/jontk/s9s/internal/views"
)

// KeyHandler is a function that handles a keyboard event
type KeyHandler func(*S9s, *tcell.EventKey) *tcell.EventKey

// setupKeyboardShortcuts configures the global keyboard input capture
func (s *S9s) setupKeyboardShortcuts() {
	s.app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		// Handle command mode - pass through
		if s.cmdVisible {
			return event
		}

		// Check if a modal is open
		isModalOpen := s.pages.GetPageCount() > 1

		// Try to handle by key type
		switch event.Key() {
		case tcell.KeyRune:
			return s.handleRuneKey(event, isModalOpen)
		case tcell.KeyEsc:
			if s.cmdVisible {
				s.hideCommandLine()
				return nil
			}
		}

		// Try to handle by special key
		if handler, ok := s.globalKeyHandlers()[event.Key()]; ok {
			return handler(s, event)
		}

		// Pass to current view if not handled and no modal is open
		if !isModalOpen {
			if currentView, err := s.viewMgr.GetCurrentView(); err == nil {
				return currentView.OnKey(event)
			}
		}

		return event
	})
}

// handleRuneKey handles keyboard input for rune (character) keys
func (s *S9s) handleRuneKey(event *tcell.EventKey, isModalOpen bool) *tcell.EventKey {
	r := event.Rune()

	// Allow command mode in modals for emergency commands
	if isModalOpen && r == ':' {
		s.showCommandLine()
		return nil
	}

	// If modal is open, pass through (except for colon already handled)
	if isModalOpen {
		return event
	}

	// Handle global rune shortcuts
	handler, ok := s.globalRuneHandlers()[r]
	if ok {
		return handler(s, event)
	}

	return event
}

// globalKeyHandlers returns a map of special keys to their handlers
func (s *S9s) globalKeyHandlers() map[tcell.Key]KeyHandler {
	return map[tcell.Key]KeyHandler{
		tcell.KeyCtrlC:  s.handleCtrlC,
		tcell.KeyF1:     s.handleF1Help,
		tcell.KeyF2:     s.handleF2Alerts,
		tcell.KeyF3:     s.handleF3Preferences,
		tcell.KeyF4:     s.handleF4LayoutSwitcher,
		tcell.KeyF5:     s.handleF5Refresh,
		tcell.KeyF10:    s.handleF10Configuration,
		tcell.KeyTab:    s.handleTabNavigation,
		tcell.KeyBacktab: s.handleBacktabNavigation,
	}
}

// globalRuneHandlers returns a map of rune keys to their handlers
func (s *S9s) globalRuneHandlers() map[rune]KeyHandler {
	return map[rune]KeyHandler{
		':': s.handleColonCommand,
		'?': s.handleQuestionHelp,
		'1': s.makeViewSwitchHandler("jobs"),
		'2': s.makeViewSwitchHandler("nodes"),
		'3': s.makeViewSwitchHandler("partitions"),
		'4': s.makeViewSwitchHandler("reservations"),
		'5': s.makeViewSwitchHandler("qos"),
		'6': s.makeViewSwitchHandler("accounts"),
		'7': s.makeViewSwitchHandler("users"),
		'8': s.makeViewSwitchHandler("dashboard"),
		'9': s.makeViewSwitchHandler("health"),
		'0': s.makeViewSwitchHandler("performance"),
		'q': s.handleQuit,
		'Q': s.handleQuit,
	}
}

// Handler implementations
func (s *S9s) handleCtrlC(_ *S9s, _ *tcell.EventKey) *tcell.EventKey {
	_ = s.Stop()
	return nil
}

func (s *S9s) handleF1Help(_ *S9s, _ *tcell.EventKey) *tcell.EventKey {
	views.ShowHelpModal(s.pages)
	return nil
}

func (s *S9s) handleF2Alerts(_ *S9s, _ *tcell.EventKey) *tcell.EventKey {
	s.showAlertsModal()
	return nil
}

func (s *S9s) handleF3Preferences(_ *S9s, _ *tcell.EventKey) *tcell.EventKey {
	s.showPreferences()
	return nil
}

func (s *S9s) handleF4LayoutSwitcher(_ *S9s, _ *tcell.EventKey) *tcell.EventKey {
	s.showLayoutSwitcher()
	return nil
}

func (s *S9s) handleF5Refresh(_ *S9s, _ *tcell.EventKey) *tcell.EventKey {
	if err := s.viewMgr.RefreshCurrentView(); err != nil {
		s.statusBar.Error(fmt.Sprintf("Failed to refresh: %v", err))
	} else {
		s.statusBar.Success("Refreshed")
		// Restore hints after a short delay
		go func() {
			time.Sleep(3500 * time.Millisecond)
			if currentView, err := s.viewMgr.GetCurrentView(); err == nil {
				s.statusBar.SetHints(currentView.Hints())
			}
		}()
	}
	return nil
}

func (s *S9s) handleF10Configuration(_ *S9s, _ *tcell.EventKey) *tcell.EventKey {
	s.showConfiguration()
	return nil
}

func (s *S9s) handleTabNavigation(_ *S9s, _ *tcell.EventKey) *tcell.EventKey {
	_ = s.viewMgr.NextView()
	s.updateCurrentView()
	return nil
}

func (s *S9s) handleBacktabNavigation(_ *S9s, _ *tcell.EventKey) *tcell.EventKey {
	_ = s.viewMgr.PreviousView()
	s.updateCurrentView()
	return nil
}

func (s *S9s) handleColonCommand(_ *S9s, _ *tcell.EventKey) *tcell.EventKey {
	s.showCommandLine()
	return nil
}

func (s *S9s) handleQuestionHelp(_ *S9s, _ *tcell.EventKey) *tcell.EventKey {
	s.showHelp()
	return nil
}

func (s *S9s) handleQuit(_ *S9s, _ *tcell.EventKey) *tcell.EventKey {
	_ = s.Stop()
	return nil
}

// makeViewSwitchHandler creates a handler for switching to a specific view
func (s *S9s) makeViewSwitchHandler(viewName string) KeyHandler {
	return func(s *S9s, _ *tcell.EventKey) *tcell.EventKey {
		s.switchToView(viewName)
		return nil
	}
}
