package cache

import "sync"

// InMemoryCache is a simple in-memory thumbnail cache using sync.RWMutex.
type InMemoryCache struct {
	mu   sync.RWMutex
	data map[string][]byte
}

func NewInMemoryCache() *InMemoryCache {
	return &InMemoryCache{
		data: make(map[string][]byte),
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
	c.data[key] = cp
	return nil
}
