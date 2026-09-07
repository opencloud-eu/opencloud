package svc_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"

	gateway "github.com/cs3org/go-cs3apis/cs3/gateway/v1beta1"
	userpb "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"
	provider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	"github.com/go-chi/chi/v5"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc"

	revactx "github.com/opencloud-eu/reva/v2/pkg/ctx"
	"github.com/opencloud-eu/reva/v2/pkg/rgrpc/status"
	"github.com/opencloud-eu/reva/v2/pkg/rgrpc/todo/pool"
	"github.com/opencloud-eu/reva/v2/pkg/signedurl"
	cs3mocks "github.com/opencloud-eu/reva/v2/tests/cs3mocks/mocks"

	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/pkg/shared"
	"github.com/opencloud-eu/opencloud/services/graph/mocks"
	"github.com/opencloud-eu/opencloud/services/graph/pkg/config/defaults"
	identitymocks "github.com/opencloud-eu/opencloud/services/graph/pkg/identity/mocks"
	"github.com/opencloud-eu/opencloud/services/graph/pkg/metrics"
	service "github.com/opencloud-eu/opencloud/services/graph/pkg/service/v0"
)

const urlSigningSecret = "url-signing-secret"

// verifySignedDownloadURL checks the signature of a download url for the given user and returns the url
func verifySignedDownloadURL(signed, userID string) *url.URL {
	GinkgoHelper()

	verifier, err := signedurl.NewJWTSignedURL(signedurl.WithSecret(urlSigningSecret))
	Expect(err).ToNot(HaveOccurred())
	subject, err := verifier.Verify(signed)
	Expect(err).ToNot(HaveOccurred())
	Expect(subject).To(Equal(userID))

	u, err := url.Parse(signed)
	Expect(err).ToNot(HaveOccurred())
	return u
}

var _ = Describe("GetDriveItemContent", func() {
	const (
		driveID = "storageid$spaceid"
		itemID  = "storageid$spaceid!nodeid"
	)

	var (
		ctx             context.Context
		gatewayClient   *cs3mocks.GatewayAPIClient
		gatewaySelector pool.Selectable[gateway.GatewayAPIClient]
		eventsPublisher mocks.Publisher
		rr              *httptest.ResponseRecorder
		fileInfo        *provider.ResourceInfo

		currentUser = &userpb.User{Id: &userpb.UserId{OpaqueId: "user"}}
	)

	newService := func(signingSecret string) service.Service {
		logger := log.NewLogger()
		metrics, _ := metrics.New(prometheus.NewRegistry(), &logger, func([]string) (string, string) { return "", "" })

		cfg := defaults.FullDefaultConfig()
		cfg.Identity.LDAP.CACert = ""
		cfg.TokenManager.JWTSecret = "loremipsum"
		cfg.Commons = &shared.Commons{URLSigningSecret: signingSecret}
		cfg.GRPCClientTLS = &shared.GRPCClientTLS{}

		svc, err := service.NewService(
			service.Config(cfg),
			service.Metrics(metrics),
			service.WithGatewaySelector(gatewaySelector),
			service.EventsPublisher(&eventsPublisher),
			service.WithIdentityBackend(&identitymocks.Backend{}),
		)
		Expect(err).ToNot(HaveOccurred())
		return svc
	}

	newRequest := func(withUser bool) *http.Request {
		r := httptest.NewRequest(http.MethodGet, "/graph/v1beta1/drives/"+driveID+"/items/"+itemID+"/content", nil)
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("driveID", driveID)
		rctx.URLParams.Add("itemID", itemID)
		c := ctx
		if withUser {
			c = revactx.ContextSetUser(c, currentUser)
		}
		return r.WithContext(context.WithValue(c, chi.RouteCtxKey, rctx))
	}

	BeforeEach(func() {
		eventsPublisher.On("Publish", mock.Anything, mock.Anything, mock.Anything).Return(nil)

		pool.RemoveSelector("GatewaySelector" + "eu.opencloud.api.gateway")
		gatewayClient = &cs3mocks.GatewayAPIClient{}
		gatewaySelector = pool.GetSelector[gateway.GatewayAPIClient](
			"GatewaySelector",
			"eu.opencloud.api.gateway",
			func(cc grpc.ClientConnInterface) gateway.GatewayAPIClient {
				return gatewayClient
			},
		)

		rr = httptest.NewRecorder()
		ctx = context.Background()

		fileInfo = &provider.ResourceInfo{
			Type: provider.ResourceType_RESOURCE_TYPE_FILE,
			Id:   &provider.ResourceId{StorageId: "storageid", SpaceId: "spaceid", OpaqueId: "nodeid"},
		}
		gatewayClient.On("Stat", mock.Anything, mock.Anything).Return(&provider.StatResponse{
			Status: status.NewOK(ctx),
			Info:   fileInfo,
		}, nil)
	})

	It("redirects to a signed download url", func() {
		newService(urlSigningSecret).GetDriveItemContent(rr, newRequest(true))
		Expect(rr.Code).To(Equal(http.StatusFound))

		target := verifySignedDownloadURL(rr.Header().Get("Location"), "user")
		Expect(target.Path).To(Equal("/dav/spaces/" + itemID))
	})

	It("returns 404 for a folder", func() {
		fileInfo.Type = provider.ResourceType_RESOURCE_TYPE_CONTAINER

		newService(urlSigningSecret).GetDriveItemContent(rr, newRequest(true))
		Expect(rr.Code).To(Equal(http.StatusNotFound))
	})

	It("returns 404 for an unknown item", func() {
		gatewayClient.ExpectedCalls = nil
		gatewayClient.On("Stat", mock.Anything, mock.Anything).Return(&provider.StatResponse{Status: status.NewNotFound(ctx, "not found")}, nil)

		newService(urlSigningSecret).GetDriveItemContent(rr, newRequest(true))
		Expect(rr.Code).To(Equal(http.StatusNotFound))
	})

	It("treats permission denied as not found", func() {
		gatewayClient.ExpectedCalls = nil
		gatewayClient.On("Stat", mock.Anything, mock.Anything).Return(&provider.StatResponse{Status: status.NewPermissionDenied(ctx, errors.New("denied"), "denied")}, nil)

		newService(urlSigningSecret).GetDriveItemContent(rr, newRequest(true))
		Expect(rr.Code).To(Equal(http.StatusNotFound))
	})

	It("returns 404 when the item belongs to another drive", func() {
		r := newRequest(true)
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("driveID", "storageid$otherspace")
		rctx.URLParams.Add("itemID", itemID)
		r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))

		newService(urlSigningSecret).GetDriveItemContent(rr, r)
		Expect(rr.Code).To(Equal(http.StatusNotFound))
		gatewayClient.AssertNotCalled(GinkgoT(), "Stat", mock.Anything, mock.Anything)
	})

	It("returns 500 without a user in the context", func() {
		newService(urlSigningSecret).GetDriveItemContent(rr, newRequest(false))
		Expect(rr.Code).To(Equal(http.StatusInternalServerError))
	})

	It("returns 500 when url signing is not configured", func() {
		newService("").GetDriveItemContent(rr, newRequest(true))
		Expect(rr.Code).To(Equal(http.StatusInternalServerError))
	})
})
