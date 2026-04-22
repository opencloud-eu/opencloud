package middleware

import (
	"net/http"

	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/pkg/oidc"
	"github.com/opencloud-eu/opencloud/services/proxy/pkg/user/backend"
)

// BasicAuthenticator is the authenticator responsible for HTTP Basic authentication.
type BasicAuthenticator struct {
	Logger        log.Logger
	UserProvider  backend.UserBackend
	UserCS3Claim  string
	UserOIDCClaim string
}

// Authenticate implements the authenticator interface to authenticate requests via basic auth.
func (m BasicAuthenticator) Authenticate(r *http.Request) (*http.Request, bool) {
	if isPublicPath(r.URL.Path) && isPublicWithShareToken(r) {
		// The authentication of public path requests is handled by another authenticator.
		// Since we can't guarantee the order of execution of the authenticators, we better
		// implement an early return here for paths we can't authenticate in this authenticator.
		return nil, false
	}

	login, password, ok := r.BasicAuth()
	if !ok {
		return nil, false
	}

	user, _, err := m.UserProvider.Authenticate(r.Context(), login, password)
	if err != nil {
		m.Logger.Error().
			Err(err).
			Str("authenticator", "basic").
			Str("path", r.URL.Path).
			Msg("failed to authenticate request")
		return nil, false
	}

	// fake oidc claims
	claims := map[string]any{
		oidc.Iss:               user.Id.Idp,
		oidc.PreferredUsername: user.Username,
		oidc.Email:             user.Mail,
		oidc.OpenCloudUUID:     user.Id.OpaqueId,
	}

	if m.UserCS3Claim == "userid" {
		// set the custom user claim only if users will be looked up by the userid on the CS3api
		// OpaqueId contains the userid configured in STORAGE_LDAP_USER_SCHEMA_UID
		claims[m.UserOIDCClaim] = user.Id.OpaqueId

	}
	m.Logger.Debug().
		Str("authenticator", "basic").
		Str("path", r.URL.Path).
		Msg("successfully authenticated request")
	return r.WithContext(oidc.NewContext(r.Context(), claims)), true
}
