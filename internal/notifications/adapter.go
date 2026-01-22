package notifications

import (
	"github.com/jontk/s9s/internal/ui/components"
)

// AlertNotifierAdapter adapts NotificationManager to the AlertNotifier interface
type AlertNotifierAdapter struct {
	manager *NotificationManager
}

// NewAlertNotifierAdapter creates a new adapter
func NewAlertNotifierAdapter(manager *NotificationManager) *AlertNotifierAdapter {
	return &AlertNotifierAdapter{
		manager: manager,
	}
}

// Notify implements the AlertNotifier interface
func (a *AlertNotifierAdapter) Notify(alert *components.Alert) {
	if a.manager != nil {
		a.manager.Notify(alert)
	}
}
