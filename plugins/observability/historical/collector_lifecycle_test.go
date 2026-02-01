package historical

import (
	"context"
	"runtime"
	"testing"
	"time"
)

// TestHistoricalDataCollector_NoGoroutineLeaks verifies that collector goroutines exit properly
func TestHistoricalDataCollector_NoGoroutineLeaks(t *testing.T) {
	// Get baseline goroutine count
	runtime.GC()
	time.Sleep(100 * time.Millisecond)
	before := runtime.NumGoroutine()

	// Create collector (nil client is fine for lifecycle tests)
	config := CollectorConfig{
		DataDir:         t.TempDir(),
		Retention:       24 * time.Hour,
		CollectInterval: 1 * time.Second,
		MaxDataPoints:   100,
		Queries:         map[string]string{"test": "up"},
	}

	collector, err := NewHistoricalDataCollector(nil, config)
	if err != nil {
		t.Fatalf("Failed to create collector: %v", err)
	}

	// Start collector (starts 2 goroutines: collectionLoop and cleanupLoop)
	ctx := context.Background()
	if err := collector.Start(ctx); err != nil {
		t.Fatalf("Failed to start collector: %v", err)
	}

	// Give goroutines time to start
	time.Sleep(200 * time.Millisecond)

	// Stop collector
	if err := collector.Stop(); err != nil {
		t.Fatalf("Failed to stop collector: %v", err)
	}

	// Give goroutines time to exit
	time.Sleep(300 * time.Millisecond)
	runtime.GC()
	time.Sleep(100 * time.Millisecond)

	// Check goroutine count
	after := runtime.NumGoroutine()

	// Allow small variance (±2) for test runtime goroutines
	if after > before+2 {
		t.Errorf("Goroutine leak detected: before=%d, after=%d (diff=%d)", before, after, after-before)
	}
}

// TestHistoricalDataCollector_MultipleStartStopCycles verifies multiple start/stop cycles don't leak
func TestHistoricalDataCollector_MultipleStartStopCycles(t *testing.T) {
	config := CollectorConfig{
		DataDir:         t.TempDir(),
		Retention:       24 * time.Hour,
		CollectInterval: 1 * time.Second,
		MaxDataPoints:   100,
		Queries:         map[string]string{"test": "up"},
	}

	collector, err := NewHistoricalDataCollector(nil, config)
	if err != nil {
		t.Fatalf("Failed to create collector: %v", err)
	}

	runtime.GC()
	time.Sleep(100 * time.Millisecond)
	before := runtime.NumGoroutine()

	// Perform multiple start/stop cycles
	ctx := context.Background()
	for i := 0; i < 5; i++ {
		if err := collector.Start(ctx); err != nil {
			t.Fatalf("Cycle %d: Failed to start: %v", i, err)
		}

		time.Sleep(100 * time.Millisecond)

		if err := collector.Stop(); err != nil {
			t.Fatalf("Cycle %d: Failed to stop: %v", i, err)
		}

		time.Sleep(100 * time.Millisecond)
	}

	runtime.GC()
	time.Sleep(200 * time.Millisecond)
	after := runtime.NumGoroutine()

	if after > before+2 {
		t.Errorf("Goroutine leak after multiple cycles: before=%d, after=%d", before, after)
	}
}

// TestHistoricalDataCollector_DoubleStart verifies that starting twice returns error
func TestHistoricalDataCollector_DoubleStart(t *testing.T) {
	config := CollectorConfig{
		DataDir:         t.TempDir(),
		Retention:       24 * time.Hour,
		CollectInterval: 1 * time.Second,
		MaxDataPoints:   100,
		Queries:         map[string]string{"test": "up"},
	}

	collector, err := NewHistoricalDataCollector(nil, config)
	if err != nil {
		t.Fatalf("Failed to create collector: %v", err)
	}

	ctx := context.Background()

	// First start should succeed
	if err := collector.Start(ctx); err != nil {
		t.Fatalf("First start failed: %v", err)
	}

	// Second start should fail
	if err := collector.Start(ctx); err == nil {
		t.Error("Second start should have returned error")
	}

	// Cleanup
	_ = collector.Stop()
}

// TestHistoricalDataCollector_DoubleStop verifies that stopping twice returns error
func TestHistoricalDataCollector_DoubleStop(t *testing.T) {
	config := CollectorConfig{
		DataDir:         t.TempDir(),
		Retention:       24 * time.Hour,
		CollectInterval: 1 * time.Second,
		MaxDataPoints:   100,
		Queries:         map[string]string{"test": "up"},
	}

	collector, err := NewHistoricalDataCollector(nil, config)
	if err != nil {
		t.Fatalf("Failed to create collector: %v", err)
	}

	ctx := context.Background()
	if err := collector.Start(ctx); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	// First stop should succeed
	if err := collector.Stop(); err != nil {
		t.Fatalf("First stop failed: %v", err)
	}

	// Second stop should fail
	if err := collector.Stop(); err == nil {
		t.Error("Second stop should have returned error")
	}
}

// TestHistoricalDataCollector_ConcurrentStop verifies that concurrent stops are safe
func TestHistoricalDataCollector_ConcurrentStop(t *testing.T) {
	config := CollectorConfig{
		DataDir:         t.TempDir(),
		Retention:       24 * time.Hour,
		CollectInterval: 1 * time.Second,
		MaxDataPoints:   100,
		Queries:         map[string]string{"test": "up"},
	}

	collector, err := NewHistoricalDataCollector(nil, config)
	if err != nil {
		t.Fatalf("Failed to create collector: %v", err)
	}

	ctx := context.Background()
	if err := collector.Start(ctx); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	// Launch multiple goroutines calling Stop() concurrently
	done := make(chan bool)
	errors := make(chan error, 10)

	for i := 0; i < 10; i++ {
		go func() {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("Stop() panicked: %v", r)
				}
				done <- true
			}()
			errors <- collector.Stop()
		}()
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}
	close(errors)

	// Exactly one Stop() should succeed, others should error
	successCount := 0
	for err := range errors {
		if err == nil {
			successCount++
		}
	}

	if successCount != 1 {
		t.Errorf("Expected exactly 1 successful stop, got %d", successCount)
	}
}

// TestHistoricalDataCollector_StopWithNil verifies that Stop() handles nil collector
func TestHistoricalDataCollector_StopWithNil(t *testing.T) {
	var collector *HistoricalDataCollector

	// Should not panic
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Stop() panicked with nil collector: %v", r)
		}
	}()

	_ = collector.Stop()
}

// TestHistoricalDataCollector_ContextCancellation verifies that context cancellation stops collection
func TestHistoricalDataCollector_ContextCancellation(t *testing.T) {
	config := CollectorConfig{
		DataDir:         t.TempDir(),
		Retention:       24 * time.Hour,
		CollectInterval: 100 * time.Millisecond, // Short interval for test
		MaxDataPoints:   100,
		Queries:         map[string]string{"test": "up"},
	}

	collector, err := NewHistoricalDataCollector(nil, config)
	if err != nil {
		t.Fatalf("Failed to create collector: %v", err)
	}

	runtime.GC()
	time.Sleep(100 * time.Millisecond)
	before := runtime.NumGoroutine()

	// Create context with cancellation
	ctx, cancel := context.WithCancel(context.Background())

	if err := collector.Start(ctx); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	// Give time for goroutines to start
	time.Sleep(200 * time.Millisecond)

	// Cancel context (should stop collectionLoop)
	cancel()

	// Also call Stop() to stop cleanupLoop
	_ = collector.Stop()

	// Give time for goroutines to exit
	time.Sleep(300 * time.Millisecond)
	runtime.GC()
	time.Sleep(100 * time.Millisecond)

	after := runtime.NumGoroutine()

	if after > before+2 {
		t.Errorf("Goroutine leak after context cancellation: before=%d, after=%d", before, after)
	}
}

// TestHistoricalDataCollector_RapidStartStop verifies rapid start/stop cycles
func TestHistoricalDataCollector_RapidStartStop(t *testing.T) {
	config := CollectorConfig{
		DataDir:         t.TempDir(),
		Retention:       24 * time.Hour,
		CollectInterval: 1 * time.Second,
		MaxDataPoints:   100,
		Queries:         map[string]string{"test": "up"},
	}

	collector, err := NewHistoricalDataCollector(nil, config)
	if err != nil {
		t.Fatalf("Failed to create collector: %v", err)
	}

	ctx := context.Background()

	// Rapid fire start/stop without delays
	for i := 0; i < 10; i++ {
		_ = collector.Start(ctx)
		_ = collector.Stop()
	}

	// No crash = success
	t.Log("Rapid start/stop cycles completed without crash")
}

// TestHistoricalDataCollector_GetStatsWhileRunning verifies stats are accessible during operation
func TestHistoricalDataCollector_GetStatsWhileRunning(t *testing.T) {
	config := CollectorConfig{
		DataDir:         t.TempDir(),
		Retention:       24 * time.Hour,
		CollectInterval: 1 * time.Second,
		MaxDataPoints:   100,
		Queries:         map[string]string{"test": "up"},
	}

	collector, err := NewHistoricalDataCollector(nil, config)
	if err != nil {
		t.Fatalf("Failed to create collector: %v", err)
	}

	ctx := context.Background()
	if err := collector.Start(ctx); err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	defer func() {
		if err := collector.Stop(); err != nil {
			t.Errorf("Stop failed: %v", err)
		}
	}()

	// Get stats multiple times concurrently
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func() {
			stats := collector.GetCollectorStats()
			if stats == nil {
				t.Error("GetCollectorStats returned nil")
			}
			if running, ok := stats["running"].(bool); !ok || !running {
				t.Error("Collector should be running")
			}
			done <- true
		}()
	}

	for i := 0; i < 10; i++ {
		<-done
	}
}
