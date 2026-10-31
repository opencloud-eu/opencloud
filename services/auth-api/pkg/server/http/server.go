package http

import (
	"context"
	"fmt"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/opencloud-eu/opencloud/pkg/log"
	opencloudmiddleware "github.com/opencloud-eu/opencloud/pkg/middleware"
	"github.com/opencloud-eu/opencloud/pkg/service/http"
	"github.com/opencloud-eu/opencloud/pkg/version"
	"github.com/opencloud-eu/opencloud/services/auth-api/pkg/config"
	"github.com/opencloud-eu/opencloud/services/auth-api/pkg/metrics"
	svc "github.com/opencloud-eu/opencloud/services/auth-api/pkg/service/http/v0"
	"go-micro.dev/v4"
	oteltrace "go.opentelemetry.io/otel/trace"
)

// Server initializes the http service and server.
func Server(
	logger *log.Logger,
	ctx context.Context,
	cfg *config.Config,
	traceProvider oteltrace.TracerProvider,
) (http.Service, error) {
	service, err := http.NewService(
		http.TLSConfig(cfg.HTTP.TLS),
		http.Logger(*logger),
		http.Name(cfg.Service.Name),
		http.Version(version.GetString()),
		http.Namespace(cfg.HTTP.Namespace),
		http.Address(cfg.HTTP.Addr),
		http.Context(ctx),
		http.TraceProvider(traceProvider),
	)
	if err != nil {
		logger.Error().
			Err(err).
			Msg("Error initializing http service")
		return http.Service{}, fmt.Errorf("could not initialize http service: %w", err)
	}

	met, err := metrics.New(logger)
	if err != nil {
		return http.Service{}, err
	}

	handle, err := svc.NewService(
		logger,
		met,
		traceProvider,
		cfg,
		middleware.RealIP,
		middleware.RequestID,
		opencloudmiddleware.Version(
			cfg.Service.Name,
			version.GetString(),
		),
		opencloudmiddleware.Logger(*logger),
	)
	if err != nil {
		return http.Service{}, err
	}

	{
		//handle = svc.NewInstrument(handle, options.Metrics)
		//handle = svc.NewLogging(handle, options.Logger)
	}

	if err := micro.RegisterHandler(service.Server(), handle); err != nil {
		return http.Service{}, err
	}

	return service, nil
}
