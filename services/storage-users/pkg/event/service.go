package event

import (
	"context"
	"time"

	apiGateway "github.com/cs3org/go-cs3apis/cs3/gateway/v1beta1"
	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/services/storage-users/pkg/config"
	"github.com/opencloud-eu/opencloud/services/storage-users/pkg/task"
	"github.com/opencloud-eu/reva/v3/pkg/events"
	"github.com/opencloud-eu/reva/v3/pkg/rgrpc/todo/pool"
)

const (
	consumerGroup = "storage-users"
)

// Service wraps all common logic that is needed to react to incoming events.
type Service struct {
	gatewaySelector pool.Selectable[apiGateway.GatewayAPIClient]
	eventStream     events.Stream
	logger          log.Logger
	config          config.Config
	ctx             context.Context
}

// NewService prepares and returns a Service implementation.
func NewService(ctx context.Context, gatewaySelector pool.Selectable[apiGateway.GatewayAPIClient], eventStream events.Stream, logger log.Logger, conf config.Config) (Service, error) {
	svc := Service{
		gatewaySelector: gatewaySelector,
		eventStream:     eventStream,
		logger:          logger,
		config:          conf,
		ctx:             ctx,
	}

	return svc, nil
}

// Run to fulfil Runner interface
func (s Service) Run() error {
	ch, err := events.Consume(s.eventStream, consumerGroup, PurgeTrashBin{})
	if err != nil {
		return err
	}

	for {
		select {
		case <-s.ctx.Done():
			s.logger.Info().Str("service", s.config.Service.Name).Msg("Context canceled. Shutting down event handler")
			return nil
		case e, more := <-ch:
			if !more {
				s.logger.Info().Str("service", s.config.Service.Name).Msg("Event channel closed. Shutting down event handler")
				// the channel was closed we can stop here
				return nil
			}
			s.handleEvent(e)
		}
	}
}

func (s Service) handleEvent(e events.Event) {
	var errs []error

	switch ev := e.Event.(type) {
	case PurgeTrashBin:
		executionTime := ev.ExecutionTime
		if executionTime.IsZero() {
			executionTime = time.Now()
		}

		tasks := map[task.SpaceType]time.Time{
			task.Project:  executionTime.Add(-s.config.Tasks.PurgeTrashBin.ProjectDeleteBefore),
			task.Personal: executionTime.Add(-s.config.Tasks.PurgeTrashBin.PersonalDeleteBefore),
		}

		for spaceType, deleteBefore := range tasks {
			// skip task execution if the deleteBefore time is the same as the now time,
			// which indicates that the duration configuration for this space type is set to 0 which is the equivalent to disabled.
			if deleteBefore.Equal(executionTime) {
				continue
			}

			if err := task.PurgeTrashBin(s.config.ServiceAccount.ServiceAccountID, deleteBefore, spaceType, s.gatewaySelector, s.config.ServiceAccount.ServiceAccountSecret); err != nil {
				errs = append(errs, err)
			}
		}

	}

	for _, err := range errs {
		s.logger.Error().Err(err).Interface("event", e).Msg("Error running PurgeTrashBin task")
	}
}
