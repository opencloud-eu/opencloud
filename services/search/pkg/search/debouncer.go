package search

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
	f          func(id *provider.StorageSpaceId)
	pending    map[string]*workItem
	inProgress sync.Map

	mutex sync.Mutex
	log   log.Logger
}

type workItem struct {
	t       *time.Timer
	timeout *time.Timer

	trigger func()
}

type AckFunc func() error

// NewSpaceDebouncer returns a new SpaceDebouncer instance
func NewSpaceDebouncer(d time.Duration, timeout time.Duration, f func(id *provider.StorageSpaceId), logger log.Logger) *SpaceDebouncer {
	return &SpaceDebouncer{
		after:      d,
		timeout:    timeout,
		f:          f,
		pending:    map[string]*workItem{},
		inProgress: sync.Map{},
		log:        logger,
	}
}

// Debounce restars the debounce timer for the given space
func (d *SpaceDebouncer) Debounce(id *provider.StorageSpaceId, ack AckFunc) {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	if wi := d.pending[id.OpaqueId]; wi != nil {
		if ack != nil {
			go ack() // Acknowledge the event immediately, the according space is already scheduled for indexing
		}
		wi.t.Reset(d.after)
		return
	}

	trigger := func() {
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
		delete(d.pending, id.OpaqueId)
		d.inProgress.Store(id.OpaqueId, true)
		defer d.inProgress.Delete(id.OpaqueId)
		d.mutex.Unlock() // release the lock early to allow other goroutines to debounce

		d.f(id)
		go func() {
			if ack != nil {
				if err := ack(); err != nil {
					d.log.Error().Err(err).Msg("error while acknowledging event")
				}
			}
		}()
	}
	t := time.AfterFunc(d.after, trigger)

	d.pending[id.OpaqueId] = &workItem{
		trigger: trigger,
		t:       t,
		timeout: time.AfterFunc(d.timeout, func() {
			d.log.Debug().Msg("timeout while waiting for space debouncer to finish")
			t.Stop()
			trigger()
		}),
	}

}
