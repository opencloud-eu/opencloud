package service

import (
	gateway "github.com/cs3org/go-cs3apis/cs3/gateway/v1beta1"
	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/services/clientlog/pkg/config"
	"github.com/opencloud-eu/reva/v3/pkg/events"
	"github.com/opencloud-eu/reva/v3/pkg/rgrpc/todo/pool"
	"go.opentelemetry.io/otel/trace"
)

// Option for the clientlog service
type Option func(*Options)

// Options for the clientlog service
type Options struct {
	Logger           log.Logger
	Stream           events.Stream
	Config           *config.Config
	GatewaySelector  pool.Selectable[gateway.GatewayAPIClient]
	RegisteredEvents []events.Unmarshaller
	TraceProvider    trace.TracerProvider
}

// Logger configures a logger for the clientlog service
func Logger(log log.Logger) Option {
	return func(o *Options) {
		o.Logger = log
	}
}

// Stream configures an event stream for the clientlog service
func Stream(s events.Stream) Option {
	return func(o *Options) {
		o.Stream = s
	}
}

// Config adds the config for the clientlog service
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

// RegisteredEvents registers the events the service should listen to
func RegisteredEvents(e []events.Unmarshaller) Option {
	return func(o *Options) {
		o.RegisteredEvents = e
	}
}

// TraceProvider adds a tracer provider for the clientlog service
func TraceProvider(tp trace.TracerProvider) Option {
	return func(o *Options) {
		o.TraceProvider = tp
	}
}
