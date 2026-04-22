package sync

import (
	"sync"
	"sync/atomic"
	"time"
)

// Cache is a barebones cache implementation.
type Cache struct {
	// capacity and length have to be the first words
	// in order to be 64-aligned on 32-bit architectures.
	capacity, length uint64 // access atomically
	entries          sync.Map
	pool             sync.Pool
}

// CacheEntry represents an entry on the cache. You can type assert on V.
type CacheEntry struct {
	V          any
	expiration time.Time
}

// NewCache returns a new instance of Cache.
func NewCache(capacity int) Cache {
	return Cache{
		capacity: uint64(capacity),
		pool: sync.Pool{New: func() any {
			return new(CacheEntry)
		}},
	}
}

// Load loads an entry by given key
func (c *Cache) Load(key string) *CacheEntry {
	if mapEntry, ok := c.entries.Load(key); ok {
		entry := mapEntry.(*CacheEntry)
		if c.expired(entry) {
			c.entries.Delete(key)
			return nil
		}
		return entry
	}
	return nil
}

// Store adds an entry for given key and value
func (c *Cache) Store(key string, val any, expiration time.Time) {
	if c.length > c.capacity {
		c.evict()
	}

	poolEntry := c.pool.Get() //nolint: ifshort
	if mapEntry, loaded := c.entries.LoadOrStore(key, poolEntry); loaded {
		entry := mapEntry.(*CacheEntry)
		entry.V = val
		entry.expiration = expiration

		c.pool.Put(poolEntry)
	} else {
		entry := poolEntry.(*CacheEntry)
		entry.V = val
		entry.expiration = expiration

		atomic.AddUint64(&c.length, 1)
	}
}

// Delete removes an entry by given key
func (c *Cache) Delete(key string) bool {
	_, loaded := c.entries.LoadAndDelete(key)

	if loaded {
		atomic.AddUint64(&c.length, ^uint64(0))
	}

	return loaded
}

// evict frees memory from the cache by removing entries that exceeded the cache TTL.
func (c *Cache) evict() {
	c.entries.Range(func(key, mapEntry any) bool {
		entry := mapEntry.(*CacheEntry)
		if c.expired(entry) {
			c.Delete(key.(string))
		}
		return true
	})
}

// expired checks if an entry is expired
func (c *Cache) expired(e *CacheEntry) bool {
	return e.expiration.Before(time.Now())
}
