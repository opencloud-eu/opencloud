package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"

	gateway "github.com/cs3org/go-cs3apis/cs3/gateway/v1beta1"
	gatewayv1beta1 "github.com/cs3org/go-cs3apis/cs3/gateway/v1beta1"
	rpcv1beta1 "github.com/cs3org/go-cs3apis/cs3/rpc/v1beta1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/opencloud-eu/opencloud/pkg/log"
	pkgmiddleware "github.com/opencloud-eu/opencloud/pkg/middleware"
	"github.com/opencloud-eu/reva/v2/pkg/rgrpc/todo/pool"
	"google.golang.org/grpc"
)

var _ = Describe("Authenticating requests", Label("PublicShareAuthenticator"), func() {
	var authenticator Authenticator
	BeforeEach(func() {
		pool.RemoveSelector("GatewaySelector" + "eu.opencloud.api.gateway")
		authenticator = PublicShareAuthenticator{
			Logger: log.NewLogger(),
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

							if clientID == "protectedtoken" {
								if clientSecret == "password|right" {
									return "protectedexampletoken", rpcv1beta1.Code_CODE_OK
								}
								return "", rpcv1beta1.Code_CODE_UNAUTHENTICATED
							}

							return "", rpcv1beta1.Code_CODE_NOT_FOUND
						},
					}
				},
			),
		}
	})
	When("the request contains correct data", func() {
		Context("using password authentication", func() {
			It("should successfully authenticate", func() {
				req := httptest.NewRequest(http.MethodGet, "http://example.com/dav/public-files/?public-token=sharetoken", http.NoBody)
				req.SetBasicAuth("public", "examples3cr3t")

				result := authenticator.Authenticate(req)

				Expect(result.State).To(Equal(AuthenticationSucceeded))
				Expect(result.Request).ToNot(BeNil())

				h := result.Request.Header
				Expect(h.Get(headerRevaAccessToken)).To(Equal("exampletoken"))
			})
		})
		Context("using signature authentication", func() {
			It("should successfully authenticate", func() {
				req := httptest.NewRequest(http.MethodGet, "http://example.com/dav/public-files/?public-token=sharetoken&signature=examplesignature&expiration=exampleexpiration", http.NoBody)

				result := authenticator.Authenticate(req)

				Expect(result.State).To(Equal(AuthenticationSucceeded))
				Expect(result.Request).ToNot(BeNil())

				h := result.Request.Header
				Expect(h.Get(headerRevaAccessToken)).To(Equal("exampletoken"))
			})
		})
	})
	When("the reguest is for the archiver", func() {
		Context("using a public-token", func() {
			It("should successfully authenticate", func() {
				req := httptest.NewRequest(http.MethodGet, "http://example.com/archiver?public-token=sharetoken", http.NoBody)
				result := authenticator.Authenticate(req)

				Expect(result.State).To(Equal(AuthenticationSucceeded))
				Expect(result.Request).ToNot(BeNil())

				h := result.Request.Header
				Expect(h.Get(headerRevaAccessToken)).To(Equal("otherexampletoken"))
			})
		})
		Context("not using a public-token", func() {
			It("should not be applicable", func() {
				req := httptest.NewRequest(http.MethodGet, "http://example.com/archiver", http.NoBody)
				result := authenticator.Authenticate(req)

				Expect(result.State).To(Equal(AuthenticationNotApplicable))
			})
		})
	})
	When("the request is a graph request", func() {
		DescribeTable("passes the request on and tells graph why the link was rejected",
			func(token, password, wantAccessToken, wantHint string) {
				req := httptest.NewRequest(http.MethodGet, "http://example.com/graph/v1.0/drives/x/items/x/children?public-token="+token, http.NoBody)
				if password != "" {
					req.SetBasicAuth("public", password)
				}

				result := authenticator.Authenticate(req)

				Expect(result.State).To(Equal(AuthenticationSucceeded))
				Expect(result.Request.Header.Get(headerRevaAccessToken)).To(Equal(wantAccessToken))
				Expect(result.Request.Header.Get(pkgmiddleware.PublicLinkAuthHeader)).To(Equal(wantHint))
			},
			Entry("right password", "protectedtoken", "right", "protectedexampletoken", ""),
			Entry("no password", "protectedtoken", "", "", pkgmiddleware.PublicLinkPasswordRequired),
			Entry("wrong password", "protectedtoken", "wrong", "", pkgmiddleware.PublicLinkInvalidPassword),
			Entry("unknown token without password", "unknowntoken", "", "", ""),
			Entry("unknown token with password", "unknowntoken", "secret", "", ""),
		)
	})
	When("a public-files request is rejected", func() {
		It("passes it on without a hint, ocdav renders the 401", func() {
			req := httptest.NewRequest(http.MethodGet, "http://example.com/dav/public-files/?public-token=protectedtoken", http.NoBody)
			req.SetBasicAuth("public", "wrong")

			result := authenticator.Authenticate(req)

			Expect(result.State).To(Equal(AuthenticationSucceeded))
			Expect(result.Request.Header.Get(pkgmiddleware.PublicLinkAuthHeader)).To(BeEmpty())
		})
	})
})

type mockGatewayClient struct {
	gatewayv1beta1.GatewayAPIClient
	AuthenticateFunc func(authType, clientID, clientSecret string) (string, rpcv1beta1.Code)
}

func (c mockGatewayClient) Authenticate(ctx context.Context, in *gatewayv1beta1.AuthenticateRequest, opts ...grpc.CallOption) (*gatewayv1beta1.AuthenticateResponse, error) {
	token, code := c.AuthenticateFunc(in.GetType(), in.GetClientId(), in.GetClientSecret())
	return &gatewayv1beta1.AuthenticateResponse{
		Status: &rpcv1beta1.Status{Code: code},
		Token:  token,
	}, nil
}
