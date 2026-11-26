package memory

import (
	"sync"

	"go-micro.dev/v4/store"
)

// MultiMemStore is a in-memory store implementation using multiple MemStore
// to provide support for multiple databases and tables.
// Each table will be mapped to its own MemStore, which will be completely
// isolated from the rest. In particular, each MemStore will have its own
// capacity, so it's possible to have 10 MemStores with full capacity (512
// by default)
//
// The options will be the same for all MemStores unless they're explicitly
// initialized otherwise.
//
// Since each MemStore is isolated, the required synchronization caused by
// concurrency will be minimal if the threads use different tables
type MultiMemStore struct {
	storeMap     map[string]*MemStore
	storeMapLock sync.RWMutex
	genOpts      []store.Option
}

// NewMultiMemStore creates a new MultiMemStore. A new MemStore will be mapped based on the options.
// A default MemStore will be mapped if no Database and Table aren't used.
func NewMultiMemStore(opts ...store.Option) store.Store {
	m := &MultiMemStore{
		storeMap: make(map[string]*MemStore),
		genOpts:  opts,
	}
	_ = m.Init(opts...)
	return m
}

func (m *MultiMemStore) getMemStore(prefix string) *MemStore {
	m.storeMapLock.RLock()
	mStore, exists := m.storeMap[prefix]

	if exists {
		m.storeMapLock.RUnlock()
		return mStore
	}

	m.storeMapLock.RUnlock()

	// if not exists
	newStore := NewMemStore(m.genOpts...).(*MemStore)

	m.storeMapLock.Lock()
	m.storeMap[prefix] = newStore
	m.storeMapLock.Unlock()
	return newStore
}

// Init initializes the mapped MemStore based on the Database and Table values
// from the options with the same options. The target MemStore will be
// reinitialized if needed.
func (m *MultiMemStore) Init(opts ...store.Option) error {
	optList := store.Options{}
	for _, opt := range opts {
		opt(&optList)
	}

	prefix := optList.Database + "/" + optList.Table

	mStore := m.getMemStore(prefix)
	return mStore.Init(opts...)
}

// Options returns the options used to create the MultiMemStore.
// Specific options for each MemStore aren't available
func (m *MultiMemStore) Options() store.Options {
	optList := store.Options{}
	for _, opt := range m.genOpts {
		opt(&optList)
	}
	return optList
}

// Write the record in the target MemStore based on the Database and Table
// values from the options. A default MemStore will be used if no Database
// and Table options are provided.
// The write options will be forwarded to the target MemStore
func (m *MultiMemStore) Write(r *store.Record, opts ...store.WriteOption) error {
	wopts := store.WriteOptions{}
	for _, opt := range opts {
		opt(&wopts)
	}

	prefix := wopts.Database + "/" + wopts.Table

	mStore := m.getMemStore(prefix)
	return mStore.Write(r, opts...)
}

// Read the matching records in the target MemStore based on the Database and Table
// values from the options. A default MemStore will be used if no Database
// and Table options are provided.
// The read options will be forwarded to the target MemStore.
//
// The expectations regarding the results (sort order, eviction policies, etc)
// will be the same as the target MemStore
func (m *MultiMemStore) Read(key string, opts ...store.ReadOption) ([]*store.Record, error) {
	ropts := store.ReadOptions{}
	for _, opt := range opts {
		opt(&ropts)
	}

	prefix := ropts.Database + "/" + ropts.Table

	mStore := m.getMemStore(prefix)
	return mStore.Read(key, opts...)
}

// Delete the matching records in the target MemStore based on the Database and Table
// values from the options. A default MemStore will be used if no Database
// and Table options are provided.
//
// Matching records from other Tables won't be affected. In fact, we won't
// access to other Tables
func (m *MultiMemStore) Delete(key string, opts ...store.DeleteOption) error {
	dopts := store.DeleteOptions{}
	for _, opt := range opts {
		opt(&dopts)
	}

	prefix := dopts.Database + "/" + dopts.Table

	mStore := m.getMemStore(prefix)
	return mStore.Delete(key, opts...)
}

// List the keys in the target MemStore based on the Database and Table
// values from the options. A default MemStore will be used if no Database
// and Table options are provided.
// The list options will be forwarded to the target MemStore.
func (m *MultiMemStore) List(opts ...store.ListOption) ([]string, error) {
	lopts := store.ListOptions{}
	for _, opt := range opts {
		opt(&lopts)
	}

	prefix := lopts.Database + "/" + lopts.Table

	mStore := m.getMemStore(prefix)
	return mStore.List(opts...)
}

// Close closes the store
func (m *MultiMemStore) Close() error {
	return nil
}

// String returns the name of the store implementation
func (m *MultiMemStore) String() string {
	return "MultiRadixMemStore"
}
