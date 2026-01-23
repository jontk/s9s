package dao

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetJobStateColor(t *testing.T) {
	tests := []struct {
		name     string
		state    string
		expected string
	}{
		{
			name:     "running job should be green",
			state:    JobStateRunning,
			expected: "green",
		},
		{
			name:     "pending job should be yellow",
			state:    JobStatePending,
			expected: "yellow",
		},
		{
			name:     "completed job should be cyan",
			state:    JobStateCompleted,
			expected: "cyan",
		},
		{
			name:     "failed job should be red",
			state:    JobStateFailed,
			expected: "red",
		},
		{
			name:     "cancelled job should be gray",
			state:    JobStateCancelled,
			expected: "gray",
		},
		{
			name:     "suspended job should be orange",
			state:    JobStateSuspended,
			expected: "orange",
		},
		{
			name:     "unknown state should be white",
			state:    "UNKNOWN",
			expected: "white",
		},
		{
			name:     "empty state should be white",
			state:    "",
			expected: "white",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetJobStateColor(tt.state)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetNodeStateColor(t *testing.T) {
	tests := []struct {
		name     string
		state    string
		expected string
	}{
		{
			name:     "idle node should be green",
			state:    NodeStateIdle,
			expected: "green",
		},
		{
			name:     "allocated node should be blue",
			state:    NodeStateAllocated,
			expected: "blue",
		},
		{
			name:     "mixed node should be blue",
			state:    NodeStateMixed,
			expected: "blue",
		},
		{
			name:     "down node should be red",
			state:    NodeStateDown,
			expected: "red",
		},
		{
			name:     "drain node should be red",
			state:    NodeStateDrain,
			expected: "red",
		},
		{
			name:     "draining node should be red",
			state:    NodeStateDraining,
			expected: "red",
		},
		{
			name:     "reserved node should be yellow",
			state:    NodeStateReserved,
			expected: "yellow",
		},
		{
			name:     "maintenance node should be orange",
			state:    NodeStateMaintenance,
			expected: "orange",
		},
		{
			name:     "compound state IDLE+DRAIN should prioritize drain (red)",
			state:    "IDLE+DRAIN",
			expected: "red",
		},
		{
			name:     "compound state ALLOCATED+DRAIN should prioritize drain (red)",
			state:    "ALLOCATED+DRAIN",
			expected: "red",
		},
		{
			name:     "compound state DOWN+DRAIN should be red",
			state:    "DOWN+DRAIN",
			expected: "red",
		},
		{
			name:     "unknown state should be white",
			state:    "UNKNOWN",
			expected: "white",
		},
		{
			name:     "empty state should be white",
			state:    "",
			expected: "white",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetNodeStateColor(tt.state)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsJobActive(t *testing.T) {
	tests := []struct {
		name     string
		state    string
		expected bool
	}{
		{
			name:     "pending job is active",
			state:    JobStatePending,
			expected: true,
		},
		{
			name:     "running job is active",
			state:    JobStateRunning,
			expected: true,
		},
		{
			name:     "suspended job is active",
			state:    JobStateSuspended,
			expected: true,
		},
		{
			name:     "configuring job is active",
			state:    JobStateConfiguring,
			expected: true,
		},
		{
			name:     "completed job is not active",
			state:    JobStateCompleted,
			expected: false,
		},
		{
			name:     "failed job is not active",
			state:    JobStateFailed,
			expected: false,
		},
		{
			name:     "cancelled job is not active",
			state:    JobStateCancelled,
			expected: false,
		},
		{
			name:     "timeout job is not active",
			state:    JobStateTimeout,
			expected: false,
		},
		{
			name:     "unknown state is not active",
			state:    "UNKNOWN",
			expected: false,
		},
		{
			name:     "empty state is not active",
			state:    "",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsJobActive(tt.state)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsNodeAvailable(t *testing.T) {
	tests := []struct {
		name     string
		state    string
		expected bool
	}{
		{
			name:     "idle node is available",
			state:    NodeStateIdle,
			expected: true,
		},
		{
			name:     "mixed node is available",
			state:    NodeStateMixed,
			expected: true,
		},
		{
			name:     "allocated node is not available",
			state:    NodeStateAllocated,
			expected: false,
		},
		{
			name:     "down node is not available",
			state:    NodeStateDown,
			expected: false,
		},
		{
			name:     "drain node is not available",
			state:    NodeStateDrain,
			expected: false,
		},
		{
			name:     "draining node is not available",
			state:    NodeStateDraining,
			expected: false,
		},
		{
			name:     "reserved node is not available",
			state:    NodeStateReserved,
			expected: false,
		},
		{
			name:     "maintenance node is not available",
			state:    NodeStateMaintenance,
			expected: false,
		},
		{
			name:     "unknown state is not available",
			state:    "UNKNOWN",
			expected: false,
		},
		{
			name:     "empty state is not available",
			state:    "",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsNodeAvailable(tt.state)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetPartitionStateColor(t *testing.T) {
	tests := []struct {
		name     string
		state    string
		expected string
	}{
		{
			name:     "up partition should be green",
			state:    PartitionStateUp,
			expected: "green",
		},
		{
			name:     "down partition should be red",
			state:    PartitionStateDown,
			expected: "red",
		},
		{
			name:     "drain partition should be orange",
			state:    PartitionStateDrain,
			expected: "orange",
		},
		{
			name:     "inactive partition should be gray",
			state:    PartitionStateInactive,
			expected: "gray",
		},
		{
			name:     "unknown state should be white",
			state:    "UNKNOWN",
			expected: "white",
		},
		{
			name:     "empty state should be white",
			state:    "",
			expected: "white",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetPartitionStateColor(tt.state)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestJobStateConstants verifies all job state constants are defined
func TestJobStateConstants(t *testing.T) {
	assert.Equal(t, "PENDING", JobStatePending)
	assert.Equal(t, "RUNNING", JobStateRunning)
	assert.Equal(t, "COMPLETED", JobStateCompleted)
	assert.Equal(t, "FAILED", JobStateFailed)
	assert.Equal(t, "CANCELLED", JobStateCancelled)
	assert.Equal(t, "SUSPENDED", JobStateSuspended)
	assert.Equal(t, "COMPLETING", JobStateCompleting)
	assert.Equal(t, "CONFIGURING", JobStateConfiguring)
	assert.Equal(t, "PREEMPTED", JobStatePreempted)
	assert.Equal(t, "TIMEOUT", JobStateTimeout)
}

// TestNodeStateConstants verifies all node state constants are defined
func TestNodeStateConstants(t *testing.T) {
	assert.Equal(t, "IDLE", NodeStateIdle)
	assert.Equal(t, "ALLOCATED", NodeStateAllocated)
	assert.Equal(t, "MIXED", NodeStateMixed)
	assert.Equal(t, "DOWN", NodeStateDown)
	assert.Equal(t, "DRAIN", NodeStateDrain)
	assert.Equal(t, "DRAINING", NodeStateDraining)
	assert.Equal(t, "RESERVED", NodeStateReserved)
	assert.Equal(t, "MAINTENANCE", NodeStateMaintenance)
}

// TestPartitionStateConstants verifies all partition state constants are defined
func TestPartitionStateConstants(t *testing.T) {
	assert.Equal(t, "UP", PartitionStateUp)
	assert.Equal(t, "DOWN", PartitionStateDown)
	assert.Equal(t, "DRAIN", PartitionStateDrain)
	assert.Equal(t, "INACTIVE", PartitionStateInactive)
}
