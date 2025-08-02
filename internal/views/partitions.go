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

// PartitionsView displays the partitions list with queue depth visualization
type PartitionsView struct {
	*BaseView
	client       dao.SlurmClient
	table        *components.Table
	partitions   []*dao.Partition
	queueInfo    map[string]*dao.QueueInfo
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
func (v *PartitionsView) SetPages(pages *tview.Pages) {
	v.pages = pages
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
		components.NewColumn("Default Time").Width(12).Build(),
		components.NewColumn("Max Time").Width(12).Build(),
		components.NewColumn("QOS").Width(20).Build(),
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

// Init initializes the partitions view
func (v *PartitionsView) Init(ctx context.Context) error {
	v.BaseView.Init(ctx)
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
		v.updateStatusBar(fmt.Sprintf("[red]Error: %v[white]", err))
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
	v.updateStatusBar("")

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
	return []string{
		"[yellow]Enter[white] Details",
		"[yellow]j[white] Jobs",
		"[yellow]n[white] Nodes",
		"[yellow]/[white] Filter",
		"[yellow]1-9[white] Sort",
		"[yellow]R[white] Refresh",
	}
}

// OnKey handles keyboard events
func (v *PartitionsView) OnKey(event *tcell.EventKey) *tcell.EventKey {
	switch event.Key() {
	case tcell.KeyRune:
		switch event.Rune() {
		case 'j', 'J':
			v.showPartitionJobs()
			return nil
		case 'n', 'N':
			v.showPartitionNodes()
			return nil
		case 'R':
			go v.Refresh()
			return nil
		case '/':
			v.app.SetFocus(v.filterInput)
			return nil
		}
	case tcell.KeyEnter:
		v.showPartitionDetails()
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

	data := make([][]string, len(v.partitions))
	for i, partition := range v.partitions {
		stateColor := GetPartitionStateColor(partition.State)
		coloredState := fmt.Sprintf("[%s]%s[white]", stateColor, partition.State)

		// Get queue info
		queueInfo := v.queueInfo[partition.Name]
		queueDepth := ""
		running := "0"
		pending := "0"
		
		if queueInfo != nil {
			queueDepth = v.createQueueDepthBar(queueInfo.PendingJobs, queueInfo.RunningJobs)
			running = fmt.Sprintf("%d", queueInfo.RunningJobs)
			pending = fmt.Sprintf("%d", queueInfo.PendingJobs)
		}

		// QOS list
		qos := strings.Join(partition.QOS, ",")
		if len(qos) > 19 {
			qos = qos[:16] + "..."
		}

		data[i] = []string{
			partition.Name,
			coloredState,
			fmt.Sprintf("%d", partition.TotalNodes),
			fmt.Sprintf("%d", partition.TotalCPUs),
			queueDepth,
			running,
			pending,
			partition.DefaultTime,
			partition.MaxTime,
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

// updateStatusBar updates the status bar
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

// scheduleRefresh schedules the next refresh
func (v *PartitionsView) scheduleRefresh() {
	// Remove automatic refresh scheduling to prevent memory leak
	// Refresh will be handled by the main app refresh timer
}

// onPartitionSelect handles partition selection
func (v *PartitionsView) onPartitionSelect(row, col int) {
	data := v.table.GetSelectedData()
	if data == nil || len(data) == 0 {
		return
	}

	partitionName := data[0]
	v.updateStatusBar(fmt.Sprintf("Selected partition: %s", partitionName))
}

// onSort handles column sorting
func (v *PartitionsView) onSort(col int, ascending bool) {
	v.updateStatusBar(fmt.Sprintf("Sorted by column %d", col+1))
}

// onFilterChange handles filter input changes
func (v *PartitionsView) onFilterChange(text string) {
	v.filter = text
	v.table.SetFilter(text)
	v.updateStatusBar("")
}

// onFilterDone handles filter input completion
func (v *PartitionsView) onFilterDone(key tcell.Key) {
	if v.app != nil {
		v.app.SetFocus(v.table.Table)
	}
}

// showPartitionDetails shows detailed information for the selected partition
func (v *PartitionsView) showPartitionDetails() {
	data := v.table.GetSelectedData()
	if data == nil || len(data) == 0 {
		return
	}

	partitionName := data[0]
	
	// Fetch full partition details
	partition, err := v.client.Partitions().Get(partitionName)
	if err != nil {
		v.updateStatusBar(fmt.Sprintf("[red]Failed to get partition details: %v[white]", err))
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
	
	stateColor := GetPartitionStateColor(partition.State)
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
			details.WriteString(fmt.Sprintf("[yellow]  Average Wait:[white] %s\n", formatDuration(queueInfo.AverageWait)))
		}
		
		if queueInfo.LongestWait > 0 {
			details.WriteString(fmt.Sprintf("[yellow]  Longest Wait:[white] %s\n", formatDuration(queueInfo.LongestWait)))
		}
	}
	v.mu.RUnlock()

	return details.String()
}

// showPartitionJobs shows jobs for the selected partition
func (v *PartitionsView) showPartitionJobs() {
	data := v.table.GetSelectedData()
	if data == nil || len(data) == 0 {
		return
	}

	partitionName := data[0]
	
	// TODO: Switch to jobs view with partition filter
	v.updateStatusBar(fmt.Sprintf("[yellow]Job view filtering not yet implemented for partition %s[white]", partitionName))
}

// showPartitionNodes shows nodes for the selected partition
func (v *PartitionsView) showPartitionNodes() {
	data := v.table.GetSelectedData()
	if data == nil || len(data) == 0 {
		return
	}

	partitionName := data[0]
	
	// TODO: Switch to nodes view with partition filter
	v.updateStatusBar(fmt.Sprintf("[yellow]Node view filtering not yet implemented for partition %s[white]", partitionName))
}