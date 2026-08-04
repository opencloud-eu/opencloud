package http

import (
	"context"

	"github.com/opencloud-eu/opencloud/pkg/log"
	settingssvc "github.com/opencloud-eu/opencloud/protogen/gen/opencloud/services/settings/v0"
	"github.com/opencloud-eu/opencloud/services/activitylog/pkg/config"
	"github.com/spf13/pflag"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
)

// Option defines a single option function.
type Option func(o *Options)

// Options defines the available options for this package.
type Options struct {
	Name          string
	Namespace     string
	Logger        log.Logger
	Context       context.Context
	Config        *config.Config
	Flags         []pflag.Flag
	Service       ActivityLogService
	TraceProvider trace.TracerProvider
	ValueClient   settingssvc.ValueService
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

// Service provides a function to set the service option.
func Service(val ActivityLogService) Option {
	return func(o *Options) {
		o.Service = val
	}
}

// Flags provides a function to set the flags option.
func Flags(flags ...pflag.Flag) Option {
	return func(o *Options) {
		o.Flags = append(o.Flags, flags...)
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

// ValueClient adds a grpc client for the value service
func ValueClient(vs settingssvc.ValueService) Option {
	return func(o *Options) {
		o.ValueClient = vs
	}
}
