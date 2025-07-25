package middleware

import (
	"net/http"

	gmmetadata "go-micro.dev/v4/metadata"
	"google.golang.org/grpc/metadata"

	"github.com/opencloud-eu/opencloud/pkg/account"
	"github.com/opencloud-eu/opencloud/pkg/log"
	opkgm "github.com/opencloud-eu/opencloud/pkg/middleware"
	"github.com/opencloud-eu/reva/v2/pkg/auth/scope"
	revactx "github.com/opencloud-eu/reva/v2/pkg/ctx"
	"github.com/opencloud-eu/reva/v2/pkg/token/manager/jwt"
)

// authOptions initializes the available default options.
func authOptions(opts ...account.Option) account.Options {
	opt := account.Options{}

	for _, o := range opts {
		o(&opt)
	}

	return opt
}

func Auth(opts ...account.Option) func(http.Handler) http.Handler {
	opt := authOptions(opts...)
	tokenManager, err := jwt.New(map[string]any{
		"secret":  opt.JWTSecret,
		"expires": int64(24 * 60 * 60),
	})
	if err != nil {
		opt.Logger.Fatal().Err(err).Msgf("Could not initialize token-manager")
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			t := r.Header.Get(revactx.TokenHeader)
			if t == "" {
				opt.Logger.Error().Str(log.RequestIDString, r.Header.Get("X-Request-ID")).Msgf("missing access token in header %v", revactx.TokenHeader)
				w.WriteHeader(http.StatusUnauthorized) // missing access token
				return
			}

			u, tokenScope, err := tokenManager.DismantleToken(r.Context(), t)
			if err != nil {
				opt.Logger.Error().Str(log.RequestIDString, r.Header.Get("X-Request-ID")).Err(err).Msgf("invalid access token in header %v", revactx.TokenHeader)
				w.WriteHeader(http.StatusUnauthorized) // invalid token
				return
			}
			if ok, err := scope.VerifyScope(ctx, tokenScope, r); err != nil || !ok {
				opt.Logger.Error().Str(log.RequestIDString, r.Header.Get("X-Request-ID")).Err(err).Msg("verifying scope failed")
				w.WriteHeader(http.StatusUnauthorized) // invalid scope
				return
			}

			ctx = revactx.ContextSetToken(ctx, t)
			ctx = revactx.ContextSetUser(ctx, u)
			ctx = gmmetadata.Set(ctx, opkgm.AccountID, u.GetId().GetOpaqueId())
			if m := u.GetOpaque().GetMap(); m != nil {
				if roles, ok := m["roles"]; ok {
					ctx = gmmetadata.Set(ctx, opkgm.RoleIDs, string(roles.GetValue()))
				}
			}
			ctx = metadata.AppendToOutgoingContext(ctx, revactx.TokenHeader, t)

			initiatorID := r.Header.Get(revactx.InitiatorHeader)
			if initiatorID != "" {
				ctx = revactx.ContextSetInitiator(ctx, initiatorID)
				ctx = metadata.AppendToOutgoingContext(ctx, revactx.InitiatorHeader, initiatorID)
			}

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
