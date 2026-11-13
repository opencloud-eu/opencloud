package middleware

import (
	"net/http"
	"net/url"

	"github.com/go-chi/chi/v5"
	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/services/graph/pkg/errorcode"
	revactx "github.com/opencloud-eu/reva/v2/pkg/ctx"
)

// RequireOwnUserID rejects requests whose URL parameter urlParam, the user id
// in the route pattern (e.g. "userID" for "/users/{userID}"), isn't the ID of
// the authenticated user, for endpoints that let users act only on
// themselves. A route without that parameter rejects every request. It must
// come after the authentication, and before any middleware that keeps state
// per user, such as a rate limit, so that requests for other users don't
// count against them.
func RequireOwnUserID(urlParam string, logger log.Logger) func(next http.Handler) http.Handler {
	l := log.Logger{Logger: logger.With().Str("middleware", "requireOwnUserID").Logger()}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			u, ok := revactx.ContextGetUser(r.Context())
			if !ok {
				errorcode.AccessDenied.Render(w, r, http.StatusUnauthorized, "Unauthorized")
				return
			}
			userID, err := url.PathUnescape(chi.URLParam(r, urlParam))
			if err != nil {
				l.Debug().Err(err).Str("userid", chi.URLParam(r, urlParam)).Msg("could not unescape the user id")
				errorcode.InvalidRequest.Render(w, r, http.StatusBadRequest, "invalid user id")
				return
			}
			if userID == "" || userID != u.GetId().GetOpaqueId() {
				l.Info().Str("userid", u.GetId().GetOpaqueId()).Str("requestedUserID", userID).Msg("user id in the request is not the user's own")
				// 403 would arguably be the more fitting status code, but
				// clients may rely on the 400 that this has always returned.
				errorcode.InvalidRequest.Render(w, r, http.StatusBadRequest, "not permitted for other users")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
