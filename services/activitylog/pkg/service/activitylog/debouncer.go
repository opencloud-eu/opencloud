package activitylog

import (
	"sync"
	"time"

	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/services/activitylog/pkg/data"
)

type Debouncer struct {
	after      time.Duration
	f          func(id string, ra []data.RawActivity) error
	pending    sync.Map
	inProgress sync.Map
	log        log.Logger

	mutex sync.Mutex
}

type queueItem struct {
	activities []data.RawActivity
	timer      *time.Timer
}

// NewDebouncer returns a new Debouncer instance
func NewDebouncer(log log.Logger, d time.Duration, f func(id string, ra []data.RawActivity) error) *Debouncer {
	return &Debouncer{
		after:      d,
		f:          f,
		pending:    sync.Map{},
		inProgress: sync.Map{},
		log:        log,
	}
}

// Debounce restarts the debounce timer for the given space
func (d *Debouncer) Debounce(id string, ra data.RawActivity, ack func() error) {
	if d.after == 0 {
		d.f(id, []data.RawActivity{ra})
		return
	}

	d.mutex.Lock()
	defer d.mutex.Unlock()

	activities := []data.RawActivity{ra}
	item := &queueItem{
		activities: activities,
	}
	if i, ok := d.pending.Load(id); ok {
		// if the item is already in the queue, append the new activities
		item, ok = i.(*queueItem)
		if ok {
			item.activities = append(item.activities, ra)
		}
	}

	if item.timer == nil {
		item.timer = time.AfterFunc(d.after, func() {
			if _, ok := d.inProgress.Load(id); ok {
				// Reschedule this run for when the previous run has finished
				d.mutex.Lock()
				if i, ok := d.pending.Load(id); ok {
					i.(*queueItem).timer.Reset(d.after)
				}

				d.mutex.Unlock()
				return
			}

			d.pending.Delete(id)
			d.inProgress.Store(id, true)
			defer d.inProgress.Delete(id)
			d.f(id, item.activities)
			go func() {
				if ack != nil {
					if err := ack(); err != nil {
						d.log.Error().Err(err).Msg("error while acknowledging event")
					}
				}
			}()
		})
	}

	d.pending.Store(id, item)
}
