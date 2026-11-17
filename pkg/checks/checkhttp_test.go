package checks

import (
	"context"
	"net"
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
		localAddr, remoteAddr, userAgent string
		want                             bool
	}{
		"check to the pod ip":       {"10.21.130.74:9200", "10.21.130.74:51234", HealthCheckUserAgent, true},
		"check to the loopback":     {"127.0.0.1:9200", "127.0.0.1:51234", HealthCheckUserAgent, true},
		"check to ipv6":             {"[fd00::5]:9200", "[fd00::5]:51234", HealthCheckUserAgent, true},
		"ingress sending the agent": {"10.21.130.74:9200", "10.21.64.12:51234", HealthCheckUserAgent, false},
		"client sending the agent":  {"10.21.130.74:9200", "203.0.113.7:51234", HealthCheckUserAgent, false},
		"local browser":             {"127.0.0.1:9200", "127.0.0.1:51234", "Mozilla/5.0", false},
	} {
		r := httptest.NewRequest(http.MethodHead, "/", nil)
		local, err := net.ResolveTCPAddr("tcp", tc.localAddr)
		require.NoError(t, err)
		r = r.WithContext(context.WithValue(r.Context(), http.LocalAddrContextKey, local))
		r.RemoteAddr = tc.remoteAddr
		r.Header.Set("User-Agent", tc.userAgent)
		assert.Equal(t, tc.want, IsLocalHealthCheck(r), name)
	}
}
