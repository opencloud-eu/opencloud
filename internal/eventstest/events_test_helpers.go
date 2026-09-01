package eventstest

import (
	"encoding/json"
	"reflect"

	"github.com/google/uuid"

	rev "github.com/opencloud-eu/reva/v2/pkg/events"
	microevents "go-micro.dev/v4/events"
)

func NewTestBus() TestBus {
	return TestBus(make(chan rev.Event))
}

type TestBus chan rev.Event

func (tb TestBus) Consume(_ string, _ ...microevents.ConsumeOption) (<-chan microevents.Event, error) {
	ch := make(chan microevents.Event)
	go func() {
		for ev := range tb {
			b, _ := json.Marshal(ev.Event)
			ch <- microevents.Event{
				Payload: b,
				Metadata: map[string]string{
					rev.MetadatakeyEventID:   ev.ID,
					rev.MetadatakeyEventType: ev.Type,
				},
			}
		}
	}()
	return ch, nil
}

func (tb TestBus) Publish(e any) string {
	ev := rev.Event{
		ID:    uuid.New().String(),
		Type:  reflect.TypeOf(e).String(),
		Event: e,
	}
	tb <- ev
	return ev.ID
}
