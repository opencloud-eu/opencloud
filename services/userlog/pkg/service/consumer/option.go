package consumer

import (
	"context"

	gateway "github.com/cs3org/go-cs3apis/cs3/gateway/v1beta1"
	"github.com/opencloud-eu/opencloud/pkg/log"
	settingssvc "github.com/opencloud-eu/opencloud/protogen/gen/opencloud/services/settings/v0"
	"github.com/opencloud-eu/opencloud/services/userlog/pkg/config"
	"github.com/opencloud-eu/opencloud/services/userlog/pkg/service"
	"github.com/opencloud-eu/reva/v2/pkg/events"
	"github.com/opencloud-eu/reva/v2/pkg/rgrpc/todo/pool"
)

// Option for the consumer service
type Option func(*Options)

// Options for the consumer service
type Options struct {
	UserlogService   *service.UserlogService
	Context          context.Context
	Logger           log.Logger
	Config           *config.Config
	Stream           events.Stream
	GatewaySelector  pool.Selectable[gateway.GatewayAPIClient]
	ValueClient      settingssvc.ValueService
	RegisteredEvents []events.Unmarshaller
}

// New creates a new consumer service
func New(ul *service.UserlogService, stream events.Stream, opts ...Option) (*Service, error) {
	o := &Options{}
	for _, opt := range opts {
		opt(o)
	}

	if o.Context == nil {
		o.Context = context.Background()
	}

	return &Service{
		ctx:              o.Context,
		log:              o.Logger,
		cfg:              o.Config,
		userlog:          ul,
		stream:           stream,
		gatewaySelector:  o.GatewaySelector,
		valueClient:      o.ValueClient,
		registeredEvents: o.RegisteredEvents,
		filter:           newUserlogFilter(o.Logger, o.ValueClient),
		stopCh:           make(chan struct{}, 1),
	}, nil
}

// UserlogService provides a function to set the userlog service
func UserlogService(ul *service.UserlogService) Option {
	return func(o *Options) {
		o.UserlogService = ul
	}
}

// Context provides a function to set the context option.
func Context(val context.Context) Option {
	return func(o *Options) {
		o.Context = val
	}
}

// Logger configures a logger for the consumer service
func Logger(log log.Logger) Option {
	return func(o *Options) {
		o.Logger = log
	}
}

// Config adds the config for the consumer service
func Config(c *config.Config) Option {
	return func(o *Options) {
		o.Config = c
	}
}

// Stream configures an event stream for the consumer service
func Stream(s events.Stream) Option {
	return func(o *Options) {
		o.Stream = s
	}
}

// GatewaySelector adds a grpc client selector for the gateway service
func GatewaySelector(gatewaySelector pool.Selectable[gateway.GatewayAPIClient]) Option {
	return func(o *Options) {
		o.GatewaySelector = gatewaySelector
	}
}

// ValueClient adds a grpc client for the value service
func ValueClient(vs settingssvc.ValueService) Option {
	return func(o *Options) {
		o.ValueClient = vs
	}
}

// RegisteredEvents registers the events the service should listen to
func RegisteredEvents(e []events.Unmarshaller) Option {
	return func(o *Options) {
		o.RegisteredEvents = e
	}
}
