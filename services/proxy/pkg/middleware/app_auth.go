package middleware

import (
	"net/http"

	gateway "github.com/cs3org/go-cs3apis/cs3/gateway/v1beta1"
	cs3rpc "github.com/cs3org/go-cs3apis/cs3/rpc/v1beta1"
	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/services/proxy/pkg/userroles"
	revactx "github.com/opencloud-eu/reva/v2/pkg/ctx"
	"github.com/opencloud-eu/reva/v2/pkg/rgrpc/todo/pool"
)

// AppAuthAuthenticator defines the app auth authenticator
type AppAuthAuthenticator struct {
	Logger              log.Logger
	RevaGatewaySelector pool.Selectable[gateway.GatewayAPIClient]
	UserRoleAssigner    userroles.UserRoleAssigner
}

// Authenticate implements the authenticator interface to authenticate requests via app auth.
func (m AppAuthAuthenticator) Authenticate(r *http.Request) (*http.Request, bool) {
	if isPublicPath(r.URL.Path) {
		// The authentication of public path requests is handled by another authenticator.
		// Since we can't guarantee the order of execution of the authenticators, we better
		// implement an early return here for paths we can't authenticate in this authenticator.
		return nil, false
	}

	username, password, ok := r.BasicAuth()
	if !ok {
		return nil, false
	}
	next, err := m.RevaGatewaySelector.Next()
	if err != nil {
		return nil, false
	}

	authenticateResponse, err := next.Authenticate(r.Context(), &gateway.AuthenticateRequest{
		Type:         "appauth",
		ClientId:     username,
		ClientSecret: password,
	})
	if err != nil {
		return nil, false
	}
	if authenticateResponse.GetStatus().GetCode() != cs3rpc.Code_CODE_OK {
		m.Logger.Debug().Str("msg", authenticateResponse.GetStatus().GetMessage()).Str("clientid", username).Msg("app auth failed")
		return nil, false
	}

	user := authenticateResponse.GetUser()
	if user, err = m.UserRoleAssigner.ApplyUserRole(r.Context(), user); err != nil {
		m.Logger.Error().Err(err).Str("clientid", username).Msg("app auth: failed to load user roles")
		return nil, false
	}

	ctx := revactx.ContextSetUser(r.Context(), user)
	ctx = revactx.ContextSetToken(ctx, authenticateResponse.GetToken())

	r = r.WithContext(ctx)

	return r, true
}
