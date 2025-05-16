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
	grouppb "github.com/cs3org/go-cs3apis/cs3/identity/group/v1beta1"
	userpb "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"
	collaboration "github.com/cs3org/go-cs3apis/cs3/sharing/collaboration/v1beta1"
	link "github.com/cs3org/go-cs3apis/cs3/sharing/link/v1beta1"
	provider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/opencloud-eu/opencloud/pkg/shared"
	"github.com/opencloud-eu/opencloud/services/graph/mocks"
	"github.com/opencloud-eu/opencloud/services/graph/pkg/config"
	"github.com/opencloud-eu/opencloud/services/graph/pkg/config/defaults"
	identitymocks "github.com/opencloud-eu/opencloud/services/graph/pkg/identity/mocks"
	"github.com/opencloud-eu/opencloud/services/graph/pkg/linktype"
	service "github.com/opencloud-eu/opencloud/services/graph/pkg/service/v0"
	"github.com/opencloud-eu/opencloud/services/graph/pkg/unifiedrole"
	"github.com/opencloud-eu/reva/v2/pkg/conversions"
	"github.com/opencloud-eu/reva/v2/pkg/rgrpc/status"
	"github.com/opencloud-eu/reva/v2/pkg/rgrpc/todo/pool"
	"github.com/opencloud-eu/reva/v2/pkg/storagespace"
	"github.com/opencloud-eu/reva/v2/pkg/utils"
	cs3mocks "github.com/opencloud-eu/reva/v2/tests/cs3mocks/mocks"
	libregraph "github.com/opencloud-eu/libre-graph-api-go"
	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc"
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

		identityBackend = &identitymocks.Backend{}

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
