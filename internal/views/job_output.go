package views

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/jontk/s9s/internal/dao"
	"github.com/jontk/s9s/internal/export"
	"github.com/jontk/s9s/pkg/slurm"
	"github.com/rivo/tview"
)

// JobOutputView displays job output (stdout/stderr)
type JobOutputView struct {
	app      *tview.Application
	pages    *tview.Pages
	client   dao.SlurmClient
	modal    *tview.Flex
	textView *tview.TextView
	jobID    string
	jobName  string
	outputType string // "stdout" or "stderr"
	autoRefresh bool
	refreshTicker *time.Ticker
	exporter *export.JobOutputExporter
}

// NewJobOutputView creates a new job output view
func NewJobOutputView(client dao.SlurmClient, app *tview.Application) *JobOutputView {
	// Get default export path
	homeDir, _ := os.UserHomeDir()
	defaultPath := ""
	if homeDir != "" {
		defaultPath = homeDir + "/slurm_exports"
	}
	
	return &JobOutputView{
		client:   client,
		app:      app,
		exporter: export.NewJobOutputExporter(defaultPath),
	}
}

// SetPages sets the pages manager for modal display
func (v *JobOutputView) SetPages(pages *tview.Pages) {
	v.pages = pages
}

// ShowJobOutput displays job output in a modal
func (v *JobOutputView) ShowJobOutput(jobID, jobName, outputType string) {
	v.jobID = jobID
	v.jobName = jobName
	v.outputType = outputType
	v.autoRefresh = false

	v.buildUI()
	v.loadOutput()
	v.show()
}

// buildUI creates the job output viewer interface
func (v *JobOutputView) buildUI() {
	// Create text view for output
	v.textView = tview.NewTextView()
	v.textView.SetDynamicColors(true)
	v.textView.SetScrollable(true)
	v.textView.SetWrap(true)
	v.textView.SetBorder(true)
	v.textView.SetTitle(fmt.Sprintf(" Job %s - %s (%s) ", v.jobID, v.jobName, strings.ToUpper(v.outputType)))
	v.textView.SetTitleAlign(tview.AlignCenter)

	// Create help text
	helpText := tview.NewTextView()
	helpText.SetDynamicColors(true)
	helpText.SetText("[yellow]Keys:[white] r=Refresh a=Auto-refresh s=Switch stdout/stderr f=Follow e=Export Esc=Close")
	helpText.SetTextAlign(tview.AlignCenter)

	// Create modal container
	v.modal = tview.NewFlex()
	v.modal.SetDirection(tview.FlexRow)
	v.modal.AddItem(nil, 0, 1, false)
	v.modal.AddItem(tview.NewFlex().
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().
			SetDirection(tview.FlexRow).
			AddItem(v.textView, 0, 1, true).
			AddItem(helpText, 1, 0, false), 0, 4, true).
		AddItem(nil, 0, 1, false), 0, 3, true)
	v.modal.AddItem(nil, 0, 1, false)

	v.modal.SetBorder(true)
	v.modal.SetTitle(" Job Output Viewer ")
	v.modal.SetTitleAlign(tview.AlignCenter)

	// Setup event handlers
	v.setupEventHandlers()
}

// setupEventHandlers configures keyboard shortcuts
func (v *JobOutputView) setupEventHandlers() {
	v.modal.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEsc:
			v.close()
			return nil
		case tcell.KeyRune:
			switch event.Rune() {
			case 'r', 'R':
				v.loadOutput()
				return nil
			case 'a', 'A':
				v.toggleAutoRefresh()
				return nil
			case 's', 'S':
				v.switchOutputType()
				return nil
			case 'f', 'F':
				v.followOutput()
				return nil
			case 'e', 'E':
				v.exportOutput()
				return nil
			}
		}
		return event
	})
}

// loadOutput loads job output from SLURM
func (v *JobOutputView) loadOutput() {
	v.textView.SetText("[yellow]Loading output...[white]")

	go func() {
		var content string
		var err error

		// Try to get job output from client
		if jobMgr := v.client.Jobs(); jobMgr != nil {
			// For mock client, generate sample output
			if _, ok := v.client.(*slurm.MockClient); ok {
				content = v.generateMockOutput()
			} else {
				// For real client, try to get actual output
				content, err = v.getJobOutput()
				if err != nil {
					content = fmt.Sprintf("[red]Error loading output: %v[white]\n\nThis could mean:\n• Job is still running\n• Output files not accessible\n• Job completed without output\n• Insufficient permissions", err)
				}
			}
		} else {
			content = "[red]Job manager not available[white]"
		}

		// Update UI on main thread
		v.app.QueueUpdateDraw(func() {
			v.textView.SetText(content)
			v.textView.ScrollToEnd()
		})
	}()
}

// getJobOutput retrieves actual job output (placeholder for real implementation)
func (v *JobOutputView) getJobOutput() (string, error) {
	// This would be implemented to read actual SLURM output files
	// For now, return a placeholder
	return fmt.Sprintf("Output for job %s (%s) would be retrieved from SLURM output files.\n\nThis requires access to the job's stdout/stderr files on the cluster filesystem.", v.jobID, v.outputType), nil
}

// generateMockOutput creates sample output for demonstration
func (v *JobOutputView) generateMockOutput() string {
	if v.outputType == "stderr" {
		return fmt.Sprintf(`[red]STDERR for Job %s (%s)[white]

[yellow]Warning:[white] This is mock stderr output for demonstration
[red]Error:[white] Sample error message
[yellow]Debug:[white] Verbose debugging information

Job started at: %s
Working directory: /home/user/jobs
Environment variables loaded: 15

Some sample error output...
`, v.jobID, v.jobName, time.Now().Add(-10*time.Minute).Format("2006-01-02 15:04:05"))
	}

	return fmt.Sprintf(`[green]STDOUT for Job %s (%s)[white]

Job started at: %s
Loading modules...
  - module load gcc/11.2.0
  - module load openmpi/4.1.1

Setting up environment...
Working directory: /home/user/simulation
Input files: config.dat, mesh.inp

[yellow]Starting simulation...[white]
Iteration 1/1000: Energy = -1234.56
Iteration 100/1000: Energy = -1245.78
Iteration 200/1000: Energy = -1251.23
Iteration 300/1000: Energy = -1255.67
...
Iteration 1000/1000: Energy = -1289.45

[green]Simulation completed successfully![white]
Results written to: output/results.dat
Execution time: 15 minutes 23 seconds
Peak memory usage: 2.3 GB

Job completed at: %s
`, v.jobID, v.jobName, 
	time.Now().Add(-25*time.Minute).Format("2006-01-02 15:04:05"),
	time.Now().Add(-10*time.Minute).Format("2006-01-02 15:04:05"))
}

// toggleAutoRefresh toggles automatic refresh
func (v *JobOutputView) toggleAutoRefresh() {
	v.autoRefresh = !v.autoRefresh

	if v.autoRefresh {
		v.textView.SetTitle(fmt.Sprintf(" Job %s - %s (%s) [AUTO-REFRESH] ", v.jobID, v.jobName, strings.ToUpper(v.outputType)))
		v.startAutoRefresh()
	} else {
		v.textView.SetTitle(fmt.Sprintf(" Job %s - %s (%s) ", v.jobID, v.jobName, strings.ToUpper(v.outputType)))
		v.stopAutoRefresh()
	}
}

// startAutoRefresh starts automatic refresh timer
func (v *JobOutputView) startAutoRefresh() {
	if v.refreshTicker != nil {
		v.refreshTicker.Stop()
	}

	v.refreshTicker = time.NewTicker(3 * time.Second)
	go func() {
		for range v.refreshTicker.C {
			if v.autoRefresh {
				v.loadOutput()
			} else {
				break
			}
		}
	}()
}

// stopAutoRefresh stops automatic refresh
func (v *JobOutputView) stopAutoRefresh() {
	if v.refreshTicker != nil {
		v.refreshTicker.Stop()
		v.refreshTicker = nil
	}
}

// switchOutputType switches between stdout and stderr
func (v *JobOutputView) switchOutputType() {
	if v.outputType == "stdout" {
		v.outputType = "stderr"
	} else {
		v.outputType = "stdout"
	}

	v.textView.SetTitle(fmt.Sprintf(" Job %s - %s (%s) ", v.jobID, v.jobName, strings.ToUpper(v.outputType)))
	v.loadOutput()
}

// followOutput scrolls to end and enables auto-refresh
func (v *JobOutputView) followOutput() {
	v.textView.ScrollToEnd()
	if !v.autoRefresh {
		v.toggleAutoRefresh()
	}
}

// exportOutput exports the current output to a file
func (v *JobOutputView) exportOutput() {
	v.showExportDialog()
}

// showExportDialog shows the export options dialog
func (v *JobOutputView) showExportDialog() {
	// Create export format selection
	list := tview.NewList()
	list.SetBorder(true)
	list.SetTitle(" Select Export Format ")
	list.SetTitleAlign(tview.AlignCenter)

	formats := v.exporter.GetSupportedFormats()
	formatDescriptions := map[export.ExportFormat]string{
		export.FormatText:     "Plain text with header information",
		export.FormatJSON:     "Structured JSON with metadata",
		export.FormatCSV:      "CSV format (line-by-line for analysis)",
		export.FormatMarkdown: "Markdown format with code blocks",
	}

	for _, format := range formats {
		desc := formatDescriptions[format]
		list.AddItem(
			fmt.Sprintf("%s (.%s)", strings.ToUpper(string(format)), string(format)),
			desc,
			0,
			func() {
				v.performExport(format)
				v.pages.RemovePage("export-dialog")
			},
		)
	}

	// Add cancel option
	list.AddItem("Cancel", "Cancel export operation", 0, func() {
		v.pages.RemovePage("export-dialog")
	})

	// Create modal container
	modal := tview.NewFlex()
	modal.SetDirection(tview.FlexRow)
	modal.AddItem(nil, 0, 1, false)
	modal.AddItem(tview.NewFlex().
		AddItem(nil, 0, 1, false).
		AddItem(list, 0, 2, true).
		AddItem(nil, 0, 1, false), 0, 1, true)
	modal.AddItem(nil, 0, 1, false)

	v.pages.AddPage("export-dialog", modal, true, true)
}

// performExport performs the actual export operation
func (v *JobOutputView) performExport(format export.ExportFormat) {
	// Get current output content
	content := v.textView.GetText(false)
	if content == "" {
		v.showNotification("No content to export")
		return
	}

	// Create export data
	currentUser := os.Getenv("USER")
	if currentUser == "" {
		currentUser = "unknown"
	}

	exportData := export.JobOutputData{
		JobID:       v.jobID,
		JobName:     v.jobName,
		OutputType:  v.outputType,
		Content:     content,
		Timestamp:   time.Now(),
		ExportedBy:  currentUser,
		ExportTime:  time.Now(),
		ContentSize: len(content),
	}

	// Perform export
	result, err := v.exporter.ExportJobOutput(exportData, format, "")
	if err != nil {
		v.showNotification(fmt.Sprintf("Export failed: %v", err))
		return
	}

	// Show success notification
	message := fmt.Sprintf("Export successful!\n\nFile: %s\nFormat: %s\nSize: %s",
		result.FilePath,
		strings.ToUpper(string(result.Format)),
		v.formatBytes(result.Size))

	v.showExportResult(message, result.FilePath)
}

// showExportResult shows export success with options
func (v *JobOutputView) showExportResult(message, filePath string) {
	modal := tview.NewModal()
	modal.SetText(message)
	modal.AddButtons([]string{"Open Folder", "Copy Path", "OK"})
	modal.SetDoneFunc(func(buttonIndex int, buttonLabel string) {
		switch buttonIndex {
		case 0: // Open Folder
			v.openExportFolder(filePath)
		case 1: // Copy Path
			v.copyPathToClipboard(filePath)
		}
		v.pages.RemovePage("export-result")
	})
	
	v.pages.AddPage("export-result", modal, true, true)
}

// openExportFolder opens the folder containing the exported file
func (v *JobOutputView) openExportFolder(filePath string) {
	// This is a platform-specific operation
	// For now, just show the path
	v.showNotification(fmt.Sprintf("Export folder:\n%s", v.exporter.GetDefaultPath()))
}

// copyPathToClipboard copies the file path to clipboard
func (v *JobOutputView) copyPathToClipboard(filePath string) {
	// This would require a clipboard library
	// For now, just show the path
	v.showNotification(fmt.Sprintf("File path:\n%s\n\n(Copy this path manually)", filePath))
}

// formatBytes formats byte size in human-readable format
func (v *JobOutputView) formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// showNotification shows a temporary notification
func (v *JobOutputView) showNotification(message string) {
	notification := tview.NewModal()
	notification.SetText(message)
	notification.AddButtons([]string{"OK"})
	notification.SetDoneFunc(func(buttonIndex int, buttonLabel string) {
		v.pages.RemovePage("notification")
	})

	v.pages.AddPage("notification", notification, true, true)
}

// show displays the modal
func (v *JobOutputView) show() {
	if v.pages != nil {
		v.pages.AddPage("job-output", v.modal, true, true)
		v.app.SetFocus(v.textView)
	}
}

// close closes the output viewer
func (v *JobOutputView) close() {
	v.stopAutoRefresh()
	if v.pages != nil {
		v.pages.RemovePage("job-output")
	}
}