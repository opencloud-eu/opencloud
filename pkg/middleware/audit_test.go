package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/opencloud-eu/reva/v2/pkg/audit"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/metadata"
)

func TestAuditContext(t *testing.T) {
	want := audit.RequestContext{ClientAddress: "203.0.113.7", AuthMethod: "oidc", SessionID: "s-1"}
	r := httptest.NewRequest(http.MethodGet, "/graph/v1.0/me", nil)
	r.Header.Set(audit.Header, want.Encode())

	var got *http.Request
	AuditContext(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { got = r })).
		ServeHTTP(httptest.NewRecorder(), r)

	rc, ok := audit.ContextGetRequest(got.Context())
	require.True(t, ok)
	assert.Equal(t, want, rc)
	md, ok := metadata.FromOutgoingContext(got.Context())
	require.True(t, ok)
	assert.Equal(t, []string{want.Encode()}, md.Get(audit.Header))
}

func TestAuditContextWithoutHeader(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	var got *http.Request
	AuditContext(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { got = r })).
		ServeHTTP(httptest.NewRecorder(), r)

	_, ok := audit.ContextGetRequest(got.Context())
	assert.False(t, ok)
}
