package app

import (
	"github.com/jontk/s9s/internal/layouts"
	"github.com/jontk/s9s/internal/views"
)

// registerLayoutWidgets registers all widgets with the layout manager.
// Must be called after initViews() so that views are available.
func (s *S9s) registerLayoutWidgets() {
	// Wrap the health view as the health widget
	if healthView, err := s.viewMgr.GetView("health"); err == nil {
		healthWidget := layouts.NewViewWidget("health", "Health", healthView)
		_ = s.layoutManager.RegisterWidget(healthWidget)
	}

	// Create standalone widgets
	metricsWidget := layouts.NewMetricsWidget("metrics", s.client)
	_ = s.layoutManager.RegisterWidget(metricsWidget)

	// Wire the layout switcher into the dashboard view
	if dashView, err := s.viewMgr.GetView("dashboard"); err == nil {
		if dv, ok := dashView.(*views.DashboardView); ok {
			dv.SetLayoutContainer(s.layoutManager.GetContainer())
			dv.SetLayoutSwitcherFunc(s.showLayoutSwitcher)
		}
	}

	// Apply saved layout from preferences (only if one was explicitly chosen)
	layoutID := s.userPrefs.Get().Layouts.CurrentLayout
	// "default" means "use the dashboard's built-in panels with no layout
	// applied", so we only call SetCurrentLayout for non-default IDs.
	if layoutID != "" && layoutID != "default" {
		if err := s.layoutManager.SetCurrentLayout(layoutID); err == nil {
			// Re-add the dashboard page so contentPages uses the layout container
			s.applyDashboardLayout()
		}
	}
}

// applyDashboardLayout replaces the dashboard page with the layout container.
func (s *S9s) applyDashboardLayout() {
	if dashView, err := s.viewMgr.GetView("dashboard"); err == nil {
		if dv, ok := dashView.(*views.DashboardView); ok {
			s.contentPages.RemovePage("dashboard")
			s.contentPages.AddPage("dashboard", dv.Render(), true, false)
			s.contentPages.SwitchToPage("dashboard")
		}
	}
}

// restoreDashboardDefault removes the layout container and restores the
// dashboard's built-in 6-panel view.
// Setting the layout container to nil causes Render() to fall through to the
// built-in panel container instead of using a custom layout.
func (s *S9s) restoreDashboardDefault() {
	if dashView, err := s.viewMgr.GetView("dashboard"); err == nil {
		if dv, ok := dashView.(*views.DashboardView); ok {
			dv.SetLayoutContainer(nil)
			s.contentPages.RemovePage("dashboard")
			s.contentPages.AddPage("dashboard", dv.Render(), true, false)
			s.contentPages.SwitchToPage("dashboard")
		}
	}
}
