package monitoring

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Mock alert listener for testing
type mockAlertListener struct {
	onAlertCalls         int
	onAlertResolvedCalls int
	lastAlert            *Alert
}

func (m *mockAlertListener) OnAlert(alert *Alert) {
	m.onAlertCalls++
	m.lastAlert = alert
}

func (m *mockAlertListener) OnAlertResolved(alert *Alert) {
	m.onAlertResolvedCalls++
	m.lastAlert = alert
}

func TestNewAlertManager(t *testing.T) {
	am := NewAlertManager()

	require.NotNil(t, am)
	assert.NotNil(t, am.alerts)
	assert.NotNil(t, am.listeners)
	assert.Empty(t, am.alerts)
	assert.Empty(t, am.listeners)
}

func TestAlertManagerAddAlert(t *testing.T) {
	am := NewAlertManager()

	alert := &Alert{
		ID:        "test-1",
		Type:      AlertTypeHealth,
		Severity:  AlertSeverityWarning,
		Title:     "Test Alert",
		Message:   "Test message",
		Component: "test-component",
		Timestamp: time.Now(),
	}

	am.AddAlert(alert)

	retrieved := am.GetAlert("test-1")
	require.NotNil(t, retrieved)
	assert.Equal(t, "test-1", retrieved.ID)
	assert.Equal(t, AlertTypeHealth, retrieved.Type)
	assert.Equal(t, AlertSeverityWarning, retrieved.Severity)
	assert.Equal(t, 1, retrieved.Count)
}

func TestAlertManagerAddSimilarAlert(t *testing.T) {
	am := NewAlertManager()

	alert1 := &Alert{
		ID:        "test-1",
		Type:      AlertTypeHealth,
		Severity:  AlertSeverityWarning,
		Title:     "Test Alert",
		Message:   "First message",
		Component: "test-component",
		Timestamp: time.Now(),
	}

	alert2 := &Alert{
		ID:        "test-2",
		Type:      AlertTypeHealth,
		Severity:  AlertSeverityWarning,
		Title:     "Test Alert",
		Message:   "Updated message",
		Component: "test-component",
		Timestamp: time.Now().Add(1 * time.Minute),
	}

	am.AddAlert(alert1)
	am.AddAlert(alert2)

	// Should update existing alert instead of creating new one
	retrieved := am.GetAlert("test-1")
	require.NotNil(t, retrieved)
	assert.Equal(t, 2, retrieved.Count)
	assert.Equal(t, "Updated message", retrieved.Message)
}

func TestAlertManagerGetAlert(t *testing.T) {
	am := NewAlertManager()

	alert := &Alert{
		ID:        "test-1",
		Type:      AlertTypeHealth,
		Severity:  AlertSeverityWarning,
		Title:     "Test Alert",
		Message:   "Test message",
		Component: "test-component",
		Timestamp: time.Now(),
	}

	am.AddAlert(alert)

	tests := []struct {
		name     string
		id       string
		expected bool
	}{
		{
			name:     "existing alert",
			id:       "test-1",
			expected: true,
		},
		{
			name:     "non-existent alert",
			id:       "non-existent",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			retrieved := am.GetAlert(tt.id)
			if tt.expected {
				assert.NotNil(t, retrieved)
			} else {
				assert.Nil(t, retrieved)
			}
		})
	}
}

func TestAlertManagerGetAlerts(t *testing.T) {
	am := NewAlertManager()

	// Add multiple alerts
	am.AddAlert(&Alert{
		ID:        "alert-1",
		Type:      AlertTypeHealth,
		Severity:  AlertSeverityCritical,
		Component: "nodes",
		Timestamp: time.Now(),
	})

	am.AddAlert(&Alert{
		ID:        "alert-2",
		Type:      AlertTypePerformance,
		Severity:  AlertSeverityWarning,
		Component: "queue",
		Timestamp: time.Now(),
	})

	am.AddAlert(&Alert{
		ID:        "alert-3",
		Type:      AlertTypeResource,
		Severity:  AlertSeverityInfo,
		Component: "storage",
		Timestamp: time.Now(),
	})

	tests := []struct {
		name          string
		filter        AlertFilter
		expectedCount int
	}{
		{
			name:          "no filter returns all alerts",
			filter:        AlertFilter{},
			expectedCount: 3,
		},
		{
			name: "filter by type",
			filter: AlertFilter{
				Types: []AlertType{AlertTypeHealth},
			},
			expectedCount: 1,
		},
		{
			name: "filter by severity",
			filter: AlertFilter{
				Severities: []AlertSeverity{AlertSeverityCritical, AlertSeverityWarning},
			},
			expectedCount: 2,
		},
		{
			name: "filter by component",
			filter: AlertFilter{
				Components: []string{"nodes"},
			},
			expectedCount: 1,
		},
		{
			name: "multiple filters",
			filter: AlertFilter{
				Types:      []AlertType{AlertTypePerformance},
				Severities: []AlertSeverity{AlertSeverityWarning},
			},
			expectedCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			alerts := am.GetAlerts(tt.filter)
			assert.Len(t, alerts, tt.expectedCount)
		})
	}
}

func TestAlertManagerGetActiveAlerts(t *testing.T) {
	am := NewAlertManager()

	// Add active alert
	am.AddAlert(&Alert{
		ID:        "active-1",
		Type:      AlertTypeHealth,
		Severity:  AlertSeverityWarning,
		Timestamp: time.Now(),
	})

	// Add resolved alert
	resolvedAlert := &Alert{
		ID:        "resolved-1",
		Type:      AlertTypeHealth,
		Severity:  AlertSeverityWarning,
		Timestamp: time.Now(),
		Resolved:  true,
	}
	now := time.Now()
	resolvedAlert.ResolvedAt = &now
	am.AddAlert(resolvedAlert)

	activeAlerts := am.GetActiveAlerts()
	assert.Len(t, activeAlerts, 1)
	assert.Equal(t, "active-1", activeAlerts[0].ID)
}

func TestAlertManagerGetCriticalAlerts(t *testing.T) {
	am := NewAlertManager()

	am.AddAlert(&Alert{
		ID:        "critical-1",
		Type:      AlertTypeHealth,
		Severity:  AlertSeverityCritical,
		Timestamp: time.Now(),
	})

	am.AddAlert(&Alert{
		ID:        "warning-1",
		Type:      AlertTypeHealth,
		Severity:  AlertSeverityWarning,
		Timestamp: time.Now(),
	})

	criticalAlerts := am.GetCriticalAlerts()
	assert.Len(t, criticalAlerts, 1)
	assert.Equal(t, "critical-1", criticalAlerts[0].ID)
}

func TestAlertManagerAcknowledgeAlert(t *testing.T) {
	am := NewAlertManager()

	alert := &Alert{
		ID:        "test-1",
		Type:      AlertTypeHealth,
		Severity:  AlertSeverityWarning,
		Timestamp: time.Now(),
	}
	am.AddAlert(alert)

	tests := []struct {
		name        string
		alertID     string
		ackedBy     string
		expectError bool
	}{
		{
			name:        "acknowledge existing alert",
			alertID:     "test-1",
			ackedBy:     "admin",
			expectError: false,
		},
		{
			name:        "acknowledge non-existent alert",
			alertID:     "non-existent",
			ackedBy:     "admin",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := am.AcknowledgeAlert(tt.alertID, tt.ackedBy)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)

				retrieved := am.GetAlert(tt.alertID)
				require.NotNil(t, retrieved)
				assert.True(t, retrieved.Acknowledged)
				assert.Equal(t, tt.ackedBy, retrieved.AckedBy)
				assert.NotNil(t, retrieved.AckedAt)
			}
		})
	}
}

func TestAlertManagerAcknowledgeAlreadyAcknowledged(t *testing.T) {
	am := NewAlertManager()

	alert := &Alert{
		ID:        "test-1",
		Type:      AlertTypeHealth,
		Severity:  AlertSeverityWarning,
		Timestamp: time.Now(),
	}
	am.AddAlert(alert)

	// Acknowledge once
	err := am.AcknowledgeAlert("test-1", "admin")
	require.NoError(t, err)

	// Try to acknowledge again
	err = am.AcknowledgeAlert("test-1", "admin2")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already acknowledged")
}

func TestAlertManagerResolveAlert(t *testing.T) {
	am := NewAlertManager()

	alert := &Alert{
		ID:        "test-1",
		Type:      AlertTypeHealth,
		Severity:  AlertSeverityWarning,
		Timestamp: time.Now(),
	}
	am.AddAlert(alert)

	tests := []struct {
		name        string
		alertID     string
		expectError bool
	}{
		{
			name:        "resolve existing alert",
			alertID:     "test-1",
			expectError: false,
		},
		{
			name:        "resolve non-existent alert",
			alertID:     "non-existent",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := am.ResolveAlert(tt.alertID)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)

				retrieved := am.GetAlert(tt.alertID)
				require.NotNil(t, retrieved)
				assert.True(t, retrieved.Resolved)
				assert.NotNil(t, retrieved.ResolvedAt)
			}
		})
	}
}

func TestAlertManagerResolveAlreadyResolved(t *testing.T) {
	am := NewAlertManager()

	alert := &Alert{
		ID:        "test-1",
		Type:      AlertTypeHealth,
		Severity:  AlertSeverityWarning,
		Timestamp: time.Now(),
	}
	am.AddAlert(alert)

	// Resolve once
	err := am.ResolveAlert("test-1")
	require.NoError(t, err)

	// Try to resolve again
	err = am.ResolveAlert("test-1")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already resolved")
}

func TestAlertManagerClearResolvedAlerts(t *testing.T) {
	am := NewAlertManager()

	// Add old resolved alert
	oldAlert := &Alert{
		ID:        "old-1",
		Type:      AlertTypeHealth,
		Severity:  AlertSeverityWarning,
		Timestamp: time.Now().Add(-2 * time.Hour),
		Resolved:  true,
	}
	oldResolvedAt := time.Now().Add(-2 * time.Hour)
	oldAlert.ResolvedAt = &oldResolvedAt
	am.AddAlert(oldAlert)

	// Add recent resolved alert
	recentAlert := &Alert{
		ID:        "recent-1",
		Type:      AlertTypeHealth,
		Severity:  AlertSeverityWarning,
		Timestamp: time.Now(),
		Resolved:  true,
	}
	recentResolvedAt := time.Now()
	recentAlert.ResolvedAt = &recentResolvedAt
	am.AddAlert(recentAlert)

	// Add active alert
	am.AddAlert(&Alert{
		ID:        "active-1",
		Type:      AlertTypeHealth,
		Severity:  AlertSeverityWarning,
		Timestamp: time.Now(),
	})

	cleared := am.ClearResolvedAlerts(1 * time.Hour)

	assert.Equal(t, 1, cleared)
	assert.Nil(t, am.GetAlert("old-1"))
	assert.NotNil(t, am.GetAlert("recent-1"))
	assert.NotNil(t, am.GetAlert("active-1"))
}

func TestAlertManagerGetStats(t *testing.T) {
	am := NewAlertManager()

	// Add various alerts
	am.AddAlert(&Alert{
		ID:        "critical-1",
		Type:      AlertTypeHealth,
		Severity:  AlertSeverityCritical,
		Timestamp: time.Now(),
	})

	am.AddAlert(&Alert{
		ID:        "warning-1",
		Type:      AlertTypeHealth,
		Severity:  AlertSeverityWarning,
		Timestamp: time.Now(),
	})

	am.AddAlert(&Alert{
		ID:        "info-1",
		Type:      AlertTypeHealth,
		Severity:  AlertSeverityInfo,
		Timestamp: time.Now(),
	})

	// Acknowledge one
	_ = am.AcknowledgeAlert("critical-1", "admin")

	// Resolve one
	_ = am.ResolveAlert("info-1")

	stats := am.GetStats()

	assert.Equal(t, 3, stats.Total)
	assert.Equal(t, 1, stats.Critical)
	assert.Equal(t, 1, stats.Warning)
	assert.Equal(t, 1, stats.Info)
	assert.Equal(t, 1, stats.Acknowledged)
	assert.Equal(t, 2, stats.Unacknowledged)
	assert.Equal(t, 2, stats.Active)
	assert.Equal(t, 1, stats.Resolved)
}

func TestAlertManagerAddListener(t *testing.T) {
	am := NewAlertManager()
	listener := &mockAlertListener{}

	am.AddListener(listener)

	alert := &Alert{
		ID:        "test-1",
		Type:      AlertTypeHealth,
		Severity:  AlertSeverityWarning,
		Timestamp: time.Now(),
	}

	am.AddAlert(alert)

	assert.Equal(t, 1, listener.onAlertCalls)
	assert.Equal(t, "test-1", listener.lastAlert.ID)
}

func TestAlertManagerRemoveListener(t *testing.T) {
	am := NewAlertManager()
	listener := &mockAlertListener{}

	am.AddListener(listener)
	am.RemoveListener(listener)

	alert := &Alert{
		ID:        "test-1",
		Type:      AlertTypeHealth,
		Severity:  AlertSeverityWarning,
		Timestamp: time.Now(),
	}

	am.AddAlert(alert)

	assert.Equal(t, 0, listener.onAlertCalls)
}

func TestAlertManagerResolveNotifiesListeners(t *testing.T) {
	am := NewAlertManager()
	listener := &mockAlertListener{}
	am.AddListener(listener)

	alert := &Alert{
		ID:        "test-1",
		Type:      AlertTypeHealth,
		Severity:  AlertSeverityWarning,
		Timestamp: time.Now(),
	}

	am.AddAlert(alert)
	_ = am.ResolveAlert("test-1")

	assert.Equal(t, 1, listener.onAlertResolvedCalls)
}

func TestGetSeverityColor(t *testing.T) {
	tests := []struct {
		name     string
		severity AlertSeverity
		expected string
	}{
		{
			name:     "critical is red",
			severity: AlertSeverityCritical,
			expected: "red",
		},
		{
			name:     "warning is yellow",
			severity: AlertSeverityWarning,
			expected: "yellow",
		},
		{
			name:     "info is blue",
			severity: AlertSeverityInfo,
			expected: "blue",
		},
		{
			name:     "unknown is white",
			severity: AlertSeverity("unknown"),
			expected: "white",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			color := GetSeverityColor(tt.severity)
			assert.Equal(t, tt.expected, color)
		})
	}
}

func TestGetSeverityIcon(t *testing.T) {
	tests := []struct {
		name     string
		severity AlertSeverity
		expected string
	}{
		{
			name:     "critical icon",
			severity: AlertSeverityCritical,
			expected: "🔴",
		},
		{
			name:     "warning icon",
			severity: AlertSeverityWarning,
			expected: "🟡",
		},
		{
			name:     "info icon",
			severity: AlertSeverityInfo,
			expected: "🔵",
		},
		{
			name:     "unknown icon",
			severity: AlertSeverity("unknown"),
			expected: "⚪",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			icon := GetSeverityIcon(tt.severity)
			assert.Equal(t, tt.expected, icon)
		})
	}
}

func TestAlertFilterTimeRange(t *testing.T) {
	am := NewAlertManager()

	baseTime := time.Now()

	// Add alerts at different times
	oldAlert := &Alert{
		ID:        "old",
		Type:      AlertTypeHealth,
		Severity:  AlertSeverityWarning,
		Timestamp: baseTime.Add(-2 * time.Hour),
	}
	am.alerts["old"] = oldAlert

	recentAlert := &Alert{
		ID:        "recent",
		Type:      AlertTypeHealth,
		Severity:  AlertSeverityWarning,
		Timestamp: baseTime.Add(-30 * time.Minute),
	}
	am.alerts["recent"] = recentAlert

	newAlert := &Alert{
		ID:        "new",
		Type:      AlertTypeHealth,
		Severity:  AlertSeverityWarning,
		Timestamp: baseTime,
	}
	am.alerts["new"] = newAlert

	sinceTime := baseTime.Add(-1 * time.Hour)
	untilTime := baseTime.Add(-15 * time.Minute)

	// Filter for alerts in the last hour
	alerts := am.GetAlerts(AlertFilter{
		SinceTime: &sinceTime,
	})
	assert.Len(t, alerts, 2) // recent and new

	// Filter for alerts before 15 minutes ago
	alerts = am.GetAlerts(AlertFilter{
		UntilTime: &untilTime,
	})
	assert.Len(t, alerts, 2) // old and recent
}

func TestAlertTypeConstants(t *testing.T) {
	assert.Equal(t, AlertType("health"), AlertTypeHealth)
	assert.Equal(t, AlertType("performance"), AlertTypePerformance)
	assert.Equal(t, AlertType("resource"), AlertTypeResource)
	assert.Equal(t, AlertType("job"), AlertTypeJob)
	assert.Equal(t, AlertType("node"), AlertTypeNode)
	assert.Equal(t, AlertType("system"), AlertTypeSystem)
}

func TestAlertSeverityConstants(t *testing.T) {
	assert.Equal(t, AlertSeverity("info"), AlertSeverityInfo)
	assert.Equal(t, AlertSeverity("warning"), AlertSeverityWarning)
	assert.Equal(t, AlertSeverity("critical"), AlertSeverityCritical)
}
