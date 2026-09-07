package middleware

import "net/http"

const (
	// PublicLinkTokenName is the query parameter and header carrying a public
	// link token on a request.
	PublicLinkTokenName = "public-token"

	// PublicLinkAuthHeader marks the outcome of a failed public link
	// authentication so a downstream service can tell "password required" from
	// "wrong password" when it renders the 401. The proxy sets it, the graph
	// service reads it.
	PublicLinkAuthHeader = "X-Public-Link-Auth"

	// PublicLinkPasswordRequired means the link is password protected and no
	// password was provided.
	PublicLinkPasswordRequired = "password-required"
	// PublicLinkInvalidPassword means a password was provided but rejected.
	PublicLinkInvalidPassword = "invalid-password"
)

// HasPublicLinkToken reports whether a public link token rides on the request,
// as a query parameter or header.
func HasPublicLinkToken(r *http.Request) bool {
	return r.URL.Query().Get(PublicLinkTokenName) != "" || r.Header.Get(PublicLinkTokenName) != ""
}
