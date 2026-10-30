package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/opencloud-eu/opencloud/pkg/oidc"
	"github.com/opencloud-eu/reva/v2/pkg/audit"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/metadata"
)

// serve runs AuditContext and returns the request the next handler sees.
func serve(t *testing.T, r *http.Request) *http.Request {
	t.Helper()
	var got *http.Request
	AuditContext(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r
	})).ServeHTTP(httptest.NewRecorder(), r)
	require.NotNil(t, got)
	return got
}

func TestAuditContextDropsClientHeader(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/graph/v1.0/me", nil)
	r.RemoteAddr = "203.0.113.7:51234"
	r.Header.Set(audit.Header, "ip=10.0.0.1&imp=admin")

	got := serve(t, r)

	want := audit.RequestContext{ClientAddress: "203.0.113.7"}
	assert.Equal(t, want.Encode(), got.Header.Get(audit.Header))
	rc, ok := audit.ContextGetRequest(got.Context())
	require.True(t, ok)
	assert.Equal(t, want, rc)
}

func TestAuditContextOIDC(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/graph/v1.0/me", nil)
	r.RemoteAddr = "203.0.113.7"
	r.Header.Set("User-Agent", "Safari")
	r = r.WithContext(oidc.NewContext(r.Context(), map[string]any{"sid": "s-1", "azp": "web"}))
	r = withAuthMethod(r, &OIDCAuthenticator{})

	got := serve(t, r)

	want := audit.RequestContext{ClientAddress: "203.0.113.7", AuthMethod: AuthMethodOIDC, SessionID: "s-1", ClientID: "web", UserAgent: "Safari"}
	rc, ok := audit.ContextGetRequest(got.Context())
	require.True(t, ok)
	assert.Equal(t, want, rc)
	assert.Equal(t, want.Encode(), got.Header.Get(audit.Header))

	md, ok := metadata.FromOutgoingContext(got.Context())
	require.True(t, ok, "gRPC calls of the proxy carry the context too")
	assert.Equal(t, []string{want.Encode()}, md.Get(audit.Header))
}

func TestAuditContextClientIDFallback(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r = r.WithContext(oidc.NewContext(r.Context(), map[string]any{"client_id": "desktop"}))
	r = withAuthMethod(r, &OIDCAuthenticator{})

	rc, ok := audit.ContextGetRequest(serve(t, r).Context())
	require.True(t, ok)
	assert.Equal(t, "desktop", rc.ClientID)
}

func TestAuditContextIgnoresClaimsOfOtherMethods(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r = r.WithContext(oidc.NewContext(r.Context(), map[string]any{"sid": "s-1"}))
	r = withAuthMethod(r, AppAuthAuthenticator{})

	rc, ok := audit.ContextGetRequest(serve(t, r).Context())
	require.True(t, ok)
	assert.Equal(t, AuthMethodAppToken, rc.AuthMethod)
	assert.Empty(t, rc.SessionID)
}

func TestAuthMethodOf(t *testing.T) {
	for _, tc := range []struct {
		a    Authenticator
		want string
	}{
		{&OIDCAuthenticator{}, AuthMethodOIDC},
		{BasicAuthenticator{}, AuthMethodBasic},
		{AppAuthAuthenticator{}, AuthMethodAppToken},
		{PublicShareAuthenticator{}, AuthMethodPublicLink},
		{&GuestLinkAuthenticator{}, AuthMethodGuestLink},
		{SignedURLAuthenticator{}, AuthMethodSignedURL},
	} {
		assert.Equal(t, tc.want, authMethodOf(tc.a))
	}
}
