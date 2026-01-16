// Package logging provides a simple logging infrastructure for the observability plugin
// It supports file-based logging with different levels and structured output
package logging

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// LogLevel represents the severity of a log message
type LogLevel int

const (
	DEBUG LogLevel = iota
	INFO
	WARN
	ERROR
)

// String returns the string representation of a log level
func (l LogLevel) String() string {
	switch l {
	case DEBUG:
		return "DEBUG"
	case INFO:
		return "INFO"
	case WARN:
		return "WARN"
	case ERROR:
		return "ERROR"
	default:
		return "UNKNOWN"
	}
}

// Logger provides structured logging for the observability plugin
type Logger struct {
	mu       sync.Mutex
	file     *os.File
	logger   *log.Logger
	level    LogLevel
	component string
}

var (
	// Global logger instance
	globalLogger *Logger
	once         sync.Once
)

// Config contains logger configuration
type Config struct {
	LogFile      string `yaml:"logFile" json:"logFile"`
	Level        string `yaml:"level" json:"level"`
	Component    string `yaml:"component" json:"component"`
	LogToConsole bool   `yaml:"logToConsole" json:"logToConsole"`
}

// DefaultConfig returns the default logging configuration
func DefaultConfig() Config {
	return Config{
		LogFile:      "data/observability/debug.log",
		Level:        "DEBUG",
		Component:    "observability",
		LogToConsole: false,
	}
}

// NewLogger creates a new logger instance
func NewLogger(config Config) (*Logger, error) {
	// Parse log level
	var level LogLevel
	switch config.Level {
	case "DEBUG":
		level = DEBUG
	case "INFO":
		level = INFO
	case "WARN":
		level = WARN
	case "ERROR":
		level = ERROR
	default:
		level = INFO
	}

	// Create log directory if it doesn't exist
	logDir := filepath.Dir(config.LogFile)
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create log directory: %w", err)
	}

	// Open log file
	file, err := os.OpenFile(config.LogFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file: %w", err)
	}

	// Create logger with file output, optionally with stdout
	var writer io.Writer = file
	if config.LogToConsole {
		writer = io.MultiWriter(file, os.Stdout)
	}
	logger := log.New(writer, "", 0) // We'll format ourselves

	return &Logger{
		file:      file,
		logger:    logger,
		level:     level,
		component: config.Component,
	}, nil
}

// GetGlobalLogger returns the global logger instance, creating it if necessary
func GetGlobalLogger() *Logger {
	once.Do(func() {
		config := DefaultConfig()
		var err error
		globalLogger, err = NewLogger(config)
		if err != nil {
			// Fallback to stdout-only logger
			globalLogger = &Logger{
				logger:    log.New(os.Stdout, "", 0),
				level:     DEBUG,
				component: "observability",
			}
		}
	})
	return globalLogger
}

// SetGlobalLogger sets the global logger instance
func SetGlobalLogger(logger *Logger) {
	globalLogger = logger
}

// formatMessage formats a log message with timestamp, level, component, and message
func (l *Logger) formatMessage(level LogLevel, component, message string) string {
	timestamp := time.Now().Format("2006-01-02 15:04:05.000")
	return fmt.Sprintf("[%s] [%s] [%s] %s", timestamp, level.String(), component, message)
}

// shouldLog checks if a message should be logged based on the current log level
func (l *Logger) shouldLog(level LogLevel) bool {
	return level >= l.level
}

// log writes a log message if it meets the level threshold
func (l *Logger) log(level LogLevel, component, format string, args ...interface{}) {
	if !l.shouldLog(level) {
		return
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	message := fmt.Sprintf(format, args...)
	formatted := l.formatMessage(level, component, message)
	l.logger.Println(formatted)
}

// Debug logs a debug message
func (l *Logger) Debug(component, format string, args ...interface{}) {
	l.log(DEBUG, component, format, args...)
}

// Info logs an info message
func (l *Logger) Info(component, format string, args ...interface{}) {
	l.log(INFO, component, format, args...)
}

// Warn logs a warning message
func (l *Logger) Warn(component, format string, args ...interface{}) {
	l.log(WARN, component, format, args...)
}

// Error logs an error message
func (l *Logger) Error(component, format string, args ...interface{}) {
	l.log(ERROR, component, format, args...)
}

// Close closes the logger and its file handle
func (l *Logger) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	
	if l.file != nil {
		return l.file.Close()
	}
	return nil
}

// SetLevel changes the log level
func (l *Logger) SetLevel(level LogLevel) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.level = level
}

// Component creates a new logger with a specific component name
func (l *Logger) Component(component string) *Logger {
	return &Logger{
		file:      l.file,
		logger:    l.logger,
		level:     l.level,
		component: component,
		mu:        l.mu, // Share mutex with parent
	}
}

// Convenience functions that use the global logger

// Debug logs a debug message using the global logger
func Debug(component, format string, args ...interface{}) {
	GetGlobalLogger().Debug(component, format, args...)
}

// Info logs an info message using the global logger
func Info(component, format string, args ...interface{}) {
	GetGlobalLogger().Info(component, format, args...)
}

// Warn logs a warning message using the global logger
func Warn(component, format string, args ...interface{}) {
	GetGlobalLogger().Warn(component, format, args...)
}

// Error logs an error message using the global logger
func Error(component, format string, args ...interface{}) {
	GetGlobalLogger().Error(component, format, args...)
}