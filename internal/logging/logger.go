package logging

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"gopkg.in/natefinch/lumberjack.v2"
)

var (
	// Global logger instance
	logger *Logger
	once   sync.Once
)

// Level represents log level
type Level int

const (
	// DebugLevel has verbose message
	DebugLevel Level = iota
	// InfoLevel is default log level
	InfoLevel
	// WarnLevel is for warning conditions
	WarnLevel
	// ErrorLevel is for error conditions
	ErrorLevel
	// FatalLevel is for fatal conditions
	FatalLevel
	// PanicLevel is for panic conditions
	PanicLevel
)

// Config holds logger configuration
type Config struct {
	// Level is the minimum log level
	Level Level

	// Console enables console output
	Console bool

	// ConsoleJSON enables JSON format for console output
	ConsoleJSON bool

	// File enables file output
	File bool

	// Filename is the file to write logs to
	Filename string

	// MaxSize is the maximum size in megabytes of the log file
	MaxSize int

	// MaxBackups is the maximum number of old log files to retain
	MaxBackups int

	// MaxAge is the maximum number of days to retain old log files
	MaxAge int

	// Compress determines if the rotated log files should be compressed
	Compress bool
}

// DefaultConfig returns default logger configuration
func DefaultConfig() *Config {
	return &Config{
		Level:       InfoLevel,
		Console:     true,
		ConsoleJSON: false,
		File:        true,
		Filename:    filepath.Join(os.TempDir(), "s9s.log"),
		MaxSize:     10,
		MaxBackups:  3,
		MaxAge:      7,
		Compress:    true,
	}
}

// Logger wraps zerolog logger
type Logger struct {
	*zerolog.Logger
	config *Config
}

// Init initializes the global logger with the given configuration
func Init(config *Config) {
	once.Do(func() {
		logger = newLogger(config)

		// Set global logger
		log.Logger = *logger.Logger
	})
}

// GetLogger returns the global logger instance
func GetLogger() *Logger {
	if logger == nil {
		// Initialize with default config if not already initialized
		Init(DefaultConfig())
	}
	return logger
}

// newLogger creates a new logger instance
func newLogger(config *Config) *Logger {
	var writers []io.Writer

	// Console writer
	if config.Console {
		if config.ConsoleJSON {
			writers = append(writers, os.Stderr)
		} else {
			writers = append(writers, zerolog.ConsoleWriter{
				Out:        os.Stderr,
				TimeFormat: time.RFC3339,
			})
		}
	}

	// File writer with rotation
	if config.File && config.Filename != "" {
		// Ensure log directory exists
		logDir := filepath.Dir(config.Filename)
		if err := os.MkdirAll(logDir, 0o750); err != nil { // rwxr-x--- for log directories
			_, _ = fmt.Fprintf(os.Stderr, "Failed to create log directory: %v\n", err)
		} else {
			fileWriter := &lumberjack.Logger{
				Filename:   config.Filename,
				MaxSize:    config.MaxSize,
				MaxBackups: config.MaxBackups,
				MaxAge:     config.MaxAge,
				Compress:   config.Compress,
			}
			writers = append(writers, fileWriter)
		}
	}

	// Create multi-writer
	var writer io.Writer
	if len(writers) == 0 {
		writer = os.Stderr
	} else if len(writers) == 1 {
		writer = writers[0]
	} else {
		writer = io.MultiWriter(writers...)
	}

	// Create logger
	zl := zerolog.New(writer).With().Timestamp().Logger()

	// Set log level
	zl = zl.Level(convertLevel(config.Level))

	return &Logger{
		Logger: &zl,
		config: config,
	}
}

// convertLevel converts our log level to zerolog level
func convertLevel(level Level) zerolog.Level {
	switch level {
	case DebugLevel:
		return zerolog.DebugLevel
	case InfoLevel:
		return zerolog.InfoLevel
	case WarnLevel:
		return zerolog.WarnLevel
	case ErrorLevel:
		return zerolog.ErrorLevel
	case FatalLevel:
		return zerolog.FatalLevel
	case PanicLevel:
		return zerolog.PanicLevel
	default:
		return zerolog.InfoLevel
	}
}

// With creates a child logger with the given fields
func (l *Logger) With(fields map[string]interface{}) *Logger {
	ctx := l.Logger.With()
	for k, v := range fields {
		ctx = ctx.Interface(k, v)
	}
	zl := ctx.Logger()
	return &Logger{
		Logger: &zl,
		config: l.config,
	}
}

// WithError creates a child logger with the error field set
func (l *Logger) WithError(err error) *Logger {
	zl := l.Logger.With().Err(err).Logger()
	return &Logger{
		Logger: &zl,
		config: l.config,
	}
}

// Global logging functions that use the global logger

// Debug logs a debug message
func Debug(msg string) {
	GetLogger().Debug().Msg(msg)
}

// Debugf logs a formatted debug message
func Debugf(format string, v ...interface{}) {
	GetLogger().Debug().Msgf(format, v...)
}

// Info logs an info message
func Info(msg string) {
	GetLogger().Info().Msg(msg)
}

// Infof logs a formatted info message
func Infof(format string, v ...interface{}) {
	GetLogger().Info().Msgf(format, v...)
}

// Warn logs a warning message
func Warn(msg string) {
	GetLogger().Warn().Msg(msg)
}

// Warnf logs a formatted warning message
func Warnf(format string, v ...interface{}) {
	GetLogger().Warn().Msgf(format, v...)
}

// Error logs an error message
func Error(msg string) {
	GetLogger().Error().Msg(msg)
}

// Errorf logs a formatted error message
func Errorf(format string, v ...interface{}) {
	GetLogger().Error().Msgf(format, v...)
}

// Fatal logs a fatal message and exits
func Fatal(msg string) {
	GetLogger().Fatal().Msg(msg)
}

// Fatalf logs a formatted fatal message and exits
func Fatalf(format string, v ...interface{}) {
	GetLogger().Fatal().Msgf(format, v...)
}

// Panic logs a panic message and panics
func Panic(msg string) {
	GetLogger().Panic().Msg(msg)
}

// Panicf logs a formatted panic message and panics
func Panicf(format string, v ...interface{}) {
	GetLogger().Panic().Msgf(format, v...)
}

// WithField adds a field to the logger
func WithField(key string, value interface{}) *Logger {
	return GetLogger().With(map[string]interface{}{key: value})
}

// WithFields adds multiple fields to the logger
func WithFields(fields map[string]interface{}) *Logger {
	return GetLogger().With(fields)
}

// WithError adds an error field to the logger
func WithError(err error) *Logger {
	return GetLogger().WithError(err)
}
