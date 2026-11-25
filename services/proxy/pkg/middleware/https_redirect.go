package middleware

import (
	"net/http"
	"net/url"

	"github.com/opencloud-eu/opencloud/pkg/log"
)

// HTTPSRedirect returns a middleware that redirects requests forwarded as
// plain HTTP (X-Forwarded-Proto: http) to HTTPS.
//
// The redirect target uses the host of baseURL, the URL where OpenCloud is
// reachable for users (OC_URL), never the Host header of the request: the
// client controls that header, so a redirect built from it could send users
// to any other site. If baseURL can't be parsed or has no host, requests are
// passed on without a redirect, and the reason is logged once.
func HTTPSRedirect(logger log.Logger, baseURL string) func(http.Handler) http.Handler {
	var host string
	u, err := url.Parse(baseURL)
	switch {
	case err != nil:
		logger.Error().Err(err).Str("url", baseURL).Msg("cannot parse the OpenCloud URL, requests forwarded as HTTP are not redirected to HTTPS")
	case u.Host == "":
		logger.Warn().Str("url", baseURL).Msg("the OpenCloud URL has no host, requests forwarded as HTTP are not redirected to HTTPS")
	default:
		host = u.Host
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
			// A reverse proxy in front of OpenCloud that terminates TLS sets
			// X-Forwarded-Proto to the protocol the client used to reach the
			// reverse proxy, not the one between the reverse proxy and
			// OpenCloud. "http" means the client connected without TLS, so it
			// is sent to the HTTPS URL instead. When the client then connects
			// over HTTPS, the reverse proxy sets the header to "https", and
			// the request is passed on without another redirect. Requests
			// without the header come straight from the client and are passed
			// on as well.
			proto := req.Header.Get("X-Forwarded-Proto")
			if host != "" && (proto == "http" || proto == "HTTP") {
				// Only the path and the query are taken from the request.
				// RawPath keeps the path as the client escaped it, e.g. a %2F
				// in a file name. The user info is left out on purpose: clients
				// send credentials in the Authorization header, and a user info
				// in the request URL would be echoed back in the Location
				// header, including a password.
				target := url.URL{
					Scheme:   "https",
					Host:     host,
					Path:     req.URL.Path,
					RawPath:  req.URL.RawPath,
					RawQuery: req.URL.RawQuery,
				}
				http.Redirect(res, req, target.String(), http.StatusPermanentRedirect)
				return
			}

			next.ServeHTTP(res, req)
		})
	}
}
