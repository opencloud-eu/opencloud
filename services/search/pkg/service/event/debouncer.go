package event

import (
	"sync"
	"time"

	provider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"

	"github.com/opencloud-eu/opencloud/pkg/log"
)

// SpaceDebouncer debounces operations on spaces for a configurable amount of time
type SpaceDebouncer struct {
	after      time.Duration
	timeout    time.Duration
	f          func(id *provider.StorageSpaceId, force bool)
	pending    map[string]*workItem
	inProgress sync.Map

	mutex sync.Mutex
	log   log.Logger
}

type workItem struct {
	t       *time.Timer
	timeout *time.Timer
	force   bool

	work func()
}

type AckFunc func() error

// NewSpaceDebouncer returns a new SpaceDebouncer instance
func NewSpaceDebouncer(d time.Duration, timeout time.Duration, f func(id *provider.StorageSpaceId, force bool), logger log.Logger) *SpaceDebouncer {
	return &SpaceDebouncer{
		after:      d,
		timeout:    timeout,
		f:          f,
		pending:    map[string]*workItem{},
		inProgress: sync.Map{},
		log:        logger,
	}
}

// Debounce restarts the debounce timer for the given space. If force is set,
// the scheduled run is a forced one. A pending run stays forced even if it is
// debounced again without force.
func (d *SpaceDebouncer) Debounce(id *provider.StorageSpaceId, ack AckFunc, force bool) {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	if wi := d.pending[id.OpaqueId]; wi != nil {
		wi.force = wi.force || force
		if ack != nil {
			go ack() // Acknowledge the event immediately, the according space is already scheduled for indexing
		}
		wi.t.Reset(d.after)
		return
	}

	wi := &workItem{force: force}
	wi.work = func() {
		if _, ok := d.inProgress.Load(id.OpaqueId); ok {
			// Reschedule this run for when the previous run has finished
			d.mutex.Lock()
			if wi := d.pending[id.OpaqueId]; wi != nil {
				wi.t.Reset(d.after)
			}
			d.mutex.Unlock()
			return
		}

		d.mutex.Lock()
		wi.timeout.Stop() // stop the timeout timer if it is running
		delete(d.pending, id.OpaqueId)
		force := wi.force
		d.inProgress.Store(id.OpaqueId, true)
		defer func() {
			d.inProgress.Delete(id.OpaqueId)
		}()
		d.mutex.Unlock() // release the lock early to allow other goroutines to debounce

		d.f(id, force)
		go func() {
			if ack != nil {
				if err := ack(); err != nil {
					d.log.Error().Err(err).Msg("error while acknowledging event")
				}
			}
		}()
	}
	wi.t = time.AfterFunc(d.after, wi.work)
	wi.timeout = time.AfterFunc(d.timeout, func() {
		d.log.Debug().Msg("timeout while waiting for space debouncer to finish")
		wi.t.Stop()
		wi.work()
	})

	d.pending[id.OpaqueId] = wi

}
