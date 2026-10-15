package middleware

import "net/http"

const (
	// PublicLinkTokenName is the query parameter and header carrying a public link token.
	PublicLinkTokenName = "public-token"

	// PublicLinkAuthHeader carries why a public link authentication failed.
	PublicLinkAuthHeader = "X-Public-Link-Auth"

	// PublicLinkPasswordRequired means the link is password protected and no
	// password was provided.
	PublicLinkPasswordRequired = "password-required"
	// PublicLinkInvalidPassword means a password was provided but rejected.
	PublicLinkInvalidPassword = "invalid-password"
)

// HasPublicLinkToken reports whether the request carries a public link token.
func HasPublicLinkToken(r *http.Request) bool {
	return r.URL.Query().Get(PublicLinkTokenName) != "" || r.Header.Get(PublicLinkTokenName) != ""
}
