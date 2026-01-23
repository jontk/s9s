package notifications

import (
	"fmt"
	"os/exec"
	"runtime"

	"github.com/jontk/s9s/internal/ui/components"
)

// DesktopNotifyConfig holds configuration for desktop notifications
type DesktopNotifyConfig struct {
	Enabled       bool `json:"enabled"`
	MinAlertLevel int  `json:"min_alert_level"`
	Timeout       int  `json:"timeout"` // Notification timeout in seconds
}

// DesktopNotifyChannel implements desktop notifications
type DesktopNotifyChannel struct {
	config    DesktopNotifyConfig
	available bool
}

// NewDesktopNotifyChannel creates a new desktop notification channel
func NewDesktopNotifyChannel(config DesktopNotifyConfig) *DesktopNotifyChannel {
	// Set defaults
	if config.Timeout <= 0 {
		config.Timeout = 10
	}

	channel := &DesktopNotifyChannel{
		config: config,
	}

	// Check if desktop notifications are available
	channel.checkAvailability()

	return channel
}

// checkAvailability checks if desktop notifications are available on this system
func (d *DesktopNotifyChannel) checkAvailability() {
	switch runtime.GOOS {
	case "linux":
		// Check for notify-send
		if _, err := exec.LookPath("notify-send"); err == nil {
			d.available = true
		}
	case "darwin":
		// macOS always has osascript
		d.available = true
	case "windows":
		// Windows notifications require more complex setup
		// For now, mark as unavailable
		d.available = false
	default:
		d.available = false
	}
}

// Name returns the channel name
func (d *DesktopNotifyChannel) Name() string {
	return "desktop_notify"
}

// IsEnabled returns whether the channel is enabled
func (d *DesktopNotifyChannel) IsEnabled() bool {
	return d.config.Enabled && d.available
}

// Notify sends a desktop notification
func (d *DesktopNotifyChannel) Notify(alert *components.Alert) error {
	// Check alert level threshold
	if int(alert.Level) < d.config.MinAlertLevel {
		return nil
	}

	// Check availability
	if !d.available {
		return fmt.Errorf("desktop notifications not available on this system")
	}

	// Get icon based on alert level
	icon := d.getIcon(alert.Level)

	// Send notification based on platform
	switch runtime.GOOS {
	case "linux":
		return d.notifyLinux(alert, icon)
	case "darwin":
		return d.notifyMacOS(alert)
	default:
		return fmt.Errorf("desktop notifications not supported on %s", runtime.GOOS)
	}
}

// notifyLinux sends a notification on Linux using notify-send
func (d *DesktopNotifyChannel) notifyLinux(alert *components.Alert, icon string) error {
	args := []string{
		"-u", d.getUrgency(alert.Level),
		"-t", fmt.Sprintf("%d000", d.config.Timeout), // Convert to milliseconds
		"-a", "S9S",
	}

	if icon != "" {
		args = append(args, "-i", icon)
	}

	args = append(args,
		fmt.Sprintf("S9S Alert: %s", alert.Title),
		alert.Message,
	)

	// nolint:gosec // G204: notify-send is a well-known system command, args are controlled
	cmd := exec.Command("notify-send", args...)
	return cmd.Run()
}

// notifyMacOS sends a notification on macOS using osascript
func (d *DesktopNotifyChannel) notifyMacOS(alert *components.Alert) error {
	script := fmt.Sprintf(`display notification "%s" with title "S9S Alert: %s" sound name "Glass"`,
		alert.Message, alert.Title)

	// nolint:gosec // G204: osascript is a well-known macOS system command, args are controlled
	cmd := exec.Command("osascript", "-e", script)
	return cmd.Run()
}

// getIcon returns an appropriate icon name for the alert level (Linux)
func (d *DesktopNotifyChannel) getIcon(level components.AlertLevel) string {
	switch level {
	case components.AlertInfo:
		return "dialog-information"
	case components.AlertWarning:
		return "dialog-warning"
	case components.AlertError:
		return "dialog-error"
	case components.AlertCritical:
		return "dialog-error" // Use same as error
	default:
		return ""
	}
}

// getUrgency returns the urgency level for notify-send
func (d *DesktopNotifyChannel) getUrgency(level components.AlertLevel) string {
	switch level {
	case components.AlertCritical:
		return "critical"
	case components.AlertError, components.AlertWarning:
		return "normal"
	default:
		return "low"
	}
}

// Configure updates channel configuration
func (d *DesktopNotifyChannel) Configure(config map[string]interface{}) error {
	if enabled, ok := config["enabled"].(bool); ok {
		d.config.Enabled = enabled
	}

	if minLevel, ok := config["min_alert_level"].(int); ok {
		d.config.MinAlertLevel = minLevel
	}

	if timeout, ok := config["timeout"].(int); ok {
		d.config.Timeout = timeout
	}

	return nil
}
