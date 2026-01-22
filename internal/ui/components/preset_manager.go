package components

import (
	"fmt"

	"github.com/gdamore/tcell/v2"
	"github.com/jontk/s9s/internal/ui/filters"
	"github.com/rivo/tview"
)

// PresetManagerUI provides a UI for managing filter presets
type PresetManagerUI struct {
	app           *tview.Application
	pages         *tview.Pages
	presetManager *filters.PresetManager
	viewType      string
	onDone        func()
}

// NewPresetManagerUI creates a new preset manager UI
func NewPresetManagerUI(app *tview.Application, presetManager *filters.PresetManager, viewType string) *PresetManagerUI {
	return &PresetManagerUI{
		app:           app,
		presetManager: presetManager,
		viewType:      viewType,
	}
}

// Show displays the preset management interface
func (pm *PresetManagerUI) Show(pages *tview.Pages, onDone func()) {
	pm.pages = pages
	pm.onDone = onDone
	pm.showPresetList()
}

// showPresetList shows the list of presets with management options
func (pm *PresetManagerUI) showPresetList() {
	presets := pm.presetManager.GetPresets(pm.viewType)

	// Create the preset list
	list := tview.NewList()
	list.SetBorder(true).
		SetTitle(fmt.Sprintf(" Manage Filter Presets - %s ", pm.viewType)).
		SetTitleAlign(tview.AlignCenter)

	// Add presets to the list
	for _, preset := range presets {
		p := preset // Capture for closure
		typeStr := ""
		if p.IsGlobal {
			typeStr = " [cyan](global)[white]"
		}

		list.AddItem(
			fmt.Sprintf("%s%s", p.Name, typeStr),
			fmt.Sprintf("%s | Filter: %s", p.Description, p.FilterStr),
			0,
			func() {
				pm.showPresetActions(p)
			},
		)
	}

	// Add separator and actions
	if len(presets) > 0 {
		list.AddItem("──────────", "", 0, nil)
	}

	list.AddItem("Create New Preset", "Add a new filter preset", 'n', func() {
		pm.showCreatePresetDialog()
	})

	list.AddItem("Import Presets", "Import presets from file", 'i', func() {
		pm.showImportDialog()
	})

	list.AddItem("Export Presets", "Export presets to file", 'e', func() {
		pm.showExportDialog()
	})

	list.AddItem("Reset to Defaults", "Restore default presets", 'r', func() {
		pm.showResetConfirmation()
	})

	list.AddItem("Close", "Return to filter bar", 'q', func() {
		pm.close()
	})

	// Handle ESC key
	list.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEsc {
			pm.close()
			return nil
		}
		return event
	})

	// Show as modal
	modal := pm.createCenteredModal(list, 80, 25)
	pm.pages.AddPage("preset-manager", modal, true, true)
	pm.app.SetFocus(list)
}

// showPresetActions shows actions for a specific preset
func (pm *PresetManagerUI) showPresetActions(preset filters.FilterPreset) {
	list := tview.NewList()
	list.SetBorder(true).
		SetTitle(fmt.Sprintf(" Actions for '%s' ", preset.Name)).
		SetTitleAlign(tview.AlignCenter)

	// Show preset details
	details := fmt.Sprintf(
		"[yellow]Name:[white] %s\n[yellow]Description:[white] %s\n[yellow]Filter:[white] %s\n[yellow]Type:[white] %s",
		preset.Name,
		preset.Description,
		preset.FilterStr,
		map[bool]string{true: "Global", false: fmt.Sprintf("View-specific (%s)", preset.ViewType)}[preset.IsGlobal],
	)

	detailsView := tview.NewTextView().
		SetDynamicColors(true).
		SetText(details).
		SetBorder(true).
		SetTitle(" Preset Details ")

	// Actions
	list.AddItem("Edit Preset", "Modify this preset", 'e', func() {
		pm.showEditPresetDialog(preset)
	})

	list.AddItem("Duplicate Preset", "Create a copy of this preset", 'd', func() {
		pm.showDuplicatePresetDialog(preset)
	})

	list.AddItem("Test Filter", "Test this filter in current view", 't', func() {
		pm.testPreset(preset)
	})

	list.AddItem("Delete Preset", "Remove this preset permanently", 'x', func() {
		pm.showDeleteConfirmation(preset)
	})

	list.AddItem("Back to List", "Return to preset list", 'b', func() {
		pm.pages.RemovePage("preset-actions")
		pm.showPresetList()
	})

	// Layout
	flex := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(detailsView, 8, 0, false).
		AddItem(list, 0, 1, true)

	// Handle ESC
	flex.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEsc {
			pm.pages.RemovePage("preset-actions")
			pm.showPresetList()
			return nil
		}
		return event
	})

	modal := pm.createCenteredModal(flex, 70, 20)
	pm.pages.AddPage("preset-actions", modal, true, true)
	pm.app.SetFocus(list)
}

// showCreatePresetDialog shows the create preset dialog
func (pm *PresetManagerUI) showCreatePresetDialog() {
	pm.showPresetForm(filters.FilterPreset{ViewType: pm.viewType}, "Create New Preset", func(preset filters.FilterPreset) {
		err := pm.presetManager.AddPreset(preset)
		if err != nil {
			pm.showError(fmt.Sprintf("Failed to create preset: %v", err))
			return
		}
		pm.pages.RemovePage("preset-form")
		pm.showPresetList()
		pm.showSuccess(fmt.Sprintf("Preset '%s' created successfully", preset.Name))
	})
}

// showEditPresetDialog shows the edit preset dialog
func (pm *PresetManagerUI) showEditPresetDialog(preset filters.FilterPreset) {
	originalName := preset.Name
	pm.showPresetForm(preset, "Edit Preset", func(updatedPreset filters.FilterPreset) {
		err := pm.presetManager.UpdatePreset(originalName, preset.ViewType, updatedPreset)
		if err != nil {
			pm.showError(fmt.Sprintf("Failed to update preset: %v", err))
			return
		}
		pm.pages.RemovePage("preset-form")
		pm.showPresetList()
		pm.showSuccess(fmt.Sprintf("Preset '%s' updated successfully", updatedPreset.Name))
	})
}

// showDuplicatePresetDialog shows the duplicate preset dialog
func (pm *PresetManagerUI) showDuplicatePresetDialog(preset filters.FilterPreset) {
	// Create a copy with modified name
	newPreset := preset
	newPreset.Name = preset.Name + " Copy"

	pm.showPresetForm(newPreset, "Duplicate Preset", func(duplicatedPreset filters.FilterPreset) {
		err := pm.presetManager.AddPreset(duplicatedPreset)
		if err != nil {
			pm.showError(fmt.Sprintf("Failed to duplicate preset: %v", err))
			return
		}
		pm.pages.RemovePage("preset-form")
		pm.showPresetList()
		pm.showSuccess(fmt.Sprintf("Preset '%s' duplicated successfully", duplicatedPreset.Name))
	})
}

// showPresetForm shows the preset creation/editing form
func (pm *PresetManagerUI) showPresetForm(preset filters.FilterPreset, title string, onSave func(filters.FilterPreset)) {
	form := tview.NewForm()

	var name, description, filterStr string
	var isGlobal bool

	// Initialize with current values
	name = preset.Name
	description = preset.Description
	filterStr = preset.FilterStr
	isGlobal = preset.IsGlobal

	// Form fields
	form.AddInputField("Name", name, 40, nil, func(text string) {
		name = text
	})

	form.AddInputField("Description", description, 60, nil, func(text string) {
		description = text
	})

	form.AddTextArea("Filter String", filterStr, 60, 3, 0, func(text string) {
		filterStr = text
	})

	form.AddCheckbox("Global (available in all views)", isGlobal, func(checked bool) {
		isGlobal = checked
	})

	// Buttons
	form.AddButton("Save", func() {
		if name == "" {
			pm.showError("Preset name is required")
			return
		}
		if filterStr == "" {
			pm.showError("Filter string is required")
			return
		}

		newPreset := filters.FilterPreset{
			Name:        name,
			Description: description,
			FilterStr:   filterStr,
			ViewType:    pm.viewType,
			IsGlobal:    isGlobal,
		}

		if isGlobal {
			newPreset.ViewType = "all"
		}

		onSave(newPreset)
	})

	form.AddButton("Cancel", func() {
		pm.pages.RemovePage("preset-form")
		pm.showPresetList()
	})

	form.SetBorder(true).
		SetTitle(fmt.Sprintf(" %s ", title)).
		SetTitleAlign(tview.AlignCenter)

	// Handle ESC
	form.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEsc {
			pm.pages.RemovePage("preset-form")
			pm.showPresetList()
			return nil
		}
		return event
	})

	modal := pm.createCenteredModal(form, 80, 18)
	pm.pages.AddPage("preset-form", modal, true, true)
	pm.app.SetFocus(form)
}

// showDeleteConfirmation shows confirmation dialog for preset deletion
func (pm *PresetManagerUI) showDeleteConfirmation(preset filters.FilterPreset) {
	modal := tview.NewModal().
		SetText(fmt.Sprintf("Are you sure you want to delete the preset '%s'?\n\nThis action cannot be undone.", preset.Name)).
		AddButtons([]string{"Delete", "Cancel"}).
		SetDoneFunc(func(buttonIndex int, buttonLabel string) {
			pm.pages.RemovePage("delete-confirmation")
			if buttonLabel == "Delete" {
				err := pm.presetManager.RemovePreset(preset.Name, preset.ViewType)
				if err != nil {
					pm.showError(fmt.Sprintf("Failed to delete preset: %v", err))
					return
				}
				pm.showPresetList()
				pm.showSuccess(fmt.Sprintf("Preset '%s' deleted successfully", preset.Name))
			} else {
				pm.showPresetActions(preset)
			}
		})

	modal.SetBackgroundColor(tcell.ColorDefault).
		SetTextColor(tcell.ColorWhite).
		SetButtonTextColor(tcell.ColorWhite).
		SetButtonBackgroundColor(tcell.ColorDarkRed)

	pm.pages.AddPage("delete-confirmation", modal, true, true)
}

// Helper methods

func (pm *PresetManagerUI) testPreset(preset filters.FilterPreset) {
	// TODO: Implement filter testing functionality
	pm.showInfo(fmt.Sprintf("Testing filter: %s\n\nThis feature will be implemented to show preview results.", preset.FilterStr))
}

func (pm *PresetManagerUI) showImportDialog() {
	pm.showInfo("Import functionality will be implemented to load presets from JSON files.")
}

func (pm *PresetManagerUI) showExportDialog() {
	pm.showInfo("Export functionality will be implemented to save presets to JSON files.")
}

func (pm *PresetManagerUI) showResetConfirmation() {
	modal := tview.NewModal().
		SetText("Reset all presets to default values?\n\nThis will remove all custom presets and restore the original defaults.").
		AddButtons([]string{"Reset", "Cancel"}).
		SetDoneFunc(func(buttonIndex int, buttonLabel string) {
			pm.pages.RemovePage("reset-confirmation")
			if buttonLabel == "Reset" {
				// TODO: Implement reset to defaults functionality
				pm.showInfo("Reset functionality will be implemented to restore default presets.")
			}
		})

	pm.pages.AddPage("reset-confirmation", modal, true, true)
}

func (pm *PresetManagerUI) showError(message string) {
	modal := tview.NewModal().
		SetText(message).
		AddButtons([]string{"OK"}).
		SetDoneFunc(func(buttonIndex int, buttonLabel string) {
			pm.pages.RemovePage("error-modal")
		})

	modal.SetBackgroundColor(tcell.ColorDefault).
		SetTextColor(tcell.ColorRed)

	pm.pages.AddPage("error-modal", modal, true, true)
}

func (pm *PresetManagerUI) showSuccess(message string) {
	modal := tview.NewModal().
		SetText(message).
		AddButtons([]string{"OK"}).
		SetDoneFunc(func(buttonIndex int, buttonLabel string) {
			pm.pages.RemovePage("success-modal")
		})

	modal.SetBackgroundColor(tcell.ColorDefault).
		SetTextColor(tcell.ColorGreen)

	pm.pages.AddPage("success-modal", modal, true, true)
}

func (pm *PresetManagerUI) showInfo(message string) {
	modal := tview.NewModal().
		SetText(message).
		AddButtons([]string{"OK"}).
		SetDoneFunc(func(buttonIndex int, buttonLabel string) {
			pm.pages.RemovePage("info-modal")
		})

	modal.SetBackgroundColor(tcell.ColorDefault).
		SetTextColor(tcell.ColorYellow)

	pm.pages.AddPage("info-modal", modal, true, true)
}

func (pm *PresetManagerUI) close() {
	pm.pages.RemovePage("preset-manager")
	if pm.onDone != nil {
		pm.onDone()
	}
}

func (pm *PresetManagerUI) createCenteredModal(content tview.Primitive, width, height int) tview.Primitive {
	return tview.NewFlex().
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(nil, 0, 1, false).
			AddItem(content, height, 1, true).
			AddItem(nil, 0, 1, false), width, 1, true).
		AddItem(nil, 0, 1, false)
}
