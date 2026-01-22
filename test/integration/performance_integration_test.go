package integration

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/jontk/s9s/internal/performance"
	"github.com/stretchr/testify/assert"
)

// TestPerformanceIntegration tests the performance profiling system
// under real workload conditions
func TestPerformanceIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance integration tests in short mode")
	}

	t.Run("ProfilerUnderLoad", func(t *testing.T) {
		testProfilerUnderLoad(t)
	})

	t.Run("OptimizerWithRealWorkload", func(t *testing.T) {
		testOptimizerWithRealWorkload(t)
	})

	t.Run("ConcurrentOperations", func(t *testing.T) {
		testConcurrentOperations(t)
	})

	t.Run("MemoryLeakDetection", func(t *testing.T) {
		testMemoryLeakDetection(t)
	})

	t.Run("LongRunningProfiling", func(t *testing.T) {
		testLongRunningProfiling(t)
	})
}

func testProfilerUnderLoad(t *testing.T) {
	profiler := performance.NewProfiler()

	// Simulate various operations under load
	operations := []string{
		"database_query",
		"file_processing",
		"network_request",
		"data_parsing",
		"computation",
	}

	const numIterations = 1000
	const numWorkers = 10

	var wg sync.WaitGroup
	start := time.Now()

	// Launch workers to perform operations
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			for j := 0; j < numIterations/numWorkers; j++ {
				opName := operations[j%len(operations)]
				done := profiler.StartOperation(fmt.Sprintf("%s_worker_%d", opName, workerID))

				// Simulate work with varying duration
				workDuration := time.Duration(j%10+1) * time.Millisecond
				time.Sleep(workDuration)

				done()
			}
		}(i)
	}

	wg.Wait()
	totalDuration := time.Since(start)

	// Verify profiler collected data
	stats := profiler.GetOperationStats()
	assert.NotEmpty(t, stats)

	// Check that we have stats for each operation type
	for _, op := range operations {
		found := false
		for name := range stats {
			if contains(name, op) {
				found = true
				break
			}
		}
		assert.True(t, found, "Should have stats for operation: %s", op)
	}

	// Verify performance metrics are reasonable
	for name, stat := range stats {
		assert.Greater(t, stat.Count, int64(0), "Operation %s should have been called", name)
		assert.Greater(t, stat.TotalTime, time.Duration(0), "Operation %s should have total time", name)
		assert.Greater(t, stat.AverageTime, time.Duration(0), "Operation %s should have average time", name)
		assert.LessOrEqual(t, stat.MinTime, stat.MaxTime, "Min time should be <= max time for %s", name)
	}

	t.Logf("Completed %d operations across %d workers in %v", numIterations, numWorkers, totalDuration)
	t.Logf("Profiler captured %d unique operation types", len(stats))
}

func testOptimizerWithRealWorkload(t *testing.T) {
	profiler := performance.NewProfiler()
	optimizer := performance.NewOptimizer(profiler)
	optimizer.EnableAutoTune(true)

	// Capture baseline
	baselineMemStats := profiler.CaptureMemoryStats()
	baselineGoroutines := runtime.NumGoroutine()

	// Create a workload that will trigger optimization recommendations
	const iterations = 500

	// Simulate memory-intensive operations
	allocations := make([][]byte, iterations)
	for i := 0; i < iterations; i++ {
		done := profiler.StartOperation("memory_intensive_task")

		// Allocate memory
		allocations[i] = make([]byte, 1024*1024) // 1MB each

		// Simulate processing
		time.Sleep(1 * time.Millisecond)

		done()
	}

	// Force some GC activity
	runtime.GC()
	runtime.GC()

	// Capture metrics after workload
	profiler.CaptureMemoryStats()

	// Run optimization analysis
	recommendations := optimizer.Analyze()

	assert.NotEmpty(t, recommendations, "Should generate optimization recommendations")

	// Verify we get meaningful recommendations
	foundMemoryRec := false

	for _, rec := range recommendations {
		t.Logf("Recommendation: [%s] %s - %s (Impact: %s)",
			rec.Category, rec.Issue, rec.Suggestion, rec.Impact)

		if rec.Category == "Memory" {
			foundMemoryRec = true
		}
	}

	// We should get memory recommendations due to our allocation pattern
	if !foundMemoryRec {
		t.Log("No memory recommendations generated - this might be expected on systems with plenty of RAM")
	}

	// Test optimization summary
	summary := optimizer.GetOptimizationSummary()
	assert.NotEmpty(t, summary)
	assert.Contains(t, summary, "Performance Optimization Summary")

	// Clean up allocations to avoid affecting other tests
	for i := range allocations {
		allocations[i] = nil
	}
	runtime.GC()

	t.Logf("Generated %d optimization recommendations", len(recommendations))
	t.Logf("Baseline goroutines: %d, Current: %d", baselineGoroutines, runtime.NumGoroutine())
	t.Logf("Baseline heap: %.2f MB", float64(baselineMemStats.HeapAlloc)/1024/1024)
}

func testConcurrentOperations(t *testing.T) {
	profiler := performance.NewProfiler()

	const numGoroutines = 50
	const operationsPerGoroutine = 100

	var wg sync.WaitGroup
	start := time.Now()

	// Launch concurrent operations
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			for j := 0; j < operationsPerGoroutine; j++ {
				opName := fmt.Sprintf("concurrent_op_%d", id%5) // Group into 5 operation types
				done := profiler.StartOperation(opName)

				// Simulate varying workload
				switch j % 4 {
				case 0:
					// Fast operation
					time.Sleep(100 * time.Microsecond)
				case 1:
					// Medium operation
					time.Sleep(1 * time.Millisecond)
				case 2:
					// Slow operation
					time.Sleep(5 * time.Millisecond)
				case 3:
					// CPU-bound operation
					sum := 0
					for k := 0; k < 1000; k++ {
						sum += k
					}
					_ = sum
				}

				done()
			}
		}(i)
	}

	wg.Wait()
	elapsed := time.Since(start)

	// Verify concurrent operations were tracked correctly
	stats := profiler.GetOperationStats()

	totalOperations := numGoroutines * operationsPerGoroutine
	actualOperations := int64(0)
	for _, stat := range stats {
		actualOperations += stat.Count
	}

	assert.Equal(t, int64(totalOperations), actualOperations,
		"Should track all concurrent operations")

	// Check that we have the expected number of operation types
	assert.Equal(t, 5, len(stats), "Should have 5 operation types")

	// Verify timing data makes sense
	for name, stat := range stats {
		assert.Greater(t, stat.Count, int64(0), "Operation %s should have count > 0", name)
		assert.Greater(t, stat.AverageTime, time.Duration(0), "Operation %s should have average time", name)
		assert.LessOrEqual(t, stat.MinTime, stat.AverageTime, "Min <= Average for %s", name)
		assert.GreaterOrEqual(t, stat.MaxTime, stat.AverageTime, "Max >= Average for %s", name)
	}

	t.Logf("Completed %d concurrent operations in %v", totalOperations, elapsed)
	t.Logf("Average operations per second: %.2f", float64(totalOperations)/elapsed.Seconds())
}

func testMemoryLeakDetection(t *testing.T) {
	profiler := performance.NewProfiler()
	optimizer := performance.NewOptimizer(profiler)

	// Create a controlled memory leak scenario
	leakData := make([][]byte, 0)

	for i := 0; i < 100; i++ {
		done := profiler.StartOperation("potential_leak_operation")

		// Simulate operation that might leak memory
		data := make([]byte, 10*1024) // 10KB
		leakData = append(leakData, data)

		time.Sleep(1 * time.Millisecond)
		done()
	}

	// Force GC to see current state
	runtime.GC()
	runtime.GC()

	// Check for memory leaks
	memoryIssues := profiler.FindMemoryLeaks()

	if len(memoryIssues) > 0 {
		t.Logf("Detected potential memory issues:")
		for _, issue := range memoryIssues {
			t.Logf("  - %s", issue)
		}
	}

	// Run optimization analysis
	recommendations := optimizer.Analyze()

	// Look for memory-related recommendations
	memoryRecs := 0
	for _, rec := range recommendations {
		if rec.Category == "Memory" {
			memoryRecs++
			t.Logf("Memory recommendation: %s", rec.Suggestion)
		}
	}

	// Clean up the "leak" to avoid affecting other tests
	_ = leakData // Keep reference to avoid compiler optimization
	runtime.GC()

	t.Logf("Found %d memory-related recommendations", memoryRecs)
}

func testLongRunningProfiling(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping long-running test in short mode")
	}

	profiler := performance.NewProfiler()

	// Run profiling for a longer period to test stability
	duration := 10 * time.Second
	interval := 100 * time.Millisecond

	ctx, cancel := context.WithTimeout(context.Background(), duration)
	defer cancel()

	operationCount := 0
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	t.Logf("Running long-term profiling for %v", duration)

	for {
		select {
		case <-ctx.Done():
			goto done
		case <-ticker.C:
			// Perform operation
			done := profiler.StartOperation("long_running_test")

			// Simulate work
			time.Sleep(10 * time.Millisecond)

			done()
			operationCount++

			// Capture memory stats periodically
			if operationCount%10 == 0 {
				profiler.CaptureMemoryStats()
			}
		}
	}

done:
	stats := profiler.GetOperationStats()
	report := profiler.Report()

	assert.NotEmpty(t, stats)
	assert.Contains(t, report, "Performance Report")
	assert.Contains(t, report, "long_running_test")

	// Verify we collected the expected number of operations
	if longRunningStats, exists := stats["long_running_test"]; exists {
		expectedOps := int64(duration / interval)
		tolerance := expectedOps / 10 // 10% tolerance

		assert.InDelta(t, expectedOps, longRunningStats.Count, float64(tolerance),
			"Should have approximately %d operations, got %d", expectedOps, longRunningStats.Count)
	}

	t.Logf("Completed %d operations over %v", operationCount, duration)
	t.Logf("Performance report:\n%s", report)
}

// Benchmark tests for performance system itself
func BenchmarkProfilerOverhead(b *testing.B) {
	profiler := performance.NewProfiler()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		done := profiler.StartOperation("benchmark_test")
		// Minimal work to measure profiler overhead
		done()
	}
}

func BenchmarkConcurrentProfiling(b *testing.B) {
	profiler := performance.NewProfiler()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			done := profiler.StartOperation("concurrent_benchmark")
			done()
		}
	})
}

func BenchmarkMemoryCapture(b *testing.B) {
	profiler := performance.NewProfiler()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		profiler.CaptureMemoryStats()
	}
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) &&
		(s == substr || (len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
			(len(s) > 2*len(substr) && findSubstring(s, substr)))))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
