package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	chimiddleware "github.com/go-chi/chi/v5/middleware"

	"gotest.tools/v3/assert"
)

// TestClientIPChain verifies the order in which the chi ClientIP middlewares
// are chained for the proxy: ClientIPFromRemoteAddr runs first and provides the
// fallback, ClientIPFromXFF overwrites it with the rightmost X-Forwarded-For
// value when the header is present. This mirrors the alice chain configured in
// services/proxy/pkg/command/server.go.
func TestClientIPChain(t *testing.T) {
	tests := []struct {
		name       string
		remoteAddr string
		forwarded  []string
		expected   string
	}{
		{
			name:       "reverse proxy sets X-Forwarded-For",
			remoteAddr: "192.0.2.1:1234",
			forwarded:  []string{"198.51.100.10"},
			expected:   "198.51.100.10",
		},
		{
			name:       "rightmost forwarded entry wins",
			remoteAddr: "192.0.2.1:1234",
			forwarded:  []string{"198.51.100.10, 203.0.113.5"},
			expected:   "203.0.113.5",
		},
		{
			name:       "direct connection falls back to remote addr",
			remoteAddr: "192.0.2.1:1234",
			expected:   "192.0.2.1",
		},
		{
			name:       "unparseable forwarded falls back to remote addr",
			remoteAddr: "192.0.2.1:1234",
			forwarded:  []string{"not-an-ip"},
			expected:   "192.0.2.1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got string
			handler := chimiddleware.ClientIPFromRemoteAddr(chimiddleware.ClientIPFromXFF()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				got = chimiddleware.GetClientIP(r.Context())
			})))

			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.RemoteAddr = tt.remoteAddr
			for _, f := range tt.forwarded {
				req.Header.Add("X-Forwarded-For", f)
			}

			handler.ServeHTTP(httptest.NewRecorder(), req)

			assert.Equal(t, got, tt.expected)
		})
	}
}
