package svc

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/riandyrn/otelchi"
	oteltrace "go.opentelemetry.io/otel/trace"

	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/pkg/tracing"
	auth_api "github.com/opencloud-eu/opencloud/services/auth-api/pkg/auth-api"
	"github.com/opencloud-eu/opencloud/services/auth-api/pkg/config"
	"github.com/opencloud-eu/opencloud/services/auth-api/pkg/metrics"
)

// Service defines the service handlers.
type Service interface {
	ServeHTTP(w http.ResponseWriter, r *http.Request)
}

// NewService returns a service implementation for Service.
func NewService(
	logger *log.Logger,
	metrics *metrics.Metrics,
	tracerProvider oteltrace.TracerProvider,
	config *config.Config,
	middlewares ...func(http.Handler) http.Handler) (Service, error) {
	m := chi.NewMux()

	mwlist := []func(http.Handler) http.Handler{}
	mwlist = append(mwlist, middlewares...)
	o := otelchi.Middleware(
		"auth-api",
		otelchi.WithChiRoutes(m),
		otelchi.WithTracerProvider(tracerProvider),
		otelchi.WithPropagators(tracing.GetPropagator()),
		otelchi.WithTraceResponseHeaders(otelchi.TraceHeaderConfig{}),
	)
	mwlist = append(mwlist, o)

	m.Use(mwlist...)

	authApi, err := auth_api.NewAuthApi(config, logger, tracerProvider, metrics, m)
	if err != nil {
		return nil, err
	}

	m.Route(config.HTTP.Root, authApi.Route)

	return authApi, nil
}
