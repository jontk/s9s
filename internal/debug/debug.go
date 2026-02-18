// Package debug provides debugging utilities and logging for development.
package debug

import (
	"io"
	"log"
	"os"

	"github.com/jontk/s9s/internal/fileperms"
)

// Logger is the debug logger instance used for debug output.
var Logger *log.Logger

// debugEnabled controls whether debug logging is active
var debugEnabled = false

func init() {
	// By default, disable debug logging by writing to io.Discard
	// This prevents debug messages from interfering with the TUI
	Logger = log.New(io.Discard, "[DEBUG] ", log.LstdFlags|log.Lmicroseconds)
}

// Enable turns on debug logging to a file
func Enable() {
	debugEnabled = true

	// Try to create debug log file
	logFile, err := os.OpenFile("s9s-debug.log", os.O_CREATE|os.O_WRONLY|os.O_TRUNC, fileperms.LogFile)
	if err != nil {
		// If we can't create the log file, keep writing to io.Discard
		// NEVER write to stderr/stdout as it interferes with the TUI
		return
	}

	// Successfully created log file, write debug output there
	Logger = log.New(logFile, "[DEBUG] ", log.LstdFlags|log.Lmicroseconds)
}

// IsEnabled returns whether debug logging is enabled
func IsEnabled() bool {
	return debugEnabled
}
