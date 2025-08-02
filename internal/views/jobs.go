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

// JobsView displays the jobs list
type JobsView struct {
	*BaseView
	client       dao.SlurmClient
	table        *components.Table
	jobs         []*dao.Job
	mu           sync.RWMutex
	refreshTimer *time.Timer
	refreshRate  time.Duration
	filter       string
	stateFilter  []string
	userFilter   string
	container    *tview.Flex
	filterInput  *tview.InputField
	statusBar    *tview.TextView
	app          *tview.Application
	pages        *tview.Pages
}

// SetPages sets the pages reference for modal handling
func (v *JobsView) SetPages(pages *tview.Pages) {
	v.pages = pages
}

// NewJobsView creates a new jobs view
func NewJobsView(client dao.SlurmClient) *JobsView {
	v := &JobsView{
		BaseView:    NewBaseView("jobs", "Jobs"),
		client:      client,
		refreshRate: 30 * time.Second,
		jobs:        []*dao.Job{},
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

	v.table = components.NewTableBuilder().
		WithColumns(columns...).
		WithSelectable(true).
		WithHeader(true).
		WithColors(tcell.ColorYellow, tcell.ColorTeal, tcell.ColorWhite).
		Build()

	// Set up callbacks
	v.table.SetOnSelect(v.onJobSelect)
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

// Init initializes the jobs view
func (v *JobsView) Init(ctx context.Context) error {
	v.BaseView.Init(ctx)
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

	// Fetch jobs from backend
	opts := &dao.ListJobsOptions{
		States: v.stateFilter,
		Limit:  1000, // TODO: Add pagination
	}

	if v.userFilter != "" {
		opts.Users = []string{v.userFilter}
	}

	jobList, err := v.client.Jobs().List(opts)
	if err != nil {
		v.SetLastError(err)
		v.updateStatusBar(fmt.Sprintf("[red]Error: %v[white]", err))
		return err
	}

	v.mu.Lock()
	v.jobs = jobList.Jobs
	v.mu.Unlock()

	// Update table
	v.updateTable()
	v.updateStatusBar("")

	// Schedule next refresh
	v.scheduleRefresh()

	return nil
}

// Stop stops the view
func (v *JobsView) Stop() error {
	if v.refreshTimer != nil {
		v.refreshTimer.Stop()
	}
	return nil
}

// Hints returns keyboard hints
func (v *JobsView) Hints() []string {
	return []string{
		"[yellow]Enter[white] Details",
		"[yellow]k[white] Kill",
		"[yellow]h[white] Hold",
		"[yellow]r[white] Release",
		"[yellow]l[white] Logs",
		"[yellow]s[white] SSH",
		"[yellow]/[white] Filter",
		"[yellow]1-9[white] Sort",
		"[yellow]R[white] Refresh",
	}
}

// OnKey handles keyboard events
func (v *JobsView) OnKey(event *tcell.EventKey) *tcell.EventKey {
	switch event.Key() {
	case tcell.KeyRune:
		switch event.Rune() {
		case 'k', 'K':
			v.killSelectedJob()
			return nil
		case 'h', 'H':
			v.holdSelectedJob()
			return nil
		case 'r':
			v.releaseSelectedJob()
			return nil
		case 'R':
			go v.Refresh()
			return nil
		case 'l', 'L':
			v.showJobLogs()
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
		case 'p', 'P':
			v.toggleStateFilter(dao.JobStatePending)
			return nil
		case 'u', 'U':
			v.promptUserFilter()
			return nil
		}
	case tcell.KeyEnter:
		v.showJobDetails()
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
func (v *JobsView) OnFocus() error {
	if v.app != nil {
		v.app.SetFocus(v.table.Table)
	}
	return nil
}

// OnLoseFocus handles loss of focus
func (v *JobsView) OnLoseFocus() error {
	return nil
}

// updateTable updates the table with current job data
func (v *JobsView) updateTable() {
	v.mu.RLock()
	defer v.mu.RUnlock()

	data := make([][]string, len(v.jobs))
	for i, job := range v.jobs {
		stateColor := dao.GetJobStateColor(job.State)
		coloredState := fmt.Sprintf("[%s]%s[white]", stateColor, job.State)

		timeUsed := job.TimeUsed
		if timeUsed == "" && job.StartTime != nil {
			timeUsed = formatDuration(time.Since(*job.StartTime))
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

// updateStatusBar updates the status bar
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

	if v.IsRefreshing() {
		status += " | [yellow]Refreshing...[white]"
	}

	v.statusBar.SetText(status)
}

// scheduleRefresh schedules the next refresh
func (v *JobsView) scheduleRefresh() {
	// Remove automatic refresh scheduling to prevent memory leak
	// Refresh will be handled by the main app refresh timer
}

// onJobSelect handles job selection
func (v *JobsView) onJobSelect(row, col int) {
	// Get selected job data
	data := v.table.GetSelectedData()
	if data == nil || len(data) == 0 {
		return
	}

	jobID := data[0]
	v.updateStatusBar(fmt.Sprintf("Selected job: %s", jobID))
}

// onSort handles column sorting
func (v *JobsView) onSort(col int, ascending bool) {
	// Sorting is handled by the table component
	v.updateStatusBar(fmt.Sprintf("Sorted by column %d", col+1))
}

// onFilterChange handles filter input changes
func (v *JobsView) onFilterChange(text string) {
	v.filter = text
	v.table.SetFilter(text)
	v.updateStatusBar("")
}

// onFilterDone handles filter input completion
func (v *JobsView) onFilterDone(key tcell.Key) {
	if v.app != nil {
		v.app.SetFocus(v.table.Table)
	}
}

// killSelectedJob kills the selected job
func (v *JobsView) killSelectedJob() {
	data := v.table.GetSelectedData()
	if data == nil || len(data) == 0 {
		return
	}

	jobID := data[0]
	state := data[4] // State column

	// Check if job can be killed
	if !strings.Contains(state, dao.JobStateRunning) && !strings.Contains(state, dao.JobStatePending) {
		v.updateStatusBar(fmt.Sprintf("[red]Job %s cannot be killed (state: %s)[white]", jobID, state))
		return
	}

	// Show confirmation dialog
	modal := tview.NewModal().
		SetText(fmt.Sprintf("Kill job %s?", jobID)).
		AddButtons([]string{"Yes", "No"}).
		SetDoneFunc(func(buttonIndex int, buttonLabel string) {
			if buttonIndex == 0 {
				go v.performKillJob(jobID)
			}
			v.app.SetRoot(v.container, true)
		})

	v.app.SetRoot(modal, true)
}

// performKillJob performs the job kill operation
func (v *JobsView) performKillJob(jobID string) {
	v.updateStatusBar(fmt.Sprintf("Killing job %s...", jobID))
	
	err := v.client.Jobs().Cancel(jobID)
	if err != nil {
		v.updateStatusBar(fmt.Sprintf("[red]Failed to kill job %s: %v[white]", jobID, err))
		return
	}

	v.updateStatusBar(fmt.Sprintf("[green]Job %s killed successfully[white]", jobID))
	
	// Refresh the view
	time.Sleep(500 * time.Millisecond)
	v.Refresh()
}

// holdSelectedJob holds the selected job
func (v *JobsView) holdSelectedJob() {
	data := v.table.GetSelectedData()
	if data == nil || len(data) == 0 {
		return
	}

	jobID := data[0]
	state := data[4] // State column

	// Check if job can be held
	if !strings.Contains(state, dao.JobStatePending) {
		v.updateStatusBar(fmt.Sprintf("[red]Job %s cannot be held (state: %s)[white]", jobID, state))
		return
	}

	go func() {
		v.updateStatusBar(fmt.Sprintf("Holding job %s...", jobID))
		
		err := v.client.Jobs().Hold(jobID)
		if err != nil {
			v.updateStatusBar(fmt.Sprintf("[red]Failed to hold job %s: %v[white]", jobID, err))
			return
		}

		v.updateStatusBar(fmt.Sprintf("[green]Job %s held successfully[white]", jobID))
		
		// Refresh the view
		time.Sleep(500 * time.Millisecond)
		v.Refresh()
	}()
}

// releaseSelectedJob releases the selected job
func (v *JobsView) releaseSelectedJob() {
	data := v.table.GetSelectedData()
	if data == nil || len(data) == 0 {
		return
	}

	jobID := data[0]
	state := data[4] // State column

	// Check if job can be released
	if !strings.Contains(state, dao.JobStateSuspended) {
		v.updateStatusBar(fmt.Sprintf("[red]Job %s is not held (state: %s)[white]", jobID, state))
		return
	}

	go func() {
		v.updateStatusBar(fmt.Sprintf("Releasing job %s...", jobID))
		
		err := v.client.Jobs().Release(jobID)
		if err != nil {
			v.updateStatusBar(fmt.Sprintf("[red]Failed to release job %s: %v[white]", jobID, err))
			return
		}

		v.updateStatusBar(fmt.Sprintf("[green]Job %s released successfully[white]", jobID))
		
		// Refresh the view
		time.Sleep(500 * time.Millisecond)
		v.Refresh()
	}()
}

// showJobDetails shows detailed information for the selected job
func (v *JobsView) showJobDetails() {
	data := v.table.GetSelectedData()
	if data == nil || len(data) == 0 {
		return
	}

	jobID := data[0]
	
	// Fetch full job details
	job, err := v.client.Jobs().Get(jobID)
	if err != nil {
		v.updateStatusBar(fmt.Sprintf("[red]Failed to get job details: %v[white]", err))
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
	details.WriteString(fmt.Sprintf("[yellow]Command:[white] %s\n", job.Command))
	
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

// showJobLogs shows job output logs
func (v *JobsView) showJobLogs() {
	data := v.table.GetSelectedData()
	if data == nil || len(data) == 0 {
		return
	}

	jobID := data[0]
	
	// TODO: Implement log streaming
	v.updateStatusBar(fmt.Sprintf("[yellow]Log streaming not yet implemented for job %s[white]", jobID))
}

// sshToNode opens SSH connection to job's node
func (v *JobsView) sshToNode() {
	data := v.table.GetSelectedData()
	if data == nil || len(data) == 0 {
		return
	}

	jobID := data[0]
	
	// TODO: Implement SSH functionality
	v.updateStatusBar(fmt.Sprintf("[yellow]SSH not yet implemented for job %s[white]", jobID))
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
	
	go v.Refresh()
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
			go v.Refresh()
		}
		v.app.SetRoot(v.container, true)
	})

	input.SetBorder(true).
		SetTitle(" User Filter ").
		SetTitleAlign(tview.AlignCenter)

	v.app.SetRoot(input, true)
}

// formatDuration formats a duration to a readable string
func formatDuration(d time.Duration) string {
	days := int(d.Hours() / 24)
	hours := int(d.Hours()) % 24
	minutes := int(d.Minutes()) % 60
	seconds := int(d.Seconds()) % 60

	if days > 0 {
		return fmt.Sprintf("%d-%02d:%02d:%02d", days, hours, minutes, seconds)
	}
	return fmt.Sprintf("%02d:%02d:%02d", hours, minutes, seconds)
}