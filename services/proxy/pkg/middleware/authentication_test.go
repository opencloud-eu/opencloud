package middleware

import (
	"context"
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
	"github.com/opencloud-eu/opencloud/services/proxy/pkg/router"
	"github.com/opencloud-eu/opencloud/services/proxy/pkg/user/backend"
	"github.com/opencloud-eu/opencloud/services/proxy/pkg/user/backend/mocks"
	"github.com/opencloud-eu/reva/v2/pkg/rgrpc/todo/pool"
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
				Expect(oidc.FromContext(r.Context())).To(Equal(map[string]interface{}{
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
				Expect(oidc.FromContext(r.Context())).To(Equal(map[string]interface{}{
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
