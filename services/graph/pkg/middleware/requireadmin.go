package middleware

import (
	"net/http"

	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/pkg/roles"
	"github.com/opencloud-eu/opencloud/services/graph/pkg/errorcode"
	settings "github.com/opencloud-eu/opencloud/services/settings/pkg/service/v0"
	revactx "github.com/opencloud-eu/reva/v2/pkg/ctx"
)

// RequireAdmin middleware is used to require the user in context to be an admin / have account management permissions
func RequireAdmin(rm *roles.Manager, logger log.Logger) func(next http.Handler) http.Handler {
	l := log.Logger{Logger: logger.With().Str("middleware", "requireAdmin").Logger()}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			u, ok := revactx.ContextGetUser(r.Context())
			if !ok {
				errorcode.AccessDenied.Render(w, r, http.StatusUnauthorized, "Unauthorized")
				return
			}
			if u.GetId().GetOpaqueId() == "" {
				l.Debug().Msg("Bad request: user does not have an id")
				errorcode.InvalidRequest.Render(w, r, http.StatusBadRequest, "user is missing an id")
				return
			}
			// get roles from context
			roleIDs, ok := roles.ReadRoleIDsFromContext(r.Context())
			if !ok {
				l.Debug().Str("userid", u.Id.OpaqueId).Msg("No roles in context, contacting settings service")
				var err error
				roleIDs, err = rm.FindRoleIDsForUser(r.Context(), u.Id.OpaqueId)
				if err != nil {
					l.Error().Err(err).Str("userid", u.Id.OpaqueId).Msg("Failed to get roles for user")
					errorcode.AccessDenied.Render(w, r, http.StatusUnauthorized, "Unauthorized")
					return
				}
				if len(roleIDs) == 0 {
					l.Error().Err(err).Str("userid", u.Id.OpaqueId).Msg("No roles assigned to user")
					errorcode.AccessDenied.Render(w, r, http.StatusUnauthorized, "Unauthorized")
					return
				}
			}

			// check if permission is present in roles of the authenticated account
			if rm.FindPermissionByID(r.Context(), roleIDs, settings.AccountManagementPermissionID) != nil {
				next.ServeHTTP(w, r)
				return
			}

			l.Debug().Str("userid", u.Id.OpaqueId).Str("permission", settings.AccountManagementPermissionID).Msg("Access denied: necessary permission not present in user's roles")
			errorcode.AccessDenied.Render(w, r, http.StatusForbidden, "Forbidden")
		})
	}
}
