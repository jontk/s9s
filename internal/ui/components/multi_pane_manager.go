package components

import (
	"fmt"
	"sync"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// PaneType defines the type of content in a pane
type PaneType string

const (
	PaneTypeView     PaneType = "view"
	PaneTypeTerminal PaneType = "terminal"
	PaneTypeLog      PaneType = "log"
	PaneTypeMonitor  PaneType = "monitor"
	PaneTypeEditor   PaneType = "editor"
)

// Pane represents a single pane in the multi-pane interface
type Pane struct {
	ID         string
	Title      string
	Type       PaneType
	Content    tview.Primitive
	Active     bool
	Closable   bool
	Resizable  bool
	MinWidth   int
	MinHeight  int
	LastFocus  time.Time
	Metadata   map[string]interface{}
	OnClose    func(string) error
	OnActivate func(string)
	OnResize   func(string, int, int)
}

// PaneLayout defines how panes are arranged
type PaneLayout string

const (
	LayoutHorizontal PaneLayout = "horizontal"
	LayoutVertical   PaneLayout = "vertical"
	LayoutGrid       PaneLayout = "grid"
	LayoutTabs       PaneLayout = "tabs"
)

// MultiPaneManager manages multiple panes in a flexible layout
type MultiPaneManager struct {
	mu         sync.RWMutex
	app        *tview.Application
	container  *tview.Flex
	panes      map[string]*Pane
	paneOrder  []string
	activePane string
	layout     PaneLayout
	// TODO(lint): Review unused code - field splitRatio is unused
	// splitRatio     []int // Ratios for splits
	maxPanes  int
	showTabs  bool
	tabBar    *tview.TextView
	statusBar *tview.TextView

	// Navigation
	navMode bool
	// TODO(lint): Review unused code - field navIndex is unused
	// navIndex       int

	// Callbacks
	onPaneSwitch   func(oldPane, newPane string)
	onLayoutChange func(PaneLayout)

	// Keyboard shortcuts
	shortcuts     map[tcell.Key]func()
	charShortcuts map[rune]func()
}

// NewMultiPaneManager creates a new multi-pane manager
func NewMultiPaneManager(app *tview.Application) *MultiPaneManager {
	mpm := &MultiPaneManager{
		app:           app,
		container:     tview.NewFlex(),
		panes:         make(map[string]*Pane),
		paneOrder:     make([]string, 0),
		layout:        LayoutTabs,
		maxPanes:      8,
		showTabs:      true,
		shortcuts:     make(map[tcell.Key]func()),
		charShortcuts: make(map[rune]func()),
	}

	mpm.initializeUI()
	mpm.setupKeyboardShortcuts()

	return mpm
}

// initializeUI sets up the basic UI structure
func (mpm *MultiPaneManager) initializeUI() {
	// Create tab bar
	mpm.tabBar = tview.NewTextView()
	mpm.tabBar.SetDynamicColors(true)
	mpm.tabBar.SetRegions(true)
	mpm.tabBar.SetWrap(false)
	mpm.tabBar.SetText("[yellow]No panes open[white]")

	// Create status bar
	mpm.statusBar = tview.NewTextView()
	mpm.statusBar.SetDynamicColors(true)
	mpm.statusBar.SetText("[green]Ready[white] | Use Ctrl+T for new tab, Ctrl+W to close, Tab/Shift+Tab to navigate")

	// Setup container
	mpm.container.SetDirection(tview.FlexRow)
	if mpm.showTabs {
		mpm.container.AddItem(mpm.tabBar, 1, 0, false)
	}
	// Content area will be added dynamically
	mpm.container.AddItem(mpm.statusBar, 1, 0, false)

	// Set up input capture for the container
	mpm.container.SetInputCapture(mpm.handleInput)
}

// setupKeyboardShortcuts configures keyboard navigation
func (mpm *MultiPaneManager) setupKeyboardShortcuts() {
	// Function key shortcuts
	mpm.shortcuts[tcell.KeyTab] = mpm.NextPane
	mpm.shortcuts[tcell.KeyBacktab] = mpm.PreviousPane
	mpm.shortcuts[tcell.KeyCtrlT] = func() { _ = mpm.NewPane("terminal", "Terminal", PaneTypeTerminal, nil) }
	mpm.shortcuts[tcell.KeyCtrlW] = mpm.CloseActivePane
	mpm.shortcuts[tcell.KeyCtrlN] = mpm.ToggleNavMode
	mpm.shortcuts[tcell.KeyF2] = mpm.RenameActivePane
	mpm.shortcuts[tcell.KeyF3] = mpm.SplitHorizontal
	mpm.shortcuts[tcell.KeyF4] = mpm.SplitVertical
	mpm.shortcuts[tcell.KeyF11] = mpm.ToggleFullscreen

	// Character shortcuts
	mpm.charShortcuts['1'] = func() { mpm.SwitchToPane(0) }
	mpm.charShortcuts['2'] = func() { mpm.SwitchToPane(1) }
	mpm.charShortcuts['3'] = func() { mpm.SwitchToPane(2) }
	mpm.charShortcuts['4'] = func() { mpm.SwitchToPane(3) }
	mpm.charShortcuts['5'] = func() { mpm.SwitchToPane(4) }
	mpm.charShortcuts['6'] = func() { mpm.SwitchToPane(5) }
	mpm.charShortcuts['7'] = func() { mpm.SwitchToPane(6) }
	mpm.charShortcuts['8'] = func() { mpm.SwitchToPane(7) }
	mpm.charShortcuts['9'] = func() { mpm.SwitchToPane(8) }
}

// handleInput processes keyboard input
func (mpm *MultiPaneManager) handleInput(event *tcell.EventKey) *tcell.EventKey {
	mpm.mu.Lock()
	defer mpm.mu.Unlock()

	// Check for function key shortcuts
	if handler, exists := mpm.shortcuts[event.Key()]; exists {
		handler()
		return nil
	}

	// Check for character shortcuts in navigation mode
	if mpm.navMode && event.Key() == tcell.KeyRune {
		if handler, exists := mpm.charShortcuts[event.Rune()]; exists {
			handler()
			return nil
		}
	}

	// Pass through to active pane if not handled
	if mpm.activePane != "" {
		pane := mpm.panes[mpm.activePane]
		if pane != nil && pane.Content != nil {
			// Try to forward the event to the pane content
			if inputCapture, ok := pane.Content.(interface {
				SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey)
			}); ok {
				// Let the pane handle it first
				_ = inputCapture
			}
		}
	}

	return event
}

// AddPane adds a new pane to the manager
func (mpm *MultiPaneManager) AddPane(pane *Pane) error {
	mpm.mu.Lock()
	defer mpm.mu.Unlock()

	if len(mpm.panes) >= mpm.maxPanes {
		return fmt.Errorf("maximum number of panes (%d) reached", mpm.maxPanes)
	}

	if _, exists := mpm.panes[pane.ID]; exists {
		return fmt.Errorf("pane with ID %s already exists", pane.ID)
	}

	// Set defaults
	if pane.MinWidth == 0 {
		pane.MinWidth = 20
	}
	if pane.MinHeight == 0 {
		pane.MinHeight = 5
	}
	pane.LastFocus = time.Now()

	// Add to collections
	mpm.panes[pane.ID] = pane
	mpm.paneOrder = append(mpm.paneOrder, pane.ID)

	// Set as active if it's the first pane
	if len(mpm.panes) == 1 {
		mpm.activePane = pane.ID
		pane.Active = true
		// Call activate callback for first pane
		if pane.OnActivate != nil {
			pane.OnActivate(pane.ID)
		}
	}

	// Refresh layout
	mpm.refreshLayout()
	mpm.updateTabBar()
	mpm.updateStatusBar()

	return nil
}

// NewPane creates and adds a new pane
func (mpm *MultiPaneManager) NewPane(id, title string, paneType PaneType, content tview.Primitive) error {
	// Generate unique ID if empty
	if id == "" {
		id = fmt.Sprintf("%s_%d", paneType, time.Now().UnixNano())
	}

	// Create default content if none provided
	if content == nil {
		content = mpm.createDefaultContent(paneType, title)
	}

	pane := &Pane{
		ID:        id,
		Title:     title,
		Type:      paneType, // Ensure type is preserved
		Content:   content,
		Closable:  true,
		Resizable: true,
		Metadata:  make(map[string]interface{}),
	}

	return mpm.AddPane(pane)
}

// createDefaultContent creates default content for different pane types
func (mpm *MultiPaneManager) createDefaultContent(paneType PaneType, title string) tview.Primitive {
	switch paneType {
	case PaneTypeTerminal:
		textView := tview.NewTextView()
		textView.SetTitle(fmt.Sprintf(" %s ", title))
		textView.SetBorder(true)
		textView.SetDynamicColors(true)
		textView.SetText(fmt.Sprintf("[green]%s Terminal[white]\n\nTerminal session would be initialized here.\nThis is a placeholder for actual terminal functionality.", title))
		return textView

	case PaneTypeLog:
		textView := tview.NewTextView()
		textView.SetTitle(fmt.Sprintf(" %s Logs ", title))
		textView.SetBorder(true)
		textView.SetScrollable(true)
		textView.SetDynamicColors(true)
		textView.SetText("[yellow]Log viewer initialized[white]\n\nLogs would be displayed here...")
		return textView

	case PaneTypeMonitor:
		textView := tview.NewTextView()
		textView.SetTitle(fmt.Sprintf(" %s Monitor ", title))
		textView.SetBorder(true)
		textView.SetDynamicColors(true)
		textView.SetText("[cyan]System Monitor[white]\n\nReal-time metrics would be displayed here...")
		return textView

	default:
		textView := tview.NewTextView()
		textView.SetTitle(fmt.Sprintf(" %s ", title))
		textView.SetBorder(true)
		textView.SetText(fmt.Sprintf("Content for %s pane", title))
		return textView
	}
}

// RemovePane removes a pane from the manager
func (mpm *MultiPaneManager) RemovePane(id string) error {
	mpm.mu.Lock()
	defer mpm.mu.Unlock()

	pane, exists := mpm.panes[id]
	if !exists {
		return fmt.Errorf("pane %s not found", id)
	}

	// Call close callback if defined
	if pane.OnClose != nil {
		if err := pane.OnClose(id); err != nil {
			return fmt.Errorf("failed to close pane: %w", err)
		}
	}

	// Remove from collections
	delete(mpm.panes, id)

	// Remove from order slice
	newOrder := make([]string, 0, len(mpm.paneOrder)-1)
	activeIndex := -1
	for i, paneID := range mpm.paneOrder {
		if paneID != id {
			newOrder = append(newOrder, paneID)
		} else {
			activeIndex = i
		}
	}
	mpm.paneOrder = newOrder

	// Handle active pane change
	if mpm.activePane == id {
		if len(mpm.paneOrder) > 0 {
			// Switch to next pane or previous if we were at the end
			if activeIndex >= len(mpm.paneOrder) {
				activeIndex = len(mpm.paneOrder) - 1
			}
			if activeIndex >= 0 {
				mpm.setActivePane(mpm.paneOrder[activeIndex])
			}
		} else {
			mpm.activePane = ""
		}
	}

	// Refresh layout
	mpm.refreshLayout()
	mpm.updateTabBar()
	mpm.updateStatusBar()

	return nil
}

// NextPane switches to the next pane
func (mpm *MultiPaneManager) NextPane() {
	mpm.mu.Lock()
	defer mpm.mu.Unlock()

	if len(mpm.paneOrder) <= 1 {
		return
	}

	currentIndex := mpm.getCurrentPaneIndex()
	nextIndex := (currentIndex + 1) % len(mpm.paneOrder)
	mpm.setActivePane(mpm.paneOrder[nextIndex])
}

// PreviousPane switches to the previous pane
func (mpm *MultiPaneManager) PreviousPane() {
	mpm.mu.Lock()
	defer mpm.mu.Unlock()

	if len(mpm.paneOrder) <= 1 {
		return
	}

	currentIndex := mpm.getCurrentPaneIndex()
	prevIndex := (currentIndex - 1 + len(mpm.paneOrder)) % len(mpm.paneOrder)
	mpm.setActivePane(mpm.paneOrder[prevIndex])
}

// SwitchToPane switches to a pane by index
func (mpm *MultiPaneManager) SwitchToPane(index int) {
	mpm.mu.Lock()
	defer mpm.mu.Unlock()

	if index < 0 || index >= len(mpm.paneOrder) {
		return
	}

	mpm.setActivePane(mpm.paneOrder[index])
}

// setActivePane sets the active pane (internal, assumes lock held)
func (mpm *MultiPaneManager) setActivePane(id string) {
	if mpm.activePane == id {
		return
	}

	// Deactivate current pane
	if mpm.activePane != "" {
		if pane := mpm.panes[mpm.activePane]; pane != nil {
			pane.Active = false
		}
	}

	oldPane := mpm.activePane
	mpm.activePane = id

	// Activate new pane
	if pane := mpm.panes[id]; pane != nil {
		pane.Active = true
		pane.LastFocus = time.Now()

		if pane.OnActivate != nil {
			pane.OnActivate(id)
		}
	}

	// Refresh UI
	mpm.updateTabBar()
	mpm.updateStatusBar()

	// Call callback
	if mpm.onPaneSwitch != nil {
		mpm.onPaneSwitch(oldPane, id)
	}

	// Set focus to the active pane
	if pane := mpm.panes[id]; pane != nil && pane.Content != nil {
		mpm.app.SetFocus(pane.Content)
	}
}

// getCurrentPaneIndex returns the index of the current active pane
func (mpm *MultiPaneManager) getCurrentPaneIndex() int {
	for i, id := range mpm.paneOrder {
		if id == mpm.activePane {
			return i
		}
	}
	return 0
}

// CloseActivePane closes the currently active pane
func (mpm *MultiPaneManager) CloseActivePane() {
	mpm.mu.RLock()
	activePane := mpm.activePane
	mpm.mu.RUnlock()

	if activePane != "" {
		_ = mpm.RemovePane(activePane)
	}
}

// ToggleNavMode toggles navigation mode for keyboard shortcuts
func (mpm *MultiPaneManager) ToggleNavMode() {
	mpm.mu.Lock()
	defer mpm.mu.Unlock()

	mpm.navMode = !mpm.navMode
	mpm.updateStatusBar()
}

// RenameActivePane allows renaming the active pane
func (mpm *MultiPaneManager) RenameActivePane() {
	// This would show a modal dialog for renaming
	// Implementation would depend on the specific UI framework usage
	mpm.updateStatusBar()
}

// SplitHorizontal splits the current pane horizontally
func (mpm *MultiPaneManager) SplitHorizontal() {
	mpm.mu.Lock()
	defer mpm.mu.Unlock()

	if mpm.layout != LayoutHorizontal {
		mpm.layout = LayoutHorizontal
		mpm.refreshLayout()
	}
}

// SplitVertical splits the current pane vertically
func (mpm *MultiPaneManager) SplitVertical() {
	mpm.mu.Lock()
	defer mpm.mu.Unlock()

	if mpm.layout != LayoutVertical {
		mpm.layout = LayoutVertical
		mpm.refreshLayout()
	}
}

// ToggleFullscreen toggles fullscreen mode for the active pane
func (mpm *MultiPaneManager) ToggleFullscreen() {
	// Implementation for fullscreen toggle
	mpm.updateStatusBar()
}

// SetLayout changes the pane layout
func (mpm *MultiPaneManager) SetLayout(layout PaneLayout) {
	mpm.mu.Lock()
	defer mpm.mu.Unlock()

	if mpm.layout != layout {
		mpm.layout = layout
		mpm.refreshLayout()

		if mpm.onLayoutChange != nil {
			mpm.onLayoutChange(layout)
		}
	}
}

// refreshLayout rebuilds the UI layout
func (mpm *MultiPaneManager) refreshLayout() {
	// Clear current content area (keep tab bar and status bar)
	items := mpm.container.GetItemCount()
	if items > 2 {
		// Remove content area items (everything between tab bar and status bar)
		for i := items - 2; i > 1; i-- {
			mpm.container.RemoveItem(mpm.container.GetItem(i - 1))
		}
	}

	if len(mpm.panes) == 0 {
		return
	}

	var contentArea tview.Primitive

	switch mpm.layout {
	case LayoutTabs:
		contentArea = mpm.createTabLayout()
	case LayoutHorizontal:
		contentArea = mpm.createHorizontalLayout()
	case LayoutVertical:
		contentArea = mpm.createVerticalLayout()
	case LayoutGrid:
		contentArea = mpm.createGridLayout()
	default:
		contentArea = mpm.createTabLayout()
	}

	// Insert content area between tab bar and status bar
	// Remove existing content area first
	itemCount := mpm.container.GetItemCount()

	// Add content area before status bar
	if itemCount > 1 {
		// We need to rebuild the container with the new content area
		mpm.container.Clear()
		if mpm.showTabs {
			mpm.container.AddItem(mpm.tabBar, 1, 0, false)
		}
		mpm.container.AddItem(contentArea, 0, 1, true)
		mpm.container.AddItem(mpm.statusBar, 1, 0, false)
	} else {
		mpm.container.AddItem(contentArea, 0, 1, true)
	}
}

// createTabLayout creates a tabbed layout
func (mpm *MultiPaneManager) createTabLayout() tview.Primitive {
	if mpm.activePane == "" || len(mpm.panes) == 0 {
		placeholder := tview.NewTextView()
		placeholder.SetText("[yellow]No active pane[white]")
		placeholder.SetTextAlign(tview.AlignCenter)
		return placeholder
	}

	activePane := mpm.panes[mpm.activePane]
	if activePane != nil && activePane.Content != nil {
		return activePane.Content
	}

	placeholder := tview.NewTextView()
	placeholder.SetText("[red]Error: Active pane content not found[white]")
	placeholder.SetTextAlign(tview.AlignCenter)
	return placeholder
}

// createHorizontalLayout creates a horizontal split layout
func (mpm *MultiPaneManager) createHorizontalLayout() tview.Primitive {
	flex := tview.NewFlex()
	flex.SetDirection(tview.FlexColumn)

	for _, id := range mpm.paneOrder {
		if pane := mpm.panes[id]; pane != nil && pane.Content != nil {
			flex.AddItem(pane.Content, 0, 1, pane.Active)
		}
	}

	return flex
}

// createVerticalLayout creates a vertical split layout
func (mpm *MultiPaneManager) createVerticalLayout() tview.Primitive {
	flex := tview.NewFlex()
	flex.SetDirection(tview.FlexRow)

	for _, id := range mpm.paneOrder {
		if pane := mpm.panes[id]; pane != nil && pane.Content != nil {
			flex.AddItem(pane.Content, 0, 1, pane.Active)
		}
	}

	return flex
}

// createGridLayout creates a grid layout
func (mpm *MultiPaneManager) createGridLayout() tview.Primitive {
	// Simple 2x2 grid for now
	if len(mpm.panes) <= 2 {
		return mpm.createHorizontalLayout()
	}

	mainFlex := tview.NewFlex()
	mainFlex.SetDirection(tview.FlexRow)

	// Top row
	topFlex := tview.NewFlex()
	topFlex.SetDirection(tview.FlexColumn)

	// Bottom row
	bottomFlex := tview.NewFlex()
	bottomFlex.SetDirection(tview.FlexColumn)

	count := 0
	for _, id := range mpm.paneOrder {
		if pane := mpm.panes[id]; pane != nil && pane.Content != nil {
			if count < 2 {
				topFlex.AddItem(pane.Content, 0, 1, pane.Active)
			} else {
				bottomFlex.AddItem(pane.Content, 0, 1, pane.Active)
			}
			count++
		}
	}

	if topFlex.GetItemCount() > 0 {
		mainFlex.AddItem(topFlex, 0, 1, false)
	}
	if bottomFlex.GetItemCount() > 0 {
		mainFlex.AddItem(bottomFlex, 0, 1, false)
	}

	return mainFlex
}

// updateTabBar updates the tab bar display
func (mpm *MultiPaneManager) updateTabBar() {
	if !mpm.showTabs {
		return
	}

	if len(mpm.panes) == 0 {
		mpm.tabBar.SetText("[yellow]No panes open[white]")
		return
	}

	var tabText string
	for i, id := range mpm.paneOrder {
		pane := mpm.panes[id]
		if pane == nil {
			continue
		}

		// Format tab with number and title
		tabPrefix := fmt.Sprintf("%d:", i+1)

		if pane.Active {
			tabText += fmt.Sprintf("[black:white]%s %s[white:black] ", tabPrefix, pane.Title)
		} else {
			tabText += fmt.Sprintf("[white]%s %s[white] ", tabPrefix, pane.Title)
		}
	}

	mpm.tabBar.SetText(tabText)
}

// updateStatusBar updates the status bar
func (mpm *MultiPaneManager) updateStatusBar() {
	status := "[green]Ready[white]"

	if len(mpm.panes) > 0 {
		status = fmt.Sprintf("[cyan]%d panes[white]", len(mpm.panes))

		if mpm.activePane != "" {
			if pane := mpm.panes[mpm.activePane]; pane != nil {
				status += fmt.Sprintf(" | Active: [yellow]%s[white]", pane.Title)
			}
		}
	}

	if mpm.navMode {
		status += " | [yellow]NAV MODE[white] - Use 1-9 to switch panes"
	}

	// Add keyboard shortcuts
	shortcuts := " | [green]Ctrl+T[white]:New [green]Ctrl+W[white]:Close [green]Tab[white]:Next [green]F3/F4[white]:Split"
	status += shortcuts

	mpm.statusBar.SetText(status)
}

// GetContainer returns the main container
func (mpm *MultiPaneManager) GetContainer() *tview.Flex {
	return mpm.container
}

// GetActivePane returns the currently active pane
func (mpm *MultiPaneManager) GetActivePane() *Pane {
	mpm.mu.RLock()
	defer mpm.mu.RUnlock()

	if mpm.activePane == "" {
		return nil
	}

	return mpm.panes[mpm.activePane]
}

// GetPane returns a specific pane by ID
func (mpm *MultiPaneManager) GetPane(id string) *Pane {
	mpm.mu.RLock()
	defer mpm.mu.RUnlock()

	return mpm.panes[id]
}

// GetPaneCount returns the number of open panes
func (mpm *MultiPaneManager) GetPaneCount() int {
	mpm.mu.RLock()
	defer mpm.mu.RUnlock()

	return len(mpm.panes)
}

// SetOnPaneSwitch sets the callback for pane switching
func (mpm *MultiPaneManager) SetOnPaneSwitch(callback func(oldPane, newPane string)) {
	mpm.mu.Lock()
	defer mpm.mu.Unlock()

	mpm.onPaneSwitch = callback
}

// SetOnLayoutChange sets the callback for layout changes
func (mpm *MultiPaneManager) SetOnLayoutChange(callback func(PaneLayout)) {
	mpm.mu.Lock()
	defer mpm.mu.Unlock()

	mpm.onLayoutChange = callback
}

// Close closes all panes and cleans up resources
func (mpm *MultiPaneManager) Close() {
	mpm.mu.Lock()
	defer mpm.mu.Unlock()

	// Close all panes
	for id := range mpm.panes {
		if pane := mpm.panes[id]; pane != nil && pane.OnClose != nil {
			_ = pane.OnClose(id)
		}
	}

	// Clear collections
	mpm.panes = make(map[string]*Pane)
	mpm.paneOrder = make([]string, 0)
	mpm.activePane = ""

	// Update UI
	mpm.refreshLayout()
	mpm.updateTabBar()
	mpm.updateStatusBar()
}
