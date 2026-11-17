package service

import (
	"context"

	gateway "github.com/cs3org/go-cs3apis/cs3/gateway/v1beta1"
	"github.com/go-chi/chi/v5"
	"github.com/opencloud-eu/opencloud/pkg/log"
	settingssvc "github.com/opencloud-eu/opencloud/protogen/gen/opencloud/services/settings/v0"
	"github.com/opencloud-eu/opencloud/services/auth-app/pkg/config"
	"github.com/opencloud-eu/reva/v3/pkg/rgrpc/todo/pool"
	"go.opentelemetry.io/otel/trace"
)

// Option defines a single option function.
type Option func(o *Options)

// Options defines the available options for this package.
type Options struct {
	Logger          log.Logger
	Context         context.Context
	Config          *config.Config
	GatewaySelector pool.Selectable[gateway.GatewayAPIClient]
	Mux             *chi.Mux
	TracerProvider  trace.TracerProvider
	RoleClient      settingssvc.RoleService
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

// GatewaySelector provides a function to configure the gateway client selector
func GatewaySelector(gatewaySelector pool.Selectable[gateway.GatewayAPIClient]) Option {
	return func(o *Options) {
		o.GatewaySelector = gatewaySelector
	}
}

// TraceProvider provides a function to set the TracerProvider option
func TraceProvider(val trace.TracerProvider) Option {
	return func(o *Options) {
		o.TracerProvider = val
	}
}

// Mux defines the muxer for the userlog service
func Mux(m *chi.Mux) Option {
	return func(o *Options) {
		o.Mux = m
	}
}

// RoleClient adds a grpc client for the role service
func RoleClient(rs settingssvc.RoleService) Option {
	return func(o *Options) {
		o.RoleClient = rs
	}
}
