package views

import (
	"fmt"
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/jontk/s9s/internal/dao"
	"github.com/jontk/s9s/internal/streaming"
	"github.com/rivo/tview"
)

// StreamMonitorView displays multiple job streams simultaneously
type StreamMonitorView struct {
	app           *tview.Application
	pages         *tview.Pages
	client        dao.SlurmClient
	streamManager *streaming.StreamManager

	// UI components
	modal     *tview.Flex
	layout    *tview.Grid
	statusBar *tview.TextView
	helpBar   *tview.TextView

	// Stream panels
	activeStreams map[string]*StreamPanel
	maxStreams    int

	// Navigation
	selectedPanel int
	isVisible     bool
}

// StreamPanel represents a single streaming job panel
type StreamPanel struct {
	jobID      string
	outputType string
	jobName    string
	outputView *tview.TextView
	statusBar  *tview.TextView
	controls   *tview.Flex
	isActive   bool
	isSelected bool
	streamChan <-chan streaming.StreamEvent
	lastUpdate time.Time
}

// NewStreamMonitorView creates a new multi-stream monitor
func NewStreamMonitorView(client dao.SlurmClient, app *tview.Application, streamManager *streaming.StreamManager) *StreamMonitorView {
	return &StreamMonitorView{
		app:           app,
		client:        client,
		streamManager: streamManager,
		activeStreams: make(map[string]*StreamPanel),
		maxStreams:    4, // 2x2 grid
		selectedPanel: 0,
		isVisible:     false,
	}
}

// SetPages sets the pages manager for modal display
func (v *StreamMonitorView) SetPages(pages *tview.Pages) {
	v.pages = pages
}

// Show displays the stream monitor
func (v *StreamMonitorView) Show() {
	v.buildUI()
	v.show()
	v.isVisible = true
}

// buildUI creates the multi-stream monitor interface
func (v *StreamMonitorView) buildUI() {
	// Create 2x2 grid layout for streams
	v.layout = tview.NewGrid()
	v.layout.SetRows(0, 0)
	v.layout.SetColumns(0, 0)
	v.layout.SetBorder(true)
	v.layout.SetTitle(" Multi-Job Stream Monitor ")
	v.layout.SetTitleAlign(tview.AlignCenter)

	// Initialize with empty panels
	for i := 0; i < v.maxStreams; i++ {
		emptyPanel := v.createEmptyPanel(i)
		row := i / 2
		col := i % 2
		v.layout.AddItem(emptyPanel, row, col, 1, 1, 0, 0, false)
	}

	// Create status bar
	v.statusBar = tview.NewTextView()
	v.statusBar.SetDynamicColors(true)
	v.statusBar.SetText(v.getStatusText())
	v.statusBar.SetTextAlign(tview.AlignCenter)

	// Create help bar
	v.helpBar = tview.NewTextView()
	v.helpBar.SetDynamicColors(true)
	v.helpBar.SetText("[yellow]Keys:[white] a=Add Stream r=Remove d=Details Tab=Next Panel s=Stop All c=Clear All Esc=Close")
	v.helpBar.SetTextAlign(tview.AlignCenter)

	// Create main container
	v.modal = tview.NewFlex()
	v.modal.SetDirection(tview.FlexRow)
	v.modal.AddItem(nil, 0, 1, false)
	v.modal.AddItem(tview.NewFlex().
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().
			SetDirection(tview.FlexRow).
			AddItem(v.layout, 0, 1, true).
			AddItem(v.statusBar, 1, 0, false).
			AddItem(v.helpBar, 1, 0, false), 0, 8, true).
		AddItem(nil, 0, 1, false), 0, 6, true)
	v.modal.AddItem(nil, 0, 1, false)

	// Setup event handlers
	v.setupEventHandlers()
}

// createEmptyPanel creates an empty panel placeholder
func (v *StreamMonitorView) createEmptyPanel(index int) *tview.TextView {
	emptyPanel := tview.NewTextView()
	emptyPanel.SetBorder(true)
	emptyPanel.SetTitle(fmt.Sprintf(" Panel %d - Empty ", index+1))
	emptyPanel.SetTitleAlign(tview.AlignCenter)
	emptyPanel.SetText("[gray]No active stream\n\nPress 'a' to add a stream[white]")
	emptyPanel.SetTextAlign(tview.AlignCenter)

	if index == v.selectedPanel {
		emptyPanel.SetBorderColor(tcell.ColorYellow)
		emptyPanel.SetTitleColor(tcell.ColorYellow)
	}

	return emptyPanel
}

// setupEventHandlers configures keyboard shortcuts
func (v *StreamMonitorView) setupEventHandlers() {
	v.modal.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEsc:
			v.close()
			return nil
		case tcell.KeyTab:
			v.nextPanel()
			return nil
		case tcell.KeyRune:
			switch event.Rune() {
			case 'a', 'A':
				v.showAddStreamDialog()
				return nil
			case 'r', 'R':
				v.removeSelectedStream()
				return nil
			case 'd', 'D':
				v.showStreamDetails()
				return nil
			case 's', 'S':
				v.stopAllStreams()
				return nil
			case 'c', 'C':
				v.clearAllStreams()
				return nil
			}
		}
		return event
	})
}

// AddStream adds a stream to the monitor
func (v *StreamMonitorView) AddStream(jobID, jobName, outputType string) error {
	if len(v.activeStreams) >= v.maxStreams {
		return fmt.Errorf("maximum streams (%d) already active", v.maxStreams)
	}

	streamKey := v.makeStreamKey(jobID, outputType)
	if _, exists := v.activeStreams[streamKey]; exists {
		return fmt.Errorf("stream already active for job %s %s", jobID, outputType)
	}

	// Start the stream
	err := v.streamManager.StartStream(jobID, outputType)
	if err != nil {
		return fmt.Errorf("failed to start stream: %w", err)
	}

	// Create stream panel
	panel := &StreamPanel{
		jobID:      jobID,
		outputType: outputType,
		jobName:    jobName,
		isActive:   true,
		lastUpdate: time.Now(),
	}

	// Create UI components for the panel
	v.createStreamPanelUI(panel)

	// Subscribe to events
	panel.streamChan = v.streamManager.Subscribe(jobID, outputType)

	// Store the panel
	v.activeStreams[streamKey] = panel

	// Update layout
	v.updateLayout()

	// Start event processing
	go v.processStreamEvents(panel)

	return nil
}

// createStreamPanelUI creates UI components for a stream panel
func (v *StreamMonitorView) createStreamPanelUI(panel *StreamPanel) {
	// Create output view
	panel.outputView = tview.NewTextView()
	panel.outputView.SetDynamicColors(true)
	panel.outputView.SetScrollable(true)
	panel.outputView.SetWrap(true)
	panel.outputView.SetBorder(true)
	panel.outputView.SetTitle(fmt.Sprintf(" %s (%s) - %s ", panel.jobID, panel.jobName, strings.ToUpper(panel.outputType)))
	panel.outputView.SetTitleAlign(tview.AlignCenter)

	// Create status bar for the panel
	panel.statusBar = tview.NewTextView()
	panel.statusBar.SetDynamicColors(true)
	panel.statusBar.SetText("[green]● ACTIVE[white] | Lines: 0 | Last update: " + panel.lastUpdate.Format("15:04:05"))

	// Create controls
	panel.controls = tview.NewFlex()
	panel.controls.AddItem(panel.outputView, 0, 1, false)
}

// updateLayout updates the grid layout with active streams
func (v *StreamMonitorView) updateLayout() {
	// Clear current layout
	v.layout.Clear()

	streamIndex := 0
	for _, panel := range v.activeStreams {
		if streamIndex >= v.maxStreams {
			break
		}

		row := streamIndex / 2
		col := streamIndex % 2

		// Create panel container
		panelContainer := tview.NewFlex()
		panelContainer.SetDirection(tview.FlexRow)
		panelContainer.AddItem(panel.outputView, 0, 1, false)
		panelContainer.AddItem(panel.statusBar, 1, 0, false)

		// Highlight selected panel
		if streamIndex == v.selectedPanel {
			panel.outputView.SetBorderColor(tcell.ColorYellow)
			panel.outputView.SetTitleColor(tcell.ColorYellow)
			panel.isSelected = true
		} else {
			panel.outputView.SetBorderColor(tcell.ColorWhite)
			panel.outputView.SetTitleColor(tcell.ColorWhite)
			panel.isSelected = false
		}

		v.layout.AddItem(panelContainer, row, col, 1, 1, 0, 0, false)
		streamIndex++
	}

	// Fill remaining slots with empty panels
	for i := streamIndex; i < v.maxStreams; i++ {
		emptyPanel := v.createEmptyPanel(i)
		row := i / 2
		col := i % 2
		v.layout.AddItem(emptyPanel, row, col, 1, 1, 0, 0, false)
	}

	// Update status
	v.updateStatus()
}

// processStreamEvents processes events for a single stream panel
func (v *StreamMonitorView) processStreamEvents(panel *StreamPanel) {
	for panel.isActive && panel.streamChan != nil {
		select {
		case event, ok := <-panel.streamChan:
			if !ok {
				// Channel closed
				panel.isActive = false
				return
			}

			v.handlePanelStreamEvent(panel, &event)

		case <-time.After(5 * time.Second):
			// Timeout - update status
			v.app.QueueUpdateDraw(func() {
				v.updatePanelStatus(panel)
			})
		}
	}
}

// handlePanelStreamEvent handles a stream event for a panel
func (v *StreamMonitorView) handlePanelStreamEvent(panel *StreamPanel, event *streaming.StreamEvent) {
	v.app.QueueUpdateDraw(func() {
		switch event.EventType {
		case streaming.StreamEventNewOutput:
			// Append new content (keep last 100 lines to prevent memory issues)
			currentText := panel.outputView.GetText(false)
			lines := strings.Split(currentText, "\n")
			if len(lines) > 100 {
				lines = lines[len(lines)-100:]
				currentText = strings.Join(lines, "\n")
			}

			newText := currentText + event.Content
			panel.outputView.SetText(newText)
			panel.outputView.ScrollToEnd()
			panel.lastUpdate = time.Now()

		case streaming.StreamEventError:
			panel.outputView.SetTitle(fmt.Sprintf(" %s (%s) - %s [ERROR] ", panel.jobID, panel.jobName, strings.ToUpper(panel.outputType)))

		case streaming.StreamEventJobComplete:
			panel.outputView.SetTitle(fmt.Sprintf(" %s (%s) - %s [COMPLETE] ", panel.jobID, panel.jobName, strings.ToUpper(panel.outputType)))
			panel.isActive = false

		case streaming.StreamEventStreamStop:
			panel.outputView.SetTitle(fmt.Sprintf(" %s (%s) - %s [STOPPED] ", panel.jobID, panel.jobName, strings.ToUpper(panel.outputType)))
			panel.isActive = false
		}

		v.updatePanelStatus(panel)
		v.updateStatus()
	})
}

// updatePanelStatus updates the status bar for a panel
func (v *StreamMonitorView) updatePanelStatus(panel *StreamPanel) {
	lineCount := len(strings.Split(panel.outputView.GetText(false), "\n"))
	status := "[green]● ACTIVE[white]"
	if !panel.isActive {
		status = "[gray]● STOPPED[white]"
	}

	statusText := fmt.Sprintf("%s | Lines: %d | Last: %s",
		status, lineCount, panel.lastUpdate.Format("15:04:05"))

	panel.statusBar.SetText(statusText)
}

// nextPanel moves selection to the next panel
func (v *StreamMonitorView) nextPanel() {
	v.selectedPanel = (v.selectedPanel + 1) % v.maxStreams
	v.updateLayout()
}

// showAddStreamDialog shows dialog to add a new stream
func (v *StreamMonitorView) showAddStreamDialog() {
	// Create input form
	form := tview.NewForm()
	form.SetBorder(true)
	form.SetTitle(" Add Stream ")
	form.SetTitleAlign(tview.AlignCenter)

	form.AddInputField("Job ID", "", 20, nil, nil)
	form.AddInputField("Job Name", "", 30, nil, nil)
	form.AddDropDown("Output Type", []string{"stdout", "stderr"}, 0, nil)

	form.AddButton("Add", func() {
		jobID := form.GetFormItemByLabel("Job ID").(*tview.InputField).GetText()
		jobName := form.GetFormItemByLabel("Job Name").(*tview.InputField).GetText()
		_, outputType := form.GetFormItemByLabel("Output Type").(*tview.DropDown).GetCurrentOption()

		if jobID == "" {
			v.showNotification("Job ID is required")
			return
		}

		if jobName == "" {
			jobName = fmt.Sprintf("Job_%s", jobID)
		}

		err := v.AddStream(jobID, jobName, outputType)
		if err != nil {
			v.showNotification(fmt.Sprintf("Failed to add stream: %v", err))
		} else {
			v.pages.RemovePage("add-stream")
		}
	})

	form.AddButton("Cancel", func() {
		v.pages.RemovePage("add-stream")
	})

	// Create modal
	modal := tview.NewFlex()
	modal.SetDirection(tview.FlexRow)
	modal.AddItem(nil, 0, 1, false)
	modal.AddItem(tview.NewFlex().
		AddItem(nil, 0, 1, false).
		AddItem(form, 0, 2, true).
		AddItem(nil, 0, 1, false), 0, 1, true)
	modal.AddItem(nil, 0, 1, false)

	v.pages.AddPage("add-stream", modal, true, true)
}

// removeSelectedStream removes the currently selected stream
func (v *StreamMonitorView) removeSelectedStream() {
	if len(v.activeStreams) == 0 {
		return
	}

	// Find the stream at the selected panel index
	streamIndex := 0
	for streamKey, panel := range v.activeStreams {
		if streamIndex == v.selectedPanel {
			v.removeStream(streamKey, panel)
			break
		}
		streamIndex++
	}
}

// removeStream removes a specific stream
func (v *StreamMonitorView) removeStream(streamKey string, panel *StreamPanel) {
	// Stop the stream
	panel.isActive = false
	_ = v.streamManager.StopStream(panel.jobID, panel.outputType)

	// Unsubscribe
	if panel.streamChan != nil {
		v.streamManager.Unsubscribe(panel.jobID, panel.outputType, panel.streamChan)
	}

	// Remove from active streams
	delete(v.activeStreams, streamKey)

	// Update layout
	v.updateLayout()
}

// stopAllStreams stops all active streams
func (v *StreamMonitorView) stopAllStreams() {
	for _, panel := range v.activeStreams {
		panel.isActive = false
		_ = v.streamManager.StopStream(panel.jobID, panel.outputType)
	}
	v.updateStatus()
}

// clearAllStreams removes all streams
func (v *StreamMonitorView) clearAllStreams() {
	for streamKey, panel := range v.activeStreams {
		v.removeStream(streamKey, panel)
	}
}

// showStreamDetails shows detailed information about the selected stream
func (v *StreamMonitorView) showStreamDetails() {
	if len(v.activeStreams) == 0 {
		v.showNotification("No active streams")
		return
	}

	// Find selected stream
	streamIndex := 0
	var selectedPanel *StreamPanel
	for _, panel := range v.activeStreams {
		if streamIndex == v.selectedPanel {
			selectedPanel = panel
			break
		}
		streamIndex++
	}

	if selectedPanel == nil {
		return
	}

	// Create details modal
	details := fmt.Sprintf(`Stream Details:

Job ID: %s
Job Name: %s
Output Type: %s
Status: %s
Lines: %d
Last Update: %s

Buffer Information:
- Content available in stream manager
- Real-time updates active: %t`,
		selectedPanel.jobID,
		selectedPanel.jobName,
		selectedPanel.outputType,
		func() string {
			if selectedPanel.isActive {
				return "ACTIVE"
			}
			return "STOPPED"
		}(),
		len(strings.Split(selectedPanel.outputView.GetText(false), "\n")),
		selectedPanel.lastUpdate.Format("2006-01-02 15:04:05"),
		selectedPanel.isActive)

	v.showNotification(details)
}

// updateStatus updates the main status bar
func (v *StreamMonitorView) updateStatus() {
	if v.statusBar != nil {
		v.statusBar.SetText(v.getStatusText())
	}
}

// getStatusText returns the current status text
func (v *StreamMonitorView) getStatusText() string {
	activeCount := 0
	for _, panel := range v.activeStreams {
		if panel.isActive {
			activeCount++
		}
	}

	return fmt.Sprintf("[green]Active Streams:[white] %d/%d | [yellow]Selected Panel:[white] %d | [blue]Press 'a' to add stream[white]",
		activeCount, v.maxStreams, v.selectedPanel+1)
}

// showNotification shows a notification modal
func (v *StreamMonitorView) showNotification(message string) {
	notification := tview.NewModal()
	notification.SetText(message)
	notification.AddButtons([]string{"OK"})
	notification.SetDoneFunc(func(_ int, _ string) {
		v.pages.RemovePage("notification")
	})

	v.pages.AddPage("notification", notification, true, true)
}

// makeStreamKey creates a unique key for a stream
func (v *StreamMonitorView) makeStreamKey(jobID, outputType string) string {
	return jobID + ":" + outputType
}

// show displays the stream monitor
func (v *StreamMonitorView) show() {
	if v.pages != nil {
		v.pages.AddPage("stream-monitor", v.modal, true, true)
		v.app.SetFocus(v.layout)
	}
}

// close closes the stream monitor
func (v *StreamMonitorView) close() {
	v.isVisible = false
	v.clearAllStreams()
	if v.pages != nil {
		v.pages.RemovePage("stream-monitor")
	}
}

// IsVisible returns true if the monitor is currently visible
func (v *StreamMonitorView) IsVisible() bool {
	return v.isVisible
}

// GetActiveStreamCount returns the number of active streams
func (v *StreamMonitorView) GetActiveStreamCount() int {
	activeCount := 0
	for _, panel := range v.activeStreams {
		if panel.isActive {
			activeCount++
		}
	}
	return activeCount
}
