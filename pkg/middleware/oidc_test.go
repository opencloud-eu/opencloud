package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOidcAuth(t *testing.T) {
	// the self-signed certificate of the test server is rejected by the default
	// http client, so the provider initialization always fails
	srv := httptest.NewTLSServer(http.NotFoundHandler())
	defer srv.Close()

	newHandler := func(called *bool) http.Handler {
		next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			*called = true
			w.WriteHeader(http.StatusOK)
		})
		return OidcAuth(
			WithLogger(log.NewLogger()),
			WithOidcIssuer(srv.URL),
			WithHttpClient(http.Client{}),
		)(next)
	}

	t.Run("failed provider initialization is retried and never reaches next handler", func(t *testing.T) {
		nextCalled := false
		handler := newHandler(&nextCalled)

		for range 2 {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.Header.Set("Authorization", "Bearer x")
			rec := httptest.NewRecorder()

			require.NotPanics(t, func() { handler.ServeHTTP(rec, req) })
			assert.Equal(t, http.StatusInternalServerError, rec.Code)
		}
		assert.False(t, nextCalled)
	})

	t.Run("request without authorization header passes through", func(t *testing.T) {
		nextCalled := false
		handler := newHandler(&nextCalled)

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.True(t, nextCalled)
	})
}
