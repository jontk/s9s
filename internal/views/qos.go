package views

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/jontk/s9s/internal/dao"
	"github.com/jontk/s9s/internal/debug"
	"github.com/jontk/s9s/internal/ui/components"
	"github.com/jontk/s9s/internal/ui/filters"
	"github.com/rivo/tview"
)

// QoSView displays the QoS (Quality of Service) list
type QoSView struct {
	*BaseView
	client         dao.SlurmClient
	table          *components.Table
	qosList        []*dao.QoS
	mu             sync.RWMutex
	refreshTimer   *time.Timer
	refreshRate    time.Duration
	filter         string
	container      *tview.Flex
	filterInput    *tview.InputField
	statusBar      *tview.TextView
	app            *tview.Application
	pages          *tview.Pages
	filterBar      *components.FilterBar
	advancedFilter *filters.Filter
	isAdvancedMode bool
	globalSearch   *GlobalSearch
	loadingManager *components.LoadingManager
	loadingWrapper *components.LoadingWrapper
}

// SetPages sets the pages reference for modal handling
func (v *QoSView) SetPages(pages *tview.Pages) {
	v.pages = pages
	// Set pages for filter bar if it exists
	if v.filterBar != nil {
		v.filterBar.SetPages(pages)
	}
}

// SetApp sets the application reference
func (v *QoSView) SetApp(app *tview.Application) {
	v.app = app

	// Initialize loading manager
	if v.pages != nil {
		v.loadingManager = components.NewLoadingManager(app, v.pages)
		v.loadingWrapper = components.NewLoadingWrapper(v.loadingManager, "qos")
	}

	// Create filter bar now that we have app reference
	v.filterBar = components.NewFilterBar("qos", app)
	v.filterBar.SetPages(v.pages)
	v.filterBar.SetOnFilterChange(v.onAdvancedFilterChange)
	v.filterBar.SetOnClose(v.closeAdvancedFilter)

	// Create global search
	v.globalSearch = NewGlobalSearch(v.client, app)
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

	// Create container layout (removed individual status bar to prevent conflicts with main status bar)
	v.container = tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(v.filterInput, 1, 0, false).
		AddItem(v.table, 0, 1, true)

	return v
}

// Init initializes the QoS view
func (v *QoSView) Init(ctx context.Context) error {
	_ = v.BaseView.Init(ctx)
	return v.Refresh()
}

// Render returns the view's main component
func (v *QoSView) Render() tview.Primitive {
	return v.container
}

// Refresh updates the QoS data
func (v *QoSView) Refresh() error {
	v.SetRefreshing(true)
	defer v.SetRefreshing(false)

	// Show loading indicator for operations that might take time
	if v.loadingWrapper != nil {
		return v.loadingWrapper.WithLoading("Loading QoS...", func() error {
			return v.refreshInternal()
		})
	}

	return v.refreshInternal()
}

// refreshInternal performs the actual refresh operation
func (v *QoSView) refreshInternal() error {
	debug.Logger.Printf("QoS refreshInternal() started at %s", time.Now().Format("15:04:05.000"))
	// Fetch QoS from backend
	qosList, err := v.client.QoS().List()
	debug.Logger.Printf("QoS client.List() finished at %s", time.Now().Format("15:04:05.000"))
	if err != nil {
		v.SetLastError(err)
		// Note: Error handling removed since individual view status bars are no longer used
		return err
	}

	v.mu.Lock()
	v.qosList = qosList.QoS
	v.mu.Unlock()
	debug.Logger.Printf("QoS data stored, calling updateTable() at %s", time.Now().Format("15:04:05.000"))

	// Update table
	v.updateTable()
	debug.Logger.Printf("QoS updateTable() finished at %s", time.Now().Format("15:04:05.000"))
	// Note: No longer updating individual view status bar since we use main app status bar for hints

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
	hints := []string{
		"[yellow]Enter[white] Details",
		"[yellow]/[white] Filter",
		"[yellow]F3[white] Adv Filter",
		"[yellow]Ctrl+F[white] Search",
		"[yellow]1-9[white] Sort",
		"[yellow]R[white] Refresh",
	}

	if v.isAdvancedMode {
		hints = append([]string{"[yellow]ESC[white] Exit Adv Filter"}, hints...)
	}

	return hints
}

// OnKey handles keyboard events
func (v *QoSView) OnKey(event *tcell.EventKey) *tcell.EventKey {
	// Check if a modal is open - if so, don't process view shortcuts
	if v.pages != nil && v.pages.GetPageCount() > 1 {
		return event // Let modal handle it
	}

	// Handle advanced filter mode
	if v.isAdvancedMode && event.Key() == tcell.KeyEsc {
		v.closeAdvancedFilter()
		return nil
	}

	if handler, ok := v.qosKeyHandlers()[event.Key()]; ok {
		handler()
		return nil
	}

	if event.Key() == tcell.KeyRune {
		if handler, ok := v.qosRuneHandlers()[event.Rune()]; ok {
			handler()
			return nil
		}
	}

	if event.Key() == tcell.KeyEsc && v.filterInput.HasFocus() {
		v.app.SetFocus(v.table.Table)
		return nil
	}

	return event
}

// qosKeyHandlers returns a map of function key handlers
func (v *QoSView) qosKeyHandlers() map[tcell.Key]func() {
	return map[tcell.Key]func(){
		tcell.KeyF3:    v.showAdvancedFilter,
		tcell.KeyCtrlF: v.showGlobalSearch,
		tcell.KeyEnter: v.showQoSDetails,
	}
}

// qosRuneHandlers returns a map of rune handlers
func (v *QoSView) qosRuneHandlers() map[rune]func() {
	return map[rune]func(){
		'R': func() { go func() { _ = v.Refresh() }() },
		'/': func() { v.app.SetFocus(v.filterInput) },
	}
}

// OnFocus handles focus events
func (v *QoSView) OnFocus() error {
	if v.app != nil {
		v.app.SetFocus(v.table.Table)
	}
	// Refresh when gaining focus if we haven't loaded data yet
	if len(v.qosList) == 0 && !v.IsRefreshing() {
		go func() { _ = v.Refresh() }()
	}
	return nil
}

// OnLoseFocus handles loss of focus
func (v *QoSView) OnLoseFocus() error {
	return nil
}

// updateTable updates the table with current QoS data
func (v *QoSView) updateTable() {
	debug.Logger.Printf("QoS updateTable() starting with %d items at %s", len(v.qosList), time.Now().Format("15:04:05.000"))
	v.mu.RLock()
	defer v.mu.RUnlock()

	// Apply advanced filter if active
	filteredQoS := v.qosList
	if v.advancedFilter != nil && len(v.advancedFilter.Expressions) > 0 {
		filteredQoS = v.applyAdvancedFilter(v.qosList)
	}
	debug.Logger.Printf("QoS filtering done, processing %d items at %s", len(filteredQoS), time.Now().Format("15:04:05.000"))

	data := make([][]string, len(filteredQoS))
	for i, qos := range filteredQoS {
		// Format priority with color
		var priorityColor string
		switch {
		case qos.Priority > 1000:
			priorityColor = "green"
		case qos.Priority > 100:
			priorityColor = "yellow"
		default:
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
	debug.Logger.Printf("QoS data processing done, calling table.SetData() at %s", time.Now().Format("15:04:05.000"))

	v.table.SetData(data)
	debug.Logger.Printf("QoS table.SetData() finished at %s", time.Now().Format("15:04:05.000"))
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

/*
TODO(lint): Review unused code - func (*QoSView).updateStatusBar is unused

updateStatusBar updates the status bar
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
*/

// scheduleRefresh schedules the next refresh
func (v *QoSView) scheduleRefresh() {
	// Remove automatic refresh scheduling to prevent memory leak
	// Refresh will be handled by the main app refresh timer
}

// onQoSSelect handles QoS selection
func (v *QoSView) onQoSSelect(_, _ int) {
	data := v.table.GetSelectedData()
	if len(data) == 0 {
		return
	}

	// Note: Selection handling removed since individual view status bars are no longer used
	_ = data[0] // qosName no longer used
}

// onSort handles column sorting
func (v *QoSView) onSort(_ int, _ bool) {
	// Note: Sort feedback removed since individual view status bars are no longer used
}

// onFilterChange handles filter input changes
func (v *QoSView) onFilterChange(text string) {
	v.filter = text
	v.table.SetFilter(text)
	// Note: Status bar update removed since individual view status bars are no longer used
}

// onFilterDone handles filter input completion
func (v *QoSView) onFilterDone(_ tcell.Key) {
	if v.app != nil {
		v.app.SetFocus(v.table.Table)
	}
}

// showQoSDetails shows detailed information for the selected QoS
func (v *QoSView) showQoSDetails() {
	data := v.table.GetSelectedData()
	if len(data) == 0 {
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
		// Note: Error message removed since individual view status bars are no longer used
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

	switch {
	case mb < 1024:
		return fmt.Sprintf("%d MB", mb)
	case mb < 1024*1024:
		return fmt.Sprintf("%.1f GB", float64(mb)/1024)
	default:
		return fmt.Sprintf("%.1f TB", float64(mb)/(1024*1024))
	}
}

// showAdvancedFilter shows the advanced filter bar
func (v *QoSView) showAdvancedFilter() {
	if v.filterBar == nil || v.pages == nil {
		return
	}

	v.isAdvancedMode = true

	// Replace the simple filter with advanced filter bar
	v.container.Clear()
	v.container.
		AddItem(v.filterBar, 5, 0, true).
		AddItem(v.table, 0, 1, false)

	v.filterBar.Show()
	// Note: Advanced filter status removed since individual view status bars are no longer used
}

// closeAdvancedFilter closes the advanced filter bar
func (v *QoSView) closeAdvancedFilter() {
	v.isAdvancedMode = false

	// Restore the simple filter
	v.container.Clear()
	v.container.
		AddItem(v.filterInput, 1, 0, false).
		AddItem(v.table, 0, 1, true)

	if v.app != nil {
		v.app.SetFocus(v.table.Table)
	}

	// Note: Status bar update removed since individual view status bars are no longer used
}

// onAdvancedFilterChange handles advanced filter changes
func (v *QoSView) onAdvancedFilterChange(filter *filters.Filter) {
	v.advancedFilter = filter
	v.updateTable()

	// Note: Status bar updates removed since individual view status bars are no longer used
}

// applyAdvancedFilter applies the advanced filter to QoS list
func (v *QoSView) applyAdvancedFilter(qosList []*dao.QoS) []*dao.QoS {
	if v.advancedFilter == nil || len(v.advancedFilter.Expressions) == 0 {
		return qosList
	}

	var filtered []*dao.QoS
	for _, qos := range qosList {
		// Convert QoS to map for filter evaluation
		qosData := v.qosToMap(qos)
		if v.advancedFilter.Evaluate(qosData) {
			filtered = append(filtered, qos)
		}
	}

	return filtered
}

// qosToMap converts a QoS to a map for filter evaluation
func (v *QoSView) qosToMap(qos *dao.QoS) map[string]interface{} {
	return map[string]interface{}{
		"Name":                 qos.Name,
		"Priority":             qos.Priority,
		"PreemptMode":          qos.PreemptMode,
		"GraceTime":            qos.GraceTime,
		"MaxJobsPerUser":       qos.MaxJobsPerUser,
		"MaxJobsPerAccount":    qos.MaxJobsPerAccount,
		"MaxSubmitJobsPerUser": qos.MaxSubmitJobsPerUser,
		"MaxCPUsPerUser":       qos.MaxCPUsPerUser,
		"MaxNodesPerUser":      qos.MaxNodesPerUser,
		"MaxWallTime":          qos.MaxWallTime,
		"MaxMemoryPerUser":     qos.MaxMemoryPerUser,
		"MinCPUs":              qos.MinCPUs,
		"MinNodes":             qos.MinNodes,
		"Flags":                strings.Join(qos.Flags, ","),
	}
}

// showGlobalSearch shows the global search interface
func (v *QoSView) showGlobalSearch() {
	if v.globalSearch == nil || v.pages == nil {
		return
	}

	v.globalSearch.Show(v.pages, func(result SearchResult) {
		// Handle search result selection
		switch result.Type {
		case "qos":
			// Focus on the selected QoS
			if qos, ok := result.Data.(*dao.QoS); ok {
				v.focusOnQoS(qos.Name)
			}
		default:
			// For other types, just close the search
			// Note: Search result status removed since individual view status bars are no longer used
		}
	})
}

// focusOnQoS focuses the table on a specific QoS
func (v *QoSView) focusOnQoS(qosName string) {
	v.mu.RLock()
	defer v.mu.RUnlock()

	// Find the QoS in our QoS list
	for i, qos := range v.qosList {
		if qos.Name == qosName {
			// Select the row in the table
			v.table.Select(i, 0)
			// Note: Focus status removed since individual view status bars are no longer used
			return
		}
	}

	// Note: Error message removed since individual view status bars are no longer used
}
