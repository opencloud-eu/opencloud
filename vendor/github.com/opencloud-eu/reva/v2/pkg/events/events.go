// Copyright 2018-2021 CERN
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
// In applying this license, CERN does not waive the privileges and immunities
// granted to it by virtue of its status as an Intergovernmental Organization
// or submit itself to any jurisdiction.

package events

import (
	"context"
	"log"
	"reflect"

	"github.com/google/uuid"
	ctxpkg "github.com/opencloud-eu/reva/v2/pkg/ctx"
	"go-micro.dev/v4/events"
	"go.opentelemetry.io/otel/propagation"
)

var (
	// MainQueueName is the name of the main queue
	// All events will go through here as they are forwarded to the consumer via the
	// group name
	// TODO: "fan-out" so not all events go through the same queue? requires investigation
	MainQueueName = "main-queue"

	// MetadatakeyEventType is the key used for the eventtype in the metadata map of the event
	MetadatakeyEventType = "eventtype"

	// MetadatakeyEventID is the key used for the eventID in the metadata map of the event
	MetadatakeyEventID = "eventid"

	// MetadatakeyTraceParent is the key used for the traceparent in the metadata map of the event
	MetadatakeyTraceParent = "traceparent"

	// MetadatakeyInitiatorID is the key used for the initiator id in the metadata map of the event
	MetadatakeyInitiatorID = "initiatorid"
)

type (
	// Unmarshaller is the interface events need to fulfill
	Unmarshaller interface {
		Unmarshal([]byte) (interface{}, error)
	}

	// Publisher is the interface publishers need to fulfill
	Publisher interface {
		Publish(string, interface{}, ...events.PublishOption) error
	}

	// Consumer is the interface consumer need to fulfill
	Consumer interface {
		Consume(string, ...events.ConsumeOption) (<-chan events.Event, error)
	}

	// Stream is the interface common to Publisher and Consumer
	Stream interface {
		Publish(string, interface{}, ...events.PublishOption) error
		Consume(string, ...events.ConsumeOption) (<-chan events.Event, error)
	}

	// Event is the envelope for events
	Event struct {
		Type        string
		ID          string
		TraceParent string
		InitiatorID string
		Event       interface{}
	}
)

// Consume returns a channel that will get all events that match the given evs
// group defines the service type: One group will get exactly one copy of a event that is emitted
// NOTE: uses reflect on initialization
func Consume(s Consumer, group string, evs ...Unmarshaller) (<-chan Event, error) {
	return ConsumeWithOptions(s, ConsumeOptions{Group: group}, evs...)
}

// ConsumeWithOptions returns a channel that will get all events that match the given evs
// opts defines the options for the consumer
func ConsumeWithOptions(s Consumer, opts ConsumeOptions, evs ...Unmarshaller) (<-chan Event, error) {
	o := []events.ConsumeOption{
		events.WithGroup(opts.Group),
	}
	if !opts.AutoAck {
		o = append(o, events.WithAutoAck(false, opts.AckWait))
	}
	c, err := s.Consume(MainQueueName, o...)
	if err != nil {
		return nil, err
	}

	registeredEvents := map[string]Unmarshaller{}
	for _, e := range evs {
		typ := reflect.TypeOf(e)
		registeredEvents[typ.String()] = e
	}

	outchan := make(chan Event)
	go func() {
		for {
			e := <-c
			et := e.Metadata[MetadatakeyEventType]
			ev, ok := registeredEvents[et]
			if !ok {
				continue
			}

			event, err := ev.Unmarshal(e.Payload)
			if err != nil {
				log.Printf("can't unmarshal event %v", err)
				continue
			}

			outchan <- Event{
				Type:        et,
				ID:          e.Metadata[MetadatakeyEventID],
				TraceParent: e.Metadata[MetadatakeyTraceParent],
				InitiatorID: e.Metadata[MetadatakeyInitiatorID],
				Event:       event,
			}
		}
	}()
	return outchan, nil
}

// ConsumeAll allows consuming all events. Note that unmarshalling must be done manually in this case, therefore Event.Event will always be of type []byte
func ConsumeAll(s Consumer, group string) (<-chan Event, error) {
	return ConsumeAllWithOptions(s, ConsumeOptions{Group: group})
}

// ConsumeAllWithOptions allows consuming all events. Note that unmarshalling must be done manually in this case, therefore Event.Event will always be of type []byte
func ConsumeAllWithOptions(s Consumer, opts ConsumeOptions) (<-chan Event, error) {
	o := []events.ConsumeOption{
		events.WithGroup(opts.Group),
	}
	if !opts.AutoAck {
		o = append(o, events.WithAutoAck(false, opts.AckWait))
	}
	c, err := s.Consume(MainQueueName, o...)
	if err != nil {
		return nil, err
	}

	outchan := make(chan Event)
	go func() {
		for {
			e := <-c
			outchan <- Event{
				Type:        e.Metadata[MetadatakeyEventType],
				ID:          e.Metadata[MetadatakeyEventID],
				TraceParent: e.Metadata[MetadatakeyTraceParent],
				InitiatorID: e.Metadata[MetadatakeyInitiatorID],
				Event:       e.Payload,
			}
		}
	}()
	return outchan, nil
}

// Publish publishes the ev to the MainQueue from where it is distributed to all subscribers
// NOTE: needs to use reflect on runtime
func Publish(ctx context.Context, s Publisher, ev interface{}) error {
	evName := reflect.TypeOf(ev).String()
	traceParent := getTraceParentFromCtx(ctx)
	iid, _ := ctxpkg.ContextGetInitiator(ctx)
	return s.Publish(MainQueueName, ev, events.WithMetadata(map[string]string{
		MetadatakeyEventType:   evName,
		MetadatakeyEventID:     uuid.New().String(),
		MetadatakeyTraceParent: traceParent,
		MetadatakeyInitiatorID: iid,
	}))
}

// GetTraceContext extracts the trace context from the event and injects it into the given
// context.
func (e *Event) GetTraceContext(ctx context.Context) context.Context {
	return propagation.TraceContext{}.Extract(ctx, propagation.MapCarrier{
		"traceparent": e.TraceParent,
	})
}

// getTraceParentFromCtx will return a traceparent from the context if it exists.
// it will be a string as specificied here: https://www.w3.org/TR/trace-context/
// If no trace info in the context, the return will be an empty string
func getTraceParentFromCtx(ctx context.Context) string {
	mc := propagation.MapCarrier{}
	tc := propagation.TraceContext{}
	tc.Inject(ctx, &mc)
	return mc["traceparent"]
}
