package app

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/jontk/s9s/internal/config"
	"github.com/jontk/s9s/internal/layouts"
	"github.com/jontk/s9s/internal/preferences"
	"github.com/jontk/s9s/internal/ui/components"
	"github.com/jontk/s9s/internal/ui/views/settings"
	"github.com/jontk/s9s/internal/views"
	"github.com/rivo/tview"
)

func (s *S9s) showHelp() {
	helpText := `[yellow]S9S - SLURM Terminal UI Help[white]

[teal]Global Keys:[white]
  [yellow]1-0[white]         Switch to Jobs/Nodes/Partitions/Reservations/QoS/Accounts/Users/Dashboard/Health/Performance view
  [yellow]Tab/Shift+Tab[white] Switch between views
  [yellow]F1[white]         Show help
  [yellow]F2[white]         Show system alerts
  [yellow]F5[white]         Refresh current view
  [yellow]F10[white]        Configuration settings
  [yellow]:[white]          Enter command mode
  [yellow]?[white]          Show this help
  [yellow]q, Ctrl+C[white]  Quit application

[teal]Commands:[white]
  [yellow]:jobs, :j[white]      Switch to Jobs view
  [yellow]:nodes, :n[white]     Switch to Nodes view
  [yellow]:partitions, :p[white] Switch to Partitions view
  [yellow]:reservations[white]  Switch to Reservations view
  [yellow]:qos[white]           Switch to QoS view
  [yellow]:accounts[white]      Switch to Accounts view
  [yellow]:users[white]         Switch to Users view
  [yellow]:dashboard[white]     Switch to Dashboard view
  [yellow]:health[white]        Switch to Health Monitor view
  [yellow]:performance[white]   Switch to Performance Monitor view
  [yellow]:refresh, :r[white]   Refresh current view
  [yellow]:quit, :q[white]      Quit application
  [yellow]:help, :h[white]      Show this help

[teal]View-specific keys vary by view.[white]
Press [yellow]ESC[white] to close this help.`

	modal := tview.NewTextView().
		SetDynamicColors(true).
		SetText(helpText).
		SetTextAlign(tview.AlignLeft)

	modal.SetBorder(true).
		SetTitle(" Help ").
		SetTitleAlign(tview.AlignCenter)

	modal.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEsc {
			s.pages.RemovePage("help")
			s.app.SetFocus(s.mainLayout)
			return nil
		}
		return event
	})

	s.pages.AddPage("help", modal, true, true)
}

// showAlertsModal displays the alerts modal
func (s *S9s) showAlertsModal() {
	alertsView := components.NewAlertsView(s.alertsManager, s.app)
	alertsView.SetPages(s.pages)
	alertsView.SetNotificationManager(s.notificationMgr)
	alertsView.SetKeyHandler()

	// Create modal layout with help text
	helpText := tview.NewTextView().
		SetDynamicColors(true).
		SetText("[yellow]Keys:[white] a=Acknowledge d=Dismiss c=Clear r=Refresh s=Settings Tab=Switch ESC=Close")

	modal := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().
			AddItem(nil, 0, 1, false).
			AddItem(tview.NewFlex().
				SetDirection(tview.FlexRow).
				AddItem(alertsView.GetView(), 0, 1, true).
				AddItem(helpText, 1, 0, false), 0, 4, true).
			AddItem(nil, 0, 1, false), 0, 3, true).
		AddItem(nil, 0, 1, false)

	modal.SetBorder(true).
		SetTitle(" System Alerts ").
		SetTitleAlign(tview.AlignCenter)

	// Handle ESC to close and 's' for settings
	modal.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEsc {
			s.pages.RemovePage("alerts")
			return nil
		}
		if event.Key() == tcell.KeyRune && event.Rune() == 's' {
			// Show notification settings
			settings.ShowNotificationSettings(s.pages, s.app, s.notificationMgr)
			return nil
		}
		return event
	})

	s.pages.AddPage("alerts", modal, true, true)
}

// showPreferences displays the preferences modal
func (s *S9s) showPreferences() {
	settings.ShowPreferences(s.pages, s.app, s.userPrefs)
}

// showConfiguration displays the configuration management interface
func (s *S9s) showConfiguration() {
	configPath := ""

	// Try to determine the configuration file path
	if homeDir, err := os.UserHomeDir(); err == nil {
		configPath = filepath.Join(homeDir, ".s9s", "config.yaml")
	}

	// Create configuration view
	configView := views.NewConfigView(s.app, s.pages, configPath)

	// Set callback for configuration changes
	configView.SetConfigChangedCallback(func(newConfig *config.Config) {
		// Apply new configuration to running application
		if newConfig != nil {
			s.config = newConfig
			// TODO: Apply configuration changes to running components
			// This would involve updating refresh rates, UI settings, etc.
			s.statusBar.Success("Configuration applied")
		}
	})

	// Add the config view as a modal-like page
	s.pages.AddPage("config", configView, true, true)
}

// showLayoutSwitcher displays the layout switcher modal
func (s *S9s) showLayoutSwitcher() {
	if layoutMgr, ok := s.layoutManager.(*layouts.LayoutManager); ok {
		layouts.ShowLayoutSwitcher(layoutMgr, s.app, s.pages, func(layoutID string) {
			// Update preferences with selected layout
			_ = s.userPrefs.Update(func(p *preferences.UserPreferences) error {
				p.Layouts.CurrentLayout = layoutID
				return nil
			})
			s.statusBar.Success(fmt.Sprintf("Switched to %s layout", layoutID))
			// Restore hints after success message expires
			go func() {
				time.Sleep(3500 * time.Millisecond) // Wait slightly longer than success message
				if currentView, err := s.viewMgr.GetCurrentView(); err == nil {
					s.statusBar.SetHints(currentView.Hints())
				}
			}()
		})
	} else {
		s.statusBar.Error("Layout manager not available")
	}
}
