package ui

import (
	"testing"
	"time"

	"github.com/jontk/s9s/internal/performance"
	"github.com/jontk/s9s/internal/ui/widgets"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPerformanceDashboard(t *testing.T) {
	// Create profiler and optimizer for testing
	profiler := performance.NewProfiler()
	require.NotNil(t, profiler)

	optimizer := performance.NewOptimizer(profiler)
	require.NotNil(t, optimizer)

	// Create dashboard
	dashboard := widgets.NewPerformanceDashboard(profiler, optimizer)
	require.NotNil(t, dashboard)

	t.Run("InitialState", func(t *testing.T) {
		assert.NotNil(t, dashboard.GetContainer())
		assert.False(t, dashboard.IsRunning())
	})

	t.Run("StartStop", func(t *testing.T) {
		// Start monitoring
		err := dashboard.Start()
		assert.NoError(t, err)
		assert.True(t, dashboard.IsRunning())

		// Should not be able to start again
		err = dashboard.Start()
		assert.Error(t, err)

		// Stop monitoring
		dashboard.Stop()
		assert.False(t, dashboard.IsRunning())

		// Should be able to stop multiple times without error
		dashboard.Stop()
		assert.False(t, dashboard.IsRunning())
	})

	t.Run("Configuration", func(t *testing.T) {
		// Test update interval
		dashboard.SetUpdateInterval(500 * time.Millisecond)

		// Test thresholds
		thresholds := widgets.PerformanceThresholds{
			CPUWarning:      60.0,
			CPUCritical:     80.0,
			MemoryWarning:   70.0,
			MemoryCritical:  90.0,
			NetworkWarning:  500.0,
			NetworkCritical: 1000.0,
			OpsWarning:      500.0,
			OpsCritical:     2000.0,
		}
		dashboard.SetThresholds(thresholds)
	})

	t.Run("WithData", func(t *testing.T) {
		// Add some performance data to profiler
		for i := 0; i < 10; i++ {
			stop := profiler.StartOperation("test-operation")
			time.Sleep(10 * time.Millisecond)
			stop()
		}

		// Start dashboard briefly to collect metrics
		err := dashboard.Start()
		require.NoError(t, err)

		// Let it collect some data
		time.Sleep(100 * time.Millisecond)

		// Stop dashboard
		dashboard.Stop()

		// Dashboard should have processed the data
		assert.False(t, dashboard.IsRunning())
	})
}

func TestPerformanceDashboardMetrics(t *testing.T) {
	// Create mock profiler with some data
	profiler := performance.NewProfiler()
	require.NotNil(t, profiler)

	optimizer := performance.NewOptimizer(profiler)
	require.NotNil(t, optimizer)

	dashboard := widgets.NewPerformanceDashboard(profiler, optimizer)

	// Add some test operations
	for i := 0; i < 5; i++ {
		stop1 := profiler.StartOperation("cpu-intensive")
		// Simulate CPU work
		for j := 0; j < 1000; j++ {
			_ = j * j
		}
		stop1()

		stop2 := profiler.StartOperation("network-request")
		time.Sleep(5 * time.Millisecond)
		stop2()
	}

	// Start dashboard
	err := dashboard.Start()
	require.NoError(t, err)
	defer dashboard.Stop()

	// Wait for some metrics to be collected
	time.Sleep(150 * time.Millisecond)

	// Verify dashboard is collecting metrics
	assert.True(t, dashboard.IsRunning())
}

func BenchmarkPerformanceDashboard(b *testing.B) {
	profiler := performance.NewProfiler()
	if profiler == nil {
		b.Skip("Profiler not available")
	}

	optimizer := performance.NewOptimizer(profiler)
	if optimizer == nil {
		b.Skip("Optimizer not available")
	}

	dashboard := widgets.NewPerformanceDashboard(profiler, optimizer)
	dashboard.SetUpdateInterval(10 * time.Millisecond) // Fast updates for benchmarking

	b.Run("MetricsCollection", func(b *testing.B) {
		err := dashboard.Start()
		if err != nil {
			b.Fatal(err)
		}
		defer dashboard.Stop()

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			stop := profiler.StartOperation("benchmark-op")
			stop()
		}
	})

	b.Run("DashboardUpdates", func(b *testing.B) {
		// Add some baseline operations
		for i := 0; i < 100; i++ {
			stop := profiler.StartOperation("test-op")
			stop()
		}

		err := dashboard.Start()
		if err != nil {
			b.Fatal(err)
		}
		defer dashboard.Stop()

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			time.Sleep(10 * time.Millisecond) // Wait for update cycle
		}
	})
}
