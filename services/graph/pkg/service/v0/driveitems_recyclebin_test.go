package svc_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
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

var _ = Describe("Drive recycle bin", func() {
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
		deletedAt   = time.Date(2026, 9, 7, 12, 0, 0, 0, time.UTC)
	)

	type params map[string]string

	specialParams := params{"driveID": "storageid$spaceid", "specialName": "recyclebin"}
	itemParams := params{"driveID": "storageid$spaceid", "driveItemID": "storageid$spaceid!nodeid"}

	// newRequest binds the given chi params; query goes into the URL, body is sent as JSON
	newRequest := func(method string, p params, query, body string) *http.Request {
		r := httptest.NewRequest(method, "/graph/v1.0/drives/storageid$spaceid"+query, strings.NewReader(body))
		rctx := chi.NewRouteContext()
		for k, v := range p {
			rctx.URLParams.Add(k, v)
		}
		return r.WithContext(context.WithValue(revactx.ContextSetUser(ctx, currentUser), chi.RouteCtxKey, rctx))
	}
	// withSpecialPath mimics what the colon-path middleware puts into the context
	withSpecialPath := func(r *http.Request, specialPath string) *http.Request {
		return r.WithContext(graphm.WithSpecialFolderPath(r.Context(), specialPath))
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

	trashedFile := &provider.RecycleItem{
		Type:         provider.ResourceType_RESOURCE_TYPE_FILE,
		Key:          "nodeid",
		Size:         42,
		DeletionTime: utils.TimeToTS(deletedAt),
		Ref:          &provider.Reference{Path: "/Documents/notes.txt"},
	}

	// mockListRecycle answers every ListRecycle with the items and records the requested keys
	mockListRecycle := func(items ...*provider.RecycleItem) *[]string {
		var keys []string
		gatewayClient.On("ListRecycle", mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
			keys = append(keys, args.Get(1).(*provider.ListRecycleRequest).GetKey())
		}).Return(&provider.ListRecycleResponse{
			Status:       status.NewOK(ctx),
			RecycleItems: items,
		}, nil)
		return &keys
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

	Describe("GetDriveSpecial", func() {
		It("rejects an unknown special folder", func() {
			svc.GetDriveSpecial(rr, newRequest(http.MethodGet, params{"driveID": "storageid$spaceid", "specialName": "nope"}, "", ""))
			Expect(rr.Code).To(Equal(http.StatusBadRequest))
			gatewayClient.AssertNotCalled(GinkgoT(), "ListRecycle", mock.Anything, mock.Anything)
		})

		It("returns the synthetic recycle bin folder without touching the storage", func() {
			svc.GetDriveSpecial(rr, newRequest(http.MethodGet, specialParams, "", ""))
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
				Status:       status.NewOK(ctx),
				RecycleItems: []*provider.RecycleItem{trashedFile},
			}, nil)

			svc.GetDriveSpecial(rr, withSpecialPath(newRequest(http.MethodGet, specialParams, "", ""), "/nodeid"))
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
			keys := mockListRecycle(
				&provider.RecycleItem{Type: provider.ResourceType_RESOURCE_TYPE_FILE, Key: "nodeid/sub/other.txt", Ref: &provider.Reference{Path: "/folder/sub/other.txt"}},
				&provider.RecycleItem{Type: provider.ResourceType_RESOURCE_TYPE_CONTAINER, Key: "nodeid/sub/inner", Ref: &provider.Reference{Path: "/folder/sub/inner"}},
			)

			svc.GetDriveSpecial(rr, withSpecialPath(newRequest(http.MethodGet, specialParams, "", ""), "/nodeid/sub/inner"))
			Expect(rr.Code).To(Equal(http.StatusOK))
			Expect(*keys).To(Equal([]string{"nodeid/sub/"}))

			item := decodeItem()
			Expect(item.GetId()).To(Equal("storageid$spaceid!nodeid/sub/inner"))
			Expect(item.GetName()).To(Equal("inner"))
			Expect(item.Folder).ToNot(BeNil())
			Expect(item.File).To(BeNil())
		})

		It("embeds the trash listing with $expand=children", func() {
			keys := mockListRecycle(trashedFile)

			svc.GetDriveSpecial(rr, newRequest(http.MethodGet, specialParams, "?$expand=children", ""))
			Expect(rr.Code).To(Equal(http.StatusOK))
			Expect(*keys).To(Equal([]string{""}))

			item := decodeItem()
			Expect(item.GetId()).To(Equal("storageid$spaceid!recyclebin"))
			Expect(item.Children).To(HaveLen(1))
			Expect(item.Children[0].GetId()).To(Equal("storageid$spaceid!nodeid"))
			Expect(item.Children[0].Trash).ToNot(BeNil())
		})

		It("embeds the children of a trashed folder with $expand=children", func() {
			var keys []string
			record := func(args mock.Arguments) {
				keys = append(keys, args.Get(1).(*provider.ListRecycleRequest).GetKey())
			}
			gatewayClient.On("ListRecycle", mock.Anything, mock.Anything).Run(record).Return(&provider.ListRecycleResponse{
				Status:       status.NewOK(ctx),
				RecycleItems: []*provider.RecycleItem{{Type: provider.ResourceType_RESOURCE_TYPE_CONTAINER, Key: "nodeid", Ref: &provider.Reference{Path: "/folder"}}},
			}, nil).Once()
			gatewayClient.On("ListRecycle", mock.Anything, mock.Anything).Run(record).Return(&provider.ListRecycleResponse{
				Status:       status.NewOK(ctx),
				RecycleItems: []*provider.RecycleItem{{Type: provider.ResourceType_RESOURCE_TYPE_FILE, Key: "nodeid/a.txt", Ref: &provider.Reference{Path: "/folder/a.txt"}}},
			}, nil).Once()

			svc.GetDriveSpecial(rr, withSpecialPath(newRequest(http.MethodGet, specialParams, "?$expand=children", ""), "/nodeid"))
			Expect(rr.Code).To(Equal(http.StatusOK))
			Expect(keys).To(Equal([]string{"nodeid", "nodeid/"}))

			item := decodeItem()
			Expect(item.GetId()).To(Equal("storageid$spaceid!nodeid"))
			Expect(item.Children).To(HaveLen(1))
			Expect(item.Children[0].GetId()).To(Equal("storageid$spaceid!nodeid/a.txt"))
		})

		It("does not expand children of a trashed file", func() {
			mockListRecycle(trashedFile)

			svc.GetDriveSpecial(rr, withSpecialPath(newRequest(http.MethodGet, specialParams, "?$expand=children", ""), "/nodeid"))
			Expect(rr.Code).To(Equal(http.StatusOK))
			gatewayClient.AssertNumberOfCalls(GinkgoT(), "ListRecycle", 1)
			Expect(decodeItem().Children).To(BeNil())
		})

		It("returns not found when the key is not in the listing", func() {
			mockListRecycle(&provider.RecycleItem{Key: "other"})

			svc.GetDriveSpecial(rr, withSpecialPath(newRequest(http.MethodGet, specialParams, "", ""), "/nodeid"))
			Expect(rr.Code).To(Equal(http.StatusNotFound))
		})
	})

	Describe("ListDriveSpecialChildren", func() {
		It("rejects an unknown special folder", func() {
			svc.ListDriveSpecialChildren(rr, newRequest(http.MethodGet, params{"driveID": "storageid$spaceid", "specialName": "nope"}, "", ""))
			Expect(rr.Code).To(Equal(http.StatusBadRequest))
		})

		DescribeTable("maps ListRecycle errors",
			func(st *cs3rpc.Status, code int) {
				gatewayClient.On("ListRecycle", mock.Anything, mock.Anything).Return(&provider.ListRecycleResponse{Status: st}, nil)
				svc.ListDriveSpecialChildren(rr, newRequest(http.MethodGet, specialParams, "", ""))
				Expect(rr.Code).To(Equal(code))
			},
			Entry("not found", status.NewNotFound(context.Background(), "not found"), http.StatusNotFound),
			Entry("permission denied as not found", status.NewPermissionDenied(context.Background(), errors.New("denied"), "denied"), http.StatusNotFound),
			Entry("unauthenticated", status.NewUnauthenticated(context.Background(), errors.New("no"), "no"), http.StatusUnauthorized),
			Entry("internal", status.NewInternal(context.Background(), "internal"), http.StatusInternalServerError),
		)

		It("lists the trash root with the trash facet", func() {
			keys := mockListRecycle(
				&provider.RecycleItem{
					Type:         provider.ResourceType_RESOURCE_TYPE_FILE,
					Key:          "file-node",
					Size:         7,
					DeletionTime: utils.TimeToTS(deletedAt),
					Ref:          &provider.Reference{Path: "/photo.jpg"},
				},
				&provider.RecycleItem{
					Type:         provider.ResourceType_RESOURCE_TYPE_CONTAINER,
					Key:          "folder-node",
					Size:         100,
					DeletionTime: utils.TimeToTS(deletedAt),
					Ref:          &provider.Reference{Path: "/Projects/old"},
				},
			)

			svc.ListDriveSpecialChildren(rr, newRequest(http.MethodGet, specialParams, "", ""))
			Expect(rr.Code).To(Equal(http.StatusOK))
			Expect(*keys).To(Equal([]string{""}))

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
			mockListRecycle()

			svc.ListDriveSpecialChildren(rr, newRequest(http.MethodGet, specialParams, "", ""))
			Expect(rr.Code).To(Equal(http.StatusOK))
			Expect(rr.Body.String()).To(MatchJSON(`{"value":[]}`))
		})

		It("lists inside a trashed folder with a trailing slash key", func() {
			keys := mockListRecycle(
				&provider.RecycleItem{Type: provider.ResourceType_RESOURCE_TYPE_FILE, Key: "folder-node/sub/a.txt", Ref: &provider.Reference{Path: "/Projects/old/sub/a.txt"}},
			)

			svc.ListDriveSpecialChildren(rr, withSpecialPath(newRequest(http.MethodGet, specialParams, "", ""), "/folder-node/sub"))
			Expect(rr.Code).To(Equal(http.StatusOK))
			Expect(*keys).To(Equal([]string{"folder-node/sub/"}))

			items := decodeList()
			Expect(items).To(HaveLen(1))
			Expect(items[0].GetId()).To(Equal("storageid$spaceid!folder-node/sub/a.txt"))
			Expect(items[0].ParentReference.GetPath()).To(Equal("/Projects/old/sub"))
		})
	})

	Describe("RestoreDriveItem", func() {
		It("rejects an item from another drive", func() {
			svc.RestoreDriveItem(rr, newRequest(http.MethodPost, params{"driveID": "storageid$other", "driveItemID": "storageid$spaceid!nodeid"}, "", ""))
			Expect(rr.Code).To(Equal(http.StatusNotFound))
			gatewayClient.AssertNotCalled(GinkgoT(), "ListRecycle", mock.Anything, mock.Anything)
		})

		It("rejects a malformed body", func() {
			svc.RestoreDriveItem(rr, newRequest(http.MethodPost, itemParams, "", `{"nope": 1}`))
			Expect(rr.Code).To(Equal(http.StatusBadRequest))
		})

		It("returns not found for an unknown key", func() {
			mockListRecycle()
			svc.RestoreDriveItem(rr, newRequest(http.MethodPost, itemParams, "", ""))
			Expect(rr.Code).To(Equal(http.StatusNotFound))
			gatewayClient.AssertNotCalled(GinkgoT(), "RestoreRecycleItem", mock.Anything, mock.Anything)
		})

		It("restores to the original location by default and returns the item", func() {
			keys := mockListRecycle(trashedFile)
			req := mockRestore(status.NewOK(ctx))
			mockStatOK()

			svc.RestoreDriveItem(rr, newRequest(http.MethodPost, itemParams, "", ""))
			Expect(rr.Code).To(Equal(http.StatusOK))
			Expect(*keys).To(Equal([]string{"nodeid"}))

			Expect((*req).GetKey()).To(Equal("nodeid"))
			Expect((*req).GetRef().GetResourceId().GetOpaqueId()).To(Equal("spaceid"))
			Expect((*req).GetRestoreRef().GetResourceId().GetOpaqueId()).To(Equal("spaceid"))
			Expect((*req).GetRestoreRef().GetPath()).To(Equal("./Documents/notes.txt"))

			item := decodeItem()
			Expect(item.GetId()).To(Equal("storageid$spaceid!nodeid"))
			Expect(item.GetName()).To(Equal("notes.txt"))
		})

		It("finds a nested key through its parent listing", func() {
			keys := mockListRecycle(
				&provider.RecycleItem{Type: provider.ResourceType_RESOURCE_TYPE_FILE, Key: "nodeid/sub/a.txt", Ref: &provider.Reference{Path: "/folder/sub/a.txt"}},
			)
			req := mockRestore(status.NewOK(ctx))
			mockStatOK()

			svc.RestoreDriveItem(rr, newRequest(http.MethodPost, params{"driveID": "storageid$spaceid", "driveItemID": "storageid$spaceid!nodeid/sub/a.txt"}, "", ""))
			Expect(rr.Code).To(Equal(http.StatusOK))
			Expect(*keys).To(Equal([]string{"nodeid/sub/"}))
			Expect((*req).GetKey()).To(Equal("nodeid/sub/a.txt"))
			Expect((*req).GetRestoreRef().GetPath()).To(Equal("./folder/sub/a.txt"))
		})

		It("accepts the v1beta1 itemID param", func() {
			mockListRecycle(trashedFile)
			mockRestore(status.NewOK(ctx))
			mockStatOK()

			svc.RestoreDriveItem(rr, newRequest(http.MethodPost, params{"driveID": "storageid$spaceid", "itemID": "storageid$spaceid!nodeid"}, "", ""))
			Expect(rr.Code).To(Equal(http.StatusOK))
		})

		It("works without a drive id (me/drive)", func() {
			mockListRecycle(trashedFile)
			mockRestore(status.NewOK(ctx))
			mockStatOK()

			svc.RestoreDriveItem(rr, newRequest(http.MethodPost, params{"itemID": "storageid$spaceid!nodeid"}, "", ""))
			Expect(rr.Code).To(Equal(http.StatusOK))
		})

		It("restores into the given parent with a new name", func() {
			mockListRecycle(trashedFile)
			req := mockRestore(status.NewOK(ctx))
			mockStatOK()

			body := `{"parentReference": {"id": "storageid$spaceid!parentid"}, "name": "renamed.txt"}`
			svc.RestoreDriveItem(rr, newRequest(http.MethodPost, itemParams, "", body))
			Expect(rr.Code).To(Equal(http.StatusOK))

			Expect((*req).GetRestoreRef().GetResourceId().GetOpaqueId()).To(Equal("parentid"))
			Expect((*req).GetRestoreRef().GetPath()).To(Equal("./renamed.txt"))
		})

		It("restores into a parent given by path", func() {
			mockListRecycle(trashedFile)
			req := mockRestore(status.NewOK(ctx))
			mockStatOK()

			svc.RestoreDriveItem(rr, newRequest(http.MethodPost, itemParams, "", `{"parentReference": {"path": "/Archive/2025"}}`))
			Expect(rr.Code).To(Equal(http.StatusOK))

			Expect((*req).GetRestoreRef().GetResourceId().GetOpaqueId()).To(Equal("spaceid"))
			Expect((*req).GetRestoreRef().GetPath()).To(Equal("./Archive/2025/notes.txt"))
		})

		It("only renames when just a name is given", func() {
			mockListRecycle(trashedFile)
			req := mockRestore(status.NewOK(ctx))
			mockStatOK()

			svc.RestoreDriveItem(rr, newRequest(http.MethodPost, itemParams, "", `{"name": "renamed.txt"}`))
			Expect(rr.Code).To(Equal(http.StatusOK))
			Expect((*req).GetRestoreRef().GetPath()).To(Equal("./Documents/renamed.txt"))
		})

		It("rejects a name with a slash", func() {
			mockListRecycle(trashedFile)
			svc.RestoreDriveItem(rr, newRequest(http.MethodPost, itemParams, "", `{"name": "a/b.txt"}`))
			Expect(rr.Code).To(Equal(http.StatusBadRequest))
			gatewayClient.AssertNotCalled(GinkgoT(), "RestoreRecycleItem", mock.Anything, mock.Anything)
		})

		It("rejects a parent in another drive", func() {
			mockListRecycle(trashedFile)
			body := `{"parentReference": {"driveId": "storageid$other", "id": "storageid$other!parentid"}}`
			svc.RestoreDriveItem(rr, newRequest(http.MethodPost, itemParams, "", body))
			Expect(rr.Code).To(Equal(http.StatusBadRequest))
			gatewayClient.AssertNotCalled(GinkgoT(), "RestoreRecycleItem", mock.Anything, mock.Anything)
		})

		DescribeTable("maps restore errors",
			func(st *cs3rpc.Status, code int) {
				mockListRecycle(trashedFile)
				mockRestore(st)
				svc.RestoreDriveItem(rr, newRequest(http.MethodPost, itemParams, "", ""))
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
		It("refuses the drive root", func() {
			svc.PermanentDeleteDriveItem(rr, newRequest(http.MethodPost, params{"driveID": "storageid$spaceid", "driveItemID": "storageid$spaceid!spaceid"}, "", ""))
			Expect(rr.Code).To(Equal(http.StatusBadRequest))
			gatewayClient.AssertNotCalled(GinkgoT(), "Delete", mock.Anything, mock.Anything)
		})

		It("refuses share jail items", func() {
			p := params{"driveID": "a0ca6a90-a365-4782-871e-d44447bbc668$a0ca6a90-a365-4782-871e-d44447bbc668", "driveItemID": "a0ca6a90-a365-4782-871e-d44447bbc668$a0ca6a90-a365-4782-871e-d44447bbc668!shareid"}
			svc.PermanentDeleteDriveItem(rr, newRequest(http.MethodPost, p, "", ""))
			Expect(rr.Code).To(Equal(http.StatusBadRequest))
			gatewayClient.AssertNotCalled(GinkgoT(), "Delete", mock.Anything, mock.Anything)
		})

		It("deletes and purges the trash key", func() {
			var delReq *provider.DeleteRequest
			gatewayClient.On("Delete", mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
				delReq = args.Get(1).(*provider.DeleteRequest)
			}).Return(&provider.DeleteResponse{Status: status.NewOK(ctx)}, nil)
			purge := mockPurge(status.NewOK(ctx))

			svc.PermanentDeleteDriveItem(rr, newRequest(http.MethodPost, itemParams, "", ""))
			Expect(rr.Code).To(Equal(http.StatusNoContent))

			Expect(delReq.GetRef().GetResourceId().GetOpaqueId()).To(Equal("nodeid"))
			Expect((*purge).GetRef().GetResourceId().GetOpaqueId()).To(Equal("spaceid"))
			Expect((*purge).GetKey()).To(Equal("nodeid"))
		})

		It("maps a failed delete and does not purge", func() {
			gatewayClient.On("Delete", mock.Anything, mock.Anything).Return(&provider.DeleteResponse{Status: status.NewNotFound(ctx, "gone")}, nil)
			svc.PermanentDeleteDriveItem(rr, newRequest(http.MethodPost, itemParams, "", ""))
			Expect(rr.Code).To(Equal(http.StatusNotFound))
			gatewayClient.AssertNotCalled(GinkgoT(), "PurgeRecycle", mock.Anything, mock.Anything)
		})

		It("reports a failed purge after a successful delete", func() {
			gatewayClient.On("Delete", mock.Anything, mock.Anything).Return(&provider.DeleteResponse{Status: status.NewOK(ctx)}, nil)
			mockPurge(status.NewInternal(ctx, "boom"))
			svc.PermanentDeleteDriveItem(rr, newRequest(http.MethodPost, itemParams, "", ""))
			Expect(rr.Code).To(Equal(http.StatusInternalServerError))
		})
	})

	Describe("DeleteDriveSpecialItem", func() {
		It("rejects an unknown special folder", func() {
			svc.DeleteDriveSpecialItem(rr, newRequest(http.MethodDelete, params{"driveID": "storageid$spaceid", "specialName": "nope", "itemID": "storageid$spaceid!nodeid"}, "", ""))
			Expect(rr.Code).To(Equal(http.StatusBadRequest))
		})

		It("rejects an item from another drive", func() {
			svc.DeleteDriveSpecialItem(rr, newRequest(http.MethodDelete, params{"driveID": "storageid$spaceid", "specialName": "recyclebin", "itemID": "storageid$other!nodeid"}, "", ""))
			Expect(rr.Code).To(Equal(http.StatusNotFound))
			gatewayClient.AssertNotCalled(GinkgoT(), "PurgeRecycle", mock.Anything, mock.Anything)
		})

		It("purges the key", func() {
			purge := mockPurge(status.NewOK(ctx))
			svc.DeleteDriveSpecialItem(rr, newRequest(http.MethodDelete, params{"driveID": "storageid$spaceid", "specialName": "recyclebin", "itemID": "storageid$spaceid!nodeid/sub/file"}, "", ""))
			Expect(rr.Code).To(Equal(http.StatusNoContent))
			Expect((*purge).GetKey()).To(Equal("nodeid/sub/file"))
			Expect((*purge).GetRef().GetResourceId().GetOpaqueId()).To(Equal("spaceid"))
		})

		It("maps permission denied to forbidden", func() {
			mockPurge(status.NewPermissionDenied(ctx, errors.New("denied"), "denied"))
			svc.DeleteDriveSpecialItem(rr, newRequest(http.MethodDelete, params{"driveID": "storageid$spaceid", "specialName": "recyclebin", "itemID": "storageid$spaceid!nodeid"}, "", ""))
			Expect(rr.Code).To(Equal(http.StatusForbidden))
		})
	})

	Describe("EmptyDriveSpecial", func() {
		It("purges the whole trash", func() {
			purge := mockPurge(status.NewOK(ctx))
			svc.EmptyDriveSpecial(rr, newRequest(http.MethodDelete, specialParams, "", ""))
			Expect(rr.Code).To(Equal(http.StatusNoContent))
			Expect((*purge).GetKey()).To(Equal(""))
		})

		It("purges only the addressed key in the colon path form", func() {
			purge := mockPurge(status.NewOK(ctx))
			svc.EmptyDriveSpecial(rr, withSpecialPath(newRequest(http.MethodDelete, specialParams, "", ""), "/nodeid"))
			Expect(rr.Code).To(Equal(http.StatusNoContent))
			Expect((*purge).GetKey()).To(Equal("nodeid"))
		})
	})
})
