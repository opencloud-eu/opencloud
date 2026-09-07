package svc_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"time"

	gateway "github.com/cs3org/go-cs3apis/cs3/gateway/v1beta1"
	userpb "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"
	cs3rpc "github.com/cs3org/go-cs3apis/cs3/rpc/v1beta1"
	provider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	"github.com/go-chi/chi/v5"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	libregraph "github.com/opencloud-eu/libre-graph-api-go"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc"

	revactx "github.com/opencloud-eu/reva/v2/pkg/ctx"
	"github.com/opencloud-eu/reva/v2/pkg/rgrpc/status"
	"github.com/opencloud-eu/reva/v2/pkg/rgrpc/todo/pool"
	"github.com/opencloud-eu/reva/v2/pkg/utils"
	cs3mocks "github.com/opencloud-eu/reva/v2/tests/cs3mocks/mocks"

	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/pkg/shared"
	"github.com/opencloud-eu/opencloud/services/graph/mocks"
	"github.com/opencloud-eu/opencloud/services/graph/pkg/config"
	"github.com/opencloud-eu/opencloud/services/graph/pkg/config/defaults"
	identitymocks "github.com/opencloud-eu/opencloud/services/graph/pkg/identity/mocks"
	"github.com/opencloud-eu/opencloud/services/graph/pkg/metrics"
	graphm "github.com/opencloud-eu/opencloud/services/graph/pkg/middleware"
	service "github.com/opencloud-eu/opencloud/services/graph/pkg/service/v0"
)

var _ = Describe("Drive special folders", func() {
	var (
		svc             service.Service
		ctx             context.Context
		cfg             *config.Config
		gatewayClient   *cs3mocks.GatewayAPIClient
		gatewaySelector pool.Selectable[gateway.GatewayAPIClient]
		eventsPublisher mocks.Publisher
		identityBackend *identitymocks.Backend

		rr *httptest.ResponseRecorder

		currentUser = &userpb.User{
			Id: &userpb.UserId{
				OpaqueId: "user",
			},
		}
		deletedAt = time.Date(2026, 9, 7, 12, 0, 0, 0, time.UTC)
	)

	// newRequest builds a request for /drives/{driveID}/special/{specialName}[/children];
	// specialPath mimics what the colon-path middleware puts into the context.
	newRequest := func(specialName, specialPath string) *http.Request {
		r := httptest.NewRequest(http.MethodGet, "/graph/v1.0/drives/storageid$spaceid/special/"+specialName, nil)
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("driveID", "storageid$spaceid")
		rctx.URLParams.Add("specialName", specialName)
		c := context.WithValue(revactx.ContextSetUser(ctx, currentUser), chi.RouteCtxKey, rctx)
		if specialPath != "" {
			c = graphm.WithSpecialFolderPath(c, specialPath)
		}
		return r.WithContext(c)
	}

	decodeItem := func() libregraph.DriveItem {
		var item libregraph.DriveItem
		Expect(json.Unmarshal(rr.Body.Bytes(), &item)).To(Succeed())
		return item
	}
	decodeList := func() []libregraph.DriveItem {
		var res struct {
			Value []libregraph.DriveItem
		}
		Expect(json.Unmarshal(rr.Body.Bytes(), &res)).To(Succeed())
		return res.Value
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
		identityBackend = &identitymocks.Backend{}
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
			service.WithIdentityBackend(identityBackend),
		)
		Expect(err).ToNot(HaveOccurred())
	})

	Describe("GetDriveSpecial", func() {
		It("rejects an unknown special folder", func() {
			svc.GetDriveSpecial(rr, newRequest("nope", ""))
			Expect(rr.Code).To(Equal(http.StatusBadRequest))
			gatewayClient.AssertNotCalled(GinkgoT(), "ListRecycle", mock.Anything, mock.Anything)
		})

		It("returns the synthetic recycle bin folder without touching the storage", func() {
			svc.GetDriveSpecial(rr, newRequest("recyclebin", ""))
			Expect(rr.Code).To(Equal(http.StatusOK))
			gatewayClient.AssertNotCalled(GinkgoT(), "ListRecycle", mock.Anything, mock.Anything)

			item := decodeItem()
			Expect(item.GetId()).To(Equal("storageid$spaceid!recyclebin"))
			Expect(item.GetName()).To(Equal("recyclebin"))
			Expect(item.Folder).ToNot(BeNil())
			Expect(item.SpecialFolder.GetName()).To(Equal("recyclebin"))
			Expect(item.ParentReference.GetDriveId()).To(Equal("storageid$spaceid"))
			Expect(item.ParentReference.GetId()).To(Equal("storageid$spaceid!spaceid"))
		})

		It("returns a top-level trashed item by key", func() {
			var req *provider.ListRecycleRequest
			gatewayClient.On("ListRecycle", mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
				req = args.Get(1).(*provider.ListRecycleRequest)
			}).Return(&provider.ListRecycleResponse{
				Status: status.NewOK(ctx),
				RecycleItems: []*provider.RecycleItem{{
					Type:         provider.ResourceType_RESOURCE_TYPE_FILE,
					Key:          "nodeid",
					Size:         42,
					DeletionTime: utils.TimeToTS(deletedAt),
					Ref:          &provider.Reference{Path: "/Documents/notes.txt"},
				}},
			}, nil)

			svc.GetDriveSpecial(rr, newRequest("recyclebin", "/nodeid"))
			Expect(rr.Code).To(Equal(http.StatusOK))
			Expect(req.GetKey()).To(Equal("nodeid"))
			Expect(req.GetRef().GetResourceId().GetStorageId()).To(Equal("storageid"))
			Expect(req.GetRef().GetResourceId().GetSpaceId()).To(Equal("spaceid"))
			Expect(req.GetRef().GetResourceId().GetOpaqueId()).To(Equal("spaceid"))

			item := decodeItem()
			Expect(item.GetId()).To(Equal("storageid$spaceid!nodeid"))
			Expect(item.GetName()).To(Equal("notes.txt"))
			Expect(item.GetSize()).To(Equal(int64(42)))
			Expect(item.File.GetMimeType()).To(Equal("text/plain; charset=utf-8"))
			Expect(item.Folder).To(BeNil())
			Expect(item.Trash.GetTrashedDateTime()).To(BeTemporally("==", deletedAt))
			Expect(item.Trash.TrashedBy).To(BeNil())
			Expect(item.ParentReference.GetDriveId()).To(Equal("storageid$spaceid"))
			Expect(item.ParentReference.GetPath()).To(Equal("/Documents"))
			Expect(item.ParentReference.GetName()).To(Equal("Documents"))
		})

		It("returns a nested trashed item through its parent listing", func() {
			var req *provider.ListRecycleRequest
			gatewayClient.On("ListRecycle", mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
				req = args.Get(1).(*provider.ListRecycleRequest)
			}).Return(&provider.ListRecycleResponse{
				Status: status.NewOK(ctx),
				RecycleItems: []*provider.RecycleItem{
					{Type: provider.ResourceType_RESOURCE_TYPE_FILE, Key: "nodeid/sub/other.txt", Ref: &provider.Reference{Path: "/folder/sub/other.txt"}},
					{Type: provider.ResourceType_RESOURCE_TYPE_CONTAINER, Key: "nodeid/sub/inner", Ref: &provider.Reference{Path: "/folder/sub/inner"}},
				},
			}, nil)

			svc.GetDriveSpecial(rr, newRequest("recyclebin", "/nodeid/sub/inner"))
			Expect(rr.Code).To(Equal(http.StatusOK))
			Expect(req.GetKey()).To(Equal("nodeid/sub/"))

			item := decodeItem()
			Expect(item.GetId()).To(Equal("storageid$spaceid!nodeid/sub/inner"))
			Expect(item.GetName()).To(Equal("inner"))
			Expect(item.Folder).ToNot(BeNil())
			Expect(item.File).To(BeNil())
		})

		It("returns not found when the key is not in the listing", func() {
			gatewayClient.On("ListRecycle", mock.Anything, mock.Anything).Return(&provider.ListRecycleResponse{
				Status:       status.NewOK(ctx),
				RecycleItems: []*provider.RecycleItem{{Key: "other"}},
			}, nil)

			svc.GetDriveSpecial(rr, newRequest("recyclebin", "/nodeid"))
			Expect(rr.Code).To(Equal(http.StatusNotFound))
		})
	})

	Describe("ListDriveSpecialChildren", func() {
		It("rejects an unknown special folder", func() {
			svc.ListDriveSpecialChildren(rr, newRequest("nope", ""))
			Expect(rr.Code).To(Equal(http.StatusBadRequest))
		})

		DescribeTable("maps ListRecycle errors",
			func(st *cs3rpc.Status, code int) {
				gatewayClient.On("ListRecycle", mock.Anything, mock.Anything).Return(&provider.ListRecycleResponse{Status: st}, nil)
				svc.ListDriveSpecialChildren(rr, newRequest("recyclebin", ""))
				Expect(rr.Code).To(Equal(code))
			},
			Entry("not found", status.NewNotFound(context.Background(), "not found"), http.StatusNotFound),
			Entry("permission denied as not found", status.NewPermissionDenied(context.Background(), errors.New("denied"), "denied"), http.StatusNotFound),
			Entry("unauthenticated", status.NewUnauthenticated(context.Background(), errors.New("no"), "no"), http.StatusUnauthorized),
			Entry("internal", status.NewInternal(context.Background(), "internal"), http.StatusInternalServerError),
		)

		It("lists the trash root with the trash facet", func() {
			var req *provider.ListRecycleRequest
			gatewayClient.On("ListRecycle", mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
				req = args.Get(1).(*provider.ListRecycleRequest)
			}).Return(&provider.ListRecycleResponse{
				Status: status.NewOK(ctx),
				RecycleItems: []*provider.RecycleItem{
					{
						Type:         provider.ResourceType_RESOURCE_TYPE_FILE,
						Key:          "file-node",
						Size:         7,
						DeletionTime: utils.TimeToTS(deletedAt),
						Ref:          &provider.Reference{Path: "/photo.jpg"},
					},
					{
						Type:         provider.ResourceType_RESOURCE_TYPE_CONTAINER,
						Key:          "folder-node",
						Size:         100,
						DeletionTime: utils.TimeToTS(deletedAt),
						Ref:          &provider.Reference{Path: "/Projects/old"},
					},
				},
			}, nil)

			svc.ListDriveSpecialChildren(rr, newRequest("recyclebin", ""))
			Expect(rr.Code).To(Equal(http.StatusOK))
			Expect(req.GetKey()).To(Equal(""))

			items := decodeList()
			Expect(items).To(HaveLen(2))

			Expect(items[0].GetId()).To(Equal("storageid$spaceid!file-node"))
			Expect(items[0].GetName()).To(Equal("photo.jpg"))
			Expect(items[0].File.GetMimeType()).To(Equal("image/jpeg"))
			Expect(items[0].Trash.GetTrashedDateTime()).To(BeTemporally("==", deletedAt))
			Expect(items[0].ParentReference.GetPath()).To(Equal("/"))
			Expect(items[0].ParentReference.Name).To(BeNil())

			Expect(items[1].GetId()).To(Equal("storageid$spaceid!folder-node"))
			Expect(items[1].GetName()).To(Equal("old"))
			Expect(items[1].Folder).ToNot(BeNil())
			Expect(items[1].GetSize()).To(Equal(int64(100)))
			Expect(items[1].ParentReference.GetPath()).To(Equal("/Projects"))
			Expect(items[1].ParentReference.GetName()).To(Equal("Projects"))
		})

		It("returns an empty list for an empty trash", func() {
			gatewayClient.On("ListRecycle", mock.Anything, mock.Anything).Return(&provider.ListRecycleResponse{
				Status: status.NewOK(ctx),
			}, nil)

			svc.ListDriveSpecialChildren(rr, newRequest("recyclebin", ""))
			Expect(rr.Code).To(Equal(http.StatusOK))
			Expect(rr.Body.String()).To(MatchJSON(`{"value":[]}`))
		})

		It("lists inside a trashed folder with a trailing slash key", func() {
			var req *provider.ListRecycleRequest
			gatewayClient.On("ListRecycle", mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
				req = args.Get(1).(*provider.ListRecycleRequest)
			}).Return(&provider.ListRecycleResponse{
				Status: status.NewOK(ctx),
				RecycleItems: []*provider.RecycleItem{
					{Type: provider.ResourceType_RESOURCE_TYPE_FILE, Key: "folder-node/sub/a.txt", Ref: &provider.Reference{Path: "/Projects/old/sub/a.txt"}},
				},
			}, nil)

			svc.ListDriveSpecialChildren(rr, newRequest("recyclebin", "/folder-node/sub"))
			Expect(rr.Code).To(Equal(http.StatusOK))
			Expect(req.GetKey()).To(Equal("folder-node/sub/"))

			items := decodeList()
			Expect(items).To(HaveLen(1))
			Expect(items[0].GetId()).To(Equal("storageid$spaceid!folder-node/sub/a.txt"))
			Expect(items[0].ParentReference.GetPath()).To(Equal("/Projects/old/sub"))
		})
	})
})
