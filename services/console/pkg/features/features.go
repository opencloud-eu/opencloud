//go:build !release

package features

import (
	"crypto/tls"
	"net/http"
)

var (
	DefaultTLSConfig     = &tls.Config{InsecureSkipVerify: true}
	DefaultHTTPTransport = http.DefaultTransport.(*http.Transport).Clone()
)

func init() {
	DefaultHTTPTransport.TLSClientConfig = DefaultTLSConfig
}
