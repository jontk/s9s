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
	"github.com/jontk/s9s/internal/ui/filters"
)

// NodesView displays the nodes list with resource utilization
type NodesView struct {
	*BaseView
	client         dao.SlurmClient
	table          *components.Table
	nodes          []*dao.Node
	mu             sync.RWMutex
	refreshTimer   *time.Timer
	refreshRate    time.Duration
	filter         string
	stateFilter    []string
	partFilter     string
	groupBy        string  // "none", "partition", "state", "features"
	groupExpanded  map[string]bool
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
func (v *NodesView) SetPages(pages *tview.Pages) {
	v.pages = pages
	// Set pages for filter bar if it exists
	if v.filterBar != nil {
		v.filterBar.SetPages(pages)
	}
}

// SetApp sets the application reference
func (v *NodesView) SetApp(app *tview.Application) {
	v.app = app
	// Create filter bar now that we have app reference
	v.filterBar = components.NewFilterBar("nodes", app)
	v.filterBar.SetPages(v.pages)
	v.filterBar.SetOnFilterChange(v.onAdvancedFilterChange)
	v.filterBar.SetOnClose(v.closeAdvancedFilter)

	// Create global search
	v.globalSearch = NewGlobalSearch(v.client, app)
}

// NewNodesView creates a new nodes view
func NewNodesView(client dao.SlurmClient) *NodesView {
	v := &NodesView{
		BaseView:      NewBaseView("nodes", "Nodes"),
		client:        client,
		refreshRate:   30 * time.Second,
		nodes:         []*dao.Node{},
		groupBy:       "none",
		groupExpanded: make(map[string]bool),
	}

	// Create table with node columns
	columns := []components.Column{
		components.NewColumn("Name").Width(15).Build(),
		components.NewColumn("State").Width(12).Sortable(true).Build(),
		components.NewColumn("Partitions").Width(15).Build(),
		components.NewColumn("CPU Usage").Width(15).Align(tview.AlignCenter).Sortable(true).Build(),
		components.NewColumn("Memory Usage").Width(15).Align(tview.AlignCenter).Sortable(true).Build(),
		components.NewColumn("CPU Total").Width(10).Align(tview.AlignRight).Sortable(true).Build(),
		components.NewColumn("Memory Total").Width(12).Align(tview.AlignRight).Sortable(true).Build(),
		components.NewColumn("Features").Width(20).Build(),
		components.NewColumn("Reason").Width(25).Build(),
	}

	v.table = components.NewTableBuilder().
		WithColumns(columns...).
		WithSelectable(true).
		WithHeader(true).
		WithColors(tcell.ColorYellow, tcell.ColorTeal, tcell.ColorWhite).
		Build()

	// Set up callbacks
	v.table.SetOnSelect(v.onNodeSelect)
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

// Init initializes the nodes view
func (v *NodesView) Init(ctx context.Context) error {
	v.BaseView.Init(ctx)
	return v.Refresh()
}

// Render returns the view's main component
func (v *NodesView) Render() tview.Primitive {
	return v.container
}

// Refresh updates the nodes data
func (v *NodesView) Refresh() error {
	v.SetRefreshing(true)
	defer v.SetRefreshing(false)

	// Fetch nodes from backend
	opts := &dao.ListNodesOptions{
		States: v.stateFilter,
	}

	if v.partFilter != "" {
		opts.Partitions = []string{v.partFilter}
	}

	nodeList, err := v.client.Nodes().List(opts)
	if err != nil {
		v.SetLastError(err)
		v.updateStatusBar(fmt.Sprintf("[red]Error: %v[white]", err))
		return err
	}

	v.mu.Lock()
	v.nodes = nodeList.Nodes
	v.mu.Unlock()

	// Update table
	v.updateTable()
	v.updateStatusBar("")

	// Schedule next refresh
	v.scheduleRefresh()

	return nil
}

// Stop stops the view
func (v *NodesView) Stop() error {
	if v.refreshTimer != nil {
		v.refreshTimer.Stop()
	}
	return nil
}

// Hints returns keyboard hints
func (v *NodesView) Hints() []string {
	hints := []string{
		"[yellow]Enter[white] Details",
		"[yellow]d[white] Drain",
		"[yellow]r[white] Resume",
		"[yellow]s[white] SSH",
		"[yellow]/[white] Filter",
		"[yellow]F3[white] Adv Filter",
		"[yellow]Ctrl+F[white] Search",
		"[yellow]1-9[white] Sort",
		"[yellow]R[white] Refresh",
		"[yellow]p[white] Partition",
		"[yellow]a[white] All States",
		"[yellow]g[white] Group By",
		"[yellow]Space[white] Toggle Group",
	}

	if v.isAdvancedMode {
		hints = append([]string{"[yellow]ESC[white] Exit Adv Filter"}, hints...)
	}

	return hints
}

// OnKey handles keyboard events
func (v *NodesView) OnKey(event *tcell.EventKey) *tcell.EventKey {
	// Check if a modal is open - if so, don't process view shortcuts
	if v.pages != nil && v.pages.GetPageCount() > 1 {
		return event // Let modal handle it
	}

	// Handle advanced filter mode
	if v.isAdvancedMode && event.Key() == tcell.KeyEsc {
		v.closeAdvancedFilter()
		return nil
	}

	switch event.Key() {
	case tcell.KeyF3:
		v.showAdvancedFilter()
		return nil
	case tcell.KeyCtrlF:
		v.showGlobalSearch()
		return nil
	case tcell.KeyRune:
		switch event.Rune() {
		case 'd', 'D':
			v.drainSelectedNode()
			return nil
		case 'r':
			v.resumeSelectedNode()
			return nil
		case 'R':
			go v.Refresh()
			return nil
		case 's', 'S':
			v.sshToNode()
			return nil
		case '/':
			v.app.SetFocus(v.filterInput)
			return nil
		case 'a', 'A':
			v.toggleStateFilter("all")
			return nil
		case 'i', 'I':
			v.toggleStateFilter(dao.NodeStateIdle)
			return nil
		case 'm', 'M':
			v.toggleStateFilter(dao.NodeStateMixed)
			return nil
		case 'p', 'P':
			v.promptPartitionFilter()
			return nil
		case 'g', 'G':
			v.promptGroupBy()
			return nil
		case ' ':
			v.toggleGroupExpansion()
			return nil
		}
	case tcell.KeyEnter:
		v.showNodeDetails()
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
func (v *NodesView) OnFocus() error {
	if v.app != nil {
		v.app.SetFocus(v.table.Table)
	}
	return nil
}

// OnLoseFocus handles loss of focus
func (v *NodesView) OnLoseFocus() error {
	return nil
}

// updateTable updates the table with current node data
func (v *NodesView) updateTable() {
	v.mu.RLock()
	defer v.mu.RUnlock()

	if v.groupBy == "none" {
		v.updateTableFlat()
	} else {
		v.updateTableGrouped()
	}
}

// updateTableFlat updates the table with flat node data
func (v *NodesView) updateTableFlat() {
	// Apply advanced filter if active
	filteredNodes := v.nodes
	if v.advancedFilter != nil && len(v.advancedFilter.Expressions) > 0 {
		filteredNodes = v.applyAdvancedFilter(v.nodes)
	}

	data := make([][]string, len(filteredNodes))
	for i, node := range filteredNodes {
		data[i] = v.formatNodeRow(node)
	}
	v.table.SetData(data)
}

// updateTableGrouped updates the table with grouped node data
func (v *NodesView) updateTableGrouped() {
	groups := v.groupNodes()
	var data [][]string

	for groupName, nodes := range groups {
		// Add group header
		expanded := v.groupExpanded[groupName]
		expandIcon := "▶"
		if expanded {
			expandIcon = "▼"
		}

		groupHeader := fmt.Sprintf("[yellow]%s %s (%d nodes)[white]", expandIcon, groupName, len(nodes))
		data = append(data, []string{groupHeader, "", "", "", "", "", "", "", ""})

		// Add nodes if expanded
		if expanded {
			for _, node := range nodes {
				nodeRow := v.formatNodeRow(node)
				nodeRow[0] = "  " + nodeRow[0] // Indent node names
				data = append(data, nodeRow)
			}
		}
	}

	v.table.SetData(data)
}

// formatNodeRow formats a single node row
func (v *NodesView) formatNodeRow(node *dao.Node) []string {
	stateColor := dao.GetNodeStateColor(node.State)
	coloredState := fmt.Sprintf("[%s]%s[white]", stateColor, node.State)

	// CPU usage bar
	cpuUsage := v.createUsageBar(node.CPUsAllocated, node.CPUsTotal)
	cpuUsageText := fmt.Sprintf("%s %d/%d", cpuUsage, node.CPUsAllocated, node.CPUsTotal)

	// Memory usage bar
	memUsage := v.createUsageBar(int(node.MemoryAllocated), int(node.MemoryTotal))
	memUsageText := fmt.Sprintf("%s %s/%s", memUsage, FormatMemory(node.MemoryAllocated), FormatMemory(node.MemoryTotal))

	// Partitions
	partitions := strings.Join(node.Partitions, ",")
	if len(partitions) > 14 {
		partitions = partitions[:11] + "..."
	}

	// Features
	features := strings.Join(node.Features, ",")
	if len(features) > 19 {
		features = features[:16] + "..."
	}

	// Reason
	reason := node.Reason
	if len(reason) > 24 {
		reason = reason[:21] + "..."
	}

	return []string{
		node.Name,
		coloredState,
		partitions,
		cpuUsageText,
		memUsageText,
		fmt.Sprintf("%d", node.CPUsTotal),
		FormatMemory(node.MemoryTotal),
		features,
		reason,
	}
}

// createUsageBar creates a visual usage bar
func (v *NodesView) createUsageBar(used, total int) string {
	if total == 0 {
		return "[gray]▱▱▱▱▱▱▱▱[white]"
	}

	percentage := float64(used) / float64(total)
	barLength := 8
	filled := int(percentage * float64(barLength))

	var bar strings.Builder

	// Choose color based on usage
	var color string
	if percentage < 0.5 {
		color = "green"
	} else if percentage < 0.8 {
		color = "yellow"
	} else {
		color = "red"
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

	bar.WriteString("[white]")

	return bar.String()
}


// updateStatusBar updates the status bar
func (v *NodesView) updateStatusBar(message string) {
	if message != "" {
		v.statusBar.SetText(message)
		return
	}

	v.mu.RLock()
	total := len(v.nodes)
	idle := 0
	allocated := 0
	mixed := 0
	down := 0
	drain := 0

	for _, node := range v.nodes {
		switch node.State {
		case dao.NodeStateIdle:
			idle++
		case dao.NodeStateAllocated:
			allocated++
		case dao.NodeStateMixed:
			mixed++
		case dao.NodeStateDown:
			down++
		case dao.NodeStateDrain, dao.NodeStateDraining:
			drain++
		}
	}
	v.mu.RUnlock()

	filtered := len(v.table.GetFilteredData())

	status := fmt.Sprintf("Total: %d | [green]Idle: %d[white] | [blue]Allocated: %d[white] | [yellow]Mixed: %d[white] | [red]Down: %d[white] | [orange]Drain: %d[white]",
		total, idle, allocated, mixed, down, drain)

	if filtered < total {
		status += fmt.Sprintf(" | Filtered: %d", filtered)
	}

	// Add grouping information
	if v.groupBy != "none" {
		groups := v.groupNodes()
		expandedCount := 0
		for _, expanded := range v.groupExpanded {
			if expanded {
				expandedCount++
			}
		}
		status += fmt.Sprintf(" | Grouped by: %s (%d groups, %d expanded)", v.groupBy, len(groups), expandedCount)
	}

	if v.IsRefreshing() {
		status += " | [yellow]Refreshing...[white]"
	}

	v.statusBar.SetText(status)
}

// scheduleRefresh schedules the next refresh
func (v *NodesView) scheduleRefresh() {
	// Remove automatic refresh scheduling to prevent memory leak
	// Refresh will be handled by the main app refresh timer
}

// onNodeSelect handles node selection
func (v *NodesView) onNodeSelect(row, col int) {
	data := v.table.GetSelectedData()
	if data == nil || len(data) == 0 {
		return
	}

	nodeName := data[0]
	v.updateStatusBar(fmt.Sprintf("Selected node: %s", nodeName))
}

// onSort handles column sorting
func (v *NodesView) onSort(col int, ascending bool) {
	v.updateStatusBar(fmt.Sprintf("Sorted by column %d", col+1))
}

// onFilterChange handles filter input changes
func (v *NodesView) onFilterChange(text string) {
	v.filter = text
	v.table.SetFilter(text)
	v.updateStatusBar("")
}

// onFilterDone handles filter input completion
func (v *NodesView) onFilterDone(key tcell.Key) {
	if v.app != nil {
		v.app.SetFocus(v.table.Table)
	}
}

// drainSelectedNode drains the selected node
func (v *NodesView) drainSelectedNode() {
	data := v.table.GetSelectedData()
	if data == nil || len(data) == 0 {
		return
	}

	nodeName := data[0]
	state := data[1]

	// Check if node can be drained
	if strings.Contains(state, dao.NodeStateDown) {
		v.updateStatusBar(fmt.Sprintf("[red]Node %s is down, cannot drain[white]", nodeName))
		return
	}

	// Prompt for reason
	input := tview.NewInputField().
		SetLabel("Drain reason: ").
		SetFieldWidth(50)

	input.SetDoneFunc(func(key tcell.Key) {
		if key == tcell.KeyEnter {
			reason := input.GetText()
			if reason == "" {
				reason = "Manual drain"
			}
			go v.performDrainNode(nodeName, reason)
		}
		v.app.SetRoot(v.container, true)
	})

	input.SetBorder(true).
		SetTitle(" Drain Node ").
		SetTitleAlign(tview.AlignCenter)

	v.app.SetRoot(input, true)
}

// performDrainNode performs the node drain operation
func (v *NodesView) performDrainNode(nodeName, reason string) {
	v.updateStatusBar(fmt.Sprintf("Draining node %s...", nodeName))

	err := v.client.Nodes().Drain(nodeName, reason)
	if err != nil {
		v.updateStatusBar(fmt.Sprintf("[red]Failed to drain node %s: %v[white]", nodeName, err))
		return
	}

	v.updateStatusBar(fmt.Sprintf("[green]Node %s drained successfully[white]", nodeName))

	// Refresh the view
	time.Sleep(500 * time.Millisecond)
	v.Refresh()
}

// resumeSelectedNode resumes the selected node
func (v *NodesView) resumeSelectedNode() {
	data := v.table.GetSelectedData()
	if data == nil || len(data) == 0 {
		return
	}

	nodeName := data[0]
	state := data[1]

	// Check if node can be resumed
	if !strings.Contains(state, dao.NodeStateDrain) {
		v.updateStatusBar(fmt.Sprintf("[red]Node %s is not drained, cannot resume[white]", nodeName))
		return
	}

	// Show confirmation dialog
	modal := tview.NewModal().
		SetText(fmt.Sprintf("Resume node %s?", nodeName)).
		AddButtons([]string{"Yes", "No"}).
		SetDoneFunc(func(buttonIndex int, buttonLabel string) {
			if buttonIndex == 0 {
				go v.performResumeNode(nodeName)
			}
			v.app.SetRoot(v.container, true)
		})

	v.app.SetRoot(modal, true)
}

// performResumeNode performs the node resume operation
func (v *NodesView) performResumeNode(nodeName string) {
	v.updateStatusBar(fmt.Sprintf("Resuming node %s...", nodeName))

	err := v.client.Nodes().Resume(nodeName)
	if err != nil {
		v.updateStatusBar(fmt.Sprintf("[red]Failed to resume node %s: %v[white]", nodeName, err))
		return
	}

	v.updateStatusBar(fmt.Sprintf("[green]Node %s resumed successfully[white]", nodeName))

	// Refresh the view
	time.Sleep(500 * time.Millisecond)
	v.Refresh()
}

// showNodeDetails shows detailed information for the selected node
func (v *NodesView) showNodeDetails() {
	data := v.table.GetSelectedData()
	if data == nil || len(data) == 0 {
		return
	}

	nodeName := data[0]

	// Fetch full node details
	node, err := v.client.Nodes().Get(nodeName)
	if err != nil {
		v.updateStatusBar(fmt.Sprintf("[red]Failed to get node details: %v[white]", err))
		return
	}

	// Create details view
	details := v.formatNodeDetails(node)

	textView := tview.NewTextView().
		SetDynamicColors(true).
		SetText(details).
		SetScrollable(true)

	modal := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(textView, 0, 1, true).
		AddItem(tview.NewTextView().SetText("Press ESC to close"), 1, 0, false)

	modal.SetBorder(true).
		SetTitle(fmt.Sprintf(" Node %s Details ", nodeName)).
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
				v.pages.RemovePage("node-details")
			}
			return nil
		}
		return event
	})

	if v.pages != nil {
		v.pages.AddPage("node-details", centeredModal, true, true)
	}
}

// formatNodeDetails formats node details for display
func (v *NodesView) formatNodeDetails(node *dao.Node) string {
	var details strings.Builder

	details.WriteString(fmt.Sprintf("[yellow]Node Name:[white] %s\n", node.Name))

	stateColor := dao.GetNodeStateColor(node.State)
	details.WriteString(fmt.Sprintf("[yellow]State:[white] [%s]%s[white]\n", stateColor, node.State))

	details.WriteString(fmt.Sprintf("[yellow]Partitions:[white] %s\n", strings.Join(node.Partitions, ", ")))

	details.WriteString("\n[teal]CPU Information:[white]\n")
	details.WriteString(fmt.Sprintf("[yellow]  Total CPUs:[white] %d\n", node.CPUsTotal))
	details.WriteString(fmt.Sprintf("[yellow]  Allocated CPUs:[white] %d\n", node.CPUsAllocated))
	details.WriteString(fmt.Sprintf("[yellow]  Idle CPUs:[white] %d\n", node.CPUsIdle))
	cpuPercent := 0.0
	if node.CPUsTotal > 0 {
		cpuPercent = float64(node.CPUsAllocated) * 100.0 / float64(node.CPUsTotal)
	}
	details.WriteString(fmt.Sprintf("[yellow]  CPU Usage:[white] %.1f%%\n", cpuPercent))

	details.WriteString("\n[teal]Memory Information:[white]\n")
	details.WriteString(fmt.Sprintf("[yellow]  Total Memory:[white] %s\n", FormatMemory(node.MemoryTotal)))
	details.WriteString(fmt.Sprintf("[yellow]  Allocated Memory:[white] %s\n", FormatMemory(node.MemoryAllocated)))
	details.WriteString(fmt.Sprintf("[yellow]  Free Memory:[white] %s\n", FormatMemory(node.MemoryFree)))
	memPercent := 0.0
	if node.MemoryTotal > 0 {
		memPercent = float64(node.MemoryAllocated) * 100.0 / float64(node.MemoryTotal)
	}
	details.WriteString(fmt.Sprintf("[yellow]  Memory Usage:[white] %.1f%%\n", memPercent))

	if len(node.Features) > 0 {
		details.WriteString(fmt.Sprintf("\n[yellow]Features:[white] %s\n", strings.Join(node.Features, ", ")))
	}

	if node.Reason != "" {
		details.WriteString(fmt.Sprintf("\n[yellow]Reason:[white] %s\n", node.Reason))
		if node.ReasonTime != nil {
			details.WriteString(fmt.Sprintf("[yellow]Reason Time:[white] %s\n", node.ReasonTime.Format("2006-01-02 15:04:05")))
		}
	}

	if len(node.AllocatedJobs) > 0 {
		details.WriteString(fmt.Sprintf("\n[yellow]Allocated Jobs:[white] %s\n", strings.Join(node.AllocatedJobs, ", ")))
	}

	return details.String()
}

// sshToNode opens SSH connection to the selected node
func (v *NodesView) sshToNode() {
	data := v.table.GetSelectedData()
	if data == nil || len(data) == 0 {
		return
	}

	nodeName := data[0]

	// TODO: Implement SSH functionality
	v.updateStatusBar(fmt.Sprintf("[yellow]SSH not yet implemented for node %s[white]", nodeName))
}

// toggleStateFilter toggles node state filter
func (v *NodesView) toggleStateFilter(state string) {
	if state == "all" {
		v.stateFilter = []string{}
	} else {
		// Toggle the state in filter
		found := false
		for i, s := range v.stateFilter {
			if s == state {
				v.stateFilter = append(v.stateFilter[:i], v.stateFilter[i+1:]...)
				found = true
				break
			}
		}
		if !found {
			v.stateFilter = append(v.stateFilter, state)
		}
	}

	go v.Refresh()
}

// promptPartitionFilter prompts for partition filter
func (v *NodesView) promptPartitionFilter() {
	input := tview.NewInputField().
		SetLabel("Filter by partition: ").
		SetFieldWidth(20).
		SetText(v.partFilter)

	input.SetDoneFunc(func(key tcell.Key) {
		if key == tcell.KeyEnter {
			v.partFilter = input.GetText()
			go v.Refresh()
		}
		v.app.SetRoot(v.container, true)
	})

	input.SetBorder(true).
		SetTitle(" Partition Filter ").
		SetTitleAlign(tview.AlignCenter)

	v.app.SetRoot(input, true)
}

// groupNodes groups nodes based on the current groupBy setting
func (v *NodesView) groupNodes() map[string][]*dao.Node {
	groups := make(map[string][]*dao.Node)

	for _, node := range v.nodes {
		var groupKey string

		switch v.groupBy {
		case "partition":
			if len(node.Partitions) > 0 {
				groupKey = node.Partitions[0] // Use first partition
			} else {
				groupKey = "<no partition>"
			}
		case "state":
			groupKey = node.State
		case "features":
			if len(node.Features) > 0 {
				groupKey = node.Features[0] // Use first feature
			} else {
				groupKey = "<no features>"
			}
		default:
			groupKey = "All Nodes"
		}

		groups[groupKey] = append(groups[groupKey], node)
	}

	return groups
}

// promptGroupBy prompts for grouping method
func (v *NodesView) promptGroupBy() {
	options := []string{"none", "partition", "state", "features"}
	currentIndex := 0

	// Find current selection
	for i, opt := range options {
		if opt == v.groupBy {
			currentIndex = i
			break
		}
	}

	modal := tview.NewList()
	modal.SetBorder(true).
		SetTitle(" Group Nodes By ").
		SetTitleAlign(tview.AlignCenter)

	for i, option := range options {
		text := strings.Title(option)
		if option == v.groupBy {
			text = fmt.Sprintf("[yellow]● %s[white]", text)
		} else {
			text = fmt.Sprintf("  %s", text)
		}

		modal.AddItem(text, "", rune('1'+i), func() {
			selectedOption := options[modal.GetCurrentItem()]
			v.setGroupBy(selectedOption)
			if v.pages != nil {
				v.pages.RemovePage("group-by")
			}
		})
	}

	modal.SetCurrentItem(currentIndex)

	// Handle ESC key
	modal.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEsc {
			if v.pages != nil {
				v.pages.RemovePage("group-by")
			}
			return nil
		}
		return event
	})

	// Create centered modal layout
	centeredModal := tview.NewFlex().
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(nil, 0, 1, false).
			AddItem(modal, 0, 3, true).
			AddItem(nil, 0, 1, false), 0, 3, true).
		AddItem(nil, 0, 1, false)

	if v.pages != nil {
		v.pages.AddPage("group-by", centeredModal, true, true)
	}
}

// setGroupBy sets the grouping method and refreshes the view
func (v *NodesView) setGroupBy(groupBy string) {
	v.groupBy = groupBy
	if groupBy != "none" {
		// Expand all groups by default when switching to grouped view
		v.expandAllGroups()
	}
	v.updateTable()
	v.updateStatusBar("")
}

// toggleGroupExpansion toggles expansion of the selected group
func (v *NodesView) toggleGroupExpansion() {
	if v.groupBy == "none" {
		return
	}

	data := v.table.GetSelectedData()
	if data == nil || len(data) == 0 {
		return
	}

	selectedText := data[0]

	// Check if this is a group header (starts with ▶ or ▼)
	if strings.HasPrefix(selectedText, "[yellow]▶") || strings.HasPrefix(selectedText, "[yellow]▼") {
		// Extract group name
		parts := strings.Split(selectedText, " ")
		if len(parts) >= 3 {
			groupName := parts[1]
			v.groupExpanded[groupName] = !v.groupExpanded[groupName]
			v.updateTable()
		}
	}
}

// expandAllGroups expands all groups
func (v *NodesView) expandAllGroups() {
	groups := v.groupNodes()
	for groupName := range groups {
		v.groupExpanded[groupName] = true
	}
}

// showAdvancedFilter shows the advanced filter bar
func (v *NodesView) showAdvancedFilter() {
	if v.filterBar == nil || v.pages == nil {
		return
	}

	v.isAdvancedMode = true

	// Replace the simple filter with advanced filter bar
	v.container.Clear()
	v.container.
		AddItem(v.filterBar, 5, 0, true).
		AddItem(v.table.Table, 0, 1, false).
		AddItem(v.statusBar, 1, 0, false)

	v.filterBar.Show()
	v.updateStatusBar("[yellow]Advanced Filter Mode - Tab for presets, F1 for help[white]")
}

// closeAdvancedFilter closes the advanced filter bar
func (v *NodesView) closeAdvancedFilter() {
	v.isAdvancedMode = false

	// Restore the simple filter
	v.container.Clear()
	v.container.
		AddItem(v.filterInput, 1, 0, false).
		AddItem(v.table.Table, 0, 1, true).
		AddItem(v.statusBar, 1, 0, false)

	if v.app != nil {
		v.app.SetFocus(v.table.Table)
	}

	v.updateStatusBar("")
}

// onAdvancedFilterChange handles advanced filter changes
func (v *NodesView) onAdvancedFilterChange(filter *filters.Filter) {
	v.advancedFilter = filter
	v.updateTable()

	if filter != nil && len(filter.Expressions) > 0 {
		v.updateStatusBar(fmt.Sprintf("[green]Filter applied: %d conditions[white]", len(filter.Expressions)))
	} else {
		v.updateStatusBar("")
	}
}

// applyAdvancedFilter applies the advanced filter to nodes
func (v *NodesView) applyAdvancedFilter(nodes []*dao.Node) []*dao.Node {
	if v.advancedFilter == nil || len(v.advancedFilter.Expressions) == 0 {
		return nodes
	}

	var filtered []*dao.Node
	for _, node := range nodes {
		// Convert node to map for filter evaluation
		nodeData := v.nodeToMap(node)
		if v.advancedFilter.Evaluate(nodeData) {
			filtered = append(filtered, node)
		}
	}

	return filtered
}

// nodeToMap converts a node to a map for filter evaluation
func (v *NodesView) nodeToMap(node *dao.Node) map[string]interface{} {
	return map[string]interface{}{
		"Name":           node.Name,
		"State":          node.State,
		"CPUsAllocated":  node.CPUsAllocated,
		"CPUsTotal":      node.CPUsTotal,
		"MemoryAllocated": node.MemoryAllocated,
		"MemoryTotal":    node.MemoryTotal,
		"Features":       strings.Join(node.Features, ","),
		"Partitions":     strings.Join(node.Partitions, ","),
		"Reason":         node.Reason,
	}
}

// showGlobalSearch shows the global search interface
func (v *NodesView) showGlobalSearch() {
	if v.globalSearch == nil || v.pages == nil {
		return
	}

	v.globalSearch.Show(v.pages, func(result SearchResult) {
		// Handle search result selection
		switch result.Type {
		case "node":
			// Focus on the selected node
			if node, ok := result.Data.(*dao.Node); ok {
				v.focusOnNode(node.Name)
			}
		default:
			// For other types, just close the search
			v.updateStatusBar(fmt.Sprintf("Selected %s: %s", result.Type, result.Name))
		}
	})
}

// focusOnNode focuses the table on a specific node
func (v *NodesView) focusOnNode(nodeName string) {
	v.mu.RLock()
	defer v.mu.RUnlock()

	// Find the node in our node list
	for i, node := range v.nodes {
		if node.Name == nodeName {
			// Select the row in the table
			v.table.Table.Select(i, 0)
			v.updateStatusBar(fmt.Sprintf("Focused on node: %s", nodeName))
			return
		}
	}

	v.updateStatusBar(fmt.Sprintf("[yellow]Node %s not found in current view[white]", nodeName))
}