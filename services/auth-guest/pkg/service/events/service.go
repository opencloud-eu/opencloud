// Copyright 2026 OpenCloud GmbH <mail@opencloud.eu>
// SPDX-License-Identifier: Apache-2.0

package events

import (
	"context"
	"sync"
	"sync/atomic"

	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/services/auth-guest/pkg/service/authguest"
	"github.com/opencloud-eu/reva/v2/pkg/events"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
)

var tracer trace.Tracer

func init() {
	tracer = otel.Tracer("github.com/opencloud-eu/opencloud/services/auth-guest/pkg/service/events")
}

var (
	_numConsumersDefault = 1
)

// EventConsumer consumes guest share events.
type EventConsumer struct {
	ctx    context.Context
	log    log.Logger
	stream events.Stream

	authGuest authguest.AuthGuest

	numConsumers int

	events []events.Unmarshaller

	stopCh  chan struct{}
	stopped *atomic.Bool
}

// NewEventConsumer creates a new event consumer.
func NewEventConsumer(stream events.Stream, opts ...Option) (*EventConsumer, error) {
	o := &Options{
		NumConsumers: _numConsumersDefault,
	}
	for _, opt := range opts {
		opt(o)
	}

	s := &EventConsumer{
		ctx:          o.Context,
		log:          o.Logger,
		stream:       stream,
		authGuest:    o.AuthGuestService,
		events:       o.RegisteredEvents,
		numConsumers: o.NumConsumers,
		stopCh:       make(chan struct{}, 1),
		stopped:      new(atomic.Bool),
	}

	return s, nil
}

// Run to fulfil Runner interface
func (s *EventConsumer) Run() error {
	ch, err := events.Consume(s.stream, "auth-guest", s.events...)
	if err != nil {
		return err
	}

	var wg sync.WaitGroup
	ctx, cancel := context.WithCancel(s.ctx)
	defer cancel()

	s.log.Debug().Int("worker.count", s.numConsumers).
		Str("messaging.consumer.group.name", "auth-guest").
		Str("messaging.system", "nats").
		Str("messaging.operation.name", "receive").
		Msg("starting event processing workers")

	// start workers
	for i := range s.numConsumers {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				case e, ok := <-ch:
					if !ok {
						return
					}
					if err := s.processEvent(e); err != nil {
						s.log.Error().Err(err).
							Int("worker", workerID).
							Interface("event", e).
							Msg("failed to process event")
					}
				}
			}
		}(i)
	}

	// wait for stop signal
	<-s.stopCh
	cancel() // signal workers to stop
	wg.Wait()

	return nil
}

// Close will make the service to stop processing, so the `Run`
// method can finish.
func (s *EventConsumer) Close() {
	if s.stopped.CompareAndSwap(false, true) {
		close(s.stopCh)
	}
}

// processEvent dispatches an event to the matching handler.
func (s *EventConsumer) processEvent(e events.Event) error {
	ctx := e.GetTraceContext(s.ctx)
	ctx, span := tracer.Start(ctx, "processEvent")
	defer span.End()

	s.log.Debug().Interface("event", e).Msg("processing event")

	switch ev := e.Event.(type) {
	case events.ShareCreated:
		return s.handleShareCreated(ctx, ev)
	case events.ShareRemoved:
		return s.handleShareRemoved(ctx, ev)
	case events.ShareExpired:
		return s.handleShareExpired(ctx, ev)
	default:
		s.log.Warn().
			Str("eventtype", e.Type).
			Msg("unhandled event")
	}

	return nil
}
