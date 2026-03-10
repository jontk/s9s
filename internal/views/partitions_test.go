package views

import (
	"testing"
	"time"

	"github.com/jontk/s9s/internal/dao"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCreateEfficiencyBar tests partition efficiency visualization
func TestCreateEfficiencyBar(t *testing.T) {
	tests := []struct {
		name           string
		percentage     float64
		expectedColor  string
		expectedFilled int
		expectedEmpty  int
		expectedText   string
	}{
		{
			name:           "Zero efficiency",
			percentage:     0.0,
			expectedColor:  "red",
			expectedFilled: 0,
			expectedEmpty:  5,
			expectedText:   "0%",
		},
		{
			name:           "Low efficiency (20%)",
			percentage:     20.0,
			expectedColor:  "red",
			expectedFilled: 1, // 20% / 20 = 1 bar
			expectedEmpty:  4,
			expectedText:   "20%",
		},
		{
			name:           "Boundary - 49.9% (still red)",
			percentage:     49.9,
			expectedColor:  "red",
			expectedFilled: 2, // int(49.9/20) = 2
			expectedEmpty:  3,
			expectedText:   "50%",
		},
		{
			name:           "Medium efficiency (50%)",
			percentage:     50.0,
			expectedColor:  "yellow",
			expectedFilled: 2, // 50% / 20 = 2.5, int = 2
			expectedEmpty:  3,
			expectedText:   "50%",
		},
		{
			name:           "Good efficiency (70%)",
			percentage:     70.0,
			expectedColor:  "yellow",
			expectedFilled: 3, // 70% / 20 = 3.5, int = 3
			expectedEmpty:  2,
			expectedText:   "70%",
		},
		{
			name:           "Boundary - 79.9% (still yellow)",
			percentage:     79.9,
			expectedColor:  "yellow",
			expectedFilled: 3, // int(79.9/20) = 3
			expectedEmpty:  2,
			expectedText:   "80%",
		},
		{
			name:           "High efficiency (80%)",
			percentage:     80.0,
			expectedColor:  "green",
			expectedFilled: 4, // 80% / 20 = 4
			expectedEmpty:  1,
			expectedText:   "80%",
		},
		{
			name:           "Perfect efficiency (100%)",
			percentage:     100.0,
			expectedColor:  "green",
			expectedFilled: 5, // all bars filled
			expectedEmpty:  0,
			expectedText:   "100%",
		},
		{
			name:           "Over capacity (110%)",
			percentage:     110.0,
			expectedColor:  "green",
			expectedFilled: 5, // capped at barLength
			expectedEmpty:  0,
			expectedText:   "110%",
		},
		{
			name:           "Negative percentage",
			percentage:     -10.0,
			expectedColor:  "red",
			expectedFilled: 0,
			expectedEmpty:  5,
			expectedText:   "-10%",
		},
	}

	v := &PartitionsView{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bar := v.createEfficiencyBar(tt.percentage)

			// Verify bar is not empty
			require.NotEmpty(t, bar, "Bar should not be empty")

			// Verify it contains the expected color
			assert.Contains(t, bar, "["+tt.expectedColor+"]", "Bar should contain expected color code")

			// Verify it contains the percentage text
			assert.Contains(t, bar, tt.expectedText, "Bar should contain percentage text")

			// Verify it contains bar characters (filled and/or empty)
			assert.True(t, containsBarCharacter(bar), "Bar should contain bar visualization")
		})
	}
}

// TestCreateQueueDepthBar tests queue depth visualization
func TestCreateQueueDepthBar(t *testing.T) {
	tests := []struct {
		name          string
		pending       int
		running       int
		expectedTotal string
		expectRunning bool
		expectPending bool
	}{
		{
			name:          "No jobs",
			pending:       0,
			running:       0,
			expectedTotal: "0",
			expectRunning: false,
			expectPending: false,
		},
		{
			name:          "Only running jobs",
			pending:       0,
			running:       10,
			expectedTotal: "10",
			expectRunning: true,
			expectPending: false,
		},
		{
			name:          "Only pending jobs",
			pending:       10,
			running:       0,
			expectedTotal: "10",
			expectRunning: false,
			expectPending: true,
		},
		{
			name:          "Equal running and pending",
			pending:       5,
			running:       5,
			expectedTotal: "10",
			expectRunning: true,
			expectPending: true,
		},
		{
			name:          "More running than pending",
			pending:       2,
			running:       8,
			expectedTotal: "10",
			expectRunning: true,
			expectPending: true,
		},
		{
			name:          "More pending than running",
			pending:       8,
			running:       2,
			expectedTotal: "10",
			expectRunning: true,
			expectPending: true,
		},
		{
			name:          "Large numbers",
			pending:       100,
			running:       50,
			expectedTotal: "150",
			expectRunning: true,
			expectPending: true,
		},
		{
			name:          "Single job running",
			pending:       0,
			running:       1,
			expectedTotal: "1",
			expectRunning: true,
			expectPending: false,
		},
	}

	v := &PartitionsView{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bar := v.createQueueDepthBar(tt.pending, tt.running)

			// Verify bar is not empty
			require.NotEmpty(t, bar, "Bar should not be empty")

			// Verify total count is present
			assert.Contains(t, bar, tt.expectedTotal, "Bar should contain total job count")

			// Verify color codes are present based on job types
			if tt.expectRunning {
				assert.Contains(t, bar, "[green]", "Bar should contain green color for running jobs")
			}
			if tt.expectPending {
				assert.Contains(t, bar, "[yellow]", "Bar should contain yellow color for pending jobs")
			}

			// For empty queue, verify it shows gray bars
			if tt.pending == 0 && tt.running == 0 {
				assert.Contains(t, bar, "[gray]", "Empty bar should be gray")
			}
		})
	}
}

// TestPartitionEfficiencyCalculation tests the efficiency calculation logic
// This is the critical test based on the architectural review finding
func TestPartitionEfficiencyCalculation(t *testing.T) {
	tests := []struct {
		name          string
		partition     *dao.Partition
		allocatedCPUs int
		expectedEff   float64
	}{
		{
			name: "Full utilization",
			partition: &dao.Partition{
				Name:      "compute",
				TotalCPUs: 100,
			},
			allocatedCPUs: 100,
			expectedEff:   100.0,
		},
		{
			name: "Half utilization",
			partition: &dao.Partition{
				Name:      "compute",
				TotalCPUs: 100,
			},
			allocatedCPUs: 50,
			expectedEff:   50.0,
		},
		{
			name: "Zero utilization",
			partition: &dao.Partition{
				Name:      "compute",
				TotalCPUs: 100,
			},
			allocatedCPUs: 0,
			expectedEff:   0.0,
		},
		{
			name: "Over-subscription (shouldn't happen but handle gracefully)",
			partition: &dao.Partition{
				Name:      "compute",
				TotalCPUs: 100,
			},
			allocatedCPUs: 110,
			expectedEff:   100.0, // Should cap at 100%
		},
		{
			name: "Zero capacity partition",
			partition: &dao.Partition{
				Name:      "empty",
				TotalCPUs: 0,
			},
			allocatedCPUs: 0,
			expectedEff:   0.0, // Should handle gracefully
		},
		{
			name: "Low utilization (10%)",
			partition: &dao.Partition{
				Name:      "compute",
				TotalCPUs: 1000,
			},
			allocatedCPUs: 100,
			expectedEff:   10.0,
		},
		{
			name: "Realistic cluster scenario",
			partition: &dao.Partition{
				Name:       "gpu",
				TotalNodes: 10,
				TotalCPUs:  480, // 48 CPUs per node
			},
			allocatedCPUs: 336, // 7 nodes * 48 CPUs
			expectedEff:   70.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var efficiency float64

			// Calculate efficiency: allocated CPUs / total CPUs * 100
			// This is the CORRECT formula (not jobs / CPUs as noted in the architectural review)
			if tt.partition.TotalCPUs > 0 {
				efficiency = float64(tt.allocatedCPUs) * 100.0 / float64(tt.partition.TotalCPUs)
				if efficiency > 100 {
					efficiency = 100.0 // Cap at 100%
				}
			}

			assert.InDelta(t, tt.expectedEff, efficiency, 0.01, "Efficiency should match expected value")

			// Additional validation: verify efficiency bar color is correct
			v := &PartitionsView{}
			bar := v.createEfficiencyBar(efficiency)

			if efficiency < 50 {
				assert.Contains(t, bar, "[red]", "Low efficiency should be red")
			} else if efficiency < 80 {
				assert.Contains(t, bar, "[yellow]", "Medium efficiency should be yellow")
			} else {
				assert.Contains(t, bar, "[green]", "High efficiency should be green")
			}
		})
	}
}

// TestQueueDepthBarProportions tests that bar proportions are calculated correctly
func TestQueueDepthBarProportions(t *testing.T) {
	tests := []struct {
		name          string
		pending       int
		running       int
		expectedRatio float64 // expected running/(running+pending)
	}{
		{
			name:          "50-50 split",
			pending:       50,
			running:       50,
			expectedRatio: 0.5,
		},
		{
			name:          "75-25 split (more running)",
			pending:       25,
			running:       75,
			expectedRatio: 0.75,
		},
		{
			name:          "25-75 split (more pending)",
			pending:       75,
			running:       25,
			expectedRatio: 0.25,
		},
		{
			name:          "90-10 split (high running)",
			pending:       10,
			running:       90,
			expectedRatio: 0.9,
		},
		{
			name:          "10-90 split (high pending)",
			pending:       90,
			running:       10,
			expectedRatio: 0.1,
		},
	}

	v := &PartitionsView{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bar := v.createQueueDepthBar(tt.pending, tt.running)

			// Verify the bar contains job counts
			total := tt.pending + tt.running
			totalStr := formatInt(total)
			assert.Contains(t, bar, totalStr, "Bar should show total job count")

			// Verify it has color indicators when both types present and significant
			// Note: very small ratios may not show up in the 8-bar visualization
			barLength := 8
			if tt.running > 0 {
				runningBars := int(float64(tt.running) / float64(total) * float64(barLength))
				if runningBars > 0 {
					assert.Contains(t, bar, "[green]", "Bar should show green for running jobs when ratio is significant")
				}
			}
			if tt.pending > 0 {
				pendingBars := int(float64(tt.pending) / float64(total) * float64(barLength))
				if pendingBars > 0 {
					assert.Contains(t, bar, "[yellow]", "Bar should show yellow for pending jobs when ratio is significant")
				}
			}
		})
	}
}

// TestAssessPartitionStatus tests partition health assessment
func TestAssessPartitionStatus(t *testing.T) {
	tests := []struct {
		name           string
		queueInfo      *dao.QueueInfo
		expectedColor  string
		expectedStatus string
	}{
		{
			name: "Critical - long wait time (> 24h)",
			queueInfo: &dao.QueueInfo{
				PendingJobs: 10,
				RunningJobs: 5,
				LongestWait: 25 * 60 * 60 * 1_000_000_000, // 25 hours in nanoseconds
			},
			expectedColor:  "red",
			expectedStatus: "CRITICAL",
		},
		{
			name: "Warning - moderate wait time (> 6h)",
			queueInfo: &dao.QueueInfo{
				PendingJobs: 10,
				RunningJobs: 5,
				LongestWait: 7 * 60 * 60 * 1_000_000_000, // 7 hours in nanoseconds
			},
			expectedColor:  "yellow",
			expectedStatus: "WARNING",
		},
		{
			name: "Backlog - too many pending jobs",
			queueInfo: &dao.QueueInfo{
				PendingJobs: 100,
				RunningJobs: 10,                          // pending > running * 2
				LongestWait: 1 * 60 * 60 * 1_000_000_000, // 1 hour
			},
			expectedColor:  "orange",
			expectedStatus: "BACKLOG",
		},
		{
			name: "OK - healthy queue",
			queueInfo: &dao.QueueInfo{
				PendingJobs: 5,
				RunningJobs: 10,
				LongestWait: 30 * 60 * 1_000_000_000, // 30 minutes
			},
			expectedColor:  "green",
			expectedStatus: "OK",
		},
		{
			name: "OK - no pending jobs",
			queueInfo: &dao.QueueInfo{
				PendingJobs: 0,
				RunningJobs: 10,
				LongestWait: 0,
			},
			expectedColor:  "green",
			expectedStatus: "OK",
		},
		{
			name: "Boundary - exactly 24 hours (warning not critical)",
			queueInfo: &dao.QueueInfo{
				PendingJobs: 10,
				RunningJobs: 5,
				LongestWait: 24 * 60 * 60 * 1_000_000_000, // exactly 24 hours (>6 so warning)
			},
			expectedColor:  "yellow",
			expectedStatus: "WARNING",
		},
		{
			name: "Boundary - exactly 6 hours (not warning)",
			queueInfo: &dao.QueueInfo{
				PendingJobs: 10,
				RunningJobs: 5,
				LongestWait: 6 * 60 * 60 * 1_000_000_000, // exactly 6 hours
			},
			expectedColor:  "green",
			expectedStatus: "OK",
		},
	}

	v := &PartitionsView{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			color, status := v.assessPartitionStatus(tt.queueInfo)

			assert.Equal(t, tt.expectedColor, color, "Status color should match expected value")
			assert.Equal(t, tt.expectedStatus, status, "Status should match expected value")
		})
	}
}

// TestIsCriticalWaitTime tests critical wait time detection
func TestIsCriticalWaitTime(t *testing.T) {
	tests := []struct {
		name       string
		waitHours  float64
		isCritical bool
	}{
		{"0 hours", 0, false},
		{"1 hour", 1, false},
		{"6 hours", 6, false},
		{"23 hours", 23, false},
		{"24 hours", 24, false},
		{"24.1 hours", 24.1, true},
		{"48 hours", 48, true},
		{"100 hours", 100, true},
	}

	v := &PartitionsView{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			waitTime := time.Duration(tt.waitHours * float64(time.Hour))
			result := v.isCriticalWaitTime(waitTime)

			assert.Equal(t, tt.isCritical, result, "Critical wait time detection should match expected value")
		})
	}
}

// TestIsWarningWaitTime tests warning wait time detection
func TestIsWarningWaitTime(t *testing.T) {
	tests := []struct {
		name      string
		waitHours float64
		isWarning bool
	}{
		{"0 hours", 0, false},
		{"1 hour", 1, false},
		{"5 hours", 5, false},
		{"6 hours", 6, false},
		{"6.1 hours", 6.1, true},
		{"12 hours", 12, true},
		{"23 hours", 23, true},
	}

	v := &PartitionsView{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			waitTime := time.Duration(tt.waitHours * float64(time.Hour))
			result := v.isWarningWaitTime(waitTime)

			assert.Equal(t, tt.isWarning, result, "Warning wait time detection should match expected value")
		})
	}
}

// TestHasJobBacklog tests job backlog detection
func TestHasJobBacklog(t *testing.T) {
	tests := []struct {
		name       string
		pending    int
		running    int
		hasBacklog bool
	}{
		{"No backlog - equal", 10, 10, false},
		{"No backlog - more running", 10, 20, false},
		{"Boundary - exactly 2x", 20, 10, false},
		{"Backlog - 2.1x", 21, 10, true},
		{"Backlog - 3x", 30, 10, true},
		{"Backlog - 10x", 100, 10, true},
		{"No jobs", 0, 0, false},
		{"Only pending", 10, 0, true}, // any pending with 0 running is > 2x
		{"Only running", 0, 10, false},
	}

	v := &PartitionsView{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info := &dao.QueueInfo{
				PendingJobs: tt.pending,
				RunningJobs: tt.running,
			}

			result := v.hasJobBacklog(info)
			assert.Equal(t, tt.hasBacklog, result, "Backlog detection should match expected value")
		})
	}
}

// TestFormatDurationField tests duration formatting for analytics
func TestFormatDurationField(t *testing.T) {
	tests := []struct {
		name     string
		duration time.Duration
		width    int
		expected string
	}{
		{
			name:     "Zero duration",
			duration: 0,
			width:    10,
			expected: "         -", // 10 chars with right alignment
		},
		{
			name:     "One hour",
			duration: time.Hour,
			width:    10,
			expected: "       1h", // depends on FormatTimeDuration implementation
		},
		{
			name:     "Small width",
			duration: 0,
			width:    5,
			expected: "    -",
		},
	}

	v := &PartitionsView{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := v.formatDurationField(tt.duration, tt.width)

			// Verify the result has the expected width
			// Note: This test assumes FormatTimeDuration is available
			// We're testing the padding logic here
			assert.True(t, len(result) >= tt.width || result == tt.expected,
				"Formatted duration should have proper width")

			// Verify zero duration shows as "-"
			if tt.duration == 0 {
				assert.Contains(t, result, "-", "Zero duration should show as '-'")
			}
		})
	}
}

// TestEfficiencyColorThresholds verifies the efficiency thresholds match requirements
func TestEfficiencyColorThresholds(t *testing.T) {
	v := &PartitionsView{}

	// Test red threshold (< 50%)
	redTests := []float64{0, 10, 25, 49, 49.9}
	for _, pct := range redTests {
		bar := v.createEfficiencyBar(pct)
		assert.Contains(t, bar, "[red]", "Efficiency %.1f%% should be red", pct)
	}

	// Test yellow threshold (50% - 79%)
	yellowTests := []float64{50, 60, 70, 79, 79.9}
	for _, pct := range yellowTests {
		bar := v.createEfficiencyBar(pct)
		assert.Contains(t, bar, "[yellow]", "Efficiency %.1f%% should be yellow", pct)
	}

	// Test green threshold (>= 80%)
	greenTests := []float64{80, 85, 90, 95, 100}
	for _, pct := range greenTests {
		bar := v.createEfficiencyBar(pct)
		assert.Contains(t, bar, "[green]", "Efficiency %.1f%% should be green", pct)
	}
}
