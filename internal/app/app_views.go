package app

import (
	"fmt"

	"github.com/jontk/s9s/internal/errors"
	"github.com/jontk/s9s/internal/views"
)

// initViews initializes all the views
func (s *S9s) initViews() error {
	// Create jobs view
	jobsView := views.NewJobsView(s.client)
	jobsView.SetApp(s.app)
	jobsView.SetStatusBar(s.statusBar)
	if err := jobsView.Init(s.ctx); err != nil {
		return errors.ViewError("jobs", "initialization", err)
	}
	if err := s.viewMgr.AddView(jobsView); err != nil {
		return errors.ViewError("jobs", "add to manager", err)
	}
	s.contentPages.AddPage("jobs", jobsView.Render(), true, false)

	// Create nodes view
	nodesView := views.NewNodesView(s.client)
	nodesView.SetApp(s.app)
	if err := nodesView.Init(s.ctx); err != nil {
		return errors.ViewError("nodes", "initialization", err)
	}
	if err := s.viewMgr.AddView(nodesView); err != nil {
		return errors.ViewError("nodes", "add to manager", err)
	}
	s.contentPages.AddPage("nodes", nodesView.Render(), true, false)

	// Create partitions view
	partitionsView := views.NewPartitionsView(s.client)
	partitionsView.SetApp(s.app)
	if err := partitionsView.Init(s.ctx); err != nil {
		return errors.ViewError("partitions", "initialization", err)
	}
	if err := s.viewMgr.AddView(partitionsView); err != nil {
		return errors.ViewError("partitions", "add to manager", err)
	}
	s.contentPages.AddPage("partitions", partitionsView.Render(), true, false)

	// Create reservations view
	reservationsView := views.NewReservationsView(s.client)
	reservationsView.SetApp(s.app)
	reservationsView.SetPages(s.pages)
	if err := reservationsView.Init(s.ctx); err != nil {
		return errors.ViewError("reservations", "initialization", err)
	}
	if err := s.viewMgr.AddView(reservationsView); err != nil {
		return errors.ViewError("reservations", "add to manager", err)
	}
	s.contentPages.AddPage("reservations", reservationsView.Render(), true, false)

	// Create QoS view
	qosView := views.NewQoSView(s.client)
	qosView.SetApp(s.app)
	qosView.SetPages(s.pages)
	if err := qosView.Init(s.ctx); err != nil {
		return errors.ViewError("qos", "initialization", err)
	}
	if err := s.viewMgr.AddView(qosView); err != nil {
		return errors.ViewError("qos", "add to manager", err)
	}
	s.contentPages.AddPage("qos", qosView.Render(), true, false)

	// Create Accounts view
	accountsView := views.NewAccountsView(s.client)
	accountsView.SetApp(s.app)
	accountsView.SetPages(s.pages)
	if err := accountsView.Init(s.ctx); err != nil {
		return errors.ViewError("accounts", "initialization", err)
	}
	if err := s.viewMgr.AddView(accountsView); err != nil {
		return errors.ViewError("accounts", "add to manager", err)
	}
	s.contentPages.AddPage("accounts", accountsView.Render(), true, false)

	// Create Users view
	usersView := views.NewUsersView(s.client)
	usersView.SetApp(s.app)
	usersView.SetPages(s.pages)
	if err := usersView.Init(s.ctx); err != nil {
		return errors.ViewError("users", "initialization", err)
	}
	if err := s.viewMgr.AddView(usersView); err != nil {
		return errors.ViewError("users", "add to manager", err)
	}
	s.contentPages.AddPage("users", usersView.Render(), true, false)

	// Create Dashboard view
	dashboardView := views.NewDashboardView(s.client)
	dashboardView.SetApp(s.app)
	dashboardView.SetPages(s.pages)
	if err := dashboardView.Init(s.ctx); err != nil {
		return errors.ViewError("dashboard", "initialization", err)
	}
	if err := s.viewMgr.AddView(dashboardView); err != nil {
		return errors.ViewError("dashboard", "add to manager", err)
	}
	s.contentPages.AddPage("dashboard", dashboardView.Render(), true, false)

	// Create Health view
	healthView := views.NewHealthView(s.client)
	healthView.SetApp(s.app)
	healthView.SetPages(s.pages)
	if err := healthView.Init(s.ctx); err != nil {
		return errors.ViewError("health", "initialization", err)
	}
	if err := s.viewMgr.AddView(healthView); err != nil {
		return errors.ViewError("health", "add to manager", err)
	}
	s.contentPages.AddPage("health", healthView.Render(), true, false)

	// Create Performance view
	performanceView := views.NewPerformanceView(s.client)
	performanceView.SetApp(s.app)
	performanceView.SetPages(s.pages)
	if err := performanceView.Init(s.ctx); err != nil {
		return errors.ViewError("performance", "initialization", err)
	}
	if err := s.viewMgr.AddView(performanceView); err != nil {
		return errors.ViewError("performance", "add to manager", err)
	}
	s.contentPages.AddPage("performance", performanceView.Render(), true, false)

	// Update header with view names
	s.header.SetViews(s.viewMgr.GetViewNames())

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
