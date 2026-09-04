package http

import (
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/opencloud-eu/opencloud/pkg/account"
	"github.com/opencloud-eu/opencloud/pkg/cors"
	"github.com/opencloud-eu/opencloud/pkg/middleware"
	ohttp "github.com/opencloud-eu/opencloud/pkg/service/http"
	"github.com/opencloud-eu/opencloud/pkg/tracing"
	"github.com/opencloud-eu/opencloud/pkg/version"
	settingssvc "github.com/opencloud-eu/opencloud/protogen/gen/opencloud/services/settings/v0"
	"github.com/riandyrn/otelchi"
	"go-micro.dev/v4"
)

// Server initializes the http service and server.
func Server(opts ...Option) (ohttp.Service, error) {
	options := newOptions(opts...)

	service, err := ohttp.NewService(
		ohttp.TLSConfig(options.Config.HTTP.TLS),
		ohttp.Logger(options.Logger),
		ohttp.Name(options.Name),
		ohttp.Version(version.GetString()),
		ohttp.Address(options.Config.HTTP.Addr),
		ohttp.Namespace(options.Config.HTTP.Namespace),
		ohttp.Context(options.Context),
		ohttp.Flags(options.Flags...),
	)
	if err != nil {
		options.Logger.Error().
			Err(err).
			Msg("Error initializing http service")
		return ohttp.Service{}, fmt.Errorf("could not initialize http service: %w", err)
	}

	handle := options.ServiceHandler

	mux := chi.NewMux()

	mux.Use(chimiddleware.ClientIPFromRemoteAddr)
	mux.Use(chimiddleware.RequestID)
	mux.Use(middleware.NoCache)
	mux.Use(middleware.Cors(
		cors.Logger(options.Logger),
		cors.AllowedOrigins(options.Config.HTTP.CORS.AllowedOrigins),
		cors.AllowedMethods(options.Config.HTTP.CORS.AllowedMethods),
		cors.AllowedHeaders(options.Config.HTTP.CORS.AllowedHeaders),
		cors.AllowCredentials(options.Config.HTTP.CORS.AllowCredentials),
	))
	mux.Use(middleware.ExtractAccountUUID(
		account.Logger(options.Logger),
		account.JWTSecret(options.Config.TokenManager.JWTSecret)),
	)

	mux.Use(middleware.Version(
		options.Name,
		version.GetString(),
	))

	mux.Use(middleware.Logger(
		options.Logger,
	))

	mux.Use(
		otelchi.Middleware(
			options.Name,
			otelchi.WithChiRoutes(mux),
			otelchi.WithTracerProvider(options.TraceProvider),
			otelchi.WithPropagators(tracing.GetPropagator()),
		),
	)

	mux.Route(options.Config.HTTP.Root, func(r chi.Router) {
		settingssvc.RegisterBundleServiceWeb(r, handle)
		settingssvc.RegisterValueServiceWeb(r, handle)
		settingssvc.RegisterRoleServiceWeb(r, handle)
		settingssvc.RegisterPermissionServiceWeb(r, handle)
	})

	_ = chi.Walk(mux, func(method string, route string, handler http.Handler, middlewares ...func(http.Handler) http.Handler) error {
		options.Logger.Debug().Str("method", method).Str("route", route).Int("middlewares", len(middlewares)).Msg("serving endpoint")
		return nil
	})

	micro.RegisterHandler(service.Server(), mux)

	return service, nil
}
