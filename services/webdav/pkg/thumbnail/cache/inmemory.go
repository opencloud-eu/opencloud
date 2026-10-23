package cache

import "sync"

// InMemoryCacheMaxBytes is how much the memory backend keeps before it drops the
// oldest entries. The backend is a convenience for a single process; file and s3
// are the ones that survive a restart.
const InMemoryCacheMaxBytes = 128 << 20

// InMemoryCache is a simple in-memory thumbnail cache, bounded by
// InMemoryCacheMaxBytes and evicting in insertion order.
type InMemoryCache struct {
	mu      sync.RWMutex
	data    map[string][]byte
	order   []string
	size    int
	maxSize int
}

func NewInMemoryCache() *InMemoryCache {
	return &InMemoryCache{
		data:    make(map[string][]byte),
		maxSize: InMemoryCacheMaxBytes,
	}
}

func (c *InMemoryCache) Get(key string) ([]byte, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	data, ok := c.data[key]
	if !ok {
		return nil, ErrCacheMiss
	}

	result := make([]byte, len(data))
	copy(result, data)
	return result, nil
}

func (c *InMemoryCache) Put(key string, data []byte) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	cp := make([]byte, len(data))
	copy(cp, data)

	if old, ok := c.data[key]; ok {
		c.size -= len(old)
	} else {
		c.order = append(c.order, key)
	}
	c.data[key] = cp
	c.size += len(cp)

	// the entry just written stays, even when it is larger than the budget
	for c.size > c.maxSize && len(c.order) > 1 {
		oldest := c.order[0]
		c.order = c.order[1:]
		c.size -= len(c.data[oldest])
		delete(c.data, oldest)
	}

	return nil
}
