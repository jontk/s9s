package views

import (
	"fmt"
	"strconv"

	"github.com/gdamore/tcell/v2"
	"github.com/jontk/s9s/internal/preferences"
	"github.com/rivo/tview"
)

// StreamingPreferencesView displays and manages streaming preferences
type StreamingPreferencesView struct {
	app          *tview.Application
	pages        *tview.Pages
	modal        *tview.Flex
	form         *tview.Form
	prefsManager *preferences.StreamingPreferencesManager
	onSave       func()
}

// NewStreamingPreferencesView creates a new streaming preferences view
func NewStreamingPreferencesView(app *tview.Application, prefsManager *preferences.StreamingPreferencesManager) *StreamingPreferencesView {
	return &StreamingPreferencesView{
		app:          app,
		prefsManager: prefsManager,
	}
}

// SetPages sets the pages manager for modal display
func (v *StreamingPreferencesView) SetPages(pages *tview.Pages) {
	v.pages = pages
}

// SetOnSave sets the callback for when preferences are saved
func (v *StreamingPreferencesView) SetOnSave(callback func()) {
	v.onSave = callback
}

// Show displays the preferences dialog
func (v *StreamingPreferencesView) Show() {
	v.buildUI()
	v.show()
}

// buildUI creates the preferences interface
func (v *StreamingPreferencesView) buildUI() {
	// Get current preferences
	prefs := v.prefsManager.GetPreferences()

	// Create form
	v.form = tview.NewForm()
	v.form.SetBorder(true)
	v.form.SetTitle(" Streaming Preferences ")
	v.form.SetTitleAlign(tview.AlignCenter)

	// General Settings Section
	v.form.AddTextView("General Settings", "", 0, 1, true, false)

	v.form.AddCheckbox("Auto-start streaming for running jobs", prefs.AutoStartForRunningJobs, nil)
	v.form.AddCheckbox("Default auto-scroll", prefs.DefaultAutoScroll, nil)
	v.form.AddCheckbox("Show timestamps", prefs.ShowTimestamps, nil)

	exportFormats := []string{"txt", "json", "csv", "md"}
	defaultIndex := 0
	for i, format := range exportFormats {
		if format == prefs.ExportFormat {
			defaultIndex = i
			break
		}
	}
	v.form.AddDropDown("Export format", exportFormats, defaultIndex, nil)

	// Performance Settings Section
	v.form.AddTextView("", "", 0, 1, true, false) // Spacer
	v.form.AddTextView("Performance Settings", "", 0, 1, true, false)

	v.form.AddInputField("Max concurrent streams", strconv.Itoa(prefs.MaxConcurrentStreams), 20, nil, nil)
	v.form.AddInputField("Buffer size (lines)", strconv.Itoa(prefs.BufferSizeLines), 20, nil, nil)
	v.form.AddInputField("Poll interval (seconds)", strconv.Itoa(prefs.PollIntervalSeconds), 20, nil, nil)
	v.form.AddInputField("Max memory (MB)", strconv.Itoa(prefs.MaxMemoryMB), 20, nil, nil)
	v.form.AddInputField("File check interval (ms)", strconv.Itoa(prefs.FileCheckIntervalMs), 20, nil, nil)

	// Remote Streaming Settings Section
	v.form.AddTextView("", "", 0, 1, true, false) // Spacer
	v.form.AddTextView("Remote Streaming Settings", "", 0, 1, true, false)

	v.form.AddCheckbox("Enable remote streaming", prefs.EnableRemoteStreaming, nil)
	v.form.AddInputField("SSH timeout (seconds)", strconv.Itoa(prefs.SSHTimeout), 20, nil, nil)
	v.form.AddInputField("Remote buffer size", strconv.Itoa(prefs.RemoteBufferSize), 20, nil, nil)

	// Display Settings Section
	v.form.AddTextView("", "", 0, 1, true, false) // Spacer
	v.form.AddTextView("Display Settings", "", 0, 1, true, false)

	gridSizes := []string{"2x2", "3x3", "2x3", "3x2", "4x4"}
	gridIndex := 0
	for i, size := range gridSizes {
		if size == prefs.MultiStreamGridSize {
			gridIndex = i
			break
		}
	}
	v.form.AddDropDown("Multi-stream grid size", gridSizes, gridIndex, nil)

	v.form.AddInputField("Stream panel height", strconv.Itoa(prefs.StreamPanelHeight), 20, nil, nil)
	v.form.AddCheckbox("Show buffer statistics", prefs.ShowBufferStats, nil)

	// Advanced Settings Section
	v.form.AddTextView("", "", 0, 1, true, false) // Spacer
	v.form.AddTextView("Advanced Settings", "", 0, 1, true, false)

	v.form.AddCheckbox("Enable compression", prefs.EnableCompression, nil)
	v.form.AddInputField("Stream history days", strconv.Itoa(prefs.StreamHistoryDays), 20, nil, nil)
	v.form.AddCheckbox("Auto-cleanup inactive streams", prefs.AutoCleanupInactive, nil)
	v.form.AddInputField("Inactive timeout (minutes)", strconv.Itoa(prefs.InactiveTimeoutMinutes), 20, nil, nil)

	// Buttons
	v.form.AddButton("Save", v.savePreferences)
	v.form.AddButton("Reset to Defaults", v.resetToDefaults)
	v.form.AddButton("Cancel", v.close)

	// Create scrollable container
	v.modal = tview.NewFlex()
	v.modal.SetDirection(tview.FlexRow)
	v.modal.AddItem(nil, 0, 1, false)
	v.modal.AddItem(tview.NewFlex().
		AddItem(nil, 0, 1, false).
		AddItem(v.form, 0, 3, true).
		AddItem(nil, 0, 1, false), 0, 8, true)
	v.modal.AddItem(nil, 0, 1, false)

	// Set up keyboard shortcuts
	v.form.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEsc {
			v.close()
			return nil
		}
		return event
	})
}

// savePreferences saves the current form values
func (v *StreamingPreferencesView) savePreferences() {
	err := v.prefsManager.UpdatePreferences(func(prefs *preferences.StreamingPreferences) {
		v.updateGeneralSettings(prefs)
		v.updatePerformanceSettings(prefs)
		v.updateRemoteSettings(prefs)
		v.updateDisplaySettings(prefs)
		v.updateAdvancedSettings(prefs)
	})

	if err != nil {
		v.showNotification(fmt.Sprintf("Failed to save preferences: %v", err))
		return
	}

	v.showNotification("Preferences saved successfully!")

	// Call the save callback if set
	if v.onSave != nil {
		v.onSave()
	}

	// Close after a short delay
	go func() {
		// Wait for notification to be visible
		v.app.QueueUpdateDraw(func() {
			v.close()
		})
	}()
}

// updateGeneralSettings updates general preference settings
func (v *StreamingPreferencesView) updateGeneralSettings(prefs *preferences.StreamingPreferences) {
	prefs.AutoStartForRunningJobs = v.getCheckboxValue("Auto-start streaming for running jobs")
	prefs.DefaultAutoScroll = v.getCheckboxValue("Default auto-scroll")
	prefs.ShowTimestamps = v.getCheckboxValue("Show timestamps")
	_, prefs.ExportFormat = v.form.GetFormItemByLabel("Export format").(*tview.DropDown).GetCurrentOption()
}

// updatePerformanceSettings updates performance preference settings
func (v *StreamingPreferencesView) updatePerformanceSettings(prefs *preferences.StreamingPreferences) {
	prefs.MaxConcurrentStreams = v.getIntField("Max concurrent streams")
	prefs.BufferSizeLines = v.getIntField("Buffer size (lines)")
	prefs.PollIntervalSeconds = v.getIntField("Poll interval (seconds)")
	prefs.MaxMemoryMB = v.getIntField("Max memory (MB)")
	prefs.FileCheckIntervalMs = v.getIntField("File check interval (ms)")
}

// updateRemoteSettings updates remote streaming preference settings
func (v *StreamingPreferencesView) updateRemoteSettings(prefs *preferences.StreamingPreferences) {
	prefs.EnableRemoteStreaming = v.getCheckboxValue("Enable remote streaming")
	prefs.SSHTimeout = v.getIntField("SSH timeout (seconds)")
	prefs.RemoteBufferSize = v.getIntField("Remote buffer size")
}

// updateDisplaySettings updates display preference settings
func (v *StreamingPreferencesView) updateDisplaySettings(prefs *preferences.StreamingPreferences) {
	_, prefs.MultiStreamGridSize = v.form.GetFormItemByLabel("Multi-stream grid size").(*tview.DropDown).GetCurrentOption()
	prefs.StreamPanelHeight = v.getIntField("Stream panel height")
	prefs.ShowBufferStats = v.getCheckboxValue("Show buffer statistics")
}

// updateAdvancedSettings updates advanced preference settings
func (v *StreamingPreferencesView) updateAdvancedSettings(prefs *preferences.StreamingPreferences) {
	prefs.EnableCompression = v.getCheckboxValue("Enable compression")
	prefs.StreamHistoryDays = v.getIntField("Stream history days")
	prefs.AutoCleanupInactive = v.getCheckboxValue("Auto-cleanup inactive streams")
	prefs.InactiveTimeoutMinutes = v.getIntField("Inactive timeout (minutes)")
}

// getIntField retrieves an integer field value from the form, returning 0 on error
func (v *StreamingPreferencesView) getIntField(label string) int {
	if val, err := strconv.Atoi(v.form.GetFormItemByLabel(label).(*tview.InputField).GetText()); err == nil {
		return val
	}
	return 0
}

// getCheckboxValue retrieves a checkbox value from the form
func (v *StreamingPreferencesView) getCheckboxValue(label string) bool {
	return v.form.GetFormItemByLabel(label).(*tview.Checkbox).IsChecked()
}

// resetToDefaults resets all preferences to default values
func (v *StreamingPreferencesView) resetToDefaults() {
	// Show confirmation dialog
	modal := tview.NewModal()
	modal.SetText("Are you sure you want to reset all streaming preferences to defaults?")
	modal.AddButtons([]string{"Reset", "Cancel"})
	modal.SetDoneFunc(func(buttonIndex int, _ string) {
		v.pages.RemovePage("reset-confirm")
		if buttonIndex == 0 { // Reset
			if err := v.prefsManager.Reset(); err != nil {
				v.showNotification(fmt.Sprintf("Failed to reset preferences: %v", err))
			} else {
				v.showNotification("Preferences reset to defaults!")
				v.buildUI() // Rebuild form with default values
			}
		}
	})

	v.pages.AddPage("reset-confirm", modal, true, true)
}

// showNotification shows a temporary notification
func (v *StreamingPreferencesView) showNotification(message string) {
	notification := tview.NewModal()
	notification.SetText(message)
	notification.AddButtons([]string{"OK"})
	notification.SetDoneFunc(func(_ int, _ string) {
		v.pages.RemovePage("notification")
	})

	v.pages.AddPage("notification", notification, true, true)
}

// show displays the preferences modal
func (v *StreamingPreferencesView) show() {
	if v.pages != nil {
		v.pages.AddPage("streaming-preferences", v.modal, true, true)
		v.app.SetFocus(v.form)
	}
}

// close closes the preferences dialog
func (v *StreamingPreferencesView) close() {
	if v.pages != nil {
		v.pages.RemovePage("streaming-preferences")
	}
}

// HighlightPatternsView manages highlight patterns separately
type HighlightPatternsView struct {
	app          *tview.Application
	pages        *tview.Pages
	modal        *tview.Flex
	list         *tview.List
	inputField   *tview.InputField
	prefsManager *preferences.StreamingPreferencesManager
}

// NewHighlightPatternsView creates a new highlight patterns view
func NewHighlightPatternsView(app *tview.Application, prefsManager *preferences.StreamingPreferencesManager) *HighlightPatternsView {
	return &HighlightPatternsView{
		app:          app,
		prefsManager: prefsManager,
	}
}

// SetPages sets the pages manager for modal display
func (v *HighlightPatternsView) SetPages(pages *tview.Pages) {
	v.pages = pages
}

// Show displays the highlight patterns dialog
func (v *HighlightPatternsView) Show() {
	v.buildUI()
	v.show()
}

// buildUI creates the highlight patterns interface
func (v *HighlightPatternsView) buildUI() {
	// Create list for patterns
	v.list = tview.NewList()
	v.list.SetBorder(true)
	v.list.SetTitle(" Highlight Patterns ")
	v.list.SetTitleAlign(tview.AlignCenter)
	v.list.ShowSecondaryText(false)

	// Load current patterns
	v.refreshPatterns()

	// Create input field for new patterns
	v.inputField = tview.NewInputField()
	v.inputField.SetLabel("Add pattern: ")
	v.inputField.SetFieldWidth(30)
	v.inputField.SetDoneFunc(func(key tcell.Key) {
		if key == tcell.KeyEnter {
			v.addPattern()
		}
	})

	// Create button row
	buttons := tview.NewFlex()
	buttons.SetDirection(tview.FlexColumn)

	addButton := tview.NewButton("Add")
	addButton.SetSelectedFunc(v.addPattern)

	removeButton := tview.NewButton("Remove")
	removeButton.SetSelectedFunc(v.removeSelectedPattern)

	closeButton := tview.NewButton("Close")
	closeButton.SetSelectedFunc(v.close)

	buttons.AddItem(addButton, 0, 1, false)
	buttons.AddItem(removeButton, 0, 1, false)
	buttons.AddItem(closeButton, 0, 1, false)

	// Create help text
	helpText := tview.NewTextView()
	helpText.SetDynamicColors(true)
	helpText.SetText("[yellow]Patterns are highlighted in streaming output. Use regex for advanced matching.[white]")
	helpText.SetTextAlign(tview.AlignCenter)

	// Create container
	container := tview.NewFlex()
	container.SetDirection(tview.FlexRow)
	container.SetBorder(true)
	container.AddItem(helpText, 1, 0, false)
	container.AddItem(v.list, 0, 1, true)
	container.AddItem(v.inputField, 3, 0, false)
	container.AddItem(buttons, 1, 0, false)

	// Create modal
	v.modal = tview.NewFlex()
	v.modal.SetDirection(tview.FlexRow)
	v.modal.AddItem(nil, 0, 1, false)
	v.modal.AddItem(tview.NewFlex().
		AddItem(nil, 0, 1, false).
		AddItem(container, 0, 2, true).
		AddItem(nil, 0, 1, false), 0, 2, true)
	v.modal.AddItem(nil, 0, 1, false)

	// Set up keyboard shortcuts
	v.modal.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEsc {
			v.close()
			return nil
		}
		return event
	})
}

// refreshPatterns refreshes the patterns list
func (v *HighlightPatternsView) refreshPatterns() {
	v.list.Clear()

	patterns := v.prefsManager.GetHighlightPatterns()
	for i, pattern := range patterns {
		index := i // Capture for closure
		v.list.AddItem(pattern, "", 0, func() {
			v.selectPattern(index)
		})
	}

	if len(patterns) == 0 {
		v.list.AddItem("[gray]No patterns defined[white]", "", 0, nil)
	}
}

// addPattern adds a new pattern
func (v *HighlightPatternsView) addPattern() {
	pattern := v.inputField.GetText()
	if pattern == "" {
		return
	}

	patterns := v.prefsManager.GetHighlightPatterns()
	patterns = append(patterns, pattern)

	if err := v.prefsManager.SetHighlightPatterns(patterns); err != nil {
		v.showNotification(fmt.Sprintf("Failed to add pattern: %v", err))
		return
	}

	v.inputField.SetText("")
	v.refreshPatterns()
}

// removeSelectedPattern removes the selected pattern
func (v *HighlightPatternsView) removeSelectedPattern() {
	index := v.list.GetCurrentItem()
	patterns := v.prefsManager.GetHighlightPatterns()

	if index < 0 || index >= len(patterns) {
		return
	}

	// Remove pattern at index
	newPatterns := make([]string, 0, len(patterns)-1)
	newPatterns = append(newPatterns, patterns[:index]...)
	newPatterns = append(newPatterns, patterns[index+1:]...)

	if err := v.prefsManager.SetHighlightPatterns(newPatterns); err != nil {
		v.showNotification(fmt.Sprintf("Failed to remove pattern: %v", err))
		return
	}

	v.refreshPatterns()
}

// selectPattern handles pattern selection
func (v *HighlightPatternsView) selectPattern(_ int) {
	// Could be used to edit patterns in the future
}

// showNotification shows a temporary notification
func (v *HighlightPatternsView) showNotification(message string) {
	notification := tview.NewModal()
	notification.SetText(message)
	notification.AddButtons([]string{"OK"})
	notification.SetDoneFunc(func(_ int, _ string) {
		v.pages.RemovePage("notification")
	})

	v.pages.AddPage("notification", notification, true, true)
}

// show displays the highlight patterns modal
func (v *HighlightPatternsView) show() {
	if v.pages != nil {
		v.pages.AddPage("highlight-patterns", v.modal, true, true)
		v.app.SetFocus(v.inputField)
	}
}

// close closes the highlight patterns dialog
func (v *HighlightPatternsView) close() {
	if v.pages != nil {
		v.pages.RemovePage("highlight-patterns")
	}
}
