package svc

import (
	"crypto/tls"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/riandyrn/otelchi"

	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/pkg/tracing"
	"github.com/opencloud-eu/opencloud/services/groupware/pkg/config"
	"github.com/opencloud-eu/opencloud/services/groupware/pkg/jmap"
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
			"groupware",
			otelchi.WithChiRoutes(m),
			otelchi.WithTracerProvider(options.TraceProvider),
			otelchi.WithPropagators(tracing.GetPropagator()),
		),
	)

	svc := NewGroupware(options.Config, &options.Logger, m)

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

type Groupware struct {
	jmapClient       jmap.JmapClient
	usernameProvider jmap.HttpJmapUsernameProvider
	config           *config.Config
	logger           *log.Logger
	mux              *chi.Mux
}

func NewGroupware(config *config.Config, logger *log.Logger, mux *chi.Mux) *Groupware {
	usernameProvider := jmap.NewRevaContextHttpJmapUsernameProvider()
	httpApiClient := httpApiClient(config, usernameProvider)
	jmapClient := jmap.NewJmapClient(httpApiClient, httpApiClient)
	return &Groupware{
		jmapClient:       jmapClient,
		usernameProvider: usernameProvider,
		config:           config,
		mux:              mux,
		logger:           logger,
	}
}

func (g Groupware) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	g.mux.ServeHTTP(w, r)
}

type IndexResponse struct {
	AccountId string
}

func (IndexResponse) Render(w http.ResponseWriter, r *http.Request) error {
	return nil
}

func (g Groupware) Ping(w http.ResponseWriter, r *http.Request) {
	g.logger.Info().Msg("groupware pinged")
	w.WriteHeader(http.StatusNoContent)
}

func httpApiClient(config *config.Config, usernameProvider jmap.HttpJmapUsernameProvider) *jmap.HttpJmapApiClient {
	tr := http.DefaultTransport.(*http.Transport).Clone()
	tr.ResponseHeaderTimeout = time.Duration(10) * time.Second
	tlsConfig := &tls.Config{InsecureSkipVerify: true}
	tr.TLSClientConfig = tlsConfig
	c := *http.DefaultClient
	c.Transport = tr

	api := jmap.NewHttpJmapApiClient(
		config.Mail.BaseUrl,
		config.Mail.JmapUrl,
		&c,
		usernameProvider,
		config.Mail.Master.Username,
		config.Mail.Master.Password,
	)
	return api
}
func (g Groupware) WellDefined(w http.ResponseWriter, r *http.Request) {
	logger := g.logger.SubloggerWithRequestID(r.Context())
	username, err := g.usernameProvider.GetUsername(r.Context(), &logger)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	jmapContext, err := g.jmapClient.FetchJmapContext(username, &logger)
	if err != nil {
		return
	}

	_ = render.Render(w, r, IndexResponse{AccountId: jmapContext.AccountId})
}
