package views

import (
	"fmt"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/jontk/s9s/internal/dao"
	"github.com/rivo/tview"
)

// JobDependency represents a job dependency relationship
type JobDependency struct {
	JobID     string
	DependsOn []string
	Type      string // "afterok", "afternotok", "afterany", "after"
	Status    string // "waiting", "satisfied", "failed"
}

// showJobDependencies shows job dependency visualization
func (v *JobsView) showJobDependencies() {
	data := v.table.GetSelectedData()
	if len(data) == 0 {
		// Note: Status bar update removed since individual view status bars are no longer used
		return
	}

	jobID := data[0]
	jobName := data[1]

	// Fetch job details to check for dependencies
	job, err := v.client.Jobs().Get(jobID)
	if err != nil {
		// Note: Status bar update removed since individual view status bars are no longer used
		return
	}

	// Create dependency visualization
	content := v.buildDependencyTree(job)

	textView := tview.NewTextView().
		SetDynamicColors(true).
		SetText(content).
		SetScrollable(true)

	// Add controls
	controlsText := "Press [yellow]ESC[white] to close | [yellow]a[white] Add Dependency | [yellow]r[white] Remove Dependency"
	controls := tview.NewTextView().
		SetDynamicColors(true).
		SetText(controlsText).
		SetTextAlign(tview.AlignCenter)

	modal := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(textView, 0, 1, true).
		AddItem(controls, 1, 0, false)

	modal.SetBorder(true).
		SetTitle(fmt.Sprintf(" Job %s (%s) Dependencies ", jobID, jobName)).
		SetTitleAlign(tview.AlignCenter)

	// Create centered modal layout
	centeredModal := tview.NewFlex().
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(nil, 0, 1, false).
			AddItem(modal, 0, 8, true).
			AddItem(nil, 0, 1, false), 0, 8, true).
		AddItem(nil, 0, 1, false)

	// Handle key events
	textView.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEsc:
			if v.pages != nil {
				v.pages.RemovePage("job-dependencies")
			}
			return nil
		case tcell.KeyRune:
			switch event.Rune() {
			case 'a', 'A':
				v.showAddDependencyForm(jobID)
				return nil
			case 'r', 'R':
				v.showRemoveDependencyForm(jobID)
				return nil
			}
		}
		return event
	})

	if v.pages != nil {
		v.pages.AddPage("job-dependencies", centeredModal, true, true)
	}
}

// buildDependencyTree builds a visual representation of job dependencies
func (v *JobsView) buildDependencyTree(job *dao.Job) string {
	var content strings.Builder

	content.WriteString("[yellow]Job Dependency Analysis[white]\n\n")
	content.WriteString(fmt.Sprintf("[teal]Job:[white] %s (%s)\n", job.ID, job.Name))
	content.WriteString(fmt.Sprintf("[teal]State:[white] %s\n", job.State))
	content.WriteString(fmt.Sprintf("[teal]Submit Time:[white] %s\n\n", job.SubmitTime.Format("2006-01-02 15:04:05")))

	// Since we don't have actual dependency data in our current DAO,
	// we'll create a mock visualization for demonstration
	dependencies := v.getMockDependencies(job.ID)

	if len(dependencies) == 0 {
		content.WriteString("[yellow]Dependencies:[white]\n")
		content.WriteString("  No dependencies found\n\n")
	} else {
		content.WriteString("[yellow]Dependencies:[white]\n")
		for _, dep := range dependencies {
			statusColor := "white"
			switch dep.Status {
			case "satisfied":
				statusColor = "green"
			case "failed":
				statusColor = "red"
			case "waiting":
				statusColor = "yellow"
			}

			content.WriteString(fmt.Sprintf("  [%s]●[white] %s\n", statusColor, dep.Type))
			for _, depJobID := range dep.DependsOn {
				content.WriteString(fmt.Sprintf("    └── Job %s [%s](%s)[white]\n", depJobID, statusColor, dep.Status))
			}
		}
		content.WriteString("\n")
	}

	// Show reverse dependencies (jobs that depend on this job)
	reverseDeps := v.getMockReverseDependencies(job.ID)
	if len(reverseDeps) == 0 {
		content.WriteString("[yellow]Jobs Depending on This Job:[white]\n")
		content.WriteString("  No dependent jobs found\n\n")
	} else {
		content.WriteString("[yellow]Jobs Depending on This Job:[white]\n")
		for _, depJobID := range reverseDeps {
			content.WriteString(fmt.Sprintf("  ├── Job %s\n", depJobID))
		}
		content.WriteString("\n")
	}

	// Show dependency workflow visualization
	content.WriteString("[yellow]Workflow Visualization:[white]\n")
	content.WriteString(v.buildWorkflowDiagram(job.ID, dependencies, reverseDeps))

	// Add legend
	content.WriteString("\n[yellow]Legend:[white]\n")
	content.WriteString("  [green]●[white] Satisfied dependency\n")
	content.WriteString("  [yellow]●[white] Waiting dependency\n")
	content.WriteString("  [red]●[white] Failed dependency\n")
	content.WriteString("  [white]●[white] Unknown status\n")

	return content.String()
}

// getMockDependencies returns mock dependencies for demonstration
func (v *JobsView) getMockDependencies(jobID string) []JobDependency {
	// In a real implementation, this would query the SLURM database
	// For now, create some mock dependencies based on job ID patterns

	var deps []JobDependency

	// Create different dependency patterns based on job characteristics
	if strings.Contains(jobID, "200") { // Higher numbered jobs might depend on others
		deps = append(deps, JobDependency{
			JobID:     jobID,
			DependsOn: []string{fmt.Sprintf("%d", atoi(jobID)-1), fmt.Sprintf("%d", atoi(jobID)-2)},
			Type:      "afterok",
			Status:    "satisfied",
		})
	} else if strings.Contains(jobID, "150") {
		deps = append(deps, JobDependency{
			JobID:     jobID,
			DependsOn: []string{fmt.Sprintf("%d", atoi(jobID)-10)},
			Type:      "afterany",
			Status:    "waiting",
		})
	}

	return deps
}

// getMockReverseDependencies returns jobs that depend on the given job
func (v *JobsView) getMockReverseDependencies(jobID string) []string {
	var reverseDeps []string

	// Mock some reverse dependencies
	if strings.Contains(jobID, "100") {
		reverseDeps = append(reverseDeps, fmt.Sprintf("%d", atoi(jobID)+1))
		reverseDeps = append(reverseDeps, fmt.Sprintf("%d", atoi(jobID)+5))
	}

	return reverseDeps
}

// buildWorkflowDiagram creates a simple ASCII workflow diagram
func (v *JobsView) buildWorkflowDiagram(jobID string, deps []JobDependency, reverseDeps []string) string {
	var diagram strings.Builder

	// Build a simple workflow representation
	if len(deps) > 0 {
		diagram.WriteString("  Dependencies:\n")
		for _, dep := range deps {
			for i, depJobID := range dep.DependsOn {
				if i == 0 {
					diagram.WriteString(fmt.Sprintf("    %s", depJobID))
				} else {
					diagram.WriteString(fmt.Sprintf(" + %s", depJobID))
				}
			}
			diagram.WriteString(fmt.Sprintf(" --[%s]--> %s\n", dep.Type, jobID))
		}
	}

	if len(reverseDeps) > 0 {
		diagram.WriteString("  Dependents:\n")
		for _, depJobID := range reverseDeps {
			diagram.WriteString(fmt.Sprintf("    %s --[afterok]--> %s\n", jobID, depJobID))
		}
	}

	if len(deps) == 0 && len(reverseDeps) == 0 {
		diagram.WriteString(fmt.Sprintf("    [%s] (standalone job)\n", jobID))
	}

	return diagram.String()
}

// showAddDependencyForm shows form to add a new dependency
func (v *JobsView) showAddDependencyForm(jobID string) {
	depForm := tview.NewForm().
		AddInputField("Depends on Job ID", "", 20, nil, nil).
		AddDropDown("Dependency Type", []string{"afterok", "afternotok", "afterany", "after"}, 0, nil)

	depForm.AddButton("Add", func() {
		dependsOnJobID := depForm.GetFormItemByLabel("Depends on Job ID").(*tview.InputField).GetText()
		_, _ = depForm.GetFormItemByLabel("Dependency Type").(*tview.DropDown).GetCurrentOption() // depType no longer used

		if dependsOnJobID == "" {
			// Note: Status bar update removed since individual view status bars are no longer used
			return
		}

		// In a real implementation, this would update the job dependencies
		// Note: Status bar update removed since individual view status bars are no longer used

		if v.pages != nil {
			v.pages.RemovePage("add-dependency")
			v.pages.RemovePage("job-dependencies") // Close parent dialog too
		}
	}).
		AddButton("Cancel", func() {
			if v.pages != nil {
				v.pages.RemovePage("add-dependency")
			}
		})

	depForm.SetBorder(true).
		SetTitle(fmt.Sprintf(" Add Dependency to Job %s ", jobID)).
		SetTitleAlign(tview.AlignCenter)

	// Create centered modal layout
	centeredModal := tview.NewFlex().
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(nil, 0, 1, false).
			AddItem(depForm, 0, 4, true).
			AddItem(nil, 0, 1, false), 0, 4, true).
		AddItem(nil, 0, 1, false)

	if v.pages != nil {
		v.pages.AddPage("add-dependency", centeredModal, true, true)
	}
}

// showRemoveDependencyForm shows form to remove dependencies
func (v *JobsView) showRemoveDependencyForm(jobID string) {
	// Get current dependencies for the job
	dependencies := v.getMockDependencies(jobID)

	if len(dependencies) == 0 {
		// Note: Status bar update removed since individual view status bars are no longer used
		return
	}

	list := tview.NewList()

	for _, dep := range dependencies {
		for _, depJobID := range dep.DependsOn {
			depInfo := fmt.Sprintf("Job %s (%s)", depJobID, dep.Type)
			list.AddItem(depInfo, "", 0, func() {
				// In a real implementation, this would remove the dependency
				// Note: Status bar update removed since individual view status bars are no longer used

				if v.pages != nil {
					v.pages.RemovePage("remove-dependency")
					v.pages.RemovePage("job-dependencies") // Close parent dialog too
				}
			})
		}
	}

	list.AddItem("Cancel", "Close without removing", 0, func() {
		if v.pages != nil {
			v.pages.RemovePage("remove-dependency")
		}
	})

	list.SetBorder(true).
		SetTitle(fmt.Sprintf(" Remove Dependency from Job %s ", jobID)).
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
				v.pages.RemovePage("remove-dependency")
			}
			return nil
		}
		return event
	})

	if v.pages != nil {
		v.pages.AddPage("remove-dependency", centeredModal, true, true)
	}
}

// atoi converts string to int with fallback
func atoi(s string) int {
	var result int
	_, _ = fmt.Sscanf(s, "%d", &result)
	return result
}
