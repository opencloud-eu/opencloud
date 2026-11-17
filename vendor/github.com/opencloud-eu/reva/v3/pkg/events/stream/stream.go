// Package stream provides streaming clients used by `Consume` and `Publish` methods
package stream

import (
	"encoding/json"
	"reflect"

	"go-micro.dev/v4/events"
)

// Chan is a channel based streaming clients
// Useful for tests or in memory applications
type Chan [2]chan interface{}

// Publish implementation
func (ch Chan) Publish(_ string, msg interface{}, _ ...events.PublishOption) error {
	go func() {
		ch[0] <- msg
	}()
	return nil
}

// Consume implementation
func (ch Chan) Consume(_ string, _ ...events.ConsumeOption) (<-chan events.Event, error) {
	evch := make(chan events.Event)
	go func() {
		for {
			e := <-ch[1]
			if e == nil {
				// channel closed
				return
			}
			b, _ := json.Marshal(e)
			evname := reflect.TypeOf(e).String()
			evch <- events.Event{
				Payload:  b,
				Metadata: map[string]string{"eventtype": evname},
			}
		}
	}()
	return evch, nil
}
