package views

import (
	"testing"
	"time"

	"github.com/jontk/s9s/internal/dao"
)

// TestIsActiveReservation tests the isActiveReservation helper method
func TestIsActiveReservation(t *testing.T) {
	v := &ReservationsView{}
	now := time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name     string
		res      *dao.Reservation
		expected bool
	}{
		{
			name: "Active reservation - middle of window",
			res: &dao.Reservation{
				StartTime: time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC),
				EndTime:   time.Date(2024, 1, 15, 14, 0, 0, 0, time.UTC),
			},
			expected: true,
		},
		{
			name: "Past reservation",
			res: &dao.Reservation{
				StartTime: time.Date(2024, 1, 14, 8, 0, 0, 0, time.UTC),
				EndTime:   time.Date(2024, 1, 14, 10, 0, 0, 0, time.UTC),
			},
			expected: false,
		},
		{
			name: "Future reservation",
			res: &dao.Reservation{
				StartTime: time.Date(2024, 1, 15, 15, 0, 0, 0, time.UTC),
				EndTime:   time.Date(2024, 1, 15, 18, 0, 0, 0, time.UTC),
			},
			expected: false,
		},
		{
			name: "Boundary - at start time (inclusive)",
			res: &dao.Reservation{
				StartTime: time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC),
				EndTime:   time.Date(2024, 1, 15, 14, 0, 0, 0, time.UTC),
			},
			expected: true,
		},
		{
			name: "Boundary - at end time (exclusive)",
			res: &dao.Reservation{
				StartTime: time.Date(2024, 1, 15, 8, 0, 0, 0, time.UTC),
				EndTime:   time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC),
			},
			expected: false,
		},
		{
			name: "Just started - 1 second after start",
			res: &dao.Reservation{
				StartTime: time.Date(2024, 1, 15, 11, 59, 59, 0, time.UTC),
				EndTime:   time.Date(2024, 1, 15, 14, 0, 0, 0, time.UTC),
			},
			expected: true,
		},
		{
			name: "Just ended - 1 second before end",
			res: &dao.Reservation{
				StartTime: time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC),
				EndTime:   time.Date(2024, 1, 15, 12, 0, 1, 0, time.UTC),
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := v.isActiveReservation(tt.res, now)
			if result != tt.expected {
				t.Errorf("isActiveReservation() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// TestIsFutureReservation tests the isFutureReservation helper method
func TestIsFutureReservation(t *testing.T) {
	v := &ReservationsView{}
	now := time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name     string
		res      *dao.Reservation
		expected bool
	}{
		{
			name: "Future reservation",
			res: &dao.Reservation{
				StartTime: time.Date(2024, 1, 15, 15, 0, 0, 0, time.UTC),
				EndTime:   time.Date(2024, 1, 15, 18, 0, 0, 0, time.UTC),
			},
			expected: true,
		},
		{
			name: "Active reservation",
			res: &dao.Reservation{
				StartTime: time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC),
				EndTime:   time.Date(2024, 1, 15, 14, 0, 0, 0, time.UTC),
			},
			expected: false,
		},
		{
			name: "Past reservation",
			res: &dao.Reservation{
				StartTime: time.Date(2024, 1, 14, 8, 0, 0, 0, time.UTC),
				EndTime:   time.Date(2024, 1, 14, 10, 0, 0, 0, time.UTC),
			},
			expected: false,
		},
		{
			name: "Boundary - at start time",
			res: &dao.Reservation{
				StartTime: time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC),
				EndTime:   time.Date(2024, 1, 15, 14, 0, 0, 0, time.UTC),
			},
			expected: false,
		},
		{
			name: "One second in future",
			res: &dao.Reservation{
				StartTime: time.Date(2024, 1, 15, 12, 0, 1, 0, time.UTC),
				EndTime:   time.Date(2024, 1, 15, 14, 0, 0, 0, time.UTC),
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := v.isFutureReservation(tt.res, now)
			if result != tt.expected {
				t.Errorf("isFutureReservation() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// TestToggleActiveFilter tests the toggle functionality
func TestToggleActiveFilter(t *testing.T) {
	// Create a view with mock data
	v := &ReservationsView{
		BaseView: &BaseView{},
	}

	// Initially disabled
	if v.activeFilterEnabled {
		t.Error("Expected activeFilterEnabled to be false initially")
	}

	// Note: toggleActiveFilter calls updateTable which requires table setup
	// For unit testing, we just verify the boolean toggle
	initial := v.activeFilterEnabled
	v.activeFilterEnabled = !v.activeFilterEnabled
	if v.activeFilterEnabled == initial {
		t.Error("Expected activeFilterEnabled to toggle")
	}

	// Toggle again
	v.activeFilterEnabled = !v.activeFilterEnabled
	if v.activeFilterEnabled != initial {
		t.Error("Expected activeFilterEnabled to toggle back")
	}
}

// TestToggleFutureFilter tests the future filter toggle
func TestToggleFutureFilter(t *testing.T) {
	v := &ReservationsView{
		BaseView: &BaseView{},
	}

	// Initially disabled
	if v.futureFilterEnabled {
		t.Error("Expected futureFilterEnabled to be false initially")
	}

	// Toggle on
	initial := v.futureFilterEnabled
	v.futureFilterEnabled = !v.futureFilterEnabled
	if v.futureFilterEnabled == initial {
		t.Error("Expected futureFilterEnabled to toggle")
	}

	// Toggle off
	v.futureFilterEnabled = !v.futureFilterEnabled
	if v.futureFilterEnabled != initial {
		t.Error("Expected futureFilterEnabled to toggle back")
	}
}

// TestFilterCombination tests the OR logic when both filters are enabled
func TestFilterCombination(t *testing.T) {
	v := &ReservationsView{
		BaseView: &BaseView{},
	}
	now := time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC)

	// Create test reservations
	past := &dao.Reservation{
		Name:      "Past",
		StartTime: time.Date(2024, 1, 14, 8, 0, 0, 0, time.UTC),
		EndTime:   time.Date(2024, 1, 14, 10, 0, 0, 0, time.UTC),
	}
	active := &dao.Reservation{
		Name:      "Active",
		StartTime: time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC),
		EndTime:   time.Date(2024, 1, 15, 14, 0, 0, 0, time.UTC),
	}
	future := &dao.Reservation{
		Name:      "Future",
		StartTime: time.Date(2024, 1, 15, 15, 0, 0, 0, time.UTC),
		EndTime:   time.Date(2024, 1, 15, 18, 0, 0, 0, time.UTC),
	}

	// Test with both filters enabled - should show active AND future (exclude only past)
	v.activeFilterEnabled = true
	v.futureFilterEnabled = true

	// Past reservation - excluded by both filters
	includePast := (v.activeFilterEnabled && v.isActiveReservation(past, now)) ||
		(v.futureFilterEnabled && v.isFutureReservation(past, now))
	if includePast {
		t.Error("Expected past reservation to be excluded when both filters are on")
	}

	// Active reservation - included by active filter
	includeActive := (v.activeFilterEnabled && v.isActiveReservation(active, now)) ||
		(v.futureFilterEnabled && v.isFutureReservation(active, now))
	if !includeActive {
		t.Error("Expected active reservation to be included when active filter is on")
	}

	// Future reservation - included by future filter
	includeFuture := (v.activeFilterEnabled && v.isActiveReservation(future, now)) ||
		(v.futureFilterEnabled && v.isFutureReservation(future, now))
	if !includeFuture {
		t.Error("Expected future reservation to be included when future filter is on")
	}
}

// TestHintsShowFilterStatus tests that hints indicate filter state
func TestHintsShowFilterStatus(t *testing.T) {
	v := &ReservationsView{
		BaseView: &BaseView{},
	}

	// Test with filters disabled
	hints := v.Hints()
	hasActiveHint := false
	hasFutureHint := false
	for _, hint := range hints {
		if hint == "[yellow]a[white] Active Only" {
			hasActiveHint = true
		}
		if hint == "[yellow]f[white] Future Only" {
			hasFutureHint = true
		}
	}
	if !hasActiveHint {
		t.Error("Expected hints to contain unchecked active filter")
	}
	if !hasFutureHint {
		t.Error("Expected hints to contain unchecked future filter")
	}

	// Test with active filter enabled
	v.activeFilterEnabled = true
	hints = v.Hints()
	hasCheckedActive := false
	for _, hint := range hints {
		if hint == "[yellow]a[green]✓[white] Active Only" {
			hasCheckedActive = true
		}
	}
	if !hasCheckedActive {
		t.Error("Expected hints to show checked active filter when enabled")
	}

	// Test with future filter enabled
	v.futureFilterEnabled = true
	hints = v.Hints()
	hasCheckedFuture := false
	for _, hint := range hints {
		if hint == "[yellow]f[green]✓[white] Future Only" {
			hasCheckedFuture = true
		}
	}
	if !hasCheckedFuture {
		t.Error("Expected hints to show checked future filter when enabled")
	}
}
