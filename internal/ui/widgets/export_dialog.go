package widgets

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/jontk/s9s/internal/export"
	"github.com/jontk/s9s/internal/performance"
	"github.com/rivo/tview"
)

// ExportType represents the type of data to export
type ExportType string

const (
	// ExportTypeJobOutput is the export type for job output.
	ExportTypeJobOutput ExportType = "job_output"
	// ExportTypePerformance is the export type for performance data.
	ExportTypePerformance ExportType = "performance"
)

// ExportDialog handles export functionality for various data types
type ExportDialog struct {
	*tview.Flex
	form           *tview.Form
	exportType     ExportType
	selectedFormat export.ExportFormat
	customPath     string
	onExport       func(format export.ExportFormat, path string)
	onCancel       func()
}

// NewExportDialog creates a new export dialog
func NewExportDialog(exportType ExportType) *ExportDialog {
	ed := &ExportDialog{
		Flex:       tview.NewFlex(),
		exportType: exportType,
	}

	ed.setupUI()
	return ed
}

// setupUI creates the dialog UI
func (ed *ExportDialog) setupUI() {
	// Create form
	ed.form = tview.NewForm()
	ed.form.SetBorder(true)
	ed.form.SetBorderPadding(1, 1, 2, 2)

	title := "Export "
	switch ed.exportType {
	case ExportTypeJobOutput:
		title += "Job Output"
	case ExportTypePerformance:
		title += "Performance Report"
	}
	ed.form.SetTitle(title)
	ed.form.SetTitleAlign(tview.AlignCenter)

	// Format dropdown
	formats := []string{"Text", "JSON", "CSV", "Markdown", "HTML"}
	ed.form.AddDropDown("Format:", formats, 0, func(option string, _ int) {
		switch option {
		case "Text":
			ed.selectedFormat = export.FormatText
		case "JSON":
			ed.selectedFormat = export.FormatJSON
		case "CSV":
			ed.selectedFormat = export.FormatCSV
		case "Markdown":
			ed.selectedFormat = export.FormatMarkdown
		case "HTML":
			ed.selectedFormat = export.FormatHTML
		}
	})
	ed.selectedFormat = export.FormatText // Default

	// Path input
	homeDir, _ := os.UserHomeDir()
	defaultPath := filepath.Join(homeDir, "slurm_exports")
	ed.form.AddInputField("Path:", defaultPath, 50, nil, func(text string) {
		ed.customPath = text
	})
	ed.customPath = defaultPath

	// Preview text
	previewText := "Preview:\n"
	switch ed.exportType {
	case ExportTypeJobOutput:
		previewText += "• Job ID and name\n"
		previewText += "• Output type (stdout/stderr)\n"
		previewText += "• Full output content\n"
		previewText += "• Timestamp information"
	case ExportTypePerformance:
		previewText += "• System metrics (CPU, Memory, etc.)\n"
		previewText += "• Operation statistics\n"
		previewText += "• Network performance\n"
		previewText += "• Optimization recommendations"
	}

	preview := tview.NewTextView().
		SetText(previewText).
		SetTextColor(tcell.ColorGray)
	preview.SetBorder(true).SetTitle("Export Contents")

	// Buttons
	ed.form.AddButton("Export", func() {
		if ed.onExport != nil {
			ed.onExport(ed.selectedFormat, ed.customPath)
		}
	})

	ed.form.AddButton("Cancel", func() {
		if ed.onCancel != nil {
			ed.onCancel()
		}
	})

	// Help text
	helpText := "[Tab] Navigate fields | [Enter] Select | [Esc] Cancel"
	help := tview.NewTextView().
		SetText(helpText).
		SetTextAlign(tview.AlignCenter).
		SetTextColor(tcell.ColorGray)

	// Layout
	ed.SetDirection(tview.FlexRow)
	ed.AddItem(ed.form, 0, 2, true)
	ed.AddItem(preview, 0, 1, false)
	ed.AddItem(help, 1, 0, false)
}

// SetExportHandler sets the export callback
func (ed *ExportDialog) SetExportHandler(handler func(format export.ExportFormat, path string)) {
	ed.onExport = handler
}

// SetCancelHandler sets the cancel callback
func (ed *ExportDialog) SetCancelHandler(handler func()) {
	ed.onCancel = handler
}

// Focus implements tview.Primitive
func (ed *ExportDialog) Focus(delegate func(p tview.Primitive)) {
	delegate(ed.form)
}

// JobOutputExportDialog specializes ExportDialog for job output
type JobOutputExportDialog struct {
	*ExportDialog
	jobID      string
	jobName    string
	outputType string
	content    string
	exporter   *export.JobOutputExporter
}

// NewJobOutputExportDialog creates a new job output export dialog
func NewJobOutputExportDialog(jobID, jobName, outputType, content string) *JobOutputExportDialog {
	dialog := &JobOutputExportDialog{
		ExportDialog: NewExportDialog(ExportTypeJobOutput),
		jobID:        jobID,
		jobName:      jobName,
		outputType:   outputType,
		content:      content,
		exporter:     export.NewJobOutputExporter(""),
	}

	// Update title with job info
	dialog.form.SetTitle(fmt.Sprintf("Export Job Output - %s (%s)", jobID, outputType))

	// Set export handler
	dialog.SetExportHandler(func(format export.ExportFormat, path string) {
		dialog.performExport(format, path)
	})

	return dialog
}

// performExport executes the export
func (jed *JobOutputExportDialog) performExport(format export.ExportFormat, basePath string) {
	// Set custom path if provided
	if basePath != "" {
		jed.exporter.SetDefaultPath(basePath)
	}

	// Create job data
	data := export.JobOutputData{
		JobID:      jed.jobID,
		JobName:    jed.jobName,
		OutputType: jed.outputType,
		Content:    jed.content,
	}

	// Export
	result, err := jed.exporter.Export(data, format, "")
	if err != nil {
		// Show error (in a real app, this would be a proper error dialog)
		jed.form.SetTitle(fmt.Sprintf("Export Failed: %v", err))
		return
	}

	// Show success
	jed.form.SetTitle(fmt.Sprintf("Exported to: %s", result.FilePath))
}

// PerformanceExportDialog specializes ExportDialog for performance reports
type PerformanceExportDialog struct {
	*ExportDialog
	profiler  *performance.Profiler
	optimizer *performance.Optimizer
	exporter  *export.PerformanceExporter
}

// NewPerformanceExportDialog creates a new performance export dialog
func NewPerformanceExportDialog(profiler *performance.Profiler, optimizer *performance.Optimizer) *PerformanceExportDialog {
	dialog := &PerformanceExportDialog{
		ExportDialog: NewExportDialog(ExportTypePerformance),
		profiler:     profiler,
		optimizer:    optimizer,
		exporter:     export.NewPerformanceExporter(""),
	}

	// Set export handler
	dialog.SetExportHandler(func(format export.ExportFormat, path string) {
		dialog.performExport(format, path)
	})

	return dialog
}

// performExport executes the export
func (ped *PerformanceExportDialog) performExport(format export.ExportFormat, basePath string) {
	// Set custom path if provided
	if basePath != "" {
		ped.exporter.SetDefaultPath(basePath)
	}

	// Export
	result, err := ped.exporter.ExportPerformanceReport(ped.profiler, ped.optimizer, format, "")
	if err != nil {
		// Show error
		ped.form.SetTitle(fmt.Sprintf("Export Failed: %v", err))
		return
	}

	// Show success
	ped.form.SetTitle(fmt.Sprintf("Exported to: %s", result.FilePath))
}

// BatchExportDialog handles exporting multiple items
type BatchExportDialog struct {
	*tview.Flex
	list           *tview.List
	form           *tview.Form
	selectedItems  map[int]bool
	exportType     ExportType
	selectedFormat export.ExportFormat
	customPath     string
	onExport       func(indices []int, format export.ExportFormat, path string)
	onCancel       func()
}

// NewBatchExportDialog creates a new batch export dialog
func NewBatchExportDialog(exportType ExportType, items []string) *BatchExportDialog {
	bd := &BatchExportDialog{
		Flex:          tview.NewFlex(),
		exportType:    exportType,
		selectedItems: make(map[int]bool),
	}

	bd.setupBatchUI(items)
	return bd
}

// setupBatchUI creates the batch export UI
func (bd *BatchExportDialog) setupBatchUI(items []string) {
	bd.setupItemList(items)
	bd.setupExportForm()

	// Layout
	bd.SetDirection(tview.FlexRow)
	bd.AddItem(bd.list, 0, 2, true)
	bd.AddItem(bd.form, 0, 1, false)
}

// setupItemList initializes the item selection list
func (bd *BatchExportDialog) setupItemList(items []string) {
	bd.list = tview.NewList()
	bd.list.ShowSecondaryText(false)
	bd.list.SetBorder(true)
	bd.list.SetTitle("Select Items to Export")

	for _, item := range items {
		prefix := "[ ] "
		bd.list.AddItem(prefix+item, "", 0, nil)
	}

	bd.list.SetSelectedFunc(bd.handleItemSelection)
}

// handleItemSelection handles item selection toggle
func (bd *BatchExportDialog) handleItemSelection(index int, mainText, _ string, _ rune) {
	bd.selectedItems[index] = !bd.selectedItems[index]

	item := strings.TrimPrefix(mainText, "[ ] ")
	item = strings.TrimPrefix(item, "[✓] ")

	if bd.selectedItems[index] {
		bd.list.SetItemText(index, "[✓] "+item, "")
	} else {
		bd.list.SetItemText(index, "[ ] "+item, "")
	}
}

// setupExportForm initializes the export options form
func (bd *BatchExportDialog) setupExportForm() {
	bd.form = tview.NewForm()
	bd.form.SetBorder(true)
	bd.form.SetTitle("Export Options")

	// Format dropdown
	formats := []string{"Text", "JSON", "CSV", "Markdown", "HTML"}
	bd.form.AddDropDown("Format:", formats, 0, func(option string, _ int) {
		bd.selectedFormat = bd.parseFormatOption(option)
	})

	// Path input
	homeDir, _ := os.UserHomeDir()
	defaultPath := filepath.Join(homeDir, "slurm_exports", "batch")
	bd.form.AddInputField("Path:", defaultPath, 50, nil, func(text string) {
		bd.customPath = text
	})

	// Buttons
	bd.form.AddButton("Export Selected", bd.handleExportSelected)
	bd.form.AddButton("Select All", bd.handleSelectAll)
	bd.form.AddButton("Clear All", bd.handleClearAll)
	bd.form.AddButton("Cancel", func() {
		if bd.onCancel != nil {
			bd.onCancel()
		}
	})
}

// parseFormatOption converts format string to export format
func (bd *BatchExportDialog) parseFormatOption(option string) export.ExportFormat {
	switch option {
	case "Text":
		return export.FormatText
	case "JSON":
		return export.FormatJSON
	case "CSV":
		return export.FormatCSV
	case "Markdown":
		return export.FormatMarkdown
	case "HTML":
		return export.FormatHTML
	}
	return export.FormatText
}

// handleExportSelected collects selected items and triggers export
func (bd *BatchExportDialog) handleExportSelected() {
	if bd.onExport == nil {
		return
	}

	indices := []int{}
	for i, selected := range bd.selectedItems {
		if selected {
			indices = append(indices, i)
		}
	}
	bd.onExport(indices, bd.selectedFormat, bd.customPath)
}

// handleSelectAll selects all items in the list
func (bd *BatchExportDialog) handleSelectAll() {
	count := bd.list.GetItemCount()
	for i := 0; i < count; i++ {
		bd.updateItemSelection(i, true)
	}
}

// handleClearAll clears all selected items
func (bd *BatchExportDialog) handleClearAll() {
	count := bd.list.GetItemCount()
	for i := 0; i < count; i++ {
		bd.updateItemSelection(i, false)
	}
}

// updateItemSelection updates the selection state of an item
func (bd *BatchExportDialog) updateItemSelection(index int, selected bool) {
	bd.selectedItems[index] = selected
	mainText, _ := bd.list.GetItemText(index)
	item := strings.TrimPrefix(mainText, "[ ] ")
	item = strings.TrimPrefix(item, "[✓] ")

	if selected {
		bd.list.SetItemText(index, "[✓] "+item, "")
	} else {
		bd.list.SetItemText(index, "[ ] "+item, "")
	}
}

// SetBatchExportHandler sets the batch export callback
func (bd *BatchExportDialog) SetBatchExportHandler(handler func(indices []int, format export.ExportFormat, path string)) {
	bd.onExport = handler
}

// SetCancelHandler sets the cancel callback
func (bd *BatchExportDialog) SetCancelHandler(handler func()) {
	bd.onCancel = handler
}
