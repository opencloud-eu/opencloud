package stringutil

import (
	"math/rand"
	"sync"
	"time"
)

var globalRandSource rand.Source

func init() {
	globalRandSource = NewLockedSource(time.Now().UTC().UnixNano())
}

// NewLockedSource creates a source of randomness using the given seed.
func NewLockedSource(seed int64) rand.Source64 {
	return &lockedSource{
		s: rand.NewSource(seed).(rand.Source64),
	}
}

type lockedSource struct {
	lock sync.Mutex
	s    rand.Source64
}

func (x *lockedSource) Int63() int64 {
	x.lock.Lock()
	defer x.lock.Unlock()
	return x.s.Int63()
}

func (x *lockedSource) Uint64() uint64 {
	x.lock.Lock()
	defer x.lock.Unlock()
	return x.s.Uint64()
}

func (x *lockedSource) Seed(seed int64) {
	x.lock.Lock()
	defer x.lock.Unlock()
	x.s.Seed(seed)
}
