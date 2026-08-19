package middleware_test

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	gateway "github.com/cs3org/go-cs3apis/cs3/gateway/v1beta1"
	rpc "github.com/cs3org/go-cs3apis/cs3/rpc/v1beta1"
	provider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	. "github.com/onsi/gomega"
	"github.com/opencloud-eu/reva/v2/pkg/rgrpc/todo/pool"
	cs3mocks "github.com/opencloud-eu/reva/v2/tests/cs3mocks/mocks"
	"github.com/stretchr/testify/mock"
	"go-micro.dev/v4/client"
	"google.golang.org/grpc"

	pMessage "github.com/opencloud-eu/opencloud/protogen/gen/opencloud/messages/policies/v0"
	policiesPG "github.com/opencloud-eu/opencloud/protogen/gen/opencloud/services/policies/v0"
	"github.com/opencloud-eu/opencloud/protogen/gen/opencloud/services/policies/v0/mocks"
	"github.com/opencloud-eu/opencloud/services/proxy/pkg/middleware"
	"github.com/opencloud-eu/opencloud/services/webdav/pkg/net"
)

func TestPolicies_NoQuery_PassThrough(t *testing.T) {
	var g = NewWithT(t)

	policiesMiddleware, _, _ := prepare("")

	responseRecorder := httptest.NewRecorder()
	policiesMiddleware.ServeHTTP(responseRecorder, httptest.NewRequest(http.MethodGet, "/policies", nil))

	g.Expect(responseRecorder.Code).To(Equal(http.StatusOK))
}

func TestPolicies_ErrorsOnEvaluationError(t *testing.T) {
	var g = NewWithT(t)

	policiesMiddleware, policiesProviderService, _ := prepare("any")
	policiesProviderService.On("Evaluate", mock.Anything, mock.Anything).Return(
		nil,
		errors.New("any"),
	)

	responseRecorder := httptest.NewRecorder()
	policiesMiddleware.ServeHTTP(responseRecorder, httptest.NewRequest(http.MethodGet, "/policies", nil))

	g.Expect(responseRecorder.Code).To(Equal(http.StatusInternalServerError))
}

func TestPolicies_ErrorsOnDeny(t *testing.T) {
	var g = NewWithT(t)

	policiesMiddleware, policiesProviderService, _ := prepare("any")
	policiesProviderService.On("Evaluate", mock.Anything, mock.Anything).Return(
		&policiesPG.EvaluateResponse{},
		nil,
	)

	responseRecorder := httptest.NewRecorder()
	policiesMiddleware.ServeHTTP(responseRecorder, httptest.NewRequest(http.MethodGet, "/policies", nil))

	result := responseRecorder.Result()
	defer func() {
		g.Expect(result.Body.Close()).ToNot(HaveOccurred())
	}()

	data, err := io.ReadAll(result.Body)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(data).To(ContainSubstring(middleware.DeniedMessage))
	g.Expect(responseRecorder.Code).To(Equal(http.StatusForbidden))
}

func TestPolicies_EvaluationEnvironment_HTTPStage(t *testing.T) {
	var g = NewWithT(t)

	policiesMiddleware, policiesProviderService, _ := prepare("any")
	policiesProviderService.On("Evaluate", mock.Anything, mock.Anything, mock.Anything).Return(
		func(_ context.Context, in *policiesPG.EvaluateRequest, _ ...client.CallOption) (*policiesPG.EvaluateResponse, error) {
			g.Expect(in.Environment.Stage).To(Equal(pMessage.Stage_STAGE_HTTP))

			return &policiesPG.EvaluateResponse{Result: false}, nil
		},
	)

	responseRecorder := httptest.NewRecorder()
	policiesMiddleware.ServeHTTP(responseRecorder, httptest.NewRequest(http.MethodGet, "/policies", nil))
}

func TestPolicies_EvaluationEnvironment_Request(t *testing.T) {
	var g = NewWithT(t)

	policiesMiddleware, policiesProviderService, _ := prepare("any")
	policiesProviderService.On("Evaluate", mock.Anything, mock.Anything, mock.Anything).Return(
		func(_ context.Context, in *policiesPG.EvaluateRequest, _ ...client.CallOption) (*policiesPG.EvaluateResponse, error) {
			g.Expect(in.Environment.Request.Method).To(Equal(http.MethodDelete))
			g.Expect(in.Environment.Request.Path).To(Equal("/whatever"))

			return &policiesPG.EvaluateResponse{Result: false}, nil
		},
	)

	responseRecorder := httptest.NewRecorder()
	policiesMiddleware.ServeHTTP(responseRecorder, httptest.NewRequest(http.MethodDelete, "/whatever", nil))
}

func TestPolicies_EvaluationEnvironment_Resource(t *testing.T) {
	var g = NewWithT(t)

	policiesMiddleware, policiesProviderService, _ := prepare("any")

	// tus metadata
	{
		responseRecorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodPost, "/remote.php/dav/spaces", nil)
		request.Header.Set(net.HeaderUploadMetadata, fmt.Sprintf("filename %v", base64.StdEncoding.EncodeToString([]byte("tus-file-name.png"))))
		policiesProviderService.On("Evaluate", mock.Anything, mock.Anything, mock.Anything).Return(
			func(_ context.Context, in *policiesPG.EvaluateRequest, _ ...client.CallOption) (*policiesPG.EvaluateResponse, error) {
				g.Expect(in.Environment.Resource.Name).To(Equal("tus-file-name.png"))

				return &policiesPG.EvaluateResponse{Result: false}, nil
			},
		).Once()
		policiesMiddleware.ServeHTTP(responseRecorder, request)
	}

	// url path
	{
		responseRecorder := httptest.NewRecorder()
		policiesProviderService.On("Evaluate", mock.Anything, mock.Anything, mock.Anything).Return(
			func(_ context.Context, in *policiesPG.EvaluateRequest, _ ...client.CallOption) (*policiesPG.EvaluateResponse, error) {
				g.Expect(in.Environment.Resource.Name).To(Equal("simple-file-name.png"))

				return &policiesPG.EvaluateResponse{Result: false}, nil
			},
		).Once()
		policiesMiddleware.ServeHTTP(responseRecorder, httptest.NewRequest(http.MethodPut, "/remote.php/dav/spaces/simple-file-name.png", nil))
	}
}

func TestPolicies_EvaluatesWithEmptyNameWhenStatFails(t *testing.T) {
	const spaceRef = "/remote.php/dav/spaces/storage-id$space-id!opaque-id"

	for _, tc := range []struct {
		name string
		stat func(*cs3mocks.GatewayAPIClient)
	}{
		{
			name: "transport error",
			stat: func(c *cs3mocks.GatewayAPIClient) {
				c.On("Stat", mock.Anything, mock.Anything).Return(nil, errors.New("any")).Once()
			},
		},
		{
			name: "non ok status",
			stat: func(c *cs3mocks.GatewayAPIClient) {
				c.On("Stat", mock.Anything, mock.Anything).Return(&provider.StatResponse{
					Status: &rpc.Status{Code: rpc.Code_CODE_NOT_FOUND},
				}, nil).Once()
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var g = NewWithT(t)

			policiesMiddleware, policiesProviderService, gatewayClient := prepare("any")
			tc.stat(gatewayClient)

			policiesProviderService.On("Evaluate", mock.Anything, mock.Anything, mock.Anything).Return(
				func(_ context.Context, in *policiesPG.EvaluateRequest, _ ...client.CallOption) (*policiesPG.EvaluateResponse, error) {
					g.Expect(in.Environment.Resource.Name).To(BeEmpty())

					return &policiesPG.EvaluateResponse{Result: false}, nil
				},
			).Once()

			responseRecorder := httptest.NewRecorder()
			policiesMiddleware.ServeHTTP(responseRecorder, httptest.NewRequest(http.MethodPut, spaceRef, nil))

			policiesProviderService.AssertCalled(t, "Evaluate", mock.Anything, mock.Anything, mock.Anything)
		})
	}
}

func TestPolicies_EvaluatesStattedName(t *testing.T) {
	var g = NewWithT(t)

	policiesMiddleware, policiesProviderService, gatewayClient := prepare("any")
	gatewayClient.On("Stat", mock.Anything, mock.Anything).Return(&provider.StatResponse{
		Status: &rpc.Status{Code: rpc.Code_CODE_OK},
		Info:   &provider.ResourceInfo{Name: "statted-file-name.png"},
	}, nil).Once()

	policiesProviderService.On("Evaluate", mock.Anything, mock.Anything, mock.Anything).Return(
		func(_ context.Context, in *policiesPG.EvaluateRequest, _ ...client.CallOption) (*policiesPG.EvaluateResponse, error) {
			g.Expect(in.Environment.Resource.Name).To(Equal("statted-file-name.png"))

			return &policiesPG.EvaluateResponse{Result: true}, nil
		},
	).Once()

	responseRecorder := httptest.NewRecorder()
	policiesMiddleware.ServeHTTP(responseRecorder, httptest.NewRequest(http.MethodPut, "/remote.php/dav/spaces/storage-id$space-id!opaque-id", nil))

	g.Expect(responseRecorder.Code).To(Equal(http.StatusOK))
}

func prepare(q string) (http.Handler, *mocks.PoliciesProviderService, *cs3mocks.GatewayAPIClient) {

	// mocked gatewaySelector
	gatewayClient := &cs3mocks.GatewayAPIClient{}
	gatewaySelector := pool.GetSelector[gateway.GatewayAPIClient](
		"GatewaySelector",
		"eu.opencloud.api.gateway",
		func(cc grpc.ClientConnInterface) gateway.GatewayAPIClient {
			return gatewayClient
		},
	)
	defer pool.RemoveSelector("GatewaySelector" + "eu.opencloud.api.gateway")

	// mocked policiesProviderService
	policiesProviderService := &mocks.PoliciesProviderService{}

	// spin up middleware
	policiesMiddleware := middleware.Policies(
		q,
		middleware.WithRevaGatewaySelector(gatewaySelector),
		middleware.PoliciesProviderService(policiesProviderService),
	)(mockHandler{})

	return policiesMiddleware, policiesProviderService, gatewayClient
}

type mockHandler struct{}

func (m mockHandler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {}
