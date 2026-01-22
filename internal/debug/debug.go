package debug

import (
	"log"
	"os"
)

var Logger *log.Logger

func init() {
	// Create debug log file
	logFile, err := os.OpenFile("s9s-debug.log", os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0666)
	if err != nil {
		log.Printf("Failed to create debug log file: %v", err)
		Logger = log.New(os.Stderr, "[DEBUG] ", log.LstdFlags|log.Lmicroseconds)
	} else {
		Logger = log.New(logFile, "[DEBUG] ", log.LstdFlags|log.Lmicroseconds)
	}
}
