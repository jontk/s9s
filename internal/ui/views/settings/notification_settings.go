package settings

import (
	"fmt"
	"strconv"

	"github.com/gdamore/tcell/v2"
	"github.com/jontk/s9s/internal/notifications"
	"github.com/jontk/s9s/internal/ui/components"
	"github.com/rivo/tview"
)

// NotificationSettingsView displays notification configuration
type NotificationSettingsView struct {
	manager interface{} // *notifications.NotificationManager
	form    *tview.Form
	pages   *tview.Pages
	app     *tview.Application
	onSave  func()
}

// NewNotificationSettingsView creates a new notification settings view
func NewNotificationSettingsView(manager interface{}, app *tview.Application, pages *tview.Pages) *NotificationSettingsView {
	v := &NotificationSettingsView{
		manager: manager,
		app:     app,
		pages:   pages,
		form:    tview.NewForm(),
	}

	v.buildForm()

	return v
}

// buildForm builds the settings form
func (v *NotificationSettingsView) buildForm() {
	// Get current config via type assertion
	var config *notifications.NotificationConfig
	if mgr, ok := v.manager.(*notifications.NotificationManager); ok {
		config = mgr.GetConfig()
	} else {
		// Default config if manager is nil
		config = &notifications.NotificationConfig{
			EnableNotifications: true,
			MinAlertLevel:       int(components.AlertWarning),
			TerminalBell: notifications.TerminalBellConfig{
				Enabled:       true,
				MinAlertLevel: int(components.AlertError),
				RepeatCount:   2,
			},
			LogFile: notifications.LogFileConfig{
				Enabled: true,
			},
			DesktopNotify: notifications.DesktopNotifyConfig{
				Enabled:       false,
				MinAlertLevel: int(components.AlertError),
				Timeout:       10,
			},
			Webhook: notifications.WebhookConfig{
				Enabled:       false,
				MinAlertLevel: int(components.AlertCritical),
			},
		}
	}

	v.form.Clear(true)

	// Global settings
	v.form.AddCheckbox("Enable Notifications", config.EnableNotifications, nil)

	alertLevels := []string{"Info", "Warning", "Error", "Critical"}
	v.form.AddDropDown("Minimum Alert Level", alertLevels, config.MinAlertLevel, nil)

	// Terminal Bell settings
	v.form.AddTextView("", "[yellow]Terminal Bell Settings[white]", 0, 1, true, false)
	v.form.AddCheckbox("Enable Terminal Bell", config.TerminalBell.Enabled, nil)
	v.form.AddDropDown("Bell Min Alert Level", alertLevels, config.TerminalBell.MinAlertLevel, nil)
	v.form.AddInputField("Bell Repeat Count", strconv.Itoa(config.TerminalBell.RepeatCount), 10, nil, nil)

	// Log File settings
	v.form.AddTextView("", "[yellow]Log File Settings[white]", 0, 1, true, false)
	v.form.AddCheckbox("Enable Log File", config.LogFile.Enabled, nil)
	v.form.AddInputField("Log Path", config.LogFile.LogPath, 50, nil, nil)

	// Desktop Notification settings
	v.form.AddTextView("", "[yellow]Desktop Notification Settings[white]", 0, 1, true, false)
	v.form.AddCheckbox("Enable Desktop Notifications", config.DesktopNotify.Enabled, nil)
	v.form.AddDropDown("Desktop Min Alert Level", alertLevels, config.DesktopNotify.MinAlertLevel, nil)
	v.form.AddInputField("Notification Timeout (sec)", strconv.Itoa(config.DesktopNotify.Timeout), 10, nil, nil)

	// Webhook settings
	v.form.AddTextView("", "[yellow]Webhook Settings[white]", 0, 1, true, false)
	v.form.AddCheckbox("Enable Webhook", config.Webhook.Enabled, nil)
	v.form.AddInputField("Webhook URL", config.Webhook.URL, 50, nil, nil)
	v.form.AddDropDown("Webhook Min Alert Level", alertLevels, config.Webhook.MinAlertLevel, nil)

	// Buttons
	v.form.AddButton("Save", v.save)
	v.form.AddButton("Cancel", v.cancel)
	v.form.AddButton("Test Notification", v.testNotification)

	// Set form properties
	v.form.SetBorder(true).
		SetTitle(" Notification Settings ").
		SetTitleAlign(tview.AlignCenter)

	// Handle escape key
	v.form.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEsc {
			v.cancel()
			return nil
		}
		return event
	})
}

// save saves the notification settings
func (v *NotificationSettingsView) save() {
	if mgr, ok := v.manager.(*notifications.NotificationManager); ok {
		// Create new config from form values
		config := &notifications.NotificationConfig{}

		// Global settings
		config.EnableNotifications = v.form.GetFormItemByLabel("Enable Notifications").(*tview.Checkbox).IsChecked()
		minLevel, _ := v.form.GetFormItemByLabel("Minimum Alert Level").(*tview.DropDown).GetCurrentOption()
		config.MinAlertLevel = minLevel

		// Terminal Bell
		config.TerminalBell.Enabled = v.form.GetFormItemByLabel("Enable Terminal Bell").(*tview.Checkbox).IsChecked()
		bellMinLevel, _ := v.form.GetFormItemByLabel("Bell Min Alert Level").(*tview.DropDown).GetCurrentOption()
		config.TerminalBell.MinAlertLevel = bellMinLevel
		repeatStr := v.form.GetFormItemByLabel("Bell Repeat Count").(*tview.InputField).GetText()
		config.TerminalBell.RepeatCount, _ = strconv.Atoi(repeatStr)

		// Log File
		config.LogFile.Enabled = v.form.GetFormItemByLabel("Enable Log File").(*tview.Checkbox).IsChecked()
		config.LogFile.LogPath = v.form.GetFormItemByLabel("Log Path").(*tview.InputField).GetText()

		// Desktop Notifications
		config.DesktopNotify.Enabled = v.form.GetFormItemByLabel("Enable Desktop Notifications").(*tview.Checkbox).IsChecked()
		desktopMinLevel, _ := v.form.GetFormItemByLabel("Desktop Min Alert Level").(*tview.DropDown).GetCurrentOption()
		config.DesktopNotify.MinAlertLevel = desktopMinLevel
		timeoutStr := v.form.GetFormItemByLabel("Notification Timeout (sec)").(*tview.InputField).GetText()
		config.DesktopNotify.Timeout, _ = strconv.Atoi(timeoutStr)

		// Webhook
		config.Webhook.Enabled = v.form.GetFormItemByLabel("Enable Webhook").(*tview.Checkbox).IsChecked()
		config.Webhook.URL = v.form.GetFormItemByLabel("Webhook URL").(*tview.InputField).GetText()
		webhookMinLevel, _ := v.form.GetFormItemByLabel("Webhook Min Alert Level").(*tview.DropDown).GetCurrentOption()
		config.Webhook.MinAlertLevel = webhookMinLevel

		// Update config
		if err := mgr.UpdateConfig(config); err != nil {
			v.showMessage(fmt.Sprintf("Failed to save settings: %v", err), tcell.ColorRed)
		} else {
			v.showMessage("Settings saved successfully", tcell.ColorGreen)
			if v.onSave != nil {
				v.onSave()
			}
		}
	}
}

// cancel closes the settings view
func (v *NotificationSettingsView) cancel() {
	if v.pages != nil {
		v.pages.RemovePage("notification-settings")
	}
}

// testNotification sends a test notification
func (v *NotificationSettingsView) testNotification() {
	// Create a test alert
	testAlert := &components.Alert{
		Level:   components.AlertWarning,
		Title:   "Test Notification",
		Message: "This is a test notification from S9S notification settings",
		Source:  "test",
	}

	// Send through notification manager
	if mgr, ok := v.manager.(*notifications.NotificationManager); ok {
		mgr.Notify(testAlert)
		v.showMessage("Test notification sent!", tcell.ColorGreen)
	}
}

// showMessage displays a temporary message
func (v *NotificationSettingsView) showMessage(message string, color tcell.Color) {
	modal := tview.NewModal().
		SetText(message).
		AddButtons([]string{"OK"}).
		SetDoneFunc(func(buttonIndex int, buttonLabel string) {
			v.pages.RemovePage("message")
		})

	modal.SetTextColor(color)
	v.pages.AddPage("message", modal, true, true)
}

// SetOnSave sets the callback for when settings are saved
func (v *NotificationSettingsView) SetOnSave(handler func()) {
	v.onSave = handler
}

// GetView returns the settings form
func (v *NotificationSettingsView) GetView() tview.Primitive {
	return v.form
}

// ShowNotificationSettings displays the notification settings modal
func ShowNotificationSettings(pages *tview.Pages, app *tview.Application, notificationMgr interface{}) {
	settingsView := NewNotificationSettingsView(notificationMgr, app, pages)

	// Create modal layout
	modal := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().
			AddItem(nil, 0, 1, false).
			AddItem(settingsView.GetView(), 80, 0, true).
			AddItem(nil, 0, 1, false), 0, 3, true).
		AddItem(nil, 0, 1, false)

	pages.AddPage("notification-settings", modal, true, true)
}
