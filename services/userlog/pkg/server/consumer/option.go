package consumer

import (
	"context"

	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/services/userlog/pkg/config"
	svc "github.com/opencloud-eu/opencloud/services/userlog/pkg/service"
	"github.com/opencloud-eu/reva/v2/pkg/events"
)

// Option defines a single option function.
type Option func(o *Options)

// Options defines the available options for this package.
type Options struct {
	Logger           log.Logger
	Context          context.Context
	Config           *config.Config
	Stream           events.Stream
	UserlogService   *svc.UserlogService
	RegisteredEvents []events.Unmarshaller
}

// newOptions initializes the available default options.
func newOptions(opts ...Option) Options {
	opt := Options{}

	for _, o := range opts {
		o(&opt)
	}

	return opt
}

// Logger provides a function to set the logger option.
func Logger(val log.Logger) Option {
	return func(o *Options) {
		o.Logger = val
	}
}

// Context provides a function to set the context option.
func Context(val context.Context) Option {
	return func(o *Options) {
		o.Context = val
	}
}

// Config provides a function to set the config option.
func Config(val *config.Config) Option {
	return func(o *Options) {
		o.Config = val
	}
}

// Stream provides a function to configure the stream
func Stream(stream events.Stream) Option {
	return func(o *Options) {
		o.Stream = stream
	}
}

// UserlogService provides a function to set the userlog service
func UserlogService(s *svc.UserlogService) Option {
	return func(o *Options) {
		o.UserlogService = s
	}
}

// RegisteredEvents provides a function to register events
func RegisteredEvents(evs []events.Unmarshaller) Option {
	return func(o *Options) {
		o.RegisteredEvents = evs
	}
}
