package middleware

import (
	"net/http"
	"strings"

	gateway "github.com/cs3org/go-cs3apis/cs3/gateway/v1beta1"
	"github.com/opencloud-eu/opencloud/pkg/log"
	revactx "github.com/opencloud-eu/reva/v2/pkg/ctx"
	"github.com/opencloud-eu/reva/v2/pkg/rgrpc/todo/pool"
)

const (
	headerRevaAccessToken   = revactx.TokenHeader
	headerShareToken        = "public-token"
	basicAuthPasswordPrefix = "password|"
	authenticationType      = "publicshares"

	_paramSignature  = "signature"
	_paramExpiration = "expiration"
)

// PublicShareAuthenticator is the authenticator which can authenticate public share requests.
// It will add the share owner into the request context.
type PublicShareAuthenticator struct {
	Logger              log.Logger
	RevaGatewaySelector pool.Selectable[gateway.GatewayAPIClient]
}

// The archiver is able to create archives from public shares in which case it needs to use the
// PublicShareAuthenticator. It might however also be called using "normal" authentication or
// using signed url, which are handled by other middleware. For this reason we can't just
// handle `/archiver` with the `isPublicPath()` check.
func isPublicShareArchive(r *http.Request) bool {
	if strings.HasPrefix(r.URL.Path, "/archiver") {
		if r.URL.Query().Get(headerShareToken) != "" || r.Header.Get(headerShareToken) != "" {
			return true
		}
	}
	return false
}

// The app open requests can be made in public share contexts. For that the PublicShareAuthenticator needs to
// augment the request context.
// The app open requests can also be made in authenticated context. In these cases the PublicShareAuthenticator
// needs to ignore the request.
func isPublicShareAppOpen(r *http.Request) bool {
	return (strings.HasPrefix(r.URL.Path, "/app/open") || strings.HasPrefix(r.URL.Path, "/app/new")) &&
		(r.URL.Query().Get(headerShareToken) != "" || r.Header.Get(headerShareToken) != "")
}

// The public-files requests can also be made in authenticated context. In these cases the OIDCAuthenticator and
// the BasicAuthenticator needs to ignore the request when the headerShareToken exist.
func isPublicWithShareToken(r *http.Request) bool {
	return (strings.HasPrefix(r.URL.Path, "/dav/public-files") || strings.HasPrefix(r.URL.Path, "/remote.php/dav/public-files")) &&
		(r.URL.Query().Get(headerShareToken) != "" || r.Header.Get(headerShareToken) != "")
}

// Authenticate implements the authenticator interface to authenticate requests via public share auth.
func (a PublicShareAuthenticator) Authenticate(r *http.Request) (*http.Request, bool) {
	if !isPublicPath(r.URL.Path) && !isPublicShareArchive(r) && !isPublicShareAppOpen(r) {
		return nil, false
	}

	query := r.URL.Query()
	shareToken := r.Header.Get(headerShareToken)
	if shareToken == "" {
		shareToken = query.Get(headerShareToken)
	}

	if shareToken == "" {
		// If the share token is not set then we don't need to inject the user to
		// the request context so we can just continue with the request.
		return r, true
	}

	var sharePassword string
	if signature := query.Get(_paramSignature); signature != "" {
		expiration := query.Get(_paramExpiration)
		if expiration == "" {
			a.Logger.Warn().Str("signature", signature).Msg("cannot do signature auth without the expiration")
			return nil, false
		}
		sharePassword = strings.Join([]string{"signature", signature, expiration}, "|")
	} else {
		// We can ignore the username since it is always set to "public" in public shares.
		_, password, ok := r.BasicAuth()

		sharePassword = basicAuthPasswordPrefix
		if ok {
			sharePassword += password
		}
	}

	client, err := a.RevaGatewaySelector.Next()
	if err != nil {
		a.Logger.Error().
			Err(err).
			Str("authenticator", "public_share").
			Str("public_share_token", shareToken).
			Str("path", r.URL.Path).
			Msg("could not select next gateway client")
		return nil, false
	}

	authResp, err := client.Authenticate(r.Context(), &gateway.AuthenticateRequest{
		Type:         authenticationType,
		ClientId:     shareToken,
		ClientSecret: sharePassword,
	})

	if err != nil {
		a.Logger.Error().
			Err(err).
			Str("authenticator", "public_share").
			Str("public_share_token", shareToken).
			Str("path", r.URL.Path).
			Msg("failed to authenticate request")
		return nil, false
	}

	r.Header.Add(headerRevaAccessToken, authResp.Token)

	a.Logger.Debug().
		Str("authenticator", "public_share").
		Str("path", r.URL.Path).
		Msg("successfully authenticated request")
	return r, true
}
