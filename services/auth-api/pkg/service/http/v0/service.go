package svc

import (
	"context"
	"crypto/tls"
	"net/http"
	"regexp"
	"time"

	"github.com/MicahParks/jwkset"
	"github.com/MicahParks/keyfunc/v3"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/golang-jwt/jwt/v5"

	"github.com/riandyrn/otelchi"
	"go.opentelemetry.io/otel/attribute"
	oteltrace "go.opentelemetry.io/otel/trace"

	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/pkg/tracing"
	"github.com/opencloud-eu/opencloud/services/auth-api/pkg/config"
	"github.com/opencloud-eu/opencloud/services/auth-api/pkg/metrics"
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
			otelchi.WithTraceResponseHeaders(otelchi.TraceHeaderConfig{}),
		),
	)

	svc, err := NewAuthenticationApi(options.Config, &options.Logger, options.Metrics, options.TraceProvider, m)
	if err != nil {
		panic(err) // TODO p.bleser what to do when we encounter an error in a NewService() ?
	}

	m.Route(options.Config.HTTP.Root, func(r chi.Router) {
		r.Post("/", svc.Authenticate)
	})

	_ = chi.Walk(m, func(method string, route string, _ http.Handler, middlewares ...func(http.Handler) http.Handler) error {
		options.Logger.Debug().Str("method", method).Str("route", route).Int("middlewares", len(middlewares)).Msg("serving endpoint")
		return nil
	})

	return svc
}

type AuthenticationApi struct {
	config     *config.Config
	logger     *log.Logger
	metrics    *metrics.Metrics
	tracer     oteltrace.Tracer
	mux        *chi.Mux
	refreshCtx context.Context
	jwksFunc   keyfunc.Keyfunc
}

func NewAuthenticationApi(
	config *config.Config,
	logger *log.Logger,
	metrics *metrics.Metrics,
	tracerProvider oteltrace.TracerProvider,
	mux *chi.Mux,
) (*AuthenticationApi, error) {

	tracer := tracerProvider.Tracer("instrumentation/" + config.HTTP.Namespace + "/" + config.Service.Name)

	var httpClient *http.Client
	{
		tr := http.DefaultTransport.(*http.Transport).Clone()
		tr.ResponseHeaderTimeout = time.Duration(10) * time.Second
		tlsConfig := &tls.Config{InsecureSkipVerify: true}
		tr.TLSClientConfig = tlsConfig
		h := http.DefaultClient
		h.Transport = tr
		httpClient = h
	}

	refreshCtx := context.Background()

	storage, err := jwkset.NewStorageFromHTTP(config.Authentication.JwkEndpoint, jwkset.HTTPClientStorageOptions{
		Client:                    httpClient,
		Ctx:                       refreshCtx,
		HTTPExpectedStatus:        http.StatusOK,
		HTTPMethod:                http.MethodGet,
		HTTPTimeout:               time.Duration(10) * time.Second,
		NoErrorReturnFirstHTTPReq: true,
		RefreshInterval:           time.Duration(10) * time.Minute,
		RefreshErrorHandler: func(ctx context.Context, err error) {
			logger.Error().Err(err).Ctx(ctx).Str("url", config.Authentication.JwkEndpoint).Msg("failed to refresh JWK Set from IDP")
		},
		//ValidateOptions: jwkset.JWKValidateOptions{},
	})
	if err != nil {
		return nil, err
	}

	jwksFunc, err := keyfunc.New(keyfunc.Options{
		Ctx:          refreshCtx,
		UseWhitelist: []jwkset.USE{jwkset.UseSig},
		Storage:      storage,
	})
	if err != nil {
		return nil, err
	}

	return &AuthenticationApi{
		config:     config,
		mux:        mux,
		logger:     logger,
		metrics:    metrics,
		tracer:     tracer,
		refreshCtx: refreshCtx,
		jwksFunc:   jwksFunc,
	}, nil
}

func (a AuthenticationApi) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	a.mux.ServeHTTP(w, r)
}

type SuccessfulAuthResponse struct {
	Subject string   `json:"subject"`
	Roles   []string `json:"roles,omitempty"`
}

func (SuccessfulAuthResponse) Render(w http.ResponseWriter, r *http.Request) error {
	return nil
}

type FailedAuthResponse struct {
	Reason string `json:"reason,omitempty"`
}

func (FailedAuthResponse) Render(w http.ResponseWriter, r *http.Request) error {
	return nil
}

type CustomClaims struct {
	Roles                               []string         `json:"roles,omitempty"`
	AuthorizedParties                   jwt.ClaimStrings `json:"azp,omitempty"`
	SessionId                           string           `json:"sid,omitempty"`
	AuthenticationContextClassReference string           `json:"acr,omitempty"`
	Scope                               jwt.ClaimStrings `json:"scope,omitempty"`
	EmailVerified                       bool             `json:"email_verified,omitempty"`
	Name                                string           `json:"name,omitempty"`
	Groups                              jwt.ClaimStrings `json:"groups,omitempty"`
	PreferredUsername                   string           `json:"preferred_username,omitempty"`
	GivenName                           string           `json:"given_name,omitempty"`
	FamilyName                          string           `json:"family_name,omitempty"`
	Uuid                                string           `json:"uuid,omitempty"`
	Email                               string           `json:"email,omitempty"`

	jwt.RegisteredClaims
}

var authRegex = regexp.MustCompile(`(?i)^(Basic|Bearer)\s+(.+)$`)

func (a AuthenticationApi) failedAuth() {
	a.metrics.Attempts.WithLabelValues(metrics.OutcomeLabel, metrics.AttemptFailureOutcome).Inc()
}
func (a AuthenticationApi) succeededAuth() {
	a.metrics.Attempts.WithLabelValues(metrics.OutcomeLabel, metrics.AttemptSuccessOutcome).Inc()
}

func (a AuthenticationApi) Authenticate(w http.ResponseWriter, r *http.Request) {
	_, span := a.tracer.Start(r.Context(), "authenticate")
	defer span.End()

	auth := r.Header.Get("Authorization")
	if auth == "" {
		a.logger.Warn().Msg("missing Authorization header")
		w.WriteHeader(http.StatusBadRequest) // authentication header is missing altogether
		_ = render.Render(w, r, FailedAuthResponse{Reason: "Missing Authorization header"})
		a.failedAuth()
		return
	}
	matches := authRegex.FindStringSubmatch(auth)
	if matches == nil || len(matches) != 3 {
		a.logger.Warn().Msg("unsupported Authorization header")
		w.WriteHeader(http.StatusBadRequest) // authentication header is unsupported
		_ = render.Render(w, r, FailedAuthResponse{Reason: "Unsupported Authorization header"})
		a.failedAuth()
		return
	}

	if matches[1] == "Basic" {
		span.SetAttributes(attribute.String("authenticate.scheme", "basic"))
		a.metrics.Attempts.WithLabelValues(metrics.TypeLabel, metrics.BasicType).Inc()

		username, password, ok := r.BasicAuth()
		if !ok {
			a.logger.Warn().Msg("failed to decode basic credentials")
			w.WriteHeader(http.StatusBadRequest) // failed to decode the basic credentials
			_ = render.Render(w, r, FailedAuthResponse{Reason: "Failed to decode basic credentials"})
			a.failedAuth()
			return
		}
		if password == "secret" {
			_ = render.Render(w, r, SuccessfulAuthResponse{Subject: username})
			a.succeededAuth()
		} else {
			a.logger.Info().Str("username", username).Msg("authentication failed")
			w.WriteHeader(http.StatusUnauthorized)
			_ = render.Render(w, r, FailedAuthResponse{Reason: "Unauthorized credentials"})
			a.failedAuth()
			return
		}
	} else if matches[1] == "Bearer" {
		span.SetAttributes(attribute.String("authenticate.scheme", "bearer"))
		a.metrics.Attempts.WithLabelValues(metrics.TypeLabel, metrics.BearerType).Inc()

		claims := &CustomClaims{}
		tokenString := matches[2]
		token, err := jwt.ParseWithClaims(tokenString, claims, a.jwksFunc.Keyfunc, jwt.WithExpirationRequired(), jwt.WithLeeway(5*time.Second))
		if err != nil {
			a.logger.Warn().Err(err).Msg("failed to parse bearer token")
			w.WriteHeader(http.StatusBadRequest) // failed to parse bearer token
			_ = render.Render(w, r, FailedAuthResponse{Reason: "Failed to parse bearer token"})
			return
		}

		a.logger.Info().Str("type", matches[1]).Interface("header", token.Header).Interface("claims", token.Claims).Bool("valid", token.Valid).Msgf("successfully parsed token")

		if typedClaims, ok := token.Claims.(*CustomClaims); ok && token.Valid {
			sub := typedClaims.PreferredUsername
			if sub == "" {
				sub, err = typedClaims.GetSubject()
				if err != nil {
					a.logger.Warn().Err(err).Msg("failed to retrieve sub claim from token")
					w.WriteHeader(http.StatusBadRequest) // failed to extract sub claim from bearer token
					_ = render.Render(w, r, FailedAuthResponse{Reason: "Failed to extract sub claim from bearer token"})
					return
				}
			}
			_ = render.Render(w, r, SuccessfulAuthResponse{Subject: sub, Roles: claims.Roles})
		} else {
			w.WriteHeader(http.StatusBadRequest) // failed to extract sub claim from bearer token
			_ = render.Render(w, r, FailedAuthResponse{Reason: "Failed to parse bearer token"})
			return
		}
	} else {
		a.metrics.Attempts.WithLabelValues(metrics.TypeLabel, metrics.UnsupportedType).Inc()

		w.WriteHeader(http.StatusBadRequest) // authentication header is unsupported
		_ = render.Render(w, r, FailedAuthResponse{Reason: "Unsupported Authorization type"})
		return
	}
}
