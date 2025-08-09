package prometheus

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// CacheEntry represents a cached metric value
type CacheEntry struct {
	Query     string
	Result    *QueryResult
	Timestamp time.Time
	TTL       time.Duration
}

// IsExpired checks if the cache entry has expired
func (ce *CacheEntry) IsExpired() bool {
	return time.Since(ce.Timestamp) > ce.TTL
}

// MetricCache provides caching for Prometheus query results
type MetricCache struct {
	mu      sync.RWMutex
	entries map[string]*CacheEntry
	ttl     time.Duration
	maxSize int
	
	// Metrics
	hits   int64
	misses int64
	evictions int64
}

// NewMetricCache creates a new metric cache
func NewMetricCache(defaultTTL time.Duration, maxSize int) *MetricCache {
	if maxSize <= 0 {
		maxSize = 1000
	}
	if defaultTTL <= 0 {
		defaultTTL = 30 * time.Second
	}
	
	cache := &MetricCache{
		entries: make(map[string]*CacheEntry),
		ttl:     defaultTTL,
		maxSize: maxSize,
	}
	
	// Start cleanup goroutine
	go cache.cleanupLoop()
	
	return cache
}

// Get retrieves a cached query result
func (mc *MetricCache) Get(query string) (*QueryResult, bool) {
	mc.mu.RLock()
	defer mc.mu.RUnlock()
	
	entry, exists := mc.entries[query]
	if !exists {
		mc.misses++
		return nil, false
	}
	
	if entry.IsExpired() {
		mc.misses++
		return nil, false
	}
	
	mc.hits++
	return entry.Result, true
}

// Set stores a query result in the cache
func (mc *MetricCache) Set(query string, result *QueryResult, ttl time.Duration) {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	
	// Use default TTL if not specified
	if ttl <= 0 {
		ttl = mc.ttl
	}
	
	// Check cache size limit
	if len(mc.entries) >= mc.maxSize {
		// Evict oldest entry
		mc.evictOldest()
	}
	
	mc.entries[query] = &CacheEntry{
		Query:     query,
		Result:    result,
		Timestamp: time.Now(),
		TTL:       ttl,
	}
}

// Delete removes a specific query from the cache
func (mc *MetricCache) Delete(query string) {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	
	delete(mc.entries, query)
}

// Clear removes all entries from the cache
func (mc *MetricCache) Clear() {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	
	mc.entries = make(map[string]*CacheEntry)
	mc.hits = 0
	mc.misses = 0
	mc.evictions = 0
}

// Stats returns cache statistics
func (mc *MetricCache) Stats() CacheStats {
	mc.mu.RLock()
	defer mc.mu.RUnlock()
	
	var hitRate float64
	total := mc.hits + mc.misses
	if total > 0 {
		hitRate = float64(mc.hits) / float64(total) * 100
	}
	
	return CacheStats{
		Entries:   len(mc.entries),
		Hits:      mc.hits,
		Misses:    mc.misses,
		Evictions: mc.evictions,
		HitRate:   hitRate,
		MaxSize:   mc.maxSize,
	}
}

// evictOldest removes the oldest cache entry
func (mc *MetricCache) evictOldest() {
	var oldestKey string
	var oldestTime time.Time
	
	for key, entry := range mc.entries {
		if oldestKey == "" || entry.Timestamp.Before(oldestTime) {
			oldestKey = key
			oldestTime = entry.Timestamp
		}
	}
	
	if oldestKey != "" {
		delete(mc.entries, oldestKey)
		mc.evictions++
	}
}

// cleanupLoop periodically removes expired entries
func (mc *MetricCache) cleanupLoop() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()
	
	for range ticker.C {
		mc.cleanup()
	}
}

// cleanup removes expired entries
func (mc *MetricCache) cleanup() {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	
	keysToDelete := []string{}
	
	for key, entry := range mc.entries {
		if entry.IsExpired() {
			keysToDelete = append(keysToDelete, key)
		}
	}
	
	for _, key := range keysToDelete {
		delete(mc.entries, key)
	}
}

// CacheStats contains cache statistics
type CacheStats struct {
	Entries   int
	Hits      int64
	Misses    int64
	Evictions int64
	HitRate   float64
	MaxSize   int
}

// CachedClient wraps a Prometheus client with caching
type CachedClient struct {
	client *Client
	cache  *MetricCache
}

// NewCachedClient creates a new cached Prometheus client
func NewCachedClient(client *Client, cacheTTL time.Duration, cacheSize int) *CachedClient {
	return &CachedClient{
		client: client,
		cache:  NewMetricCache(cacheTTL, cacheSize),
	}
}

// Query executes a query with caching
func (cc *CachedClient) Query(ctx context.Context, query string, time time.Time) (*QueryResult, error) {
	// Generate cache key
	cacheKey := fmt.Sprintf("%s@%d", query, time.Unix())
	
	// Check cache
	if result, found := cc.cache.Get(cacheKey); found {
		return result, nil
	}
	
	// Execute query
	result, err := cc.client.Query(ctx, query, time)
	if err != nil {
		return nil, err
	}
	
	// Cache result
	cc.cache.Set(cacheKey, result, 0) // Use default TTL
	
	return result, nil
}

// QueryRange executes a range query with caching
func (cc *CachedClient) QueryRange(ctx context.Context, query string, start, end time.Time, step time.Duration) (*QueryResult, error) {
	// Generate cache key
	cacheKey := fmt.Sprintf("%s@%d-%d-%s", query, start.Unix(), end.Unix(), step)
	
	// Check cache
	if result, found := cc.cache.Get(cacheKey); found {
		return result, nil
	}
	
	// Execute query
	result, err := cc.client.QueryRange(ctx, query, start, end, step)
	if err != nil {
		return nil, err
	}
	
	// Cache result with shorter TTL for range queries
	cc.cache.Set(cacheKey, result, 5*time.Minute)
	
	return result, nil
}

// ClearCache clears the cache
func (cc *CachedClient) ClearCache() {
	cc.cache.Clear()
}

// CacheStats returns cache statistics
func (cc *CachedClient) CacheStats() CacheStats {
	return cc.cache.Stats()
}

// InvalidatePattern removes cache entries matching a pattern
func (cc *CachedClient) InvalidatePattern(pattern string) {
	cc.cache.mu.Lock()
	defer cc.cache.mu.Unlock()
	
	keysToDelete := []string{}
	
	for key := range cc.cache.entries {
		// Simple pattern matching - could be enhanced
		if contains(key, pattern) {
			keysToDelete = append(keysToDelete, key)
		}
	}
	
	for _, key := range keysToDelete {
		delete(cc.cache.entries, key)
	}
}

// contains checks if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && s[:len(substr)] == substr
}