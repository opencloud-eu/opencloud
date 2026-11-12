package middleware

import (
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/httprate"
	revactx "github.com/opencloud-eu/reva/v2/pkg/ctx"
)

var errNoAuthenticatedUser = errors.New("no authenticated user")

// RateLimitPerAuthenticatedUser limits the number of requests that each
// authenticated user can make within the given window. Requests beyond the
// limit get a 429 response with a Retry-After header. A limit of 0 or less
// disables the limit.
//
// It requires an authenticated user in the request context, and so must come
// after the middleware that authenticates the request: requests without one
// get a 401 response. The requests are counted by the id of that user, not by
// anything in the request, so that clients can't get around the limit by
// varying the URL.
//
// The counters are kept in memory, separately by every instance of the
// service, and by every call of this function: use one call per route that
// needs its own limit.
func RateLimitPerAuthenticatedUser(limit int, window time.Duration) func(http.Handler) http.Handler {
	if limit <= 0 {
		return func(next http.Handler) http.Handler {
			return next
		}
	}
	return httprate.LimitBy(limit, window, authenticatedUserID,
		httprate.WithErrorHandler(func(w http.ResponseWriter, _ *http.Request, _ error) {
			// the only error is a missing user: the default answer
			// would be a 428, which is misleading
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		}),
	)
}

// authenticatedUserID is the httprate.KeyFunc of RateLimitPerAuthenticatedUser.
func authenticatedUserID(r *http.Request) (string, error) {
	u, ok := revactx.ContextGetUser(r.Context())
	if !ok || u.GetId().GetOpaqueId() == "" {
		return "", errNoAuthenticatedUser
	}
	return u.GetId().GetOpaqueId(), nil
}
