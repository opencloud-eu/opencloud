package middleware

import (
	"crypto/rand"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/justinas/alice"
	"github.com/opencloud-eu/opencloud/pkg/log"
	ocmiddleware "github.com/opencloud-eu/opencloud/pkg/middleware"
	"github.com/opencloud-eu/opencloud/pkg/registry"
	"github.com/opencloud-eu/opencloud/services/proxy/pkg/config/defaults"
	"github.com/opencloud-eu/opencloud/services/proxy/pkg/router"
	"github.com/opencloud-eu/reva/v2/pkg/audit"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go-micro.dev/v4/selector"
)

// Downloads through /data skip the reva data gateway: the proxy rewrites them
// to the data provider itself. They must still pass AuditContext, so the data
// provider gets a header the proxy set and never one the client sent.
func TestDataGatewayDownloadsGetTheProxyAuditHeader(t *testing.T) {
	secret := rand.Text()
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, &transferClaims{
		RegisteredClaims: jwt.RegisteredClaims{ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour))},
		Target:           "http://localhost:9158/data/spaces/storage$space!item",
	}).SignedString([]byte(secret))
	require.NoError(t, err)

	cfg := defaults.FullDefaultConfig()
	trusted, err := ocmiddleware.ParseTrustedProxies(cfg.ClientIP.TrustedProxies)
	require.NoError(t, err)
	sel := selector.NewSelector(selector.Registry(registry.GetRegistry()))

	var got *http.Request
	chain := alice.New(
		ocmiddleware.ClientIP(trusted, cfg.ClientIP.Header),
		NewDataGatewayMiddleware("/data", secret, time.Minute).Handler,
		router.Middleware(sel, cfg.PolicySelector, cfg.Policies, func(*http.Request) {}, log.NopLogger()),
		// the real authenticators are not needed: the route must be unprotected,
		// otherwise this one would answer 401 and the test fails
		Authentication([]Authenticator{fakeAuthenticator{method: AuthMethodOIDC, result: Failed()}}),
		AuditContext,
	).Then(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { got = r }))

	r := httptest.NewRequest(http.MethodGet, "/data/"+token, nil)
	r.RemoteAddr = "203.0.113.7:51234"
	r.Header.Set("X-Forwarded-For", "198.51.100.1")
	r.Header.Set(audit.Header, "ip=10.0.0.1&imp=admin")
	chain.ServeHTTP(httptest.NewRecorder(), r)

	require.NotNil(t, got, "the /data request reaches the reverse proxy")
	assert.Equal(t, "localhost:9158", got.URL.Host, "rewritten to the data provider")
	assert.Equal(t, audit.RequestContext{ClientAddress: "203.0.113.7"}.Encode(), got.Header.Get(audit.Header),
		"the header the client sent is replaced, the address is the real one")
}
