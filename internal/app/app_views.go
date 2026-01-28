package app

import (
	"fmt"

	"github.com/jontk/s9s/internal/errs"
	"github.com/jontk/s9s/internal/views"
)

// initViews initializes all the views
func (s *S9s) initViews() error {
	viewRegistry := []struct {
		name      string
		register  func() error
	}{
		{"jobs", s.registerJobsView},
		{"nodes", s.registerNodesView},
		{"partitions", s.registerPartitionsView},
		{"reservations", s.registerReservationsView},
		{"qos", s.registerQoSView},
		{"accounts", s.registerAccountsView},
		{"users", s.registerUsersView},
		{"dashboard", s.registerDashboardView},
		{"health", s.registerHealthView},
		{"performance", s.registerPerformanceView},
	}

	for _, v := range viewRegistry {
		if err := v.register(); err != nil {
			return err
		}
	}

	s.header.SetViews(s.viewMgr.GetViewNames())
	return nil
}

// registerJobsView registers the jobs view
func (s *S9s) registerJobsView() error {
	view := views.NewJobsView(s.client)
	view.SetApp(s.app)
	view.SetStatusBar(s.statusBar)
	view.SetPages(s.pages)
	return s.addViewToApp("jobs", view)
}

// registerNodesView registers the nodes view
func (s *S9s) registerNodesView() error {
	view := views.NewNodesView(s.client)
	view.SetApp(s.app)
	view.SetPages(s.pages)
	return s.addViewToApp("nodes", view)
}

// registerPartitionsView registers the partitions view
func (s *S9s) registerPartitionsView() error {
	view := views.NewPartitionsView(s.client)
	view.SetApp(s.app)
	view.SetPages(s.pages)
	return s.addViewToApp("partitions", view)
}

// registerReservationsView registers the reservations view
func (s *S9s) registerReservationsView() error {
	view := views.NewReservationsView(s.client)
	view.SetApp(s.app)
	view.SetPages(s.pages)
	return s.addViewToApp("reservations", view)
}

// registerQoSView registers the QoS view
func (s *S9s) registerQoSView() error {
	view := views.NewQoSView(s.client)
	view.SetApp(s.app)
	view.SetPages(s.pages)
	return s.addViewToApp("qos", view)
}

// registerAccountsView registers the accounts view
func (s *S9s) registerAccountsView() error {
	view := views.NewAccountsView(s.client)
	view.SetApp(s.app)
	view.SetPages(s.pages)
	return s.addViewToApp("accounts", view)
}

// registerUsersView registers the users view
func (s *S9s) registerUsersView() error {
	view := views.NewUsersView(s.client)
	view.SetApp(s.app)
	view.SetPages(s.pages)
	return s.addViewToApp("users", view)
}

// registerDashboardView registers the dashboard view
func (s *S9s) registerDashboardView() error {
	view := views.NewDashboardView(s.client)
	view.SetApp(s.app)
	view.SetPages(s.pages)
	return s.addViewToApp("dashboard", view)
}

// registerHealthView registers the health view
func (s *S9s) registerHealthView() error {
	view := views.NewHealthView(s.client)
	view.SetApp(s.app)
	view.SetPages(s.pages)
	return s.addViewToApp("health", view)
}

// registerPerformanceView registers the performance view
func (s *S9s) registerPerformanceView() error {
	view := views.NewPerformanceView(s.client)
	view.SetApp(s.app)
	view.SetPages(s.pages)
	return s.addViewToApp("performance", view)
}

// addViewToApp initializes and adds a view to the application
func (s *S9s) addViewToApp(name string, view views.View) error {
	if err := view.Init(s.ctx); err != nil {
		return errs.ViewError(name, "initialization", err)
	}

	if err := s.viewMgr.AddView(view); err != nil {
		return errs.ViewError(name, "add to manager", err)
	}

	s.contentPages.AddPage(name, view.Render(), true, false)
	return nil
}

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

	// Switch to the appropriate page in contentPages
	// This maintains the stable layout without any manipulation
	s.contentPages.SwitchToPage(currentView.Name())

	// Set focus to the view
	s.app.SetFocus(currentView.Render())
}
