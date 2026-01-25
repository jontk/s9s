package prometheus

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
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

	// Optimized key generation
	keyGen *CacheKeyGenerator

	// Metrics
	hits      int64
	misses    int64
	evictions int64

	// Lifecycle management
	stopChan chan struct{}
	stopped  bool
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
		entries:  make(map[string]*CacheEntry),
		ttl:      defaultTTL,
		maxSize:  maxSize,
		keyGen:   NewCacheKeyGenerator(),
		stopChan: make(chan struct{}),
		stopped:  false,
	}

	// Start cleanup goroutine
	go cache.cleanupLoop()

	return cache
}

// Get retrieves a cached query result using raw query string (legacy method)
func (mc *MetricCache) Get(query string) (*QueryResult, bool) {
	return mc.GetWithKey(query, query)
}

// GetInstantQuery retrieves a cached instant query result with optimized key
func (mc *MetricCache) GetInstantQuery(query string, queryTime *time.Time) (*QueryResult, bool) {
	key := mc.keyGen.GenerateInstantQueryKey(query, queryTime)
	return mc.GetWithKey(key, query)
}

// GetRangeQuery retrieves a cached range query result with optimized key
func (mc *MetricCache) GetRangeQuery(query string, start, end time.Time, step time.Duration) (*QueryResult, bool) {
	key := mc.keyGen.GenerateRangeQueryKey(query, start, end, step)
	return mc.GetWithKey(key, query)
}

// GetSeries retrieves a cached series query result with optimized key
func (mc *MetricCache) GetSeries(matches []string, start, end time.Time) ([]map[string]string, bool) {
	key := mc.keyGen.GenerateSeriesQueryKey(matches, start, end)
	if _, exists := mc.GetWithKey(key, key); exists {
		// For series queries, we need to extract the series data from the result
		// This is a simplified version - in practice you'd store series data directly
		return nil, false // TODO: Implement series result extraction
	}
	return nil, false
}

// GetWithKey retrieves a cached query result using a specific cache key
func (mc *MetricCache) GetWithKey(cacheKey, _ string) (*QueryResult, bool) {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	entry, exists := mc.entries[cacheKey]
	if !exists {
		atomic.AddInt64(&mc.misses, 1)
		return nil, false
	}

	if entry.IsExpired() {
		atomic.AddInt64(&mc.misses, 1)
		return nil, false
	}

	atomic.AddInt64(&mc.hits, 1)
	return entry.Result, true
}

// Set stores a query result in the cache using raw query string (legacy method)
func (mc *MetricCache) Set(query string, result *QueryResult, ttl time.Duration) {
	mc.SetWithKey(query, query, result, ttl)
}

// SetInstantQuery stores an instant query result with optimized key
func (mc *MetricCache) SetInstantQuery(query string, queryTime *time.Time, result *QueryResult, ttl time.Duration) {
	key := mc.keyGen.GenerateInstantQueryKey(query, queryTime)
	mc.SetWithKey(key, query, result, ttl)
}

// SetRangeQuery stores a range query result with optimized key
func (mc *MetricCache) SetRangeQuery(query string, start, end time.Time, step time.Duration, result *QueryResult, ttl time.Duration) {
	key := mc.keyGen.GenerateRangeQueryKey(query, start, end, step)
	mc.SetWithKey(key, query, result, ttl)
}

// SetWithKey stores a query result in the cache using a specific cache key
func (mc *MetricCache) SetWithKey(cacheKey, originalQuery string, result *QueryResult, ttl time.Duration) {
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

	mc.entries[cacheKey] = &CacheEntry{
		Query:     originalQuery, // Store original query for debugging
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
	atomic.StoreInt64(&mc.hits, 0)
	atomic.StoreInt64(&mc.misses, 0)
	atomic.StoreInt64(&mc.evictions, 0)
}

// Stats returns cache statistics
func (mc *MetricCache) Stats() CacheStats {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	hits := atomic.LoadInt64(&mc.hits)
	misses := atomic.LoadInt64(&mc.misses)
	evictions := atomic.LoadInt64(&mc.evictions)

	var hitRate float64
	total := hits + misses
	if total > 0 {
		hitRate = float64(hits) / float64(total) * 100
	}

	return CacheStats{
		Entries:   len(mc.entries),
		Hits:      hits,
		Misses:    misses,
		Evictions: evictions,
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
		atomic.AddInt64(&mc.evictions, 1)
	}
}

// cleanupLoop periodically removes expired entries
func (mc *MetricCache) cleanupLoop() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-mc.stopChan:
			return
		case <-ticker.C:
			mc.cleanup()
		}
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

// Stop gracefully shuts down the cache cleanup goroutine
func (mc *MetricCache) Stop() {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	if !mc.stopped {
		mc.stopped = true
		close(mc.stopChan)
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
	client PrometheusClientInterface
	cache  *MetricCache
}

// NewCachedClient creates a new cached Prometheus client
func NewCachedClient(client *Client, cacheTTL time.Duration, cacheSize int) *CachedClient {
	return &CachedClient{
		client: client,
		cache:  NewMetricCache(cacheTTL, cacheSize),
	}
}

// NewCachedClientWithInterface creates a new cached client using the interface
func NewCachedClientWithInterface(client PrometheusClientInterface, cacheTTL time.Duration, cacheSize int) *CachedClient {
	return &CachedClient{
		client: client,
		cache:  NewMetricCache(cacheTTL, cacheSize),
	}
}

// TestConnection tests the connection to Prometheus
func (cc *CachedClient) TestConnection(ctx context.Context) error {
	return cc.client.TestConnection(ctx)
}

// Stop gracefully shuts down the cached client
func (cc *CachedClient) Stop() {
	if cc == nil {
		return
	}
	if cc.cache != nil {
		cc.cache.Stop()
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

// BatchQuery executes multiple queries with caching
func (cc *CachedClient) BatchQuery(ctx context.Context, queries map[string]string, ts time.Time) (map[string]*QueryResult, error) {
	results := make(map[string]*QueryResult)
	uncachedQueries := make(map[string]string)

	// Check cache for each query
	for name, query := range queries {
		cacheKey := fmt.Sprintf("%s@%d", query, ts.Unix())
		if result, found := cc.cache.Get(cacheKey); found {
			results[name] = result
		} else {
			uncachedQueries[name] = query
		}
	}

	// Execute uncached queries if any
	if len(uncachedQueries) > 0 {
		uncachedResults, err := cc.client.BatchQuery(ctx, uncachedQueries, ts)
		if err != nil {
			// Return partial results if we have cached ones
			if len(results) > 0 {
				return results, nil
			}
			return nil, err
		}

		// Cache and merge results
		for name, result := range uncachedResults {
			query := uncachedQueries[name]
			cacheKey := fmt.Sprintf("%s@%d", query, ts.Unix())
			cc.cache.Set(cacheKey, result, 0) // Use default TTL
			results[name] = result
		}
	}

	return results, nil
}

// Series executes a series query with caching
func (cc *CachedClient) Series(ctx context.Context, matches []string, start, end time.Time) ([]map[string]string, error) {
	// For now, delegate to underlying client without caching
	// Series queries are typically not cached due to their variability
	return cc.client.Series(ctx, matches, start, end)
}

// Labels executes a labels query with caching
func (cc *CachedClient) Labels(ctx context.Context) ([]string, error) {
	// Generate cache key for labels
	cacheKey := "labels_query"

	// Check cache
	if result, found := cc.cache.Get(cacheKey); found {
		// Extract labels from cached result - this is a simplified approach
		// In practice, you'd want to cache the actual labels slice
		_ = result
	}

	// Execute labels query
	labels, err := cc.client.Labels(ctx)
	if err != nil {
		return nil, err
	}

	// Note: We're not caching labels for now as the QueryResult structure
	// is designed for metric queries, not label queries
	// TODO: Implement proper labels caching

	return labels, nil
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
