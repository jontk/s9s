package app

import (
	"fmt"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/jontk/s9s/internal/views"
	"github.com/rivo/tview"
)

// KeyHandler is a function that handles a keyboard event
type KeyHandler func(*S9s, *tcell.EventKey) *tcell.EventKey

// setupKeyboardShortcuts configures the global keyboard input capture
func (s *S9s) setupKeyboardShortcuts() {
	s.app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		// Handle command mode - pass through to allow command line input
		if s.cmdVisible {
			return event
		}

		// Check if a modal is open
		isModalOpen := s.pages.GetPageCount() > 1

		// If a modal is open, let it handle its own keys.
		// Pages that manage their own Escape (like config) handle
		// closing internally via their input capture.
		if isModalOpen {
			return event
		}

		// Try to handle by key type - rune keys go to global rune handlers first
		if event.Key() == tcell.KeyRune {
			result := s.handleRuneKey(event, isModalOpen)
			// If handler consumed the event (returned nil), we're done
			if result == nil {
				return nil
			}
			// If handler returned the event unhandled, fall through to special key and view handlers
			event = result
		}

		if event.Key() == tcell.KeyEsc {
			if s.cmdVisible {
				s.hideCommandLine()
				return nil
			}
		}

		// Check if an input field has focus - if so, skip global special key handlers
		// (except Ctrl+C for safety). This allows filter fields to receive all input.
		inputHasFocus := s.hasInputFieldFocus()

		// Try to handle by special key (only when no modal is open and no input has focus)
		// Ctrl+C is always handled globally for safety
		if handler, ok := s.globalKeyHandlers()[event.Key()]; ok {
			// Always allow Ctrl+C, skip other global handlers when input has focus
			if event.Key() == tcell.KeyCtrlC || !inputHasFocus {
				result := handler(s, event)
				// If handler consumed the event (returned nil), we're done
				if result == nil {
					return nil
				}
				// If handler returned the event unhandled, pass to view
				event = result
			}
		}

		// Pass to current view
		if currentView, err := s.viewMgr.GetCurrentView(); err == nil {
			return currentView.OnKey(event)
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

	// Check if an input field has focus - if so, let the input handle all rune keys
	// This prevents global shortcuts (like 1,2,3 for view switching) from
	// interrupting filter input
	if s.hasInputFieldFocus() {
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
		tcell.KeyCtrlC:   s.handleCtrlC,
		tcell.KeyCtrlK:   s.handleClusterSwitch,
		tcell.KeyF1:      s.handleF1Help,
		tcell.KeyF2:      s.handleF2Alerts,
		tcell.KeyF5:      s.handleF5Refresh,
		tcell.KeyF6:      s.handleF6ToggleAutoRefresh,
		tcell.KeyF10:     s.handleF10Configuration,
		tcell.KeyTab:     s.handleTabNavigation,
		tcell.KeyBacktab: s.handleBacktabNavigation,
	}
}

// globalRuneHandlers returns a map of rune keys to their handlers
func (s *S9s) globalRuneHandlers() map[rune]KeyHandler {
	return map[rune]KeyHandler{
		':': s.handleColonCommand,
		'?': s.handleQuestionHelp,
		'h': s.handlePreviousView,
		'l': s.handleNextView,
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

func (s *S9s) handleClusterSwitch(_ *S9s, _ *tcell.EventKey) *tcell.EventKey {
	s.showClusterSwitcher()
	return nil
}

func (s *S9s) handleF1Help(_ *S9s, _ *tcell.EventKey) *tcell.EventKey {
	views.ShowFullHelpModal(s.pages, s.viewMgr)
	return nil
}

func (s *S9s) handleF2Alerts(_ *S9s, _ *tcell.EventKey) *tcell.EventKey {
	s.showAlertsModal()
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

func (s *S9s) handleF6ToggleAutoRefresh(_ *S9s, _ *tcell.EventKey) *tcell.EventKey {
	enabled := !s.autoRefresh.Load()
	s.autoRefresh.Store(enabled)
	if enabled {
		s.statusBar.Success("Auto-refresh enabled")
	} else {
		s.statusBar.Info("Auto-refresh paused")
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

func (s *S9s) handlePreviousView(_ *S9s, _ *tcell.EventKey) *tcell.EventKey {
	_ = s.viewMgr.PreviousView()
	s.updateCurrentView()
	return nil
}

func (s *S9s) handleNextView(_ *S9s, _ *tcell.EventKey) *tcell.EventKey {
	_ = s.viewMgr.NextView()
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

// hasInputFieldFocus checks if the currently focused primitive is an input field
// that should receive all character input (filter fields, text inputs, etc.)
func (s *S9s) hasInputFieldFocus() bool {
	focused := s.app.GetFocus()
	if focused == nil {
		return false
	}

	// Check for InputField (filter inputs, text inputs)
	if _, ok := focused.(*tview.InputField); ok {
		return true
	}

	// Check for TextArea (multi-line text inputs)
	if _, ok := focused.(*tview.TextArea); ok {
		return true
	}

	return false
}
