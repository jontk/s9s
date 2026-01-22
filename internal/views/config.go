package views

import (
	"path/filepath"

	"github.com/gdamore/tcell/v2"
	"github.com/jontk/s9s/internal/config"
	"github.com/jontk/s9s/internal/ui/widgets"
	"github.com/rivo/tview"
)

// ConfigView provides a configuration management interface
type ConfigView struct {
	*tview.Flex
	app           *tview.Application
	pages         *tview.Pages
	configManager *widgets.ConfigManager

	// State
	configPath    string
	originalTitle string
	hasChanges    bool

	// Callbacks
	onConfigChanged func(*config.Config)
}

// NewConfigView creates a new configuration view
func NewConfigView(app *tview.Application, pages *tview.Pages, configPath string) *ConfigView {
	cv := &ConfigView{
		Flex:       tview.NewFlex(),
		app:        app,
		pages:      pages,
		configPath: configPath,
	}

	cv.setupView()
	return cv
}

// setupView initializes the configuration view
func (cv *ConfigView) setupView() {
	// Create config manager
	cv.configManager = widgets.NewConfigManager(cv.app, cv.configPath)
	cv.configManager.SetPages(cv.pages)

	// Set up callbacks
	cv.configManager.SetCallbacks(
		cv.onSave,
		cv.onApply,
		cv.onCancel,
	)

	// Track changes for title updates
	cv.originalTitle = "Configuration"
	cv.hasChanges = false

	// Add the config manager to the view
	cv.AddItem(cv.configManager, 0, 1, true)

	// Set up input handling
	cv.SetInputCapture(cv.handleInput)
}

// onSave handles configuration saving
func (cv *ConfigView) onSave(cfg *config.Config) error {
	// Notify parent of config change
	if cv.onConfigChanged != nil {
		cv.onConfigChanged(cfg)
	}

	cv.hasChanges = false
	cv.updateTitle()
	return nil
}

// onApply handles configuration application without saving
func (cv *ConfigView) onApply(cfg *config.Config) error {
	// Apply configuration to running application
	if cv.onConfigChanged != nil {
		cv.onConfigChanged(cfg)
	}

	return nil
}

// onCancel handles configuration cancellation
func (cv *ConfigView) onCancel() {
	cv.hasChanges = false
	cv.updateTitle()
}

// updateTitle updates the view title to reflect change state
func (cv *ConfigView) updateTitle() {
	title := cv.originalTitle
	if cv.hasChanges {
		title += " (Modified)"
	}
	cv.SetTitle(title)
}

// handleInput processes keyboard shortcuts
func (cv *ConfigView) handleInput(event *tcell.EventKey) *tcell.EventKey {
	switch event.Key() {
	case tcell.KeyEscape:
		// Exit configuration view
		cv.exitView()
		return nil
	case tcell.KeyF1:
		// Show help
		cv.showHelp()
		return nil
	}

	// Pass through to config manager
	return event
}

// exitView handles exiting the configuration view
func (cv *ConfigView) exitView() {
	if cv.configManager.HasChanges() {
		// Show confirmation dialog
		cv.showExitConfirmation()
	} else {
		// Safe to exit
		cv.pages.RemovePage("config")
	}
}

// showExitConfirmation shows a confirmation dialog for unsaved changes
func (cv *ConfigView) showExitConfirmation() {
	modal := tview.NewModal()
	modal.SetText("You have unsaved configuration changes.\nWhat would you like to do?")
	modal.AddButtons([]string{"Save & Exit", "Discard & Exit", "Cancel"})
	modal.SetDoneFunc(func(buttonIndex int, buttonLabel string) {
		cv.pages.RemovePage("exit-confirm")

		switch buttonIndex {
		case 0: // Save & Exit
			cv.configManager.GetCurrentConfig()
			cv.pages.RemovePage("config")
		case 1: // Discard & Exit
			cv.pages.RemovePage("config")
		case 2: // Cancel
			// Do nothing, stay in config view
		}
	})

	cv.pages.AddPage("exit-confirm", modal, false, true)
}

// showHelp shows configuration help
func (cv *ConfigView) showHelp() {
	helpText := `Configuration Help

Navigation:
  Tab          - Navigate between sidebar and form
  Enter        - Select/modify field
  Space        - Toggle checkboxes
  Esc          - Exit configuration

Shortcuts:
  Ctrl+S       - Save configuration
  Ctrl+Z       - Cancel changes
  F5           - Reset current group to defaults
  F1           - Show this help

Configuration Groups:
  General      - Basic application settings
  UI           - Interface appearance
  Cluster      - Connection contexts
  Views        - Table display options
  Features     - Advanced features
  Shortcuts    - Keyboard shortcuts
  Aliases      - Command aliases
  Plugins      - External plugins

File Location:
  ~/.s9s/config.yaml

Environment Variables:
  S9S_REFRESH_RATE     - Override refresh rate
  S9S_MAX_RETRIES      - Override max retries
  S9S_CURRENT_CONTEXT  - Override current context
  SLURM_REST_URL       - Cluster endpoint
  SLURM_JWT           - Authentication token`

	modal := tview.NewModal()
	modal.SetText(helpText)
	modal.AddButtons([]string{"Close"})
	modal.SetDoneFunc(func(buttonIndex int, buttonLabel string) {
		cv.pages.RemovePage("config-help")
		cv.app.SetFocus(cv.configManager)
	})

	cv.pages.AddPage("config-help", modal, false, true)
}

// SetConfigChangedCallback sets the callback for configuration changes
func (cv *ConfigView) SetConfigChangedCallback(callback func(*config.Config)) {
	cv.onConfigChanged = callback
}

// GetConfigPath returns the configuration file path
func (cv *ConfigView) GetConfigPath() string {
	if cv.configPath != "" {
		return cv.configPath
	}

	// Return default path
	homeDir := "~" // Simplified for display
	return filepath.Join(homeDir, ".s9s", "config.yaml")
}

// GetCurrentConfig returns the current configuration
func (cv *ConfigView) GetCurrentConfig() *config.Config {
	if cv.configManager != nil {
		return cv.configManager.GetCurrentConfig()
	}
	return nil
}

// HasUnsavedChanges returns whether there are unsaved changes
func (cv *ConfigView) HasUnsavedChanges() bool {
	if cv.configManager != nil {
		return cv.configManager.HasChanges()
	}
	return false
}

// Refresh reloads the configuration
func (cv *ConfigView) Refresh() {
	// Recreate config manager to reload from file
	cv.configManager = widgets.NewConfigManager(cv.app, cv.configPath)
	cv.configManager.SetPages(cv.pages)
	cv.configManager.SetCallbacks(cv.onSave, cv.onApply, cv.onCancel)

	// Update view
	cv.Clear()
	cv.AddItem(cv.configManager, 0, 1, true)

	cv.hasChanges = false
	cv.updateTitle()
}

// Focus implements tview.Primitive
func (cv *ConfigView) Focus(delegate func(p tview.Primitive)) {
	if cv.configManager != nil {
		delegate(cv.configManager)
	}
}
