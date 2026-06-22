package debug

import (
	"net/http"

	"github.com/opencloud-eu/opencloud/pkg/checks"
	"github.com/opencloud-eu/opencloud/pkg/handlers"
	"github.com/opencloud-eu/opencloud/pkg/nats"
	"github.com/opencloud-eu/opencloud/pkg/service/debug"
	"github.com/opencloud-eu/opencloud/pkg/version"
)

// Server initializes the debug service and server.
func Server(opts ...Option) (*http.Server, error) {
	options := newOptions(opts...)

	secureOption := nats.Secure(
		options.Config.Events.EnableTLS,
		options.Config.Events.TLSInsecure,
		options.Config.Events.TLSRootCaCertPath,
	)
	readyHandler := handlers.NewCheckHandler(handlers.NewCheckHandlerConfiguration().
		WithLogger(options.Logger).
		WithCheck("nats reachability", checks.NewNatsCheck(options.Config.Events.Addr, secureOption)).
		WithCheck("grpc reachability", checks.NewGRPCCheck(options.Config.GRPC.Addr)),
	)

	return debug.NewService(
		debug.Logger(options.Logger),
		debug.Context(options.Context),
		debug.Name(options.Config.Service.Name),
		debug.Version(version.GetString()),
		debug.Address(options.Config.Debug.Addr),
		debug.Token(options.Config.Debug.Token),
		debug.Pprof(options.Config.Debug.Pprof),
		debug.Zpages(options.Config.Debug.Zpages),
		debug.Ready(readyHandler),
		//debug.CorsAllowedOrigins(options.Config.HTTP.CORS.AllowedOrigins),
		//debug.CorsAllowedMethods(options.Config.HTTP.CORS.AllowedMethods),
		//debug.CorsAllowedHeaders(options.Config.HTTP.CORS.AllowedHeaders),
		//debug.CorsAllowCredentials(options.Config.HTTP.CORS.AllowCredentials),
	), nil
}
