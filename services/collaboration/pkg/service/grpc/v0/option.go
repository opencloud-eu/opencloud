package service

import (
	gatewayv1beta1 "github.com/cs3org/go-cs3apis/cs3/gateway/v1beta1"
	"github.com/opencloud-eu/reva/v2/pkg/rgrpc/todo/pool"
	microstore "go-micro.dev/v4/store"

	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/services/collaboration/pkg/config"
	"github.com/opencloud-eu/opencloud/services/collaboration/pkg/helpers"
)

// Option defines a single option function.
type Option func(o *Options)

// Options defines the available options for this package.
type Options struct {
	Logger          log.Logger
	Config          *config.Config
	AppURLs         *helpers.AppURLs
	GatewaySelector pool.Selectable[gatewayv1beta1.GatewayAPIClient]
	Store           microstore.Store
}

// newOptions initializes the available default options.
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

// AppURLs provides a function to set the AppURLs option.
func AppURLs(val *helpers.AppURLs) Option {
	return func(o *Options) {
		o.AppURLs = val
	}
}

// GatewaySelector provides a function to set the GatewaySelector option.
func GatewaySelector(val pool.Selectable[gatewayv1beta1.GatewayAPIClient]) Option {
	return func(o *Options) {
		o.GatewaySelector = val
	}
}

// Store proivdes a function to set the store
func Store(val microstore.Store) Option {
	return func(o *Options) {
		o.Store = val
	}
}
