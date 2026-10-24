package activitylog

import (
	"sync"
	"time"
)

// Debouncer is used to debounce writes to the activity log store.
type Debouncer struct {
	after      time.Duration
	f          func(id string, ra []RawActivity) error
	pending    sync.Map
	inProgress sync.Map

	mutex sync.Mutex
}

type queueItem struct {
	activities []RawActivity
	timer      *time.Timer
}

// NewDebouncer returns a new Debouncer instance.
func NewDebouncer(d time.Duration, f func(id string, ra []RawActivity) error) *Debouncer {
	return &Debouncer{
		after:      d,
		f:          f,
		pending:    sync.Map{},
		inProgress: sync.Map{},
	}
}

// Debounce restarts the debounce timer for the given space.
func (d *Debouncer) Debounce(id string, ra RawActivity) {
	if d.after == 0 {
		d.f(id, []RawActivity{ra})
		return
	}

	d.mutex.Lock()
	defer d.mutex.Unlock()

	item := &queueItem{
		activities: []RawActivity{ra},
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
		})
	}

	d.pending.Store(id, item)
}
