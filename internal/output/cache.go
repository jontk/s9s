package output

import (
	"sync"
	"sync/atomic"
	"time"
)

// CacheEntry represents a cached output result
type CacheEntry struct {
	Content   *OutputContent
	Timestamp time.Time
	TTL       time.Duration
}

// IsExpired checks if the cache entry has expired
func (ce *CacheEntry) IsExpired() bool {
	return time.Since(ce.Timestamp) > ce.TTL
}

// OutputCache provides LRU caching for job output
type OutputCache struct {
	mu       sync.RWMutex
	entries  map[string]*CacheEntry
	maxSize  int
	defaultTTL time.Duration

	// Statistics (using atomic for thread-safe access)
	hits      atomic.Int64
	misses    atomic.Int64
	evictions atomic.Int64
}

// NewOutputCache creates a new output cache
func NewOutputCache(defaultTTL time.Duration, maxSize int) *OutputCache {
	if maxSize <= 0 {
		maxSize = 100 // Default cache size
	}
	if defaultTTL <= 0 {
		defaultTTL = 5 * time.Minute // Default TTL
	}

	return &OutputCache{
		entries:    make(map[string]*CacheEntry),
		maxSize:    maxSize,
		defaultTTL: defaultTTL,
	}
}

// Get retrieves an output from the cache
func (c *OutputCache) Get(key string) (*OutputContent, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, exists := c.entries[key]
	if !exists {
		c.misses.Add(1)
		return nil, false
	}

	// Check if expired
	if entry.IsExpired() {
		c.misses.Add(1)
		return nil, false
	}

	c.hits.Add(1)
	return entry.Content, true
}

// Set stores an output in the cache
func (c *OutputCache) Set(key string, content *OutputContent, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Use default TTL if not specified
	if ttl <= 0 {
		ttl = c.defaultTTL
	}

	// Check if we need to evict
	if len(c.entries) >= c.maxSize {
		c.evictOldest()
	}

	c.entries[key] = &CacheEntry{
		Content:   content,
		Timestamp: time.Now(),
		TTL:       ttl,
	}
}

// Delete removes an entry from the cache
func (c *OutputCache) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.entries, key)
}

// Clear removes all entries from the cache
func (c *OutputCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.entries = make(map[string]*CacheEntry)
	c.hits.Store(0)
	c.misses.Store(0)
	c.evictions.Store(0)
}

// CleanupExpired removes expired entries
func (c *OutputCache) CleanupExpired() int {
	c.mu.Lock()
	defer c.mu.Unlock()

	keysToDelete := []string{}

	for key, entry := range c.entries {
		if entry.IsExpired() {
			keysToDelete = append(keysToDelete, key)
		}
	}

	for _, key := range keysToDelete {
		delete(c.entries, key)
	}

	return len(keysToDelete)
}

// evictOldest removes the oldest cache entry (caller must hold lock)
func (c *OutputCache) evictOldest() {
	var oldestKey string
	var oldestTime time.Time

	for key, entry := range c.entries {
		if oldestKey == "" || entry.Timestamp.Before(oldestTime) {
			oldestKey = key
			oldestTime = entry.Timestamp
		}
	}

	if oldestKey != "" {
		delete(c.entries, oldestKey)
		c.evictions.Add(1)
	}
}

// Stats returns cache statistics
func (c *OutputCache) Stats() CacheStats {
	c.mu.RLock()
	defer c.mu.RUnlock()

	hits := c.hits.Load()
	misses := c.misses.Load()
	evictions := c.evictions.Load()

	var hitRate float64
	total := hits + misses
	if total > 0 {
		hitRate = float64(hits) / float64(total) * 100
	}

	return CacheStats{
		Entries:   len(c.entries),
		Hits:      hits,
		Misses:    misses,
		Evictions: evictions,
		HitRate:   hitRate,
		MaxSize:   c.maxSize,
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

// GenerateCacheKey generates a cache key for job output
func GenerateCacheKey(jobID, outputType string) string {
	return jobID + ":" + outputType
}
