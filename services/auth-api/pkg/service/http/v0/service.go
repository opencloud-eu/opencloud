package svc

import (
	"net/http"
	"regexp"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/golang-jwt/jwt/v5"
	"github.com/riandyrn/otelchi"

	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/pkg/tracing"
	"github.com/opencloud-eu/opencloud/services/auth-api/pkg/config"
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
			"auth-api",
			otelchi.WithChiRoutes(m),
			otelchi.WithTracerProvider(options.TraceProvider),
			otelchi.WithPropagators(tracing.GetPropagator()),
		),
	)

	svc := NewAuthenticationApi(options.Config, &options.Logger, m)

	m.Route(options.Config.HTTP.Root, func(r chi.Router) {
		r.Get("/", svc.Authenticate)
		r.Post("/", svc.Authenticate)
	})

	_ = chi.Walk(m, func(method string, route string, _ http.Handler, middlewares ...func(http.Handler) http.Handler) error {
		options.Logger.Debug().Str("method", method).Str("route", route).Int("middlewares", len(middlewares)).Msg("serving endpoint")
		return nil
	})

	return svc
}

type AuthenticationApi struct {
	config *config.Config
	logger *log.Logger
	mux    *chi.Mux
}

func NewAuthenticationApi(config *config.Config, logger *log.Logger, mux *chi.Mux) *AuthenticationApi {
	return &AuthenticationApi{
		config: config,
		mux:    mux,
		logger: logger,
	}
}

func (a AuthenticationApi) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	a.mux.ServeHTTP(w, r)
}

type AuthResponse struct {
	Subject string
}

func (AuthResponse) Render(w http.ResponseWriter, r *http.Request) error {
	return nil
}

var authRegex = regexp.MustCompile("^(i:Basic|Bearer)\\s+(.+)$")

func (a AuthenticationApi) Authenticate(w http.ResponseWriter, r *http.Request) {
	auth := r.Header.Get("Authorization")
	if auth == "" {
		w.WriteHeader(http.StatusBadRequest) // authentication header is missing altogether
		return
	}
	matches := authRegex.FindAllString(auth, 2)
	if matches == nil {
		w.WriteHeader(http.StatusBadRequest) // authentication header is unsupported
		return
	}

	if matches[0] == "Basic" {
		username, password, ok := r.BasicAuth()
		if !ok {
			w.WriteHeader(http.StatusBadRequest) // failed to decode the basic credentials
		}
		if password == "secret" {
			_ = render.Render(w, r, AuthResponse{Subject: username})
		} else {
			w.WriteHeader(http.StatusUnauthorized)
		}
	} else if matches[0] == "Bearer" {
		claims := jwt.MapClaims{}
		publicKey := nil
		tokenString := matches[1]
		token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
			token.Header["kid"]
			return publicKey, nil
		}, jwt.WithExpirationRequired(), jwt.WithLeeway(5*time.Second))
		if err != nil {
			w.WriteHeader(http.StatusBadRequest) // failed to parse bearer token
		}
		sub, err := token.Claims.GetSubject()
		if err != nil {
			w.WriteHeader(http.StatusBadRequest) // failed to extract sub claim from bearer token
		}
		_ = render.Render(w, r, AuthResponse{Subject: sub})
	} else {
		w.WriteHeader(http.StatusBadRequest) // authentication header is unsupported
		return
	}

	// TODO

	/*
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	*/

	_ = render.Render(w, r, AuthResponse{Subject: "todo"})
}
