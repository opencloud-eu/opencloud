package service

import (
	gateway "github.com/cs3org/go-cs3apis/cs3/gateway/v1beta1"
	"github.com/opencloud-eu/reva/v2/pkg/rgrpc/todo/pool"
	"go.opentelemetry.io/otel/trace"

	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/services/search/pkg/config"
	"github.com/opencloud-eu/opencloud/services/search/pkg/metrics"
	"github.com/opencloud-eu/opencloud/services/search/pkg/search"
)

// Option defines a single option function.
type Option func(o *Options)

// Options defines the available options for this package.
type Options struct {
	Logger          log.Logger
	Config          *config.Config
	JWTSecret       string
	TracerProvider  trace.TracerProvider
	Metrics         *metrics.Metrics
	GatewaySelector *pool.Selector[gateway.GatewayAPIClient]
	Searcher        search.Searcher
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

// Metrics provides a function to set the Metrics option.
func Metrics(val *metrics.Metrics) Option {
	return func(o *Options) {
		if val != nil {
			o.Metrics = val
		}
	}
}

// GatewaySelector provides a function to set the GatewaySelector option.
func GatewaySelector(val *pool.Selector[gateway.GatewayAPIClient]) Option {
	return func(o *Options) {
		o.GatewaySelector = val
	}
}

// Searcher provides a function to set the Searcher option.
func Searcher(val search.Searcher) Option {
	return func(o *Options) {
		o.Searcher = val
	}
}
