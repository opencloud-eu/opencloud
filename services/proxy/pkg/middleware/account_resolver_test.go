package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	gateway "github.com/cs3org/go-cs3apis/cs3/gateway/v1beta1"
	tenantpb "github.com/cs3org/go-cs3apis/cs3/identity/tenant/v1beta1"
	userv1beta1 "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"
	rpcpb "github.com/cs3org/go-cs3apis/cs3/rpc/v1beta1"
	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/pkg/oidc"
	"github.com/opencloud-eu/opencloud/services/proxy/pkg/config"
	"github.com/opencloud-eu/opencloud/services/proxy/pkg/router"
	"github.com/opencloud-eu/opencloud/services/proxy/pkg/user/backend"
	"github.com/opencloud-eu/opencloud/services/proxy/pkg/user/backend/mocks"
	userRoleMocks "github.com/opencloud-eu/opencloud/services/proxy/pkg/userroles/mocks"
	"github.com/opencloud-eu/reva/v2/pkg/auth/scope"
	revactx "github.com/opencloud-eu/reva/v2/pkg/ctx"
	"github.com/opencloud-eu/reva/v2/pkg/rgrpc/todo/pool"
	"github.com/opencloud-eu/reva/v2/pkg/token/manager/jwt"
	cs3mocks "github.com/opencloud-eu/reva/v2/tests/cs3mocks/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc"
)

const (
	testIdP               = "https://idx.example.com"
	testTenantA           = "tenant-a"
	testTenantB           = "tenant-b"
	testJWTSecret         = "change-me"
	testSvcAccountID      = "svc-account-id"
	testSvcAccountSecret  = "svc-account-secret"
	testSvcAccountToken   = "svc-account-token"
)

func TestTokenIsAddedWithMailClaim(t *testing.T) {
	sut := newMockAccountResolver(&userv1beta1.User{
		Id:   &userv1beta1.UserId{Idp: testIdP, OpaqueId: "123"},
		Mail: "foo@example.com",
	}, nil, oidc.Email, "mail", false)

	req, rw := mockRequest(map[string]interface{}{
		oidc.Iss:   testIdP,
		oidc.Email: "foo@example.com",
	})

	sut.ServeHTTP(rw, req)

	token := req.Header.Get(revactx.TokenHeader)
	assert.NotEmpty(t, token)
	assert.Contains(t, token, "eyJ")
}

func TestTokenIsAddedWithUsernameClaim(t *testing.T) {
	sut := newMockAccountResolver(&userv1beta1.User{
		Id:   &userv1beta1.UserId{Idp: testIdP, OpaqueId: "123"},
		Mail: "foo@example.com",
	}, nil, oidc.PreferredUsername, "username", false)

	req, rw := mockRequest(map[string]interface{}{
		oidc.Iss:               testIdP,
		oidc.PreferredUsername: "foo",
	})

	sut.ServeHTTP(rw, req)

	token := req.Header.Get(revactx.TokenHeader)
	assert.NotEmpty(t, token)

	assert.Contains(t, token, "eyJ")
}

func TestTokenIsAddedWithDotUsernamePathClaim(t *testing.T) {
	sut := newMockAccountResolver(&userv1beta1.User{
		Id:   &userv1beta1.UserId{Idp: testIdP, OpaqueId: "123"},
		Mail: "foo@example.com",
	}, nil, "li.un", "username", false)

	// This is how lico adds the username to the access token
	req, rw := mockRequest(map[string]interface{}{
		oidc.Iss: testIdP,
		"li": map[string]interface{}{
			"un": "foo",
		},
	})

	sut.ServeHTTP(rw, req)

	token := req.Header.Get(revactx.TokenHeader)
	assert.NotEmpty(t, token)

	assert.Contains(t, token, "eyJ")
}

func TestTokenIsAddedWithDottedUsernameClaim(t *testing.T) {
	tests := []struct {
		name      string
		oidcClaim string
		// comment describing what the claim exercises
		desc string
	}{
		{
			name:      "escaped dot treated as literal key",
			oidcClaim: "li\\.un",
			desc:      "li\\.un escapes the dot so the claim is looked up as the literal key \"li.un\"",
		},
		{
			name:      "dotted path falls back to literal key",
			oidcClaim: "li.un",
			desc:      "li.un is first tried as a nested path; when \"un\" is absent under \"li\", it falls back to the literal key \"li.un\"",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			sut := newMockAccountResolver(&userv1beta1.User{
				Id:   &userv1beta1.UserId{Idp: testIdP, OpaqueId: "123"},
				Mail: "foo@example.com",
			}, nil, tc.oidcClaim, "username", false)

			req, rw := mockRequest(map[string]interface{}{
				oidc.Iss: testIdP,
				"li.un":  "foo",
			})

			sut.ServeHTTP(rw, req)

			token := req.Header.Get(revactx.TokenHeader)
			assert.NotEmpty(t, token)
			assert.Contains(t, token, "eyJ")
		})
	}
}

func TestNSkipOnNoClaims(t *testing.T) {
	sut := newMockAccountResolver(nil, backend.ErrAccountDisabled, oidc.Email, "mail", false)
	req, rw := mockRequest(nil)

	sut.ServeHTTP(rw, req)

	token := req.Header.Get("x-access-token")
	assert.Empty(t, token)
	assert.Equal(t, http.StatusOK, rw.Code)
}

func TestUnauthorizedOnUserNotFound(t *testing.T) {
	sut := newMockAccountResolver(nil, backend.ErrAccountNotFound, oidc.PreferredUsername, "username", false)
	req, rw := mockRequest(map[string]interface{}{
		oidc.Iss:               testIdP,
		oidc.PreferredUsername: "foo",
	})

	sut.ServeHTTP(rw, req)

	token := req.Header.Get(revactx.TokenHeader)
	assert.Empty(t, token)
	assert.Equal(t, http.StatusUnauthorized, rw.Code)
}

func TestUnauthorizedOnUserDisabled(t *testing.T) {
	sut := newMockAccountResolver(nil, backend.ErrAccountDisabled, oidc.PreferredUsername, "username", false)
	req, rw := mockRequest(map[string]interface{}{
		oidc.Iss:               testIdP,
		oidc.PreferredUsername: "foo",
	})

	sut.ServeHTTP(rw, req)

	token := req.Header.Get(revactx.TokenHeader)
	assert.Empty(t, token)
	assert.Equal(t, http.StatusUnauthorized, rw.Code)
}

func TestInternalServerErrorOnMissingMailAndUsername(t *testing.T) {
	sut := newMockAccountResolver(nil, backend.ErrAccountNotFound, oidc.Email, "mail", false)
	req, rw := mockRequest(map[string]interface{}{
		oidc.Iss: testIdP,
	})

	sut.ServeHTTP(rw, req)

	token := req.Header.Get(revactx.TokenHeader)
	assert.Empty(t, token)
	assert.Equal(t, http.StatusInternalServerError, rw.Code)
}

func TestUnauthorizedOnMissingTenantId(t *testing.T) {
	sut := newMockAccountResolver(
		&userv1beta1.User{
			Id:       &userv1beta1.UserId{Idp: testIdP, OpaqueId: "123"},
			Username: "foo",
		},
		nil, oidc.PreferredUsername, "username", true)
	req, rw := mockRequest(map[string]any{
		oidc.Iss:               testIdP,
		oidc.PreferredUsername: "foo",
	})

	sut.ServeHTTP(rw, req)

	token := req.Header.Get(revactx.TokenHeader)
	assert.Empty(t, token)
	assert.Equal(t, http.StatusUnauthorized, rw.Code)
}

func TestTokenIsAddedWhenUserHasTenantId(t *testing.T) {
	sut := newMockAccountResolver(
		&userv1beta1.User{
			Id: &userv1beta1.UserId{
				Idp:      testIdP,
				OpaqueId: "123",
				TenantId: "tenant1",
			},
			Username: "foo",
		},
		nil, oidc.PreferredUsername, "username", true)
	req, rw := mockRequest(map[string]any{
		oidc.Iss:               testIdP,
		oidc.PreferredUsername: "foo",
	})

	sut.ServeHTTP(rw, req)

	token := req.Header.Get(revactx.TokenHeader)
	assert.NotEmpty(t, token)
	assert.Contains(t, token, "eyJ")
}

func TestTenantClaimValidation(t *testing.T) {
	tests := []struct {
		name           string
		requestTenant  string
		wantToken      bool
		wantStatusCode int
	}{
		{
			name:           "token added when tenant claim matches",
			requestTenant:  testTenantA,
			wantToken:      true,
			wantStatusCode: http.StatusOK,
		},
		{
			name:           "unauthorized when tenant claim does not match",
			requestTenant:  testTenantB,
			wantToken:      false,
			wantStatusCode: http.StatusUnauthorized,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			user := &userv1beta1.User{
				Id: &userv1beta1.UserId{
					Idp:      testIdP,
					OpaqueId: "123",
					TenantId: testTenantA,
				},
				Username: "foo",
			}

			tokenManager, _ := jwt.New(map[string]interface{}{"secret": testJWTSecret, "expires": int64(60)})
			s, _ := scope.AddOwnerScope(nil)
			token, _ := tokenManager.MintToken(context.Background(), user, s)

			ub := mocks.UserBackend{}
			ub.On("GetUserByClaims", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(user, token, nil)
			ra := userRoleMocks.UserRoleAssigner{}
			ra.On("UpdateUserRoleAssignment", mock.Anything, mock.Anything, mock.Anything).Return(user, nil)

			sut := AccountResolver(
				Logger(log.NewLogger()),
				UserProvider(&ub),
				UserRoleAssigner(&ra),
				UserOIDCClaim(oidc.PreferredUsername),
				UserCS3Claim("username"),
				TenantOIDCClaim("tenant_id"),
				MultiTenantEnabled(true),
			)(mockHandler{})

			req, rw := mockRequest(map[string]interface{}{
				oidc.Iss:               testIdP,
				oidc.PreferredUsername: "foo",
				"tenant_id":            tc.requestTenant,
			})

			sut.ServeHTTP(rw, req)

			if tc.wantToken {
				assert.NotEmpty(t, req.Header.Get(revactx.TokenHeader))
			} else {
				assert.Empty(t, req.Header.Get(revactx.TokenHeader))
			}
			assert.Equal(t, tc.wantStatusCode, rw.Code)
		})
	}
}

func newMockAccountResolver(userBackendResult *userv1beta1.User, userBackendErr error, oidcclaim, cs3claim string, multiTenant bool) http.Handler {
	tokenManager, _ := jwt.New(map[string]interface{}{
		"secret":  testJWTSecret,
		"expires": int64(60),
	})

	token := ""
	if userBackendResult != nil {
		s, _ := scope.AddOwnerScope(nil)
		token, _ = tokenManager.MintToken(context.Background(), userBackendResult, s)
	}

	ub := mocks.UserBackend{}
	ub.On("GetUserByClaims", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(userBackendResult, token, userBackendErr)
	ub.On("GetUserRoles", mock.Anything, mock.Anything).Return(userBackendResult, nil)

	ra := userRoleMocks.UserRoleAssigner{}
	ra.On("UpdateUserRoleAssignment", mock.Anything, mock.Anything, mock.Anything).Return(userBackendResult, nil)

	return AccountResolver(
		Logger(log.NewLogger()),
		UserProvider(&ub),
		UserRoleAssigner(&ra),
		SkipUserInfo(false),
		UserOIDCClaim(oidcclaim),
		UserCS3Claim(cs3claim),
		AutoprovisionAccounts(false),
		MultiTenantEnabled(multiTenant),
	)(mockHandler{})
}

func mockRequest(claims map[string]interface{}) (*http.Request, *httptest.ResponseRecorder) {
	if claims == nil {
		return httptest.NewRequest("GET", "http://example.com/foo", nil), httptest.NewRecorder()
	}

	ctx := oidc.NewContext(context.Background(), claims)
	ctx = router.SetRoutingInfo(ctx, router.RoutingInfo{})
	req := httptest.NewRequest("GET", "http://example.com/foo", nil).WithContext(ctx)
	rw := httptest.NewRecorder()

	return req, rw
}

type mockHandler struct{}

func (m mockHandler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {}

func TestTenantIDMapping(t *testing.T) {
	const (
		externalTenantID = "external-tenant-x"
		internalTenantID = testTenantA
	)

	user := &userv1beta1.User{
		Id: &userv1beta1.UserId{
			Idp:      testIdP,
			OpaqueId: "123",
			TenantId: internalTenantID,
		},
		Username: "foo",
	}

	tokenManager, _ := jwt.New(map[string]interface{}{"secret": testJWTSecret, "expires": int64(60)})
	s, _ := scope.AddOwnerScope(nil)
	token, _ := tokenManager.MintToken(context.Background(), user, s)

	newSUT := func(t *testing.T, gatewayClient gateway.GatewayAPIClient) http.Handler {
		t.Helper()
		gatewaySelector := pool.GetSelector[gateway.GatewayAPIClient](
			"GatewaySelector",
			"eu.opencloud.api.gateway",
			func(cc grpc.ClientConnInterface) gateway.GatewayAPIClient {
				return gatewayClient
			},
		)
		t.Cleanup(func() { pool.RemoveSelector("GatewaySelector" + "eu.opencloud.api.gateway") })

		ub := mocks.UserBackend{}
		ub.On("GetUserByClaims", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(user, token, nil)
		ra := userRoleMocks.UserRoleAssigner{}
		ra.On("UpdateUserRoleAssignment", mock.Anything, mock.Anything, mock.Anything).Return(user, nil)

		return AccountResolver(
			Logger(log.NewLogger()),
			UserProvider(&ub),
			UserRoleAssigner(&ra),
			UserOIDCClaim(oidc.PreferredUsername),
			UserCS3Claim("username"),
			TenantOIDCClaim("tenant_id"),
			MultiTenantEnabled(true),
			TenantIDMappingEnabled(true),
			ServiceAccount(config.ServiceAccount{
				ServiceAccountID:     testSvcAccountID,
				ServiceAccountSecret: testSvcAccountSecret,
			}),
			WithRevaGatewaySelector(gatewaySelector),
		)(mockHandler{})
	}

	tests := []struct {
		name           string
		tenantResponse *tenantpb.GetTenantByClaimResponse
		wantToken      bool
		wantStatusCode int
	}{
		{
			name: "token added when external tenant maps to user internal tenant",
			tenantResponse: &tenantpb.GetTenantByClaimResponse{
				Status: &rpcpb.Status{Code: rpcpb.Code_CODE_OK},
				Tenant: &tenantpb.Tenant{Id: internalTenantID, ExternalId: externalTenantID},
			},
			wantToken:      true,
			wantStatusCode: http.StatusOK,
		},
		{
			name: "unauthorized when external tenant maps to a different internal tenant",
			tenantResponse: &tenantpb.GetTenantByClaimResponse{
				Status: &rpcpb.Status{Code: rpcpb.Code_CODE_OK},
				Tenant: &tenantpb.Tenant{Id: testTenantB, ExternalId: externalTenantID},
			},
			wantToken:      false,
			wantStatusCode: http.StatusUnauthorized,
		},
		{
			name: "unauthorized when external tenant is not found",
			tenantResponse: &tenantpb.GetTenantByClaimResponse{
				Status: &rpcpb.Status{Code: rpcpb.Code_CODE_NOT_FOUND, Message: "not found"},
			},
			wantToken:      false,
			wantStatusCode: http.StatusUnauthorized,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			gwc := &cs3mocks.GatewayAPIClient{}
			gwc.On("Authenticate", mock.Anything, &gateway.AuthenticateRequest{
				Type:         "serviceaccounts",
				ClientId:     testSvcAccountID,
				ClientSecret: testSvcAccountSecret,
			}).Return(&gateway.AuthenticateResponse{
				Status: &rpcpb.Status{Code: rpcpb.Code_CODE_OK},
				Token:  testSvcAccountToken,
			}, nil)
			gwc.On("GetTenantByClaim", mock.Anything, &tenantpb.GetTenantByClaimRequest{
				Claim: "externalid",
				Value: externalTenantID,
			}).Return(tc.tenantResponse, nil)

			req, rw := mockRequest(map[string]interface{}{
				oidc.Iss:               testIdP,
				oidc.PreferredUsername: "foo",
				"tenant_id":            externalTenantID,
			})
			newSUT(t, gwc).ServeHTTP(rw, req)

			if tc.wantToken {
				assert.NotEmpty(t, req.Header.Get(revactx.TokenHeader))
			} else {
				assert.Empty(t, req.Header.Get(revactx.TokenHeader))
			}
			assert.Equal(t, tc.wantStatusCode, rw.Code)
			gwc.AssertExpectations(t)
		})
	}
}
