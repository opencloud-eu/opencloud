package events

import (
	"context"
	"time"

	gateway "github.com/cs3org/go-cs3apis/cs3/gateway/v1beta1"
	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/services/activitylog/pkg/config"
	"github.com/opencloud-eu/reva/v2/pkg/events"
	"github.com/opencloud-eu/reva/v2/pkg/rgrpc/todo/pool"
)

// Option for the activitylog service
type Option func(*Options)

// Options for the activitylog service
type Options struct {
	Context             context.Context
	Logger              log.Logger
	ServiceAccount      config.ServiceAccount
	Stream              events.Stream
	RegisteredEvents    []events.Unmarshaller
	GatewaySelector     pool.Selectable[gateway.GatewayAPIClient]
	WriteBufferDuration time.Duration
	NumConsumers        int
}

func Context(ctx context.Context) Option {
	return func(o *Options) {
		o.Context = ctx
	}
}

// Logger configures a logger for the activitylog service
func Logger(log log.Logger) Option {
	return func(o *Options) {
		o.Logger = log
	}
}

// ServiceAccount configures a service account for the activitylog service
func ServiceAccount(sa config.ServiceAccount) Option {
	return func(o *Options) {
		o.ServiceAccount = sa
	}
}

// Stream configures an event stream for the clientlog service
func Stream(s events.Stream) Option {
	return func(o *Options) {
		o.Stream = s
	}
}

// RegisteredEvents registers the events the service should listen to
func RegisteredEvents(e []events.Unmarshaller) Option {
	return func(o *Options) {
		o.RegisteredEvents = e
	}
}

// GatewaySelector adds a grpc client selector for the gateway service
func GatewaySelector(gatewaySelector pool.Selectable[gateway.GatewayAPIClient]) Option {
	return func(o *Options) {
		o.GatewaySelector = gatewaySelector
	}
}

func NumConsumers(num int) Option {
	return func(o *Options) {
		o.NumConsumers = num
	}
}
