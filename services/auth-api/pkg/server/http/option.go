package http

import (
	"context"

	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/services/auth-api/pkg/config"
	"github.com/opencloud-eu/opencloud/services/auth-api/pkg/metrics"
	"github.com/urfave/cli/v2"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
)

// Option defines a single option function.
type Option func(o *Options)

// Options defines the available options for this package.
type Options struct {
	Namespace     string
	Logger        log.Logger
	Context       context.Context
	Config        *config.Config
	Metrics       *metrics.Metrics
	Flags         []cli.Flag
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

// Namespace provides a function to set the Namespace option.
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
			o.TraceProvider = noop.NewTracerProvider()
		}
	}
}
