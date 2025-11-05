package middleware_test

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/services/proxy/pkg/middleware"
)

func TestFilterChain(t *testing.T) {
	tests := []struct {
		name           string
		defaultHandler http.Handler
		middleware     middleware.Constructor
		request        func(*httptest.Server) (*http.Request, error)
		evaluate       func(*testing.T, *http.Response, error)
	}{
		{
			name: "it uses the default handler without a return",
			defaultHandler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				_, _ = w.Write([]byte("defaultHandler"))
			}),
			middleware: middleware.FilterChain(func(_ http.ResponseWriter, r *http.Request) ([]middleware.Constructor, error) {
				return nil, nil
			}, log.NopLogger()),
			request: func(srv *httptest.Server) (*http.Request, error) {
				return http.NewRequest("POST", srv.URL, strings.NewReader("body"))
			},
			evaluate: func(t *testing.T, response *http.Response, err error) {
				assert.NotNil(t, response)
				assert.NoError(t, err)
				assert.Equal(t, http.StatusOK, response.StatusCode)

				b, err := io.ReadAll(response.Body)
				assert.NoError(t, err)
				assert.Equal(t, string(b), "defaultHandler")
			},
		},
		{
			name: "it ignores nil constructors",
			defaultHandler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				_, _ = w.Write([]byte("defaultHandler"))
			}),
			middleware: middleware.FilterChain(func(_ http.ResponseWriter, r *http.Request) ([]middleware.Constructor, error) {
				return []middleware.Constructor{
					nil,
					nil,
					nil,
					nil,
					nil,
					nil,
					func(next http.Handler) http.Handler {
						return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
							_, _ = w.Write([]byte("mw1"))
							next.ServeHTTP(w, r)
						})
					},
					nil,
					nil,
					nil,
					func(next http.Handler) http.Handler {
						return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
							_, _ = w.Write([]byte("mw2"))
							next.ServeHTTP(w, r)
						})
					},
					nil,
					nil,
					nil,
				}, nil
			}, log.NopLogger()),
			request: func(srv *httptest.Server) (*http.Request, error) {
				return http.NewRequest("POST", srv.URL, strings.NewReader("body"))
			},
			evaluate: func(t *testing.T, response *http.Response, err error) {
				assert.NotNil(t, response)
				assert.NoError(t, err)
				assert.Equal(t, http.StatusOK, response.StatusCode)

				b, err := io.ReadAll(response.Body)
				assert.NoError(t, err)
				assert.Equal(t, string(b), "mw1mw2defaultHandler")
			},
		},
		{
			name: "it uses the returned handlers",
			defaultHandler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				_, _ = w.Write([]byte("defaultHandler"))
			}),
			middleware: middleware.FilterChain(func(_ http.ResponseWriter, r *http.Request) ([]middleware.Constructor, error) {
				return []middleware.Constructor{
					func(next http.Handler) http.Handler {
						return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
							_, _ = w.Write([]byte("mw1"))
							next.ServeHTTP(w, r)
						})
					},
					func(next http.Handler) http.Handler {
						return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
							_, _ = w.Write([]byte("mw2"))
							next.ServeHTTP(w, r)
						})
					},
				}, nil
			}, log.NopLogger()),
			request: func(srv *httptest.Server) (*http.Request, error) {
				return http.NewRequest("POST", srv.URL, strings.NewReader("body"))
			},
			evaluate: func(t *testing.T, response *http.Response, err error) {
				assert.NotNil(t, response)
				assert.NoError(t, err)
				assert.Equal(t, http.StatusOK, response.StatusCode)

				b, err := io.ReadAll(response.Body)
				assert.NoError(t, err)
				assert.Equal(t, string(b), "mw1mw2defaultHandler")
			},
		},
		{
			name: "it stops calling other middlewares if the factory fails short",
			defaultHandler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				_, _ = w.Write([]byte("defaultHandler"))
			}),
			middleware: middleware.FilterChain(func(w http.ResponseWriter, r *http.Request) ([]middleware.Constructor, error) {
				status := http.StatusInternalServerError
				http.Error(w, http.StatusText(status), status)

				return []middleware.Constructor{
					func(next http.Handler) http.Handler {
						return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
							_, _ = w.Write([]byte("mw1"))
							next.ServeHTTP(w, r)
						})
					},
				}, errors.New(http.StatusText(status))
			}, log.NopLogger()),
			request: func(srv *httptest.Server) (*http.Request, error) {
				return http.NewRequest("POST", srv.URL, strings.NewReader("body"))
			},
			evaluate: func(t *testing.T, response *http.Response, err error) {
				assert.NotNil(t, response)
				assert.NoError(t, err)
				assert.Equal(t, http.StatusInternalServerError, response.StatusCode)

				b, err := io.ReadAll(response.Body)
				assert.NoError(t, err)
				assert.Equal(t, string(bytes.TrimSpace(b)), http.StatusText(http.StatusInternalServerError))
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			srv := httptest.NewServer(tc.middleware(tc.defaultHandler))
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
