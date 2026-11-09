package cache

import (
	"sync"

	"github.com/opencloud-eu/opencloud/services/webdav/pkg/thumbnail"
)

// InMemoryCacheMaxBytes is how much the memory backend keeps before it drops the
// oldest entries. The backend is a convenience for a single process; file and s3
// are the ones that survive a restart.
const InMemoryCacheMaxBytes = 128 << 20

// InMemoryCache is a simple in-memory thumbnail cache, bounded by
// InMemoryCacheMaxBytes and evicting in insertion order.
type InMemoryCache struct {
	mu      sync.RWMutex
	data    map[string]thumbnail.Image
	order   []string
	size    int
	maxSize int
}

func NewInMemoryCache() *InMemoryCache {
	return &InMemoryCache{
		data:    make(map[string]thumbnail.Image),
		maxSize: InMemoryCacheMaxBytes,
	}
}

func (c *InMemoryCache) Get(key string) (thumbnail.Image, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	img, ok := c.data[key]
	if !ok {
		return thumbnail.Image{}, ErrCacheMiss
	}

	data := make([]byte, len(img.Data))
	copy(data, img.Data)
	return thumbnail.Image{Data: data, ContentType: img.ContentType}, nil
}

func (c *InMemoryCache) Put(key string, img thumbnail.Image) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	data := make([]byte, len(img.Data))
	copy(data, img.Data)

	if old, ok := c.data[key]; ok {
		c.size -= len(old.Data)
	} else {
		c.order = append(c.order, key)
	}
	c.data[key] = thumbnail.Image{Data: data, ContentType: img.ContentType}
	c.size += len(data)

	// the entry just written stays, even when it is larger than the budget
	for c.size > c.maxSize && len(c.order) > 1 {
		oldest := c.order[0]
		c.order = c.order[1:]
		c.size -= len(c.data[oldest].Data)
		delete(c.data, oldest)
	}

	return nil
}
