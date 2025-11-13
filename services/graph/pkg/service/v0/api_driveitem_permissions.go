package svc

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"slices"
	"strings"

	"github.com/CiscoM31/godata"
	gateway "github.com/cs3org/go-cs3apis/cs3/gateway/v1beta1"
	grouppb "github.com/cs3org/go-cs3apis/cs3/identity/group/v1beta1"
	userpb "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"
	ocmprovider "github.com/cs3org/go-cs3apis/cs3/ocm/provider/v1beta1"
	collaboration "github.com/cs3org/go-cs3apis/cs3/sharing/collaboration/v1beta1"
	link "github.com/cs3org/go-cs3apis/cs3/sharing/link/v1beta1"
	ocm "github.com/cs3org/go-cs3apis/cs3/sharing/ocm/v1beta1"
	storageprovider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	types "github.com/cs3org/go-cs3apis/cs3/types/v1beta1"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	libregraph "github.com/opencloud-eu/libre-graph-api-go"

	revactx "github.com/opencloud-eu/reva/v2/pkg/ctx"
	"github.com/opencloud-eu/reva/v2/pkg/publicshare"
	"github.com/opencloud-eu/reva/v2/pkg/rgrpc/todo/pool"
	"github.com/opencloud-eu/reva/v2/pkg/share"
	"github.com/opencloud-eu/reva/v2/pkg/storagespace"
	"github.com/opencloud-eu/reva/v2/pkg/utils"

	"github.com/opencloud-eu/opencloud/pkg/l10n"
	l10n_pkg "github.com/opencloud-eu/opencloud/services/graph/pkg/l10n"
	"github.com/opencloud-eu/opencloud/services/graph/pkg/odata"

	"github.com/opencloud-eu/opencloud/pkg/conversions"
	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/services/graph/pkg/config"
	"github.com/opencloud-eu/opencloud/services/graph/pkg/errorcode"
	"github.com/opencloud-eu/opencloud/services/graph/pkg/identity"
	"github.com/opencloud-eu/opencloud/services/graph/pkg/identity/cache"
	"github.com/opencloud-eu/opencloud/services/graph/pkg/unifiedrole"
	"github.com/opencloud-eu/opencloud/services/graph/pkg/validate"
)

const (
	invalidIdMsg              = "invalid driveID or itemID"
	parseDriveIDErrMsg        = "could not parse driveID"
	federatedRolesODataFilter = "@libre.graph.permissions.roles.allowedValues/rolePermissions/any(p:contains(p/condition, '@Subject.UserType==\"Federated\"'))"
	noLinksODataFilter        = "grantedToV2 ne ''"
)

// DriveItemPermissionsProvider contains the methods related to handling permissions on drive items
type DriveItemPermissionsProvider interface {
	Invite(ctx context.Context, resourceId *storageprovider.ResourceId, invite libregraph.DriveItemInvite) (libregraph.Permission, error)
	SpaceRootInvite(ctx context.Context, driveID *storageprovider.ResourceId, invite libregraph.DriveItemInvite) (libregraph.Permission, error)
	ListPermissions(ctx context.Context, itemID *storageprovider.ResourceId, queryOptions ListPermissionsQueryOptions) (libregraph.CollectionOfPermissionsWithAllowedValues, error)
	ListSpaceRootPermissions(ctx context.Context, driveID *storageprovider.ResourceId, queryOptions ListPermissionsQueryOptions) (libregraph.CollectionOfPermissionsWithAllowedValues, error)
	DeletePermission(ctx context.Context, itemID *storageprovider.ResourceId, permissionID string) error
	DeleteSpaceRootPermission(ctx context.Context, driveID *storageprovider.ResourceId, permissionID string) error
	UpdatePermission(ctx context.Context, itemID *storageprovider.ResourceId, permissionID string, newPermission libregraph.Permission) (libregraph.Permission, error)
	UpdateSpaceRootPermission(ctx context.Context, driveID *storageprovider.ResourceId, permissionID string, newPermission libregraph.Permission) (libregraph.Permission, error)
	CreateLink(ctx context.Context, driveItemID *storageprovider.ResourceId, createLink libregraph.DriveItemCreateLink) (libregraph.Permission, error)
	CreateSpaceRootLink(ctx context.Context, driveID *storageprovider.ResourceId, createLink libregraph.DriveItemCreateLink) (libregraph.Permission, error)
	SetPublicLinkPassword(ctx context.Context, driveItemID *storageprovider.ResourceId, permissionID string, password string) (libregraph.Permission, error)
	SetPublicLinkPasswordOnSpaceRoot(ctx context.Context, driveID *storageprovider.ResourceId, permissionID string, password string) (libregraph.Permission, error)
}

// DriveItemPermissionsService contains the production business logic for everything that relates to permissions on drive items.
type DriveItemPermissionsService struct {
	BaseGraphService
}

type permissionType int

const (
	Unknown permissionType = iota
	Public
	User
	Space
	OCM
)

type ListPermissionsQueryOptions struct {
	Count                bool
	NoValues             bool
	NoLinkPermissions    bool
	FilterFederatedRoles bool
	SelectedAttrs        []string
}

// NewDriveItemPermissionsService creates a new DriveItemPermissionsService
func NewDriveItemPermissionsService(logger log.Logger, gatewaySelector pool.Selectable[gateway.GatewayAPIClient], identityCache cache.IdentityCache, config *config.Config) (DriveItemPermissionsService, error) {
	return DriveItemPermissionsService{
		BaseGraphService: BaseGraphService{
			logger:          &log.Logger{Logger: logger.With().Str("graph api", "DrivesDriveItemService").Logger()},
			gatewaySelector: gatewaySelector,
			identityCache:   identityCache,
			config:          config,
			availableRoles:  unifiedrole.GetRoles(unifiedrole.RoleFilterIDs(config.UnifiedRoles.AvailableRoles...)),
		},
	}, nil
}

// Invite invites a user to a drive item.
func (s DriveItemPermissionsService) Invite(ctx context.Context, resourceId *storageprovider.ResourceId, invite libregraph.DriveItemInvite) (libregraph.Permission, error) {
	tenantId := revactx.ContextMustGetUser(ctx).GetId().GetTenantId()
	gatewayClient, err := s.gatewaySelector.Next()
	if err != nil {
		return libregraph.Permission{}, err
	}

	statResponse, err := gatewayClient.Stat(ctx, &storageprovider.StatRequest{Ref: &storageprovider.Reference{ResourceId: resourceId}})
	if err := errorcode.FromStat(statResponse, err); err != nil {
		s.logger.Warn().Err(err).Interface("stat.res", statResponse).Msg("stat failed")
		return libregraph.Permission{}, err
	}

	var condition string
	if condition, err = roleConditionForResourceType(statResponse.GetInfo()); err != nil {
		return libregraph.Permission{}, err
	}

	unifiedRolePermissions := []*libregraph.UnifiedRolePermission{{AllowedResourceActions: invite.LibreGraphPermissionsActions}}
	for _, roleID := range invite.GetRoles() {
		// only allow roles that are enabled in the config
		if !slices.Contains(s.config.UnifiedRoles.AvailableRoles, roleID) {
			return libregraph.Permission{}, unifiedrole.ErrUnknownRole
		}

		role, err := unifiedrole.GetRole(unifiedrole.RoleFilterIDs(roleID))
		if err != nil {
			s.logger.Debug().Err(err).Interface("role", invite.GetRoles()[0]).Msg("unable to convert requested role")
			return libregraph.Permission{}, err
		}

		allowedResourceActions := unifiedrole.GetAllowedResourceActions(role, condition)
		if len(allowedResourceActions) == 0 && role.GetId() != unifiedrole.UnifiedRoleDeniedID {
			return libregraph.Permission{}, errorcode.New(errorcode.InvalidRequest, "role not applicable to this resource")
		}

		unifiedRolePermissions = append(unifiedRolePermissions, conversions.ToPointerSlice(role.GetRolePermissions())...)
	}

	driveRecipient := invite.GetRecipients()[0]

	objectID := driveRecipient.GetObjectId()
	cs3ResourcePermissions := unifiedrole.PermissionsToCS3ResourcePermissions(unifiedRolePermissions)

	permission := &libregraph.Permission{}
	availableRoles := unifiedrole.GetRoles(unifiedrole.RoleFilterIDs(s.config.UnifiedRoles.AvailableRoles...))
	if role := unifiedrole.CS3ResourcePermissionsToRole(availableRoles, cs3ResourcePermissions, condition, false); role != nil {
		permission.Roles = []string{role.GetId()}
	}

	if len(permission.GetRoles()) == 0 {
		permission.LibreGraphPermissionsActions = unifiedrole.CS3ResourcePermissionsToLibregraphActions(cs3ResourcePermissions)
	}

	var shareid string
	var expiration *types.Timestamp
	var cTime *types.Timestamp
	switch driveRecipient.GetLibreGraphRecipientType() {
	case "group":
		group, err := s.identityCache.GetGroup(ctx, objectID)
		if err != nil {
			s.logger.Debug().Err(err).Interface("groupId", objectID).Msg("failed group lookup")
			return libregraph.Permission{}, errorcode.New(errorcode.InvalidRequest, err.Error())
		}
		permission.GrantedToV2 = &libregraph.SharePointIdentitySet{
			Group: &libregraph.Identity{
				DisplayName: group.GetDisplayName(),
				Id:          conversions.ToPointer(group.GetId()),
			},
		}
		createShareRequest := createShareRequestToGroup(group, statResponse.GetInfo(), cs3ResourcePermissions)
		if invite.ExpirationDateTime != nil {
			createShareRequest.GetGrant().Expiration = utils.TimeToTS(*invite.ExpirationDateTime)
		}
		createShareResponse, err := gatewayClient.CreateShare(ctx, createShareRequest)
		if err := errorcode.FromCS3Status(createShareResponse.GetStatus(), err); err != nil {
			s.logger.Debug().Err(err).Msg("share creation failed")
			return libregraph.Permission{}, err
		}
		shareid = createShareResponse.GetShare().GetId().GetOpaqueId()
		cTime = createShareResponse.GetShare().GetCtime()
		expiration = createShareResponse.GetShare().GetExpiration()
	default:
		user, err := s.identityCache.GetCS3User(ctx, tenantId, objectID)
		if errors.Is(err, identity.ErrNotFound) && s.config.IncludeOCMSharees {
			user, err = s.identityCache.GetAcceptedCS3User(ctx, objectID)
			if err == nil && IsSpaceRoot(statResponse.GetInfo().GetId()) {
				return libregraph.Permission{}, errorcode.New(errorcode.InvalidRequest, "federated user can not become a space member")
			}
		}
		if err != nil {
			s.logger.Debug().Err(err).Interface("userId", objectID).Msg("failed user lookup")
			return libregraph.Permission{}, errorcode.New(errorcode.InvalidRequest, err.Error())
		}
		permission.GrantedToV2 = &libregraph.SharePointIdentitySet{
			User: &libregraph.Identity{
				DisplayName:        user.GetDisplayName(),
				Id:                 conversions.ToPointer(user.GetId().GetOpaqueId()),
				LibreGraphUserType: conversions.ToPointer(identity.CS3UserTypeToGraph(user.GetId().GetType())),
			},
		}

		if user.GetId().GetType() == userpb.UserType_USER_TYPE_FEDERATED {
			providerInfoResp, err := gatewayClient.GetInfoByDomain(ctx, &ocmprovider.GetInfoByDomainRequest{
				Domain: user.GetId().GetIdp(),
			})
			if err = errorcode.FromCS3Status(providerInfoResp.GetStatus(), err); err != nil {
				s.logger.Error().Err(err).Msg("getting provider info failed")
				return libregraph.Permission{}, err
			}

			createShareRequest := createShareRequestToFederatedUser(user, statResponse.GetInfo().GetId(), providerInfoResp.ProviderInfo, cs3ResourcePermissions)
			if invite.ExpirationDateTime != nil {
				createShareRequest.Expiration = utils.TimeToTS(*invite.ExpirationDateTime)
			}
			createShareResponse, err := gatewayClient.CreateOCMShare(ctx, createShareRequest)
			if err = errorcode.FromCS3Status(createShareResponse.GetStatus(), err); err != nil {
				s.logger.Error().Err(err).Msg("share creation failed")
				return libregraph.Permission{}, err
			}
			shareid = createShareResponse.GetShare().GetId().GetOpaqueId()
			cTime = createShareResponse.GetShare().GetCtime()
			expiration = createShareResponse.GetShare().GetExpiration()
		} else {
			createShareRequest := createShareRequestToUser(user, statResponse.GetInfo(), cs3ResourcePermissions)
			if invite.ExpirationDateTime != nil {
				createShareRequest.GetGrant().Expiration = utils.TimeToTS(*invite.ExpirationDateTime)
			}
			createShareResponse, err := gatewayClient.CreateShare(ctx, createShareRequest)
			if err = errorcode.FromCS3Status(createShareResponse.GetStatus(), err); err != nil {
				s.logger.Error().Err(err).Msg("share creation failed")
				return libregraph.Permission{}, err
			}
			shareid = createShareResponse.GetShare().GetId().GetOpaqueId()
			cTime = createShareResponse.GetShare().GetCtime()
			expiration = createShareResponse.GetShare().GetExpiration()
		}

	}

	if shareid != "" {
		permission.Id = conversions.ToPointer(shareid)
	} else if IsSpaceRoot(statResponse.GetInfo().GetId()) {
		// permissions on a space root are not handled by a share manager so
		// they don't get a share-id
		permission.SetId(identitySetToSpacePermissionID(permission.GetGrantedToV2()))
	}

	if expiration != nil {
		permission.SetExpirationDateTime(utils.TSToTime(expiration))
	}

	// set cTime
	if cTime != nil {
		permission.SetCreatedDateTime(cs3TimestampToTime(cTime))
	}

	if user, ok := revactx.ContextGetUser(ctx); ok {
		userIdentity, err := userIdToIdentity(ctx, s.identityCache, tenantId, user.GetId().GetOpaqueId())
		if err != nil {
			s.logger.Error().Err(err).Msg("identity lookup failed")
			return libregraph.Permission{}, errorcode.New(errorcode.InvalidRequest, err.Error())
		}
		permission.SetInvitation(libregraph.SharingInvitation{
			InvitedBy: &libregraph.IdentitySet{
				User: &userIdentity,
			},
		})
	}

	return *permission, nil
}

func createShareRequestToGroup(group libregraph.Group, info *storageprovider.ResourceInfo, cs3ResourcePermissions *storageprovider.ResourcePermissions) *collaboration.CreateShareRequest {
	return &collaboration.CreateShareRequest{
		ResourceInfo: info,
		Grant: &collaboration.ShareGrant{
			Grantee: &storageprovider.Grantee{
				Type: storageprovider.GranteeType_GRANTEE_TYPE_GROUP,
				Id: &storageprovider.Grantee_GroupId{GroupId: &grouppb.GroupId{
					OpaqueId: group.GetId(),
				}},
			},
			Permissions: &collaboration.SharePermissions{
				Permissions: cs3ResourcePermissions,
			},
		},
	}
}

func createShareRequestToUser(user *userpb.User, info *storageprovider.ResourceInfo, cs3ResourcePermissions *storageprovider.ResourcePermissions) *collaboration.CreateShareRequest {
	return &collaboration.CreateShareRequest{
		ResourceInfo: info,
		Grant: &collaboration.ShareGrant{
			Grantee: &storageprovider.Grantee{
				Type: storageprovider.GranteeType_GRANTEE_TYPE_USER,
				Id: &storageprovider.Grantee_UserId{
					UserId: user.GetId(),
				},
			},
			Permissions: &collaboration.SharePermissions{
				Permissions: cs3ResourcePermissions,
			},
		},
	}
}

func createShareRequestToFederatedUser(user *userpb.User, resourceId *storageprovider.ResourceId, providerInfo *ocmprovider.ProviderInfo, cs3ResourcePermissions *storageprovider.ResourcePermissions) *ocm.CreateOCMShareRequest {
	return &ocm.CreateOCMShareRequest{
		ResourceId: resourceId,
		Grantee: &storageprovider.Grantee{
			Type: storageprovider.GranteeType_GRANTEE_TYPE_USER,
			Id: &storageprovider.Grantee_UserId{
				UserId: user.GetId(),
			},
		},
		RecipientMeshProvider: providerInfo,
		AccessMethods: []*ocm.AccessMethod{
			{
				Term: &ocm.AccessMethod_WebdavOptions{
					WebdavOptions: &ocm.WebDAVAccessMethod{
						Permissions: cs3ResourcePermissions,
					},
				},
			},
		},
	}
}

// SpaceRootInvite handles invitation request on project spaces
func (s DriveItemPermissionsService) SpaceRootInvite(ctx context.Context, driveID *storageprovider.ResourceId, invite libregraph.DriveItemInvite) (libregraph.Permission, error) {
	gatewayClient, err := s.gatewaySelector.Next()
	if err != nil {
		return libregraph.Permission{}, err
	}

	space, err := utils.GetSpace(ctx, storagespace.FormatResourceID(driveID), gatewayClient)
	if err != nil {
		return libregraph.Permission{}, errorcode.FromUtilsStatusCodeError(err)
	}

	if space.SpaceType != _spaceTypeProject {
		return libregraph.Permission{}, errorcode.New(errorcode.InvalidRequest, "unsupported space type")
	}

	rootResourceID := space.GetRoot()
	return s.Invite(ctx, rootResourceID, invite)
}

// ListPermissions lists the permissions of a driveItem
func (s DriveItemPermissionsService) ListPermissions(ctx context.Context, itemID *storageprovider.ResourceId, queryOptions ListPermissionsQueryOptions) (libregraph.CollectionOfPermissionsWithAllowedValues, error) {
	collectionOfPermissions := libregraph.CollectionOfPermissionsWithAllowedValues{}
	gatewayClient, err := s.gatewaySelector.Next()
	if err != nil {
		return collectionOfPermissions, err
	}

	statResponse, err := gatewayClient.Stat(ctx, &storageprovider.StatRequest{Ref: &storageprovider.Reference{ResourceId: itemID}})
	if err := errorcode.FromStat(statResponse, err); err != nil {
		s.logger.Warn().Err(err).Interface("stat.res", statResponse).Msg("stat failed")
		return collectionOfPermissions, err
	}

	var condition string
	if condition, err = roleConditionForResourceType(statResponse.GetInfo()); err != nil {
		return collectionOfPermissions, err
	}

	permissionSet := statResponse.GetInfo().GetPermissionSet()
	allowedActions := unifiedrole.CS3ResourcePermissionsToLibregraphActions(permissionSet)

	collectionOfPermissions = libregraph.CollectionOfPermissionsWithAllowedValues{}

	if len(queryOptions.SelectedAttrs) == 0 || slices.Contains(queryOptions.SelectedAttrs, "@libre.graph.permissions.actions.allowedValues") {
		collectionOfPermissions.LibreGraphPermissionsActionsAllowedValues = allowedActions
	}

	if len(queryOptions.SelectedAttrs) == 0 || slices.Contains(queryOptions.SelectedAttrs, "@libre.graph.permissions.roles.allowedValues") {
		collectionOfPermissions.LibreGraphPermissionsRolesAllowedValues = conversions.ToValueSlice(
			unifiedrole.GetRolesByPermissions(
				unifiedrole.GetRoles(unifiedrole.RoleFilterIDs(s.config.UnifiedRoles.AvailableRoles...)),
				allowedActions,
				condition,
				queryOptions.FilterFederatedRoles,
				false,
			),
		)
	}

	for i, definition := range collectionOfPermissions.LibreGraphPermissionsRolesAllowedValues {
		// the openapi spec defines that the rolePermissions should not be part of the response
		definition.RolePermissions = nil
		collectionOfPermissions.LibreGraphPermissionsRolesAllowedValues[i] = definition
	}

	if len(queryOptions.SelectedAttrs) > 0 {
		// no need to fetch shares, we are only interested allowedActions and/or allowedRoles
		return collectionOfPermissions, nil
	}

	driveItems := make(driveItemsByResourceID, 1)
	// we can use the statResponse to build the drive item before fetching the shares
	item, err := cs3ResourceToDriveItem(s.logger, statResponse.GetInfo())
	if err != nil {
		return collectionOfPermissions, err
	}
	driveItems[storagespace.FormatResourceID(statResponse.GetInfo().GetId())] = *item

	var permissionsCount int

	if IsSpaceRoot(statResponse.GetInfo().GetId()) {
		var permissions []libregraph.Permission
		permissions, permissionsCount, err = s.getSpaceRootPermissions(ctx, statResponse.GetInfo().GetSpace().GetId(), queryOptions.NoValues)
		if err != nil {
			return collectionOfPermissions, err
		}
		collectionOfPermissions.Value = permissions
	} else {
		// "normal" driveItem, populate user  permissions via share providers
		driveItems, err = s.listUserShares(ctx, []*collaboration.Filter{
			share.ResourceIDFilter(itemID),
		}, driveItems)
		if err != nil {
			return collectionOfPermissions, err
		}
		if s.config.IncludeOCMSharees {
			driveItems, err = s.listOCMShares(ctx, []*ocm.ListOCMSharesRequest_Filter{
				{
					Type: ocm.ListOCMSharesRequest_Filter_TYPE_RESOURCE_ID,
					Term: &ocm.ListOCMSharesRequest_Filter_ResourceId{ResourceId: itemID},
				},
			}, driveItems)
			if err != nil {
				return collectionOfPermissions, err
			}
		}
	}

	if !queryOptions.NoLinkPermissions {
		// finally get public shares, which are possible for spaceroots and "normal" resources
		driveItems, err = s.listPublicShares(ctx, []*link.ListPublicSharesRequest_Filter{
			publicshare.ResourceIDFilter(itemID),
		}, driveItems)
		if err != nil {
			return collectionOfPermissions, err
		}
	}

	for _, driveItem := range driveItems {
		permissionsCount += len(driveItem.Permissions)
		if !queryOptions.NoValues {
			collectionOfPermissions.Value = append(collectionOfPermissions.Value, driveItem.Permissions...)
		}
	}

	if queryOptions.Count {
		collectionOfPermissions.SetOdataCount(int32(permissionsCount))
	}

	return collectionOfPermissions, nil
}

// ListSpaceRootPermissions handles ListPermissions request on project spaces
func (s DriveItemPermissionsService) ListSpaceRootPermissions(ctx context.Context, driveID *storageprovider.ResourceId, queryOptions ListPermissionsQueryOptions) (libregraph.CollectionOfPermissionsWithAllowedValues, error) {
	collectionOfPermissions := libregraph.CollectionOfPermissionsWithAllowedValues{}
	gatewayClient, err := s.gatewaySelector.Next()
	if err != nil {
		return collectionOfPermissions, err
	}

	space, err := utils.GetSpace(ctx, storagespace.FormatResourceID(driveID), gatewayClient)
	if err != nil {
		return collectionOfPermissions, errorcode.FromUtilsStatusCodeError(err)
	}

	isSupportedSpaceType := slices.Contains([]string{_spaceTypeProject, _spaceTypePersonal, _spaceTypeVirtual}, space.GetSpaceType())
	if !isSupportedSpaceType {
		return collectionOfPermissions, errorcode.New(errorcode.InvalidRequest, "unsupported space type")
	}

	rootResourceID := space.GetRoot()
	return s.ListPermissions(ctx, rootResourceID, queryOptions) // federated roles are not supported for spaces
}

// DeletePermission deletes a permission from a drive item
func (s DriveItemPermissionsService) DeletePermission(ctx context.Context, itemID *storageprovider.ResourceId, permissionID string) error {
	var permissionType permissionType

	sharedResourceID, err := s.getLinkPermissionResourceID(ctx, permissionID)
	switch {
	// Check if the ID is referring to a public share
	case err == nil:
		permissionType = Public
	// If the item id is referring to a space root and this is not a public share
	// we have to deal with space permissions
	case IsSpaceRoot(itemID):
		permissionType = Space
		sharedResourceID = itemID
		err = nil
	// If this is neither a public share nor a space permission, check if this is a
	// user share
	default:
		sharedResourceID, err = s.getUserPermissionResourceID(ctx, permissionID)
		if err == nil {
			permissionType = User
		}
	}

	if sharedResourceID == nil && s.config.IncludeOCMSharees {
		sharedResourceID, err = s.getOCMPermissionResourceID(ctx, permissionID)
		if err == nil {
			permissionType = OCM
		}
	}

	switch {
	case err != nil:
		return err
	case permissionType == Unknown:
		return errorcode.New(errorcode.ItemNotFound, "permission not found")
	case sharedResourceID == nil:
		return errorcode.New(errorcode.ItemNotFound, "failed to resolve resource id for shared resource")
	}

	// The resourceID of the shared resource need to match the item ID from the Request Path
	// otherwise this is an invalid Request.
	if !utils.ResourceIDEqual(sharedResourceID, itemID) {
		s.logger.Debug().Msg("resourceID of shared does not match itemID")
		return errorcode.New(errorcode.InvalidRequest, "permissionID and itemID do not match")
	}

	switch permissionType {
	case User:
		return s.removeUserShare(ctx, permissionID)
	case Public:
		return s.removePublicShare(ctx, permissionID)
	case Space:
		return s.removeSpacePermission(ctx, permissionID, sharedResourceID)
	case OCM:
		return s.removeOCMPermission(ctx, permissionID)
	default:
		// This should never be reached
		return errorcode.New(errorcode.GeneralException, "failed to delete permission")
	}
}

// DeleteSpaceRootPermission deletes a permission on the root item of a project space
func (s DriveItemPermissionsService) DeleteSpaceRootPermission(ctx context.Context, driveID *storageprovider.ResourceId, permissionID string) error {
	gatewayClient, err := s.gatewaySelector.Next()
	if err != nil {
		return err
	}

	space, err := utils.GetSpace(ctx, storagespace.FormatResourceID(driveID), gatewayClient)
	if err != nil {
		return errorcode.FromUtilsStatusCodeError(err)
	}

	if space.SpaceType != _spaceTypeProject {
		return errorcode.New(errorcode.InvalidRequest, "unsupported space type")
	}

	rootResourceID := space.GetRoot()
	return s.DeletePermission(ctx, rootResourceID, permissionID)
}

// UpdatePermission updates a permission on a drive item
func (s DriveItemPermissionsService) UpdatePermission(ctx context.Context, itemID *storageprovider.ResourceId, permissionID string, newPermission libregraph.Permission) (libregraph.Permission, error) {
	oldPermission, sharedResourceID, err := s.getPermissionByID(ctx, permissionID, itemID)

	// try to get the permission from ocm if the permission was not found first place
	if err != nil && s.config.IncludeOCMSharees {
		oldPermission, sharedResourceID, err = s.getOCMPermissionByID(ctx, permissionID, itemID)
	}

	// if we still can't find the permission, return an error
	if err != nil {
		return libregraph.Permission{}, err
	}

	// The resourceID of the shared resource need to match the item ID from the Request Path
	// otherwise this is an invalid Request.
	if !utils.ResourceIDEqual(sharedResourceID, itemID) {
		s.logger.Debug().Msg("resourceID of shared does not match itemID")
		return libregraph.Permission{}, errorcode.New(errorcode.InvalidRequest, "permissionID and itemID do not match")
	}

	// This is a public link
	if _, ok := oldPermission.GetLinkOk(); ok {
		updatedPermission, err := s.updatePublicLinkPermission(ctx, permissionID, itemID, &newPermission)
		if err != nil {
			return libregraph.Permission{}, err
		}
		return *updatedPermission, nil
	}

	// This is a user share
	updatedPermission, err := s.updateUserShare(ctx, permissionID, sharedResourceID, &newPermission)
	if err == nil && updatedPermission != nil {
		return *updatedPermission, nil
	}

	// This is an ocm share
	if s.config.IncludeOCMSharees {
		updatePermission, err := s.updateOCMPermission(ctx, permissionID, itemID, &newPermission)
		if err == nil {
			return *updatePermission, nil
		}
	}

	return libregraph.Permission{}, err

}

// UpdateSpaceRootPermission updates a permission on the root item of a project space
func (s DriveItemPermissionsService) UpdateSpaceRootPermission(ctx context.Context, driveID *storageprovider.ResourceId, permissionID string, newPermission libregraph.Permission) (libregraph.Permission, error) {
	gatewayClient, err := s.gatewaySelector.Next()
	if err != nil {
		return libregraph.Permission{}, err
	}

	space, err := utils.GetSpace(ctx, storagespace.FormatResourceID(driveID), gatewayClient)
	if err != nil {
		return libregraph.Permission{}, errorcode.FromUtilsStatusCodeError(err)
	}

	if space.SpaceType != _spaceTypeProject {
		return libregraph.Permission{}, errorcode.New(errorcode.InvalidRequest, "unsupported space type")
	}

	rootResourceID := space.GetRoot()
	return s.UpdatePermission(ctx, rootResourceID, permissionID, newPermission)
}

// DriveItemPermissionsApi is the api that registers the http endpoints which expose needed operation to the graph api.
// the business logic is delegated to the permissions service and further down to the cs3 client.
type DriveItemPermissionsApi struct {
	logger                      log.Logger
	driveItemPermissionsService DriveItemPermissionsProvider
	config                      *config.Config
}

// NewDriveItemPermissionsApi creates a new DriveItemPermissionsApi
func NewDriveItemPermissionsApi(driveItemPermissionService DriveItemPermissionsProvider, logger log.Logger, c *config.Config) (DriveItemPermissionsApi, error) {
	return DriveItemPermissionsApi{
		logger:                      log.Logger{Logger: logger.With().Str("graph api", "DrivesDriveItemApi").Logger()},
		driveItemPermissionsService: driveItemPermissionService,
		config:                      c,
	}, nil
}

// Invite handles DriveItemInvite requests
func (api DriveItemPermissionsApi) Invite(w http.ResponseWriter, r *http.Request) {
	_, itemID, err := GetDriveAndItemIDParam(r, &api.logger)
	if err != nil {
		api.logger.Debug().Err(err).Msg(invalidIdMsg)
		errorcode.InvalidRequest.Render(w, r, http.StatusUnprocessableEntity, invalidIdMsg)
		return
	}

	driveItemInvite := &libregraph.DriveItemInvite{}
	if err = StrictJSONUnmarshal(r.Body, driveItemInvite); err != nil {
		api.logger.Debug().Err(err).Interface("Body", r.Body).Msg("failed unmarshalling request body")
		errorcode.InvalidRequest.Render(w, r, http.StatusBadRequest, "invalid request body")
		return
	}

	ctx := validate.ContextWithAllowedRoleIDs(r.Context(), api.config.UnifiedRoles.AvailableRoles)
	if err = validate.StructCtx(ctx, driveItemInvite); err != nil {
		api.logger.Debug().Err(err).Interface("Body", r.Body).Msg("invalid request body")
		errorcode.InvalidRequest.Render(w, r, http.StatusBadRequest, err.Error())
		return
	}

	permission, err := api.driveItemPermissionsService.Invite(ctx, itemID, *driveItemInvite)
	if err != nil {
		errorcode.RenderError(w, r, err)
		return
	}

	render.Status(r, http.StatusOK)
	render.JSON(w, r, &ListResponse{Value: []interface{}{permission}})
}

// SpaceRootInvite handles DriveItemInvite requests on a space root
func (api DriveItemPermissionsApi) SpaceRootInvite(w http.ResponseWriter, r *http.Request) {
	driveID, err := parseIDParam(r, "driveID")
	if err != nil {
		api.logger.Debug().Err(err).Msg(parseDriveIDErrMsg)
		errorcode.InvalidRequest.Render(w, r, http.StatusUnprocessableEntity, parseDriveIDErrMsg)
		return
	}

	driveItemInvite := &libregraph.DriveItemInvite{}
	if err = StrictJSONUnmarshal(r.Body, driveItemInvite); err != nil {
		api.logger.Debug().Err(err).Interface("Body", r.Body).Msg("failed unmarshalling request body")
		errorcode.InvalidRequest.Render(w, r, http.StatusBadRequest, "invalid request body")
		return
	}

	ctx := validate.ContextWithAllowedRoleIDs(r.Context(), api.config.UnifiedRoles.AvailableRoles)
	if err = validate.StructCtx(ctx, driveItemInvite); err != nil {
		api.logger.Debug().Err(err).Interface("Body", r.Body).Msg("invalid request body")
		errorcode.InvalidRequest.Render(w, r, http.StatusBadRequest, err.Error())
		return
	}
	permission, err := api.driveItemPermissionsService.SpaceRootInvite(ctx, &driveID, *driveItemInvite)

	if err != nil {
		errorcode.RenderError(w, r, err)
		return
	}

	render.Status(r, http.StatusOK)
	render.JSON(w, r, &ListResponse{Value: []interface{}{permission}})
}

// ListPermissions handles ListPermissions requests
func (api DriveItemPermissionsApi) ListPermissions(w http.ResponseWriter, r *http.Request) {
	_, itemID, err := GetDriveAndItemIDParam(r, &api.logger)
	if err != nil {
		api.logger.Debug().Err(err).Msg(invalidIdMsg)
		errorcode.RenderError(w, r, err)
		return
	}

	sanitizedPath := strings.TrimPrefix(r.URL.Path, "/graph/v1.0/")
	odataReq, err := godata.ParseRequest(r.Context(), sanitizedPath, r.URL.Query())
	if err != nil {
		api.logger.Debug().Err(err).Interface("query", r.URL.Query()).Msg("Error parsing ListPermissionRequest: query error")
		errorcode.InvalidRequest.Render(w, r, http.StatusBadRequest, err.Error())
		return
	}

	var queryOptions ListPermissionsQueryOptions
	queryOptions, err = api.getListPermissionsQueryOptions(odataReq)
	if err != nil {
		api.logger.Debug().Err(err).Interface("query", r.URL.Query()).Msg("Error parsing ListPermissionRequest query options")
		errorcode.InvalidRequest.Render(w, r, http.StatusBadRequest, err.Error())
		return
	}

	ctx := r.Context()

	permissions, err := api.driveItemPermissionsService.ListPermissions(ctx, itemID, queryOptions)
	if err != nil {
		errorcode.RenderError(w, r, err)
		return
	}

	loc := r.Header.Get(l10n.HeaderAcceptLanguage)
	w.Header().Add("Content-Language", loc)
	if loc != "" && loc != "en" {
		err := l10n_pkg.TranslateEntity(loc, "en", permissions,
			l10n.TranslateEach("LibreGraphPermissionsRolesAllowedValues",
				l10n.TranslateField("Description"),
				l10n.TranslateField("DisplayName"),
			),
		)
		if err != nil {
			api.logger.Error().Err(err).Msg("tranlation error")
		}
	}

	render.Status(r, http.StatusOK)
	render.JSON(w, r, permissions)
}

// ListSpaceRootPermissions handles ListPermissions requests on a space root
func (api DriveItemPermissionsApi) ListSpaceRootPermissions(w http.ResponseWriter, r *http.Request) {
	driveID, err := parseIDParam(r, "driveID")
	if err != nil {
		api.logger.Debug().Err(err).Msg(parseDriveIDErrMsg)
		errorcode.InvalidRequest.Render(w, r, http.StatusUnprocessableEntity, parseDriveIDErrMsg)
		return
	}

	sanitizedPath := strings.TrimPrefix(r.URL.Path, "/graph/v1.0/")
	odataReq, err := godata.ParseRequest(r.Context(), sanitizedPath, r.URL.Query())
	if err != nil {
		api.logger.Debug().Err(err).Interface("query", r.URL.Query()).Msg("Error parsing ListPermissionRequest: query error")
		errorcode.InvalidRequest.Render(w, r, http.StatusBadRequest, err.Error())
		return
	}

	var queryOptions ListPermissionsQueryOptions
	queryOptions, err = api.getListPermissionsQueryOptions(odataReq)
	if err != nil {
		api.logger.Debug().Err(err).Interface("query", r.URL.Query()).Msg("Error parsing ListPermissionRequest query options")
		errorcode.InvalidRequest.Render(w, r, http.StatusBadRequest, err.Error())
		return
	}

	ctx := r.Context()
	permissions, err := api.driveItemPermissionsService.ListSpaceRootPermissions(ctx, &driveID, queryOptions)

	if err != nil {
		errorcode.RenderError(w, r, err)
		return
	}

	loc := r.Header.Get(l10n.HeaderAcceptLanguage)
	w.Header().Add("Content-Language", loc)
	if loc != "" && loc != "en" {
		err := l10n_pkg.TranslateEntity(loc, "en", permissions,
			l10n.TranslateEach("LibreGraphPermissionsRolesAllowedValues",
				l10n.TranslateField("Description"),
				l10n.TranslateField("DisplayName"),
			),
		)
		if err != nil {
			api.logger.Error().Err(err).Msg("tranlation error")
		}
	}

	render.Status(r, http.StatusOK)
	render.JSON(w, r, permissions)
}

func (api DriveItemPermissionsApi) getListPermissionsQueryOptions(odataReq *godata.GoDataRequest) (ListPermissionsQueryOptions, error) {
	queryOptions := ListPermissionsQueryOptions{}
	if odataReq.Query.Filter != nil {
		switch odataReq.Query.Filter.RawValue {
		case federatedRolesODataFilter:
			queryOptions.FilterFederatedRoles = true
		case noLinksODataFilter:
			queryOptions.NoLinkPermissions = true
		default:
			return ListPermissionsQueryOptions{}, errorcode.New(errorcode.InvalidRequest, "invalid filter value")
		}
	}

	selectAttrs, err := odata.GetSelectValues(odataReq.Query)
	if err != nil {
		return ListPermissionsQueryOptions{}, err
	}

	queryOptions.SelectedAttrs = selectAttrs
	if odataReq.Query.Count != nil {
		queryOptions.Count = bool(*odataReq.Query.Count)
	}
	if odataReq.Query.Top != nil {
		top := int(*odataReq.Query.Top)
		switch {
		case top != 0:
			return ListPermissionsQueryOptions{}, err
		case top == 0 && !queryOptions.Count:
			return ListPermissionsQueryOptions{}, err
		default:
			queryOptions.NoValues = true
		}
	}
	return queryOptions, nil
}

// DeletePermission handles DeletePermission requests
func (api DriveItemPermissionsApi) DeletePermission(w http.ResponseWriter, r *http.Request) {
	_, itemID, err := GetDriveAndItemIDParam(r, &api.logger)
	if err != nil {
		api.logger.Debug().Err(err).Msg(invalidIdMsg)
		errorcode.RenderError(w, r, err)
		return
	}

	permissionID, err := url.PathUnescape(chi.URLParam(r, "permissionID"))
	if err != nil {
		api.logger.Debug().Err(err).Msg("could not parse permissionID")
		errorcode.InvalidRequest.Render(w, r, http.StatusBadRequest, "invalid permissionID")
		return
	}

	ctx := r.Context()
	err = api.driveItemPermissionsService.DeletePermission(ctx, itemID, permissionID)
	if err != nil {
		errorcode.RenderError(w, r, err)
		return
	}

	render.Status(r, http.StatusNoContent)
	render.NoContent(w, r)
}

// DeleteSpaceRootPermission handles DeletePermission requests on a space root
func (api DriveItemPermissionsApi) DeleteSpaceRootPermission(w http.ResponseWriter, r *http.Request) {
	driveID, err := parseIDParam(r, "driveID")
	if err != nil {
		api.logger.Debug().Err(err).Msg(parseDriveIDErrMsg)
		errorcode.InvalidRequest.Render(w, r, http.StatusUnprocessableEntity, parseDriveIDErrMsg)
		return
	}

	permissionID, err := url.PathUnescape(chi.URLParam(r, "permissionID"))
	if err != nil {
		api.logger.Debug().Err(err).Msg("could not parse permissionID")
		errorcode.InvalidRequest.Render(w, r, http.StatusBadRequest, "invalid permissionID")
		return
	}

	ctx := r.Context()
	err = api.driveItemPermissionsService.DeleteSpaceRootPermission(ctx, &driveID, permissionID)
	if err != nil {
		errorcode.RenderError(w, r, err)
		return
	}

	render.Status(r, http.StatusNoContent)
	render.NoContent(w, r)
}

// UpdatePermission handles UpdatePermission requests
func (api DriveItemPermissionsApi) UpdatePermission(w http.ResponseWriter, r *http.Request) {
	_, itemID, err := GetDriveAndItemIDParam(r, &api.logger)
	if err != nil {
		api.logger.Debug().Err(err).Msg(invalidIdMsg)
		errorcode.RenderError(w, r, err)
		return
	}

	permissionID, err := url.PathUnescape(chi.URLParam(r, "permissionID"))
	if err != nil {
		api.logger.Debug().Err(err).Msg("could not parse permissionID")
		errorcode.InvalidRequest.Render(w, r, http.StatusBadRequest, "invalid permissionID")
		return
	}

	permission := libregraph.Permission{}
	if err = StrictJSONUnmarshal(r.Body, &permission); err != nil {
		api.logger.Debug().Err(err).Interface("Body", r.Body).Msg("failed unmarshalling request body")
		errorcode.InvalidRequest.Render(w, r, http.StatusBadRequest, "invalid request body")
		return
	}

	ctx := r.Context()
	if err = validate.StructCtx(ctx, permission); err != nil {
		api.logger.Debug().Err(err).Interface("Body", r.Body).Msg("invalid request body")
		errorcode.InvalidRequest.Render(w, r, http.StatusBadRequest, err.Error())
		return
	}

	updatedPermission, err := api.driveItemPermissionsService.UpdatePermission(ctx, itemID, permissionID, permission)
	if err != nil {
		errorcode.RenderError(w, r, err)
		return
	}
	render.Status(r, http.StatusOK)
	render.JSON(w, r, &updatedPermission)
}

// UpdateSpaceRootPermission handles UpdatePermission requests on a space root
func (api DriveItemPermissionsApi) UpdateSpaceRootPermission(w http.ResponseWriter, r *http.Request) {
	driveID, err := parseIDParam(r, "driveID")
	if err != nil {
		api.logger.Debug().Err(err).Msg(parseDriveIDErrMsg)
		errorcode.InvalidRequest.Render(w, r, http.StatusUnprocessableEntity, parseDriveIDErrMsg)
		return
	}

	permissionID, err := url.PathUnescape(chi.URLParam(r, "permissionID"))
	if err != nil {
		api.logger.Debug().Err(err).Msg("could not parse permissionID")
		errorcode.InvalidRequest.Render(w, r, http.StatusBadRequest, "invalid permissionID")
		return
	}

	permission := libregraph.Permission{}
	if err = StrictJSONUnmarshal(r.Body, &permission); err != nil {
		api.logger.Debug().Err(err).Interface("Body", r.Body).Msg("failed unmarshalling request body")
		errorcode.InvalidRequest.Render(w, r, http.StatusBadRequest, "invalid request body")
		return
	}

	ctx := r.Context()
	if err = validate.StructCtx(ctx, permission); err != nil {
		api.logger.Debug().Err(err).Interface("Body", r.Body).Msg("invalid request body")
		errorcode.InvalidRequest.Render(w, r, http.StatusBadRequest, err.Error())
		return
	}

	updatedPermission, err := api.driveItemPermissionsService.UpdateSpaceRootPermission(ctx, &driveID, permissionID, permission)
	if err != nil {
		errorcode.RenderError(w, r, err)
		return
	}
	render.Status(r, http.StatusOK)
	render.JSON(w, r, &updatedPermission)
}
