package components

import (
	"fmt"
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// StatusBar displays status information and keyboard hints
type StatusBar struct {
	*tview.TextView
	hints         []string
	message       string
	messageExpiry time.Time
	flashColor    tcell.Color
}

// NewStatusBar creates a new status bar component
func NewStatusBar() *StatusBar {
	s := &StatusBar{
		TextView:   tview.NewTextView(),
		hints:      []string{},
		flashColor: tcell.ColorDefault,
	}

	s.TextView.
		SetDynamicColors(true).
		SetTextAlign(tview.AlignLeft)

	return s
}

// SetHints sets the keyboard hints to display
func (s *StatusBar) SetHints(hints []string) {
	s.hints = hints
	s.updateDisplay()
}

// SetMessage sets a temporary message with optional expiry
func (s *StatusBar) SetMessage(message string, duration time.Duration) {
	s.message = message
	if duration > 0 {
		s.messageExpiry = time.Now().Add(duration)
		// Clear message after duration
		go func() {
			time.Sleep(duration)
			if time.Now().After(s.messageExpiry) {
				s.ClearMessage()
			}
		}()
	} else {
		s.messageExpiry = time.Time{}
	}
	s.updateDisplay()
}

// ClearMessage clears the current message
func (s *StatusBar) ClearMessage() {
	s.message = ""
	s.messageExpiry = time.Time{}
	s.updateDisplay()
}

// Flash displays a temporary flash message with color
func (s *StatusBar) Flash(message string, color tcell.Color, duration time.Duration) {
	s.flashColor = color
	s.SetMessage(message, duration)
	
	// Reset flash color after duration
	go func() {
		time.Sleep(duration)
		s.flashColor = tcell.ColorDefault
		s.updateDisplay()
	}()
}

// Success displays a success message
func (s *StatusBar) Success(message string) {
	s.Flash(fmt.Sprintf("✓ %s", message), tcell.ColorGreen, 3*time.Second)
}

// Error displays an error message
func (s *StatusBar) Error(message string) {
	s.Flash(fmt.Sprintf("✗ %s", message), tcell.ColorRed, 5*time.Second)
}

// Warning displays a warning message
func (s *StatusBar) Warning(message string) {
	s.Flash(fmt.Sprintf("⚠ %s", message), tcell.ColorYellow, 4*time.Second)
}

// Info displays an info message
func (s *StatusBar) Info(message string) {
	s.Flash(fmt.Sprintf("ℹ %s", message), tcell.ColorTeal, 3*time.Second)
}

// updateDisplay updates the status bar display
func (s *StatusBar) updateDisplay() {
	var content strings.Builder

	// Show message if present and not expired
	if s.message != "" && (s.messageExpiry.IsZero() || time.Now().Before(s.messageExpiry)) {
		if s.flashColor != tcell.ColorDefault {
			colorName := getColorName(s.flashColor)
			content.WriteString(fmt.Sprintf("[%s]%s[white]", colorName, s.message))
		} else {
			content.WriteString(s.message)
		}
	} else if len(s.hints) > 0 {
		// Show hints if no message
		content.WriteString(s.formatHints())
	}

	s.TextView.SetText(content.String())
}

// formatHints formats the keyboard hints for display
func (s *StatusBar) formatHints() string {
	if len(s.hints) == 0 {
		return ""
	}

	// Join hints with separators
	return strings.Join(s.hints, "  ")
}

// getColorName returns the color name for tview markup
func getColorName(color tcell.Color) string {
	switch color {
	case tcell.ColorRed:
		return "red"
	case tcell.ColorGreen:
		return "green"
	case tcell.ColorYellow:
		return "yellow"
	case tcell.ColorBlue:
		return "blue"
	case tcell.ColorPurple:
		return "purple"
	case tcell.ColorTeal:
		return "teal"
	case tcell.ColorWhite:
		return "white"
	case tcell.ColorBlack:
		return "black"
	case tcell.ColorOrange:
		return "orange"
	case tcell.ColorGray:
		return "gray"
	default:
		return "white"
	}
}

// StatusBarBuilder provides a fluent interface for building status bars
type StatusBarBuilder struct {
	statusBar *StatusBar
}

// NewStatusBarBuilder creates a new status bar builder
func NewStatusBarBuilder() *StatusBarBuilder {
	return &StatusBarBuilder{
		statusBar: NewStatusBar(),
	}
}

// WithHints sets the keyboard hints
func (sb *StatusBarBuilder) WithHints(hints ...string) *StatusBarBuilder {
	sb.statusBar.SetHints(hints)
	return sb
}

// WithBackground sets the background color
func (sb *StatusBarBuilder) WithBackground(color tcell.Color) *StatusBarBuilder {
	sb.statusBar.SetBackgroundColor(color)
	return sb
}

// WithTextColor sets the text color
func (sb *StatusBarBuilder) WithTextColor(color tcell.Color) *StatusBarBuilder {
	sb.statusBar.SetTextColor(color)
	return sb
}

// Build returns the status bar
func (sb *StatusBarBuilder) Build() *StatusBar {
	return sb.statusBar
}

// StatusMessage represents a status message with metadata
type StatusMessage struct {
	Text      string
	Level     StatusLevel
	Timestamp time.Time
	Duration  time.Duration
}

// StatusLevel represents the level of a status message
type StatusLevel int

const (
	StatusLevelInfo StatusLevel = iota
	StatusLevelSuccess
	StatusLevelWarning
	StatusLevelError
)

// NewStatusMessage creates a new status message
func NewStatusMessage(text string, level StatusLevel, duration time.Duration) *StatusMessage {
	return &StatusMessage{
		Text:      text,
		Level:     level,
		Timestamp: time.Now(),
		Duration:  duration,
	}
}

// IsExpired checks if the message has expired
func (sm *StatusMessage) IsExpired() bool {
	if sm.Duration == 0 {
		return false
	}
	return time.Since(sm.Timestamp) > sm.Duration
}

// GetColor returns the color for this message level
func (sm *StatusMessage) GetColor() tcell.Color {
	switch sm.Level {
	case StatusLevelSuccess:
		return tcell.ColorGreen
	case StatusLevelWarning:
		return tcell.ColorYellow
	case StatusLevelError:
		return tcell.ColorRed
	default:
		return tcell.ColorTeal
	}
}

// GetIcon returns the icon for this message level
func (sm *StatusMessage) GetIcon() string {
	switch sm.Level {
	case StatusLevelSuccess:
		return "✓"
	case StatusLevelWarning:
		return "⚠"
	case StatusLevelError:
		return "✗"
	default:
		return "ℹ"
	}
}

// Format formats the message for display
func (sm *StatusMessage) Format() string {
	icon := sm.GetIcon()
	return fmt.Sprintf("%s %s", icon, sm.Text)
}