package svc_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"time"

	gateway "github.com/cs3org/go-cs3apis/cs3/gateway/v1beta1"
	userprovider "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"
	provider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	typesv1beta1 "github.com/cs3org/go-cs3apis/cs3/types/v1beta1"
	"github.com/go-chi/chi/v5"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	libregraph "github.com/opencloud-eu/libre-graph-api-go"
	"github.com/opencloud-eu/reva/v2/pkg/conversions"
	revactx "github.com/opencloud-eu/reva/v2/pkg/ctx"
	"github.com/opencloud-eu/reva/v2/pkg/rgrpc/status"
	"github.com/opencloud-eu/reva/v2/pkg/rgrpc/todo/pool"
	"github.com/opencloud-eu/reva/v2/pkg/utils"
	cs3mocks "github.com/opencloud-eu/reva/v2/tests/cs3mocks/mocks"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/mock"
	"github.com/tidwall/gjson"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"google.golang.org/grpc"

	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/pkg/shared"
	v0 "github.com/opencloud-eu/opencloud/protogen/gen/opencloud/messages/settings/v0"
	settingssvc "github.com/opencloud-eu/opencloud/protogen/gen/opencloud/services/settings/v0"
	"github.com/opencloud-eu/opencloud/services/graph/mocks"
	"github.com/opencloud-eu/opencloud/services/graph/pkg/config"
	"github.com/opencloud-eu/opencloud/services/graph/pkg/config/defaults"
	"github.com/opencloud-eu/opencloud/services/graph/pkg/errorcode"
	"github.com/opencloud-eu/opencloud/services/graph/pkg/identity"
	identitymocks "github.com/opencloud-eu/opencloud/services/graph/pkg/identity/mocks"
	"github.com/opencloud-eu/opencloud/services/graph/pkg/metrics"
	service "github.com/opencloud-eu/opencloud/services/graph/pkg/service/v0"
	"github.com/opencloud-eu/opencloud/services/graph/pkg/unifiedrole"
)

var _ = Describe("Graph", func() {
	var (
		svc               service.Service
		gatewayClient     *cs3mocks.GatewayAPIClient
		gatewaySelector   pool.Selectable[gateway.GatewayAPIClient]
		eventsPublisher   mocks.Publisher
		permissionService mocks.Permissions
		identityBackend   *identitymocks.Backend
		ctx               context.Context
		cfg               *config.Config
		rr                *httptest.ResponseRecorder

		currentUser = &userprovider.User{
			Id: &userprovider.UserId{
				OpaqueId: "user",
			},
		}
	)

	BeforeEach(func() {
		rr = httptest.NewRecorder()

		logger := log.NewLogger()
		metrics, _ := metrics.New(prometheus.NewRegistry(), &logger, func([]string) (string, string) { return "", "" })

		ctx = revactx.ContextSetUser(context.Background(), &userprovider.User{Id: &userprovider.UserId{Type: userprovider.UserType_USER_TYPE_PRIMARY, OpaqueId: "testuser"}, Username: "testuser"})
		cfg = defaults.FullDefaultConfig()
		cfg.Identity.LDAP.CACert = "" // skip the startup checks, we don't use LDAP at all in this tests
		cfg.TokenManager.JWTSecret = "loremipsum"
		cfg.Commons = &shared.Commons{}
		cfg.GRPCClientTLS = &shared.GRPCClientTLS{}

		pool.RemoveSelector("GatewaySelector" + "eu.opencloud.api.gateway")
		gatewayClient = &cs3mocks.GatewayAPIClient{}
		gatewaySelector = pool.GetSelector[gateway.GatewayAPIClient](
			"GatewaySelector",
			"eu.opencloud.api.gateway",
			func(cc grpc.ClientConnInterface) gateway.GatewayAPIClient {
				return gatewayClient
			},
		)

		eventsPublisher = mocks.Publisher{}
		permissionService = mocks.Permissions{}
		identityBackend = &identitymocks.Backend{}

		var err error
		svc, err = service.NewService(
			service.Config(cfg),
			service.Metrics(metrics),
			service.WithGatewaySelector(gatewaySelector),
			service.EventsPublisher(&eventsPublisher),
			service.PermissionService(&permissionService),
			service.WithIdentityBackend(identityBackend),
		)
		Expect(err).ToNot(HaveOccurred())
	})

	Describe("NewService", func() {
		It("returns a service", func() {
			Expect(svc).ToNot(BeNil())
		})
	})

	Describe("permissions tracing", func() {
		It("records a span for the permissions service when a TraceProvider is configured", func() {
			sr := tracetest.NewSpanRecorder()
			tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr))
			DeferCleanup(func() {
				Expect(tp.Shutdown(context.Background())).To(Succeed())
			})

			logger := log.NewLogger()
			m, err := metrics.New(prometheus.NewRegistry(), &logger, func([]string) (string, string) { return "", "" })
			Expect(err).ToNot(HaveOccurred())
			tracedSvc, err := service.NewService(
				service.Config(cfg),
				service.Metrics(m),
				service.WithGatewaySelector(gatewaySelector),
				service.EventsPublisher(&eventsPublisher),
				service.PermissionService(&permissionService),
				service.TraceProvider(tp),
			)
			Expect(err).ToNot(HaveOccurred())

			gatewayClient.On("ListStorageSpaces", mock.Anything, mock.Anything).Return(&provider.ListStorageSpacesResponse{
				Status: status.NewOK(ctx),
			}, nil)

			r := httptest.NewRequest(http.MethodGet, "/graph/v1beta1/drives/1$2/root/permissions", nil).WithContext(ctx)
			tracedSvc.ServeHTTP(rr, r)

			// the request was routed to the permissions service, which looked up the space
			Expect(rr.Code).To(Equal(http.StatusNotFound))
			gatewayClient.AssertCalled(GinkgoT(), "ListStorageSpaces", mock.Anything, mock.Anything)
			// http tracing works with this provider
			Expect(sr.Ended()).ToNot(BeEmpty())

			var names []string
			for _, s := range sr.Ended() {
				names = append(names, s.Name())
			}
			Expect(names).To(ContainElement("ListSpaceRootPermissions"))
		})
	})

	Describe("Drives", func() {
		Describe("GetDrivesV1 and GetAllDrivesV1", func() {
			BeforeEach(func() {
				permissionService.On("GetPermissionByID", mock.Anything, mock.Anything).Return(&settingssvc.GetPermissionByIDResponse{
					Permission: &v0.Permission{
						// the permission is not relevant for this test
					},
				}, nil)
			})
			It("can list an empty list of spaces", func() {
				gatewayClient.On("ListStorageSpaces", mock.Anything, mock.Anything).Return(&provider.ListStorageSpacesResponse{
					Status:        status.NewOK(ctx),
					StorageSpaces: []*provider.StorageSpace{},
				}, nil)

				r := httptest.NewRequest(http.MethodGet, "/graph/v1.0/me/drives", nil)
				r = r.WithContext(ctx)
				rr := httptest.NewRecorder()
				svc.GetDrivesV1(rr, r)
				Expect(rr.Code).To(Equal(http.StatusOK))
			})

			It("can list an empty list of all spaces", func() {
				gatewayClient.On("ListStorageSpaces", mock.Anything, mock.Anything).Times(1).Return(&provider.ListStorageSpacesResponse{
					Status:        status.NewOK(ctx),
					StorageSpaces: []*provider.StorageSpace{},
				}, nil)

				r := httptest.NewRequest(http.MethodGet, "/graph/v1.0/drives", nil)
				r = r.WithContext(ctx)
				rr := httptest.NewRecorder()
				svc.GetAllDrivesV1(rr, r)
				Expect(rr.Code).To(Equal(http.StatusOK))
			})

			It("can list a space without owner", func() {
				gatewayClient.On("ListStorageSpaces", mock.Anything, mock.Anything).Times(1).Return(&provider.ListStorageSpacesResponse{
					Status: status.NewOK(ctx),
					StorageSpaces: []*provider.StorageSpace{
						{
							Id:        &provider.StorageSpaceId{OpaqueId: "sameID"},
							SpaceType: "aspacetype",
							Root: &provider.ResourceId{
								StorageId: "pro-1",
								SpaceId:   "sameID",
								OpaqueId:  "sameID",
							},
							Name: "aspacename",
						},
					},
				}, nil)
				gatewayClient.On("InitiateFileDownload", mock.Anything, mock.Anything).Return(&gateway.InitiateFileDownloadResponse{
					Status: status.NewNotFound(ctx, "not found"),
				}, nil)
				gatewayClient.On("GetQuota", mock.Anything, mock.Anything).Return(&provider.GetQuotaResponse{
					Status: status.NewUnimplemented(ctx, fmt.Errorf("not supported"), "not supported"),
				}, nil)

				r := httptest.NewRequest(http.MethodGet, "/graph/v1.0/me/drives", nil)
				r = r.WithContext(ctx)
				rr := httptest.NewRecorder()
				svc.GetDrivesV1(rr, r)

				Expect(rr.Code).To(Equal(http.StatusOK))

				body, _ := io.ReadAll(rr.Body)
				Expect(body).To(MatchJSON(`
			{
				"value":[
					{
						"driveType":"aspacetype",
						"id":"pro-1$sameID",
						"name":"aspacename",
						"quota": {},
						"root":{
							"id":"pro-1$sameID",
							"webDavUrl":"https://localhost:9200/dav/spaces/pro-1$sameID"
						},
						"webUrl": "https://localhost:9200/f/pro-1$sameID"
					}
				]
			}
			`))
			})
			It("can list a spaces with sort", func() {
				gatewayClient.On("ListStorageSpaces", mock.Anything, mock.Anything).Return(&provider.ListStorageSpacesResponse{
					Status: status.NewOK(ctx),
					StorageSpaces: []*provider.StorageSpace{
						{
							Id:        &provider.StorageSpaceId{OpaqueId: "bsameID"},
							SpaceType: "bspacetype",
							Root: &provider.ResourceId{
								StorageId: "pro-1",
								SpaceId:   "bsameID",
								OpaqueId:  "bsameID",
							},
							Name: "bspacename",
							Opaque: &typesv1beta1.Opaque{
								Map: map[string]*typesv1beta1.OpaqueEntry{
									"spaceAlias": {Decoder: "plain", Value: []byte("bspacetype/bspacename")},
									"etag":       {Decoder: "plain", Value: []byte("123456789")},
								},
							},
						},
						{
							Id:        &provider.StorageSpaceId{OpaqueId: "asameID"},
							SpaceType: "aspacetype",
							Root: &provider.ResourceId{
								StorageId: "pro-1",
								SpaceId:   "asameID",
								OpaqueId:  "asameID",
							},
							Name: "aspacename",
							Opaque: &typesv1beta1.Opaque{
								Map: map[string]*typesv1beta1.OpaqueEntry{
									"spaceAlias": {Decoder: "plain", Value: []byte("aspacetype/aspacename")},
									"etag":       {Decoder: "plain", Value: []byte("101112131415")},
								},
							},
						},
					},
				}, nil)
				gatewayClient.On("InitiateFileDownload", mock.Anything, mock.Anything).Return(&gateway.InitiateFileDownloadResponse{
					Status: status.NewNotFound(ctx, "not found"),
				}, nil)
				gatewayClient.On("GetQuota", mock.Anything, mock.Anything).Return(&provider.GetQuotaResponse{
					Status: status.NewUnimplemented(ctx, fmt.Errorf("not supported"), "not supported"),
				}, nil)
				gatewayClient.On("Stat", mock.Anything, mock.Anything).Return(&provider.StatResponse{
					Status: status.NewNotFound(ctx, "no special files found"),
				}, nil)

				r := httptest.NewRequest(http.MethodGet, "/graph/v1.0/me/drives?$orderby=name%20asc", nil)
				r = r.WithContext(ctx)
				rr := httptest.NewRecorder()
				svc.GetDrivesV1(rr, r)

				Expect(rr.Code).To(Equal(http.StatusOK))

				body, _ := io.ReadAll(rr.Body)
				Expect(body).To(MatchJSON(`
			{
				"value":[
					{
						"driveAlias":"aspacetype/aspacename",
						"driveType":"aspacetype",
						"id":"pro-1$asameID",
						"name":"aspacename",
						"quota": {},
						"root":{
							"eTag":"101112131415",
							"id":"pro-1$asameID",
							"webDavUrl":"https://localhost:9200/dav/spaces/pro-1$asameID"
						},
						"webUrl": "https://localhost:9200/f/pro-1$asameID"
					},
					{
						"driveAlias":"bspacetype/bspacename",
						"driveType":"bspacetype",
						"id":"pro-1$bsameID",
						"name":"bspacename",
						"quota": {},
						"root":{
							"eTag":"123456789",
							"id":"pro-1$bsameID",
							"webDavUrl":"https://localhost:9200/dav/spaces/pro-1$bsameID"
						},
						"webUrl": "https://localhost:9200/f/pro-1$bsameID"
					}
				]
			}
			`))
			})
			It("can list a spaces type mountpoint", func() {
				gatewayClient.On("ListStorageSpaces", mock.Anything, mock.Anything).Return(&provider.ListStorageSpacesResponse{
					Status: status.NewOK(ctx),
					StorageSpaces: []*provider.StorageSpace{
						{
							Id:        &provider.StorageSpaceId{OpaqueId: "prID$aID!differentID"},
							SpaceType: "mountpoint",
							Root: &provider.ResourceId{
								StorageId: "prID",
								SpaceId:   "aID",
								OpaqueId:  "differentID",
							},
							Name: "New Folder",
							Opaque: &typesv1beta1.Opaque{
								Map: map[string]*typesv1beta1.OpaqueEntry{
									"spaceAlias":     {Decoder: "plain", Value: []byte("mountpoint/new-folder")},
									"etag":           {Decoder: "plain", Value: []byte("101112131415")},
									"grantStorageID": {Decoder: "plain", Value: []byte("ownerStorageID")},
									"grantSpaceID":   {Decoder: "plain", Value: []byte("spaceID")},
									"grantOpaqueID":  {Decoder: "plain", Value: []byte("opaqueID")},
								},
							},
							RootInfo: &provider.ResourceInfo{Path: "New Folder", Name: "Project Mars"},
						},
					},
				}, nil)
				var opaque *typesv1beta1.Opaque
				opaque = utils.AppendPlainToOpaque(opaque, "spaceAlias", "project/project-mars")
				gatewayClient.On("Stat", mock.Anything, mock.Anything).Return(&provider.StatResponse{
					Status: status.NewOK(ctx),
					Info: &provider.ResourceInfo{
						Etag:  "123456789",
						Type:  provider.ResourceType_RESOURCE_TYPE_CONTAINER,
						Id:    &provider.ResourceId{StorageId: "ownerStorageID", SpaceId: "spaceID", OpaqueId: "opaqueID"},
						Path:  "Folder/New Folder",
						Mtime: &typesv1beta1.Timestamp{Seconds: 1648327606, Nanos: 0},
						Size:  uint64(1234),
						Name:  "New Folder",
						Space: &provider.StorageSpace{
							Name:      "Project Mars",
							SpaceType: "project",
							Opaque:    opaque,
							Root:      &provider.ResourceId{StorageId: "ownerStorageID", SpaceId: "spaceID", OpaqueId: "opaqueID"},
						},
					},
				}, nil)
				gatewayClient.On("GetQuota", mock.Anything, mock.Anything).Return(&provider.GetQuotaResponse{
					Status: status.NewUnimplemented(ctx, fmt.Errorf("not supported"), "not supported"),
				}, nil)

				r := httptest.NewRequest(http.MethodGet, "/graph/v1.0/me/drives", nil)
				r = r.WithContext(ctx)
				rr := httptest.NewRecorder()
				svc.GetDrivesV1(rr, r)

				Expect(rr.Code).To(Equal(http.StatusOK))

				body, _ := io.ReadAll(rr.Body)

				var response map[string][]libregraph.Drive
				err := json.Unmarshal(body, &response)
				Expect(err).ToNot(HaveOccurred())
				Expect(len(response["value"])).To(Equal(1))
				value := response["value"][0]
				Expect(*value.DriveAlias).To(Equal("mountpoint/new-folder"))
				Expect(*value.DriveType).To(Equal("mountpoint"))
				Expect(*value.Id).To(Equal("prID$aID!differentID"))
				Expect(value.Name).To(Equal("New Folder"))
				Expect(*value.Root.WebDavUrl).To(Equal("https://localhost:9200/dav/spaces/prID$aID%21differentID"))
				Expect(*value.Root.ETag).To(Equal("101112131415"))
				Expect(*value.Root.Id).To(Equal("prID$aID!differentID"))
				Expect(*value.Root.RemoteItem.ETag).To(Equal("123456789"))
				Expect(*value.Root.RemoteItem.Id).To(Equal("ownerStorageID$spaceID!opaqueID"))
				Expect(value.Root.RemoteItem.LastModifiedDateTime.UTC()).To(Equal(time.Unix(1648327606, 0).UTC()))
				Expect(*value.Root.RemoteItem.Folder).To(Equal(libregraph.Folder{}))
				Expect(*value.Root.RemoteItem.Name).To(Equal("New Folder"))
				Expect(*value.Root.RemoteItem.Path).To(Equal("Folder/New Folder"))
				Expect(*value.Root.RemoteItem.Size).To(Equal(int64(1234)))
				Expect(*value.Root.RemoteItem.WebDavUrl).To(Equal("https://localhost:9200/dav/spaces/ownerStorageID$spaceID%21opaqueID/Folder/New%20Folder"))
			})
			It("can not list spaces with wrong sort parameter", func() {
				gatewayClient.On("ListStorageSpaces", mock.Anything, mock.Anything).Return(&provider.ListStorageSpacesResponse{
					Status:        status.NewOK(ctx),
					StorageSpaces: []*provider.StorageSpace{}}, nil)
				gatewayClient.On("InitiateFileDownload", mock.Anything, mock.Anything).Return(&gateway.InitiateFileDownloadResponse{
					Status: status.NewNotFound(ctx, "not found"),
				}, nil)
				gatewayClient.On("GetQuota", mock.Anything, mock.Anything).Return(&provider.GetQuotaResponse{
					Status: status.NewUnimplemented(ctx, fmt.Errorf("not supported"), "not supported"),
				}, nil)

				r := httptest.NewRequest(http.MethodGet, "/graph/v1.0/me/drives?$orderby=owner%20asc", nil)
				r = r.WithContext(ctx)
				rr := httptest.NewRecorder()
				svc.GetDrivesV1(rr, r)
				Expect(rr.Code).To(Equal(http.StatusBadRequest))

				body, _ := io.ReadAll(rr.Body)
				var libreError libregraph.OdataError
				err := json.Unmarshal(body, &libreError)
				Expect(err).To(Not(HaveOccurred()))
				Expect(libreError.Error.Message).To(Equal("we do not support <owner> as a order parameter"))
				Expect(libreError.Error.Code).To(Equal(errorcode.InvalidRequest.String()))
			})
			It("can list a spaces with invalid query parameter", func() {
				r := httptest.NewRequest(http.MethodGet, "/graph/v1.0/me/drives?§orderby=owner%20asc", nil)
				rr := httptest.NewRecorder()
				svc.GetDrivesV1(rr, r)
				Expect(rr.Code).To(Equal(http.StatusBadRequest))

				body, _ := io.ReadAll(rr.Body)
				var libreError libregraph.OdataError
				err := json.Unmarshal(body, &libreError)
				Expect(err).To(Not(HaveOccurred()))
				Expect(libreError.Error.Message).To(Equal("Query parameter '§orderby' is not supported. Cause: Query parameter '§orderby' is not supported"))
				Expect(libreError.Error.Code).To(Equal(errorcode.InvalidRequest.String()))
			})
			It("can list a spaces with an unsupported operand", func() {
				r := httptest.NewRequest(http.MethodGet, "/graph/v1.0/me/drives?$filter=contains(driveType,personal)", nil)
				rr := httptest.NewRecorder()
				svc.GetDrivesV1(rr, r)
				Expect(rr.Code).To(Equal(http.StatusNotImplemented))

				body, _ := io.ReadAll(rr.Body)
				var libreError libregraph.OdataError
				err := json.Unmarshal(body, &libreError)
				Expect(err).To(Not(HaveOccurred()))
				Expect(libreError.Error.Message).To(Equal("unsupported filter operand: contains"))
				Expect(libreError.Error.Code).To(Equal(errorcode.NotSupported.String()))
			})
			It("transport error", func() {
				gatewayClient.On("ListStorageSpaces", mock.Anything, mock.Anything).Return(nil, errors.New("transport error"))

				r := httptest.NewRequest(http.MethodGet, "/graph/v1.0/me/drives)", nil)
				r = r.WithContext(ctx)
				rr := httptest.NewRecorder()
				svc.GetDrivesV1(rr, r)
				Expect(rr.Code).To(Equal(http.StatusInternalServerError))

				body, _ := io.ReadAll(rr.Body)
				var libreError libregraph.OdataError
				err := json.Unmarshal(body, &libreError)
				Expect(err).To(Not(HaveOccurred()))
				Expect(libreError.Error.Message).To(Equal("transport error"))
				Expect(libreError.Error.Code).To(Equal(errorcode.GeneralException.String()))
			})
			It("grpc error", func() {
				gatewayClient.On("ListStorageSpaces", mock.Anything, mock.Anything).Return(&provider.ListStorageSpacesResponse{
					Status:        status.NewInternal(ctx, "internal error"),
					StorageSpaces: []*provider.StorageSpace{}}, nil)

				r := httptest.NewRequest(http.MethodGet, "/graph/v1.0/me/drives)", nil)
				r = r.WithContext(ctx)
				rr := httptest.NewRecorder()
				svc.GetDrivesV1(rr, r)
				Expect(rr.Code).To(Equal(http.StatusInternalServerError))

				body, _ := io.ReadAll(rr.Body)
				var libreError libregraph.OdataError
				err := json.Unmarshal(body, &libreError)
				Expect(err).To(Not(HaveOccurred()))
				Expect(libreError.Error.Message).To(Equal("internal error"))
				Expect(libreError.Error.Code).To(Equal(errorcode.GeneralException.String()))
			})
			It("grpc not found", func() {
				gatewayClient.On("ListStorageSpaces", mock.Anything, mock.Anything).Return(&provider.ListStorageSpacesResponse{
					Status:        status.NewNotFound(ctx, "no spaces found"),
					StorageSpaces: []*provider.StorageSpace{}}, nil)

				r := httptest.NewRequest(http.MethodGet, "/graph/v1.0/me/drives)", nil)
				r = r.WithContext(ctx)
				rr := httptest.NewRecorder()
				svc.GetDrivesV1(rr, r)
				Expect(rr.Code).To(Equal(http.StatusOK))

				body, _ := io.ReadAll(rr.Body)

				var response map[string][]libregraph.Drive
				err := json.Unmarshal(body, &response)
				Expect(err).ToNot(HaveOccurred())
				Expect(len(response)).To(Equal(0))
			})
			It("quota error", func() {
				gatewayClient.On("ListStorageSpaces", mock.Anything, mock.Anything).Return(&provider.ListStorageSpacesResponse{
					Status: status.NewOK(ctx),
					StorageSpaces: []*provider.StorageSpace{
						{
							Id:        &provider.StorageSpaceId{OpaqueId: "sameID"},
							SpaceType: "aspacetype",
							Root: &provider.ResourceId{
								StorageId: "pro-1",
								SpaceId:   "sameID",
								OpaqueId:  "sameID",
							},
							Name: "aspacename",
						},
					},
				}, nil)
				gatewayClient.On("InitiateFileDownload", mock.Anything, mock.Anything).Return(&gateway.InitiateFileDownloadResponse{
					Status: status.NewNotFound(ctx, "not found"),
				}, nil)
				gatewayClient.On("GetQuota", mock.Anything, mock.Anything).Return(&provider.GetQuotaResponse{
					Status: status.NewInternal(ctx, "internal quota error"),
				}, nil)

				r := httptest.NewRequest(http.MethodGet, "/graph/v1.0/me/drives", nil)
				r = r.WithContext(ctx)
				rr := httptest.NewRecorder()
				svc.GetDrivesV1(rr, r)

				Expect(rr.Code).To(Equal(http.StatusInternalServerError))

				body, _ := io.ReadAll(rr.Body)
				var libreError libregraph.OdataError
				err := json.Unmarshal(body, &libreError)
				Expect(err).To(Not(HaveOccurred()))
				Expect(libreError.Error.Message).To(Equal("internal quota error"))
				Expect(libreError.Error.Code).To(Equal(errorcode.GeneralException.String()))
			})
			It("omit permissions by default", func() {
				gatewayClient.On("ListStorageSpaces", mock.Anything, mock.Anything).Times(1).Return(&provider.ListStorageSpacesResponse{
					Status: status.NewOK(ctx),
					StorageSpaces: []*provider.StorageSpace{
						{
							Opaque: utils.AppendJSONToOpaque(nil, "grants", map[string]provider.ResourcePermissions{
								"1": *conversions.NewManagerRole().CS3ResourcePermissions(),
							}),
							Root: &provider.ResourceId{},
						},
					},
				}, nil)
				gatewayClient.On("InitiateFileDownload", mock.Anything, mock.Anything).Return(&gateway.InitiateFileDownloadResponse{
					Status: status.NewNotFound(ctx, "not found"),
				}, nil)
				gatewayClient.On("GetQuota", mock.Anything, mock.Anything).Return(&provider.GetQuotaResponse{
					Status: status.NewUnimplemented(ctx, fmt.Errorf("not supported"), "not supported"),
				}, nil)
				gatewayClient.On("GetUser", mock.Anything, mock.Anything).Return(&userprovider.GetUserResponse{
					Status: status.NewUnimplemented(ctx, fmt.Errorf("not supported"), "not supported"),
				}, nil)

				r := httptest.NewRequest(http.MethodGet, "/graph/v1.0/me/drives", nil)
				r = r.WithContext(ctx)
				rr := httptest.NewRecorder()
				svc.GetDrivesV1(rr, r)

				Expect(rr.Code).To(Equal(http.StatusOK))

				jsonData := gjson.Get(rr.Body.String(), "value")

				Expect(jsonData.Get("#").Num).To(Equal(float64(1)))
				Expect(jsonData.Get("0.root.permissions").Exists()).To(BeFalse())
			})
		})
		DescribeTable("GetDrivesV1Beta1 and GetAllDrivesV1Beta1 expands root permissions",
			func(check func(gjson.Result), resourcePermissions provider.ResourcePermissions) {
				permissionService.On("GetPermissionByID", mock.Anything, mock.Anything).Return(&settingssvc.GetPermissionByIDResponse{
					Permission: &v0.Permission{
						// the permission is not relevant for this test
					},
				}, nil)
				gatewayClient.On("ListStorageSpaces", mock.Anything, mock.Anything).Times(1).Return(&provider.ListStorageSpacesResponse{
					Status: status.NewOK(ctx),
					StorageSpaces: []*provider.StorageSpace{
						{
							Opaque: utils.AppendJSONToOpaque(nil, "grants", map[string]provider.ResourcePermissions{
								"1": resourcePermissions,
							}),
							Root: &provider.ResourceId{},
						},
					},
				}, nil)
				gatewayClient.On("InitiateFileDownload", mock.Anything, mock.Anything).Return(&gateway.InitiateFileDownloadResponse{
					Status: status.NewNotFound(ctx, "not found"),
				}, nil)
				gatewayClient.On("GetQuota", mock.Anything, mock.Anything).Return(&provider.GetQuotaResponse{
					Status: status.NewUnimplemented(ctx, fmt.Errorf("not supported"), "not supported"),
				}, nil)
				gatewayClient.On("GetUser", mock.Anything, mock.Anything).Return(&userprovider.GetUserResponse{
					Status: status.NewUnimplemented(ctx, fmt.Errorf("not supported"), "not supported"),
				}, nil)

				r := httptest.NewRequest(http.MethodGet, "/graph/v1beta1.0/me/drives?$expand=root($expand=permissions)", nil)
				r = r.WithContext(ctx)
				rr := httptest.NewRecorder()
				svc.GetDrivesV1Beta1(rr, r)

				Expect(rr.Code).To(Equal(http.StatusOK))

				jsonData := gjson.Get(rr.Body.String(), "value")

				Expect(jsonData.Get("#").Num).To(Equal(float64(1)))
				Expect(jsonData.Get("0.root.permissions.#").Num).To(Equal(float64(1)))
				Expect(jsonData.Get("0.root.permissions.0.grantedToIdentities").Exists()).To(BeFalse())
				Expect(jsonData.Get("0.root.permissions.0.grantedToIdentities").Exists()).To(BeFalse())
				Expect(jsonData.Get("0.root.permissions.0.grantedToV2.user.id").Str).To(Equal("1"))
				Expect(jsonData.Get("0.root.permissions.0.roles.#").Num).To(Equal(float64(1)))

				check(jsonData)
			},
			Entry("injects grantedToV2", func(jsonData gjson.Result) {},
				*conversions.NewSpaceViewerRole().CS3ResourcePermissions()),
			Entry("remaps manager role to the unified counterpart", func(jsonData gjson.Result) {
				Expect(jsonData.Get("0.root.permissions.0.roles.0").Str).To(Equal(unifiedrole.UnifiedRoleManagerID))
			}, *conversions.NewManagerRole().CS3ResourcePermissions()),
			Entry("remaps editor role to the unified counterpart", func(jsonData gjson.Result) {
				Expect(jsonData.Get("0.root.permissions.0.roles.0").Str).To(Equal(unifiedrole.UnifiedRoleSpaceEditorID))
			}, *conversions.NewSpaceEditorRole().CS3ResourcePermissions()),
			Entry("remaps viewer role to the unified counterpart", func(jsonData gjson.Result) {
				Expect(jsonData.Get("0.root.permissions.0.roles.0").Str).To(Equal(unifiedrole.UnifiedRoleSpaceViewerID))
			}, *conversions.NewSpaceViewerRole().CS3ResourcePermissions()),
		)
		Describe("Create Drive", func() {
			It("cannot create a space without valid user in context", func() {
				jsonBody := []byte(`{"Name": "Test Space"}`)
				r := httptest.NewRequest(http.MethodPost, "/graph/v1.0/drives", bytes.NewBuffer(jsonBody))
				rr := httptest.NewRecorder()
				svc.CreateDrive(rr, r)
				Expect(rr.Code).To(Equal(http.StatusUnauthorized))

				body, _ := io.ReadAll(rr.Body)
				var libreError libregraph.OdataError
				err := json.Unmarshal(body, &libreError)
				Expect(err).To(Not(HaveOccurred()))
				Expect(libreError.Error.Message).To(Equal("invalid user"))
				Expect(libreError.Error.Code).To(Equal(errorcode.NotAllowed.String()))
			})
			It("cannot create a space without permission", func() {
				permissionService.On("GetPermissionByID", mock.Anything, mock.Anything).Return(&settingssvc.GetPermissionByIDResponse{
					Permission: &v0.Permission{
						Operation:  v0.Permission_OPERATION_UNKNOWN,
						Constraint: v0.Permission_CONSTRAINT_OWN,
					},
				}, nil)
				jsonBody := []byte(`{"Name": "Test Space"}`)
				r := httptest.NewRequest(http.MethodPost, "/graph/v1.0/drives", bytes.NewBuffer(jsonBody)).WithContext(ctx)
				rr := httptest.NewRecorder()
				svc.CreateDrive(rr, r)
				Expect(rr.Code).To(Equal(http.StatusForbidden))

				body, _ := io.ReadAll(rr.Body)
				var libreError libregraph.OdataError
				err := json.Unmarshal(body, &libreError)
				Expect(err).To(Not(HaveOccurred()))
				Expect(libreError.Error.Message).To(Equal("insufficient permissions to create a space."))
				Expect(libreError.Error.Code).To(Equal(errorcode.NotAllowed.String()))
			})
			It("cannot create a space with wrong request body", func() {
				permissionService.On("GetPermissionByID", mock.Anything, mock.Anything).Return(&settingssvc.GetPermissionByIDResponse{
					Permission: &v0.Permission{
						Operation:  v0.Permission_OPERATION_READWRITE,
						Constraint: v0.Permission_CONSTRAINT_ALL,
					},
				}, nil)
				jsonBody := []byte(`{"name": "Test Space"`)
				r := httptest.NewRequest(http.MethodPost, "/graph/v1.0/drives", bytes.NewBuffer(jsonBody)).WithContext(ctx)
				rr := httptest.NewRecorder()
				svc.CreateDrive(rr, r)
				Expect(rr.Code).To(Equal(http.StatusBadRequest))

				body, _ := io.ReadAll(rr.Body)
				var libreError libregraph.OdataError
				err := json.Unmarshal(body, &libreError)
				Expect(err).To(Not(HaveOccurred()))
				Expect(libreError.Error.Message).To(Equal("invalid body schema definition"))
				Expect(libreError.Error.Code).To(Equal(errorcode.InvalidRequest.String()))
			})
			It("transport error", func() {
				permissionService.On("GetPermissionByID", mock.Anything, mock.Anything).Return(&settingssvc.GetPermissionByIDResponse{
					Permission: &v0.Permission{
						Operation:  v0.Permission_OPERATION_READWRITE,
						Constraint: v0.Permission_CONSTRAINT_ALL,
					},
				}, nil)
				gatewayClient.On("CreateStorageSpace", mock.Anything, mock.Anything).Return(&provider.CreateStorageSpaceResponse{}, errors.New("transport error"))
				jsonBody := []byte(`{"name": "Test Space"}`)
				r := httptest.NewRequest(http.MethodPost, "/graph/v1.0/drives", bytes.NewBuffer(jsonBody)).WithContext(ctx)
				rr := httptest.NewRecorder()
				svc.CreateDrive(rr, r)
				Expect(rr.Code).To(Equal(http.StatusInternalServerError))

				body, _ := io.ReadAll(rr.Body)
				var libreError libregraph.OdataError
				err := json.Unmarshal(body, &libreError)
				Expect(err).To(Not(HaveOccurred()))
				Expect(libreError.Error.Message).To(Equal("transport error"))
				Expect(libreError.Error.Code).To(Equal(errorcode.GeneralException.String()))
			})
			It("grpc permission denied error", func() {
				permissionService.On("GetPermissionByID", mock.Anything, mock.Anything).Return(&settingssvc.GetPermissionByIDResponse{
					Permission: &v0.Permission{
						Operation:  v0.Permission_OPERATION_READWRITE,
						Constraint: v0.Permission_CONSTRAINT_ALL,
					},
				}, nil)
				gatewayClient.On("CreateStorageSpace", mock.Anything, mock.Anything).Return(&provider.CreateStorageSpaceResponse{
					Status: status.NewPermissionDenied(ctx, nil, "grpc permission denied"),
				}, nil)

				jsonBody := []byte(`{"name": "Test Space"}`)
				r := httptest.NewRequest(http.MethodPost, "/graph/v1.0/drives", bytes.NewBuffer(jsonBody)).WithContext(ctx)
				rr := httptest.NewRecorder()
				svc.CreateDrive(rr, r)
				Expect(rr.Code).To(Equal(http.StatusForbidden))

				body, _ := io.ReadAll(rr.Body)
				var libreError libregraph.OdataError
				err := json.Unmarshal(body, &libreError)
				Expect(err).To(Not(HaveOccurred()))
				Expect(libreError.Error.Message).To(Equal("permission denied"))
				Expect(libreError.Error.Code).To(Equal(errorcode.NotAllowed.String()))
			})
			It("grpc general error", func() {
				permissionService.On("GetPermissionByID", mock.Anything, mock.Anything).Return(&settingssvc.GetPermissionByIDResponse{
					Permission: &v0.Permission{
						Operation:  v0.Permission_OPERATION_READWRITE,
						Constraint: v0.Permission_CONSTRAINT_ALL,
					},
				}, nil)
				gatewayClient.On("CreateStorageSpace", mock.Anything, mock.Anything).Return(&provider.CreateStorageSpaceResponse{
					Status: status.NewInternal(ctx, "grpc error"),
				}, nil)

				jsonBody := []byte(`{"name": "Test Space"}`)
				r := httptest.NewRequest(http.MethodPost, "/graph/v1.0/drives", bytes.NewBuffer(jsonBody)).WithContext(ctx)
				rr := httptest.NewRecorder()
				svc.CreateDrive(rr, r)
				Expect(rr.Code).To(Equal(http.StatusInternalServerError))

				body, _ := io.ReadAll(rr.Body)
				var libreError libregraph.OdataError
				err := json.Unmarshal(body, &libreError)
				Expect(err).To(Not(HaveOccurred()))
				Expect(libreError.Error.Message).To(Equal("grpc error"))
				Expect(libreError.Error.Code).To(Equal(errorcode.GeneralException.String()))
			})
			It("cannot create a space with empty name", func() {
				permissionService.On("GetPermissionByID", mock.Anything, mock.Anything).Return(&settingssvc.GetPermissionByIDResponse{
					Permission: &v0.Permission{
						Operation:  v0.Permission_OPERATION_READWRITE,
						Constraint: v0.Permission_CONSTRAINT_ALL,
					},
				}, nil)
				jsonBody := []byte(`{"name": ""}`)
				r := httptest.NewRequest(http.MethodPost, "/graph/v1.0/drives", bytes.NewBuffer(jsonBody)).WithContext(ctx)
				rr := httptest.NewRecorder()
				svc.CreateDrive(rr, r)
				Expect(rr.Code).To(Equal(http.StatusBadRequest))

				body, _ := io.ReadAll(rr.Body)
				var libreError libregraph.OdataError
				err := json.Unmarshal(body, &libreError)
				Expect(err).To(Not(HaveOccurred()))
				Expect(libreError.Error.Message).To(Equal("invalid spacename: spacename must not be empty"))
				Expect(libreError.Error.Code).To(Equal(errorcode.InvalidRequest.String()))
			})
			It("cannot create a space with a name that exceeds 255 chars", func() {
				permissionService.On("GetPermissionByID", mock.Anything, mock.Anything).Return(&settingssvc.GetPermissionByIDResponse{
					Permission: &v0.Permission{
						Operation:  v0.Permission_OPERATION_READWRITE,
						Constraint: v0.Permission_CONSTRAINT_ALL,
					},
				}, nil)
				jsonBody := []byte(`{"name": "uufZ2MEUjUMJa84RkPsjJ1zf4XXRTdVMxRsJGfevwHuUBojo5JEdNU22O1FGgzXXTi9tl5ZKWaluIef8pPmEAxn9lHGIjyDVYeRQPiX5PCAZ7rVszrpLJryY5x1p6fFGQ6WQsPpNaqnKnfMliJDsbkAwMf7rCpzo0GUuadgHY9s2mfoXHDnpxqEmDsheucqVAFcNlFZNbNHoZAebHfv78KYc8C0WnhWvqvSPGBkNPQbZUkFCOAIlqpQ2Q3MubgI2"}`)
				r := httptest.NewRequest(http.MethodPost, "/graph/v1.0/drives", bytes.NewBuffer(jsonBody)).WithContext(ctx)
				rr := httptest.NewRecorder()
				svc.CreateDrive(rr, r)
				Expect(rr.Code).To(Equal(http.StatusBadRequest))

				body, _ := io.ReadAll(rr.Body)
				var libreError libregraph.OdataError
				err := json.Unmarshal(body, &libreError)
				Expect(err).To(Not(HaveOccurred()))
				Expect(libreError.Error.Message).To(Equal("invalid spacename: spacename must be smaller than 255"))
				Expect(libreError.Error.Code).To(Equal(errorcode.InvalidRequest.String()))
			})
			It("cannot create a space with a wrong type", func() {
				permissionService.On("GetPermissionByID", mock.Anything, mock.Anything).Return(&settingssvc.GetPermissionByIDResponse{
					Permission: &v0.Permission{
						Operation:  v0.Permission_OPERATION_READWRITE,
						Constraint: v0.Permission_CONSTRAINT_ALL,
					},
				}, nil)
				jsonBody := []byte(`{"name": "Test", "DriveType": "media"}`)
				r := httptest.NewRequest(http.MethodPost, "/graph/v1.0/drives", bytes.NewBuffer(jsonBody)).WithContext(ctx)
				rr := httptest.NewRecorder()
				svc.CreateDrive(rr, r)
				Expect(rr.Code).To(Equal(http.StatusBadRequest))

				body, _ := io.ReadAll(rr.Body)
				var libreError libregraph.OdataError
				err := json.Unmarshal(body, &libreError)
				Expect(err).To(Not(HaveOccurred()))
				Expect(libreError.Error.Message).To(Equal("drives of this type cannot be created via this api"))
				Expect(libreError.Error.Code).To(Equal(errorcode.InvalidRequest.String()))
			})
			It("cannot create a space with a name that contains invalid chars", func() {
				permissionService.On("GetPermissionByID", mock.Anything, mock.Anything).Return(&settingssvc.GetPermissionByIDResponse{
					Permission: &v0.Permission{
						Operation:  v0.Permission_OPERATION_READWRITE,
						Constraint: v0.Permission_CONSTRAINT_ALL,
					},
				}, nil)
				jsonBody := []byte(`{"name": "Space / Name"}`)
				r := httptest.NewRequest(http.MethodPost, "/graph/v1.0/drives", bytes.NewBuffer(jsonBody)).WithContext(ctx)
				rr := httptest.NewRecorder()
				svc.CreateDrive(rr, r)
				Expect(rr.Code).To(Equal(http.StatusBadRequest))

				body, _ := io.ReadAll(rr.Body)
				var libreError libregraph.OdataError
				err := json.Unmarshal(body, &libreError)
				Expect(err).To(Not(HaveOccurred()))
				Expect(libreError.Error.Message).To(Equal("invalid spacename: spacenames must not contain [/ \\ . : ? * \" > < |]"))
				Expect(libreError.Error.Code).To(Equal(errorcode.InvalidRequest.String()))
			})
			It("can create a project space", func() {
				gatewayClient.On("CreateStorageSpace", mock.Anything, mock.Anything).Return(&provider.CreateStorageSpaceResponse{
					Status: status.NewOK(ctx),
					StorageSpace: &provider.StorageSpace{
						Id:        &provider.StorageSpaceId{OpaqueId: "newID"},
						Name:      "Test Space",
						SpaceType: "project",
						Root: &provider.ResourceId{
							StorageId: "pro-1",
							SpaceId:   "newID",
							OpaqueId:  "newID",
						},
						Opaque: &typesv1beta1.Opaque{
							Map: map[string]*typesv1beta1.OpaqueEntry{
								"description": {Decoder: "plain", Value: []byte("This space is for testing")},
								"spaceAlias":  {Decoder: "plain", Value: []byte("project/testspace")},
								"contentType": {Decoder: "plain", Value: []byte("application/vnd.opencloud.test")},
							},
						},
					},
				}, nil)

				permissionService.On("GetPermissionByID", mock.Anything, mock.Anything).Return(&settingssvc.GetPermissionByIDResponse{
					Permission: &v0.Permission{
						Operation:  v0.Permission_OPERATION_READWRITE,
						Constraint: v0.Permission_CONSTRAINT_ALL,
					},
				}, nil)

				gatewayClient.On("GetQuota", mock.Anything, mock.Anything).Return(&provider.GetQuotaResponse{
					Status:     status.NewOK(ctx),
					TotalBytes: 500,
				}, nil)

				jsonBody := []byte(`{"name": "Test Space", "driveType": "project", "description": "This space is for testing", "DriveAlias": "project/testspace", "@libre.graph.contentType": "application/vnd.opencloud.test"}`)
				r := httptest.NewRequest(http.MethodPost, "/graph/v1.0/drives", bytes.NewBuffer(jsonBody)).WithContext(ctx)
				rr := httptest.NewRecorder()
				svc.CreateDrive(rr, r)
				Expect(rr.Code).To(Equal(http.StatusCreated))

				body, _ := io.ReadAll(rr.Body)
				var response libregraph.Drive
				err := json.Unmarshal(body, &response)
				Expect(err).ToNot(HaveOccurred())
				Expect(response.Name).To(Equal("Test Space"))
				Expect(*response.DriveType).To(Equal("project"))
				Expect(*response.DriveAlias).To(Equal("project/testspace"))
				Expect(*response.Description).To(Equal("This space is for testing"))
				Expect(*response.LibreGraphContentType).To(Equal("application/vnd.opencloud.test"))
			})
			Context("drive id handling", func() {
				const spaceID = "0b9a7b84-3e71-4a4e-9c34-0e6a4c3f5f21"

				// csr captures the request the handler sends to the gateway
				var csr *provider.CreateStorageSpaceRequest
				mockSuccessfulCreate := func(createdSpaceID string) {
					gatewayClient.On("CreateStorageSpace", mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
						csr = args.Get(1).(*provider.CreateStorageSpaceRequest)
					}).Return(&provider.CreateStorageSpaceResponse{
						Status: status.NewOK(ctx),
						StorageSpace: &provider.StorageSpace{
							Id:        &provider.StorageSpaceId{OpaqueId: "pro-1$" + createdSpaceID},
							Name:      "Test Space",
							SpaceType: "project",
							Root:      &provider.ResourceId{StorageId: "pro-1", SpaceId: createdSpaceID, OpaqueId: createdSpaceID},
						},
					}, nil)
					gatewayClient.On("GetQuota", mock.Anything, mock.Anything).Return(&provider.GetQuotaResponse{
						Status:     status.NewOK(ctx),
						TotalBytes: 500,
					}, nil)
				}

				BeforeEach(func() {
					csr = nil
					permissionService.On("GetPermissionByID", mock.Anything, mock.Anything).Return(&settingssvc.GetPermissionByIDResponse{
						Permission: &v0.Permission{
							Operation:  v0.Permission_OPERATION_READWRITE,
							Constraint: v0.Permission_CONSTRAINT_ALL,
						},
					}, nil)
				})

				DescribeTable("lets the storage provider generate the id when none is supplied, without looking up users",
					func(jsonBody string) {
						mockSuccessfulCreate("generated-id")

						r := httptest.NewRequest(http.MethodPost, "/graph/v1.0/drives", bytes.NewBufferString(jsonBody)).WithContext(ctx)
						rr := httptest.NewRecorder()
						svc.CreateDrive(rr, r)
						Expect(rr.Code).To(Equal(http.StatusCreated))
						Expect(csr).ToNot(BeNil())
						Expect(csr.GetOpaque().GetMap()).ToNot(HaveKey("spaceid"))
						identityBackend.AssertNotCalled(GinkgoT(), "GetUser", mock.Anything, mock.Anything, mock.Anything)
					},
					Entry("absent id", `{"name": "Test Space"}`),
					Entry("empty id", `{"name": "Test Space", "id": ""}`),
				)

				It("forwards the id to the storage provider when it does not belong to a user", func() {
					identityBackend.On("GetUser", mock.Anything, spaceID, mock.Anything).Return(nil, identity.ErrNotFound)
					mockSuccessfulCreate(spaceID)

					jsonBody := []byte(`{"name": "Test Space", "id": "` + spaceID + `"}`)
					r := httptest.NewRequest(http.MethodPost, "/graph/v1.0/drives", bytes.NewBuffer(jsonBody)).WithContext(ctx)
					rr := httptest.NewRecorder()
					svc.CreateDrive(rr, r)
					Expect(rr.Code).To(Equal(http.StatusCreated))
					Expect(utils.ReadPlainFromOpaque(csr.GetOpaque(), "spaceid")).To(Equal(spaceID))

					var response libregraph.Drive
					Expect(json.Unmarshal(rr.Body.Bytes(), &response)).To(Succeed())
					Expect(response.GetId()).To(Equal("pro-1$" + spaceID))
					identityBackend.AssertCalled(GinkgoT(), "GetUser", mock.Anything, spaceID, mock.Anything)
				})

				// In the next two tests a successful CreateStorageSpace is mocked on purpose: a handler that
				// skips the user lookup then fails on the status and AssertNotCalled checks, not on an
				// unexpected mock call.
				It("returns 409 Conflict without creating the space when the id belongs to a user", func() {
					identityBackend.On("GetUser", mock.Anything, spaceID, mock.Anything).Return(&libregraph.User{Id: libregraph.PtrString(spaceID)}, nil)
					mockSuccessfulCreate(spaceID)

					jsonBody := []byte(`{"name": "Test Space", "id": "` + spaceID + `"}`)
					r := httptest.NewRequest(http.MethodPost, "/graph/v1.0/drives", bytes.NewBuffer(jsonBody)).WithContext(ctx)
					rr := httptest.NewRecorder()
					svc.CreateDrive(rr, r)
					Expect(rr.Code).To(Equal(http.StatusConflict))

					var libreError libregraph.OdataError
					Expect(json.Unmarshal(rr.Body.Bytes(), &libreError)).To(Succeed())
					Expect(libreError.Error.Code).To(Equal(errorcode.NameAlreadyExists.String()))
					Expect(libreError.Error.Message).To(Equal("drive already exists"))
					gatewayClient.AssertNotCalled(GinkgoT(), "CreateStorageSpace", mock.Anything, mock.Anything)
				})

				It("does not create the space when the identity backend fails to look up the id", func() {
					identityBackend.On("GetUser", mock.Anything, spaceID, mock.Anything).Return(nil, errorcode.New(errorcode.ServiceNotAvailable, "identity backend unavailable"))
					mockSuccessfulCreate(spaceID)

					jsonBody := []byte(`{"name": "Test Space", "id": "` + spaceID + `"}`)
					r := httptest.NewRequest(http.MethodPost, "/graph/v1.0/drives", bytes.NewBuffer(jsonBody)).WithContext(ctx)
					rr := httptest.NewRecorder()
					svc.CreateDrive(rr, r)
					Expect(rr.Code).To(Equal(http.StatusInternalServerError))

					var libreError libregraph.OdataError
					Expect(json.Unmarshal(rr.Body.Bytes(), &libreError)).To(Succeed())
					Expect(libreError.Error.Code).To(Equal(errorcode.ServiceNotAvailable.String()))
					gatewayClient.AssertNotCalled(GinkgoT(), "CreateStorageSpace", mock.Anything, mock.Anything)
				})

				DescribeTable("rejects ids that are not canonical UUIDs before looking up users or calling the storage provider",
					func(id string) {
						gatewayClient.On("CreateStorageSpace", mock.Anything, mock.Anything).Return(&provider.CreateStorageSpaceResponse{
							Status: status.NewInternal(ctx, "storage provider must not be called"),
						}, nil).Maybe()

						body, err := json.Marshal(map[string]string{"name": "Test Space", "id": id})
						Expect(err).ToNot(HaveOccurred())
						r := httptest.NewRequest(http.MethodPost, "/graph/v1.0/drives", bytes.NewBuffer(body)).WithContext(ctx)
						rr := httptest.NewRecorder()
						svc.CreateDrive(rr, r)
						Expect(rr.Code).To(Equal(http.StatusBadRequest))

						var libreError libregraph.OdataError
						Expect(json.Unmarshal(rr.Body.Bytes(), &libreError)).To(Succeed())
						Expect(libreError.Error.Code).To(Equal(errorcode.InvalidRequest.String()))
						Expect(libreError.Error.Message).To(Equal("invalid drive id: must be a canonical UUID"))
						identityBackend.AssertNotCalled(GinkgoT(), "GetUser", mock.Anything, mock.Anything, mock.Anything)
						gatewayClient.AssertNotCalled(GinkgoT(), "CreateStorageSpace", mock.Anything, mock.Anything)
					},
					Entry("path traversal", "../../x"),
					Entry("storage id delimiter", "pro-1$"+spaceID),
					Entry("opaque id delimiter", spaceID+"!"+spaceID),
					Entry("arbitrary string", "not-a-uuid"),
					Entry("uppercase", strings.ToUpper(spaceID)),
					Entry("braces", "{"+spaceID+"}"),
					Entry("urn", "urn:uuid:"+spaceID),
					Entry("no hyphens", strings.ReplaceAll(spaceID, "-", "")),
				)

				It("returns 409 Conflict when the storage provider reports that the space already exists", func() {
					identityBackend.On("GetUser", mock.Anything, spaceID, mock.Anything).Return(nil, identity.ErrNotFound)
					gatewayClient.On("CreateStorageSpace", mock.Anything, mock.Anything).Return(&provider.CreateStorageSpaceResponse{
						Status: status.NewAlreadyExists(ctx, nil, "decomposedfs: spaces: space already exists"),
					}, nil)

					jsonBody := []byte(`{"name": "Test Space", "id": "` + spaceID + `"}`)
					r := httptest.NewRequest(http.MethodPost, "/graph/v1.0/drives", bytes.NewBuffer(jsonBody)).WithContext(ctx)
					rr := httptest.NewRecorder()
					svc.CreateDrive(rr, r)
					Expect(rr.Code).To(Equal(http.StatusConflict))

					var libreError libregraph.OdataError
					Expect(json.Unmarshal(rr.Body.Bytes(), &libreError)).To(Succeed())
					Expect(libreError.Error.Code).To(Equal(errorcode.NameAlreadyExists.String()))
					Expect(libreError.Error.Message).To(Equal("drive already exists"))
				})
			})
			It("Incomplete space", func() {
				gatewayClient.On("CreateStorageSpace", mock.Anything, mock.Anything).Return(&provider.CreateStorageSpaceResponse{
					Status: status.NewOK(ctx),
					StorageSpace: &provider.StorageSpace{
						Id:        &provider.StorageSpaceId{OpaqueId: "newID"},
						Name:      "Test Space",
						SpaceType: "project",
					},
				}, nil)

				permissionService.On("GetPermissionByID", mock.Anything, mock.Anything).Return(&settingssvc.GetPermissionByIDResponse{
					Permission: &v0.Permission{
						Operation:  v0.Permission_OPERATION_READWRITE,
						Constraint: v0.Permission_CONSTRAINT_ALL,
					},
				}, nil)
				jsonBody := []byte(`{"name": "Test Space"}`)
				r := httptest.NewRequest(http.MethodPost, "/graph/v1.0/drives", bytes.NewBuffer(jsonBody)).WithContext(ctx)
				rr := httptest.NewRecorder()
				svc.CreateDrive(rr, r)
				Expect(rr.Code).To(Equal(http.StatusInternalServerError))

				body, _ := io.ReadAll(rr.Body)
				var libreError libregraph.OdataError
				err := json.Unmarshal(body, &libreError)
				Expect(err).To(Not(HaveOccurred()))
				Expect(libreError.Error.Message).To(Equal("space has no root"))
				Expect(libreError.Error.Code).To(Equal(errorcode.GeneralException.String()))
			})
		})
	})

	Describe("Get a single drive", func() {
		BeforeEach(func() {
			gatewayClient.On("GetQuota", mock.Anything, mock.Anything).Return(&provider.GetQuotaResponse{
				Status:     status.NewOK(ctx),
				TotalBytes: 500,
			}, nil)
			permissionService.On("GetPermissionByID", mock.Anything, mock.Anything).Return(&settingssvc.GetPermissionByIDResponse{
				Permission: &v0.Permission{
					// the permission is not relevant for this test
				},
			}, nil)
		})

		It("handles missing drive id", func() {
			r := httptest.NewRequest(http.MethodGet, "/graph/v1.0/drives/{driveID}/", nil)
			svc.GetSingleDrive(rr, r)
			Expect(rr.Code).To(Equal(http.StatusBadRequest))
		})

		It("handles not found response", func() {
			gatewayClient.On("ListStorageSpaces", mock.Anything, mock.Anything).Return(&provider.ListStorageSpacesResponse{
				Status:        status.NewNotFound(ctx, "no spaces found"),
				StorageSpaces: []*provider.StorageSpace{},
			}, nil)

			r := httptest.NewRequest(http.MethodGet, "/graph/v1.0/drives/{driveID}/", nil)
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("driveID", "spaceid")
			r = r.WithContext(context.WithValue(revactx.ContextSetUser(ctx, nil), chi.RouteCtxKey, rctx))
			svc.GetSingleDrive(rr, r)
			Expect(rr.Code).To(Equal(http.StatusNotFound))
		})

		It("fails when more than one space is returned from the backend", func() {
			space := &provider.StorageSpace{
				Opaque: &typesv1beta1.Opaque{
					Map: map[string]*typesv1beta1.OpaqueEntry{
						"trashed": {Decoder: "plain", Value: []byte("trashed")},
					},
				},
				Id:        &provider.StorageSpaceId{OpaqueId: "spaceid"},
				SpaceType: "aspacetype",
				Root: &provider.ResourceId{
					StorageId: "pro-1",
					SpaceId:   "sameID",
					OpaqueId:  "sameID",
				},
				Name: "aspacename",
			}

			gatewayClient.On("ListStorageSpaces", mock.Anything, mock.Anything).Return(&provider.ListStorageSpacesResponse{
				Status:        status.NewOK(ctx),
				StorageSpaces: []*provider.StorageSpace{space, space},
			}, nil)

			r := httptest.NewRequest(http.MethodGet, "/graph/v1.0/drives/{driveID}/", nil)
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("driveID", "spaceid")
			r = r.WithContext(context.WithValue(revactx.ContextSetUser(ctx, nil), chi.RouteCtxKey, rctx))
			svc.GetSingleDrive(rr, r)
			Expect(rr.Code).To(Equal(http.StatusInternalServerError))
		})

		It("returns not found when the space wasn't found", func() {
			gatewayClient.On("ListStorageSpaces", mock.Anything, mock.Anything).Return(&provider.ListStorageSpacesResponse{
				Status:        status.NewOK(ctx),
				StorageSpaces: []*provider.StorageSpace{},
			}, nil)

			r := httptest.NewRequest(http.MethodGet, "/graph/v1.0/drives/{driveID}/", nil)
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("driveID", "spaceid")
			r = r.WithContext(context.WithValue(revactx.ContextSetUser(ctx, nil), chi.RouteCtxKey, rctx))
			svc.GetSingleDrive(rr, r)
			Expect(rr.Code).To(Equal(http.StatusNotFound))
		})

		It("doesn't return a quota for disabled drives", func() {
			gatewayClient.On("ListStorageSpaces",
				mock.Anything,
				mock.MatchedBy(
					func(req *provider.ListStorageSpacesRequest) bool {
						return len(req.Filters) == 1 && req.Filters[0].Term.(*provider.ListStorageSpacesRequest_Filter_Id).Id.OpaqueId == "spaceid"
					})).
				Return(&provider.ListStorageSpacesResponse{
					Status: status.NewOK(ctx),
					StorageSpaces: []*provider.StorageSpace{
						{
							Opaque: &typesv1beta1.Opaque{
								Map: map[string]*typesv1beta1.OpaqueEntry{
									"trashed": {Decoder: "plain", Value: []byte("trashed")},
								},
							},
							Id:        &provider.StorageSpaceId{OpaqueId: "spaceid"},
							SpaceType: "aspacetype",
							Root: &provider.ResourceId{
								StorageId: "pro-1",
								SpaceId:   "sameID",
								OpaqueId:  "sameID",
							},
							Name: "aspacename",
						},
					},
				}, nil)

			r := httptest.NewRequest(http.MethodGet, "/graph/v1.0/drives/{driveID}/", nil)
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("driveID", "spaceid")
			r = r.WithContext(context.WithValue(revactx.ContextSetUser(ctx, nil), chi.RouteCtxKey, rctx))
			svc.GetSingleDrive(rr, r)
			Expect(rr.Code).To(Equal(http.StatusOK))

			data, err := io.ReadAll(rr.Body)
			Expect(err).ToNot(HaveOccurred())

			drive := libregraph.Drive{}
			err = json.Unmarshal(data, &drive)
			Expect(err).ToNot(HaveOccurred())
			Expect(drive.GetQuota().Total).To(BeNil())
		})

		It("returns the drive", func() {
			gatewayClient.On("ListStorageSpaces",
				mock.Anything,
				mock.MatchedBy(
					func(req *provider.ListStorageSpacesRequest) bool {
						return len(req.Filters) == 1 && req.Filters[0].Term.(*provider.ListStorageSpacesRequest_Filter_Id).Id.OpaqueId == "spaceid"
					})).
				Return(&provider.ListStorageSpacesResponse{
					Status: status.NewOK(ctx),
					StorageSpaces: []*provider.StorageSpace{
						{
							Id:        &provider.StorageSpaceId{OpaqueId: "spaceid"},
							SpaceType: "aspacetype",
							Root: &provider.ResourceId{
								StorageId: "pro-1",
								SpaceId:   "sameID",
								OpaqueId:  "sameID",
							},
							Name: "aspacename",
						},
					},
				}, nil)

			r := httptest.NewRequest(http.MethodGet, "/graph/v1.0/drives/{driveID}/", nil)
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("driveID", "spaceid")
			r = r.WithContext(context.WithValue(revactx.ContextSetUser(ctx, nil), chi.RouteCtxKey, rctx))
			svc.GetSingleDrive(rr, r)
			Expect(rr.Code).To(Equal(http.StatusOK))

			data, err := io.ReadAll(rr.Body)
			Expect(err).ToNot(HaveOccurred())

			drive := libregraph.Drive{}
			err = json.Unmarshal(data, &drive)
			Expect(err).ToNot(HaveOccurred())
			Expect(*drive.GetQuota().Total).To(Equal(int64(500)))
		})

		It("gets the special drive items", func() {
			gatewayClient.On("GetPath", mock.Anything, mock.Anything).Return(&provider.GetPathResponse{
				Status: status.NewOK(ctx),
				Path:   "thepath",
			}, nil)
			// no stat for the image
			gatewayClient.On("Stat",
				mock.Anything,
				mock.MatchedBy(
					func(req *provider.StatRequest) bool {
						return req.Ref.Path == "/.space/logo.png"
					})).
				Return(&provider.StatResponse{
					Status: status.NewNotFound(ctx, "not found"),
				}, nil)
			// mock readme stats
			gatewayClient.On("Stat",
				mock.Anything,
				mock.Anything).
				Return(&provider.StatResponse{
					Status: status.NewOK(ctx),
					Info: &provider.ResourceInfo{
						Id: &provider.ResourceId{
							StorageId: "pro-1",
							SpaceId:   "spaceID",
							OpaqueId:  "specialID",
						},
					},
				}, nil)
			gatewayClient.On("ListStorageSpaces",
				mock.Anything,
				mock.MatchedBy(
					func(req *provider.ListStorageSpacesRequest) bool {
						return len(req.Filters) == 1 && req.Filters[0].Term.(*provider.ListStorageSpacesRequest_Filter_Id).Id.OpaqueId == "spaceid"
					})).
				Return(&provider.ListStorageSpacesResponse{
					Status: status.NewOK(ctx),
					StorageSpaces: []*provider.StorageSpace{
						{
							Opaque: &typesv1beta1.Opaque{
								Map: map[string]*typesv1beta1.OpaqueEntry{
									service.ReadmeSpecialFolderName: {
										Decoder: "plain",
										Value:   []byte("readme"),
									},
								},
							},
							Id:        &provider.StorageSpaceId{OpaqueId: "spaceid"},
							SpaceType: "aspacetype",
							Root: &provider.ResourceId{
								StorageId: "pro-1",
								SpaceId:   "sameID",
								OpaqueId:  "sameID",
							},
							Name: "aspacename",
						},
					},
				}, nil)

			r := httptest.NewRequest(http.MethodGet, "/graph/v1.0/drives/{driveID}/", nil)
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("driveID", "spaceid")
			r = r.WithContext(context.WithValue(revactx.ContextSetUser(ctx, nil), chi.RouteCtxKey, rctx))
			svc.GetSingleDrive(rr, r)
			Expect(rr.Code).To(Equal(http.StatusOK))

			data, err := io.ReadAll(rr.Body)
			Expect(err).ToNot(HaveOccurred())

			drive := libregraph.Drive{}
			err = json.Unmarshal(data, &drive)
			Expect(err).ToNot(HaveOccurred())
			Expect(*drive.GetQuota().Total).To(Equal(int64(500)))
			Expect(len(drive.GetSpecial())).To(Equal(1))
			Expect(drive.GetSpecial()[0].GetId()).To(Equal("pro-1$spaceID!specialID"))
		})
	})

	Describe("Update a drive", func() {
		BeforeEach(func() {
			gatewayClient.On("GetQuota", mock.Anything, mock.Anything).Return(&provider.GetQuotaResponse{
				Status:     status.NewOK(ctx),
				TotalBytes: 500,
			}, nil)
		})

		It("fails on missing drive id", func() {
			r := httptest.NewRequest(http.MethodPatch, "/graph/v1.0/drives/{driveID}/", nil)
			svc.UpdateDrive(rr, r)
			Expect(rr.Code).To(Equal(http.StatusBadRequest))

			r = httptest.NewRequest(http.MethodPatch, "/graph/v1.0/drives/{driveID}/", nil)
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("driveID", "")
			r = r.WithContext(context.WithValue(revactx.ContextSetUser(ctx, nil), chi.RouteCtxKey, rctx))
			svc.UpdateDrive(rr, r)
			Expect(rr.Code).To(Equal(http.StatusBadRequest))
		})

		It("fails on bad payload", func() {
			r := httptest.NewRequest(http.MethodPatch, "/graph/v1.0/drives/{driveID}/", bytes.NewBufferString("{invalid"))
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("driveID", "spaceid")
			r = r.WithContext(context.WithValue(revactx.ContextSetUser(ctx, nil), chi.RouteCtxKey, rctx))
			svc.UpdateDrive(rr, r)
			Expect(rr.Code).To(Equal(http.StatusBadRequest))
		})

		It("sets the description, alias, name and content type", func() {
			drive := libregraph.NewDrive("thename")
			drive.SetDriveAlias("thealias")
			drive.SetDescription("thedescription")
			drive.SetName("thename")
			drive.SetLibreGraphContentType("thecontenttype")
			driveJson, err := json.Marshal(drive)
			Expect(err).ToNot(HaveOccurred())

			gatewayClient.On("UpdateStorageSpace", mock.Anything, mock.Anything).Return(func(_ context.Context, req *provider.UpdateStorageSpaceRequest, _ ...grpc.CallOption) *provider.UpdateStorageSpaceResponse {
				return &provider.UpdateStorageSpaceResponse{
					Status:       status.NewOK(ctx),
					StorageSpace: req.StorageSpace,
				}
			}, nil)
			gatewayClient.On("Stat", mock.Anything, mock.Anything).Return(&provider.StatResponse{
				Status: status.NewNotFound(ctx, "no special files found"),
			}, nil)

			r := httptest.NewRequest(http.MethodPatch, "/graph/v1.0/drives/{driveID}/", bytes.NewBuffer(driveJson))
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("driveID", "spaceid")
			r = r.WithContext(context.WithValue(revactx.ContextSetUser(ctx, nil), chi.RouteCtxKey, rctx))
			svc.UpdateDrive(rr, r)
			Expect(rr.Code).To(Equal(http.StatusOK))

			gatewayClient.AssertCalled(GinkgoT(), "UpdateStorageSpace", mock.Anything, mock.MatchedBy(func(req *provider.UpdateStorageSpaceRequest) bool {
				return req.StorageSpace.Id.OpaqueId == "spaceid" &&
					utils.ReadPlainFromOpaque(req.StorageSpace.Opaque, "description") == drive.GetDescription() &&
					utils.ReadPlainFromOpaque(req.StorageSpace.Opaque, "spaceAlias") == drive.GetDriveAlias() &&
					utils.ReadPlainFromOpaque(req.StorageSpace.Opaque, "contentType") == drive.GetLibreGraphContentType()
			}))
		})

		DescribeTable("rejects an invalid name without updating the space",
			func(name string) {
				body, err := json.Marshal(map[string]string{"name": name})
				Expect(err).ToNot(HaveOccurred())
				r := httptest.NewRequest(http.MethodPatch, "/graph/v1.0/drives/{driveID}/", bytes.NewBuffer(body))
				rctx := chi.NewRouteContext()
				rctx.URLParams.Add("driveID", "spaceid")
				r = r.WithContext(context.WithValue(revactx.ContextSetUser(ctx, nil), chi.RouteCtxKey, rctx))
				svc.UpdateDrive(rr, r)
				Expect(rr.Code).To(Equal(http.StatusBadRequest))

				gatewayClient.AssertNotCalled(GinkgoT(), "UpdateStorageSpace", mock.Anything, mock.Anything)
			},
			// the names are written as escape sequences on purpose: written directly, these
			// characters would be invisible or look like ordinary spaces in the source code
			Entry("empty name", ""),
			Entry("name of only spaces", "   "),
			Entry("name of only tabs and line breaks", "\t\n\r"),
			// U+00A0 no-break space, U+202F narrow no-break space
			Entry("name of only no-break spaces", "\u00a0\u202f"),
			// U+2003 em space, U+2009 thin space, U+3000 ideographic space
			Entry("name of only wide spaces", "\u2003\u2009\u3000"),
			// U+200B zero-width space: invisible, but not whitespace in Unicode
			Entry("name of only zero-width spaces", "\u200b\u200b"),
			// U+200C zero-width non-joiner, U+200D zero-width joiner
			Entry("name of only zero-width joiners", "\u200c\u200d"),
			// U+2060 word joiner, U+FEFF byte order mark (zero-width no-break space)
			Entry("name of only a word joiner and a byte order mark", "\u2060\ufeff"),
			// space, U+200B zero-width space, U+00A0 no-break space, U+FEFF byte order mark, space
			Entry("name of mixed invisible characters", " \u200b\u00a0\ufeff "),
		)

		DescribeTable("keeps the name when the request doesn't set one",
			func(jsonBody string) {
				gatewayClient.On("UpdateStorageSpace", mock.Anything, mock.Anything).Return(func(_ context.Context, req *provider.UpdateStorageSpaceRequest, _ ...grpc.CallOption) *provider.UpdateStorageSpaceResponse {
					return &provider.UpdateStorageSpaceResponse{
						Status:       status.NewOK(ctx),
						StorageSpace: req.StorageSpace,
					}
				}, nil)
				gatewayClient.On("Stat", mock.Anything, mock.Anything).Return(&provider.StatResponse{
					Status: status.NewNotFound(ctx, "no special files found"),
				}, nil).Maybe()

				r := httptest.NewRequest(http.MethodPatch, "/graph/v1.0/drives/{driveID}/", bytes.NewBufferString(jsonBody))
				rctx := chi.NewRouteContext()
				rctx.URLParams.Add("driveID", "spaceid")
				r = r.WithContext(context.WithValue(revactx.ContextSetUser(ctx, nil), chi.RouteCtxKey, rctx))
				svc.UpdateDrive(rr, r)
				Expect(rr.Code).To(Equal(http.StatusOK))

				gatewayClient.AssertCalled(GinkgoT(), "UpdateStorageSpace", mock.Anything, mock.MatchedBy(func(req *provider.UpdateStorageSpaceRequest) bool {
					return req.StorageSpace.Id.OpaqueId == "spaceid" &&
						req.StorageSpace.Name == "" &&
						utils.ReadPlainFromOpaque(req.StorageSpace.Opaque, "description") == "thedescription"
				}))
			},
			Entry("no name", `{"description": "thedescription"}`),
			Entry("null name", `{"name": null, "description": "thedescription"}`),
		)

		It("restores", func() {
			drive := libregraph.NewDrive("thename")
			driveJson, err := json.Marshal(drive)
			Expect(err).ToNot(HaveOccurred())

			gatewayClient.On("UpdateStorageSpace", mock.Anything, mock.Anything).Return(func(_ context.Context, req *provider.UpdateStorageSpaceRequest, _ ...grpc.CallOption) *provider.UpdateStorageSpaceResponse {
				return &provider.UpdateStorageSpaceResponse{
					Status:       status.NewOK(ctx),
					StorageSpace: req.StorageSpace,
				}
			}, nil)

			r := httptest.NewRequest(http.MethodPatch, "/graph/v1.0/drives/{driveID}/", bytes.NewBuffer(driveJson))
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("driveID", "spaceid")
			r = r.WithContext(context.WithValue(revactx.ContextSetUser(ctx, nil), chi.RouteCtxKey, rctx))
			r.Header.Add("Restore", "1")
			svc.UpdateDrive(rr, r)
			Expect(rr.Code).To(Equal(http.StatusOK))

			gatewayClient.AssertCalled(GinkgoT(), "UpdateStorageSpace", mock.Anything, mock.MatchedBy(func(req *provider.UpdateStorageSpaceRequest) bool {
				return req.StorageSpace.Id.OpaqueId == "spaceid" && utils.ReadPlainFromOpaque(req.Opaque, "restore") == "true"
			}))
		})

		It("sets the quota", func() {
			drive := libregraph.NewDrive("thename")
			quota := libregraph.Quota{}
			quota.SetTotal(1000)
			drive.SetQuota(quota)
			driveJson, err := json.Marshal(drive)
			Expect(err).ToNot(HaveOccurred())

			gatewayClient.On("ListStorageSpaces", mock.Anything, mock.Anything).Return(&provider.ListStorageSpacesResponse{
				Status:        status.NewOK(ctx),
				StorageSpaces: []*provider.StorageSpace{{SpaceType: "project"}},
			}, nil)
			gatewayClient.On("UpdateStorageSpace", mock.Anything, mock.Anything).Return(func(_ context.Context, req *provider.UpdateStorageSpaceRequest, _ ...grpc.CallOption) *provider.UpdateStorageSpaceResponse {
				return &provider.UpdateStorageSpaceResponse{
					Status:       status.NewOK(ctx),
					StorageSpace: req.StorageSpace,
				}
			}, nil)
			permissionService.On("GetPermissionByID", mock.Anything, mock.Anything).Return(&settingssvc.GetPermissionByIDResponse{
				Permission: &v0.Permission{
					Operation:  v0.Permission_OPERATION_UNKNOWN,
					Constraint: v0.Permission_CONSTRAINT_OWN,
				},
			}, nil)

			r := httptest.NewRequest(http.MethodPatch, "/graph/v1.0/drives/{driveID}/", bytes.NewBuffer(driveJson))
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("driveID", "spaceid")
			r = r.WithContext(context.WithValue(revactx.ContextSetUser(ctx, currentUser), chi.RouteCtxKey, rctx))
			svc.UpdateDrive(rr, r)
			Expect(rr.Code).To(Equal(http.StatusOK))

			gatewayClient.AssertCalled(GinkgoT(), "UpdateStorageSpace", mock.Anything, mock.MatchedBy(func(req *provider.UpdateStorageSpaceRequest) bool {
				return req.StorageSpace.Id.OpaqueId == "spaceid" && req.StorageSpace.Quota.QuotaMaxBytes == uint64(1000)
			}))
		})
	})

	Describe("Delete a drive", func() {
		It("fails on invalid drive ids", func() {
			r := httptest.NewRequest(http.MethodDelete, "/graph/v1.0/drives/{driveID}/", nil)
			svc.DeleteDrive(rr, r)
			Expect(rr.Code).To(Equal(http.StatusBadRequest))

			r = httptest.NewRequest(http.MethodDelete, "/graph/v1.0/drives/{driveID}/", nil)
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("driveID", "")
			r = r.WithContext(context.WithValue(revactx.ContextSetUser(ctx, nil), chi.RouteCtxKey, rctx))
			svc.DeleteDrive(rr, r)
			Expect(rr.Code).To(Equal(http.StatusBadRequest))
		})

		It("deletes", func() {
			gatewayClient.On("DeleteStorageSpace", mock.Anything, mock.Anything).Return(&provider.DeleteStorageSpaceResponse{
				Status: status.NewOK(ctx),
			}, nil)

			r := httptest.NewRequest(http.MethodDelete, "/graph/v1.0/drives/{driveID}/", nil)
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("driveID", "spaceid")
			r = r.WithContext(context.WithValue(revactx.ContextSetUser(ctx, currentUser), chi.RouteCtxKey, rctx))
			svc.DeleteDrive(rr, r)
			Expect(rr.Code).To(Equal(http.StatusNoContent))

			gatewayClient.AssertCalled(GinkgoT(), "DeleteStorageSpace", mock.Anything, mock.MatchedBy(func(req *provider.DeleteStorageSpaceRequest) bool {
				return req.Id.OpaqueId == "spaceid"
			}))
		})

		It("purges", func() {
			gatewayClient.On("DeleteStorageSpace", mock.Anything, mock.Anything).Return(&provider.DeleteStorageSpaceResponse{
				Status: status.NewOK(ctx),
			}, nil)

			r := httptest.NewRequest(http.MethodDelete, "/graph/v1.0/drives/{driveID}/", nil)
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("driveID", "spaceid")
			r = r.WithContext(context.WithValue(revactx.ContextSetUser(ctx, currentUser), chi.RouteCtxKey, rctx))
			r.Header.Add("Purge", "1")
			svc.DeleteDrive(rr, r)
			Expect(rr.Code).To(Equal(http.StatusNoContent))

			gatewayClient.AssertCalled(GinkgoT(), "DeleteStorageSpace", mock.Anything, mock.MatchedBy(func(req *provider.DeleteStorageSpaceRequest) bool {
				return req.Id.OpaqueId == "spaceid" && utils.ExistsInOpaque(req.Opaque, "purge")
			}))
		})
	})
})
