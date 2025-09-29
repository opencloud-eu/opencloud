package debug

import (
	"net/http"

	"github.com/opencloud-eu/opencloud/pkg/checks"
	"github.com/opencloud-eu/opencloud/pkg/handlers"
	"github.com/opencloud-eu/opencloud/pkg/service/debug"
	"github.com/opencloud-eu/opencloud/pkg/version"
)

// Server initializes the debug service and server.
func Server(opts ...Option) (*http.Server, error) {
	options := newOptions(opts...)

	checkHandler := handlers.NewCheckHandler(
		handlers.NewCheckHandlerConfiguration().
			WithLogger(options.Logger).
			WithCheck("web reachability", checks.NewHTTPCheck(options.Config.HTTP.Addr)).
			WithCheck("grpc reachability", checks.NewGRPCCheck(options.Config.GRPC.Addr)),
	)

	return debug.NewService(
		debug.Logger(options.Logger),
		debug.Name(options.Config.Service.Name),
		debug.Version(version.GetString()),
		debug.Address(options.Config.Debug.Addr),
		debug.Token(options.Config.Debug.Token),
		debug.Pprof(options.Config.Debug.Pprof),
		debug.Zpages(options.Config.Debug.Zpages),
		debug.Health(checkHandler),
		debug.Ready(checkHandler),
		//debug.CorsAllowedOrigins(options.Config.HTTP.CORS.AllowedOrigins),
		//debug.CorsAllowedMethods(options.Config.HTTP.CORS.AllowedMethods),
		//debug.CorsAllowedHeaders(options.Config.HTTP.CORS.AllowedHeaders),
		//debug.CorsAllowCredentials(options.Config.HTTP.CORS.AllowCredentials),
	), nil
}
