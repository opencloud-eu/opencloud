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

				req2, valid := authenticator.Authenticate(req)

				Expect(valid).To(Equal(true))
				Expect(req2).ToNot(BeNil())

				h := req2.Header
				Expect(h.Get(headerRevaAccessToken)).To(Equal("exampletoken"))
			})
		})
		Context("using signature authentication", func() {
			It("should successfully authenticate", func() {
				req := httptest.NewRequest(http.MethodGet, "http://example.com/dav/public-files/?public-token=sharetoken&signature=examplesignature&expiration=exampleexpiration", http.NoBody)

				req2, valid := authenticator.Authenticate(req)

				Expect(valid).To(Equal(true))
				Expect(req2).ToNot(BeNil())

				h := req2.Header
				Expect(h.Get(headerRevaAccessToken)).To(Equal("exampletoken"))
			})
		})
	})
	When("the reguest is for the archiver", func() {
		Context("using a public-token", func() {
			It("should successfully authenticate", func() {
				req := httptest.NewRequest(http.MethodGet, "http://example.com/archiver?public-token=sharetoken", http.NoBody)
				req2, valid := authenticator.Authenticate(req)

				Expect(valid).To(Equal(true))
				Expect(req2).ToNot(BeNil())

				h := req2.Header
				Expect(h.Get(headerRevaAccessToken)).To(Equal("otherexampletoken"))
			})
		})
		Context("not using a public-token", func() {
			It("should fail to authenticate", func() {
				req := httptest.NewRequest(http.MethodGet, "http://example.com/archiver", http.NoBody)
				req2, valid := authenticator.Authenticate(req)

				Expect(valid).To(Equal(false))
				Expect(req2).To(BeNil())
			})
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
