package navigation

import (
	"time"

	"github.com/gdamore/tcell/v2"
)

// VimMode handles vim-style keyboard navigation
type VimMode struct {
	lastKey     rune
	lastKeyTime time.Time
	repeatCount int
}

// NewVimMode creates a new vim mode handler
func NewVimMode() *VimMode {
	return &VimMode{}
}

// HandleKey processes a key event and returns the translated event
func (v *VimMode) HandleKey(event *tcell.EventKey) *tcell.EventKey {
	if event.Key() != tcell.KeyRune {
		// Reset on non-rune keys
		v.reset()
		return event
	}

	currentTime := time.Now()

	// Check for double key press (like gg)
	if v.lastKey == event.Rune() && currentTime.Sub(v.lastKeyTime) < 500*time.Millisecond {
		// Double key press detected
		switch event.Rune() {
		case 'g':
			// gg - go to top
			v.reset()
			return tcell.NewEventKey(tcell.KeyHome, 0, tcell.ModNone)
		case 'd':
			// dd - delete line (in future, for now just reset)
			v.reset()
			return nil
		}
	}

	// Handle number prefix for repeat count
	if event.Rune() >= '1' && event.Rune() <= '9' {
		if v.repeatCount == 0 {
			v.repeatCount = int(event.Rune() - '0')
		} else {
			v.repeatCount = v.repeatCount*10 + int(event.Rune()-'0')
		}
		v.lastKey = event.Rune()
		v.lastKeyTime = currentTime
		return nil // Consume the number
	}

	// Handle single key commands
	switch event.Rune() {
	case 'j':
		// Move down (with repeat count)
		count := v.getRepeatCount()
		v.reset()
		if count == 1 {
			return tcell.NewEventKey(tcell.KeyDown, 0, tcell.ModNone)
		}
		// For multiple moves, we'll need to handle this differently
		// For now, just move once
		return tcell.NewEventKey(tcell.KeyDown, 0, tcell.ModNone)

	case 'k':
		// Move up
		v.reset()
		return tcell.NewEventKey(tcell.KeyUp, 0, tcell.ModNone)

	case 'h':
		// Move left
		v.reset()
		return tcell.NewEventKey(tcell.KeyLeft, 0, tcell.ModNone)

	case 'l':
		// Move right
		v.reset()
		return tcell.NewEventKey(tcell.KeyRight, 0, tcell.ModNone)

	case 'G':
		// Go to bottom (or line N with count)
		v.reset()
		return tcell.NewEventKey(tcell.KeyEnd, 0, tcell.ModNone)

	case '0':
		// Go to beginning of line
		v.reset()
		return tcell.NewEventKey(tcell.KeyHome, 0, tcell.ModNone)

	case '$':
		// Go to end of line
		v.reset()
		return tcell.NewEventKey(tcell.KeyEnd, 0, tcell.ModNone)

	case 'g':
		// First 'g' - wait for second key
		v.lastKey = 'g'
		v.lastKeyTime = currentTime
		return nil // Consume the key

	case '/':
		// Search - let it pass through
		v.reset()
		return event

	default:
		// Unknown vim command, reset and pass through
		v.reset()
		return event
	}
}

// getRepeatCount returns the current repeat count (default 1)
func (v *VimMode) getRepeatCount() int {
	if v.repeatCount == 0 {
		return 1
	}
	return v.repeatCount
}

// reset clears the vim mode state
func (v *VimMode) reset() {
	v.lastKey = 0
	v.repeatCount = 0
}

// IsWaitingForKey returns true if vim mode is waiting for another key
func (v *VimMode) IsWaitingForKey() bool {
	return v.lastKey == 'g' || v.repeatCount > 0
}
