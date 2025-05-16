package service

import (
	"time"

	gateway "github.com/cs3org/go-cs3apis/cs3/gateway/v1beta1"
	"github.com/go-chi/chi/v5"
	"github.com/opencloud-eu/opencloud/pkg/log"
	ehsvc "github.com/opencloud-eu/opencloud/protogen/gen/opencloud/services/eventhistory/v0"
	settingssvc "github.com/opencloud-eu/opencloud/protogen/gen/opencloud/services/settings/v0"
	"github.com/opencloud-eu/opencloud/services/activitylog/pkg/config"
	"github.com/opencloud-eu/reva/v2/pkg/events"
	"github.com/opencloud-eu/reva/v2/pkg/rgrpc/todo/pool"
	microstore "go-micro.dev/v4/store"
	"go.opentelemetry.io/otel/trace"
)

// Option for the activitylog service
type Option func(*Options)

// Options for the activitylog service
type Options struct {
	Logger              log.Logger
	Config              *config.Config
	TraceProvider       trace.TracerProvider
	Stream              events.Stream
	RegisteredEvents    []events.Unmarshaller
	Store               microstore.Store
	GatewaySelector     pool.Selectable[gateway.GatewayAPIClient]
	Mux                 *chi.Mux
	HistoryClient       ehsvc.EventHistoryService
	ValueClient         settingssvc.ValueService
	WriteBufferDuration time.Duration
}

// Logger configures a logger for the activitylog service
func Logger(log log.Logger) Option {
	return func(o *Options) {
		o.Logger = log
	}
}

// Config adds the config for the activitylog service
func Config(c *config.Config) Option {
	return func(o *Options) {
		o.Config = c
	}
}

// TraceProvider adds a tracer provider for the activitylog service
func TraceProvider(tp trace.TracerProvider) Option {
	return func(o *Options) {
		o.TraceProvider = tp
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

// Store configures the store to use
func Store(store microstore.Store) Option {
	return func(o *Options) {
		o.Store = store
	}
}

// GatewaySelector adds a grpc client selector for the gateway service
func GatewaySelector(gatewaySelector pool.Selectable[gateway.GatewayAPIClient]) Option {
	return func(o *Options) {
		o.GatewaySelector = gatewaySelector
	}
}

// Mux defines the muxer for the service
func Mux(m *chi.Mux) Option {
	return func(o *Options) {
		o.Mux = m
	}
}

// HistoryClient adds a grpc client for the eventhistory service
func HistoryClient(hc ehsvc.EventHistoryService) Option {
	return func(o *Options) {
		o.HistoryClient = hc
	}
}

// ValueClient adds a grpc client for the value service
func ValueClient(vs settingssvc.ValueService) Option {
	return func(o *Options) {
		o.ValueClient = vs
	}
}

// WriteBufferDuration sets the write buffer duration
func WriteBufferDuration(d time.Duration) Option {
	return func(o *Options) {
		o.WriteBufferDuration = d
	}
}
