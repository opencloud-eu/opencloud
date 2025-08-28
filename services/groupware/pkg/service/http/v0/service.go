package svc

import (
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/riandyrn/otelchi"

	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/pkg/tracing"
	"github.com/opencloud-eu/opencloud/services/groupware/pkg/groupware"
	"github.com/opencloud-eu/opencloud/services/groupware/pkg/metrics"
)

// Service defines the service handlers.
type Service interface {
	ServeHTTP(w http.ResponseWriter, r *http.Request)
}

// NewService returns a service implementation for Service.
func NewService(opts ...Option) (Service, error) {
	options := newOptions(opts...)

	m := chi.NewMux()
	m.Use(options.Middleware...)

	m.Use(
		otelchi.Middleware(
			"groupware",
			otelchi.WithChiRoutes(m),
			otelchi.WithTracerProvider(options.TraceProvider),
			otelchi.WithPropagators(tracing.GetPropagator()),
		),
	)

	logger := &options.Logger

	registerer := metrics.NewLoggingPrometheusRegisterer(prometheus.DefaultRegisterer, logger)

	gw, err := groupware.NewGroupware(options.Config, logger, m, registerer)
	if err != nil {
		return nil, err
	}

	m.Route(options.Config.HTTP.Root, gw.Route)

	{
		level := options.Logger.Debug()
		if level.Enabled() {
			routes := []string{}
			_ = chi.Walk(m, func(method string, route string, _ http.Handler, middlewares ...func(http.Handler) http.Handler) error {
				routes = append(routes, fmt.Sprintf("%s %s", method, route))
				return nil
			})
			level.Array("routes", log.StringArray(routes)).Msgf("serving %v endpoints", len(routes))
		}
	}

	metrics.StartupMetrics(registerer)

	return gw, nil
}
