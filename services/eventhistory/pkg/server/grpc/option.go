package grpc

import (
	"context"

	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/services/eventhistory/pkg/config"
	"github.com/opencloud-eu/opencloud/services/eventhistory/pkg/metrics"
	"go.opentelemetry.io/otel/trace"
)

// Option defines a single option function.
type Option func(o *Options)

// Options defines the available options for this package.
type Options struct {
	Name          string
	Address       string
	Logger        log.Logger
	Context       context.Context
	Config        *config.Config
	Metrics       *metrics.Metrics
	Namespace     string
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

// Logger provides a function to set the logger option.
func Logger(val log.Logger) Option {
	return func(o *Options) {
		o.Logger = val
	}
}

// Name provides a name for the service.
func Name(val string) Option {
	return func(o *Options) {
		o.Name = val
	}
}

// Address provides an address for the service.
func Address(val string) Option {
	return func(o *Options) {
		o.Address = val
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

// Namespace provides a function to set the namespace option.
func Namespace(val string) Option {
	return func(o *Options) {
		o.Namespace = val
	}
}

// TraceProvider provides a function to configure the trace provider
func TraceProvider(traceProvider trace.TracerProvider) Option {
	return func(o *Options) {
		if traceProvider != nil {
			o.TraceProvider = traceProvider
		} else {
			o.TraceProvider = trace.NewNoopTracerProvider()
		}
	}
}
