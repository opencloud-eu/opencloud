package svc_test

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"

	gateway "github.com/cs3org/go-cs3apis/cs3/gateway/v1beta1"
	userpb "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"
	provider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc"

	revactx "github.com/opencloud-eu/reva/v2/pkg/ctx"
	"github.com/opencloud-eu/reva/v2/pkg/events"
	"github.com/opencloud-eu/reva/v2/pkg/rgrpc/status"
	"github.com/opencloud-eu/reva/v2/pkg/rgrpc/todo/pool"
	cs3mocks "github.com/opencloud-eu/reva/v2/tests/cs3mocks/mocks"

	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/pkg/shared"
	"github.com/opencloud-eu/opencloud/services/graph/mocks"
	"github.com/opencloud-eu/opencloud/services/graph/pkg/config"
	"github.com/opencloud-eu/opencloud/services/graph/pkg/config/defaults"
	identitymocks "github.com/opencloud-eu/opencloud/services/graph/pkg/identity/mocks"
	"github.com/opencloud-eu/opencloud/services/graph/pkg/metrics"
	service "github.com/opencloud-eu/opencloud/services/graph/pkg/service/v0"
)

var _ = Describe("Tags", func() {
	var (
		svc             service.Service
		ctx             context.Context
		cfg             *config.Config
		gatewayClient   *cs3mocks.GatewayAPIClient
		gatewaySelector pool.Selectable[gateway.GatewayAPIClient]
		eventsPublisher *mocks.Publisher
		rr              *httptest.ResponseRecorder
		published       []events.TagsAdded

		currentUser = &userpb.User{Id: &userpb.UserId{OpaqueId: "user"}}
		resourceID  = "storage$space!node"
	)

	BeforeEach(func() {
		published = nil
		eventsPublisher = &mocks.Publisher{}
		eventsPublisher.On("Publish", mock.Anything, mock.Anything, mock.Anything).
			Run(func(args mock.Arguments) {
				if ev, ok := args.Get(1).(events.TagsAdded); ok {
					published = append(published, ev)
				}
			}).Return(nil)

		pool.RemoveSelector("GatewaySelector" + "eu.opencloud.api.gateway")
		gatewayClient = &cs3mocks.GatewayAPIClient{}
		gatewaySelector = pool.GetSelector[gateway.GatewayAPIClient](
			"GatewaySelector",
			"eu.opencloud.api.gateway",
			func(cc grpc.ClientConnInterface) gateway.GatewayAPIClient {
				return gatewayClient
			},
		)

		logger := log.NewLogger()
		m, _ := metrics.New(prometheus.NewRegistry(), &logger, func([]string) (string, string) { return "", "" })
		rr = httptest.NewRecorder()
		ctx = context.Background()

		cfg = defaults.FullDefaultConfig()
		cfg.Identity.LDAP.CACert = ""
		cfg.TokenManager.JWTSecret = "loremipsum"
		cfg.Commons = &shared.Commons{}
		cfg.GRPCClientTLS = &shared.GRPCClientTLS{}

		var err error
		svc, err = service.NewService(
			service.Config(cfg),
			service.Metrics(m),
			service.WithGatewaySelector(gatewaySelector),
			service.EventsPublisher(eventsPublisher),
			service.WithIdentityBackend(&identitymocks.Backend{}),
		)
		Expect(err).ToNot(HaveOccurred())
	})

	numberedTags := func(prefix string, n int) []string {
		ts := make([]string, 0, n)
		for i := 0; i < n; i++ {
			ts = append(ts, fmt.Sprintf("%s%d", prefix, i))
		}
		return ts
	}

	stat := func(currentTags string) {
		gatewayClient.On("Stat", mock.Anything, mock.Anything).Return(&provider.StatResponse{
			Status: status.NewOK(ctx),
			Info: &provider.ResourceInfo{
				Id:                &provider.ResourceId{StorageId: "storage", SpaceId: "space", OpaqueId: "node"},
				Owner:             currentUser.Id,
				PermissionSet:     &provider.ResourcePermissions{InitiateFileUpload: true},
				ArbitraryMetadata: &provider.ArbitraryMetadata{Metadata: map[string]string{"tags": currentTags}},
			},
		}, nil)
	}

	assign := func(ts ...string) {
		body := fmt.Sprintf(`{"resourceId":%q,"tags":["%s"]}`, resourceID, strings.Join(ts, `","`))
		r := httptest.NewRequest(http.MethodPut, "/graph/v1.0/extensions/org.libregraph/tags", bytes.NewBufferString(body))
		r = r.WithContext(revactx.ContextSetUser(ctx, currentUser))
		svc.AssignTags(rr, r)
	}

	Describe("AssignTags", func() {
		It("rejects the whole request when the new tags exceed the maximum", func() {
			stat(strings.Join(numberedTags("old", 95), ","))

			assign(numberedTags("new", 10)...)

			Expect(rr.Code).To(Equal(http.StatusBadRequest))
			gatewayClient.AssertNotCalled(GinkgoT(), "SetArbitraryMetadata", mock.Anything, mock.Anything)
			Expect(published).To(BeEmpty())
		})

		It("accepts tags up to the maximum", func() {
			stat(strings.Join(numberedTags("old", 95), ","))
			gatewayClient.On("SetArbitraryMetadata", mock.Anything, mock.Anything).Return(&provider.SetArbitraryMetadataResponse{
				Status: status.NewOK(ctx),
			}, nil)

			assign(numberedTags("new", 5)...)

			Expect(rr.Code).To(Equal(http.StatusOK))
			gatewayClient.AssertNumberOfCalls(GinkgoT(), "SetArbitraryMetadata", 1)
		})

		It("only publishes the tags that were actually added", func() {
			stat("a,b")
			gatewayClient.On("SetArbitraryMetadata", mock.Anything, mock.Anything).Return(&provider.SetArbitraryMetadataResponse{
				Status: status.NewOK(ctx),
			}, nil)

			assign("a", "c", "c", "d")

			Expect(rr.Code).To(Equal(http.StatusOK))
			Expect(published).To(HaveLen(1))
			Expect(published[0].Tags).To(Equal("c,d"))
		})
	})
})
