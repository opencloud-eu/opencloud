package svc_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"

	gateway "github.com/cs3org/go-cs3apis/cs3/gateway/v1beta1"
	userpb "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"
	provider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	"github.com/go-chi/chi/v5"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/mock"
	"github.com/tidwall/gjson"
	"google.golang.org/grpc"

	revactx "github.com/opencloud-eu/reva/v2/pkg/ctx"
	"github.com/opencloud-eu/reva/v2/pkg/rgrpc/status"
	"github.com/opencloud-eu/reva/v2/pkg/rgrpc/todo/pool"
	cs3mocks "github.com/opencloud-eu/reva/v2/tests/cs3mocks/mocks"

	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/pkg/shared"
	"github.com/opencloud-eu/opencloud/services/graph/mocks"
	"github.com/opencloud-eu/opencloud/services/graph/pkg/config"
	"github.com/opencloud-eu/opencloud/services/graph/pkg/config/defaults"
	"github.com/opencloud-eu/opencloud/services/graph/pkg/metrics"
	service "github.com/opencloud-eu/opencloud/services/graph/pkg/service/v0"
)

var _ = Describe("OpenExtensions", func() {
	var (
		svc             service.Service
		ctx             context.Context
		cfg             *config.Config
		gatewayClient   *cs3mocks.GatewayAPIClient
		gatewaySelector pool.Selectable[gateway.GatewayAPIClient]
		eventsPublisher mocks.Publisher
		rr              *httptest.ResponseRecorder

		itemID   = &provider.ResourceId{StorageId: "storageid", SpaceId: "spaceid", OpaqueId: "nodeid"}
		info     *provider.ResourceInfo
		writable = &provider.ResourcePermissions{InitiateFileUpload: true}
		readonly = &provider.ResourcePermissions{Stat: true}

		currentUser = &userpb.User{Id: &userpb.UserId{OpaqueId: "user"}}
	)

	key := func(name, property string) string {
		return "http://opencloud.eu/ns/extensions/" + name + "/" + property
	}

	// request builds a request against the extension routes with the chi
	// parameters set the way the router would.
	request := func(method, name, body string, query ...string) *http.Request {
		var r *http.Request
		if body != "" {
			r = httptest.NewRequest(method, "/graph/v1beta1/drives/storageid$spaceid/items/storageid$spaceid!nodeid/extensions/"+name, strings.NewReader(body))
		} else {
			r = httptest.NewRequest(method, "/graph/v1beta1/drives/storageid$spaceid/items/storageid$spaceid!nodeid/extensions/"+name, nil)
		}
		if len(query) > 0 {
			r.URL.RawQuery = query[0]
		}
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("driveID", "storageid$spaceid")
		rctx.URLParams.Add("itemID", "storageid$spaceid!nodeid")
		rctx.URLParams.Add("driveItemID", "storageid$spaceid!nodeid")
		if name != "" {
			rctx.URLParams.Add("extensionName", name)
		}
		return r.WithContext(context.WithValue(revactx.ContextSetUser(ctx, currentUser), chi.RouteCtxKey, rctx))
	}

	statReturns := func(info *provider.ResourceInfo) {
		gatewayClient.On("Stat", mock.Anything, mock.Anything).Return(&provider.StatResponse{Status: status.NewOK(ctx), Info: info}, nil)
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

		logger := log.NewLogger()
		metrics, _ := metrics.New(prometheus.NewRegistry(), &logger, func([]string) (string, string) { return "", "" })

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
			service.Metrics(metrics),
			service.WithGatewaySelector(gatewaySelector),
			service.EventsPublisher(&eventsPublisher),
		)
		Expect(err).ToNot(HaveOccurred())

		info = &provider.ResourceInfo{
			Type:          provider.ResourceType_RESOURCE_TYPE_FILE,
			Id:            itemID,
			Etag:          "etag",
			PermissionSet: writable,
			ArbitraryMetadata: &provider.ArbitraryMetadata{Metadata: map[string]string{
				"tags":                                 "a,b",
				key("com.example.project", "due"):      "d:2026-10-01T00:00:00Z",
				key("com.example.project", "priority"): "n:3",
				key("com.example.project", "status"):   "s:open",
				key("com.example.audit", "reviewed"):   "b:true",
			}},
		}
	})

	Describe("ListOpenExtensions", func() {
		It("lists every extension with its annotations, sorted by name", func() {
			statReturns(info)
			svc.ListOpenExtensions(rr, request(http.MethodGet, "", ""))
			Expect(rr.Code).To(Equal(http.StatusOK))
			body := rr.Body.String()
			Expect(gjson.Get(body, "value.#").Int()).To(Equal(int64(2)))
			Expect(gjson.Get(body, "value.0.extensionName").String()).To(Equal("com.example.audit"))
			Expect(gjson.Get(body, "value.0.reviewed").Bool()).To(BeTrue())
			Expect(gjson.Get(body, "value.1.extensionName").String()).To(Equal("com.example.project"))
			Expect(gjson.Get(body, "value.1.priority").Int()).To(Equal(int64(3)))
			Expect(gjson.Get(body, "value.1.due@odata\\.type").String()).To(Equal("#DateTimeOffset"))
			Expect(gjson.Get(body, "value.1.status").String()).To(Equal("open"), "the stored type code stays internal")
			Expect(gjson.Get(body, "value.1.status@odata\\.type").Exists()).To(BeFalse(), "inferable types are not annotated")
		})

		It("answers an item without extensions with an empty collection", func() {
			info.ArbitraryMetadata = nil
			statReturns(info)
			svc.ListOpenExtensions(rr, request(http.MethodGet, "", ""))
			Expect(rr.Code).To(Equal(http.StatusOK))
			Expect(rr.Body.String()).To(MatchJSON(`{"value":[]}`))
		})

		It("hides items the caller cannot see", func() {
			gatewayClient.On("Stat", mock.Anything, mock.Anything).Return(&provider.StatResponse{Status: status.NewPermissionDenied(ctx, nil, "denied")}, nil)
			svc.ListOpenExtensions(rr, request(http.MethodGet, "", ""))
			Expect(rr.Code).To(Equal(http.StatusNotFound))
		})
	})

	Describe("GetOpenExtension", func() {
		It("returns the extension", func() {
			statReturns(info)
			svc.GetOpenExtension(rr, request(http.MethodGet, "com.example.project", ""))
			Expect(rr.Code).To(Equal(http.StatusOK))
			Expect(rr.Body.String()).To(MatchJSON(`{"extensionName":"com.example.project","due":"2026-10-01T00:00:00Z","due@odata.type":"#DateTimeOffset","priority":3,"status":"open"}`))
		})

		It("answers 404 for an extension the item does not have", func() {
			statReturns(info)
			svc.GetOpenExtension(rr, request(http.MethodGet, "com.example.other", ""))
			Expect(rr.Code).To(Equal(http.StatusNotFound))
		})

		It("rejects a name that is not reverse DNS", func() {
			svc.GetOpenExtension(rr, request(http.MethodGet, "project", ""))
			Expect(rr.Code).To(Equal(http.StatusBadRequest))
			gatewayClient.AssertNotCalled(GinkgoT(), "Stat", mock.Anything, mock.Anything)
		})
	})

	Describe("UpsertOpenExtension", func() {
		var written map[string]string

		BeforeEach(func() {
			written = nil
			gatewayClient.On("SetArbitraryMetadata", mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
				req := args.Get(1).(*provider.SetArbitraryMetadataRequest)
				Expect(req.GetRef().GetResourceId()).To(Equal(itemID))
				written = req.GetArbitraryMetadata().GetMetadata()
			}).Return(&provider.SetArbitraryMetadataResponse{Status: status.NewOK(ctx)}, nil)
		})

		It("creates an extension, one typed value per property", func() {
			statReturns(info)
			svc.UpsertOpenExtension(rr, request(http.MethodPut, "com.example.new",
				`{"extensionName":"com.example.new","state":"open","site":{"latitude":52.5,"longitude":13.4},"site@odata.type":"#microsoft.graph.geoCoordinates"}`))
			Expect(rr.Code).To(Equal(http.StatusCreated))
			Expect(written).To(Equal(map[string]string{
				key("com.example.new", "site"):  "g:52.5,13.4",
				key("com.example.new", "state"): "s:open",
			}))
			Expect(rr.Body.String()).To(MatchJSON(`{"extensionName":"com.example.new","site":{"latitude":52.5,"longitude":13.4},"site@odata.type":"#microsoft.graph.geoCoordinates","state":"open"}`))
		})

		It("merges into an existing extension, null removes", func() {
			statReturns(info)
			var removed []string
			gatewayClient.On("UnsetArbitraryMetadata", mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
				removed = args.Get(1).(*provider.UnsetArbitraryMetadataRequest).GetArbitraryMetadataKeys()
			}).Return(&provider.UnsetArbitraryMetadataResponse{Status: status.NewOK(ctx)}, nil)
			svc.UpsertOpenExtension(rr, request(http.MethodPut, "com.example.project", `{"status":"approved","priority":null,"effort":2.5}`))
			Expect(rr.Code).To(Equal(http.StatusOK))
			Expect(written).To(Equal(map[string]string{
				key("com.example.project", "status"): "s:approved",
				key("com.example.project", "effort"): "n:2.5",
			}), "only the properties of the request are written")
			Expect(removed).To(Equal([]string{key("com.example.project", "priority")}))
			Expect(rr.Body.String()).To(MatchJSON(`{"extensionName":"com.example.project","due":"2026-10-01T00:00:00Z","due@odata.type":"#DateTimeOffset","effort":2.5,"status":"approved"}`))
		})

		It("re-types a property written without its old annotation", func() {
			statReturns(info)
			svc.UpsertOpenExtension(rr, request(http.MethodPut, "com.example.project", `{"due":"later"}`))
			Expect(rr.Code).To(Equal(http.StatusOK))
			Expect(written).To(Equal(map[string]string{key("com.example.project", "due"): "s:later"}))
		})

		It("skips the removal of a property that is not stored", func() {
			statReturns(info)
			svc.UpsertOpenExtension(rr, request(http.MethodPut, "com.example.project", `{"nope":null}`))
			Expect(rr.Code).To(Equal(http.StatusOK))
			gatewayClient.AssertNotCalled(GinkgoT(), "SetArbitraryMetadata", mock.Anything, mock.Anything)
			gatewayClient.AssertNotCalled(GinkgoT(), "UnsetArbitraryMetadata", mock.Anything, mock.Anything)
		})

		It("answers 204 once every property is removed", func() {
			statReturns(info)
			gatewayClient.On("UnsetArbitraryMetadata", mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
				req := args.Get(1).(*provider.UnsetArbitraryMetadataRequest)
				Expect(req.GetArbitraryMetadataKeys()).To(Equal([]string{key("com.example.audit", "reviewed")}))
			}).Return(&provider.UnsetArbitraryMetadataResponse{Status: status.NewOK(ctx)}, nil)
			svc.UpsertOpenExtension(rr, request(http.MethodPut, "com.example.audit", `{"reviewed":null}`))
			Expect(rr.Code).To(Equal(http.StatusNoContent))
			gatewayClient.AssertNotCalled(GinkgoT(), "SetArbitraryMetadata", mock.Anything, mock.Anything)
		})

		It("rejects an invalid body before touching the storage", func() {
			svc.UpsertOpenExtension(rr, request(http.MethodPut, "com.example.project", `{"assignee":{"name":"alice"}}`))
			Expect(rr.Code).To(Equal(http.StatusBadRequest))
			Expect(gjson.Get(rr.Body.String(), "error.message").String()).To(ContainSubstring("objects are only allowed as geoCoordinates"))
			gatewayClient.AssertNotCalled(GinkgoT(), "Stat", mock.Anything, mock.Anything)
		})

		It("needs write access", func() {
			info.PermissionSet = readonly
			statReturns(info)
			svc.UpsertOpenExtension(rr, request(http.MethodPut, "com.example.project", `{"status":"x"}`))
			Expect(rr.Code).To(Equal(http.StatusForbidden))
			gatewayClient.AssertNotCalled(GinkgoT(), "SetArbitraryMetadata", mock.Anything, mock.Anything)
		})

		It("reports a locked item", func() {
			statReturns(info)
			gatewayClient.ExpectedCalls = nil
			statReturns(info)
			gatewayClient.On("SetArbitraryMetadata", mock.Anything, mock.Anything).Return(&provider.SetArbitraryMetadataResponse{Status: status.NewLocked(ctx, "locked")}, nil)
			svc.UpsertOpenExtension(rr, request(http.MethodPut, "com.example.project", `{"status":"x"}`))
			Expect(rr.Code).To(Equal(http.StatusLocked))
		})
	})

	Describe("DeleteOpenExtension", func() {
		It("unsets every stored property", func() {
			statReturns(info)
			gatewayClient.On("UnsetArbitraryMetadata", mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
				req := args.Get(1).(*provider.UnsetArbitraryMetadataRequest)
				Expect(req.GetArbitraryMetadataKeys()).To(Equal([]string{key("com.example.project", "due"), key("com.example.project", "priority"), key("com.example.project", "status")}))
			}).Return(&provider.UnsetArbitraryMetadataResponse{Status: status.NewOK(ctx)}, nil)
			svc.DeleteOpenExtension(rr, request(http.MethodDelete, "com.example.project", ""))
			Expect(rr.Code).To(Equal(http.StatusNoContent))
		})

		It("answers 404 for an extension the item does not have", func() {
			statReturns(info)
			svc.DeleteOpenExtension(rr, request(http.MethodDelete, "com.example.other", ""))
			Expect(rr.Code).To(Equal(http.StatusNotFound))
			gatewayClient.AssertNotCalled(GinkgoT(), "UnsetArbitraryMetadata", mock.Anything, mock.Anything)
		})

		It("needs write access", func() {
			info.PermissionSet = readonly
			statReturns(info)
			svc.DeleteOpenExtension(rr, request(http.MethodDelete, "com.example.project", ""))
			Expect(rr.Code).To(Equal(http.StatusForbidden))
		})
	})

	Describe("GetDriveItem", func() {
		It("expands the extensions on request", func() {
			statReturns(info)
			svc.GetDriveItem(rr, request(http.MethodGet, "", "", "$expand=extensions"))
			Expect(rr.Code).To(Equal(http.StatusOK))
			body := rr.Body.String()
			Expect(gjson.Get(body, "id").String()).To(Equal("storageid$spaceid!nodeid"))
			Expect(gjson.Get(body, "extensions.#").Int()).To(Equal(int64(2)))
			Expect(gjson.Get(body, "extensions.1.extensionName").String()).To(Equal("com.example.project"))
			Expect(gjson.Get(body, "extensions.1.due@odata\\.type").String()).To(Equal("#DateTimeOffset"))
		})

		It("leaves the extensions out without $expand", func() {
			statReturns(info)
			svc.GetDriveItem(rr, request(http.MethodGet, "", ""))
			Expect(rr.Code).To(Equal(http.StatusOK))
			Expect(gjson.Get(rr.Body.String(), "extensions").Exists()).To(BeFalse())
		})
	})
})
