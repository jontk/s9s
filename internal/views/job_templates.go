package views

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/jontk/s9s/internal/dao"
	"github.com/jontk/s9s/internal/fileperms"
	"github.com/rivo/tview"
)

// JobTemplate represents a saved job configuration
type JobTemplate struct {
	Name          string             `json:"name"`
	Description   string             `json:"description"`
	JobSubmission *dao.JobSubmission `json:"job_submission"`
}

// JobTemplateManager manages job templates
type JobTemplateManager struct {
	templatesDir string
	templates    []JobTemplate
}

// NewJobTemplateManager creates a new job template manager
func NewJobTemplateManager() *JobTemplateManager {
	homeDir, _ := os.UserHomeDir()
	templatesDir := filepath.Join(homeDir, ".s9s", "templates")

	// Create templates directory if it doesn't exist
	_ = os.MkdirAll(templatesDir, fileperms.ConfigDir)

	manager := &JobTemplateManager{
		templatesDir: templatesDir,
		templates:    []JobTemplate{},
	}

	// Load existing templates
	manager.loadTemplates()

	// Add default templates if none exist
	if len(manager.templates) == 0 {
		manager.createDefaultTemplates()
	}

	return manager
}

// loadTemplates loads templates from disk
func (m *JobTemplateManager) loadTemplates() {
	files, err := os.ReadDir(m.templatesDir)
	if err != nil {
		return
	}

	m.templates = []JobTemplate{}
	for _, file := range files {
		if strings.HasSuffix(file.Name(), ".json") {
			templatePath := filepath.Join(m.templatesDir, file.Name())
			data, err := os.ReadFile(templatePath)
			if err != nil {
				continue
			}

			var template JobTemplate
			if err := json.Unmarshal(data, &template); err != nil {
				continue
			}

			m.templates = append(m.templates, template)
		}
	}
}

// saveTemplate saves a template to disk
func (m *JobTemplateManager) saveTemplate(template JobTemplate) error {
	// Sanitize template name for filename
	filename := strings.ReplaceAll(template.Name, " ", "_")
	filename = strings.ReplaceAll(filename, "/", "_")
	filename = strings.ToLower(filename) + ".json"

	templatePath := filepath.Join(m.templatesDir, filename)

	data, err := json.MarshalIndent(template, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(templatePath, data, fileperms.ConfigFile)
}

/*
TODO(lint): Review unused code - func (*JobTemplateManager).deleteTemplate is unused

deleteTemplate deletes a template from disk
func (m *JobTemplateManager) deleteTemplate(name string) error {
	// Find and remove from memory
	for i, template := range m.templates {
		if template.Name == name {
			m.templates = append(m.templates[:i], m.templates[i+1:]...)
			break
		}
	}

	// Remove from disk
	filename := strings.ReplaceAll(name, " ", "_")
	filename = strings.ReplaceAll(filename, "/", "_")
	filename = strings.ToLower(filename) + ".json"
	templatePath := filepath.Join(m.templatesDir, filename)

	return os.Remove(templatePath)
}
*/

// getTemplates returns all available templates
func (m *JobTemplateManager) getTemplates() []JobTemplate {
	return m.templates
}

/*
TODO(lint): Review unused code - func (*JobTemplateManager).getTemplate is unused

getTemplate returns a specific template by name
func (m *JobTemplateManager) getTemplate(name string) (*JobTemplate, error) {
	for _, template := range m.templates {
		if template.Name == name {
			return &template, nil
		}
	}
	return nil, fmt.Errorf("template %s not found", name)
}
*/

// addTemplate adds a new template
func (m *JobTemplateManager) addTemplate(template JobTemplate) error {
	// Check if template already exists
	for i, existing := range m.templates {
		if existing.Name == template.Name {
			m.templates[i] = template // Update existing
			return m.saveTemplate(template)
		}
	}

	// Add new template
	m.templates = append(m.templates, template)
	return m.saveTemplate(template)
}

// createDefaultTemplates creates default job templates
func (m *JobTemplateManager) createDefaultTemplates() {
	defaultTemplates := []JobTemplate{
		{
			Name:        "Simple CPU Job",
			Description: "Basic CPU-only job template",
			JobSubmission: &dao.JobSubmission{
				Name:        "cpu_job",
				Command:     "echo 'Hello World'",
				Partition:   "compute",
				Nodes:       1,
				CPUsPerNode: 1,
				Memory:      "1G",
				TimeLimit:   "1:00:00",
				QoS:         "normal",
			},
		},
		{
			Name:        "GPU Job",
			Description: "GPU-accelerated job template",
			JobSubmission: &dao.JobSubmission{
				Name:        "gpu_job",
				Command:     "nvidia-smi",
				Partition:   "gpu",
				Nodes:       1,
				CPUsPerNode: 4,
				Memory:      "8G",
				TimeLimit:   "2:00:00",
				QoS:         "gpu-normal",
			},
		},
		{
			Name:        "Python Data Analysis",
			Description: "Python script for data analysis",
			JobSubmission: &dao.JobSubmission{
				Name:        "data_analysis",
				Command:     "python analysis.py",
				Partition:   "compute",
				Nodes:       1,
				CPUsPerNode: 8,
				Memory:      "16G",
				TimeLimit:   "4:00:00",
				QoS:         "normal",
			},
		},
		{
			Name:        "MPI Parallel Job",
			Description: "Multi-node MPI job template",
			JobSubmission: &dao.JobSubmission{
				Name:        "mpi_job",
				Command:     "mpirun -n 32 ./simulation",
				Partition:   "compute",
				Nodes:       4,
				CPUsPerNode: 8,
				Memory:      "32G",
				TimeLimit:   "8:00:00",
				QoS:         "normal",
			},
		},
		{
			Name:        "Long Running Job",
			Description: "Template for long-running computations",
			JobSubmission: &dao.JobSubmission{
				Name:        "long_job",
				Command:     "./long_simulation",
				Partition:   "compute",
				Nodes:       2,
				CPUsPerNode: 16,
				Memory:      "64G",
				TimeLimit:   "3-00:00:00",
				QoS:         "normal",
			},
		},
	}

	for _, template := range defaultTemplates {
		_ = m.addTemplate(template)
	}
}

// Removed showJobTemplateSelector - moved to jobs.go to use new wizard

// OLDShowJobTemplateSelector shows template selection dialog  - DEPRECATED
func OLDShowJobTemplateSelector(v *JobsView) {
	if v.templateManager == nil {
		v.templateManager = NewJobTemplateManager()
	}

	templates := v.templateManager.getTemplates()
	list := tview.NewList()

	// Add templates
	v.addTemplatesToList(list, templates)

	// Add custom job and conditional save option
	list.AddItem("Custom Job", "Create a new job from scratch", 0, func() {
		v.showJobSubmissionForm()
		v.removeTemplateSelectorPage()
	})

	data := v.table.GetSelectedData()
	if len(data) > 0 {
		list.AddItem("Save Current Job as Template", "Save selected job as a reusable template", 0, func() {
			v.saveJobAsTemplate(data[0])
			v.removeTemplateSelectorPage()
		})
	}

	list.AddItem("Cancel", "Close template selector", 0, func() {
		v.removeTemplateSelectorPage()
	})

	// Configure list styling and input handling
	list.SetBorder(true).SetTitle(" Job Templates ").SetTitleAlign(tview.AlignCenter)
	v.setupTemplateSelectorInput(list)

	// Show modal
	centeredModal := createCenteredModal(list, 0, 0)
	if v.pages != nil {
		v.pages.AddPage("template-selector", centeredModal, true, true)
	}
}

// addTemplatesToList adds template items to the list
func (v *JobsView) addTemplatesToList(list *tview.List, templates []JobTemplate) {
	for _, template := range templates {
		template := template // Capture for closure
		list.AddItem(template.Name, template.Description, 0, func() {
			v.loadJobTemplate(&template)
			v.removeTemplateSelectorPage()
		})
	}
}

// removeTemplateSelectorPage removes the template selector page
func (v *JobsView) removeTemplateSelectorPage() {
	if v.pages != nil {
		v.pages.RemovePage("template-selector")
	}
}

// setupTemplateSelectorInput configures keyboard event handling for template selector
func (v *JobsView) setupTemplateSelectorInput(list *tview.List) {
	list.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEsc {
			v.removeTemplateSelectorPage()
			return nil
		}
		return event
	})
}

// loadJobTemplate loads a template and shows the submission form
func (v *JobsView) loadJobTemplate(template *JobTemplate) {
	v.showJobSubmissionFormWithTemplate(template.JobSubmission)
}

// showJobSubmissionFormWithTemplate shows job submission form pre-filled with template data
func (v *JobsView) showJobSubmissionFormWithTemplate(template *dao.JobSubmission) {
	// Create form with template values
	form := tview.NewForm().
		AddInputField("Job Name", template.Name, 30, nil, nil).
		AddInputField("Command", template.Command, 50, nil, nil).
		AddInputField("Partition", template.Partition, 20, nil, nil).
		AddInputField("Nodes", fmt.Sprintf("%d", template.Nodes), 10, nil, nil).
		AddInputField("CPUs per Node", fmt.Sprintf("%d", template.CPUsPerNode), 10, nil, nil).
		AddInputField("Time Limit", template.TimeLimit, 15, nil, nil).
		AddInputField("Memory", template.Memory, 10, nil, nil).
		AddInputField("Account", template.Account, 20, nil, nil).
		AddInputField("QoS", template.QoS, 15, nil, nil).
		AddInputField("Working Directory", template.WorkingDir, 40, nil, nil)

	form.AddButton("Submit", func() {
		// v.submitJobFromForm - removed, use wizard instead(form)
	}).
		AddButton("Save as Template", func() {
			v.saveFormAsTemplate(form)
		}).
		AddButton("Cancel", func() {
			if v.pages != nil {
				v.pages.RemovePage("job-submission")
			}
		})

	form.SetBorder(true).
		SetTitle(" Submit Job from Template ").
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
			// v.submitJobFromForm - removed, use wizard instead(form)
			return nil
		case tcell.KeyEnter:
			// Check if we're on a button - if so, activate it
			formIndex, buttonIndex := form.GetFocusedItemIndex()
			switch {
			case buttonIndex >= 0:
				// We're on a button, let the form handle it
				return event
			case formIndex >= 10:
				// We're past the input fields (unlikely but safe)
				return event
			default:
				// We're on an input field, submit the form
				// v.submitJobFromForm - removed, use wizard instead(form)
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

// saveJobAsTemplate saves the selected job as a template
func (v *JobsView) saveJobAsTemplate(jobID string) {
	// Fetch job details
	job, err := v.client.Jobs().Get(jobID)
	if err != nil {
		// Note: Status bar update removed since individual view status bars are no longer used
		return
	}

	// Show template save form
	v.showSaveTemplateForm(job)
}

// saveFormAsTemplate saves the current form data as a template
func (v *JobsView) saveFormAsTemplate(form *tview.Form) {
	// Extract form values
	jobSub := v.extractJobSubmissionFromForm(form)

	// Show template save form
	v.showSaveTemplateFormFromSubmission(jobSub)
}

// extractJobSubmissionFromForm extracts JobSubmission from form
func (v *JobsView) extractJobSubmissionFromForm(form *tview.Form) *dao.JobSubmission {
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

	nodeCount := 1
	if nodes != "" {
		_, _ = fmt.Sscanf(nodes, "%d", &nodeCount)
	}

	cpusPerNodeCount := 1
	if cpusPerNode != "" {
		_, _ = fmt.Sscanf(cpusPerNode, "%d", &cpusPerNodeCount)
	}

	return &dao.JobSubmission{
		Name:        jobName,
		Command:     command,
		Partition:   partition,
		Account:     account,
		QoS:         qos,
		Nodes:       nodeCount,
		CPUsPerNode: cpusPerNodeCount,
		Memory:      memory,
		TimeLimit:   timeLimit,
		WorkingDir:  workingDir,
	}
}

// showSaveTemplateForm shows form to save job as template
func (v *JobsView) showSaveTemplateForm(job *dao.Job) {
	jobSub := &dao.JobSubmission{
		Name:       job.Name,
		Command:    job.Command,
		Partition:  job.Partition,
		Account:    job.Account,
		QoS:        job.QOS,
		Nodes:      job.NodeCount,
		TimeLimit:  job.TimeLimit,
		WorkingDir: job.WorkingDir,
	}

	v.showSaveTemplateFormFromSubmission(jobSub)
}

// showSaveTemplateFormFromSubmission shows template save form
func (v *JobsView) showSaveTemplateFormFromSubmission(jobSub *dao.JobSubmission) {
	if v.templateManager == nil {
		v.templateManager = NewJobTemplateManager()
	}

	form := tview.NewForm().
		AddInputField("Template Name", jobSub.Name+"_template", 30, nil, nil).
		AddInputField("Description", "Custom job template", 50, nil, nil)

	form.AddButton("Save", func() {
		templateName := form.GetFormItemByLabel("Template Name").(*tview.InputField).GetText()
		description := form.GetFormItemByLabel("Description").(*tview.InputField).GetText()

		if templateName == "" {
			// Note: Status bar update removed since individual view status bars are no longer used
			return
		}

		template := JobTemplate{
			Name:          templateName,
			Description:   description,
			JobSubmission: jobSub,
		}

		// Note: Status bar updates removed since individual view status bars are no longer used
		_ = v.templateManager.addTemplate(template)

		if v.pages != nil {
			v.pages.RemovePage("save-template")
		}
	}).
		AddButton("Cancel", func() {
			if v.pages != nil {
				v.pages.RemovePage("save-template")
			}
		})

	form.SetBorder(true).
		SetTitle(" Save Job Template ").
		SetTitleAlign(tview.AlignCenter)

	// Create centered modal layout
	centeredModal := tview.NewFlex().
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(nil, 0, 1, false).
			AddItem(form, 0, 4, true).
			AddItem(nil, 0, 1, false), 0, 4, true).
		AddItem(nil, 0, 1, false)

	if v.pages != nil {
		v.pages.AddPage("save-template", centeredModal, true, true)
	}
}
