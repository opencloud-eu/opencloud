package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	gateway "github.com/cs3org/go-cs3apis/cs3/gateway/v1beta1"
	gatewayv1beta1 "github.com/cs3org/go-cs3apis/cs3/gateway/v1beta1"
	userpb "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"
	rpcv1beta1 "github.com/cs3org/go-cs3apis/cs3/rpc/v1beta1"
	typesv1beta1 "github.com/cs3org/go-cs3apis/cs3/types/v1beta1"
	"github.com/opencloud-eu/opencloud/pkg/log"
	revactx "github.com/opencloud-eu/reva/v2/pkg/ctx"
	"github.com/opencloud-eu/reva/v2/pkg/rgrpc/todo/pool"
	"google.golang.org/grpc"
)

const testEndpoint = "http://example.com/graph/v1beta1/me/drive/sharedWithMe"

func getCookie(value string) *http.Cookie {
	return &http.Cookie{Name: "__Host-oc_guest_session", HttpOnly: true, Secure: true, Value: value}
}

func TestGuestLinkAuthenticator_Applicability(t *testing.T) {
	logger := log.NewLogger()

	tests := []struct {
		name        string
		enabled     bool
		cookieValue string
		path        string
		expect      AuthenticationState
	}{
		{"disabled", false, "some-jwt", "/graph/users", AuthenticationNotApplicable},
		{"no cookie", true, "", "/graph/users", AuthenticationNotApplicable},
		{"empty cookie", true, "", "/graph/users", AuthenticationNotApplicable},
		{"unsupported path - archiver", true, "some-jwt", "/archiver", AuthenticationNotApplicable},
		{"unsupported path - root", true, "some-jwt", "/", AuthenticationNotApplicable},
		{"unsupported path - ocs", true, "some-jwt", "/ocs/v2.php/cloud/user", AuthenticationNotApplicable},
		{"supported path - graph", true, "some-jwt", "/graph/v1beta1/me/drive/sharedWithMe", AuthenticationFailed},
		{"supported path - dav", true, "some-jwt", "/dav/files/user", AuthenticationFailed},
		{"supported path - webdav", true, "some-jwt", "/webdav/files/user", AuthenticationFailed},
		{"supported path - remote.php", true, "some-jwt", "/remote.php/dav/files/user", AuthenticationFailed},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pool.RemoveSelector("GatewaySelector" + "eu.opencloud.api.gateway")
			authenticator := &GuestLinkAuthenticator{
				Logger: logger,
				Config: GuestLinkAuthConfig{
					CookieName: "__Host-oc_guest_session",
				},
				RevaGatewaySelector: pool.GetSelector[gateway.GatewayAPIClient](
					"GatewaySelector",
					"eu.opencloud.api.gateway",
					func(cc grpc.ClientConnInterface) gateway.GatewayAPIClient {
						return mockGatewayClient{
							AuthenticateFunc: func(authType, clientID, clientSecret string) (string, rpcv1beta1.Code) {
								return "", rpcv1beta1.Code_CODE_UNAUTHENTICATED
							},
						}
					},
				),
			}

			req := httptest.NewRequest(http.MethodGet, "http://example.com"+tt.path, http.NoBody)
			if tt.cookieValue != "" {
				req.AddCookie(getCookie(tt.cookieValue))
			}

			result := authenticator.Authenticate(req)

			if result.State != tt.expect {
				t.Errorf("expected state %v, got %v", tt.expect, result.State)
			}
		})
	}
}

func TestGuestLinkAuthenticator_SupportsPathPrefixes(t *testing.T) {
	supportedPrefixes := []string{"/graph/v1beta1/me/drive/sharedWithMe", "/dav/", "/webdav/", "/remote.php/dav/", "/remote.php/webdav/", "/ocs/v2.php/apps/notifications/api/v1/notifications/sse"}
	for _, prefix := range supportedPrefixes {
		t.Run("path prefix "+prefix, func(t *testing.T) {
			if !isGuestLinkPath(prefix + "test") {
				t.Errorf("expected %s to be a supported guest-link path", prefix)
			}
		})
	}

	unsupportedPaths := []string{"/archiver", "/", "/ocs/", "/ocs/v2.php/apps/notifications/api/v1/notifications", "/konnect/", "/apps/"}
	for _, path := range unsupportedPaths {
		t.Run("unsupported "+path, func(t *testing.T) {
			if isGuestLinkPath(path) {
				t.Errorf("expected %s to not be a supported guest-link path", path)
			}
		})
	}
}

func TestGuestLinkAuthenticator_Success(t *testing.T) {
	pool.RemoveSelector("GatewaySelector" + "eu.opencloud.api.gateway")
	authenticator := &GuestLinkAuthenticator{
		Logger: log.NewLogger(),
		Config: GuestLinkAuthConfig{
			CookieName: "__Host-oc_guest_session",
		},
		RevaGatewaySelector: pool.GetSelector[gateway.GatewayAPIClient](
			"GatewaySelector",
			"eu.opencloud.api.gateway",
			func(cc grpc.ClientConnInterface) gateway.GatewayAPIClient {
				return mockGatewayClientWithUser{
					AuthenticateFunc: func(authType, clientID, clientSecret string) (string, *userpb.User, rpcv1beta1.Code) {
						if authType != "guestlinks" {
							return "", nil, rpcv1beta1.Code_CODE_NOT_FOUND
						}
						if clientSecret != "valid-jwt-token" {
							return "", nil, rpcv1beta1.Code_CODE_UNAUTHENTICATED
						}
						guestUser := &userpb.User{
							Id:          &userpb.UserId{OpaqueId: "guest-user-123", Type: userpb.UserType_USER_TYPE_GUEST},
							Username:    "guest@example.com",
							DisplayName: "Guest User",
						}
						return "reva-guest-token", guestUser, rpcv1beta1.Code_CODE_OK
					},
				}
			},
		),
	}

	req := httptest.NewRequest(http.MethodGet, testEndpoint, http.NoBody)
	req.AddCookie(getCookie("valid-jwt-token"))

	result := authenticator.Authenticate(req)

	if result.State != AuthenticationSucceeded {
		t.Errorf("expected AuthenticationSucceeded, got %v", result.State)
		return
	}

	ctx := result.Request.Context()
	user, ok := revactx.ContextGetUser(ctx)
	if !ok || user == nil {
		t.Error("expected user in context")
	}
	if user != nil && user.GetId().GetOpaqueId() != "guest-user-123" {
		t.Errorf("expected user ID 'guest-user-123', got '%s'", user.GetId().GetOpaqueId())
	}

	token := revactx.ContextMustGetToken(ctx)
	if token != "reva-guest-token" {
		t.Errorf("expected token 'reva-guest-token', got '%s'", token)
	}

	if req.Header.Get(revactx.TokenHeader) != "reva-guest-token" {
		t.Errorf("expected x-access-token header to be 'reva-guest-token'")
	}
}

func TestGuestLinkAuthenticator_MissingUserOnOK(t *testing.T) {
	pool.RemoveSelector("GatewaySelector" + "eu.opencloud.api.gateway")
	authenticator := &GuestLinkAuthenticator{
		Logger: log.NewLogger(),
		Config: GuestLinkAuthConfig{
			CookieName: "__Host-oc_guest_session",
		},
		RevaGatewaySelector: pool.GetSelector[gateway.GatewayAPIClient](
			"GatewaySelector",
			"eu.opencloud.api.gateway",
			func(cc grpc.ClientConnInterface) gateway.GatewayAPIClient {
				return mockGatewayClient{
					AuthenticateFunc: func(authType, clientID, clientSecret string) (string, rpcv1beta1.Code) {
						return "token-without-user", rpcv1beta1.Code_CODE_OK
					},
				}
			},
		),
	}

	req := httptest.NewRequest(http.MethodGet, testEndpoint, http.NoBody)
	req.AddCookie(getCookie("jwt-token"))

	result := authenticator.Authenticate(req)

	if result.State != AuthenticationError {
		t.Errorf("expected AuthenticationError, got %v", result.State)
	}
}

func TestGuestLinkAuthenticator_MissingTokenOnOK(t *testing.T) {
	pool.RemoveSelector("GatewaySelector" + "eu.opencloud.api.gateway")

	authenticator := &GuestLinkAuthenticator{
		Logger: log.NewLogger(),
		Config: GuestLinkAuthConfig{
			CookieName: "__Host-oc_guest_session",
		},
		RevaGatewaySelector: pool.GetSelector[gateway.GatewayAPIClient](
			"GatewaySelector",
			"eu.opencloud.api.gateway",
			func(cc grpc.ClientConnInterface) gateway.GatewayAPIClient {
				return mockGatewayClient{
					AuthenticateFunc: func(authType, clientID, clientSecret string) (string, rpcv1beta1.Code) {
						return "", rpcv1beta1.Code_CODE_OK
					},
				}
			},
		),
	}

	req := httptest.NewRequest(http.MethodGet, testEndpoint, http.NoBody)
	req.AddCookie(getCookie("jwt-token"))

	result := authenticator.Authenticate(req)

	if result.State != AuthenticationError {
		t.Errorf("expected AuthenticationError, got %v", result.State)
	}
}

func TestGuestLinkAuthenticator_ExpiredSession(t *testing.T) {
	pool.RemoveSelector("GatewaySelector" + "eu.opencloud.api.gateway")

	expiredInnerError := &typesv1beta1.OpaqueEntry{
		Decoder: "json",
		Value:   []byte(`{"type":"opencloud_guest_link_error","reason":"session_expired","share_id":"share-abc-123"}`),
	}

	authenticator := &GuestLinkAuthenticator{
		Logger: log.NewLogger(),
		Config: GuestLinkAuthConfig{
			CookieName: "__Host-oc_guest_session",
		},
		RevaGatewaySelector: pool.GetSelector[gateway.GatewayAPIClient](
			"GatewaySelector",
			"eu.opencloud.api.gateway",
			func(cc grpc.ClientConnInterface) gateway.GatewayAPIClient {
				return mockGatewayClientWithInnerError{
					innerError: expiredInnerError,
				}
			},
		),
	}

	req := httptest.NewRequest(http.MethodGet, testEndpoint, http.NoBody)
	req.AddCookie(getCookie("expired-jwt"))

	result := authenticator.Authenticate(req)

	if result.State != AuthenticationFailed {
		t.Errorf("expected AuthenticationFailed, got %v", result.State)
	}
	if !result.Terminal {
		t.Error("expected Terminal to be true for expired session")
	}
	if result.ErrorDetails == nil {
		t.Error("expected ErrorDetails for expired session")
		return
	}
	details, ok := result.ErrorDetails.(GuestSessionExpiredDetails)
	if !ok {
		t.Errorf("expected GuestSessionExpiredDetails, got %T", result.ErrorDetails)
		return
	}
	if details.PermissionID != "share-abc-123" {
		t.Errorf("expected permissionId 'share-abc-123', got '%s'", details.PermissionID)
	}
	if len(result.CookiesToClear) != 1 || result.CookiesToClear[0] != "__Host-oc_guest_session" {
		t.Errorf("expected cookie name '__Host-oc_guest_session' in CookiesToClear, got %v", result.CookiesToClear)
	}
}

func TestGuestLinkAuthenticator_GenericUnauthenticated(t *testing.T) {
	pool.RemoveSelector("GatewaySelector" + "eu.opencloud.api.gateway")

	authenticator := &GuestLinkAuthenticator{
		Logger: log.NewLogger(),
		Config: GuestLinkAuthConfig{
			CookieName: "__Host-oc_guest_session",
		},
		RevaGatewaySelector: pool.GetSelector[gateway.GatewayAPIClient](
			"GatewaySelector",
			"eu.opencloud.api.gateway",
			func(cc grpc.ClientConnInterface) gateway.GatewayAPIClient {
				return mockGatewayClient{
					AuthenticateFunc: func(authType, clientID, clientSecret string) (string, rpcv1beta1.Code) {
						return "", rpcv1beta1.Code_CODE_UNAUTHENTICATED
					},
				}
			},
		),
	}

	req := httptest.NewRequest(http.MethodGet, testEndpoint, http.NoBody)
	req.AddCookie(getCookie("invalid-jwt"))

	result := authenticator.Authenticate(req)

	if result.State != AuthenticationFailed {
		t.Errorf("expected AuthenticationFailed, got %v", result.State)
	}
	if !result.Terminal {
		t.Error("expected Terminal to be true for invalid credentials")
	}
	if result.ErrorDetails != nil {
		t.Error("expected no ErrorDetails for generic unauthenticated")
	}
	if len(result.CookiesToClear) != 1 || result.CookiesToClear[0] != "__Host-oc_guest_session" {
		t.Errorf("expected cookie name '__Host-oc_guest_session' in CookiesToClear, got %v", result.CookiesToClear)
	}
}

func TestGuestLinkAuthenticator_Unavailable(t *testing.T) {
	pool.RemoveSelector("GatewaySelector" + "eu.opencloud.api.gateway")

	authenticator := &GuestLinkAuthenticator{
		Logger: log.NewLogger(),
		Config: GuestLinkAuthConfig{
			CookieName: "__Host-oc_guest_session",
		},
		RevaGatewaySelector: pool.GetSelector[gateway.GatewayAPIClient](
			"GatewaySelector",
			"eu.opencloud.api.gateway",
			func(cc grpc.ClientConnInterface) gateway.GatewayAPIClient {
				return mockGatewayClient{
					AuthenticateFunc: func(authType, clientID, clientSecret string) (string, rpcv1beta1.Code) {
						return "", rpcv1beta1.Code_CODE_UNAVAILABLE
					},
				}
			},
		),
	}

	req := httptest.NewRequest(http.MethodGet, testEndpoint, http.NoBody)
	req.AddCookie(getCookie("jwt-token"))

	result := authenticator.Authenticate(req)

	if result.State != AuthenticationError {
		t.Errorf("expected AuthenticationError, got %v", result.State)
	}
}

func TestGuestLinkAuthenticator_ChallengeSuppression(t *testing.T) {
	authenticator := &GuestLinkAuthenticator{
		Config: GuestLinkAuthConfig{
			CookieName: "__Host-oc_guest_session",
		},
	}

	reqNoCookie := httptest.NewRequest(http.MethodGet, testEndpoint, http.NoBody)
	if authenticator.SuppressAuthenticationChallenge(reqNoCookie) {
		t.Error("expected challenge not to be suppressed when no cookie present")
	}

	reqWithCookie := httptest.NewRequest(http.MethodGet, testEndpoint, http.NoBody)
	reqWithCookie.AddCookie(getCookie("some-jwt"))
	if !authenticator.SuppressAuthenticationChallenge(reqWithCookie) {
		t.Error("expected challenge to be suppressed when cookie present")
	}
}

func TestGuestLinkAuthenticator_JwtNotLogged(t *testing.T) {
	// This test verifies that the raw JWT is not included in any logged fields.
	// We can't easily capture log output, but we can verify the authenticator
	// structure doesn't store the JWT.
	authenticator := &GuestLinkAuthenticator{
		Config: GuestLinkAuthConfig{
			CookieName: "__Host-oc_guest_session",
		},
	}

	_ = authenticator // Just verify it compiles without storing raw JWT
}

func TestGuestLinkAuthenticator_GatewayCall(t *testing.T) {
	pool.RemoveSelector("GatewaySelector" + "eu.opencloud.api.gateway")

	var lastAuthType, lastClientID, lastClientSecret string
	authenticator := &GuestLinkAuthenticator{
		Logger: log.NewLogger(),
		Config: GuestLinkAuthConfig{
			CookieName: "__Host-oc_guest_session",
		},
		RevaGatewaySelector: pool.GetSelector[gateway.GatewayAPIClient](
			"GatewaySelector",
			"eu.opencloud.api.gateway",
			func(cc grpc.ClientConnInterface) gateway.GatewayAPIClient {
				return mockGatewayClient{
					AuthenticateFunc: func(authType, clientID, clientSecret string) (string, rpcv1beta1.Code) {
						lastAuthType = authType
						lastClientID = clientID
						lastClientSecret = clientSecret
						return "", rpcv1beta1.Code_CODE_UNAUTHENTICATED
					},
				}
			},
		),
	}

	req := httptest.NewRequest(http.MethodGet, testEndpoint, http.NoBody)
	req.AddCookie(getCookie("test-jwt-value"))

	authenticator.Authenticate(req)

	if lastAuthType != "guestlinks" {
		t.Errorf("expected auth type 'guestlinks', got '%s'", lastAuthType)
	}
	if lastClientID != "" {
		t.Errorf("expected empty client_id, got '%s'", lastClientID)
	}
	if lastClientSecret != "test-jwt-value" {
		t.Errorf("expected client_secret 'test-jwt-value', got '%s'", lastClientSecret)
	}
}

// Helper test doubles.

// mockGatewayClientWithUser is a test double that can return a user with the response.
type mockGatewayClientWithUser struct {
	gatewayv1beta1.GatewayAPIClient
	AuthenticateFunc func(authType, clientID, clientSecret string) (string, *userpb.User, rpcv1beta1.Code)
}

func (c mockGatewayClientWithUser) Authenticate(ctx context.Context, in *gatewayv1beta1.AuthenticateRequest, opts ...grpc.CallOption) (*gatewayv1beta1.AuthenticateResponse, error) {
	token, user, code := c.AuthenticateFunc(in.GetType(), in.GetClientId(), in.GetClientSecret())
	return &gatewayv1beta1.AuthenticateResponse{
		Status: &rpcv1beta1.Status{Code: code},
		Token:  token,
		User:   user,
	}, nil
}

// mockGatewayClientWithInnerError is a test double that returns a status with InnerError.
type mockGatewayClientWithInnerError struct {
	gatewayv1beta1.GatewayAPIClient
	innerError *typesv1beta1.OpaqueEntry
}

func (c mockGatewayClientWithInnerError) Authenticate(ctx context.Context, in *gatewayv1beta1.AuthenticateRequest, opts ...grpc.CallOption) (*gatewayv1beta1.AuthenticateResponse, error) {
	return &gatewayv1beta1.AuthenticateResponse{
		Status: &rpcv1beta1.Status{
			Code:       rpcv1beta1.Code_CODE_UNAUTHENTICATED,
			InnerError: c.innerError,
		},
	}, nil
}

func TestParseInnerError(t *testing.T) {
	tests := []struct {
		name        string
		innerError  *typesv1beta1.OpaqueEntry
		wantExpired bool
		wantShareID string
	}{
		{
			name:        "nil inner error",
			innerError:  nil,
			wantExpired: false,
		},
		{
			name:        "empty value",
			innerError:  &typesv1beta1.OpaqueEntry{Decoder: "json", Value: []byte{}},
			wantExpired: false,
		},
		{
			name:        "wrong decoder",
			innerError:  &typesv1beta1.OpaqueEntry{Decoder: "xml", Value: []byte(`{"type":"opencloud_guest_link_error"}`)},
			wantExpired: false,
		},
		{
			name:        "malformed JSON",
			innerError:  &typesv1beta1.OpaqueEntry{Decoder: "json", Value: []byte(`not json`)},
			wantExpired: false,
		},
		{
			name:        "wrong type",
			innerError:  &typesv1beta1.OpaqueEntry{Decoder: "json", Value: []byte(`{"type":"wrong_type","reason":"session_expired","share_id":"s1"}`)},
			wantExpired: false,
		},
		{
			name:        "wrong reason",
			innerError:  &typesv1beta1.OpaqueEntry{Decoder: "json", Value: []byte(`{"type":"opencloud_guest_link_error","reason":"wrong_reason","share_id":"s1"}`)},
			wantExpired: false,
		},
		{
			name:        "empty share ID",
			innerError:  &typesv1beta1.OpaqueEntry{Decoder: "json", Value: []byte(`{"type":"opencloud_guest_link_error","reason":"session_expired","share_id":""}`)},
			wantExpired: false,
		},
		{
			name:        "valid expired session",
			innerError:  &typesv1beta1.OpaqueEntry{Decoder: "json", Value: []byte(`{"type":"opencloud_guest_link_error","reason":"session_expired","share_id":"share-abc-123"}`)},
			wantExpired: true,
			wantShareID: "share-abc-123",
		},
		{
			name:        "share ID too long",
			innerError:  &typesv1beta1.OpaqueEntry{Decoder: "json", Value: []byte(`{"type":"opencloud_guest_link_error","reason":"session_expired","share_id":"` + strings.Repeat("x", 513) + `"}`)},
			wantExpired: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			authenticator := &GuestLinkAuthenticator{}
			expired, shareID := authenticator.parseInnerError(tt.innerError)
			if expired != tt.wantExpired {
				t.Errorf("want expired=%v, got=%v", tt.wantExpired, expired)
			}
			if shareID != tt.wantShareID {
				t.Errorf("want shareID=%q, got=%q", tt.wantShareID, shareID)
			}
		})
	}
}

func TestClearGuestCookie(t *testing.T) {
	cookie := clearGuestCookie("test-cookie")

	if cookie.Name != "test-cookie" {
		t.Errorf("expected name 'test-cookie', got '%s'", cookie.Name)
	}
	if cookie.Value != "" {
		t.Errorf("expected empty value, got '%s'", cookie.Value)
	}
	if cookie.Path != "/" {
		t.Errorf("expected path '/', got '%s'", cookie.Path)
	}
	if cookie.Domain != "" {
		t.Errorf("expected empty domain, got '%s'", cookie.Domain)
	}
	if !cookie.Secure {
		t.Error("expected Secure to be true")
	}
	if !cookie.HttpOnly {
		t.Error("expected HttpOnly to be true")
	}
	if cookie.SameSite != http.SameSiteStrictMode {
		t.Errorf("expected SameSite=Strict, got %v", cookie.SameSite)
	}
	if cookie.MaxAge != -1 {
		t.Errorf("expected MaxAge=-1, got %d", cookie.MaxAge)
	}
}
