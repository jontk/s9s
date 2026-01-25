package views

import (
	"context"
	"fmt"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/jontk/s9s/internal/dao"
	"github.com/jontk/s9s/internal/export"
	"github.com/jontk/s9s/internal/streaming"
	"github.com/jontk/s9s/internal/ui/widgets"
	"github.com/rivo/tview"
)

// FilteredJobOutputView displays job output with filtering and search capabilities
type FilteredJobOutputView struct {
	app       *tview.Application
	pages     *tview.Pages
	client    dao.SlurmClient
	modal     *tview.Flex
	container *tview.Flex
	textView  *tview.TextView
	statusBar *tview.TextView

	// Filter and search components
	filterControls *widgets.FilterControls
	searchBar      *widgets.SearchBar
	filterManager  *streaming.FilteredStreamManager

	// View state
	jobID       string
	jobName     string
	outputType  string
	isStreaming bool
	autoScroll  bool
	showFilters bool
	showSearch  bool

	// Stream context
	streamCtx     context.Context
	streamCancel  context.CancelFunc
	streamChannel <-chan *streaming.StreamEvent

	// Export
	exporter *export.JobOutputExporter
}

// NewFilteredJobOutputView creates a new filtered job output view
func NewFilteredJobOutputView(client dao.SlurmClient, app *tview.Application, configPath string) (*FilteredJobOutputView, error) {
	// Create filtered stream manager
	filterManager, err := streaming.NewFilteredStreamManager(
		client,
		nil, // SSH manager would be provided here
		streaming.DefaultSlurmConfig(),
		configPath,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create filter manager: %w", err)
	}

	v := &FilteredJobOutputView{
		client:        client,
		app:           app,
		filterManager: filterManager,
		exporter:      export.NewJobOutputExporter(configPath),
		autoScroll:    true,
		showFilters:   false,
		showSearch:    false,
	}

	// Create UI components
	v.filterControls = widgets.NewFilterControls(filterManager)
	v.searchBar = widgets.NewSearchBar(filterManager)

	// Set callbacks
	v.filterControls.SetOnFilterChange(v.onFilterChange)
	v.searchBar.SetOnSearchResult(v.onSearchResult)
	v.searchBar.SetOnHighlight(v.highlightLine)

	return v, nil
}

// SetPages sets the pages manager for modal display
func (v *FilteredJobOutputView) SetPages(pages *tview.Pages) {
	v.pages = pages
}

// ShowJobOutput displays job output with filtering support
func (v *FilteredJobOutputView) ShowJobOutput(jobID, jobName, outputType string) {
	v.jobID = jobID
	v.jobName = jobName
	v.outputType = outputType

	v.buildUI()
	v.startStream()
	v.show()
}

// buildUI creates the filtered job output interface
func (v *FilteredJobOutputView) buildUI() {
	// Create main container
	v.container = tview.NewFlex().SetDirection(tview.FlexRow)

	// Create text view for output
	v.textView = tview.NewTextView()
	v.textView.SetDynamicColors(true)
	v.textView.SetScrollable(true)
	v.textView.SetWrap(true)
	v.textView.SetBorder(true)
	v.textView.SetTitle(fmt.Sprintf(" Job %s - %s (%s) ", v.jobID, v.jobName, strings.ToUpper(v.outputType)))
	v.textView.SetTitleAlign(tview.AlignCenter)

	// Create status bar
	v.statusBar = tview.NewTextView()
	v.statusBar.SetDynamicColors(true)
	v.statusBar.SetTextAlign(tview.AlignCenter)
	v.updateStatusBar()

	// Create controls bar
	_ = v.createControlsBar()

	// Build initial layout
	v.updateLayout()

	// Create modal wrapper
	v.modal = tview.NewFlex().SetDirection(tview.FlexRow)
	v.modal.AddItem(nil, 0, 1, false)
	v.modal.AddItem(tview.NewFlex().SetDirection(tview.FlexColumn).
		AddItem(nil, 0, 1, false).
		AddItem(v.container, 0, 4, true).
		AddItem(nil, 0, 1, false), 0, 3, true)
	v.modal.AddItem(nil, 0, 1, false)

	// Set up input handling
	v.modal.SetInputCapture(v.handleInput)
}

// createControlsBar creates the controls bar
func (v *FilteredJobOutputView) createControlsBar() *tview.Flex {
	controls := tview.NewFlex().SetDirection(tview.FlexColumn)

	// Status text
	statusText := tview.NewTextView()
	statusText.SetDynamicColors(true)
	statusText.SetText("[green]●[white] Streaming")

	// Control hints
	hints := tview.NewTextView()
	hints.SetDynamicColors(true)
	hints.SetTextAlign(tview.AlignRight)
	hints.SetText("F: Filters | S: Search | A: Auto-scroll | R: Refresh | E: Export | ESC: Close")

	controls.AddItem(statusText, 20, 0, false)
	controls.AddItem(hints, 0, 1, false)

	return controls
}

// updateLayout updates the UI layout based on visible components
func (v *FilteredJobOutputView) updateLayout() {
	v.container.Clear()

	// Add filter controls if visible
	if v.showFilters {
		v.container.AddItem(v.filterControls.GetContainer(), 12, 0, false)
	}

	// Add search bar if visible
	if v.showSearch {
		v.container.AddItem(v.searchBar.GetContainer(), 8, 0, false)
	}

	// Add main text view
	v.container.AddItem(v.textView, 0, 1, true)

	// Add controls and status bar
	controlsBar := v.createControlsBar()
	v.container.AddItem(controlsBar, 1, 0, false)
	v.container.AddItem(v.statusBar, 1, 0, false)
}

// handleInput processes keyboard input
func (v *FilteredJobOutputView) handleInput(event *tcell.EventKey) *tcell.EventKey {
	// Pass to search bar if active
	if v.showSearch && v.searchBar.IsActive() {
		if result := v.searchBar.HandleInput(event); result == nil {
			return nil
		}
	}

	// Pass to filter controls if active
	if v.showFilters {
		if result := v.filterControls.HandleInput(event); result == nil {
			return nil
		}
	}

	if handler, ok := v.keyHandlers()[event.Key()]; ok {
		handler()
		return nil
	}

	if handler, ok := v.runeHandlers()[event.Rune()]; ok {
		handler()
		return nil
	}

	return event
}

// keyHandlers returns a map of function key handlers
func (v *FilteredJobOutputView) keyHandlers() map[tcell.Key]func() {
	return map[tcell.Key]func(){
		tcell.KeyEsc: v.close,
		tcell.KeyF5:  v.refresh,
	}
}

// runeHandlers returns a map of rune handlers
func (v *FilteredJobOutputView) runeHandlers() map[rune]func() {
	return map[rune]func(){
		'f': v.toggleFilters,
		'F': v.toggleFilters,
		's': v.toggleSearch,
		'S': v.toggleSearch,
		'a': v.toggleAutoScroll,
		'A': v.toggleAutoScroll,
		'r': v.refresh,
		'R': v.refresh,
		'e': v.exportOutput,
		'E': v.exportOutput,
		'c': v.clearFilters,
		'C': v.clearFilters,
		't': v.switchOutputType,
		'T': v.switchOutputType,
	}
}

// startStream starts the filtered output stream
func (v *FilteredJobOutputView) startStream() {
	// Stop any existing stream
	v.stopStream()

	// Clear text view
	v.textView.Clear()
	v.textView.SetText("[yellow]Starting stream...[white]")

	// Create stream context
	v.streamCtx, v.streamCancel = context.WithCancel(context.Background())

	// Start filtered stream
	err := v.filterManager.StartFilteredStream(v.jobID, v.outputType)
	if err != nil {
		v.textView.SetText(fmt.Sprintf("[red]Error starting stream: %v[white]", err))
		return
	}

	// Get stream channel
	streamChan, err := v.filterManager.StreamWithContext(v.streamCtx, v.jobID, v.outputType)
	if err != nil {
		v.textView.SetText(fmt.Sprintf("[red]Error subscribing to stream: %v[white]", err))
		return
	}

	v.streamChannel = streamChan
	v.isStreaming = true

	// Set stream for search bar
	v.searchBar.SetStream(v.jobID, v.outputType)

	// Start processing stream events
	go v.processStreamEvents()

	v.updateStatusBar()
}

// stopStream stops the output stream
func (v *FilteredJobOutputView) stopStream() {
	if v.streamCancel != nil {
		v.streamCancel()
		v.streamCancel = nil
	}

	if v.isStreaming {
		_ = v.filterManager.StopFilteredStream(v.jobID, v.outputType)
		v.isStreaming = false
	}

	v.updateStatusBar()
}

// processStreamEvents processes incoming stream events
func (v *FilteredJobOutputView) processStreamEvents() {
	for {
		select {
		case <-v.streamCtx.Done():
			return
		case event, ok := <-v.streamChannel:
			if !ok {
				v.app.QueueUpdateDraw(func() {
					v.isStreaming = false
					v.updateStatusBar()
				})
				return
			}

			v.handleStreamEvent(event)
		}
	}
}

// handleStreamEvent handles a single stream event
func (v *FilteredJobOutputView) handleStreamEvent(event *streaming.StreamEvent) {
	v.app.QueueUpdateDraw(func() {
		switch event.EventType {
		case streaming.StreamEventNewOutput:
			// Add new lines to text view
			for _, line := range event.NewLines {
				// Apply search highlighting if active
				if v.searchBar.IsActive() {
					// Simple highlighting - would be enhanced in real implementation
					line = "[cyan]" + line + "[white]"
				}

				_, _ = v.textView.Write([]byte(line + "\n"))
			}

			// Auto-scroll if enabled
			if v.autoScroll {
				v.textView.ScrollToEnd()
			}

		case streaming.StreamEventError:
			_, _ = fmt.Fprintf(v.textView, "\n[red]Error: %v[white]\n", event.Error)

		case streaming.StreamEventJobComplete:
			_, _ = v.textView.Write([]byte("\n[green]Job completed[white]\n"))
			v.isStreaming = false

		case streaming.StreamEventStreamStop:
			v.isStreaming = false
		}

		v.updateStatusBar()
	})
}

// onFilterChange handles filter changes
func (v *FilteredJobOutputView) onFilterChange() {
	// Refresh the view to show filtered content
	v.refresh()
}

// onSearchResult handles search results
func (v *FilteredJobOutputView) onSearchResult(result *streaming.SearchResult) {
	// Scroll to the result line
	v.scrollToLine(result.LineNumber)
}

// highlightLine highlights a specific line
func (v *FilteredJobOutputView) highlightLine(_ int) {
	// This would require tracking line positions in the text view
	// For now, just ensure we're not auto-scrolling
	v.autoScroll = false
	v.updateStatusBar()
}

// scrollToLine scrolls to a specific line number
func (v *FilteredJobOutputView) scrollToLine(lineNumber int) {
	// This is a simplified implementation
	// A full implementation would need to track line positions
	_, _ = v.textView.GetScrollOffset()
	v.textView.ScrollTo(lineNumber-1, 0)

	// Flash the line by temporarily changing colors
	// This would require more sophisticated text manipulation
}

// toggleFilters shows/hides filter controls
func (v *FilteredJobOutputView) toggleFilters() {
	v.showFilters = !v.showFilters
	v.updateLayout()

	if v.showFilters {
		v.filterControls.Focus()
	}
}

// toggleSearch shows/hides search bar
func (v *FilteredJobOutputView) toggleSearch() {
	v.showSearch = !v.showSearch
	v.updateLayout()

	if v.showSearch {
		// Focus search input
		v.app.SetFocus(v.searchBar.GetContainer())
	}
}

// toggleAutoScroll toggles auto-scrolling
func (v *FilteredJobOutputView) toggleAutoScroll() {
	v.autoScroll = !v.autoScroll
	v.updateStatusBar()

	if v.autoScroll {
		v.textView.ScrollToEnd()
	}
}

// clearFilters removes all active filters
func (v *FilteredJobOutputView) clearFilters() {
	v.filterManager.ClearFilters()
	v.refresh()
}

// switchOutputType switches between stdout and stderr
func (v *FilteredJobOutputView) switchOutputType() {
	if v.outputType == "stdout" {
		v.outputType = "stderr"
	} else {
		v.outputType = "stdout"
	}

	v.textView.SetTitle(fmt.Sprintf(" Job %s - %s (%s) ", v.jobID, v.jobName, strings.ToUpper(v.outputType)))
	v.startStream()
}

// refresh refreshes the output
func (v *FilteredJobOutputView) refresh() {
	// Get filtered content
	lines, err := v.filterManager.GetFilteredContent(v.jobID, v.outputType, true)
	if err != nil {
		v.textView.SetText(fmt.Sprintf("[red]Error getting filtered content: %v[white]", err))
		return
	}

	// Update text view
	v.textView.Clear()
	for _, line := range lines {
		_, _ = v.textView.Write([]byte(line + "\n"))
	}

	if v.autoScroll {
		v.textView.ScrollToEnd()
	}

	// Update filter stats
	if v.showFilters {
		v.filterControls.Refresh()
	}
}

// exportOutput shows the export dialog
func (v *FilteredJobOutputView) exportOutput() {
	// Get filtered content
	lines, err := v.filterManager.GetFilteredContent(v.jobID, v.outputType, false)
	if err != nil {
		v.statusBar.SetText(fmt.Sprintf("[red]Export error: %v[white]", err))
		return
	}

	content := strings.Join(lines, "\n")

	// Create export dialog
	exportDialog := widgets.NewJobOutputExportDialog(v.jobID, v.jobName, v.outputType, content)

	// Set export handler
	exportDialog.SetExportHandler(func(_ export.ExportFormat, _ string) {
		// Close dialog
		if v.pages != nil {
			v.pages.RemovePage("export-dialog")
		}

		// Update status bar to show successful export
		v.app.QueueUpdateDraw(func() {
			v.statusBar.SetText("[green]Export completed successfully[white]")
		})
	})

	// Set cancel handler
	exportDialog.SetCancelHandler(func() {
		if v.pages != nil {
			v.pages.RemovePage("export-dialog")
		}
	})

	// Show dialog
	if v.pages != nil {
		v.pages.AddPage("export-dialog", exportDialog, true, true)
	}
}

// updateStatusBar updates the status bar
func (v *FilteredJobOutputView) updateStatusBar() {
	status := []string{}

	// Streaming status
	if v.isStreaming {
		status = append(status, "[green]●[white] Streaming")
	} else {
		status = append(status, "[gray]●[white] Not streaming")
	}

	// Auto-scroll status
	if v.autoScroll {
		status = append(status, "Auto-scroll: [green]ON[white]")
	} else {
		status = append(status, "Auto-scroll: [red]OFF[white]")
	}

	// Filter status
	activeFilters := v.filterManager.GetActiveFilters()
	if len(activeFilters) > 0 {
		status = append(status, fmt.Sprintf("Filters: [yellow]%d active[white]", len(activeFilters)))
	}

	// Search status
	if v.searchBar.IsActive() {
		current, total := v.searchBar.GetCurrentMatch()
		status = append(status, fmt.Sprintf("Search: [cyan]%d/%d[white]", current, total))
	}

	v.statusBar.SetText(strings.Join(status, " | "))
}

// show displays the modal
func (v *FilteredJobOutputView) show() {
	if v.pages != nil {
		v.pages.AddPage("job-output", v.modal, true, true)
	}
}

// close closes the view
func (v *FilteredJobOutputView) close() {
	v.stopStream()

	if v.pages != nil {
		v.pages.RemovePage("job-output")
	}
}

// Close cleans up resources
func (v *FilteredJobOutputView) Close() error {
	v.stopStream()
	return v.filterManager.CloseFiltered()
}
