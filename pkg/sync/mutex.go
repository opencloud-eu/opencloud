package sync

import (
	"sync"
)

// NamedRWMutex works the same as RWMutex, the only difference is that it stores mutexes in a map and reuses them.
// It's handy if you want to write-lock, write-unlock, read-lock and read-unlock for specific names only.
type NamedRWMutex struct {
	pool sync.Pool
	mus  sync.Map
}

// NewNamedRWMutex returns a new instance of NamedRWMutex.
func NewNamedRWMutex() NamedRWMutex {
	return NamedRWMutex{pool: sync.Pool{New: func() any {
		return new(sync.RWMutex)
	}}}
}

// Lock locks rw for writing.
func (m *NamedRWMutex) Lock(name string) {
	m.loadOrStore(name).Lock()
}

// Unlock unlocks rw for writing.
func (m *NamedRWMutex) Unlock(name string) {
	m.loadOrStore(name).Unlock()
}

// RLock locks rw for reading.
func (m *NamedRWMutex) RLock(name string) {
	m.loadOrStore(name).RLock()
}

// RUnlock undoes a single RLock call.
func (m *NamedRWMutex) RUnlock(name string) {
	m.loadOrStore(name).RUnlock()
}

func (m *NamedRWMutex) loadOrStore(name string) *sync.RWMutex {
	pmu := m.pool.Get()
	mmu, loaded := m.mus.LoadOrStore(name, pmu)
	if loaded {
		m.pool.Put(pmu)
	}

	return mmu.(*sync.RWMutex)
}
