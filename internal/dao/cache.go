package dao

import (
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// DAOCache provides a generic TTL cache for DAO operations
type DAOCache struct {
	mu         sync.RWMutex
	entries    map[string]*daoCacheEntry
	defaultTTL time.Duration
	maxSize    int

	// Statistics
	hits   atomic.Int64
	misses atomic.Int64
}

type daoCacheEntry struct {
	data      any
	timestamp time.Time
	ttl       time.Duration
}

// NewDAOCache creates a new DAO cache with the given default TTL
func NewDAOCache(defaultTTL time.Duration, maxSize int) *DAOCache {
	if defaultTTL <= 0 {
		defaultTTL = 10 * time.Second
	}
	if maxSize <= 0 {
		maxSize = 50
	}
	return &DAOCache{
		entries:    make(map[string]*daoCacheEntry),
		defaultTTL: defaultTTL,
		maxSize:    maxSize,
	}
}

// Get retrieves an item from the cache. Returns the item and true if found and not expired.
// Expired entries are lazily deleted.
func (c *DAOCache) Get(key string) (any, bool) {
	c.mu.RLock()
	entry, exists := c.entries[key]
	if !exists {
		c.mu.RUnlock()
		c.misses.Add(1)
		return nil, false
	}

	if time.Since(entry.timestamp) > entry.ttl {
		c.mu.RUnlock()
		// Upgrade to write lock to delete expired entry.
		// Note: between RUnlock and Lock, another goroutine could refresh this key.
		// The double-check below handles that; worst case is one extra fetch.
		c.mu.Lock()
		if e, ok := c.entries[key]; ok && time.Since(e.timestamp) > e.ttl {
			delete(c.entries, key)
		}
		c.mu.Unlock()
		c.misses.Add(1)
		return nil, false
	}

	data := entry.data
	c.mu.RUnlock()
	c.hits.Add(1)
	return data, true
}

// Set stores an item in the cache. If ttl is 0, the default TTL is used.
func (c *DAOCache) Set(key string, data any, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if ttl <= 0 {
		ttl = c.defaultTTL
	}

	// Evict oldest if at capacity
	if len(c.entries) >= c.maxSize {
		c.evictOldest()
	}

	c.entries[key] = &daoCacheEntry{
		data:      data,
		timestamp: time.Now(),
		ttl:       ttl,
	}
}

// Invalidate removes specific keys from the cache
func (c *DAOCache) Invalidate(keys ...string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	for _, key := range keys {
		delete(c.entries, key)
	}
}

// InvalidatePrefix removes all entries whose keys start with the given prefix
func (c *DAOCache) InvalidatePrefix(prefix string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	for key := range c.entries {
		if strings.HasPrefix(key, prefix) {
			delete(c.entries, key)
		}
	}
}

// InvalidateAll removes all entries from the cache
func (c *DAOCache) InvalidateAll() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.entries = make(map[string]*daoCacheEntry)
}

// evictOldest removes the oldest entry (caller must hold the write lock).
// O(n) scan — fine for small maxSize, consider a heap if maxSize grows large.
func (c *DAOCache) evictOldest() {
	var oldestKey string
	var oldestTime time.Time

	for key, entry := range c.entries {
		if oldestKey == "" || entry.timestamp.Before(oldestTime) {
			oldestKey = key
			oldestTime = entry.timestamp
		}
	}

	if oldestKey != "" {
		delete(c.entries, oldestKey)
	}
}
