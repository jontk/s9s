package dao

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewDAOCache(t *testing.T) {
	t.Run("uses defaults for zero values", func(t *testing.T) {
		c := NewDAOCache(0, 0)
		assert.NotNil(t, c)
		assert.Equal(t, 10*time.Second, c.defaultTTL)
		assert.Equal(t, 50, c.maxSize)
	})

	t.Run("uses provided values", func(t *testing.T) {
		c := NewDAOCache(5*time.Second, 10)
		assert.Equal(t, 5*time.Second, c.defaultTTL)
		assert.Equal(t, 10, c.maxSize)
	})
}

func TestDAOCache_GetSet(t *testing.T) {
	t.Run("miss on empty cache", func(t *testing.T) {
		c := NewDAOCache(time.Minute, 10)
		val, ok := c.Get("missing")
		assert.False(t, ok)
		assert.Nil(t, val)
	})

	t.Run("hit after set", func(t *testing.T) {
		c := NewDAOCache(time.Minute, 10)
		c.Set("key", "value", 0)
		val, ok := c.Get("key")
		assert.True(t, ok)
		assert.Equal(t, "value", val)
	})

	t.Run("expired entry returns miss", func(t *testing.T) {
		c := NewDAOCache(time.Minute, 10)
		c.Set("key", "value", 1*time.Millisecond)
		time.Sleep(5 * time.Millisecond)
		val, ok := c.Get("key")
		assert.False(t, ok)
		assert.Nil(t, val)
	})

	t.Run("expired entry is lazily deleted", func(t *testing.T) {
		c := NewDAOCache(time.Minute, 10)
		c.Set("key", "value", 1*time.Millisecond)
		time.Sleep(5 * time.Millisecond)
		_, _ = c.Get("key")

		c.mu.RLock()
		_, exists := c.entries["key"]
		c.mu.RUnlock()
		assert.False(t, exists)
	})

	t.Run("set with zero ttl uses default", func(t *testing.T) {
		c := NewDAOCache(time.Minute, 10)
		c.Set("key", "value", 0)

		c.mu.RLock()
		entry := c.entries["key"]
		c.mu.RUnlock()
		assert.Equal(t, time.Minute, entry.ttl)
	})
}

func TestDAOCache_Eviction(t *testing.T) {
	c := NewDAOCache(time.Minute, 3)
	c.Set("a", 1, 0)
	time.Sleep(time.Millisecond)
	c.Set("b", 2, 0)
	time.Sleep(time.Millisecond)
	c.Set("c", 3, 0)
	time.Sleep(time.Millisecond)

	// Adding a 4th entry should evict the oldest ("a")
	c.Set("d", 4, 0)

	_, ok := c.Get("a")
	assert.False(t, ok, "oldest entry should have been evicted")

	val, ok := c.Get("d")
	assert.True(t, ok)
	assert.Equal(t, 4, val)
}

func TestDAOCache_Invalidate(t *testing.T) {
	t.Run("invalidate specific keys", func(t *testing.T) {
		c := NewDAOCache(time.Minute, 10)
		c.Set("a", 1, 0)
		c.Set("b", 2, 0)
		c.Set("c", 3, 0)

		c.Invalidate("a", "c")

		_, ok := c.Get("a")
		assert.False(t, ok)
		_, ok = c.Get("b")
		assert.True(t, ok)
		_, ok = c.Get("c")
		assert.False(t, ok)
	})

	t.Run("invalidate by prefix", func(t *testing.T) {
		c := NewDAOCache(time.Minute, 10)
		c.Set("jobs:list:1", 1, 0)
		c.Set("jobs:list:2", 2, 0)
		c.Set("nodes:list:1", 3, 0)

		c.InvalidatePrefix("jobs:")

		_, ok := c.Get("jobs:list:1")
		assert.False(t, ok)
		_, ok = c.Get("jobs:list:2")
		assert.False(t, ok)
		_, ok = c.Get("nodes:list:1")
		assert.True(t, ok)
	})

	t.Run("invalidate all", func(t *testing.T) {
		c := NewDAOCache(time.Minute, 10)
		c.Set("a", 1, 0)
		c.Set("b", 2, 0)

		c.InvalidateAll()

		_, ok := c.Get("a")
		assert.False(t, ok)
		_, ok = c.Get("b")
		assert.False(t, ok)
	})
}

func TestDAOCache_Statistics(t *testing.T) {
	c := NewDAOCache(time.Minute, 10)
	c.Set("key", "value", 0)

	_, _ = c.Get("key")     // hit
	_, _ = c.Get("key")     // hit
	_, _ = c.Get("missing") // miss

	assert.Equal(t, int64(2), c.hits.Load())
	assert.Equal(t, int64(1), c.misses.Load())
}

func TestDAOCache_ConcurrentAccess(t *testing.T) {
	c := NewDAOCache(time.Minute, 100)
	var wg sync.WaitGroup

	// Concurrent writers
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				key := string(rune('a'+i)) + string(rune('0'+j%10))
				c.Set(key, j, 0)
			}
		}(i)
	}

	// Concurrent readers
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				key := string(rune('a'+i)) + string(rune('0'+j%10))
				c.Get(key)
			}
		}(i)
	}

	// Concurrent invalidators
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 50; i++ {
			c.InvalidatePrefix("a")
		}
	}()

	wg.Wait()
}
