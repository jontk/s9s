package output

import (
	"testing"
	"time"
)

// TestOutputCache_GetSet tests basic get/set operations
func TestOutputCache_GetSet(t *testing.T) {
	cache := NewOutputCache(5*time.Minute, 10)

	// Create test content
	content := &OutputContent{
		Content: "Test output",
		Source:  "local",
	}

	// Set in cache
	cache.Set("job123:stdout", content, 0)

	// Get from cache
	retrieved, found := cache.Get("job123:stdout")
	if !found {
		t.Fatal("Expected to find entry in cache")
	}

	if retrieved.Content != content.Content {
		t.Errorf("Content mismatch: got %q, want %q", retrieved.Content, content.Content)
	}
}

// TestOutputCache_Expiration tests TTL expiration
func TestOutputCache_Expiration(t *testing.T) {
	cache := NewOutputCache(100*time.Millisecond, 10)

	content := &OutputContent{Content: "Test"}
	cache.Set("job123:stdout", content, 50*time.Millisecond)

	// Should be available immediately
	_, found := cache.Get("job123:stdout")
	if !found {
		t.Fatal("Expected to find entry immediately after set")
	}

	// Wait for expiration
	time.Sleep(60 * time.Millisecond)

	// Should be expired now
	_, found = cache.Get("job123:stdout")
	if found {
		t.Error("Expected entry to be expired")
	}
}

// TestOutputCache_Eviction tests LRU eviction
func TestOutputCache_Eviction(t *testing.T) {
	cache := NewOutputCache(5*time.Minute, 3) // Small cache

	// Fill cache
	for i := 1; i <= 3; i++ {
		content := &OutputContent{Content: "Test"}
		cache.Set("job"+string(rune('0'+i))+":stdout", content, 0)
		time.Sleep(10 * time.Millisecond) // Ensure different timestamps
	}

	// Verify all entries exist
	stats := cache.Stats()
	if stats.Entries != 3 {
		t.Errorf("Expected 3 entries, got %d", stats.Entries)
	}

	// Add one more entry - should evict oldest
	content := &OutputContent{Content: "New"}
	cache.Set("job4:stdout", content, 0)

	stats = cache.Stats()
	if stats.Entries != 3 {
		t.Errorf("Expected 3 entries after eviction, got %d", stats.Entries)
	}

	if stats.Evictions != 1 {
		t.Errorf("Expected 1 eviction, got %d", stats.Evictions)
	}

	// First entry should be evicted
	_, found := cache.Get("job1:stdout")
	if found {
		t.Error("Expected first entry to be evicted")
	}

	// New entry should exist
	_, found = cache.Get("job4:stdout")
	if !found {
		t.Error("Expected new entry to exist")
	}
}

// TestOutputCache_Delete tests deletion
func TestOutputCache_Delete(t *testing.T) {
	cache := NewOutputCache(5*time.Minute, 10)

	content := &OutputContent{Content: "Test"}
	cache.Set("job123:stdout", content, 0)

	// Verify it exists
	_, found := cache.Get("job123:stdout")
	if !found {
		t.Fatal("Expected entry to exist after set")
	}

	// Delete it
	cache.Delete("job123:stdout")

	// Verify it's gone
	_, found = cache.Get("job123:stdout")
	if found {
		t.Error("Expected entry to be deleted")
	}
}

// TestOutputCache_Clear tests clearing all entries
func TestOutputCache_Clear(t *testing.T) {
	cache := NewOutputCache(5*time.Minute, 10)

	// Add multiple entries
	for i := 1; i <= 5; i++ {
		content := &OutputContent{Content: "Test"}
		cache.Set("job"+string(rune('0'+i))+":stdout", content, 0)
	}

	stats := cache.Stats()
	if stats.Entries != 5 {
		t.Errorf("Expected 5 entries, got %d", stats.Entries)
	}

	// Clear cache
	cache.Clear()

	stats = cache.Stats()
	if stats.Entries != 0 {
		t.Errorf("Expected 0 entries after clear, got %d", stats.Entries)
	}

	if stats.Hits != 0 || stats.Misses != 0 {
		t.Errorf("Expected stats to be reset, got hits=%d, misses=%d", stats.Hits, stats.Misses)
	}
}

// TestOutputCache_CleanupExpired tests cleanup of expired entries
func TestOutputCache_CleanupExpired(t *testing.T) {
	cache := NewOutputCache(100*time.Millisecond, 10)

	// Add entries with short TTL
	for i := 1; i <= 3; i++ {
		content := &OutputContent{Content: "Test"}
		cache.Set("job"+string(rune('0'+i))+":stdout", content, 50*time.Millisecond)
	}

	// Add one with long TTL
	content := &OutputContent{Content: "Persistent"}
	cache.Set("job4:stdout", content, 10*time.Second)

	// Wait for short TTL entries to expire
	time.Sleep(60 * time.Millisecond)

	// Cleanup expired
	cleaned := cache.CleanupExpired()
	if cleaned != 3 {
		t.Errorf("Expected to clean 3 entries, cleaned %d", cleaned)
	}

	stats := cache.Stats()
	if stats.Entries != 1 {
		t.Errorf("Expected 1 entry remaining, got %d", stats.Entries)
	}

	// Long TTL entry should still exist
	_, found := cache.Get("job4:stdout")
	if !found {
		t.Error("Expected persistent entry to still exist")
	}
}

// TestOutputCache_Stats tests statistics tracking
func TestOutputCache_Stats(t *testing.T) {
	cache := NewOutputCache(5*time.Minute, 10)

	content := &OutputContent{Content: "Test"}
	cache.Set("job123:stdout", content, 0)

	// Cache hit
	_, found := cache.Get("job123:stdout")
	if !found {
		t.Fatal("Expected cache hit")
	}

	// Cache miss
	_, found = cache.Get("job999:stdout")
	if found {
		t.Fatal("Expected cache miss")
	}

	stats := cache.Stats()
	if stats.Hits != 1 {
		t.Errorf("Expected 1 hit, got %d", stats.Hits)
	}
	if stats.Misses != 1 {
		t.Errorf("Expected 1 miss, got %d", stats.Misses)
	}

	expectedHitRate := 50.0 // 1 hit out of 2 total = 50%
	if stats.HitRate != expectedHitRate {
		t.Errorf("Expected hit rate %.1f%%, got %.1f%%", expectedHitRate, stats.HitRate)
	}
}

// TestOutputCache_ConcurrentAccess tests thread safety
func TestOutputCache_ConcurrentAccess(t *testing.T) {
	cache := NewOutputCache(5*time.Minute, 100)

	done := make(chan bool)

	// Multiple goroutines writing
	for i := 0; i < 10; i++ {
		go func(id int) {
			for j := 0; j < 10; j++ {
				content := &OutputContent{Content: "Test"}
				cache.Set("job"+string(rune('0'+id))+":stdout", content, 0)
			}
			done <- true
		}(i)
	}

	// Multiple goroutines reading
	for i := 0; i < 10; i++ {
		go func(id int) {
			for j := 0; j < 10; j++ {
				_, _ = cache.Get("job" + string(rune('0'+id)) + ":stdout")
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 20; i++ {
		<-done
	}

	// No panics = success
	t.Log("Concurrent access completed without panics")
}

// TestGenerateCacheKey tests cache key generation
func TestGenerateCacheKey(t *testing.T) {
	tests := []struct {
		jobID      string
		outputType string
		expected   string
	}{
		{"12345", "stdout", "12345:stdout"},
		{"67890", "stderr", "67890:stderr"},
		{"job-abc", "stdout", "job-abc:stdout"},
	}

	for _, tt := range tests {
		result := GenerateCacheKey(tt.jobID, tt.outputType)
		if result != tt.expected {
			t.Errorf("GenerateCacheKey(%q, %q) = %q, want %q",
				tt.jobID, tt.outputType, result, tt.expected)
		}
	}
}

// TestCacheEntry_IsExpired tests expiration check
func TestCacheEntry_IsExpired(t *testing.T) {
	// Not expired
	entry := &CacheEntry{
		Timestamp: time.Now(),
		TTL:       1 * time.Hour,
	}

	if entry.IsExpired() {
		t.Error("Expected entry not to be expired")
	}

	// Expired
	entry = &CacheEntry{
		Timestamp: time.Now().Add(-2 * time.Hour),
		TTL:       1 * time.Hour,
	}

	if !entry.IsExpired() {
		t.Error("Expected entry to be expired")
	}
}
