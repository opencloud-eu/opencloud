package svc

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	userv1beta1 "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"
	"github.com/go-chi/chi/v5"
	revactx "github.com/opencloud-eu/reva/v2/pkg/ctx"
	"github.com/stretchr/testify/assert"

	"github.com/opencloud-eu/opencloud/pkg/log"
)

// The handler checks the user id on its own as well, so that a route without
// the RequireOwnUserID middleware still can't export the data of other users.
// It is called directly here, without any middleware, and with a Graph that
// has no gateway: a request that got past the check would panic.
func TestExportPersonalDataRejectsOtherUsersWithoutMiddleware(t *testing.T) {
	logger := log.NopLogger()
	g := Graph{BaseGraphService: BaseGraphService{logger: &logger}}

	for _, tc := range []struct {
		name    string
		user    string
		urlUser string
	}{
		{name: "other user", user: "alice", urlUser: "bob"},
		{name: "invalid escape", user: "alice", urlUser: "al%zzce"},
		{name: "user without an id and no user id in the URL", user: "", urlUser: ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("userID", tc.urlUser)
			ctx := context.WithValue(context.Background(), chi.RouteCtxKey, rctx)
			ctx = revactx.ContextSetUser(ctx, &userv1beta1.User{Id: &userv1beta1.UserId{OpaqueId: tc.user}})

			rr := httptest.NewRecorder()
			g.ExportPersonalData(rr, httptest.NewRequest(http.MethodPost, "/", nil).WithContext(ctx))

			assert.Equal(t, http.StatusBadRequest, rr.Code)
		})
	}
}
