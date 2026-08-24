package http

import (
	"reflect"

	gateway "github.com/cs3org/go-cs3apis/cs3/gateway/v1beta1"
	"github.com/opencloud-eu/opencloud/pkg/log"
	settingssvc "github.com/opencloud-eu/opencloud/protogen/gen/opencloud/services/settings/v0"
	"github.com/opencloud-eu/opencloud/services/userlog/pkg/config"
	"github.com/opencloud-eu/opencloud/services/userlog/pkg/service"
	"github.com/opencloud-eu/reva/v2/pkg/events"
	"github.com/opencloud-eu/reva/v2/pkg/rgrpc/todo/pool"
	"go.opentelemetry.io/otel/trace"
)

// Option for the http service
type Option func(*Options)

// Options for the http service
type Options struct {
	UserlogService   *service.UserlogService
	Logger           log.Logger
	Config           *config.Config
	GatewaySelector  pool.Selectable[gateway.GatewayAPIClient]
	ValueClient      settingssvc.ValueService
	RegisteredEvents []events.Unmarshaller
	TraceProvider    trace.TracerProvider
}

// New creates a new http service
func New(ul *service.UserlogService, opts ...Option) (*Service, error) {
	o := &Options{}
	for _, opt := range opts {
		opt(o)
	}

	registeredEvents := make(map[string]events.Unmarshaller)
	for _, e := range o.RegisteredEvents {
		typ := reflect.TypeOf(e)
		registeredEvents[typ.String()] = e
	}

	return &Service{
		log:              o.Logger,
		cfg:              o.Config,
		userlog:          ul,
		gatewaySelector:  o.GatewaySelector,
		valueClient:      o.ValueClient,
		registeredEvents: registeredEvents,
		tp:               o.TraceProvider,
		tracer:           o.TraceProvider.Tracer("github.com/opencloud-eu/opencloud/services/userlog/pkg/service/http"),
	}, nil
}

// UserlogService provides a function to set the userlog service
func UserlogService(ul *service.UserlogService) Option {
	return func(o *Options) {
		o.UserlogService = ul
	}
}

// Logger configures a logger for the http service
func Logger(log log.Logger) Option {
	return func(o *Options) {
		o.Logger = log
	}
}

// Config adds the config for the http service
func Config(c *config.Config) Option {
	return func(o *Options) {
		o.Config = c
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

// TraceProvider adds a tracer provider for the http service
func TraceProvider(tp trace.TracerProvider) Option {
	return func(o *Options) {
		o.TraceProvider = tp
	}
}
