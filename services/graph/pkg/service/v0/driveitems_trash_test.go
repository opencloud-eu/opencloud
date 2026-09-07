package svc_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"

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

var _ = Describe("Drive item trash operations", func() {
	var (
		svc             service.Service
		ctx             context.Context
		cfg             *config.Config
		gatewayClient   *cs3mocks.GatewayAPIClient
		gatewaySelector pool.Selectable[gateway.GatewayAPIClient]
		eventsPublisher mocks.Publisher
		identityBackend *identitymocks.Backend

		rr *httptest.ResponseRecorder

		currentUser = &userpb.User{Id: &userpb.UserId{OpaqueId: "user"}}
	)

	type params map[string]string

	// newRequest binds the given chi params; body is sent as the JSON request body
	newRequest := func(method, url string, p params, body string) *http.Request {
		r := httptest.NewRequest(method, url, strings.NewReader(body))
		rctx := chi.NewRouteContext()
		for k, v := range p {
			rctx.URLParams.Add(k, v)
		}
		return r.WithContext(context.WithValue(revactx.ContextSetUser(ctx, currentUser), chi.RouteCtxKey, rctx))
	}

	trashedFile := &provider.RecycleItem{
		Type: provider.ResourceType_RESOURCE_TYPE_FILE,
		Key:  "nodeid",
		Ref:  &provider.Reference{Path: "/Documents/notes.txt"},
	}

	mockListRecycle := func(items ...*provider.RecycleItem) {
		gatewayClient.On("ListRecycle", mock.Anything, mock.Anything).Return(&provider.ListRecycleResponse{
			Status:       status.NewOK(ctx),
			RecycleItems: items,
		}, nil)
	}
	mockRestore := func(st *cs3rpc.Status) **provider.RestoreRecycleItemRequest {
		var req *provider.RestoreRecycleItemRequest
		gatewayClient.On("RestoreRecycleItem", mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
			req = args.Get(1).(*provider.RestoreRecycleItemRequest)
		}).Return(&provider.RestoreRecycleItemResponse{Status: st}, nil)
		return &req
	}
	mockStatOK := func() {
		gatewayClient.On("Stat", mock.Anything, mock.Anything).Return(&provider.StatResponse{
			Status: status.NewOK(ctx),
			Info: &provider.ResourceInfo{
				Type: provider.ResourceType_RESOURCE_TYPE_FILE,
				Id:   &provider.ResourceId{StorageId: "storageid", SpaceId: "spaceid", OpaqueId: "nodeid"},
				Path: "./Documents/notes.txt",
			},
		}, nil)
	}
	mockPurge := func(st *cs3rpc.Status) **provider.PurgeRecycleRequest {
		var req *provider.PurgeRecycleRequest
		gatewayClient.On("PurgeRecycle", mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
			req = args.Get(1).(*provider.PurgeRecycleRequest)
		}).Return(&provider.PurgeRecycleResponse{Status: st}, nil)
		return &req
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

	Describe("RestoreDriveItem", func() {
		v1Params := params{"driveID": "storageid$spaceid", "driveItemID": "storageid$spaceid!nodeid"}

		It("rejects an item from another drive", func() {
			svc.RestoreDriveItem(rr, newRequest(http.MethodPost, "/", params{"driveID": "storageid$other", "driveItemID": "storageid$spaceid!nodeid"}, ""))
			Expect(rr.Code).To(Equal(http.StatusNotFound))
			gatewayClient.AssertNotCalled(GinkgoT(), "ListRecycle", mock.Anything, mock.Anything)
		})

		It("rejects a malformed body", func() {
			svc.RestoreDriveItem(rr, newRequest(http.MethodPost, "/", v1Params, `{"nope": 1}`))
			Expect(rr.Code).To(Equal(http.StatusBadRequest))
		})

		It("returns not found for an unknown key", func() {
			mockListRecycle()
			svc.RestoreDriveItem(rr, newRequest(http.MethodPost, "/", v1Params, ""))
			Expect(rr.Code).To(Equal(http.StatusNotFound))
			gatewayClient.AssertNotCalled(GinkgoT(), "RestoreRecycleItem", mock.Anything, mock.Anything)
		})

		It("restores to the original location by default and returns the item", func() {
			mockListRecycle(trashedFile)
			req := mockRestore(status.NewOK(ctx))
			mockStatOK()

			svc.RestoreDriveItem(rr, newRequest(http.MethodPost, "/", v1Params, ""))
			Expect(rr.Code).To(Equal(http.StatusOK))

			Expect((*req).GetKey()).To(Equal("nodeid"))
			Expect((*req).GetRef().GetResourceId().GetOpaqueId()).To(Equal("spaceid"))
			Expect((*req).GetRestoreRef().GetResourceId().GetOpaqueId()).To(Equal("spaceid"))
			Expect((*req).GetRestoreRef().GetPath()).To(Equal("./Documents/notes.txt"))

			var item libregraph.DriveItem
			Expect(json.Unmarshal(rr.Body.Bytes(), &item)).To(Succeed())
			Expect(item.GetId()).To(Equal("storageid$spaceid!nodeid"))
			Expect(item.GetName()).To(Equal("notes.txt"))
		})

		It("accepts the v1beta1 itemID param", func() {
			mockListRecycle(trashedFile)
			mockRestore(status.NewOK(ctx))
			mockStatOK()

			svc.RestoreDriveItem(rr, newRequest(http.MethodPost, "/", params{"driveID": "storageid$spaceid", "itemID": "storageid$spaceid!nodeid"}, ""))
			Expect(rr.Code).To(Equal(http.StatusOK))
		})

		It("works without a drive id (me/drive)", func() {
			mockListRecycle(trashedFile)
			mockRestore(status.NewOK(ctx))
			mockStatOK()

			svc.RestoreDriveItem(rr, newRequest(http.MethodPost, "/", params{"itemID": "storageid$spaceid!nodeid"}, ""))
			Expect(rr.Code).To(Equal(http.StatusOK))
		})

		It("restores into the given parent with a new name", func() {
			mockListRecycle(trashedFile)
			req := mockRestore(status.NewOK(ctx))
			mockStatOK()

			body := `{"parentReference": {"id": "storageid$spaceid!parentid"}, "name": "renamed.txt"}`
			svc.RestoreDriveItem(rr, newRequest(http.MethodPost, "/", v1Params, body))
			Expect(rr.Code).To(Equal(http.StatusOK))

			Expect((*req).GetRestoreRef().GetResourceId().GetOpaqueId()).To(Equal("parentid"))
			Expect((*req).GetRestoreRef().GetPath()).To(Equal("./renamed.txt"))
		})

		It("restores into a parent given by path", func() {
			mockListRecycle(trashedFile)
			req := mockRestore(status.NewOK(ctx))
			mockStatOK()

			body := `{"parentReference": {"path": "/Archive/2025"}}`
			svc.RestoreDriveItem(rr, newRequest(http.MethodPost, "/", v1Params, body))
			Expect(rr.Code).To(Equal(http.StatusOK))

			Expect((*req).GetRestoreRef().GetResourceId().GetOpaqueId()).To(Equal("spaceid"))
			Expect((*req).GetRestoreRef().GetPath()).To(Equal("./Archive/2025/notes.txt"))
		})

		It("only renames when just a name is given", func() {
			mockListRecycle(trashedFile)
			req := mockRestore(status.NewOK(ctx))
			mockStatOK()

			svc.RestoreDriveItem(rr, newRequest(http.MethodPost, "/", v1Params, `{"name": "renamed.txt"}`))
			Expect(rr.Code).To(Equal(http.StatusOK))
			Expect((*req).GetRestoreRef().GetPath()).To(Equal("./Documents/renamed.txt"))
		})

		It("rejects a name with a slash", func() {
			mockListRecycle(trashedFile)
			svc.RestoreDriveItem(rr, newRequest(http.MethodPost, "/", v1Params, `{"name": "a/b.txt"}`))
			Expect(rr.Code).To(Equal(http.StatusBadRequest))
			gatewayClient.AssertNotCalled(GinkgoT(), "RestoreRecycleItem", mock.Anything, mock.Anything)
		})

		It("rejects a parent in another drive", func() {
			mockListRecycle(trashedFile)
			body := `{"parentReference": {"driveId": "storageid$other", "id": "storageid$other!parentid"}}`
			svc.RestoreDriveItem(rr, newRequest(http.MethodPost, "/", v1Params, body))
			Expect(rr.Code).To(Equal(http.StatusBadRequest))
			gatewayClient.AssertNotCalled(GinkgoT(), "RestoreRecycleItem", mock.Anything, mock.Anything)
		})

		DescribeTable("maps restore errors",
			func(st *cs3rpc.Status, code int) {
				mockListRecycle(trashedFile)
				mockRestore(st)
				svc.RestoreDriveItem(rr, newRequest(http.MethodPost, "/", v1Params, ""))
				Expect(rr.Code).To(Equal(code))
				gatewayClient.AssertNotCalled(GinkgoT(), "Stat", mock.Anything, mock.Anything)
			},
			Entry("already exists", status.NewAlreadyExists(context.Background(), errors.New("exists"), "exists"), http.StatusConflict),
			Entry("permission denied", status.NewPermissionDenied(context.Background(), errors.New("denied"), "denied"), http.StatusForbidden),
			Entry("not found", status.NewNotFound(context.Background(), "gone"), http.StatusNotFound),
			Entry("internal", status.NewInternal(context.Background(), "internal"), http.StatusInternalServerError),
		)
	})

	Describe("PermanentDeleteDriveItem", func() {
		v1Params := params{"driveID": "storageid$spaceid", "driveItemID": "storageid$spaceid!nodeid"}

		It("refuses the drive root", func() {
			svc.PermanentDeleteDriveItem(rr, newRequest(http.MethodPost, "/", params{"driveID": "storageid$spaceid", "driveItemID": "storageid$spaceid!spaceid"}, ""))
			Expect(rr.Code).To(Equal(http.StatusBadRequest))
			gatewayClient.AssertNotCalled(GinkgoT(), "Delete", mock.Anything, mock.Anything)
		})

		It("refuses share jail items", func() {
			p := params{"driveID": "a0ca6a90-a365-4782-871e-d44447bbc668$a0ca6a90-a365-4782-871e-d44447bbc668", "driveItemID": "a0ca6a90-a365-4782-871e-d44447bbc668$a0ca6a90-a365-4782-871e-d44447bbc668!shareid"}
			svc.PermanentDeleteDriveItem(rr, newRequest(http.MethodPost, "/", p, ""))
			Expect(rr.Code).To(Equal(http.StatusBadRequest))
			gatewayClient.AssertNotCalled(GinkgoT(), "Delete", mock.Anything, mock.Anything)
		})

		It("deletes and purges the trash key", func() {
			var delReq *provider.DeleteRequest
			gatewayClient.On("Delete", mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
				delReq = args.Get(1).(*provider.DeleteRequest)
			}).Return(&provider.DeleteResponse{Status: status.NewOK(ctx)}, nil)
			purge := mockPurge(status.NewOK(ctx))

			svc.PermanentDeleteDriveItem(rr, newRequest(http.MethodPost, "/", v1Params, ""))
			Expect(rr.Code).To(Equal(http.StatusNoContent))

			Expect(delReq.GetRef().GetResourceId().GetOpaqueId()).To(Equal("nodeid"))
			Expect((*purge).GetRef().GetResourceId().GetOpaqueId()).To(Equal("spaceid"))
			Expect((*purge).GetKey()).To(Equal("nodeid"))
		})

		It("maps a failed delete and does not purge", func() {
			gatewayClient.On("Delete", mock.Anything, mock.Anything).Return(&provider.DeleteResponse{Status: status.NewNotFound(ctx, "gone")}, nil)
			svc.PermanentDeleteDriveItem(rr, newRequest(http.MethodPost, "/", v1Params, ""))
			Expect(rr.Code).To(Equal(http.StatusNotFound))
			gatewayClient.AssertNotCalled(GinkgoT(), "PurgeRecycle", mock.Anything, mock.Anything)
		})

		It("reports a failed purge after a successful delete", func() {
			gatewayClient.On("Delete", mock.Anything, mock.Anything).Return(&provider.DeleteResponse{Status: status.NewOK(ctx)}, nil)
			mockPurge(status.NewInternal(ctx, "boom"))
			svc.PermanentDeleteDriveItem(rr, newRequest(http.MethodPost, "/", v1Params, ""))
			Expect(rr.Code).To(Equal(http.StatusInternalServerError))
		})
	})

	Describe("DeleteDriveSpecialItem", func() {
		It("rejects an unknown special folder", func() {
			svc.DeleteDriveSpecialItem(rr, newRequest(http.MethodDelete, "/", params{"driveID": "storageid$spaceid", "specialName": "nope", "itemID": "storageid$spaceid!nodeid"}, ""))
			Expect(rr.Code).To(Equal(http.StatusBadRequest))
		})

		It("rejects an item from another drive", func() {
			svc.DeleteDriveSpecialItem(rr, newRequest(http.MethodDelete, "/", params{"driveID": "storageid$spaceid", "specialName": "recyclebin", "itemID": "storageid$other!nodeid"}, ""))
			Expect(rr.Code).To(Equal(http.StatusNotFound))
			gatewayClient.AssertNotCalled(GinkgoT(), "PurgeRecycle", mock.Anything, mock.Anything)
		})

		It("purges the key", func() {
			purge := mockPurge(status.NewOK(ctx))
			svc.DeleteDriveSpecialItem(rr, newRequest(http.MethodDelete, "/", params{"driveID": "storageid$spaceid", "specialName": "recyclebin", "itemID": "storageid$spaceid!nodeid/sub/file"}, ""))
			Expect(rr.Code).To(Equal(http.StatusNoContent))
			Expect((*purge).GetKey()).To(Equal("nodeid/sub/file"))
			Expect((*purge).GetRef().GetResourceId().GetOpaqueId()).To(Equal("spaceid"))
		})

		It("maps permission denied to forbidden", func() {
			mockPurge(status.NewPermissionDenied(ctx, errors.New("denied"), "denied"))
			svc.DeleteDriveSpecialItem(rr, newRequest(http.MethodDelete, "/", params{"driveID": "storageid$spaceid", "specialName": "recyclebin", "itemID": "storageid$spaceid!nodeid"}, ""))
			Expect(rr.Code).To(Equal(http.StatusForbidden))
		})
	})

	Describe("EmptyDriveSpecial", func() {
		It("purges the whole trash", func() {
			purge := mockPurge(status.NewOK(ctx))
			svc.EmptyDriveSpecial(rr, newRequest(http.MethodDelete, "/", params{"driveID": "storageid$spaceid", "specialName": "recyclebin"}, ""))
			Expect(rr.Code).To(Equal(http.StatusNoContent))
			Expect((*purge).GetKey()).To(Equal(""))
		})

		It("purges only the addressed key in the colon path form", func() {
			purge := mockPurge(status.NewOK(ctx))
			r := newRequest(http.MethodDelete, "/", params{"driveID": "storageid$spaceid", "specialName": "recyclebin"}, "")
			r = r.WithContext(graphm.WithSpecialFolderPath(r.Context(), "/nodeid"))
			svc.EmptyDriveSpecial(rr, r)
			Expect(rr.Code).To(Equal(http.StatusNoContent))
			Expect((*purge).GetKey()).To(Equal("nodeid"))
		})
	})
})
