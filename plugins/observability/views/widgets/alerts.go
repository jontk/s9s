package widgets

import (
	"fmt"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	"github.com/jontk/s9s/plugins/observability/models"
)

// AlertsWidget displays active alerts
type AlertsWidget struct {
	*tview.TextView
	alerts      []models.Alert
	maxAlerts   int
	showTime    bool
}

// NewAlertsWidget creates a new alerts widget
func NewAlertsWidget() *AlertsWidget {
	w := &AlertsWidget{
		TextView:    tview.NewTextView(),
		alerts:      []models.Alert{},
		maxAlerts:   10,
		showTime:    true,
	}
	
	w.SetDynamicColors(true).
		SetBorder(true).
		SetTitle(" Active Alerts ").
		SetTitleAlign(tview.AlignLeft)
	
	w.updateDisplay()
	return w
}

// SetAlerts updates the alerts list
func (w *AlertsWidget) SetAlerts(alerts []models.Alert) {
	w.alerts = alerts
	w.updateDisplay()
}

// AddAlert adds a new alert
func (w *AlertsWidget) AddAlert(alert models.Alert) {
	w.alerts = append(w.alerts, alert)
	
	// Keep only the most recent alerts
	if len(w.alerts) > w.maxAlerts {
		w.alerts = w.alerts[len(w.alerts)-w.maxAlerts:]
	}
	
	w.updateDisplay()
}

// ClearAlerts removes all alerts
func (w *AlertsWidget) ClearAlerts() {
	w.alerts = []models.Alert{}
	w.updateDisplay()
}

// GetPrimitive returns the primitive for this widget
func (w *AlertsWidget) GetPrimitive() tview.Primitive {
	return w
}

// updateDisplay refreshes the alert display
func (w *AlertsWidget) updateDisplay() {
	w.Clear()
	
	if len(w.alerts) == 0 {
		w.SetText("[green]No active alerts")
		return
	}
	
	// Display alerts
	text := ""
	for i, alert := range w.alerts {
		if i > 0 {
			text += "\n"
		}
		
		// Format alert
		icon := w.getSeverityIcon(alert.Severity)
		color := w.getSeverityColor(alert.Severity)
		
		timeStr := ""
		if w.showTime {
			timeStr = fmt.Sprintf(" [%s]", w.formatTime(alert.Timestamp))
		}
		
		text += fmt.Sprintf("[%s]%s %s[white]%s - %s",
			color, icon, alert.Name, timeStr, alert.Message)
		
		// Add resolution hint if available
		if alert.Resolution != "" {
			text += fmt.Sprintf("\n    [gray]→ %s", alert.Resolution)
		}
	}
	
	w.SetText(text)
}

// getSeverityIcon returns an icon for the severity level
func (w *AlertsWidget) getSeverityIcon(severity string) string {
	switch severity {
	case "critical":
		return "⚠️ "
	case "warning":
		return "⚡"
	case "info":
		return "ℹ️ "
	default:
		return "• "
	}
}

// getSeverityColor returns a color for the severity level
func (w *AlertsWidget) getSeverityColor(severity string) string {
	switch severity {
	case "critical":
		return "red"
	case "warning":
		return "yellow"
	case "info":
		return "blue"
	default:
		return "white"
	}
}

// formatTime formats the timestamp for display
func (w *AlertsWidget) formatTime(t time.Time) string {
	now := time.Now()
	diff := now.Sub(t)
	
	if diff < time.Minute {
		return fmt.Sprintf("%ds ago", int(diff.Seconds()))
	} else if diff < time.Hour {
		return fmt.Sprintf("%dm ago", int(diff.Minutes()))
	} else if diff < 24*time.Hour {
		return fmt.Sprintf("%dh ago", int(diff.Hours()))
	}
	
	return t.Format("Jan 2 15:04")
}

// AlertSummaryWidget shows a compact alert summary
type AlertSummaryWidget struct {
	*tview.TextView
	counts map[string]int
}

// NewAlertSummaryWidget creates a new alert summary widget
func NewAlertSummaryWidget() *AlertSummaryWidget {
	w := &AlertSummaryWidget{
		TextView: tview.NewTextView(),
		counts:   make(map[string]int),
	}
	
	w.SetDynamicColors(true).
		SetTextAlign(tview.AlignCenter)
	
	w.updateDisplay()
	return w
}

// UpdateCounts updates the alert counts
func (w *AlertSummaryWidget) UpdateCounts(alerts []models.Alert) {
	w.counts = make(map[string]int)
	
	for _, alert := range alerts {
		w.counts[alert.Severity]++
	}
	
	w.updateDisplay()
}

// updateDisplay refreshes the summary display
func (w *AlertSummaryWidget) updateDisplay() {
	critical := w.counts["critical"]
	warning := w.counts["warning"]
	info := w.counts["info"]
	
	text := ""
	
	if critical > 0 {
		text += fmt.Sprintf("[red]⚠️  %d critical ", critical)
	}
	if warning > 0 {
		if text != "" {
			text += " "
		}
		text += fmt.Sprintf("[yellow]⚡ %d warning ", warning)
	}
	if info > 0 {
		if text != "" {
			text += " "
		}
		text += fmt.Sprintf("[blue]ℹ️  %d info ", info)
	}
	
	if text == "" {
		text = "[green]✓ No alerts"
	}
	
	w.SetText(text)
}

// AlertHistoryWidget displays alert history with scrolling
type AlertHistoryWidget struct {
	*tview.Table
	history     []models.Alert
	maxHistory  int
	selectedRow int
}

// NewAlertHistoryWidget creates a new alert history widget
func NewAlertHistoryWidget() *AlertHistoryWidget {
	w := &AlertHistoryWidget{
		Table:       tview.NewTable(),
		history:     []models.Alert{},
		maxHistory:  100,
		selectedRow: -1,
	}
	
	w.Table.SetBorder(true)
	w.Table.SetTitle(" Alert History ")
	w.Table.SetSelectable(true, false)
	w.Table.SetFixed(1, 0)
	
	// Set up headers
	headers := []string{"Time", "Severity", "Alert", "Message", "Status"}
	for i, header := range headers {
		cell := tview.NewTableCell(header).
			SetTextColor(tcell.ColorYellow).
			SetAttributes(tcell.AttrBold).
			SetSelectable(false)
		w.SetCell(0, i, cell)
	}
	
	w.SetSelectionChangedFunc(func(row, col int) {
		w.selectedRow = row - 1 // Adjust for header
	})
	
	return w
}

// AddToHistory adds an alert to the history
func (w *AlertHistoryWidget) AddToHistory(alert models.Alert) {
	w.history = append([]models.Alert{alert}, w.history...)
	
	if len(w.history) > w.maxHistory {
		w.history = w.history[:w.maxHistory]
	}
	
	w.updateDisplay()
}

// updateDisplay refreshes the history display
func (w *AlertHistoryWidget) updateDisplay() {
	// Clear existing rows (except header)
	for i := w.GetRowCount() - 1; i > 0; i-- {
		w.RemoveRow(i)
	}
	
	// Add history rows
	for i, alert := range w.history {
		row := i + 1
		
		// Time
		w.SetCell(row, 0, tview.NewTableCell(
			alert.Timestamp.Format("15:04:05")))
		
		// Severity with color
		severityCell := tview.NewTableCell(alert.Severity)
		switch alert.Severity {
		case "critical":
			severityCell.SetTextColor(tcell.ColorRed)
		case "warning":
			severityCell.SetTextColor(tcell.ColorYellow)
		case "info":
			severityCell.SetTextColor(tcell.ColorBlue)
		}
		w.SetCell(row, 1, severityCell)
		
		// Alert name
		w.SetCell(row, 2, tview.NewTableCell(alert.Name))
		
		// Message (truncated if needed)
		message := alert.Message
		if len(message) > 50 {
			message = message[:47] + "..."
		}
		w.SetCell(row, 3, tview.NewTableCell(message))
		
		// Status
		status := "Active"
		if alert.Resolved {
			status = "Resolved"
		}
		statusCell := tview.NewTableCell(status)
		if alert.Resolved {
			statusCell.SetTextColor(tcell.ColorGreen)
		} else {
			statusCell.SetTextColor(tcell.ColorRed)
		}
		w.SetCell(row, 4, statusCell)
	}
}

// GetSelectedAlert returns the currently selected alert
func (w *AlertHistoryWidget) GetSelectedAlert() *models.Alert {
	if w.selectedRow >= 0 && w.selectedRow < len(w.history) {
		return &w.history[w.selectedRow]
	}
	return nil
}