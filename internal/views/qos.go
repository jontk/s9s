package views

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/jontk/s9s/internal/dao"
	"github.com/jontk/s9s/internal/ui/components"
)

// QoSView displays the QoS (Quality of Service) list
type QoSView struct {
	*BaseView
	client       dao.SlurmClient
	table        *components.Table
	qosList      []*dao.QoS
	mu           sync.RWMutex
	refreshTimer *time.Timer
	refreshRate  time.Duration
	filter       string
	container    *tview.Flex
	filterInput  *tview.InputField
	statusBar    *tview.TextView
	app          *tview.Application
	pages        *tview.Pages
}

// SetPages sets the pages reference for modal handling
func (v *QoSView) SetPages(pages *tview.Pages) {
	v.pages = pages
}

// NewQoSView creates a new QoS view
func NewQoSView(client dao.SlurmClient) *QoSView {
	v := &QoSView{
		BaseView:    NewBaseView("qos", "QoS"),
		client:      client,
		refreshRate: 30 * time.Second,
		qosList:     []*dao.QoS{},
	}

	// Create table with QoS columns
	columns := []components.Column{
		components.NewColumn("Name").Width(20).Build(),
		components.NewColumn("Priority").Width(10).Align(tview.AlignRight).Sortable(true).Build(),
		components.NewColumn("Preempt Mode").Width(15).Build(),
		components.NewColumn("Max Jobs/User").Width(13).Align(tview.AlignRight).Build(),
		components.NewColumn("Max Submit/User").Width(15).Align(tview.AlignRight).Build(),
		components.NewColumn("Max CPUs/User").Width(13).Align(tview.AlignRight).Build(),
		components.NewColumn("Max Nodes/User").Width(14).Align(tview.AlignRight).Build(),
		components.NewColumn("Max Wall Time").Width(13).Align(tview.AlignRight).Build(),
		components.NewColumn("Grace Time").Width(12).Align(tview.AlignRight).Build(),
	}

	v.table = components.NewTableBuilder().
		WithColumns(columns...).
		WithSelectable(true).
		WithHeader(true).
		WithColors(tcell.ColorYellow, tcell.ColorTeal, tcell.ColorWhite).
		Build()

	// Set up callbacks
	v.table.SetOnSelect(v.onQoSSelect)
	v.table.SetOnSort(v.onSort)

	// Create filter input
	v.filterInput = tview.NewInputField().
		SetLabel("Filter: ").
		SetFieldWidth(30).
		SetChangedFunc(v.onFilterChange).
		SetDoneFunc(v.onFilterDone)

	// Create status bar
	v.statusBar = tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignLeft)

	// Create container layout
	v.container = tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(v.filterInput, 1, 0, false).
		AddItem(v.table.Table, 0, 1, true).
		AddItem(v.statusBar, 1, 0, false)

	return v
}

// Init initializes the QoS view
func (v *QoSView) Init(ctx context.Context) error {
	v.BaseView.Init(ctx)
	// Don't refresh on init - let it happen when view is shown
	return nil
}

// Render returns the view's main component
func (v *QoSView) Render() tview.Primitive {
	return v.container
}

// Refresh updates the QoS data
func (v *QoSView) Refresh() error {
	v.SetRefreshing(true)
	defer v.SetRefreshing(false)

	// Fetch QoS from backend
	qosList, err := v.client.QoS().List()
	if err != nil {
		v.SetLastError(err)
		v.updateStatusBar(fmt.Sprintf("[red]Error: %v[white]", err))
		return err
	}

	v.mu.Lock()
	v.qosList = qosList.QoS
	v.mu.Unlock()

	// Update table
	v.updateTable()
	v.updateStatusBar("")

	// Schedule next refresh
	v.scheduleRefresh()

	return nil
}

// Stop stops the view
func (v *QoSView) Stop() error {
	if v.refreshTimer != nil {
		v.refreshTimer.Stop()
	}
	return nil
}

// Hints returns keyboard hints
func (v *QoSView) Hints() []string {
	return []string{
		"[yellow]Enter[white] Details",
		"[yellow]/[white] Filter",
		"[yellow]1-9[white] Sort",
		"[yellow]R[white] Refresh",
	}
}

// OnKey handles keyboard events
func (v *QoSView) OnKey(event *tcell.EventKey) *tcell.EventKey {
	switch event.Key() {
	case tcell.KeyRune:
		switch event.Rune() {
		case 'R':
			go v.Refresh()
			return nil
		case '/':
			v.app.SetFocus(v.filterInput)
			return nil
		}
	case tcell.KeyEnter:
		v.showQoSDetails()
		return nil
	case tcell.KeyEsc:
		if v.filterInput.HasFocus() {
			v.app.SetFocus(v.table.Table)
			return nil
		}
	}

	return event
}

// OnFocus handles focus events
func (v *QoSView) OnFocus() error {
	if v.app != nil {
		v.app.SetFocus(v.table.Table)
	}
	// Refresh when gaining focus if we haven't loaded data yet
	if len(v.qosList) == 0 && !v.IsRefreshing() {
		go v.Refresh()
	}
	return nil
}

// OnLoseFocus handles loss of focus
func (v *QoSView) OnLoseFocus() error {
	return nil
}

// updateTable updates the table with current QoS data
func (v *QoSView) updateTable() {
	v.mu.RLock()
	defer v.mu.RUnlock()

	data := make([][]string, len(v.qosList))
	for i, qos := range v.qosList {
		// Format priority with color
		var priorityColor string
		if qos.Priority > 1000 {
			priorityColor = "green"
		} else if qos.Priority > 100 {
			priorityColor = "yellow"
		} else {
			priorityColor = "white"
		}
		priority := fmt.Sprintf("[%s]%d[white]", priorityColor, qos.Priority)

		// Format limits
		maxJobs := formatQoSLimit(qos.MaxJobsPerUser)
		maxSubmit := formatQoSLimit(qos.MaxSubmitJobsPerUser)
		maxCPUs := formatQoSLimit(qos.MaxCPUsPerUser)
		maxNodes := formatQoSLimit(qos.MaxNodesPerUser)

		// Format times
		maxWallTime := formatQoSTimeLimit(qos.MaxWallTime)
		graceTime := formatQoSTimeLimit(qos.GraceTime)

		data[i] = []string{
			qos.Name,
			priority,
			qos.PreemptMode,
			maxJobs,
			maxSubmit,
			maxCPUs,
			maxNodes,
			maxWallTime,
			graceTime,
		}
	}

	v.table.SetData(data)
}

// formatQoSLimit formats a limit value (0 or -1 means unlimited)
func formatQoSLimit(limit int) string {
	if limit <= 0 {
		return "unlimited"
	}
	return fmt.Sprintf("%d", limit)
}

// formatQoSTimeLimit formats a time limit in minutes
func formatQoSTimeLimit(minutes int) string {
	if minutes <= 0 {
		return "unlimited"
	}

	days := minutes / (24 * 60)
	hours := (minutes % (24 * 60)) / 60
	mins := minutes % 60

	if days > 0 {
		return fmt.Sprintf("%dd %dh", days, hours)
	} else if hours > 0 {
		return fmt.Sprintf("%dh %dm", hours, mins)
	}
	return fmt.Sprintf("%dm", mins)
}

// updateStatusBar updates the status bar
func (v *QoSView) updateStatusBar(message string) {
	if message != "" {
		v.statusBar.SetText(message)
		return
	}

	v.mu.RLock()
	total := len(v.qosList)
	v.mu.RUnlock()

	filtered := len(v.table.GetFilteredData())

	status := fmt.Sprintf("Total QoS: %d", total)

	if filtered < total {
		status += fmt.Sprintf(" | Filtered: %d", filtered)
	}

	if v.IsRefreshing() {
		status += " | [yellow]Refreshing...[white]"
	}

	v.statusBar.SetText(status)
}

// scheduleRefresh schedules the next refresh
func (v *QoSView) scheduleRefresh() {
	// Remove automatic refresh scheduling to prevent memory leak
	// Refresh will be handled by the main app refresh timer
}

// onQoSSelect handles QoS selection
func (v *QoSView) onQoSSelect(row, col int) {
	data := v.table.GetSelectedData()
	if data == nil || len(data) == 0 {
		return
	}

	qosName := data[0]
	v.updateStatusBar(fmt.Sprintf("Selected QoS: %s", qosName))
}

// onSort handles column sorting
func (v *QoSView) onSort(col int, ascending bool) {
	v.updateStatusBar(fmt.Sprintf("Sorted by column %d", col+1))
}

// onFilterChange handles filter input changes
func (v *QoSView) onFilterChange(text string) {
	v.filter = text
	v.table.SetFilter(text)
	v.updateStatusBar("")
}

// onFilterDone handles filter input completion
func (v *QoSView) onFilterDone(key tcell.Key) {
	if v.app != nil {
		v.app.SetFocus(v.table.Table)
	}
}

// showQoSDetails shows detailed information for the selected QoS
func (v *QoSView) showQoSDetails() {
	data := v.table.GetSelectedData()
	if data == nil || len(data) == 0 {
		return
	}

	qosName := data[0]

	// Find the full QoS object
	var qos *dao.QoS
	v.mu.RLock()
	for _, q := range v.qosList {
		if q.Name == qosName {
			qos = q
			break
		}
	}
	v.mu.RUnlock()

	if qos == nil {
		v.updateStatusBar(fmt.Sprintf("[red]QoS %s not found[white]", qosName))
		return
	}

	// Create details view
	details := v.formatQoSDetails(qos)

	textView := tview.NewTextView().
		SetDynamicColors(true).
		SetText(details).
		SetScrollable(true)

	modal := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(textView, 0, 1, true).
		AddItem(tview.NewTextView().SetText("Press ESC to close"), 1, 0, false)

	modal.SetBorder(true).
		SetTitle(fmt.Sprintf(" QoS %s Details ", qosName)).
		SetTitleAlign(tview.AlignCenter)

	// Create centered modal layout
	centeredModal := tview.NewFlex().
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(nil, 0, 1, false).
			AddItem(modal, 0, 8, true).
			AddItem(nil, 0, 1, false), 0, 8, true).
		AddItem(nil, 0, 1, false)

	// Handle ESC key
	modal.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEsc {
			if v.pages != nil {
				v.pages.RemovePage("qos-details")
			}
			return nil
		}
		return event
	})

	if v.pages != nil {
		v.pages.AddPage("qos-details", centeredModal, true, true)
	}
}

// formatQoSDetails formats QoS details for display
func (v *QoSView) formatQoSDetails(qos *dao.QoS) string {
	var details strings.Builder

	details.WriteString(fmt.Sprintf("[yellow]QoS Name:[white] %s\n", qos.Name))
	details.WriteString(fmt.Sprintf("[yellow]Priority:[white] %d\n", qos.Priority))
	details.WriteString(fmt.Sprintf("[yellow]Preempt Mode:[white] %s\n", qos.PreemptMode))

	// Job limits
	details.WriteString("\n[teal]Job Limits:[white]\n")
	details.WriteString(fmt.Sprintf("[yellow]  Max Jobs per User:[white] %s\n", formatQoSLimit(qos.MaxJobsPerUser)))
	details.WriteString(fmt.Sprintf("[yellow]  Max Submit Jobs per User:[white] %s\n", formatQoSLimit(qos.MaxSubmitJobsPerUser)))
	details.WriteString(fmt.Sprintf("[yellow]  Max Jobs per Account:[white] %s\n", formatQoSLimit(qos.MaxJobsPerAccount)))

	// Resource limits
	details.WriteString("\n[teal]Resource Limits per User:[white]\n")
	details.WriteString(fmt.Sprintf("[yellow]  Max CPUs:[white] %s\n", formatQoSLimit(qos.MaxCPUsPerUser)))
	details.WriteString(fmt.Sprintf("[yellow]  Max Nodes:[white] %s\n", formatQoSLimit(qos.MaxNodesPerUser)))
	details.WriteString(fmt.Sprintf("[yellow]  Max Memory:[white] %s\n", formatMemoryLimit(qos.MaxMemoryPerUser)))

	// Time limits
	details.WriteString("\n[teal]Time Limits:[white]\n")
	details.WriteString(fmt.Sprintf("[yellow]  Max Wall Time:[white] %s\n", formatQoSTimeLimit(qos.MaxWallTime)))
	details.WriteString(fmt.Sprintf("[yellow]  Grace Time:[white] %s\n", formatQoSTimeLimit(qos.GraceTime)))

	// Flags
	if len(qos.Flags) > 0 {
		details.WriteString(fmt.Sprintf("\n[yellow]Flags:[white] %s\n", strings.Join(qos.Flags, ", ")))
	}

	return details.String()
}

// formatMemoryLimit formats a memory limit in MB
func formatMemoryLimit(mb int64) string {
	if mb <= 0 {
		return "unlimited"
	}

	if mb < 1024 {
		return fmt.Sprintf("%d MB", mb)
	} else if mb < 1024*1024 {
		return fmt.Sprintf("%.1f GB", float64(mb)/1024)
	} else {
		return fmt.Sprintf("%.1f TB", float64(mb)/(1024*1024))
	}
}