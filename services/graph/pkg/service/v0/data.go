package svc

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"

	"github.com/go-chi/chi/v5"
	revactx "github.com/opencloud-eu/reva/v3/pkg/ctx"

	"github.com/opencloud-eu/opencloud/services/graph/pkg/errorcode"
)

// HTTPDataHandler returns data from the request, it should exit early and return false in the case of any error
type HTTPDataHandler[T any] func(w http.ResponseWriter, r *http.Request) (T, bool)

var (
	// ErrNoUser is returned when no user is found
	ErrNoUser = errors.New("no user found")
)

// GetUserIDFromCTX extracts the user from the request
func GetUserIDFromCTX(w http.ResponseWriter, r *http.Request) (string, bool) {
	u, ok := revactx.ContextGetUser(r.Context())
	if !ok {
		errorcode.GeneralException.Render(w, r, http.StatusMethodNotAllowed, ErrNoUser.Error())
	}

	return u.GetId().GetOpaqueId(), ok
}

func GetSlugValue(key string) HTTPDataHandler[string] {
	return func(w http.ResponseWriter, r *http.Request) (string, bool) {
		v, err := url.PathUnescape(chi.URLParam(r, key))
		if err != nil {
			errorcode.GeneralException.Render(w, r, http.StatusInternalServerError, fmt.Sprintf(`failed to get slug: "%s"`, key))
		}

		return v, err == nil
	}
}
