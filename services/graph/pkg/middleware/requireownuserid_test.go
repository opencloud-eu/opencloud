package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	userv1beta1 "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"
	"github.com/go-chi/chi/v5"
	"github.com/opencloud-eu/opencloud/pkg/log"
	revactx "github.com/opencloud-eu/reva/v2/pkg/ctx"
)

func TestRequireOwnUserID(t *testing.T) {
	router := chi.NewRouter()
	router.With(RequireOwnUserID("userID", log.NopLogger())).
		Post("/users/{userID}/action", dummyHandler{}.ServeHTTP)
	// a misconfigured route, whose parameter has another name
	router.With(RequireOwnUserID("userID", log.NopLogger())).
		Post("/users/{id}/other", dummyHandler{}.ServeHTTP)

	tests := []struct {
		name   string
		path   string
		user   string
		noUser bool // no user in the context at all
		status int
	}{
		{name: "own user id", path: "/users/alice/action", user: "alice", status: http.StatusOK},
		{name: "own user id, escaped", path: "/users/al%69ce/action", user: "alice", status: http.StatusOK},
		{name: "own user id with an escaped slash", path: "/users/a%2Fb/action", user: "a/b", status: http.StatusOK},
		{name: "other user id", path: "/users/bob/action", user: "alice", status: http.StatusBadRequest},
		// U+200B ZERO WIDTH SPACE after the user id
		{name: "own user id with an invisible character", path: "/users/alice%E2%80%8B/action", user: "alice", status: http.StatusBadRequest},
		// "e" followed by U+0301 COMBINING ACUTE ACCENT, the decomposed form of the "\u00e9" in the user id
		{name: "own user id in another unicode normalization form", path: "/users/jos%65%CC%81/action", user: "jos\u00e9", status: http.StatusBadRequest},
		{name: "own user id with different case", path: "/users/Alice/action", user: "alice", status: http.StatusBadRequest},
		{name: "invalid escape", path: "/users/al%zzce/action", user: "alice", status: http.StatusBadRequest},
		{name: "route without the parameter", path: "/users/alice/other", user: "alice", status: http.StatusBadRequest},
		{name: "user without an id, route without the parameter", path: "/users/alice/other", user: "", status: http.StatusBadRequest},
		{name: "no authenticated user", path: "/users/alice/action", noUser: true, status: http.StatusUnauthorized},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/", nil)
			// set the path afterwards: NewRequest rejects invalid escapes
			req.URL.Path = ""
			req.URL.RawPath = tt.path
			ctx := context.Background()
			if !tt.noUser {
				ctx = revactx.ContextSetUser(ctx, &userv1beta1.User{Id: &userv1beta1.UserId{OpaqueId: tt.user}})
			}
			rr := httptest.NewRecorder()
			router.ServeHTTP(rr, req.WithContext(ctx))
			if rr.Code != tt.status {
				t.Errorf("got status %d, want %d", rr.Code, tt.status)
			}
		})
	}
}
