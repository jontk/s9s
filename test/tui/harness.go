package tui

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/jontk/s9s/internal/app"
	"github.com/jontk/s9s/internal/config"
)

// TestHarness provides a test harness for TUI testing using SimulationScreen
type TestHarness struct {
	t      *testing.T
	screen tcell.SimulationScreen
	app    *app.S9s
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
	mu     sync.Mutex
	width  int
	height int
}

//nolint:revive // type alias for backward compatibility
type TUITestHarness = TestHarness

// NewTUITestHarness creates a new TUI test harness with a simulated screen
func NewTUITestHarness(t *testing.T) *TestHarness {
	return NewTUITestHarnessWithSize(t, 120, 40)
}

// NewTUITestHarnessWithSize creates a new TUI test harness with custom screen dimensions
func NewTUITestHarnessWithSize(t *testing.T, width, height int) *TUITestHarness {
	// Create simulation screen
	screen := tcell.NewSimulationScreen("UTF-8")
	if err := screen.Init(); err != nil {
		t.Fatalf("Failed to initialize simulation screen: %v", err)
	}
	screen.SetSize(width, height)

	// Create test config with mock client and default values
	cfg := &config.Config{
		UseMockClient:  true,
		RefreshRate:    "0s", // Disable auto-refresh in tests
		CurrentContext: "test",
		MaxRetries:     3,
		UI: config.UIConfig{
			Skin:        "default",
			EnableMouse: false,
			Logoless:    false,
			Statusless:  false,
			NoIcons:     false,
		},
		Views: config.ViewsConfig{
			Jobs: config.JobsViewConfig{
				ShowOnlyActive: false,
				DefaultSort:    "id",
				MaxJobs:        100,
			},
			Nodes: config.NodesViewConfig{
				GroupBy:          "state",
				ShowUtilization:  true,
			},
		},
		Features: config.FeaturesConfig{
			Streaming: false,
			Pulseye:   false,
			Xray:      false,
		},
	}

	// Create context
	ctx, cancel := context.WithCancel(context.Background())

	// Create app with simulation screen
	s9sApp, err := app.NewWithScreen(ctx, cfg, screen)
	if err != nil {
		cancel()
		screen.Fini()
		t.Fatalf("Failed to create app: %v", err)
	}

	h := &TUITestHarness{
		t:      t,
		screen: screen,
		app:    s9sApp,
		ctx:    ctx,
		cancel: cancel,
		width:  width,
		height: height,
	}

	return h
}

// Start starts the application in a goroutine
func (h *TestHarness) Start() {
	h.wg.Add(1)
	go func() {
		defer h.wg.Done()
		if err := h.app.Run(); err != nil {
			h.t.Logf("App run error: %v", err)
		}
	}()

	// Give the app a moment to initialize
	time.Sleep(50 * time.Millisecond)
}

// Cleanup stops the application and cleans up resources
func (h *TestHarness) Cleanup() {
	// Cancel context first
	h.cancel()

	// Stop the app
	if err := h.app.Stop(); err != nil {
		h.t.Logf("Error stopping app: %v", err)
	}

	// Wait for app goroutine to finish
	h.wg.Wait()

	// Note: We don't call h.screen.Fini() here because the app.Stop()
	// already finalizes the screen through tview.Application.Stop()
}

// SendKey sends a key event to the application
func (h *TestHarness) SendKey(key tcell.Key, r rune, mod tcell.ModMask) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.screen.InjectKey(key, r, mod)

	// Give the app time to process
	time.Sleep(20 * time.Millisecond)
	h.screen.Sync()

	// Log the event for debugging
	h.t.Logf("Sent key event: key=%v rune=%c mod=%v", key, r, mod)
}

// SendRune sends a rune (character) to the application
func (h *TestHarness) SendRune(r rune) {
	h.SendKey(tcell.KeyRune, r, tcell.ModNone)
}

// SendEnter sends the Enter key
func (h *TestHarness) SendEnter() {
	h.SendKey(tcell.KeyEnter, 0, tcell.ModNone)
}

// SendEsc sends the Escape key
func (h *TestHarness) SendEsc() {
	h.SendKey(tcell.KeyEsc, 0, tcell.ModNone)
}

// SendTab sends the Tab key
func (h *TestHarness) SendTab() {
	h.SendKey(tcell.KeyTab, 0, tcell.ModNone)
}

// SendBacktab sends the Shift+Tab key
func (h *TestHarness) SendBacktab() {
	h.SendKey(tcell.KeyBacktab, 0, tcell.ModNone)
}

// SendCtrlC sends Ctrl+C
func (h *TestHarness) SendCtrlC() {
	h.SendKey(tcell.KeyCtrlC, 0, tcell.ModCtrl)
}

// GetScreenText returns all text on the screen as a single string
func (h *TestHarness) GetScreenText() string {
	h.mu.Lock()
	defer h.mu.Unlock()

	var result strings.Builder
	width, height := h.screen.Size()

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			mainc, _, _, _ := h.screen.GetContent(x, y)
			if mainc != 0 {
				result.WriteRune(mainc)
			} else {
				result.WriteRune(' ')
			}
		}
		result.WriteRune('\n')
	}

	return result.String()
}

// GetScreenLines returns the screen content as a slice of lines
func (h *TestHarness) GetScreenLines() []string {
	text := h.GetScreenText()
	return strings.Split(text, "\n")
}

// ContainsText checks if the screen contains the specified text
func (h *TestHarness) ContainsText(expected string) bool {
	screenText := h.GetScreenText()
	return strings.Contains(screenText, expected)
}

// ContainsTextCaseInsensitive checks if the screen contains the text (case-insensitive)
func (h *TestHarness) ContainsTextCaseInsensitive(expected string) bool {
	screenText := strings.ToLower(h.GetScreenText())
	return strings.Contains(screenText, strings.ToLower(expected))
}

// GetCellAt returns the character and style at the specified position
func (h *TestHarness) GetCellAt(x, y int) (rune, tcell.Style) {
	h.mu.Lock()
	defer h.mu.Unlock()

	mainc, _, style, _ := h.screen.GetContent(x, y)
	return mainc, style
}

// WaitForText waits for the specified text to appear on screen, with a timeout
func (h *TestHarness) WaitForText(text string, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if h.ContainsText(text) {
			return true
		}
		time.Sleep(50 * time.Millisecond)
	}
	return false
}

// WaitForTextToDisappear waits for text to disappear from screen, with a timeout
func (h *TestHarness) WaitForTextToDisappear(text string, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if !h.ContainsText(text) {
			return true
		}
		time.Sleep(50 * time.Millisecond)
	}
	return false
}

// DumpScreen dumps the screen contents to the test log for debugging
func (h *TestHarness) DumpScreen() {
	h.t.Log("=== Screen Dump ===")
	lines := h.GetScreenLines()
	for i, line := range lines {
		h.t.Logf("%3d: %s", i, line)
	}
	h.t.Log("===================")
}

// GetCurrentView returns the name of the current view
func (h *TestHarness) GetCurrentView() string {
	return h.app.GetCurrentViewName()
}

// IsModalOpen checks if a modal is currently open
func (h *TestHarness) IsModalOpen() bool {
	return h.app.IsModalOpen()
}

// GetModalName returns the name of the currently open modal
func (h *TestHarness) GetModalName() string {
	return h.app.GetModalName()
}

// IsCmdVisible checks if the command line is visible
func (h *TestHarness) IsCmdVisible() bool {
	return h.app.IsCmdVisible()
}

// AssertContainsText asserts that the screen contains the specified text
func (h *TestHarness) AssertContainsText(expected string) {
	if !h.ContainsText(expected) {
		h.DumpScreen()
		h.t.Errorf("Expected screen to contain %q, but it was not found", expected)
	}
}

// AssertNotContainsText asserts that the screen does not contain the specified text
func (h *TestHarness) AssertNotContainsText(expected string) {
	if h.ContainsText(expected) {
		h.DumpScreen()
		h.t.Errorf("Expected screen to NOT contain %q, but it was found", expected)
	}
}

// AssertCurrentView asserts that the current view matches the expected view name
func (h *TestHarness) AssertCurrentView(expected string) {
	actual := h.GetCurrentView()
	if actual != expected {
		h.DumpScreen()
		h.t.Errorf("Expected current view to be %q, but got %q", expected, actual)
	}
}

// AssertModalOpen asserts that a modal is open
func (h *TestHarness) AssertModalOpen() {
	if !h.IsModalOpen() {
		h.DumpScreen()
		h.t.Error("Expected modal to be open, but no modal was found")
	}
}

// AssertModalClosed asserts that no modal is open
func (h *TestHarness) AssertModalClosed() {
	if h.IsModalOpen() {
		h.DumpScreen()
		h.t.Errorf("Expected no modal to be open, but found: %s", h.GetModalName())
	}
}

// AssertWaitForText asserts that text appears within the timeout
func (h *TestHarness) AssertWaitForText(text string, timeout time.Duration) {
	if !h.WaitForText(text, timeout) {
		h.DumpScreen()
		h.t.Errorf("Timeout waiting for text %q to appear", text)
	}
}

// GetApp returns the underlying app instance for advanced testing
func (h *TestHarness) GetApp() *app.S9s {
	return h.app
}

// GetScreen returns the simulation screen for low-level testing
func (h *TestHarness) GetScreen() tcell.SimulationScreen {
	return h.screen
}

// String provides a string representation useful for debugging
func (h *TestHarness) String() string {
	return fmt.Sprintf("TUITestHarness{size=%dx%d, view=%s, modal=%s}",
		h.width, h.height, h.GetCurrentView(), h.GetModalName())
}
