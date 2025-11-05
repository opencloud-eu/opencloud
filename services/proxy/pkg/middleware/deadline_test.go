package middleware_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/services/proxy/pkg/middleware"
)

//
// ─── TESTS ─────────────────────────────────────────────────────────────
//

func TestReadDeadline(t *testing.T) {
	tests := []struct {
		name       string
		handler    http.Handler
		middleware middleware.Constructor
		request    func(*httptest.Server) (*http.Request, error)
		evaluate   func(*testing.T, *http.Response, error)
	}{
		{
			name: "OK",
			handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				_, err := io.Copy(w, r.Body)
				if err != nil {
					http.Error(w, http.StatusText(http.StatusRequestTimeout), http.StatusRequestTimeout)
					return
				}
			}),
			middleware: middleware.ReadDeadline(15*time.Millisecond, log.NopLogger()),
			request: func(srv *httptest.Server) (*http.Request, error) {
				return http.NewRequest("POST", srv.URL, &slowReader{inner: strings.NewReader("body"), delay: 10 * time.Millisecond})
			},
			evaluate: func(t *testing.T, response *http.Response, err error) {
				assert.NotNil(t, response)
				assert.NoError(t, err)
				assert.Equal(t, http.StatusOK, response.StatusCode)

				b, err := io.ReadAll(response.Body)
				assert.NoError(t, err)
				assert.Equal(t, string(b), "body")
			},
		},
		{
			name: "Pass through without timeout",
			handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				_, err := io.Copy(w, r.Body)
				if err != nil {
					http.Error(w, http.StatusText(http.StatusRequestTimeout), http.StatusRequestTimeout)
					return
				}
			}),
			middleware: middleware.ReadDeadline(0, log.NopLogger()),
			request: func(srv *httptest.Server) (*http.Request, error) {
				return http.NewRequest("POST", srv.URL, &slowReader{inner: strings.NewReader("body"), delay: 10 * time.Millisecond})
			},
			evaluate: func(t *testing.T, response *http.Response, err error) {
				assert.NotNil(t, response)
				assert.NoError(t, err)
				assert.Equal(t, http.StatusOK, response.StatusCode)

				b, err := io.ReadAll(response.Body)
				assert.NoError(t, err)
				assert.Equal(t, string(b), "body")
			},
		},
		{
			name: "ReadDeadline exceeded",
			handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				_, err := io.Copy(w, r.Body)
				if err != nil {
					http.Error(w, http.StatusText(http.StatusRequestTimeout), http.StatusRequestTimeout)
					return
				}
			}),
			middleware: middleware.ReadDeadline(5*time.Millisecond, log.NopLogger()),
			request: func(srv *httptest.Server) (*http.Request, error) {
				return http.NewRequest("POST", srv.URL, &slowReader{inner: strings.NewReader("test"), delay: 10 * time.Millisecond})
			},
			evaluate: func(t *testing.T, response *http.Response, err error) {
				assert.NotNil(t, response)
				assert.NoError(t, err)
				assert.Equal(t, http.StatusRequestTimeout, response.StatusCode)

				b, err := io.ReadAll(response.Body)
				assert.NoError(t, err)
				assert.Equal(t, string(b), http.StatusText(http.StatusRequestTimeout)+"\n")
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			srv := httptest.NewServer(tc.middleware(tc.handler))
			defer srv.Close()

			req, err := tc.request(srv)
			assert.NoError(t, err)

			resp, err := (&http.Client{}).Do(req)
			defer func() {
				if err != nil {
					assert.NoError(t, resp.Body.Close())
				}
			}()

			tc.evaluate(t, resp, err)
		})
	}
}

//
// ─── HELPERS ───────────────────────────────────────────────────────────
//

type slowReader struct {
	inner io.Reader
	delay time.Duration
}

func (r *slowReader) Read(p []byte) (n int, err error) {
	time.Sleep(r.delay)
	return r.inner.Read(p)
}
