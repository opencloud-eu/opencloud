package svc

import (
	"io/fs"
	"net/http"

	gateway "github.com/cs3org/go-cs3apis/cs3/gateway/v1beta1"
	"github.com/opencloud-eu/reva/v2/pkg/rgrpc/todo/pool"
	"go.opentelemetry.io/otel/trace"

	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/pkg/x/io/fsx"
	"github.com/opencloud-eu/opencloud/services/web/pkg/announcement"
	"github.com/opencloud-eu/opencloud/services/web/pkg/config"
)

// Option defines a single option function.
type Option func(o *Options)

// Options define the available options for this package.
type Options struct {
	Logger            log.Logger
	Config            *config.Config
	Middleware        []func(http.Handler) http.Handler
	GatewaySelector   pool.Selectable[gateway.GatewayAPIClient]
	TraceProvider     trace.TracerProvider
	AppsHTTPEndpoint  string
	CoreFS            fs.FS
	AppFS             fs.FS
	ThemeFS           *fsx.FallbackFS
	AnnouncementStore *announcement.Store
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

// Config provides a function to set the config option.
func Config(val *config.Config) Option {
	return func(o *Options) {
		o.Config = val
	}
}

// Middleware provides a function to set the middleware option.
func Middleware(val ...func(http.Handler) http.Handler) Option {
	return func(o *Options) {
		o.Middleware = val
	}
}

// GatewaySelector provides a function to set the gatewaySelector option.
func GatewaySelector(gatewaySelector pool.Selectable[gateway.GatewayAPIClient]) Option {
	return func(o *Options) {
		o.GatewaySelector = gatewaySelector
	}
}

// TraceProvider provides a function to set the traceProvider option.
func TraceProvider(val trace.TracerProvider) Option {
	return func(o *Options) {
		o.TraceProvider = val
	}
}

// AppFS provides a function to set the appFS option.
func AppFS(val fs.FS) Option {
	return func(o *Options) {
		o.AppFS = val
	}
}

// ThemeFS provides a function to set the themeFS option.
func ThemeFS(val *fsx.FallbackFS) Option {
	return func(o *Options) {
		o.ThemeFS = val
	}
}

// AnnouncementStore provides a function to set the announcement store option.
func AnnouncementStore(val *announcement.Store) Option {
	return func(o *Options) {
		o.AnnouncementStore = val
	}
}

// AppsHTTPEndpoint provides a function to set the appsHTTPEndpoint option.
func AppsHTTPEndpoint(val string) Option {
	return func(o *Options) {
		o.AppsHTTPEndpoint = val
	}
}

// CoreFS provides a function to set the coreFS option.
func CoreFS(val fs.FS) Option {
	return func(o *Options) {
		o.CoreFS = val
	}
}
