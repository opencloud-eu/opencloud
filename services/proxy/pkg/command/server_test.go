package command

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	gofig "github.com/gookit/config/v2"
	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/pkg/shared"
	"github.com/opencloud-eu/opencloud/services/proxy/pkg/config"
	"github.com/opencloud-eu/opencloud/services/proxy/pkg/config/defaults"
	"github.com/opencloud-eu/opencloud/services/proxy/pkg/metrics"
	"github.com/opencloud-eu/opencloud/services/proxy/pkg/middleware"
	"github.com/opencloud-eu/reva/v3/pkg/audit"
	"github.com/stretchr/testify/require"
	"go-micro.dev/v4/selector"
	microstore "go-micro.dev/v4/store"
	"go.opentelemetry.io/otel/trace/noop"
)

var (
	// metrics.New registers its collectors globally, so it may only run once per test binary.
	testMetricsOnce sync.Once
	testMetrics     *metrics.Metrics
	testMetricsErr  error
)

// newTestMetrics returns metrics for the routes of the default config.
func newTestMetrics(t *testing.T) metrics.Metrics {
	t.Helper()
	testMetricsOnce.Do(func() {
		cfg := defaults.FullDefaultConfig()
		logger := log.NopLogger()
		testMetrics, testMetricsErr = metrics.New(func(yield func(config.Route) bool) {
			for _, pol := range cfg.Policies {
				for _, r := range pol.Routes {
					if !yield(r) {
						return
					}
				}
			}
		}, &logger)
	})
	require.NoError(t, testMetricsErr)
	return *testMetrics
}

func TestUnauthenticatedRequestChallenges(t *testing.T) {
	const (
		basicChallenge  = `Basic realm="cloud.example.test", charset="UTF-8"`
		bearerChallenge = `Bearer realm="cloud.example.test", charset="UTF-8"`
	)

	for _, tt := range []struct {
		name            string
		enableBasicAuth bool
		allowAppAuth    bool
		want            []string
	}{
		{
			name:         "app tokens only",
			allowAppAuth: true,
			want:         []string{bearerChallenge, basicChallenge},
		},
		{
			name:            "basic auth only",
			enableBasicAuth: true,
			want:            []string{bearerChallenge, basicChallenge},
		},
		{
			name:            "basic auth and app tokens",
			enableBasicAuth: true,
			allowAppAuth:    true,
			want:            []string{bearerChallenge, basicChallenge},
		},
		{
			name: "neither basic auth nor app tokens",
			want: []string{bearerChallenge},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			// Setting up the middlewares changes package-level state: the challenges are
			// appended to a package variable, and the CSP config is loaded into the global
			// gookit config, which refuses to be set up a second time.
			saved := middleware.SupportedAuthStrategies
			middleware.SupportedAuthStrategies = nil
			gofig.Reset()
			t.Cleanup(func() {
				middleware.SupportedAuthStrategies = saved
				gofig.Reset()
			})

			cfg := defaults.FullDefaultConfig()
			cfg.EnableBasicAuth = tt.enableBasicAuth
			cfg.AuthMiddleware.AllowAppAuth = tt.allowAppAuth
			cfg.OIDC.JWKS = config.JWKS{} // No background refresh goroutines in tests.
			cfg.Commons = &shared.Commons{URLSigningSecret: "test-url-secret"}

			chain, err := loadMiddlewares(
				log.NopLogger(), cfg,
				microstore.NewMemoryStore(), microstore.NewMemoryStore(),
				noop.NewTracerProvider(), newTestMetrics(t),
				nil, nil, audit.Discard{}, nil, selector.NewSelector(),
			)
			require.NoError(t, err)

			handler := chain.ThenFunc(func(w http.ResponseWriter, _ *http.Request) {
				t.Error("an unauthenticated request must not reach the backend")
			})

			req := httptest.NewRequest(http.MethodGet, "https://cloud.example.test/graph/v1.0/me", nil)
			req.Header.Set("User-Agent", "challenge-test-client")
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			require.Equal(t, http.StatusUnauthorized, rec.Code)
			require.Equal(t, tt.want, rec.Header().Values(middleware.WwwAuthenticate))
		})
	}
}
