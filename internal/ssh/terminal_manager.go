package ssh

import (
	"fmt"
	"sync"
	"time"

	"github.com/rivo/tview"
)

// Terminal represents an SSH terminal interface
type Terminal interface {
	tview.Primitive
	SendCommand(command string) error
	Resize(width, height int) error
	Close() error
	IsActive() bool
	GetNodeID() string
}

// TerminalInfo contains information about a terminal
type TerminalInfo struct {
	NodeID       string
	Hostname     string
	Username     string
	IsActive     bool
	ConnectedAt  time.Time
	LastActivity time.Time
}

// TerminalManager manages multiple SSH terminal sessions
type TerminalManager struct {
	app          *tview.Application
	terminals    map[string]Terminal
	order        []string // Maintains order of terminals
	current      int      // Current terminal index
	mu           sync.RWMutex
	maxTerminals int
	onSwitch     func(nodeID string)
}

// NewTerminalManager creates a new terminal manager
func NewTerminalManager(app *tview.Application) *TerminalManager {
	return &TerminalManager{
		app:          app,
		terminals:    make(map[string]Terminal),
		order:        make([]string, 0),
		current:      -1,
		maxTerminals: 10, // Default max terminals
	}
}

// SetMaxTerminals sets the maximum number of terminals
func (tm *TerminalManager) SetMaxTerminals(maxVal int) {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	tm.maxTerminals = maxVal
}

// SetOnSwitch sets the callback for terminal switching
func (tm *TerminalManager) SetOnSwitch(callback func(nodeID string)) {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	tm.onSwitch = callback
}

// AddTerminal adds a new terminal
func (tm *TerminalManager) AddTerminal(nodeID string, terminal Terminal) error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	// Check if terminal already exists
	if _, exists := tm.terminals[nodeID]; exists {
		return fmt.Errorf("terminal for node %s already exists", nodeID)
	}

	// Check max terminals
	if len(tm.terminals) >= tm.maxTerminals {
		return fmt.Errorf("maximum number of terminals (%d) reached", tm.maxTerminals)
	}

	// Add terminal
	tm.terminals[nodeID] = terminal
	tm.order = append(tm.order, nodeID)

	// Set as current if it's the first terminal
	if tm.current == -1 {
		tm.current = 0
	}

	return nil
}

// RemoveTerminal removes a terminal
func (tm *TerminalManager) RemoveTerminal(nodeID string) error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	terminal, exists := tm.terminals[nodeID]
	if !exists {
		return fmt.Errorf("terminal for node %s not found", nodeID)
	}

	// Close the terminal
	_ = terminal.Close()

	// Remove from map
	delete(tm.terminals, nodeID)

	// Remove from order slice
	newOrder := make([]string, 0, len(tm.order)-1)
	removedIndex := -1
	for i, id := range tm.order {
		if id != nodeID {
			newOrder = append(newOrder, id)
		} else {
			removedIndex = i
		}
	}
	tm.order = newOrder

	// Adjust current index if needed
	switch {
	case len(tm.order) == 0:
		tm.current = -1
	case tm.current >= len(tm.order):
		tm.current = len(tm.order) - 1
	case removedIndex < tm.current:
		tm.current--
	}

	return nil
}

// GetTerminal gets a terminal by node ID
func (tm *TerminalManager) GetTerminal(nodeID string) (Terminal, error) {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	terminal, exists := tm.terminals[nodeID]
	if !exists {
		return nil, fmt.Errorf("terminal for node %s not found", nodeID)
	}

	return terminal, nil
}

// GetCurrentTerminal gets the current terminal
func (tm *TerminalManager) GetCurrentTerminal() Terminal {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	if tm.current < 0 || tm.current >= len(tm.order) {
		return nil
	}

	nodeID := tm.order[tm.current]
	return tm.terminals[nodeID]
}

// GetCurrentNodeID gets the current node ID
func (tm *TerminalManager) GetCurrentNodeID() string {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	if tm.current < 0 || tm.current >= len(tm.order) {
		return ""
	}

	return tm.order[tm.current]
}

// NextTerminal switches to the next terminal
func (tm *TerminalManager) NextTerminal() {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	if len(tm.order) == 0 {
		return
	}

	tm.current = (tm.current + 1) % len(tm.order)

	if tm.onSwitch != nil {
		tm.onSwitch(tm.order[tm.current])
	}
}

// PreviousTerminal switches to the previous terminal
func (tm *TerminalManager) PreviousTerminal() {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	if len(tm.order) == 0 {
		return
	}

	tm.current = (tm.current - 1 + len(tm.order)) % len(tm.order)

	if tm.onSwitch != nil {
		tm.onSwitch(tm.order[tm.current])
	}
}

// SwitchToTerminal switches to a specific terminal
func (tm *TerminalManager) SwitchToTerminal(nodeID string) error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	// Find the terminal index
	for i, id := range tm.order {
		if id == nodeID {
			tm.current = i
			if tm.onSwitch != nil {
				tm.onSwitch(nodeID)
			}
			return nil
		}
	}

	return fmt.Errorf("terminal for node %s not found", nodeID)
}

// HasTerminal checks if a terminal exists
func (tm *TerminalManager) HasTerminal(nodeID string) bool {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	_, exists := tm.terminals[nodeID]
	return exists
}

// GetActiveCount returns the number of active terminals
func (tm *TerminalManager) GetActiveCount() int {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	count := 0
	for _, terminal := range tm.terminals {
		if terminal.IsActive() {
			count++
		}
	}
	return count
}

// GetTerminalInfos returns information about all terminals
func (tm *TerminalManager) GetTerminalInfos() []TerminalInfo {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	infos := make([]TerminalInfo, 0, len(tm.terminals))

	for nodeID, terminal := range tm.terminals {
		info := TerminalInfo{
			NodeID:       nodeID,
			IsActive:     terminal.IsActive(),
			ConnectedAt:  time.Now(), // Would need to track this properly
			LastActivity: time.Now(), // Would need to track this properly
		}
		infos = append(infos, info)
	}

	return infos
}

// CloseAll closes all terminals
func (tm *TerminalManager) CloseAll() {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	for nodeID, terminal := range tm.terminals {
		_ = terminal.Close()
		delete(tm.terminals, nodeID)
	}

	tm.order = make([]string, 0)
	tm.current = -1
}

// SendCommand sends a command to a specific terminal
func (tm *TerminalManager) SendCommand(nodeID string, command string) error {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	terminal, exists := tm.terminals[nodeID]
	if !exists {
		return fmt.Errorf("terminal for node %s not found", nodeID)
	}

	return terminal.SendCommand(command)
}

// ResizeTerminal resizes a specific terminal
func (tm *TerminalManager) ResizeTerminal(nodeID string, width, height int) error {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	terminal, exists := tm.terminals[nodeID]
	if !exists {
		return fmt.Errorf("terminal for node %s not found", nodeID)
	}

	return terminal.Resize(width, height)
}

// BroadcastCommand sends a command to all active terminals
func (tm *TerminalManager) BroadcastCommand(command string) []error {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	var errors []error
	for nodeID, terminal := range tm.terminals {
		if terminal.IsActive() {
			if err := terminal.SendCommand(command); err != nil {
				errors = append(errors, fmt.Errorf("node %s: %w", nodeID, err))
			}
		}
	}

	return errors
}

// GetTerminalList returns an ordered list of terminal node IDs
func (tm *TerminalManager) GetTerminalList() []string {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	result := make([]string, len(tm.order))
	copy(result, tm.order)
	return result
}
