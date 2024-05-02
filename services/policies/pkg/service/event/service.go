package eventSVC

import (
	"context"
	"sync/atomic"

	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/services/policies/pkg/engine"
	"github.com/opencloud-eu/reva/v2/pkg/events"
	"go.opentelemetry.io/otel/trace"
)

// Service defines the service handlers.
type Service struct {
	ctx     context.Context
	query   string
	log     log.Logger
	stream  events.Stream
	engine  engine.Engine
	tp      trace.TracerProvider
	stopCh  chan struct{}
	stopped *atomic.Bool
}

// New returns a service implementation for Service.
func New(ctx context.Context, stream events.Stream, logger log.Logger, tp trace.TracerProvider, engine engine.Engine, query string) (Service, error) {
	svc := Service{
		ctx:     ctx,
		log:     logger,
		query:   query,
		tp:      tp,
		engine:  engine,
		stream:  stream,
		stopCh:  make(chan struct{}, 1),
		stopped: new(atomic.Bool),
	}

	return svc, nil
}

// Run to fulfil Runner interface
func (s Service) Run() error {
	ch, err := events.Consume(s.stream, "policies", events.StartPostprocessingStep{})
	if err != nil {
		return err
	}

EventLoop:
	for {
		select {
		case <-s.stopCh:
			break EventLoop
		case e, ok := <-ch:
			if !ok {
				break EventLoop
			}

			err := s.processEvent(e)
			if err != nil {
				return err
			}

			if s.stopped.Load() {
				break EventLoop
			}
		}
	}

	return nil
}

// Close will make the policies service to stop processing, so the `Run`
// method can finish.
// TODO: Underlying services can't be stopped. This means that some goroutines
// will get stuck trying to push events through a channel nobody is reading
// from, so resources won't be freed and there will be memory leaks. For now,
// if the service is stopped, you should close the app soon after.
func (s Service) Close() {
	if s.stopped.CompareAndSwap(false, true) {
		close(s.stopCh)
	}
}

func (s Service) processEvent(e events.Event) error {
	ctx := e.GetTraceContext(s.ctx)
	ctx, span := s.tp.Tracer("policies").Start(ctx, "processEvent")
	defer span.End()

	switch ev := e.Event.(type) {
	case events.StartPostprocessingStep:
		if ev.StepToStart != events.PPStepPolicies {
			return nil
		}

		outcome := events.PPOutcomeContinue

		if s.query != "" {
			env := engine.Environment{
				Stage: engine.StagePP,
				Resource: engine.Resource{
					Name: ev.Filename,
					URL:  ev.URL,
					Size: ev.Filesize,
				},
			}

			if ev.ExecutingUser != nil {
				env.User = *ev.ExecutingUser
			}

			if ev.ResourceID != nil {
				env.Resource.ID = *ev.ResourceID
			}

			result, err := s.engine.Evaluate(context.TODO(), s.query, env)
			if err != nil {
				s.log.Error().Err(err).Msg("unable evaluate policy")
			}

			if !result {
				outcome = events.PPOutcomeDelete
			}
		}

		if err := events.Publish(ctx, s.stream, events.PostprocessingStepFinished{
			Outcome:       outcome,
			UploadID:      ev.UploadID,
			ExecutingUser: ev.ExecutingUser,
			Filename:      ev.Filename,
			FinishedStep:  ev.StepToStart,
		}); err != nil {
			return err
		}
	}
	return nil
}
