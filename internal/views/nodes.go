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
	"github.com/jontk/s9s/internal/ssh"
	"github.com/jontk/s9s/internal/ui/components"
	"github.com/jontk/s9s/internal/ui/filters"
	"github.com/rivo/tview"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
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
	groupBy        string // "none", "partition", "state", "features"
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
	sshClient      *ssh.SSHClient
	sshTerminal    *SSHTerminalView
}

// SetPages sets the pages reference for modal handling
func (v *NodesView) SetPages(pages *tview.Pages) {
	v.pages = pages
	// Set pages for filter bar if it exists
	if v.filterBar != nil {
		v.filterBar.SetPages(pages)
	}
	// Set pages for SSH terminal if it exists
	if v.sshTerminal != nil {
		v.sshTerminal.SetPages(pages)
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

	// Initialize SSH client with default configuration
	v.sshClient = ssh.NewSSHClient(ssh.DefaultSSHConfig())

	// Initialize SSH terminal view
	v.sshTerminal = NewSSHTerminalView(app)
	if v.pages != nil {
		v.sshTerminal.SetPages(v.pages)
	}
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
		components.NewColumn("CPU Usage").Width(20).Align(tview.AlignCenter).Sortable(true).Build(),
		components.NewColumn("Memory Usage").Width(20).Align(tview.AlignCenter).Sortable(true).Build(),
		components.NewColumn("CPU Total").Width(10).Align(tview.AlignRight).Sortable(true).Build(),
		components.NewColumn("Memory Total").Width(15).Align(tview.AlignRight).Sortable(true).Build(),
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

	// Create container layout (removed individual status bar to prevent conflicts with main status bar)
	v.container = tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(v.filterInput, 1, 0, false).
		AddItem(v.table, 0, 1, true)

	return v
}

// Init initializes the nodes view
func (v *NodesView) Init(ctx context.Context) error {
	_ = v.BaseView.Init(ctx)
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
		// Note: Error handling removed since individual view status bars are no longer used
		return err
	}

	v.mu.Lock()
	v.nodes = nodeList.Nodes
	v.mu.Unlock()

	// Update table
	v.updateTable()
	// Note: No longer updating individual view status bar since we use main app status bar for hints

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
		"Bar: █=Used ▒=Alloc ▱=Free",
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

	// If filter input has focus, let it handle the key events (except ESC)
	if v.filterInput != nil && v.filterInput.HasFocus() {
		if event.Key() == tcell.KeyEsc {
			v.app.SetFocus(v.table.Table)
			return nil
		}
		return event
	}

	// Handle advanced filter mode
	if v.isAdvancedMode && event.Key() == tcell.KeyEsc {
		v.closeAdvancedFilter()
		return nil
	}

	// Handle by key type
	if event.Key() == tcell.KeyRune {
		return v.handleNodesViewRune(event)
	}

	// Handle by special key
	if handler, ok := v.nodesKeyHandlers()[event.Key()]; ok {
		return handler(v, event)
	}

	return event
}

// handleNodesViewRune handles rune key presses in the nodes view
func (v *NodesView) handleNodesViewRune(event *tcell.EventKey) *tcell.EventKey {
	handler, ok := v.nodesRuneHandlers()[event.Rune()]
	if ok {
		return handler(v, event)
	}
	return event
}

// nodesKeyHandlers returns a map of special keys to their handlers
func (v *NodesView) nodesKeyHandlers() map[tcell.Key]func(*NodesView, *tcell.EventKey) *tcell.EventKey {
	return map[tcell.Key]func(*NodesView, *tcell.EventKey) *tcell.EventKey{
		tcell.KeyF3:     func(v *NodesView, _ *tcell.EventKey) *tcell.EventKey { v.showAdvancedFilter(); return nil },
		tcell.KeyCtrlF:  func(v *NodesView, _ *tcell.EventKey) *tcell.EventKey { v.showGlobalSearch(); return nil },
		tcell.KeyEnter:  func(v *NodesView, _ *tcell.EventKey) *tcell.EventKey { v.showNodeDetails(); return nil },
	}
}

// nodesRuneHandlers returns a map of rune keys to their handlers
func (v *NodesView) nodesRuneHandlers() map[rune]func(*NodesView, *tcell.EventKey) *tcell.EventKey {
	return map[rune]func(*NodesView, *tcell.EventKey) *tcell.EventKey{
		'd': func(v *NodesView, _ *tcell.EventKey) *tcell.EventKey { v.drainSelectedNode(); return nil },
		'D': func(v *NodesView, _ *tcell.EventKey) *tcell.EventKey { v.drainSelectedNode(); return nil },
		'r': func(v *NodesView, _ *tcell.EventKey) *tcell.EventKey { v.resumeSelectedNode(); return nil },
		'R': func(v *NodesView, _ *tcell.EventKey) *tcell.EventKey { go func() { _ = v.Refresh() }(); return nil },
		's': func(v *NodesView, _ *tcell.EventKey) *tcell.EventKey { v.sshToNode(); return nil },
		'S': func(v *NodesView, _ *tcell.EventKey) *tcell.EventKey { v.sshToNode(); return nil },
		'/': func(v *NodesView, _ *tcell.EventKey) *tcell.EventKey { v.app.SetFocus(v.filterInput); return nil },
		'a': func(v *NodesView, _ *tcell.EventKey) *tcell.EventKey { v.toggleStateFilter("all"); return nil },
		'A': func(v *NodesView, _ *tcell.EventKey) *tcell.EventKey { v.toggleStateFilter("all"); return nil },
		'i': func(v *NodesView, _ *tcell.EventKey) *tcell.EventKey { v.toggleStateFilter(dao.NodeStateIdle); return nil },
		'I': func(v *NodesView, _ *tcell.EventKey) *tcell.EventKey { v.toggleStateFilter(dao.NodeStateIdle); return nil },
		'm': func(v *NodesView, _ *tcell.EventKey) *tcell.EventKey { v.toggleStateFilter(dao.NodeStateMixed); return nil },
		'M': func(v *NodesView, _ *tcell.EventKey) *tcell.EventKey { v.toggleStateFilter(dao.NodeStateMixed); return nil },
		'p': func(v *NodesView, _ *tcell.EventKey) *tcell.EventKey { v.promptPartitionFilter(); return nil },
		'P': func(v *NodesView, _ *tcell.EventKey) *tcell.EventKey { v.promptPartitionFilter(); return nil },
		'g': func(v *NodesView, _ *tcell.EventKey) *tcell.EventKey { v.promptGroupBy(); return nil },
		'G': func(v *NodesView, _ *tcell.EventKey) *tcell.EventKey { v.promptGroupBy(); return nil },
		' ': func(v *NodesView, _ *tcell.EventKey) *tcell.EventKey { v.toggleGroupExpansion(); return nil },
	}
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
	displayState := v.getNodeDisplayState(node)
	stateColor := dao.GetNodeStateColor(displayState)
	coloredState := fmt.Sprintf("[%s]%s[white]", stateColor, displayState)

	cpuUsageText := v.formatNodeCPUUsage(node)
	memUsageText := v.formatNodeMemoryUsage(node)

	partitions := truncateString(strings.Join(node.Partitions, ","), 14, 11)
	features := truncateString(strings.Join(node.Features, ","), 19, 16)
	reason := truncateString(node.Reason, 24, 21)

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

// getNodeDisplayState determines the display state for a node
func (v *NodesView) getNodeDisplayState(node *dao.Node) string {
	displayState := node.State
	isDrainedByReason := node.Reason != "" && node.Reason != "Not responding"

	if node.State == "IDLE" && isDrainedByReason {
		displayState = "IDLE+DRAIN"
	}

	return displayState
}

// formatNodeCPUUsage formats CPU usage text for a node row
func (v *NodesView) formatNodeCPUUsage(node *dao.Node) string {
	cpuActualUsed := v.calculateCPUActualUsed(node)
	cpuUsage := v.createDualUsageBar(node.CPUsAllocated, cpuActualUsed, node.CPUsTotal)

	if node.CPULoad >= 0 {
		return fmt.Sprintf("%s %d/%d (%.1f)", cpuUsage, node.CPUsAllocated, node.CPUsTotal, node.CPULoad)
	}
	return fmt.Sprintf("%s %d/%d", cpuUsage, node.CPUsAllocated, node.CPUsTotal)
}

// calculateCPUActualUsed calculates actual CPU usage from load average
func (v *NodesView) calculateCPUActualUsed(node *dao.Node) int {
	cpuActualUsed := node.CPUsAllocated
	if node.CPULoad >= 0 {
		cpuActualUsed = int(node.CPULoad + 0.5)
		if cpuActualUsed > node.CPUsTotal {
			cpuActualUsed = node.CPUsTotal
		}
		if cpuActualUsed < 0 {
			cpuActualUsed = 0
		}
	}
	return cpuActualUsed
}

// formatNodeMemoryUsage formats memory usage text for a node row
func (v *NodesView) formatNodeMemoryUsage(node *dao.Node) string {
	memActualUsed := int(node.MemoryTotal - node.MemoryFree)
	if memActualUsed < 0 {
		memActualUsed = int(node.MemoryAllocated)
	}

	memUsage := v.createDualUsageBar(int(node.MemoryAllocated), memActualUsed, int(node.MemoryTotal))
	return fmt.Sprintf("%s %s/%s", memUsage, FormatMemory(node.MemoryAllocated), FormatMemory(node.MemoryTotal))
}

// truncateString truncates a string with ellipsis if it exceeds max length
func truncateString(s string, maxLen, truncateLen int) string {
	if len(s) > maxLen {
		return s[:truncateLen] + "..."
	}
	return s
}

/*
TODO(lint): Review unused code - func (*NodesView).createUsageBar is unused

createUsageBar creates a visual usage bar
func (v *NodesView) createUsageBar(used, total int) string {
	if total == 0 {
		return "[gray]········[white]"
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
		bar.WriteString("■")
	}

	// Add empty bars
	bar.WriteString("[gray]")
	for i := filled; i < barLength; i++ {
		bar.WriteString("·")
	}

	bar.WriteString("[white]")

	return bar.String()
}
*/

// createDualUsageBar creates a dual-view bar showing allocation and actual usage
// allocated: amount allocated by SLURM
// used: actual amount in use
// total: total available
func (v *NodesView) createDualUsageBar(allocated, used, total int) string {
	if total == 0 {
		return "[gray]········[white]"
	}

	allocPercentage := float64(allocated) / float64(total)
	usedPercentage := float64(used) / float64(total)
	barLength := 8
	allocFilled := int(allocPercentage * float64(barLength))
	usedFilled := int(usedPercentage * float64(barLength))

	var bar strings.Builder

	// Choose color based on allocation percentage
	var allocColor string
	switch {
	case allocPercentage < 0.5:
		allocColor = "green"
	case allocPercentage < 0.8:
		allocColor = "yellow"
	default:
		allocColor = "red"
	}

	// Build the bar using safe Unicode characters
	for i := 0; i < barLength; i++ {
		switch {
		case i < usedFilled:
			// Actual usage - solid block
			bar.WriteString(fmt.Sprintf("[%s]■[white]", allocColor))
		case i < allocFilled:
			// Allocated but not used - outlined square
			bar.WriteString(fmt.Sprintf("[%s]□[white]", allocColor))
		default:
			// Not allocated - middle dot
			bar.WriteString("[gray]·[white]")
		}
	}

	return bar.String()
}

/*
TODO(lint): Review unused code - func (*NodesView).updateStatusBar is unused

updateStatusBar updates the status bar
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
*/

// scheduleRefresh schedules the next refresh
func (v *NodesView) scheduleRefresh() {
	// Remove automatic refresh scheduling to prevent memory leak
	// Refresh will be handled by the main app refresh timer
}

// onNodeSelect handles node selection
func (v *NodesView) onNodeSelect(_, _ int) {
	data := v.table.GetSelectedData()
	if len(data) == 0 {
		return
	}

	// Note: Status bar update removed since individual view status bars are no longer used
	_ = data[0] // nodeName no longer used
}

// onSort handles column sorting
func (v *NodesView) onSort(_ int, _ bool) {
	// Note: Status bar update removed since individual view status bars are no longer used
}

// onFilterChange handles filter input changes
func (v *NodesView) onFilterChange(text string) {
	v.filter = text
	v.table.SetFilter(text)
	// Note: Status bar update removed since individual view status bars are no longer used
}

// onFilterDone handles filter input completion
func (v *NodesView) onFilterDone(_ tcell.Key) {
	if v.app != nil {
		v.app.SetFocus(v.table.Table)
	}
}

// drainSelectedNode drains the selected node
func (v *NodesView) drainSelectedNode() {
	debug.Logger.Printf("drainSelectedNode() called")
	data := v.table.GetSelectedData()
	if len(data) == 0 {
		debug.Logger.Printf("drainSelectedNode() - no data selected")
		return
	}

	nodeName := data[0] // Still used for drain operation
	state := data[1]

	// Check if node can be drained
	if strings.Contains(state, dao.NodeStateDown) {
		// Note: Status bar update removed since individual view status bars are no longer used
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
	err := v.client.Nodes().Drain(nodeName, reason)
	if err != nil {
		// Show error modal
		if v.pages != nil {
			errorModal := tview.NewModal().
				SetText(fmt.Sprintf("Failed to drain node %s: %v", nodeName, err)).
				AddButtons([]string{"OK"}).
				SetDoneFunc(func(_ int, _ string) {
					v.pages.RemovePage("error")
					v.app.SetFocus(v.table.Table)
				})
			v.pages.AddPage("error", errorModal, true, true)
		}
		return
	}

	// Show success message
	if v.pages != nil {
		successModal := tview.NewModal().
			SetText(fmt.Sprintf("Node %s drained successfully with reason: %s", nodeName, reason)).
			AddButtons([]string{"OK"}).
			SetDoneFunc(func(_ int, _ string) {
				v.pages.RemovePage("success")
				v.app.SetFocus(v.table.Table)
			})
		v.pages.AddPage("success", successModal, true, true)
	}

	// Refresh the view
	time.Sleep(500 * time.Millisecond)
	_ = v.Refresh()
}

// resumeSelectedNode resumes the selected node
func (v *NodesView) resumeSelectedNode() {
	debug.Logger.Printf("resumeSelectedNode() called")
	data := v.table.GetSelectedData()
	if len(data) == 0 {
		debug.Logger.Printf("resumeSelectedNode() - no data selected")
		return
	}

	nodeName := data[0]
	state := data[1]

	node := v.findNode(nodeName)
	if node == nil {
		debug.Logger.Printf("resumeSelectedNode() - node %s not found in node list", nodeName)
		return
	}

	cleanState := v.cleanNodeState(state)
	debug.Logger.Printf("resumeSelectedNode() - clean state: %s, reason: '%s'", cleanState, node.Reason)

	if !v.isNodeDrained(cleanState, node) {
		v.showResumeError(nodeName, cleanState)
		debug.Logger.Printf("resumeSelectedNode() - node %s cannot be resumed, state: %s, reason: %s", nodeName, cleanState, node.Reason)
		return
	}

	// Show confirmation dialog
	modal := tview.NewModal().
		SetText(fmt.Sprintf("Resume node %s?", nodeName)).
		AddButtons([]string{"Yes", "No"}).
		SetDoneFunc(func(buttonIndex int, _ string) {
			if buttonIndex == 0 {
				go v.performResumeNode(nodeName)
			}
			v.app.SetRoot(v.container, true)
		})

	v.app.SetRoot(modal, true)
}

// findNode finds a node by name in the node list
func (v *NodesView) findNode(nodeName string) *dao.Node {
	v.mu.RLock()
	defer v.mu.RUnlock()

	for _, n := range v.nodes {
		if n.Name == nodeName {
			return n
		}
	}
	return nil
}

// cleanNodeState removes color codes from node state string
func (v *NodesView) cleanNodeState(state string) string {
	colorCodes := []string{"[green]", "[white]", "[yellow]", "[red]", "[blue]", "[orange]"}
	cleanState := state

	for _, code := range colorCodes {
		cleanState = strings.ReplaceAll(cleanState, code, "")
	}

	return cleanState
}

// isNodeDrained checks if a node is in a drained state
func (v *NodesView) isNodeDrained(cleanState string, node *dao.Node) bool {
	return strings.Contains(cleanState, dao.NodeStateDrain) ||
		strings.Contains(cleanState, dao.NodeStateDraining) ||
		(node.Reason != "" && node.Reason != "Not responding")
}

// showResumeError shows an error modal for resume operation
func (v *NodesView) showResumeError(nodeName, state string) {
	if v.pages == nil {
		return
	}

	errorModal := tview.NewModal().
		SetText(fmt.Sprintf("Node %s is in state '%s' with no drain reason. Only drained nodes can be resumed.", nodeName, state)).
		AddButtons([]string{"OK"}).
		SetDoneFunc(func(_ int, _ string) {
			v.pages.RemovePage("error")
			v.app.SetFocus(v.table.Table)
		})
	v.pages.AddPage("error", errorModal, true, true)
}

// performResumeNode performs the node resume operation
func (v *NodesView) performResumeNode(nodeName string) {
	err := v.client.Nodes().Resume(nodeName)
	if err != nil {
		// Log the error for debugging
		if v.pages != nil {
			// Show error modal
			errorModal := tview.NewModal().
				SetText(fmt.Sprintf("Failed to resume node %s: %v", nodeName, err)).
				AddButtons([]string{"OK"}).
				SetDoneFunc(func(_ int, _ string) {
					v.pages.RemovePage("error")
					v.app.SetFocus(v.table.Table)
				})
			v.pages.AddPage("error", errorModal, true, true)
		}
		return
	}

	// Show success message
	if v.pages != nil {
		successModal := tview.NewModal().
			SetText(fmt.Sprintf("Node %s resumed successfully", nodeName)).
			AddButtons([]string{"OK"}).
			SetDoneFunc(func(_ int, _ string) {
				v.pages.RemovePage("success")
				v.app.SetFocus(v.table.Table)
			})
		v.pages.AddPage("success", successModal, true, true)
	}

	// Refresh the view
	time.Sleep(500 * time.Millisecond)
	_ = v.Refresh()
}

// showNodeDetails shows detailed information for the selected node
func (v *NodesView) showNodeDetails() {
	data := v.table.GetSelectedData()
	if len(data) == 0 {
		return
	}

	nodeName := data[0]

	// Fetch full node details
	node, err := v.client.Nodes().Get(nodeName)
	if err != nil {
		// Note: Status bar update removed since individual view status bars are no longer used
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

	v.writeCPUDetails(&details, node)
	v.writeMemoryDetails(&details, node)

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

// writeCPUDetails writes CPU information section to the details
func (v *NodesView) writeCPUDetails(w *strings.Builder, node *dao.Node) {
	w.WriteString("\n[teal]CPU Information:[white]\n")
	fmt.Fprintf(w, "[yellow]  Total CPUs:[white] %d\n", node.CPUsTotal)
	fmt.Fprintf(w, "[yellow]  Allocated CPUs:[white] %d\n", node.CPUsAllocated)
	fmt.Fprintf(w, "[yellow]  Idle CPUs:[white] %d\n", node.CPUsIdle)

	cpuAllocPercent := 0.0
	if node.CPUsTotal > 0 {
		cpuAllocPercent = float64(node.CPUsAllocated) * 100.0 / float64(node.CPUsTotal)
	}
	fmt.Fprintf(w, "[yellow]  CPU Allocation:[white] %.1f%% (SLURM allocated)\n", cpuAllocPercent)

	if node.CPULoad >= 0 {
		fmt.Fprintf(w, "[yellow]  CPU Load:[white] %.2f (1-minute load average)\n", node.CPULoad)
		if node.CPUsAllocated > 0 {
			efficiency := (node.CPULoad / float64(node.CPUsAllocated)) * 100.0
			if efficiency > 100 {
				efficiency = 100
			}
			fmt.Fprintf(w, "[yellow]  CPU Efficiency:[white] %.1f%% (load/allocation)\n", efficiency)
		}
	}
}

// writeMemoryDetails writes memory information section to the details
func (v *NodesView) writeMemoryDetails(w *strings.Builder, node *dao.Node) {
	w.WriteString("\n[teal]Memory Information:[white]\n")
	fmt.Fprintf(w, "[yellow]  Total Memory:[white] %s\n", FormatMemory(node.MemoryTotal))
	fmt.Fprintf(w, "[yellow]  Allocated Memory:[white] %s", FormatMemory(node.MemoryAllocated))

	memAllocPercent := 0.0
	if node.MemoryTotal > 0 {
		memAllocPercent = float64(node.MemoryAllocated) * 100.0 / float64(node.MemoryTotal)
	}
	fmt.Fprintf(w, " (%.1f%% allocated by SLURM)\n", memAllocPercent)

	memActualUsed := node.MemoryTotal - node.MemoryFree
	if memActualUsed < 0 {
		memActualUsed = node.MemoryAllocated
	}
	memUsagePercent := 0.0
	if node.MemoryTotal > 0 {
		memUsagePercent = float64(memActualUsed) * 100.0 / float64(node.MemoryTotal)
	}
	fmt.Fprintf(w, "[yellow]  Used Memory:[white] %s (%.1f%% actual usage)\n", FormatMemory(memActualUsed), memUsagePercent)
	fmt.Fprintf(w, "[yellow]  Free Memory:[white] %s\n", FormatMemory(node.MemoryFree))

	if node.MemoryAllocated > 0 {
		efficiency := float64(memActualUsed) * 100.0 / float64(node.MemoryAllocated)
		if efficiency > 100 {
			efficiency = 100
		}
		fmt.Fprintf(w, "[yellow]  Memory Efficiency:[white] %.1f%% (used/allocated)\n", efficiency)
	}
}

// sshToNode opens SSH connection to the selected node
func (v *NodesView) sshToNode() {
	data := v.table.GetSelectedData()
	if len(data) == 0 {
		return
	}

	nodeName := data[0]
	nodeState := data[2]

	// Check if SSH is available
	if !ssh.IsSSHAvailable() {
		v.showError("SSH command not found. Please install OpenSSH client.")
		return
	}

	// Check if node is in a good state for SSH
	if strings.Contains(nodeState, "DOWN") || strings.Contains(nodeState, "DRAIN") {
		message := fmt.Sprintf("Node %s is in state %s. SSH may not work.\n\nDo you want to try anyway?", nodeName, nodeState)
		v.confirmSSH(nodeName, message)
		return
	}

	// Proceed with SSH
	v.performSSH(nodeName)
}

// confirmSSH shows confirmation dialog for SSH to problematic nodes
func (v *NodesView) confirmSSH(nodeName, message string) {
	modal := tview.NewModal()
	modal.SetText(message)
	modal.AddButtons([]string{"Yes", "No"})
	modal.SetDoneFunc(func(buttonIndex int, _ string) {
		v.pages.RemovePage("ssh-confirm")
		if buttonIndex == 0 { // Yes
			v.performSSH(nodeName)
		}
	})

	if v.pages != nil {
		v.pages.AddPage("ssh-confirm", modal, true, true)
	}
}

// performSSH initiates SSH connection to the node
func (v *NodesView) performSSH(nodeName string) {
	if v.sshClient == nil {
		v.showError("SSH client not initialized")
		return
	}

	// Show SSH connection modal with options
	v.showSSHOptionsModal(nodeName)
}

// showSSHOptionsModal shows SSH connection options
func (v *NodesView) showSSHOptionsModal(nodeName string) {
	list := tview.NewList()
	list.SetBorder(true)
	list.SetTitle(fmt.Sprintf(" SSH to %s ", nodeName))
	list.SetTitleAlign(tview.AlignCenter)

	// Add SSH options
	list.AddItem("🖥  SSH Terminal Manager", "Advanced SSH session management", 't', func() {
		v.pages.RemovePage("ssh-options")
		v.showSSHTerminalManager(nodeName)
	})

	list.AddItem("⚡ Quick Connect", "Direct SSH connection", 'q', func() {
		v.pages.RemovePage("ssh-options")
		v.sshToTerminal(nodeName)
	})

	list.AddItem("🔍 Test Connection", "Test SSH connectivity", 'c', func() {
		v.pages.RemovePage("ssh-options")
		v.testSSHConnection(nodeName)
	})

	list.AddItem("ℹ  Get Node Info", "Retrieve detailed node information", 'i', func() {
		v.pages.RemovePage("ssh-options")
		v.getNodeInfoViaSSH(nodeName)
	})

	list.AddItem("❌ Cancel", "Cancel SSH operation", 'x', func() {
		v.pages.RemovePage("ssh-options")
	})

	// Create modal
	modal := tview.NewFlex()
	modal.AddItem(nil, 0, 1, false)
	modal.AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(nil, 0, 1, false).
		AddItem(list, 12, 0, true).
		AddItem(nil, 0, 1, false), 40, 0, true)
	modal.AddItem(nil, 0, 1, false)

	// Handle keyboard shortcuts
	list.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEsc {
			v.pages.RemovePage("ssh-options")
			return nil
		}
		return event
	})

	if v.pages != nil {
		v.pages.AddPage("ssh-options", modal, true, true)
		v.app.SetFocus(list)
	}
}

// sshToTerminal opens SSH in a new terminal (placeholder implementation)
func (v *NodesView) sshToTerminal(nodeName string) {
	// This would actually open SSH in external terminal
	// For now, show a notification about the SSH command
	message := fmt.Sprintf("SSH command to run:\n\nssh %s\n\nNote: This would normally open in a new terminal window.", nodeName)
	v.showNotification("SSH Terminal", message)
}

// testSSHConnection tests SSH connectivity to the node
func (v *NodesView) testSSHConnection(nodeName string) {
	v.showProgressDialog("Testing SSH connection to " + nodeName + "...")

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		err := v.sshClient.TestConnection(ctx, nodeName)

		v.app.QueueUpdateDraw(func() {
			v.pages.RemovePage("progress")

			if err != nil {
				v.showError(fmt.Sprintf("SSH connection failed: %v", err))
			} else {
				v.showNotification("SSH Test", fmt.Sprintf("SSH connection to %s successful!", nodeName))
			}
		})
	}()
}

// getNodeInfoViaSSH retrieves node information via SSH
func (v *NodesView) getNodeInfoViaSSH(nodeName string) {
	v.showProgressDialog("Retrieving node information from " + nodeName + "...")

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		info, err := v.sshClient.GetNodeInfo(ctx, nodeName)

		v.app.QueueUpdateDraw(func() {
			v.pages.RemovePage("progress")

			if err != nil {
				v.showError(fmt.Sprintf("Failed to get node info: %v", err))
			} else {
				v.showNodeInfoModal(nodeName, info)
			}
		})
	}()
}

// showNodeInfoModal displays node information retrieved via SSH
func (v *NodesView) showNodeInfoModal(nodeName string, info map[string]string) {
	var content strings.Builder
	content.WriteString(fmt.Sprintf("[yellow]Node Information for %s[white]\n\n", nodeName))

	for key, value := range info {
		content.WriteString(fmt.Sprintf("[blue]%s:[white] %s\n", key, value))
	}

	textView := tview.NewTextView()
	textView.SetDynamicColors(true)
	textView.SetText(content.String())
	textView.SetBorder(true)
	textView.SetTitle(fmt.Sprintf(" %s Info ", nodeName))
	textView.SetTitleAlign(tview.AlignCenter)

	// Create modal
	modal := tview.NewFlex()
	modal.AddItem(nil, 0, 1, false)
	modal.AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(nil, 0, 1, false).
		AddItem(textView, 0, 3, true).
		AddItem(nil, 0, 1, false), 0, 3, true)
	modal.AddItem(nil, 0, 1, false)

	textView.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEsc {
			v.pages.RemovePage("node-info")
			return nil
		}
		return event
	})

	if v.pages != nil {
		v.pages.AddPage("node-info", modal, true, true)
		v.app.SetFocus(textView)
	}
}

// showProgressDialog shows a progress dialog
func (v *NodesView) showProgressDialog(message string) {
	modal := tview.NewModal()
	modal.SetText(message)
	modal.SetBorder(true)
	modal.SetTitle(" Working... ")
	modal.SetTitleAlign(tview.AlignCenter)

	if v.pages != nil {
		v.pages.AddPage("progress", modal, true, true)
	}
}

// showNotification shows a notification dialog
func (v *NodesView) showNotification(_, message string) {
	modal := tview.NewModal()
	modal.SetText(message)
	modal.AddButtons([]string{"OK"})
	modal.SetDoneFunc(func(_ int, _ string) {
		v.pages.RemovePage("notification")
	})

	if v.pages != nil {
		v.pages.AddPage("notification", modal, true, true)
	}
}

// showError shows an error dialog
func (v *NodesView) showError(message string) {
	modal := tview.NewModal()
	modal.SetText("[red]Error:[white] " + message)
	modal.AddButtons([]string{"OK"})
	modal.SetDoneFunc(func(_ int, _ string) {
		v.pages.RemovePage("error")
	})

	if v.pages != nil {
		v.pages.AddPage("error", modal, true, true)
	}
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

	go func() { _ = v.Refresh() }()
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
			go func() { _ = v.Refresh() }()
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
		text := cases.Title(language.English).String(option)
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
	// Note: Status bar update removed since individual view status bars are no longer used
}

// toggleGroupExpansion toggles expansion of the selected group
func (v *NodesView) toggleGroupExpansion() {
	if v.groupBy == "none" {
		return
	}

	data := v.table.GetSelectedData()
	if len(data) == 0 {
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
		AddItem(v.table, 0, 1, false)

	v.filterBar.Show()
	// Note: Advanced filter status removed since individual view status bars are no longer used
}

// closeAdvancedFilter closes the advanced filter bar
func (v *NodesView) closeAdvancedFilter() {
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
func (v *NodesView) onAdvancedFilterChange(filter *filters.Filter) {
	v.advancedFilter = filter
	v.updateTable()

	// Note: Status bar updates removed since individual view status bars are no longer used
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
		"Name":            node.Name,
		"State":           node.State,
		"CPUsAllocated":   node.CPUsAllocated,
		"CPUsTotal":       node.CPUsTotal,
		"MemoryAllocated": node.MemoryAllocated,
		"MemoryTotal":     node.MemoryTotal,
		"Features":        strings.Join(node.Features, ","),
		"Partitions":      strings.Join(node.Partitions, ","),
		"Reason":          node.Reason,
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
			// Note: Status bar update removed since individual view status bars are no longer used
		}
	})
}

// showSSHTerminalManager shows the SSH terminal manager interface
func (v *NodesView) showSSHTerminalManager(nodeName string) {
	if v.sshTerminal == nil {
		v.showError("SSH terminal not initialized")
		return
	}

	// Get all node names for the SSH terminal
	v.mu.RLock()
	nodeNames := make([]string, len(v.nodes))
	for i, node := range v.nodes {
		nodeNames[i] = node.Name
	}
	v.mu.RUnlock()

	// Set the available nodes
	v.sshTerminal.SetNodes(nodeNames)

	// Show the SSH terminal interface with the selected node
	v.sshTerminal.ShowSSHInterface(nodeName)
}

// focusOnNode focuses the table on a specific node
func (v *NodesView) focusOnNode(nodeName string) {
	v.mu.RLock()
	defer v.mu.RUnlock()

	// Find the node in our node list
	for i, node := range v.nodes {
		if node.Name == nodeName {
			// Select the row in the table
			v.table.Select(i, 0)
			// Note: Status bar update removed since individual view status bars are no longer used
			return
		}
	}

	// Note: Status bar update removed since individual view status bars are no longer used
}
