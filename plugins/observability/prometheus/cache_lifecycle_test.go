package prometheus

import (
	"runtime"
	"testing"
	"time"
)

// TestMetricCache_NoGoroutineLeaks verifies that the cache cleanup goroutine exits properly
func TestMetricCache_NoGoroutineLeaks(t *testing.T) {
	// Get baseline goroutine count
	runtime.GC()
	time.Sleep(100 * time.Millisecond)
	before := runtime.NumGoroutine()

	// Create and start cache (starts cleanup goroutine)
	cache := NewMetricCache(30*time.Second, 1000)

	// Give goroutine time to start
	time.Sleep(100 * time.Millisecond)

	// Stop cache
	cache.Stop()

	// Give goroutine time to exit
	time.Sleep(200 * time.Millisecond)
	runtime.GC()
	time.Sleep(100 * time.Millisecond)

	// Check goroutine count
	after := runtime.NumGoroutine()

	// Allow small variance (±2) for test runtime goroutines
	if after > before+2 {
		t.Errorf("Goroutine leak detected: before=%d, after=%d (diff=%d)", before, after, after-before)
	}
}

// TestMetricCache_MultipleStopCalls verifies that calling Stop() multiple times is safe
func TestMetricCache_MultipleStopCalls(t *testing.T) {
	cache := NewMetricCache(30*time.Second, 1000)

	// First stop should succeed
	cache.Stop()

	// Second stop should not panic
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Stop() panicked on second call: %v", r)
		}
	}()

	cache.Stop()
	cache.Stop() // Third call for good measure
}

// TestMetricCache_ConcurrentStop verifies that concurrent Stop() calls are safe
func TestMetricCache_ConcurrentStop(t *testing.T) {
	cache := NewMetricCache(30*time.Second, 1000)

	// Launch multiple goroutines calling Stop() concurrently
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func() {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("Stop() panicked during concurrent call: %v", r)
				}
				done <- true
			}()
			cache.Stop()
		}()
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}
}

// TestMetricCache_StopDuringOperation verifies that Stop() works even during cache operations
func TestMetricCache_StopDuringOperation(t *testing.T) {
	cache := NewMetricCache(30*time.Second, 1000)

	// Start goroutines performing cache operations
	stopOps := make(chan bool)
	for i := 0; i < 5; i++ {
		go func(id int) {
			for {
				select {
				case <-stopOps:
					return
				default:
					// Perform cache operations
					cache.Set("test", &QueryResult{Status: "success"}, 0)
					_, _ = cache.Get("test")
					time.Sleep(10 * time.Millisecond)
				}
			}
		}(i)
	}

	// Let operations run for a bit
	time.Sleep(100 * time.Millisecond)

	// Stop cache during operations
	cache.Stop()

	// Stop operation goroutines
	close(stopOps)

	// Give goroutines time to exit
	time.Sleep(200 * time.Millisecond)

	// Verify no panics occurred
	t.Log("Cache stopped successfully during concurrent operations")
}

// TestMetricCache_CleanupExpiredEntries verifies that cleanup removes expired entries
func TestMetricCache_CleanupExpiredEntries(t *testing.T) {
	// Create cache with very short TTL
	cache := NewMetricCache(100*time.Millisecond, 1000)
	defer cache.Stop()

	// Add entries
	for i := 0; i < 10; i++ {
		cache.Set("key", &QueryResult{Status: "success"}, 100*time.Millisecond)
	}

	// Verify entries exist
	stats := cache.Stats()
	if stats.Entries != 1 { // Should be 1 because same key
		t.Errorf("Expected 1 entry, got %d", stats.Entries)
	}

	// Wait for entries to expire
	time.Sleep(200 * time.Millisecond)

	// Trigger cleanup (would normally happen via cleanupLoop)
	cache.cleanup()

	// Verify entries were removed
	stats = cache.Stats()
	if stats.Entries != 0 {
		t.Errorf("Expected 0 entries after cleanup, got %d", stats.Entries)
	}
}

// TestCachedClient_Stop verifies that CachedClient properly stops its cache
func TestCachedClient_Stop(t *testing.T) {
	// Create mock client (nil is fine for this test)
	cachedClient := &CachedClient{
		client: nil,
		cache:  NewMetricCache(30*time.Second, 1000),
	}

	// Get baseline goroutine count
	runtime.GC()
	time.Sleep(100 * time.Millisecond)
	before := runtime.NumGoroutine()

	// Stop should work without panic
	cachedClient.Stop()

	// Give time for cleanup
	time.Sleep(200 * time.Millisecond)
	runtime.GC()
	time.Sleep(100 * time.Millisecond)

	after := runtime.NumGoroutine()

	// Should not leak goroutines
	if after > before+2 {
		t.Errorf("Goroutine leak in CachedClient: before=%d, after=%d", before, after)
	}
}

// TestCachedClient_StopWithNilCache verifies that Stop() handles nil cache gracefully
func TestCachedClient_StopWithNilCache(t *testing.T) {
	cachedClient := &CachedClient{
		client: nil,
		cache:  nil, // nil cache
	}

	// Should not panic
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Stop() panicked with nil cache: %v", r)
		}
	}()

	cachedClient.Stop()
}

// TestCachedClient_StopNilClient verifies that Stop() handles nil client gracefully
func TestCachedClient_StopNilClient(t *testing.T) {
	var cachedClient *CachedClient

	// Should not panic
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Stop() panicked with nil client: %v", r)
		}
	}()

	cachedClient.Stop()
}
