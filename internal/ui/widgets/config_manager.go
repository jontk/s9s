package widgets

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/jontk/s9s/internal/config"
	"github.com/rivo/tview"
)

// ConfigManager provides a comprehensive configuration management interface
type ConfigManager struct {
	*tview.Flex
	app   *tview.Application
	pages *tview.Pages

	// UI Components
	sidebar   *tview.List
	content   *tview.Flex
	form      *tview.Form
	statusBar *tview.TextView

	// Data
	schema           *config.ConfigSchema
	currentConfig    *config.Config
	originalConfig   *config.Config
	selectedGroup    string
	configPath       string
	hasChanges       bool
	validationErrors map[string]config.ValidationResult

	// Callbacks
	onSave   func(*config.Config) error
	onCancel func()
	onApply  func(*config.Config) error
}

// NewConfigManager creates a new configuration manager
func NewConfigManager(app *tview.Application, configPath string) *ConfigManager {
	cm := &ConfigManager{
		Flex:             tview.NewFlex(),
		app:              app,
		schema:           config.GetConfigSchema(),
		configPath:       configPath,
		validationErrors: make(map[string]config.ValidationResult),
	}

	cm.initializeUI()
	cm.loadConfiguration()

	return cm
}

// initializeUI creates the configuration management interface
func (cm *ConfigManager) initializeUI() {
	// Create sidebar for group navigation
	cm.sidebar = tview.NewList()
	cm.sidebar.SetBorder(true)
	cm.sidebar.SetTitle(" Configuration Groups ")
	cm.sidebar.SetTitleAlign(tview.AlignCenter)
	cm.sidebar.ShowSecondaryText(false)

	// Populate sidebar with groups
	for _, group := range cm.schema.Groups {
		title := fmt.Sprintf("%s %s", group.Icon, group.Name)
		cm.sidebar.AddItem(title, group.Description, 0, nil)
	}

	// Set sidebar selection handler
	cm.sidebar.SetSelectedFunc(func(index int, mainText, secondaryText string, shortcut rune) {
		if index < len(cm.schema.Groups) {
			cm.selectGroup(cm.schema.Groups[index].ID)
		}
	})

	// Create content area
	cm.content = tview.NewFlex().SetDirection(tview.FlexRow)
	cm.content.SetBorder(true)
	cm.content.SetTitle(" Settings ")
	cm.content.SetTitleAlign(tview.AlignCenter)

	// Create form for configuration fields
	cm.form = tview.NewForm()
	cm.form.SetFieldBackgroundColor(tcell.ColorDefault)
	cm.form.SetFieldTextColor(tcell.ColorWhite)
	cm.form.SetLabelColor(tcell.ColorYellow)
	cm.form.SetButtonBackgroundColor(tcell.ColorNavy)
	cm.form.SetButtonTextColor(tcell.ColorWhite)

	// Create status bar
	cm.statusBar = tview.NewTextView()
	cm.statusBar.SetDynamicColors(true)
	cm.statusBar.SetTextAlign(tview.AlignCenter)
	cm.statusBar.SetText("[gray]Select a configuration group to begin[white]")

	// Add action buttons
	cm.addActionButtons()

	// Layout
	cm.content.AddItem(cm.form, 0, 1, true)
	cm.content.AddItem(cm.statusBar, 2, 0, false)

	cm.SetDirection(tview.FlexColumn)
	cm.AddItem(cm.sidebar, 30, 0, false)
	cm.AddItem(cm.content, 0, 1, true)

	// Set input handling
	cm.SetInputCapture(cm.handleInput)

	// Select first group by default
	if len(cm.schema.Groups) > 0 {
		cm.selectGroup(cm.schema.Groups[0].ID)
		cm.sidebar.SetCurrentItem(0)
	}
}

// addActionButtons adds save, cancel, and apply buttons to the form
func (cm *ConfigManager) addActionButtons() {
	cm.form.AddButton("Save", func() {
		cm.saveConfiguration()
	})

	cm.form.AddButton("Apply", func() {
		cm.applyConfiguration()
	})

	cm.form.AddButton("Reset", func() {
		cm.resetToDefaults()
	})

	cm.form.AddButton("Cancel", func() {
		cm.cancelChanges()
	})
}

// loadConfiguration loads the current configuration
func (cm *ConfigManager) loadConfiguration() {
	var err error
	if cm.configPath != "" {
		cm.currentConfig, err = config.LoadWithPath(cm.configPath)
	} else {
		cm.currentConfig, err = config.Load()
	}

	if err != nil {
		cm.updateStatusBar(fmt.Sprintf("[red]Error loading configuration: %v[white]", err))
		// Create a default configuration
		cm.currentConfig = &config.Config{}
	}

	// Make a copy of the original configuration
	cm.originalConfig = cm.copyConfig(cm.currentConfig)
	cm.hasChanges = false
}

// selectGroup switches to a specific configuration group
func (cm *ConfigManager) selectGroup(groupID string) {
	cm.selectedGroup = groupID
	cm.buildForm()

	// Update content title
	for _, group := range cm.schema.Groups {
		if group.ID == groupID {
			cm.content.SetTitle(fmt.Sprintf(" %s %s ", group.Icon, group.Name))
			break
		}
	}
}

// buildForm creates the form for the selected group
func (cm *ConfigManager) buildForm() {
	cm.form.Clear(true)

	// Get fields for the selected group
	fields := cm.schema.GetFieldsByGroup(cm.selectedGroup)

	if len(fields) == 0 {
		cm.form.AddTextView("No Settings", "No configurable settings in this group.", 0, 1, false, false)
		cm.addActionButtons()
		return
	}

	// Sort fields by order
	for i := 0; i < len(fields); i++ {
		for j := i + 1; j < len(fields); j++ {
			if fields[i].Order > fields[j].Order {
				fields[i], fields[j] = fields[j], fields[i]
			}
		}
	}

	// Add form fields
	for _, field := range fields {
		cm.addFormField(field)
	}

	cm.addActionButtons()
	cm.validateAllFields()
}

// addFormField adds a single field to the form
func (cm *ConfigManager) addFormField(field config.ConfigField) {
	currentValue := cm.getConfigValue(field.Key)

	// Create field label with description
	label := field.Label
	if field.Required {
		label += "*"
	}

	switch field.Type {
	case config.FieldTypeString:
		initialValue := ""
		if currentValue != nil {
			initialValue = fmt.Sprintf("%v", currentValue)
		}

		if field.Sensitive {
			cm.form.AddPasswordField(label, initialValue, 0, '*', func(text string) {
				cm.setConfigValue(field.Key, text)
				cm.validateField(field, text)
			})
		} else {
			cm.form.AddInputField(label, initialValue, 0, nil, func(text string) {
				cm.setConfigValue(field.Key, text)
				cm.validateField(field, text)
			})
		}

		// Add field description as a tooltip-like behavior
		if field.Description != "" {
			cm.addFieldDescription(field.Description)
		}

	case config.FieldTypeInt:
		initialValue := ""
		if currentValue != nil {
			initialValue = fmt.Sprintf("%v", currentValue)
		}

		cm.form.AddInputField(label, initialValue, 0, func(textToCheck string, lastChar rune) bool {
			// Only allow digits and minus sign
			return lastChar >= '0' && lastChar <= '9' || lastChar == '-'
		}, func(text string) {
			if text != "" {
				if val, err := strconv.Atoi(text); err == nil {
					cm.setConfigValue(field.Key, val)
					cm.validateField(field, val)
				}
			}
		})

		if field.Description != "" {
			cm.addFieldDescription(field.Description)
		}

	case config.FieldTypeBool:
		initialValue := false
		if currentValue != nil {
			if val, ok := currentValue.(bool); ok {
				initialValue = val
			}
		}

		cm.form.AddCheckbox(label, initialValue, func(checked bool) {
			cm.setConfigValue(field.Key, checked)
			cm.validateField(field, checked)
		})

		if field.Description != "" {
			cm.addFieldDescription(field.Description)
		}

	case config.FieldTypeSelect:
		currentIndex := 0
		currentStr := ""
		if currentValue != nil {
			currentStr = fmt.Sprintf("%v", currentValue)
			for i, option := range field.Options {
				if option == currentStr {
					currentIndex = i
					break
				}
			}
		}

		cm.form.AddDropDown(label, field.Options, currentIndex, func(text string, index int) {
			cm.setConfigValue(field.Key, text)
			cm.validateField(field, text)
		})

		if field.Description != "" {
			cm.addFieldDescription(field.Description)
		}

	case config.FieldTypeArray:
		initialValue := ""
		if currentValue != nil {
			if arr, ok := currentValue.([]interface{}); ok {
				var strArr []string
				for _, v := range arr {
					strArr = append(strArr, fmt.Sprintf("%v", v))
				}
				initialValue = strings.Join(strArr, ", ")
			} else if arr, ok := currentValue.([]string); ok {
				initialValue = strings.Join(arr, ", ")
			}
		}

		cm.form.AddInputField(label, initialValue, 0, nil, func(text string) {
			if text == "" {
				cm.setConfigValue(field.Key, []string{})
			} else {
				parts := strings.Split(text, ",")
				var trimmed []string
				for _, part := range parts {
					trimmed = append(trimmed, strings.TrimSpace(part))
				}
				cm.setConfigValue(field.Key, trimmed)
				cm.validateField(field, trimmed)
			}
		})

		description := field.Description
		if len(field.Options) > 0 {
			description += fmt.Sprintf(" (Options: %s)", strings.Join(field.Options, ", "))
		}
		cm.addFieldDescription(description + " - Separate multiple values with commas")

	case config.FieldTypeDuration:
		initialValue := ""
		if currentValue != nil {
			initialValue = fmt.Sprintf("%v", currentValue)
		}

		cm.form.AddInputField(label, initialValue, 0, nil, func(text string) {
			cm.setConfigValue(field.Key, text)
			cm.validateField(field, text)
		})

		description := field.Description + " (e.g., 1s, 5m, 1h30m)"
		if len(field.Examples) > 0 {
			description += fmt.Sprintf(" Examples: %s", strings.Join(field.Examples, ", "))
		}
		cm.addFieldDescription(description)

	case config.FieldTypeContext:
		cm.addContextField(field)

	case config.FieldTypeShortcut:
		cm.addShortcutField(field)

	case config.FieldTypeAlias:
		cm.addAliasField(field)

	case config.FieldTypePlugin:
		cm.addPluginField(field)
	}
}

// addFieldDescription adds a description text view after a field
func (cm *ConfigManager) addFieldDescription(description string) {
	cm.form.AddTextView("", fmt.Sprintf("[gray]%s[white]", description), 0, 2, false, false)
}

// getConfigValue retrieves a configuration value by key path
func (cm *ConfigManager) getConfigValue(key string) interface{} {
	parts := strings.Split(key, ".")
	return cm.getNestedValue(cm.currentConfig, parts)
}

// setConfigValue sets a configuration value by key path
func (cm *ConfigManager) setConfigValue(key string, value interface{}) {
	parts := strings.Split(key, ".")
	cm.setNestedValue(cm.currentConfig, parts, value)
	cm.hasChanges = true
	cm.updateStatusBar("[yellow]Configuration modified - remember to save changes[white]")
}

// getNestedValue retrieves a nested value from a struct using reflection-like path traversal
func (cm *ConfigManager) getNestedValue(obj interface{}, path []string) interface{} {
	if len(path) == 0 {
		return obj
	}

	cfg, ok := obj.(*config.Config)
	if !ok {
		return nil
	}

	switch path[0] {
	case "refreshRate":
		return cfg.RefreshRate
	case "maxRetries":
		return cfg.MaxRetries
	case "currentContext":
		return cfg.CurrentContext
	case "useMockClient":
		return cfg.UseMockClient
	case "ui":
		if len(path) == 1 {
			return cfg.UI
		}
		return cm.getUIValue(&cfg.UI, path[1:])
	case "views":
		if len(path) == 1 {
			return cfg.Views
		}
		return cm.getViewsValue(&cfg.Views, path[1:])
	case "features":
		if len(path) == 1 {
			return cfg.Features
		}
		return cm.getFeaturesValue(&cfg.Features, path[1:])
	}

	return nil
}

// setNestedValue sets a nested value in a struct
func (cm *ConfigManager) setNestedValue(obj interface{}, path []string, value interface{}) {
	if len(path) == 0 {
		return
	}

	cfg, ok := obj.(*config.Config)
	if !ok {
		return
	}

	switch path[0] {
	case "refreshRate":
		if v, ok := value.(string); ok {
			cfg.RefreshRate = v
		}
	case "maxRetries":
		if v, ok := value.(int); ok {
			cfg.MaxRetries = v
		}
	case "currentContext":
		if v, ok := value.(string); ok {
			cfg.CurrentContext = v
		}
	case "useMockClient":
		if v, ok := value.(bool); ok {
			cfg.UseMockClient = v
		}
	case "ui":
		cm.setUIValue(&cfg.UI, path[1:], value)
	case "views":
		cm.setViewsValue(&cfg.Views, path[1:], value)
	case "features":
		cm.setFeaturesValue(&cfg.Features, path[1:], value)
	}
}

// Helper methods for nested struct access
func (cm *ConfigManager) getUIValue(ui *config.UIConfig, path []string) interface{} {
	if len(path) == 0 {
		return ui
	}

	switch path[0] {
	case "skin":
		return ui.Skin
	case "enableMouse":
		return ui.EnableMouse
	case "logoless":
		return ui.Logoless
	case "statusless":
		return ui.Statusless
	case "noIcons":
		return ui.NoIcons
	}
	return nil
}

func (cm *ConfigManager) setUIValue(ui *config.UIConfig, path []string, value interface{}) {
	if len(path) == 0 {
		return
	}

	switch path[0] {
	case "skin":
		if v, ok := value.(string); ok {
			ui.Skin = v
		}
	case "enableMouse":
		if v, ok := value.(bool); ok {
			ui.EnableMouse = v
		}
	case "logoless":
		if v, ok := value.(bool); ok {
			ui.Logoless = v
		}
	case "statusless":
		if v, ok := value.(bool); ok {
			ui.Statusless = v
		}
	case "noIcons":
		if v, ok := value.(bool); ok {
			ui.NoIcons = v
		}
	}
}

func (cm *ConfigManager) getViewsValue(views *config.ViewsConfig, path []string) interface{} {
	if len(path) < 2 {
		return nil
	}

	switch path[0] {
	case "jobs":
		return cm.getJobsViewValue(&views.Jobs, path[1:])
	case "nodes":
		return cm.getNodesViewValue(&views.Nodes, path[1:])
	}
	return nil
}

func (cm *ConfigManager) setViewsValue(views *config.ViewsConfig, path []string, value interface{}) {
	if len(path) < 2 {
		return
	}

	switch path[0] {
	case "jobs":
		cm.setJobsViewValue(&views.Jobs, path[1:], value)
	case "nodes":
		cm.setNodesViewValue(&views.Nodes, path[1:], value)
	}
}

func (cm *ConfigManager) getJobsViewValue(jobs *config.JobsViewConfig, path []string) interface{} {
	if len(path) == 0 {
		return jobs
	}

	switch path[0] {
	case "columns":
		return jobs.Columns
	case "showOnlyActive":
		return jobs.ShowOnlyActive
	case "defaultSort":
		return jobs.DefaultSort
	case "maxJobs":
		return jobs.MaxJobs
	}
	return nil
}

func (cm *ConfigManager) setJobsViewValue(jobs *config.JobsViewConfig, path []string, value interface{}) {
	if len(path) == 0 {
		return
	}

	switch path[0] {
	case "columns":
		if v, ok := value.([]string); ok {
			jobs.Columns = v
		}
	case "showOnlyActive":
		if v, ok := value.(bool); ok {
			jobs.ShowOnlyActive = v
		}
	case "defaultSort":
		if v, ok := value.(string); ok {
			jobs.DefaultSort = v
		}
	case "maxJobs":
		if v, ok := value.(int); ok {
			jobs.MaxJobs = v
		}
	}
}

func (cm *ConfigManager) getNodesViewValue(nodes *config.NodesViewConfig, path []string) interface{} {
	if len(path) == 0 {
		return nodes
	}

	switch path[0] {
	case "groupBy":
		return nodes.GroupBy
	case "showUtilization":
		return nodes.ShowUtilization
	}
	return nil
}

func (cm *ConfigManager) setNodesViewValue(nodes *config.NodesViewConfig, path []string, value interface{}) {
	if len(path) == 0 {
		return
	}

	switch path[0] {
	case "groupBy":
		if v, ok := value.(string); ok {
			nodes.GroupBy = v
		}
	case "showUtilization":
		if v, ok := value.(bool); ok {
			nodes.ShowUtilization = v
		}
	}
}

func (cm *ConfigManager) getFeaturesValue(features *config.FeaturesConfig, path []string) interface{} {
	if len(path) == 0 {
		return features
	}

	switch path[0] {
	case "streaming":
		return features.Streaming
	case "pulseye":
		return features.Pulseye
	case "xray":
		return features.Xray
	}
	return nil
}

func (cm *ConfigManager) setFeaturesValue(features *config.FeaturesConfig, path []string, value interface{}) {
	if len(path) == 0 {
		return
	}

	switch path[0] {
	case "streaming":
		if v, ok := value.(bool); ok {
			features.Streaming = v
		}
	case "pulseye":
		if v, ok := value.(bool); ok {
			features.Pulseye = v
		}
	case "xray":
		if v, ok := value.(bool); ok {
			features.Xray = v
		}
	}
}

// validateField validates a single field and updates the UI
func (cm *ConfigManager) validateField(field config.ConfigField, value interface{}) {
	result := field.ValidateField(value)
	cm.validationErrors[field.Key] = result

	if !result.Valid {
		cm.updateStatusBar(fmt.Sprintf("[red]%s: %s[white]", field.Label, strings.Join(result.Errors, ", ")))
	} else if cm.hasChanges {
		cm.updateStatusBar("[yellow]Configuration modified - remember to save changes[white]")
	}
}

// validateAllFields validates all fields in the current group
func (cm *ConfigManager) validateAllFields() {
	fields := cm.schema.GetFieldsByGroup(cm.selectedGroup)
	hasErrors := false

	for _, field := range fields {
		value := cm.getConfigValue(field.Key)
		result := field.ValidateField(value)
		cm.validationErrors[field.Key] = result

		if !result.Valid {
			hasErrors = true
		}
	}

	if hasErrors {
		cm.updateStatusBar("[red]Some fields have validation errors[white]")
	}
}

// saveConfiguration saves the current configuration to file
func (cm *ConfigManager) saveConfiguration() {
	// Validate all configuration
	allResults := cm.schema.ValidateConfig(cm.currentConfig)
	hasErrors := false

	for _, result := range allResults {
		if !result.Valid {
			hasErrors = true
			break
		}
	}

	if hasErrors {
		cm.updateStatusBar("[red]Cannot save: configuration has validation errors[white]")
		return
	}

	// Determine save path
	savePath := cm.configPath
	if savePath == "" {
		homeDir, _ := os.UserHomeDir()
		savePath = filepath.Join(homeDir, ".s9s", "config.yaml")
	}

	// Save configuration
	if err := cm.currentConfig.SaveToFile(savePath); err != nil {
		cm.updateStatusBar(fmt.Sprintf("[red]Error saving configuration: %v[white]", err))
		return
	}

	// Update original config and reset change tracking
	cm.originalConfig = cm.copyConfig(cm.currentConfig)
	cm.hasChanges = false

	cm.updateStatusBar(fmt.Sprintf("[green]Configuration saved to %s[white]", savePath))

	// Call save callback if set
	if cm.onSave != nil {
		_ = cm.onSave(cm.currentConfig)
	}
}

// applyConfiguration applies configuration without saving to file
func (cm *ConfigManager) applyConfiguration() {
	// Validate configuration
	allResults := cm.schema.ValidateConfig(cm.currentConfig)
	hasErrors := false

	for _, result := range allResults {
		if !result.Valid {
			hasErrors = true
			break
		}
	}

	if hasErrors {
		cm.updateStatusBar("[red]Cannot apply: configuration has validation errors[white]")
		return
	}

	cm.updateStatusBar("[green]Configuration applied[white]")

	// Call apply callback if set
	if cm.onApply != nil {
		_ = cm.onApply(cm.currentConfig)
	}
}

// resetToDefaults resets the current group to default values
func (cm *ConfigManager) resetToDefaults() {
	fields := cm.schema.GetFieldsByGroup(cm.selectedGroup)

	for _, field := range fields {
		if field.Default != nil {
			cm.setConfigValue(field.Key, field.Default)
		}
	}

	cm.buildForm()
	cm.updateStatusBar("[yellow]Reset to defaults - remember to save changes[white]")
}

// cancelChanges reverts all changes
func (cm *ConfigManager) cancelChanges() {
	if cm.hasChanges {
		cm.currentConfig = cm.copyConfig(cm.originalConfig)
		cm.hasChanges = false
		cm.buildForm()
		cm.updateStatusBar("[gray]Changes cancelled[white]")
	}

	if cm.onCancel != nil {
		cm.onCancel()
	}
}

// copyConfig creates a deep copy of the configuration
func (cm *ConfigManager) copyConfig(original *config.Config) *config.Config {
	if original == nil {
		return &config.Config{}
	}

	// Create a new config and copy values
	copy := &config.Config{
		RefreshRate:    original.RefreshRate,
		MaxRetries:     original.MaxRetries,
		CurrentContext: original.CurrentContext,
		UI:             original.UI,
		Views:          original.Views,
		Features:       original.Features,
		UseMockClient:  original.UseMockClient,
		Cluster:        original.Cluster,
	}

	// Copy contexts slice
	copy.Contexts = make([]config.ContextConfig, len(original.Contexts))
	for i, ctx := range original.Contexts {
		copy.Contexts[i] = ctx
	}

	// Copy shortcuts slice
	copy.Shortcuts = make([]config.ShortcutConfig, len(original.Shortcuts))
	for i, shortcut := range original.Shortcuts {
		copy.Shortcuts[i] = shortcut
	}

	// Copy aliases map
	if original.Aliases != nil {
		copy.Aliases = make(map[string]string)
		for k, v := range original.Aliases {
			copy.Aliases[k] = v
		}
	}

	// Copy plugins slice
	copy.Plugins = make([]config.PluginConfig, len(original.Plugins))
	for i, plugin := range original.Plugins {
		copy.Plugins[i] = plugin
	}

	return copy
}

// updateStatusBar updates the status bar text
func (cm *ConfigManager) updateStatusBar(text string) {
	cm.statusBar.SetText(text)
}

// handleInput processes keyboard input
func (cm *ConfigManager) handleInput(event *tcell.EventKey) *tcell.EventKey {
	switch event.Key() {
	case tcell.KeyCtrlS:
		cm.saveConfiguration()
		return nil
	case tcell.KeyCtrlZ:
		cm.cancelChanges()
		return nil
	case tcell.KeyF5:
		cm.resetToDefaults()
		return nil
	case tcell.KeyTab:
		// Switch focus between sidebar and form
		if cm.sidebar.HasFocus() {
			cm.app.SetFocus(cm.form)
		} else {
			cm.app.SetFocus(cm.sidebar)
		}
		return nil
	}

	switch event.Rune() {
	case 's':
		if event.Modifiers()&tcell.ModCtrl != 0 {
			cm.saveConfiguration()
			return nil
		}
	case 'q':
		cm.cancelChanges()
		return nil
	}

	return event
}

// SetCallbacks sets the callback functions
func (cm *ConfigManager) SetCallbacks(onSave, onApply func(*config.Config) error, onCancel func()) {
	cm.onSave = onSave
	cm.onApply = onApply
	cm.onCancel = onCancel
}

// SetPages sets the pages manager for modal display
func (cm *ConfigManager) SetPages(pages *tview.Pages) {
	cm.pages = pages
}

// GetCurrentConfig returns the current configuration
func (cm *ConfigManager) GetCurrentConfig() *config.Config {
	return cm.currentConfig
}

// HasChanges returns whether there are unsaved changes
func (cm *ConfigManager) HasChanges() bool {
	return cm.hasChanges
}

// addContextField adds a context management field
func (cm *ConfigManager) addContextField(field config.ConfigField) {
	contextCount := len(cm.currentConfig.Contexts)
	summary := fmt.Sprintf("Contexts: %d configured", contextCount)
	if cm.currentConfig.CurrentContext != "" {
		summary += fmt.Sprintf(" (Current: %s)", cm.currentConfig.CurrentContext)
	}

	cm.form.AddButton("Manage Contexts", func() {
		cm.showContextManager()
	})
	cm.addFieldDescription(summary)
}

// addShortcutField adds a shortcut management field
func (cm *ConfigManager) addShortcutField(field config.ConfigField) {
	shortcutCount := len(cm.currentConfig.Shortcuts)
	summary := fmt.Sprintf("Shortcuts: %d configured", shortcutCount)

	cm.form.AddButton("Manage Shortcuts", func() {
		cm.showShortcutManager()
	})
	cm.addFieldDescription(summary)
}

// addAliasField adds an alias management field
func (cm *ConfigManager) addAliasField(field config.ConfigField) {
	aliasCount := len(cm.currentConfig.Aliases)
	summary := fmt.Sprintf("Aliases: %d configured", aliasCount)

	cm.form.AddButton("Manage Aliases", func() {
		cm.showAliasManager()
	})
	cm.addFieldDescription(summary)
}

// addPluginField adds a plugin management field
func (cm *ConfigManager) addPluginField(field config.ConfigField) {
	pluginCount := len(cm.currentConfig.Plugins)
	summary := fmt.Sprintf("Plugins: %d configured", pluginCount)

	cm.form.AddButton("Manage Plugins", func() {
		cm.showPluginManager()
	})
	cm.addFieldDescription(summary)
}

// showContextManager shows a modal for managing contexts
func (cm *ConfigManager) showContextManager() {
	if cm.pages == nil {
		cm.updateStatusBar("[red]Context manager not available - no pages manager set[white]")
		return
	}

	modal := tview.NewModal()
	modal.SetText("Context Manager\n(Implementation pending)")
	modal.AddButtons([]string{"Close"})
	modal.SetDoneFunc(func(buttonIndex int, buttonLabel string) {
		cm.pages.RemovePage("context-modal")
		cm.app.SetFocus(cm)
	})

	_ = cm.pages.AddPage("context-modal", modal, false, true)
}

// showShortcutManager shows a modal for managing shortcuts
func (cm *ConfigManager) showShortcutManager() {
	if cm.pages == nil {
		cm.updateStatusBar("[red]Shortcut manager not available - no pages manager set[white]")
		return
	}

	modal := tview.NewModal()
	modal.SetText("Shortcut Manager\n(Implementation pending)")
	modal.AddButtons([]string{"Close"})
	modal.SetDoneFunc(func(buttonIndex int, buttonLabel string) {
		cm.pages.RemovePage("shortcut-modal")
		cm.app.SetFocus(cm)
	})

	_ = cm.pages.AddPage("shortcut-modal", modal, false, true)
}

// showAliasManager shows a modal for managing aliases
func (cm *ConfigManager) showAliasManager() {
	if cm.pages == nil {
		cm.updateStatusBar("[red]Alias manager not available - no pages manager set[white]")
		return
	}

	modal := tview.NewModal()
	modal.SetText("Alias Manager\n(Implementation pending)")
	modal.AddButtons([]string{"Close"})
	modal.SetDoneFunc(func(buttonIndex int, buttonLabel string) {
		cm.pages.RemovePage("alias-modal")
		cm.app.SetFocus(cm)
	})

	_ = cm.pages.AddPage("alias-modal", modal, false, true)
}

// showPluginManager shows a modal for managing plugins
func (cm *ConfigManager) showPluginManager() {
	if cm.pages == nil {
		cm.updateStatusBar("[red]Plugin manager not available - no pages manager set[white]")
		return
	}

	modal := tview.NewModal()
	modal.SetText("Plugin Manager\n(Implementation pending)")
	modal.AddButtons([]string{"Close"})
	modal.SetDoneFunc(func(buttonIndex int, buttonLabel string) {
		cm.pages.RemovePage("plugin-modal")
		cm.app.SetFocus(cm)
	})

	_ = cm.pages.AddPage("plugin-modal", modal, false, true)
}
