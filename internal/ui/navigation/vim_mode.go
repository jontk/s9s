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
	r := event.Rune()

	// Check for double key press (like gg)
	if doubleKeyEvent := v.handleDoubleKeyPress(r, currentTime); doubleKeyEvent != nil {
		return doubleKeyEvent
	}

	// Handle number prefix for repeat count
	if v.handleNumberPrefix(r, currentTime) {
		return nil // Consume the number
	}

	// Handle single key commands
	return v.handleSingleKeyCommand(r, currentTime, event)
}

func (v *VimMode) handleDoubleKeyPress(r rune, currentTime time.Time) *tcell.EventKey {
	if v.lastKey != r || currentTime.Sub(v.lastKeyTime) >= 500*time.Millisecond {
		return nil
	}

	// Double key press detected
	switch r {
	case 'g':
		// gg - go to top
		v.reset()
		return tcell.NewEventKey(tcell.KeyHome, 0, tcell.ModNone)
	case 'd':
		// dd - delete line (in future, for now just reset)
		v.reset()
		return nil
	}

	return nil
}

func (v *VimMode) handleNumberPrefix(r rune, currentTime time.Time) bool {
	if r < '1' || r > '9' {
		return false
	}

	if v.repeatCount == 0 {
		v.repeatCount = int(r - '0')
	} else {
		v.repeatCount = v.repeatCount*10 + int(r-'0')
	}

	v.lastKey = r
	v.lastKeyTime = currentTime
	return true
}

func (v *VimMode) handleSingleKeyCommand(r rune, currentTime time.Time, event *tcell.EventKey) *tcell.EventKey {
	handlers := map[rune]func() *tcell.EventKey{
		'j': func() *tcell.EventKey {
			v.reset()
			return tcell.NewEventKey(tcell.KeyDown, 0, tcell.ModNone)
		},
		'k': func() *tcell.EventKey {
			v.reset()
			return tcell.NewEventKey(tcell.KeyUp, 0, tcell.ModNone)
		},
		'h': func() *tcell.EventKey {
			v.reset()
			return tcell.NewEventKey(tcell.KeyLeft, 0, tcell.ModNone)
		},
		'l': func() *tcell.EventKey {
			v.reset()
			return tcell.NewEventKey(tcell.KeyRight, 0, tcell.ModNone)
		},
		'G': func() *tcell.EventKey {
			v.reset()
			return tcell.NewEventKey(tcell.KeyEnd, 0, tcell.ModNone)
		},
		'0': func() *tcell.EventKey {
			v.reset()
			return tcell.NewEventKey(tcell.KeyHome, 0, tcell.ModNone)
		},
		'$': func() *tcell.EventKey {
			v.reset()
			return tcell.NewEventKey(tcell.KeyEnd, 0, tcell.ModNone)
		},
		'g': func() *tcell.EventKey {
			// First 'g' - wait for second key
			v.lastKey = 'g'
			v.lastKeyTime = currentTime
			return nil // Consume the key
		},
		'/': func() *tcell.EventKey {
			// Search - let it pass through
			v.reset()
			return event
		},
	}

	if handler, ok := handlers[r]; ok {
		return handler()
	}

	// Unknown vim command, reset and pass through
	v.reset()
	return event
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
