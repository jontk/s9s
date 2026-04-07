package app

import (
	"fmt"
	"strings"
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
  [yellow]1-0[white]         Switch views (Jobs/Nodes/Partitions/Reservations/QoS/Accounts/Users/Dashboard/Health/Performance)
  [yellow]Tab/Shift+Tab[white] Cycle between views
  [yellow]F1[white]         Show help
  [yellow]F2[white]         Show system alerts
  [yellow]F5[white]         Refresh current view
  [yellow]F6[white]         Pause/resume auto-refresh
  [yellow]F10[white]        Configuration settings
  [yellow]:[white]          Command mode (Tab to browse commands)
  [yellow]?[white]          Show this help
  [yellow]Ctrl+K[white]     Switch cluster
  [yellow]q, Ctrl+C[white]  Quit

[teal]Commands:[white]
  [yellow]:jobs, :j[white]      Jobs view       [yellow]:accounts[white]      Accounts view
  [yellow]:nodes, :n[white]     Nodes view      [yellow]:users[white]         Users view
  [yellow]:partitions, :p[white] Partitions     [yellow]:dashboard[white]     Dashboard view
  [yellow]:reservations[white]  Reservations    [yellow]:health[white]        Health view
  [yellow]:qos[white]           QoS view        [yellow]:performance[white]   Performance view
  [yellow]:refresh, :r[white]   Refresh         [yellow]:layout[white]        Layout switcher
  [yellow]:quit, :q[white]      Quit            [yellow]:help, :h[white]      Help

[teal]Common View Keys:[white] [gray](available in all data views)[white]
  [yellow]/[white] Filter    [yellow]f[white] Adv Filter    [yellow]Ctrl+F[white] Search    [yellow]S[white] Sort    [yellow]R[white] Refresh    [yellow]e[white] Export
`

	// Append current view shortcuts
	if currentView, err := s.viewMgr.GetCurrentView(); err == nil {
		viewName := currentView.Name()
		hints := currentView.Hints()
		if len(hints) > 0 {
			helpText += fmt.Sprintf("\n[teal]%s View:[white]\n", s.formatViewName(viewName))
			for _, hint := range hints {
				helpText += fmt.Sprintf("  %s\n", hint)
			}
		}
	}

	helpText += "\nPress [yellow]ESC[white] to close  •  [yellow]F1[white] for full keyboard reference"

	modal := tview.NewTextView().
		SetDynamicColors(true).
		SetText(helpText).
		SetTextAlign(tview.AlignLeft).
		SetScrollable(true)

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

// formatViewName capitalizes a view name for display
func (s *S9s) formatViewName(name string) string {
	if name == "" {
		return ""
	}
	return strings.ToUpper(name[:1]) + name[1:]
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

// showConfiguration displays the configuration management interface
func (s *S9s) showConfiguration() {
	// Create configuration view using the path the config was loaded from
	configView := views.NewConfigView(s.app, s.pages, s.config.ConfigPath)

	// Set callback for configuration changes
	configView.SetConfigChangedCallback(func(newConfig *config.Config) {
		if newConfig == nil {
			return
		}
		// Apply new configuration to running components (currently:
		// swap in the new config and re-arm the refresh ticker with
		// the new cadence). Other UI settings are picked up on the
		// next redraw or view switch.
		s.ApplyConfig(newConfig)
		s.statusBar.Success("Configuration applied")
	})

	// Add the config view as a modal-like page
	s.pages.AddPage("config", configView, true, true)
}

// showClusterSwitcher displays a modal to switch between configured clusters
func (s *S9s) showClusterSwitcher() {
	if len(s.config.Clusters) <= 1 {
		s.statusBar.Info("Only one cluster configured")
		return
	}

	list := tview.NewList()
	list.SetBorder(true).
		SetTitle(" Switch Cluster ").
		SetTitleAlign(tview.AlignCenter)

	for _, cl := range s.config.Clusters {
		name := cl.Name
		secondary := cl.Cluster.Endpoint
		if name == s.config.DefaultCluster {
			secondary += " (current)"
		}
		list.AddItem(name, secondary, 0, func() {
			s.pages.RemovePage("cluster-switcher")
			s.switchCluster(name)
		})
	}

	list.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEsc {
			s.pages.RemovePage("cluster-switcher")
			return nil
		}
		return event
	})

	// Center the list in a modal-like layout
	modal := tview.NewFlex().
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().
			SetDirection(tview.FlexRow).
			AddItem(nil, 0, 1, false).
			AddItem(list, min(len(s.config.Clusters)*2+2, 16), 0, true).
			AddItem(nil, 0, 1, false), 50, 0, true).
		AddItem(nil, 0, 1, false)

	s.pages.AddPage("cluster-switcher", modal, true, true)
	s.app.SetFocus(list)
}

// switchCluster switches the active cluster connection
func (s *S9s) switchCluster(clusterName string) {
	if clusterName == s.config.DefaultCluster {
		return
	}

	s.statusBar.Info(fmt.Sprintf("Switching to cluster %s...", clusterName))

	// Update config
	s.config.DefaultCluster = clusterName

	// Create new client
	newClient, err := createSlurmClient(s.ctx, s.config, s.cancel)
	if err != nil {
		s.statusBar.Error(fmt.Sprintf("Failed to connect to %s: %v", clusterName, err))
		return
	}

	// Update app client
	s.client = newClient

	// Update all views
	for _, view := range s.viewMgr.GetViews() {
		if setter, ok := view.(views.ClientSetter); ok {
			setter.SetClient(newClient)
		}
	}

	// Update header
	s.header.SetClusterName(clusterName)

	// Refresh current view
	if err := s.viewMgr.RefreshCurrentView(); err != nil {
		s.statusBar.Error(fmt.Sprintf("Connected to %s but refresh failed: %v", clusterName, err))
		return
	}

	s.statusBar.Success(fmt.Sprintf("Switched to cluster %s", clusterName))
}

// showLayoutSwitcher displays the layout switcher modal
func (s *S9s) showLayoutSwitcher() {
	if s.layoutManager != nil {
		layouts.ShowLayoutSwitcher(s.layoutManager, s.app, s.pages, func(layoutID string) {
			// Update preferences with selected layout
			_ = s.userPrefs.Update(func(p *preferences.UserPreferences) error {
				p.Layouts.CurrentLayout = layoutID
				return nil
			})

			if layoutID == "default" {
				s.restoreDashboardDefault()
				s.statusBar.Success("Restored default dashboard")
			} else {
				// Re-wire layout container only if coming from default (no container set)
				if dashView, err := s.viewMgr.GetView("dashboard"); err == nil {
					if dv, ok := dashView.(*views.DashboardView); ok && !dv.HasLayoutContainer() {
						dv.SetLayoutContainer(s.layoutManager.GetContainer())
					}
				}
				s.applyDashboardLayout()
				s.statusBar.Success(fmt.Sprintf("Switched to %s layout", layoutID))
			}

			// Force redraw so the new layout is visible immediately
			s.app.ForceDraw()
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
