package views

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/jontk/s9s/internal/dao"
	"github.com/jontk/s9s/internal/monitoring"
	"github.com/rivo/tview"
)

// HealthView displays cluster health monitoring information
type HealthView struct {
	*BaseView
	client        dao.SlurmClient
	healthMonitor *monitoring.HealthMonitor
	container     *tview.Flex
	healthBox     *tview.TextView
	alertsBox     *tview.TextView
	checksBox     *tview.TextView
	statusBar     *tview.TextView
	app           *tview.Application
	pages         *tview.Pages
	// TODO(lint): Review unused code - field mu is unused
	// mu            sync.RWMutex
	refreshTimer *time.Timer
	refreshRate  time.Duration
}

// NewHealthView creates a new health monitoring view
func NewHealthView(client dao.SlurmClient) *HealthView {
	v := &HealthView{
		BaseView:      NewBaseView("health", "Health Monitor"),
		client:        client,
		refreshRate:   10 * time.Second,                                    // Refresh every 10 seconds
		healthMonitor: monitoring.NewHealthMonitor(client, 30*time.Second), // Check every 30 seconds
	}

	// Create health overview box
	v.healthBox = tview.NewTextView()
	v.healthBox.SetDynamicColors(true).
		SetWordWrap(true).
		SetTitle(" Cluster Health Overview ").
		SetBorder(true)

	// Create alerts box
	v.alertsBox = tview.NewTextView()
	v.alertsBox.SetDynamicColors(true).
		SetWordWrap(true).
		SetScrollable(true).
		SetTitle(" Active Alerts ").
		SetBorder(true)

	// Create health checks box
	v.checksBox = tview.NewTextView()
	v.checksBox.SetDynamicColors(true).
		SetWordWrap(true).
		SetScrollable(true).
		SetTitle(" Health Checks ").
		SetBorder(true)

	// Create status bar
	v.statusBar = tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignLeft)

	// Create main layout
	topRow := tview.NewFlex().
		AddItem(v.healthBox, 0, 1, false).
		AddItem(v.alertsBox, 0, 1, false)

	v.container = tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(topRow, 0, 1, true).
		AddItem(v.checksBox, 0, 1, false)

	return v
}

// SetApp sets the application reference
func (v *HealthView) SetApp(app *tview.Application) {
	v.app = app
}

// SetPages sets the pages reference for modal handling
func (v *HealthView) SetPages(pages *tview.Pages) {
	v.pages = pages
}

// Init initializes the health view
func (v *HealthView) Init(ctx context.Context) error {
	_ = v.BaseView.Init(ctx)

	// Start health monitoring
	v.healthMonitor.Start()

	return v.Refresh()
}

// Render returns the view's main component
func (v *HealthView) Render() tview.Primitive {
	return v.container
}

// Refresh updates the health monitoring data
func (v *HealthView) Refresh() error {
	v.SetRefreshing(true)
	defer v.SetRefreshing(false)

	// Update health overview
	v.updateHealthOverview()

	// Update alerts
	v.updateAlerts()

	// Update health checks
	v.updateHealthChecks()

	// Note: Status bar update removed since individual view status bars are no longer used

	// Schedule next refresh
	v.scheduleRefresh()

	return nil
}

// Stop stops the view
func (v *HealthView) Stop() error {
	if v.refreshTimer != nil {
		v.refreshTimer.Stop()
	}
	if v.healthMonitor != nil {
		v.healthMonitor.Stop()
	}
	return nil
}

// Hints returns keyboard hints
func (v *HealthView) Hints() []string {
	return []string{
		"[yellow]Enter[white] Alert Details",
		"[yellow]a[white] Acknowledge Alert",
		"[yellow]r[white] Resolve Alert",
		"[yellow]c[white] Clear Resolved",
		"[yellow]R[white] Refresh",
		"[yellow]h[white] Health Details",
		"[yellow]s[white] Health Stats",
	}
}

// OnKey handles keyboard events
func (v *HealthView) OnKey(event *tcell.EventKey) *tcell.EventKey {
	// Check if a modal is open
	if v.pages != nil && v.pages.GetPageCount() > 1 {
		return event
	}

	switch event.Key() {
	case tcell.KeyRune:
		switch event.Rune() {
		case 'a', 'A':
			v.acknowledgeAlert()
			return nil
		case 'r':
			v.resolveAlert()
			return nil
		case 'R':
			go func() { _ = v.Refresh() }()
			return nil
		case 'c', 'C':
			v.clearResolvedAlerts()
			return nil
		case 'h', 'H':
			v.showHealthDetails()
			return nil
		case 's', 'S':
			v.showHealthStats()
			return nil
		}
	case tcell.KeyEnter:
		v.showAlertDetails()
		return nil
	}

	return event
}

// OnFocus handles focus events
func (v *HealthView) OnFocus() error {
	if v.app != nil {
		v.app.SetFocus(v.container)
	}
	// Restart the refresh timer when regaining focus
	v.scheduleRefresh()
	return nil
}

// OnLoseFocus handles loss of focus
func (v *HealthView) OnLoseFocus() error {
	// Stop the refresh timer to prevent background updates
	if v.refreshTimer != nil {
		v.refreshTimer.Stop()
	}
	return nil
}

// updateHealthOverview updates the health overview display
func (v *HealthView) updateHealthOverview() {
	health := v.healthMonitor.GetHealth()

	var overview strings.Builder

	// Overall status
	statusColor := v.getStatusColor(health.OverallStatus)
	overview.WriteString(fmt.Sprintf("[%s]● Overall Status: %s[white]\n\n",
		statusColor, strings.ToUpper(string(health.OverallStatus))))

	// Health check summary
	totalChecks := len(health.Checks)
	healthyChecks := 0
	warningChecks := 0
	criticalChecks := 0

	for _, check := range health.Checks {
		switch check.Status {
		case monitoring.HealthStatusHealthy:
			healthyChecks++
		case monitoring.HealthStatusWarning:
			warningChecks++
		case monitoring.HealthStatusCritical:
			criticalChecks++
		}
	}

	overview.WriteString("[teal]Health Checks:[white]\n")
	overview.WriteString(fmt.Sprintf("  [green]✓ Healthy: %d[white]\n", healthyChecks))
	if warningChecks > 0 {
		overview.WriteString(fmt.Sprintf("  [yellow]⚠ Warning: %d[white]\n", warningChecks))
	}
	if criticalChecks > 0 {
		overview.WriteString(fmt.Sprintf("  [red]✗ Critical: %d[white]\n", criticalChecks))
	}
	overview.WriteString(fmt.Sprintf("  Total: %d checks\n\n", totalChecks))

	// Alert summary
	alertStats := v.healthMonitor.GetAlertManager().GetStats()
	overview.WriteString("[teal]Alerts:[white]\n")
	overview.WriteString(fmt.Sprintf("  [red]Critical: %d[white]\n", alertStats.Critical))
	overview.WriteString(fmt.Sprintf("  [yellow]Warning: %d[white]\n", alertStats.Warning))
	overview.WriteString(fmt.Sprintf("  [blue]Info: %d[white]\n", alertStats.Info))
	overview.WriteString(fmt.Sprintf("  Active: %d | Acknowledged: %d\n\n",
		alertStats.Active, alertStats.Acknowledged))

	// Last update
	overview.WriteString(fmt.Sprintf("[gray]Last Updated: %s[white]",
		health.LastUpdated.Format("15:04:05")))

	v.healthBox.SetText(overview.String())
}

// updateAlerts updates the alerts display
func (v *HealthView) updateAlerts() {
	alertManager := v.healthMonitor.GetAlertManager()
	activeAlerts := alertManager.GetActiveAlerts()

	var alertsText strings.Builder

	if len(activeAlerts) == 0 {
		alertsText.WriteString("[green]No active alerts[white]")
	} else {
		for i, alert := range activeAlerts {
			if i >= 10 { // Limit to 10 most recent alerts
				alertsText.WriteString(fmt.Sprintf("\n[gray]... and %d more alerts[white]", len(activeAlerts)-i))
				break
			}

			severityColor := monitoring.GetSeverityColor(alert.Severity)
			icon := monitoring.GetSeverityIcon(alert.Severity)
			ackStatus := ""
			if alert.Acknowledged {
				ackStatus = " [gray](acked)[white]"
			}

			alertsText.WriteString(fmt.Sprintf("[%s]%s %s[white]%s\n",
				severityColor, icon, alert.Title, ackStatus))
			alertsText.WriteString(fmt.Sprintf("   %s\n", alert.Message))
			alertsText.WriteString(fmt.Sprintf("   [gray]%s[white]\n\n",
				alert.Timestamp.Format("15:04:05")))
		}
	}

	v.alertsBox.SetText(alertsText.String())
}

// updateHealthChecks updates the health checks display
func (v *HealthView) updateHealthChecks() {
	health := v.healthMonitor.GetHealth()

	var checksText strings.Builder

	for name, check := range health.Checks {
		statusColor := v.getStatusColor(check.Status)
		icon := v.getStatusIcon(check.Status)

		checksText.WriteString(fmt.Sprintf("[%s]%s %s[white]\n",
			statusColor, icon, strings.Title(name)))
		checksText.WriteString(fmt.Sprintf("   %s\n", check.Message))
		checksText.WriteString(fmt.Sprintf("   [gray]Last Check: %s | Count: %d[white]\n\n",
			check.LastCheck.Format("15:04:05"), check.CheckCount))
	}

	v.checksBox.SetText(checksText.String())
}

/*
TODO(lint): Review unused code - func (*HealthView).updateStatusBar is unused

updateStatusBar updates the status bar
func (v *HealthView) updateStatusBar() {
	health := v.healthMonitor.GetHealth()
	alertStats := v.healthMonitor.GetAlertManager().GetStats()

	status := fmt.Sprintf("Health: [%s]%s[white] | Alerts: [red]%d Critical[white], [yellow]%d Warning[white] | Checks: %d",
		v.getStatusColor(health.OverallStatus), strings.ToUpper(string(health.OverallStatus)),
		alertStats.Critical, alertStats.Warning, len(health.Checks))

	if v.IsRefreshing() {
		status += " | [yellow]Refreshing...[white]"
	}

	v.statusBar.SetText(status)
}
*/

// scheduleRefresh schedules the next refresh
func (v *HealthView) scheduleRefresh() {
	if v.refreshTimer != nil {
		v.refreshTimer.Stop()
	}

	v.refreshTimer = time.AfterFunc(v.refreshRate, func() {
		if v.app != nil {
			v.app.QueueUpdateDraw(func() {
				_ = v.Refresh()
			})
		}
	})
}

// getStatusColor returns the color for a health status
func (v *HealthView) getStatusColor(status monitoring.HealthStatus) string {
	switch status {
	case monitoring.HealthStatusHealthy:
		return "green"
	case monitoring.HealthStatusWarning:
		return "yellow"
	case monitoring.HealthStatusCritical:
		return "red"
	default:
		return "gray"
	}
}

// getStatusIcon returns an icon for a health status
func (v *HealthView) getStatusIcon(status monitoring.HealthStatus) string {
	switch status {
	case monitoring.HealthStatusHealthy:
		return "✓"
	case monitoring.HealthStatusWarning:
		return "⚠"
	case monitoring.HealthStatusCritical:
		return "✗"
	default:
		return "?"
	}
}

// acknowledgeAlert acknowledges the first unacknowledged alert
func (v *HealthView) acknowledgeAlert() {
	alertManager := v.healthMonitor.GetAlertManager()
	alerts := alertManager.GetActiveAlerts()

	for _, alert := range alerts {
		if !alert.Acknowledged {
			err := alertManager.AcknowledgeAlert(alert.ID, "user")
			if err != nil {
				// Note: Status bar update removed since individual view status bars are no longer used
			} else {
				v.updateAlerts()
			}
			break
		}
	}
}

// resolveAlert resolves the first unresolved alert
func (v *HealthView) resolveAlert() {
	alertManager := v.healthMonitor.GetAlertManager()
	alerts := alertManager.GetActiveAlerts()

	if len(alerts) > 0 {
		err := alertManager.ResolveAlert(alerts[0].ID)
		if err != nil {
			// Note: Status bar update removed since individual view status bars are no longer used
		} else {
			v.updateAlerts()
		}
	}
}

// clearResolvedAlerts clears old resolved alerts
func (v *HealthView) clearResolvedAlerts() {
	alertManager := v.healthMonitor.GetAlertManager()
	cleared := alertManager.ClearResolvedAlerts(24 * time.Hour) // Clear alerts older than 24 hours

	v.statusBar.SetText(fmt.Sprintf("[green]Cleared %d resolved alerts[white]", cleared))
	v.updateAlerts()
}

// showAlertDetails shows detailed information about alerts
func (v *HealthView) showAlertDetails() {
	if v.pages == nil {
		return
	}

	alertManager := v.healthMonitor.GetAlertManager()
	allAlerts := alertManager.GetAlerts(monitoring.AlertFilter{})

	var details strings.Builder
	details.WriteString("[yellow]Alert Details[white]\n\n")

	if len(allAlerts) == 0 {
		details.WriteString("No alerts found.")
	} else {
		for i, alert := range allAlerts {
			if i >= 20 { // Limit to prevent huge modals
				details.WriteString(fmt.Sprintf("... and %d more alerts\n", len(allAlerts)-i))
				break
			}

			severityColor := monitoring.GetSeverityColor(alert.Severity)
			resolvedText := ""
			if alert.Resolved {
				resolvedText = " [green](Resolved)[white]"
			}
			ackedText := ""
			if alert.Acknowledged {
				ackedText = " [gray](Acknowledged)[white]"
			}

			details.WriteString(fmt.Sprintf("[%s]%s %s[white]%s%s\n",
				severityColor, monitoring.GetSeverityIcon(alert.Severity),
				alert.Title, resolvedText, ackedText))
			details.WriteString(fmt.Sprintf("Component: %s\n", alert.Component))
			details.WriteString(fmt.Sprintf("Message: %s\n", alert.Message))
			details.WriteString(fmt.Sprintf("Time: %s\n", alert.Timestamp.Format("2006-01-02 15:04:05")))
			if alert.Count > 1 {
				details.WriteString(fmt.Sprintf("Count: %d (last seen: %s)\n",
					alert.Count, alert.LastSeen.Format("15:04:05")))
			}
			details.WriteString("\n")
		}
	}

	v.showModal("alert-details", "Alert Details", details.String())
}

// showHealthDetails shows detailed health information
func (v *HealthView) showHealthDetails() {
	if v.pages == nil {
		return
	}

	health := v.healthMonitor.GetHealth()

	var details strings.Builder
	details.WriteString("[yellow]Health Check Details[white]\n\n")

	for name, check := range health.Checks {
		statusColor := v.getStatusColor(check.Status)
		details.WriteString(fmt.Sprintf("[%s]%s %s[white]\n",
			statusColor, v.getStatusIcon(check.Status), strings.Title(name)))
		details.WriteString(fmt.Sprintf("Description: %s\n", check.Description))
		details.WriteString(fmt.Sprintf("Status: %s\n", check.Status))
		details.WriteString(fmt.Sprintf("Message: %s\n", check.Message))
		details.WriteString(fmt.Sprintf("Last Check: %s\n", check.LastCheck.Format("2006-01-02 15:04:05")))
		details.WriteString(fmt.Sprintf("Check Count: %d\n", check.CheckCount))

		// Show thresholds if available
		if check.Threshold.WarningMax != nil || check.Threshold.CriticalMax != nil {
			details.WriteString("Thresholds:\n")
			if check.Threshold.WarningMax != nil {
				details.WriteString(fmt.Sprintf("  Warning: %.1f\n", *check.Threshold.WarningMax))
			}
			if check.Threshold.CriticalMax != nil {
				details.WriteString(fmt.Sprintf("  Critical: %.1f\n", *check.Threshold.CriticalMax))
			}
		}
		details.WriteString("\n")
	}

	v.showModal("health-details", "Health Check Details", details.String())
}

// showHealthStats shows health statistics
func (v *HealthView) showHealthStats() {
	if v.pages == nil {
		return
	}

	alertStats := v.healthMonitor.GetAlertManager().GetStats()
	health := v.healthMonitor.GetHealth()

	var stats strings.Builder
	stats.WriteString("[yellow]Health Monitoring Statistics[white]\n\n")

	stats.WriteString("[teal]Alert Statistics:[white]\n")
	stats.WriteString(fmt.Sprintf("Total Alerts: %d\n", alertStats.Total))
	stats.WriteString(fmt.Sprintf("Critical: %d\n", alertStats.Critical))
	stats.WriteString(fmt.Sprintf("Warning: %d\n", alertStats.Warning))
	stats.WriteString(fmt.Sprintf("Info: %d\n", alertStats.Info))
	stats.WriteString(fmt.Sprintf("Active: %d\n", alertStats.Active))
	stats.WriteString(fmt.Sprintf("Resolved: %d\n", alertStats.Resolved))
	stats.WriteString(fmt.Sprintf("Acknowledged: %d\n", alertStats.Acknowledged))
	stats.WriteString(fmt.Sprintf("Unacknowledged: %d\n\n", alertStats.Unacknowledged))

	stats.WriteString("[teal]Health Check Statistics:[white]\n")
	stats.WriteString(fmt.Sprintf("Total Checks: %d\n", len(health.Checks)))
	stats.WriteString(fmt.Sprintf("Overall Status: %s\n", health.OverallStatus))
	stats.WriteString(fmt.Sprintf("Last Updated: %s\n", health.LastUpdated.Format("2006-01-02 15:04:05")))

	v.showModal("health-stats", "Health Statistics", stats.String())
}

// showModal displays a modal dialog
func (v *HealthView) showModal(name, title, content string) {
	textView := tview.NewTextView().
		SetDynamicColors(true).
		SetText(content).
		SetScrollable(true)

	modal := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(textView, 0, 1, true).
		AddItem(tview.NewTextView().SetText("Press ESC to close"), 1, 0, false)

	modal.SetBorder(true).
		SetTitle(fmt.Sprintf(" %s ", title)).
		SetTitleAlign(tview.AlignCenter)

	// Handle ESC key
	modal.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEsc {
			v.pages.RemovePage(name)
			return nil
		}
		return event
	})

	// Create centered modal layout
	centeredModal := tview.NewFlex().
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(nil, 0, 1, false).
			AddItem(modal, 0, 8, true).
			AddItem(nil, 0, 1, false), 0, 8, true).
		AddItem(nil, 0, 1, false)

	v.pages.AddPage(name, centeredModal, true, true)
}
