package app

import (
	"context"
	"fmt"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/jontk/s9s/internal/config"
	"github.com/jontk/s9s/internal/dao"
	"github.com/jontk/s9s/internal/ui/components"
	"github.com/jontk/s9s/internal/views"
	"github.com/jontk/s9s/pkg/slurm"
)

// S9s represents the main application
type S9s struct {
	ctx    context.Context
	cancel context.CancelFunc
	config *config.Config

	// SLURM client
	client dao.SlurmClient

	// UI components
	app        *tview.Application
	pages      *tview.Pages
	header     *components.Header
	statusBar  *components.StatusBar
	viewMgr    *views.ViewManager

	// Main layout
	mainLayout *tview.Flex

	// Command line
	cmdLine    *tview.InputField
	cmdVisible bool

	// State
	refreshTicker *time.Ticker
	isRunning     bool
}

// New creates a new S9s application instance
func New(ctx context.Context, cfg *config.Config) (*S9s, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config is required")
	}

	// Create cancellable context
	appCtx, cancel := context.WithCancel(ctx)

	// Create SLURM client
	var client dao.SlurmClient
	if cfg.UseMockClient {
		client = slurm.NewMockClient()
	} else {
		// Get current cluster config
		var clusterConfig *config.ClusterConfig
		if len(cfg.Contexts) > 0 {
			for _, ctx := range cfg.Contexts {
				if ctx.Name == cfg.CurrentContext {
					clusterConfig = &ctx.Cluster
					break
				}
			}
		}

		if clusterConfig == nil {
			return nil, fmt.Errorf("no cluster configuration found for context: %s", cfg.CurrentContext)
		}

		// Create real SLURM adapter
		adapter, err := dao.NewSlurmAdapter(appCtx, clusterConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to create SLURM adapter: %w", err)
		}
		client = adapter
	}

	// Create tview application
	app := tview.NewApplication()

	s9s := &S9s{
		ctx:     appCtx,
		cancel:  cancel,
		config:  cfg,
		client:  client,
		app:     app,
		pages:   tview.NewPages(),
	}

	// Initialize UI components
	if err := s9s.initUI(); err != nil {
		cancel()
		return nil, fmt.Errorf("failed to initialize UI: %w", err)
	}

	// Initialize views
	if err := s9s.initViews(); err != nil {
		cancel()
		return nil, fmt.Errorf("failed to initialize views: %w", err)
	}

	// Setup keyboard shortcuts
	s9s.setupKeyboardShortcuts()

	return s9s, nil
}

// Run starts the application
func (s *S9s) Run() error {
	s.isRunning = true

	// Start background services (removed updateLoop to prevent duplicate refreshes)

	// Start refresh ticker if configured
	if s.config.RefreshRate != "" {
		duration, err := time.ParseDuration(s.config.RefreshRate)
		if err == nil && duration > 0 {
			s.startRefreshTimer(duration)
		}
	}

	// Set initial view
	if err := s.viewMgr.SetCurrentView("jobs"); err != nil {
		s.statusBar.Error(fmt.Sprintf("Failed to set initial view: %v", err))
	} else {
		// Update the display to show the initial view
		s.updateCurrentView()
	}

	// Set the root and run the application
	s.app.SetRoot(s.pages, true)

	// Run will block until Stop is called
	return s.app.Run()
}

// ShowModal displays a modal dialog over the main interface
func (s *S9s) ShowModal(name string, modal tview.Primitive) {
	s.pages.AddPage(name, modal, true, true)
}

// HideModal removes a modal dialog
func (s *S9s) HideModal(name string) {
	s.pages.RemovePage(name)
	s.app.SetFocus(s.mainLayout)
}

// Stop gracefully stops the application
func (s *S9s) Stop() error {
	s.isRunning = false

	// Stop refresh timer
	if s.refreshTicker != nil {
		s.refreshTicker.Stop()
	}

	// Stop header
	s.header.Stop()

	// Stop all views
	s.viewMgr.StopAll()

	// Stop the tview application
	s.app.Stop()

	// Cancel the application context
	s.cancel()

	return nil
}

// initUI initializes the UI components
func (s *S9s) initUI() error {
	// Create header
	s.header = components.NewHeader()

	// Create status bar
	s.statusBar = components.NewStatusBar()

	// Create view manager
	s.viewMgr = views.NewViewManager(s.app)
	s.viewMgr.SetPages(s.pages)

	// Create command line (hidden by default)
	s.cmdLine = tview.NewInputField().
		SetLabel(":").
		SetFieldBackgroundColor(tcell.ColorBlack).
		SetFieldTextColor(tcell.ColorWhite).
		SetDoneFunc(s.onCommandDone)
	s.cmdLine.SetBorder(false)

	// Create main layout placeholder
	s.mainLayout = tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(s.header, 2, 0, false).
		AddItem(tview.NewTextView().SetText("Loading..."), 0, 1, true).
		AddItem(s.cmdLine, 1, 0, false).
		AddItem(s.statusBar, 1, 0, false)

	// Hide command line initially
	s.hideCommandLine()

	// Add main page
	s.pages.AddPage("main", s.mainLayout, true, true)

	return nil
}

// initViews initializes all the views
func (s *S9s) initViews() error {
	// Create jobs view
	jobsView := views.NewJobsView(s.client)
	jobsView.SetApp(s.app)
	if err := jobsView.Init(s.ctx); err != nil {
		return fmt.Errorf("failed to initialize jobs view: %w", err)
	}
	s.viewMgr.AddView(jobsView)

	// Create nodes view
	nodesView := views.NewNodesView(s.client)
	nodesView.SetApp(s.app)
	if err := nodesView.Init(s.ctx); err != nil {
		return fmt.Errorf("failed to initialize nodes view: %w", err)
	}
	s.viewMgr.AddView(nodesView)

	// Create partitions view
	partitionsView := views.NewPartitionsView(s.client)
	partitionsView.SetApp(s.app)
	if err := partitionsView.Init(s.ctx); err != nil {
		return fmt.Errorf("failed to initialize partitions view: %w", err)
	}
	s.viewMgr.AddView(partitionsView)

	// Create reservations view
	reservationsView := views.NewReservationsView(s.client)
	if err := reservationsView.Init(s.ctx); err != nil {
		return fmt.Errorf("failed to initialize reservations view: %w", err)
	}
	s.viewMgr.AddView(reservationsView)

	// Create QoS view
	qosView := views.NewQoSView(s.client)
	if err := qosView.Init(s.ctx); err != nil {
		return fmt.Errorf("failed to initialize qos view: %w", err)
	}
	s.viewMgr.AddView(qosView)

	// Create Accounts view
	accountsView := views.NewAccountsView(s.client)
	if err := accountsView.Init(s.ctx); err != nil {
		return fmt.Errorf("failed to initialize accounts view: %w", err)
	}
	s.viewMgr.AddView(accountsView)

	// Create Users view
	usersView := views.NewUsersView(s.client)
	if err := usersView.Init(s.ctx); err != nil {
		return fmt.Errorf("failed to initialize users view: %w", err)
	}
	s.viewMgr.AddView(usersView)

	// Create Health view
	healthView := views.NewHealthView(s.client)
	healthView.SetApp(s.app)
	healthView.SetPages(s.pages)
	if err := healthView.Init(s.ctx); err != nil {
		return fmt.Errorf("failed to initialize health view: %w", err)
	}
	s.viewMgr.AddView(healthView)

	// Update header with view names
	s.header.SetViews(s.viewMgr.GetViewNames())

	return nil
}

// setupKeyboardShortcuts configures global keyboard shortcuts
func (s *S9s) setupKeyboardShortcuts() {
	s.app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		// Handle command mode
		if s.cmdVisible {
			return event // Let command line handle it
		}

		// Check if a modal is open by checking if there are multiple pages
		isModalOpen := s.pages.GetPageCount() > 1

		// Global shortcuts
		switch event.Key() {
		case tcell.KeyCtrlC:
			s.app.Stop()
			return nil
		case tcell.KeyF1:
			// Show help modal
			views.ShowHelpModal(s.pages)
			return nil
		case tcell.KeyF5:
			// Refresh current view
			if err := s.viewMgr.RefreshCurrentView(); err != nil {
				s.statusBar.Error(fmt.Sprintf("Failed to refresh: %v", err))
			}
			return nil
		case tcell.KeyEsc:
			if s.cmdVisible {
				s.hideCommandLine()
				return nil
			}
		case tcell.KeyTab:
			if isModalOpen {
				// Let the modal handle tab navigation
				return event
			}
			s.viewMgr.NextView()
			s.updateCurrentView()
			return nil
		case tcell.KeyBacktab:
			if isModalOpen {
				// Let the modal handle shift+tab navigation
				return event
			}
			s.viewMgr.PreviousView()
			s.updateCurrentView()
			return nil
		case tcell.KeyRune:
			// If a modal is open, let it handle all character input except a few special cases
			if isModalOpen {
				switch event.Rune() {
				case ':':
					// Allow command mode even in modals for emergency commands
					s.showCommandLine()
					return nil
				default:
					// Let modal handle all other character input (including 1-7, s, c, etc.)
					return event
				}
			}

			// Normal global shortcuts when no modal is open
			switch event.Rune() {
			case ':':
				s.showCommandLine()
				return nil
			case '?':
				s.showHelp()
				return nil
			case '1':
				s.switchToView("jobs")
				return nil
			case '2':
				s.switchToView("nodes")
				return nil
			case '3':
				s.switchToView("partitions")
				return nil
			case '4':
				s.switchToView("reservations")
				return nil
			case '5':
				s.switchToView("qos")
				return nil
			case '6':
				s.switchToView("accounts")
				return nil
			case '7':
				s.switchToView("users")
				return nil
			case '8':
				s.switchToView("health")
				return nil
			case 'q', 'Q':
				s.app.Stop()
				return nil
			}
		}

		// Pass to current view if not handled and no modal is open
		if !isModalOpen {
			if currentView, err := s.viewMgr.GetCurrentView(); err == nil {
				return currentView.OnKey(event)
			}
		}

		return event
	})
}

// updateLoop runs periodic updates
func (s *S9s) updateLoop() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if s.isRunning {
				s.updateClusterInfo()
			}
		case <-s.ctx.Done():
			return
		}
	}
}

// updateClusterInfo updates cluster information in header
func (s *S9s) updateClusterInfo() {
	// Get cluster info
	clusterInfo, err := s.client.ClusterInfo()
	if err == nil {
		s.header.SetClusterInfo(clusterInfo)
	}

	// Get cluster metrics
	if infoMgr := s.client.Info(); infoMgr != nil {
		metrics, err := infoMgr.GetStats()
		if err == nil {
			s.header.SetMetrics(metrics)
		}
	}
}

// switchToView switches to the specified view
func (s *S9s) switchToView(viewName string) {
	if err := s.viewMgr.SetCurrentView(viewName); err != nil {
		s.statusBar.Error(fmt.Sprintf("Failed to switch to %s view: %v", viewName, err))
		return
	}
	s.updateCurrentView()
}

// updateCurrentView updates the display for the current view
func (s *S9s) updateCurrentView() {
	currentView, err := s.viewMgr.GetCurrentView()
	if err != nil {
		s.statusBar.Error(fmt.Sprintf("No current view: %v", err))
		return
	}

	// Update header with current view
	s.header.SetCurrentView(currentView.Name())

	// Update status bar with view hints
	s.statusBar.SetHints(currentView.Hints())

	// Replace the main content with the current view
	s.mainLayout.RemoveItem(s.mainLayout.GetItem(1))
	s.mainLayout.AddItem(currentView.Render(), 0, 1, true)

	// Set focus to the view
	s.app.SetFocus(currentView.Render())
}

// showCommandLine shows the command input field
func (s *S9s) showCommandLine() {
	s.cmdVisible = true
	s.cmdLine.SetText("")
	s.mainLayout.ResizeItem(s.cmdLine, 1, 0)
	s.app.SetFocus(s.cmdLine)
}

// hideCommandLine hides the command input field
func (s *S9s) hideCommandLine() {
	s.cmdVisible = false
	s.mainLayout.ResizeItem(s.cmdLine, 0, 0)

	// Return focus to current view
	if currentView, err := s.viewMgr.GetCurrentView(); err == nil {
		s.app.SetFocus(currentView.Render())
	}
}

// onCommandDone handles command line completion
func (s *S9s) onCommandDone(key tcell.Key) {
	defer s.hideCommandLine()

	if key != tcell.KeyEnter {
		return
	}

	command := s.cmdLine.GetText()
	if command == "" {
		return
	}

	s.executeCommand(command)
}

// executeCommand executes a command
func (s *S9s) executeCommand(command string) {
	switch command {
	case "q", "quit":
		s.app.Stop()
	case "jobs", "j":
		s.switchToView("jobs")
	case "nodes", "n":
		s.switchToView("nodes")
	case "partitions", "p":
		s.switchToView("partitions")
	case "reservations":
		s.switchToView("reservations")
	case "qos":
		s.switchToView("qos")
	case "accounts":
		s.switchToView("accounts")
	case "users":
		s.switchToView("users")
	case "health":
		s.switchToView("health")
	case "refresh", "r":
		if currentView, err := s.viewMgr.GetCurrentView(); err == nil {
			go func() {
				if err := currentView.Refresh(); err != nil {
					s.statusBar.Error(fmt.Sprintf("Refresh failed: %v", err))
				} else {
					s.statusBar.Success("Refreshed")
				}
			}()
		}
	case "help", "h":
		s.showHelp()
	default:
		s.statusBar.Error(fmt.Sprintf("Unknown command: %s", command))
	}
}

// showHelp displays help information
func (s *S9s) showHelp() {
	helpText := `[yellow]S9S - SLURM Terminal UI Help[white]

[teal]Global Keys:[white]
  [yellow]1-8[white]         Switch to Jobs/Nodes/Partitions/Reservations/QoS/Accounts/Users/Health view
  [yellow]Tab/Shift+Tab[white] Switch between views
  [yellow]:[white]          Enter command mode
  [yellow]?[white]          Show this help
  [yellow]q, Ctrl+C[white]  Quit application

[teal]Commands:[white]
  [yellow]:jobs, :j[white]      Switch to Jobs view
  [yellow]:nodes, :n[white]     Switch to Nodes view
  [yellow]:partitions, :p[white] Switch to Partitions view
  [yellow]:reservations[white]  Switch to Reservations view
  [yellow]:qos[white]           Switch to QoS view
  [yellow]:accounts[white]      Switch to Accounts view
  [yellow]:users[white]         Switch to Users view
  [yellow]:health[white]        Switch to Health Monitor view
  [yellow]:refresh, :r[white]   Refresh current view
  [yellow]:quit, :q[white]      Quit application
  [yellow]:help, :h[white]      Show this help

[teal]View-specific keys vary by view.[white]
Press [yellow]ESC[white] to close this help.`

	modal := tview.NewTextView().
		SetDynamicColors(true).
		SetText(helpText).
		SetTextAlign(tview.AlignLeft)

	modal.SetBorder(true).
		SetTitle(" Help ").
		SetTitleAlign(tview.AlignCenter)

	modal.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEsc {
			s.pages.RemovePage("help")
			s.app.SetFocus(s.mainLayout)
			return nil
		}
		return event
	})

	s.pages.AddPage("help", modal, true, true)
}

// startRefreshTimer starts the automatic refresh timer
func (s *S9s) startRefreshTimer(duration time.Duration) {
	s.refreshTicker = time.NewTicker(duration)

	go func() {
		for {
			select {
			case <-s.refreshTicker.C:
				if s.isRunning {
					if currentView, err := s.viewMgr.GetCurrentView(); err == nil {
						// Refresh synchronously to prevent goroutine explosion
						currentView.Refresh()
					}
				}
			case <-s.ctx.Done():
				return
			}
		}
	}()
}