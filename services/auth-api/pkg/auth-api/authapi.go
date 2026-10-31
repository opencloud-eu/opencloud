package auth_api

import (
	"encoding/json"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/golang-jwt/jwt/v5"

	oteltrace "go.opentelemetry.io/otel/trace"

	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/pkg/version"

	"github.com/opencloud-eu/opencloud/services/auth-api/pkg/config"
	"github.com/opencloud-eu/opencloud/services/auth-api/pkg/metrics"
)

const defaultLeeway int64 = 5

type appId string

type AuthApi struct {
	mux                 *chi.Mux
	logger              *log.Logger
	metrics             *metrics.Metrics
	tracer              oteltrace.Tracer
	parser              *jwt.Parser
	keyFunc             func(token *jwt.Token) (any, error)
	audiences           []string
	requireSharedSecret bool
	sharedSecrets       map[string]appId
}

func parseSecrets(s string) (map[string]appId, error) {
	s = strings.TrimSpace(s)
	if len(s) == 0 {
		return map[string]appId{}, nil
	}
	result := map[string]appId{}
	for item := range strings.SplitSeq(s, ";") {
		item = strings.TrimSpace(item)
		parts := strings.Split(item, "=")
		switch len(parts) {
		case 0:
		case 1:
			result[item] = appId("")
		case 2:
			result[parts[1]] = appId(parts[0])
		default:
			result[strings.Join(parts[1:], "=")] = appId(parts[0])
		}
	}
	return result, nil
}

func NewAuthApi(
	config *config.Config,
	logger *log.Logger,
	tracerProvider oteltrace.TracerProvider,
	metrics *metrics.Metrics,
	mux *chi.Mux,
) (*AuthApi, error) {
	jwtSecret := []byte(config.TokenManager.JWTSecret)
	parser := jwt.NewParser(jwt.WithAudience("reva"), jwt.WithLeeway(time.Duration(defaultLeeway)*time.Second))

	var tracer oteltrace.Tracer
	if tracerProvider != nil {
		tracer = tracerProvider.Tracer("instrumentation/" + config.HTTP.Namespace + "/" + config.Service.Name)
	}

	metrics.BuildInfo.WithLabelValues(version.GetString()).Set(1)

	sharedSecrets, err := parseSecrets(config.Auth.SharedSecrets)
	if err != nil {
		return nil, err
	}

	return &AuthApi{
		mux:                 mux,
		logger:              logger,
		metrics:             metrics,
		tracer:              tracer,
		parser:              parser,
		keyFunc:             func(token *jwt.Token) (any, error) { return jwtSecret, nil },
		audiences:           config.Auth.Audiences,
		requireSharedSecret: config.Auth.RequireSharedSecret,
		sharedSecrets:       sharedSecrets,
	}, nil
}

func (a *AuthApi) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	a.mux.ServeHTTP(w, r)
}

func (a *AuthApi) Route(r chi.Router) {
	r.Get("/", a.Unauthenticated)
	r.Post("/", a.Unauthenticated)
	r.Get("/{secret}", a.Authenticated)
	r.Post("/{secret}", a.Authenticated)
}

type StalwartClaims struct {
	Audience  []string         `json:"aud"`
	Subject   string           `json:"sub"`
	Name      string           `json:"name"`
	Username  string           `json:"preferred_username"`
	Nickname  string           `json:"nickname,omitempty"`
	Email     string           `json:"email"`
	Expires   *jwt.NumericDate `json:"exp"`
	IssuedAt  *jwt.NumericDate `json:"iat"`
	Issuer    string           `json:"iss,omitempty"`
	NotBefore *jwt.NumericDate `json:"nbf,omitzero"`
}

func invalidRequest(w http.ResponseWriter) {
	w.Header().Add("WWW-Authenticate", `Bearer error="invalid_request"`)
	w.WriteHeader(http.StatusBadRequest)
}

func invalidToken(w http.ResponseWriter) {
	w.Header().Add("WWW-Authenticate", `Bearer error="invalid_token"`)
	w.WriteHeader(http.StatusUnauthorized)
}

func (a *AuthApi) unsupportedAuth() {
	a.metrics.Attempts.WithLabelValues(
		metrics.AttemptFailureOutcome,
		metrics.UnsupportedType,
	).Inc()
}
func (a *AuthApi) failedAuth(duration time.Duration) {
	a.metrics.Attempts.WithLabelValues(
		metrics.AttemptFailureOutcome,
		metrics.BearerType,
	).Inc()
	a.metrics.Duration.WithLabelValues(
		metrics.AttemptFailureOutcome,
	).Observe(duration.Seconds())
}
func (a *AuthApi) succeededAuth(duration time.Duration) {
	a.metrics.Attempts.WithLabelValues(
		metrics.AttemptSuccessOutcome,
		metrics.BearerType,
	).Inc()
	a.metrics.Duration.WithLabelValues(
		metrics.AttemptSuccessOutcome,
	).Observe(duration.Seconds())
}

func (a *AuthApi) Unauthenticated(w http.ResponseWriter, r *http.Request) {
	if a.requireSharedSecret {
		a.unsupportedAuth()
		a.logger.Warn().Str("reason", "missing-shared-secret").Msgf("authentication failure: request did not provide a required shared secret")
		invalidRequest(w)
		return
	} else {
		a.authenticate(w, r, "")
	}
}

func (a *AuthApi) Authenticated(w http.ResponseWriter, r *http.Request) {
	secret := chi.URLParam(r, "secret")
	if app, ok := a.sharedSecrets[secret]; ok {
		a.authenticate(w, r, app)
	} else {
		a.unsupportedAuth()
		a.logger.Warn().Str("reason", "invalid-shared-secret").Msgf("authentication failure: request did not provide a valid shared secret")
		invalidRequest(w)
		return
	}
}

func (a *AuthApi) authenticate(w http.ResponseWriter, r *http.Request, app appId) {
	start := time.Now()
	logger := a.logger
	if app != "" {
		logger = log.From(logger.With().Str("app", log.SafeString(string(app))))
	}

	var span oteltrace.Span = nil
	if a.tracer != nil {
		_, span = a.tracer.Start(r.Context(), "authenticate")
		defer span.End()
	}

	auth := r.Header.Get("Authorization")
	if auth == "" {
		a.unsupportedAuth()
		logger.Warn().Str("reason", "missing-authorization-header").Msgf("authentication failure: missing 'Authorization' header")
		invalidRequest(w)
		return
	}
	if !strings.HasPrefix(auth, "Bearer ") {
		a.unsupportedAuth()
		logger.Warn().Str("reason", "authorization-header-not-bearer").Msgf("authentication failure: 'Authorization' header does not start with 'Bearer '")
		invalidRequest(w)
		return
	}

	tokenStr := auth[len("Bearer "):]

	claims := jwt.MapClaims{}
	token, err := a.parser.ParseWithClaims(tokenStr, claims, a.keyFunc)
	//token, _, err := jwt.NewParser().ParseUnverified(tokenStr, claims)
	if err != nil {
		a.failedAuth(time.Since(start))
		logger.Warn().Err(err).Str("reason", "token-parsing-failed").Msgf("authentication failure: failed to parse token")
		invalidToken(w)
		return
	}
	if !token.Valid {
		a.failedAuth(time.Since(start))
		logger.Warn().Err(err).Str("reason", "token-invalid").Msgf("authentication failure: the token is invalid")
		invalidToken(w)
		return
	}

	user, ok := claims["user"].(map[string]any)
	if !ok {
		a.failedAuth(time.Since(start))
		logger.Warn().Err(err).Str("reason", "token-missing-user").Msgf("authentication failure: token has no 'user' claim")
		invalidToken(w)
		return
	}
	id, ok := user["id"].(map[string]any)
	if !ok {
		a.failedAuth(time.Since(start))
		logger.Warn().Err(err).Str("reason", "token-missing-id").Msgf("authentication failure: token has no 'id' attribute in the 'user' claim")
		invalidToken(w)
		return
	}
	opaqueId, ok := id["opaque_id"].(string)
	if !ok {
		a.failedAuth(time.Since(start))
		logger.Warn().Err(err).Str("reason", "token-missing-id-opaqueid").Msgf("authentication failure: token has no 'id/opaque_id' attribute in the 'user' claim")
		invalidToken(w)
		return
	}
	username, ok := user["username"].(string)
	if !ok {
		a.failedAuth(time.Since(start))
		logger.Warn().Err(err).Str("reason", "token-missing-username").Msgf("authentication failure: token has no 'username' attribute in the 'user' claim")
		invalidToken(w)
		return
	}
	displayName, ok := user["display_name"].(string)
	if !ok {
		a.failedAuth(time.Since(start))
		logger.Warn().Err(err).Str("reason", "token-missing-displayname").Msgf("authentication failure: token has no 'display_name' attribute in the 'user' claim")
		invalidToken(w)
		return
	}
	mail, ok := user["mail"].(string)
	if !ok {
		a.failedAuth(time.Since(start))
		logger.Warn().Err(err).Str("reason", "token-missing-mail").Msgf("authentication failure: token has no 'mail' attribute in the 'user' claim")
		invalidToken(w)
		return
	}

	exp, err := token.Claims.GetExpirationTime()
	if err != nil {
		a.failedAuth(time.Since(start))
		logger.Warn().Err(err).Str("reason", "token-invalid-exp").Msgf("authentication failure: token has invalid 'exp'")
		invalidToken(w)
		return
	}
	iat, err := token.Claims.GetIssuedAt()
	if err != nil {
		a.failedAuth(time.Since(start))
		logger.Warn().Err(err).Str("reason", "token-invalid-iat").Msgf("authentication failure: token has invalid 'iat'")
		invalidToken(w)
		return
	}
	nbf, err := token.Claims.GetNotBefore()
	if err != nil {
		a.failedAuth(time.Since(start))
		logger.Warn().Err(err).Str("reason", "token-invalid-nbf").Msgf("authentication failure: token has invalid 'nbf'")
		invalidToken(w)
		return
	}
	iss, err := token.Claims.GetIssuer()
	if err != nil {
		a.failedAuth(time.Since(start))
		logger.Warn().Err(err).Str("reason", "token-invalid-iss").Msgf("authentication failure: token has invalid 'iss'")
		invalidToken(w)
		return
	}
	aud, err := claims.GetAudience()
	if err != nil {
		a.failedAuth(time.Since(start))
		logger.Warn().Err(err).Str("reason", "token-invalid-aud").Msgf("authentication failure: token has invalid 'aud'")
		invalidToken(w)
		return
	}

	audiences := aud
	if len(a.audiences) > 0 {
		audiences = slices.Concat(aud, a.audiences)
	}

	rc := StalwartClaims{
		Audience:  audiences,
		Subject:   opaqueId,
		Name:      displayName,
		Username:  username,
		Email:     mail,
		Expires:   exp,
		IssuedAt:  iat,
		NotBefore: nbf,
		Issuer:    iss,
	}
	response, err := json.Marshal(rc)
	if err != nil {
		a.failedAuth(time.Since(start))
		logger.Warn().Err(err).Str("reason", "response-serialization-failure").Msgf("authentication failure: failed to serialize response")
		invalidToken(w)
		return
	}

	logger.Debug().Str("username", username).Msg("successfully authenticated token")
	a.succeededAuth(time.Since(start))
	w.WriteHeader(http.StatusOK)
	w.Write(response)
}
