//go:build release

package features

import (
	"crypto/tls"
	"net/http"
)

var (
	DefaultTLSConfig     *tls.Config = nil
	DefaultHTTPTransport             = http.DefaultTransport.(*http.Transport).Clone()
)
