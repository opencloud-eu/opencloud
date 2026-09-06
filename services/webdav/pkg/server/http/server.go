package http

import (
	"fmt"

	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/opencloud-eu/opencloud/pkg/cors"
	"github.com/opencloud-eu/opencloud/pkg/middleware"
	"github.com/opencloud-eu/opencloud/pkg/service/http"
	"github.com/opencloud-eu/opencloud/pkg/version"
	svc "github.com/opencloud-eu/opencloud/services/webdav/pkg/service/v0"
	"go-micro.dev/v4"
)

// Server initializes the http service and server.
func Server(opts ...Option) (http.Service, error) {
	options := newOptions(opts...)

	service, err := http.NewService(
		http.TLSConfig(options.Config.HTTP.TLS),
		http.Logger(options.Logger),
		http.Namespace(options.Config.HTTP.Namespace),
		http.Name(options.Config.Service.Name),
		http.Version(version.GetString()),
		http.Address(options.Config.HTTP.Addr),
		http.Context(options.Context),
		http.Flags(options.Flags...),
		http.TraceProvider(options.TraceProvider),
	)
	if err != nil {
		options.Logger.Error().
			Err(err).
			Msg("Error initializing http service")
		return http.Service{}, fmt.Errorf("could not initialize http service: %w", err)
	}

	handle, err := svc.NewService(
		svc.Logger(options.Logger),
		svc.Config(options.Config),
		svc.Middleware(
			chimiddleware.ClientIPFromRemoteAddr,
			chimiddleware.RequestID,
			middleware.NoCache,
			middleware.Cors(
				cors.Logger(options.Logger),
				cors.AllowedOrigins(options.Config.HTTP.CORS.AllowedOrigins),
				cors.AllowedMethods(options.Config.HTTP.CORS.AllowedMethods),
				cors.AllowedHeaders(options.Config.HTTP.CORS.AllowedHeaders),
				cors.AllowCredentials(options.Config.HTTP.CORS.AllowCredentials),
			),
			middleware.Version(
				options.Config.Service.Name,
				version.GetString(),
			),
			middleware.Logger(
				options.Logger,
			),
		),
		svc.TraceProvider(options.TraceProvider),
	)
	if err != nil {
		return http.Service{}, err
	}

	{
		handle = svc.NewInstrument(handle, options.Metrics)
		handle = svc.NewLogging(handle, options.Logger)
	}

	if err := micro.RegisterHandler(service.Server(), handle); err != nil {
		return http.Service{}, err
	}

	return service, nil
}
