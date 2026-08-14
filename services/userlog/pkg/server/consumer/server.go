package consumer

import (
	"errors"

	"github.com/opencloud-eu/opencloud/pkg/runner"
	"github.com/opencloud-eu/reva/v2/pkg/events"
)

// Server starts the event consumer for the userlog service
func Server(opts ...Option) (*runner.Runner, error) {
	options := newOptions(opts...)

	if options.UserlogService == nil {
		return nil, errors.New("need non nil userlog service to consume events")
	}

	ch, err := events.Consume(options.Stream, "userlog", options.RegisteredEvents...)
	if err != nil {
		return nil, err
	}

	options.UserlogService.MemorizeEvents(ch)

	return runner.New(options.Config.Service.Name+".consumer", func() error {
		<-options.Context.Done()
		return nil
	}, func() {}), nil
}
