package svc

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/riandyrn/otelchi"

	"github.com/opencloud-eu/opencloud/pkg/tracing"
	"github.com/opencloud-eu/opencloud/services/groupware/pkg/groupware"
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

	gw, err := groupware.NewGroupware(options.Config, &options.Logger, m)
	if err != nil {
		return nil, err
	}

	m.Route(options.Config.HTTP.Root, gw.Route)

	_ = chi.Walk(m, func(method string, route string, _ http.Handler, middlewares ...func(http.Handler) http.Handler) error {
		options.Logger.Debug().Str("method", method).Str("route", route).Int("middlewares", len(middlewares)).Msg("serving endpoint")
		return nil
	})

	return gw, nil
}
