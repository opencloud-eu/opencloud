//go:build enable_antithesis_sdk

package assert

import (
	"sync"
	"sync/atomic"

	"github.com/antithesishq/antithesis-sdk-go/internal"
)

// Tracking info for one assertion id.
type trackerInfo struct {
	Filename  string
	Classname string

	passEmitted atomic.Bool
	failEmitted atomic.Bool
	// Serializes only first-emission attempts (including retries after an
	// emission error); never taken once a flag is set.
	emitMutex sync.Mutex
}

// assertTracker keeps track of the unique asserts evaluated.
var assertTracker sync.Map // message key -> *trackerInfo

func getTrackerEntry(messageKey string, filename string, classname string) *trackerInfo {
	if entry, ok := assertTracker.Load(messageKey); ok {
		return entry.(*trackerInfo)
	}
	entry, _ := assertTracker.LoadOrStore(messageKey,
		&trackerInfo{Filename: filename, Classname: classname})
	return entry.(*trackerInfo)
}

func (ti *trackerInfo) emit(ai *assertInfo) {
	if ti == nil || ai == nil {
		return
	}

	// Registrations are just sent to voidstar
	if !ai.Hit {
		emitAssert(ai)
		return
	}

	flag := &ti.passEmitted
	if !ai.Condition {
		flag = &ti.failEmitted
	}
	if flag.Load() {
		return
	}
	ti.emitMutex.Lock()
	defer ti.emitMutex.Unlock()
	if flag.Load() {
		return
	}
	// The flag is only set on success, so an assertion whose emission
	// failed is retried on its next evaluation (as before).
	if emitAssert(ai) == nil {
		flag.Store(true)
	}
}

// Whether this assertion id has already emitted for this condition — i.e.
// whether an evaluation can return before capturing its location.
func trackerEmitted(id string, condition bool) bool {
	entry, ok := assertTracker.Load(id)
	if !ok {
		return false
	}
	ti := entry.(*trackerInfo)
	if condition {
		return ti.passEmitted.Load()
	}
	return ti.failEmitted.Load()
}

func emitAssert(ai *assertInfo) error {
	// The version message is emitted by the internal package when the
	// handler initializes, before any output can happen.
	return internal.Json_data(wrappedAssertInfo{ai})
}
