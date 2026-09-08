package proxy

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"testing"

	chimiddleware "github.com/go-chi/chi/v5/middleware"

	pkgmiddleware "github.com/opencloud-eu/opencloud/pkg/middleware"
	"github.com/opencloud-eu/opencloud/services/proxy/pkg/middleware"
	"gotest.tools/v3/assert"
)

// TestRewriteForwardsClientIP verifies that the proxy overwrites the forwarded
// IP header on the outbound request with the client IP resolved by the
// ClientIPFrom* middleware, so downstream services can trust it.
func TestRewriteForwardsClientIP(t *testing.T) {
	cfg := testConfig(nil)

	rp := newTestProxy(cfg, func(req *http.Request) *http.Response {
		return &http.Response{StatusCode: 200, Body: io.NopCloser(bytes.NewBufferString("OK")), Header: make(http.Header)}
	})

	inReq := httptest.NewRequest(http.MethodGet, "/", nil)
	inReq.RemoteAddr = "192.0.2.1:1234"

	// Resolve the client IP into the request context via a chi middleware.
	var withCtx *http.Request
	chimiddleware.ClientIPFromRemoteAddr(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		withCtx = r
	})).ServeHTTP(httptest.NewRecorder(), inReq)

	// Take the datagateway skip path so Rewrite does not need routing info.
	ctx := context.WithValue(withCtx.Context(), middleware.DatagatewaySkipRoutingKey, true)
	withCtx = withCtx.WithContext(ctx)

	pr := &httputil.ProxyRequest{
		In:  withCtx,
		Out: withCtx.Clone(context.Background()),
	}
	rp.Rewrite(pr)

	assert.Equal(t, pr.Out.Header.Get(pkgmiddleware.DefaultClientIPHeader), "192.0.2.1")
}

// TestRewriteDoesNotForwardEmptyClientIP verifies the header is left unset when
// the client IP could not be resolved (no middleware populated the context).
func TestRewriteDoesNotForwardEmptyClientIP(t *testing.T) {
	cfg := testConfig(nil)

	rp := newTestProxy(cfg, func(req *http.Request) *http.Response {
		return &http.Response{StatusCode: 200, Body: io.NopCloser(bytes.NewBufferString("OK")), Header: make(http.Header)}
	})

	inReq := httptest.NewRequest(http.MethodGet, "/", nil)
	inReq.RemoteAddr = "192.0.2.1:1234"

	ctx := context.WithValue(inReq.Context(), middleware.DatagatewaySkipRoutingKey, true)
	inReq = inReq.WithContext(ctx)

	pr := &httputil.ProxyRequest{
		In:  inReq,
		Out: inReq.Clone(context.Background()),
	}
	rp.Rewrite(pr)

	assert.Equal(t, pr.Out.Header.Get(pkgmiddleware.DefaultClientIPHeader), "")
}
