package svc

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/riandyrn/otelchi"

	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/pkg/tracing"
	"github.com/opencloud-eu/opencloud/services/groupware/pkg/config"
	"github.com/opencloud-eu/opencloud/services/groupware/pkg/jmap"
)

/*
type contextKey string

const (
	keyContextKey contextKey = "key"
)
*/

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
			"groupware",
			otelchi.WithChiRoutes(m),
			otelchi.WithTracerProvider(options.TraceProvider),
			otelchi.WithPropagators(tracing.GetPropagator()),
		),
	)

	svc := Groupware{
		config: options.Config,
		mux:    m,
		logger: options.Logger,
	}

	m.Route(options.Config.HTTP.Root, func(r chi.Router) {
		r.Get("/", svc.WellDefined)
		r.Get("/ping", svc.Ping)
	})

	_ = chi.Walk(m, func(method string, route string, _ http.Handler, middlewares ...func(http.Handler) http.Handler) error {
		options.Logger.Debug().Str("method", method).Str("route", route).Int("middlewares", len(middlewares)).Msg("serving endpoint")
		return nil
	})

	return svc
}

// Thumbnails implements the business logic for Service.
type Groupware struct {
	config     *config.Config
	logger     log.Logger
	mux        *chi.Mux
	httpClient *http.Client
}

// ServeHTTP implements the Service interface.
func (s Groupware) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

type IndexResponse struct {
	AccountId string
}

func (IndexResponse) Render(w http.ResponseWriter, r *http.Request) error {
	return nil
}

func (g Groupware) Ping(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNoContent)
}

func (g Groupware) WellDefined(w http.ResponseWriter, r *http.Request) {
	//logger := g.logger.SubloggerWithRequestID(r.Context())

	client := jmap.New(g.httpClient, r.Context(), "alan", "demo", "https://stalwart.opencloud.test/jmap", "cs")
	wellKnown := client.FetchWellKnown()
	_ = render.Render(w, r, IndexResponse{AccountId: wellKnown.PrimaryAccounts[jmap.JmapMail]})
}
