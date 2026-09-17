package command

import (
	"net/http"
	"net/http/httptest"
	"testing"

	chimiddleware "github.com/go-chi/chi/v5/middleware"

	"github.com/opencloud-eu/opencloud/services/proxy/pkg/config"
	"gotest.tools/v3/assert"
)

// TestClientIPMiddleware verifies that clientIPMiddleware maps the configured
// strategy to the right chi ClientIPFrom* middleware and that the resolved IP
// matches the expected behaviour for each supported deployment case.
func TestClientIPMiddleware(t *testing.T) {
	tests := []struct {
		name       string
		strategy   string
		cfg        config.ClientIP
		remoteAddr string
		headers    map[string]string
		expected   string
	}{
		{
			name:       "remote_addr direct exposure",
			strategy:   "remote_addr",
			remoteAddr: "192.0.2.1:1234",
			expected:   "192.0.2.1",
		},
		{
			name:       "remote_addr ignores spoofed xff",
			strategy:   "remote_addr",
			remoteAddr: "192.0.2.1:1234",
			headers:    map[string]string{"X-Forwarded-For": "203.0.113.9"},
			expected:   "192.0.2.1",
		},
		{
			name:       "header strategy reads trusted header",
			strategy:   "header",
			cfg:        config.ClientIP{Header: "X-Real-IP"},
			remoteAddr: "192.0.2.1:1234",
			headers:    map[string]string{"X-Real-IP": "198.51.100.7"},
			expected:   "198.51.100.7",
		},
		{
			name:       "xff rightmost entry wins",
			strategy:   "xff",
			remoteAddr: "192.0.2.1:1234",
			headers:    map[string]string{"X-Forwarded-For": "198.51.100.10, 203.0.113.5"},
			expected:   "203.0.113.5",
		},
		{
			name:       "xff skips trusted prefixes",
			strategy:   "xff",
			cfg:        config.ClientIP{TrustedPrefixes: []string{"10.0.0.0/8"}},
			remoteAddr: "10.0.0.5:1234",
			headers:    map[string]string{"X-Forwarded-For": "198.51.100.10, 10.0.0.5"},
			expected:   "198.51.100.10",
		},
		{
			name:       "xff_trusted_hops reads nth hop",
			strategy:   "xff_trusted_hops",
			cfg:        config.ClientIP{TrustedHops: 1},
			remoteAddr: "10.0.0.5:1234",
			headers:    map[string]string{"X-Forwarded-For": "198.51.100.10, 10.0.0.5"},
			expected:   "10.0.0.5",
		},
		{
			name:       "unknown strategy falls back to remote_addr",
			strategy:   "something_else",
			remoteAddr: "192.0.2.1:1234",
			headers:    map[string]string{"X-Forwarded-For": "203.0.113.9"},
			expected:   "192.0.2.1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := tt.cfg
			cfg.Strategy = tt.strategy

			var got string
			handler := clientIPMiddleware(&config.Config{ClientIP: cfg})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				got = chimiddleware.GetClientIP(r.Context())
			}))

			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.RemoteAddr = tt.remoteAddr
			for k, v := range tt.headers {
				req.Header.Set(k, v)
			}

			handler.ServeHTTP(httptest.NewRecorder(), req)

			assert.Equal(t, got, tt.expected)
		})
	}
}
