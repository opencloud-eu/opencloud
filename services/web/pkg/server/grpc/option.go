package grpc

import (
	"context"

	"go.opentelemetry.io/otel/trace"

	"github.com/opencloud-eu/opencloud/pkg/log"
	svc "github.com/opencloud-eu/opencloud/services/search/pkg/service/grpc/v0"
	"github.com/opencloud-eu/opencloud/services/web/pkg/config"
	"github.com/opencloud-eu/opencloud/services/web/pkg/metrics"
)

// Option defines a single option function.
type Option func(o *Options)

// Options defines the available options for this package.
type Options struct {
	Name          string
	Logger        log.Logger
	Context       context.Context
	Config        *config.Config
	Metrics       *metrics.Metrics
	Handler       *svc.Service
	JWTSecret     string
	TraceProvider trace.TracerProvider
}

// newOptions initializes the available default options.
func newOptions(opts ...Option) Options {
	opt := Options{}

	for _, o := range opts {
		o(&opt)
	}

	return opt
}

// Name provides a name for the service.
func Name(val string) Option {
	return func(o *Options) {
		o.Name = val
	}
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

// Metrics provides a function to set the metrics option.
func Metrics(val *metrics.Metrics) Option {
	return func(o *Options) {
		o.Metrics = val
	}
}

// Handler provides a function to set the handler option.
func Handler(val *svc.Service) Option {
	return func(o *Options) {
		o.Handler = val
	}
}

// JWTSecret provides a function to set the Config option.
func JWTSecret(val string) Option {
	return func(o *Options) {
		o.JWTSecret = val
	}
}

// TraceProvider provides a function to set the trace provider option.
func TraceProvider(val trace.TracerProvider) Option {
	return func(o *Options) {
		o.TraceProvider = val
	}
}
