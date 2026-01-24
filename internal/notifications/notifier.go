package notifications

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/jontk/s9s/internal/fileperms"
	"github.com/jontk/s9s/internal/logging"
	"github.com/jontk/s9s/internal/ui/components"
)

// NotificationChannel represents a notification delivery channel
type NotificationChannel interface {
	// Name returns the channel name
	Name() string

	// IsEnabled returns whether the channel is enabled
	IsEnabled() bool

	// Notify sends a notification through this channel
	Notify(alert *components.Alert) error

	// Configure updates channel configuration
	Configure(config map[string]interface{}) error
}

// NotificationManager manages all notification channels
type NotificationManager struct {
	mu       sync.RWMutex
	channels map[string]NotificationChannel
	config   *NotificationConfig
	alertLog *AlertLogger
}

// NotificationConfig holds configuration for all notification channels
type NotificationConfig struct {
	// Global settings
	EnableNotifications bool `json:"enable_notifications"`
	MinAlertLevel       int  `json:"min_alert_level"` // Minimum alert level to notify

	// Channel-specific settings
	TerminalBell  TerminalBellConfig  `json:"terminal_bell"`
	LogFile       LogFileConfig       `json:"log_file"`
	DesktopNotify DesktopNotifyConfig `json:"desktop_notify"`
	Webhook       WebhookConfig       `json:"webhook"`
}

// NewNotificationManager creates a new notification manager
func NewNotificationManager(configPath string) (*NotificationManager, error) {
	nm := &NotificationManager{
		channels: make(map[string]NotificationChannel),
	}

	// Load configuration
	config, err := loadConfig(configPath)
	if err != nil {
		// Use default config if file doesn't exist
		config = defaultConfig()
	}
	nm.config = config

	// Initialize alert logger
	nm.alertLog = NewAlertLogger(config.LogFile.LogPath)

	// Initialize channels based on config
	nm.initializeChannels()

	return nm, nil
}

// Notify sends an alert through all enabled channels
func (nm *NotificationManager) Notify(alert *components.Alert) {
	nm.mu.RLock()
	defer nm.mu.RUnlock()

	// Check if notifications are enabled
	if !nm.config.EnableNotifications {
		return
	}

	// Check alert level threshold
	if int(alert.Level) < nm.config.MinAlertLevel {
		return
	}

	// Log the alert first
	if err := nm.alertLog.LogAlert(alert); err != nil {
		logging.Errorf("Failed to log alert: %v", err)
	}

	// Send through all enabled channels
	var wg sync.WaitGroup
	for name, channel := range nm.channels {
		if channel.IsEnabled() {
			wg.Add(1)
			go func(ch NotificationChannel, chName string) {
				defer wg.Done()
				if err := ch.Notify(alert); err != nil {
					logging.Errorf("Failed to send notification via %s: %v", chName, err)
				}
			}(channel, name)
		}
	}
	wg.Wait()
}

// initializeChannels sets up all notification channels
func (nm *NotificationManager) initializeChannels() {
	// Terminal bell
	nm.channels["terminal_bell"] = NewTerminalBellChannel(nm.config.TerminalBell)

	// Log file
	nm.channels["log_file"] = NewLogFileChannel(nm.config.LogFile)

	// Desktop notifications (if available)
	if desktopChannel := NewDesktopNotifyChannel(nm.config.DesktopNotify); desktopChannel != nil {
		nm.channels["desktop_notify"] = desktopChannel
	}

	// Webhook
	if nm.config.Webhook.URL != "" {
		nm.channels["webhook"] = NewWebhookChannel(nm.config.Webhook)
	}
}

// UpdateConfig updates the notification configuration
func (nm *NotificationManager) UpdateConfig(config *NotificationConfig) error {
	nm.mu.Lock()
	defer nm.mu.Unlock()

	nm.config = config

	// Reinitialize channels with new config
	nm.channels = make(map[string]NotificationChannel)
	nm.initializeChannels()

	// Save config to file
	return saveConfig(config, getConfigPath())
}

// GetConfig returns the current configuration
func (nm *NotificationManager) GetConfig() *NotificationConfig {
	nm.mu.RLock()
	defer nm.mu.RUnlock()

	return nm.config
}

// AlertLogger handles logging alerts to file
type AlertLogger struct {
	mu      sync.Mutex
	logPath string
	maxSize int64         // Max log file size in bytes
	maxAge  time.Duration // Max age of log entries
}

// NewAlertLogger creates a new alert logger
func NewAlertLogger(logPath string) *AlertLogger {
	return &AlertLogger{
		logPath: logPath,
		maxSize: 10 * 1024 * 1024,   // 10MB
		maxAge:  7 * 24 * time.Hour, // 7 days
	}
}

// LogAlert logs an alert to file
func (al *AlertLogger) LogAlert(alert *components.Alert) error {
	al.mu.Lock()
	defer al.mu.Unlock()

	// Ensure log directory exists
	logDir := filepath.Dir(al.logPath)
	if err := os.MkdirAll(logDir, fileperms.LogDir); err != nil {
		return fmt.Errorf("failed to create log directory: %w", err)
	}

	// Open log file
	file, err := os.OpenFile(al.logPath, os.O_APPEND|os.O_WRONLY|os.O_CREATE, fileperms.LogFile)
	if err != nil {
		return fmt.Errorf("failed to open log file: %w", err)
	}
	defer func() { _ = file.Close() }()

	// Create log entry
	entry := AlertLogEntry{
		Timestamp: alert.Timestamp,
		Level:     getLevelString(alert.Level),
		Title:     alert.Title,
		Message:   alert.Message,
		Source:    alert.Source,
		ID:        alert.ID,
	}

	// Write as JSON
	encoder := json.NewEncoder(file)
	if err := encoder.Encode(entry); err != nil {
		return fmt.Errorf("failed to write log entry: %w", err)
	}

	// Check if rotation is needed
	if info, err := file.Stat(); err == nil && info.Size() > al.maxSize {
		_ = al.rotateLog()
	}

	return nil
}

// AlertLogEntry represents a log entry for an alert
type AlertLogEntry struct {
	Timestamp time.Time `json:"timestamp"`
	Level     string    `json:"level"`
	Title     string    `json:"title"`
	Message   string    `json:"message"`
	Source    string    `json:"source"`
	ID        string    `json:"id"`
}

// rotateLog rotates the log file
func (al *AlertLogger) rotateLog() error {
	// Rename current log file
	timestamp := time.Now().Format("20060102-150405")
	rotatedPath := fmt.Sprintf("%s.%s", al.logPath, timestamp)

	if err := os.Rename(al.logPath, rotatedPath); err != nil {
		return err
	}

	// Clean up old log files
	go al.cleanOldLogs()

	return nil
}

// cleanOldLogs removes log files older than maxAge
func (al *AlertLogger) cleanOldLogs() {
	logDir := filepath.Dir(al.logPath)
	baseName := filepath.Base(al.logPath)

	files, err := os.ReadDir(logDir)
	if err != nil {
		return
	}

	cutoff := time.Now().Add(-al.maxAge)

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		// Check if it's a rotated log file
		if len(file.Name()) > len(baseName) && file.Name()[:len(baseName)] == baseName {
			info, err := file.Info()
			if err != nil {
				continue
			}

			if info.ModTime().Before(cutoff) {
				_ = os.Remove(filepath.Join(logDir, file.Name()))
			}
		}
	}
}

// Helper functions

func loadConfig(path string) (*NotificationConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var config NotificationConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	return &config, nil
}

func saveConfig(config *NotificationConfig, path string) error {
	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, fileperms.ConfigDir); err != nil {
		return err
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, fileperms.ConfigFile)
}

func getConfigPath() string {
	homeDir, _ := os.UserHomeDir()
	return filepath.Join(homeDir, ".s9s", "notifications.json")
}

func defaultConfig() *NotificationConfig {
	homeDir, _ := os.UserHomeDir()
	return &NotificationConfig{
		EnableNotifications: true,
		MinAlertLevel:       int(components.AlertWarning),
		TerminalBell: TerminalBellConfig{
			Enabled:       true,
			MinAlertLevel: int(components.AlertError),
		},
		LogFile: LogFileConfig{
			Enabled: true,
			LogPath: filepath.Join(homeDir, ".s9s", "alerts.log"),
		},
		DesktopNotify: DesktopNotifyConfig{
			Enabled:       false, // Disabled by default
			MinAlertLevel: int(components.AlertError),
		},
		Webhook: WebhookConfig{
			Enabled:       false,
			MinAlertLevel: int(components.AlertCritical),
		},
	}
}

func getLevelString(level components.AlertLevel) string {
	switch level {
	case components.AlertInfo:
		return "INFO"
	case components.AlertWarning:
		return "WARNING"
	case components.AlertError:
		return "ERROR"
	case components.AlertCritical:
		return "CRITICAL"
	default:
		return "UNKNOWN"
	}
}
