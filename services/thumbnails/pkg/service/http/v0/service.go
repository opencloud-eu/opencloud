package svc

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
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
	// the parser validated this at startup, so only a test gets here with a
	// value that does not parse, and then the body stays unbounded
	maxBody, err := limits.MaxInputBytes()
	if err != nil {
		options.Logger.Error().Err(err).Msg("ignoring the configured maximum input image file size")
	}

	svc := Thumbnails{
		logger:       options.Logger,
		mux:          m,
		maxWidth:     limits.MaxInputWidth,
		maxHeight:    limits.MaxInputHeight,
		maxOutWidth:  limits.MaxOutputWidth,
		maxOutHeight: limits.MaxOutputHeight,
		maxFrames:    limits.MaxAnimationFrames,
		maxBody:      maxBody,
		limiter:      newConcurrencyLimiter(limits.MaxConcurrentRequests),
	}

	// Push-based thumbnail generation endpoint (imagor-compatible). The optional
	// operation segment selects the resize/crop mode; its absence is the default
	// center-crop fill. Everything after the box is imagor's tail: an optional
	// filters segment, which may be empty, followed by an optional image key,
	// which an upload loader has no use for and which is ignored here too.
	// StripSlashes makes the trailing slash optional, the operation segment and
	// the tail are the four remaining combinations.
	m.Use(middleware.StripSlashes)
	m.Post("/unsafe/{operation}/{width}x{height}/*", svc.pushHandler)
	m.Post("/unsafe/{width}x{height}/*", svc.pushHandler)
	m.Post("/unsafe/{operation}/{width}x{height}", svc.pushHandler)
	m.Post("/unsafe/{width}x{height}", svc.pushHandler)

	_ = chi.Walk(m, func(method string, route string, _ http.Handler, middlewares ...func(http.Handler) http.Handler) error {
		options.Logger.Debug().Str("method", method).Str("route", route).Int("middlewares", len(middlewares)).Msg("serving endpoint")
		return nil
	})

	return svc
}

// Thumbnails implements the business logic for Service.
type Thumbnails struct {
	logger       log.Logger
	mux          *chi.Mux
	maxWidth     int
	maxHeight    int
	maxOutWidth  int
	maxOutHeight int
	maxFrames    int
	maxBody      uint64
	limiter      *concurrencyLimiter
}

// ServeHTTP implements the Service interface.
func (s Thumbnails) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}
