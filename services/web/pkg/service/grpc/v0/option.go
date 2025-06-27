package service

import (
	"go.opentelemetry.io/otel/trace"

	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/pkg/x/io/fsx"
	"github.com/opencloud-eu/opencloud/services/web/pkg/config"
	"github.com/opencloud-eu/opencloud/services/web/pkg/theme"
)

// Option defines a single option function.
type Option func(o *Options)

// Options defines the available options for this package.
type Options struct {
	Logger         log.Logger
	Config         *config.Config
	JWTSecret      string
	TracerProvider trace.TracerProvider
	ThemeService   *theme.Service
	ThemeFS        *fsx.FallbackFS
}

func newOptions(opts ...Option) Options {
	opt := Options{}

	for _, o := range opts {
		o(&opt)
	}

	return opt
}

// Logger provides a function to set the Logger option.
func Logger(val log.Logger) Option {
	return func(o *Options) {
		o.Logger = val
	}
}

// Config provides a function to set the Config option.
func Config(val *config.Config) Option {
	return func(o *Options) {
		o.Config = val
	}
}

// JWTSecret provides a function to set the Config option.
func JWTSecret(val string) Option {
	return func(o *Options) {
		o.JWTSecret = val
	}
}

// TracerProvider provides a function to set the TracerProvider option
func TracerProvider(val trace.TracerProvider) Option {
	return func(o *Options) {
		o.TracerProvider = val
	}
}

// ThemeFS provides a function to set the themeFS option.
func ThemeFS(val *fsx.FallbackFS) Option {
	return func(o *Options) {
		o.ThemeFS = val
	}
}
