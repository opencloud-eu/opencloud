package staticroutes_test

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/pkg/oidc"
	"github.com/opencloud-eu/opencloud/services/proxy/pkg/config/defaults"
	"github.com/opencloud-eu/opencloud/services/proxy/pkg/middleware"
	"github.com/opencloud-eu/opencloud/services/proxy/pkg/router"
	"github.com/opencloud-eu/opencloud/services/proxy/pkg/staticroutes"
	"github.com/stretchr/testify/require"
	"go-micro.dev/v4/store"
)

func TestBackchannelLogoutInvalidatesEveryCachedToken(t *testing.T) {
	for _, skipUserInfo := range []bool{false, true} {
		for _, mode := range []string{"subject", "session", "subject and session"} {
			t.Run(fmt.Sprintf("skip_user_info=%t/%s", skipUserInfo, mode), func(t *testing.T) {
				idp := newLogoutTestIDP(t)
				cache := &logoutTestCache{Store: store.NewMemoryStore(), writes: make(chan string, 32)}
				auth, routes := idp.handlers(cache, skipUserInfo)
				first := idp.accessToken(t, "alice", "session-a", "first", time.Hour)
				second := idp.accessToken(t, "alice", "session-a", "second", 5*time.Minute)
				otherSubject := "alice"
				if mode == "subject" {
					otherSubject = "bob"
				}
				unrelated := idp.accessToken(t, otherSubject, "session-b", "unrelated", time.Hour)
				for _, token := range []string{first, second, unrelated} {
					require.Equal(t, http.StatusOK, logoutTestRequest(auth, token))
					cache.waitForSession(t)
				}
				subject, session := "alice", "session-a"
				if mode == "subject" {
					session = ""
				} else if mode == "session" {
					subject = ""
				}
				logout := idp.logoutToken(t, subject, session)
				require.Equal(t, http.StatusOK, logoutTestLogout(routes, logout).Code)
				before := idp.userinfoRequests.Load()
				for _, token := range []string{first, second} {
					require.Equal(t, http.StatusUnauthorized, logoutTestRequest(auth, token), "all previously accepted tokens of the logged-out session must be rejected")
				}
				require.Equal(t, before, idp.userinfoRequests.Load(), "revoked tokens must be rejected before requesting userinfo")
				require.Equal(t, http.StatusOK, logoutTestRequest(auth, unrelated), "unrelated sessions must remain authenticated")
				// Repeated logout callbacks must succeed without re-enabling tokens.
				require.Equal(t, http.StatusOK, logoutTestLogout(routes, logout).Code)
				require.Equal(t, http.StatusUnauthorized, logoutTestRequest(auth, first))
				// Sharing a persistent store with another proxy must retain revocations.
				otherAuth, _ := idp.handlers(cache, skipUserInfo)
				require.Equal(t, http.StatusUnauthorized, logoutTestRequest(otherAuth, second))
			})
		}
	}
}

type logoutTestIDP struct {
	server           *httptest.Server
	key              *rsa.PrivateKey
	userinfoRequests atomic.Int64
}

func newLogoutTestIDP(t *testing.T) *logoutTestIDP {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	idp := &logoutTestIDP{key: key}
	mux := http.NewServeMux()
	idp.server = httptest.NewServer(mux)
	t.Cleanup(idp.server.Close)
	mux.HandleFunc("/.well-known/openid-configuration", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"issuer": idp.server.URL, "jwks_uri": idp.server.URL + "/jwks", "userinfo_endpoint": idp.server.URL + "/userinfo",
			"id_token_signing_alg_values_supported": []string{"RS256"},
		})
	})
	mux.HandleFunc("/jwks", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"keys": []any{map[string]any{
			"kty": "RSA", "kid": "test", "alg": "RS256", "use": "sig",
			"n": base64.RawURLEncoding.EncodeToString(key.N.Bytes()), "e": "AQAB",
		}}})
	})
	mux.HandleFunc("/userinfo", func(w http.ResponseWriter, r *http.Request) {
		idp.userinfoRequests.Add(1)
		claims := jwt.MapClaims{}
		_, _, err := new(jwt.Parser).ParseUnverified(strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer "), claims)
		if err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"sub": claims["sub"], "preferred_username": claims["sub"]})
	})
	return idp
}

func (idp *logoutTestIDP) sign(t *testing.T, claims jwt.MapClaims) string {
	t.Helper()
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = "test"
	signed, err := token.SignedString(idp.key)
	require.NoError(t, err)
	return signed
}

func (idp *logoutTestIDP) accessToken(t *testing.T, subject, session, id string, lifetime time.Duration) string {
	return idp.sign(t, jwt.MapClaims{
		"iss": idp.server.URL, "sub": subject, "sid": session, "aud": "opencloud", "jti": id,
		"iat": time.Now().Unix(), "exp": time.Now().Add(lifetime).Unix(),
	})
}

func (idp *logoutTestIDP) logoutToken(t *testing.T, subject, session string) string {
	return idp.sign(t, jwt.MapClaims{
		"iss": idp.server.URL, "sub": subject, "sid": session, "aud": "opencloud", "jti": "logout",
		"iat": time.Now().Unix(), "exp": time.Now().Add(time.Minute).Unix(),
		"events": map[string]any{"http://schemas.openid.net/event/backchannel-logout": map[string]any{}},
	})
}

func (idp *logoutTestIDP) handlers(cache store.Store, skipUserInfo bool) (middleware.Authenticator, http.Handler) {
	cfg := defaults.FullDefaultConfig()
	cfg.OIDC.Issuer = idp.server.URL
	cfg.OIDC.SkipUserInfo = skipUserInfo
	client := oidc.NewOIDCClient(oidc.WithOidcIssuer(idp.server.URL), oidc.WithHTTPClient(idp.server.Client()), oidc.WithAccessTokenVerifyMethod("jwt"), oidc.WithLogger(log.NopLogger()))
	auth := middleware.NewOIDCAuthenticator(
		middleware.Logger(log.NopLogger()), middleware.UserInfoCache(cache), middleware.OIDCClient(client),
		middleware.OIDCIss(idp.server.URL), middleware.SkipUserInfo(skipUserInfo),
		middleware.AccessTokenVerifyMethod("jwt"), middleware.DefaultAccessTokenTTL(time.Minute),
		middleware.HTTPClient(idp.server.Client()),
	)
	routes := &staticroutes.StaticRouteHandler{
		Prefix: "/", Config: *cfg, Logger: log.NopLogger(), OidcClient: client, UserInfoCache: cache, Proxy: http.NotFoundHandler(),
	}
	return auth, routes.Handler()
}

func logoutTestRequest(auth middleware.Authenticator, token string) int {
	handler := middleware.Authentication([]middleware.Authenticator{auth})(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) }))
	req := httptest.NewRequest(http.MethodGet, "/protected", http.NoBody)
	req = req.WithContext(router.SetRoutingInfo(req.Context(), router.RoutingInfo{}))
	req.Header.Set("Authorization", "Bearer "+token)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, req)
	return recorder.Code
}

func logoutTestLogout(routes http.Handler, token string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, "/backchannel_logout", strings.NewReader(url.Values{"logout_token": {token}}.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	recorder := httptest.NewRecorder()
	routes.ServeHTTP(recorder, req)
	return recorder
}

type logoutTestCache struct {
	store.Store
	writes chan string
}

func (cache *logoutTestCache) Write(record *store.Record, opts ...store.WriteOption) error {
	err := cache.Store.Write(record, opts...)
	if err == nil {
		cache.writes <- record.Key
	}
	return err
}

func (cache *logoutTestCache) waitForSession(t *testing.T) {
	t.Helper()
	timer := time.NewTimer(5 * time.Second)
	defer timer.Stop()
	for {
		select {
		case key := <-cache.writes:
			if strings.Contains(key, ".") {
				return
			}
		case <-timer.C:
			t.Fatal("timed out waiting for the session lookup to be written")
		}
	}
}
