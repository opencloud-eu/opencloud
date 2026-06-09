package http

import (
	"context"

	microstore "go-micro.dev/v4/store"
	"go.opentelemetry.io/otel/trace"

	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/services/collaboration/pkg/config"
	"github.com/opencloud-eu/opencloud/services/collaboration/pkg/connector"
	"github.com/opencloud-eu/opencloud/services/collaboration/pkg/font"
	"github.com/opencloud-eu/opencloud/services/collaboration/pkg/notification"
)

// Option defines a single option function.
type Option func(o *Options)

// Options define the available options for this package.
type Options struct {
	Adapter             *connector.HttpAdapter
	Logger              log.Logger
	Context             context.Context
	Config              *config.Config
	TracerProvider      trace.TracerProvider
	Store               microstore.Store
	FontService         font.Service
	NotificationService notification.Service
}

// newOptions initializes the available default options.
func newOptions(opts ...Option) Options {
	opt := Options{}

	for _, o := range opts {
		o(&opt)
	}

	return opt
}

// Adapter provides a function to set the Adapter option.
func Adapter(val *connector.HttpAdapter) Option {
	return func(o *Options) {
		o.Adapter = val
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

// TracerProvider provides a function to set the TracerProvider option
func TracerProvider(val trace.TracerProvider) Option {
	return func(o *Options) {
		o.TracerProvider = val
	}
}

// Store provides a function to set the Store option
func Store(val microstore.Store) Option {
	return func(o *Options) {
		o.Store = val
	}
}

// FontService provides a function to set the FontService option
func FontService(val font.Service) Option {
	return func(o *Options) {
		o.FontService = val
	}
}

// NotificationService provides a function to set the NotificationService option
func NotificationService(val notification.Service) Option {
	return func(o *Options) {
		o.NotificationService = val
	}
}
