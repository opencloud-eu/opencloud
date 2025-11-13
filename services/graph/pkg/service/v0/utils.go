package svc

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"reflect"

	gateway "github.com/cs3org/go-cs3apis/cs3/gateway/v1beta1"
	cs3User "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"
	collaboration "github.com/cs3org/go-cs3apis/cs3/sharing/collaboration/v1beta1"
	ocm "github.com/cs3org/go-cs3apis/cs3/sharing/ocm/v1beta1"
	storageprovider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	libregraph "github.com/opencloud-eu/libre-graph-api-go"
	"github.com/opencloud-eu/reva/v2/pkg/storagespace"
	"github.com/opencloud-eu/reva/v2/pkg/utils"
	"golang.org/x/sync/errgroup"

	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/services/graph/pkg/errorcode"
	"github.com/opencloud-eu/opencloud/services/graph/pkg/identity/cache"
	"github.com/opencloud-eu/opencloud/services/graph/pkg/unifiedrole"
)

// StrictJSONUnmarshal is a wrapper around json.Unmarshal that returns an error if the json contains unknown fields.
func StrictJSONUnmarshal(r io.Reader, v interface{}) error {
	dec := json.NewDecoder(r)
	dec.DisallowUnknownFields()
	return dec.Decode(v)
}

// IsSpaceRoot returns true if the resourceID is a space root.
func IsSpaceRoot(rid *storageprovider.ResourceId) bool {
	if rid == nil {
		return false
	}
	if rid.GetSpaceId() == "" || rid.GetOpaqueId() == "" {
		return false
	}

	return rid.GetSpaceId() == rid.GetOpaqueId()
}

// GetDriveAndItemIDParam parses the driveID and itemID from the request,
// validates the common fields and returns the parsed IDs if ok.
func GetDriveAndItemIDParam(r *http.Request, logger *log.Logger) (*storageprovider.ResourceId, *storageprovider.ResourceId, error) {
	empty := &storageprovider.ResourceId{}

	driveID, err := parseIDParam(r, "driveID")
	if err != nil {
		logger.Debug().Err(err).Msg("could not parse driveID")
		return empty, empty, errorcode.New(errorcode.InvalidRequest, "invalid driveID")
	}

	itemID, err := parseIDParam(r, "itemID")
	if err != nil {
		logger.Debug().Err(err).Msg("could not parse itemID")
		return empty, empty, errorcode.New(errorcode.InvalidRequest, "invalid itemID")
	}

	if itemID.GetOpaqueId() == "" {
		logger.Debug().Interface("driveID", &driveID).Interface("itemID", &itemID).Msg("empty item opaqueID")
		return empty, empty, errorcode.New(errorcode.InvalidRequest, "invalid itemID")
	}

	if driveID.GetStorageId() != itemID.GetStorageId() || driveID.GetSpaceId() != itemID.GetSpaceId() {
		logger.Debug().Interface("driveID", &driveID).Interface("itemID", &itemID).Msg("driveID and itemID do not match")
		return empty, empty, errorcode.New(errorcode.ItemNotFound, "driveID and itemID do not match")
	}

	return &driveID, &itemID, nil
}

// GetGatewayClient returns a gateway client from the gatewaySelector.
func (g Graph) GetGatewayClient(w http.ResponseWriter, r *http.Request) (gateway.GatewayAPIClient, bool) {
	gatewayClient, err := g.gatewaySelector.Next()
	if err != nil {
		g.logger.Debug().Err(err).Msg("selecting gatewaySelector failed")
		errorcode.GeneralException.Render(w, r, http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError))
		return nil, false
	}

	return gatewayClient, true
}

// IsShareJail returns true if given id is a share jail id.
func IsShareJail(id *storageprovider.ResourceId) bool {
	return id.GetStorageId() == utils.ShareStorageProviderID && id.GetSpaceId() == utils.ShareStorageSpaceID
}

// userIdToIdentity looks the user for the supplied id using the cache and returns it
// as a libregraph.Identity
func userIdToIdentity(ctx context.Context, cache cache.IdentityCache, tennantId, userID string) (libregraph.Identity, error) {
	identity := libregraph.Identity{
		Id: libregraph.PtrString(userID),
	}
	user, err := cache.GetUser(ctx, tennantId, userID)
	if err == nil {
		identity.SetDisplayName(user.GetDisplayName())
		identity.SetLibreGraphUserType(user.GetUserType())
	}
	return identity, err
}

// federatedIdToIdentity looks the user for the supplied id using the cache and returns it
// as a libregraph.Identity
func federatedIdToIdentity(ctx context.Context, cache cache.IdentityCache, userID string) (libregraph.Identity, error) {
	identity := libregraph.Identity{
		Id:                 libregraph.PtrString(userID),
		LibreGraphUserType: libregraph.PtrString("Federated"),
	}
	user, err := cache.GetAcceptedUser(ctx, userID)
	if err == nil {
		identity.SetDisplayName(user.GetDisplayName())
		identity.SetLibreGraphUserType(user.GetUserType())
	}
	return identity, err
}

// cs3UserIdToIdentity looks up the user for the supplied cs3 userid using the cache and returns it
// as a libregraph.Identity. Skips the user lookup if the id type is USER_TYPE_SPACE_OWNER
func cs3UserIdToIdentity(ctx context.Context, cache cache.IdentityCache, cs3UserID *cs3User.UserId) (libregraph.Identity, error) {
	if cs3UserID.GetType() == cs3User.UserType_USER_TYPE_FEDERATED {
		return federatedIdToIdentity(ctx, cache, cs3UserID.GetOpaqueId())
	}
	if cs3UserID.GetType() != cs3User.UserType_USER_TYPE_SPACE_OWNER {
		return userIdToIdentity(ctx, cache, cs3UserID.GetTenantId(), cs3UserID.GetOpaqueId())
	}
	return libregraph.Identity{Id: libregraph.PtrString(cs3UserID.GetOpaqueId())}, nil
}

// groupIdToIdentity looks up the group for the supplied cs3 groupid using the cache and returns it
// as a libregraph.Identity.
func groupIdToIdentity(ctx context.Context, cache cache.IdentityCache, groupID string) (libregraph.Identity, error) {
	identity := libregraph.Identity{
		Id: libregraph.PtrString(groupID),
	}
	group, err := cache.GetGroup(ctx, groupID)
	if err == nil {
		identity.SetDisplayName(group.GetDisplayName())
	}
	return identity, err
}

// identitySetToSpacePermissionID generates an Id for a permission from an identitySet. In libregraph
// permissions need to have an id. For user share permission we just use the cs3 share id as the permission-id
// As permissions on space to not map to a cs3 share we need something else of the ids. So we just
// construct the id for the id of the user or group that the permission applies to and prefix that
// with a "u:" for userids and "g:" for group ids.
func identitySetToSpacePermissionID(identitySet libregraph.SharePointIdentitySet) (id string) {
	switch {
	case identitySet.HasUser():
		id = "u:" + identitySet.User.GetId()
	case identitySet.HasGroup():
		id = "g:" + identitySet.Group.GetId()
	}
	return id
}

func cs3ReceivedSharesToDriveItems(ctx context.Context,
	logger *log.Logger,
	gatewayClient gateway.GatewayAPIClient,
	identityCache cache.IdentityCache,
	receivedShares []*collaboration.ReceivedShare,
	availableRoles []*libregraph.UnifiedRoleDefinition,
) ([]libregraph.DriveItem, error) {

	group := new(errgroup.Group)
	// Set max concurrency
	group.SetLimit(10)

	receivedSharesByResourceID := make(map[string][]*collaboration.ReceivedShare, len(receivedShares))
	for _, receivedShare := range receivedShares {
		if receivedShare == nil {
			continue
		}

		rIDStr := storagespace.FormatResourceID(receivedShare.GetShare().GetResourceId())
		receivedSharesByResourceID[rIDStr] = append(receivedSharesByResourceID[rIDStr], receivedShare)
	}

	ch := make(chan libregraph.DriveItem, len(receivedSharesByResourceID))
	for _, receivedSharesForResource := range receivedSharesByResourceID {
		receivedShares := receivedSharesForResource

		group.Go(func() error {
			var err error // redeclare
			shareStat, err := gatewayClient.Stat(ctx, &storageprovider.StatRequest{
				Ref: &storageprovider.Reference{
					ResourceId: receivedShares[0].GetShare().GetResourceId(),
				},
			})

			if err := errorcode.FromCS3Status(shareStat.GetStatus(), err); err != nil {
				logger.Debug().Err(err).
					Str("shareid", receivedShares[0].GetShare().GetId().GetOpaqueId()).
					Str("resourceid", storagespace.FormatResourceID(receivedShares[0].GetShare().GetResourceId())).
					Msg("could not stat received share, skipping")
				return nil
			}

			driveItem, err := fillDriveItemPropertiesFromReceivedShare(ctx, logger, identityCache, receivedShares, shareStat.GetInfo(), availableRoles)
			if err != nil {
				logger.Debug().Err(err).
					Str("shareid", receivedShares[0].GetShare().GetId().GetOpaqueId()).
					Str("resourceid", storagespace.FormatResourceID(receivedShares[0].GetShare().GetResourceId())).
					Msg("could not fill drive item properties from received share, skipping")
				return nil
			}

			if !driveItem.HasUIHidden() {
				driveItem.SetUIHidden(false)
			}
			if !driveItem.HasClientSynchronize() {
				driveItem.SetClientSynchronize(false)
				if name := shareStat.GetInfo().GetName(); name != "" {
					driveItem.SetName(name)
				}
			}

			remoteItem := driveItem.RemoteItem
			{
				if id := shareStat.GetInfo().GetId(); id != nil {
					remoteItem.SetId(storagespace.FormatResourceID(id))
				}

				if name := shareStat.GetInfo().GetName(); name != "" {
					remoteItem.SetName(name)
				}

				if etag := shareStat.GetInfo().GetEtag(); etag != "" {
					remoteItem.SetETag(etag)
				}

				if mTime := shareStat.GetInfo().GetMtime(); mTime != nil {
					remoteItem.SetLastModifiedDateTime(cs3TimestampToTime(mTime))
				}

				if size := shareStat.GetInfo().GetSize(); size != 0 {
					remoteItem.SetSize(int64(size))
				}

				parentReference := libregraph.NewItemReference()
				if spaceType := shareStat.GetInfo().GetSpace().GetSpaceType(); spaceType != "" {
					parentReference.SetDriveType(spaceType)
				}

				if root := shareStat.GetInfo().GetSpace().GetRoot(); root != nil {
					parentReference.SetDriveId(storagespace.FormatResourceID(root))
				}
				if !reflect.ValueOf(*parentReference).IsZero() {
					remoteItem.ParentReference = parentReference
				}

			}

			// the parentReference of the outer driveItem should be the drive
			// containing the mountpoint i.e. the share jail
			driveItem.ParentReference = libregraph.NewItemReference()
			driveItem.ParentReference.SetDriveType(_spaceTypeVirtual)
			driveItem.ParentReference.SetDriveId(storagespace.FormatStorageID(utils.ShareStorageProviderID, utils.ShareStorageSpaceID))
			driveItem.ParentReference.SetId(storagespace.FormatResourceID(&storageprovider.ResourceId{
				StorageId: utils.ShareStorageProviderID,
				OpaqueId:  utils.ShareStorageSpaceID,
				SpaceId:   utils.ShareStorageSpaceID,
			}))
			if etag := shareStat.GetInfo().GetEtag(); etag != "" {
				driveItem.SetETag(etag)
			}

			// connect the dots
			{
				if mTime := shareStat.GetInfo().GetMtime(); mTime != nil {
					t := cs3TimestampToTime(mTime)

					driveItem.SetLastModifiedDateTime(t)
					remoteItem.SetLastModifiedDateTime(t)
				}

				if size := shareStat.GetInfo().GetSize(); size != 0 {
					s := int64(size)

					driveItem.SetSize(s)
					remoteItem.SetSize(s)
				}

				if userID := shareStat.GetInfo().GetOwner(); userID != nil && userID.Type != cs3User.UserType_USER_TYPE_SPACE_OWNER {
					identity, err := cs3UserIdToIdentity(ctx, identityCache, userID)
					if err != nil {
						// TODO: define a proper error behavior here. We don't
						// want the whole request to fail just because a single
						// resource owner couldn't be resolved. But, should we
						// really return the affected share in the response?
						// For now we just log a warning. The returned
						// identitySet will just contain the userid.
						logger.Warn().Err(err).Str("userid", userID.String()).Msg("could not get owner of shared resource")
					}

					remoteItem.SetCreatedBy(libregraph.IdentitySet{User: &identity})
					driveItem.SetCreatedBy(libregraph.IdentitySet{User: &identity})
				}
				switch info := shareStat.GetInfo(); {
				case info.GetType() == storageprovider.ResourceType_RESOURCE_TYPE_CONTAINER:
					folder := libregraph.NewFolder()

					remoteItem.Folder = folder
					driveItem.Folder = folder
				case info.GetType() == storageprovider.ResourceType_RESOURCE_TYPE_FILE:
					file := libregraph.NewOpenGraphFile()

					if mimeType := info.GetMimeType(); mimeType != "" {
						file.MimeType = &mimeType
					}

					remoteItem.File = file
					driveItem.File = file
				}
			}

			ch <- *driveItem

			return nil
		})
	}

	var err error
	// wait for concurrent requests to finish
	go func() {
		err = group.Wait()
		close(ch)
	}()

	driveItems := make([]libregraph.DriveItem, 0, len(receivedSharesByResourceID))
	for di := range ch {
		driveItems = append(driveItems, di)
	}

	return driveItems, err
}

func fillDriveItemPropertiesFromReceivedShare(ctx context.Context, logger *log.Logger,
	identityCache cache.IdentityCache, receivedShares []*collaboration.ReceivedShare,
	resourceInfo *storageprovider.ResourceInfo, availableRoles []*libregraph.UnifiedRoleDefinition) (*libregraph.DriveItem, error) {

	driveItem := libregraph.NewDriveItem()
	permissions := make([]libregraph.Permission, 0, len(receivedShares))

	var oldestReceivedShare *collaboration.ReceivedShare
	for _, receivedShare := range receivedShares {
		switch {
		case oldestReceivedShare == nil:
			fallthrough
		case utils.TSToTime(receivedShare.GetShare().GetCtime()).Before(utils.TSToTime(oldestReceivedShare.GetShare().GetCtime())):
			oldestReceivedShare = receivedShare
		}

		permission, err := cs3ReceivedShareToLibreGraphPermissions(ctx, logger, identityCache, receivedShare, resourceInfo, availableRoles)
		if err != nil {
			return driveItem, err
		}

		// If at least one of the shares was accepted, we consider the driveItem's synchronized
		// flag enabled.
		// Also we use the Mountpoint name of the first accepted mountpoint as the name for
		// the driveItem
		if receivedShare.GetState() == collaboration.ShareState_SHARE_STATE_ACCEPTED {
			driveItem.SetClientSynchronize(true)
			if name := receivedShare.GetMountPoint().GetPath(); name != "" && driveItem.GetName() == "" {
				driveItem.SetName(receivedShare.GetMountPoint().GetPath())
			}
		}

		// if at least one share is marked as hidden, consider the whole driveItem to be hidden
		if receivedShare.GetHidden() {
			driveItem.SetUIHidden(true)
		}

		if userID := receivedShare.GetShare().GetCreator(); userID != nil {
			identity, err := cs3UserIdToIdentity(ctx, identityCache, userID)
			if err != nil {
				logger.Warn().Err(err).Str("userid", userID.String()).Msg("could not get creator of the share")
			}

			permission.SetInvitation(
				libregraph.SharingInvitation{
					InvitedBy: &libregraph.IdentitySet{
						User: &identity,
					},
				},
			)
		}
		permissions = append(permissions, *permission)
		// To stay compatible with the usershareprovider and the webdav
		// service the id of the driveItem is composed of the StorageID and
		// SpaceID of the sharestorage appended with the opaque ID of
		// the oldest share for the resource:
		// '<sharestorageid>$<sharespaceid>!<share-opaque-id>
		// Note: This means that the driveitem ID will change when the oldest
		//   share is removed. It would be good to have are more stable ID here (e.g.
		//   derived from the shared resource's ID. But as we need to use the same
		//   ID across all services this means we needed to make similar adjustments
		//   to the sharejail (usershareprovider, webdav). Which we can't currently do
		//   as some clients rely on the IDs used there having a special format.
		driveItem.SetId(storagespace.FormatResourceID(&storageprovider.ResourceId{
			StorageId: utils.ShareStorageProviderID,
			OpaqueId:  oldestReceivedShare.GetShare().GetId().GetOpaqueId(),
			SpaceId:   utils.ShareStorageSpaceID,
		}))

	}
	driveItem.RemoteItem = libregraph.NewRemoteItem()
	driveItem.RemoteItem.Permissions = permissions
	return driveItem, nil
}

func cs3ReceivedShareToLibreGraphPermissions(ctx context.Context, logger *log.Logger,
	identityCache cache.IdentityCache, receivedShare *collaboration.ReceivedShare,
	resourceInfo *storageprovider.ResourceInfo, availableRoles []*libregraph.UnifiedRoleDefinition) (*libregraph.Permission, error) {
	permission := libregraph.NewPermission()
	if id := receivedShare.GetShare().GetId().GetOpaqueId(); id != "" {
		permission.SetId(id)
	}

	if expiration := receivedShare.GetShare().GetExpiration(); expiration != nil {
		permission.SetExpirationDateTime(cs3TimestampToTime(expiration))
	}

	if cTime := receivedShare.GetShare().GetCtime(); cTime != nil {
		permission.SetCreatedDateTime(cs3TimestampToTime(cTime))
	}

	if permissionSet := receivedShare.GetShare().GetPermissions().GetPermissions(); permissionSet != nil {
		condition, err := roleConditionForResourceType(resourceInfo)
		if err != nil {
			return nil, err
		}

		role := unifiedrole.CS3ResourcePermissionsToRole(
			availableRoles,
			permissionSet,
			condition,
			false,
		)
		if role != nil {
			permission.SetRoles([]string{role.GetId()})
		}

		actions := unifiedrole.CS3ResourcePermissionsToLibregraphActions(permissionSet)

		// actions only make sense if no role is set
		if role == nil && len(actions) > 0 {
			permission.SetLibreGraphPermissionsActions(actions)
		}
	}
	switch grantee := receivedShare.GetShare().GetGrantee(); {
	case grantee.GetType() == storageprovider.GranteeType_GRANTEE_TYPE_USER:
		user, err := cs3UserIdToIdentity(ctx, identityCache, grantee.GetUserId())
		if err != nil {
			logger.Error().Err(err).Msg("could not get user")
			return nil, err
		}
		permission.SetGrantedToV2(libregraph.SharePointIdentitySet{User: &user})
	case grantee.GetType() == storageprovider.GranteeType_GRANTEE_TYPE_GROUP:
		group, err := groupIdToIdentity(ctx, identityCache, grantee.GetGroupId().GetOpaqueId())
		if err != nil {
			logger.Error().Err(err).Msg("could not get group")
			return nil, err
		}
		permission.SetGrantedToV2(libregraph.SharePointIdentitySet{Group: &group})
	}

	return permission, nil
}

func roleConditionForResourceType(ri *storageprovider.ResourceInfo) (string, error) {
	switch {
	case utils.ResourceIDEqual(ri.GetSpace().GetRoot(), ri.GetId()):
		return unifiedrole.UnifiedRoleConditionDrive, nil
	case ri.Type == storageprovider.ResourceType_RESOURCE_TYPE_CONTAINER:
		return unifiedrole.UnifiedRoleConditionFolder, nil
	case ri.Type == storageprovider.ResourceType_RESOURCE_TYPE_FILE:
		return unifiedrole.UnifiedRoleConditionFile, nil
	default:
		return "", errorcode.New(errorcode.InvalidRequest, "unsupported resource type")
	}
}

func federatedRoleConditionForResourceType(ri *storageprovider.ResourceInfo) (string, error) {
	switch ri.GetType() {
	case storageprovider.ResourceType_RESOURCE_TYPE_CONTAINER:
		return unifiedrole.UnifiedRoleConditionFolderFederatedUser, nil
	case storageprovider.ResourceType_RESOURCE_TYPE_FILE:
		return unifiedrole.UnifiedRoleConditionFileFederatedUser, nil
	default:
		return "", errorcode.New(errorcode.InvalidRequest, "unsupported resource type for federated role")
	}
}

// ExtractShareIdFromResourceId is a bit of a hack.
// We should not rely on a specific format of the item id.
// But currently there is no other way to get the ShareID.
func ExtractShareIdFromResourceId(rid *storageprovider.ResourceId) *collaboration.ShareId {
	return &collaboration.ShareId{
		OpaqueId: rid.GetOpaqueId(),
	}
}

func cs3ReceivedOCMSharesToDriveItems(ctx context.Context,
	logger *log.Logger,
	gatewayClient gateway.GatewayAPIClient,
	identityCache cache.IdentityCache,
	receivedShares []*ocm.ReceivedShare, availableRoles []*libregraph.UnifiedRoleDefinition) ([]libregraph.DriveItem, error) {

	group := new(errgroup.Group)
	// Set max concurrency
	group.SetLimit(10)

	receivedSharesByResourceID := make(map[string][]*ocm.ReceivedShare, len(receivedShares))
	for _, receivedShare := range receivedShares {
		rIDStr := receivedShare.GetRemoteShareId()
		receivedSharesByResourceID[rIDStr] = append(receivedSharesByResourceID[rIDStr], receivedShare)
	}

	ch := make(chan libregraph.DriveItem, len(receivedSharesByResourceID))
	for _, receivedSharesForResource := range receivedSharesByResourceID {
		receivedShares := receivedSharesForResource

		group.Go(func() error {
			var err error // redeclare

			// for OCM shares the opaqueID is the '/' for shared directories and '/filename' for
			// file shares
			resOpaqueID := "/"
			if receivedShares[0].GetResourceType() == storageprovider.ResourceType_RESOURCE_TYPE_FILE {
				resOpaqueID += receivedShares[0].GetName()
			}

			shareStat, err := gatewayClient.Stat(ctx, &storageprovider.StatRequest{
				Ref: &storageprovider.Reference{
					ResourceId: &storageprovider.ResourceId{
						// TODO maybe the reference is wrong
						StorageId: utils.OCMStorageProviderID,
						SpaceId:   receivedShares[0].GetId().GetOpaqueId(),
						OpaqueId:  base64.StdEncoding.EncodeToString([]byte(resOpaqueID)),
					},
				},
			})

			if err := errorcode.FromCS3Status(shareStat.GetStatus(), err); err != nil {
				logger.Debug().Err(err).
					Str("shareid", receivedShares[0].GetId().GetOpaqueId()).
					Str("remoteshareid", receivedShares[0].GetRemoteShareId()).
					Msg("could not stat received ocm share, skipping")
				return nil
			}

			driveItem, err := fillDriveItemPropertiesFromReceivedOCMShare(ctx, logger, identityCache, receivedShares, shareStat.GetInfo(), availableRoles)
			if err != nil {
				logger.Debug().Err(err).
					Str("shareid", receivedShares[0].GetId().GetOpaqueId()).
					Str("remoteshareid", receivedShares[0].GetRemoteShareId()).
					Msg("could not fill drive item properties from received ocm share, skipping")
				return nil
			}

			if !driveItem.HasUIHidden() {
				driveItem.SetUIHidden(false)
			}
			if !driveItem.HasClientSynchronize() {
				driveItem.SetClientSynchronize(false)
				if name := shareStat.GetInfo().GetName(); name != "" {
					driveItem.SetName(name) // FIXME name is not set???
				}
			}

			remoteItem := driveItem.RemoteItem
			{
				if id := shareStat.GetInfo().GetId(); id != nil {
					remoteItem.SetId(storagespace.FormatResourceID(id))
				}

				if name := shareStat.GetInfo().GetName(); name != "" {
					remoteItem.SetName(name)
				}

				if etag := shareStat.GetInfo().GetEtag(); etag != "" {
					remoteItem.SetETag(etag)
				}

				if mTime := shareStat.GetInfo().GetMtime(); mTime != nil {
					remoteItem.SetLastModifiedDateTime(cs3TimestampToTime(mTime))
				}

				if size := shareStat.GetInfo().GetSize(); size != 0 {
					remoteItem.SetSize(int64(size))
				}

				parentReference := libregraph.NewItemReference()
				if spaceType := shareStat.GetInfo().GetSpace().GetSpaceType(); spaceType != "" {
					parentReference.SetDriveType(spaceType)
				}

				if root := shareStat.GetInfo().GetSpace().GetRoot(); root != nil {
					parentReference.SetDriveId(storagespace.FormatResourceID(root))
				}
				if !reflect.ValueOf(*parentReference).IsZero() {
					remoteItem.ParentReference = parentReference
				}

			}

			// the parentReference of the outer driveItem should be the drive
			// containing the mountpoint i.e. the share jail
			driveItem.ParentReference = libregraph.NewItemReference()
			driveItem.ParentReference.SetDriveType("virtual")
			driveItem.ParentReference.SetDriveId(storagespace.FormatStorageID(utils.ShareStorageProviderID, utils.ShareStorageSpaceID))
			driveItem.ParentReference.SetId(storagespace.FormatResourceID(&storageprovider.ResourceId{
				StorageId: utils.ShareStorageProviderID,
				OpaqueId:  utils.ShareStorageSpaceID,
				SpaceId:   utils.ShareStorageSpaceID,
			}))
			if etag := shareStat.GetInfo().GetEtag(); etag != "" {
				driveItem.SetETag(etag)
			}

			// connect the dots
			{
				if mTime := shareStat.GetInfo().GetMtime(); mTime != nil {
					t := cs3TimestampToTime(mTime)

					driveItem.SetLastModifiedDateTime(t)
					remoteItem.SetLastModifiedDateTime(t)
				}

				if size := shareStat.GetInfo().GetSize(); size != 0 {
					s := int64(size)

					driveItem.SetSize(s)
					remoteItem.SetSize(s)
				}

				if userID := shareStat.GetInfo().GetOwner(); userID != nil && userID.Type != cs3User.UserType_USER_TYPE_SPACE_OWNER {
					identity, err := cs3UserIdToIdentity(ctx, identityCache, userID)
					if err != nil {
						// TODO: define a proper error behavior here. We don't
						// want the whole request to fail just because a single
						// resource owner couldn't be resolved. But, should we
						// really return the affected share in the response?
						// For now we just log a warning. The returned
						// identitySet will just contain the userid.
						logger.Warn().Err(err).Str("userid", userID.String()).Msg("could not get owner of shared resource")
					}

					remoteItem.SetCreatedBy(libregraph.IdentitySet{User: &identity})
					driveItem.SetCreatedBy(libregraph.IdentitySet{User: &identity})
				}
				switch info := shareStat.GetInfo(); {
				case info.GetType() == storageprovider.ResourceType_RESOURCE_TYPE_CONTAINER:
					folder := libregraph.NewFolder()

					remoteItem.Folder = folder
					driveItem.Folder = folder
				case info.GetType() == storageprovider.ResourceType_RESOURCE_TYPE_FILE:
					file := libregraph.NewOpenGraphFile()

					if mimeType := info.GetMimeType(); mimeType != "" {
						file.MimeType = &mimeType
					}

					remoteItem.File = file
					driveItem.File = file
				}
			}

			ch <- *driveItem

			return nil
		})
	}

	var err error
	// wait for concurrent requests to finish
	go func() {
		err = group.Wait()
		close(ch)
	}()

	driveItems := make([]libregraph.DriveItem, 0, len(receivedSharesByResourceID))
	for di := range ch {
		driveItems = append(driveItems, di)
	}

	return driveItems, err
}

func fillDriveItemPropertiesFromReceivedOCMShare(ctx context.Context, logger *log.Logger,
	identityCache cache.IdentityCache, receivedShares []*ocm.ReceivedShare,
	resourceInfo *storageprovider.ResourceInfo, availableRoles []*libregraph.UnifiedRoleDefinition) (*libregraph.DriveItem, error) {

	driveItem := libregraph.NewDriveItem()
	permissions := make([]libregraph.Permission, 0, len(receivedShares))

	var oldestReceivedShare *ocm.ReceivedShare
	for _, receivedShare := range receivedShares {
		switch {
		case oldestReceivedShare == nil:
			fallthrough
		case utils.TSToTime(receivedShare.GetCtime()).Before(utils.TSToTime(oldestReceivedShare.GetCtime())):
			oldestReceivedShare = receivedShare
		}

		permission, err := cs3ReceivedOCMShareToLibreGraphPermissions(ctx, logger, identityCache, receivedShare, resourceInfo, availableRoles)
		if err != nil {
			return driveItem, err
		}

		driveItem.SetName(resourceInfo.GetName())

		// If at least one of the shares was accepted, we consider the driveItem's synchronized
		// flag enabled.
		// Also we use the Mountpoint name of the first accepted mountpoint as the name for
		// the driveItem
		if receivedShare.GetState() == ocm.ShareState_SHARE_STATE_ACCEPTED {
			driveItem.SetClientSynchronize(true)
			if name := receivedShare.GetName(); name != "" && driveItem.GetName() == "" {
				driveItem.SetName(receivedShare.GetName())
			}
		}

		// if at least one share is marked as hidden, consider the whole driveItem to be hidden
		/*
			if receivedShare.GetHidden() {
				driveItem.SetUIHidden(true)
			}
		*/

		if userID := receivedShare.GetCreator(); userID != nil {
			identity, err := cs3UserIdToIdentity(ctx, identityCache, userID)
			if err != nil {
				logger.Warn().Err(err).Str("userid", userID.String()).Msg("could not get creator of the ocm share")
			}

			permission.SetInvitation(
				libregraph.SharingInvitation{
					InvitedBy: &libregraph.IdentitySet{
						User: &identity,
					},
				},
			)
		}
		permissions = append(permissions, *permission)
		// To stay compatible with the usershareprovider and the webdav
		// service the id of the driveItem is composed of the StorageID and
		// SpaceID of the sharestorage appended with the opaque ID of
		// the oldest share for the resource:
		// '<sharestorageid>$<sharespaceid>!<share-opaque-id>
		// Note: This means that the driveitem ID will change when the oldest
		//   share is removed. It would be good to have are more stable ID here (e.g.
		//   derived from the shared resource's ID. But as we need to use the same
		//   ID across all services this means we needed to make similar adjustments
		//   to the sharejail (usershareprovider, webdav). Which we can't currently do
		//   as some clients rely on the IDs used there having a special format.
		driveItem.SetId(storagespace.FormatResourceID(&storageprovider.ResourceId{
			StorageId: utils.OCMStorageProviderID,
			SpaceId:   utils.OCMStorageSpaceID,
			OpaqueId:  oldestReceivedShare.GetRemoteShareId(),
		}))

	}
	driveItem.RemoteItem = libregraph.NewRemoteItem()
	driveItem.RemoteItem.Permissions = permissions
	return driveItem, nil
}

func cs3ReceivedOCMShareToLibreGraphPermissions(ctx context.Context, logger *log.Logger,
	identityCache cache.IdentityCache, receivedShare *ocm.ReceivedShare,
	resourceInfo *storageprovider.ResourceInfo, availableRoles []*libregraph.UnifiedRoleDefinition) (*libregraph.Permission, error) {
	permission := libregraph.NewPermission()
	if id := receivedShare.GetId().GetOpaqueId(); id != "" {
		permission.SetId(id)
	}

	if cTime := receivedShare.GetCtime(); cTime != nil {
		permission.SetCreatedDateTime(cs3TimestampToTime(cTime))
	}

	if expiration := receivedShare.GetExpiration(); expiration != nil {
		permission.SetExpirationDateTime(cs3TimestampToTime(expiration))
	}

	var permissions *storageprovider.ResourcePermissions
	for _, protocol := range receivedShare.GetProtocols() {
		if protocol.GetWebdavOptions().GetPermissions() != nil {
			permissions = protocol.GetWebdavOptions().GetPermissions().GetPermissions()
		}
	}
	condition, err := federatedRoleConditionForResourceType(resourceInfo)
	if err != nil {
		return nil, err
	}
	role := unifiedrole.CS3ResourcePermissionsToRole(
		availableRoles,
		permissions,
		condition,
		true,
	)
	if role != nil {
		permission.SetRoles([]string{role.GetId()})
	} else {
		actions := unifiedrole.CS3ResourcePermissionsToLibregraphActions(permissions)
		permission.SetLibreGraphPermissionsActions(actions)
		permission.SetRoles(nil)
	}

	switch grantee := receivedShare.GetGrantee(); {
	case grantee.GetType() == storageprovider.GranteeType_GRANTEE_TYPE_USER:
		user, err := cs3UserIdToIdentity(ctx, identityCache, grantee.GetUserId())
		if err != nil {
			logger.Error().Err(err).Msg("could not get user")
			return nil, err
		}
		permission.SetGrantedToV2(libregraph.SharePointIdentitySet{User: &user})
	case grantee.GetType() == storageprovider.GranteeType_GRANTEE_TYPE_GROUP:
		group, err := groupIdToIdentity(ctx, identityCache, grantee.GetGroupId().GetOpaqueId())
		if err != nil {
			logger.Error().Err(err).Msg("could not get group")
			return nil, err
		}
		permission.SetGrantedToV2(libregraph.SharePointIdentitySet{Group: &group})
	}

	return permission, nil
}
