package layouts

import (
	"fmt"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/jontk/s9s/internal/dao"
	"github.com/jontk/s9s/internal/ui/components"
	"github.com/jontk/s9s/internal/views"
	"github.com/rivo/tview"
)

// BaseWidget provides common widget functionality
type BaseWidget struct {
	id          string
	name        string
	description string
	widgetType  WidgetType
	primitive   tview.Primitive
	minWidth    int
	minHeight   int
	maxWidth    int
	maxHeight   int
	focused     bool
	updateFunc  func() error
	configFunc  func() error
}

// ID returns the widget ID
func (w *BaseWidget) ID() string {
	return w.id
}

// Name returns the widget name
func (w *BaseWidget) Name() string {
	return w.name
}

// Description returns the widget description
func (w *BaseWidget) Description() string {
	return w.description
}

// Type returns the widget type
func (w *BaseWidget) Type() WidgetType {
	return w.widgetType
}

// Render returns the widget's tview primitive
func (w *BaseWidget) Render() tview.Primitive {
	return w.primitive
}

// Update calls the widget's update function
func (w *BaseWidget) Update() error {
	if w.updateFunc != nil {
		return w.updateFunc()
	}
	return nil
}

// Configure calls the widget's configuration function
func (w *BaseWidget) Configure() error {
	if w.configFunc != nil {
		return w.configFunc()
	}
	return nil
}

// MinSize returns the minimum widget size
func (w *BaseWidget) MinSize() (int, int) {
	return w.minWidth, w.minHeight
}

// MaxSize returns the maximum widget size
func (w *BaseWidget) MaxSize() (int, int) {
	return w.maxWidth, w.maxHeight
}

// OnResize handles widget resizing
func (w *BaseWidget) OnResize(width, height int) {
	// Default implementation - can be overridden
}

// OnFocus handles widget focus changes
func (w *BaseWidget) OnFocus(focus bool) {
	w.focused = focus
}

// ViewWidget wraps a view as a widget
type ViewWidget struct {
	*BaseWidget
	view views.View
}

// NewViewWidget creates a new view widget
func NewViewWidget(id, name string, view views.View) *ViewWidget {
	widget := &ViewWidget{
		BaseWidget: &BaseWidget{
			id:          id,
			name:        name,
			description: fmt.Sprintf("%s view widget", name),
			widgetType:  WidgetTypeView,
			primitive:   view.Render(),
			minWidth:    20,
			minHeight:   10,
			maxWidth:    0, // No max
			maxHeight:   0, // No max
		},
		view: view,
	}

	widget.updateFunc = func() error {
		return view.Refresh()
	}

	return widget
}

// MetricsWidget displays system metrics
type MetricsWidget struct {
	*BaseWidget
	client      dao.SlurmClient
	textView    *tview.TextView
	updateTimer *time.Ticker
}

// NewMetricsWidget creates a new metrics widget
func NewMetricsWidget(id string, client dao.SlurmClient) *MetricsWidget {
	textView := tview.NewTextView()
	textView.SetDynamicColors(true)
	textView.SetBorder(true)
	textView.SetTitle(" Cluster Metrics ")
	textView.SetTitleAlign(tview.AlignCenter)

	widget := &MetricsWidget{
		BaseWidget: &BaseWidget{
			id:          id,
			name:        "Cluster Metrics",
			description: "Real-time cluster resource metrics",
			widgetType:  WidgetTypeMetrics,
			primitive:   textView,
			minWidth:    25,
			minHeight:   8,
			maxWidth:    60,
			maxHeight:   20,
		},
		client:   client,
		textView: textView,
	}

	widget.updateFunc = widget.updateMetrics
	widget.startAutoUpdate()

	return widget
}

// updateMetrics refreshes the metrics display
func (w *MetricsWidget) updateMetrics() error {
	if w.client.Info() == nil {
		w.textView.SetText("[red]No cluster info available[white]")
		return nil
	}

	stats, err := w.client.Info().GetStats()
	if err != nil {
		w.textView.SetText(fmt.Sprintf("[red]Error: %v[white]", err))
		return err
	}

	content := fmt.Sprintf(`[yellow]CPU Usage:[white] %.1f%%
[yellow]Memory:[white] %.1f%%
[yellow]Jobs:[white] %d running, %d pending
[yellow]Nodes:[white] %d total, %d active

[green]Updated:[white] %s`,
		stats.CPUUsage,
		stats.MemoryUsage,
		stats.RunningJobs, stats.PendingJobs,
		stats.TotalNodes, stats.ActiveNodes,
		time.Now().Format("15:04:05"))

	w.textView.SetText(content)
	return nil
}

// startAutoUpdate starts automatic metric updates
func (w *MetricsWidget) startAutoUpdate() {
	w.updateTimer = time.NewTicker(5 * time.Second)
	go func() {
		for range w.updateTimer.C {
			_ = w.updateMetrics()
		}
	}()
}

// StatusWidget displays cluster status
type StatusWidget struct {
	*BaseWidget
	client    dao.SlurmClient
	statusBar *components.StatusBar
}

// NewStatusWidget creates a new status widget
func NewStatusWidget(id string, client dao.SlurmClient) *StatusWidget {
	statusBar := components.NewStatusBar()

	widget := &StatusWidget{
		BaseWidget: &BaseWidget{
			id:          id,
			name:        "Cluster Status",
			description: "Current cluster status and information",
			widgetType:  WidgetTypeStatus,
			primitive:   statusBar,
			minWidth:    30,
			minHeight:   1,
			maxWidth:    0, // No max
			maxHeight:   3,
		},
		client:    client,
		statusBar: statusBar,
	}

	widget.updateFunc = widget.updateStatus
	return widget
}

// updateStatus refreshes the status display
func (w *StatusWidget) updateStatus() error {
	clusterInfo, err := w.client.ClusterInfo()
	if err != nil {
		w.statusBar.Error(fmt.Sprintf("Cluster: %v", err))
		return err
	}

	w.statusBar.Info(fmt.Sprintf("Cluster: %s | Version: %s | Endpoint: %s",
		clusterInfo.Name, clusterInfo.Version, clusterInfo.Endpoint))

	return nil
}

// AlertsWidget displays system alerts
type AlertsWidget struct {
	*BaseWidget
	alertsManager *components.AlertsManager
	listView      *tview.List
}

// NewAlertsWidget creates a new alerts widget
func NewAlertsWidget(id string, alertsManager *components.AlertsManager) *AlertsWidget {
	listView := tview.NewList()
	listView.SetBorder(true)
	listView.SetTitle(" Active Alerts ")
	listView.SetTitleAlign(tview.AlignCenter)

	widget := &AlertsWidget{
		BaseWidget: &BaseWidget{
			id:          id,
			name:        "System Alerts",
			description: "Current system alerts and notifications",
			widgetType:  WidgetTypeAlerts,
			primitive:   listView,
			minWidth:    25,
			minHeight:   6,
			maxWidth:    80,
			maxHeight:   25,
		},
		alertsManager: alertsManager,
		listView:      listView,
	}

	widget.updateFunc = widget.updateAlerts
	return widget
}

// updateAlerts refreshes the alerts display
func (w *AlertsWidget) updateAlerts() error {
	w.listView.Clear()

	alerts := w.alertsManager.GetUnacknowledgedAlerts()
	if len(alerts) == 0 {
		w.listView.AddItem("No active alerts", "", 0, nil)
		return nil
	}

	for i, alert := range alerts {
		if i >= 10 { // Limit to 10 alerts in widget
			break
		}

		var color string
		switch alert.Level {
		case components.AlertError:
			color = "[red]"
		case components.AlertWarning:
			color = "[yellow]"
		case components.AlertInfo:
			color = "[blue]"
		default:
			color = "[white]"
		}

		text := fmt.Sprintf("%s%s: %s", color, alert.Title, alert.Message)
		w.listView.AddItem(text, "", 0, nil)
	}

	return nil
}

// ClockWidget displays current time
type ClockWidget struct {
	*BaseWidget
	textView    *tview.TextView
	updateTimer *time.Ticker
}

// NewClockWidget creates a new clock widget
func NewClockWidget(id string) *ClockWidget {
	textView := tview.NewTextView()
	textView.SetTextAlign(tview.AlignCenter)
	textView.SetBorder(true)
	textView.SetTitle(" Time ")
	textView.SetTitleAlign(tview.AlignCenter)

	widget := &ClockWidget{
		BaseWidget: &BaseWidget{
			id:          id,
			name:        "Clock",
			description: "Current date and time",
			widgetType:  WidgetTypeClock,
			primitive:   textView,
			minWidth:    20,
			minHeight:   5,
			maxWidth:    30,
			maxHeight:   8,
		},
		textView: textView,
	}

	widget.updateFunc = widget.updateTime
	widget.startAutoUpdate()

	return widget
}

// updateTime refreshes the time display
func (w *ClockWidget) updateTime() error {
	now := time.Now()
	content := fmt.Sprintf(`[yellow]%s[white]

[green]%s[white]
[blue]%s[white]`,
		now.Format("15:04:05"),
		now.Format("Monday"),
		now.Format("Jan 2, 2006"))

	w.textView.SetText(content)
	return nil
}

// startAutoUpdate starts automatic time updates
func (w *ClockWidget) startAutoUpdate() {
	w.updateTimer = time.NewTicker(1 * time.Second)
	go func() {
		for range w.updateTimer.C {
			_ = w.updateTime()
		}
	}()
}

// QuickStartWidget provides quick action buttons
type QuickStartWidget struct {
	*BaseWidget
	listView *tview.List
	actions  []QuickAction
}

// QuickAction represents a quick action
type QuickAction struct {
	Name        string
	Description string
	Shortcut    string
	Action      func()
}

// NewQuickStartWidget creates a new quick start widget
func NewQuickStartWidget(id string) *QuickStartWidget {
	listView := tview.NewList()
	listView.SetBorder(true)
	listView.SetTitle(" Quick Actions ")
	listView.SetTitleAlign(tview.AlignCenter)

	widget := &QuickStartWidget{
		BaseWidget: &BaseWidget{
			id:          id,
			name:        "Quick Actions",
			description: "Quick access to common actions",
			widgetType:  WidgetTypeQuickStart,
			primitive:   listView,
			minWidth:    25,
			minHeight:   8,
			maxWidth:    40,
			maxHeight:   15,
		},
		listView: listView,
		actions:  []QuickAction{},
	}

	widget.updateFunc = widget.updateActions
	widget.initializeActions()

	return widget
}

// initializeActions sets up default quick actions
func (w *QuickStartWidget) initializeActions() {
	w.actions = []QuickAction{
		{
			Name:        "Submit Job",
			Description: "Open job submission wizard",
			Shortcut:    "F8",
			Action:      func() { /* TODO: Implement job submission */ },
		},
		{
			Name:        "View Jobs",
			Description: "Switch to jobs view",
			Shortcut:    "1",
			Action:      func() { /* TODO: Implement view switching */ },
		},
		{
			Name:        "View Nodes",
			Description: "Switch to nodes view",
			Shortcut:    "2",
			Action:      func() { /* TODO: Implement view switching */ },
		},
		{
			Name:        "System Health",
			Description: "Open health monitoring",
			Shortcut:    "8",
			Action:      func() { /* TODO: Implement health view */ },
		},
		{
			Name:        "Preferences",
			Description: "Open user preferences",
			Shortcut:    "F3",
			Action:      func() { /* TODO: Implement preferences */ },
		},
		{
			Name:        "Help",
			Description: "Show help information",
			Shortcut:    "F1",
			Action:      func() { /* TODO: Implement help */ },
		},
	}
}

// updateActions refreshes the actions display
func (w *QuickStartWidget) updateActions() error {
	w.listView.Clear()

	for i, action := range w.actions {
		text := fmt.Sprintf("[yellow]%s[white] (%s)", action.Name, action.Shortcut)
		w.listView.AddItem(text, action.Description, rune('1'+i), action.Action)
	}

	return nil
}

// AddAction adds a new quick action
func (w *QuickStartWidget) AddAction(action QuickAction) {
	w.actions = append(w.actions, action)
	_ = w.updateActions()
}

// TerminalWidget provides a terminal interface
type TerminalWidget struct {
	*BaseWidget
	inputField *tview.InputField
	outputView *tview.TextView
	flex       *tview.Flex
	history    []string
}

// NewTerminalWidget creates a new terminal widget
func NewTerminalWidget(id string) *TerminalWidget {
	inputField := tview.NewInputField()
	inputField.SetLabel("$ ")
	inputField.SetFieldBackgroundColor(tcell.ColorBlack)

	outputView := tview.NewTextView()
	outputView.SetDynamicColors(true)
	outputView.SetScrollable(true)
	outputView.SetWrap(true)

	flex := tview.NewFlex()
	flex.SetDirection(tview.FlexRow)
	flex.AddItem(outputView, 0, 1, false)
	flex.AddItem(inputField, 1, 0, true)
	flex.SetBorder(true)
	flex.SetTitle(" Terminal ")
	flex.SetTitleAlign(tview.AlignCenter)

	widget := &TerminalWidget{
		BaseWidget: &BaseWidget{
			id:          id,
			name:        "Terminal",
			description: "Command terminal interface",
			widgetType:  WidgetTypeTerminal,
			primitive:   flex,
			minWidth:    30,
			minHeight:   8,
			maxWidth:    0, // No max
			maxHeight:   0, // No max
		},
		inputField: inputField,
		outputView: outputView,
		flex:       flex,
		history:    []string{},
	}

	widget.setupTerminal()
	return widget
}

// setupTerminal configures the terminal widget
func (w *TerminalWidget) setupTerminal() {
	w.inputField.SetDoneFunc(func(key tcell.Key) {
		if key == tcell.KeyEnter {
			command := w.inputField.GetText()
			if command != "" {
				w.executeCommand(command)
				w.history = append(w.history, command)
				w.inputField.SetText("")
			}
		}
	})

	// Initial message
	w.outputView.SetText("[green]S9S Terminal Widget[white]\nEnter commands below...\n\n")
}

// executeCommand processes a terminal command
func (w *TerminalWidget) executeCommand(command string) {
	output := fmt.Sprintf("[yellow]$ %s[white]\n", command)

	switch command {
	case "clear":
		w.outputView.SetText("")
		return
	case "help":
		output += "[blue]Available commands:[white]\n"
		output += "  clear  - Clear terminal\n"
		output += "  help   - Show this help\n"
		output += "  sinfo  - Show cluster info\n"
		output += "  squeue - Show job queue\n"
	case "sinfo":
		output += "[green]Mock cluster information[white]\n"
		output += "PARTITION AVAIL  TIMELIMIT  NODES  STATE NODELIST\n"
		output += "compute*     up   infinite     10   idle node[001-010]\n"
	case "squeue":
		output += "[green]Mock job queue[white]\n"
		output += "JOBID PARTITION     NAME     USER ST       TIME  NODES NODELIST\n"
		output += " 1234   compute test_job  user01  R      12:34      2 node[001-002]\n"
	default:
		output += fmt.Sprintf("[red]Command not found: %s[white]\n", command)
		output += "Type 'help' for available commands.\n"
	}

	// Append to existing content
	current := w.outputView.GetText(false)
	w.outputView.SetText(current + output + "\n")

	// Scroll to bottom
	w.outputView.ScrollToEnd()
}

// LogsWidget displays system logs
type LogsWidget struct {
	*BaseWidget
	textView *tview.TextView
	logs     []string
}

// NewLogsWidget creates a new logs widget
func NewLogsWidget(id string) *LogsWidget {
	textView := tview.NewTextView()
	textView.SetDynamicColors(true)
	textView.SetScrollable(true)
	textView.SetWrap(true)
	textView.SetBorder(true)
	textView.SetTitle(" System Logs ")
	textView.SetTitleAlign(tview.AlignCenter)

	widget := &LogsWidget{
		BaseWidget: &BaseWidget{
			id:          id,
			name:        "System Logs",
			description: "Recent system log entries",
			widgetType:  WidgetTypeLogs,
			primitive:   textView,
			minWidth:    40,
			minHeight:   10,
			maxWidth:    0, // No max
			maxHeight:   0, // No max
		},
		textView: textView,
		logs:     []string{},
	}

	widget.updateFunc = widget.updateLogs
	widget.initializeMockLogs()

	return widget
}

// initializeMockLogs adds some mock log entries
func (w *LogsWidget) initializeMockLogs() {
	now := time.Now()
	w.logs = []string{
		fmt.Sprintf("[%s] [INFO] SLURM daemon started", now.Add(-10*time.Minute).Format("15:04:05")),
		fmt.Sprintf("[%s] [INFO] Job 1234 submitted by user01", now.Add(-8*time.Minute).Format("15:04:05")),
		fmt.Sprintf("[%s] [INFO] Job 1234 started on node001-002", now.Add(-7*time.Minute).Format("15:04:05")),
		fmt.Sprintf("[%s] [WARN] Node003 high memory usage (95%%)", now.Add(-5*time.Minute).Format("15:04:05")),
		fmt.Sprintf("[%s] [INFO] Job 1235 submitted by user02", now.Add(-3*time.Minute).Format("15:04:05")),
		fmt.Sprintf("[%s] [INFO] Backup completed successfully", now.Add(-1*time.Minute).Format("15:04:05")),
	}
}

// updateLogs refreshes the logs display
func (w *LogsWidget) updateLogs() error {
	content := ""
	for _, log := range w.logs {
		if len(content) > 0 {
			content += "\n"
		}

		// Color code based on log level
		if contains(log, "[ERROR]") {
			content += "[red]" + log + "[white]"
		} else if contains(log, "[WARN]") {
			content += "[yellow]" + log + "[white]"
		} else if contains(log, "[INFO]") {
			content += "[green]" + log + "[white]"
		} else {
			content += log
		}
	}

	w.textView.SetText(content)
	w.textView.ScrollToEnd()
	return nil
}

// AddLog adds a new log entry
func (w *LogsWidget) AddLog(message string) {
	timestamp := time.Now().Format("15:04:05")
	logEntry := fmt.Sprintf("[%s] %s", timestamp, message)

	w.logs = append(w.logs, logEntry)

	// Keep only last 100 logs
	if len(w.logs) > 100 {
		w.logs = w.logs[len(w.logs)-100:]
	}

	_ = w.updateLogs()
}

// Helper function to check if string contains substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && s[len(s)-len(substr):] == substr ||
		len(s) > len(substr) && findInString(s, substr)
}

func findInString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
