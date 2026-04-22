package web

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/opencloud-eu/reva/v2/pkg/events"

	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/services/console/pkg/console"
	"github.com/opencloud-eu/opencloud/services/sse/pkg/service"
)

type SSEServiceOptions struct {
	EventStream events.Stream `validate:"required"`
	Logger      log.Logger
}

func (o SSEServiceOptions) Validate() error {
	if err := console.Validate.Struct(o); err != nil {
		return fmt.Errorf("(%w): %w", console.ErrOptionsInvalid, err)
	}

	return nil
}

type SSEService struct {
	eventStream events.Stream
	logger      log.Logger
}

func NewSSEService(o SSEServiceOptions) (SSEService, error) {
	if err := o.Validate(); err != nil {
		return SSEService{}, err
	}

	return SSEService{
		eventStream: o.EventStream,
		logger:      o.Logger,
	}, nil
}

func (s SSEService) ThemeApply(ctx context.Context) error {
	return events.Publish(ctx, s.eventStream, events.SendSSE{
		UserIDs: []string{service.SSETopicAllUsers},
		Type:    "console-notification",
		Message: []byte(fmt.Sprintf(`{"id":"%s", "itemid":"%s"}`, uuid.New().String(), "theme has changed, please reload")),
	})
}

func (s SSEService) ThemeRemove(context.Context) error {
	return nil
}
