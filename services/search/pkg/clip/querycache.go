package clip

import (
	"container/list"
	"context"
	"strings"
	"sync"

	"golang.org/x/sync/singleflight"
)

// QueryCache caches text embeddings behind an LRU: users type the same terms
// repeatedly, and every miss is a network round-trip to the inference service.
// A singleflight collapses the concurrent misses the per-space search fan-out
// produces for one and the same query.
type QueryCache struct {
	next interface {
		VectorizeText(ctx context.Context, text string) ([]float32, error)
	}
	group singleflight.Group

	mu      sync.Mutex
	maxSize int
	entries map[string]*list.Element
	order   *list.List // front = most recently used
}

type cacheEntry struct {
	key    string
	vector []float32
}

// NewQueryCache wraps a text vectorizer with an LRU of the given size.
func NewQueryCache(next *Client, size int) *QueryCache {
	if size <= 0 {
		size = 512
	}
	return &QueryCache{
		next:    next,
		maxSize: size,
		entries: map[string]*list.Element{},
		order:   list.New(),
	}
}

// VectorizeText returns the cached embedding for text or fetches and caches it.
func (c *QueryCache) VectorizeText(ctx context.Context, text string) ([]float32, error) {
	key := strings.Join(strings.Fields(strings.ToLower(text)), " ")

	c.mu.Lock()
	if el, ok := c.entries[key]; ok {
		c.order.MoveToFront(el)
		vector := el.Value.(*cacheEntry).vector
		c.mu.Unlock()
		return vector, nil
	}
	c.mu.Unlock()

	v, err, _ := c.group.Do(key, func() (any, error) {
		vector, err := c.next.VectorizeText(ctx, text)
		if err != nil {
			return nil, err
		}
		c.mu.Lock()
		defer c.mu.Unlock()
		if el, ok := c.entries[key]; ok {
			c.order.MoveToFront(el)
			return el.Value.(*cacheEntry).vector, nil
		}
		c.entries[key] = c.order.PushFront(&cacheEntry{key: key, vector: vector})
		if c.order.Len() > c.maxSize {
			oldest := c.order.Back()
			c.order.Remove(oldest)
			delete(c.entries, oldest.Value.(*cacheEntry).key)
		}
		return vector, nil
	})
	if err != nil {
		return nil, err
	}
	return v.([]float32), nil
}
