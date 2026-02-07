package views

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/jontk/s9s/internal/dao"
	"github.com/jontk/s9s/internal/ui/components"
	"github.com/jontk/s9s/internal/ui/filters"
	"github.com/jontk/s9s/internal/ui/styles"
	"github.com/rivo/tview"
)

// PartitionsView displays the partitions list with queue depth visualization
type PartitionsView struct {
	*BaseView
	client         dao.SlurmClient
	table          *components.Table
	partitions     []*dao.Partition
	queueInfo      map[string]*dao.QueueInfo
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
}

// SetPages sets the pages reference for modal handling
func (v *PartitionsView) SetPages(pages *tview.Pages) {
	v.pages = pages
	// Set pages for filter bar if it exists
	if v.filterBar != nil {
		v.filterBar.SetPages(pages)
	}
}

// SetApp sets the application reference
func (v *PartitionsView) SetApp(app *tview.Application) {
	v.app = app
	// Create filter bar now that we have app reference
	v.filterBar = components.NewFilterBar("partitions", app)
	v.filterBar.SetPages(v.pages)
	v.filterBar.SetOnFilterChange(v.onAdvancedFilterChange)
	v.filterBar.SetOnClose(v.closeAdvancedFilter)

	// Create global search
	v.globalSearch = NewGlobalSearch(v.client, app)
}

// NewPartitionsView creates a new partitions view
func NewPartitionsView(client dao.SlurmClient) *PartitionsView {
	v := &PartitionsView{
		BaseView:    NewBaseView("partitions", "Partitions"),
		client:      client,
		refreshRate: 30 * time.Second,
		partitions:  []*dao.Partition{},
		queueInfo:   make(map[string]*dao.QueueInfo),
	}

	// Create table with partition columns
	columns := []components.Column{
		components.NewColumn("Name").Width(15).Build(),
		components.NewColumn("State").Width(10).Sortable(true).Build(),
		components.NewColumn("Nodes").Width(8).Align(tview.AlignRight).Sortable(true).Build(),
		components.NewColumn("CPUs").Width(8).Align(tview.AlignRight).Sortable(true).Build(),
		components.NewColumn("Queue Depth").Width(20).Align(tview.AlignCenter).Build(),
		components.NewColumn("Running").Width(8).Align(tview.AlignRight).Sortable(true).Build(),
		components.NewColumn("Pending").Width(8).Align(tview.AlignRight).Sortable(true).Build(),
		components.NewColumn("Avg Wait").Width(10).Align(tview.AlignRight).Sortable(true).Build(),
		components.NewColumn("Max Wait").Width(10).Align(tview.AlignRight).Sortable(true).Build(),
		components.NewColumn("Efficiency").Width(12).Align(tview.AlignCenter).Build(),
		components.NewColumn("QOS").Width(15).Build(),
	}

	v.table = components.NewTableBuilder().
		WithColumns(columns...).
		WithSelectable(true).
		WithHeader(true).
		WithColors(tcell.ColorYellow, tcell.ColorTeal, tcell.ColorWhite).
		Build()

	// Set up callbacks
	v.table.SetOnSelect(v.onPartitionSelect)
	v.table.SetOnSort(v.onSort)

	// Create filter input with styled colors for visibility across themes
	v.filterInput = styles.NewStyledInputField().
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

// Init initializes the partitions view
func (v *PartitionsView) Init(ctx context.Context) error {
	_ = v.BaseView.Init(ctx)
	return v.Refresh()
}

// Render returns the view's main component
func (v *PartitionsView) Render() tview.Primitive {
	return v.container
}

// Refresh updates the partitions data
func (v *PartitionsView) Refresh() error {
	v.SetRefreshing(true)
	defer v.SetRefreshing(false)

	// Fetch partitions from backend
	partitionList, err := v.client.Partitions().List()
	if err != nil {
		v.SetLastError(err)
		// Note: Error handling removed since individual view status bars are no longer used
		return err
	}

	// Fetch queue information for each partition
	queueInfo := make(map[string]*dao.QueueInfo)
	for _, partition := range partitionList.Partitions {
		// Calculate queue info from jobs
		info, err := v.calculateQueueInfo(partition.Name)
		if err == nil {
			queueInfo[partition.Name] = info
		}
	}

	v.mu.Lock()
	v.partitions = partitionList.Partitions
	v.queueInfo = queueInfo
	v.mu.Unlock()

	// Update table
	v.updateTable()
	// Note: No longer updating individual view status bar since we use main app status bar for hints

	// Schedule next refresh
	v.scheduleRefresh()

	return nil
}

// Stop stops the view
func (v *PartitionsView) Stop() error {
	if v.refreshTimer != nil {
		v.refreshTimer.Stop()
	}
	return nil
}

// Hints returns keyboard hints
func (v *PartitionsView) Hints() []string {
	hints := []string{
		"[yellow]Enter[white] Details",
		"[yellow]J[white] Jobs",
		"[yellow]N[white] Nodes",
		"[yellow]A[white] Analytics",
		"[yellow]W[white] Wait Times",
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
func (v *PartitionsView) OnKey(event *tcell.EventKey) *tcell.EventKey {
	// Always prioritize filter input handling if it has focus
	// This allows the filter to maintain focus even when modals are present
	if v.filterInput != nil && v.filterInput.HasFocus() {
		if event.Key() == tcell.KeyEsc {
			v.app.SetFocus(v.table.Table)
			return nil
		}
		// Let the filter handle all keys when it has focus
		return event
	}

	// If a modal is open (and filter doesn't have focus), let it handle keys
	if v.isModalOpen() {
		return event
	}

	// Handle other keyboard events
	// handlePartitionKey returns nil if handled, event if not handled
	return v.handlePartitionKey(event)
}

// isModalOpen checks if a modal page is currently open
func (v *PartitionsView) isModalOpen() bool {
	return v.pages != nil && v.pages.GetPageCount() > 1
}

// handlePartitionKey handles non-filter keyboard events
// Returns nil if the key was handled (consumed), or the event if not handled
func (v *PartitionsView) handlePartitionKey(event *tcell.EventKey) *tcell.EventKey {
	// Handle advanced filter mode ESC
	if v.isAdvancedMode && event.Key() == tcell.KeyEsc {
		v.closeAdvancedFilter()
		return nil
	}

	// Handle mapped key handlers
	if handler, ok := v.partitionsKeyHandlers()[event.Key()]; ok {
		handler()
		return nil
	}

	// Handle rune commands
	if event.Key() == tcell.KeyRune && v.handleRuneCommand(event.Rune()) {
		return nil
	}

	// Key not handled - return event so it can be processed by the table
	return event
}

// partitionsKeyHandlers returns a map of function key handlers
func (v *PartitionsView) partitionsKeyHandlers() map[tcell.Key]func() {
	return map[tcell.Key]func(){
		tcell.KeyF3:    v.showAdvancedFilter,
		tcell.KeyCtrlF: v.showGlobalSearch,
		tcell.KeyEnter: v.showPartitionDetails,
	}
}

func (v *PartitionsView) handleRuneCommand(r rune) bool {
	switch r {
	case 'J':
		v.showPartitionJobs()
		return true
	case 'N':
		v.showPartitionNodes()
		return true
	case 'A':
		v.showPartitionAnalytics()
		return true
	case 'W':
		v.showWaitTimeAnalytics()
		return true
	case 'R':
		go func() { _ = v.Refresh() }()
		return true
	case '/':
		v.app.SetFocus(v.filterInput)
		return true
	}

	return false // Unhandled rune
}

// OnFocus handles focus events
func (v *PartitionsView) OnFocus() error {
	if v.app != nil {
		v.app.SetFocus(v.table.Table)
	}
	return nil
}

// OnLoseFocus handles loss of focus
func (v *PartitionsView) OnLoseFocus() error {
	return nil
}

// updateTable updates the table with current partition data
func (v *PartitionsView) updateTable() {
	v.mu.RLock()
	defer v.mu.RUnlock()

	// Apply advanced filter if active
	filteredPartitions := v.partitions
	if v.advancedFilter != nil && len(v.advancedFilter.Expressions) > 0 {
		filteredPartitions = v.applyAdvancedFilter(v.partitions)
	}

	data := make([][]string, len(filteredPartitions))
	for i, partition := range filteredPartitions {
		stateColor := dao.GetPartitionStateColor(partition.State)
		coloredState := fmt.Sprintf("[%s]%s[white]", stateColor, partition.State)

		// Get queue info
		queueInfo := v.queueInfo[partition.Name]
		queueDepth := ""
		running := "0"
		pending := "0"
		avgWait := "-"
		maxWait := "-"
		var efficiency string

		if queueInfo != nil {
			queueDepth = v.createQueueDepthBar(queueInfo.PendingJobs, queueInfo.RunningJobs)
			running = fmt.Sprintf("%d", queueInfo.RunningJobs)
			pending = fmt.Sprintf("%d", queueInfo.PendingJobs)

			if queueInfo.AverageWait > 0 {
				avgWait = FormatTimeDuration(queueInfo.AverageWait)
			}
			if queueInfo.LongestWait > 0 {
				maxWait = FormatTimeDuration(queueInfo.LongestWait)
			}

			// Calculate efficiency (allocated CPUs / total capacity)
			if partition.TotalCPUs > 0 {
				allocatedCPUs := v.calculateAllocatedCPUs(partition.Name)
				efficiencyPct := float64(allocatedCPUs) * 100.0 / float64(partition.TotalCPUs)
				if efficiencyPct > 100 {
					efficiencyPct = 100 // Cap at 100%
				}
				efficiency = v.createEfficiencyBar(efficiencyPct)
			} else {
				efficiency = "[gray]▱▱▱▱▱[white]"
			}
		} else {
			efficiency = "[gray]▱▱▱▱▱[white]"
		}

		// QOS list
		qos := strings.Join(partition.QOS, ",")
		if len(qos) > 14 {
			qos = qos[:11] + "..."
		}

		data[i] = []string{
			partition.Name,
			coloredState,
			fmt.Sprintf("%d", partition.TotalNodes),
			fmt.Sprintf("%d", partition.TotalCPUs),
			queueDepth,
			running,
			pending,
			avgWait,
			maxWait,
			efficiency,
			qos,
		}
	}

	v.table.SetData(data)
}

// createQueueDepthBar creates a visual queue depth representation
func (v *PartitionsView) createQueueDepthBar(pending, running int) string {
	total := pending + running
	if total == 0 {
		return "[gray]▱▱▱▱▱▱▱▱ 0[white]"
	}

	barLength := 8
	var bar strings.Builder

	// Calculate proportions
	runningRatio := float64(running) / float64(total)
	pendingRatio := float64(pending) / float64(total)

	runningBars := int(runningRatio * float64(barLength))
	pendingBars := int(pendingRatio * float64(barLength))

	// Ensure we don't exceed bar length
	if runningBars+pendingBars > barLength {
		if runningBars > pendingBars {
			runningBars = barLength - pendingBars
		} else {
			pendingBars = barLength - runningBars
		}
	}

	// Running jobs (green)
	if runningBars > 0 {
		bar.WriteString("[green]")
		for i := 0; i < runningBars; i++ {
			bar.WriteString("▰")
		}
	}

	// Pending jobs (yellow)
	if pendingBars > 0 {
		bar.WriteString("[yellow]")
		for i := 0; i < pendingBars; i++ {
			bar.WriteString("▰")
		}
	}

	// Empty space (gray)
	remaining := barLength - runningBars - pendingBars
	if remaining > 0 {
		bar.WriteString("[gray]")
		for i := 0; i < remaining; i++ {
			bar.WriteString("▱")
		}
	}

	bar.WriteString(fmt.Sprintf("[white] %d", total))
	return bar.String()
}

// createEfficiencyBar creates a visual efficiency representation
func (v *PartitionsView) createEfficiencyBar(percentage float64) string {
	barLength := 5
	filled := int(percentage / 20.0) // Each bar represents 20%

	if filled > barLength {
		filled = barLength
	}

	var bar strings.Builder

	// Choose color based on efficiency
	var color string
	switch {
	case percentage < 50:
		color = "red" // Low efficiency
	case percentage < 80:
		color = "yellow" // Medium efficiency
	default:
		color = "green" // High efficiency
	}

	bar.WriteString(fmt.Sprintf("[%s]", color))

	// Add filled bars
	for i := 0; i < filled; i++ {
		bar.WriteString("▰")
	}

	// Add empty bars
	bar.WriteString("[gray]")
	for i := filled; i < barLength; i++ {
		bar.WriteString("▱")
	}

	bar.WriteString(fmt.Sprintf("[white] %.0f%%", percentage))

	return bar.String()
}

// calculateAllocatedCPUs estimates allocated CPUs for running jobs in a partition
func (v *PartitionsView) calculateAllocatedCPUs(partitionName string) int {
	// Fetch running jobs for this partition
	opts := &dao.ListJobsOptions{
		Partitions: []string{partitionName},
		States:     []string{dao.JobStateRunning},
		Limit:      1000,
	}

	jobList, err := v.client.Jobs().List(opts)
	if err != nil {
		// If we can't get jobs, return 0
		return 0
	}

	// Estimate allocated CPUs based on node count
	// Assume each node contributes proportionally to partition's CPUs/nodes ratio
	// This is an approximation since we don't have per-job CPU allocation data
	totalNodes := 0
	for _, job := range jobList.Jobs {
		totalNodes += job.NodeCount
	}

	// Find the partition to get CPUs per node ratio
	v.mu.RLock()
	cpusPerNode := 1.0 // default fallback
	for _, p := range v.partitions {
		if p.Name == partitionName && p.TotalNodes > 0 {
			cpusPerNode = float64(p.TotalCPUs) / float64(p.TotalNodes)
			break
		}
	}
	v.mu.RUnlock()

	return int(float64(totalNodes) * cpusPerNode)
}

// calculateQueueInfo calculates queue information for a partition
func (v *PartitionsView) calculateQueueInfo(partitionName string) (*dao.QueueInfo, error) {
	// Fetch jobs for this partition
	opts := &dao.ListJobsOptions{
		Partitions: []string{partitionName},
		Limit:      1000,
	}

	jobList, err := v.client.Jobs().List(opts)
	if err != nil {
		return nil, err
	}

	info := &dao.QueueInfo{
		Partition: partitionName,
	}

	var waitTimes []time.Duration
	now := time.Now()

	for _, job := range jobList.Jobs {
		switch job.State {
		case dao.JobStateRunning:
			info.RunningJobs++
		case dao.JobStatePending:
			info.PendingJobs++
			// Calculate wait time
			waitTime := now.Sub(job.SubmitTime)
			waitTimes = append(waitTimes, waitTime)
		case dao.JobStateCompleting:
			info.RunningJobs++ // Count completing jobs as running
		}
		info.TotalJobs++
	}

	// Calculate average and longest wait times
	if len(waitTimes) > 0 {
		var totalWait time.Duration
		var longest time.Duration

		for _, wait := range waitTimes {
			totalWait += wait
			if wait > longest {
				longest = wait
			}
		}

		info.AverageWait = totalWait / time.Duration(len(waitTimes))
		info.LongestWait = longest
	}

	return info, nil
}

/*
TODO(lint): Review unused code - func (*PartitionsView).updateStatusBar is unused

updateStatusBar updates the status bar
func (v *PartitionsView) updateStatusBar(message string) {
	if message != "" {
		v.statusBar.SetText(message)
		return
	}

	v.mu.RLock()
	total := len(v.partitions)
	totalJobs := 0
	totalRunning := 0
	totalPending := 0

	for _, info := range v.queueInfo {
		totalJobs += info.TotalJobs
		totalRunning += info.RunningJobs
		totalPending += info.PendingJobs
	}
	v.mu.RUnlock()

	filtered := len(v.table.GetFilteredData())

	status := fmt.Sprintf("Partitions: %d | Jobs: [green]%d running[white], [yellow]%d pending[white], %d total",
		total, totalRunning, totalPending, totalJobs)

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
func (v *PartitionsView) scheduleRefresh() {
	// Remove automatic refresh scheduling to prevent memory leak
	// Refresh will be handled by the main app refresh timer
}

// onPartitionSelect handles partition selection
func (v *PartitionsView) onPartitionSelect(_, _ int) {
	data := v.table.GetSelectedData()
	if len(data) == 0 {
		return
	}

	// Note: Status bar update removed since individual view status bars are no longer used
	_ = data[0] // partitionName no longer used
}

// onSort handles column sorting
func (v *PartitionsView) onSort(_ int, _ bool) {
	// Note: Status bar update removed since individual view status bars are no longer used
}

// onFilterChange handles filter input changes
func (v *PartitionsView) onFilterChange(text string) {
	v.filter = text
	v.table.SetFilter(text)
	// Note: Status bar update removed since individual view status bars are no longer used
}

// onFilterDone handles filter input completion
func (v *PartitionsView) onFilterDone(_ tcell.Key) {
	if v.app != nil {
		v.app.SetFocus(v.table.Table)
	}
}

// showPartitionDetails shows detailed information for the selected partition
func (v *PartitionsView) showPartitionDetails() {
	data := v.table.GetSelectedData()
	if len(data) == 0 {
		return
	}

	partitionName := data[0]

	// Fetch full partition details
	partition, err := v.client.Partitions().Get(partitionName)
	if err != nil {
		// Note: Status bar update removed since individual view status bars are no longer used
		return
	}

	// Create details view
	details := v.formatPartitionDetails(partition)

	textView := tview.NewTextView().
		SetDynamicColors(true).
		SetText(details).
		SetScrollable(true)

	modal := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(textView, 0, 1, true).
		AddItem(tview.NewTextView().SetText("Press ESC to close"), 1, 0, false)

	modal.SetBorder(true).
		SetTitle(fmt.Sprintf(" Partition %s Details ", partitionName)).
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
				v.pages.RemovePage("partition-details")
			}
			return nil
		}
		return event
	})

	if v.pages != nil {
		v.pages.AddPage("partition-details", centeredModal, true, true)
	}
}

// formatPartitionDetails formats partition details for display
func (v *PartitionsView) formatPartitionDetails(partition *dao.Partition) string {
	var details strings.Builder

	details.WriteString(fmt.Sprintf("[yellow]Partition Name:[white] %s\n", partition.Name))

	stateColor := dao.GetPartitionStateColor(partition.State)
	details.WriteString(fmt.Sprintf("[yellow]State:[white] [%s]%s[white]\n", stateColor, partition.State))

	details.WriteString(fmt.Sprintf("[yellow]Total Nodes:[white] %d\n", partition.TotalNodes))
	details.WriteString(fmt.Sprintf("[yellow]Total CPUs:[white] %d\n", partition.TotalCPUs))

	details.WriteString(fmt.Sprintf("[yellow]Default Time:[white] %s\n", partition.DefaultTime))
	details.WriteString(fmt.Sprintf("[yellow]Max Time:[white] %s\n", partition.MaxTime))

	if len(partition.QOS) > 0 {
		details.WriteString(fmt.Sprintf("[yellow]QOS:[white] %s\n", strings.Join(partition.QOS, ", ")))
	}

	if len(partition.Nodes) > 0 {
		nodeList := strings.Join(partition.Nodes, ", ")
		if len(nodeList) > 100 {
			nodeList = nodeList[:97] + "..."
		}
		details.WriteString(fmt.Sprintf("[yellow]Nodes:[white] %s\n", nodeList))
	}

	// Add queue information if available
	v.mu.RLock()
	if queueInfo, exists := v.queueInfo[partition.Name]; exists {
		details.WriteString("\n[teal]Queue Information:[white]\n")
		details.WriteString(fmt.Sprintf("[yellow]  Total Jobs:[white] %d\n", queueInfo.TotalJobs))
		details.WriteString(fmt.Sprintf("[yellow]  Running Jobs:[white] %d\n", queueInfo.RunningJobs))
		details.WriteString(fmt.Sprintf("[yellow]  Pending Jobs:[white] %d\n", queueInfo.PendingJobs))

		if queueInfo.AverageWait > 0 {
			details.WriteString(fmt.Sprintf("[yellow]  Average Wait:[white] %s\n", FormatTimeDuration(queueInfo.AverageWait)))
		}

		if queueInfo.LongestWait > 0 {
			details.WriteString(fmt.Sprintf("[yellow]  Longest Wait:[white] %s\n", FormatTimeDuration(queueInfo.LongestWait)))
		}
	}
	v.mu.RUnlock()

	return details.String()
}

// showPartitionJobs shows jobs for the selected partition
func (v *PartitionsView) showPartitionJobs() {
	data := v.table.GetSelectedData()
	if len(data) == 0 {
		return
	}

	partitionName := data[0]
	v.SwitchToView("jobs")

	// Apply partition filter to Jobs view (filters at data fetch level)
	if jv, err := v.viewMgr.GetView("jobs"); err == nil {
		if jobsView, ok := jv.(*JobsView); ok {
			jobsView.SetPartitionFilter(partitionName)
		}
	}
}

// showPartitionNodes shows nodes for the selected partition
func (v *PartitionsView) showPartitionNodes() {
	data := v.table.GetSelectedData()
	if len(data) == 0 {
		return
	}

	partitionName := data[0]
	v.SwitchToView("nodes")

	// Apply partition filter to Nodes view (filters at data fetch level)
	if nv, err := v.viewMgr.GetView("nodes"); err == nil {
		if nodesView, ok := nv.(*NodesView); ok {
			nodesView.SetPartitionFilter(partitionName)
		}
	}
}

// showPartitionAnalytics shows comprehensive analytics for the selected partition
func (v *PartitionsView) showPartitionAnalytics() {
	data := v.table.GetSelectedData()
	if len(data) == 0 {
		return
	}

	partitionName := data[0]

	// Fetch full partition details
	partition, err := v.client.Partitions().Get(partitionName)
	if err != nil {
		// Note: Status bar update removed since individual view status bars are no longer used
		return
	}

	// Create analytics view
	analytics := v.formatPartitionAnalytics(partition)

	textView := tview.NewTextView().
		SetDynamicColors(true).
		SetText(analytics).
		SetScrollable(true)

	modal := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(textView, 0, 1, true).
		AddItem(tview.NewTextView().SetText("Press ESC to close | R to refresh"), 1, 0, false)

	modal.SetBorder(true).
		SetTitle(fmt.Sprintf(" Analytics: %s ", partitionName)).
		SetTitleAlign(tview.AlignCenter)

	// Create centered modal layout
	centeredModal := tview.NewFlex().
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(nil, 0, 1, false).
			AddItem(modal, 0, 8, true).
			AddItem(nil, 0, 1, false), 0, 8, true).
		AddItem(nil, 0, 1, false)

	// Handle keys
	modal.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEsc:
			if v.pages != nil {
				v.pages.RemovePage("partition-analytics")
			}
			return nil
		case tcell.KeyRune:
			if event.Rune() == 'R' || event.Rune() == 'r' {
				// Refresh and update the display
				go func() {
					_ = v.Refresh()
					// Update the analytics display
					newAnalytics := v.formatPartitionAnalytics(partition)
					textView.SetText(newAnalytics)
				}()
				return nil
			}
		}
		return event
	})

	if v.pages != nil {
		v.pages.AddPage("partition-analytics", centeredModal, true, true)
	}
}

// formatPartitionAnalytics formats comprehensive analytics for display
func (v *PartitionsView) formatPartitionAnalytics(partition *dao.Partition) string {
	var analytics strings.Builder

	analytics.WriteString(fmt.Sprintf("[yellow]Partition Analytics: %s[white]\n\n", partition.Name))
	analytics.WriteString(v.formatBasicInformation(partition))

	// Get queue info
	v.mu.RLock()
	queueInfo := v.queueInfo[partition.Name]
	v.mu.RUnlock()

	if queueInfo != nil {
		analytics.WriteString(v.formatQueueAnalytics(queueInfo))
		analytics.WriteString(v.formatResourceUtilization(partition, queueInfo))
		analytics.WriteString(v.formatWaitTimeSection(queueInfo))
	} else {
		analytics.WriteString("\n[yellow]Queue information not available[white]\n")
	}

	analytics.WriteString(v.formatQoSSection(partition))
	analytics.WriteString("\n[gray]Last updated: " + time.Now().Format("15:04:05") + "[white]")

	return analytics.String()
}

func (v *PartitionsView) formatBasicInformation(partition *dao.Partition) string {
	var output strings.Builder
	stateColor := dao.GetPartitionStateColor(partition.State)

	output.WriteString("[teal]Basic Information:[white]\n")
	output.WriteString(fmt.Sprintf("[yellow]  State:[white] [%s]%s[white]\n", stateColor, partition.State))
	output.WriteString(fmt.Sprintf("[yellow]  Total Nodes:[white] %d\n", partition.TotalNodes))
	output.WriteString(fmt.Sprintf("[yellow]  Total CPUs:[white] %d\n", partition.TotalCPUs))
	output.WriteString(fmt.Sprintf("[yellow]  Default Time Limit:[white] %s\n", partition.DefaultTime))
	output.WriteString(fmt.Sprintf("[yellow]  Maximum Time Limit:[white] %s\n", partition.MaxTime))

	return output.String()
}

func (v *PartitionsView) formatQueueAnalytics(queueInfo *dao.QueueInfo) string {
	var output strings.Builder

	output.WriteString("\n[teal]Queue Analytics:[white]\n")
	output.WriteString(fmt.Sprintf("[yellow]  Total Jobs:[white] %d\n", queueInfo.TotalJobs))
	output.WriteString(fmt.Sprintf("[yellow]  Running Jobs:[white] [green]%d[white]\n", queueInfo.RunningJobs))
	output.WriteString(fmt.Sprintf("[yellow]  Pending Jobs:[white] [yellow]%d[white]\n", queueInfo.PendingJobs))

	if queueInfo.TotalJobs > 0 {
		runningPct := float64(queueInfo.RunningJobs) * 100.0 / float64(queueInfo.TotalJobs)
		pendingPct := float64(queueInfo.PendingJobs) * 100.0 / float64(queueInfo.TotalJobs)
		output.WriteString(fmt.Sprintf("[yellow]  Running Percentage:[white] %.1f%%\n", runningPct))
		output.WriteString(fmt.Sprintf("[yellow]  Pending Percentage:[white] %.1f%%\n", pendingPct))
	}

	return output.String()
}

func (v *PartitionsView) formatResourceUtilization(partition *dao.Partition, queueInfo *dao.QueueInfo) string {
	if partition.TotalCPUs == 0 {
		return ""
	}

	var output strings.Builder
	utilizationPct := float64(queueInfo.RunningJobs) * 100.0 / float64(partition.TotalCPUs)

	output.WriteString("\n[teal]Resource Utilization:[white]\n")
	output.WriteString(fmt.Sprintf("[yellow]  CPU Utilization:[white] %.1f%%\n", utilizationPct))

	utilizationBar := v.createEfficiencyBar(utilizationPct)
	output.WriteString(fmt.Sprintf("[yellow]  Utilization Visual:[white] %s\n", utilizationBar))

	output.WriteString("\n[teal]Performance Assessment:[white]\n")
	output.WriteString(v.formatPerformanceStatus(utilizationPct))

	return output.String()
}

func (v *PartitionsView) formatPerformanceStatus(utilizationPct float64) string {
	switch {
	case utilizationPct < 30:
		return "[yellow]  Status:[white] [red]Under-utilized[white] - Consider job promotion or resource reallocation\n"
	case utilizationPct < 70:
		return "[yellow]  Status:[white] [yellow]Moderate utilization[white] - Room for growth\n"
	case utilizationPct < 95:
		return "[yellow]  Status:[white] [green]Well-utilized[white] - Optimal performance\n"
	default:
		return "[yellow]  Status:[white] [red]Over-subscribed[white] - Consider expanding capacity\n"
	}
}

func (v *PartitionsView) formatWaitTimeSection(queueInfo *dao.QueueInfo) string {
	if queueInfo.AverageWait == 0 && queueInfo.LongestWait == 0 {
		return ""
	}

	var output strings.Builder
	output.WriteString("\n[teal]Wait Time Analytics:[white]\n")

	if queueInfo.AverageWait > 0 {
		output.WriteString(fmt.Sprintf("[yellow]  Average Wait Time:[white] %s\n", FormatTimeDuration(queueInfo.AverageWait)))
	}

	if queueInfo.LongestWait > 0 {
		output.WriteString(fmt.Sprintf("[yellow]  Longest Wait Time:[white] %s\n", FormatTimeDuration(queueInfo.LongestWait)))
		output.WriteString(v.formatWaitAssessment(queueInfo.LongestWait.Hours()))
	}

	return output.String()
}

func (v *PartitionsView) formatWaitAssessment(hours float64) string {
	switch {
	case hours < 1:
		return "[yellow]  Wait Assessment:[white] [green]Excellent[white] - Quick turnaround\n"
	case hours < 6:
		return "[yellow]  Wait Assessment:[white] [yellow]Good[white] - Reasonable wait times\n"
	case hours < 24:
		return "[yellow]  Wait Assessment:[white] [orange]Moderate[white] - Some delays expected\n"
	default:
		return "[yellow]  Wait Assessment:[white] [red]Poor[white] - Long wait times detected\n"
	}
}

func (v *PartitionsView) formatQoSSection(partition *dao.Partition) string {
	if len(partition.QOS) == 0 {
		return ""
	}

	var output strings.Builder
	output.WriteString("\n[teal]Quality of Service:[white]\n")
	for _, qos := range partition.QOS {
		output.WriteString(fmt.Sprintf("[yellow]  - %s[white]\n", qos))
	}

	return output.String()
}

// showWaitTimeAnalytics shows detailed wait time analytics for all partitions
func (v *PartitionsView) showWaitTimeAnalytics() {
	// Create wait time analytics view
	analytics := v.formatWaitTimeAnalytics()

	textView := tview.NewTextView().
		SetDynamicColors(true).
		SetText(analytics).
		SetScrollable(true)

	modal := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(textView, 0, 1, true).
		AddItem(tview.NewTextView().SetText("Press ESC to close | R to refresh"), 1, 0, false)

	modal.SetBorder(true).
		SetTitle(" Wait Time Analytics (All Partitions) ").
		SetTitleAlign(tview.AlignCenter)

	// Create centered modal layout
	centeredModal := tview.NewFlex().
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(nil, 0, 1, false).
			AddItem(modal, 0, 8, true).
			AddItem(nil, 0, 1, false), 0, 8, true).
		AddItem(nil, 0, 1, false)

	// Handle keys
	modal.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEsc:
			if v.pages != nil {
				v.pages.RemovePage("wait-analytics")
			}
			return nil
		case tcell.KeyRune:
			if event.Rune() == 'R' || event.Rune() == 'r' {
				// Refresh and update the display
				go func() {
					_ = v.Refresh()
					// Update the analytics display
					newAnalytics := v.formatWaitTimeAnalytics()
					textView.SetText(newAnalytics)
				}()
				return nil
			}
		}
		return event
	})

	if v.pages != nil {
		v.pages.AddPage("wait-analytics", centeredModal, true, true)
	}
}

// formatWaitTimeAnalytics formats wait time analytics for all partitions
func (v *PartitionsView) formatWaitTimeAnalytics() string {
	var analytics strings.Builder

	analytics.WriteString("[yellow]Cluster-Wide Wait Time Analytics[white]\n\n")

	v.mu.RLock()
	defer v.mu.RUnlock()

	if len(v.queueInfo) == 0 {
		analytics.WriteString("[yellow]No queue information available[white]\n")
		return analytics.String()
	}

	totalPending, totalRunning, allWaitTimes, longestWait := v.calculateClusterStats()

	// Cluster summary
	v.writeClusterSummary(&analytics, totalPending, totalRunning, longestWait, allWaitTimes)

	// Per-partition breakdown
	analytics.WriteString("\n[teal]Per-Partition Breakdown:[white]\n")
	analytics.WriteString("[yellow]Partition          Pending  Running  Avg Wait    Max Wait     Status[white]\n")
	analytics.WriteString("─────────────────────────────────────────────────────────────────────\n")

	for _, partition := range v.partitions {
		info := v.queueInfo[partition.Name]
		if info == nil {
			continue
		}
		analytics.WriteString(v.formatPartitionRow(partition.Name, info))
	}

	analytics.WriteString("\n[gray]Last updated: " + time.Now().Format("15:04:05") + "[white]")

	return analytics.String()
}

// calculateClusterStats calculates cluster-wide queue statistics
func (v *PartitionsView) calculateClusterStats() (int, int, []time.Duration, time.Duration) {
	var totalPending, totalRunning int
	var allWaitTimes []time.Duration
	var longestWait time.Duration

	for _, info := range v.queueInfo {
		totalPending += info.PendingJobs
		totalRunning += info.RunningJobs
		if info.AverageWait > 0 {
			allWaitTimes = append(allWaitTimes, info.AverageWait)
		}
		if info.LongestWait > longestWait {
			longestWait = info.LongestWait
		}
	}
	return totalPending, totalRunning, allWaitTimes, longestWait
}

// writeClusterSummary writes cluster summary to the analytics output
func (v *PartitionsView) writeClusterSummary(w *strings.Builder, totalPending, totalRunning int, longestWait time.Duration, allWaitTimes []time.Duration) {
	w.WriteString("[teal]Cluster Summary:[white]\n")
	fmt.Fprintf(w, "[yellow]  Total Pending Jobs:[white] %d\n", totalPending)
	fmt.Fprintf(w, "[yellow]  Total Running Jobs:[white] %d\n", totalRunning)
	fmt.Fprintf(w, "[yellow]  Cluster-wide Longest Wait:[white] %s\n", FormatTimeDuration(longestWait))

	if len(allWaitTimes) > 0 {
		var totalWait time.Duration
		for _, wait := range allWaitTimes {
			totalWait += wait
		}
		avgClusterWait := totalWait / time.Duration(len(allWaitTimes))
		fmt.Fprintf(w, "[yellow]  Average Wait Across Partitions:[white] %s\n", FormatTimeDuration(avgClusterWait))
	}
}

// formatPartitionRow formats a single partition row for analytics output
func (v *PartitionsView) formatPartitionRow(partitionName string, info *dao.QueueInfo) string {
	name := partitionName
	if len(name) > 18 {
		name = name[:15] + "..."
	}
	name = fmt.Sprintf("%-18s", name)

	pending := fmt.Sprintf("%7d", info.PendingJobs)
	running := fmt.Sprintf("%7d", info.RunningJobs)

	avgWait := v.formatDurationField(info.AverageWait, 10)
	maxWait := v.formatDurationField(info.LongestWait, 10)

	statusColor, status := v.assessPartitionStatus(info)

	return fmt.Sprintf("%s %s %s %s %s [%s]%s[white]\n",
		name, pending, running, avgWait, maxWait, statusColor, status)
}

// formatDurationField formats a duration field with default value
func (v *PartitionsView) formatDurationField(dur time.Duration, width int) string {
	if dur > 0 {
		return fmt.Sprintf("%*s", width, FormatTimeDuration(dur))
	}
	return fmt.Sprintf("%*s", width, "-")
}

// assessPartitionStatus assesses the status of a partition based on queue information
func (v *PartitionsView) assessPartitionStatus(info *dao.QueueInfo) (string, string) {
	statusChecks := []struct {
		condition bool
		color     string
		status    string
	}{
		{v.isCriticalWaitTime(info.LongestWait), "red", "CRITICAL"},
		{v.isWarningWaitTime(info.LongestWait), "yellow", "WARNING"},
		{v.hasJobBacklog(info), "orange", "BACKLOG"},
	}

	for _, check := range statusChecks {
		if check.condition {
			return check.color, check.status
		}
	}

	return "green", "OK"
}

// isCriticalWaitTime checks if wait time is critical (> 24 hours)
func (v *PartitionsView) isCriticalWaitTime(wait time.Duration) bool {
	return wait.Hours() > 24
}

// isWarningWaitTime checks if wait time is warning level (> 6 hours)
func (v *PartitionsView) isWarningWaitTime(wait time.Duration) bool {
	return wait.Hours() > 6
}

// hasJobBacklog checks if there's a job backlog
func (v *PartitionsView) hasJobBacklog(info *dao.QueueInfo) bool {
	return info.PendingJobs > info.RunningJobs*2
}

// showAdvancedFilter shows the advanced filter bar
func (v *PartitionsView) showAdvancedFilter() {
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
func (v *PartitionsView) closeAdvancedFilter() {
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
func (v *PartitionsView) onAdvancedFilterChange(filter *filters.Filter) {
	v.advancedFilter = filter
	v.updateTable()

	// Note: Status bar updates removed since individual view status bars are no longer used
}

// applyAdvancedFilter applies the advanced filter to partitions
func (v *PartitionsView) applyAdvancedFilter(partitions []*dao.Partition) []*dao.Partition {
	if v.advancedFilter == nil || len(v.advancedFilter.Expressions) == 0 {
		return partitions
	}

	var filtered []*dao.Partition
	for _, partition := range partitions {
		// Convert partition to map for filter evaluation
		partitionData := v.partitionToMap(partition)
		if v.advancedFilter.Evaluate(partitionData) {
			filtered = append(filtered, partition)
		}
	}

	return filtered
}

// partitionToMap converts a partition to a map for filter evaluation
func (v *PartitionsView) partitionToMap(partition *dao.Partition) map[string]interface{} {
	data := map[string]interface{}{
		"Name":        partition.Name,
		"State":       partition.State,
		"TotalNodes":  partition.TotalNodes,
		"TotalCPUs":   partition.TotalCPUs,
		"DefaultTime": partition.DefaultTime,
		"MaxTime":     partition.MaxTime,
		"QOS":         strings.Join(partition.QOS, ","),
	}

	// Add queue information if available
	if queueInfo := v.queueInfo[partition.Name]; queueInfo != nil {
		data["PendingJobs"] = queueInfo.PendingJobs
		data["RunningJobs"] = queueInfo.RunningJobs
		data["AverageWait"] = queueInfo.AverageWait
		data["LongestWait"] = queueInfo.LongestWait
	}

	return data
}

// showGlobalSearch shows the global search interface
func (v *PartitionsView) showGlobalSearch() {
	if v.globalSearch == nil || v.pages == nil {
		return
	}

	v.globalSearch.Show(v.pages, func(result SearchResult) {
		// This callback is called from an event handler, so direct primitive
		// manipulation is safe. Do NOT use QueueUpdateDraw here - it will deadlock!
		switch result.Type {
		case "partition":
			if partition, ok := result.Data.(*dao.Partition); ok {
				v.focusOnPartition(partition.Name)
			}
		case "job":
			if job, ok := result.Data.(*dao.Job); ok {
				v.SwitchToView("jobs")
				if jv, err := v.viewMgr.GetView("jobs"); err == nil {
					if jobsView, ok := jv.(*JobsView); ok {
						jobsView.focusOnJob(job.ID)
					}
				}
			}
		case "node":
			if node, ok := result.Data.(*dao.Node); ok {
				v.SwitchToView("nodes")
				if nv, err := v.viewMgr.GetView("nodes"); err == nil {
					if nodesView, ok := nv.(*NodesView); ok {
						nodesView.focusOnNode(node.Name)
					}
				}
			}
		case "user":
			if user, ok := result.Data.(*dao.User); ok {
				v.SwitchToView("users")
				if uv, err := v.viewMgr.GetView("users"); err == nil {
					if usersView, ok := uv.(*UsersView); ok {
						usersView.focusOnUser(user.Name)
					}
				}
			}
		case "account":
			if account, ok := result.Data.(*dao.Account); ok {
				v.SwitchToView("accounts")
				if av, err := v.viewMgr.GetView("accounts"); err == nil {
					if accountsView, ok := av.(*AccountsView); ok {
						accountsView.focusOnAccount(account.Name)
					}
				}
			}
		case "qos":
			if qos, ok := result.Data.(*dao.QoS); ok {
				v.SwitchToView("qos")
				if qv, err := v.viewMgr.GetView("qos"); err == nil {
					if qosView, ok := qv.(*QoSView); ok {
						qosView.focusOnQoS(qos.Name)
					}
				}
			}
		case "reservation":
			if reservation, ok := result.Data.(*dao.Reservation); ok {
				v.SwitchToView("reservations")
				if rv, err := v.viewMgr.GetView("reservations"); err == nil {
					if reservationsView, ok := rv.(*ReservationsView); ok {
						reservationsView.focusOnReservation(reservation.Name)
					}
				}
			}
		}
	})
}

// focusOnPartition focuses the table on a specific partition
func (v *PartitionsView) focusOnPartition(partitionName string) {
	v.mu.RLock()
	defer v.mu.RUnlock()

	// Find the partition in our partition list
	for i, partition := range v.partitions {
		if partition.Name == partitionName {
			// Select the row in the table
			v.table.Select(i, 0)
			// Note: Status bar update removed since individual view status bars are no longer used
			return
		}
	}

	// Note: Status bar update removed since individual view status bars are no longer used
}
