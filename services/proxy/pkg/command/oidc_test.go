package command

import (
	"bytes"
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
	"github.com/opencloud-eu/opencloud/services/proxy/pkg/config"
	"github.com/opencloud-eu/opencloud/services/proxy/pkg/config/defaults"
	"github.com/opencloud-eu/opencloud/services/proxy/pkg/middleware"
	"github.com/opencloud-eu/opencloud/services/proxy/pkg/router"
	"github.com/opencloud-eu/opencloud/services/proxy/pkg/staticroutes"
	bcl "github.com/opencloud-eu/opencloud/services/proxy/pkg/staticroutes/backchannellogout"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"
	"github.com/vmihailenco/msgpack/v5"
	"go-micro.dev/v4/store"
	"golang.org/x/crypto/sha3"
)

func TestOIDCAudienceAuthentication(t *testing.T) {
	for _, skipUserInfo := range []bool{false, true} {
		t.Run(fmt.Sprintf("skip_user_info=%t", skipUserInfo), func(t *testing.T) {
			idp := newAudienceTestIDP(t, "opencloud")
			for _, tt := range []struct {
				name      string
				audiences []string
				aud       any
				want      int
			}{
				{name: "matching string", audiences: []string{"opencloud"}, aud: "opencloud", want: http.StatusOK},
				{name: "matching array", audiences: []string{"opencloud", "opencloud-api"}, aud: []string{"immich", "opencloud-api"}, want: http.StatusOK},
				{name: "foreign despite matching userinfo", audiences: []string{"opencloud"}, aud: "immich", want: http.StatusUnauthorized},
				{name: "missing despite matching userinfo", audiences: []string{"opencloud"}, want: http.StatusUnauthorized},
				{name: "disabled accepts foreign", aud: "immich", want: http.StatusOK},
				{name: "disabled accepts missing", want: http.StatusOK},
			} {
				t.Run(tt.name, func(t *testing.T) {
					cache := newAudienceTestCache()
					cfg := audienceTestConfig(idp, tt.audiences, skipUserInfo)
					auth := newOIDCAuthenticator(log.NopLogger(), cfg, cache, idp.server.Client())
					token := idp.accessToken(t, jwt.MapClaims{"aud": tt.aud})
					before := idp.userinfoRequests.Load()
					response := audienceRequest(auth, token)
					require.Equal(t, tt.want, response.status)
					if tt.want == http.StatusUnauthorized {
						require.Nil(t, response.claims, "the protected handler must not run")
						require.Equal(t, before, idp.userinfoRequests.Load(), "reject before requesting userinfo")
						require.Empty(t, cache.writes, "rejected tokens must not be cached")
						return
					}
					require.Equal(t, "alice", response.claims["sub"])
					require.True(t, response.newSession)
					cache.waitForSession(t)
					expectedRequests := before
					if !skipUserInfo {
						expectedRequests++
					}
					require.Equal(t, expectedRequests, idp.userinfoRequests.Load())
					response = audienceRequest(auth, token)
					require.Equal(t, http.StatusOK, response.status)
					require.False(t, response.newSession)
					require.Equal(t, expectedRequests, idp.userinfoRequests.Load(), "reuse cached userinfo")
				})
			}
		})
	}
}

func TestOIDCAudiencePolicyChangesWithCachedUserinfo(t *testing.T) {
	for _, skipUserInfo := range []bool{false, true} {
		for _, initialAudiences := range [][]string{nil, {"immich"}} {
			t.Run(fmt.Sprintf("skip_user_info=%t/initial=%v", skipUserInfo, initialAudiences), func(t *testing.T) {
				idp := newAudienceTestIDP(t, "opencloud")
				cache := newAudienceTestCache()
				cfg := audienceTestConfig(idp, initialAudiences, skipUserInfo)
				token := idp.accessToken(t, jwt.MapClaims{"aud": "immich"})
				auth := newOIDCAuthenticator(log.NopLogger(), cfg, cache, idp.server.Client())
				require.Equal(t, http.StatusOK, audienceRequest(auth, token).status)
				cache.waitForSession(t)
				requests := idp.userinfoRequests.Load()

				// Reuse the store just like a persistent/shared cache after a proxy restart.
				cfg.OIDC.Audiences = []string{"opencloud"}
				auth = newOIDCAuthenticator(log.NopLogger(), cfg, cache, idp.server.Client())
				response := audienceRequest(auth, token)
				require.Equal(t, http.StatusUnauthorized, response.status)
				require.Nil(t, response.claims)
				require.Equal(t, requests, idp.userinfoRequests.Load())

				// Disabling the opt-in restores the existing cache behavior.
				cfg.OIDC.Audiences = nil
				auth = newOIDCAuthenticator(log.NopLogger(), cfg, cache, idp.server.Client())
				response = audienceRequest(auth, token)
				require.Equal(t, http.StatusOK, response.status)
				require.False(t, response.newSession)
			})
		}
	}
}

func TestOIDCAudienceUsesAccessTokenInsteadOfUserinfo(t *testing.T) {
	idp := newAudienceTestIDP(t, "different-userinfo-audience")
	cache := newAudienceTestCache()
	auth := newOIDCAuthenticator(log.NopLogger(), audienceTestConfig(idp, []string{"opencloud"}, false), cache, idp.server.Client())
	token := idp.accessToken(t, jwt.MapClaims{"aud": "opencloud"})
	require.Equal(t, http.StatusOK, audienceRequest(auth, token).status)
	cache.waitForSession(t)
	response := audienceRequest(auth, token)
	require.Equal(t, http.StatusOK, response.status)
	require.Equal(t, "different-userinfo-audience", response.claims["aud"])
	require.EqualValues(t, 1, idp.userinfoRequests.Load())
	require.EqualValues(t, 1, idp.discoveryRequests.Load())
	require.EqualValues(t, 1, idp.jwksRequests.Load())
}

func TestOIDCAudienceValidatesTokensBeforeCache(t *testing.T) {
	idp := newAudienceTestIDP(t, "opencloud")
	for _, tt := range []struct {
		name   string
		claims jwt.MapClaims
		mangle bool
	}{
		{name: "expired", claims: jwt.MapClaims{"exp": time.Now().Add(-time.Hour).Unix()}},
		{name: "not yet valid", claims: jwt.MapClaims{"nbf": time.Now().Add(time.Hour).Unix()}},
		{name: "wrong issuer", claims: jwt.MapClaims{"iss": "https://other.example"}},
		{name: "missing audience", claims: jwt.MapClaims{"aud": nil}},
		{name: "invalid signature", mangle: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			cache := newAudienceTestCache()
			token := idp.accessToken(t, tt.claims)
			if tt.mangle {
				parts := strings.Split(token, ".")
				sig, err := base64.RawURLEncoding.DecodeString(parts[2])
				require.NoError(t, err)
				sig[0] ^= 1
				parts[2] = base64.RawURLEncoding.EncodeToString(sig)
				token = strings.Join(parts, ".")
			}
			cached, err := msgpack.Marshal(map[string]any{
				"sub": "alice", "aud": "opencloud", "exp": time.Now().Add(time.Hour).Unix(),
			})
			require.NoError(t, err)
			require.NoError(t, cache.Store.Write(&store.Record{Key: audienceTokenCacheKey(token), Value: cached, Expiry: time.Hour}))
			auth := newOIDCAuthenticator(log.NopLogger(), audienceTestConfig(idp, []string{"opencloud"}, false), cache, idp.server.Client())
			require.Equal(t, http.StatusUnauthorized, audienceRequest(auth, token).status)
			require.Zero(t, idp.userinfoRequests.Load())
		})
	}
}

func TestOIDCAudienceRefreshesExpiredOrCorruptCachedClaims(t *testing.T) {
	for _, skipUserInfo := range []bool{false, true} {
		for _, corrupt := range []bool{false, true} {
			t.Run(fmt.Sprintf("skip_user_info=%t/corrupt=%t", skipUserInfo, corrupt), func(t *testing.T) {
				idp := newAudienceTestIDP(t, "opencloud")
				cache := newAudienceTestCache()
				token := idp.accessToken(t, nil)
				cached, err := msgpack.Marshal(map[string]any{"sub": "stale", "exp": time.Now().Add(-time.Hour).Unix()})
				require.NoError(t, err)
				if corrupt {
					cached = []byte{0xc1} // Reserved/invalid MessagePack marker.
				}
				require.NoError(t, cache.Store.Write(&store.Record{Key: audienceTokenCacheKey(token), Value: cached, Expiry: time.Hour}))
				auth := newOIDCAuthenticator(log.NopLogger(), audienceTestConfig(idp, []string{"opencloud"}, skipUserInfo), cache, idp.server.Client())
				response := audienceRequest(auth, token)
				require.Equal(t, http.StatusOK, response.status)
				require.Equal(t, "alice", response.claims["sub"])
				require.True(t, response.newSession)
				cache.waitForSession(t)
				require.False(t, audienceRequest(auth, token).newSession)
			})
		}
	}
}

func TestOIDCAudiencePreservesBackchannelLogout(t *testing.T) {
	for _, skipUserInfo := range []bool{false, true} {
		t.Run(fmt.Sprintf("skip_user_info=%t", skipUserInfo), func(t *testing.T) {
			idp := newAudienceTestIDP(t, "opencloud")
			cache := newAudienceTestCache()
			cfg := audienceTestConfig(idp, []string{"opencloud"}, skipUserInfo)
			auth := newOIDCAuthenticator(log.NopLogger(), cfg, cache, idp.server.Client())
			token := idp.accessToken(t, nil)
			require.Equal(t, http.StatusOK, audienceRequest(auth, token).status)
			cache.waitForSession(t)

			sessionKey, err := bcl.NewKey("alice", "session")
			require.NoError(t, err)
			records, err := cache.Read(sessionKey)
			require.NoError(t, err)
			require.Len(t, records, 1)
			require.Equal(t, audienceTokenCacheKey(token), string(records[0].Value))

			logoutClient := oidc.NewOIDCClient(
				oidc.WithLogger(log.NopLogger()),
				oidc.WithOidcIssuer(idp.server.URL),
				oidc.WithHTTPClient(idp.server.Client()),
				oidc.WithAccessTokenAudiences([]string{"opencloud"}),
			)
			routes := &staticroutes.StaticRouteHandler{
				Prefix: "/", Config: *cfg, Logger: log.NopLogger(), OidcClient: logoutClient,
				UserInfoCache: cache, Proxy: http.NotFoundHandler(),
			}
			// Subject logout invalidates all sessions, without requiring a user/event backend.
			logoutToken := idp.sign(t, jwt.MapClaims{
				"iss": idp.server.URL, "sub": "alice", "aud": "web-client",
				"events": map[string]any{"http://schemas.openid.net/event/backchannel-logout": map[string]any{}},
			})
			form := url.Values{"logout_token": {logoutToken}}
			req := httptest.NewRequest(http.MethodPost, "/backchannel_logout", strings.NewReader(form.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			response := httptest.NewRecorder()
			routes.Handler().ServeHTTP(response, req)
			require.Equal(t, http.StatusOK, response.Code, response.Body.String())
			_, err = cache.Read(sessionKey)
			require.ErrorIs(t, err, store.ErrNotFound)
			_, err = cache.Read(audienceTokenCacheKey(token))
			require.ErrorIs(t, err, store.ErrNotFound)
		})
	}
}

func TestOIDCAudienceStartupWarning(t *testing.T) {
	// Match log.NewLogger's global level while testing the per-service filter.
	previousLevel := zerolog.GlobalLevel()
	zerolog.SetGlobalLevel(zerolog.TraceLevel)
	t.Cleanup(func() { zerolog.SetGlobalLevel(previousLevel) })
	idp := newAudienceTestIDP(t, "opencloud")
	for _, tt := range []struct {
		name      string
		audiences []string
		level     zerolog.Level
		inactive  bool
		want      int
	}{
		{name: "disabled", level: zerolog.WarnLevel, want: 1},
		{name: "enabled", audiences: []string{"opencloud"}, level: zerolog.WarnLevel},
		{name: "filtered", level: zerolog.ErrorLevel},
		{name: "OIDC inactive", inactive: true, level: zerolog.WarnLevel},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var output bytes.Buffer
			logger := log.Logger{Logger: zerolog.New(&output).Level(tt.level)}
			cfg := audienceTestConfig(idp, tt.audiences, true)
			if tt.inactive {
				cfg.OIDC.Issuer = ""
			}
			cache := newAudienceTestCache()
			auth := newOIDCAuthenticator(logger, cfg, cache, idp.server.Client())
			if !tt.inactive {
				token := idp.accessToken(t, nil)
				require.Equal(t, http.StatusOK, audienceRequest(auth, token).status)
				cache.waitForSession(t)
				for range 3 {
					require.Equal(t, http.StatusOK, audienceRequest(auth, token).status)
				}
			}
			require.Equal(t, tt.want, strings.Count(output.String(), "PROXY_OIDC_AUDIENCES"))
			if tt.want == 1 {
				require.Contains(t, output.String(), "\"level\":\"warn\"")
			} else {
				require.Empty(t, output.String())
			}
		})
	}
}

type audienceTestIDP struct {
	server            *httptest.Server
	key               *rsa.PrivateKey
	discoveryRequests atomic.Int32
	jwksRequests      atomic.Int32
	userinfoRequests  atomic.Int32
}

func newAudienceTestIDP(t *testing.T, userinfoAudience string) *audienceTestIDP {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	idp := &audienceTestIDP{key: key}
	mux := http.NewServeMux()
	idp.server = httptest.NewServer(mux)
	t.Cleanup(idp.server.Close)
	mux.HandleFunc("/.well-known/openid-configuration", func(w http.ResponseWriter, r *http.Request) {
		idp.discoveryRequests.Add(1)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"issuer": idp.server.URL, "jwks_uri": idp.server.URL + "/jwks",
			"userinfo_endpoint":                     idp.server.URL + "/userinfo",
			"id_token_signing_alg_values_supported": []string{"RS256"},
		})
	})
	mux.HandleFunc("/jwks", func(w http.ResponseWriter, r *http.Request) {
		idp.jwksRequests.Add(1)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"keys": []any{map[string]any{
			"kty": "RSA", "kid": "test", "alg": "RS256", "use": "sig",
			"n": base64.RawURLEncoding.EncodeToString(key.N.Bytes()), "e": "AQAB",
		}}})
	})
	mux.HandleFunc("/userinfo", func(w http.ResponseWriter, r *http.Request) {
		idp.userinfoRequests.Add(1)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"sub": "alice", "preferred_username": "alice", "aud": userinfoAudience})
	})
	return idp
}

func (idp *audienceTestIDP) sign(t *testing.T, claims jwt.MapClaims) string {
	t.Helper()
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = "test"
	signed, err := token.SignedString(idp.key)
	require.NoError(t, err)
	return signed
}

func (idp *audienceTestIDP) accessToken(t *testing.T, overrides jwt.MapClaims) string {
	t.Helper()
	claims := jwt.MapClaims{
		"iss": idp.server.URL, "sub": "alice", "sid": "session", "aud": "opencloud",
		"exp": time.Now().Add(time.Hour).Unix(),
	}
	for key, value := range overrides {
		if value == nil {
			delete(claims, key)
		} else {
			claims[key] = value
		}
	}
	return idp.sign(t, claims)
}

func audienceTestConfig(idp *audienceTestIDP, audiences []string, skipUserInfo bool) *config.Config {
	cfg := defaults.FullDefaultConfig()
	cfg.OIDC.Issuer = idp.server.URL
	cfg.OIDC.Audiences = audiences
	cfg.OIDC.SkipUserInfo = skipUserInfo
	cfg.OIDC.JWKS = config.JWKS{} // No background refresh goroutines in tests.
	return cfg
}

type audienceTestResponse struct {
	status     int
	newSession bool
	claims     map[string]any
}

func audienceRequest(auth middleware.Authenticator, token string) audienceTestResponse {
	result := audienceTestResponse{}
	handler := middleware.Authentication([]middleware.Authenticator{auth})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		result.newSession = oidc.NewSessionFlagFromContext(r.Context())
		result.claims = oidc.FromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequest(http.MethodGet, "/protected", http.NoBody)
	req = req.WithContext(router.SetRoutingInfo(req.Context(), router.RoutingInfo{}))
	req.Header.Set("Authorization", "Bearer "+token)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, req)
	result.status = recorder.Code
	return result
}

// Wait for the asynchronous session write instead of sleeping or racing the cache.
type audienceTestCache struct {
	store.Store
	writes chan string
}

func newAudienceTestCache() *audienceTestCache {
	return &audienceTestCache{Store: store.NewMemoryStore(), writes: make(chan string, 16)}
}

func (cache *audienceTestCache) Write(record *store.Record, opts ...store.WriteOption) error {
	err := cache.Store.Write(record, opts...)
	if err == nil {
		cache.writes <- record.Key
	}
	return err
}

func (cache *audienceTestCache) waitForSession(t *testing.T) {
	t.Helper()
	key, err := bcl.NewKey("alice", "session")
	require.NoError(t, err)
	timer := time.NewTimer(5 * time.Second)
	defer timer.Stop()
	for {
		select {
		case written := <-cache.writes:
			if written == key {
				return
			}
		case <-timer.C:
			t.Fatal("timed out waiting for session cache write")
		}
	}
}

func audienceTokenCacheKey(token string) string {
	hash := make([]byte, 64)
	sha3.ShakeSum256(hash, []byte(token))
	return base64.URLEncoding.EncodeToString(hash)
}
