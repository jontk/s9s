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
	client          dao.SlurmClient
	table           *components.Table
	jobs            []*dao.Job
	mu              sync.RWMutex
	refreshTimer    *time.Timer
	refreshRate     time.Duration
	filter          string
	stateFilter     []string
	userFilter      string
	container       *tview.Flex
	filterInput     *tview.InputField
	statusBar       *tview.TextView
	app             *tview.Application
	pages           *tview.Pages
	templateManager *JobTemplateManager
	autoRefresh     bool
	selectedJobs    map[string]bool
}

// SetPages sets the pages reference for modal handling
func (v *JobsView) SetPages(pages *tview.Pages) {
	v.pages = pages
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
		"[yellow]F1[white] Actions Menu",
		"[yellow]R[white] Refresh",
	}
}

// OnKey handles keyboard events
func (v *JobsView) OnKey(event *tcell.EventKey) *tcell.EventKey {
	// Check if a modal is open - if so, don't process view shortcuts
	if v.pages != nil && v.pages.GetPageCount() > 1 {
		return event // Let modal handle it
	}

	switch event.Key() {
	case tcell.KeyF2:
		v.showJobTemplateSelector()
		return nil
	case tcell.KeyRune:
		switch event.Rune() {
		case 'c', 'C':
			v.cancelSelectedJob()
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
		case 'o', 'O':
			v.showJobOutput()
			return nil
		case 's', 'S':
			v.showJobSubmissionForm()
			return nil
		case 'd', 'D':
			v.showJobDependencies()
			return nil
		case 'b', 'B':
			v.showBatchOperations()
			return nil
		case 'm', 'M':
			v.toggleAutoRefresh()
			return nil
		case 'q', 'Q':
			v.requeueSelectedJob()
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
	case tcell.KeyF1:
		v.showJobActions()
		return nil
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

// scheduleRefresh schedules the next refresh
func (v *JobsView) scheduleRefresh() {
	// Only schedule if auto-refresh is enabled
	if !v.autoRefresh {
		return
	}

	if v.refreshTimer != nil {
		v.refreshTimer.Stop()
	}

	v.refreshTimer = time.AfterFunc(v.refreshRate, func() {
		go v.Refresh()
	})
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

// cancelSelectedJob cancels the selected job
func (v *JobsView) cancelSelectedJob() {
	data := v.table.GetSelectedData()
	if data == nil || len(data) == 0 {
		return
	}

	jobID := data[0]
	jobName := data[1]
	state := data[4] // State column

	// Check if job can be cancelled
	if !strings.Contains(state, dao.JobStateRunning) && !strings.Contains(state, dao.JobStatePending) {
		v.updateStatusBar(fmt.Sprintf("[red]Job %s cannot be cancelled (state: %s)[white]", jobID, state))
		return
	}

	// Show confirmation dialog with more context
	confirmText := fmt.Sprintf("Cancel job %s (%s)?\n\nThis will terminate the job immediately.\nThis action cannot be undone.", jobID, jobName)
	modal := tview.NewModal().
		SetText(confirmText).
		AddButtons([]string{"Cancel Job", "Keep Running"}).
		SetDoneFunc(func(buttonIndex int, buttonLabel string) {
			if buttonIndex == 0 {
				go v.performCancelJob(jobID)
			}
			if v.pages != nil {
				v.pages.RemovePage("cancel-confirmation")
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
	v.updateStatusBar(fmt.Sprintf("Cancelling job %s...", jobID))

	err := v.client.Jobs().Cancel(jobID)
	if err != nil {
		v.updateStatusBar(fmt.Sprintf("[red]Failed to cancel job %s: %v[white]", jobID, err))
		return
	}

	v.updateStatusBar(fmt.Sprintf("[green]Job %s cancelled successfully[white]", jobID))

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

// showJobOutput shows job output logs
func (v *JobsView) showJobOutput() {
	data := v.table.GetSelectedData()
	if data == nil || len(data) == 0 {
		return
	}

	jobID := data[0]
	jobName := data[1]

	v.updateStatusBar(fmt.Sprintf("Fetching output for job %s...", jobID))

	go func() {
		output, err := v.client.Jobs().GetOutput(jobID)
		if err != nil {
			v.updateStatusBar(fmt.Sprintf("[red]Failed to get output for job %s: %v[white]", jobID, err))
			return
		}

		// Create output view
		textView := tview.NewTextView().
			SetDynamicColors(true).
			SetText(output).
			SetScrollable(true).
			SetWrap(true)

		// Add basic controls info
		textView.SetBorder(true).
			SetTitle(fmt.Sprintf(" Job %s (%s) Output ", jobID, jobName)).
			SetTitleAlign(tview.AlignCenter)

		// Create a flex layout with controls
		controlsText := "Press [yellow]ESC[white] to close | [yellow]↑↓[white] scroll | [yellow]Page Up/Down[white] page scroll"
		controls := tview.NewTextView().
			SetDynamicColors(true).
			SetText(controlsText).
			SetTextAlign(tview.AlignCenter)

		modal := tview.NewFlex().
			SetDirection(tview.FlexRow).
			AddItem(textView, 0, 1, true).
			AddItem(controls, 1, 0, false)

		// Create centered modal layout
		centeredModal := tview.NewFlex().
			AddItem(nil, 0, 1, false).
			AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
				AddItem(nil, 0, 1, false).
				AddItem(modal, 0, 8, true).
				AddItem(nil, 0, 1, false), 0, 8, true).
			AddItem(nil, 0, 1, false)

		// Handle ESC key
		textView.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
			if event.Key() == tcell.KeyEsc {
				if v.pages != nil {
					v.pages.RemovePage("job-output")
				}
				return nil
			}
			return event
		})

		if v.pages != nil {
			v.pages.AddPage("job-output", centeredModal, true, true)
		}

		v.updateStatusBar("")
	}()
}

// showJobSubmissionForm shows job submission form
func (v *JobsView) showJobSubmissionForm() {
	// Create form
	form := tview.NewForm().
		AddInputField("Job Name", "", 30, nil, nil).
		AddInputField("Command", "", 50, nil, nil).
		AddInputField("Partition", "compute", 20, nil, nil).
		AddInputField("Nodes", "1", 10, nil, nil).
		AddInputField("CPUs per Node", "1", 10, nil, nil).
		AddInputField("Time Limit", "1:00:00", 15, nil, nil).
		AddInputField("Memory", "1G", 10, nil, nil).
		AddInputField("Account", "", 20, nil, nil).
		AddInputField("QoS", "normal", 15, nil, nil).
		AddInputField("Working Directory", "", 40, nil, nil)

	form.AddButton("Submit", func() {
		v.submitJobFromForm(form)
	}).
	AddButton("Cancel", func() {
		if v.pages != nil {
			v.pages.RemovePage("job-submission")
		}
	})

	form.SetBorder(true).
		SetTitle(" Submit New Job ").
		SetTitleAlign(tview.AlignCenter)

	// Add help text
	helpText := "Navigation: [yellow]Tab/Shift+Tab[white] move between fields | [yellow]Enter[white] submit form | [yellow]Ctrl+S[white] submit | [yellow]ESC[white] cancel | Global shortcuts disabled"
	helpView := tview.NewTextView().
		SetDynamicColors(true).
		SetText(helpText).
		SetTextAlign(tview.AlignCenter)

	// Create form container with help
	formContainer := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(form, 0, 1, true).
		AddItem(helpView, 1, 0, false)

	// Create centered modal layout
	centeredModal := tview.NewFlex().
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(nil, 0, 1, false).
			AddItem(formContainer, 0, 6, true).
			AddItem(nil, 0, 1, false), 0, 6, true).
		AddItem(nil, 0, 1, false)

	// Handle keys for the form
	form.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEsc:
			if v.pages != nil {
				v.pages.RemovePage("job-submission")
			}
			return nil
		case tcell.KeyCtrlS:
			// Ctrl+S as alternative submit shortcut
			v.submitJobFromForm(form)
			return nil
		case tcell.KeyEnter:
			// Check if we're on a button - if so, activate it
			formIndex, buttonIndex := form.GetFocusedItemIndex()
			if buttonIndex >= 0 {
				// We're on a button, let the form handle it
				return event
			} else if formIndex >= 10 {
				// We're past the input fields (unlikely but safe)
				return event
			} else {
				// We're on an input field, submit the form
				v.submitJobFromForm(form)
				return nil
			}
		}
		// Let form handle all other keys (Tab, Shift+Tab, etc.)
		return event
	})

	if v.pages != nil {
		v.pages.AddPage("job-submission", centeredModal, true, true)
	}
}

// submitJobFromForm submits a job from the form data
func (v *JobsView) submitJobFromForm(form *tview.Form) {
	// Extract form values
	jobName := form.GetFormItemByLabel("Job Name").(*tview.InputField).GetText()
	command := form.GetFormItemByLabel("Command").(*tview.InputField).GetText()
	partition := form.GetFormItemByLabel("Partition").(*tview.InputField).GetText()
	nodes := form.GetFormItemByLabel("Nodes").(*tview.InputField).GetText()
	cpusPerNode := form.GetFormItemByLabel("CPUs per Node").(*tview.InputField).GetText()
	timeLimit := form.GetFormItemByLabel("Time Limit").(*tview.InputField).GetText()
	memory := form.GetFormItemByLabel("Memory").(*tview.InputField).GetText()
	account := form.GetFormItemByLabel("Account").(*tview.InputField).GetText()
	qos := form.GetFormItemByLabel("QoS").(*tview.InputField).GetText()
	workingDir := form.GetFormItemByLabel("Working Directory").(*tview.InputField).GetText()

	// Validate required fields
	if jobName == "" {
		v.updateStatusBar("[red]Job name is required[white]")
		return
	}
	if command == "" {
		v.updateStatusBar("[red]Command is required[white]")
		return
	}

	// Parse numeric fields
	nodeCount := 1
	if nodes != "" {
		if n, err := fmt.Sscanf(nodes, "%d", &nodeCount); err != nil || n != 1 {
			v.updateStatusBar("[red]Invalid node count[white]")
			return
		}
	}

	cpusPerNodeCount := 1
	if cpusPerNode != "" {
		if n, err := fmt.Sscanf(cpusPerNode, "%d", &cpusPerNodeCount); err != nil || n != 1 {
			v.updateStatusBar("[red]Invalid CPUs per node[white]")
			return
		}
	}

	// Create job submission
	jobSub := &dao.JobSubmission{
		Name:        jobName,
		Command:     command,
		Partition:   partition,
		Account:     account,
		QOS:         qos,
		Nodes:       nodeCount,
		CPUsPerNode: cpusPerNodeCount,
		Memory:      memory,
		TimeLimit:   timeLimit,
		WorkingDir:  workingDir,
	}

	// Close the form
	if v.pages != nil {
		v.pages.RemovePage("job-submission")
	}

	// Submit the job
	go v.performJobSubmission(jobSub)
}

// performJobSubmission performs the actual job submission
func (v *JobsView) performJobSubmission(jobSub *dao.JobSubmission) {
	v.updateStatusBar(fmt.Sprintf("Submitting job %s...", jobSub.Name))

	job, err := v.client.Jobs().Submit(jobSub)
	if err != nil {
		v.updateStatusBar(fmt.Sprintf("[red]Failed to submit job: %v[white]", err))
		return
	}

	v.updateStatusBar(fmt.Sprintf("[green]Job %s submitted successfully (ID: %s)[white]", job.Name, job.ID))

	// Refresh the view to show the new job
	time.Sleep(500 * time.Millisecond)
	v.Refresh()
}

// requeueSelectedJob requeues the selected job
func (v *JobsView) requeueSelectedJob() {
	data := v.table.GetSelectedData()
	if data == nil || len(data) == 0 {
		return
	}

	jobID := data[0]
	jobName := data[1]
	state := data[4]

	// Check if job can be requeued (usually completed or failed jobs)
	if !strings.Contains(state, dao.JobStateCompleted) && !strings.Contains(state, dao.JobStateFailed) && !strings.Contains(state, dao.JobStateCancelled) {
		v.updateStatusBar(fmt.Sprintf("[red]Job %s cannot be requeued (state: %s)[white]", jobID, state))
		return
	}

	// Show confirmation dialog
	confirmText := fmt.Sprintf("Requeue job %s (%s)?\n\nThis will create a new job with the same parameters.", jobID, jobName)
	modal := tview.NewModal().
		SetText(confirmText).
		AddButtons([]string{"Requeue", "Cancel"}).
		SetDoneFunc(func(buttonIndex int, buttonLabel string) {
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
	v.updateStatusBar(fmt.Sprintf("Requeuing job %s...", jobID))

	newJob, err := v.client.Jobs().Requeue(jobID)
	if err != nil {
		v.updateStatusBar(fmt.Sprintf("[red]Failed to requeue job %s: %v[white]", jobID, err))
		return
	}

	v.updateStatusBar(fmt.Sprintf("[green]Job %s requeued successfully (new ID: %s)[white]", jobID, newJob.ID))

	// Refresh the view to show the new job
	time.Sleep(500 * time.Millisecond)
	v.Refresh()
}

// showJobActions shows an action menu for the selected job
func (v *JobsView) showJobActions() {
	data := v.table.GetSelectedData()
	if data == nil || len(data) == 0 {
		v.updateStatusBar("[yellow]No job selected[white]")
		return
	}

	jobID := data[0]
	jobName := data[1]
	state := data[4]

	// Create action menu based on job state
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
	v.autoRefresh = !v.autoRefresh

	if v.autoRefresh {
		v.updateStatusBar("[green]Auto-refresh enabled[white]")
		v.scheduleRefresh()
	} else {
		v.updateStatusBar("[yellow]Auto-refresh disabled[white]")
		if v.refreshTimer != nil {
			v.refreshTimer.Stop()
		}
	}
}

// showBatchOperations shows batch operations menu
func (v *JobsView) showBatchOperations() {
	// Create batch operations menu
	list := tview.NewList()

	list.AddItem("Select All Running Jobs", "Select all currently running jobs", 0, func() {
		v.selectJobsByState(dao.JobStateRunning)
		v.closeBatchMenu()
	})

	list.AddItem("Select All Pending Jobs", "Select all pending jobs", 0, func() {
		v.selectJobsByState(dao.JobStatePending)
		v.closeBatchMenu()
	})

	list.AddItem("Cancel All Selected", "Cancel all selected jobs", 0, func() {
		v.batchCancelSelected()
		v.closeBatchMenu()
	})

	list.AddItem("Hold All Selected", "Hold all selected jobs", 0, func() {
		v.batchHoldSelected()
		v.closeBatchMenu()
	})

	list.AddItem("Release All Selected", "Release all selected jobs", 0, func() {
		v.batchReleaseSelected()
		v.closeBatchMenu()
	})

	list.AddItem("Clear Selection", "Clear all selected jobs", 0, func() {
		v.clearJobSelection()
		v.closeBatchMenu()
	})

	list.AddItem("Show Selected Jobs", "View currently selected jobs", 0, func() {
		v.showSelectedJobs()
		v.closeBatchMenu()
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

	v.updateStatusBar(fmt.Sprintf("[green]Selected %d jobs in state %s[white]", count, state))
}

// clearJobSelection clears all selected jobs
func (v *JobsView) clearJobSelection() {
	v.selectedJobs = make(map[string]bool)
	v.updateStatusBar("[yellow]Job selection cleared[white]")
}

// showSelectedJobs shows list of currently selected jobs
func (v *JobsView) showSelectedJobs() {
	if len(v.selectedJobs) == 0 {
		v.updateStatusBar("[yellow]No jobs selected[white]")
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

// batchCancelSelected cancels all selected jobs
func (v *JobsView) batchCancelSelected() {
	if len(v.selectedJobs) == 0 {
		v.updateStatusBar("[yellow]No jobs selected[white]")
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

// performBatchCancel performs batch job cancellation
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

	v.updateStatusBar(fmt.Sprintf("[green]Batch cancel completed: %d success, %d failed[white]", success, failed))
	v.clearJobSelection()

	// Refresh the view
	time.Sleep(500 * time.Millisecond)
	v.Refresh()
}

// batchHoldSelected holds all selected jobs
func (v *JobsView) batchHoldSelected() {
	if len(v.selectedJobs) == 0 {
		v.updateStatusBar("[yellow]No jobs selected[white]")
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

		v.updateStatusBar(fmt.Sprintf("[green]Batch hold completed: %d success, %d failed[white]", success, failed))
		v.clearJobSelection()

		// Refresh the view
		time.Sleep(500 * time.Millisecond)
		v.Refresh()
	}()
}

// batchReleaseSelected releases all selected jobs
func (v *JobsView) batchReleaseSelected() {
	if len(v.selectedJobs) == 0 {
		v.updateStatusBar("[yellow]No jobs selected[white]")
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

		v.updateStatusBar(fmt.Sprintf("[green]Batch release completed: %d success, %d failed[white]", success, failed))
		v.clearJobSelection()

		// Refresh the view
		time.Sleep(500 * time.Millisecond)
		v.Refresh()
	}()
}