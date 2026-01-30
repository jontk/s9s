package views

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/jontk/s9s/internal/dao"
	"github.com/rivo/tview"
)

// JobSubmissionWizard provides an interactive interface for submitting jobs
type JobSubmissionWizard struct {
	client     dao.SlurmClient
	app        *tview.Application
	pages      *tview.Pages
	form       *tview.Form
	templates  map[string]*dao.JobTemplate
	onSubmit   func(jobID string)
	onCancel   func()
	workingDir string // Current working directory at application start
}

// NewJobSubmissionWizard creates a new job submission wizard
func NewJobSubmissionWizard(client dao.SlurmClient, app *tview.Application) *JobSubmissionWizard {
	// Get current working directory at application start
	workingDir, err := os.Getwd()
	if err != nil {
		workingDir = "" // Fallback to empty if error
	}

	return &JobSubmissionWizard{
		client:     client,
		app:        app,
		templates:  initializeTemplates(),
		workingDir: workingDir,
	}
}

// Show displays the job submission wizard
func (w *JobSubmissionWizard) Show(pages *tview.Pages, onSubmit func(jobID string), onCancel func()) {
	w.pages = pages
	w.onSubmit = onSubmit
	w.onCancel = onCancel

	// Show template selection first
	w.showTemplateSelection()
}

// showTemplateSelection shows the template selection screen
func (w *JobSubmissionWizard) showTemplateSelection() {
	list := tview.NewList()
	list.SetBorder(true).
		SetTitle(" Select Job Template ").
		SetTitleAlign(tview.AlignCenter)

	// Add custom job option
	list.AddItem("Custom Job", "Create a job from scratch", '0', func() {
		w.showJobForm(nil)
	})

	// Add templates
	templateNames := []string{
		"Basic Batch Job",
		"MPI Parallel Job",
		"GPU Job",
		"Array Job",
		"Interactive Job",
		"Long-Running Job",
		"High Memory Job",
		"Development/Debug Job",
	}

	for i, name := range templateNames {
		template := w.templates[name]
		if template != nil {
			list.AddItem(
				name,
				template.Description,
				rune('1'+i),
				func() {
					w.showJobForm(template)
				},
			)
		}
	}

	// Add cancel option
	list.AddItem("Cancel", "Return to jobs view", 'q', func() {
		w.pages.RemovePage("job-wizard-templates")
		if w.onCancel != nil {
			w.onCancel()
		}
	})

	// Handle ESC key
	list.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEsc {
			w.pages.RemovePage("job-wizard-templates")
			if w.onCancel != nil {
				w.onCancel()
			}
			return nil
		}
		return event
	})

	// Create centered layout
	centered := createCenteredModal(list, 50, 20)
	w.pages.AddPage("job-wizard-templates", centered, true, true)
}

// showJobForm shows the job submission form
func (w *JobSubmissionWizard) showJobForm(template *dao.JobTemplate) {
	form := tview.NewForm()
	w.form = form

	// Initialize job from template or defaults
	job := &dao.JobSubmission{
		TimeLimit:  "01:00:00",
		Memory:     "4G",
		CPUs:       1,
		Nodes:      1,
		WorkingDir: w.workingDir, // Default to current working directory
	}

	if template != nil {
		job = &template.JobSubmission
		// If template doesn't specify working directory, use current directory
		if job.WorkingDir == "" {
			job.WorkingDir = w.workingDir
		}
		form.SetTitle(fmt.Sprintf(" Submit Job - %s ", template.Name))
	} else {
		form.SetTitle(" Submit Job - Custom ")
	}

	// Add form fields
	w.addJobFormFields(form, job)

	// Add buttons
	w.addJobFormButtons(form, job)

	// Set styling and input handling
	form.SetBorder(true).SetTitleAlign(tview.AlignCenter)
	w.setupJobFormHandlers(form, job)

	centered := createCenteredModal(form, 80, 35)
	w.pages.AddPage("job-wizard-form", centered, true, true)
	w.pages.RemovePage("job-wizard-templates")
}

// addJobFormFields adds all form fields for job configuration
func (w *JobSubmissionWizard) addJobFormFields(form *tview.Form, job *dao.JobSubmission) {
	form.AddInputField("Job Name", job.Name, 50, nil, func(text string) {
		job.Name = text
	})

	form.AddTextArea("Script/Command", job.Script, 50, 5, 0, func(text string) {
		job.Script = text
	})

	// Add Partition dropdown
	partitions := w.getAvailablePartitions()
	if len(partitions) > 0 {
		form.AddDropDown("Partition", partitions, w.getPartitionIndex(partitions, job.Partition), func(option string, index int) {
			job.Partition = option
		})
	} else {
		// Fallback to input field if no partitions available
		form.AddInputField("Partition", job.Partition, 30, nil, func(text string) {
			job.Partition = text
		})
	}

	form.AddInputField("Time Limit (HH:MM:SS)", job.TimeLimit, 30, nil, func(text string) {
		job.TimeLimit = text
	})

	form.AddInputField("Nodes", fmt.Sprintf("%d", job.Nodes), 15, nil, func(text string) {
		if n, err := strconv.Atoi(text); err == nil && n > 0 {
			job.Nodes = n
		}
	})

	form.AddInputField("CPUs per Node", fmt.Sprintf("%d", job.CPUs), 15, nil, func(text string) {
		if n, err := strconv.Atoi(text); err == nil && n > 0 {
			job.CPUs = n
		}
	})

	form.AddInputField("Memory (e.g., 4G, 1024M)", job.Memory, 30, nil, func(text string) {
		job.Memory = text
	})

	w.addOptionalJobFields(form, job)
	w.addEmailNotificationFields(form, job)
}

// addOptionalJobFields adds optional fields for advanced job configuration
func (w *JobSubmissionWizard) addOptionalJobFields(form *tview.Form, job *dao.JobSubmission) {
	gpusStr := ""
	if job.GPUs > 0 {
		gpusStr = fmt.Sprintf("%d", job.GPUs)
	}
	form.AddInputField("GPUs (optional)", gpusStr, 15, nil, func(text string) {
		if text == "" {
			job.GPUs = 0
		} else if n, err := strconv.Atoi(text); err == nil && n >= 0 {
			job.GPUs = n
		}
	})

	form.AddInputField("QoS (optional)", job.QoS, 30, nil, func(text string) {
		job.QoS = text
	})

	// Add Account dropdown
	accounts := w.getAvailableAccounts()
	if len(accounts) > 0 {
		form.AddDropDown("Account (optional)", accounts, w.getAccountIndex(accounts, job.Account), func(option string, index int) {
			job.Account = option
		})
	} else {
		// Fallback to input field if no accounts available
		form.AddInputField("Account (optional)", job.Account, 30, nil, func(text string) {
			job.Account = text
		})
	}

	form.AddInputField("Working Directory", job.WorkingDir, 50, nil, func(text string) {
		job.WorkingDir = text
	})

	form.AddInputField("Output File", job.OutputFile, 50, nil, func(text string) {
		job.OutputFile = text
	})

	form.AddInputField("Error File", job.ErrorFile, 50, nil, func(text string) {
		job.ErrorFile = text
	})
}

// addEmailNotificationFields adds email notification configuration fields
func (w *JobSubmissionWizard) addEmailNotificationFields(form *tview.Form, job *dao.JobSubmission) {
	form.AddCheckbox("Email Notifications", job.EmailNotify, func(checked bool) {
		job.EmailNotify = checked
	})

	if job.EmailNotify {
		form.AddInputField("Email Address", job.Email, 40, nil, func(text string) {
			job.Email = text
		})
	}
}

// addJobFormButtons adds buttons for form submission
func (w *JobSubmissionWizard) addJobFormButtons(form *tview.Form, job *dao.JobSubmission) {
	form.AddButton("Submit", func() {
		if err := w.validateAndSubmitJob(job); err != nil {
			w.showError(err.Error())
		}
	})

	form.AddButton("Preview", func() {
		w.showJobPreview(job)
	})

	form.AddButton("Cancel", func() {
		w.pages.RemovePage("job-wizard-form")
		w.showTemplateSelection()
	})
}

// setupJobFormHandlers configures keyboard event handling for the job form
func (w *JobSubmissionWizard) setupJobFormHandlers(form *tview.Form, job *dao.JobSubmission) {
	form.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEsc:
			w.pages.RemovePage("job-wizard-form")
			w.showTemplateSelection()
			return nil
		case tcell.KeyCtrlS:
			if err := w.validateAndSubmitJob(job); err != nil {
				w.showError(err.Error())
			}
			return nil
		}
		// Let the form handle Tab and other navigation keys naturally
		return event
	})
}

// validateAndSubmitJob validates and submits the job
func (w *JobSubmissionWizard) validateAndSubmitJob(job *dao.JobSubmission) error {
	// Validate required fields
	if job.Name == "" {
		return fmt.Errorf("job name is required")
	}
	if job.Script == "" {
		return fmt.Errorf("script/command is required")
	}
	if job.Partition == "" {
		return fmt.Errorf("partition is required")
	}

	// Validate time format
	if !isValidTimeFormat(job.TimeLimit) {
		return fmt.Errorf("invalid time format (use HH:MM:SS or D-HH:MM:SS)")
	}

	// Validate memory format
	if !isValidMemoryFormat(job.Memory) {
		return fmt.Errorf("invalid memory format (use M or G suffix, e.g., 1024M or 4G)")
	}

	// Submit the job
	jobID, err := w.client.Jobs().Submit(job)
	if err != nil {
		return fmt.Errorf("failed to submit job: %w", err)
	}

	// Show success message
	w.showSuccess(fmt.Sprintf("Job successfully submitted!\n\nJob ID: %s\nJob Name: %s", jobID, job.Name))

	// Close wizard and callback
	w.pages.RemovePage("job-wizard-form")
	if w.onSubmit != nil {
		w.onSubmit(jobID)
	}

	return nil
}

// showJobPreview shows a preview of the job submission
func (w *JobSubmissionWizard) showJobPreview(job *dao.JobSubmission) {
	preview := generateJobScript(job)

	textView := tview.NewTextView().
		SetText(preview).
		SetDynamicColors(true).
		SetScrollable(true)

	modal := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(textView, 0, 1, true).
		AddItem(tview.NewTextView().SetText("Press ESC to close"), 1, 0, false)

	modal.SetBorder(true).
		SetTitle(" Job Script Preview ").
		SetTitleAlign(tview.AlignCenter)

	modal.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEsc {
			w.pages.RemovePage("job-preview")
			return nil
		}
		return event
	})

	centered := createCenteredModal(modal, 70, 25)
	w.pages.AddPage("job-preview", centered, true, true)
}

// showError shows an error message
func (w *JobSubmissionWizard) showError(message string) {
	modal := tview.NewModal().
		SetText(message).
		AddButtons([]string{"OK"}).
		SetDoneFunc(func(_ int, _ string) {
			w.pages.RemovePage("job-error")
		})

	modal.SetBackgroundColor(tcell.ColorDefault).
		SetTextColor(tcell.ColorRed)

	w.pages.AddPage("job-error", modal, true, true)
}

// showSuccess shows a success message
func (w *JobSubmissionWizard) showSuccess(message string) {
	modal := tview.NewModal().
		SetText(message).
		AddButtons([]string{"OK"}).
		SetDoneFunc(func(_ int, _ string) {
			w.pages.RemovePage("job-success")
		})

	modal.SetBackgroundColor(tcell.ColorDefault).
		SetTextColor(tcell.ColorGreen)

	w.pages.AddPage("job-success", modal, true, true)
}

// Helper functions

func initializeTemplates() map[string]*dao.JobTemplate {
	templates := make(map[string]*dao.JobTemplate)

	// Basic Batch Job
	templates["Basic Batch Job"] = &dao.JobTemplate{
		Name:        "Basic Batch Job",
		Description: "Simple batch job for serial computations",
		JobSubmission: dao.JobSubmission{
			Name:       "basic_job",
			Partition:  "normal",
			TimeLimit:  "01:00:00",
			Nodes:      1,
			CPUs:       1,
			Memory:     "4G",
			Script:     "#!/bin/bash\n#SBATCH --job-name=basic_job\n\n# Your commands here\necho \"Hello from SLURM job $SLURM_JOB_ID\"\n",
			OutputFile: "job_%j.out",
			ErrorFile:  "job_%j.err",
		},
	}

	// MPI Parallel Job
	templates["MPI Parallel Job"] = &dao.JobTemplate{
		Name:        "MPI Parallel Job",
		Description: "Parallel job using MPI across multiple nodes",
		JobSubmission: dao.JobSubmission{
			Name:       "mpi_job",
			Partition:  "parallel",
			TimeLimit:  "04:00:00",
			Nodes:      4,
			CPUs:       16,
			Memory:     "8G",
			Script:     "#!/bin/bash\n#SBATCH --job-name=mpi_job\n#SBATCH --ntasks-per-node=16\n\nmodule load mpi\nmpirun -np $SLURM_NTASKS ./my_mpi_program\n",
			OutputFile: "mpi_%j.out",
			ErrorFile:  "mpi_%j.err",
		},
	}

	// GPU Job
	templates["GPU Job"] = &dao.JobTemplate{
		Name:        "GPU Job",
		Description: "Job requiring GPU resources",
		JobSubmission: dao.JobSubmission{
			Name:       "gpu_job",
			Partition:  "gpu",
			TimeLimit:  "02:00:00",
			Nodes:      1,
			CPUs:       8,
			Memory:     "16G",
			GPUs:       1,
			Script:     "#!/bin/bash\n#SBATCH --job-name=gpu_job\n#SBATCH --gres=gpu:1\n\nmodule load cuda\n./my_gpu_program\n",
			OutputFile: "gpu_%j.out",
			ErrorFile:  "gpu_%j.err",
		},
	}

	// Array Job
	templates["Array Job"] = &dao.JobTemplate{
		Name:        "Array Job",
		Description: "Array job for processing multiple similar tasks",
		JobSubmission: dao.JobSubmission{
			Name:       "array_job",
			Partition:  "normal",
			TimeLimit:  "00:30:00",
			Nodes:      1,
			CPUs:       1,
			Memory:     "2G",
			Script:     "#!/bin/bash\n#SBATCH --job-name=array_job\n#SBATCH --array=1-100\n\n# Process file based on array task ID\n./process_file.sh input_${SLURM_ARRAY_TASK_ID}.dat\n",
			OutputFile: "array_%A_%a.out",
			ErrorFile:  "array_%A_%a.err",
		},
	}

	// Interactive Job
	templates["Interactive Job"] = &dao.JobTemplate{
		Name:        "Interactive Job",
		Description: "Interactive session for development and testing",
		JobSubmission: dao.JobSubmission{
			Name:      "interactive",
			Partition: "interactive",
			TimeLimit: "04:00:00",
			Nodes:     1,
			CPUs:      4,
			Memory:    "8G",
			Script:    "#!/bin/bash\n# Request interactive session with:\n# srun --pty bash\n",
		},
	}

	// Long-Running Job
	templates["Long-Running Job"] = &dao.JobTemplate{
		Name:        "Long-Running Job",
		Description: "Job for long-running computations (> 24 hours)",
		JobSubmission: dao.JobSubmission{
			Name:        "long_job",
			Partition:   "long",
			TimeLimit:   "7-00:00:00",
			Nodes:       1,
			CPUs:        8,
			Memory:      "32G",
			Script:      "#!/bin/bash\n#SBATCH --job-name=long_job\n\n# Enable checkpointing\n./my_long_computation --checkpoint-interval=3600\n",
			OutputFile:  "long_%j.out",
			ErrorFile:   "long_%j.err",
			EmailNotify: true,
		},
	}

	// High Memory Job
	templates["High Memory Job"] = &dao.JobTemplate{
		Name:        "High Memory Job",
		Description: "Job requiring large memory allocation",
		JobSubmission: dao.JobSubmission{
			Name:       "highmem_job",
			Partition:  "highmem",
			TimeLimit:  "12:00:00",
			Nodes:      1,
			CPUs:       32,
			Memory:     "256G",
			Script:     "#!/bin/bash\n#SBATCH --job-name=highmem_job\n\n# Large memory computation\n./memory_intensive_program\n",
			OutputFile: "highmem_%j.out",
			ErrorFile:  "highmem_%j.err",
		},
	}

	// Development/Debug Job
	templates["Development/Debug Job"] = &dao.JobTemplate{
		Name:        "Development/Debug Job",
		Description: "Short job for debugging and development",
		JobSubmission: dao.JobSubmission{
			Name:       "debug_job",
			Partition:  "debug",
			TimeLimit:  "00:15:00",
			Nodes:      1,
			CPUs:       2,
			Memory:     "4G",
			QoS:        "debug",
			Script:     "#!/bin/bash\n#SBATCH --job-name=debug_job\n\n# Debug commands\nset -x\necho \"Debug output\"\n./my_program --debug\n",
			OutputFile: "debug_%j.out",
			ErrorFile:  "debug_%j.err",
		},
	}

	return templates
}

func isValidTimeFormat(timeStr string) bool {
	// Validate time format: HH:MM:SS or D-HH:MM:SS
	// Simple validation - just check for colon presence and basic structure
	if !strings.Contains(timeStr, ":") {
		return false
	}

	// Check for D-HH:MM:SS format
	if strings.Contains(timeStr, "-") {
		parts := strings.Split(timeStr, "-")
		if len(parts) != 2 {
			return false
		}
	}

	return true
}

func isValidMemoryFormat(memStr string) bool {
	// Validate memory format: number followed by M or G
	if len(memStr) < 2 {
		return false
	}

	suffix := memStr[len(memStr)-1:]
	if suffix != "M" && suffix != "G" {
		return false
	}

	numberPart := memStr[:len(memStr)-1]
	_, err := strconv.Atoi(numberPart)
	return err == nil
}

func generateJobScript(job *dao.JobSubmission) string {
	var script strings.Builder

	script.WriteString("[yellow]#!/bin/bash[white]\n")
	script.WriteString(fmt.Sprintf("[green]#SBATCH --job-name=[white]%s\n", job.Name))
	script.WriteString(fmt.Sprintf("[green]#SBATCH --partition=[white]%s\n", job.Partition))
	script.WriteString(fmt.Sprintf("[green]#SBATCH --time=[white]%s\n", job.TimeLimit))
	script.WriteString(fmt.Sprintf("[green]#SBATCH --nodes=[white]%d\n", job.Nodes))
	script.WriteString(fmt.Sprintf("[green]#SBATCH --cpus-per-task=[white]%d\n", job.CPUs))
	script.WriteString(fmt.Sprintf("[green]#SBATCH --mem=[white]%s\n", job.Memory))

	if job.GPUs > 0 {
		script.WriteString(fmt.Sprintf("[green]#SBATCH --gres=[white]gpu:%d\n", job.GPUs))
	}

	if job.QoS != "" {
		script.WriteString(fmt.Sprintf("[green]#SBATCH --qos=[white]%s\n", job.QoS))
	}

	if job.Account != "" {
		script.WriteString(fmt.Sprintf("[green]#SBATCH --account=[white]%s\n", job.Account))
	}

	if job.WorkingDir != "" {
		script.WriteString(fmt.Sprintf("[green]#SBATCH --chdir=[white]%s\n", job.WorkingDir))
	}

	if job.OutputFile != "" {
		script.WriteString(fmt.Sprintf("[green]#SBATCH --output=[white]%s\n", job.OutputFile))
	}

	if job.ErrorFile != "" {
		script.WriteString(fmt.Sprintf("[green]#SBATCH --error=[white]%s\n", job.ErrorFile))
	}

	if job.EmailNotify && job.Email != "" {
		script.WriteString("[green]#SBATCH --mail-type=[white]ALL\n")
		script.WriteString(fmt.Sprintf("[green]#SBATCH --mail-user=[white]%s\n", job.Email))
	}

	script.WriteString("\n[cyan]# Job script[white]\n")
	script.WriteString(job.Script)

	return script.String()
}

func createCenteredModal(content tview.Primitive, width, height int) tview.Primitive {
	return tview.NewFlex().
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(nil, 0, 1, false).
			AddItem(content, height, 1, true).
			AddItem(nil, 0, 1, false), width, 1, true).
		AddItem(nil, 0, 1, false)
}

// getAvailablePartitions fetches the list of available partitions from the SLURM cluster
func (w *JobSubmissionWizard) getAvailablePartitions() []string {
	partitionList, _ := w.client.Partitions().List()
	if partitionList == nil || len(partitionList.Partitions) == 0 {
		return []string{}
	}

	var result []string
	for _, p := range partitionList.Partitions {
		if p != nil {
			result = append(result, p.Name)
		}
	}
	return result
}

// getPartitionIndex returns the index of the given partition in the list
func (w *JobSubmissionWizard) getPartitionIndex(partitions []string, partition string) int {
	if partition == "" {
		return 0
	}
	for i, p := range partitions {
		if p == partition {
			return i
		}
	}
	return 0
}

// getAvailableAccounts fetches the list of available accounts from the SLURM cluster
func (w *JobSubmissionWizard) getAvailableAccounts() []string {
	accountList, _ := w.client.Accounts().List()
	if accountList == nil || len(accountList.Accounts) == 0 {
		return []string{}
	}

	var result []string
	for _, a := range accountList.Accounts {
		if a != nil {
			result = append(result, a.Name)
		}
	}
	return result
}

// getAccountIndex returns the index of the given account in the list
func (w *JobSubmissionWizard) getAccountIndex(accounts []string, account string) int {
	if account == "" {
		return 0
	}
	for i, a := range accounts {
		if a == account {
			return i
		}
	}
	return 0
}
