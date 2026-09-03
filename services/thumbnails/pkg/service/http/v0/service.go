package svc

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/riandyrn/otelchi"

	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/pkg/tracing"
)

// Service defines the service handlers.
type Service interface {
	ServeHTTP(w http.ResponseWriter, r *http.Request)
}

// NewService returns a service implementation for Service.
func NewService(opts ...Option) Service {
	options := newOptions(opts...)

	m := chi.NewMux()
	m.Use(options.Middleware...)

	m.Use(
		otelchi.Middleware(
			"thumbnails",
			otelchi.WithChiRoutes(m),
			otelchi.WithTracerProvider(options.TraceProvider),
			otelchi.WithPropagators(tracing.GetPropagator()),
		),
	)

	limits := options.Config.Thumbnail

	svc := Thumbnails{
		logger:    options.Logger,
		mux:       m,
		maxWidth:  limits.MaxInputWidth,
		maxHeight: limits.MaxInputHeight,
	}

	// Push-based thumbnail generation endpoint (imagor-compatible). The optional
	// operation segment selects the resize/crop mode; its absence is the default
	// center-crop fill. The filters segment is captured whole and parsed by the
	// handler so any set/order of imagor filters is accepted.
	handler := svc.pushHandler
	m.Post("/unsafe/{operation}/{width}x{height}/filters:{filters}/", handler)
	m.Post("/unsafe/{operation}/{width}x{height}/filters:{filters}", handler)
	m.Post("/unsafe/{width}x{height}/filters:{filters}/", handler)
	m.Post("/unsafe/{width}x{height}/filters:{filters}", handler)

	_ = chi.Walk(m, func(method string, route string, _ http.Handler, middlewares ...func(http.Handler) http.Handler) error {
		options.Logger.Debug().Str("method", method).Str("route", route).Int("middlewares", len(middlewares)).Msg("serving endpoint")
		return nil
	})

	return svc
}

// Thumbnails implements the business logic for Service.
type Thumbnails struct {
	logger    log.Logger
	mux       *chi.Mux
	maxWidth  int
	maxHeight int
}

// ServeHTTP implements the Service interface.
func (s Thumbnails) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}
