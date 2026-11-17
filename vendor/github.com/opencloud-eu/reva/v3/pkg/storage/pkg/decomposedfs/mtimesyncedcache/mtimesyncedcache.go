package mtimesyncedcache

import (
	"sync"
	"time"
)

type Cache[K comparable, T any] struct {
	entries Map[K, *entry[T]]
}

type entry[T any] struct {
	mtime time.Time
	value T

	mu sync.Mutex
}

func New[K comparable, T any]() Cache[K, T] {
	return Cache[K, T]{
		entries: Map[K, *entry[T]]{},
	}
}

func (c *Cache[K, T]) Store(key K, mtime time.Time, value T) error {
	c.entries.Store(key, &entry[T]{
		mtime: mtime,
		value: value,
	})
	return nil
}

func (c *Cache[K, T]) Load(key K) (T, bool) {
	entry, ok := c.entries.Load(key)
	if !ok {
		var t T
		return t, false
	}
	return entry.value, true
}

func (c *Cache[K, T]) LoadOrStore(key K, mtime time.Time, f func() (T, error)) (T, error) {
	e, _ := c.entries.LoadOrStore(key, &entry[T]{})

	e.mu.Lock()
	defer e.mu.Unlock()
	if mtime.After(e.mtime) {
		e.mtime = mtime
		v, err := f()
		if err != nil {
			var t T
			return t, err
		}
		e.value = v
		c.entries.Store(key, e)
	}

	return e.value, nil
}
