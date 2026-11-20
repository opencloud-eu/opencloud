package staticroutes_test

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	userv1beta1 "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"
	"github.com/golang-jwt/jwt/v5"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	microevents "go-micro.dev/v4/events"
	"go-micro.dev/v4/store"

	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/pkg/oidc"
	"github.com/opencloud-eu/opencloud/services/proxy/pkg/config/defaults"
	"github.com/opencloud-eu/opencloud/services/proxy/pkg/middleware"
	"github.com/opencloud-eu/opencloud/services/proxy/pkg/router"
	"github.com/opencloud-eu/opencloud/services/proxy/pkg/staticroutes"
	bcl "github.com/opencloud-eu/opencloud/services/proxy/pkg/staticroutes/backchannellogout"
	backendmocks "github.com/opencloud-eu/opencloud/services/proxy/pkg/user/backend/mocks"
	"github.com/opencloud-eu/reva/v2/pkg/events"
)

func TestBackchannelLogoutRejectsEveryCoveredToken(t *testing.T) {
	for _, skipUserInfo := range []bool{false, true} {
		for _, mode := range []string{"subject", "session", "subject and session"} {
			t.Run(fmt.Sprintf("skip_user_info=%t/%s", skipUserInfo, mode), func(t *testing.T) {
				idp := newLogoutTestIDP(t)
				env := newLogoutTestEnv(idp, skipUserInfo)
				now := time.Now()
				first := idp.accessToken(t, "alice", "session-a", now, time.Hour)
				second := idp.accessToken(t, "alice", "session-a", now, 5*time.Minute)
				// The proxy never sees this one before the logout.
				unseen := idp.accessToken(t, "alice", "session-a", now.Add(-time.Second), time.Hour)
				otherSubject := "alice"
				if mode == "subject" {
					otherSubject = "bob"
				}
				unrelated := idp.accessToken(t, otherSubject, "session-b", now, time.Hour)
				for _, token := range []string{first, second, unrelated} {
					require.Equal(t, http.StatusOK, env.request(token))
				}
				env.waitForLookup(t, "alice", "session-a")

				subject, session := "alice", "session-a"
				if mode == "subject" {
					session = ""
				} else if mode == "session" {
					subject = ""
				}
				logout := idp.logoutToken(t, subject, session, now)
				require.Equal(t, http.StatusOK, env.logout(logout).Code)

				before := idp.userinfoRequests.Load()
				for _, token := range []string{first, second, unseen} {
					require.Equal(t, http.StatusUnauthorized, env.request(token))
				}
				require.Equal(t, before, idp.userinfoRequests.Load(), "covered tokens must be rejected before requesting userinfo")
				require.Equal(t, http.StatusOK, env.request(unrelated), "unrelated sessions must stay authenticated")

				// Repeated logout requests must succeed and keep the tokens rejected.
				require.Equal(t, http.StatusOK, env.logout(logout).Code)
				require.Equal(t, http.StatusUnauthorized, env.request(first))

				// Another proxy sharing the stores must reject the tokens as well.
				other := newLogoutTestEnvWithStores(idp, skipUserInfo, env.cache, env.markerStore)
				require.Equal(t, http.StatusUnauthorized, other.request(second))
			})
		}
	}
}

// Keycloak keeps issuing tokens with the same sid to a client with an offline
// token after a logout, and reuses the sid for a new login after revoking consent.
func TestBackchannelLogoutAcceptsTokensIssuedLater(t *testing.T) {
	for _, skipUserInfo := range []bool{false, true} {
		for _, mode := range []string{"subject", "session"} {
			t.Run(fmt.Sprintf("skip_user_info=%t/%s", skipUserInfo, mode), func(t *testing.T) {
				idp := newLogoutTestIDP(t)
				env := newLogoutTestEnv(idp, skipUserInfo)
				logoutAt := time.Now()
				old := idp.accessToken(t, "alice", "session-a", logoutAt, time.Hour)
				require.Equal(t, http.StatusOK, env.request(old))

				subject, session := "alice", "session-a"
				if mode == "subject" {
					session = ""
				}
				require.Equal(t, http.StatusOK, env.logout(idp.logoutToken(t, subject, session, logoutAt)).Code)
				require.Equal(t, http.StatusUnauthorized, env.request(old))

				newer := idp.accessToken(t, "alice", "session-a", logoutAt.Add(time.Second), time.Hour)
				require.Equal(t, http.StatusOK, env.request(newer))
				// The second request is served from the cached claims.
				env.waitForLookup(t, "alice", "session-a")
				require.Equal(t, http.StatusOK, env.request(newer))
			})
		}
	}
}

func TestBackchannelLogoutCoversTokensFromTheCache(t *testing.T) {
	idp := newLogoutTestIDP(t)
	env := newLogoutTestEnv(idp, true)
	now := time.Now()
	token := idp.accessToken(t, "alice", "session-a", now, time.Hour)
	require.Equal(t, http.StatusOK, env.request(token))
	env.waitForLookup(t, "alice", "session-a")

	// A failing cleanup leaves the cached claims in place, which must not help the token.
	env.cache.failDeletes(errors.New("delete failed"))
	require.Equal(t, http.StatusOK, env.logout(idp.logoutToken(t, "alice", "session-a", now)).Code)
	require.Equal(t, http.StatusUnauthorized, env.request(token))
}

// A userinfo response can contain anything, so it must not decide which logouts cover a token.
func TestBackchannelLogoutIgnoresClaimsFromUserinfo(t *testing.T) {
	idp := newLogoutTestIDP(t)
	idp.userinfoExtra = map[string]any{
		"sid": "other-session",
		"iat": time.Now().Add(24 * time.Hour).Unix(),
	}
	env := newLogoutTestEnv(idp, false)
	now := time.Now()
	token := idp.accessToken(t, "alice", "session-a", now, time.Hour)
	require.Equal(t, http.StatusOK, env.request(token))
	env.waitForLookup(t, "alice", "session-a")

	// Keep the cached claims, so the check has to use the record from the cache.
	env.cache.failDeletes(errors.New("delete failed"))
	require.Equal(t, http.StatusOK, env.logout(idp.logoutToken(t, "", "session-a", now)).Code)
	require.Equal(t, http.StatusUnauthorized, env.request(token))
}

func TestBackchannelLogoutIsIndependentOfNotifications(t *testing.T) {
	idp := newLogoutTestIDP(t)
	env := newLogoutTestEnv(idp, true)
	now := time.Now()
	token := idp.accessToken(t, "alice", "session-a", now, time.Hour)
	require.Equal(t, http.StatusOK, env.request(token))
	env.waitForLookup(t, "alice", "session-a")

	// The test environment has no events publisher, so publishing the event fails.
	require.Equal(t, http.StatusOK, env.logout(idp.logoutToken(t, "", "session-a", now)).Code)
	require.Equal(t, http.StatusUnauthorized, env.request(token))

	key, err := bcl.NewKey("alice", "session-a")
	require.NoError(t, err)
	_, err = env.cache.Read(key)
	require.ErrorIs(t, err, store.ErrNotFound, "a failed notification must not prevent the cleanup")
}

func TestBackchannelLogoutPublishesEvent(t *testing.T) {
	idp := newLogoutTestIDP(t)
	env := newLogoutTestEnv(idp, false)
	publisher := &logoutTestPublisher{}
	users := &backendmocks.UserBackend{}
	users.On("GetUserByClaims", mock.Anything, "username", "alice").
		Return(&userv1beta1.User{Id: &userv1beta1.UserId{OpaqueId: "alice-id"}}, "", nil)
	env.handler.EventsPublisher = publisher
	env.handler.UserProvider = users

	now := time.Now()
	token := idp.accessToken(t, "alice", "session-a", now, time.Hour)
	require.Equal(t, http.StatusOK, env.request(token))
	env.waitForLookup(t, "alice", "session-a")

	require.Equal(t, http.StatusOK, env.logout(idp.logoutToken(t, "alice", "session-a", now)).Code)

	require.Len(t, publisher.messages, 1)
	event, ok := publisher.messages[0].(events.BackchannelLogout)
	require.True(t, ok)
	require.Equal(t, "alice-id", event.Executant.GetOpaqueId())
	require.Equal(t, "session-a", event.SessionId)
}

func TestBackchannelLogoutReportsMarkerFailures(t *testing.T) {
	idp := newLogoutTestIDP(t)
	env := newLogoutTestEnv(idp, true)
	now := time.Now()
	token := idp.accessToken(t, "alice", "session-a", now, time.Hour)
	require.Equal(t, http.StatusOK, env.request(token))
	env.waitForLookup(t, "alice", "session-a")

	t.Run("a logout that cannot be recorded fails, so the IDP knows about it", func(t *testing.T) {
		env.markerStore.writeErr.Store(errors.New("write failed"))
		defer env.markerStore.writeErr.Store(nil)
		require.Equal(t, http.StatusBadRequest, env.logout(idp.logoutToken(t, "", "session-a", now)).Code)
	})

	t.Run("a marker that cannot be read rejects the request if there is no userinfo lookup", func(t *testing.T) {
		env.markerStore.readErr.Store(errors.New("read failed"))
		defer env.markerStore.readErr.Store(nil)
		require.Equal(t, http.StatusUnauthorized, env.request(token))
	})

	require.Equal(t, http.StatusOK, env.request(token), "the token must work again once the store has recovered")
}

// While the markers cannot be read, the IDP decides about every request.
func TestBackchannelLogoutFallsBackToUserinfo(t *testing.T) {
	showLogoutTestWarnings(t)
	idp := newLogoutTestIDP(t)
	env := newLogoutTestEnv(idp, false)
	token := idp.accessToken(t, "alice", "session-a", time.Now(), time.Hour)
	require.Equal(t, http.StatusOK, env.request(token))
	env.waitForLookup(t, "alice", "session-a")

	env.markerStore.readErr.Store(errors.New("read failed"))

	before := idp.userinfoRequests.Load()
	require.Equal(t, http.StatusOK, env.request(token))
	require.Equal(t, before+1, idp.userinfoRequests.Load(), "the cached claims must not be used")
	require.Contains(t, env.logs.String(), "could not read the backchannel logout markers")

	idp.rejectUserinfo.Store(true)
	require.Equal(t, http.StatusUnauthorized, env.request(token), "the IDP no longer accepts the token")

	idp.rejectUserinfo.Store(false)
	env.markerStore.readErr.Store(nil)
	before = idp.userinfoRequests.Load()
	require.Equal(t, http.StatusOK, env.request(token))
	require.Equal(t, before, idp.userinfoRequests.Load(), "the cached claims are used again once the store has recovered")
}

// A corrupt marker is not an outage, so the userinfo lookup must not replace it.
func TestBackchannelLogoutRejectsCorruptMarker(t *testing.T) {
	idp := newLogoutTestIDP(t)
	env := newLogoutTestEnv(idp, false)
	token := idp.accessToken(t, "alice", "session-a", time.Now(), time.Hour)

	key, err := bcl.NewKey("", "session-a")
	require.NoError(t, err)
	require.NoError(t, env.markerStore.Write(&store.Record{Key: key, Value: []byte("not-a-number")}))

	require.Equal(t, http.StatusUnauthorized, env.request(token))
	require.Zero(t, idp.userinfoRequests.Load())
}

func TestBackchannelLogoutRequiresIssueTimeInLogoutToken(t *testing.T) {
	idp := newLogoutTestIDP(t)
	env := newLogoutTestEnv(idp, true)
	logout := idp.sign(t, jwt.MapClaims{
		"iss": idp.server.URL, "sid": "session-a", "aud": "opencloud", "jti": "logout",
		"exp":    time.Now().Add(time.Minute).Unix(),
		"events": map[string]any{"http://schemas.openid.net/event/backchannel-logout": map[string]any{}},
	})
	require.Equal(t, http.StatusBadRequest, env.logout(logout).Code)
}

func TestAccessTokenWithoutIssueTime(t *testing.T) {
	idp := newLogoutTestIDP(t)
	env := newLogoutTestEnv(idp, true)
	token := idp.sign(t, jwt.MapClaims{
		"iss": idp.server.URL, "sub": "alice", "sid": "session-a", "aud": "opencloud",
		"exp": time.Now().Add(time.Hour).Unix(),
	})
	require.Equal(t, http.StatusOK, env.request(token), "without a logout the token is fine")
	env.waitForLookup(t, "alice", "session-a")

	require.Equal(t, http.StatusOK, env.logout(idp.logoutToken(t, "", "session-a", time.Now())).Code)
	require.Equal(t, http.StatusUnauthorized, env.request(token), "it cannot prove that it is newer than the logout")
}

// A token that can outlive its marker is accepted, but the operator is told about it.
func TestAccessTokenLifetimeWarning(t *testing.T) {
	showLogoutTestWarnings(t)
	idp := newLogoutTestIDP(t)
	env := newLogoutTestEnv(idp, true)
	now := time.Now()

	sign := func(claims jwt.MapClaims) string {
		claims["iss"], claims["sub"], claims["sid"], claims["aud"] = idp.server.URL, "alice", "session-a", "opencloud"
		return idp.sign(t, claims)
	}
	const warning = "a backchannel logout might not cover it"
	tests := []struct {
		name  string
		token string
		want  string
	}{
		{"at the limit", sign(jwt.MapClaims{"iat": now.Unix(), "exp": now.Add(logoutTestMaxLifetime).Unix()}), ""},
		{"above the limit", sign(jwt.MapClaims{"iat": now.Unix(), "exp": now.Add(logoutTestMaxLifetime + time.Second).Unix()}), "exceeds PROXY_OIDC_MAX_ACCESS_TOKEN_LIFETIME"},
		{"without expiration", sign(jwt.MapClaims{"iat": now.Unix()}), "has no expiration"},
		{"without iat, remaining lifetime above the limit", sign(jwt.MapClaims{"exp": now.Add(logoutTestMaxLifetime + time.Minute).Unix()}), "exceeds PROXY_OIDC_MAX_ACCESS_TOKEN_LIFETIME"},
		{"without iat, remaining lifetime within the limit", sign(jwt.MapClaims{"exp": now.Add(logoutTestMaxLifetime - time.Minute).Unix()}), ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env.logs.Reset()
			require.Equal(t, http.StatusOK, env.request(tt.token))
			if tt.want == "" {
				require.NotContains(t, env.logs.String(), warning)
				return
			}
			require.Contains(t, env.logs.String(), tt.want)
			require.Contains(t, env.logs.String(), warning)
		})
	}
}

// Cached claims written while a logout runs must not bring a token back.
func TestBackchannelLogoutDuringConcurrentRequests(t *testing.T) {
	idp := newLogoutTestIDP(t)
	env := newLogoutTestEnv(idp, false)
	now := time.Now()
	tokens := make([]string, 8)
	for i := range tokens {
		tokens[i] = idp.sign(t, jwt.MapClaims{
			"iss": idp.server.URL, "sub": "alice", "sid": "session-a", "aud": "opencloud", "jti": fmt.Sprintf("token-%d", i),
			"iat": now.Unix(), "exp": now.Add(time.Hour).Unix(),
		})
	}

	var wg sync.WaitGroup
	start := make(chan struct{})
	for _, token := range tokens {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			for range 10 {
				env.request(token)
			}
		}()
	}
	wg.Add(1)
	go func() {
		defer wg.Done()
		<-start
		if code := env.logout(idp.logoutToken(t, "", "session-a", now)).Code; code != http.StatusOK {
			t.Errorf("logout failed with status %d", code)
		}
	}()
	close(start)
	wg.Wait()

	// Give the cache writes of the last requests time to finish.
	time.Sleep(100 * time.Millisecond)
	for _, token := range tokens {
		require.Equal(t, http.StatusUnauthorized, env.request(token))
	}
}

const logoutTestMaxLifetime = 2 * time.Hour

type logoutTestIDP struct {
	server           *httptest.Server
	key              *rsa.PrivateKey
	userinfoRequests atomic.Int64
	userinfoExtra    map[string]any
	// rejectUserinfo makes the userinfo endpoint answer like an IDP that has ended the session.
	rejectUserinfo atomic.Bool
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
		if idp.rejectUserinfo.Load() {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		claims := jwt.MapClaims{}
		if _, _, err := new(jwt.Parser).ParseUnverified(strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer "), claims); err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		response := map[string]any{"sub": claims["sub"], "preferred_username": claims["sub"]}
		for key, value := range idp.userinfoExtra {
			response[key] = value
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response)
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

var logoutTestTokenID atomic.Int64

func (idp *logoutTestIDP) accessToken(t *testing.T, subject, session string, issuedAt time.Time, lifetime time.Duration) string {
	return idp.sign(t, jwt.MapClaims{
		"iss": idp.server.URL, "sub": subject, "sid": session, "aud": "opencloud",
		"jti": fmt.Sprintf("token-%d", logoutTestTokenID.Add(1)),
		"iat": issuedAt.Unix(), "exp": issuedAt.Add(lifetime).Unix(),
	})
}

func (idp *logoutTestIDP) logoutToken(t *testing.T, subject, session string, issuedAt time.Time) string {
	return idp.sign(t, jwt.MapClaims{
		"iss": idp.server.URL, "sub": subject, "sid": session, "aud": "opencloud", "jti": "logout",
		"iat": issuedAt.Unix(), "exp": time.Now().Add(time.Minute).Unix(),
		"events": map[string]any{"http://schemas.openid.net/event/backchannel-logout": map[string]any{}},
	})
}

type logoutTestPublisher struct {
	messages []any
}

func (p *logoutTestPublisher) Publish(_ string, message any, _ ...microevents.PublishOption) error {
	p.messages = append(p.messages, message)
	return nil
}

type logoutTestEnv struct {
	auth        middleware.Authenticator
	handler     *staticroutes.StaticRouteHandler
	routes      http.Handler
	cache       *logoutTestStore
	markerStore *logoutTestStore
	logs        *logoutTestLog
}

// logoutTestLog collects what the authenticator logs. Requests run concurrently in some tests.
type logoutTestLog struct {
	mu     sync.Mutex
	output bytes.Buffer
}

func (l *logoutTestLog) Write(p []byte) (int, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.output.Write(p)
}

func (l *logoutTestLog) String() string {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.output.String()
}

func (l *logoutTestLog) Reset() {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.output.Reset()
}

// showLogoutTestWarnings matches the global level of log.NewLogger, so that warnings reach the test log.
func showLogoutTestWarnings(t *testing.T) {
	t.Helper()
	previous := zerolog.GlobalLevel()
	zerolog.SetGlobalLevel(zerolog.TraceLevel)
	t.Cleanup(func() { zerolog.SetGlobalLevel(previous) })
}

func newLogoutTestEnv(idp *logoutTestIDP, skipUserInfo bool) *logoutTestEnv {
	return newLogoutTestEnvWithStores(idp, skipUserInfo,
		&logoutTestStore{Store: store.NewMemoryStore()},
		&logoutTestStore{Store: store.NewMemoryStore()},
	)
}

func newLogoutTestEnvWithStores(idp *logoutTestIDP, skipUserInfo bool, cache, markerStore *logoutTestStore) *logoutTestEnv {
	cfg := defaults.FullDefaultConfig()
	cfg.OIDC.Issuer = idp.server.URL
	cfg.OIDC.SkipUserInfo = skipUserInfo
	client, err := oidc.NewOIDCClient(
		oidc.WithOidcIssuer(idp.server.URL),
		oidc.WithHTTPClient(idp.server.Client()),
		oidc.WithAccessTokenVerifyMethod("jwt"),
		oidc.WithLogger(log.NopLogger()),
	)
	if err != nil {
		panic(err)
	}
	markers := bcl.NewLogoutMarkers(markerStore, logoutTestMaxLifetime+bcl.MarkerRetentionBuffer)
	logs := &logoutTestLog{}
	auth := middleware.NewOIDCAuthenticator(
		middleware.Logger(log.Logger{Logger: zerolog.New(logs).Level(zerolog.WarnLevel)}), middleware.UserInfoCache(cache), middleware.OIDCClient(client),
		middleware.LogoutMarkers(markers), middleware.MaxAccessTokenLifetime(logoutTestMaxLifetime),
		middleware.OIDCIss(idp.server.URL), middleware.SkipUserInfo(skipUserInfo),
		middleware.AccessTokenVerifyMethod("jwt"), middleware.DefaultAccessTokenTTL(time.Minute),
		middleware.HTTPClient(idp.server.Client()),
	)
	routes := &staticroutes.StaticRouteHandler{
		Prefix: "/", Config: *cfg, Logger: log.NopLogger(), OidcClient: client,
		UserInfoCache: cache, LogoutMarkers: markers, Proxy: http.NotFoundHandler(),
	}
	return &logoutTestEnv{auth: auth, handler: routes, routes: routes.Handler(), cache: cache, markerStore: markerStore, logs: logs}
}

func (env *logoutTestEnv) request(token string) int {
	handler := middleware.Authentication([]middleware.Authenticator{env.auth})(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) }))
	req := httptest.NewRequest(http.MethodGet, "/protected", http.NoBody)
	req = req.WithContext(router.SetRoutingInfo(req.Context(), router.RoutingInfo{}))
	req.Header.Set("Authorization", "Bearer "+token)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, req)
	return recorder.Code
}

func (env *logoutTestEnv) logout(token string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, "/backchannel_logout", strings.NewReader(url.Values{"logout_token": {token}}.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	recorder := httptest.NewRecorder()
	env.routes.ServeHTTP(recorder, req)
	return recorder
}

// waitForLookup waits until the proxy has written the lookup record, which happens after the response.
func (env *logoutTestEnv) waitForLookup(t *testing.T, subject, session string) {
	t.Helper()
	key, err := bcl.NewKey(subject, session)
	require.NoError(t, err)
	require.Eventually(t, func() bool {
		records, err := env.cache.Read(key)
		return err == nil && len(records) > 0
	}, 5*time.Second, 5*time.Millisecond)
	// The lookup is written after the claims, so the token is fully cached now.
}

type logoutTestStore struct {
	store.Store
	readErr, writeErr, deleteErr atomicError
}

func (s *logoutTestStore) failDeletes(err error) {
	s.deleteErr.Store(err)
}

func (s *logoutTestStore) Read(key string, opts ...store.ReadOption) ([]*store.Record, error) {
	if err := s.readErr.Load(); err != nil {
		return nil, err
	}
	return s.Store.Read(key, opts...)
}

func (s *logoutTestStore) Write(record *store.Record, opts ...store.WriteOption) error {
	if err := s.writeErr.Load(); err != nil {
		return err
	}
	return s.Store.Write(record, opts...)
}

func (s *logoutTestStore) Delete(key string, opts ...store.DeleteOption) error {
	if err := s.deleteErr.Load(); err != nil {
		return err
	}
	return s.Store.Delete(key, opts...)
}

type atomicError struct {
	value atomic.Pointer[error]
}

func (a *atomicError) Store(err error) {
	if err == nil {
		a.value.Store(nil)
		return
	}
	a.value.Store(&err)
}

func (a *atomicError) Load() error {
	if err := a.value.Load(); err != nil {
		return *err
	}
	return nil
}
