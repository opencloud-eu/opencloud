package middleware

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"time"

	gateway "github.com/cs3org/go-cs3apis/cs3/gateway/v1beta1"
	userv1beta1 "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"
	rpcv1beta1 "github.com/cs3org/go-cs3apis/cs3/rpc/v1beta1"
	"github.com/golang-jwt/jwt/v5"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/pkg/oidc"
	oidcmocks "github.com/opencloud-eu/opencloud/pkg/oidc/mocks"
	"github.com/opencloud-eu/opencloud/services/proxy/pkg/config"
	"github.com/opencloud-eu/opencloud/services/proxy/pkg/router"
	"github.com/opencloud-eu/opencloud/services/proxy/pkg/user/backend"
	"github.com/opencloud-eu/opencloud/services/proxy/pkg/user/backend/mocks"
	userrolesmocks "github.com/opencloud-eu/opencloud/services/proxy/pkg/userroles/mocks"
	revactx "github.com/opencloud-eu/reva/v2/pkg/ctx"
	"github.com/opencloud-eu/reva/v2/pkg/rgrpc/todo/pool"
	"github.com/opencloud-eu/reva/v2/pkg/signedurl"
	"github.com/stretchr/testify/mock"
	"go-micro.dev/v4/store"
	"google.golang.org/grpc"
)

var _ = Describe("authentication helpers", func() {
	DescribeTable("isPublicPath should recognize public paths",
		func(input string, expected bool) {
			isPublic := isPublicPath(input)
			Expect(isPublic).To(Equal(expected))
		},
		Entry("public files path", "/remote.php/dav/public-files/", true),
		Entry("public files path without remote.php", "/remote.php/dav/public-files/", true),
		Entry("token info path", "/ocs/v1.php/apps/files_sharing/api/v1/tokeninfo/unprotected", true),
		Entry("token info path", "/ocs/v2.php/apps/files_sharing/api/v1/tokeninfo/unprotected", true),
		Entry("capabilities", "/ocs/v1.php/cloud/capabilities", true),
	)
})

var _ = Describe("Authenticating requests", Label("Authentication"), func() {
	var (
		authenticators []Authenticator
	)
	oc := oidcmocks.OIDCClient{}
	oc.On("VerifyAccessToken", mock.Anything, mock.Anything).Return(
		oidc.RegClaimsWithSID{
			SessionID: "a-session-id",
			RegisteredClaims: jwt.RegisteredClaims{
				ExpiresAt: jwt.NewNumericDate(time.Unix(1147483647, 0)),
			},
		}, jwt.MapClaims{
			"sid": "a-session-id",
			"exp": 1147483647,
		},
		nil,
	)

	ub := mocks.UserBackend{}
	ub.On("Authenticate", mock.Anything, "testuser", "testpassword").Return(
		&userv1beta1.User{
			Id: &userv1beta1.UserId{
				Idp:      "IdpId",
				OpaqueId: "OpaqueId",
			},
			Username: "testuser",
			Mail:     "testuser@example.com",
		},
		"",
		nil,
	)
	ub.On("Authenticate", mock.Anything, mock.Anything, mock.Anything).Return(nil, "", backend.ErrAccountNotFound)

	BeforeEach(func() {
		pool.RemoveSelector("GatewaySelector" + "eu.opencloud.api.gateway")

		logger := log.NewLogger()
		authenticators = []Authenticator{
			BasicAuthenticator{
				Logger:       logger,
				UserProvider: &ub,
			},
			&OIDCAuthenticator{
				OIDCIss:       "http://idp.example.com",
				Logger:        logger,
				oidcClient:    &oc,
				userInfoCache: store.NewMemoryStore(),
				skipUserInfo:  true,
			},
			PublicShareAuthenticator{
				Logger: logger,
				RevaGatewaySelector: pool.GetSelector[gateway.GatewayAPIClient](
					"GatewaySelector",
					"eu.opencloud.api.gateway",
					func(cc grpc.ClientConnInterface) gateway.GatewayAPIClient {
						return mockGatewayClient{
							AuthenticateFunc: func(authType, clientID, clientSecret string) (string, rpcv1beta1.Code) {
								if authType != "publicshares" {
									return "", rpcv1beta1.Code_CODE_NOT_FOUND
								}

								if clientID == "sharetoken" && (clientSecret == "password|examples3cr3t" || clientSecret == "signature|examplesignature|exampleexpiration") {
									return "exampletoken", rpcv1beta1.Code_CODE_OK
								}

								if clientID == "sharetoken" && clientSecret == "password|" {
									return "otherexampletoken", rpcv1beta1.Code_CODE_OK
								}

								return "", rpcv1beta1.Code_CODE_NOT_FOUND
							},
						}
					},
				),
			},
		}
	})

	When("the public request must contains correct data", func() {
		It("ensures the context oidc data when the Bearer authentication is successful", func() {
			req := httptest.NewRequest("PROPFIND", "http://example.com/remote.php/dav/public-files/", http.NoBody)
			req = req.WithContext(router.SetRoutingInfo(context.Background(), router.RoutingInfo{}))
			req.Header.Set(_headerAuthorization, "Bearer jwt.token.sig")

			handler := Authentication(authenticators,
				EnableBasicAuth(true),
			)
			testHandler := handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				Expect(oidc.FromContext(r.Context())).To(Equal(map[string]any{
					"sid": "a-session-id",
					"exp": int64(1147483647),
				}))
			}))
			rr := httptest.NewRecorder()
			testHandler.ServeHTTP(rr, req)
			Expect(rr).To(HaveHTTPStatus(http.StatusOK))
		})
		It("ensures the context oidc data when user the Basic authentication is successful", func() {
			req := httptest.NewRequest("PROPFIND", "http://example.com/remote.php/dav/public-files/", http.NoBody)
			req = req.WithContext(router.SetRoutingInfo(context.Background(), router.RoutingInfo{}))
			req.SetBasicAuth("testuser", "testpassword")

			handler := Authentication(authenticators,
				EnableBasicAuth(true),
			)
			testHandler := handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				Expect(oidc.FromContext(r.Context())).To(Equal(map[string]any{
					"email":              "testuser@example.com",
					"openclouduuid":      "OpaqueId",
					"iss":                "IdpId",
					"preferred_username": "testuser",
				}))
			}))
			rr := httptest.NewRecorder()
			testHandler.ServeHTTP(rr, req)
			Expect(rr).To(HaveHTTPStatus(http.StatusOK))
		})
		It("ensures the x-access-token header when public-token URL parameter is set", func() {
			req := httptest.NewRequest("PROPFIND", "http://example.com/dav/public-files/?public-token=sharetoken", http.NoBody)
			req = req.WithContext(router.SetRoutingInfo(context.Background(), router.RoutingInfo{}))

			handler := Authentication(authenticators,
				EnableBasicAuth(true),
			)
			testHandler := handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				Expect(r.Header.Get(headerRevaAccessToken)).To(Equal("otherexampletoken"))
			}))
			rr := httptest.NewRecorder()
			testHandler.ServeHTTP(rr, req)
			Expect(rr).To(HaveHTTPStatus(http.StatusOK))
		})
		It("ensures the x-access-token header when public-token URL parameter and BasicAuth are set", func() {
			req := httptest.NewRequest("PROPFIND", "http://example.com/dav/public-files/?public-token=sharetoken", http.NoBody)
			req.SetBasicAuth("public", "examples3cr3t")
			req = req.WithContext(router.SetRoutingInfo(context.Background(), router.RoutingInfo{}))

			handler := Authentication(authenticators,
				EnableBasicAuth(true),
			)
			testHandler := handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				Expect(r.Header.Get(headerRevaAccessToken)).To(Equal("exampletoken"))
			}))
			rr := httptest.NewRecorder()
			testHandler.ServeHTTP(rr, req)
			Expect(rr).To(HaveHTTPStatus(http.StatusOK))
		})
		It("ensures the x-access-token header when public-token BasicAuth is set", func() {
			req := httptest.NewRequest("GET", "http://example.com/archiver", http.NoBody)
			req.Header.Set("public-token", "sharetoken")
			req = req.WithContext(router.SetRoutingInfo(context.Background(), router.RoutingInfo{}))

			handler := Authentication(authenticators,
				EnableBasicAuth(true),
			)
			testHandler := handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				Expect(r.Header.Get(headerRevaAccessToken)).To(Equal("otherexampletoken"))
			}))
			rr := httptest.NewRecorder()
			testHandler.ServeHTTP(rr, req)
			Expect(rr).To(HaveHTTPStatus(http.StatusOK))
		})
	})
})

var _ = Describe("Authenticating guest session requests", Label("Authentication"), func() {
	const (
		guestID    = "guest@example.com"
		guestToken = "guest-reva-token"
		cookieName = "__Host-oc_guest_session"
		fileURL    = "http://example.com/dav/files/guest@example.com/file.txt"
	)

	var (
		authenticators       []Authenticator
		signer               *signedurl.JWTSignedURL
		guestAuthCalls       int
		publicShareAuthCalls int
	)

	serve := func(req *http.Request) (*httptest.ResponseRecorder, *http.Request) {
		var seen *http.Request
		req = req.WithContext(router.SetRoutingInfo(req.Context(), router.RoutingInfo{}))
		handler := Authentication(authenticators)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			seen = r
			w.WriteHeader(http.StatusOK)
		}))
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		return rr, seen
	}

	guestGet := func(target string) *http.Request {
		req := httptest.NewRequest(http.MethodGet, target, http.NoBody)
		req.AddCookie(&http.Cookie{Name: cookieName, Value: "valid-guest-session"})
		return req
	}

	signedGet := func(id, subject string, ttl time.Duration) *http.Request {
		signedURL, err := signer.Sign("http://example.com/dav/files/"+id+"/file.txt", subject, ttl)
		Expect(err).ToNot(HaveOccurred())
		return guestGet(signedURL)
	}

	BeforeEach(func() {
		pool.RemoveSelector("GatewaySelector" + "eu.opencloud.api.gateway")

		guestAuthCalls = 0
		publicShareAuthCalls = 0

		logger := log.NewLogger()
		guestUser := &userv1beta1.User{
			Id:       &userv1beta1.UserId{OpaqueId: guestID, Type: userv1beta1.UserType_USER_TYPE_GUEST},
			Username: guestID,
		}
		accountUser := &userv1beta1.User{
			Id:       &userv1beta1.UserId{OpaqueId: "account-id"},
			Username: "account",
		}

		selector := pool.GetSelector[gateway.GatewayAPIClient](
			"GatewaySelector",
			"eu.opencloud.api.gateway",
			func(cc grpc.ClientConnInterface) gateway.GatewayAPIClient {
				return mockGatewayClientWithUser{
					AuthenticateFunc: func(authType, clientID, clientSecret string) (string, *userv1beta1.User, rpcv1beta1.Code) {
						switch authType {
						case "publicshares":
							publicShareAuthCalls++
							return "public-share-token", nil, rpcv1beta1.Code_CODE_OK
						case "guestlinks":
							guestAuthCalls++
							if clientSecret == "valid-guest-session" {
								return guestToken, guestUser, rpcv1beta1.Code_CODE_OK
							}
						}
						return "", nil, rpcv1beta1.Code_CODE_UNAUTHENTICATED
					},
				}
			},
		)

		var err error
		signer, err = signedurl.NewJWTSignedURL(signedurl.WithSecret("secret"))
		Expect(err).ToNot(HaveOccurred())

		ub := &mocks.UserBackend{}
		ub.On("GetUserByClaims", mock.Anything, "userid", guestID).Return(nil, "", backend.ErrAccountNotFound)
		ub.On("GetUserByClaims", mock.Anything, "userid", "other@example.com").Return(nil, "", backend.ErrAccountNotFound)
		ub.On("GetUserByClaims", mock.Anything, "userid", "account-id").Return(accountUser, "", nil)

		ra := &userrolesmocks.UserRoleAssigner{}
		ra.On("ApplyUserRole", mock.Anything, mock.Anything).Return(accountUser, nil)

		oc := &oidcmocks.OIDCClient{}
		oc.On("VerifyAccessToken", mock.Anything, mock.Anything).Return(
			oidc.RegClaimsWithSID{},
			jwt.MapClaims{},
			errors.New("token is expired"),
		)

		authenticators = []Authenticator{
			&OIDCAuthenticator{
				OIDCIss:       "http://idp.example.com",
				Logger:        logger,
				oidcClient:    oc,
				userInfoCache: store.NewMemoryStore(),
				skipUserInfo:  true,
			},
			PublicShareAuthenticator{
				Logger:              logger,
				RevaGatewaySelector: selector,
			},
			SignedURLAuthenticator{
				Logger:             logger,
				PreSignedURLConfig: config.PreSignedURL{AllowedHTTPMethods: []string{http.MethodGet}},
				UserProvider:       ub,
				UserRoleAssigner:   ra,
				Now:                time.Now,
				URLVerifier:        signer,
			},
			&GuestLinkAuthenticator{
				Logger:              logger,
				RevaGatewaySelector: selector,
				Config:              GuestLinkAuthConfig{CookieName: cookieName},
				URLVerifier:         signer,
			},
		}
	})

	It("authenticates a guest cookie only request as the guest", func() {
		rr, seen := serve(guestGet(fileURL))

		Expect(rr).To(HaveHTTPStatus(http.StatusOK))
		Expect(seen).ToNot(BeNil())
		user, ok := revactx.ContextGetUser(seen.Context())
		Expect(ok).To(BeTrue())
		Expect(user.GetId().GetOpaqueId()).To(Equal(guestID))
		Expect(seen.Header.Get(revactx.TokenHeader)).To(Equal(guestToken))
	})

	It("authenticates a guest cookie with a signed URL issued for the guest as the guest", func() {
		rr, seen := serve(signedGet(guestID, guestID, time.Minute))

		Expect(rr).To(HaveHTTPStatus(http.StatusOK))
		Expect(seen).ToNot(BeNil())
		user, ok := revactx.ContextGetUser(seen.Context())
		Expect(ok).To(BeTrue())
		Expect(user.GetId().GetOpaqueId()).To(Equal(guestID))
		Expect(seen.Header.Get(revactx.TokenHeader)).To(Equal(guestToken))
	})

	It("rejects a guest cookie with a signed URL issued for another user", func() {
		rr, seen := serve(signedGet(guestID, "other@example.com", time.Minute))

		Expect(rr).To(HaveHTTPStatus(http.StatusUnauthorized))
		Expect(seen).To(BeNil())
	})

	It("rejects a guest cookie with a tampered signed URL", func() {
		req := signedGet(guestID, guestID, time.Minute)
		req.URL.RawQuery = "oc-jwt-sig=invalid"
		rr, seen := serve(req)

		Expect(rr).To(HaveHTTPStatus(http.StatusUnauthorized))
		Expect(seen).To(BeNil())
		Expect(guestAuthCalls).To(BeZero())
	})

	It("rejects a guest cookie with an expired signed URL", func() {
		rr, seen := serve(signedGet(guestID, guestID, -time.Minute))

		Expect(rr).To(HaveHTTPStatus(http.StatusUnauthorized))
		Expect(seen).To(BeNil())
		Expect(guestAuthCalls).To(BeZero())
	})

	It("authenticates a guest cookie with a signed URL issued for an account as the account", func() {
		rr, seen := serve(signedGet("account-id", "account-id", time.Minute))

		Expect(rr).To(HaveHTTPStatus(http.StatusOK))
		Expect(seen).ToNot(BeNil())
		user, ok := revactx.ContextGetUser(seen.Context())
		Expect(ok).To(BeTrue())
		Expect(user.GetId().GetOpaqueId()).To(Equal("account-id"))
		Expect(guestAuthCalls).To(BeZero())
	})

	It("does not fall back to the guest when the bearer token is expired", func() {
		req := guestGet(fileURL)
		req.Header.Set(_headerAuthorization, "Bearer expired-token")
		rr, seen := serve(req)

		Expect(rr).To(HaveHTTPStatus(http.StatusUnauthorized))
		Expect(seen).To(BeNil())
		Expect(guestAuthCalls).To(BeZero())
	})

	It("authenticates a TUS upload creation with an unusable Authorization header as the guest", func() {
		req := httptest.NewRequest(http.MethodPost, "http://example.com/dav/spaces/space-id/", http.NoBody)
		req.AddCookie(&http.Cookie{Name: cookieName, Value: "valid-guest-session"})
		req.Header.Set(_headerAuthorization, "undefined")
		req.Header.Set("Tus-Resumable", "1.0.0")
		rr, seen := serve(req)

		Expect(rr).To(HaveHTTPStatus(http.StatusOK))
		Expect(seen).ToNot(BeNil())
		user, ok := revactx.ContextGetUser(seen.Context())
		Expect(ok).To(BeTrue())
		Expect(user.GetId().GetOpaqueId()).To(Equal(guestID))
		Expect(seen.Header.Get(revactx.TokenHeader)).To(Equal(guestToken))
	})

	It("authenticates a guest cookie with a public share token as the public share", func() {
		req := guestGet("http://example.com/dav/public-files/sharetoken/file.txt")
		req.Header.Set(headerShareToken, "sharetoken")
		rr, seen := serve(req)

		Expect(rr).To(HaveHTTPStatus(http.StatusOK))
		Expect(seen).ToNot(BeNil())
		Expect(seen.Header.Get(revactx.TokenHeader)).To(Equal("public-share-token"))
		Expect(guestAuthCalls).To(BeZero())
		Expect(publicShareAuthCalls).To(Equal(1))
	})
})
