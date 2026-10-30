package proxy

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"testing"

	pkgmiddleware "github.com/opencloud-eu/opencloud/pkg/middleware"
	"github.com/opencloud-eu/opencloud/services/proxy/pkg/middleware"
	"gotest.tools/v3/assert"
)

func rewrite(t *testing.T, in *http.Request) *http.Request {
	t.Helper()
	rp := newTestProxy(testConfig(nil), func(req *http.Request) *http.Response {
		return &http.Response{StatusCode: 200, Body: io.NopCloser(bytes.NewBufferString("OK")), Header: make(http.Header)}
	})

	// Take the datagateway skip path so Rewrite does not need routing info.
	in = in.WithContext(context.WithValue(in.Context(), middleware.DatagatewaySkipRoutingKey, true))
	pr := &httputil.ProxyRequest{
		In:  in,
		Out: in.Clone(context.Background()),
	}
	rp.Rewrite(pr)
	return pr.Out
}

// TestRewriteForwardsClientIP verifies that the services get the client
// address in X-Forwarded-For, not the reverse proxy in front of us.
func TestRewriteForwardsClientIP(t *testing.T) {
	trusted, err := pkgmiddleware.ParseTrustedProxies([]string{"10.0.0.0/8"})
	assert.NilError(t, err)

	in := httptest.NewRequest(http.MethodGet, "/", nil)
	in.RemoteAddr = "10.0.0.5:443"
	in.Header.Set("X-Forwarded-For", "198.51.100.1, 203.0.113.7")

	var withCtx *http.Request
	pkgmiddleware.ClientIP(trusted, pkgmiddleware.DefaultClientIPHeader)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		withCtx = r
	})).ServeHTTP(httptest.NewRecorder(), in)

	out := rewrite(t, withCtx)

	assert.Equal(t, out.Header.Get("X-Forwarded-For"), "203.0.113.7")
}

// TestRewriteForwardsPeerWithoutClientIP verifies the peer is forwarded when
// the client IP middleware did not run, and a header sent by the client is
// replaced.
func TestRewriteForwardsPeerWithoutClientIP(t *testing.T) {
	in := httptest.NewRequest(http.MethodGet, "/", nil)
	in.RemoteAddr = "192.0.2.1:1234"
	in.Header.Set("X-Forwarded-For", "203.0.113.9")

	out := rewrite(t, in)

	assert.Equal(t, out.Header.Get("X-Forwarded-For"), "192.0.2.1")
}
