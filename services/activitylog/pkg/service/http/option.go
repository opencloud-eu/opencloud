package http

import (
	gateway "github.com/cs3org/go-cs3apis/cs3/gateway/v1beta1"
	"github.com/opencloud-eu/opencloud/pkg/log"
	ehsvc "github.com/opencloud-eu/opencloud/protogen/gen/opencloud/services/eventhistory/v0"
	"github.com/opencloud-eu/reva/v2/pkg/events"
	"github.com/opencloud-eu/reva/v2/pkg/rgrpc/todo/pool"
)

// Option defines a single option function.
type Option func(o *Options)

// Options defines the available options for this package.
type Options struct {
	Logger           log.Logger
	RegisteredEvents []events.Unmarshaller
	GatewaySelector  pool.Selectable[gateway.GatewayAPIClient]
	HistoryClient    ehsvc.EventHistoryService
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

// HistoryClient adds a grpc client for the eventhistory service
func HistoryClient(hc ehsvc.EventHistoryService) Option {
	return func(o *Options) {
		o.HistoryClient = hc
	}
}
