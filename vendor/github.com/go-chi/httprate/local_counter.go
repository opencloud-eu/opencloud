package httprate

import (
	"sync"
	"time"

	"github.com/zeebo/xxh3"
)

// NewLocalLimitCounter creates an instance of localCounter,
// which is an in-memory implementation of http.LimitCounter.
//
// All methods are guaranteed to always return nil error.
func NewLocalLimitCounter(windowLength time.Duration) *localCounter {
	return &localCounter{
		windowLength:     windowLength,
		latestWindow:     time.Now().UTC(),
		latestCounters:   make(map[uint64]int),
		previousCounters: make(map[uint64]int),
	}
}

var _ LimitCounter = (*localCounter)(nil)

type localCounter struct {
	windowLength     time.Duration
	latestWindow     time.Time
	latestCounters   map[uint64]int
	previousCounters map[uint64]int
	mu               sync.RWMutex
}

func (c *localCounter) IncrementBy(key string, currentWindow time.Time, amount int) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.evict(currentWindow)

	hkey := limitCounterKey(key)

	count, _ := c.latestCounters[hkey]
	c.latestCounters[hkey] = count + amount

	return nil
}

func (c *localCounter) Get(key string, currentWindow, previousWindow time.Time) (int, int, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.latestWindow == currentWindow {
		curr, _ := c.latestCounters[limitCounterKey(key)]
		prev, _ := c.previousCounters[limitCounterKey(key)]
		return curr, prev, nil
	}

	if c.latestWindow == previousWindow {
		prev, _ := c.latestCounters[limitCounterKey(key)]
		return 0, prev, nil
	}

	return 0, 0, nil
}

func (c *localCounter) Config(requestLimit int, windowLength time.Duration) {
	c.windowLength = windowLength
	c.latestWindow = time.Now().UTC().Truncate(windowLength)
}

func (c *localCounter) Increment(key string, currentWindow time.Time) error {
	return c.IncrementBy(key, currentWindow, 1)
}

func (c *localCounter) evict(currentWindow time.Time) {
	if c.latestWindow == currentWindow {
		return
	}

	previousWindow := currentWindow.Add(-c.windowLength)
	if c.latestWindow == previousWindow {
		c.latestWindow = currentWindow
		// Shift the windows without map re-allocation.
		clear(c.previousCounters)
		c.latestCounters, c.previousCounters = c.previousCounters, c.latestCounters
		return
	}

	c.latestWindow = currentWindow

	clear(c.previousCounters)
	clear(c.latestCounters)
}

func limitCounterKey(key string) uint64 {
	return xxh3.HashString(key)
}
