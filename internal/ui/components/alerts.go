// Package components provides reusable UI components for the s9s interface.
package components

import (
	"fmt"
	"sync"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/jontk/s9s/internal/monitoring"
	"github.com/rivo/tview"
)

// AlertLevel represents the severity of an alert
type AlertLevel int

const (
	// AlertInfo is the info alert level.
	AlertInfo AlertLevel = iota
	// AlertWarning is the warning alert level.
	AlertWarning
	// AlertError is the error alert level.
	AlertError
	// AlertCritical is the critical alert level.
	AlertCritical
)

// Alert represents a system alert
type Alert struct {
	ID           string
	Level        AlertLevel
	Title        string
	Message      string
	Source       string // e.g., "nodes", "jobs", "cluster"
	Timestamp    time.Time
	Acknowledged bool
	AutoDismiss  bool
	DismissAfter time.Duration
}

// AlertsManager manages system alerts
type AlertsManager struct {
	mu          sync.RWMutex
	alerts      []*Alert
	maxAlerts   int
	listeners   []func(*Alert)
	dismissChan chan string
	notifier    AlertNotifier
}

// AlertNotifier is an interface for sending alert notifications
type AlertNotifier interface {
	Notify(alert *Alert)
}

// NewAlertsManager creates a new alerts manager
func NewAlertsManager(maxAlerts int) *AlertsManager {
	am := &AlertsManager{
		alerts:      make([]*Alert, 0),
		maxAlerts:   maxAlerts,
		listeners:   make([]func(*Alert), 0),
		dismissChan: make(chan string, 10),
	}

	// Start auto-dismiss routine
	go am.autoDismissRoutine()

	return am
}

// SetNotifier sets the alert notifier
func (am *AlertsManager) SetNotifier(notifier AlertNotifier) {
	am.mu.Lock()
	defer am.mu.Unlock()
	am.notifier = notifier
}

// AddAlert adds a new alert
func (am *AlertsManager) AddAlert(alert *Alert) {
	am.mu.Lock()
	defer am.mu.Unlock()

	// Generate ID if not provided
	if alert.ID == "" {
		alert.ID = fmt.Sprintf("%s-%d", alert.Source, time.Now().UnixNano())
	}

	// Set timestamp
	alert.Timestamp = time.Now()

	// Add to beginning of slice (newest first)
	am.alerts = append([]*Alert{alert}, am.alerts...)

	// Trim if over max
	if len(am.alerts) > am.maxAlerts {
		am.alerts = am.alerts[:am.maxAlerts]
	}

	// Notify listeners
	for _, listener := range am.listeners {
		go listener(alert)
	}

	// Send notification through notifier
	if am.notifier != nil && !alert.Acknowledged {
		go am.notifier.Notify(alert)
	}

	// Schedule auto-dismiss if enabled
	if alert.AutoDismiss && alert.DismissAfter > 0 {
		go func() {
			time.Sleep(alert.DismissAfter)
			am.dismissChan <- alert.ID
		}()
	}
}

// GetAlerts returns all current alerts
func (am *AlertsManager) GetAlerts() []*Alert {
	am.mu.RLock()
	defer am.mu.RUnlock()

	// Return copy to prevent external modification
	alerts := make([]*Alert, len(am.alerts))
	copy(alerts, am.alerts)
	return alerts
}

// GetUnacknowledgedAlerts returns only unacknowledged alerts
func (am *AlertsManager) GetUnacknowledgedAlerts() []*Alert {
	am.mu.RLock()
	defer am.mu.RUnlock()

	var unacked []*Alert
	for _, alert := range am.alerts {
		if !alert.Acknowledged {
			unacked = append(unacked, alert)
		}
	}
	return unacked
}

// AcknowledgeAlert marks an alert as acknowledged
func (am *AlertsManager) AcknowledgeAlert(id string) {
	am.mu.Lock()
	defer am.mu.Unlock()

	for _, alert := range am.alerts {
		if alert.ID == id {
			alert.Acknowledged = true
			break
		}
	}
}

// DismissAlert removes an alert
func (am *AlertsManager) DismissAlert(id string) {
	am.mu.Lock()
	defer am.mu.Unlock()

	for i, alert := range am.alerts {
		if alert.ID == id {
			am.alerts = append(am.alerts[:i], am.alerts[i+1:]...)
			break
		}
	}
}

// ClearAllAlerts removes all alerts
func (am *AlertsManager) ClearAllAlerts() {
	am.mu.Lock()
	defer am.mu.Unlock()

	am.alerts = make([]*Alert, 0)
}

// OnAlert registers a listener for new alerts
func (am *AlertsManager) OnAlert(listener func(*Alert)) {
	am.mu.Lock()
	defer am.mu.Unlock()

	am.listeners = append(am.listeners, listener)
}

// autoDismissRoutine handles automatic alert dismissal
func (am *AlertsManager) autoDismissRoutine() {
	for id := range am.dismissChan {
		am.DismissAlert(id)
	}
}

// CheckClusterHealth generates alerts based on cluster health
func (am *AlertsManager) CheckClusterHealth(health *monitoring.ClusterHealth) {
	// Check overall status
	switch health.OverallStatus {
	case monitoring.HealthStatusCritical:
		am.AddAlert(&Alert{
			Level:   AlertCritical,
			Title:   "Critical Cluster Health",
			Message: "Cluster is in critical state - immediate attention required",
			Source:  "cluster",
		})
	case monitoring.HealthStatusWarning:
		am.AddAlert(&Alert{
			Level:        AlertWarning,
			Title:        "Cluster Health Warning",
			Message:      "Cluster health is degraded - review health checks",
			Source:       "cluster",
			AutoDismiss:  true,
			DismissAfter: 10 * time.Minute,
		})
	}

	// Check individual health checks
	for _, check := range health.Checks {
		switch check.Status {
		case monitoring.HealthStatusCritical:
			am.AddAlert(&Alert{
				Level:   AlertCritical,
				Title:   fmt.Sprintf("Critical: %s", check.Name),
				Message: check.Message,
				Source:  "health-check",
			})
		case monitoring.HealthStatusWarning:
			am.AddAlert(&Alert{
				Level:        AlertWarning,
				Title:        fmt.Sprintf("Warning: %s", check.Name),
				Message:      check.Message,
				Source:       "health-check",
				AutoDismiss:  true,
				DismissAfter: 15 * time.Minute,
			})
		}
	}

	// Check for issues
	for _, issue := range health.Issues {
		if !issue.Resolved {
			level := AlertInfo
			switch issue.Severity {
			case monitoring.HealthStatusCritical:
				level = AlertCritical
			case monitoring.HealthStatusWarning:
				level = AlertWarning
			}

			am.AddAlert(&Alert{
				ID:      issue.ID,
				Level:   level,
				Title:   issue.Title,
				Message: issue.Description,
				Source:  issue.Component,
			})
		}
	}
}

// AlertsView displays alerts in a modal
type AlertsView struct {
	manager         *AlertsManager
	list            *tview.List
	detailView      *tview.TextView
	flex            *tview.Flex
	app             *tview.Application
	pages           *tview.Pages
	notificationMgr interface{} // *notifications.NotificationManager
}

// NewAlertsView creates a new alerts view
func NewAlertsView(manager *AlertsManager, app *tview.Application) *AlertsView {
	av := &AlertsView{
		manager:    manager,
		list:       tview.NewList(),
		detailView: tview.NewTextView(),
		app:        app,
	}

	// Configure list
	av.list.SetBorder(true).SetTitle(" Alerts ")
	av.list.SetSelectedFunc(av.onAlertSelected)
	av.list.SetChangedFunc(av.onAlertChanged)

	// Configure detail view
	av.detailView.SetBorder(true).SetTitle(" Alert Details ")
	av.detailView.SetDynamicColors(true)
	av.detailView.SetWordWrap(true)

	// Create layout
	av.flex = tview.NewFlex().
		AddItem(av.list, 0, 1, true).
		AddItem(av.detailView, 0, 2, false)

	// Refresh alerts
	av.refreshAlerts()

	return av
}

// refreshAlerts updates the alerts list
func (av *AlertsView) refreshAlerts() {
	av.list.Clear()

	alerts := av.manager.GetAlerts()
	if len(alerts) == 0 {
		av.list.AddItem("No alerts", "", 0, nil)
		av.detailView.SetText("[gray]No alerts to display[white]")
		return
	}

	for _, alert := range alerts {
		icon := av.getAlertIcon(alert.Level)
		color := av.getAlertColor(alert.Level)

		title := fmt.Sprintf("%s %s", icon, alert.Title)
		if alert.Acknowledged {
			title = fmt.Sprintf("%s (ack)", title)
		}

		secondary := fmt.Sprintf("%s - %s", alert.Source, alert.Timestamp.Format("15:04:05"))

		av.list.AddItem(title, secondary, 0, nil).
			SetItemText(av.list.GetItemCount()-1, title, secondary).
			ShowSecondaryText(true)

		// Set color based on level
		av.list.SetItemText(av.list.GetItemCount()-1,
			fmt.Sprintf("[%s]%s[white]", color, title),
			fmt.Sprintf("[gray]%s[white]", secondary))
	}

	// Show first alert details
	if len(alerts) > 0 {
		av.showAlertDetails(alerts[0])
	}
}

// getAlertIcon returns an icon for the alert level
func (av *AlertsView) getAlertIcon(level AlertLevel) string {
	switch level {
	case AlertInfo:
		return "ℹ"
	case AlertWarning:
		return "⚠"
	case AlertError:
		return "⚡"
	case AlertCritical:
		return "🔴"
	default:
		return "•"
	}
}

// getAlertColor returns a color for the alert level
func (av *AlertsView) getAlertColor(level AlertLevel) string {
	switch level {
	case AlertInfo:
		return "cyan"
	case AlertWarning:
		return "yellow"
	case AlertError:
		return "orange"
	case AlertCritical:
		return "red"
	default:
		return "white"
	}
}

// onAlertSelected handles alert selection
func (av *AlertsView) onAlertSelected(index int, _, _ string, _ rune) {
	alerts := av.manager.GetAlerts()
	if index >= 0 && index < len(alerts) {
		alert := alerts[index]

		// Mark as acknowledged
		if !alert.Acknowledged {
			av.manager.AcknowledgeAlert(alert.ID)
			av.refreshAlerts()
		}
	}
}

// onAlertChanged handles cursor movement in the alerts list
func (av *AlertsView) onAlertChanged(index int, _, _ string, _ rune) {
	alerts := av.manager.GetAlerts()
	if index >= 0 && index < len(alerts) {
		av.showAlertDetails(alerts[index])
	}
}

// showAlertDetails displays detailed information about an alert
func (av *AlertsView) showAlertDetails(alert *Alert) {
	color := av.getAlertColor(alert.Level)
	levelStr := av.getLevelString(alert.Level)

	details := fmt.Sprintf(`[%s]%s Alert[white]

[yellow]Title:[white] %s
[yellow]Level:[white] [%s]%s[white]
[yellow]Source:[white] %s
[yellow]Time:[white] %s
[yellow]Status:[white] %s

[yellow]Message:[white]
%s

[gray]Alert ID: %s[white]`,
		color, levelStr,
		alert.Title,
		color, levelStr,
		alert.Source,
		alert.Timestamp.Format("2006-01-02 15:04:05"),
		av.getStatusString(alert),
		alert.Message,
		alert.ID,
	)

	av.detailView.SetText(details)
}

// getLevelString returns a string representation of the alert level
func (av *AlertsView) getLevelString(level AlertLevel) string {
	switch level {
	case AlertInfo:
		return "Information"
	case AlertWarning:
		return "Warning"
	case AlertError:
		return "Error"
	case AlertCritical:
		return "Critical"
	default:
		return "Unknown"
	}
}

// getStatusString returns the status of an alert
func (av *AlertsView) getStatusString(alert *Alert) string {
	if alert.Acknowledged {
		return "[green]Acknowledged[white]"
	}
	return "[red]Unacknowledged[white]"
}

// GetView returns the alerts view component
func (av *AlertsView) GetView() tview.Primitive {
	return av.flex
}

// SetKeyHandler sets up keyboard shortcuts
func (av *AlertsView) SetKeyHandler() {
	av.flex.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyRune:
			if av.handleAlertRune(event.Rune()) {
				return nil
			}
		case tcell.KeyTab:
			av.handleTabNavigation()
			return nil
		}
		return event
	})
}

// handleAlertRune handles character shortcuts for alert management
func (av *AlertsView) handleAlertRune(r rune) bool {
	switch r {
	case 'a', 'A':
		av.acknowledgeCurrentAlert()
		return true
	case 'd', 'D':
		av.dismissCurrentAlert()
		return true
	case 'c', 'C':
		av.manager.ClearAllAlerts()
		av.refreshAlerts()
		return true
	case 'r', 'R':
		av.refreshAlerts()
		return true
	case 's', 'S':
		// Show notification settings
		// Import cycle prevented - notification settings moved to separate package
		// The parent component (app.go) will handle showing notification settings
		return true
	}
	return false
}

// acknowledgeCurrentAlert acknowledges the current alert
func (av *AlertsView) acknowledgeCurrentAlert() {
	index := av.list.GetCurrentItem()
	alerts := av.manager.GetAlerts()
	if index >= 0 && index < len(alerts) {
		av.manager.AcknowledgeAlert(alerts[index].ID)
		av.refreshAlerts()
	}
}

// dismissCurrentAlert dismisses the current alert
func (av *AlertsView) dismissCurrentAlert() {
	index := av.list.GetCurrentItem()
	alerts := av.manager.GetAlerts()
	if index >= 0 && index < len(alerts) {
		av.manager.DismissAlert(alerts[index].ID)
		av.refreshAlerts()
	}
}

// handleTabNavigation switches focus between list and details
func (av *AlertsView) handleTabNavigation() {
	if av.app.GetFocus() == av.list {
		av.app.SetFocus(av.detailView)
	} else {
		av.app.SetFocus(av.list)
	}
}

// SetPages sets the pages reference for modal navigation
func (av *AlertsView) SetPages(pages *tview.Pages) {
	av.pages = pages
}

// SetNotificationManager sets the notification manager reference
func (av *AlertsView) SetNotificationManager(mgr interface{}) {
	av.notificationMgr = mgr
}

// AlertsBadge displays a small alert count badge
type AlertsBadge struct {
	manager *AlertsManager
	text    *tview.TextView
}

// NewAlertsBadge creates a new alerts badge
func NewAlertsBadge(manager *AlertsManager) *AlertsBadge {
	badge := &AlertsBadge{
		manager: manager,
		text:    tview.NewTextView(),
	}

	badge.text.SetDynamicColors(true)
	badge.text.SetTextAlign(tview.AlignCenter)
	badge.update()

	// Listen for new alerts
	manager.OnAlert(func(_ *Alert) {
		badge.update()
	})

	return badge
}

// update refreshes the badge display
func (ab *AlertsBadge) update() {
	unacked := ab.manager.GetUnacknowledgedAlerts()

	if len(unacked) == 0 {
		ab.text.SetText("")
		return
	}

	// Determine highest severity
	highestLevel := AlertInfo
	for _, alert := range unacked {
		if alert.Level > highestLevel {
			highestLevel = alert.Level
		}
	}

	// Set color based on highest severity
	var color string
	switch highestLevel {
	case AlertCritical:
		color = "red"
	case AlertError:
		color = "orange"
	case AlertWarning:
		color = "yellow"
	default:
		color = "cyan"
	}

	// Display count
	ab.text.SetText(fmt.Sprintf("[%s]🔔 %d[white]", color, len(unacked)))
}

// GetView returns the badge component
func (ab *AlertsBadge) GetView() tview.Primitive {
	return ab.text
}
