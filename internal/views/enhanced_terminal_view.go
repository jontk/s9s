package views

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/jontk/s9s/internal/dao"
	"github.com/jontk/s9s/internal/logging"
	"github.com/jontk/s9s/internal/ssh"
	"github.com/jontk/s9s/internal/ui/components"
	"github.com/jontk/s9s/internal/ui/navigation"
	"github.com/rivo/tview"
)

// EnhancedTerminalView provides an advanced multi-pane terminal interface
type EnhancedTerminalView struct {
	BaseView

	// Dependencies
	client         dao.SlurmClient
	sessionManager *ssh.SessionManager

	// UI Components
	paneManager *components.MultiPaneManager
	navManager  *navigation.NavigationManager
	container   *tview.Flex

	// Terminal Panes
	terminalPanes map[string]*components.TerminalPane

	// State
	mu          sync.RWMutex
	nodes       []string
	activeNodes map[string]bool

	// Configuration
	maxTerminals  int
	autoConnect   bool
	defaultLayout components.PaneLayout
}

// NewEnhancedTerminalView creates a new enhanced terminal view
func NewEnhancedTerminalView(client dao.SlurmClient) *EnhancedTerminalView {
	etv := &EnhancedTerminalView{
		client:        client,
		terminalPanes: make(map[string]*components.TerminalPane),
		activeNodes:   make(map[string]bool),
		maxTerminals:  8,
		autoConnect:   false,
		defaultLayout: components.LayoutTabs,
	}

	etv.BaseView = BaseView{
		name:  "terminal",
		title: "Enhanced Terminal",
	}

	// Initialize SSH session manager
	sshConfig := ssh.DefaultSSHConfig()
	var err error
	etv.sessionManager, err = ssh.NewSessionManager(sshConfig)
	if err != nil {
		logging.Warnf("Failed to initialize SSH session manager: %v", err)
	}

	return etv
}

// Init initializes the enhanced terminal view
func (etv *EnhancedTerminalView) Init(ctx context.Context) error {
	etv.ctx = ctx

	// Initialize UI components
	etv.paneManager = components.NewMultiPaneManager(etv.app)
	etv.navManager = navigation.NewNavigationManager(etv.app)

	// Setup callbacks
	etv.setupCallbacks()

	// Create main container
	etv.container = tview.NewFlex()
	etv.container.SetDirection(tview.FlexRow)

	// Add navigation breadcrumb
	etv.container.AddItem(etv.navManager.GetBreadcrumb(), 1, 0, false)

	// Add main pane manager
	etv.container.AddItem(etv.paneManager.GetContainer(), 0, 1, true)

	// Add quick help bar
	etv.container.AddItem(etv.navManager.GetQuickHelpBar(), 1, 0, false)

	// Setup input handling
	etv.container.SetInputCapture(etv.handleInput)

	// Register navigation targets
	etv.registerNavigationTargets()

	// Load available nodes
	etv.loadNodes()

	// Create initial terminal if auto-connect is enabled
	if etv.autoConnect && len(etv.nodes) > 0 {
		etv.createTerminalPane(etv.nodes[0], "")
	}

	return nil
}

// setupCallbacks configures callbacks for UI components
func (etv *EnhancedTerminalView) setupCallbacks() {
	// Pane manager callbacks
	etv.paneManager.SetOnPaneSwitch(func(oldPane, newPane string) {
		etv.onPaneSwitch(oldPane, newPane)
	})

	etv.paneManager.SetOnLayoutChange(func(layout components.PaneLayout) {
		etv.onLayoutChange(layout)
	})

	// Navigation manager callbacks
	etv.navManager.SetOnNavigate(func(from, to string) {
		etv.onNavigate(from, to)
	})

	etv.navManager.SetOnModeChange(func(mode navigation.NavigationMode) {
		etv.onModeChange(mode)
	})

	etv.navManager.SetOnSearch(func(query string) []string {
		return etv.performSearch(query)
	})
}

// registerNavigationTargets registers navigation targets
func (etv *EnhancedTerminalView) registerNavigationTargets() {
	// Terminal management targets
	targets := []*navigation.NavigationTarget{
		{
			ID:          "new-terminal",
			Name:        "New Terminal",
			Description: "Create a new terminal pane",
			Type:        "action",
			Shortcut:    'n',
			KeyBinding:  tcell.KeyCtrlT,
			Priority:    10,
			OnActivate:  etv.createNewTerminal,
		},
		{
			ID:          "close-terminal",
			Name:        "Close Terminal",
			Description: "Close the active terminal pane",
			Type:        "action",
			Shortcut:    'c',
			KeyBinding:  tcell.KeyCtrlW,
			Priority:    9,
			OnActivate:  etv.closeActiveTerminal,
		},
		{
			ID:          "split-horizontal",
			Name:        "Split Horizontal",
			Description: "Split panes horizontally",
			Type:        "layout",
			Shortcut:    'h',
			KeyBinding:  tcell.KeyF3,
			Priority:    8,
			OnActivate:  etv.splitHorizontal,
		},
		{
			ID:          "split-vertical",
			Name:        "Split Vertical",
			Description: "Split panes vertically",
			Type:        "layout",
			Shortcut:    'v',
			KeyBinding:  tcell.KeyF4,
			Priority:    8,
			OnActivate:  etv.splitVertical,
		},
		{
			ID:          "tab-layout",
			Name:        "Tab Layout",
			Description: "Switch to tabbed layout",
			Type:        "layout",
			Shortcut:    't',
			Priority:    7,
			OnActivate:  etv.switchToTabLayout,
		},
		{
			ID:          "grid-layout",
			Name:        "Grid Layout",
			Description: "Switch to grid layout",
			Type:        "layout",
			Shortcut:    'g',
			Priority:    7,
			OnActivate:  etv.switchToGridLayout,
		},
	}

	for _, target := range targets {
		if err := etv.navManager.RegisterTarget(target); err != nil {
			logging.Warnf("Failed to register navigation target %s: %v", target.ID, err)
		}
	}
}

// loadNodes loads available cluster nodes
func (etv *EnhancedTerminalView) loadNodes() {
	if etv.client == nil {
		return
	}

	// Try to get nodes from the cluster
	if nodesMgr := etv.client.Nodes(); nodesMgr != nil {
		nodesList, err := nodesMgr.List(nil) // Pass nil for default options
		if err == nil && nodesList != nil {
			etv.mu.Lock()
			etv.nodes = make([]string, 0, len(nodesList.Nodes))
			for _, node := range nodesList.Nodes {
				etv.nodes = append(etv.nodes, node.Name)
			}
			etv.mu.Unlock()
		}
	}

	// If no nodes found, add some default options
	if len(etv.nodes) == 0 {
		etv.nodes = []string{"localhost"}
	}
}

// handleInput processes keyboard input
func (etv *EnhancedTerminalView) handleInput(event *tcell.EventKey) *tcell.EventKey {
	// Let navigation manager handle navigation shortcuts first
	if handled := etv.navManager.HandleInput(event); handled == nil {
		return nil
	}

	// Let pane manager handle pane-specific shortcuts
	return event
}

// createNewTerminal creates a new terminal pane
func (etv *EnhancedTerminalView) createNewTerminal() {
	etv.showNodeSelectionDialog()
}

// showNodeSelectionDialog shows a dialog to select a node for new terminal
func (etv *EnhancedTerminalView) showNodeSelectionDialog() {
	form := tview.NewForm()
	form.SetBorder(true)
	form.SetTitle(" New Terminal Connection ")
	form.SetTitleAlign(tview.AlignCenter)

	selectedNode := ""
	username := ""

	// Add node selection dropdown
	nodeOptions := make([]string, len(etv.nodes), len(etv.nodes)+1)
	copy(nodeOptions, etv.nodes)
	nodeOptions = append(nodeOptions, "Custom...")

	form.AddDropDown("Node", nodeOptions, 0, func(option string, optionIndex int) {
		selectedNode = option
	})

	// Add username field
	form.AddInputField("Username", "", 20, nil, func(text string) {
		username = text
	})

	// Add custom hostname field (initially hidden)
	customHostname := ""
	form.AddInputField("Custom Hostname", "", 30, nil, func(text string) {
		customHostname = text
	})

	// Connect button
	form.AddButton("Connect", func() {
		hostname := selectedNode
		if selectedNode == "Custom..." {
			hostname = customHostname
		}

		if hostname != "" {
			etv.createTerminalPane(hostname, username)
		}

		etv.hideModal("new-terminal")
	})

	// Cancel button
	form.AddButton("Cancel", func() {
		etv.hideModal("new-terminal")
	})

	etv.showModal("new-terminal", form)
}

// createTerminalPane creates a new terminal pane for the specified host
func (etv *EnhancedTerminalView) createTerminalPane(hostname, username string) {
	etv.mu.Lock()
	defer etv.mu.Unlock()

	// Generate unique ID
	paneID := fmt.Sprintf("terminal_%s_%d", hostname, time.Now().Unix())

	// Use current user if username not specified
	if username == "" {
		username = "user" // Default username
	}

	// Create terminal pane
	terminalPane := components.NewTerminalPane(paneID, hostname, username, etv.sessionManager)

	// Set callbacks
	terminalPane.SetOnClose(func(id string) error {
		return etv.closeTerminalPane(id)
	})

	terminalPane.SetOnTitleChange(func(id, title string) {
		etv.updatePaneTitle(id, title)
	})

	// Create pane for pane manager
	pane := &components.Pane{
		ID:        paneID,
		Title:     fmt.Sprintf("%s@%s", username, hostname),
		Type:      components.PaneTypeTerminal,
		Content:   terminalPane.GetContainer(),
		Closable:  true,
		Resizable: true,
		Metadata: map[string]interface{}{
			"hostname": hostname,
			"username": username,
		},
		OnClose: func(id string) error {
			return etv.closeTerminalPane(id)
		},
		OnActivate: func(id string) {
			etv.onTerminalActivate(id)
		},
	}

	// Add to pane manager
	if err := etv.paneManager.AddPane(pane); err != nil {
		logging.Errorf("Error adding terminal pane: %v", err)
		return
	}

	// Store terminal pane reference
	etv.terminalPanes[paneID] = terminalPane
	etv.activeNodes[hostname] = true
}

// closeActiveTerminal closes the currently active terminal
func (etv *EnhancedTerminalView) closeActiveTerminal() {
	activePane := etv.paneManager.GetActivePane()
	if activePane != nil && activePane.Type == components.PaneTypeTerminal {
		_ = etv.closeTerminalPane(activePane.ID)
	}
}

// closeTerminalPane closes a specific terminal pane
func (etv *EnhancedTerminalView) closeTerminalPane(paneID string) error {
	etv.mu.Lock()
	defer etv.mu.Unlock()

	// Close the terminal pane
	if terminalPane, exists := etv.terminalPanes[paneID]; exists {
		_ = terminalPane.Close()
		delete(etv.terminalPanes, paneID)
	}

	// Remove from pane manager
	return etv.paneManager.RemovePane(paneID)
}

// updatePaneTitle updates the title of a pane
func (etv *EnhancedTerminalView) updatePaneTitle(paneID, title string) {
	pane := etv.paneManager.GetPane(paneID)
	if pane != nil {
		pane.Title = title
	}
}

// Layout management methods
func (etv *EnhancedTerminalView) splitHorizontal() {
	etv.paneManager.SetLayout(components.LayoutHorizontal)
}

func (etv *EnhancedTerminalView) splitVertical() {
	etv.paneManager.SetLayout(components.LayoutVertical)
}

func (etv *EnhancedTerminalView) switchToTabLayout() {
	etv.paneManager.SetLayout(components.LayoutTabs)
}

func (etv *EnhancedTerminalView) switchToGridLayout() {
	etv.paneManager.SetLayout(components.LayoutGrid)
}

// Event handlers
func (etv *EnhancedTerminalView) onPaneSwitch(oldPane, newPane string) {
	// Handle pane switching logic
	logging.Infof("Switched from pane %s to %s", oldPane, newPane)
}

func (etv *EnhancedTerminalView) onLayoutChange(layout components.PaneLayout) {
	// Handle layout changes
	logging.Infof("Layout changed to %s", layout)
}

func (etv *EnhancedTerminalView) onNavigate(from, to string) {
	// Handle navigation
	logging.Infof("Navigation from %s to %s", from, to)
}

func (etv *EnhancedTerminalView) onModeChange(mode navigation.NavigationMode) {
	// Handle navigation mode changes
	logging.Infof("Navigation mode changed to %s", mode)
}

func (etv *EnhancedTerminalView) onTerminalActivate(paneID string) {
	// Handle terminal activation
	if terminalPane, exists := etv.terminalPanes[paneID]; exists {
		logging.Infof("Terminal %s (%s@%s) activated",
			paneID, terminalPane.GetUsername(), terminalPane.GetHostname())
	}
}

// performSearch performs a search across terminals and nodes
func (etv *EnhancedTerminalView) performSearch(query string) []string {
	results := make([]string, 0)
	query = strings.ToLower(query)

	// Search through active terminals
	etv.mu.RLock()
	for paneID, terminalPane := range etv.terminalPanes {
		hostname := strings.ToLower(terminalPane.GetHostname())
		username := strings.ToLower(terminalPane.GetUsername())

		if strings.Contains(hostname, query) || strings.Contains(username, query) {
			results = append(results, paneID)
		}
	}

	// Search through available nodes
	for _, node := range etv.nodes {
		if strings.Contains(strings.ToLower(node), query) {
			results = append(results, "node_"+node)
		}
	}
	etv.mu.RUnlock()

	return results
}

// Modal helpers
func (etv *EnhancedTerminalView) showModal(name string, content tview.Primitive) {
	if etv.pages != nil {
		etv.pages.AddPage(name, content, true, true)
	}
}

func (etv *EnhancedTerminalView) hideModal(name string) {
	if etv.pages != nil {
		etv.pages.RemovePage(name)
	}
}

// BaseView interface implementation
func (etv *EnhancedTerminalView) Render() tview.Primitive {
	return etv.container
}

func (etv *EnhancedTerminalView) Update() error {
	// Update terminal status and connections
	etv.mu.RLock()
	for _, terminalPane := range etv.terminalPanes {
		// Update terminal pane status if needed
		_ = terminalPane
	}
	etv.mu.RUnlock()

	return nil
}

func (etv *EnhancedTerminalView) Refresh() error {
	// Refresh node list and terminal states
	etv.loadNodes()
	return etv.Update()
}

func (etv *EnhancedTerminalView) OnKey(event *tcell.EventKey) *tcell.EventKey {
	return etv.handleInput(event)
}

func (etv *EnhancedTerminalView) Hints() []string {
	return []string{
		"Ctrl+T: New Terminal",
		"Ctrl+W: Close Terminal",
		"F3: Split Horizontal",
		"F4: Split Vertical",
		"F4: Quick Nav",
		"Ctrl+P: Command Palette",
		"Tab: Switch Panes",
	}
}

func (etv *EnhancedTerminalView) Stop() {
	// Close all terminals
	etv.mu.Lock()
	for paneID := range etv.terminalPanes {
		_ = etv.closeTerminalPane(paneID)
	}
	etv.mu.Unlock()

	// Close pane manager
	etv.paneManager.Close()
}

// GetTerminalCount returns the number of active terminals
func (etv *EnhancedTerminalView) GetTerminalCount() int {
	etv.mu.RLock()
	defer etv.mu.RUnlock()
	return len(etv.terminalPanes)
}

// GetActiveNodes returns the list of active node connections
func (etv *EnhancedTerminalView) GetActiveNodes() []string {
	etv.mu.RLock()
	defer etv.mu.RUnlock()

	nodes := make([]string, 0, len(etv.activeNodes))
	for node := range etv.activeNodes {
		nodes = append(nodes, node)
	}
	return nodes
}

// BroadcastCommand sends a command to all active terminals
func (etv *EnhancedTerminalView) BroadcastCommand(command string) error {
	etv.mu.RLock()
	defer etv.mu.RUnlock()

	var errors []string
	for paneID, terminalPane := range etv.terminalPanes {
		if terminalPane.IsConnected() {
			if err := terminalPane.SendCommand(command); err != nil {
				errors = append(errors, fmt.Sprintf("%s: %v", paneID, err))
			}
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("broadcast failed for some terminals: %s", strings.Join(errors, "; "))
	}

	return nil
}
