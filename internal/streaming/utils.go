package streaming

import (
	"time"
)

// GetCurrentTime returns the current time - allows for easier testing with time mocking
func GetCurrentTime() time.Time {
	return time.Now()
}
