package memory

import (
	"container/list"
	"context"
	"strings"
	"sync"
	"time"

	"github.com/armon/go-radix"
	"go-micro.dev/v4/store"
)

// MemStore is a in-memory store implementation using radix tree for fast
// prefix and suffix searches.
// Insertions are expected to be a bit slow due to the data structures, but
// searches are expected to be fast, including exact key search, as well as
// prefix and suffix searches (based on the number of elements to be returned).
// Prefix+suffix search isn't optimized and will depend on how many items we
// need to skip.
// It's also recommended to use reasonable limits when using prefix or suffix
// searches because we'll need to traverse the data structures to provide the
// results. The traversal will stop a soon as we have the required number of
// results, so it will be faster if we use a short limit.
//
// The overall performance will depend on how the radix trees are built.
// The number of elements won't directly affect the performance but how the
// keys are dispersed. The more dispersed the keys are, the faster the search
// will be, regardless of the number of keys. This happens due to the number
// of hops we need to do to reach the target element.
// This also mean that if the keys are too similar, the performance might be
// slower than expected even if the number of elements isn't too big.
type MemStore struct {
	preRadix     *radix.Tree
	sufRadix     *radix.Tree
	evictionList *list.List

	options store.Options

	lockGlob     sync.RWMutex
	lockEvicList sync.RWMutex // Read operation will modify the eviction list
}

type storeRecord struct {
	Key       string
	Value     []byte
	Metadata  map[string]interface{}
	Expiry    time.Duration
	ExpiresAt time.Time
}

type contextKey string

var targetContextKey contextKey

// NewContext prepares a context to be used with the memory implementation.
// The context is used to set up custom parameters to the specific implementation.
// In this case, you can configure the maximum capacity for the MemStore
// implementation as shown below.
// ```
// cache := NewMemStore(
//
//	store.WithContext(
//	  NewContext(
//	    ctx,
//	    map[string]interface{}{
//	      "maxCap": 50,
//	    },
//	  ),
//	),
//
// )
// ```
//
// Available options for the MemStore are:
// * "maxCap" -> 512 (int) The maximum number of elements the cache will hold.
// Adding additional elements will remove old elements to ensure we aren't over
// the maximum capacity.
//
// For convenience, this can also be used for the MultiMemStore.
func NewContext(ctx context.Context, storeParams map[string]interface{}) context.Context {
	return context.WithValue(ctx, targetContextKey, storeParams)
}

// NewMemStore creates a new MemStore instance
func NewMemStore(opts ...store.Option) store.Store {
	m := &MemStore{}
	_ = m.Init(opts...)
	return m
}

// Get the maximum capacity configured. If no maxCap has been configured
// (via `NewContext`), 512 will be used as maxCap.
func (m *MemStore) getMaxCap() int {
	maxCap := 512

	ctx := m.options.Context
	if ctx == nil {
		return maxCap
	}

	ctxValue := ctx.Value(targetContextKey)
	if ctxValue == nil {
		return maxCap
	}
	additionalOpts := ctxValue.(map[string]interface{})

	confCap, exists := additionalOpts["maxCap"]
	if exists {
		maxCap = confCap.(int)
	}
	return maxCap
}

// Init initializes the MemStore. If the MemStore was used, this will reset
// all the internal structures and the new options (passed as parameters)
// will be used.
func (m *MemStore) Init(opts ...store.Option) error {
	optList := store.Options{}
	for _, opt := range opts {
		opt(&optList)
	}

	m.lockGlob.Lock()
	defer m.lockGlob.Unlock()

	m.preRadix = radix.New()
	m.sufRadix = radix.New()
	m.evictionList = list.New()
	m.options = optList

	return nil
}

// Options returns the options being used
func (m *MemStore) Options() store.Options {
	m.lockGlob.RLock()
	defer m.lockGlob.RUnlock()

	return m.options
}

// Write the record in the MemStore.
// Note that Database and Table options will be ignored.
// Expiration options will take the following precedence:
// TTL option > expiration option > TTL record
//
// New elements will take the last position in the eviction list. Updating
// an element will also move the element to the last position.
//
// Although not recommended, new elements might be inserted with an
// already-expired date
func (m *MemStore) Write(r *store.Record, opts ...store.WriteOption) error {
	var element *list.Element

	wopts := store.WriteOptions{}
	for _, opt := range opts {
		opt(&wopts)
	}
	cRecord := toStoreRecord(r, wopts)

	m.lockGlob.Lock()
	defer m.lockGlob.Unlock()

	ele, exists := m.preRadix.Get(cRecord.Key)
	if exists {
		element = ele.(*list.Element)
		element.Value = cRecord

		m.evictionList.MoveToBack(element)
	} else {
		if m.evictionList.Len() >= m.getMaxCap() {
			elementToDelete := m.evictionList.Front()
			if elementToDelete != nil {
				recordToDelete := elementToDelete.Value.(*storeRecord)
				_, _ = m.preRadix.Delete(recordToDelete.Key)
				_, _ = m.sufRadix.Delete(recordToDelete.Key)
				m.evictionList.Remove(elementToDelete)
			}
		}
		element = m.evictionList.PushBack(cRecord)
		_, _ = m.preRadix.Insert(cRecord.Key, element)
		_, _ = m.sufRadix.Insert(reverseString(cRecord.Key), element)
	}
	return nil
}

// Read the key from the MemStore. A list of records will be returned even if
// you're asking for the exact key (only one record is expected in that case).
//
// Reading the exact element will move such element to the last position of
// the eviction list. This WON'T apply for prefix and / or suffix reads.
//
// This method guarantees that no expired element will be returned. For the
// case of exact read, the element will be removed and a "not found" error
// will be returned.
// For prefix and suffix reads, all the elements that we traverse through
// will be removed. This includes the elements we need to skip as well as
// the elements that might have gotten into the the result. Note that the
// elements that are over the limit won't be touched
//
// All read options are supported except Database and Table.
//
// For prefix and prefix+suffix options, the records will be returned in
// alphabetical order on the keys.
// For the suffix option (just suffix, no prefix), the records will be
// returned in alphabetical order after reversing the keys. This means,
// reverse all the keys and then sort them alphabetically. This just affects
// the sorting order; the keys will be returned as expected.
// This means that ["aboz", "caaz", "ziuz"] will be sorted as ["caaz", "aboz", "ziuz"]
// for the key "z" as suffix.
//
// Note that offset are supported but not recommended. There is no direct access
// to the record X. We'd need to skip all the records until we reach the specified
// offset, which could be problematic.
// Performance for prefix and suffix searches should be good assuming we limit
// the number of results we need to return.
func (m *MemStore) Read(key string, opts ...store.ReadOption) ([]*store.Record, error) {
	var element *list.Element

	ropts := store.ReadOptions{}
	for _, opt := range opts {
		opt(&ropts)
	}

	if !ropts.Prefix && !ropts.Suffix {
		m.lockGlob.RLock()
		ele, exists := m.preRadix.Get(key)
		if !exists {
			m.lockGlob.RUnlock()
			return nil, store.ErrNotFound
		}

		element = ele.(*list.Element)
		record := element.Value.(*storeRecord)
		if record.Expiry != 0 && record.ExpiresAt.Before(time.Now()) {
			// record expired -> need to delete
			m.lockGlob.RUnlock()
			m.lockGlob.Lock()
			defer m.lockGlob.Unlock()

			m.evictionList.Remove(element)
			_, _ = m.preRadix.Delete(key)
			_, _ = m.sufRadix.Delete(reverseString(key))
			return nil, store.ErrNotFound
		}

		m.lockEvicList.Lock()
		m.evictionList.MoveToBack(element)
		m.lockEvicList.Unlock()

		foundRecords := []*store.Record{
			fromStoreRecord(record),
		}
		m.lockGlob.RUnlock()

		return foundRecords, nil
	}

	records := []*store.Record{}
	expiredElements := make(map[string]*list.Element)

	m.lockGlob.RLock()
	if ropts.Prefix && ropts.Suffix {
		// if we need to check both prefix and suffix, go through the
		// prefix tree and skip elements without the right suffix. We
		// don't need to check the suffix tree because the elements
		// must be in both trees
		m.preRadix.WalkPrefix(key, m.radixTreeCallBackCheckSuffix(ropts.Offset, ropts.Limit, key, &records, expiredElements))
	} else {
		if ropts.Prefix {
			m.preRadix.WalkPrefix(key, m.radixTreeCallBack(ropts.Offset, ropts.Limit, &records, expiredElements))
		}
		if ropts.Suffix {
			m.sufRadix.WalkPrefix(reverseString(key), m.radixTreeCallBack(ropts.Offset, ropts.Limit, &records, expiredElements))
		}
	}
	m.lockGlob.RUnlock()

	// if there are expired elements, get a write lock and delete the expired elements
	if len(expiredElements) > 0 {
		m.lockGlob.Lock()
		for key, element := range expiredElements {
			m.evictionList.Remove(element)
			_, _ = m.preRadix.Delete(key)
			_, _ = m.sufRadix.Delete(reverseString(key))
		}
		m.lockGlob.Unlock()
	}
	return records, nil
}

// Delete removes the record based on the key. It won't return any error if it's missing
//
// Database and Table options aren't supported
func (m *MemStore) Delete(key string, opts ...store.DeleteOption) error {
	m.lockGlob.Lock()
	defer m.lockGlob.Unlock()

	ele, exists := m.preRadix.Get(key)
	if exists {
		element := ele.(*list.Element)
		m.evictionList.Remove(element)
		_, _ = m.preRadix.Delete(key)
		_, _ = m.sufRadix.Delete(reverseString(key))
	}
	return nil
}

// List the keys currently used in the MemStore
//
// # All options are supported except Database and Table
//
// For prefix and prefix+suffix options, the keys will be returned in
// alphabetical order.
// For the suffix option (just suffix, no prefix), the keys will be
// returned in alphabetical order after reversing the keys. This means,
// reverse all the keys and then sort them alphabetically. This just affects
// the sorting order; the keys will be returned as expected.
// This means that ["aboz", "caaz", "ziuz"] will be sorted as ["caaz", "aboz", "ziuz"]
func (m *MemStore) List(opts ...store.ListOption) ([]string, error) {
	records := []string{}
	expiredElements := make(map[string]*list.Element)

	lopts := store.ListOptions{}
	for _, opt := range opts {
		opt(&lopts)
	}

	if lopts.Prefix == "" && lopts.Suffix == "" {
		m.lockGlob.RLock()
		m.preRadix.Walk(m.radixTreeCallBackKeysOnly(lopts.Offset, lopts.Limit, &records, expiredElements))
		m.lockGlob.RUnlock()

		// if there are expired elements, get a write lock and delete the expired elements
		if len(expiredElements) > 0 {
			m.lockGlob.Lock()
			for key, element := range expiredElements {
				m.evictionList.Remove(element)
				_, _ = m.preRadix.Delete(key)
				_, _ = m.sufRadix.Delete(reverseString(key))
			}
			m.lockGlob.Unlock()
		}
		return records, nil
	}

	m.lockGlob.RLock()
	if lopts.Prefix != "" && lopts.Suffix != "" {
		// if we need to check both prefix and suffix, go through the
		// prefix tree and skip elements without the right suffix. We
		// don't need to check the suffix tree because the elements
		// must be in both trees
		m.preRadix.WalkPrefix(lopts.Prefix, m.radixTreeCallBackKeysOnlyWithSuffix(lopts.Offset, lopts.Limit, lopts.Suffix, &records, expiredElements))
	} else {
		if lopts.Prefix != "" {
			m.preRadix.WalkPrefix(lopts.Prefix, m.radixTreeCallBackKeysOnly(lopts.Offset, lopts.Limit, &records, expiredElements))
		}
		if lopts.Suffix != "" {
			m.sufRadix.WalkPrefix(reverseString(lopts.Suffix), m.radixTreeCallBackKeysOnly(lopts.Offset, lopts.Limit, &records, expiredElements))
		}
	}
	m.lockGlob.RUnlock()

	// if there are expired elements, get a write lock and delete the expired elements
	if len(expiredElements) > 0 {
		m.lockGlob.Lock()
		for key, element := range expiredElements {
			m.evictionList.Remove(element)
			_, _ = m.preRadix.Delete(key)
			_, _ = m.sufRadix.Delete(reverseString(key))
		}
		m.lockGlob.Unlock()
	}
	return records, nil
}

// Close closes the store
func (m *MemStore) Close() error {
	return nil
}

// String returns the name of the store implementation
func (m *MemStore) String() string {
	return "RadixMemStore"
}

// Len returns the number of items in the store
func (m *MemStore) Len() (int, bool) {
	eLen := m.evictionList.Len()
	pLen := m.preRadix.Len()
	sLen := m.sufRadix.Len()
	if eLen == pLen && pLen == sLen {
		return eLen, true
	}
	return 0, false
}

func (m *MemStore) radixTreeCallBack(offset, limit uint, result *[]*store.Record, expiredElements map[string]*list.Element) radix.WalkFn {
	currentIndex := new(uint) // needs to be a pointer so the value persist across callback calls
	maxIndex := new(uint)     // needs to be a pointer so the value persist across callback calls
	*maxIndex = offset + limit
	return func(key string, value interface{}) bool {
		element := value.(*list.Element)
		record := element.Value.(*storeRecord)

		if record.Expiry != 0 && record.ExpiresAt.Before(time.Now()) {
			// record has expired -> add element to the expiredElements map
			// and jump directly to the next element without increasing the index
			expiredElements[record.Key] = element
			return false
		}

		if *currentIndex >= offset && (*currentIndex < *maxIndex || *maxIndex == offset) {
			// if it's within expected range, add a copy to the results
			*result = append(*result, fromStoreRecord(record))
		}

		*currentIndex++

		if *currentIndex < *maxIndex || *maxIndex == offset {
			return false
		}
		return true
	}
}

func (m *MemStore) radixTreeCallBackCheckSuffix(offset, limit uint, presuf string, result *[]*store.Record, expiredElements map[string]*list.Element) radix.WalkFn {
	currentIndex := new(uint) // needs to be a pointer so the value persist across callback calls
	maxIndex := new(uint)     // needs to be a pointer so the value persist across callback calls
	*maxIndex = offset + limit
	return func(key string, value interface{}) bool {
		if !strings.HasSuffix(key, presuf) {
			return false
		}

		element := value.(*list.Element)
		record := element.Value.(*storeRecord)

		if record.Expiry != 0 && record.ExpiresAt.Before(time.Now()) {
			// record has expired -> add element to the expiredElements map
			// and jump directly to the next element without increasing the index
			expiredElements[record.Key] = element
			return false
		}

		if *currentIndex >= offset && (*currentIndex < *maxIndex || *maxIndex == offset) {
			*result = append(*result, fromStoreRecord(record))
		}

		*currentIndex++

		if *currentIndex < *maxIndex || *maxIndex == offset {
			return false
		}
		return true
	}
}

func (m *MemStore) radixTreeCallBackKeysOnly(offset, limit uint, result *[]string, expiredElements map[string]*list.Element) radix.WalkFn {
	currentIndex := new(uint) // needs to be a pointer so the value persist across callback calls
	maxIndex := new(uint)     // needs to be a pointer so the value persist across callback calls
	*maxIndex = offset + limit
	return func(key string, value interface{}) bool {
		element := value.(*list.Element)
		record := element.Value.(*storeRecord)

		if record.Expiry != 0 && record.ExpiresAt.Before(time.Now()) {
			// record has expired -> add element to the expiredElements map
			// and jump directly to the next element without increasing the index
			expiredElements[record.Key] = element
			return false
		}

		if *currentIndex >= offset && (*currentIndex < *maxIndex || *maxIndex == offset) {
			*result = append(*result, record.Key)
		}

		*currentIndex++

		if *currentIndex < *maxIndex || *maxIndex == offset {
			return false
		}
		return true
	}
}

func (m *MemStore) radixTreeCallBackKeysOnlyWithSuffix(offset, limit uint, presuf string, result *[]string, expiredElements map[string]*list.Element) radix.WalkFn {
	currentIndex := new(uint) // needs to be a pointer so the value persist across callback calls
	maxIndex := new(uint)     // needs to be a pointer so the value persist across callback calls
	*maxIndex = offset + limit
	return func(key string, value interface{}) bool {
		if !strings.HasSuffix(key, presuf) {
			return false
		}

		element := value.(*list.Element)
		record := element.Value.(*storeRecord)

		if record.Expiry != 0 && record.ExpiresAt.Before(time.Now()) {
			// record has expired -> add element to the expiredElements map
			// and jump directly to the next element without increasing the index
			expiredElements[record.Key] = element
			return false
		}

		if *currentIndex >= offset && (*currentIndex < *maxIndex || *maxIndex == offset) {
			*result = append(*result, record.Key)
		}

		*currentIndex++

		if *currentIndex < *maxIndex || *maxIndex == offset {
			return false
		}
		return true
	}
}
