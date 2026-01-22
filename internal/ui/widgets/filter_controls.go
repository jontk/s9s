package widgets

import (
	"fmt"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/jontk/s9s/internal/streaming"
	"github.com/rivo/tview"
)

// FilterControls provides a UI widget for managing stream filters
type FilterControls struct {
	container     *tview.Flex
	filterInput   *tview.InputField
	filterType    *tview.DropDown
	presetList    *tview.List
	activeFilters *tview.TextView
	statsView     *tview.TextView

	filterManager  *streaming.FilteredStreamManager
	onFilterChange func()

	// UI state
	showPresets bool
	currentType streaming.FilterType
}

// NewFilterControls creates a new filter controls widget
func NewFilterControls(filterManager *streaming.FilteredStreamManager) *FilterControls {
	fc := &FilterControls{
		filterManager: filterManager,
		currentType:   streaming.FilterTypeKeyword,
		showPresets:   false,
	}

	fc.buildUI()
	return fc
}

// buildUI creates the filter controls interface
func (fc *FilterControls) buildUI() {
	// Create main container
	fc.container = tview.NewFlex().SetDirection(tview.FlexRow)

	// Filter input row
	filterRow := tview.NewFlex().SetDirection(tview.FlexColumn)

	// Filter type dropdown
	fc.filterType = tview.NewDropDown()
	fc.filterType.SetLabel("Type: ")
	fc.filterType.SetOptions([]string{
		"Keyword",
		"Regex",
		"Log Level",
		"Time Range",
	}, func(option string, index int) {
		switch index {
		case 0:
			fc.currentType = streaming.FilterTypeKeyword
		case 1:
			fc.currentType = streaming.FilterTypeRegex
		case 2:
			fc.currentType = streaming.FilterTypeLogLevel
		case 3:
			fc.currentType = streaming.FilterTypeTimeRange
		}
	})
	fc.filterType.SetCurrentOption(0)
	fc.filterType.SetFieldWidth(15)

	// Filter input field
	fc.filterInput = tview.NewInputField()
	fc.filterInput.SetLabel("Filter: ")
	fc.filterInput.SetFieldBackgroundColor(tcell.ColorBlack)
	fc.filterInput.SetPlaceholder("Enter filter pattern...")
	fc.filterInput.SetDoneFunc(func(key tcell.Key) {
		if key == tcell.KeyEnter {
			fc.applyFilter()
		}
	})

	// Quick action buttons
	applyBtn := tview.NewButton("Apply")
	applyBtn.SetSelectedFunc(fc.applyFilter)

	clearBtn := tview.NewButton("Clear")
	clearBtn.SetSelectedFunc(fc.clearFilters)

	presetsBtn := tview.NewButton("Presets")
	presetsBtn.SetSelectedFunc(fc.togglePresets)

	// Add to filter row
	filterRow.AddItem(fc.filterType, 20, 0, false)
	filterRow.AddItem(fc.filterInput, 0, 1, true)
	filterRow.AddItem(applyBtn, 10, 0, false)
	filterRow.AddItem(clearBtn, 10, 0, false)
	filterRow.AddItem(presetsBtn, 12, 0, false)

	// Active filters display
	fc.activeFilters = tview.NewTextView()
	fc.activeFilters.SetDynamicColors(true)
	fc.activeFilters.SetBorder(true)
	fc.activeFilters.SetTitle(" Active Filters ")
	fc.activeFilters.SetText("[gray]No active filters[white]")

	// Filter statistics
	fc.statsView = tview.NewTextView()
	fc.statsView.SetDynamicColors(true)
	fc.statsView.SetBorder(true)
	fc.statsView.SetTitle(" Filter Stats ")
	fc.statsView.SetText("[gray]No statistics available[white]")

	// Preset list (hidden by default)
	fc.presetList = tview.NewList()
	fc.presetList.SetBorder(true)
	fc.presetList.SetTitle(" Filter Presets ")
	fc.presetList.ShowSecondaryText(true)
	fc.loadPresets()

	// Build layout
	fc.container.AddItem(filterRow, 3, 0, true)
	fc.container.AddItem(fc.activeFilters, 5, 0, false)
	fc.container.AddItem(fc.statsView, 3, 0, false)

	// Add preset list when visible
	if fc.showPresets {
		fc.container.AddItem(fc.presetList, 0, 1, false)
	}
}

// applyFilter applies the current filter
func (fc *FilterControls) applyFilter() {
	pattern := fc.filterInput.GetText()
	if pattern == "" {
		return
	}

	// Apply the filter
	err := fc.filterManager.SetQuickFilter(pattern, fc.currentType)
	if err != nil {
		fc.activeFilters.SetText(fmt.Sprintf("[red]Error: %v[white]", err))
		return
	}

	// Update display
	fc.updateActiveFilters()

	// Clear input
	fc.filterInput.SetText("")

	// Trigger callback
	if fc.onFilterChange != nil {
		fc.onFilterChange()
	}
}

// clearFilters removes all active filters
func (fc *FilterControls) clearFilters() {
	fc.filterManager.ClearFilters()
	fc.updateActiveFilters()

	if fc.onFilterChange != nil {
		fc.onFilterChange()
	}
}

// togglePresets shows/hides the preset list
func (fc *FilterControls) togglePresets() {
	fc.showPresets = !fc.showPresets

	// Rebuild UI
	fc.container.Clear()
	fc.buildUI()
}

// loadPresets loads available filter presets
func (fc *FilterControls) loadPresets() {
	fc.presetList.Clear()

	presets := fc.filterManager.GetFilterPresets()

	// Group presets by category
	categories := make(map[string][]*streaming.FilterPreset)
	for _, preset := range presets {
		category := preset.Category
		if category == "" {
			category = "General"
		}
		categories[category] = append(categories[category], preset)
	}

	// Add presets to list
	for category, presetList := range categories {
		// Add category header
		fc.presetList.AddItem(fmt.Sprintf("[yellow]── %s ──[white]", category), "", 0, nil)

		// Add presets
		for _, preset := range presetList {
			mainText := preset.Name
			secondaryText := preset.Description

			p := preset // Capture for closure
			fc.presetList.AddItem(mainText, secondaryText, 0, func() {
				fc.applyPreset(p.ID)
			})
		}
	}

	// Add save option
	fc.presetList.AddItem("[green]+ Save Current Filters[white]", "Save active filters as preset", 0, fc.savePreset)
}

// applyPreset applies a filter preset
func (fc *FilterControls) applyPreset(presetID string) {
	err := fc.filterManager.LoadFilterPreset(presetID)
	if err != nil {
		fc.activeFilters.SetText(fmt.Sprintf("[red]Error loading preset: %v[white]", err))
		return
	}

	fc.updateActiveFilters()

	if fc.onFilterChange != nil {
		fc.onFilterChange()
	}
}

// savePreset saves current filters as a preset
func (fc *FilterControls) savePreset() {
	// This would normally show a dialog to get preset details
	// For now, create with default values
	preset, err := fc.filterManager.SaveFilterPreset(
		"Custom Filter",
		"User-defined filter preset",
		"Custom",
	)

	if err != nil {
		fc.activeFilters.SetText(fmt.Sprintf("[red]Error saving preset: %v[white]", err))
		return
	}

	// Reload presets
	fc.loadPresets()

	fc.activeFilters.SetText(fmt.Sprintf("[green]Preset saved: %s[white]", preset.Name))
}

// updateActiveFilters updates the active filters display
func (fc *FilterControls) updateActiveFilters() {
	filters := fc.filterManager.GetActiveFilters()

	if len(filters) == 0 {
		fc.activeFilters.SetText("[gray]No active filters[white]")
		fc.statsView.SetText("[gray]No statistics available[white]")
		return
	}

	// Build filter display
	var filterText strings.Builder
	for i, filter := range filters {
		color := "green"
		if !filter.Enabled {
			color = "gray"
		}

		filterText.WriteString(fmt.Sprintf("[%s]%d. %s: %s[white]\n",
			color, i+1, filter.Type, filter.Pattern))
	}

	fc.activeFilters.SetText(filterText.String())

	// Update stats
	fc.updateStats()
}

// updateStats updates filter statistics
func (fc *FilterControls) updateStats() {
	stats := fc.filterManager.GetFilterStats()

	if len(stats) == 0 {
		fc.statsView.SetText("[gray]No statistics available[white]")
		return
	}

	var statsText strings.Builder
	totalMatches := int64(0)
	totalProcessed := int64(0)

	for _, stat := range stats {
		totalMatches += stat.MatchCount
		totalProcessed += stat.ProcessedLines
	}

	if totalProcessed > 0 {
		percentage := float64(totalMatches) / float64(totalProcessed) * 100
		statsText.WriteString(fmt.Sprintf("Matches: [yellow]%d[white] / %d lines ([green]%.1f%%[white])\n",
			totalMatches, totalProcessed, percentage))
	}

	fc.statsView.SetText(statsText.String())
}

// SetOnFilterChange sets the callback for filter changes
func (fc *FilterControls) SetOnFilterChange(callback func()) {
	fc.onFilterChange = callback
}

// GetContainer returns the main container
func (fc *FilterControls) GetContainer() tview.Primitive {
	return fc.container
}

// Focus sets focus to the filter input
func (fc *FilterControls) Focus() {
	// Simple focus without delegate
	// This method will be called by the app to set focus
}

// HandleInput processes keyboard input
func (fc *FilterControls) HandleInput(event *tcell.EventKey) *tcell.EventKey {
	switch event.Key() {
	case tcell.KeyF3:
		// Quick filter for errors
		fc.filterInput.SetText("ERROR|FATAL")
		fc.currentType = streaming.FilterTypeRegex
		fc.filterType.SetCurrentOption(1)
		fc.applyFilter()
		return nil
	case tcell.KeyF4:
		// Quick filter for warnings
		fc.filterInput.SetText("WARN|WARNING")
		fc.currentType = streaming.FilterTypeRegex
		fc.filterType.SetCurrentOption(1)
		fc.applyFilter()
		return nil
	case tcell.KeyCtrlF:
		// Clear filters
		fc.clearFilters()
		return nil
	}

	return event
}

// Refresh updates the filter controls display
func (fc *FilterControls) Refresh() {
	fc.updateActiveFilters()
	fc.updateStats()
}
