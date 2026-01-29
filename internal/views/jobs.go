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

// JobsView displays the jobs list
type JobsView struct {
	*BaseView
	client              dao.SlurmClient
	table               *components.MultiSelectTable
	jobs                []*dao.Job
	mu                  sync.RWMutex
	refreshTimer        *time.Timer
	refreshRate         time.Duration
	filter              string
	stateFilter         []string
	userFilter          string
	container           *tview.Flex
	filterInput         *tview.InputField
	statusBar           *tview.TextView
	app                 *tview.Application
	pages               *tview.Pages
	templateManager     *JobTemplateManager
	autoRefresh         bool
	selectedJobs        map[string]bool
	filterBar           *components.FilterBar
	advancedFilter      *filters.Filter
	isAdvancedMode      bool
	globalSearch        *GlobalSearch
	jobOutputView       *JobOutputView
	batchOpsView        *BatchOperationsView
	multiSelectMode     bool
	selectionStatusText *tview.TextView
	loadingManager      *components.LoadingManager
	loadingWrapper      *components.LoadingWrapper
	mainStatusBar       *components.StatusBar // Reference to main app status bar
}

// SetPages sets the pages reference for modal handling
func (v *JobsView) SetPages(pages *tview.Pages) {
	v.pages = pages
	// Set pages for filter bar if it exists
	if v.filterBar != nil {
		v.filterBar.SetPages(pages)
	}
	// Set pages for other views
	if v.jobOutputView != nil {
		v.jobOutputView.SetPages(pages)
	}
	if v.batchOpsView != nil {
		v.batchOpsView.SetPages(pages)
	}
}

// SetApp sets the application reference
func (v *JobsView) SetApp(app *tview.Application) {
	v.app = app

	// Initialize loading manager
	if v.pages != nil {
		v.loadingManager = components.NewLoadingManager(app, v.pages)
		v.loadingWrapper = components.NewLoadingWrapper(v.loadingManager, "jobs")
	}

	// Create filter bar now that we have app reference
	v.filterBar = components.NewFilterBar("jobs", app)
	v.filterBar.SetPages(v.pages)
	v.filterBar.SetOnFilterChange(v.onAdvancedFilterChange)
	v.filterBar.SetOnClose(v.closeAdvancedFilter)

	// Create global search
	v.globalSearch = NewGlobalSearch(v.client, app)

	// Create job output view
	v.jobOutputView = NewJobOutputView(v.client, app)

	// Create batch operations view
	v.batchOpsView = NewBatchOperationsView(v.client, app)
}

// SetStatusBar sets the main status bar reference
func (v *JobsView) SetStatusBar(statusBar *components.StatusBar) {
	v.mainStatusBar = statusBar
}

// NewJobsView creates a new jobs view
func NewJobsView(client dao.SlurmClient) *JobsView {
	v := &JobsView{
		BaseView:     NewBaseView("jobs", "Jobs"),
		client:       client,
		refreshRate:  30 * time.Second,
		jobs:         []*dao.Job{},
		autoRefresh:  true,
		selectedJobs: make(map[string]bool),
	}

	// Create table with job columns
	columns := []components.Column{
		components.NewColumn("ID").Width(10).Build(),
		components.NewColumn("Name").Width(20).Build(),
		components.NewColumn("User").Width(10).Build(),
		components.NewColumn("Account").Width(12).Build(),
		components.NewColumn("State").Width(12).Sortable(true).Build(),
		components.NewColumn("Partition").Width(10).Build(),
		components.NewColumn("Nodes").Width(8).Align(tview.AlignRight).Build(),
		components.NewColumn("Time").Width(10).Align(tview.AlignRight).Build(),
		components.NewColumn("Time Limit").Width(10).Align(tview.AlignRight).Build(),
		components.NewColumn("Priority").Width(8).Align(tview.AlignRight).Sortable(true).Build(),
		components.NewColumn("Submit Time").Width(19).Sortable(true).Build(),
	}

	// Create multi-select table
	config := components.DefaultTableConfig()
	config.Columns = columns
	config.Selectable = true
	config.ShowHeader = true
	config.SelectedColor = tcell.ColorYellow
	config.HeaderColor = tcell.ColorTeal
	config.BorderColor = tcell.ColorWhite

	v.table = components.NewMultiSelectTable(config)

	// Set up callbacks
	v.table.SetOnSelect(v.onJobSelect)
	v.table.SetOnSort(v.onSort)
	v.table.SetOnSelectionChange(v.onSelectionChange)
	v.table.SetOnRowToggle(v.onRowToggle)

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

	// Create selection status text
	v.selectionStatusText = tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignRight).
		SetText("[gray]Multi-select: Off[white]")

	// Create info bar with filter and selection status
	infoBar := tview.NewFlex().
		SetDirection(tview.FlexColumn).
		AddItem(v.filterInput, 0, 2, false).
		AddItem(v.selectionStatusText, 0, 1, false)

	// Create container layout (removed individual status bar to prevent conflicts with main status bar)
	// Use infoBar but ensure it has proper sizing
	v.container = tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(infoBar, 1, 0, false).
		AddItem(v.table, 0, 1, true)

	return v
}

// Init initializes the jobs view
func (v *JobsView) Init(ctx context.Context) error {
	_ = v.BaseView.Init(ctx)
	return v.Refresh()
}

// Render returns the view's main component
func (v *JobsView) Render() tview.Primitive {
	return v.container
}

// Refresh updates the jobs data
func (v *JobsView) Refresh() error {
	v.SetRefreshing(true)
	defer v.SetRefreshing(false)

	// Show loading indicator for operations that might take time
	if v.loadingWrapper != nil {
		return v.loadingWrapper.WithLoading("Refreshing jobs...", func() error {
			return v.refreshInternal()
		})
	}

	return v.refreshInternal()
}

// refreshInternal performs the actual refresh operation
func (v *JobsView) refreshInternal() error {
	debug.Logger.Printf("Jobs refreshInternal() started at %s", time.Now().Format("15:04:05.000"))
	// Fetch jobs from backend
	opts := &dao.ListJobsOptions{
		States: v.stateFilter,
		Limit:  1000, // TODO: Add pagination
	}

	if v.userFilter != "" {
		opts.Users = []string{v.userFilter}
	}

	jobList, err := v.client.Jobs().List(opts)
	debug.Logger.Printf("Jobs client.List() finished at %s", time.Now().Format("15:04:05.000"))
	if err != nil {
		v.SetLastError(err)
		// Note: Error handling removed since individual view status bars are no longer used
		return err
	}

	v.mu.Lock()
	v.jobs = jobList.Jobs
	v.mu.Unlock()
	debug.Logger.Printf("Jobs data stored, calling updateTable() at %s", time.Now().Format("15:04:05.000"))

	// Update table
	v.updateTable()
	debug.Logger.Printf("Jobs updateTable() finished at %s", time.Now().Format("15:04:05.000"))
	// Note: No longer updating individual view status bar since we use main app status bar for hints

	// Schedule next refresh
	v.scheduleRefresh()

	return nil
}

// Stop stops the view
func (v *JobsView) Stop() error {
	v.mu.Lock()
	defer v.mu.Unlock()

	if v.refreshTimer != nil {
		v.refreshTimer.Stop()
	}
	return nil
}

// Hints returns keyboard hints
func (v *JobsView) Hints() []string {
	hints := []string{
		"[yellow]Enter[white] Details",
		"[yellow]s[white] Submit Job",
		"[yellow]F2[white] Templates",
		"[yellow]c[white] Cancel",
		"[yellow]h[white] Hold",
		"[yellow]r[white] Release",
		"[yellow]o[white] Output",
		"[yellow]d[white] Dependencies",
		"[yellow]q[white] Requeue",
		"[yellow]b[white] Batch Ops",
		"[yellow]m[white] Monitor",
		"[yellow]/[white] Filter",
		"[yellow]F3[white] Adv Filter",
		"[yellow]Ctrl+F[white] Search",
		"[yellow]F1[white] Actions Menu",
		"[yellow]v[white] Multi-Select",
		"[yellow]R[white] Refresh",
	}

	if v.isAdvancedMode {
		hints = append([]string{"[yellow]ESC[white] Exit Adv Filter"}, hints...)
	}

	// Add multi-select specific hints when in multi-select mode
	if v.multiSelectMode {
		multiSelectHints := v.table.GetMultiSelectHints()
		hints = append(hints, multiSelectHints...)
	}

	return hints
}

// OnKey handles keyboard events
func (v *JobsView) OnKey(event *tcell.EventKey) *tcell.EventKey {
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
	if v.pages != nil && v.pages.GetPageCount() > 1 {
		return event // Let modal handle it
	}

	// Handle advanced filter mode
	if v.isAdvancedMode && event.Key() == tcell.KeyEsc {
		v.closeAdvancedFilter()
		return nil
	}

	// Handle by special key first
	handlers := v.jobsKeyHandlers()
	if handler, ok := handlers[event.Key()]; ok {
		return handler(v, event)
	}

	// Handle by key type
	if event.Key() == tcell.KeyRune {
		return v.handleJobsViewRune(event)
	}

	return event
}

// handleJobsViewRune handles rune key presses in the jobs view
func (v *JobsView) handleJobsViewRune(event *tcell.EventKey) *tcell.EventKey {
	handler, ok := v.jobsRuneHandlers()[event.Rune()]
	if ok {
		return handler(v, event)
	}
	return event
}

// jobsKeyHandlers returns a map of special keys to their handlers
func (v *JobsView) jobsKeyHandlers() map[tcell.Key]func(*JobsView, *tcell.EventKey) *tcell.EventKey {
	return map[tcell.Key]func(*JobsView, *tcell.EventKey) *tcell.EventKey{
		tcell.KeyF1:    func(v *JobsView, _ *tcell.EventKey) *tcell.EventKey { v.showJobActions(); return nil },
		tcell.KeyF2:    func(v *JobsView, _ *tcell.EventKey) *tcell.EventKey { v.showJobTemplateSelector(); return nil },
		tcell.KeyF3:    func(v *JobsView, _ *tcell.EventKey) *tcell.EventKey { v.showAdvancedFilter(); return nil },
		tcell.KeyCtrlF: func(v *JobsView, _ *tcell.EventKey) *tcell.EventKey { v.showGlobalSearch(); return nil },
		tcell.KeyEnter: func(v *JobsView, _ *tcell.EventKey) *tcell.EventKey { v.showJobDetails(); return nil },
	}
}

// jobsRuneHandlers returns a map of rune keys to their handlers
func (v *JobsView) jobsRuneHandlers() map[rune]func(*JobsView, *tcell.EventKey) *tcell.EventKey {
	return map[rune]func(*JobsView, *tcell.EventKey) *tcell.EventKey{
		'c': func(v *JobsView, _ *tcell.EventKey) *tcell.EventKey { v.cancelSelectedJob(); return nil },
		'C': func(v *JobsView, _ *tcell.EventKey) *tcell.EventKey { v.cancelSelectedJob(); return nil },
		'h': func(v *JobsView, _ *tcell.EventKey) *tcell.EventKey { v.holdSelectedJob(); return nil },
		'H': func(v *JobsView, _ *tcell.EventKey) *tcell.EventKey { v.holdSelectedJob(); return nil },
		'r': func(v *JobsView, _ *tcell.EventKey) *tcell.EventKey { v.releaseSelectedJob(); return nil },
		'R': func(v *JobsView, _ *tcell.EventKey) *tcell.EventKey { go func() { _ = v.Refresh() }(); return nil },
		'o': func(v *JobsView, _ *tcell.EventKey) *tcell.EventKey { v.showJobOutput(); return nil },
		'O': func(v *JobsView, _ *tcell.EventKey) *tcell.EventKey { v.showJobOutput(); return nil },
		's': func(v *JobsView, _ *tcell.EventKey) *tcell.EventKey { v.showJobSubmissionForm(); return nil },
		'S': func(v *JobsView, _ *tcell.EventKey) *tcell.EventKey { v.showJobSubmissionForm(); return nil },
		'd': func(v *JobsView, _ *tcell.EventKey) *tcell.EventKey { v.showJobDependencies(); return nil },
		'D': func(v *JobsView, _ *tcell.EventKey) *tcell.EventKey { v.showJobDependencies(); return nil },
		'b': func(v *JobsView, _ *tcell.EventKey) *tcell.EventKey { v.showBatchOperations(); return nil },
		'B': func(v *JobsView, _ *tcell.EventKey) *tcell.EventKey { v.showBatchOperations(); return nil },
		'm': func(v *JobsView, _ *tcell.EventKey) *tcell.EventKey { v.toggleAutoRefresh(); return nil },
		'M': func(v *JobsView, _ *tcell.EventKey) *tcell.EventKey { v.toggleAutoRefresh(); return nil },
		'q': func(v *JobsView, _ *tcell.EventKey) *tcell.EventKey { v.requeueSelectedJob(); return nil },
		'Q': func(v *JobsView, _ *tcell.EventKey) *tcell.EventKey { v.requeueSelectedJob(); return nil },
		'/': func(v *JobsView, _ *tcell.EventKey) *tcell.EventKey { v.app.SetFocus(v.filterInput); return nil },
		'a': func(v *JobsView, _ *tcell.EventKey) *tcell.EventKey { v.toggleStateFilter("all"); return nil },
		'A': func(v *JobsView, _ *tcell.EventKey) *tcell.EventKey { v.toggleStateFilter("all"); return nil },
		'p': func(v *JobsView, _ *tcell.EventKey) *tcell.EventKey {
			v.toggleStateFilter(dao.JobStatePending)
			return nil
		},
		'P': func(v *JobsView, _ *tcell.EventKey) *tcell.EventKey {
			v.toggleStateFilter(dao.JobStatePending)
			return nil
		},
		'u': func(v *JobsView, _ *tcell.EventKey) *tcell.EventKey { v.promptUserFilter(); return nil },
		'U': func(v *JobsView, _ *tcell.EventKey) *tcell.EventKey { v.promptUserFilter(); return nil },
		'v': func(v *JobsView, _ *tcell.EventKey) *tcell.EventKey { v.toggleMultiSelectMode(); return nil },
		'V': func(v *JobsView, _ *tcell.EventKey) *tcell.EventKey { v.toggleMultiSelectMode(); return nil },
	}
}

// OnFocus handles focus events
func (v *JobsView) OnFocus() error {
	if v.app != nil {
		v.app.SetFocus(v.table.Table)
	}
	return nil
}

// onSelectionChange handles multi-select selection changes
func (v *JobsView) onSelectionChange(selectedCount int, allSelected bool) {
	// Update selection status text
	if v.multiSelectMode {
		var statusText string
		switch {
		case selectedCount == 0:
			statusText = "[green]Multi-select: On[white] | [gray]0 selected[white]"
		case allSelected:
			statusText = fmt.Sprintf("[green]Multi-select: On[white] | [yellow]%d selected (all)[white]", selectedCount)
		default:
			statusText = fmt.Sprintf("[green]Multi-select: On[white] | [yellow]%d selected[white]", selectedCount)
		}
		if v.selectionStatusText != nil {
			v.selectionStatusText.SetText(statusText)
		}
	}

	// Sync with legacy selectedJobs map for compatibility with existing batch operations
	v.syncSelectedJobs()
}

// onRowToggle handles individual row selection toggles
func (v *JobsView) onRowToggle(_ int, selected bool, data []string) {
	if len(data) > 0 {
		jobID := data[0] // First column is job ID
		if selected {
			v.selectedJobs[jobID] = true
		} else {
			delete(v.selectedJobs, jobID)
		}
	}
}

// syncSelectedJobs synchronizes the legacy selectedJobs map with multi-select table
func (v *JobsView) syncSelectedJobs() {
	v.selectedJobs = make(map[string]bool)
	selectedData := v.table.GetAllSelectedData()
	for _, rowData := range selectedData {
		if len(rowData) > 0 {
			jobID := rowData[0] // First column is job ID
			v.selectedJobs[jobID] = true
		}
	}
}

// toggleMultiSelectMode toggles multi-select mode on/off
func (v *JobsView) toggleMultiSelectMode() {
	v.multiSelectMode = !v.multiSelectMode
	v.table.SetMultiSelectMode(v.multiSelectMode)

	if v.multiSelectMode {
		v.selectionStatusText.SetText("[green]Multi-select: On[white] | [gray]0 selected[white]")
	} else {
		v.selectionStatusText.SetText("[gray]Multi-select: Off[white]")
		v.selectedJobs = make(map[string]bool) // Clear selections when disabling
	}
}

// OnLoseFocus handles loss of focus
func (v *JobsView) OnLoseFocus() error {
	return nil
}

// updateTable updates the table with current job data
func (v *JobsView) updateTable() {
	v.mu.RLock()
	defer v.mu.RUnlock()

	// Apply advanced filter if active
	filteredJobs := v.jobs
	if v.advancedFilter != nil && len(v.advancedFilter.Expressions) > 0 {
		filteredJobs = v.applyAdvancedFilter(v.jobs)
	}

	data := make([][]string, len(filteredJobs))
	for i, job := range filteredJobs {
		stateColor := dao.GetJobStateColor(job.State)
		coloredState := fmt.Sprintf("[%s]%s[white]", stateColor, job.State)

		timeUsed := job.TimeUsed
		if timeUsed == "" && job.StartTime != nil {
			timeUsed = FormatDurationDetailed(time.Since(*job.StartTime))
		}

		priority := fmt.Sprintf("%.0f", job.Priority)
		submitTime := job.SubmitTime.Format("2006-01-02 15:04:05")

		data[i] = []string{
			job.ID,
			job.Name,
			job.User,
			job.Account,
			coloredState,
			job.Partition,
			fmt.Sprintf("%d", job.NodeCount),
			timeUsed,
			job.TimeLimit,
			priority,
			submitTime,
		}
	}

	v.table.SetData(data)
}

/*
TODO(lint): Review unused code - func (*JobsView).updateStatusBar is unused

updateStatusBar updates the status bar
func (v *JobsView) updateStatusBar(message string) {
	if message != "" {
		v.statusBar.SetText(message)
		return
	}

	v.mu.RLock()
	total := len(v.jobs)
	v.mu.RUnlock()

	filtered := len(v.table.GetFilteredData())

	status := fmt.Sprintf("Total: %d", total)
	if filtered < total {
		status += fmt.Sprintf(" | Filtered: %d", filtered)
	}

	if len(v.selectedJobs) > 0 {
		status += fmt.Sprintf(" | Selected: %d", len(v.selectedJobs))
	}

	if v.autoRefresh {
		status += " | [green]Auto-refresh: ON[white]"
	} else {
		status += " | [yellow]Auto-refresh: OFF[white]"
	}

	if v.IsRefreshing() {
		status += " | [yellow]Refreshing...[white]"
	}

	v.statusBar.SetText(status)
}
*/

// scheduleRefresh schedules the next refresh
func (v *JobsView) scheduleRefresh() {
	v.mu.Lock()
	defer v.mu.Unlock()

	// Only schedule if auto-refresh is enabled
	if !v.autoRefresh {
		return
	}

	if v.refreshTimer != nil {
		v.refreshTimer.Stop()
	}

	v.refreshTimer = time.AfterFunc(v.refreshRate, func() {
		go func() { _ = v.Refresh() }()
	})
}

// onJobSelect handles job selection
func (v *JobsView) onJobSelect(_, _ int) {
	// Get selected job data
	data := v.table.GetSelectedData()
	if len(data) == 0 {
		return
	}

	// Note: Selection handling removed since individual view status bars are no longer used
	_ = data[0] // jobID no longer used
}

// onSort handles column sorting
func (v *JobsView) onSort(_ int, _ bool) {
	// Sorting is handled by the table component
	// Note: Sort feedback removed since individual view status bars are no longer used
}

// onFilterChange handles filter input changes
func (v *JobsView) onFilterChange(text string) {
	v.filter = text
	v.table.SetFilter(text)
	// Note: Status bar update removed since individual view status bars are no longer used
}

// onFilterDone handles filter input completion
func (v *JobsView) onFilterDone(_ tcell.Key) {
	if v.app != nil {
		v.app.SetFocus(v.table.Table)
	}
}

// cancelSelectedJob cancels the selected job
func (v *JobsView) cancelSelectedJob() {
	data := v.table.GetSelectedData()
	if len(data) == 0 {
		debug.Logger.Printf("cancelSelectedJob() - no data selected")
		return
	}

	jobID := data[0]
	jobName := data[1]
	state := data[4] // State column
	debug.Logger.Printf("cancelSelectedJob() - job: %s, name: %s, state: %s", jobID, jobName, state)

	// Clean color codes from state
	cleanState := strings.ReplaceAll(strings.ReplaceAll(state, "[green]", ""), "[white]", "")
	cleanState = strings.ReplaceAll(strings.ReplaceAll(cleanState, "[yellow]", ""), "[red]", "")
	cleanState = strings.ReplaceAll(strings.ReplaceAll(cleanState, "[blue]", ""), "[orange]", "")
	cleanState = strings.ReplaceAll(strings.ReplaceAll(cleanState, "[cyan]", ""), "[gray]", "")
	debug.Logger.Printf("cancelSelectedJob() - clean state: %s", cleanState)

	// Check if job can be canceled
	if !strings.Contains(cleanState, dao.JobStateRunning) && !strings.Contains(cleanState, dao.JobStatePending) {
		if v.mainStatusBar != nil {
			v.mainStatusBar.Warning(fmt.Sprintf("Job %s is not in a cancellable state (current: %s)", jobID, cleanState))
		}
		return
	}

	// Show confirmation dialog with more context
	confirmText := fmt.Sprintf("Cancel job %s (%s)?\n\nThis will terminate the job immediately.\nThis action cannot be undone.", jobID, jobName)
	modal := tview.NewModal().
		SetText(confirmText).
		AddButtons([]string{"Cancel Job", "Keep Running"}).
		SetDoneFunc(func(buttonIndex int, _ string) {
			if v.pages != nil {
				v.pages.RemovePage("cancel-confirmation")
			}
			if buttonIndex == 0 {
				v.performCancelJob(jobID)
			}
		})

	modal.SetBorder(true).
		SetTitle(" Confirm Job Cancellation ").
		SetTitleAlign(tview.AlignCenter)

	if v.pages != nil {
		v.pages.AddPage("cancel-confirmation", modal, true, true)
	}
}

// performCancelJob performs the job cancel operation
func (v *JobsView) performCancelJob(jobID string) {
	debug.Logger.Printf("performCancelJob() - starting for job %s", jobID)
	if v.mainStatusBar != nil {
		v.mainStatusBar.Info(fmt.Sprintf("Canceling job %s...", jobID))
	}

	// Attempt to cancel
	debug.Logger.Printf("Calling Cancel API for job %s", jobID)
	err := v.client.Jobs().Cancel(jobID)
	if err != nil {
		debug.Logger.Printf("Cancel API failed for job %s: %v", jobID, err)
		if v.mainStatusBar != nil {
			v.mainStatusBar.Error(fmt.Sprintf("Failed to cancel job %s: %v", jobID, err))
		}
		return
	}

	debug.Logger.Printf("Cancel API returned success for job %s - refreshing view", jobID)

	// If the API call succeeded, assume the cancel worked and show success
	// The API returns success, so we trust it and refresh
	if v.mainStatusBar != nil {
		v.mainStatusBar.Success(fmt.Sprintf("Job %s canceled", jobID))
	}

	// Refresh the view to get updated state
	time.Sleep(500 * time.Millisecond)
	debug.Logger.Printf("Refreshing jobs view after cancel")
	_ = v.Refresh()
}

// holdSelectedJob holds the selected job
func (v *JobsView) holdSelectedJob() {
	data := v.table.GetSelectedData()
	if len(data) == 0 {
		debug.Logger.Printf("holdSelectedJob() - no data selected")
		return
	}

	jobID := data[0]
	state := data[4] // State column
	debug.Logger.Printf("holdSelectedJob() - job: %s, state: %s", jobID, state)

	// Clean color codes from state
	cleanState := strings.ReplaceAll(strings.ReplaceAll(state, "[green]", ""), "[white]", "")
	cleanState = strings.ReplaceAll(strings.ReplaceAll(cleanState, "[yellow]", ""), "[red]", "")
	cleanState = strings.ReplaceAll(strings.ReplaceAll(cleanState, "[blue]", ""), "[orange]", "")
	cleanState = strings.ReplaceAll(strings.ReplaceAll(cleanState, "[cyan]", ""), "[gray]", "")
	debug.Logger.Printf("holdSelectedJob() - clean state: %s", cleanState)

	// Check if job can be held
	if !strings.Contains(cleanState, dao.JobStatePending) {
		if v.mainStatusBar != nil {
			v.mainStatusBar.Warning(fmt.Sprintf("Job %s is not in a holdable state (current: %s)", jobID, cleanState))
		}
		return
	}

	// Perform hold
	err := v.client.Jobs().Hold(jobID)
	if err != nil {
		if v.mainStatusBar != nil {
			v.mainStatusBar.Error(fmt.Sprintf("Failed to hold job %s: %v", jobID, err))
		}
		return
	}

	if v.mainStatusBar != nil {
		v.mainStatusBar.Success(fmt.Sprintf("Job %s held", jobID))
	}

	// Refresh the view
	time.Sleep(500 * time.Millisecond)
	_ = v.Refresh()
}

// releaseSelectedJob releases the selected job
func (v *JobsView) releaseSelectedJob() {
	data := v.table.GetSelectedData()
	if len(data) == 0 {
		debug.Logger.Printf("releaseSelectedJob() - no data selected")
		return
	}

	jobID := data[0]
	state := data[4] // State column
	debug.Logger.Printf("releaseSelectedJob() - job: %s, state: %s", jobID, state)

	// Clean color codes from state
	cleanState := strings.ReplaceAll(strings.ReplaceAll(state, "[green]", ""), "[white]", "")
	cleanState = strings.ReplaceAll(strings.ReplaceAll(cleanState, "[yellow]", ""), "[red]", "")
	cleanState = strings.ReplaceAll(strings.ReplaceAll(cleanState, "[blue]", ""), "[orange]", "")
	cleanState = strings.ReplaceAll(strings.ReplaceAll(cleanState, "[cyan]", ""), "[gray]", "")
	debug.Logger.Printf("releaseSelectedJob() - clean state: %s", cleanState)

	// Check if job can be released
	// Jobs can be released if they are SUSPENDED or PENDING (held)
	if !strings.Contains(cleanState, dao.JobStateSuspended) && !strings.Contains(cleanState, dao.JobStatePending) {
		if v.mainStatusBar != nil {
			v.mainStatusBar.Warning(fmt.Sprintf("Job %s is not in a releasable state (current: %s)", jobID, cleanState))
		}
		return
	}

	// Get job details before release
	jobBefore, _ := v.client.Jobs().Get(jobID)
	if jobBefore != nil {
		debug.Logger.Printf("Pre-release job %s - State: %s", jobID, jobBefore.State)
	}

	// Perform release
	debug.Logger.Printf("Calling Release API for job %s", jobID)
	err := v.client.Jobs().Release(jobID)
	if err != nil {
		debug.Logger.Printf("Release API failed for job %s: %v", jobID, err)
		if v.mainStatusBar != nil {
			v.mainStatusBar.Error(fmt.Sprintf("Failed to release job %s: %v", jobID, err))
		}
		return
	}

	debug.Logger.Printf("Release API returned success for job %s", jobID)

	// Verify release worked by checking job state
	time.Sleep(500 * time.Millisecond)
	jobAfter, _ := v.client.Jobs().Get(jobID)
	if jobAfter != nil {
		debug.Logger.Printf("Post-release job %s - State: %s", jobID, jobAfter.State)
	}

	if v.mainStatusBar != nil {
		v.mainStatusBar.Success(fmt.Sprintf("Job %s released", jobID))
	}

	// Refresh the view
	time.Sleep(500 * time.Millisecond)
	_ = v.Refresh()
}

// showJobDetails shows detailed information for the selected job
func (v *JobsView) showJobDetails() {
	data := v.table.GetSelectedData()
	if len(data) == 0 {
		return
	}

	jobID := data[0]

	// Fetch full job details
	job, err := v.client.Jobs().Get(jobID)
	if err != nil {
		// Note: Status bar update removed since individual view status bars are no longer used
		return
	}

	// Create details view
	details := v.formatJobDetails(job)

	textView := tview.NewTextView().
		SetDynamicColors(true).
		SetText(details).
		SetScrollable(true)

	modal := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(textView, 0, 1, true).
		AddItem(tview.NewTextView().SetText("Press ESC to close"), 1, 0, false)

	modal.SetBorder(true).
		SetTitle(fmt.Sprintf(" Job %s Details ", jobID)).
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
				v.pages.RemovePage("job-details")
			}
			return nil
		}
		return event
	})

	if v.pages != nil {
		v.pages.AddPage("job-details", centeredModal, true, true)
	}
}

// formatJobDetails formats job details for display
func (v *JobsView) formatJobDetails(job *dao.Job) string {
	var details strings.Builder

	details.WriteString(fmt.Sprintf("[yellow]Job ID:[white] %s\n", job.ID))
	details.WriteString(fmt.Sprintf("[yellow]Name:[white] %s\n", job.Name))
	details.WriteString(fmt.Sprintf("[yellow]User:[white] %s\n", job.User))
	details.WriteString(fmt.Sprintf("[yellow]Account:[white] %s\n", job.Account))

	stateColor := dao.GetJobStateColor(job.State)
	details.WriteString(fmt.Sprintf("[yellow]State:[white] [%s]%s[white]\n", stateColor, job.State))

	details.WriteString(fmt.Sprintf("[yellow]Partition:[white] %s\n", job.Partition))
	details.WriteString(fmt.Sprintf("[yellow]QOS:[white] %s\n", job.QOS))
	details.WriteString(fmt.Sprintf("[yellow]Priority:[white] %.0f\n", job.Priority))
	details.WriteString(fmt.Sprintf("[yellow]Node Count:[white] %d\n", job.NodeCount))

	if job.NodeList != "" {
		details.WriteString(fmt.Sprintf("[yellow]Node List:[white] %s\n", job.NodeList))
	}

	details.WriteString(fmt.Sprintf("[yellow]Time Limit:[white] %s\n", job.TimeLimit))
	details.WriteString(fmt.Sprintf("[yellow]Time Used:[white] %s\n", job.TimeUsed))

	details.WriteString(fmt.Sprintf("[yellow]Submit Time:[white] %s\n", job.SubmitTime.Format("2006-01-02 15:04:05")))

	if job.StartTime != nil {
		details.WriteString(fmt.Sprintf("[yellow]Start Time:[white] %s\n", job.StartTime.Format("2006-01-02 15:04:05")))
	}

	if job.EndTime != nil {
		details.WriteString(fmt.Sprintf("[yellow]End Time:[white] %s\n", job.EndTime.Format("2006-01-02 15:04:05")))
	}

	details.WriteString(fmt.Sprintf("[yellow]Working Dir:[white] %s\n", job.WorkingDir))

	// Note: SLURM API doesn't return the actual command/script, only job metadata
	// For job details, see StdOut file path below
	if job.Command != "" {
		details.WriteString(fmt.Sprintf("[yellow]Command:[white] %s\n", job.Command))
	}

	if job.StdOut != "" {
		details.WriteString(fmt.Sprintf("[yellow]StdOut:[white] %s\n", job.StdOut))
	}

	if job.StdErr != "" {
		details.WriteString(fmt.Sprintf("[yellow]StdErr:[white] %s\n", job.StdErr))
	}

	if job.ExitCode != nil {
		details.WriteString(fmt.Sprintf("[yellow]Exit Code:[white] %d\n", *job.ExitCode))
	}

	return details.String()
}

// showJobOutput shows job output logs using the new output viewer
func (v *JobsView) showJobOutput() {
	data := v.table.GetSelectedData()
	if len(data) == 0 {
		if v.mainStatusBar != nil {
			v.mainStatusBar.Warning("No job selected")
		}
		return
	}

	jobID := data[0]
	jobName := data[1]

	// Use the new job output viewer
	if v.jobOutputView != nil {
		v.jobOutputView.ShowJobOutput(jobID, jobName, "stdout")
	}
}

/*
TODO(lint): Review unused code - func (*JobsView).performJobSubmission is unused

performJobSubmission performs the actual job submission
func (v *JobsView) performJobSubmission(jobSub *dao.JobSubmission) {
	// Note: Status bar update removed since individual view status bars are no longer used

	_, err := v.client.Jobs().Submit(jobSub) // jobID no longer used for status updates
	if err != nil {
		// Note: Status bar update removed since individual view status bars are no longer used
		return
	}

	// Note: Status bar update removed since individual view status bars are no longer used

	// Refresh the view to show the new job
	time.Sleep(500 * time.Millisecond)
	_ = v.Refresh()
}
*/

// requeueSelectedJob requeues the selected job
func (v *JobsView) requeueSelectedJob() {
	data := v.table.GetSelectedData()
	if len(data) == 0 {
		return
	}

	jobID := data[0]
	jobName := data[1]
	state := data[4]

	// Check if job can be requeued (usually completed or failed jobs)
	if !strings.Contains(state, dao.JobStateCompleted) && !strings.Contains(state, dao.JobStateFailed) && !strings.Contains(state, dao.JobStateCancelled) {
		if v.mainStatusBar != nil {
			v.mainStatusBar.Warning(fmt.Sprintf("Job %s is not in a requeueable state (current: %s)", jobID, state))
		}
		return
	}

	// Show confirmation dialog
	confirmText := fmt.Sprintf("Requeue job %s (%s)?\n\nThis will create a new job with the same parameters.", jobID, jobName)
	modal := tview.NewModal().
		SetText(confirmText).
		AddButtons([]string{"Requeue", "Cancel"}).
		SetDoneFunc(func(buttonIndex int, _ string) {
			if buttonIndex == 0 {
				v.performRequeueJob(jobID)
			}
			if v.pages != nil {
				v.pages.RemovePage("requeue-confirmation")
			}
		})

	modal.SetBorder(true).
		SetTitle(" Confirm Job Requeue ").
		SetTitleAlign(tview.AlignCenter)

	if v.pages != nil {
		v.pages.AddPage("requeue-confirmation", modal, true, true)
	}
}

// performRequeueJob performs job requeue operation
func (v *JobsView) performRequeueJob(jobID string) {
	if v.mainStatusBar != nil {
		v.mainStatusBar.Info(fmt.Sprintf("Requeuing job %s...", jobID))
	}

	newJob, err := v.client.Jobs().Requeue(jobID)
	if err != nil {
		if v.mainStatusBar != nil {
			v.mainStatusBar.Error(fmt.Sprintf("Failed to requeue job %s: %v", jobID, err))
		}
		return
	}

	if v.mainStatusBar != nil {
		v.mainStatusBar.Success(fmt.Sprintf("Job %s requeued as job %s", jobID, newJob.ID))
	}

	// Refresh the view to show the new job
	time.Sleep(500 * time.Millisecond)
	_ = v.Refresh()
}

// showJobSubmissionForm shows job submission form using the wizard
func (v *JobsView) showJobSubmissionForm() {
	wizard := NewJobSubmissionWizard(v.client, v.app)
	wizard.Show(v.pages, func(_ string) {
		// Success callback
		// Note: Status bar update removed since individual view status bars are no longer used
		go func() { _ = v.Refresh() }()
	}, func() {
		// Cancel callback
		// Note: Status bar update removed since individual view status bars are no longer used
	})
}

// showJobTemplateSelector shows the job template selector (alias for submission form)
func (v *JobsView) showJobTemplateSelector() {
	v.showJobSubmissionForm()
}

// showJobActions shows an action menu for the selected job
func (v *JobsView) showJobActions() {
	data := v.table.GetSelectedData()
	if len(data) == 0 {
		// Note: Status bar update removed since individual view status bars are no longer used
		return
	}

	jobID := data[0]
	jobName := data[1]
	state := data[4]

	actions, handlers := v.buildJobActions(state)

	// Create action menu
	list := tview.NewList()
	for i, action := range actions {
		// Capture the handler in closure
		handler := handlers[i]
		list.AddItem(action, "", 0, func() {
			if v.pages != nil {
				v.pages.RemovePage("job-actions")
			}
			handler()
		})
	}

	list.SetBorder(true).
		SetTitle(fmt.Sprintf(" Actions for Job %s (%s) ", jobID, jobName)).
		SetTitleAlign(tview.AlignCenter)

	// Create centered modal layout
	centeredModal := tview.NewFlex().
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(nil, 0, 1, false).
			AddItem(list, 0, 4, true).
			AddItem(nil, 0, 1, false), 0, 4, true).
		AddItem(nil, 0, 1, false)

	// Handle ESC key
	list.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEsc {
			if v.pages != nil {
				v.pages.RemovePage("job-actions")
			}
			return nil
		}
		return event
	})

	if v.pages != nil {
		v.pages.AddPage("job-actions", centeredModal, true, true)
	}
}

// buildJobActions builds the list of available actions based on job state
func (v *JobsView) buildJobActions(state string) ([]string, []func()) {
	var actions []string
	var handlers []func()

	// Always available actions
	actions = append(actions, "View Details")
	handlers = append(handlers, v.showJobDetails)

	actions = append(actions, "View Output")
	handlers = append(handlers, v.showJobOutput)

	actions = append(actions, "View Dependencies")
	handlers = append(handlers, v.showJobDependencies)

	// State-specific actions
	if strings.Contains(state, dao.JobStateRunning) || strings.Contains(state, dao.JobStatePending) {
		actions = append(actions, "Cancel Job")
		handlers = append(handlers, v.cancelSelectedJob)
	}

	if strings.Contains(state, dao.JobStatePending) {
		actions = append(actions, "Hold Job")
		handlers = append(handlers, v.holdSelectedJob)
	}

	if strings.Contains(state, dao.JobStateSuspended) {
		actions = append(actions, "Release Job")
		handlers = append(handlers, v.releaseSelectedJob)
	}

	if strings.Contains(state, dao.JobStateCompleted) || strings.Contains(state, dao.JobStateFailed) || strings.Contains(state, dao.JobStateCancelled) {
		actions = append(actions, "Requeue Job")
		handlers = append(handlers, v.requeueSelectedJob)
	}

	actions = append(actions, "Cancel")
	handlers = append(handlers, func() {
		if v.pages != nil {
			v.pages.RemovePage("job-actions")
		}
	})

	return actions, handlers
}

// toggleStateFilter toggles job state filter
func (v *JobsView) toggleStateFilter(state string) {
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

// promptUserFilter prompts for user filter
func (v *JobsView) promptUserFilter() {
	input := tview.NewInputField().
		SetLabel("Filter by user: ").
		SetFieldWidth(20).
		SetText(v.userFilter)

	input.SetDoneFunc(func(key tcell.Key) {
		if key == tcell.KeyEnter {
			v.userFilter = input.GetText()
			go func() { _ = v.Refresh() }()
		}
		v.app.SetRoot(v.container, true)
	})

	input.SetBorder(true).
		SetTitle(" User Filter ").
		SetTitleAlign(tview.AlignCenter)

	v.app.SetRoot(input, true)
}

// FormatDurationDetailed formats a duration to a readable string
func FormatDurationDetailed(d time.Duration) string {
	days := int(d.Hours() / 24)
	hours := int(d.Hours()) % 24
	minutes := int(d.Minutes()) % 60
	seconds := int(d.Seconds()) % 60

	if days > 0 {
		return fmt.Sprintf("%d-%02d:%02d:%02d", days, hours, minutes, seconds)
	}
	return fmt.Sprintf("%02d:%02d:%02d", hours, minutes, seconds)
}

// toggleAutoRefresh toggles automatic refresh mode
func (v *JobsView) toggleAutoRefresh() {
	debug.Logger.Printf("toggleAutoRefresh() called, current state: %v", v.autoRefresh)
	v.autoRefresh = !v.autoRefresh

	if v.autoRefresh {
		if v.mainStatusBar != nil {
			v.mainStatusBar.Success("Auto-refresh enabled")
		}
		v.scheduleRefresh()
	} else {
		if v.mainStatusBar != nil {
			v.mainStatusBar.Info("Auto-refresh disabled")
		}
		if v.refreshTimer != nil {
			v.refreshTimer.Stop()
		}
	}
}

// showBatchOperations shows batch operations menu
func (v *JobsView) showBatchOperations() {
	debug.Logger.Printf("showBatchOperations() called, batchOpsView: %v", v.batchOpsView != nil)
	// Get currently selected jobs or allow manual selection
	var selectedJobs []string
	var selectedJobsData []map[string]interface{}

	// Check if any jobs are already selected
	if len(v.selectedJobs) > 0 {
		v.mu.RLock()
		for _, job := range v.jobs {
			if v.selectedJobs[job.ID] {
				selectedJobs = append(selectedJobs, job.ID)
				jobData := map[string]interface{}{
					"name":  job.Name,
					"state": job.State,
					"user":  job.User,
				}
				selectedJobsData = append(selectedJobsData, jobData)
			}
		}
		v.mu.RUnlock()
	} else {
		// If no jobs selected, use currently highlighted job
		data := v.table.GetSelectedData()
		if len(data) > 0 {
			selectedJobs = append(selectedJobs, data[0])
			jobData := map[string]interface{}{
				"name":  data[1],
				"state": data[4],
				"user":  data[2],
			}
			selectedJobsData = append(selectedJobsData, jobData)
		}
	}

	// Initialize batch operations view if not already done
	if v.batchOpsView == nil && v.app != nil {
		debug.Logger.Printf("Initializing batchOpsView in showBatchOperations")
		v.batchOpsView = NewBatchOperationsView(v.client, v.app)
		if v.pages != nil {
			v.batchOpsView.SetPages(v.pages)
		}
	}

	// Use the new batch operations view
	if v.batchOpsView != nil && len(selectedJobs) > 0 {
		v.batchOpsView.ShowBatchOperations(selectedJobs, selectedJobsData, func() {
			// Refresh the jobs view after batch operations complete
			go func() { _ = v.Refresh() }()
		})
	} else {
		// Show job selection menu if no jobs selected
		v.showJobSelectionMenu()
	}
}

// showJobSelectionMenu shows a menu to select jobs for batch operations
func (v *JobsView) showJobSelectionMenu() {
	list := tview.NewList()

	list.AddItem("Select All Running Jobs", "Select all currently running jobs", 0, func() {
		v.selectJobsByState(dao.JobStateRunning)
		v.closeBatchMenu()
		v.showBatchOperations() // Show batch operations after selection
	})

	list.AddItem("Select All Pending Jobs", "Select all pending jobs", 0, func() {
		v.selectJobsByState(dao.JobStatePending)
		v.closeBatchMenu()
		v.showBatchOperations() // Show batch operations after selection
	})

	list.AddItem("Select Current Job", "Select the currently highlighted job", 0, func() {
		data := v.table.GetSelectedData()
		if len(data) > 0 {
			v.selectedJobs[data[0]] = true
		}
		v.closeBatchMenu()
		v.showBatchOperations() // Show batch operations after selection
	})

	list.AddItem("Cancel", "Close batch operations menu", 0, func() {
		v.closeBatchMenu()
	})

	list.SetBorder(true).
		SetTitle(fmt.Sprintf(" Batch Operations (%d selected) ", len(v.selectedJobs))).
		SetTitleAlign(tview.AlignCenter)

	// Create centered modal layout
	centeredModal := tview.NewFlex().
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(nil, 0, 1, false).
			AddItem(list, 0, 6, true).
			AddItem(nil, 0, 1, false), 0, 6, true).
		AddItem(nil, 0, 1, false)

	// Handle ESC key
	list.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEsc {
			v.closeBatchMenu()
			return nil
		}
		return event
	})

	if v.pages != nil {
		v.pages.AddPage("batch-operations", centeredModal, true, true)
	}
}

// closeBatchMenu closes the batch operations menu
func (v *JobsView) closeBatchMenu() {
	if v.pages != nil {
		v.pages.RemovePage("batch-operations")
	}
}

// selectJobsByState selects all jobs in a given state
func (v *JobsView) selectJobsByState(state string) {
	v.mu.RLock()
	count := 0
	for _, job := range v.jobs {
		if job.State == state {
			v.selectedJobs[job.ID] = true
			count++
		}
	}
	v.mu.RUnlock()

	// Note: Status bar update removed since individual view status bars are no longer used
}

/*
TODO(lint): Review unused code - func (*JobsView).clearJobSelection is unused

clearJobSelection clears all selected jobs
func (v *JobsView) clearJobSelection() {
	v.selectedJobs = make(map[string]bool)
	// Note: Status bar update removed since individual view status bars are no longer used
}
*/

/*
TODO(lint): Review unused code - func (*JobsView).showSelectedJobs is unused

showSelectedJobs shows list of currently selected jobs
func (v *JobsView) showSelectedJobs() {
	if len(v.selectedJobs) == 0 {
		// Note: Status bar update removed since individual view status bars are no longer used
		return
	}

	list := tview.NewList()

	v.mu.RLock()
	for _, job := range v.jobs {
		if v.selectedJobs[job.ID] {
			jobInfo := fmt.Sprintf("%s - %s (%s)", job.ID, job.Name, job.State)
			list.AddItem(jobInfo, "", 0, nil)
		}
	}
	v.mu.RUnlock()

	list.AddItem("Close", "Close this view", 0, func() {
		if v.pages != nil {
			v.pages.RemovePage("selected-jobs")
		}
	})

	list.SetBorder(true).
		SetTitle(fmt.Sprintf(" Selected Jobs (%d) ", len(v.selectedJobs))).
		SetTitleAlign(tview.AlignCenter)

	// Create centered modal layout
	centeredModal := tview.NewFlex().
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(nil, 0, 1, false).
			AddItem(list, 0, 6, true).
			AddItem(nil, 0, 1, false), 0, 6, true).
		AddItem(nil, 0, 1, false)

	// Handle ESC key
	list.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEsc {
			if v.pages != nil {
				v.pages.RemovePage("selected-jobs")
			}
			return nil
		}
		return event
	})

	if v.pages != nil {
		v.pages.AddPage("selected-jobs", centeredModal, true, true)
	}
}
*/

/*
TODO(lint): Review unused code - func (*JobsView).batchCancelSelected is unused

batchCancelSelected cancels all selected jobs
func (v *JobsView) batchCancelSelected() {
	if len(v.selectedJobs) == 0 {
		// Note: Status bar update removed since individual view status bars are no longer used
		return
	}

	count := len(v.selectedJobs)
	confirmText := fmt.Sprintf("Cancel %d selected jobs?\n\nThis action cannot be undone.", count)

	modal := tview.NewModal().
		SetText(confirmText).
		AddButtons([]string{"Cancel Jobs", "Keep Running"}).
		SetDoneFunc(func(buttonIndex int, buttonLabel string) {
			if buttonIndex == 0 {
				go v.performBatchCancel()
			}
			if v.pages != nil {
				v.pages.RemovePage("batch-cancel-confirm")
			}
		})

	modal.SetBorder(true).
		SetTitle(" Confirm Batch Cancellation ").
		SetTitleAlign(tview.AlignCenter)

	if v.pages != nil {
		v.pages.AddPage("batch-cancel-confirm", modal, true, true)
	}
}
*/

/*
TODO(lint): Review unused code - func (*JobsView).performBatchCancel is unused

performBatchCancel performs batch job cancellation
func (v *JobsView) performBatchCancel() {
	success := 0
	failed := 0

	for jobID := range v.selectedJobs {
		err := v.client.Jobs().Cancel(jobID)
		if err != nil {
			failed++
		} else {
			success++
		}
	}

	// Note: Status bar update removed since individual view status bars are no longer used
	v.clearJobSelection()

	// Refresh the view
	time.Sleep(500 * time.Millisecond)
	_ = v.Refresh()
}
*/

/*
TODO(lint): Review unused code - func (*JobsView).batchHoldSelected is unused

batchHoldSelected holds all selected jobs
func (v *JobsView) batchHoldSelected() {
	if len(v.selectedJobs) == 0 {
		// Note: Status bar update removed since individual view status bars are no longer used
		return
	}

	go func() {
		success := 0
		failed := 0

		for jobID := range v.selectedJobs {
			err := v.client.Jobs().Hold(jobID)
			if err != nil {
				failed++
			} else {
				success++
			}
		}

		// Note: Status bar update removed since individual view status bars are no longer used
		v.clearJobSelection()

		// Refresh the view
		time.Sleep(500 * time.Millisecond)
		_ = v.Refresh()
	}()
}
*/

/*
TODO(lint): Review unused code - func (*JobsView).batchReleaseSelected is unused

batchReleaseSelected releases all selected jobs
func (v *JobsView) batchReleaseSelected() {
	if len(v.selectedJobs) == 0 {
		// Note: Status bar update removed since individual view status bars are no longer used
		return
	}

	go func() {
		success := 0
		failed := 0

		for jobID := range v.selectedJobs {
			err := v.client.Jobs().Release(jobID)
			if err != nil {
				failed++
			} else {
				success++
			}
		}

		// Note: Status bar update removed since individual view status bars are no longer used
		v.clearJobSelection()

		// Refresh the view
		time.Sleep(500 * time.Millisecond)
		_ = v.Refresh()
	}()
}
*/

// showAdvancedFilter shows the advanced filter bar
func (v *JobsView) showAdvancedFilter() {
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
func (v *JobsView) closeAdvancedFilter() {
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
func (v *JobsView) onAdvancedFilterChange(filter *filters.Filter) {
	v.advancedFilter = filter
	v.updateTable()

	// Note: Status bar updates removed since individual view status bars are no longer used
}

// applyAdvancedFilter applies the advanced filter to jobs
func (v *JobsView) applyAdvancedFilter(jobs []*dao.Job) []*dao.Job {
	if v.advancedFilter == nil || len(v.advancedFilter.Expressions) == 0 {
		return jobs
	}

	var filtered []*dao.Job
	for _, job := range jobs {
		// Convert job to map for filter evaluation
		jobData := v.jobToMap(job)
		if v.advancedFilter.Evaluate(jobData) {
			filtered = append(filtered, job)
		}
	}

	return filtered
}

// jobToMap converts a job to a map for filter evaluation
func (v *JobsView) jobToMap(job *dao.Job) map[string]interface{} {
	return map[string]interface{}{
		"ID":         job.ID,
		"Name":       job.Name,
		"User":       job.User,
		"Account":    job.Account,
		"State":      job.State,
		"Partition":  job.Partition,
		"NodeCount":  job.NodeCount,
		"NodeList":   job.NodeList,
		"TimeLimit":  job.TimeLimit,
		"TimeUsed":   job.TimeUsed,
		"Priority":   job.Priority,
		"QoS":        job.QOS,
		"SubmitTime": job.SubmitTime,
		"StartTime":  job.StartTime,
		"EndTime":    job.EndTime,
		"WorkingDir": job.WorkingDir,
		"Command":    job.Command,
	}
}

// showGlobalSearch shows the global search interface
func (v *JobsView) showGlobalSearch() {
	if v.globalSearch == nil || v.pages == nil {
		return
	}

	v.globalSearch.Show(v.pages, func(result SearchResult) {
		// Handle search result selection
		switch result.Type {
		case "job":
			// Focus on the selected job
			if job, ok := result.Data.(*dao.Job); ok {
				v.focusOnJob(job.ID)
			}
		default:
			// For other types, just close the search
			// Note: Status bar update removed since individual view status bars are no longer used
		}
	})
}

// focusOnJob focuses the table on a specific job
func (v *JobsView) focusOnJob(jobID string) {
	v.mu.RLock()
	defer v.mu.RUnlock()

	// Find the job in our job list
	for i, job := range v.jobs {
		if job.ID == jobID {
			// Select the row in the table
			v.table.Select(i, 0)
			// Note: Status bar update removed since individual view status bars are no longer used
			return
		}
	}

	// Note: Status bar update removed since individual view status bars are no longer used
}
