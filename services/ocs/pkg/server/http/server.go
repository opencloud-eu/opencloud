package http

import (
	"fmt"

	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/opencloud-eu/opencloud/pkg/cors"
	"github.com/opencloud-eu/opencloud/pkg/middleware"
	"github.com/opencloud-eu/opencloud/pkg/service/http"
	"github.com/opencloud-eu/opencloud/pkg/version"
	svc "github.com/opencloud-eu/opencloud/services/ocs/pkg/service/v0"
	"github.com/opencloud-eu/reva/v2/pkg/store"
	"go-micro.dev/v4"
	microstore "go-micro.dev/v4/store"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

// Server initializes the http service and server.
func Server(opts ...Option) (http.Service, error) {
	options := newOptions(opts...)

	service, err := http.NewService(
		http.TLSConfig(options.Config.HTTP.TLS),
		http.Logger(options.Logger),
		http.Name(options.Config.Service.Name),
		http.Version(version.GetString()),
		http.Namespace(options.Config.HTTP.Namespace),
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

	signingKeyStore := store.Create(
		store.Store(options.Config.SigningKeys.Store),
		store.TTL(options.Config.SigningKeys.TTL),
		microstore.Nodes(options.Config.SigningKeys.Nodes...),
		microstore.Database("proxy"),
		microstore.Table("signing-keys"),
		store.Authentication(options.Config.SigningKeys.AuthUsername, options.Config.SigningKeys.AuthPassword),
		store.TLSEnabled(options.Config.SigningKeys.EnableTLS),
		store.TLSInsecure(options.Config.SigningKeys.TLSInsecure),
		store.TLSRootCA(options.Config.SigningKeys.TLSRootCACertificate),
	)

	handle := svc.NewService(
		svc.Logger(options.Logger),
		svc.Config(options.Config),
		svc.Middleware(
			chimiddleware.RealIP,
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
			middleware.Logger(options.Logger),
			middleware.TraceContext,
			otelhttp.NewMiddleware(options.Config.Service.Name, otelhttp.WithTracerProvider(options.TraceProvider)),
		),
		svc.Store(signingKeyStore),
	)

	if err := micro.RegisterHandler(service.Server(), handle); err != nil {
		return http.Service{}, err
	}

	return service, nil
}
