package views

import (
	"encoding/base64"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/jontk/s9s/internal/config"
	"github.com/jontk/s9s/internal/dao"
	"github.com/jontk/s9s/internal/ui/styles"
	"github.com/rivo/tview"
)

// JobSubmissionWizard provides an interactive interface for submitting jobs
type JobSubmissionWizard struct {
	client           dao.SlurmClient
	app              *tview.Application
	pages            *tview.Pages
	form             *tview.Form
	templates        []*dao.JobTemplate
	onSubmit         func(jobID string)
	onCancel         func()
	workingDir       string // Current working directory at application start
	slurmUser        string // Resolved SLURM username for user lookups
	submissionConfig *config.JobSubmissionConfig
	selectedTemplate *dao.JobTemplate   // Track currently selected template for hidden fields
	currentJob       *dao.JobSubmission // Track current job for field visibility
}

// NewJobSubmissionWizard creates a new job submission wizard
func NewJobSubmissionWizard(client dao.SlurmClient, app *tview.Application, cfg *config.JobSubmissionConfig, slurmUser string) *JobSubmissionWizard {
	// Get current working directory at application start
	workingDir, err := os.Getwd()
	if err != nil {
		workingDir = "" // Fallback to empty if error
	}

	w := &JobSubmissionWizard{
		client:           client,
		app:              app,
		workingDir:       workingDir,
		slurmUser:        slurmUser,
		submissionConfig: cfg,
	}
	w.templates = w.mergeTemplates()
	return w
}

// Show displays the job submission wizard
func (w *JobSubmissionWizard) Show(pages *tview.Pages, onSubmit func(jobID string), onCancel func()) {
	w.pages = pages
	w.onSubmit = onSubmit
	w.onCancel = onCancel

	// Show template selection first
	w.showTemplateSelection()
}

// mergeTemplates performs 3-tier template merging with name-based override.
// Precedence (highest to lowest):
//  1. User-saved templates from ~/.s9s/templates/*.json
//  2. Config YAML templates
//  3. Built-in hardcoded templates
//
// Which tiers are active is controlled by the templateSources config option.
//
//nolint:cyclop // 3-tier template merge logic
func (w *JobSubmissionWizard) mergeTemplates() []*dao.JobTemplate {
	sources := config.ResolveTemplateSources(w.submissionConfig)
	seen := make(map[string]int) // name -> index in result
	var result []*dao.JobTemplate

	// 1. Built-in templates (lowest priority)
	if config.HasTemplateSource(sources, "builtin") {
		for _, t := range BuiltinTemplates() {
			seen[t.Name] = len(result)
			result = append(result, t)
		}
	}

	// 2. Config YAML templates
	if config.HasTemplateSource(sources, "config") && w.submissionConfig != nil {
		for _, ct := range w.submissionConfig.Templates {
			vals := config.JobSubmissionFromMap(ct.Defaults)
			js := ConfigValuesToJobSubmission(&vals)
			t := &dao.JobTemplate{
				Name:          ct.Name,
				Description:   ct.Description,
				JobSubmission: js,
			}
			if idx, ok := seen[ct.Name]; ok {
				result[idx] = t
			} else {
				seen[ct.Name] = len(result)
				result = append(result, t)
			}
		}
	}

	// 3. User-saved templates (highest priority)
	if config.HasTemplateSource(sources, "saved") {
		mgr := NewJobTemplateManager()
		for _, saved := range mgr.GetTemplates() {
			t := &dao.JobTemplate{
				Name:        saved.Name,
				Description: saved.Description,
			}
			if saved.JobSubmission != nil {
				t.JobSubmission = *saved.JobSubmission
			}
			if idx, ok := seen[saved.Name]; ok {
				result[idx] = t
			} else {
				seen[saved.Name] = len(result)
				result = append(result, t)
			}
		}
	}

	return result
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

	// Add merged templates
	for i, template := range w.templates {
		t := template // capture
		shortcut := rune(0)
		if i < 9 {
			shortcut = rune('1' + i)
		}
		list.AddItem(
			t.Name,
			t.Description,
			shortcut,
			func() {
				w.showJobForm(t)
			},
		)
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
//
//nolint:cyclop // multi-step form initialization
func (w *JobSubmissionWizard) showJobForm(template *dao.JobTemplate) {
	form := styles.StyleForm(tview.NewForm())
	w.form = form

	// 1. Start with hardcoded defaults
	job := &dao.JobSubmission{
		TimeLimit: "01:00:00",
		CPUs:      1,
		Nodes:     1,
	}

	// 2. Overlay formDefaults from config
	if w.submissionConfig != nil && w.submissionConfig.FormDefaults != nil {
		vals := config.JobSubmissionFromMap(w.submissionConfig.FormDefaults)
		cfgDefaults := ConfigValuesToJobSubmission(&vals)
		overlayJobDefaults(job, &cfgDefaults)
	}

	// 3. If template selected, overlay template defaults
	if template != nil {
		w.selectedTemplate = template
		overlayJobDefaults(job, &template.JobSubmission)
		form.SetTitle(fmt.Sprintf(" Submit Job - %s ", template.Name))
	} else {
		w.selectedTemplate = nil
		form.SetTitle(" Submit Job - Custom ")
	}

	// 4. Set defaults from current SLURM user (account, QoS) if not already set
	if job.Account == "" || job.QoS == "" {
		if user := w.getCurrentUser(); user != nil {
			if job.Account == "" {
				if user.DefaultAccount != "" {
					job.Account = user.DefaultAccount
				} else if user.Name != "" {
					// Fall back to username as account name
					job.Account = user.Name
				}
			}
			if job.QoS == "" && user.DefaultQoS != "" {
				job.QoS = user.DefaultQoS
			}
		}
	}

	// 5. Set workingDir from cwd if still empty
	if job.WorkingDir == "" {
		job.WorkingDir = w.workingDir
	}

	// Store current job so isFieldHidden can check for non-zero values
	w.currentJob = job

	// Add form fields
	w.addJobFormFields(form, job)

	// Add buttons
	w.addJobFormButtons(form, job)

	// Set styling and input handling
	form.SetBorder(true).SetTitleAlign(tview.AlignCenter)
	w.setupJobFormHandlers(form, job)

	centered := createCenteredModal(form, 100, 40)
	w.pages.AddPage("job-wizard-form", centered, true, true)
	w.pages.RemovePage("job-wizard-templates")
}

// ConfigValuesToJobSubmission converts config.JobSubmissionValues to dao.JobSubmission
func ConfigValuesToJobSubmission(v *config.JobSubmissionValues) dao.JobSubmission {
	return dao.JobSubmission{
		Name:                v.Name,
		Script:              v.Script,
		Partition:           v.Partition,
		Account:             v.Account,
		QoS:                 v.QoS,
		Nodes:               v.Nodes,
		CPUs:                v.CPUs,
		Memory:              v.Memory,
		GPUs:                v.GPUs,
		TimeLimit:           v.TimeLimit,
		WorkingDir:          v.WorkingDir,
		OutputFile:          v.OutputFile,
		ErrorFile:           v.ErrorFile,
		EmailNotify:         v.EmailNotify,
		Email:               v.Email,
		ArraySpec:           v.ArraySpec,
		Exclusive:           v.Exclusive,
		Requeue:             v.Requeue,
		Constraints:         v.Constraints,
		NTasks:              v.NTasks,
		NTasksPerNode:       v.NTasksPerNode,
		Gres:                v.Gres,
		Hold:                v.Hold,
		Reservation:         v.Reservation,
		Licenses:            v.Licenses,
		Wckey:               v.Wckey,
		ExcludeNodes:        v.ExcludeNodes,
		Priority:            v.Priority,
		Nice:                v.Nice,
		MemoryPerCPU:        v.MemoryPerCPU,
		BeginTime:           v.BeginTime,
		Comment:             v.Comment,
		Distribution:        v.Distribution,
		Prefer:              v.Prefer,
		RequiredNodes:       v.RequiredNodes,
		StandardInput:       v.StandardInput,
		Container:           v.Container,
		ThreadsPerCore:      v.ThreadsPerCore,
		TasksPerCore:        v.TasksPerCore,
		TasksPerSocket:      v.TasksPerSocket,
		SocketsPerNode:      v.SocketsPerNode,
		MaximumNodes:        v.MaximumNodes,
		MaximumCPUs:         v.MaximumCPUs,
		MinimumCPUsPerNode:  v.MinimumCPUsPerNode,
		TimeMinimum:         v.TimeMinimum,
		Contiguous:          v.Contiguous,
		Overcommit:          v.Overcommit,
		KillOnNodeFail:      v.KillOnNodeFail,
		WaitAllNodes:        v.WaitAllNodes,
		OpenMode:            v.OpenMode,
		TRESPerTask:         v.TRESPerTask,
		TRESPerSocket:       v.TRESPerSocket,
		Signal:              v.Signal,
		TmpDiskPerNode:      v.TmpDiskPerNode,
		Deadline:            v.Deadline,
		NTasksPerTRES:       v.NTasksPerTRES,
		CPUBinding:          v.CPUBinding,
		CPUFrequency:        v.CPUFrequency,
		Network:             v.Network,
		X11:                 v.X11,
		Immediate:           v.Immediate,
		BurstBuffer:         v.BurstBuffer,
		BatchFeatures:       v.BatchFeatures,
		TRESBind:            v.TRESBind,
		TRESFreq:            v.TRESFreq,
		CoreSpecification:   v.CoreSpecification,
		ThreadSpecification: v.ThreadSpecification,
		MemoryBinding:       v.MemoryBinding,
		MinimumCPUs:         v.MinimumCPUs,
		TRESPerJob:          v.TRESPerJob,
		CPUsPerTRES:         v.CPUsPerTRES,
		MemoryPerTRES:       v.MemoryPerTRES,
		Argv:                v.Argv,
		Flags:               v.Flags,
		ProfileTypes:        v.ProfileTypes,
		CPUBindingFlags:     v.CPUBindingFlags,
		MemoryBindingType:   v.MemoryBindingType,
		RequiredSwitches:    v.RequiredSwitches,
		WaitForSwitch:       v.WaitForSwitch,
		ClusterConstraint:   v.ClusterConstraint,
		Clusters:            v.Clusters,
		Dependencies:        v.Dependencies,
	}
}

// overlayJobDefaults overlays non-zero values from src onto dst
//
//nolint:cyclop // linear field-by-field overlay for 86 fields
func overlayJobDefaults(dst, src *dao.JobSubmission) {
	if src.Name != "" {
		dst.Name = src.Name
	}
	if src.Script != "" {
		dst.Script = src.Script
	}
	if src.Command != "" && dst.Script == "" {
		dst.Script = src.Command
	}
	if src.Partition != "" {
		dst.Partition = src.Partition
	}
	if src.Account != "" {
		dst.Account = src.Account
	}
	if src.QoS != "" {
		dst.QoS = src.QoS
	}
	if src.Nodes > 0 {
		dst.Nodes = src.Nodes
	}
	if src.CPUs > 0 {
		dst.CPUs = src.CPUs
	}
	if src.CPUsPerNode > 0 && dst.CPUs == 0 {
		dst.CPUs = src.CPUsPerNode
	}
	if src.Memory != "" {
		dst.Memory = src.Memory
	}
	if src.GPUs > 0 {
		dst.GPUs = src.GPUs
	}
	if src.TimeLimit != "" {
		dst.TimeLimit = src.TimeLimit
	}
	if src.WorkingDir != "" {
		dst.WorkingDir = src.WorkingDir
	}
	if src.OutputFile != "" {
		dst.OutputFile = src.OutputFile
	}
	if src.ErrorFile != "" {
		dst.ErrorFile = src.ErrorFile
	}
	if src.EmailNotify {
		dst.EmailNotify = src.EmailNotify
	}
	if src.Email != "" {
		dst.Email = src.Email
	}
	if src.ArraySpec != "" {
		dst.ArraySpec = src.ArraySpec
	}
	if src.Exclusive {
		dst.Exclusive = src.Exclusive
	}
	if src.Requeue {
		dst.Requeue = src.Requeue
	}
	if src.Constraints != "" {
		dst.Constraints = src.Constraints
	}
	if src.NTasks > 0 {
		dst.NTasks = src.NTasks
	}
	if src.NTasksPerNode > 0 {
		dst.NTasksPerNode = src.NTasksPerNode
	}
	if src.Gres != "" {
		dst.Gres = src.Gres
	}
	if src.Hold {
		dst.Hold = src.Hold
	}
	if src.Reservation != "" {
		dst.Reservation = src.Reservation
	}
	if src.Licenses != "" {
		dst.Licenses = src.Licenses
	}
	if src.Wckey != "" {
		dst.Wckey = src.Wckey
	}
	if src.ExcludeNodes != "" {
		dst.ExcludeNodes = src.ExcludeNodes
	}
	if src.Priority > 0 {
		dst.Priority = src.Priority
	}
	if src.Nice != 0 {
		dst.Nice = src.Nice
	}
	if src.MemoryPerCPU != "" {
		dst.MemoryPerCPU = src.MemoryPerCPU
	}
	if src.BeginTime != "" {
		dst.BeginTime = src.BeginTime
	}
	if src.Comment != "" {
		dst.Comment = src.Comment
	}
	if src.Distribution != "" {
		dst.Distribution = src.Distribution
	}
	if src.Prefer != "" {
		dst.Prefer = src.Prefer
	}
	if src.RequiredNodes != "" {
		dst.RequiredNodes = src.RequiredNodes
	}
	if src.StandardInput != "" {
		dst.StandardInput = src.StandardInput
	}
	if src.Container != "" {
		dst.Container = src.Container
	}
	if src.ThreadsPerCore > 0 {
		dst.ThreadsPerCore = src.ThreadsPerCore
	}
	if src.TasksPerCore > 0 {
		dst.TasksPerCore = src.TasksPerCore
	}
	if src.TasksPerSocket > 0 {
		dst.TasksPerSocket = src.TasksPerSocket
	}
	if src.SocketsPerNode > 0 {
		dst.SocketsPerNode = src.SocketsPerNode
	}
	if src.MaximumNodes > 0 {
		dst.MaximumNodes = src.MaximumNodes
	}
	if src.MaximumCPUs > 0 {
		dst.MaximumCPUs = src.MaximumCPUs
	}
	if src.MinimumCPUsPerNode > 0 {
		dst.MinimumCPUsPerNode = src.MinimumCPUsPerNode
	}
	if src.TimeMinimum != "" {
		dst.TimeMinimum = src.TimeMinimum
	}
	if src.Contiguous {
		dst.Contiguous = src.Contiguous
	}
	if src.Overcommit {
		dst.Overcommit = src.Overcommit
	}
	if src.KillOnNodeFail {
		dst.KillOnNodeFail = src.KillOnNodeFail
	}
	if src.WaitAllNodes {
		dst.WaitAllNodes = src.WaitAllNodes
	}
	if src.OpenMode != "" {
		dst.OpenMode = src.OpenMode
	}
	if src.TRESPerTask != "" {
		dst.TRESPerTask = src.TRESPerTask
	}
	if src.TRESPerSocket != "" {
		dst.TRESPerSocket = src.TRESPerSocket
	}
	if src.Signal != "" {
		dst.Signal = src.Signal
	}
	if src.TmpDiskPerNode > 0 {
		dst.TmpDiskPerNode = src.TmpDiskPerNode
	}
	if src.Deadline != "" {
		dst.Deadline = src.Deadline
	}
	if src.NTasksPerTRES > 0 {
		dst.NTasksPerTRES = src.NTasksPerTRES
	}
	if src.CPUBinding != "" {
		dst.CPUBinding = src.CPUBinding
	}
	if src.CPUFrequency != "" {
		dst.CPUFrequency = src.CPUFrequency
	}
	if src.Network != "" {
		dst.Network = src.Network
	}
	if src.X11 != "" {
		dst.X11 = src.X11
	}
	if src.Immediate {
		dst.Immediate = src.Immediate
	}
	if src.BurstBuffer != "" {
		dst.BurstBuffer = src.BurstBuffer
	}
	if src.BatchFeatures != "" {
		dst.BatchFeatures = src.BatchFeatures
	}
	if src.TRESBind != "" {
		dst.TRESBind = src.TRESBind
	}
	if src.TRESFreq != "" {
		dst.TRESFreq = src.TRESFreq
	}
	if src.CoreSpecification > 0 {
		dst.CoreSpecification = src.CoreSpecification
	}
	if src.ThreadSpecification > 0 {
		dst.ThreadSpecification = src.ThreadSpecification
	}
	if src.MemoryBinding != "" {
		dst.MemoryBinding = src.MemoryBinding
	}
	if src.MinimumCPUs > 0 {
		dst.MinimumCPUs = src.MinimumCPUs
	}
	if src.TRESPerJob != "" {
		dst.TRESPerJob = src.TRESPerJob
	}
	if src.CPUsPerTRES != "" {
		dst.CPUsPerTRES = src.CPUsPerTRES
	}
	if src.MemoryPerTRES != "" {
		dst.MemoryPerTRES = src.MemoryPerTRES
	}
	if src.Argv != "" {
		dst.Argv = src.Argv
	}
	if src.Flags != "" {
		dst.Flags = src.Flags
	}
	if src.ProfileTypes != "" {
		dst.ProfileTypes = src.ProfileTypes
	}
	if src.CPUBindingFlags != "" {
		dst.CPUBindingFlags = src.CPUBindingFlags
	}
	if src.MemoryBindingType != "" {
		dst.MemoryBindingType = src.MemoryBindingType
	}
	if src.RequiredSwitches > 0 {
		dst.RequiredSwitches = src.RequiredSwitches
	}
	if src.WaitForSwitch > 0 {
		dst.WaitForSwitch = src.WaitForSwitch
	}
	if src.ClusterConstraint != "" {
		dst.ClusterConstraint = src.ClusterConstraint
	}
	if src.Clusters != "" {
		dst.Clusters = src.Clusters
	}
	if len(src.Dependencies) > 0 {
		dst.Dependencies = src.Dependencies
	}
}

// defaultVisibleFields are always shown in the form.
// All other fields are hidden by default and only shown when they have
// a non-zero value (set by template or formDefaults).
var defaultVisibleFields = map[string]bool{
	"name":        true,
	"script":      true,
	"partition":   true,
	"timeLimit":   true,
	"nodes":       true,
	"cpus":        true,
	"memory":      true,
	"gpus":        true,
	"qos":         true,
	"account":     true,
	"workingDir":  true,
	"outputFile":  true,
	"errorFile":   true,
	"emailNotify": true,
	"email":       true,
}

// isFieldHidden checks whether a field should be hidden from the form.
// Default-visible fields are always shown unless explicitly in hiddenFields.
// Advanced fields are hidden by default unless they have a non-zero value
// (set by template or formDefaults) or are in a "show all" mode.
func (w *JobSubmissionWizard) isFieldHidden(fieldName string) bool {
	// Check explicit hiddenFields (global + per-template) — always wins
	if w.isExplicitlyHidden(fieldName) {
		return true
	}

	// Default-visible fields are always shown
	if defaultVisibleFields[fieldName] {
		return false
	}

	// Advanced fields: show only if the current job has a non-zero value
	// (meaning a template or formDefaults populated it)
	if w.currentJob != nil && fieldHasValue(w.currentJob, fieldName) {
		return false
	}

	// Hide advanced fields that have no value
	return true
}

// fieldHasValue returns true if the named field on the job has a non-zero value.
//
//nolint:cyclop // switch on all field names
func fieldHasValue(job *dao.JobSubmission, field string) bool {
	switch field {
	// string fields
	case "arraySpec":
		return job.ArraySpec != ""
	case "constraints":
		return job.Constraints != ""
	case "gres":
		return job.Gres != ""
	case "reservation":
		return job.Reservation != ""
	case "licenses":
		return job.Licenses != ""
	case "wckey":
		return job.Wckey != ""
	case "excludeNodes":
		return job.ExcludeNodes != ""
	case "memoryPerCPU":
		return job.MemoryPerCPU != ""
	case "beginTime":
		return job.BeginTime != ""
	case "comment":
		return job.Comment != ""
	case "distribution":
		return job.Distribution != ""
	case "prefer":
		return job.Prefer != ""
	case "requiredNodes":
		return job.RequiredNodes != ""
	case "standardInput":
		return job.StandardInput != ""
	case "container":
		return job.Container != ""
	case "timeMinimum":
		return job.TimeMinimum != ""
	case "openMode":
		return job.OpenMode != ""
	case "tresPerTask":
		return job.TRESPerTask != ""
	case "tresPerSocket":
		return job.TRESPerSocket != ""
	case "signal":
		return job.Signal != ""
	case "deadline":
		return job.Deadline != ""
	case "cpuBinding":
		return job.CPUBinding != ""
	case "cpuFrequency":
		return job.CPUFrequency != ""
	case "network":
		return job.Network != ""
	case "x11":
		return job.X11 != ""
	case "burstBuffer":
		return job.BurstBuffer != ""
	case "batchFeatures":
		return job.BatchFeatures != ""
	case "tresBind":
		return job.TRESBind != ""
	case "tresFreq":
		return job.TRESFreq != ""
	case "memoryBinding":
		return job.MemoryBinding != ""
	case "cpusPerTRES":
		return job.CPUsPerTRES != ""
	case "memoryPerTRES":
		return job.MemoryPerTRES != ""
	case "argv":
		return job.Argv != ""
	case "flags":
		return job.Flags != ""
	case "profile":
		return job.ProfileTypes != ""
	case "cpuBindingFlags":
		return job.CPUBindingFlags != ""
	case "memoryBindingType":
		return job.MemoryBindingType != ""
	case "clusterConstraint":
		return job.ClusterConstraint != ""
	case "clusters":
		return job.Clusters != ""
	case "tresPerJob":
		return job.TRESPerJob != ""
	// int fields
	case "ntasks":
		return job.NTasks > 0
	case "ntasksPerNode":
		return job.NTasksPerNode > 0
	case "priority":
		return job.Priority > 0
	case "nice":
		return job.Nice != 0
	case "threadsPerCore":
		return job.ThreadsPerCore > 0
	case "tasksPerCore":
		return job.TasksPerCore > 0
	case "tasksPerSocket":
		return job.TasksPerSocket > 0
	case "socketsPerNode":
		return job.SocketsPerNode > 0
	case "maximumNodes":
		return job.MaximumNodes > 0
	case "maximumCPUs":
		return job.MaximumCPUs > 0
	case "minimumCPUsPerNode":
		return job.MinimumCPUsPerNode > 0
	case "minimumCPUs":
		return job.MinimumCPUs > 0
	case "tmpDiskPerNode":
		return job.TmpDiskPerNode > 0
	case "ntasksPerTRES":
		return job.NTasksPerTRES > 0
	case "coreSpecification":
		return job.CoreSpecification > 0
	case "threadSpecification":
		return job.ThreadSpecification > 0
	case "requiredSwitches":
		return job.RequiredSwitches > 0
	case "waitForSwitch":
		return job.WaitForSwitch > 0
	// bool fields
	case "exclusive":
		return job.Exclusive
	case "requeue":
		return job.Requeue
	case "hold":
		return job.Hold
	case "contiguous":
		return job.Contiguous
	case "overcommit":
		return job.Overcommit
	case "killOnNodeFail":
		return job.KillOnNodeFail
	case "waitAllNodes":
		return job.WaitAllNodes
	case "immediate":
		return job.Immediate
	// slice fields
	case "dependencies":
		return len(job.Dependencies) > 0
	default:
		return false
	}
}

// isExplicitlyHidden checks global and per-template hiddenFields lists
func (w *JobSubmissionWizard) isExplicitlyHidden(fieldName string) bool {
	if w.submissionConfig == nil {
		return false
	}
	for _, f := range w.submissionConfig.HiddenFields {
		if f == fieldName {
			return true
		}
	}
	if w.selectedTemplate != nil {
		for _, ct := range w.submissionConfig.Templates {
			if ct.Name == w.selectedTemplate.Name {
				for _, f := range ct.HiddenFields {
					if f == fieldName {
						return true
					}
				}
				break
			}
		}
	}
	return false
}

// addJobFormFields adds all form fields for job configuration
//
//nolint:cyclop // form field builder for core fields
func (w *JobSubmissionWizard) addJobFormFields(form *tview.Form, job *dao.JobSubmission) {
	if !w.isFieldHidden("name") {
		form.AddInputField("Job Name", job.Name, 50, nil, func(text string) {
			job.Name = text
		})
	}

	if !w.isFieldHidden("script") {
		scriptArea := tview.NewTextArea().
			SetLabel("Script/Command").
			SetSize(5, 0).
			SetMaxLength(0)
		scriptArea.SetText(job.Script, false)
		scriptArea.SetChangedFunc(func() {
			job.Script = scriptArea.GetText()
		})
		form.AddFormItem(scriptArea)
	}

	if !w.isFieldHidden("partition") {
		// Partition dropdown with field option filtering
		partitions := w.getAvailablePartitions()
		partitions = w.filterFieldOptions(partitions, "partition")
		if len(partitions) > 0 {
			form.AddDropDown("Partition", partitions, w.getPartitionIndex(partitions, job.Partition), func(option string, index int) {
				job.Partition = option
			})
		} else {
			form.AddInputField("Partition", job.Partition, 30, nil, func(text string) {
				job.Partition = text
			})
		}
	}

	if !w.isFieldHidden("timeLimit") {
		form.AddInputField("Time Limit (HH:MM:SS)", job.TimeLimit, 30, nil, func(text string) {
			job.TimeLimit = text
		})
	}

	if !w.isFieldHidden("nodes") {
		form.AddInputField("Nodes", fmt.Sprintf("%d", job.Nodes), 15, nil, func(text string) {
			if n, err := strconv.Atoi(text); err == nil && n > 0 {
				job.Nodes = n
			}
		})
	}

	if !w.isFieldHidden("cpus") {
		form.AddInputField("CPUs per Node", fmt.Sprintf("%d", job.CPUs), 15, nil, func(text string) {
			if n, err := strconv.Atoi(text); err == nil && n > 0 {
				job.CPUs = n
			}
		})
	}

	if !w.isFieldHidden("memory") {
		form.AddInputField("Memory (e.g., 4G, 1024M)", job.Memory, 30, nil, func(text string) {
			job.Memory = text
		})
	}

	w.addOptionalJobFields(form, job)
	w.addEmailNotificationFields(form, job)
}

// addOptionalJobFields adds optional fields for advanced job configuration
//
//nolint:cyclop // form field builder for all optional fields
func (w *JobSubmissionWizard) addOptionalJobFields(form *tview.Form, job *dao.JobSubmission) {
	if !w.isFieldHidden("gpus") {
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
	}

	if !w.isFieldHidden("qos") {
		// Fetch QoS from cluster, then filter with config fieldOptions
		qosOptions := w.getAvailableQoS()
		qosOptions = w.filterFieldOptions(qosOptions, "qos")
		if len(qosOptions) == 0 {
			// Fall back to config-only fieldOptions if cluster returned nothing
			qosOptions = w.getConfigFieldOptions("qos")
		}
		if len(qosOptions) > 0 {
			// Prepend empty option for "not set"
			qosOptions = append([]string{""}, qosOptions...)
			form.AddDropDown("QoS", qosOptions, w.getStringIndex(qosOptions, job.QoS), func(option string, index int) {
				job.QoS = option
			})
		} else {
			form.AddInputField("QoS", job.QoS, 30, nil, func(text string) {
				job.QoS = text
			})
		}
	}

	if !w.isFieldHidden("account") {
		accounts := w.getAvailableAccounts()
		accounts = w.filterFieldOptions(accounts, "account")
		if len(accounts) > 0 {
			// Prepend empty option for "not set"
			accounts = append([]string{""}, accounts...)
			form.AddDropDown("Account", accounts, w.getAccountIndex(accounts, job.Account), func(option string, index int) {
				job.Account = option
			})
		} else {
			form.AddInputField("Account", job.Account, 30, nil, func(text string) {
				job.Account = text
			})
		}
	}

	if !w.isFieldHidden("workingDir") {
		form.AddInputField("Working Directory", job.WorkingDir, 50, nil, func(text string) {
			job.WorkingDir = text
		})
	}

	if !w.isFieldHidden("outputFile") {
		form.AddInputField("Output File", job.OutputFile, 50, nil, func(text string) {
			job.OutputFile = text
		})
	}

	if !w.isFieldHidden("errorFile") {
		form.AddInputField("Error File", job.ErrorFile, 50, nil, func(text string) {
			job.ErrorFile = text
		})
	}

	if !w.isFieldHidden("arraySpec") {
		form.AddInputField("Array Spec (e.g., 1-100, 1-10%5)", job.ArraySpec, 30, nil, func(text string) {
			job.ArraySpec = text
		})
	}

	if !w.isFieldHidden("exclusive") {
		form.AddCheckbox("Exclusive Node Access", job.Exclusive, func(checked bool) {
			job.Exclusive = checked
		})
	}

	if !w.isFieldHidden("requeue") {
		form.AddCheckbox("Requeue on Failure", job.Requeue, func(checked bool) {
			job.Requeue = checked
		})
	}

	if !w.isFieldHidden("dependencies") {
		depStr := strings.Join(job.Dependencies, ",")
		form.AddInputField("Dependencies (job IDs, comma-sep)", depStr, 50, nil, func(text string) {
			if text == "" {
				job.Dependencies = nil
			} else {
				job.Dependencies = strings.Split(text, ",")
			}
		})
	}

	if !w.isFieldHidden("constraints") {
		form.AddInputField("Constraints", job.Constraints, 50, nil, func(text string) {
			job.Constraints = text
		})
	}

	if !w.isFieldHidden("ntasks") {
		ntasksStr := ""
		if job.NTasks > 0 {
			ntasksStr = fmt.Sprintf("%d", job.NTasks)
		}
		form.AddInputField("Number of Tasks", ntasksStr, 15, nil, func(text string) {
			if text == "" {
				job.NTasks = 0
			} else if n, err := strconv.Atoi(text); err == nil && n >= 0 {
				job.NTasks = n
			}
		})
	}

	if !w.isFieldHidden("ntasksPerNode") {
		ntasksPerNodeStr := ""
		if job.NTasksPerNode > 0 {
			ntasksPerNodeStr = fmt.Sprintf("%d", job.NTasksPerNode)
		}
		form.AddInputField("Tasks per Node", ntasksPerNodeStr, 15, nil, func(text string) {
			if text == "" {
				job.NTasksPerNode = 0
			} else if n, err := strconv.Atoi(text); err == nil && n >= 0 {
				job.NTasksPerNode = n
			}
		})
	}

	if !w.isFieldHidden("gres") {
		form.AddInputField("GRES (e.g., gpu:2,shard:4)", job.Gres, 50, nil, func(text string) {
			job.Gres = text
		})
	}

	if !w.isFieldHidden("hold") {
		form.AddCheckbox("Submit in Held State", job.Hold, func(checked bool) {
			job.Hold = checked
		})
	}

	if !w.isFieldHidden("reservation") {
		form.AddInputField("Reservation", job.Reservation, 30, nil, func(text string) {
			job.Reservation = text
		})
	}

	if !w.isFieldHidden("licenses") {
		form.AddInputField("Licenses (e.g., matlab:1)", job.Licenses, 30, nil, func(text string) {
			job.Licenses = text
		})
	}

	if !w.isFieldHidden("wckey") {
		form.AddInputField("WC Key", job.Wckey, 30, nil, func(text string) {
			job.Wckey = text
		})
	}

	if !w.isFieldHidden("excludeNodes") {
		form.AddInputField("Exclude Nodes", job.ExcludeNodes, 50, nil, func(text string) {
			job.ExcludeNodes = text
		})
	}

	if !w.isFieldHidden("priority") {
		prioStr := ""
		if job.Priority > 0 {
			prioStr = fmt.Sprintf("%d", job.Priority)
		}
		form.AddInputField("Priority", prioStr, 15, nil, func(text string) {
			if text == "" {
				job.Priority = 0
			} else if n, err := strconv.Atoi(text); err == nil && n >= 0 {
				job.Priority = n
			}
		})
	}

	if !w.isFieldHidden("nice") {
		niceStr := ""
		if job.Nice != 0 {
			niceStr = fmt.Sprintf("%d", job.Nice)
		}
		form.AddInputField("Nice Adjustment", niceStr, 15, nil, func(text string) {
			if text == "" {
				job.Nice = 0
			} else if n, err := strconv.Atoi(text); err == nil {
				job.Nice = n
			}
		})
	}

	if !w.isFieldHidden("memoryPerCPU") {
		form.AddInputField("Memory per CPU (e.g., 4G, 1024M)", job.MemoryPerCPU, 30, nil, func(text string) {
			job.MemoryPerCPU = text
		})
	}

	if !w.isFieldHidden("beginTime") {
		form.AddInputField("Begin Time", job.BeginTime, 30, nil, func(text string) {
			job.BeginTime = text
		})
	}

	if !w.isFieldHidden("comment") {
		form.AddInputField("Comment", job.Comment, 50, nil, func(text string) {
			job.Comment = text
		})
	}

	if !w.isFieldHidden("distribution") {
		form.AddInputField("Distribution (block, cyclic, plane, arbitrary)", job.Distribution, 30, nil, func(text string) {
			job.Distribution = text
		})
	}

	if !w.isFieldHidden("prefer") {
		form.AddInputField("Preferred Features", job.Prefer, 50, nil, func(text string) {
			job.Prefer = text
		})
	}

	if !w.isFieldHidden("requiredNodes") {
		form.AddInputField("Required Nodes", job.RequiredNodes, 50, nil, func(text string) {
			job.RequiredNodes = text
		})
	}

	if !w.isFieldHidden("standardInput") {
		form.AddInputField("Standard Input File", job.StandardInput, 50, nil, func(text string) {
			job.StandardInput = text
		})
	}

	if !w.isFieldHidden("container") {
		form.AddInputField("Container Path", job.Container, 50, nil, func(text string) {
			job.Container = text
		})
	}

	if !w.isFieldHidden("threadsPerCore") {
		tpcStr := ""
		if job.ThreadsPerCore > 0 {
			tpcStr = fmt.Sprintf("%d", job.ThreadsPerCore)
		}
		form.AddInputField("Threads per Core", tpcStr, 15, nil, func(text string) {
			if text == "" {
				job.ThreadsPerCore = 0
			} else if n, err := strconv.Atoi(text); err == nil && n >= 0 {
				job.ThreadsPerCore = n
			}
		})
	}

	if !w.isFieldHidden("tasksPerCore") {
		tpcStr := ""
		if job.TasksPerCore > 0 {
			tpcStr = fmt.Sprintf("%d", job.TasksPerCore)
		}
		form.AddInputField("Tasks per Core", tpcStr, 15, nil, func(text string) {
			if text == "" {
				job.TasksPerCore = 0
			} else if n, err := strconv.Atoi(text); err == nil && n >= 0 {
				job.TasksPerCore = n
			}
		})
	}

	if !w.isFieldHidden("tasksPerSocket") {
		tpsStr := ""
		if job.TasksPerSocket > 0 {
			tpsStr = fmt.Sprintf("%d", job.TasksPerSocket)
		}
		form.AddInputField("Tasks per Socket", tpsStr, 15, nil, func(text string) {
			if text == "" {
				job.TasksPerSocket = 0
			} else if n, err := strconv.Atoi(text); err == nil && n >= 0 {
				job.TasksPerSocket = n
			}
		})
	}

	if !w.isFieldHidden("socketsPerNode") {
		spnStr := ""
		if job.SocketsPerNode > 0 {
			spnStr = fmt.Sprintf("%d", job.SocketsPerNode)
		}
		form.AddInputField("Sockets per Node", spnStr, 15, nil, func(text string) {
			if text == "" {
				job.SocketsPerNode = 0
			} else if n, err := strconv.Atoi(text); err == nil && n >= 0 {
				job.SocketsPerNode = n
			}
		})
	}

	if !w.isFieldHidden("maximumNodes") {
		mnStr := ""
		if job.MaximumNodes > 0 {
			mnStr = fmt.Sprintf("%d", job.MaximumNodes)
		}
		form.AddInputField("Maximum Nodes", mnStr, 15, nil, func(text string) {
			if text == "" {
				job.MaximumNodes = 0
			} else if n, err := strconv.Atoi(text); err == nil && n >= 0 {
				job.MaximumNodes = n
			}
		})
	}

	if !w.isFieldHidden("maximumCPUs") {
		mcStr := ""
		if job.MaximumCPUs > 0 {
			mcStr = fmt.Sprintf("%d", job.MaximumCPUs)
		}
		form.AddInputField("Maximum CPUs", mcStr, 15, nil, func(text string) {
			if text == "" {
				job.MaximumCPUs = 0
			} else if n, err := strconv.Atoi(text); err == nil && n >= 0 {
				job.MaximumCPUs = n
			}
		})
	}

	if !w.isFieldHidden("minimumCPUsPerNode") {
		mcpnStr := ""
		if job.MinimumCPUsPerNode > 0 {
			mcpnStr = fmt.Sprintf("%d", job.MinimumCPUsPerNode)
		}
		form.AddInputField("Minimum CPUs per Node", mcpnStr, 15, nil, func(text string) {
			if text == "" {
				job.MinimumCPUsPerNode = 0
			} else if n, err := strconv.Atoi(text); err == nil && n >= 0 {
				job.MinimumCPUsPerNode = n
			}
		})
	}

	if !w.isFieldHidden("timeMinimum") {
		form.AddInputField("Time Minimum (HH:MM:SS)", job.TimeMinimum, 30, nil, func(text string) {
			job.TimeMinimum = text
		})
	}

	if !w.isFieldHidden("contiguous") {
		form.AddCheckbox("Contiguous Nodes", job.Contiguous, func(checked bool) {
			job.Contiguous = checked
		})
	}

	if !w.isFieldHidden("overcommit") {
		form.AddCheckbox("Overcommit Resources", job.Overcommit, func(checked bool) {
			job.Overcommit = checked
		})
	}

	if !w.isFieldHidden("killOnNodeFail") {
		form.AddCheckbox("Kill on Node Failure", job.KillOnNodeFail, func(checked bool) {
			job.KillOnNodeFail = checked
		})
	}

	if !w.isFieldHidden("waitAllNodes") {
		form.AddCheckbox("Wait for All Nodes", job.WaitAllNodes, func(checked bool) {
			job.WaitAllNodes = checked
		})
	}

	if !w.isFieldHidden("openMode") {
		form.AddInputField("Open Mode (append or truncate)", job.OpenMode, 15, nil, func(text string) {
			job.OpenMode = text
		})
	}

	if !w.isFieldHidden("tresPerTask") {
		form.AddInputField("TRES per Task", job.TRESPerTask, 50, nil, func(text string) {
			job.TRESPerTask = text
		})
	}

	if !w.isFieldHidden("tresPerSocket") {
		form.AddInputField("TRES per Socket", job.TRESPerSocket, 50, nil, func(text string) {
			job.TRESPerSocket = text
		})
	}

	if !w.isFieldHidden("signal") {
		form.AddInputField("Signal ([B:]sig[@time])", job.Signal, 30, nil, func(text string) {
			job.Signal = text
		})
	}

	if !w.isFieldHidden("tmpDiskPerNode") {
		tmpStr := ""
		if job.TmpDiskPerNode > 0 {
			tmpStr = fmt.Sprintf("%d", job.TmpDiskPerNode)
		}
		form.AddInputField("Tmp Disk per Node (MB)", tmpStr, 15, nil, func(text string) {
			if text == "" {
				job.TmpDiskPerNode = 0
			} else if n, err := strconv.Atoi(text); err == nil && n >= 0 {
				job.TmpDiskPerNode = n
			}
		})
	}

	if !w.isFieldHidden("deadline") {
		form.AddInputField("Deadline", job.Deadline, 30, nil, func(text string) {
			job.Deadline = text
		})
	}

	if !w.isFieldHidden("ntasksPerTRES") {
		nptStr := ""
		if job.NTasksPerTRES > 0 {
			nptStr = fmt.Sprintf("%d", job.NTasksPerTRES)
		}
		form.AddInputField("Tasks per GPU (--ntasks-per-gpu)", nptStr, 15, nil, func(text string) {
			if text == "" {
				job.NTasksPerTRES = 0
			} else if n, err := strconv.Atoi(text); err == nil && n >= 0 {
				job.NTasksPerTRES = n
			}
		})
	}

	if !w.isFieldHidden("cpuBinding") {
		form.AddInputField("CPU Bind (cores, rank, map_cpu:)", job.CPUBinding, 50, nil, func(text string) {
			job.CPUBinding = text
		})
	}

	if !w.isFieldHidden("cpuFrequency") {
		form.AddInputField("CPU Frequency (low, medium, high, or KHz)", job.CPUFrequency, 30, nil, func(text string) {
			job.CPUFrequency = text
		})
	}

	if !w.isFieldHidden("network") {
		form.AddInputField("Network", job.Network, 50, nil, func(text string) {
			job.Network = text
		})
	}

	if !w.isFieldHidden("x11") {
		form.AddInputField("X11 Forwarding (batch, first, last, all)", job.X11, 15, nil, func(text string) {
			job.X11 = text
		})
	}

	if !w.isFieldHidden("immediate") {
		form.AddCheckbox("Immediate (fail if no resources)", job.Immediate, func(checked bool) {
			job.Immediate = checked
		})
	}

	if !w.isFieldHidden("burstBuffer") {
		form.AddInputField("Burst Buffer Spec", job.BurstBuffer, 50, nil, func(text string) {
			job.BurstBuffer = text
		})
	}

	if !w.isFieldHidden("batchFeatures") {
		form.AddInputField("Batch Features", job.BatchFeatures, 50, nil, func(text string) {
			job.BatchFeatures = text
		})
	}

	if !w.isFieldHidden("tresBind") {
		form.AddInputField("TRES Bind (gres/gpu:closest)", job.TRESBind, 50, nil, func(text string) {
			job.TRESBind = text
		})
	}

	if !w.isFieldHidden("tresFreq") {
		form.AddInputField("TRES Frequency", job.TRESFreq, 30, nil, func(text string) {
			job.TRESFreq = text
		})
	}

	if !w.isFieldHidden("coreSpecification") {
		csStr := ""
		if job.CoreSpecification > 0 {
			csStr = fmt.Sprintf("%d", job.CoreSpecification)
		}
		form.AddInputField("Core Spec (reserved system cores)", csStr, 15, nil, func(text string) {
			if text == "" {
				job.CoreSpecification = 0
			} else if n, err := strconv.Atoi(text); err == nil && n >= 0 {
				job.CoreSpecification = n
			}
		})
	}

	if !w.isFieldHidden("threadSpecification") {
		tsStr := ""
		if job.ThreadSpecification > 0 {
			tsStr = fmt.Sprintf("%d", job.ThreadSpecification)
		}
		form.AddInputField("Thread Spec (reserved system threads)", tsStr, 15, nil, func(text string) {
			if text == "" {
				job.ThreadSpecification = 0
			} else if n, err := strconv.Atoi(text); err == nil && n >= 0 {
				job.ThreadSpecification = n
			}
		})
	}

	if !w.isFieldHidden("memoryBinding") {
		form.AddInputField("Memory Bind (local, rank, map_mem:)", job.MemoryBinding, 50, nil, func(text string) {
			job.MemoryBinding = text
		})
	}

	if !w.isFieldHidden("minimumCPUs") {
		mcStr := ""
		if job.MinimumCPUs > 0 {
			mcStr = fmt.Sprintf("%d", job.MinimumCPUs)
		}
		form.AddInputField("Minimum CPUs (total floor)", mcStr, 15, nil, func(text string) {
			if text == "" {
				job.MinimumCPUs = 0
			} else if n, err := strconv.Atoi(text); err == nil && n >= 0 {
				job.MinimumCPUs = n
			}
		})
	}

	if !w.isFieldHidden("tresPerJob") {
		form.AddInputField("TRES per Job", job.TRESPerJob, 50, nil, func(text string) {
			job.TRESPerJob = text
		})
	}

	if !w.isFieldHidden("cpusPerTRES") {
		form.AddInputField("CPUs per GPU (--cpus-per-gpu)", job.CPUsPerTRES, 30, nil, func(text string) {
			job.CPUsPerTRES = text
		})
	}

	if !w.isFieldHidden("memoryPerTRES") {
		form.AddInputField("Memory per GPU (--mem-per-gpu)", job.MemoryPerTRES, 30, nil, func(text string) {
			job.MemoryPerTRES = text
		})
	}

	if !w.isFieldHidden("argv") {
		form.AddInputField("Script Arguments", job.Argv, 50, nil, func(text string) {
			job.Argv = text
		})
	}

	if !w.isFieldHidden("flags") {
		form.AddInputField("Job Flags (SPREAD_JOB, etc.)", job.Flags, 50, nil, func(text string) {
			job.Flags = text
		})
	}

	if !w.isFieldHidden("profile") {
		form.AddInputField("Profile (ENERGY, NETWORK, TASK)", job.ProfileTypes, 50, nil, func(text string) {
			job.ProfileTypes = text
		})
	}

	if !w.isFieldHidden("cpuBindingFlags") {
		form.AddInputField("CPU Bind Flags (VERBOSE, etc.)", job.CPUBindingFlags, 50, nil, func(text string) {
			job.CPUBindingFlags = text
		})
	}

	if !w.isFieldHidden("memoryBindingType") {
		form.AddInputField("Memory Bind Type (LOCAL, RANK, etc.)", job.MemoryBindingType, 30, nil, func(text string) {
			job.MemoryBindingType = text
		})
	}

	if !w.isFieldHidden("requiredSwitches") {
		rsStr := ""
		if job.RequiredSwitches > 0 {
			rsStr = fmt.Sprintf("%d", job.RequiredSwitches)
		}
		form.AddInputField("Required Switches", rsStr, 15, nil, func(text string) {
			if text == "" {
				job.RequiredSwitches = 0
			} else if n, err := strconv.Atoi(text); err == nil && n >= 0 {
				job.RequiredSwitches = n
			}
		})
	}

	if !w.isFieldHidden("waitForSwitch") {
		wfsStr := ""
		if job.WaitForSwitch > 0 {
			wfsStr = fmt.Sprintf("%d", job.WaitForSwitch)
		}
		form.AddInputField("Wait for Switch (seconds)", wfsStr, 15, nil, func(text string) {
			if text == "" {
				job.WaitForSwitch = 0
			} else if n, err := strconv.Atoi(text); err == nil && n >= 0 {
				job.WaitForSwitch = n
			}
		})
	}

	if !w.isFieldHidden("clusterConstraint") {
		form.AddInputField("Cluster Constraint", job.ClusterConstraint, 50, nil, func(text string) {
			job.ClusterConstraint = text
		})
	}

	if !w.isFieldHidden("clusters") {
		form.AddInputField("Clusters", job.Clusters, 50, nil, func(text string) {
			job.Clusters = text
		})
	}
}

// addEmailNotificationFields adds email notification configuration fields
func (w *JobSubmissionWizard) addEmailNotificationFields(form *tview.Form, job *dao.JobSubmission) {
	if !w.isFieldHidden("emailNotify") {
		form.AddCheckbox("Email Notifications", job.EmailNotify, func(checked bool) {
			job.EmailNotify = checked
		})
	}

	if !w.isFieldHidden("email") {
		if job.EmailNotify {
			form.AddInputField("Email Address", job.Email, 40, nil, func(text string) {
				job.Email = text
			})
		}
	}
}

// filterFieldOptions filters fetched options against config-specified allowed values.
// If fieldOptions specifies allowed values, returns the intersection with fetched values
// (preserving allowed order). Falls back to all fetched values if intersection is empty.
func (w *JobSubmissionWizard) filterFieldOptions(fetched []string, fieldName string) []string {
	if w.submissionConfig == nil || w.submissionConfig.FieldOptions == nil {
		return fetched
	}
	allowed, ok := w.submissionConfig.FieldOptions[fieldName]
	if !ok || len(allowed) == 0 {
		return fetched
	}
	if len(fetched) == 0 {
		return allowed
	}
	// Compute intersection preserving allowed order
	fetchedSet := make(map[string]bool, len(fetched))
	for _, v := range fetched {
		fetchedSet[v] = true
	}
	var intersection []string
	for _, v := range allowed {
		if fetchedSet[v] {
			intersection = append(intersection, v)
		}
	}
	if len(intersection) == 0 {
		return fetched // fallback
	}
	return intersection
}

// getConfigFieldOptions returns the configured allowed options for a field
func (w *JobSubmissionWizard) getConfigFieldOptions(fieldName string) []string {
	if w.submissionConfig == nil || w.submissionConfig.FieldOptions == nil {
		return nil
	}
	return w.submissionConfig.FieldOptions[fieldName]
}

// getStringIndex returns the index of value in options, or 0 if not found
func (w *JobSubmissionWizard) getStringIndex(options []string, value string) int {
	if value == "" {
		return 0
	}
	for i, o := range options {
		if o == value {
			return i
		}
	}
	return 0
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

	// Validate memory format (optional - empty means use SLURM's partition default)
	if job.Memory != "" && !isValidMemoryFormat(job.Memory) {
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

	helpText := tview.NewTextView().
		SetDynamicColors(true).
		SetText("[yellow]ESC[white] close  [yellow]Ctrl+Y[white] copy to clipboard")

	modal := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(textView, 0, 1, true).
		AddItem(helpText, 1, 0, false)

	modal.SetBorder(true).
		SetTitle(" Job Script Preview ").
		SetTitleAlign(tview.AlignCenter)

	modal.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEsc {
			w.pages.RemovePage("job-preview")
			return nil
		}
		if event.Key() == tcell.KeyCtrlY {
			// Generate a clean script without tview color tags
			cleanScript := generateCleanJobScript(job)
			copyToClipboard(cleanScript)
			// Brief visual feedback
			helpText.SetText("[green]Copied to clipboard!")
			go func() {
				time.Sleep(2 * time.Second)
				if w.app != nil {
					w.app.QueueUpdateDraw(func() {
						helpText.SetText("[yellow]ESC[white] close  [yellow]Ctrl+Y[white] copy to clipboard")
					})
				}
			}()
			return nil // consume the event so tview doesn't quit
		}
		return event
	})

	centered := createCenteredModal(modal, 70, 25)
	w.pages.AddPage("job-preview", centered, true, true)
}

// generateCleanJobScript produces a plain-text sbatch script without color tags
func generateCleanJobScript(job *dao.JobSubmission) string {
	colored := generateJobScript(job)
	// Strip tview color tags: [green], [white], [yellow], [cyan], etc.
	re := regexp.MustCompile(`\[[a-zA-Z]+\]`)
	return re.ReplaceAllString(colored, "")
}

// copyToClipboard writes text to the system clipboard using OSC 52 escape sequence.
// This works in most modern terminals (iTerm2, kitty, tmux, Windows Terminal, etc.)
func copyToClipboard(text string) {
	encoded := base64.StdEncoding.EncodeToString([]byte(text))
	// OSC 52: \033]52;c;<base64-data>\a
	fmt.Fprintf(os.Stderr, "\033]52;c;%s\a", encoded)
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

// BuiltinTemplates returns the hardcoded built-in job templates as an ordered slice
func BuiltinTemplates() []*dao.JobTemplate {
	return []*dao.JobTemplate{
		{
			Name:        "Basic Batch Job",
			Description: "Simple batch job for serial computations",
			JobSubmission: dao.JobSubmission{
				Name:       "basic_job",
				Partition:  "normal",
				TimeLimit:  "01:00:00",
				Nodes:      1,
				CPUs:       1,
				Memory:     "4G",
				Script:     "#!/bin/bash\n\n# Your commands here\necho \"Hello from SLURM job $SLURM_JOB_ID\"\n",
				OutputFile: "job_%j.out",
				ErrorFile:  "job_%j.err",
			},
		},
		{
			Name:        "MPI Parallel Job",
			Description: "Parallel job using MPI across multiple nodes",
			JobSubmission: dao.JobSubmission{
				Name:       "mpi_job",
				Partition:  "parallel",
				TimeLimit:  "04:00:00",
				Nodes:      4,
				CPUs:       16,
				Memory:     "8G",
				Script:     "#!/bin/bash\n\nmodule load mpi\nmpirun -np $SLURM_NTASKS ./my_mpi_program\n",
				OutputFile: "mpi_%j.out",
				ErrorFile:  "mpi_%j.err",
			},
		},
		{
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
				Script:     "#!/bin/bash\n\nmodule load cuda\n./my_gpu_program\n",
				OutputFile: "gpu_%j.out",
				ErrorFile:  "gpu_%j.err",
			},
		},
		{
			Name:        "Array Job",
			Description: "Array job for processing multiple similar tasks",
			JobSubmission: dao.JobSubmission{
				Name:       "array_job",
				Partition:  "normal",
				TimeLimit:  "00:30:00",
				Nodes:      1,
				CPUs:       1,
				Memory:     "2G",
				Script:     "#!/bin/bash\n\n# Process file based on array task ID\n./process_file.sh input_${SLURM_ARRAY_TASK_ID}.dat\n",
				OutputFile: "array_%A_%a.out",
				ErrorFile:  "array_%A_%a.err",
			},
		},
		{
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
		},
		{
			Name:        "Long-Running Job",
			Description: "Job for long-running computations (> 24 hours)",
			JobSubmission: dao.JobSubmission{
				Name:        "long_job",
				Partition:   "long",
				TimeLimit:   "7-00:00:00",
				Nodes:       1,
				CPUs:        8,
				Memory:      "32G",
				Script:      "#!/bin/bash\n\n# Enable checkpointing\n./my_long_computation --checkpoint-interval=3600\n",
				OutputFile:  "long_%j.out",
				ErrorFile:   "long_%j.err",
				EmailNotify: true,
			},
		},
		{
			Name:        "High Memory Job",
			Description: "Job requiring large memory allocation",
			JobSubmission: dao.JobSubmission{
				Name:       "highmem_job",
				Partition:  "highmem",
				TimeLimit:  "12:00:00",
				Nodes:      1,
				CPUs:       32,
				Memory:     "256G",
				Script:     "#!/bin/bash\n\n# Large memory computation\n./memory_intensive_program\n",
				OutputFile: "highmem_%j.out",
				ErrorFile:  "highmem_%j.err",
			},
		},
		{
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
				Script:     "#!/bin/bash\n\n# Debug commands\nset -x\necho \"Debug output\"\n./my_program --debug\n",
				OutputFile: "debug_%j.out",
				ErrorFile:  "debug_%j.err",
			},
		},
	}
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

//nolint:cyclop // linear script preview builder for all fields
func generateJobScript(job *dao.JobSubmission) string {
	var script strings.Builder

	script.WriteString("[yellow]#!/bin/bash[white]\n")
	script.WriteString(fmt.Sprintf("[green]#SBATCH --job-name=[white]%s\n", job.Name))
	script.WriteString(fmt.Sprintf("[green]#SBATCH --partition=[white]%s\n", job.Partition))
	script.WriteString(fmt.Sprintf("[green]#SBATCH --time=[white]%s\n", job.TimeLimit))
	if job.MaximumNodes > 0 {
		script.WriteString(fmt.Sprintf("[green]#SBATCH --nodes=[white]%d-%d\n", job.Nodes, job.MaximumNodes))
	} else {
		script.WriteString(fmt.Sprintf("[green]#SBATCH --nodes=[white]%d\n", job.Nodes))
	}
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

	if job.ArraySpec != "" {
		script.WriteString(fmt.Sprintf("[green]#SBATCH --array=[white]%s\n", job.ArraySpec))
	}

	if job.Exclusive {
		script.WriteString("[green]#SBATCH --exclusive[white]\n")
	}

	if job.Requeue {
		script.WriteString("[green]#SBATCH --requeue[white]\n")
	}

	if len(job.Dependencies) > 0 {
		script.WriteString(fmt.Sprintf("[green]#SBATCH --dependency=[white]afterok:%s\n", strings.Join(job.Dependencies, ":")))
	}

	if job.Constraints != "" {
		script.WriteString(fmt.Sprintf("[green]#SBATCH --constraint=[white]%s\n", job.Constraints))
	}

	if job.NTasks > 0 {
		script.WriteString(fmt.Sprintf("[green]#SBATCH --ntasks=[white]%d\n", job.NTasks))
	}

	if job.NTasksPerNode > 0 {
		script.WriteString(fmt.Sprintf("[green]#SBATCH --ntasks-per-node=[white]%d\n", job.NTasksPerNode))
	}

	if job.Gres != "" {
		script.WriteString(fmt.Sprintf("[green]#SBATCH --gres=[white]%s\n", job.Gres))
	}

	if job.Hold {
		script.WriteString("[green]#SBATCH --hold[white]\n")
	}

	if job.Reservation != "" {
		script.WriteString(fmt.Sprintf("[green]#SBATCH --reservation=[white]%s\n", job.Reservation))
	}

	if job.Licenses != "" {
		script.WriteString(fmt.Sprintf("[green]#SBATCH --licenses=[white]%s\n", job.Licenses))
	}

	if job.Wckey != "" {
		script.WriteString(fmt.Sprintf("[green]#SBATCH --wckey=[white]%s\n", job.Wckey))
	}

	if job.ExcludeNodes != "" {
		script.WriteString(fmt.Sprintf("[green]#SBATCH --exclude=[white]%s\n", job.ExcludeNodes))
	}

	if job.Priority > 0 {
		script.WriteString(fmt.Sprintf("[green]#SBATCH --priority=[white]%d\n", job.Priority))
	}

	if job.Nice != 0 {
		script.WriteString(fmt.Sprintf("[green]#SBATCH --nice=[white]%d\n", job.Nice))
	}

	if job.MemoryPerCPU != "" {
		script.WriteString(fmt.Sprintf("[green]#SBATCH --mem-per-cpu=[white]%s\n", job.MemoryPerCPU))
	}

	if job.BeginTime != "" {
		script.WriteString(fmt.Sprintf("[green]#SBATCH --begin=[white]%s\n", job.BeginTime))
	}

	if job.Comment != "" {
		script.WriteString(fmt.Sprintf("[green]#SBATCH --comment=[white]%s\n", job.Comment))
	}

	if job.Distribution != "" {
		script.WriteString(fmt.Sprintf("[green]#SBATCH --distribution=[white]%s\n", job.Distribution))
	}

	if job.Prefer != "" {
		script.WriteString(fmt.Sprintf("[green]#SBATCH --prefer=[white]%s\n", job.Prefer))
	}

	if job.RequiredNodes != "" {
		script.WriteString(fmt.Sprintf("[green]#SBATCH --nodelist=[white]%s\n", job.RequiredNodes))
	}

	if job.StandardInput != "" {
		script.WriteString(fmt.Sprintf("[green]#SBATCH --input=[white]%s\n", job.StandardInput))
	}

	if job.Container != "" {
		script.WriteString(fmt.Sprintf("[green]#SBATCH --container=[white]%s\n", job.Container))
	}

	if job.ThreadsPerCore > 0 {
		script.WriteString(fmt.Sprintf("[green]#SBATCH --threads-per-core=[white]%d\n", job.ThreadsPerCore))
	}

	if job.TasksPerCore > 0 {
		script.WriteString(fmt.Sprintf("[green]#SBATCH --ntasks-per-core=[white]%d\n", job.TasksPerCore))
	}

	if job.TasksPerSocket > 0 {
		script.WriteString(fmt.Sprintf("[green]#SBATCH --ntasks-per-socket=[white]%d\n", job.TasksPerSocket))
	}

	if job.SocketsPerNode > 0 {
		script.WriteString(fmt.Sprintf("[green]#SBATCH --sockets-per-node=[white]%d\n", job.SocketsPerNode))
	}

	if job.MaximumCPUs > 0 {
		script.WriteString(fmt.Sprintf("[green]# --cpus range up to [white]%d\n", job.MaximumCPUs))
	}

	if job.MinimumCPUsPerNode > 0 {
		script.WriteString(fmt.Sprintf("[green]#SBATCH --mincpus=[white]%d\n", job.MinimumCPUsPerNode))
	}

	if job.TimeMinimum != "" {
		script.WriteString(fmt.Sprintf("[green]#SBATCH --time-min=[white]%s\n", job.TimeMinimum))
	}

	if job.Contiguous {
		script.WriteString("[green]#SBATCH --contiguous[white]\n")
	}

	if job.Overcommit {
		script.WriteString("[green]#SBATCH --overcommit[white]\n")
	}

	if job.KillOnNodeFail {
		script.WriteString("[green]#SBATCH --no-kill=off[white]\n")
	}

	if job.WaitAllNodes {
		script.WriteString("[green]#SBATCH --wait-all-nodes=1[white]\n")
	}

	if job.OpenMode != "" {
		script.WriteString(fmt.Sprintf("[green]#SBATCH --open-mode=[white]%s\n", job.OpenMode))
	}

	if job.TRESPerTask != "" {
		script.WriteString(fmt.Sprintf("[green]#SBATCH --tres-per-task=[white]%s\n", job.TRESPerTask))
	}

	if job.TRESPerSocket != "" {
		script.WriteString(fmt.Sprintf("[green]#SBATCH --tres-per-socket=[white]%s\n", job.TRESPerSocket))
	}

	if job.Signal != "" {
		script.WriteString(fmt.Sprintf("[green]#SBATCH --signal=[white]%s\n", job.Signal))
	}

	if job.TmpDiskPerNode > 0 {
		script.WriteString(fmt.Sprintf("[green]#SBATCH --tmp=[white]%dM\n", job.TmpDiskPerNode))
	}

	if job.Deadline != "" {
		script.WriteString(fmt.Sprintf("[green]#SBATCH --deadline=[white]%s\n", job.Deadline))
	}

	if job.NTasksPerTRES > 0 {
		script.WriteString(fmt.Sprintf("[green]#SBATCH --ntasks-per-gpu=[white]%d\n", job.NTasksPerTRES))
	}

	if job.CPUBinding != "" {
		script.WriteString(fmt.Sprintf("[green]#SBATCH --cpu-bind=[white]%s\n", job.CPUBinding))
	}

	if job.CPUFrequency != "" {
		script.WriteString(fmt.Sprintf("[green]#SBATCH --cpu-freq=[white]%s\n", job.CPUFrequency))
	}

	if job.Network != "" {
		script.WriteString(fmt.Sprintf("[green]#SBATCH --network=[white]%s\n", job.Network))
	}

	if job.X11 != "" {
		script.WriteString(fmt.Sprintf("[green]#SBATCH --x11=[white]%s\n", job.X11))
	}

	if job.Immediate {
		script.WriteString("[green]#SBATCH --immediate[white]\n")
	}

	if job.BurstBuffer != "" {
		script.WriteString(fmt.Sprintf("[green]#SBATCH --bb=[white]%s\n", job.BurstBuffer))
	}

	if job.BatchFeatures != "" {
		script.WriteString(fmt.Sprintf("[green]#SBATCH --batch=[white]%s\n", job.BatchFeatures))
	}

	if job.TRESBind != "" {
		script.WriteString(fmt.Sprintf("[green]#SBATCH --tres-bind=[white]%s\n", job.TRESBind))
	}

	if job.TRESFreq != "" {
		script.WriteString(fmt.Sprintf("[green]#SBATCH --tres-freq=[white]%s\n", job.TRESFreq))
	}

	if job.CoreSpecification > 0 {
		script.WriteString(fmt.Sprintf("[green]#SBATCH --core-spec=[white]%d\n", job.CoreSpecification))
	}

	if job.ThreadSpecification > 0 {
		script.WriteString(fmt.Sprintf("[green]#SBATCH --thread-spec=[white]%d\n", job.ThreadSpecification))
	}

	if job.MemoryBinding != "" {
		script.WriteString(fmt.Sprintf("[green]#SBATCH --mem-bind=[white]%s\n", job.MemoryBinding))
	}

	if job.MinimumCPUs > 0 {
		script.WriteString(fmt.Sprintf("[green]# minimum total CPUs=[white]%d\n", job.MinimumCPUs))
	}

	if job.TRESPerJob != "" {
		script.WriteString(fmt.Sprintf("[green]#SBATCH --tres-per-job=[white]%s\n", job.TRESPerJob))
	}

	if job.CPUsPerTRES != "" {
		script.WriteString(fmt.Sprintf("[green]#SBATCH --cpus-per-gpu=[white]%s\n", job.CPUsPerTRES))
	}

	if job.MemoryPerTRES != "" {
		script.WriteString(fmt.Sprintf("[green]#SBATCH --mem-per-gpu=[white]%s\n", job.MemoryPerTRES))
	}

	if job.ProfileTypes != "" {
		script.WriteString(fmt.Sprintf("[green]#SBATCH --profile=[white]%s\n", job.ProfileTypes))
	}

	if job.RequiredSwitches > 0 {
		if job.WaitForSwitch > 0 {
			script.WriteString(fmt.Sprintf("[green]#SBATCH --switches=[white]%d@%d\n", job.RequiredSwitches, job.WaitForSwitch))
		} else {
			script.WriteString(fmt.Sprintf("[green]#SBATCH --switches=[white]%d\n", job.RequiredSwitches))
		}
	}

	if job.ClusterConstraint != "" {
		script.WriteString(fmt.Sprintf("[green]#SBATCH --cluster-constraint=[white]%s\n", job.ClusterConstraint))
	}

	if job.Clusters != "" {
		script.WriteString(fmt.Sprintf("[green]#SBATCH --clusters=[white]%s\n", job.Clusters))
	}

	if job.Flags != "" {
		script.WriteString(fmt.Sprintf("[green]# Job flags: [white]%s\n", job.Flags))
	}

	if job.EmailNotify && job.Email != "" {
		script.WriteString("[green]#SBATCH --mail-type=[white]ALL\n")
		script.WriteString(fmt.Sprintf("[green]#SBATCH --mail-user=[white]%s\n", job.Email))
	}

	script.WriteString("\n[cyan]# Job script[white]\n")
	// Strip shebang from script body — it's already at the top of the preview
	scriptBody := job.Script
	scriptBody = strings.TrimPrefix(scriptBody, "#!/bin/bash\n")
	scriptBody = strings.TrimPrefix(scriptBody, "#!/bin/sh\n")
	scriptBody = strings.TrimLeft(scriptBody, "\n")
	script.WriteString(scriptBody)

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

// getAvailableQoS fetches the list of available QoS from the SLURM cluster
func (w *JobSubmissionWizard) getAvailableQoS() []string {
	qosList, _ := w.client.QoS().List()
	if qosList == nil || len(qosList.QoS) == 0 {
		return []string{}
	}

	var result []string
	for _, q := range qosList.QoS {
		if q != nil {
			result = append(result, q.Name)
		}
	}
	return result
}

// getCurrentUser fetches the current OS user's SLURM user record
func (w *JobSubmissionWizard) getCurrentUser() *dao.User {
	if w.slurmUser == "" {
		return nil
	}
	user, _ := w.client.Users().Get(w.slurmUser)
	return user
}
