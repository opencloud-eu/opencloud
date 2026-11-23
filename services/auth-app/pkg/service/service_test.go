package service

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	applications "github.com/cs3org/go-cs3apis/cs3/auth/applications/v1beta1"
	gateway "github.com/cs3org/go-cs3apis/cs3/gateway/v1beta1"
	userpb "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"
	rpc "github.com/cs3org/go-cs3apis/cs3/rpc/v1beta1"
	"github.com/go-chi/chi/v5"
	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/pkg/middleware"
	settingsmsg "github.com/opencloud-eu/opencloud/protogen/gen/opencloud/messages/settings/v0"
	settingssvc "github.com/opencloud-eu/opencloud/protogen/gen/opencloud/services/settings/v0"
	"github.com/opencloud-eu/opencloud/services/auth-app/pkg/config"
	settings "github.com/opencloud-eu/opencloud/services/settings/pkg/service/v0"
	ctxpkg "github.com/opencloud-eu/reva/v3/pkg/ctx"
	"github.com/opencloud-eu/reva/v3/pkg/rgrpc/todo/pool"
	cs3mocks "github.com/opencloud-eu/reva/v3/tests/cs3mocks/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go-micro.dev/v4/client"
	"go-micro.dev/v4/metadata"
)

const (
	adminRoleID = "test-admin-role"
	userRoleID  = "test-user-role"
)

// staticGatewaySelector always hands out the same gateway client.
type staticGatewaySelector struct {
	client gateway.GatewayAPIClient
}

func (s staticGatewaySelector) Next(...pool.Option) (gateway.GatewayAPIClient, error) {
	return s.client, nil
}

// fakeRoleService knows two roles: one with the account management permission and one without.
type fakeRoleService struct {
	settingssvc.RoleService
}

func (fakeRoleService) ListRoles(_ context.Context, in *settingssvc.ListBundlesRequest, _ ...client.CallOption) (*settingssvc.ListBundlesResponse, error) {
	res := &settingssvc.ListBundlesResponse{}
	for _, id := range in.GetBundleIds() {
		switch id {
		case adminRoleID:
			res.Bundles = append(res.Bundles, &settingsmsg.Bundle{
				Id:       adminRoleID,
				Settings: []*settingsmsg.Setting{{Id: settings.AccountManagementPermissionID}},
			})
		case userRoleID:
			res.Bundles = append(res.Bundles, &settingsmsg.Bundle{Id: userRoleID})
		}
	}
	return res, nil
}

func newTestService(t *testing.T, gwc gateway.GatewayAPIClient, allowImpersonation bool) *AuthAppService {
	t.Helper()
	svc, err := NewAuthAppService(
		Logger(log.NopLogger()),
		Config(&config.Config{AllowImpersonation: allowImpersonation, MachineAuthAPIKey: "machine-key"}),
		GatewaySelector(staticGatewaySelector{client: gwc}),
		Mux(chi.NewMux()),
		RoleClient(fakeRoleService{}),
	)
	require.NoError(t, err)
	return svc
}

func newCreateRequest(t *testing.T, query string, roleID string) *http.Request {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/auth-app/tokens?"+query, nil)
	ctx := ctxpkg.ContextSetUser(req.Context(), &userpb.User{Id: &userpb.UserId{OpaqueId: "admin-id"}, Username: "admin"})
	roleIDs, err := json.Marshal([]string{roleID})
	require.NoError(t, err)
	ctx = metadata.Set(ctx, middleware.RoleIDs, string(roleIDs))
	return req.WithContext(ctx)
}

func authenticateResponse(code rpc.Code) *gateway.AuthenticateResponse {
	return &gateway.AuthenticateResponse{Status: &rpc.Status{Code: code, Message: code.String()}}
}

func TestHandleCreateImpersonation(t *testing.T) {
	tests := []struct {
		name               string
		query              string
		roleID             string
		allowImpersonation bool
		authRes            *gateway.AuthenticateResponse
		authErr            error
		wantClientID       string // empty: Authenticate must not be called
		wantCode           int
		wantBody           string
	}{
		{
			name:               "unknown user name",
			query:              "expiry=72h&userName=nobody",
			roleID:             adminRoleID,
			allowImpersonation: true,
			authRes:            authenticateResponse(rpc.Code_CODE_NOT_FOUND),
			wantClientID:       "username:nobody",
			wantCode:           http.StatusNotFound,
			wantBody:           "user not found",
		},
		{
			name:               "unknown user id",
			query:              "expiry=72h&userID=4c510ada-c86b-4815-8820-42cdf82c3d51",
			roleID:             adminRoleID,
			allowImpersonation: true,
			authRes:            authenticateResponse(rpc.Code_CODE_NOT_FOUND),
			wantClientID:       "userid:4c510ada-c86b-4815-8820-42cdf82c3d51",
			wantCode:           http.StatusNotFound,
			wantBody:           "user not found",
		},
		{
			name:               "user provider unavailable",
			query:              "expiry=72h&userName=alice",
			roleID:             adminRoleID,
			allowImpersonation: true,
			authRes:            authenticateResponse(rpc.Code_CODE_UNAVAILABLE),
			wantClientID:       "username:alice",
			wantCode:           http.StatusInternalServerError,
		},
		{
			name:               "machine auth rejected",
			query:              "expiry=72h&userName=alice",
			roleID:             adminRoleID,
			allowImpersonation: true,
			authRes:            authenticateResponse(rpc.Code_CODE_UNAUTHENTICATED),
			wantClientID:       "username:alice",
			wantCode:           http.StatusInternalServerError,
		},
		{
			name:               "gateway transport error",
			query:              "expiry=72h&userName=alice",
			roleID:             adminRoleID,
			allowImpersonation: true,
			authErr:            errors.New("connection refused"),
			wantClientID:       "username:alice",
			wantCode:           http.StatusInternalServerError,
		},
		{
			name:               "user id and user name belong to different users",
			query:              "expiry=72h&userID=alice-id&userName=bob",
			roleID:             adminRoleID,
			allowImpersonation: true,
			authRes: &gateway.AuthenticateResponse{
				Status: &rpc.Status{Code: rpc.Code_CODE_OK},
				User:   &userpb.User{Id: &userpb.UserId{OpaqueId: "alice-id"}, Username: "alice"},
				Token:  "alice-token",
			},
			wantClientID: "userid:alice-id",
			wantCode:     http.StatusBadRequest,
		},
		{
			name:               "caller without account management permission",
			query:              "expiry=72h&userName=nobody",
			roleID:             userRoleID,
			allowImpersonation: true,
			wantCode:           http.StatusForbidden,
		},
		{
			name:               "impersonation disabled",
			query:              "expiry=72h&userName=nobody",
			roleID:             adminRoleID,
			allowImpersonation: false,
			wantCode:           http.StatusForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gwc := cs3mocks.NewGatewayAPIClient(t)
			if tt.wantClientID != "" {
				gwc.EXPECT().Authenticate(mock.Anything, mock.MatchedBy(func(req *gateway.AuthenticateRequest) bool {
					return req.GetType() == "machine" && req.GetClientId() == tt.wantClientID && req.GetClientSecret() == "machine-key"
				})).Return(tt.authRes, tt.authErr).Once()
			}

			rec := httptest.NewRecorder()
			newTestService(t, gwc, tt.allowImpersonation).ServeHTTP(rec, newCreateRequest(t, tt.query, tt.roleID))

			assert.Equal(t, tt.wantCode, rec.Code)
			if tt.wantBody != "" {
				assert.Equal(t, tt.wantBody+"\n", rec.Body.String())
			}
			// the mock fails the test on any unexpected call, e.g. GenerateAppPassword after a failed lookup
		})
	}
}

func TestHandleCreateImpersonationExistingUser(t *testing.T) {
	gwc := cs3mocks.NewGatewayAPIClient(t)
	gwc.EXPECT().Authenticate(mock.Anything, mock.Anything).Return(&gateway.AuthenticateResponse{
		Status: &rpc.Status{Code: rpc.Code_CODE_OK},
		User:   &userpb.User{Id: &userpb.UserId{OpaqueId: "alice-id"}, Username: "alice"},
		Token:  "alice-token",
	}, nil).Once()
	gwc.EXPECT().GenerateAppPassword(mock.Anything, mock.MatchedBy(func(req *applications.GenerateAppPasswordRequest) bool {
		return req.GetLabel() == "Generated via API (Impersonation)"
	})).Return(&applications.GenerateAppPasswordResponse{
		Status:      &rpc.Status{Code: rpc.Code_CODE_OK},
		AppPassword: &applications.AppPassword{Password: "app-password", Label: "Generated via API (Impersonation)"},
	}, nil).Once()

	rec := httptest.NewRecorder()
	newTestService(t, gwc, true).ServeHTTP(rec, newCreateRequest(t, "expiry=72h&userName=alice", adminRoleID))

	require.Equal(t, http.StatusOK, rec.Code)
	var token AuthAppToken
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &token))
	assert.Equal(t, "app-password", token.Token)
	assert.Equal(t, "Generated via API (Impersonation)", token.Label)
}
