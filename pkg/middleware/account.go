package middleware

import (
	"encoding/json"
	"net/http"

	"github.com/opencloud-eu/reva/v2/pkg/auth/scope"

	"github.com/opencloud-eu/opencloud/pkg/account"
	revactx "github.com/opencloud-eu/reva/v2/pkg/ctx"
	"github.com/opencloud-eu/reva/v2/pkg/token/manager/jwt"
	"go-micro.dev/v4/metadata"
)

// newAccountOptions initializes the available default options.
func newAccountOptions(opts ...account.Option) account.Options {
	opt := account.Options{}

	for _, o := range opts {
		o(&opt)
	}

	return opt
}

// AccountID serves as key for the account uuid in the context
const AccountID string = "Account-Id"

// RoleIDs serves as key for the roles in the context
const RoleIDs string = "Role-Ids"

// ExtractAccountUUID provides a middleware to extract the account uuid from the x-access-token header value
// and write it to the context. If there is no x-access-token the middleware is omitted.
func ExtractAccountUUID(opts ...account.Option) func(http.Handler) http.Handler {
	opt := newAccountOptions(opts...)
	tokenManager, err := jwt.New(map[string]interface{}{
		"secret":  opt.JWTSecret,
		"expires": int64(24 * 60 * 60),
	})
	if err != nil {
		opt.Logger.Fatal().Err(err).Msgf("Could not initialize token-manager")
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := r.Header.Get(revactx.TokenHeader)
			if len(token) == 0 {
				roleIDsJSON, _ := json.Marshal([]string{})
				ctx := metadata.Set(r.Context(), RoleIDs, string(roleIDsJSON))
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}

			u, tokenScope, err := tokenManager.DismantleToken(r.Context(), token)
			if err != nil {
				opt.Logger.Error().Err(err)
				return
			}
			if ok, err := scope.VerifyScope(r.Context(), tokenScope, r); err != nil || !ok {
				opt.Logger.Error().Err(err).Msg("verifying scope failed")
				return
			}

			// store user in context for request
			ctx := revactx.ContextSetUser(r.Context(), u)

			// Important: user.Id.OpaqueId is the AccountUUID. Set this way in the account uuid middleware in OpenCloud proxy.
			// https://github.com/opencloud-eu/opencloud-proxy/blob/ea254d6036592cf9469d757d1295e0c4309d1e63/pkg/middleware/account_uuid.go#L109
			// TODO: implement token manager in cs3org/reva that uses generic metadata instead of access token from header.
			ctx = metadata.Set(ctx, AccountID, u.Id.OpaqueId)
			if u.Opaque != nil {
				if roles, ok := u.Opaque.Map["roles"]; ok {
					ctx = metadata.Set(ctx, RoleIDs, string(roles.Value))
				}
			}
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
