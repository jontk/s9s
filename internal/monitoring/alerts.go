// Package monitoring provides cluster monitoring and alert management.
package monitoring

import (
	"fmt"
	"slices"
	"sync"
	"time"
)

// AlertType represents the type of alert
type AlertType string

const (
	// AlertTypeHealth is the alert type for health-related alerts.
	AlertTypeHealth AlertType = "health"
	// AlertTypePerformance is the alert type for performance-related alerts.
	AlertTypePerformance AlertType = "performance"
	// AlertTypeResource is the alert type for resource-related alerts.
	AlertTypeResource AlertType = "resource"
	// AlertTypeJob is the alert type for job-related alerts.
	AlertTypeJob AlertType = "job"
	// AlertTypeNode is the alert type for node-related alerts.
	AlertTypeNode AlertType = "node"
	// AlertTypeSystem is the alert type for system-related alerts.
	AlertTypeSystem AlertType = "system"
)

// AlertSeverity represents the severity level of an alert
type AlertSeverity string

const (
	// AlertSeverityInfo is the info severity level for alerts.
	AlertSeverityInfo AlertSeverity = "info"
	// AlertSeverityWarning is the warning severity level for alerts.
	AlertSeverityWarning AlertSeverity = "warning"
	// AlertSeverityCritical is the critical severity level for alerts.
	AlertSeverityCritical AlertSeverity = "critical"
)

// Alert represents a cluster alert
type Alert struct {
	ID           string
	Type         AlertType
	Severity     AlertSeverity
	Title        string
	Message      string
	Component    string
	Timestamp    time.Time
	Acknowledged bool
	AckedBy      string
	AckedAt      *time.Time
	Resolved     bool
	ResolvedAt   *time.Time
	Count        int
	LastSeen     time.Time
	Metadata     map[string]interface{}
}

// AlertManager manages cluster alerts
type AlertManager struct {
	alerts    map[string]*Alert
	listeners []AlertListener
	mu        sync.RWMutex
}

// AlertListener defines the interface for alert listeners
type AlertListener interface {
	OnAlert(alert *Alert)
	OnAlertResolved(alert *Alert)
}

// AlertStats provides statistics about alerts
type AlertStats struct {
	Total          int
	Critical       int
	Warning        int
	Info           int
	Acknowledged   int
	Unacknowledged int
	Active         int
	Resolved       int
}

// NewAlertManager creates a new alert manager
func NewAlertManager() *AlertManager {
	return &AlertManager{
		alerts:    make(map[string]*Alert),
		listeners: []AlertListener{},
	}
}

// AddAlert adds a new alert or updates an existing one
func (am *AlertManager) AddAlert(alert *Alert) {
	am.mu.Lock()
	defer am.mu.Unlock()

	// Check if similar alert exists
	existingID := am.findSimilarAlert(alert)
	if existingID != "" {
		// Update existing alert
		existing := am.alerts[existingID]
		existing.Count++
		existing.LastSeen = alert.Timestamp
		existing.Message = alert.Message

		// If it was resolved, mark as active again
		if existing.Resolved {
			existing.Resolved = false
			existing.ResolvedAt = nil
		}

		alert = existing
	} else {
		// New alert
		alert.Count = 1
		alert.LastSeen = alert.Timestamp
		am.alerts[alert.ID] = alert
	}

	// Notify listeners
	for _, listener := range am.listeners {
		listener.OnAlert(alert)
	}
}

// GetAlert retrieves an alert by ID
func (am *AlertManager) GetAlert(id string) *Alert {
	am.mu.RLock()
	defer am.mu.RUnlock()

	if alert, exists := am.alerts[id]; exists {
		// Return a copy to avoid race conditions
		alertCopy := *alert
		return &alertCopy
	}
	return nil
}

// GetAlerts returns all alerts, optionally filtered by parameters
func (am *AlertManager) GetAlerts(filter *AlertFilter) []*Alert {
	am.mu.RLock()
	defer am.mu.RUnlock()

	var result []*Alert
	for _, alert := range am.alerts {
		if am.matchesFilter(alert, filter) {
			// Create copy to avoid race conditions
			alertCopy := *alert
			result = append(result, &alertCopy)
		}
	}

	return result
}

// AlertFilter defines criteria for filtering alerts
type AlertFilter struct {
	Types        []AlertType
	Severities   []AlertSeverity
	Components   []string
	Acknowledged *bool
	Resolved     *bool
	SinceTime    *time.Time
	UntilTime    *time.Time
}

// GetActiveAlerts returns all unresolved alerts
func (am *AlertManager) GetActiveAlerts() []*Alert {
	resolved := false
	return am.GetAlerts(&AlertFilter{Resolved: &resolved})
}

// GetCriticalAlerts returns all critical alerts
func (am *AlertManager) GetCriticalAlerts() []*Alert {
	return am.GetAlerts(&AlertFilter{
		Severities: []AlertSeverity{AlertSeverityCritical},
		Resolved:   boolPtr(false),
	})
}

// AcknowledgeAlert marks an alert as acknowledged
func (am *AlertManager) AcknowledgeAlert(id, ackedBy string) error {
	am.mu.Lock()
	defer am.mu.Unlock()

	alert, exists := am.alerts[id]
	if !exists {
		return fmt.Errorf("alert %s not found", id)
	}

	if alert.Acknowledged {
		return fmt.Errorf("alert %s already acknowledged", id)
	}

	now := time.Now()
	alert.Acknowledged = true
	alert.AckedBy = ackedBy
	alert.AckedAt = &now

	return nil
}

// ResolveAlert marks an alert as resolved
func (am *AlertManager) ResolveAlert(id string) error {
	am.mu.Lock()
	defer am.mu.Unlock()

	alert, exists := am.alerts[id]
	if !exists {
		return fmt.Errorf("alert %s not found", id)
	}

	if alert.Resolved {
		return fmt.Errorf("alert %s already resolved", id)
	}

	now := time.Now()
	alert.Resolved = true
	alert.ResolvedAt = &now

	// Notify listeners
	for _, listener := range am.listeners {
		listener.OnAlertResolved(alert)
	}

	return nil
}

// ClearResolvedAlerts removes all resolved alerts older than the specified duration
func (am *AlertManager) ClearResolvedAlerts(olderThan time.Duration) int {
	am.mu.Lock()
	defer am.mu.Unlock()

	cutoff := time.Now().Add(-olderThan)
	cleared := 0

	for id, alert := range am.alerts {
		if alert.Resolved && alert.ResolvedAt != nil && alert.ResolvedAt.Before(cutoff) {
			delete(am.alerts, id)
			cleared++
		}
	}

	return cleared
}

// GetStats returns statistics about alerts
func (am *AlertManager) GetStats() AlertStats {
	am.mu.RLock()
	defer am.mu.RUnlock()

	stats := AlertStats{}

	for _, alert := range am.alerts {
		stats.Total++

		switch alert.Severity {
		case AlertSeverityCritical:
			stats.Critical++
		case AlertSeverityWarning:
			stats.Warning++
		case AlertSeverityInfo:
			stats.Info++
		}

		if alert.Acknowledged {
			stats.Acknowledged++
		} else {
			stats.Unacknowledged++
		}

		if alert.Resolved {
			stats.Resolved++
		} else {
			stats.Active++
		}
	}

	return stats
}

// AddListener adds an alert listener
func (am *AlertManager) AddListener(listener AlertListener) {
	am.mu.Lock()
	defer am.mu.Unlock()
	am.listeners = append(am.listeners, listener)
}

// RemoveListener removes an alert listener
func (am *AlertManager) RemoveListener(listener AlertListener) {
	am.mu.Lock()
	defer am.mu.Unlock()

	for i, l := range am.listeners {
		if l == listener {
			am.listeners = append(am.listeners[:i], am.listeners[i+1:]...)
			break
		}
	}
}

// findSimilarAlert finds an existing alert that's similar to the new one
func (am *AlertManager) findSimilarAlert(newAlert *Alert) string {
	for id, existing := range am.alerts {
		if existing.Type == newAlert.Type &&
			existing.Component == newAlert.Component &&
			existing.Severity == newAlert.Severity &&
			!existing.Resolved {
			return id
		}
	}
	return ""
}

// matchesFilter checks if an alert matches the given filter
func (am *AlertManager) matchesFilter(alert *Alert, filter *AlertFilter) bool {
	return am.matchesTypeFilter(alert, filter) &&
		am.matchesSeverityFilter(alert, filter) &&
		am.matchesComponentFilter(alert, filter) &&
		am.matchesAcknowledgedFilter(alert, filter) &&
		am.matchesResolvedFilter(alert, filter) &&
		am.matchesTimeFilter(alert, filter)
}

// matchesTypeFilter checks if alert type matches filter
func (am *AlertManager) matchesTypeFilter(alert *Alert, filter *AlertFilter) bool {
	if len(filter.Types) == 0 {
		return true
	}
	return slices.Contains(filter.Types, alert.Type)
}

// matchesSeverityFilter checks if alert severity matches filter
func (am *AlertManager) matchesSeverityFilter(alert *Alert, filter *AlertFilter) bool {
	if len(filter.Severities) == 0 {
		return true
	}
	return slices.Contains(filter.Severities, alert.Severity)
}

// matchesComponentFilter checks if alert component matches filter
func (am *AlertManager) matchesComponentFilter(alert *Alert, filter *AlertFilter) bool {
	if len(filter.Components) == 0 {
		return true
	}
	return slices.Contains(filter.Components, alert.Component)
}

// matchesAcknowledgedFilter checks if alert acknowledged status matches filter
func (am *AlertManager) matchesAcknowledgedFilter(alert *Alert, filter *AlertFilter) bool {
	if filter.Acknowledged == nil {
		return true
	}
	return alert.Acknowledged == *filter.Acknowledged
}

// matchesResolvedFilter checks if alert resolved status matches filter
func (am *AlertManager) matchesResolvedFilter(alert *Alert, filter *AlertFilter) bool {
	if filter.Resolved == nil {
		return true
	}
	return alert.Resolved == *filter.Resolved
}

// matchesTimeFilter checks if alert timestamp is within filter time range
func (am *AlertManager) matchesTimeFilter(alert *Alert, filter *AlertFilter) bool {
	if filter.SinceTime != nil && alert.Timestamp.Before(*filter.SinceTime) {
		return false
	}
	if filter.UntilTime != nil && alert.Timestamp.After(*filter.UntilTime) {
		return false
	}
	return true
}

// boolPtr returns a pointer to a boolean value
func boolPtr(b bool) *bool {
	return &b
}

// GetSeverityColor returns the color for alert severity display
func GetSeverityColor(severity AlertSeverity) string {
	switch severity {
	case AlertSeverityCritical:
		return "red"
	case AlertSeverityWarning:
		return "yellow"
	case AlertSeverityInfo:
		return "blue"
	default:
		return "white"
	}
}

// GetSeverityIcon returns an icon for alert severity
func GetSeverityIcon(severity AlertSeverity) string {
	switch severity {
	case AlertSeverityCritical:
		return "🔴"
	case AlertSeverityWarning:
		return "🟡"
	case AlertSeverityInfo:
		return "🔵"
	default:
		return "⚪"
	}
}
