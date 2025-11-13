package svc_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"slices"
	"time"

	gateway "github.com/cs3org/go-cs3apis/cs3/gateway/v1beta1"
	grouppb "github.com/cs3org/go-cs3apis/cs3/identity/group/v1beta1"
	userpb "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"
	collaboration "github.com/cs3org/go-cs3apis/cs3/sharing/collaboration/v1beta1"
	link "github.com/cs3org/go-cs3apis/cs3/sharing/link/v1beta1"
	provider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	types "github.com/cs3org/go-cs3apis/cs3/types/v1beta1"
	"github.com/go-chi/chi/v5"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	libregraph "github.com/opencloud-eu/libre-graph-api-go"
	"github.com/opencloud-eu/opencloud/services/graph/pkg/config"
	"github.com/stretchr/testify/mock"
	"github.com/tidwall/gjson"
	"google.golang.org/grpc"

	roleconversions "github.com/opencloud-eu/reva/v2/pkg/conversions"
	revactx "github.com/opencloud-eu/reva/v2/pkg/ctx"
	"github.com/opencloud-eu/reva/v2/pkg/rgrpc/status"
	"github.com/opencloud-eu/reva/v2/pkg/storagespace"
	"github.com/opencloud-eu/reva/v2/pkg/utils"
	cs3mocks "github.com/opencloud-eu/reva/v2/tests/cs3mocks/mocks"

	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/services/graph/mocks"
	"github.com/opencloud-eu/opencloud/services/graph/pkg/config/defaults"
	"github.com/opencloud-eu/opencloud/services/graph/pkg/errorcode"
	identitycache "github.com/opencloud-eu/opencloud/services/graph/pkg/identity/cache"
	"github.com/opencloud-eu/opencloud/services/graph/pkg/linktype"
	svc "github.com/opencloud-eu/opencloud/services/graph/pkg/service/v0"
	"github.com/opencloud-eu/opencloud/services/graph/pkg/unifiedrole"
)

var _ = Describe("DriveItemPermissionsService", func() {
	var (
		driveItemPermissionsService svc.DriveItemPermissionsService
		gatewayClient               *cs3mocks.GatewayAPIClient
		gatewaySelector             *mocks.Selectable[gateway.GatewayAPIClient]
		getUserResponse             *userpb.GetUserResponse
		listPublicSharesResponse    *link.ListPublicSharesResponse
		listSpacesResponse          *provider.ListStorageSpacesResponse
		currentUser                 = &userpb.User{
			Id: &userpb.UserId{
				OpaqueId: "user",
			},
		}
		cache        identitycache.IdentityCache
		statResponse *provider.StatResponse
		driveItemId  *provider.ResourceId
		ctx          context.Context
		cfg          *config.Config
	)

	BeforeEach(func() {
		logger := log.NewLogger()
		gatewayClient = cs3mocks.NewGatewayAPIClient(GinkgoT())

		gatewaySelector = mocks.NewSelectable[gateway.GatewayAPIClient](GinkgoT())
		gatewaySelector.On("Next").Return(gatewayClient, nil)

		cache = identitycache.NewIdentityCache(identitycache.IdentityCacheWithGatewaySelector(gatewaySelector))

		cfg = defaults.FullDefaultConfig()
		service, err := svc.NewDriveItemPermissionsService(logger, gatewaySelector, cache, cfg)
		Expect(err).ToNot(HaveOccurred())
		driveItemPermissionsService = service
		ctx = revactx.ContextSetUser(context.Background(), currentUser)
		statResponse = &provider.StatResponse{
			Status: status.NewOK(ctx),
		}
		getUserResponse = &userpb.GetUserResponse{
			Status: status.NewOK(ctx),
			User: &userpb.User{
				Id:          &userpb.UserId{OpaqueId: "1"},
				DisplayName: "Cem Kaner",
			},
		}
		listPublicSharesResponse = &link.ListPublicSharesResponse{
			Status: status.NewOK(ctx),
		}

		driveItemId = &provider.ResourceId{
			StorageId: "1",
			SpaceId:   "2",
			OpaqueId:  "3",
		}

	})

	Describe("Invite", func() {
		var (
			createShareResponse *collaboration.CreateShareResponse
			driveItemInvite     libregraph.DriveItemInvite
			getGroupResponse    *grouppb.GetGroupResponse
		)

		BeforeEach(func() {
			gatewayClient.On("Stat", mock.Anything, mock.Anything).Return(statResponse, nil)
			statResponse.Info = &provider.ResourceInfo{
				Id:   driveItemId,
				Type: provider.ResourceType_RESOURCE_TYPE_FILE,
			}

			getGroupResponse = &grouppb.GetGroupResponse{
				Status: status.NewOK(ctx),
				Group: &grouppb.Group{
					Id:          &grouppb.GroupId{OpaqueId: "2"},
					GroupName:   "Florida Institute of Technology",
					DisplayName: "Florida Institute of Technology",
				},
			}

			createShareResponse = &collaboration.CreateShareResponse{
				Status: status.NewOK(ctx),
			}
		})

		It("creates user shares as expected (happy path)", func() {
			gatewayClient.On("GetUser", mock.Anything, mock.Anything).Return(getUserResponse, nil)
			gatewayClient.On("CreateShare", mock.Anything, mock.Anything).Return(createShareResponse, nil)
			driveItemInvite.Recipients = []libregraph.DriveRecipient{
				{ObjectId: libregraph.PtrString("1"), LibreGraphRecipientType: libregraph.PtrString("user")},
			}
			driveItemInvite.ExpirationDateTime = libregraph.PtrTime(time.Now().Add(time.Hour))
			createShareResponse.Share = &collaboration.Share{
				Id:         &collaboration.ShareId{OpaqueId: "123"},
				Expiration: utils.TimeToTS(*driveItemInvite.ExpirationDateTime),
			}

			permission, err := driveItemPermissionsService.Invite(ctx, driveItemId, driveItemInvite)
			Expect(err).ToNot(HaveOccurred())
			Expect(permission.GetId()).To(Equal("123"))
			Expect(permission.GetExpirationDateTime().Equal(*driveItemInvite.ExpirationDateTime)).To(BeTrue())
			Expect(permission.GrantedToV2.User.GetDisplayName()).To(Equal(getUserResponse.User.DisplayName))
			Expect(permission.GrantedToV2.User.GetId()).To(Equal("1"))
		})

		It("creates group shares as expected (happy path)", func() {
			gatewayClient.On("GetGroup", mock.Anything, mock.Anything).Return(getGroupResponse, nil)
			gatewayClient.On("GetUser", mock.Anything, mock.Anything).Return(getUserResponse, nil)
			gatewayClient.On("CreateShare", mock.Anything, mock.Anything).Return(createShareResponse, nil)
			driveItemInvite.Recipients = []libregraph.DriveRecipient{
				{ObjectId: libregraph.PtrString("2"), LibreGraphRecipientType: libregraph.PtrString("group")},
			}
			driveItemInvite.ExpirationDateTime = libregraph.PtrTime(time.Now().Add(time.Hour))
			createShareResponse.Share = &collaboration.Share{
				Id:         &collaboration.ShareId{OpaqueId: "123"},
				Expiration: utils.TimeToTS(*driveItemInvite.ExpirationDateTime),
			}

			permission, err := driveItemPermissionsService.Invite(ctx, driveItemId, driveItemInvite)
			Expect(err).ToNot(HaveOccurred())
			Expect(permission.GetId()).To(Equal("123"))
			Expect(permission.GetExpirationDateTime().Equal(*driveItemInvite.ExpirationDateTime)).To(BeTrue())
			Expect(permission.GrantedToV2.Group.GetDisplayName()).To(Equal(getGroupResponse.Group.DisplayName))
			Expect(permission.GrantedToV2.Group.GetId()).To(Equal("2"))
		})

		It("succeeds with file roles (happy path)", func() {
			gatewayClient.On("GetUser", mock.Anything, mock.Anything).Return(getUserResponse, nil)
			gatewayClient.On("CreateShare", mock.Anything, mock.Anything).Return(createShareResponse, nil)
			driveItemInvite.Recipients = []libregraph.DriveRecipient{
				{ObjectId: libregraph.PtrString("1"), LibreGraphRecipientType: libregraph.PtrString("user")},
			}
			driveItemInvite.Roles = []string{unifiedrole.UnifiedRoleViewerID}

			permission, err := driveItemPermissionsService.Invite(ctx, driveItemId, driveItemInvite)
			Expect(err).ToNot(HaveOccurred())

			Expect(permission.GetRoles()).To(HaveLen(1))
			Expect(permission.GetRoles()[0]).To(Equal(unifiedrole.UnifiedRoleViewerID))
		})

		It("succeeds with folder roles (happy path)", func() {
			statResponse.Info.Type = provider.ResourceType_RESOURCE_TYPE_CONTAINER
			gatewayClient.On("GetUser", mock.Anything, mock.Anything).Return(getUserResponse, nil)
			gatewayClient.On("CreateShare", mock.Anything, mock.Anything).Return(createShareResponse, nil)
			driveItemInvite.Recipients = []libregraph.DriveRecipient{
				{ObjectId: libregraph.PtrString("1"), LibreGraphRecipientType: libregraph.PtrString("user")},
			}
			driveItemInvite.Roles = []string{unifiedrole.UnifiedRoleEditorID}

			permission, err := driveItemPermissionsService.Invite(ctx, driveItemId, driveItemInvite)
			Expect(err).ToNot(HaveOccurred())

			Expect(permission.GetRoles()).To(HaveLen(1))
			Expect(permission.GetRoles()[0]).To(Equal(unifiedrole.UnifiedRoleEditorID))
		})

		It("fails with when trying to set a space role on a file", func() {
			driveItemInvite.Recipients = []libregraph.DriveRecipient{
				{ObjectId: libregraph.PtrString("1"), LibreGraphRecipientType: libregraph.PtrString("user")},
			}
			driveItemInvite.Roles = []string{unifiedrole.UnifiedRoleManagerID}
			permission, err := driveItemPermissionsService.Invite(ctx, driveItemId, driveItemInvite)

			Expect(err).To(MatchError(errorcode.New(errorcode.InvalidRequest, "role not applicable to this resource")))
			Expect(permission).To(BeZero())
		})

		It("fails with when trying to set a folder role on a file", func() {
			driveItemInvite.Recipients = []libregraph.DriveRecipient{
				{ObjectId: libregraph.PtrString("1"), LibreGraphRecipientType: libregraph.PtrString("user")},
			}
			driveItemInvite.Roles = []string{unifiedrole.UnifiedRoleEditorID}
			permission, err := driveItemPermissionsService.Invite(ctx, driveItemId, driveItemInvite)

			Expect(err).To(MatchError(errorcode.New(errorcode.InvalidRequest, "role not applicable to this resource")))
			Expect(permission).To(BeZero())
		})

		It("fails with when trying to set a file role on a folder", func() {
			statResponse.Info.Type = provider.ResourceType_RESOURCE_TYPE_CONTAINER
			driveItemInvite.Recipients = []libregraph.DriveRecipient{
				{ObjectId: libregraph.PtrString("1"), LibreGraphRecipientType: libregraph.PtrString("user")},
			}
			driveItemInvite.Roles = []string{unifiedrole.UnifiedRoleFileEditorID}
			permission, err := driveItemPermissionsService.Invite(ctx, driveItemId, driveItemInvite)

			Expect(err).To(MatchError(errorcode.New(errorcode.InvalidRequest, "role not applicable to this resource")))
			Expect(permission).To(BeZero())
		})

		It("with actions (happy path)", func() {
			gatewayClient.On("GetUser", mock.Anything, mock.Anything).Return(getUserResponse, nil)
			gatewayClient.On("CreateShare", mock.Anything, mock.Anything).Return(createShareResponse, nil)
			driveItemInvite.Recipients = []libregraph.DriveRecipient{
				{ObjectId: libregraph.PtrString("1"), LibreGraphRecipientType: libregraph.PtrString("user")},
			}
			driveItemInvite.Roles = nil
			driveItemInvite.LibreGraphPermissionsActions = []string{unifiedrole.DriveItemContentRead}

			permission, err := driveItemPermissionsService.Invite(ctx, driveItemId, driveItemInvite)
			Expect(err).ToNot(HaveOccurred())

			Expect(permission).NotTo(BeZero())
			Expect(permission.GetRoles()).To(HaveLen(0))
			Expect(permission.GetLibreGraphPermissionsActions()).To(HaveLen(1))
			Expect(permission.GetLibreGraphPermissionsActions()[0]).To(Equal(unifiedrole.DriveItemContentRead))
		})

		It("fails with a missing driveritem", func() {
			statResponse.Status = status.NewNotFound(context.Background(), "not found")
			permission, err := driveItemPermissionsService.Invite(ctx, driveItemId, driveItemInvite)
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(errorcode.New(errorcode.ItemNotFound, "not found").WithOrigin(errorcode.ErrorOriginCS3)))
			Expect(permission).To(BeZero())
		})

		It("fails with unknown or disable role", func() {
			cfg := defaults.FullDefaultConfig()
			slices.DeleteFunc(cfg.UnifiedRoles.AvailableRoles, func(s string) bool {
				// SecureViewer is enabled in ci, we need to remove it in the unit test
				return s != unifiedrole.UnifiedRoleSecureViewerID
			})
			service, err := svc.NewDriveItemPermissionsService(log.NewLogger(), gatewaySelector, cache, cfg)
			Expect(err).ToNot(HaveOccurred())

			driveItemInvite.Roles = []string{unifiedrole.UnifiedRoleViewerID, unifiedrole.UnifiedRoleSecureViewerID}
			_, err = service.Invite(ctx, driveItemId, driveItemInvite)
			Expect(err).To(MatchError(unifiedrole.ErrUnknownRole))
		})
	})
	Describe("SpaceRootInvite", func() {
		var (
			createShareResponse *collaboration.CreateShareResponse
			driveItemInvite     libregraph.DriveItemInvite
			driveId             *provider.ResourceId
			getUserResponse     *userpb.GetUserResponse
		)

		BeforeEach(func() {
			driveId = &provider.ResourceId{
				StorageId: "1",
				SpaceId:   "2",
			}

			listSpacesResponse = &provider.ListStorageSpacesResponse{
				Status: status.NewOK(ctx),
				StorageSpaces: []*provider.StorageSpace{
					{
						Id: &provider.StorageSpaceId{
							OpaqueId: "2",
						},
					},
				},
			}

			getUserResponse = &userpb.GetUserResponse{
				Status: status.NewOK(ctx),
				User: &userpb.User{
					Id:          &userpb.UserId{OpaqueId: "1"},
					DisplayName: "Cem Kaner",
				},
			}

			createShareResponse = &collaboration.CreateShareResponse{
				Status: status.NewOK(ctx),
			}
			statResponse.Info = &provider.ResourceInfo{}
		})

		It("adds a user to a space as expected (happy path)", func() {
			listSpacesResponse.StorageSpaces[0].SpaceType = "project"
			listSpacesResponse.StorageSpaces[0].Root = &provider.ResourceId{
				StorageId: "1",
				SpaceId:   "2",
				OpaqueId:  "3",
			}
			statResponse.Info.Id = listSpacesResponse.StorageSpaces[0].Root
			statResponse.Info.Space = &provider.StorageSpace{
				Root: listSpacesResponse.StorageSpaces[0].Root,
			}

			gatewayClient.On("ListStorageSpaces", mock.Anything, mock.Anything).Return(listSpacesResponse, nil)
			gatewayClient.On("GetUser", mock.Anything, mock.Anything).Return(getUserResponse, nil)
			gatewayClient.On("Stat", mock.Anything, mock.Anything).Return(statResponse, nil)
			gatewayClient.On("CreateShare", mock.Anything, mock.Anything).Return(createShareResponse, nil)
			driveItemInvite.Recipients = []libregraph.DriveRecipient{
				{ObjectId: libregraph.PtrString("1"), LibreGraphRecipientType: libregraph.PtrString("user")},
			}
			driveItemInvite.ExpirationDateTime = libregraph.PtrTime(time.Now().Add(time.Hour))
			createShareResponse.Share = &collaboration.Share{
				Id:         &collaboration.ShareId{OpaqueId: "123"},
				Expiration: utils.TimeToTS(*driveItemInvite.ExpirationDateTime),
			}

			permission, err := driveItemPermissionsService.SpaceRootInvite(ctx, driveId, driveItemInvite)
			Expect(err).ToNot(HaveOccurred())
			Expect(permission.GetId()).To(Equal("123"))
			Expect(permission.GetExpirationDateTime().Equal(*driveItemInvite.ExpirationDateTime)).To(BeTrue())
			Expect(permission.GrantedToV2.User.GetDisplayName()).To(Equal(getUserResponse.User.DisplayName))
			Expect(permission.GrantedToV2.User.GetId()).To(Equal("1"))
		})
		It("rejects to add a user to a personal space", func() {
			gatewayClient.On("ListStorageSpaces", mock.Anything, mock.Anything).Return(listSpacesResponse, nil)
			driveItemInvite.Recipients = []libregraph.DriveRecipient{
				{ObjectId: libregraph.PtrString("1"), LibreGraphRecipientType: libregraph.PtrString("user")},
			}
			driveItemInvite.ExpirationDateTime = libregraph.PtrTime(time.Now().Add(time.Hour))
			createShareResponse.Share = &collaboration.Share{
				Id:         &collaboration.ShareId{OpaqueId: "123"},
				Expiration: utils.TimeToTS(*driveItemInvite.ExpirationDateTime),
			}

			permission, err := driveItemPermissionsService.SpaceRootInvite(ctx, driveId, driveItemInvite)
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(errorcode.New(errorcode.InvalidRequest, "unsupported space type")))
			Expect(permission).To(BeZero())
		})
	})
	Describe("ListPermissions", func() {
		var (
			itemID             *provider.ResourceId
			listSharesResponse *collaboration.ListSharesResponse
		)
		BeforeEach(func() {
			itemID = &provider.ResourceId{
				StorageId: "1",
				SpaceId:   "2",
				OpaqueId:  "3",
			}
			listSharesResponse = &collaboration.ListSharesResponse{
				Status: status.NewOK(ctx),
				Shares: []*collaboration.Share{},
			}
			statResponse.Info = &provider.ResourceInfo{
				Id:            itemID,
				Type:          provider.ResourceType_RESOURCE_TYPE_FILE,
				PermissionSet: roleconversions.NewViewerRole().CS3ResourcePermissions(),
			}
		})
		It("populates allowedValues for files that are not shared", func() {
			gatewayClient.On("Stat", mock.Anything, mock.Anything).Return(statResponse, nil)
			gatewayClient.On("ListShares", mock.Anything, mock.Anything).Return(listSharesResponse, nil)
			gatewayClient.On("ListPublicShares", mock.Anything, mock.Anything).Return(listPublicSharesResponse, nil)
			permissions, err := driveItemPermissionsService.ListPermissions(context.Background(), itemID, svc.ListPermissionsQueryOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(len(permissions.LibreGraphPermissionsActionsAllowedValues)).ToNot(BeZero())
			Expect(len(permissions.LibreGraphPermissionsRolesAllowedValues)).ToNot(BeZero())
		})
		It("returns one permission per share", func() {
			statResponse.Info.PermissionSet = roleconversions.NewEditorRole().CS3ResourcePermissions()
			listSharesResponse.Shares = []*collaboration.Share{
				{
					Id: &collaboration.ShareId{OpaqueId: "1"},
					Permissions: &collaboration.SharePermissions{
						Permissions: roleconversions.NewViewerRole().CS3ResourcePermissions(),
					},
					ResourceId: &provider.ResourceId{
						StorageId: "1",
						SpaceId:   "2",
						OpaqueId:  "3",
					},
					Grantee: &provider.Grantee{
						Type: provider.GranteeType_GRANTEE_TYPE_USER,
						Id: &provider.Grantee_UserId{
							UserId: &userpb.UserId{
								OpaqueId: "user-id",
							},
						},
					},
				},
			}
			listPublicSharesResponse.Share = []*link.PublicShare{
				{
					Id: &link.PublicShareId{
						OpaqueId: "public-share-id",
					},
					Token: "public-share-token",
					// the link shares the same resource id
					ResourceId: &provider.ResourceId{
						StorageId: "1",
						SpaceId:   "2",
						OpaqueId:  "3",
					},
					Permissions: &link.PublicSharePermissions{Permissions: roleconversions.NewViewerRole().CS3ResourcePermissions()},
				},
			}

			gatewayClient.On("Stat", mock.Anything, mock.Anything).Return(statResponse, nil)
			gatewayClient.On("ListShares", mock.Anything, mock.Anything).Return(listSharesResponse, nil)
			gatewayClient.On("GetUser", mock.Anything, mock.Anything).Return(getUserResponse, nil)
			gatewayClient.On("ListPublicShares", mock.Anything, mock.Anything).Return(listPublicSharesResponse, nil)
			permissions, err := driveItemPermissionsService.ListPermissions(context.Background(), itemID, svc.ListPermissionsQueryOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(len(permissions.LibreGraphPermissionsActionsAllowedValues)).ToNot(BeZero())
			Expect(len(permissions.LibreGraphPermissionsRolesAllowedValues)).ToNot(BeZero())
			Expect(len(permissions.Value)).To(Equal(2))
		})
		It("returns role denied", func() {
			statResponse.Info.PermissionSet = roleconversions.NewManagerRole().CS3ResourcePermissions()
			listSharesResponse.Shares = []*collaboration.Share{
				{
					Id: &collaboration.ShareId{OpaqueId: "1"},
					Permissions: &collaboration.SharePermissions{
						Permissions: roleconversions.NewDeniedRole().CS3ResourcePermissions(),
					},
					ResourceId: &provider.ResourceId{
						StorageId: "1",
						SpaceId:   "2",
						OpaqueId:  "3",
					},
					Grantee: &provider.Grantee{
						Type: provider.GranteeType_GRANTEE_TYPE_USER,
						Id: &provider.Grantee_UserId{
							UserId: &userpb.UserId{
								OpaqueId: "user-id",
							},
						},
					},
				},
			}
			listPublicSharesResponse.Share = []*link.PublicShare{}

			cfg = defaults.FullDefaultConfig()
			cfg.UnifiedRoles.AvailableRoles = []string{unifiedrole.UnifiedRoleViewerID, unifiedrole.UnifiedRoleDeniedID, unifiedrole.UnifiedRoleManagerID}
			service, err := svc.NewDriveItemPermissionsService(log.NewLogger(), gatewaySelector, cache, cfg)

			gatewayClient.On("Stat", mock.Anything, mock.Anything).Return(statResponse, nil)
			gatewayClient.On("ListShares", mock.Anything, mock.Anything).Return(listSharesResponse, nil)
			gatewayClient.On("GetUser", mock.Anything, mock.Anything).Return(getUserResponse, nil)
			gatewayClient.On("ListPublicShares", mock.Anything, mock.Anything).Return(listPublicSharesResponse, nil)
			permissions, err := service.ListPermissions(context.Background(), itemID, svc.ListPermissionsQueryOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(len(permissions.LibreGraphPermissionsActionsAllowedValues)).ToNot(BeZero())
			Expect(len(permissions.LibreGraphPermissionsRolesAllowedValues)).ToNot(BeZero())
			Expect(len(permissions.Value)).To(Equal(1))
			Expect(permissions.Value[0].GetRoles()[0]).To(Equal(unifiedrole.UnifiedRoleDeniedID))
		})
		It("returns access denied when no role and no actions are set (full denial)", func() {
			statResponse.Info.PermissionSet = roleconversions.NewManagerRole().CS3ResourcePermissions()
			listSharesResponse.Shares = []*collaboration.Share{
				{
					Id: &collaboration.ShareId{OpaqueId: "1"},
					Permissions: &collaboration.SharePermissions{
						Permissions: roleconversions.NewDeniedRole().CS3ResourcePermissions(),
					},
					ResourceId: &provider.ResourceId{
						StorageId: "1",
						SpaceId:   "2",
						OpaqueId:  "3",
					},
					Grantee: &provider.Grantee{
						Type: provider.GranteeType_GRANTEE_TYPE_USER,
						Id: &provider.Grantee_UserId{
							UserId: &userpb.UserId{
								OpaqueId: "user-id",
							},
						},
					},
				},
			}
			listPublicSharesResponse.Share = []*link.PublicShare{}

			gatewayClient.On("Stat", mock.Anything, mock.Anything).Return(statResponse, nil)
			gatewayClient.On("ListShares", mock.Anything, mock.Anything).Return(listSharesResponse, nil)
			gatewayClient.On("GetUser", mock.Anything, mock.Anything).Return(getUserResponse, nil)
			gatewayClient.On("ListPublicShares", mock.Anything, mock.Anything).Return(listPublicSharesResponse, nil)
			permissions, err := driveItemPermissionsService.ListPermissions(context.Background(), itemID, svc.ListPermissionsQueryOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(len(permissions.LibreGraphPermissionsActionsAllowedValues)).ToNot(BeZero())
			Expect(len(permissions.LibreGraphPermissionsRolesAllowedValues)).ToNot(BeZero())
			Expect(len(permissions.Value)).To(Equal(1))
			Expect(permissions.Value[0].GetLibreGraphPermissionsActions()[0]).To(Equal("none"))
		})
		It("Does not list public shares when requested so", func() {
			opt := svc.ListPermissionsQueryOptions{
				NoLinkPermissions: true,
			}
			gatewayClient.On("Stat", mock.Anything, mock.Anything).Return(statResponse, nil)
			gatewayClient.On("ListShares", mock.Anything, mock.Anything).Return(listSharesResponse, nil)
			permissions, err := driveItemPermissionsService.ListPermissions(context.Background(), itemID, opt)
			Expect(err).ToNot(HaveOccurred())
			Expect(len(permissions.LibreGraphPermissionsActionsAllowedValues)).ToNot(BeZero())
			Expect(len(permissions.LibreGraphPermissionsRolesAllowedValues)).ToNot(BeZero())
		})
		It("Does not return permissions when the NoValues option is set", func() {
			opt := svc.ListPermissionsQueryOptions{
				NoValues: true,
			}
			listSharesResponse.Shares = []*collaboration.Share{
				{
					Id: &collaboration.ShareId{OpaqueId: "1"},
					Permissions: &collaboration.SharePermissions{
						Permissions: roleconversions.NewViewerRole().CS3ResourcePermissions(),
					},
					ResourceId: &provider.ResourceId{
						StorageId: "1",
						SpaceId:   "2",
						OpaqueId:  "3",
					},
					Grantee: &provider.Grantee{
						Type: provider.GranteeType_GRANTEE_TYPE_USER,
						Id: &provider.Grantee_UserId{
							UserId: &userpb.UserId{
								OpaqueId: "user-id",
							},
						},
					},
				},
			}
			listPublicSharesResponse.Share = []*link.PublicShare{
				{
					Id: &link.PublicShareId{
						OpaqueId: "public-share-id",
					},
					Token: "public-share-token",
					// the link shares the same resource id
					ResourceId: &provider.ResourceId{
						StorageId: "1",
						SpaceId:   "2",
						OpaqueId:  "3",
					},
					Permissions: &link.PublicSharePermissions{Permissions: roleconversions.NewViewerRole().CS3ResourcePermissions()},
				},
			}
			gatewayClient.On("Stat", mock.Anything, mock.Anything).Return(statResponse, nil)
			gatewayClient.On("ListShares", mock.Anything, mock.Anything).Return(listSharesResponse, nil)
			gatewayClient.On("GetUser", mock.Anything, mock.Anything).Return(getUserResponse, nil)
			gatewayClient.On("ListPublicShares", mock.Anything, mock.Anything).Return(listPublicSharesResponse, nil)
			permissions, err := driveItemPermissionsService.ListPermissions(context.Background(), itemID, opt)
			Expect(err).ToNot(HaveOccurred())
			Expect(len(permissions.LibreGraphPermissionsActionsAllowedValues)).ToNot(BeZero())
			Expect(len(permissions.LibreGraphPermissionsRolesAllowedValues)).ToNot(BeZero())
			Expect(len(permissions.Value)).To(BeZero())
		})
		It("Returns a count when the Count option is set", func() {
			opt := svc.ListPermissionsQueryOptions{
				Count: true,
			}
			listSharesResponse.Shares = []*collaboration.Share{
				{
					Id: &collaboration.ShareId{OpaqueId: "1"},
					Permissions: &collaboration.SharePermissions{
						Permissions: roleconversions.NewViewerRole().CS3ResourcePermissions(),
					},
					ResourceId: &provider.ResourceId{
						StorageId: "1",
						SpaceId:   "2",
						OpaqueId:  "3",
					},
					Grantee: &provider.Grantee{
						Type: provider.GranteeType_GRANTEE_TYPE_USER,
						Id: &provider.Grantee_UserId{
							UserId: &userpb.UserId{
								OpaqueId: "user-id",
							},
						},
					},
				},
			}
			listPublicSharesResponse.Share = []*link.PublicShare{
				{
					Id: &link.PublicShareId{
						OpaqueId: "public-share-id",
					},
					Token: "public-share-token",
					// the link shares the same resource id
					ResourceId: &provider.ResourceId{
						StorageId: "1",
						SpaceId:   "2",
						OpaqueId:  "3",
					},
					Permissions: &link.PublicSharePermissions{Permissions: roleconversions.NewViewerRole().CS3ResourcePermissions()},
				},
			}
			gatewayClient.On("Stat", mock.Anything, mock.Anything).Return(statResponse, nil)
			gatewayClient.On("ListShares", mock.Anything, mock.Anything).Return(listSharesResponse, nil)
			gatewayClient.On("GetUser", mock.Anything, mock.Anything).Return(getUserResponse, nil)
			gatewayClient.On("ListPublicShares", mock.Anything, mock.Anything).Return(listPublicSharesResponse, nil)
			permissions, err := driveItemPermissionsService.ListPermissions(context.Background(), itemID, opt)
			Expect(err).ToNot(HaveOccurred())
			Expect(len(permissions.LibreGraphPermissionsActionsAllowedValues)).ToNot(BeZero())
			Expect(len(permissions.LibreGraphPermissionsRolesAllowedValues)).ToNot(BeZero())
			count := int(permissions.GetOdataCount())
			Expect(count).To(Equal(2)) // 1 share + 1 public share
			Expect(len(permissions.Value)).To(Equal(count))
		})
	})
	Describe("ListSpaceRootPermissions", func() {
		var (
			driveId *provider.ResourceId
		)

		BeforeEach(func() {
			driveId = &provider.ResourceId{
				StorageId: "1",
				SpaceId:   "2",
			}

			listSpacesResponse = &provider.ListStorageSpacesResponse{
				Status: status.NewOK(ctx),
				StorageSpaces: []*provider.StorageSpace{
					{
						Id: &provider.StorageSpaceId{
							OpaqueId: "2",
						},
					},
				},
			}
			statResponse.Info = &provider.ResourceInfo{
				Type:          provider.ResourceType_RESOURCE_TYPE_FILE,
				PermissionSet: roleconversions.NewViewerRole().CS3ResourcePermissions(),
			}
		})

		It("adds a user to a space as expected (happy path)", func() {
			listSpacesResponse.StorageSpaces[0].SpaceType = "project"
			listSpacesResponse.StorageSpaces[0].Root = &provider.ResourceId{
				StorageId: "1",
				SpaceId:   "2",
				OpaqueId:  "2",
			}

			gatewayClient.On("ListStorageSpaces", mock.Anything, mock.Anything).Return(listSpacesResponse, nil)
			gatewayClient.On("ListPublicShares", mock.Anything, mock.Anything).Return(listPublicSharesResponse, nil)
			statResponse.Info.Id = listSpacesResponse.StorageSpaces[0].Root
			gatewayClient.On("Stat", mock.Anything, mock.Anything).Return(statResponse, nil)
			permissions, err := driveItemPermissionsService.ListSpaceRootPermissions(context.Background(), driveId, svc.ListPermissionsQueryOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(len(permissions.LibreGraphPermissionsActionsAllowedValues)).ToNot(BeZero())
		})

	})
	Describe("DeletePermission", func() {
		var (
			getShareResponse       collaboration.GetShareResponse
			getPublicShareResponse link.GetPublicShareResponse
		)
		BeforeEach(func() {
			getPublicShareResponse.Status = status.NewOK(context.Background())
			getShareResponse.Status = status.NewOK(context.Background())
			getShareResponse.Share = &collaboration.Share{
				Id: &collaboration.ShareId{
					OpaqueId: "permissionid",
				},
				ResourceId: &provider.ResourceId{
					StorageId: "1",
					SpaceId:   "2",
					OpaqueId:  "3",
				},
			}
		})
		It("fails to deletes a public link permission when it can be resolved", func() {
			gatewayClient.On("GetPublicShare", mock.Anything, mock.Anything).Return(&getPublicShareResponse, nil)

			err := driveItemPermissionsService.DeletePermission(context.Background(),
				getShareResponse.Share.ResourceId,
				"permissionid",
			)
			Expect(err).To(MatchError(errorcode.New(errorcode.ItemNotFound, "failed to resolve resource id for shared resource")))
		})
		It("deletes a user permission as expected", func() {
			getPublicShareResponse.Status = status.NewNotFound(context.Background(), "")
			gatewayClient.On("GetPublicShare", mock.Anything, mock.Anything).Return(&getPublicShareResponse, nil)
			gatewayClient.On("GetShare",
				mock.Anything,
				mock.MatchedBy(func(req *collaboration.GetShareRequest) bool {
					return req.GetRef().GetId().GetOpaqueId() == "permissionid"
				}),
			).Return(&getShareResponse, nil)

			rmShareMockResponse := &collaboration.RemoveShareResponse{
				Status: status.NewOK(ctx),
			}
			gatewayClient.On("RemoveShare",
				mock.Anything,
				mock.MatchedBy(func(req *collaboration.RemoveShareRequest) bool {
					return req.GetRef().GetId().GetOpaqueId() == "permissionid"
				}),
			).Return(rmShareMockResponse, nil)

			err := driveItemPermissionsService.DeletePermission(context.Background(),
				getShareResponse.Share.ResourceId,
				"permissionid",
			)
			Expect(err).ToNot(HaveOccurred())
		})
		It("deletes a link permission as expected", func() {
			getPublicShareResponse.Share = &link.PublicShare{
				Id: &link.PublicShareId{
					OpaqueId: "linkpermissionid",
				},
				ResourceId: &provider.ResourceId{
					StorageId: "1",
					SpaceId:   "2",
					OpaqueId:  "3",
				},
			}
			gatewayClient.On("GetPublicShare", mock.Anything, mock.Anything).Return(&getPublicShareResponse, nil)

			gatewayClient.On("RemovePublicShare",
				mock.Anything,
				mock.MatchedBy(func(req *link.RemovePublicShareRequest) bool {
					return req.GetRef().GetId().GetOpaqueId() == "linkpermissionid"
				}),
			).Return(
				&link.RemovePublicShareResponse{
					Status: status.NewOK(ctx),
				}, nil,
			)

			err := driveItemPermissionsService.DeletePermission(context.Background(),
				getShareResponse.Share.ResourceId,
				"linkpermissionid",
			)
			Expect(err).ToNot(HaveOccurred())
		})
		It("deletes a space permission as expected", func() {
			getPublicShareResponse.Status = status.NewNotFound(context.Background(), "")
			gatewayClient.On("GetPublicShare", mock.Anything, mock.Anything).Return(&getPublicShareResponse, nil)

			gatewayClient.On("RemoveShare",
				mock.Anything,
				mock.Anything,
			).Return(func(ctx context.Context, in *collaboration.RemoveShareRequest, opts ...grpc.CallOption) (*collaboration.RemoveShareResponse, error) {
				Expect(in.Ref.GetKey()).ToNot(BeNil())
				Expect(in.Ref.GetKey().GetGrantee().GetUserId().GetOpaqueId()).To(Equal("userid"))
				return &collaboration.RemoveShareResponse{Status: status.NewOK(ctx)}, nil
			})

			err := driveItemPermissionsService.DeletePermission(context.Background(),
				&provider.ResourceId{
					StorageId: "1",
					SpaceId:   "2",
					OpaqueId:  "2",
				},
				"u:userid",
			)
			Expect(err).ToNot(HaveOccurred())
		})

		It("fails to delete permission when the item id does not match the shared resource's id", func() {
			getPublicShareResponse.Status = status.NewNotFound(context.Background(), "")
			gatewayClient.On("GetPublicShare", mock.Anything, mock.Anything).Return(&getPublicShareResponse, nil)
			getShareResponse.Share.ResourceId = &provider.ResourceId{
				StorageId: "3",
				SpaceId:   "4",
				OpaqueId:  "5",
			}
			gatewayClient.On("GetShare",
				mock.Anything,
				mock.MatchedBy(func(req *collaboration.GetShareRequest) bool {
					return req.GetRef().GetId().GetOpaqueId() == "permissionid"
				}),
			).Return(&getShareResponse, nil)

			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("driveID", "1$2")
			rctx.URLParams.Add("itemID", "1$2!3")
			rctx.URLParams.Add("permissionID", "permissionid")

			ctx = context.WithValue(context.Background(), chi.RouteCtxKey, rctx)

			err := driveItemPermissionsService.DeletePermission(context.Background(),
				&provider.ResourceId{
					StorageId: "1",
					SpaceId:   "2",
					OpaqueId:  "3",
				},
				"permissionid",
			)
			Expect(err).To(MatchError(errorcode.New(errorcode.InvalidRequest, "permissionID and itemID do not match")))

		})
	})
	Describe("UpdatePermission", func() {
		var (
			driveItemPermission           libregraph.Permission
			getShareMockResponse          *collaboration.GetShareResponse
			getPublicShareMockResponse    *link.GetPublicShareResponse
			updateShareMockResponse       *collaboration.UpdateShareResponse
			updatePublicShareMockResponse *link.UpdatePublicShareResponse
		)
		const TestLinkName = "Test Link"
		BeforeEach(func() {
			ctx = revactx.ContextSetUser(context.Background(), currentUser)
			driveItemPermission = libregraph.Permission{}

			share := &collaboration.Share{
				Id: &collaboration.ShareId{
					OpaqueId: "permissionid",
				},
				ResourceId: driveItemId,
				Grantee: &provider.Grantee{
					Type: provider.GranteeType_GRANTEE_TYPE_USER,
					Id: &provider.Grantee_UserId{
						UserId: &userpb.UserId{
							OpaqueId: "userid",
						},
					},
				},
				Permissions: &collaboration.SharePermissions{
					Permissions: roleconversions.NewViewerRole().CS3ResourcePermissions(),
				},
			}
			getShareMockResponse = &collaboration.GetShareResponse{
				Status: status.NewOK(ctx),
				Share:  share,
			}

			updateShareMockResponse = &collaboration.UpdateShareResponse{
				Status: status.NewOK(ctx),
				Share:  share,
			}

			updatePublicShareMockResponse = &link.UpdatePublicShareResponse{
				Status: status.NewOK(ctx),
				Share:  &link.PublicShare{DisplayName: TestLinkName},
			}

			getPublicShareMockResponse = &link.GetPublicShareResponse{
				Status: status.NewOK(ctx),
				Share: &link.PublicShare{
					Id: &link.PublicShareId{
						OpaqueId: "permissionid",
					},
					ResourceId: driveItemId,
					Permissions: &link.PublicSharePermissions{
						Permissions: linktype.NewViewLinkPermissionSet().GetPermissions(),
					},
					Token: "token",
				},
			}
			statResponse = &provider.StatResponse{
				Status: status.NewOK(ctx),
				Info: &provider.ResourceInfo{
					Id:   driveItemId,
					Type: provider.ResourceType_RESOURCE_TYPE_CONTAINER,
				},
			}

			grantMapJSON, _ := json.Marshal(
				map[string]*provider.ResourcePermissions{
					"userid": roleconversions.NewSpaceViewerRole().CS3ResourcePermissions(),
				},
			)
			spaceOpaque := &types.Opaque{
				Map: map[string]*types.OpaqueEntry{
					"grants": {
						Decoder: "json",
						Value:   grantMapJSON,
					},
				},
			}
			listSpacesResponse = &provider.ListStorageSpacesResponse{
				Status: status.NewOK(ctx),
				StorageSpaces: []*provider.StorageSpace{
					{
						Id: &provider.StorageSpaceId{
							OpaqueId: "2",
						},
						Opaque: spaceOpaque,
					},
				},
			}
		})
		It("fails when no share is found", func() {
			getShareMockResponse.Share = nil
			getShareMockResponse.Status = status.NewNotFound(ctx, "not found")
			gatewayClient.On("GetShare", mock.Anything, mock.MatchedBy(func(req *collaboration.GetShareRequest) bool {
				return req.GetRef().GetId().GetOpaqueId() == "permissionid"
			})).Return(getShareMockResponse, nil)

			getPublicShareMockResponse.Share = nil
			getPublicShareMockResponse.Status = status.NewNotFound(ctx, "not found")
			gatewayClient.On("GetPublicShare", mock.Anything, mock.MatchedBy(func(req *link.GetPublicShareRequest) bool {
				return req.GetRef().GetId().GetOpaqueId() == "permissionid"
			})).Return(getPublicShareMockResponse, nil)

			res, err := driveItemPermissionsService.UpdatePermission(context.Background(), driveItemId, "permissionid", driveItemPermission)
			Expect(err).To(HaveOccurred())
			Expect(res).To(BeZero())
		})
		It("fails to update permission when the resourceID mismatches with the shared resource's id", func() {
			getShareMockResponse.Share = nil
			getShareMockResponse.Status = status.NewNotFound(ctx, "not found")
			getPublicShareMockResponse.Share.ResourceId = &provider.ResourceId{
				StorageId: "1",
				SpaceId:   "2",
				OpaqueId:  "4",
			}
			gatewayClient.On("GetPublicShare", mock.Anything, mock.MatchedBy(func(req *link.GetPublicShareRequest) bool {
				return req.GetRef().GetId().GetOpaqueId() == "permissionid"
			})).Return(getPublicShareMockResponse, nil)

			driveItemPermission.SetExpirationDateTime(time.Now().Add(time.Hour))
			res, err := driveItemPermissionsService.UpdatePermission(context.Background(), driveItemId, "permissionid", driveItemPermission)
			Expect(err).To(MatchError(errorcode.New(errorcode.InvalidRequest, "permissionID and itemID do not match")))
			Expect(res).To(BeZero())
		})
		It("succeeds when trying to update a link permission with displayname", func() {
			gatewayClient.On("GetPublicShare", mock.Anything, mock.MatchedBy(func(req *link.GetPublicShareRequest) bool {
				return req.GetRef().GetId().GetOpaqueId() == "permissionid"
			})).Return(getPublicShareMockResponse, nil)

			gatewayClient.On("Stat", mock.Anything, mock.MatchedBy(func(req *provider.StatRequest) bool {
				return utils.ResourceIDEqual(req.GetRef().GetResourceId(), driveItemId) && req.GetRef().GetPath() == "."
			})).Return(statResponse, nil)

			gatewayClient.On("UpdatePublicShare",
				mock.Anything,
				mock.MatchedBy(func(req *link.UpdatePublicShareRequest) bool {
					if req.GetRef().GetId().GetOpaqueId() == "permissionid" {
						return req.Update.GetDisplayName() == TestLinkName
					}
					return false
				}),
			).Return(updatePublicShareMockResponse, nil)

			link := libregraph.NewSharingLink()
			link.SetLibreGraphDisplayName(TestLinkName)

			driveItemPermission.SetLink(*link)
			res, err := driveItemPermissionsService.UpdatePermission(context.Background(), driveItemId, "permissionid", driveItemPermission)
			Expect(err).ToNot(HaveOccurred())
			Expect(res.Link).ToNot(BeNil())
			Expect(res.Link.GetLibreGraphDisplayName() == TestLinkName)
		})
		It("updates the expiration date", func() {
			gatewayClient.On("Stat", mock.Anything, mock.Anything).Return(statResponse, nil)
			gatewayClient.On("GetUser", mock.Anything, mock.Anything).Return(getUserResponse, nil)
			getPublicShareMockResponse.Share = nil
			getPublicShareMockResponse.Status = status.NewNotFound(ctx, "not found")
			gatewayClient.On("GetPublicShare", mock.Anything, mock.MatchedBy(func(req *link.GetPublicShareRequest) bool {
				return req.GetRef().GetId().GetOpaqueId() == "permissionid"
			})).Return(getPublicShareMockResponse, nil)

			gatewayClient.On("GetShare", mock.Anything, mock.MatchedBy(func(req *collaboration.GetShareRequest) bool {
				return req.GetRef().GetId().GetOpaqueId() == "permissionid"
			})).Return(getShareMockResponse, nil)

			expiration := time.Now().Add(time.Hour)
			updateShareMockResponse.Share.Expiration = utils.TimeToTS(expiration)
			gatewayClient.On("UpdateShare",
				mock.Anything,
				mock.MatchedBy(func(req *collaboration.UpdateShareRequest) bool {
					if req.GetShare().GetId().GetOpaqueId() == "permissionid" {
						return expiration.Equal(utils.TSToTime(req.GetShare().GetExpiration()))
					}
					return false
				}),
			).Return(updateShareMockResponse, nil)

			driveItemPermission.SetExpirationDateTime(expiration)
			res, err := driveItemPermissionsService.UpdatePermission(context.Background(), driveItemId, "permissionid", driveItemPermission)
			Expect(err).ToNot(HaveOccurred())
			Expect(res.GetExpirationDateTime().Equal(expiration)).To(BeTrue())
		})
		It("deletes the expiration date", func() {
			gatewayClient.On("Stat", mock.Anything, mock.Anything).Return(statResponse, nil)
			gatewayClient.On("GetUser", mock.Anything, mock.Anything).Return(getUserResponse, nil)
			getPublicShareMockResponse.Share = nil
			getPublicShareMockResponse.Status = status.NewNotFound(ctx, "not found")
			gatewayClient.On("GetPublicShare", mock.Anything, mock.MatchedBy(func(req *link.GetPublicShareRequest) bool {
				return req.GetRef().GetId().GetOpaqueId() == "permissionid"
			})).Return(getPublicShareMockResponse, nil)

			gatewayClient.On("GetShare", mock.Anything, mock.MatchedBy(func(req *collaboration.GetShareRequest) bool {
				return req.GetRef().GetId().GetOpaqueId() == "permissionid"
			})).Return(getShareMockResponse, nil)
			gatewayClient.On("UpdateShare",
				mock.Anything,
				mock.MatchedBy(func(req *collaboration.UpdateShareRequest) bool {
					return req.GetShare().GetId().GetOpaqueId() == "permissionid"
				}),
			).Return(updateShareMockResponse, nil)

			driveItemPermission.SetExpirationDateTimeNil()
			res, err := driveItemPermissionsService.UpdatePermission(context.Background(), driveItemId, "permissionid", driveItemPermission)
			Expect(err).ToNot(HaveOccurred())
			_, ok := res.GetExpirationDateTimeOk()
			Expect(ok).To(BeFalse())
		})
		It("fails to update the share permissions for a file share when setting a space specific role", func() {
			gatewayClient.On("Stat", mock.Anything, mock.Anything).Return(statResponse, nil)
			getPublicShareMockResponse.Share = nil
			getPublicShareMockResponse.Status = status.NewNotFound(ctx, "not found")
			gatewayClient.On("GetPublicShare", mock.Anything, mock.MatchedBy(func(req *link.GetPublicShareRequest) bool {
				return req.GetRef().GetId().GetOpaqueId() == "permissionid"
			})).Return(getPublicShareMockResponse, nil)

			gatewayClient.On("GetShare", mock.Anything, mock.MatchedBy(func(req *collaboration.GetShareRequest) bool {
				return req.GetRef().GetId().GetOpaqueId() == "permissionid"
			})).Return(getShareMockResponse, nil)

			gatewayClient.On("GetUser", mock.Anything, mock.Anything).Return(getUserResponse, nil)

			driveItemPermission.SetRoles([]string{unifiedrole.UnifiedRoleSpaceViewerID})
			res, err := driveItemPermissionsService.UpdatePermission(context.Background(), driveItemId, "permissionid", driveItemPermission)
			Expect(err).To(MatchError(errorcode.New(errorcode.InvalidRequest, "role not applicable to this resource")))
			Expect(res).To(BeZero())
		})
		It("fails to update the space permissions for a space share when setting a file specific role", func() {
			gatewayClient.On("Stat", mock.Anything, mock.Anything).Return(statResponse, nil)
			getPublicShareMockResponse.Share = nil
			getPublicShareMockResponse.Status = status.NewNotFound(ctx, "not found")
			gatewayClient.On("GetPublicShare", mock.Anything, mock.Anything).Return(getPublicShareMockResponse, nil)

			gatewayClient.On("ListStorageSpaces", mock.Anything, mock.Anything).Return(listSpacesResponse, nil)

			statResponse.Info.Id = listSpacesResponse.StorageSpaces[0].Root
			gatewayClient.On("Stat", mock.Anything, mock.Anything).Return(statResponse, nil)
			gatewayClient.On("GetUser", mock.Anything, mock.Anything).Return(getUserResponse, nil)

			driveItemPermission.SetRoles([]string{unifiedrole.UnifiedRoleFileEditorID})
			spaceId := &provider.ResourceId{
				StorageId: "1",
				SpaceId:   "2",
				OpaqueId:  "2",
			}
			res, err := driveItemPermissionsService.UpdatePermission(ctx, spaceId, "u:userid", driveItemPermission)
			Expect(err).To(MatchError(errorcode.New(errorcode.InvalidRequest, "role not applicable to this resource")))
			Expect(res).To(BeZero())
		})
		It("updates the share permissions when changing the resource permission actions", func() {
			gatewayClient.On("Stat", mock.Anything, mock.Anything).Return(statResponse, nil)
			getPublicShareMockResponse.Share = nil
			getPublicShareMockResponse.Status = status.NewNotFound(ctx, "not found")
			gatewayClient.On("GetPublicShare", mock.Anything, mock.Anything).Return(getPublicShareMockResponse, nil)

			gatewayClient.On("GetShare", mock.Anything, mock.MatchedBy(func(req *collaboration.GetShareRequest) bool {
				return req.GetRef().GetId().GetOpaqueId() == "permissionid"
			})).Return(getShareMockResponse, nil)

			gatewayClient.On("GetUser", mock.Anything, mock.Anything).Return(getUserResponse, nil)

			updateShareMockResponse.Share.Permissions = &collaboration.SharePermissions{
				Permissions: &provider.ResourcePermissions{
					GetPath: true,
				},
			}
			gatewayClient.On("UpdateShare",
				mock.Anything,
				mock.MatchedBy(func(req *collaboration.UpdateShareRequest) bool {
					return req.GetShare().GetId().GetOpaqueId() == "permissionid"
				}),
			).Return(updateShareMockResponse, nil)

			driveItemPermission.SetLibreGraphPermissionsActions([]string{unifiedrole.DriveItemPathRead})
			res, err := driveItemPermissionsService.UpdatePermission(context.Background(), driveItemId, "permissionid", driveItemPermission)
			Expect(err).ToNot(HaveOccurred())
			_, ok := res.GetRolesOk()
			Expect(ok).To(BeFalse())
			_, ok = res.GetLibreGraphPermissionsActionsOk()
			Expect(ok).To(BeTrue())
		})
		It("updates the expiration date on a public share", func() {
			gatewayClient.On("GetPublicShare", mock.Anything, mock.Anything).Return(getPublicShareMockResponse, nil)
			gatewayClient.On("Stat", mock.Anything, mock.Anything).Return(statResponse, nil)

			expiration := time.Now().UTC().Add(time.Hour)
			updatePublicShareMockResponse.Share.Expiration = utils.TimeToTS(expiration)
			gatewayClient.On("UpdatePublicShare",
				mock.Anything,
				mock.MatchedBy(func(req *link.UpdatePublicShareRequest) bool {
					return req.GetRef().GetId().GetOpaqueId() == "permissionid"
				}),
			).Return(updatePublicShareMockResponse, nil)

			driveItemPermission.SetExpirationDateTime(expiration)
			res, err := driveItemPermissionsService.UpdatePermission(context.Background(), driveItemId, "permissionid", driveItemPermission)
			Expect(err).ToNot(HaveOccurred())
			Expect(res.GetExpirationDateTime().Equal(expiration)).To(BeTrue())
		})
		It("updates the permissions on a public share", func() {
			gatewayClient.On("GetPublicShare", mock.Anything, mock.Anything).Return(getPublicShareMockResponse, nil)
			gatewayClient.On("Stat", mock.Anything, mock.Anything).Return(statResponse, nil)

			newLink := libregraph.NewSharingLink()
			newLinkType, err := libregraph.NewSharingLinkTypeFromValue("edit")
			Expect(err).ToNot(HaveOccurred())
			newLink.SetType(*newLinkType)

			updatePublicShareMockResponse.Share.Permissions = &link.PublicSharePermissions{
				Permissions: linktype.NewFolderEditLinkPermissionSet().Permissions,
			}
			gatewayClient.On("UpdatePublicShare",
				mock.Anything,
				mock.MatchedBy(func(req *link.UpdatePublicShareRequest) bool {
					return req.GetRef().GetId().GetOpaqueId() == "permissionid"
				}),
			).Return(updatePublicShareMockResponse, nil)

			driveItemPermission.SetLink(*newLink)
			res, err := driveItemPermissionsService.UpdatePermission(context.Background(), driveItemId, "permissionid", driveItemPermission)
			Expect(err).ToNot(HaveOccurred())
			linkType := res.Link.GetType()
			Expect(string(linkType)).To(Equal("edit"))
		})
		It("updates the public share to internal link", func() {
			gatewayClient.On("Stat", mock.Anything, mock.Anything).Return(statResponse, nil)
			getPublicShareMockResponse.Share = &link.PublicShare{
				Id: &link.PublicShareId{
					OpaqueId: "permissionid",
				},
				ResourceId: &provider.ResourceId{
					StorageId: "1",
					SpaceId:   "2",
					OpaqueId:  "3",
				},
				PasswordProtected: true,
				Permissions: &link.PublicSharePermissions{
					Permissions: linktype.NewFileEditLinkPermissionSet().GetPermissions(),
				},
				Token: "token",
			}
			gatewayClient.On("GetPublicShare",
				mock.Anything,
				mock.MatchedBy(func(req *link.GetPublicShareRequest) bool {
					return req.GetRef().GetId().GetOpaqueId() == "permissionid"
				}),
			).Return(getPublicShareMockResponse, nil)

			newLink := libregraph.NewSharingLink()
			newLinkType, err := libregraph.NewSharingLinkTypeFromValue("internal")
			Expect(err).ToNot(HaveOccurred())
			newLink.SetType(*newLinkType)

			updatePublicShareMockResponse.Share.Permissions = &link.PublicSharePermissions{
				Permissions: linktype.NewInternalLinkPermissionSet().Permissions,
			}
			gatewayClient.On("UpdatePublicShare",
				mock.Anything,
				mock.MatchedBy(func(req *link.UpdatePublicShareRequest) bool {
					return req.GetRef().GetId().GetOpaqueId() == "permissionid"
				}),
			).Return(updatePublicShareMockResponse, nil)

			driveItemPermission.SetLink(*newLink)
			res, err := driveItemPermissionsService.UpdatePermission(context.Background(), driveItemId, "permissionid", driveItemPermission)
			Expect(err).ToNot(HaveOccurred())
			linkType := res.Link.GetType()
			Expect(string(linkType)).To(Equal("internal"))
			pp, hasPP := res.GetHasPasswordOk()
			Expect(hasPP).To(Equal(true))
			Expect(*pp).To(Equal(false))
		})
		It("fails when updating the expiration date on a public share", func() {
			gatewayClient.On("Stat", mock.Anything, mock.Anything).Return(statResponse, nil)
			gatewayClient.On("GetPublicShare", mock.Anything, mock.Anything).Return(getPublicShareMockResponse, nil)
			expiration := time.Now().UTC().AddDate(0, 0, -1)
			updatePublicShareMock := gatewayClient.On("UpdatePublicShare",
				mock.Anything,
				mock.MatchedBy(func(req *link.UpdatePublicShareRequest) bool {
					return req.GetRef().GetId().GetOpaqueId() == "permissionid"
				}),
			)

			updatePublicShareMockResponse.Share = nil
			updatePublicShareMockResponse.Status = status.NewFailedPrecondition(ctx, nil, "expiration date is in the past")
			updatePublicShareMock.Return(updatePublicShareMockResponse, nil)

			driveItemPermission.SetExpirationDateTime(expiration)
			res, err := driveItemPermissionsService.UpdatePermission(context.Background(), driveItemId, "permissionid", driveItemPermission)
			Expect(err).To(MatchError(errorcode.New(errorcode.InvalidRequest, "expiration date is in the past").WithOrigin(errorcode.ErrorOriginCS3)))
			Expect(res).To(BeZero())
		})
	})

})

var _ = Describe("DriveItemPermissionsApi", func() {
	var (
		mockProvider *mocks.DriveItemPermissionsProvider
		httpAPI      svc.DriveItemPermissionsApi
		rCTX         *chi.Context
		invite       libregraph.DriveItemInvite
	)

	BeforeEach(func() {
		logger := log.NewLogger()

		mockProvider = mocks.NewDriveItemPermissionsProvider(GinkgoT())
		api, err := svc.NewDriveItemPermissionsApi(mockProvider, logger, defaults.FullDefaultConfig())
		Expect(err).ToNot(HaveOccurred())

		httpAPI = api

		rCTX = chi.NewRouteContext()
		rCTX.URLParams.Add("driveID", "1$2")

		invite = libregraph.DriveItemInvite{
			Recipients: []libregraph.DriveRecipient{
				{
					ObjectId:                libregraph.PtrString("1"),
					LibreGraphRecipientType: libregraph.PtrString("user")},
			},
			Roles: []string{unifiedrole.UnifiedRoleViewerID},
		}
	})

	checkDriveIDAndItemIDValidation := func(handler http.HandlerFunc) {
		rCTX.URLParams.Add("itemID", "3$4!5")

		responseRecorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodPost, "/", nil).
			WithContext(
				context.WithValue(context.Background(), chi.RouteCtxKey, rCTX),
			)

		handler(responseRecorder, request)

		Expect(responseRecorder.Code).To(Equal(http.StatusUnprocessableEntity))

		jsonData := gjson.Get(responseRecorder.Body.String(), "error")
		Expect(jsonData.Get("message").String()).To(Equal("invalid driveID or itemID"))
	}

	Describe("Invite", func() {
		It("validates the driveID and itemID url param", func() {
			checkDriveIDAndItemIDValidation(httpAPI.Invite)
		})

		It("return an error when the Invite provider errors", func() {
			rCTX.URLParams.Add("itemID", "1$2!3")
			responseRecorder := httptest.NewRecorder()
			inviteJson, err := json.Marshal(invite)
			Expect(err).ToNot(HaveOccurred())

			request := httptest.NewRequest(http.MethodPost, "/", bytes.NewBuffer(inviteJson)).
				WithContext(
					context.WithValue(context.Background(), chi.RouteCtxKey, rCTX),
				)

			onInvite := mockProvider.On("Invite", mock.Anything, mock.Anything, mock.Anything)

			onInvite.Return(func(ctx context.Context, resourceID *provider.ResourceId, invite libregraph.DriveItemInvite) (libregraph.Permission, error) {
				return libregraph.Permission{}, errors.New("any")
			}).Once()

			httpAPI.Invite(responseRecorder, request)

			Expect(responseRecorder.Code).To(Equal(http.StatusInternalServerError))
		})

		It("call the Invite provider with the correct arguments", func() {
			rCTX.URLParams.Add("itemID", "1$2!3")
			responseRecorder := httptest.NewRecorder()
			inviteJson, err := json.Marshal(invite)
			Expect(err).ToNot(HaveOccurred())

			onInvite := mockProvider.On("Invite", mock.Anything, mock.Anything, mock.Anything)
			onInvite.Return(func(ctx context.Context, resourceID *provider.ResourceId, invite libregraph.DriveItemInvite) (libregraph.Permission, error) {
				Expect(storagespace.FormatResourceID(resourceID)).To(Equal("1$2!3"))
				return libregraph.Permission{}, nil
			}).Once()

			request := httptest.NewRequest(http.MethodPost, "/", bytes.NewBuffer(inviteJson)).
				WithContext(
					context.WithValue(context.Background(), chi.RouteCtxKey, rCTX),
				)
			httpAPI.Invite(responseRecorder, request)

			Expect(responseRecorder.Code).To(Equal(http.StatusOK))
		})

		It("fails with unknown or disable role", func() {
			rCTX.URLParams.Add("itemID", "1$2!3")
			responseRecorder := httptest.NewRecorder()
			invite.Roles = []string{unifiedrole.UnifiedRoleViewerID, unifiedrole.UnifiedRoleSecureViewerID}
			inviteJson, err := json.Marshal(invite)
			Expect(err).ToNot(HaveOccurred())

			request := httptest.NewRequest(http.MethodPost, "/", bytes.NewBuffer(inviteJson)).
				WithContext(
					context.WithValue(context.Background(), chi.RouteCtxKey, rCTX),
				)
			httpAPI.Invite(responseRecorder, request)

			Expect(responseRecorder.Code).To(Equal(http.StatusBadRequest))
		})
	})
	Describe("SpaceRootInvite", func() {
		It("call the Invite provider with the correct arguments", func() {
			responseRecorder := httptest.NewRecorder()
			inviteJson, err := json.Marshal(invite)
			Expect(err).ToNot(HaveOccurred())

			onInvite := mockProvider.On("SpaceRootInvite", mock.Anything, mock.Anything, mock.Anything)
			onInvite.Return(func(ctx context.Context, driveID *provider.ResourceId, invite libregraph.DriveItemInvite) (libregraph.Permission, error) {
				Expect(storagespace.FormatResourceID(driveID)).To(Equal("1$2"))
				return libregraph.Permission{}, nil
			}).Once()

			request := httptest.NewRequest(http.MethodPost, "/", bytes.NewBuffer(inviteJson)).
				WithContext(
					context.WithValue(context.Background(), chi.RouteCtxKey, rCTX),
				)
			httpAPI.SpaceRootInvite(responseRecorder, request)

			Expect(responseRecorder.Code).To(Equal(http.StatusOK))
		})
		It("fails with an empty driveid", func() {
			rCTX.URLParams.Add("driveID", "")
			responseRecorder := httptest.NewRecorder()
			inviteJson, err := json.Marshal(invite)
			Expect(err).ToNot(HaveOccurred())

			request := httptest.NewRequest(http.MethodPost, "/", bytes.NewBuffer(inviteJson)).
				WithContext(
					context.WithValue(context.Background(), chi.RouteCtxKey, rCTX),
				)
			httpAPI.SpaceRootInvite(responseRecorder, request)

			Expect(responseRecorder.Code).To(Equal(http.StatusUnprocessableEntity))
		})

		It("fails with unknown or disable role", func() {
			responseRecorder := httptest.NewRecorder()
			invite.Roles = []string{unifiedrole.UnifiedRoleViewerID, unifiedrole.UnifiedRoleSecureViewerID}
			inviteJson, err := json.Marshal(invite)
			Expect(err).ToNot(HaveOccurred())

			request := httptest.NewRequest(http.MethodPost, "/", bytes.NewBuffer(inviteJson)).
				WithContext(
					context.WithValue(context.Background(), chi.RouteCtxKey, rCTX),
				)
			httpAPI.SpaceRootInvite(responseRecorder, request)

			Expect(responseRecorder.Code).To(Equal(http.StatusBadRequest))
		})
	})
	Describe("ListPermissions", func() {
		It("calls the ListPermissions provider with the correct arguments", func() {
			rCTX.URLParams.Add("itemID", "1$2!3")
			responseRecorder := httptest.NewRecorder()
			inviteJson, err := json.Marshal(invite)
			Expect(err).ToNot(HaveOccurred())

			mockProvider.On("ListPermissions", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
				Return(func(ctx context.Context, itemid *provider.ResourceId, opt svc.ListPermissionsQueryOptions) (libregraph.CollectionOfPermissionsWithAllowedValues, error) {
					Expect(storagespace.FormatResourceID(itemid)).To(Equal("1$2!3"))
					return libregraph.CollectionOfPermissionsWithAllowedValues{}, nil
				}).Once()

			request := httptest.NewRequest(http.MethodPost, "/", bytes.NewBuffer(inviteJson)).
				WithContext(
					context.WithValue(context.Background(), chi.RouteCtxKey, rCTX),
				)
			httpAPI.ListPermissions(responseRecorder, request)

			Expect(responseRecorder.Code).To(Equal(http.StatusOK))
		})
		It("fails with an empty itemid", func() {
			responseRecorder := httptest.NewRecorder()
			inviteJson, err := json.Marshal(invite)
			Expect(err).ToNot(HaveOccurred())

			request := httptest.NewRequest(http.MethodPost, "/", bytes.NewBuffer(inviteJson)).
				WithContext(
					context.WithValue(context.Background(), chi.RouteCtxKey, rCTX),
				)
			httpAPI.ListPermissions(responseRecorder, request)

			Expect(responseRecorder.Code).To(Equal(http.StatusBadRequest))
		})
	})
})
