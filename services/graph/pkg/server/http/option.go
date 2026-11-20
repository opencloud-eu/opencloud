package http

import (
	"context"

	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/services/graph/pkg/config"
	"github.com/opencloud-eu/opencloud/services/graph/pkg/metrics"
	svc "github.com/opencloud-eu/opencloud/services/graph/pkg/service/v0"

	"github.com/nats-io/nats.go/jetstream"
	"github.com/spf13/pflag"
	"go.opentelemetry.io/otel/trace"
)

// Option defines a single option function.
type Option func(o *Options)

// Options defines the available options for this package.
type Options struct {
	UserProfilePhotoService svc.UsersUserProfilePhotoProvider
	Logger                  log.Logger
	Context                 context.Context
	Config                  *config.Config
	Metrics                 *metrics.Metrics
	Flags                   []pflag.Flag
	Namespace               string
	TraceProvider           trace.TracerProvider
	NatsKeyValue            jetstream.KeyValue
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

// Flags provides a function to set the flags option.
func Flags(flags ...pflag.Flag) Option {
	return func(o *Options) {
		o.Flags = append(o.Flags, flags...)
	}
}

// Namespace provides a function to set the Namespace option.
func Namespace(val string) Option {
	return func(o *Options) {
		o.Namespace = val
	}
}

// TraceProvider provides a function to set the TraceProvider option.
func TraceProvider(val trace.TracerProvider) Option {
	return func(o *Options) {
		o.TraceProvider = val
	}
}

// NatsKeyValue provides a function to set the NatsKeyValue option.
func NatsKeyValue(val jetstream.KeyValue) Option {
	return func(o *Options) {
		o.NatsKeyValue = val
	}
}

// UserProfilePhotoService supplies the shared photo service to the HTTP API.
func UserProfilePhotoService(provider svc.UsersUserProfilePhotoProvider) Option {
	return func(o *Options) { o.UserProfilePhotoService = provider }
}
