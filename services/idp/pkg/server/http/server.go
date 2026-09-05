package http

import (
	"fmt"
	"os"

	chimiddleware "github.com/go-chi/chi/v5/middleware"
	pkgcrypto "github.com/opencloud-eu/opencloud/pkg/crypto"
	"github.com/opencloud-eu/opencloud/pkg/middleware"
	"github.com/opencloud-eu/opencloud/pkg/service/http"
	"github.com/opencloud-eu/opencloud/pkg/shared"
	"github.com/opencloud-eu/opencloud/pkg/version"
	svc "github.com/opencloud-eu/opencloud/services/idp/pkg/service/v0"
	"go-micro.dev/v4"
)

// Server initializes the http service and server.
func Server(opts ...Option) (http.Service, error) {
	options := newOptions(opts...)

	if options.Config.HTTP.TLS {
		_, certErr := os.Stat(options.Config.HTTP.TLSCert)
		_, keyErr := os.Stat(options.Config.HTTP.TLSKey)

		if os.IsNotExist(certErr) || os.IsNotExist(keyErr) {
			options.Logger.Info().Msgf("Generating certs")
			if err := pkgcrypto.GenCert(options.Config.HTTP.TLSCert, options.Config.HTTP.TLSKey, options.Logger); err != nil {
				return http.Service{}, fmt.Errorf("could not setup TLS: %w", err)
			}
		}
	}

	service, err := http.NewService(
		http.Logger(options.Logger),
		http.Namespace(options.Config.HTTP.Namespace),
		http.Name(options.Config.Service.Name),
		http.Version(version.GetString()),
		http.Address(options.Config.HTTP.Addr),
		http.Context(options.Context),
		http.Flags(options.Flags...),
		http.TLSConfig(shared.HTTPServiceTLS{
			Enabled: options.Config.HTTP.TLS,
			Cert:    options.Config.HTTP.TLSCert,
			Key:     options.Config.HTTP.TLSKey,
		}),
		http.TraceProvider(options.TraceProvider),
	)
	if err != nil {
		options.Logger.Error().
			Err(err).
			Msg("Error initializing http service")
		return http.Service{}, fmt.Errorf("could not initialize http service: %w", err)
	}

	handle := svc.NewService(
		svc.Logger(options.Logger),
		svc.Config(options.Config),
		svc.Middleware(
			chimiddleware.RealIP,
			chimiddleware.RequestID,
			middleware.TraceContext,
			middleware.NoCache,
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

	{
		handle = svc.NewInstrument(handle, options.Metrics)
		handle = svc.NewLoggingHandler(handle, options.Logger)
	}

	if err := micro.RegisterHandler(service.Server(), handle); err != nil {
		return http.Service{}, err
	}

	return service, nil
}
