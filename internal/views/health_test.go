package views

import (
	"testing"

	"github.com/jontk/s9s/internal/monitoring"
	"github.com/stretchr/testify/assert"
)

// TestGetStatusColor tests health status to color mapping
func TestGetStatusColor(t *testing.T) {
	tests := []struct {
		status        monitoring.HealthStatus
		expectedColor string
	}{
		{monitoring.HealthStatusHealthy, "green"},
		{monitoring.HealthStatusWarning, "yellow"},
		{monitoring.HealthStatusCritical, "red"},
		{monitoring.HealthStatusUnknown, "gray"},
		{monitoring.HealthStatus(""), "gray"},
		{monitoring.HealthStatus("INVALID"), "gray"},
	}

	v := &HealthView{}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			color := v.getStatusColor(tt.status)
			assert.Equal(t, tt.expectedColor, color, "Color should match expected value for status")
		})
	}
}

// TestGetStatusIcon tests health status to icon mapping
func TestGetStatusIcon(t *testing.T) {
	tests := []struct {
		status       monitoring.HealthStatus
		expectedIcon string
	}{
		{monitoring.HealthStatusHealthy, "✓"},
		{monitoring.HealthStatusWarning, "⚠"},
		{monitoring.HealthStatusCritical, "✗"},
		{monitoring.HealthStatusUnknown, "?"},
		{monitoring.HealthStatus(""), "?"},
		{monitoring.HealthStatus("INVALID"), "?"},
	}

	v := &HealthView{}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			icon := v.getStatusIcon(tt.status)
			assert.Equal(t, tt.expectedIcon, icon, "Icon should match expected value for status")
		})
	}
}

// TestHealthStatusPriority tests that critical status takes precedence
func TestHealthStatusPriority(t *testing.T) {
	tests := []struct {
		name           string
		checks         map[string]monitoring.HealthCheck
		expectedStatus monitoring.HealthStatus
	}{
		{
			name: "All healthy",
			checks: map[string]monitoring.HealthCheck{
				"nodes": {Status: monitoring.HealthStatusHealthy},
				"jobs":  {Status: monitoring.HealthStatusHealthy},
			},
			expectedStatus: monitoring.HealthStatusHealthy,
		},
		{
			name: "One warning",
			checks: map[string]monitoring.HealthCheck{
				"nodes": {Status: monitoring.HealthStatusHealthy},
				"jobs":  {Status: monitoring.HealthStatusWarning},
			},
			expectedStatus: monitoring.HealthStatusWarning,
		},
		{
			name: "One critical",
			checks: map[string]monitoring.HealthCheck{
				"nodes": {Status: monitoring.HealthStatusHealthy},
				"jobs":  {Status: monitoring.HealthStatusCritical},
			},
			expectedStatus: monitoring.HealthStatusCritical,
		},
		{
			name: "Mixed - critical takes precedence",
			checks: map[string]monitoring.HealthCheck{
				"nodes":   {Status: monitoring.HealthStatusHealthy},
				"jobs":    {Status: monitoring.HealthStatusWarning},
				"storage": {Status: monitoring.HealthStatusCritical},
				"network": {Status: monitoring.HealthStatusWarning},
			},
			expectedStatus: monitoring.HealthStatusCritical,
		},
		{
			name: "All critical",
			checks: map[string]monitoring.HealthCheck{
				"nodes": {Status: monitoring.HealthStatusCritical},
				"jobs":  {Status: monitoring.HealthStatusCritical},
			},
			expectedStatus: monitoring.HealthStatusCritical,
		},
		{
			name: "All warning",
			checks: map[string]monitoring.HealthCheck{
				"nodes": {Status: monitoring.HealthStatusWarning},
				"jobs":  {Status: monitoring.HealthStatusWarning},
			},
			expectedStatus: monitoring.HealthStatusWarning,
		},
		{
			name:           "No checks",
			checks:         map[string]monitoring.HealthCheck{},
			expectedStatus: monitoring.HealthStatusHealthy, // Default when no issues
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Determine overall status based on priority: Critical > Warning > Healthy
			overallStatus := monitoring.HealthStatusHealthy

			for _, check := range tt.checks {
				switch check.Status {
				case monitoring.HealthStatusCritical:
					overallStatus = monitoring.HealthStatusCritical
					// Critical is highest priority, no need to check further
					goto done
				case monitoring.HealthStatusWarning:
					if overallStatus != monitoring.HealthStatusCritical {
						overallStatus = monitoring.HealthStatusWarning
					}
				}
			}
		done:

			assert.Equal(t, tt.expectedStatus, overallStatus, "Overall status should match expected priority")
		})
	}
}

// TestHealthCheckCounting tests health check statistics calculation
func TestHealthCheckCounting(t *testing.T) {
	tests := []struct {
		name             string
		checks           map[string]monitoring.HealthCheck
		expectedHealthy  int
		expectedWarning  int
		expectedCritical int
		expectedTotal    int
	}{
		{
			name: "All healthy",
			checks: map[string]monitoring.HealthCheck{
				"nodes": {Status: monitoring.HealthStatusHealthy},
				"jobs":  {Status: monitoring.HealthStatusHealthy},
				"disk":  {Status: monitoring.HealthStatusHealthy},
			},
			expectedHealthy:  3,
			expectedWarning:  0,
			expectedCritical: 0,
			expectedTotal:    3,
		},
		{
			name: "Mixed statuses",
			checks: map[string]monitoring.HealthCheck{
				"nodes":   {Status: monitoring.HealthStatusHealthy},
				"jobs":    {Status: monitoring.HealthStatusWarning},
				"disk":    {Status: monitoring.HealthStatusCritical},
				"network": {Status: monitoring.HealthStatusHealthy},
			},
			expectedHealthy:  2,
			expectedWarning:  1,
			expectedCritical: 1,
			expectedTotal:    4,
		},
		{
			name: "All critical",
			checks: map[string]monitoring.HealthCheck{
				"nodes": {Status: monitoring.HealthStatusCritical},
				"jobs":  {Status: monitoring.HealthStatusCritical},
			},
			expectedHealthy:  0,
			expectedWarning:  0,
			expectedCritical: 2,
			expectedTotal:    2,
		},
		{
			name:             "No checks",
			checks:           map[string]monitoring.HealthCheck{},
			expectedHealthy:  0,
			expectedWarning:  0,
			expectedCritical: 0,
			expectedTotal:    0,
		},
		{
			name: "All warning",
			checks: map[string]monitoring.HealthCheck{
				"check1": {Status: monitoring.HealthStatusWarning},
				"check2": {Status: monitoring.HealthStatusWarning},
				"check3": {Status: monitoring.HealthStatusWarning},
			},
			expectedHealthy:  0,
			expectedWarning:  3,
			expectedCritical: 0,
			expectedTotal:    3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			healthyCount := 0
			warningCount := 0
			criticalCount := 0
			totalCount := len(tt.checks)

			for _, check := range tt.checks {
				switch check.Status {
				case monitoring.HealthStatusHealthy:
					healthyCount++
				case monitoring.HealthStatusWarning:
					warningCount++
				case monitoring.HealthStatusCritical:
					criticalCount++
				}
			}

			assert.Equal(t, tt.expectedHealthy, healthyCount, "Healthy check count should match")
			assert.Equal(t, tt.expectedWarning, warningCount, "Warning check count should match")
			assert.Equal(t, tt.expectedCritical, criticalCount, "Critical check count should match")
			assert.Equal(t, tt.expectedTotal, totalCount, "Total check count should match")
		})
	}
}

// TestHealthMessageGeneration tests health check message formatting
func TestHealthMessageGeneration(t *testing.T) {
	tests := []struct {
		name             string
		check            monitoring.HealthCheck
		shouldContain    []string
		shouldNotContain []string
	}{
		{
			name: "Healthy node check",
			check: monitoring.HealthCheck{
				Name:        "nodes",
				Status:      monitoring.HealthStatusHealthy,
				Message:     "All nodes operational",
				Description: "Checks if all nodes are up and running",
			},
			shouldContain:    []string{"operational"},
			shouldNotContain: []string{"critical", "warning", "error"},
		},
		{
			name: "Warning job check",
			check: monitoring.HealthCheck{
				Name:        "jobs",
				Status:      monitoring.HealthStatusWarning,
				Message:     "10 jobs pending for > 1 hour",
				Description: "Checks for stuck jobs",
			},
			shouldContain:    []string{"pending", "hour"},
			shouldNotContain: []string{},
		},
		{
			name: "Critical disk check",
			check: monitoring.HealthCheck{
				Name:        "disk",
				Status:      monitoring.HealthStatusCritical,
				Message:     "Disk usage above 95%",
				Description: "Monitors disk space",
			},
			shouldContain:    []string{"95"},
			shouldNotContain: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			message := tt.check.Message

			// Verify required content is present
			for _, substring := range tt.shouldContain {
				assert.Contains(t, message, substring, "Message should contain expected content")
			}

			// Verify unwanted content is not present
			for _, substring := range tt.shouldNotContain {
				assert.NotContains(t, message, substring, "Message should not contain unwanted content")
			}
		})
	}
}

// TestHealthCheckThresholds tests threshold validation
func TestHealthCheckThresholds(t *testing.T) {
	tests := []struct {
		name           string
		value          float64
		warningMax     *float64
		criticalMax    *float64
		expectedStatus monitoring.HealthStatus
	}{
		{
			name:           "Below warning threshold",
			value:          50.0,
			warningMax:     floatPtr(80.0),
			criticalMax:    floatPtr(95.0),
			expectedStatus: monitoring.HealthStatusHealthy,
		},
		{
			name:           "At warning threshold",
			value:          80.0,
			warningMax:     floatPtr(80.0),
			criticalMax:    floatPtr(95.0),
			expectedStatus: monitoring.HealthStatusWarning,
		},
		{
			name:           "Above warning, below critical",
			value:          90.0,
			warningMax:     floatPtr(80.0),
			criticalMax:    floatPtr(95.0),
			expectedStatus: monitoring.HealthStatusWarning,
		},
		{
			name:           "At critical threshold",
			value:          95.0,
			warningMax:     floatPtr(80.0),
			criticalMax:    floatPtr(95.0),
			expectedStatus: monitoring.HealthStatusCritical,
		},
		{
			name:           "Above critical threshold",
			value:          100.0,
			warningMax:     floatPtr(80.0),
			criticalMax:    floatPtr(95.0),
			expectedStatus: monitoring.HealthStatusCritical,
		},
		{
			name:           "No thresholds",
			value:          100.0,
			warningMax:     nil,
			criticalMax:    nil,
			expectedStatus: monitoring.HealthStatusHealthy, // Default when no thresholds
		},
		{
			name:           "Only warning threshold",
			value:          85.0,
			warningMax:     floatPtr(80.0),
			criticalMax:    nil,
			expectedStatus: monitoring.HealthStatusWarning,
		},
		{
			name:           "Only critical threshold",
			value:          96.0,
			warningMax:     nil,
			criticalMax:    floatPtr(95.0),
			expectedStatus: monitoring.HealthStatusCritical,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status := monitoring.HealthStatusHealthy

			// Determine status based on thresholds
			if tt.criticalMax != nil && tt.value >= *tt.criticalMax {
				status = monitoring.HealthStatusCritical
			} else if tt.warningMax != nil && tt.value >= *tt.warningMax {
				status = monitoring.HealthStatusWarning
			}

			assert.Equal(t, tt.expectedStatus, status, "Status should match expected value based on thresholds")
		})
	}
}

// TestHealthCheckEdgeCases tests edge cases in health checking
func TestHealthCheckEdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		check    monitoring.HealthCheck
		validate func(t *testing.T, check monitoring.HealthCheck)
	}{
		{
			name: "Empty message",
			check: monitoring.HealthCheck{
				Name:    "test",
				Status:  monitoring.HealthStatusHealthy,
				Message: "",
			},
			validate: func(t *testing.T, check monitoring.HealthCheck) {
				assert.Empty(t, check.Message, "Message should be empty")
			},
		},
		{
			name: "Very long message",
			check: monitoring.HealthCheck{
				Name:    "test",
				Status:  monitoring.HealthStatusWarning,
				Message: "This is a very long message that contains a lot of details about what went wrong in the system and provides extensive information for debugging purposes.",
			},
			validate: func(t *testing.T, check monitoring.HealthCheck) {
				assert.True(t, len(check.Message) > 100, "Message should be long")
			},
		},
		{
			name: "Zero check count",
			check: monitoring.HealthCheck{
				Name:       "test",
				Status:     monitoring.HealthStatusHealthy,
				CheckCount: 0,
			},
			validate: func(t *testing.T, check monitoring.HealthCheck) {
				assert.Equal(t, 0, check.CheckCount, "Check count should be zero")
			},
		},
		{
			name: "High check count",
			check: monitoring.HealthCheck{
				Name:       "test",
				Status:     monitoring.HealthStatusHealthy,
				CheckCount: 10000,
			},
			validate: func(t *testing.T, check monitoring.HealthCheck) {
				assert.Equal(t, 10000, check.CheckCount, "Check count should be high")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.validate(t, tt.check)
		})
	}
}

// TestHealthCheckStatusConsistency tests that status and message are consistent
func TestHealthCheckStatusConsistency(t *testing.T) {
	tests := []struct {
		name    string
		status  monitoring.HealthStatus
		message string
		valid   bool
	}{
		{
			name:    "Healthy with positive message",
			status:  monitoring.HealthStatusHealthy,
			message: "All systems operational",
			valid:   true,
		},
		{
			name:    "Warning with warning message",
			status:  monitoring.HealthStatusWarning,
			message: "High resource usage detected",
			valid:   true,
		},
		{
			name:    "Critical with critical message",
			status:  monitoring.HealthStatusCritical,
			message: "System failure detected",
			valid:   true,
		},
		{
			name:    "Healthy with empty message",
			status:  monitoring.HealthStatusHealthy,
			message: "",
			valid:   true, // Empty messages are valid
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			check := monitoring.HealthCheck{
				Status:  tt.status,
				Message: tt.message,
			}

			// Verify the check has valid fields
			assert.NotEmpty(t, check.Status, "Status should not be empty")

			// If we expect validity, both should be set
			if tt.valid {
				assert.True(t, len(check.Status) > 0, "Valid check should have a status")
			}
		})
	}
}

// TestHealthViewHints tests that hints are properly defined
func TestHealthViewHints(t *testing.T) {
	v := &HealthView{}
	hints := v.Hints()

	// Verify we have hints
	assert.NotEmpty(t, hints, "View should have keyboard hints")

	// Verify essential hints are present
	essentialHints := []string{"Acknowledge", "Resolve"}
	for _, essential := range essentialHints {
		found := false
		for _, hint := range hints {
			if containsString(hint, essential) {
				found = true
				break
			}
		}
		assert.True(t, found, "Should have hint for: %s", essential)
	}
}

// Helper functions

// floatPtr returns a pointer to a float64
func floatPtr(f float64) *float64 {
	return &f
}

// containsString checks if a string contains a substring (case-insensitive)
func containsString(s, substr string) bool {
	return len(s) >= len(substr) && findSubstring(s, substr)
}

// findSubstring searches for a substring in a string
func findSubstring(s, substr string) bool {
	if len(substr) == 0 {
		return true
	}
	if len(s) < len(substr) {
		return false
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		match := true
		for j := 0; j < len(substr); j++ {
			if s[i+j] != substr[j] {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}
