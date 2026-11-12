package svc_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"time"

	gateway "github.com/cs3org/go-cs3apis/cs3/gateway/v1beta1"
	grouppb "github.com/cs3org/go-cs3apis/cs3/identity/group/v1beta1"
	userpb "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"
	collaboration "github.com/cs3org/go-cs3apis/cs3/sharing/collaboration/v1beta1"
	link "github.com/cs3org/go-cs3apis/cs3/sharing/link/v1beta1"
	ocm "github.com/cs3org/go-cs3apis/cs3/sharing/ocm/v1beta1"
	provider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	"github.com/go-chi/chi/v5/middleware"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	libregraph "github.com/opencloud-eu/libre-graph-api-go"
	"github.com/opencloud-eu/reva/v2/pkg/conversions"
	revactx "github.com/opencloud-eu/reva/v2/pkg/ctx"
	"github.com/opencloud-eu/reva/v2/pkg/rgrpc/status"
	"github.com/opencloud-eu/reva/v2/pkg/rgrpc/todo/pool"
	"github.com/opencloud-eu/reva/v2/pkg/storagespace"
	"github.com/opencloud-eu/reva/v2/pkg/utils"
	cs3mocks "github.com/opencloud-eu/reva/v2/tests/cs3mocks/mocks"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc"

	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/pkg/shared"
	"github.com/opencloud-eu/opencloud/services/graph/mocks"
	"github.com/opencloud-eu/opencloud/services/graph/pkg/config"
	"github.com/opencloud-eu/opencloud/services/graph/pkg/config/defaults"
	"github.com/opencloud-eu/opencloud/services/graph/pkg/errorcode"
	identitymocks "github.com/opencloud-eu/opencloud/services/graph/pkg/identity/mocks"
	"github.com/opencloud-eu/opencloud/services/graph/pkg/linktype"
	"github.com/opencloud-eu/opencloud/services/graph/pkg/metrics"
	service "github.com/opencloud-eu/opencloud/services/graph/pkg/service/v0"
	"github.com/opencloud-eu/opencloud/services/graph/pkg/unifiedrole"
)

var _ = Describe("sharedbyme", func() {
	var (
		svc                 service.Service
		ctx                 context.Context
		cfg                 *config.Config
		gatewayClient       *cs3mocks.GatewayAPIClient
		gatewaySelector     pool.Selectable[gateway.GatewayAPIClient]
		eventsPublisher     mocks.Publisher
		identityBackend     *identitymocks.Backend
		driveItemCreateLink *libregraph.DriveItemCreateLink
		publicShare         link.PublicShare

		rr *httptest.ResponseRecorder
	)
	expiration := time.Now()

	editorResourcePermissions := conversions.NewEditorRole().CS3ResourcePermissions()
	userShare := collaboration.Share{
		Id: &collaboration.ShareId{
			OpaqueId: "share-id",
		},
		ResourceId: &provider.ResourceId{
			StorageId: "storageid",
			SpaceId:   "spaceid",
			OpaqueId:  "opaqueid",
		},
		Grantee: &provider.Grantee{
			Type: provider.GranteeType_GRANTEE_TYPE_USER,
			Id: &provider.Grantee_UserId{
				UserId: &userpb.UserId{
					OpaqueId: "user-id",
				},
			},
		},
		Permissions: &collaboration.SharePermissions{
			Permissions: editorResourcePermissions,
		},
	}
	groupShare := collaboration.Share{
		Id: &collaboration.ShareId{
			OpaqueId: "share-id",
		},
		ResourceId: &provider.ResourceId{
			StorageId: "storageid",
			SpaceId:   "spaceid",
			OpaqueId:  "opaqueid",
		},
		Grantee: &provider.Grantee{
			Type: provider.GranteeType_GRANTEE_TYPE_GROUP,
			Id: &provider.Grantee_GroupId{
				GroupId: &grouppb.GroupId{
					OpaqueId: "group-id",
				},
			},
		},
		Permissions: &collaboration.SharePermissions{
			Permissions: editorResourcePermissions,
		},
	}
	userShareWithExpiration := collaboration.Share{
		Id: &collaboration.ShareId{
			OpaqueId: "expire-share-id",
		},
		ResourceId: &provider.ResourceId{
			StorageId: "storageid",
			SpaceId:   "spaceid",
			OpaqueId:  "expire-opaqueid",
		},
		Grantee: &provider.Grantee{
			Type: provider.GranteeType_GRANTEE_TYPE_USER,
			Id: &provider.Grantee_UserId{
				UserId: &userpb.UserId{
					OpaqueId: "user-id",
				},
			},
		},
		Permissions: &collaboration.SharePermissions{
			Permissions: editorResourcePermissions,
		},
		Expiration: utils.TimeToTS(expiration),
	}
	driveItemCreateLink = &libregraph.DriveItemCreateLink{}

	BeforeEach(func() {
		eventsPublisher.On("Publish", mock.Anything, mock.Anything, mock.Anything).Return(nil)
		linkType, err := libregraph.NewSharingLinkTypeFromValue("view")
		Expect(err).To(BeNil())
		driveItemCreateLink.Type = linkType
		driveItemCreateLink.ExpirationDateTime = libregraph.PtrTime(time.Now().Add(time.Hour))
		permissions, err := linktype.CS3ResourcePermissionsFromSharingLink(*driveItemCreateLink, provider.ResourceType_RESOURCE_TYPE_CONTAINER)
		Expect(err).To(BeNil())

		publicShare = link.PublicShare{
			Id: &link.PublicShareId{
				OpaqueId: "public-share-id",
			},
			Token: "public-share-token",
			ResourceId: &provider.ResourceId{
				StorageId: "storageid",
				SpaceId:   "spaceid",
				OpaqueId:  "public-share-opaqueid",
			},
			Permissions: &link.PublicSharePermissions{Permissions: permissions},
		}

		rr = httptest.NewRecorder()

		pool.RemoveSelector("GatewaySelector" + "eu.opencloud.api.gateway")
		gatewayClient = &cs3mocks.GatewayAPIClient{}

		gatewayClient.On("Stat",
			mock.Anything,
			mock.MatchedBy(
				func(req *provider.StatRequest) bool {
					return req.Ref.ResourceId.OpaqueId == userShareWithExpiration.ResourceId.OpaqueId
				})).
			Return(&provider.StatResponse{
				Status: status.NewOK(ctx),
				Info: &provider.ResourceInfo{
					Id: userShareWithExpiration.ResourceId,
				},
			}, nil)
		gatewayClient.On("Stat",
			mock.Anything,
			mock.MatchedBy(
				func(req *provider.StatRequest) bool {
					return req.Ref.ResourceId.OpaqueId == publicShare.ResourceId.OpaqueId
				})).
			Return(&provider.StatResponse{
				Status: status.NewOK(ctx),
				Info: &provider.ResourceInfo{
					Id: publicShare.ResourceId,
				},
			}, nil)
		gatewayClient.On("Stat",
			mock.Anything,
			mock.Anything).
			Return(&provider.StatResponse{
				Status: status.NewOK(ctx),
				Info: &provider.ResourceInfo{
					Id:   userShare.ResourceId,
					Type: provider.ResourceType_RESOURCE_TYPE_CONTAINER,
				},
			}, nil)

		gatewayClient.On("GetUser",
			mock.Anything,
			mock.MatchedBy(func(req *userpb.GetUserRequest) bool {
				return req.UserId.OpaqueId == "user-id"
			})).
			Return(&userpb.GetUserResponse{
				Status: status.NewOK(ctx),
				User: &userpb.User{
					Id: &userpb.UserId{
						Idp:      "idp",
						OpaqueId: "user-id",
					},
					DisplayName: "User Name",
				},
			}, nil)
		gatewayClient.On("GetUser",
			mock.Anything,
			mock.Anything).
			Return(&userpb.GetUserResponse{
				Status: status.NewNotFound(ctx, "mock user not found"),
				User:   nil,
			}, nil)
		gatewayClient.On("GetGroup",
			mock.Anything,
			mock.MatchedBy(func(req *grouppb.GetGroupRequest) bool {
				return req.GroupId.OpaqueId == "group-id"
			})).
			Return(&grouppb.GetGroupResponse{
				Status: status.NewOK(ctx),
				Group: &grouppb.Group{
					Id: &grouppb.GroupId{
						Idp:      "idp",
						OpaqueId: "group-id",
					},
					DisplayName: "Group Name",
				},
			}, nil)
		gatewayClient.On("GetGroup",
			mock.Anything,
			mock.Anything).
			Return(&grouppb.GetGroupResponse{
				Status: status.NewNotFound(ctx, "mock group not found"),
				Group:  nil,
			}, nil)

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
		cfg.Identity.LDAP.CACert = "" // skip the startup checks, we don't use LDAP at all in this tests
		cfg.TokenManager.JWTSecret = "loremipsum"
		cfg.Commons = &shared.Commons{}
		cfg.GRPCClientTLS = &shared.GRPCClientTLS{}

		svc, err = service.NewService(
			service.Config(cfg),
			service.Metrics(metrics),
			service.WithGatewaySelector(gatewaySelector),
			service.EventsPublisher(&eventsPublisher),
			service.WithIdentityBackend(identityBackend),
		)
		Expect(err).ToNot(HaveOccurred())
	})

	emptyListPublicSharesMock := func() {
		gatewayClient.On("ListPublicShares", mock.Anything, mock.Anything).Return(
			&link.ListPublicSharesResponse{
				Status: status.NewOK(ctx),
				Share:  []*link.PublicShare{},
			},
			nil,
		)
	}
	emptyListSharesMock := func() {
		gatewayClient.On("ListShares", mock.Anything, mock.Anything).Return(
			&collaboration.ListSharesResponse{
				Status: status.NewOK(ctx),
				Shares: []*collaboration.Share{},
			},
			nil,
		)
	}
	Describe("GetSharedByMe logging", func() {
		var logBuf *bytes.Buffer

		newLoggedService := func(selector pool.Selectable[gateway.GatewayAPIClient]) service.Service {
			logBuf = &bytes.Buffer{}
			logger := log.NewLogger()
			m, err := metrics.New(prometheus.NewRegistry(), &logger, func([]string) (string, string) { return "", "" })
			Expect(err).ToNot(HaveOccurred())
			s, err := service.NewService(
				service.Config(cfg),
				service.Metrics(m),
				service.WithGatewaySelector(selector),
				service.EventsPublisher(&eventsPublisher),
				service.WithIdentityBackend(identityBackend),
				service.Logger(log.Logger{Logger: zerolog.New(logBuf)}),
			)
			Expect(err).ToNot(HaveOccurred())
			return s
		}
		errorEntries := func() []map[string]any {
			return entriesAtLevel(logEntries(logBuf), "error")
		}
		odataError := func() libregraph.OdataError {
			var odataErr libregraph.OdataError
			Expect(json.Unmarshal(rr.Body.Bytes(), &odataErr)).To(Succeed())
			return odataErr
		}

		It("logs a gateway selection failure while listing shares as error in the helper and the operation", func() {
			failingSelector := mocks.NewSelectable[gateway.GatewayAPIClient](GinkgoT())
			failingSelector.EXPECT().Next().Return(nil, errors.New("no gateway available"))
			s := newLoggedService(failingSelector)

			s.GetSharedByMe(rr, httptest.NewRequest(http.MethodGet, "/graph/v1.0/me/drives/sharedByMe", nil))

			Expect(rr.Code).To(Equal(http.StatusInternalServerError))
			Expect(odataError().Error.Code).To(Equal("generalException"))
			Expect(odataError().Error.Message).To(Equal("no gateway available"))
			// the helper and the handler both log the reason, the handler with the error code
			Expect(errorEntries()).To(ConsistOf(
				gatewayHelperEntry("could not select next gateway client", "no gateway available"),
				SatisfyAll(
					HaveKeyWithValue("message", "could not list user shares"),
					HaveKeyWithValue("errorCode", "generalException"),
					HaveKeyWithValue("error", errorcode.New(errorcode.GeneralException, "no gateway available").Error()),
				),
			))
		})

		It("logs a share listing failure once as error", func() {
			emptyListPublicSharesMock()
			gatewayClient.On("ListShares", mock.Anything, mock.Anything).Return(
				&collaboration.ListSharesResponse{Status: status.NewInternal(ctx, "error listing shares")},
				nil,
			)
			s := newLoggedService(gatewaySelector)

			r := httptest.NewRequest(http.MethodGet, "/graph/v1.0/me/drives/sharedByMe", nil)
			reqCtx := context.WithValue(r.Context(), middleware.RequestIDKey, "request-42")
			reqCtx = revactx.ContextSetUser(reqCtx, &userpb.User{Id: &userpb.UserId{OpaqueId: "user-42"}})
			s.GetSharedByMe(rr, r.WithContext(reqCtx))

			Expect(rr.Code).To(Equal(http.StatusInternalServerError))
			Expect(odataError().Error.Code).To(Equal("generalException"))
			Expect(odataError().Error.Message).To(Equal("error listing shares"))
			entries := errorEntries()
			Expect(entries).To(HaveLen(1))
			Expect(entries[0]).To(SatisfyAll(
				HaveKeyWithValue("message", "could not list user shares"),
				HaveKeyWithValue("error", errorcode.New(errorcode.GeneralException, "error listing shares").Error()),
				HaveKeyWithValue("errorCode", "generalException"),
				HaveKeyWithValue("request-id", "request-42"),
				HaveKeyWithValue("userID", "user-42"),
			))
		})

		DescribeTable("logs the identity backend error when a share grantee lookup fails",
			func(setup func(identityClient *cs3mocks.GatewayAPIClient, resourceID *provider.ResourceId, groupGrantee *provider.Grantee), listingMethod string) {
				const identityBackendMessage = "ldap search failed for cn=private-group-dn-42,ou=groups"
				identityClient := cs3mocks.NewGatewayAPIClient(GinkgoT())
				selector := mocks.NewSelectable[gateway.GatewayAPIClient](GinkgoT())
				selector.EXPECT().Next().Return(identityClient, nil)
				resourceID := &provider.ResourceId{StorageId: "storageid", SpaceId: "spaceid", OpaqueId: "opaqueid"}
				groupGrantee := &provider.Grantee{
					Type: provider.GranteeType_GRANTEE_TYPE_GROUP,
					Id:   &provider.Grantee_GroupId{GroupId: &grouppb.GroupId{OpaqueId: "groupid"}},
				}
				setup(identityClient, resourceID, groupGrantee)
				identityClient.On("Stat", mock.Anything, mock.Anything).Return(&provider.StatResponse{
					Status: status.NewOK(ctx),
					Info:   &provider.ResourceInfo{Id: resourceID, Type: provider.ResourceType_RESOURCE_TYPE_CONTAINER},
				}, nil)
				identityClient.On("GetGroup", mock.Anything, mock.Anything).Return(&grouppb.GetGroupResponse{
					Status: status.NewInternal(ctx, identityBackendMessage),
				}, nil)
				s := newLoggedService(selector)

				s.GetSharedByMe(rr, httptest.NewRequest(http.MethodGet, "/graph/v1.0/me/drives/sharedByMe", nil))

				// the rendered error is unchanged
				Expect(rr.Code).To(Equal(http.StatusInternalServerError))
				Expect(odataError().Error.Code).To(Equal("generalException"))
				Expect(odataError().Error.Message).To(Equal(
					errorcode.New(errorcode.GeneralException,
						errorcode.New(errorcode.GeneralException, identityBackendMessage).Error()).Error()))
				// the shares were listed and the group grantee was looked up, listing stops before the public shares
				identityClient.AssertCalled(GinkgoT(), listingMethod, mock.Anything, mock.Anything)
				identityClient.AssertCalled(GinkgoT(), "GetGroup", mock.Anything, mock.MatchedBy(func(req *grouppb.GetGroupRequest) bool {
					return req.GetGroupId().GetOpaqueId() == "groupid"
				}))
				identityClient.AssertNotCalled(GinkgoT(), "ListPublicShares", mock.Anything, mock.Anything)

				entries := logEntries(logBuf)
				errorEntries := entriesAtLevel(entries, "error")
				Expect(errorEntries).To(HaveLen(1))
				// the handler logs the error text, which carries the identity backend message
				Expect(errorEntries[0]).To(HaveKeyWithValue("error", ContainSubstring(identityBackendMessage)))
				for _, payloadKey := range []string{"shares", "share", "group", "res", "response"} {
					Expect(entries).ToNot(ContainElement(HaveKey(payloadKey)))
				}
			},
			Entry("user share listing",
				func(identityClient *cs3mocks.GatewayAPIClient, resourceID *provider.ResourceId, groupGrantee *provider.Grantee) {
					identityClient.On("ListShares", mock.Anything, mock.Anything).Return(&collaboration.ListSharesResponse{
						Status: status.NewOK(ctx),
						Shares: []*collaboration.Share{{
							Id:          &collaboration.ShareId{OpaqueId: "groupshareid"},
							ResourceId:  resourceID,
							Grantee:     groupGrantee,
							Permissions: &collaboration.SharePermissions{Permissions: editorResourcePermissions},
						}},
					}, nil)
				},
				"ListShares",
			),
			Entry("ocm share listing",
				func(identityClient *cs3mocks.GatewayAPIClient, resourceID *provider.ResourceId, groupGrantee *provider.Grantee) {
					cfg.IncludeOCMSharees = true
					identityClient.On("ListShares", mock.Anything, mock.Anything).Return(&collaboration.ListSharesResponse{
						Status: status.NewOK(ctx),
						Shares: []*collaboration.Share{},
					}, nil)
					identityClient.On("ListOCMShares", mock.Anything, mock.Anything).Return(&ocm.ListOCMSharesResponse{
						Status: status.NewOK(ctx),
						Shares: []*ocm.Share{{
							Id:         &ocm.ShareId{OpaqueId: "ocmgroupshareid"},
							ResourceId: resourceID,
							Grantee:    groupGrantee,
						}},
					}, nil)
				},
				"ListOCMShares",
			),
		)

		It("logs an ocm share listing failure once as error", func() {
			cfg.IncludeOCMSharees = true
			emptyListSharesMock()
			gatewayClient.On("ListOCMShares", mock.Anything, mock.Anything).Return(
				&ocm.ListOCMSharesResponse{Status: status.NewInternal(ctx, "error listing ocm shares")},
				nil,
			)
			s := newLoggedService(gatewaySelector)

			s.GetSharedByMe(rr, httptest.NewRequest(http.MethodGet, "/graph/v1.0/me/drives/sharedByMe", nil))

			Expect(rr.Code).To(Equal(http.StatusInternalServerError))
			Expect(odataError().Error.Code).To(Equal("generalException"))
			Expect(odataError().Error.Message).To(Equal("error listing ocm shares"))
			gatewayClient.AssertNotCalled(GinkgoT(), "ListPublicShares", mock.Anything, mock.Anything)
			entries := errorEntries()
			Expect(entries).To(HaveLen(1))
			Expect(entries[0]).To(SatisfyAll(
				HaveKeyWithValue("message", "could not list ocm shares"),
				HaveKeyWithValue("error", errorcode.New(errorcode.GeneralException, "error listing ocm shares").Error()),
			))
		})

		It("logs a gateway selection failure while listing ocm shares as error in the helper and the operation", func() {
			cfg.IncludeOCMSharees = true
			emptyListSharesMock()
			// the user share listing gets a client, the ocm share listing does not
			selector := mocks.NewSelectable[gateway.GatewayAPIClient](GinkgoT())
			selector.EXPECT().Next().Return(gatewayClient, nil).Times(1)
			selector.EXPECT().Next().Return(nil, errors.New("no gateway available")).Times(1)
			s := newLoggedService(selector)

			s.GetSharedByMe(rr, httptest.NewRequest(http.MethodGet, "/graph/v1.0/me/drives/sharedByMe", nil))

			Expect(rr.Code).To(Equal(http.StatusInternalServerError))
			Expect(odataError().Error.Code).To(Equal("generalException"))
			Expect(odataError().Error.Message).To(Equal("no gateway available"))
			gatewayClient.AssertCalled(GinkgoT(), "ListShares", mock.Anything, mock.Anything)
			gatewayClient.AssertNotCalled(GinkgoT(), "ListOCMShares", mock.Anything, mock.Anything)
			// the helper and the handler both log the reason, the handler with the error code
			Expect(errorEntries()).To(ConsistOf(
				gatewayHelperEntry("could not select next gateway client", "no gateway available"),
				SatisfyAll(
					HaveKeyWithValue("message", "could not list ocm shares"),
					HaveKeyWithValue("error", errorcode.New(errorcode.GeneralException, "no gateway available").Error()),
				),
			))
		})

		It("logs a public share listing failure once as error", func() {
			emptyListSharesMock()
			gatewayClient.On("ListPublicShares", mock.Anything, mock.Anything).Return(
				&link.ListPublicSharesResponse{Status: status.NewInternal(ctx, "listing failed for /Users/alice/Private/salary-2026.ods")},
				nil,
			)
			s := newLoggedService(gatewaySelector)

			s.GetSharedByMe(rr, httptest.NewRequest(http.MethodGet, "/graph/v1.0/me/drives/sharedByMe", nil))

			Expect(rr.Code).To(Equal(http.StatusInternalServerError))
			Expect(odataError().Error.Code).To(Equal("generalException"))
			// the response is unchanged
			Expect(odataError().Error.Message).To(Equal("listing failed for /Users/alice/Private/salary-2026.ods"))
			gatewayClient.AssertCalled(GinkgoT(), "ListShares", mock.Anything, mock.Anything)
			entries := errorEntries()
			Expect(entries).To(HaveLen(1))
			Expect(entries[0]).To(SatisfyAll(
				HaveKeyWithValue("message", "could not list public shares"),
				HaveKeyWithValue("errorCode", "generalException"),
				HaveKeyWithValue("error", errorcode.New(errorcode.GeneralException, "listing failed for /Users/alice/Private/salary-2026.ods").Error()),
			))
		})

		It("logs a gateway selection failure while listing public shares as error in the helper and the operation", func() {
			emptyListSharesMock()
			// the user share listing gets a client, the public share listing does not
			selector := mocks.NewSelectable[gateway.GatewayAPIClient](GinkgoT())
			selector.EXPECT().Next().Return(gatewayClient, nil).Times(1)
			selector.EXPECT().Next().Return(nil, errors.New("no gateway available")).Times(1)
			s := newLoggedService(selector)

			s.GetSharedByMe(rr, httptest.NewRequest(http.MethodGet, "/graph/v1.0/me/drives/sharedByMe", nil))

			Expect(rr.Code).To(Equal(http.StatusInternalServerError))
			Expect(odataError().Error.Code).To(Equal("generalException"))
			Expect(odataError().Error.Message).To(Equal("no gateway available"))
			gatewayClient.AssertCalled(GinkgoT(), "ListShares", mock.Anything, mock.Anything)
			gatewayClient.AssertNotCalled(GinkgoT(), "ListPublicShares", mock.Anything, mock.Anything)
			// the helper and the handler both log the reason, the handler with the error code
			Expect(errorEntries()).To(ConsistOf(
				gatewayHelperEntry("could not select next gateway client", "no gateway available"),
				SatisfyAll(
					HaveKeyWithValue("message", "could not list public shares"),
					HaveKeyWithValue("error", errorcode.New(errorcode.GeneralException, "no gateway available").Error()),
				),
			))
		})
	})

	Describe("GetSharedByMe", func() {
		It("handles a failing ListShares", func() {
			emptyListPublicSharesMock()
			gatewayClient.On("ListShares", mock.Anything, mock.Anything).Return(nil, errors.New("some error"))
			r := httptest.NewRequest(http.MethodGet, "/graph/v1.0/me/drives/sharedByMe", nil)
			svc.GetSharedByMe(rr, r)
			Expect(rr.Code).To(Equal(http.StatusInternalServerError))
		})

		It("handles ListShares returning an error status", func() {
			emptyListPublicSharesMock()
			gatewayClient.On("ListShares", mock.Anything, mock.Anything).Return(
				&collaboration.ListSharesResponse{Status: status.NewInternal(ctx, "error listing shares")},
				nil,
			)

			r := httptest.NewRequest(http.MethodGet, "/graph/v1.0/me/drives/sharedByMe", nil)
			svc.GetSharedByMe(rr, r)
			Expect(rr.Code).To(Equal(http.StatusInternalServerError))
		})

		It("succeeds, when no shares are returned", func() {
			emptyListPublicSharesMock()
			emptyListSharesMock()

			r := httptest.NewRequest(http.MethodGet, "/graph/v1.0/me/drives/sharedByMe", nil)
			svc.GetSharedByMe(rr, r)
			Expect(rr.Code).To(Equal(http.StatusOK))

			data, err := io.ReadAll(rr.Body)
			Expect(err).ToNot(HaveOccurred())

			res := itemsList{}
			err = json.Unmarshal(data, &res)
			Expect(err).ToNot(HaveOccurred())

			Expect(len(res.Value)).To(Equal(0))
		})

		It("returns a proper driveItem, when a single user share is returned", func() {
			emptyListPublicSharesMock()
			gatewayClient.On("ListShares", mock.Anything, mock.Anything).Return(
				&collaboration.ListSharesResponse{
					Status: status.NewOK(ctx),
					Shares: []*collaboration.Share{
						&userShare,
					},
				},
				nil,
			)

			r := httptest.NewRequest(http.MethodGet, "/graph/v1.0/me/drives/sharedByMe", nil)
			svc.GetSharedByMe(rr, r)
			Expect(rr.Code).To(Equal(http.StatusOK))

			data, err := io.ReadAll(rr.Body)
			Expect(err).ToNot(HaveOccurred())

			res := itemsList{}
			err = json.Unmarshal(data, &res)
			Expect(err).ToNot(HaveOccurred())

			Expect(len(res.Value)).To(Equal(1))

			di := res.Value[0]
			Expect(di.GetId()).To(Equal(storagespace.FormatResourceID(userShare.GetResourceId())))

			perm := di.GetPermissions()
			Expect(perm[0].GetId()).To(Equal(userShare.GetId().GetOpaqueId()))
			_, ok := perm[0].GetExpirationDateTimeOk()
			Expect(ok).To(BeFalse())
			_, ok = perm[0].GrantedToV2.GetGroupOk()
			Expect(ok).To(BeFalse())
			user, ok := perm[0].GrantedToV2.GetUserOk()
			Expect(ok).To(BeTrue())
			Expect(user.GetId()).To(Equal(userShare.GetGrantee().GetUserId().GetOpaqueId()))
			_, ok = perm[0].GetLinkOk()
			Expect(ok).To(BeFalse())
			roles, ok := perm[0].GetRolesOk()
			Expect(ok).To(BeTrue())
			Expect(len(roles)).To(Equal(1))
			Expect(roles[0]).To(Equal(unifiedrole.UnifiedRoleEditorID))
		})

		It("returns a proper driveItem, when a single group share is returned", func() {
			emptyListPublicSharesMock()
			gatewayClient.On("ListShares", mock.Anything, mock.Anything).Return(
				&collaboration.ListSharesResponse{
					Status: status.NewOK(ctx),
					Shares: []*collaboration.Share{
						&groupShare,
					},
				},
				nil,
			)
			r := httptest.NewRequest(http.MethodGet, "/graph/v1.0/me/drives/sharedByMe", nil)
			svc.GetSharedByMe(rr, r)
			Expect(rr.Code).To(Equal(http.StatusOK))

			data, err := io.ReadAll(rr.Body)
			Expect(err).ToNot(HaveOccurred())

			res := itemsList{}
			err = json.Unmarshal(data, &res)
			Expect(err).ToNot(HaveOccurred())

			Expect(len(res.Value)).To(Equal(1))

			di := res.Value[0]
			Expect(di.GetId()).To(Equal(storagespace.FormatResourceID(groupShare.GetResourceId())))

			perm := di.GetPermissions()
			Expect(perm[0].GetId()).To(Equal(userShare.GetId().GetOpaqueId()))
			_, ok := perm[0].GetExpirationDateTimeOk()
			Expect(ok).To(BeFalse())
			_, ok = perm[0].GrantedToV2.GetUserOk()
			Expect(ok).To(BeFalse())
			group, ok := perm[0].GrantedToV2.GetGroupOk()
			Expect(ok).To(BeTrue())
			Expect(group.GetId()).To(Equal(groupShare.GetGrantee().GetGroupId().GetOpaqueId()))
			_, ok = perm[0].GetLinkOk()
			Expect(ok).To(BeFalse())
			roles, ok := perm[0].GetRolesOk()
			Expect(ok).To(BeTrue())
			Expect(len(roles)).To(Equal(1))
			Expect(roles[0]).To(Equal(unifiedrole.UnifiedRoleEditorID))
		})

		It("returns a single driveItem, when a mulitple shares for the same resource are returned", func() {
			emptyListPublicSharesMock()
			gatewayClient.On("ListShares", mock.Anything, mock.Anything).Return(
				&collaboration.ListSharesResponse{
					Status: status.NewOK(ctx),
					Shares: []*collaboration.Share{
						&groupShare,
						&userShare,
					},
				},
				nil,
			)
			r := httptest.NewRequest(http.MethodGet, "/graph/v1.0/me/drives/sharedByMe", nil)
			svc.GetSharedByMe(rr, r)
			Expect(rr.Code).To(Equal(http.StatusOK))

			data, err := io.ReadAll(rr.Body)
			Expect(err).ToNot(HaveOccurred())

			res := itemsList{}
			err = json.Unmarshal(data, &res)
			Expect(err).ToNot(HaveOccurred())

			Expect(len(res.Value)).To(Equal(1))

			di := res.Value[0]
			Expect(di.GetId()).To(Equal(storagespace.FormatResourceID(groupShare.GetResourceId())))

			// one permission per share
			Expect(len(di.GetPermissions())).To(Equal(2))
		})

		It("return a driveItem with the expiration date set, for expiring shares", func() {
			emptyListPublicSharesMock()
			gatewayClient.On("ListShares", mock.Anything, mock.Anything).Return(
				&collaboration.ListSharesResponse{
					Status: status.NewOK(ctx),
					Shares: []*collaboration.Share{
						&userShareWithExpiration,
					},
				},
				nil,
			)
			r := httptest.NewRequest(http.MethodGet, "/graph/v1.0/me/drives/sharedByMe", nil)
			svc.GetSharedByMe(rr, r)
			Expect(rr.Code).To(Equal(http.StatusOK))

			data, err := io.ReadAll(rr.Body)
			Expect(err).ToNot(HaveOccurred())

			res := itemsList{}
			err = json.Unmarshal(data, &res)
			Expect(err).ToNot(HaveOccurred())

			Expect(len(res.Value)).To(Equal(1))

			di := res.Value[0]
			Expect(di.GetId()).To(Equal(storagespace.FormatResourceID(userShareWithExpiration.GetResourceId())))

			perm := di.GetPermissions()
			Expect(perm[0].GetId()).To(Equal(userShareWithExpiration.GetId().GetOpaqueId()))
			exp, ok := perm[0].GetExpirationDateTimeOk()
			Expect(ok).To(BeTrue())
			Expect(exp.Equal(expiration)).To(BeTrue())
			_, ok = perm[0].GrantedToV2.GetGroupOk()
			Expect(ok).To(BeFalse())
			user, ok := perm[0].GrantedToV2.GetUserOk()
			Expect(ok).To(BeTrue())
			Expect(user.GetId()).To(Equal(userShareWithExpiration.GetGrantee().GetUserId().GetOpaqueId()))
			_, ok = perm[0].GetLinkOk()
			Expect(ok).To(BeFalse())
		})

		// Public Shares / "links" in graph terms
		It("handles a failing ListPublicShares", func() {
			gatewayClient.On("ListPublicShares", mock.Anything, mock.Anything).Return(nil, errors.New("some error"))
			emptyListSharesMock()
			r := httptest.NewRequest(http.MethodGet, "/graph/v1.0/me/drives/sharedByMe", nil)
			svc.GetSharedByMe(rr, r)
			Expect(rr.Code).To(Equal(http.StatusInternalServerError))
		})

		It("handles ListPublicShares returning an error status", func() {
			emptyListSharesMock()
			gatewayClient.On("ListPublicShares", mock.Anything, mock.Anything).Return(
				&link.ListPublicSharesResponse{Status: status.NewInternal(ctx, "error listing shares")},
				nil,
			)

			r := httptest.NewRequest(http.MethodGet, "/graph/v1.0/me/drives/sharedByMe", nil)
			svc.GetSharedByMe(rr, r)
			Expect(rr.Code).To(Equal(http.StatusInternalServerError))
		})

		It("returns a proper driveItem, when a single public share is returned", func() {
			emptyListSharesMock()
			gatewayClient.On("ListPublicShares", mock.Anything, mock.Anything).Return(
				&link.ListPublicSharesResponse{
					Status: status.NewOK(ctx),
					Share: []*link.PublicShare{
						&publicShare,
					},
				},
				nil,
			)

			r := httptest.NewRequest(http.MethodGet, "/graph/v1.0/me/drives/sharedByMe", nil)
			svc.GetSharedByMe(rr, r)
			Expect(rr.Code).To(Equal(http.StatusOK))

			data, err := io.ReadAll(rr.Body)
			Expect(err).ToNot(HaveOccurred())

			res := itemsList{}
			err = json.Unmarshal(data, &res)
			Expect(err).ToNot(HaveOccurred())

			Expect(len(res.Value)).To(Equal(1))

			di := res.Value[0]
			Expect(di.GetId()).To(Equal(storagespace.FormatResourceID(publicShare.GetResourceId())))

			perm := di.GetPermissions()
			Expect(perm[0].GetId()).To(Equal(publicShare.GetId().GetOpaqueId()))
			_, ok := perm[0].GetExpirationDateTimeOk()
			Expect(ok).To(BeFalse())
			_, ok = perm[0].GetGrantedToV2Ok()
			Expect(ok).To(BeFalse())
			link, ok := perm[0].GetLinkOk()
			Expect(ok).To(BeTrue())
			Expect(link.GetWebUrl()).To(Equal("https://localhost:9200/s/" + publicShare.GetToken()))
		})

	})
})
