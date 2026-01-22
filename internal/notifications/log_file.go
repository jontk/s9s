package notifications

import (
	"github.com/jontk/s9s/internal/ui/components"
)

// LogFileConfig holds configuration for log file notifications
type LogFileConfig struct {
	Enabled bool   `json:"enabled"`
	LogPath string `json:"log_path"`
}

// LogFileChannel implements log file notifications
type LogFileChannel struct {
	config LogFileConfig
}

// NewLogFileChannel creates a new log file notification channel
func NewLogFileChannel(config LogFileConfig) *LogFileChannel {
	return &LogFileChannel{
		config: config,
	}
}

// Name returns the channel name
func (l *LogFileChannel) Name() string {
	return "log_file"
}

// IsEnabled returns whether the channel is enabled
func (l *LogFileChannel) IsEnabled() bool {
	return l.config.Enabled
}

// Notify logs the alert (actual logging is handled by AlertLogger in NotificationManager)
func (l *LogFileChannel) Notify(alert *components.Alert) error {
	// The actual logging is handled by the AlertLogger in NotificationManager
	// This channel just needs to be enabled for logging to occur
	return nil
}

// Configure updates channel configuration
func (l *LogFileChannel) Configure(config map[string]interface{}) error {
	if enabled, ok := config["enabled"].(bool); ok {
		l.config.Enabled = enabled
	}

	if logPath, ok := config["log_path"].(string); ok {
		l.config.LogPath = logPath
	}

	return nil
}
