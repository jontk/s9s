package views

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/jontk/s9s/internal/dao"
	"github.com/jontk/s9s/internal/debug"
	"github.com/jontk/s9s/internal/fileperms"
	"github.com/jontk/s9s/internal/ui/styles"
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

			normalizeJobSubmission(template.JobSubmission)
			m.templates = append(m.templates, template)
		}
	}
}

// normalizeJobSubmission bridges legacy field names to current ones.
// The old form flow used Command/CPUsPerNode; the wizard uses Script/CPUs.
func normalizeJobSubmission(js *dao.JobSubmission) {
	if js == nil {
		return
	}
	if js.Script == "" && js.Command != "" {
		js.Script = js.Command
	}
	if js.CPUs == 0 && js.CPUsPerNode > 0 {
		js.CPUs = js.CPUsPerNode
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

// GetTemplates returns all available templates
func (m *JobTemplateManager) GetTemplates() []JobTemplate {
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

// createDefaultTemplates creates default job templates from the single
// authoritative set defined by BuiltinTemplates().
func (m *JobTemplateManager) createDefaultTemplates() {
	for _, bt := range BuiltinTemplates() {
		js := bt.JobSubmission // value copy
		template := JobTemplate{
			Name:          bt.Name,
			Description:   bt.Description,
			JobSubmission: &js,
		}
		_ = m.addTemplate(template)
	}
}

// saveJobAsTemplate saves the selected job as a template
func (v *JobsView) saveJobAsTemplate(jobID string) {
	go func() {
		// Fetch job details off the UI thread
		job, err := v.client.Jobs().Get(jobID)
		if err != nil {
			debug.Logger.Printf("saveJobAsTemplate() - failed to get job %s: %v", jobID, err)
			return
		}

		if v.app != nil {
			v.app.QueueUpdateDraw(func() {
				// Show template save form
				v.showSaveTemplateForm(job)
			})
		}
	}()
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

	form := styles.StyleForm(tview.NewForm()).
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
