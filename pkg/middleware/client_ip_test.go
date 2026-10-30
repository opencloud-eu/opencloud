package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClientIP(t *testing.T) {
	trusted, err := ParseTrustedProxies([]string{"10.0.0.0/8", "192.0.2.10", "fd00::/8"})
	require.NoError(t, err)

	for _, tc := range []struct {
		name    string
		header  string
		remote  string
		xff     []string
		xRealIP string
		want    string
	}{
		{name: "direct client, no headers", remote: "203.0.113.7:51234", want: "203.0.113.7"},
		{name: "direct client spoofs X-Forwarded-For", remote: "203.0.113.7:51234", xff: []string{"198.51.100.1"}, want: "203.0.113.7"},
		{name: "direct client spoofs the configured header", header: "X-Real-IP", remote: "203.0.113.7:51234", xRealIP: "198.51.100.2", want: "203.0.113.7"},
		{name: "one trusted proxy", remote: "10.0.0.5:443", xff: []string{"203.0.113.7"}, want: "203.0.113.7"},
		{name: "client prepends a fake address", remote: "10.0.0.5:443", xff: []string{"198.51.100.1, 203.0.113.7"}, want: "203.0.113.7"},
		{name: "chain of trusted proxies", remote: "10.0.0.5:443", xff: []string{"203.0.113.7, 192.0.2.10", "10.1.2.3"}, want: "203.0.113.7"},
		{name: "only trusted addresses", remote: "10.0.0.5:443", xff: []string{"10.9.9.9, 10.1.2.3"}, want: "10.0.0.5"},
		{name: "broken entry", remote: "10.0.0.5:443", xff: []string{"203.0.113.7, garbage"}, want: "10.0.0.5"},
		{name: "trusted proxy without a header", remote: "10.0.0.5:443", want: "10.0.0.5"},
		{name: "X-Real-IP is ignored by default", remote: "10.0.0.5:443", xRealIP: "203.0.113.7", want: "10.0.0.5"},
		{name: "configured header from a trusted proxy", header: "x-real-ip", remote: "10.0.0.5:443", xff: []string{"198.51.100.1"}, xRealIP: "203.0.113.7", want: "203.0.113.7"},
		{name: "IPv6 behind an IPv6 proxy", remote: "[fd00::1]:443", xff: []string{"2001:db8::7"}, want: "2001:db8::7"},
		{name: "IPv4 mapped remote address", remote: "[::ffff:10.0.0.5]:443", xff: []string{"203.0.113.7"}, want: "203.0.113.7"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			r.RemoteAddr = tc.remote
			for _, v := range tc.xff {
				r.Header.Add("X-Forwarded-For", v)
			}
			if tc.xRealIP != "" {
				r.Header.Set("X-Real-Ip", tc.xRealIP)
			}
			header := tc.header
			if header == "" {
				header = DefaultClientIPHeader
			}

			var got, peer string
			ClientIP(trusted, header)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				got = ClientAddress(r)
				peer = r.RemoteAddr
			})).ServeHTTP(httptest.NewRecorder(), r)

			assert.Equal(t, tc.want, got)
			assert.Equal(t, tc.remote, peer, "RemoteAddr stays the peer")
		})
	}
}

func TestClientIPWithoutTrustedProxies(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.RemoteAddr = "127.0.0.1:443"
	r.Header.Set("X-Forwarded-For", "203.0.113.7")

	var got string
	ClientIP(nil, DefaultClientIPHeader)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = ClientAddress(r)
	})).ServeHTTP(httptest.NewRecorder(), r)

	assert.Equal(t, "127.0.0.1", got)
}

func TestClientAddressWithoutMiddleware(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.RemoteAddr = "203.0.113.7:51234"
	assert.Equal(t, "203.0.113.7", ClientAddress(r))
}

func TestParseTrustedProxies(t *testing.T) {
	p, err := ParseTrustedProxies([]string{" 10.0.0.1 ", "172.16.0.0/12", "", "::1"})
	require.NoError(t, err)
	assert.Len(t, p, 3)

	_, err = ParseTrustedProxies(DefaultTrustedProxies())
	require.NoError(t, err)

	_, err = ParseTrustedProxies([]string{"10.0.0.0/33"})
	assert.Error(t, err)
	_, err = ParseTrustedProxies([]string{"proxy.example.com"})
	assert.Error(t, err)
}
