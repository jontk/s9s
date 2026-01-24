// Package navigation provides navigation management and keyboard input handling.
package navigation

import (
	"fmt"
	"strings"
	"sync"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// Mode defines different navigation modes
type Mode string

const (
	// NavigationModeNormal is the normal navigation mode.
	NavigationModeNormal Mode = "normal"
	// NavigationModeQuick is the quick access navigation mode.
	NavigationModeQuick Mode = "quick"
	// NavigationModeCommand is the command palette navigation mode.
	NavigationModeCommand Mode = "command"
	// NavigationModeSearch is the search navigation mode.
	NavigationModeSearch Mode = "search"
)

// NavigationMode is an alias for backward compatibility
type NavigationMode = Mode

// Target represents a navigable target
type Target struct {
	ID           string
	Name         string
	Description  string
	Type         string
	Component    tview.Primitive
	Shortcut     rune
	KeyBinding   tcell.Key
	Priority     int
	Visible      bool
	OnActivate   func()
	OnDeactivate func()
}

// NavigationTarget is an alias for backward compatibility
type NavigationTarget = Target

// History tracks navigation history
type History struct {
	items   []string
	current int
	maxSize int
}

// NavigationHistory is an alias for backward compatibility
type NavigationHistory = History

// Manager handles advanced navigation throughout the application
type Manager struct {
	mu             sync.RWMutex
	app            *tview.Application
	targets        map[string]*Target
	targetOrder    []string
	currentTarget  string
	mode           Mode
	history        *History
	quickAccessMap map[rune]string
	keyBindings    map[tcell.Key]string

	// UI Components
	breadcrumb     *tview.TextView
	quickHelpBar   *tview.TextView
	commandPalette *tview.InputField
	searchOverlay  *tview.TextView

	// State
	quickModeActive bool
	commandMode     bool
	searchMode      bool
	searchQuery     string
	searchResults   []string

	// Callbacks
	onNavigate   func(from, to string)
	onModeChange func(NavigationMode)
	onSearch     func(string) []string

	// Configuration
	enableBreadcrumb bool
	enableQuickHelp  bool
	enableHistory    bool
	maxHistorySize   int
}

// NewManager creates a new navigation manager
func NewManager(app *tview.Application) *Manager {
	nm := &Manager{
		app:              app,
		targets:          make(map[string]*NavigationTarget),
		targetOrder:      make([]string, 0),
		mode:             NavigationModeNormal,
		quickAccessMap:   make(map[rune]string),
		keyBindings:      make(map[tcell.Key]string),
		enableBreadcrumb: true,
		enableQuickHelp:  true,
		enableHistory:    true,
		maxHistorySize:   50,
	}

	nm.history = &History{
		items:   make([]string, 0),
		current: -1,
		maxSize: nm.maxHistorySize,
	}

	nm.initializeUI()
	nm.setupDefaultBindings()

	return nm
}

// NewNavigationManager is an alias for backward compatibility
func NewNavigationManager(app *tview.Application) *Manager {
	return NewManager(app)
}

// initializeUI sets up the navigation UI components
func (nm *Manager) initializeUI() {
	// Breadcrumb navigation
	nm.breadcrumb = tview.NewTextView()
	nm.breadcrumb.SetDynamicColors(true)
	nm.breadcrumb.SetText("[gray]Home[white]")

	// Quick help bar
	nm.quickHelpBar = tview.NewTextView()
	nm.quickHelpBar.SetDynamicColors(true)
	nm.quickHelpBar.SetText(nm.getQuickHelpText())

	// Command palette
	nm.commandPalette = tview.NewInputField()
	nm.commandPalette.SetLabel("Command: ")
	nm.commandPalette.SetFieldBackgroundColor(tcell.ColorBlack)
	nm.commandPalette.SetFieldTextColor(tcell.ColorWhite)

	// Search overlay
	nm.searchOverlay = tview.NewTextView()
	nm.searchOverlay.SetDynamicColors(true)
	nm.searchOverlay.SetTextAlign(tview.AlignCenter)
	nm.searchOverlay.SetBackgroundColor(tcell.ColorBlack)
}

// setupDefaultBindings sets up default key bindings
func (nm *Manager) setupDefaultBindings() {
	// Function key bindings
	nm.keyBindings[tcell.KeyF1] = "help"
	nm.keyBindings[tcell.KeyF2] = "search"
	nm.keyBindings[tcell.KeyF3] = "command"
	nm.keyBindings[tcell.KeyF4] = "quick"
	nm.keyBindings[tcell.KeyF5] = "refresh"
	nm.keyBindings[tcell.KeyCtrlP] = "command"
	nm.keyBindings[tcell.KeyCtrlF] = "search"
	nm.keyBindings[tcell.KeyCtrlH] = "history"
	nm.keyBindings[tcell.KeyCtrlB] = "back"
	nm.keyBindings[tcell.KeyCtrlN] = "forward"

	// Quick access shortcuts (1-9, a-z)
	shortcuts := "123456789abcdefghijklmnopqrstuvwxyz"
	for _, char := range shortcuts {
		nm.quickAccessMap[char] = ""
	}
}

// RegisterTarget registers a navigation target
func (nm *Manager) RegisterTarget(target *NavigationTarget) error {
	nm.mu.Lock()
	defer nm.mu.Unlock()

	if target.ID == "" {
		return fmt.Errorf("target ID cannot be empty")
	}

	// Check for duplicate shortcuts
	if target.Shortcut != 0 {
		if existingID, exists := nm.quickAccessMap[target.Shortcut]; exists && existingID != "" {
			return fmt.Errorf("shortcut '%c' already assigned to target %s", target.Shortcut, existingID)
		}
	}

	// Check for duplicate key bindings
	if target.KeyBinding != tcell.KeyNUL {
		if existingID, exists := nm.keyBindings[target.KeyBinding]; exists && existingID != target.ID {
			return fmt.Errorf("key binding already assigned to target %s", existingID)
		}
	}

	// Set defaults
	if target.Priority == 0 {
		target.Priority = 5
	}
	target.Visible = true

	// Register target
	nm.targets[target.ID] = target
	nm.targetOrder = append(nm.targetOrder, target.ID)

	// Register shortcuts
	if target.Shortcut != 0 {
		nm.quickAccessMap[target.Shortcut] = target.ID
	}
	if target.KeyBinding != tcell.KeyNUL {
		nm.keyBindings[target.KeyBinding] = target.ID
	}

	nm.updateQuickHelp()

	return nil
}

// UnregisterTarget removes a navigation target
func (nm *Manager) UnregisterTarget(id string) error {
	nm.mu.Lock()
	defer nm.mu.Unlock()

	target, exists := nm.targets[id]
	if !exists {
		return fmt.Errorf("target %s not found", id)
	}

	// Remove from maps
	delete(nm.targets, id)

	// Remove from order slice
	newOrder := make([]string, 0, len(nm.targetOrder)-1)
	for _, targetID := range nm.targetOrder {
		if targetID != id {
			newOrder = append(newOrder, targetID)
		}
	}
	nm.targetOrder = newOrder

	// Remove shortcuts
	if target.Shortcut != 0 {
		nm.quickAccessMap[target.Shortcut] = ""
	}
	if target.KeyBinding != tcell.KeyNUL {
		delete(nm.keyBindings, target.KeyBinding)
	}

	// Handle current target change
	if nm.currentTarget == id {
		if len(nm.targetOrder) > 0 {
			_ = nm.navigateToTargetLocked(nm.targetOrder[0], false)
		} else {
			nm.currentTarget = ""
		}
	}

	nm.updateQuickHelp()

	return nil
}

// NavigateTo navigates to a specific target
func (nm *Manager) NavigateTo(targetID string) error {
	return nm.navigateToTarget(targetID, true)
}

// navigateToTarget performs the actual navigation (acquires lock)
func (nm *Manager) navigateToTarget(targetID string, addToHistory bool) error {
	nm.mu.Lock()
	defer nm.mu.Unlock()

	return nm.navigateToTargetLocked(targetID, addToHistory)
}

// navigateToTargetLocked performs navigation assuming lock is already held
func (nm *Manager) navigateToTargetLocked(targetID string, addToHistory bool) error {
	target, exists := nm.targets[targetID]
	if !exists {
		return fmt.Errorf("target %s not found", targetID)
	}

	if !target.Visible {
		return fmt.Errorf("target %s is not visible", targetID)
	}

	// Deactivate current target
	if nm.currentTarget != "" {
		if currentTarget := nm.targets[nm.currentTarget]; currentTarget != nil && currentTarget.OnDeactivate != nil {
			currentTarget.OnDeactivate()
		}
	}

	oldTarget := nm.currentTarget
	nm.currentTarget = targetID

	// Add to history
	if addToHistory && nm.enableHistory {
		nm.addToHistory(targetID)
	}

	// Activate new target
	if target.OnActivate != nil {
		target.OnActivate()
	}

	// Update breadcrumb
	nm.updateBreadcrumb()

	// Call callback
	if nm.onNavigate != nil {
		nm.onNavigate(oldTarget, targetID)
	}

	return nil
}

// HandleInput processes navigation input
func (nm *Manager) HandleInput(event *tcell.EventKey) *tcell.EventKey {
	nm.mu.Lock()
	defer nm.mu.Unlock()

	// Handle mode-specific input
	switch nm.mode {
	case NavigationModeCommand:
		return nm.handleCommandInput(event)
	case NavigationModeSearch:
		return nm.handleSearchInput(event)
	case NavigationModeQuick:
		return nm.handleQuickInput(event)
	}

	// Handle global key bindings
	if targetID, exists := nm.keyBindings[event.Key()]; exists {
		nm.handleSpecialCommand(targetID)
		return nil
	}

	// Handle character shortcuts in quick mode
	if nm.quickModeActive && event.Key() == tcell.KeyRune {
		if targetID, exists := nm.quickAccessMap[event.Rune()]; exists && targetID != "" {
			_ = nm.navigateToTargetLocked(targetID, true)
			nm.ExitQuickMode()
			return nil
		}
	}

	return event
}

// handleCommandInput handles input in command mode
func (nm *Manager) handleCommandInput(event *tcell.EventKey) *tcell.EventKey {
	switch event.Key() {
	case tcell.KeyEsc:
		nm.ExitCommandMode()
		return nil
	case tcell.KeyEnter:
		command := nm.commandPalette.GetText()
		nm.executeCommand(command)
		nm.ExitCommandMode()
		return nil
	}

	return event
}

// handleSearchInput handles input in search mode
// Assumes lock is already held by caller (HandleInput)
func (nm *Manager) handleSearchInput(event *tcell.EventKey) *tcell.EventKey {
	switch event.Key() {
	case tcell.KeyEsc:
		nm.ExitSearchMode()
		return nil
	case tcell.KeyEnter:
		if len(nm.searchResults) > 0 {
			_ = nm.navigateToTargetLocked(nm.searchResults[0], true)
			nm.ExitSearchMode()
		}
		return nil
	case tcell.KeyRune:
		nm.searchQuery += string(event.Rune())
		nm.performSearch()
		return nil
	case tcell.KeyBackspace, tcell.KeyBackspace2:
		if len(nm.searchQuery) > 0 {
			nm.searchQuery = nm.searchQuery[:len(nm.searchQuery)-1]
			nm.performSearch()
		}
		return nil
	}

	return event
}

// handleQuickInput handles input in quick navigation mode
// Assumes lock is already held by caller (HandleInput)
func (nm *Manager) handleQuickInput(event *tcell.EventKey) *tcell.EventKey {
	switch event.Key() {
	case tcell.KeyEsc:
		nm.ExitQuickMode()
		return nil
	case tcell.KeyRune:
		if targetID, exists := nm.quickAccessMap[event.Rune()]; exists && targetID != "" {
			_ = nm.navigateToTargetLocked(targetID, true)
			nm.ExitQuickMode()
			return nil
		}
	}

	return event
}

// handleSpecialCommand handles special navigation commands
func (nm *Manager) handleSpecialCommand(command string) {
	switch command {
	case "help":
		nm.ShowHelp()
	case "search":
		nm.EnterSearchMode()
	case "command":
		nm.EnterCommandMode()
	case "quick":
		nm.EnterQuickMode()
	case "refresh":
		nm.RefreshCurrentTarget()
	case "back":
		nm.GoBack()
	case "forward":
		nm.GoForward()
	case "history":
		nm.ShowHistory()
	}
}

// EnterQuickMode enters quick navigation mode
func (nm *Manager) EnterQuickMode() {
	nm.mode = NavigationModeQuick
	nm.quickModeActive = true

	nm.updateQuickHelp()

	if nm.onModeChange != nil {
		nm.onModeChange(nm.mode)
	}
}

// ExitQuickMode exits quick navigation mode
func (nm *Manager) ExitQuickMode() {
	nm.mode = NavigationModeNormal
	nm.quickModeActive = false

	nm.updateQuickHelp()

	if nm.onModeChange != nil {
		nm.onModeChange(nm.mode)
	}
}

// EnterCommandMode enters command palette mode
func (nm *Manager) EnterCommandMode() {
	nm.mode = NavigationModeCommand
	nm.commandMode = true

	nm.commandPalette.SetText("")

	if nm.onModeChange != nil {
		nm.onModeChange(nm.mode)
	}
}

// ExitCommandMode exits command palette mode
func (nm *Manager) ExitCommandMode() {
	nm.mode = NavigationModeNormal
	nm.commandMode = false

	if nm.onModeChange != nil {
		nm.onModeChange(nm.mode)
	}
}

// EnterSearchMode enters search mode
func (nm *Manager) EnterSearchMode() {
	nm.mode = NavigationModeSearch
	nm.searchMode = true
	nm.searchQuery = ""
	nm.searchResults = nil

	nm.updateSearchOverlay()

	if nm.onModeChange != nil {
		nm.onModeChange(nm.mode)
	}
}

// ExitSearchMode exits search mode
func (nm *Manager) ExitSearchMode() {
	nm.mode = NavigationModeNormal
	nm.searchMode = false
	nm.searchQuery = ""
	nm.searchResults = nil

	if nm.onModeChange != nil {
		nm.onModeChange(nm.mode)
	}
}

// performSearch performs a search and updates results
func (nm *Manager) performSearch() {
	if nm.onSearch != nil {
		nm.searchResults = nm.onSearch(nm.searchQuery)
	} else {
		// Default search implementation
		nm.searchResults = nm.defaultSearch(nm.searchQuery)
	}

	nm.updateSearchOverlay()
}

// defaultSearch provides default search functionality
func (nm *Manager) defaultSearch(query string) []string {
	var results []string

	query = strings.ToLower(query)

	for _, targetID := range nm.targetOrder {
		target := nm.targets[targetID]
		if target == nil || !target.Visible {
			continue
		}

		if strings.Contains(strings.ToLower(target.Name), query) ||
			strings.Contains(strings.ToLower(target.Description), query) ||
			strings.Contains(strings.ToLower(target.Type), query) {
			results = append(results, targetID)
		}
	}

	return results
}

// executeCommand executes a command from the command palette
// Assumes lock is already held by caller (via handleCommandInput from HandleInput)
func (nm *Manager) executeCommand(command string) {
	// Parse and execute command
	parts := strings.Fields(command)
	if len(parts) == 0 {
		return
	}

	cmd := strings.ToLower(parts[0])

	switch cmd {
	case "go", "nav", "navigate":
		if len(parts) >= 2 {
			targetName := strings.Join(parts[1:], " ")
			if targetID := nm.findTargetByName(targetName); targetID != "" {
				_ = nm.navigateToTargetLocked(targetID, true)
			}
		}
	case "search", "find":
		if len(parts) >= 2 {
			query := strings.Join(parts[1:], " ")
			nm.searchQuery = query
			nm.performSearch()
		}
	case "help":
		nm.ShowHelp()
	case "back":
		nm.GoBack()
	case "forward":
		nm.GoForward()
	case "refresh":
		nm.RefreshCurrentTarget()
	}
}

// findTargetByName finds a target by name
func (nm *Manager) findTargetByName(name string) string {
	name = strings.ToLower(name)

	for targetID, target := range nm.targets {
		if strings.ToLower(target.Name) == name {
			return targetID
		}
	}

	return ""
}

// GoBack navigates back in history
func (nm *Manager) GoBack() {
	if nm.history.current > 0 {
		nm.history.current--
		targetID := nm.history.items[nm.history.current]
		_ = nm.navigateToTarget(targetID, false)
	}
}

// GoForward navigates forward in history
func (nm *Manager) GoForward() {
	if nm.history.current < len(nm.history.items)-1 {
		nm.history.current++
		targetID := nm.history.items[nm.history.current]
		_ = nm.navigateToTarget(targetID, false)
	}
}

// addToHistory adds a target to navigation history
func (nm *Manager) addToHistory(targetID string) {
	// Remove items after current position
	nm.history.items = nm.history.items[:nm.history.current+1]

	// Add new item
	nm.history.items = append(nm.history.items, targetID)
	nm.history.current = len(nm.history.items) - 1

	// Trim history if too long
	if len(nm.history.items) > nm.history.maxSize {
		nm.history.items = nm.history.items[1:]
		nm.history.current--
	}
}

// ShowHelp displays navigation help
func (nm *Manager) ShowHelp() {
	// This would show a help modal with navigation instructions
}

// ShowHistory displays navigation history
func (nm *Manager) ShowHistory() {
	// This would show a history modal
}

// RefreshCurrentTarget refreshes the current target
func (nm *Manager) RefreshCurrentTarget() {
	if nm.currentTarget != "" {
		if target := nm.targets[nm.currentTarget]; target != nil && target.OnActivate != nil {
			target.OnActivate()
		}
	}
}

// updateBreadcrumb updates the breadcrumb navigation
func (nm *Manager) updateBreadcrumb() {
	if !nm.enableBreadcrumb {
		return
	}

	var breadcrumbText string

	if nm.currentTarget != "" {
		if target := nm.targets[nm.currentTarget]; target != nil {
			breadcrumbText = fmt.Sprintf("[cyan]%s[white]", target.Name)
		}
	} else {
		breadcrumbText = "[gray]Home[white]"
	}

	nm.breadcrumb.SetText(breadcrumbText)
}

// updateQuickHelp updates the quick help display
func (nm *Manager) updateQuickHelp() {
	if !nm.enableQuickHelp {
		return
	}

	nm.quickHelpBar.SetText(nm.getQuickHelpText())
}

// getQuickHelpText returns the quick help text based on current mode
func (nm *Manager) getQuickHelpText() string {
	switch nm.mode {
	case NavigationModeQuick:
		return "[yellow]QUICK MODE[white] - Press shortcut key or Esc to cancel"
	case NavigationModeCommand:
		return "[yellow]COMMAND MODE[white] - Type command and press Enter, Esc to cancel"
	case NavigationModeSearch:
		return "[yellow]SEARCH MODE[white] - Type to search, Enter to navigate, Esc to cancel"
	default:
		shortcuts := make([]string, 0)
		for char, targetID := range nm.quickAccessMap {
			if targetID != "" {
				if target := nm.targets[targetID]; target != nil && target.Visible {
					shortcuts = append(shortcuts, fmt.Sprintf("[green]%c[white]:%s", char, target.Name))
				}
			}
		}

		if len(shortcuts) > 8 {
			shortcuts = shortcuts[:8]
			shortcuts = append(shortcuts, "...")
		}

		return fmt.Sprintf("Shortcuts: %s | [green]F4[white]:Quick [green]Ctrl+P[white]:Command", strings.Join(shortcuts, " "))
	}
}

// updateSearchOverlay updates the search overlay
func (nm *Manager) updateSearchOverlay() {
	if !nm.searchMode {
		return
	}

	text := fmt.Sprintf("[yellow]Search:[white] %s\n", nm.searchQuery)

	if len(nm.searchResults) > 0 {
		text += "[cyan]Results:[white]\n"
		for i, targetID := range nm.searchResults {
			if i >= 5 { // Limit displayed results
				text += "...\n"
				break
			}
			if target := nm.targets[targetID]; target != nil {
				text += fmt.Sprintf("  %s\n", target.Name)
			}
		}
	} else if nm.searchQuery != "" {
		text += "[red]No results found[white]"
	}

	nm.searchOverlay.SetText(text)
}

// GetBreadcrumb returns the breadcrumb component
func (nm *Manager) GetBreadcrumb() *tview.TextView {
	return nm.breadcrumb
}

// GetQuickHelpBar returns the quick help bar component
func (nm *Manager) GetQuickHelpBar() *tview.TextView {
	return nm.quickHelpBar
}

// GetCommandPalette returns the command palette component
func (nm *Manager) GetCommandPalette() *tview.InputField {
	return nm.commandPalette
}

// GetSearchOverlay returns the search overlay component
func (nm *Manager) GetSearchOverlay() *tview.TextView {
	return nm.searchOverlay
}

// GetCurrentTarget returns the current target ID
func (nm *Manager) GetCurrentTarget() string {
	nm.mu.RLock()
	defer nm.mu.RUnlock()
	return nm.currentTarget
}

// GetMode returns the current navigation mode
func (nm *Manager) GetMode() NavigationMode {
	nm.mu.RLock()
	defer nm.mu.RUnlock()
	return nm.mode
}

// SetOnNavigate sets the navigation callback
func (nm *Manager) SetOnNavigate(callback func(from, to string)) {
	nm.mu.Lock()
	defer nm.mu.Unlock()
	nm.onNavigate = callback
}

// SetOnModeChange sets the mode change callback
func (nm *Manager) SetOnModeChange(callback func(NavigationMode)) {
	nm.mu.Lock()
	defer nm.mu.Unlock()
	nm.onModeChange = callback
}

// SetOnSearch sets the search callback
func (nm *Manager) SetOnSearch(callback func(string) []string) {
	nm.mu.Lock()
	defer nm.mu.Unlock()
	nm.onSearch = callback
}
