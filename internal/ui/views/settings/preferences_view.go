package settings

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/jontk/s9s/internal/preferences"
	"github.com/rivo/tview"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

// PreferencesView displays the preferences interface
type PreferencesView struct {
	prefs    *preferences.UserPreferences
	app      *tview.Application
	pages    *tview.Pages
	layout   *tview.Flex
	tree     *tview.TreeView
	form     *tview.Form
	rootNode *tview.TreeNode
	modified bool
	onSave   func()
	onCancel func()
}

// NewPreferencesView creates a new preferences view
func NewPreferencesView(prefs *preferences.UserPreferences, app *tview.Application, pages *tview.Pages) *PreferencesView {
	pv := &PreferencesView{
		prefs: prefs,
		app:   app,
		pages: pages,
	}

	pv.buildUI()
	return pv
}

// buildUI builds the preferences interface
func (pv *PreferencesView) buildUI() {
	// Create tree view for categories
	pv.tree = tview.NewTreeView()
	pv.tree.SetBorder(true).SetTitle(" Categories ")

	// Build category tree
	pv.buildCategoryTree()

	// Create form for settings
	pv.form = tview.NewForm()
	pv.form.SetBorder(true).SetTitle(" Settings ")

	// Show general settings by default
	pv.showGeneralSettings()

	// Create layout
	pv.layout = tview.NewFlex().
		AddItem(pv.tree, 30, 0, true).
		AddItem(pv.form, 0, 1, false)

	// Handle tree selection
	pv.tree.SetSelectedFunc(pv.onCategorySelected)

	// Handle keyboard shortcuts
	pv.layout.SetInputCapture(pv.handleKeyboard)
}

// buildCategoryTree builds the category tree
func (pv *PreferencesView) buildCategoryTree() {
	root := tview.NewTreeNode("Preferences")
	pv.rootNode = root
	pv.tree.SetRoot(root).SetCurrentNode(root)

	// General category
	general := tview.NewTreeNode("General").
		SetReference("general").
		SetSelectable(true)
	root.AddChild(general)

	// Display category
	display := tview.NewTreeNode("Display").
		SetReference("display").
		SetSelectable(true)
	root.AddChild(display)

	// Colors category
	colors := tview.NewTreeNode("Colors & Theme").
		SetReference("colors").
		SetSelectable(true)
	root.AddChild(colors)

	// Key Bindings category
	keybindings := tview.NewTreeNode("Key Bindings").
		SetReference("keybindings").
		SetSelectable(true)
	root.AddChild(keybindings)

	// Views category
	views := tview.NewTreeNode("View Settings").
		SetReference("views").
		SetSelectable(true)
	root.AddChild(views)

	// Add view subcategories
	viewNames := []string{"Jobs", "Nodes", "Partitions", "Reservations", "QoS", "Accounts", "Users", "Health"}
	for _, name := range viewNames {
		child := tview.NewTreeNode(name).
			SetReference("view:" + strings.ToLower(name)).
			SetSelectable(true)
		views.AddChild(child)
	}

	// Filters category
	filters := tview.NewTreeNode("Filters & Search").
		SetReference("filters").
		SetSelectable(true)
	root.AddChild(filters)

	// Job Submission category
	jobsub := tview.NewTreeNode("Job Submission").
		SetReference("jobsubmission").
		SetSelectable(true)
	root.AddChild(jobsub)

	// Alerts category
	alerts := tview.NewTreeNode("Alerts").
		SetReference("alerts").
		SetSelectable(true)
	root.AddChild(alerts)

	// Performance category
	performance := tview.NewTreeNode("Performance").
		SetReference("performance").
		SetSelectable(true)
	root.AddChild(performance)

	// Expand root by default
	root.SetExpanded(true)
}

// onCategorySelected handles category selection
func (pv *PreferencesView) onCategorySelected(node *tview.TreeNode) {
	ref := node.GetReference()
	if ref == nil {
		return
	}

	category := ref.(string)

	pv.form.Clear(true)
	pv.showCategorySettings(category)
	pv.addFormButtons()
}

// showCategorySettings displays settings for the selected category
func (pv *PreferencesView) showCategorySettings(category string) {
	handlers := map[string]func(){
		"general":       pv.showGeneralSettings,
		"display":       pv.showDisplaySettings,
		"colors":        pv.showColorSettings,
		"keybindings":   pv.showKeyBindings,
		"filters":       pv.showFilterSettings,
		"jobsubmission": pv.showJobSubmissionSettings,
		"alerts":        pv.showAlertSettings,
		"performance":   pv.showPerformanceSettings,
	}

	if handler, ok := handlers[category]; ok {
		handler()
	} else if strings.HasPrefix(category, "view:") {
		viewName := strings.TrimPrefix(category, "view:")
		pv.showViewSettings(viewName)
	}
}

// addFormButtons adds save/cancel/reset buttons to the form
func (pv *PreferencesView) addFormButtons() {
	pv.form.
		AddButton("Save", pv.save).
		AddButton("Cancel", pv.cancel).
		AddButton("Reset", pv.reset)
}

// showGeneralSettings shows general preferences
func (pv *PreferencesView) showGeneralSettings() {
	prefs := pv.prefs.Get()

	pv.form.
		AddCheckbox("Auto Refresh", prefs.General.AutoRefresh, nil).
		AddInputField("Refresh Interval", prefs.General.RefreshInterval, 20, nil, nil).
		AddDropDown("Theme", []string{"default", "dark", "light"},
			pv.getThemeIndex(prefs.General.Theme), nil).
		AddInputField("Date Format", prefs.General.DateFormat, 30, nil, nil).
		AddCheckbox("Show Relative Time", prefs.General.RelativeTime, nil).
		AddCheckbox("Confirm on Exit", prefs.General.ConfirmOnExit, nil).
		AddCheckbox("Show Welcome Screen", prefs.General.ShowWelcome, nil).
		AddDropDown("Default View", []string{"jobs", "nodes", "partitions", "reservations", "qos", "accounts", "users", "health"},
			pv.getViewIndex(prefs.General.DefaultView), nil).
		AddCheckbox("Save Window Size", prefs.General.SaveWindowSize, nil)

	pv.form.SetTitle(" General Settings ")
}

// showDisplaySettings shows display preferences
func (pv *PreferencesView) showDisplaySettings() {
	prefs := pv.prefs.Get()

	pv.form.
		AddCheckbox("Show Header", prefs.Display.ShowHeader, nil).
		AddCheckbox("Show Status Bar", prefs.Display.ShowStatusBar, nil).
		AddCheckbox("Show Line Numbers", prefs.Display.ShowLineNumbers, nil).
		AddCheckbox("Compact Mode", prefs.Display.CompactMode, nil).
		AddCheckbox("Show Grid Lines", prefs.Display.ShowGridLines, nil).
		AddCheckbox("Enable Animations", prefs.Display.AnimationsEnabled, nil).
		AddCheckbox("Highlight Changes", prefs.Display.HighlightChanges, nil).
		AddCheckbox("Truncate Long Text", prefs.Display.TruncateLongText, nil).
		AddInputField("Max Column Width", strconv.Itoa(prefs.Display.MaxColumnWidth), 10, nil, nil).
		AddDropDown("Time Zone", []string{"Local", "UTC"},
			pv.getTimeZoneIndex(prefs.Display.TimeZone), nil)

	pv.form.SetTitle(" Display Settings ")
}

// showColorSettings shows color preferences
func (pv *PreferencesView) showColorSettings() {
	prefs := pv.prefs.Get()

	schemes := []string{"default", "solarized", "monokai", "dracula", "nord"}
	colorBlindModes := []string{"None", "Protanopia", "Deuteranopia", "Tritanopia"}

	pv.form.
		AddDropDown("Color Scheme", schemes,
			pv.getSchemeIndex(prefs.Colors.Scheme), nil).
		AddCheckbox("High Contrast", prefs.Colors.HighContrast, nil).
		AddDropDown("Color Blind Mode", colorBlindModes,
			pv.getColorBlindIndex(prefs.Colors.ColorBlindMode), nil).
		AddCheckbox("Syntax Highlighting", prefs.Colors.SyntaxHighlight, nil)

	pv.form.SetTitle(" Color & Theme Settings ")
}

// showKeyBindings shows key binding preferences
func (pv *PreferencesView) showKeyBindings() {
	prefs := pv.prefs.Get()

	pv.form.AddTextView("Info",
		"[yellow]Key bindings can be customized below.[white]\n"+
			"Format: Single key (q) or modifier+key (Ctrl+s)",
		0, 3, true, false)

	// Common actions
	actions := []struct {
		name   string
		action string
	}{
		{"Quit", "quit"},
		{"Help", "help"},
		{"Refresh", "refresh"},
		{"Search", "search"},
		{"Filter", "filter"},
		{"Next View", "next_view"},
		{"Previous View", "prev_view"},
		{"Select", "select"},
		{"Cancel", "cancel"},
		{"Delete", "delete"},
		{"Edit", "edit"},
		{"New", "new"},
		{"Save", "save"},
	}

	for _, a := range actions {
		binding := prefs.KeyBindings[a.action]
		pv.form.AddInputField(a.name, binding, 20, nil, nil)
	}

	pv.form.SetTitle(" Key Bindings ")
}

// showFilterSettings shows filter preferences
func (pv *PreferencesView) showFilterSettings() {
	prefs := pv.prefs.Get()

	operators := []string{"contains", "equals", "starts with", "ends with", "regex"}

	pv.form.
		AddCheckbox("Save Filter History", prefs.Filters.SaveHistory, nil).
		AddInputField("History Size", strconv.Itoa(prefs.Filters.HistorySize), 10, nil, nil).
		AddDropDown("Default Operator", operators,
			pv.getOperatorIndex(prefs.Filters.DefaultOperator), nil).
		AddCheckbox("Case Sensitive", prefs.Filters.CaseSensitive, nil).
		AddCheckbox("Use Regular Expressions", prefs.Filters.UseRegex, nil).
		AddCheckbox("Show Advanced Options", prefs.Filters.ShowAdvanced, nil)

	pv.form.SetTitle(" Filter & Search Settings ")
}

// showJobSubmissionSettings shows job submission preferences
func (pv *PreferencesView) showJobSubmissionSettings() {
	prefs := pv.prefs.Get()

	pv.form.
		AddCheckbox("Save Submission History", prefs.JobSubmission.SaveHistory, nil).
		AddInputField("History Size", strconv.Itoa(prefs.JobSubmission.HistorySize), 10, nil, nil).
		AddCheckbox("Validate on Type", prefs.JobSubmission.ValidateOnType, nil).
		AddCheckbox("Show Advanced Options", prefs.JobSubmission.ShowAdvancedOptions, nil).
		AddCheckbox("Auto Suggest", prefs.JobSubmission.AutoSuggest, nil).
		AddInputField("Default Template", prefs.JobSubmission.DefaultTemplate, 30, nil, nil)

	pv.form.SetTitle(" Job Submission Settings ")
}

// showAlertSettings shows alert preferences
func (pv *PreferencesView) showAlertSettings() {
	prefs := pv.prefs.Get()

	positions := []string{"top-right", "top-left", "bottom-right", "bottom-left"}

	pv.form.
		AddCheckbox("Show Alert Badge", prefs.Alerts.ShowBadge, nil).
		AddDropDown("Badge Position", positions,
			pv.getBadgePositionIndex(prefs.Alerts.BadgePosition), nil).
		AddCheckbox("Auto Dismiss Info Alerts", prefs.Alerts.AutoDismissInfo, nil).
		AddInputField("Info Dismiss Time", prefs.Alerts.InfoDismissTime, 10, nil, nil).
		AddCheckbox("Play Sound", prefs.Alerts.PlaySound, nil).
		AddCheckbox("Flash Window", prefs.Alerts.FlashWindow, nil).
		AddCheckbox("Show Desktop Notifications", prefs.Alerts.ShowDesktopNotif, nil)

	pv.form.SetTitle(" Alert Settings ")
}

// showPerformanceSettings shows performance preferences
func (pv *PreferencesView) showPerformanceSettings() {
	prefs := pv.prefs.Get()

	pv.form.
		AddCheckbox("Enable Lazy Loading", prefs.Performance.LazyLoading, nil).
		AddInputField("Cache Size (MB)", strconv.Itoa(prefs.Performance.CacheSize), 10, nil, nil).
		AddInputField("Max Concurrent Requests", strconv.Itoa(prefs.Performance.MaxConcurrentReq), 10, nil, nil).
		AddInputField("Request Timeout", prefs.Performance.RequestTimeout, 10, nil, nil).
		AddCheckbox("Enable Profiling", prefs.Performance.EnableProfiling, nil).
		AddCheckbox("Debug Mode", prefs.Performance.DebugMode, nil)

	pv.form.SetTitle(" Performance Settings ")
}

// showViewSettings shows settings for a specific view
func (pv *PreferencesView) showViewSettings(viewName string) {
	settings := pv.prefs.GetViewSettings(viewName)

	pv.form.
		AddInputField("Sort Column", settings.SortColumn, 20, nil, nil).
		AddDropDown("Sort Order", []string{"asc", "desc"},
			pv.getSortOrderIndex(settings.SortOrder), nil).
		AddCheckbox("Show Details", settings.ShowDetails, nil).
		AddCheckbox("Auto Expand Groups", settings.AutoExpandGroups, nil).
		AddInputField("Page Size", strconv.Itoa(settings.PageSize), 10, nil, nil).
		AddInputField("Group By", settings.GroupBy, 20, nil, nil)

	pv.form.SetTitle(fmt.Sprintf(" %s View Settings ", cases.Title(language.English).String(viewName)))
}

// save saves the preferences
func (pv *PreferencesView) save() {
	// Extract values from form and update preferences
	err := pv.prefs.Update(func(_ *preferences.UserPreferences) error {
		// This is simplified - in production you'd extract all form values
		// and update the preferences structure
		return nil
	})

	if err != nil {
		pv.showError(fmt.Sprintf("Failed to save preferences: %v", err))
		return
	}

	pv.modified = false
	if pv.onSave != nil {
		pv.onSave()
	}

	// Close preferences
	if pv.pages != nil {
		pv.pages.RemovePage("preferences")
	}
}

// cancel cancels preference changes
func (pv *PreferencesView) cancel() {
	if pv.modified {
		// Show confirmation dialog
		modal := tview.NewModal().
			SetText("Discard unsaved changes?").
			AddButtons([]string{"Yes", "No"}).
			SetDoneFunc(func(buttonIndex int, _ string) {
				if buttonIndex == 0 {
					pv.pages.RemovePage("confirm")
					if pv.onCancel != nil {
						pv.onCancel()
					}
					pv.pages.RemovePage("preferences")
				} else {
					pv.pages.RemovePage("confirm")
				}
			})
		pv.pages.AddPage("confirm", modal, true, true)
	} else {
		if pv.onCancel != nil {
			pv.onCancel()
		}
		if pv.pages != nil {
			pv.pages.RemovePage("preferences")
		}
	}
}

// reset resets preferences to defaults
func (pv *PreferencesView) reset() {
	modal := tview.NewModal().
		SetText("Reset all preferences to defaults?").
		AddButtons([]string{"Yes", "No"}).
		SetDoneFunc(func(buttonIndex int, _ string) {
			if buttonIndex == 0 {
				if err := pv.prefs.Reset(); err != nil {
					pv.showError(fmt.Sprintf("Failed to reset preferences: %v", err))
				} else {
					// Refresh current category
					if node := pv.tree.GetCurrentNode(); node != nil {
						pv.onCategorySelected(node)
					}
				}
			}
			pv.pages.RemovePage("reset-confirm")
		})
	pv.pages.AddPage("reset-confirm", modal, true, true)
}

// handleKeyboard handles keyboard shortcuts
func (pv *PreferencesView) handleKeyboard(event *tcell.EventKey) *tcell.EventKey {
	switch event.Key() {
	case tcell.KeyEsc:
		pv.cancel()
		return nil
	case tcell.KeyCtrlS:
		pv.save()
		return nil
	case tcell.KeyTab:
		// Switch focus between tree and form
		if pv.app.GetFocus() == pv.tree {
			pv.app.SetFocus(pv.form)
		} else {
			pv.app.SetFocus(pv.tree)
		}
		return nil
	}
	return event
}

// Helper methods for dropdown indices
func (pv *PreferencesView) getThemeIndex(theme string) int {
	themes := []string{"default", "dark", "light"}
	for i, t := range themes {
		if t == theme {
			return i
		}
	}
	return 0
}

func (pv *PreferencesView) getViewIndex(view string) int {
	views := []string{"jobs", "nodes", "partitions", "reservations", "qos", "accounts", "users", "health"}
	for i, v := range views {
		if v == view {
			return i
		}
	}
	return 0
}

func (pv *PreferencesView) getTimeZoneIndex(tz string) int {
	if tz == "UTC" {
		return 1
	}
	return 0
}

func (pv *PreferencesView) getSchemeIndex(scheme string) int {
	schemes := []string{"default", "solarized", "monokai", "dracula", "nord"}
	for i, s := range schemes {
		if s == scheme {
			return i
		}
	}
	return 0
}

func (pv *PreferencesView) getColorBlindIndex(mode string) int {
	modes := []string{"", "protanopia", "deuteranopia", "tritanopia"}
	for i, m := range modes {
		if m == mode {
			return i
		}
	}
	return 0
}

func (pv *PreferencesView) getOperatorIndex(op string) int {
	operators := []string{"contains", "equals", "starts with", "ends with", "regex"}
	for i, o := range operators {
		if o == op {
			return i
		}
	}
	return 0
}

func (pv *PreferencesView) getBadgePositionIndex(pos string) int {
	positions := []string{"top-right", "top-left", "bottom-right", "bottom-left"}
	for i, p := range positions {
		if p == pos {
			return i
		}
	}
	return 0
}

func (pv *PreferencesView) getSortOrderIndex(order string) int {
	if order == "desc" {
		return 1
	}
	return 0
}

// showError shows an error message
func (pv *PreferencesView) showError(message string) {
	modal := tview.NewModal().
		SetText(message).
		AddButtons([]string{"OK"}).
		SetDoneFunc(func(_ int, _ string) {
			pv.pages.RemovePage("error")
		})
	modal.SetTextColor(tcell.ColorRed)
	pv.pages.AddPage("error", modal, true, true)
}

// GetView returns the preferences view
func (pv *PreferencesView) GetView() tview.Primitive {
	return pv.layout
}

// SetOnSave sets the save callback
func (pv *PreferencesView) SetOnSave(fn func()) {
	pv.onSave = fn
}

// SetOnCancel sets the cancel callback
func (pv *PreferencesView) SetOnCancel(fn func()) {
	pv.onCancel = fn
}

// ShowPreferences displays the preferences modal
func ShowPreferences(pages *tview.Pages, app *tview.Application, prefs *preferences.UserPreferences) {
	prefsView := NewPreferencesView(prefs, app, pages)

	// Create modal layout
	modal := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().
			AddItem(nil, 0, 1, false).
			AddItem(prefsView.GetView(), 100, 0, true).
			AddItem(nil, 0, 1, false), 0, 8, true).
		AddItem(nil, 0, 1, false)

	modal.SetBorder(true).
		SetTitle(" User Preferences ").
		SetTitleAlign(tview.AlignCenter)

	pages.AddPage("preferences", modal, true, true)
}
