package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go-micro.dev/v4/store"

	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/pkg/oidc"
	oidcmocks "github.com/opencloud-eu/opencloud/pkg/oidc/mocks"
	bcl "github.com/opencloud-eu/opencloud/services/proxy/pkg/staticroutes/backchannellogout"
	"github.com/opencloud-eu/reva/v3/pkg/audit"
)

// tokenClient verifies every token as a token of the given session and client.
func tokenClient(t *testing.T, tokens map[string][2]string) *oidcmocks.OIDCClient {
	client := &oidcmocks.OIDCClient{}
	exp := time.Now().Add(time.Hour)
	for token, sc := range tokens {
		claims := jwt.MapClaims{"sub": "alice", "exp": exp.Unix()}
		if sc[0] != "" {
			claims["sid"] = sc[0]
		}
		if sc[1] != "" {
			claims["azp"] = sc[1]
		}
		client.On("VerifyAccessToken", mock.Anything, token).Return(oidc.RegClaimsWithSID{
			SessionID:        sc[0],
			RegisteredClaims: jwt.RegisteredClaims{Subject: "alice", ExpiresAt: jwt.NewNumericDate(exp)},
		}, claims, nil)
	}
	return client
}

func authenticate(t *testing.T, a *OIDCAuthenticator, token string) (newSession, newLogin bool) {
	t.Helper()
	r := httptest.NewRequest(http.MethodGet, "/", http.NoBody)
	r.Header.Set("Authorization", "Bearer "+token)
	res := a.Authenticate(r)
	require.Equal(t, AuthenticationSucceeded, res.State, token)
	ctx := res.Request.Context()
	return oidc.NewSessionFlagFromContext(ctx), newLoginFromContext(ctx)
}

func TestNewLoginOncePerSessionAndClient(t *testing.T) {
	client := tokenClient(t, map[string][2]string{
		"web-1":     {"s-1", "web"},
		"web-2":     {"s-1", "web"}, // renewed token of the same session
		"desktop-1": {"s-1", "desktop"},
		"other-1":   {"s-2", "web"},
		"nosid-1":   {"", "web"},
		"nosid-2":   {"", "web"},
	})
	a := NewOIDCAuthenticator(
		Logger(log.NopLogger()), OIDCClient(client), UserInfoCache(store.NewMemoryStore()),
		SkipUserInfo(true), OIDCIss("https://idp.example.org"), AuditEmitter(newRecordingEmitter()),
	)

	for _, tc := range []struct {
		token string
		login bool
	}{
		{"web-1", true},
		{"web-2", false},
		{"desktop-1", true},
		{"other-1", true},
		{"nosid-1", true},
		{"nosid-2", true},
	} {
		newSession, newLogin := authenticate(t, a, tc.token)
		assert.True(t, newSession, "every new token still flags a new session for the sign-in event: %s", tc.token)
		assert.Equal(t, tc.login, newLogin, tc.token)
	}
}

func TestNewLoginNeedsAudit(t *testing.T) {
	client := tokenClient(t, map[string][2]string{"web-1": {"s-1", "web"}})
	cache := store.NewMemoryStore()
	a := NewOIDCAuthenticator(
		Logger(log.NopLogger()), OIDCClient(client), UserInfoCache(cache),
		SkipUserInfo(true), OIDCIss("https://idp.example.org"), AuditEmitter(audit.Discard{}),
	)

	_, newLogin := authenticate(t, a, "web-1")

	assert.False(t, newLogin)
	keys, err := cache.List()
	require.NoError(t, err)
	for _, k := range keys {
		assert.False(t, strings.HasPrefix(k, "auditlogin-"), "no login records without audit")
	}
}

func TestLoginKeyIsNotFoundByTheBackchannelLogout(t *testing.T) {
	cache := store.NewMemoryStore()
	sessionKey, err := bcl.NewKey("alice", "s-1")
	require.NoError(t, err)
	require.NoError(t, cache.Write(&store.Record{Key: sessionKey, Value: []byte("claims")}))
	require.NoError(t, cache.Write(&store.Record{Key: loginKey("s-1", "web"), Value: []byte{1}}))
	assert.NotContains(t, loginKey("s-1", "web"), ".")

	bySubject, _ := bcl.NewKey("alice", "")
	bySession, _ := bcl.NewKey("", "s-1")
	for _, key := range []string{bySubject, bySession} {
		suse, err := bcl.NewSuSe(key)
		require.NoError(t, err)
		records, err := bcl.GetLogoutRecords(suse, cache)
		require.NoError(t, err)
		require.Len(t, records, 1, key)
		assert.Equal(t, sessionKey, records[0].Key)
	}
}
