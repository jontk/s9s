package notifications

import (
	"fmt"
	"os"

	"github.com/jontk/s9s/internal/ui/components"
)

// TerminalBellConfig holds configuration for terminal bell notifications
type TerminalBellConfig struct {
	Enabled       bool `json:"enabled"`
	MinAlertLevel int  `json:"min_alert_level"`
	RepeatCount   int  `json:"repeat_count"` // Number of bells for critical alerts
}

// TerminalBellChannel implements terminal bell notifications
type TerminalBellChannel struct {
	config TerminalBellConfig
}

// NewTerminalBellChannel creates a new terminal bell notification channel
func NewTerminalBellChannel(config TerminalBellConfig) *TerminalBellChannel {
	// Set defaults if not specified
	if config.RepeatCount <= 0 {
		config.RepeatCount = 1
	}

	return &TerminalBellChannel{
		config: config,
	}
}

// Name returns the channel name
func (t *TerminalBellChannel) Name() string {
	return "terminal_bell"
}

// IsEnabled returns whether the channel is enabled
func (t *TerminalBellChannel) IsEnabled() bool {
	return t.config.Enabled
}

// Notify sends a terminal bell notification
func (t *TerminalBellChannel) Notify(alert *components.Alert) error {
	// Check alert level threshold
	if int(alert.Level) < t.config.MinAlertLevel {
		return nil
	}

	// Determine bell count based on severity
	bellCount := 1
	if alert.Level == components.AlertCritical && t.config.RepeatCount > 1 {
		bellCount = t.config.RepeatCount
	}

	// Send terminal bell(s)
	for i := 0; i < bellCount; i++ {
		fmt.Fprint(os.Stderr, "\a") // ASCII bell character
		if i < bellCount-1 {
			// Small delay between bells for critical alerts
			// Note: Using simple loop as sleep would block
			for j := 0; j < 100000000; j++ {
				// Busy wait ~100ms
			}
		}
	}

	return nil
}

// Configure updates channel configuration
func (t *TerminalBellChannel) Configure(config map[string]interface{}) error {
	if enabled, ok := config["enabled"].(bool); ok {
		t.config.Enabled = enabled
	}

	if minLevel, ok := config["min_alert_level"].(int); ok {
		t.config.MinAlertLevel = minLevel
	}

	if repeatCount, ok := config["repeat_count"].(int); ok {
		t.config.RepeatCount = repeatCount
	}

	return nil
}
