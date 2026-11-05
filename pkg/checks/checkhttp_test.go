package checks

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewHTTPCheck(t *testing.T) {
	var got *http.Request
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer srv.Close()

	require.NoError(t, NewHTTPCheck(srv.URL)(context.Background()), "any answer counts")
	assert.Equal(t, http.MethodHead, got.Method)
	assert.True(t, IsLocalHealthCheck(got))
}

func TestIsLocalHealthCheck(t *testing.T) {
	for name, tc := range map[string]struct {
		remoteAddr, userAgent string
		want                  bool
	}{
		"check from ipv4 loopback": {"127.0.0.1:51234", HealthCheckUserAgent, true},
		"check from ipv6 loopback": {"[::1]:51234", HealthCheckUserAgent, true},
		"client sending the agent": {"203.0.113.7:51234", HealthCheckUserAgent, false},
		"proxy in front":           {"10.0.0.5:51234", HealthCheckUserAgent, false},
		"local browser":            {"127.0.0.1:51234", "Mozilla/5.0", false},
	} {
		r := httptest.NewRequest(http.MethodHead, "/", nil)
		r.RemoteAddr = tc.remoteAddr
		r.Header.Set("User-Agent", tc.userAgent)
		assert.Equal(t, tc.want, IsLocalHealthCheck(r), name)
	}
}
