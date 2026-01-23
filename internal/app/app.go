package app

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/jontk/s9s/internal/config"
	"github.com/jontk/s9s/internal/dao"
	"github.com/jontk/s9s/internal/errors"
	"github.com/jontk/s9s/internal/layouts"
	"github.com/jontk/s9s/internal/logging"
	"github.com/jontk/s9s/internal/plugins"
	"github.com/jontk/s9s/internal/preferences"
	"github.com/jontk/s9s/internal/ui/components"
	"github.com/jontk/s9s/internal/views"
	"github.com/jontk/s9s/pkg/slurm"
	"github.com/rivo/tview"
)

// contextKey is a custom type for context keys to avoid collisions
type contextKey string

const (
	// appContextKey is the context key for the tview application
	appContextKey contextKey = "app"
)

// S9s represents the main application
type S9s struct {
	ctx    context.Context
	cancel context.CancelFunc
	config *config.Config
	logger *logging.Logger

	// SLURM client
	client dao.SlurmClient

	// Plugin system
	pluginManager plugins.PluginManager

	// UI components
	app             *tview.Application
	pages           *tview.Pages
	header          *components.Header
	statusBar       *components.StatusBar
	viewMgr         *views.ViewManager
	alertsManager   *components.AlertsManager
	alertsBadge     *components.AlertsBadge
	notificationMgr interface{} // Will be set to *notifications.NotificationManager

	userPrefs     *preferences.UserPreferences
	layoutManager interface{} // Will be set to *layouts.LayoutManager

	// Main layout
	mainLayout   *tview.Flex
	contentPages *tview.Pages // Pages widget for stable view switching

	// Command line
	cmdLine    *tview.InputField
	cmdVisible bool

	// State
	refreshTicker *time.Ticker
	isRunning     bool
}

// New creates a new S9s application instance
func New(ctx context.Context, cfg *config.Config) (*S9s, error) {
	return NewWithScreen(ctx, cfg, nil)
}

// NewWithScreen creates a new S9s application instance with an optional screen for testing
func NewWithScreen(ctx context.Context, cfg *config.Config, screen tcell.Screen) (*S9s, error) {
	if cfg == nil {
		return nil, errors.Config("config is required")
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
			cancel()
			return nil, errors.Configf("no cluster configuration found for context: %s", cfg.CurrentContext)
		}

		// Create real SLURM adapter
		adapter, err := dao.NewSlurmAdapter(appCtx, clusterConfig)
		if err != nil {
			cancel()
			return nil, errors.DAOError("create", "SLURM adapter", err)
		}
		client = adapter
	}

	// Create tview application
	app := tview.NewApplication()

	// If a screen is provided (for testing), set it
	if screen != nil {
		app.SetScreen(screen)
	}

	s9s := &S9s{
		ctx:           appCtx,
		cancel:        cancel,
		config:        cfg,
		logger:        logging.GetLogger(),
		client:        client,
		app:           app,
		pages:         tview.NewPages(),
		contentPages:  tview.NewPages(),
		pluginManager: plugins.NewManager(appCtx, client),
	}

	// Initialize user preferences
	prefsPath := filepath.Join(filepath.Join(os.Getenv("HOME"), ".s9s"), "preferences.json")
	userPrefs, err := preferences.NewUserPreferences(prefsPath)
	if err != nil {
		s9s.logger.Warn().Err(err).Msg("Failed to load user preferences, using defaults")
		// Continue with defaults
		userPrefs, _ = preferences.NewUserPreferences("")
	}
	s9s.userPrefs = userPrefs

	// Initialize layout manager
	s9s.layoutManager = layouts.NewLayoutManager(app)

	// Initialize UI components
	if err := s9s.initUI(); err != nil {
		cancel()
		return nil, errors.Wrap(err, errors.ErrorTypeInternal, "failed to initialize UI")
	}

	// Initialize views
	if err := s9s.initViews(); err != nil {
		cancel()
		return nil, errors.ViewError("all", "initialization", err)
	}

	// Setup keyboard shortcuts
	s9s.setupKeyboardShortcuts()

	// Load plugins
	if err := s9s.loadPlugins(); err != nil {
		// Don't fail startup for plugin errors, just log them
		s9s.logger.Warn().Err(err).Msg("Failed to load plugins")
	}

	// Register plugin views
	if err := s9s.registerPluginViews(); err != nil {
		// Don't fail startup for plugin view registration errors, just log them
		s9s.logger.Warn().Err(err).Msg("Failed to register plugin views")
	}

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
	_ = s.viewMgr.StopAll()

	// Stop the tview application
	s.app.Stop()

	// Cancel the application context
	s.cancel()

	return nil
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
						_ = currentView.Refresh()
					}
				}
			case <-s.ctx.Done():
				return
			}
		}
	}()
}

// GetCurrentViewName returns the name of the current view
func (s *S9s) GetCurrentViewName() string {
	if currentView, err := s.viewMgr.GetCurrentView(); err == nil {
		return currentView.Name()
	}
	return ""
}

// GetViewManager returns the view manager (for testing)
func (s *S9s) GetViewManager() *views.ViewManager {
	return s.viewMgr
}

// GetApp returns the tview application (for testing)
func (s *S9s) GetApp() *tview.Application {
	return s.app
}

// GetPages returns the pages container (for testing)
func (s *S9s) GetPages() *tview.Pages {
	return s.pages
}

// IsModalOpen checks if a modal dialog is open
func (s *S9s) IsModalOpen() bool {
	return s.pages.GetPageCount() > 1
}

// GetModalName returns the name of the currently open modal, if any
func (s *S9s) GetModalName() string {
	count := s.pages.GetPageCount()
	if count <= 1 {
		return ""
	}
	// Get the topmost page name
	name, _ := s.pages.GetFrontPage()
	if name == "main" {
		return ""
	}
	return name
}

// IsCmdVisible returns whether the command line is visible
func (s *S9s) IsCmdVisible() bool {
	return s.cmdVisible
}
