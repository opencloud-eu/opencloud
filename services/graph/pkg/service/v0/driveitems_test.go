package svc_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"time"

	gateway "github.com/cs3org/go-cs3apis/cs3/gateway/v1beta1"
	userpb "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"
	provider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	"github.com/go-chi/chi/v5"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	libregraph "github.com/opencloud-eu/libre-graph-api-go"
	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc"

	revactx "github.com/opencloud-eu/reva/v2/pkg/ctx"
	"github.com/opencloud-eu/reva/v2/pkg/rgrpc/status"
	"github.com/opencloud-eu/reva/v2/pkg/rgrpc/todo/pool"
	"github.com/opencloud-eu/reva/v2/pkg/utils"
	cs3mocks "github.com/opencloud-eu/reva/v2/tests/cs3mocks/mocks"

	"github.com/opencloud-eu/opencloud/pkg/shared"
	"github.com/opencloud-eu/opencloud/services/graph/mocks"
	"github.com/opencloud-eu/opencloud/services/graph/pkg/config"
	"github.com/opencloud-eu/opencloud/services/graph/pkg/config/defaults"
	identitymocks "github.com/opencloud-eu/opencloud/services/graph/pkg/identity/mocks"
	service "github.com/opencloud-eu/opencloud/services/graph/pkg/service/v0"
)

type itemsList struct {
	Value []*libregraph.DriveItem
}

var _ = Describe("Driveitems", func() {
	var (
		svc             service.Service
		ctx             context.Context
		cfg             *config.Config
		gatewayClient   *cs3mocks.GatewayAPIClient
		gatewaySelector pool.Selectable[gateway.GatewayAPIClient]
		eventsPublisher mocks.Publisher
		identityBackend *identitymocks.Backend

		rr *httptest.ResponseRecorder

		newGroup *libregraph.Group

		currentUser = &userpb.User{
			Id: &userpb.UserId{
				OpaqueId: "user",
			},
		}
	)

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

		identityBackend = &identitymocks.Backend{}
		newGroup = libregraph.NewGroup()
		newGroup.SetMembersodataBind([]string{"/users/user1"})
		newGroup.SetId("group1")

		rr = httptest.NewRecorder()
		ctx = context.Background()

		cfg = defaults.FullDefaultConfig()
		cfg.Identity.LDAP.CACert = "" // skip the startup checks, we don't use LDAP at all in this tests
		cfg.TokenManager.JWTSecret = "loremipsum"
		cfg.Commons = &shared.Commons{}
		cfg.GRPCClientTLS = &shared.GRPCClientTLS{}

		svc, _ = service.NewService(
			service.Config(cfg),
			service.WithGatewaySelector(gatewaySelector),
			service.EventsPublisher(&eventsPublisher),
			service.WithIdentityBackend(identityBackend),
		)
	})

	Describe("GetRootDriveChildren", func() {
		It("handles ListStorageSpaces not found", func() {
			gatewayClient.On("ListStorageSpaces", mock.Anything, mock.Anything).Return(&provider.ListStorageSpacesResponse{
				Status: status.NewNotFound(ctx, "not found"),
			}, nil)

			r := httptest.NewRequest(http.MethodGet, "/graph/v1.0/me/drive/root/children", nil)
			r = r.WithContext(revactx.ContextSetUser(ctx, currentUser))
			svc.GetRootDriveChildren(rr, r)
			Expect(rr.Code).To(Equal(http.StatusNotFound))
		})

		It("handles ListStorageSpaces error", func() {
			gatewayClient.On("ListStorageSpaces", mock.Anything, mock.Anything).Return(&provider.ListStorageSpacesResponse{
				Status: status.NewInternal(ctx, "internal error"),
			}, nil)

			r := httptest.NewRequest(http.MethodGet, "/graph/v1.0/me/drive/root/children", nil)
			r = r.WithContext(revactx.ContextSetUser(ctx, currentUser))
			svc.GetRootDriveChildren(rr, r)
			Expect(rr.Code).To(Equal(http.StatusInternalServerError))
		})

		It("handles ListContainer not found", func() {
			gatewayClient.On("ListStorageSpaces", mock.Anything, mock.Anything).Return(&provider.ListStorageSpacesResponse{
				Status:        status.NewOK(ctx),
				StorageSpaces: []*provider.StorageSpace{{Owner: currentUser, Root: &provider.ResourceId{}}},
			}, nil)
			gatewayClient.On("ListContainer", mock.Anything, mock.Anything).Return(&provider.ListContainerResponse{
				Status: status.NewNotFound(ctx, "not found"),
			}, nil)

			r := httptest.NewRequest(http.MethodGet, "/graph/v1.0/me/drive/root/children", nil)
			r = r.WithContext(revactx.ContextSetUser(ctx, currentUser))
			svc.GetRootDriveChildren(rr, r)
			Expect(rr.Code).To(Equal(http.StatusNotFound))
		})

		It("handles ListContainer permission denied", func() {
			gatewayClient.On("ListStorageSpaces", mock.Anything, mock.Anything).Return(&provider.ListStorageSpacesResponse{
				Status:        status.NewOK(ctx),
				StorageSpaces: []*provider.StorageSpace{{Owner: currentUser, Root: &provider.ResourceId{}}},
			}, nil)
			gatewayClient.On("ListContainer", mock.Anything, mock.Anything).Return(&provider.ListContainerResponse{
				Status: status.NewPermissionDenied(ctx, errors.New("denied"), "denied"),
			}, nil)

			r := httptest.NewRequest(http.MethodGet, "/graph/v1.0/me/drive/root/children", nil)
			r = r.WithContext(revactx.ContextSetUser(ctx, currentUser))
			svc.GetRootDriveChildren(rr, r)
			Expect(rr.Code).To(Equal(http.StatusForbidden))
		})

		It("handles ListContainer error", func() {
			gatewayClient.On("ListStorageSpaces", mock.Anything, mock.Anything).Return(&provider.ListStorageSpacesResponse{
				Status:        status.NewOK(ctx),
				StorageSpaces: []*provider.StorageSpace{{Owner: currentUser, Root: &provider.ResourceId{}}},
			}, nil)
			gatewayClient.On("ListContainer", mock.Anything, mock.Anything).Return(&provider.ListContainerResponse{
				Status: status.NewInternal(ctx, "internal"),
			}, nil)

			r := httptest.NewRequest(http.MethodGet, "/graph/v1.0/me/drive/root/children", nil)
			r = r.WithContext(revactx.ContextSetUser(ctx, currentUser))
			svc.GetRootDriveChildren(rr, r)
			Expect(rr.Code).To(Equal(http.StatusInternalServerError))
		})

		It("succeeds", func() {
			mtime := time.Now()
			gatewayClient.On("ListStorageSpaces", mock.Anything, mock.Anything).Return(&provider.ListStorageSpacesResponse{
				Status:        status.NewOK(ctx),
				StorageSpaces: []*provider.StorageSpace{{Owner: currentUser, Root: &provider.ResourceId{}}},
			}, nil)
			gatewayClient.On("ListContainer", mock.Anything, mock.Anything).Return(&provider.ListContainerResponse{
				Status: status.NewOK(ctx),
				Infos: []*provider.ResourceInfo{
					{
						Type:  provider.ResourceType_RESOURCE_TYPE_FILE,
						Id:    &provider.ResourceId{StorageId: "storageid", SpaceId: "spaceid", OpaqueId: "opaqueid"},
						Etag:  "etag",
						Mtime: utils.TimeToTS(mtime),
					},
				},
			}, nil)
			r := httptest.NewRequest(http.MethodGet, "/graph/v1.0/me/drive/root/children", nil)
			r = r.WithContext(revactx.ContextSetUser(ctx, currentUser))
			svc.GetRootDriveChildren(rr, r)
			Expect(rr.Code).To(Equal(http.StatusOK))
			data, err := io.ReadAll(rr.Body)
			Expect(err).ToNot(HaveOccurred())

			res := itemsList{}

			err = json.Unmarshal(data, &res)
			Expect(err).ToNot(HaveOccurred())

			Expect(len(res.Value)).To(Equal(1))
			Expect(res.Value[0].GetLastModifiedDateTime().Equal(mtime)).To(BeTrue())
			Expect(res.Value[0].GetETag()).To(Equal("etag"))
			Expect(res.Value[0].GetId()).To(Equal("storageid$spaceid!opaqueid"))
		})
	})

	Describe("GetDriveItemChildren", func() {
		It("handles ListContainer not found", func() {
			gatewayClient.On("ListContainer", mock.Anything, mock.Anything).Return(&provider.ListContainerResponse{
				Status: status.NewNotFound(ctx, "not found"),
			}, nil)

			r := httptest.NewRequest(http.MethodGet, "/graph/v1.0/drives/storageid$spaceid/items/storageid$spaceid!nodeid/children", nil)
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("driveID", "storageid$spaceid")
			rctx.URLParams.Add("driveItemID", "storageid$spaceid!nodeid")
			r = r.WithContext(context.WithValue(revactx.ContextSetUser(ctx, currentUser), chi.RouteCtxKey, rctx))
			svc.GetDriveItemChildren(rr, r)
			Expect(rr.Code).To(Equal(http.StatusNotFound))
		})

		It("handles ListContainer permission denied as not found", func() {
			gatewayClient.On("ListContainer", mock.Anything, mock.Anything).Return(&provider.ListContainerResponse{
				Status: status.NewPermissionDenied(ctx, errors.New("denied"), "denied"),
			}, nil)

			r := httptest.NewRequest(http.MethodGet, "/graph/v1.0/drives/storageid$spaceid/items/storageid$spaceid!nodeid/children", nil)
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("driveID", "storageid$spaceid")
			rctx.URLParams.Add("driveItemID", "storageid$spaceid!nodeid")
			r = r.WithContext(context.WithValue(revactx.ContextSetUser(ctx, currentUser), chi.RouteCtxKey, rctx))
			svc.GetDriveItemChildren(rr, r)
			Expect(rr.Code).To(Equal(http.StatusNotFound))
		})

		It("handles ListContainer error", func() {
			gatewayClient.On("ListContainer", mock.Anything, mock.Anything).Return(&provider.ListContainerResponse{
				Status: status.NewInternal(ctx, "internal"),
			}, nil)

			r := httptest.NewRequest(http.MethodGet, "/graph/v1.0/drives/storageid$spaceid/items/storageid$spaceid!nodeid/children", nil)
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("driveID", "storageid$spaceid")
			rctx.URLParams.Add("driveItemID", "storageid$spaceid!nodeid")
			r = r.WithContext(context.WithValue(revactx.ContextSetUser(ctx, currentUser), chi.RouteCtxKey, rctx))
			svc.GetDriveItemChildren(rr, r)
			Expect(rr.Code).To(Equal(http.StatusInternalServerError))
		})

		Context("it succeeds", func() {
			var (
				r     *http.Request
				mtime = time.Now()
			)

			BeforeEach(func() {
				r = httptest.NewRequest(http.MethodGet, "/graph/v1.0/drives/storageid$spaceid/items/storageid$spaceid!nodeid/children", nil)
				rctx := chi.NewRouteContext()
				rctx.URLParams.Add("driveID", "storageid$spaceid")
				rctx.URLParams.Add("driveItemID", "storageid$spaceid!nodeid")
				r = r.WithContext(context.WithValue(revactx.ContextSetUser(ctx, currentUser), chi.RouteCtxKey, rctx))
			})

			assertItemsList := func(length int) itemsList {
				svc.GetDriveItemChildren(rr, r)
				Expect(rr.Code).To(Equal(http.StatusOK))
				data, err := io.ReadAll(rr.Body)
				Expect(err).ToNot(HaveOccurred())

				res := itemsList{}

				err = json.Unmarshal(data, &res)
				Expect(err).ToNot(HaveOccurred())

				Expect(len(res.Value)).To(Equal(length))
				Expect(res.Value[0].GetLastModifiedDateTime().Equal(mtime)).To(BeTrue())
				Expect(res.Value[0].GetETag()).To(Equal("etag"))
				Expect(res.Value[0].GetId()).To(Equal("storageid$spaceid!opaqueid"))
				Expect(res.Value[0].GetId()).To(Equal("storageid$spaceid!opaqueid"))

				return res
			}

			It("returns a generic file", func() {
				gatewayClient.On("ListContainer", mock.Anything, mock.Anything).Return(&provider.ListContainerResponse{
					Status: status.NewOK(ctx),
					Infos: []*provider.ResourceInfo{
						{
							Type:              provider.ResourceType_RESOURCE_TYPE_FILE,
							Id:                &provider.ResourceId{StorageId: "storageid", SpaceId: "spaceid", OpaqueId: "opaqueid"},
							Etag:              "etag",
							Mtime:             utils.TimeToTS(mtime),
							ArbitraryMetadata: nil,
						},
					},
				}, nil)

				res := assertItemsList(1)
				Expect(res.Value[0].Audio).To(BeNil())
				Expect(res.Value[0].Location).To(BeNil())
			})

			It("returns the audio facet if metadata is available", func() {
				gatewayClient.On("ListContainer", mock.Anything, mock.Anything).Return(&provider.ListContainerResponse{
					Status: status.NewOK(ctx),
					Infos: []*provider.ResourceInfo{
						{
							Type:     provider.ResourceType_RESOURCE_TYPE_FILE,
							Id:       &provider.ResourceId{StorageId: "storageid", SpaceId: "spaceid", OpaqueId: "opaqueid"},
							Etag:     "etag",
							Mtime:    utils.TimeToTS(mtime),
							MimeType: "audio/mpeg",
							ArbitraryMetadata: &provider.ArbitraryMetadata{
								Metadata: map[string]string{
									"libre.graph.audio.album":             "Some Album",
									"libre.graph.audio.albumArtist":       "Some AlbumArtist",
									"libre.graph.audio.artist":            "Some Artist",
									"libre.graph.audio.bitrate":           "192",
									"libre.graph.audio.composers":         "Some Composers",
									"libre.graph.audio.copyright":         "Some Copyright",
									"libre.graph.audio.disc":              "2",
									"libre.graph.audio.discCount":         "5",
									"libre.graph.audio.duration":          "225000",
									"libre.graph.audio.genre":             "Some Genre",
									"libre.graph.audio.hasDrm":            "false",
									"libre.graph.audio.isVariableBitrate": "true",
									"libre.graph.audio.title":             "Some Title",
									"libre.graph.audio.track":             "6",
									"libre.graph.audio.trackCount":        "9",
									"libre.graph.audio.year":              "1994",
								},
							},
						},
					},
				}, nil)

				res := assertItemsList(1)
				audio := res.Value[0].Audio

				Expect(audio).ToNot(BeNil())
				Expect(audio.Album).To(Equal(libregraph.PtrString("Some Album")))
				Expect(audio.AlbumArtist).To(Equal(libregraph.PtrString("Some AlbumArtist")))
				Expect(audio.Artist).To(Equal(libregraph.PtrString("Some Artist")))
				Expect(audio.Bitrate).To(Equal(libregraph.PtrInt64(192)))
				Expect(audio.Composers).To(Equal(libregraph.PtrString("Some Composers")))
				Expect(audio.Copyright).To(Equal(libregraph.PtrString("Some Copyright")))
				Expect(audio.Disc).To(Equal(libregraph.PtrInt32(2)))
				Expect(audio.DiscCount).To(Equal(libregraph.PtrInt32(5)))
				Expect(audio.Duration).To(Equal(libregraph.PtrInt64(225000)))
				Expect(audio.Genre).To(Equal(libregraph.PtrString("Some Genre")))
				Expect(audio.HasDrm).To(Equal(libregraph.PtrBool(false)))
				Expect(audio.IsVariableBitrate).To(Equal(libregraph.PtrBool(true)))
				Expect(audio.Title).To(Equal(libregraph.PtrString("Some Title")))
				Expect(audio.Track).To(Equal(libregraph.PtrInt32(6)))
				Expect(audio.TrackCount).To(Equal(libregraph.PtrInt32(9)))
				Expect(audio.Year).To(Equal(libregraph.PtrInt32(1994)))
			})

			It("returns the location facet if metadata is available", func() {
				gatewayClient.On("ListContainer", mock.Anything, mock.Anything).Return(&provider.ListContainerResponse{
					Status: status.NewOK(ctx),
					Infos: []*provider.ResourceInfo{
						{
							Type:     provider.ResourceType_RESOURCE_TYPE_FILE,
							Id:       &provider.ResourceId{StorageId: "storageid", SpaceId: "spaceid", OpaqueId: "opaqueid"},
							Etag:     "etag",
							Mtime:    utils.TimeToTS(mtime),
							MimeType: "image/jpeg",
							ArbitraryMetadata: &provider.ArbitraryMetadata{
								Metadata: map[string]string{
									"libre.graph.location.altitude":  "1047.7",
									"libre.graph.location.latitude":  "49.48675890884328",
									"libre.graph.location.longitude": "11.103870357204285",
								},
							},
						},
					},
				}, nil)

				res := assertItemsList(1)
				location := res.Value[0].Location

				Expect(location).ToNot(BeNil())
				Expect(location.Altitude).To(Equal(libregraph.PtrFloat64(1047.7)))
				Expect(location.Latitude).To(Equal(libregraph.PtrFloat64(49.48675890884328)))
				Expect(location.Longitude).To(Equal(libregraph.PtrFloat64(11.103870357204285)))
			})

			It("returns the image facet if metadata is available", func() {
				gatewayClient.On("ListContainer", mock.Anything, mock.Anything).Return(&provider.ListContainerResponse{
					Status: status.NewOK(ctx),
					Infos: []*provider.ResourceInfo{
						{
							Type:     provider.ResourceType_RESOURCE_TYPE_FILE,
							Id:       &provider.ResourceId{StorageId: "storageid", SpaceId: "spaceid", OpaqueId: "opaqueid"},
							Etag:     "etag",
							Mtime:    utils.TimeToTS(mtime),
							MimeType: "image/jpeg",
							ArbitraryMetadata: &provider.ArbitraryMetadata{
								Metadata: map[string]string{
									"libre.graph.image.width":  "1234",
									"libre.graph.image.height": "987",
								},
							},
						},
					},
				}, nil)

				res := assertItemsList(1)
				image := res.Value[0].Image

				Expect(image).ToNot(BeNil())
				Expect(image.Width).To(Equal(libregraph.PtrInt32(1234)))
				Expect(image.Height).To(Equal(libregraph.PtrInt32(987)))
			})

			It("returns the photo facet if metadata is available", func() {
				gatewayClient.On("ListContainer", mock.Anything, mock.Anything).Return(&provider.ListContainerResponse{
					Status: status.NewOK(ctx),
					Infos: []*provider.ResourceInfo{
						{
							Type:     provider.ResourceType_RESOURCE_TYPE_FILE,
							Id:       &provider.ResourceId{StorageId: "storageid", SpaceId: "spaceid", OpaqueId: "opaqueid"},
							Etag:     "etag",
							Mtime:    utils.TimeToTS(mtime),
							MimeType: "image/jpeg",
							ArbitraryMetadata: &provider.ArbitraryMetadata{
								Metadata: map[string]string{
									"libre.graph.photo.cameraMake":          "Canon",
									"libre.graph.photo.cameraModel":         "Cannon EOS 5D Mark III",
									"libre.graph.photo.exposureDenominator": "100",
									"libre.graph.photo.exposureNumerator":   "1",
									"libre.graph.photo.fNumber":             "1.8",
									"libre.graph.photo.focalLength":         "50",
									"libre.graph.photo.iso":                 "400",
									"libre.graph.photo.orientation":         "1",
									"libre.graph.photo.takenDateTime":       "2018-01-01T12:34:56Z",
								},
							},
						},
					},
				}, nil)

				res := assertItemsList(1)
				photo := res.Value[0].Photo

				Expect(photo).ToNot(BeNil())
				Expect(photo.CameraMake).To(Equal(libregraph.PtrString("Canon")))
				Expect(photo.CameraModel).To(Equal(libregraph.PtrString("Cannon EOS 5D Mark III")))
				Expect(photo.ExposureDenominator).To(Equal(libregraph.PtrFloat64(100)))
				Expect(photo.ExposureNumerator).To(Equal(libregraph.PtrFloat64(1)))
				Expect(photo.FNumber).To(Equal(libregraph.PtrFloat64(1.8)))
				Expect(photo.FocalLength).To(Equal(libregraph.PtrFloat64(50)))
				Expect(photo.Iso).To(Equal(libregraph.PtrInt32(400)))
				Expect(photo.Orientation).To(Equal(libregraph.PtrInt32(1)))
				Expect(photo.TakenDateTime).To(Equal(libregraph.PtrTime(time.Date(2018, 1, 1, 12, 34, 56, 0, time.UTC))))
			})
		})
	})
})
