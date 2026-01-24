package app

import (
	"github.com/gdamore/tcell/v2"
	"github.com/jontk/s9s/internal/notifications"
	"github.com/jontk/s9s/internal/ui/components"
	"github.com/jontk/s9s/internal/views"
	"github.com/rivo/tview"
)

// initUI initializes the UI components
// Returns error for extensibility, currently always returns nil
//nolint:unparam
func (s *S9s) initUI() error {
	// Create alerts manager
	s.alertsManager = components.NewAlertsManager(100) // Keep last 100 alerts

	// Initialize notification system
	configPath := "" // Will use default path
	notificationMgr, err := notifications.NewNotificationManager(configPath)
	if err != nil {
		s.logger.Warn().Err(err).Msg("Failed to initialize notification manager, continuing without notifications")
		// Continue without notifications
	} else {
		s.notificationMgr = notificationMgr
		// Connect notification manager to alerts manager
		adapter := notifications.NewAlertNotifierAdapter(notificationMgr)
		s.alertsManager.SetNotifier(adapter)
	}

	// Create header
	s.header = components.NewHeader()

	// Create status bar
	s.statusBar = components.NewStatusBar()

	// Create alerts badge for header
	s.alertsBadge = components.NewAlertsBadge(s.alertsManager)
	s.header.SetAlertsBadge(s.alertsBadge)

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

	// Create main layout with stable structure using contentPages
	s.mainLayout = tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(s.header, 2, 0, false).
		AddItem(s.contentPages, 0, 1, true).
		AddItem(s.cmdLine, 1, 0, false).
		AddItem(s.statusBar, 1, 0, false)

	// Hide command line initially
	s.hideCommandLine()

	// Add main page
	s.pages.AddPage("main", s.mainLayout, true, true)

	return nil
}
