package components

import (
	"context"
	"sync"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// LoadingIndicator shows a loading spinner and message
type LoadingIndicator struct {
	*tview.Modal
	app           *tview.Application
	ctx           context.Context
	cancel        context.CancelFunc
	mu            sync.Mutex
	isActive      bool
	spinnerFrames []string
	currentFrame  int
	baseMessage   string
}

// NewLoadingIndicator creates a new loading indicator
func NewLoadingIndicator(app *tview.Application) *LoadingIndicator {
	li := &LoadingIndicator{
		app: app,
		spinnerFrames: []string{
			"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏",
		},
		currentFrame: 0,
	}

	// Create modal with simple loading text
	li.Modal = tview.NewModal().
		SetText("Loading...").
		AddButtons([]string{}).
		SetBackgroundColor(tcell.ColorDefault)

	return li
}

// Show displays the loading indicator with the given message
func (li *LoadingIndicator) Show(message string) {
	li.mu.Lock()
	defer li.mu.Unlock()

	if li.isActive {
		return
	}

	li.isActive = true
	li.baseMessage = message
	li.currentFrame = 0

	// Create context for cancellation
	li.ctx, li.cancel = context.WithCancel(context.Background())

	// Set initial message
	li.SetText(li.spinnerFrames[0] + " " + message)

	// Start spinner animation
	go li.animate()
}

// Hide stops and hides the loading indicator
func (li *LoadingIndicator) Hide() {
	li.mu.Lock()
	defer li.mu.Unlock()

	if !li.isActive {
		return
	}

	li.isActive = false

	if li.cancel != nil {
		li.cancel()
		li.cancel = nil
	}
}

// IsActive returns true if the loading indicator is currently active
func (li *LoadingIndicator) IsActive() bool {
	li.mu.Lock()
	defer li.mu.Unlock()
	return li.isActive
}

// animate runs the spinner animation loop
func (li *LoadingIndicator) animate() {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-li.ctx.Done():
			return
		case <-ticker.C:
			li.mu.Lock()
			if !li.isActive {
				li.mu.Unlock()
				return
			}

			// Update spinner frame
			li.currentFrame = (li.currentFrame + 1) % len(li.spinnerFrames)
			spinnerText := li.spinnerFrames[li.currentFrame]
			message := li.baseMessage
			li.mu.Unlock()

			if li.app != nil {
				li.app.QueueUpdateDraw(func() {
					li.SetText(spinnerText + " " + message)
				})
			}
		}
	}
}

// SetMessage updates the loading message
func (li *LoadingIndicator) SetMessage(message string) {
	li.mu.Lock()
	defer li.mu.Unlock()

	li.baseMessage = message
	if li.app != nil {
		li.app.QueueUpdateDraw(func() {
			spinnerText := li.spinnerFrames[li.currentFrame]
			li.SetText(spinnerText + " " + message)
		})
	}
}

// LoadingManager manages loading indicators across views
type LoadingManager struct {
	indicators    map[string]*LoadingIndicator
	pages         *tview.Pages
	app           *tview.Application
	mu            sync.RWMutex
	previousFocus map[string]tview.Primitive // Track focus before showing loading
}

// NewLoadingManager creates a new loading manager
func NewLoadingManager(app *tview.Application, pages *tview.Pages) *LoadingManager {
	return &LoadingManager{
		indicators:    make(map[string]*LoadingIndicator),
		pages:         pages,
		app:           app,
		previousFocus: make(map[string]tview.Primitive),
	}
}

// Show displays a loading indicator for the given view
func (lm *LoadingManager) Show(viewName, message string) {
	lm.mu.Lock()
	defer lm.mu.Unlock()

	indicator, exists := lm.indicators[viewName]
	if !exists {
		indicator = NewLoadingIndicator(lm.app)
		lm.indicators[viewName] = indicator
	}

	// Save current focus before showing the loading indicator
	if lm.app != nil {
		lm.previousFocus[viewName] = lm.app.GetFocus()
	}

	indicator.Show(message)

	if lm.pages != nil {
		modalName := "loading_" + viewName
		lm.pages.AddPage(modalName, indicator, true, true)
	}
}

// Hide hides the loading indicator for the given view
func (lm *LoadingManager) Hide(viewName string) {
	lm.mu.Lock()
	indicator, exists := lm.indicators[viewName]
	previousFocus := lm.previousFocus[viewName]
	delete(lm.previousFocus, viewName)
	lm.mu.Unlock()

	if !exists {
		return
	}

	indicator.Hide()

	if lm.pages != nil {
		modalName := "loading_" + viewName
		lm.pages.RemovePage(modalName)
	}

	// Restore focus to where it was before showing the loading indicator
	if lm.app != nil && previousFocus != nil {
		lm.app.SetFocus(previousFocus)
	}
}

// SetMessage updates the loading message for a view
func (lm *LoadingManager) SetMessage(viewName, message string) {
	lm.mu.RLock()
	indicator, exists := lm.indicators[viewName]
	lm.mu.RUnlock()

	if exists {
		indicator.SetMessage(message)
	}
}

// IsActive returns true if loading is active for the given view
func (lm *LoadingManager) IsActive(viewName string) bool {
	lm.mu.RLock()
	indicator, exists := lm.indicators[viewName]
	lm.mu.RUnlock()

	if !exists {
		return false
	}

	return indicator.IsActive()
}

// LoadingWrapper provides a convenient way to wrap operations with loading indicators
type LoadingWrapper struct {
	manager  *LoadingManager
	viewName string
}

// NewLoadingWrapper creates a new loading wrapper for a specific view
func NewLoadingWrapper(manager *LoadingManager, viewName string) *LoadingWrapper {
	return &LoadingWrapper{
		manager:  manager,
		viewName: viewName,
	}
}

// WithLoading executes the given function with a loading indicator
func (lw *LoadingWrapper) WithLoading(message string, fn func() error) error {
	lw.manager.Show(lw.viewName, message)
	defer lw.manager.Hide(lw.viewName)

	return fn()
}

// WithLoadingAsync executes the given function asynchronously with a loading indicator
func (lw *LoadingWrapper) WithLoadingAsync(message string, fn func() error, callback func(error)) {
	lw.manager.Show(lw.viewName, message)

	go func() {
		err := fn()
		lw.manager.Hide(lw.viewName)
		if callback != nil {
			callback(err)
		}
	}()
}

// UpdateMessage updates the loading message
func (lw *LoadingWrapper) UpdateMessage(message string) {
	lw.manager.SetMessage(lw.viewName, message)
}
