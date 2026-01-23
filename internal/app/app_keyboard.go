package app

import (
	"fmt"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/jontk/s9s/internal/views"
)

func (s *S9s) setupKeyboardShortcuts() {
	s.app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		// Handle command mode
		if s.cmdVisible {
			return event // Let command line handle it
		}

		// Check if a modal is open by checking if there are multiple pages
		isModalOpen := s.pages.GetPageCount() > 1

		// Global shortcuts
		switch event.Key() {
		case tcell.KeyCtrlC:
			_ = s.Stop()
			return nil
		case tcell.KeyF1:
			// Show help modal
			views.ShowHelpModal(s.pages)
			return nil
		case tcell.KeyF2:
			// Show alerts modal
			s.showAlertsModal()
			return nil
		case tcell.KeyF3:
			// Show preferences modal
			s.showPreferences()
			return nil
		case tcell.KeyF4:
			// Show layout switcher
			s.showLayoutSwitcher()
			return nil
		case tcell.KeyF5:
			// Refresh current view
			if err := s.viewMgr.RefreshCurrentView(); err != nil {
				s.statusBar.Error(fmt.Sprintf("Failed to refresh: %v", err))
			} else {
				// Show success message briefly, then restore hints
				s.statusBar.Success("Refreshed")
				// Restore hints after a short delay
				go func() {
					time.Sleep(3500 * time.Millisecond) // Wait slightly longer than success message
					if currentView, err := s.viewMgr.GetCurrentView(); err == nil {
						s.statusBar.SetHints(currentView.Hints())
					}
				}()
			}
			return nil
		case tcell.KeyF10:
			// Show configuration modal
			s.showConfiguration()
			return nil
		case tcell.KeyEsc:
			if s.cmdVisible {
				s.hideCommandLine()
				return nil
			}
		case tcell.KeyTab:
			if isModalOpen {
				// Let the modal handle tab navigation
				return event
			}
			_ = s.viewMgr.NextView()
			s.updateCurrentView()
			return nil
		case tcell.KeyBacktab:
			if isModalOpen {
				// Let the modal handle shift+tab navigation
				return event
			}
			_ = s.viewMgr.PreviousView()
			s.updateCurrentView()
			return nil
		case tcell.KeyRune:
			// If a modal is open, let it handle all character input except a few special cases
			if isModalOpen {
				switch event.Rune() {
				case ':':
					// Allow command mode even in modals for emergency commands
					s.showCommandLine()
					return nil
				default:
					// Let modal handle all other character input (including 1-7, s, c, etc.)
					return event
				}
			}

			// Normal global shortcuts when no modal is open
			switch event.Rune() {
			case ':':
				s.showCommandLine()
				return nil
			case '?':
				s.showHelp()
				return nil
			case '1':
				s.switchToView("jobs")
				return nil
			case '2':
				s.switchToView("nodes")
				return nil
			case '3':
				s.switchToView("partitions")
				return nil
			case '4':
				s.switchToView("reservations")
				return nil
			case '5':
				s.switchToView("qos")
				return nil
			case '6':
				s.switchToView("accounts")
				return nil
			case '7':
				s.switchToView("users")
				return nil
			case '8':
				s.switchToView("dashboard")
				return nil
			case '9':
				s.switchToView("health")
				return nil
			case '0':
				s.switchToView("performance")
				return nil
			case 'q', 'Q':
				_ = s.Stop()
				return nil
			}
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
