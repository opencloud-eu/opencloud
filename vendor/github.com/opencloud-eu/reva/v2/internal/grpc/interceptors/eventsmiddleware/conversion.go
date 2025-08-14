// Copyright 2018-2021 CERN
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
// In applying this license, CERN does not waive the privileges and immunities
// granted to it by virtue of its status as an Intergovernmental Organization
// or submit itself to any jurisdiction.

package eventsmiddleware

import (
	"strings"
	"time"

	group "github.com/cs3org/go-cs3apis/cs3/identity/group/v1beta1"
	user "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"
	ocmcore "github.com/cs3org/go-cs3apis/cs3/ocm/core/v1beta1"
	collaboration "github.com/cs3org/go-cs3apis/cs3/sharing/collaboration/v1beta1"
	link "github.com/cs3org/go-cs3apis/cs3/sharing/link/v1beta1"
	provider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	"github.com/opencloud-eu/reva/v2/pkg/events"
	"github.com/opencloud-eu/reva/v2/pkg/rhttp/router"
	"github.com/opencloud-eu/reva/v2/pkg/storagespace"
	"github.com/opencloud-eu/reva/v2/pkg/utils"
)

// ContainerCreated converts the response to an event
func ContainerCreated(r *provider.CreateContainerResponse, req *provider.CreateContainerRequest, spaceOwner *user.UserId, executant *user.User) events.ContainerCreated {
	return events.ContainerCreated{
		SpaceOwner:        spaceOwner,
		Executant:         executant.GetId(),
		Ref:               req.Ref,
		Timestamp:         utils.TSNow(),
		ImpersonatingUser: extractImpersonator(executant),
	}
}

// ShareCreated converts the response to an event
func ShareCreated(r *collaboration.CreateShareResponse, executant *user.User) events.ShareCreated {
	return events.ShareCreated{
		ShareID:        r.Share.GetId(),
		Executant:      executant.GetId(),
		Sharer:         r.Share.Creator,
		GranteeUserID:  r.Share.GetGrantee().GetUserId(),
		GranteeGroupID: r.Share.GetGrantee().GetGroupId(),
		ItemID:         r.Share.ResourceId,
		ResourceName:   utils.ReadPlainFromOpaque(r.Opaque, "resourcename"),
		CTime:          r.Share.Ctime,
		Permissions:    r.Share.Permissions,
	}
}

// ShareRemoved converts the response to an event
func ShareRemoved(r *collaboration.RemoveShareResponse, req *collaboration.RemoveShareRequest, executant *user.User) events.ShareRemoved {
	var (
		userid  *user.UserId
		groupid *group.GroupId
		rid     *provider.ResourceId
	)
	_ = utils.ReadJSONFromOpaque(r.Opaque, "granteeuserid", &userid)
	_ = utils.ReadJSONFromOpaque(r.Opaque, "granteegroupid", &userid)
	_ = utils.ReadJSONFromOpaque(r.Opaque, "resourceid", &rid)
	return events.ShareRemoved{
		Executant:      executant.GetId(),
		ShareID:        req.Ref.GetId(),
		ShareKey:       req.Ref.GetKey(),
		GranteeUserID:  userid,
		GranteeGroupID: groupid,
		ItemID:         rid,
		ResourceName:   utils.ReadPlainFromOpaque(r.Opaque, "resourcename"),
		Timestamp:      time.Now(),
	}
}

// ShareUpdated converts the response to an event
func ShareUpdated(r *collaboration.UpdateShareResponse, req *collaboration.UpdateShareRequest, executant *user.User) events.ShareUpdated {
	return events.ShareUpdated{
		Executant:      executant.GetId(),
		ShareID:        r.Share.Id,
		ItemID:         r.Share.ResourceId,
		ResourceName:   utils.ReadPlainFromOpaque(r.Opaque, "resourcename"),
		Permissions:    r.Share.Permissions,
		GranteeUserID:  r.Share.GetGrantee().GetUserId(),
		GranteeGroupID: r.Share.GetGrantee().GetGroupId(),
		Sharer:         r.Share.Creator,
		MTime:          r.Share.Mtime,
		UpdateMask:     req.GetUpdateMask().GetPaths(),
	}
}

// ReceivedShareUpdated converts the response to an event
func ReceivedShareUpdated(r *collaboration.UpdateReceivedShareResponse, executant *user.User) events.ReceivedShareUpdated {
	return events.ReceivedShareUpdated{
		Executant:      executant.GetId(),
		ShareID:        r.Share.Share.Id,
		ItemID:         r.Share.Share.ResourceId,
		Permissions:    r.Share.Share.Permissions,
		GranteeUserID:  r.Share.Share.GetGrantee().GetUserId(),
		GranteeGroupID: r.Share.Share.GetGrantee().GetGroupId(),
		Sharer:         r.Share.Share.Creator,
		MTime:          r.Share.Share.Mtime,
		State:          collaboration.ShareState_name[int32(r.Share.State)],
	}
}

// LinkCreated converts the response to an event
func LinkCreated(r *link.CreatePublicShareResponse, executant *user.User) events.LinkCreated {
	return events.LinkCreated{
		Executant:         executant.GetId(),
		ShareID:           r.Share.Id,
		Sharer:            r.Share.Creator,
		ItemID:            r.Share.ResourceId,
		ResourceName:      utils.ReadPlainFromOpaque(r.Opaque, "resourcename"),
		Permissions:       r.Share.Permissions,
		DisplayName:       r.Share.DisplayName,
		Expiration:        r.Share.Expiration,
		PasswordProtected: r.Share.PasswordProtected,
		CTime:             r.Share.Ctime,
		Token:             r.Share.Token,
	}
}

// LinkUpdated converts the response to an event
func LinkUpdated(r *link.UpdatePublicShareResponse, req *link.UpdatePublicShareRequest, executant *user.User) events.LinkUpdated {
	return events.LinkUpdated{
		Executant:         executant.GetId(),
		ShareID:           r.Share.Id,
		Sharer:            r.Share.Creator,
		ItemID:            r.Share.ResourceId,
		ResourceName:      utils.ReadPlainFromOpaque(r.Opaque, "resourcename"),
		Permissions:       r.Share.Permissions,
		DisplayName:       r.Share.DisplayName,
		Expiration:        r.Share.Expiration,
		PasswordProtected: r.Share.PasswordProtected,
		MTime:             r.Share.Mtime,
		Token:             r.Share.Token,
		FieldUpdated:      link.UpdatePublicShareRequest_Update_Type_name[int32(req.Update.GetType())],
	}
}

// LinkAccessed converts the response to an event
func LinkAccessed(r *link.GetPublicShareByTokenResponse, executant *user.User) events.LinkAccessed {
	return events.LinkAccessed{
		Executant:         executant.GetId(),
		ShareID:           r.Share.Id,
		Sharer:            r.Share.Creator,
		ItemID:            r.Share.ResourceId,
		Permissions:       r.Share.Permissions,
		DisplayName:       r.Share.DisplayName,
		Expiration:        r.Share.Expiration,
		PasswordProtected: r.Share.PasswordProtected,
		CTime:             r.Share.Ctime,
		Token:             r.Share.Token,
	}
}

// LinkAccessFailed converts the response to an event
func LinkAccessFailed(r *link.GetPublicShareByTokenResponse, req *link.GetPublicShareByTokenRequest, executant *user.User) events.LinkAccessFailed {
	e := events.LinkAccessFailed{
		Executant: executant.GetId(),
		Status:    r.Status.Code,
		Message:   r.Status.Message,
		Timestamp: utils.TSNow(),
		Token:     req.Token,
	}
	if r.Share != nil {
		e.ShareID = r.Share.Id
		e.Token = r.Share.Token
	}
	return e
}

// LinkRemoved converts the response to an event
func LinkRemoved(r *link.RemovePublicShareResponse, req *link.RemovePublicShareRequest, executant *user.User) events.LinkRemoved {
	var rid *provider.ResourceId
	_ = utils.ReadJSONFromOpaque(r.Opaque, "resourceid", &rid)
	return events.LinkRemoved{
		Executant:    executant.GetId(),
		ShareID:      req.Ref.GetId(),
		ShareToken:   req.Ref.GetToken(),
		Timestamp:    utils.TSNow(),
		ItemID:       rid,
		ResourceName: utils.ReadPlainFromOpaque(r.Opaque, "resourcename"),
	}
}

func OCMCoreShareCreated(r *ocmcore.CreateOCMCoreShareResponse, req *ocmcore.CreateOCMCoreShareRequest, executant *user.User) events.OCMCoreShareCreated {
	var permissions *provider.ResourcePermissions
	for _, p := range req.GetProtocols() {
		if p.GetWebdavOptions() != nil {
			permissions = p.GetWebdavOptions().GetPermissions().GetPermissions()
			break
		}
	}
	return events.OCMCoreShareCreated{
		ShareID:       r.GetId(),
		Executant:     executant.GetId(),
		Sharer:        req.GetSender(),
		GranteeUserID: req.GetShareWith(),
		ItemID:        req.GetResourceId(),
		ResourceName:  req.GetName(),
		CTime:         r.GetCreated(),
		Permissions:   permissions,
	}
}

// FileTouched converts the response to an event
func FileTouched(r *provider.TouchFileResponse, req *provider.TouchFileRequest, spaceOwner *user.UserId, executant *user.User) events.FileTouched {
	return events.FileTouched{
		SpaceOwner:        spaceOwner,
		Executant:         executant.GetId(),
		Ref:               req.Ref,
		Timestamp:         utils.TSNow(),
		ImpersonatingUser: extractImpersonator(executant),
	}
}

// FileUploaded converts the response to an event
func FileUploaded(r *provider.InitiateFileUploadResponse, req *provider.InitiateFileUploadRequest, spaceOwner *user.UserId, executant *user.User) events.FileUploaded {
	return events.FileUploaded{
		SpaceOwner:        spaceOwner,
		Executant:         executant.GetId(),
		Ref:               req.Ref,
		Timestamp:         utils.TSNow(),
		ImpersonatingUser: extractImpersonator(executant),
	}
}

// FileDownloaded converts the response to an event
func FileDownloaded(r *provider.InitiateFileDownloadResponse, req *provider.InitiateFileDownloadRequest, executant *user.User) events.FileDownloaded {
	return events.FileDownloaded{
		Executant:         executant.GetId(),
		Ref:               req.Ref,
		Timestamp:         utils.TSNow(),
		ImpersonatingUser: extractImpersonator(executant),
	}
}

// FileLocked converts the response to an events
func FileLocked(r *provider.SetLockResponse, req *provider.SetLockRequest, owner *user.UserId, executant *user.User) events.FileLocked {
	return events.FileLocked{
		Executant:         executant.GetId(),
		Ref:               req.Ref,
		Timestamp:         utils.TSNow(),
		ImpersonatingUser: extractImpersonator(executant),
	}
}

// FileUnlocked converts the response to an event
func FileUnlocked(r *provider.UnlockResponse, req *provider.UnlockRequest, owner *user.UserId, executant *user.User) events.FileUnlocked {
	return events.FileUnlocked{
		Executant:         executant.GetId(),
		Ref:               req.Ref,
		Timestamp:         utils.TSNow(),
		ImpersonatingUser: extractImpersonator(executant),
	}
}

// ItemTrashed converts the response to an event
func ItemTrashed(r *provider.DeleteResponse, req *provider.DeleteRequest, spaceOwner *user.UserId, executant *user.User) events.ItemTrashed {
	opaqueID := utils.ReadPlainFromOpaque(r.Opaque, "opaque_id")
	return events.ItemTrashed{
		SpaceOwner: spaceOwner,
		Executant:  executant.GetId(),
		Ref:        req.Ref,
		ID: &provider.ResourceId{
			StorageId: req.Ref.GetResourceId().GetStorageId(),
			SpaceId:   req.Ref.GetResourceId().GetSpaceId(),
			OpaqueId:  opaqueID,
		},
		Timestamp:         utils.TSNow(),
		ImpersonatingUser: extractImpersonator(executant),
	}
}

// ItemMoved converts the response to an event
func ItemMoved(r *provider.MoveResponse, req *provider.MoveRequest, spaceOwner *user.UserId, executant *user.User) events.ItemMoved {
	return events.ItemMoved{
		SpaceOwner:        spaceOwner,
		Executant:         executant.GetId(),
		Ref:               req.Destination,
		OldReference:      req.Source,
		Timestamp:         utils.TSNow(),
		ImpersonatingUser: extractImpersonator(executant),
	}
}

// TrashbinPurged converts the response to an event
func TrashbinPurged(r *provider.PurgeRecycleResponse, req *provider.PurgeRecycleRequest, executant *user.User) events.TrashbinPurged {
	return events.TrashbinPurged{
		Executant: executant.GetId(),
		Ref: &provider.Reference{
			ResourceId: &provider.ResourceId{
				StorageId: req.Ref.GetResourceId().GetStorageId(),
				SpaceId:   req.Ref.GetResourceId().GetSpaceId(),
				OpaqueId:  req.Ref.GetResourceId().GetSpaceId(),
			},
		},
		Timestamp:         utils.TSNow(),
		ImpersonatingUser: extractImpersonator(executant),
	}
}

// ItemPurged converts the response to an event
func ItemPurged(r *provider.PurgeRecycleResponse, req *provider.PurgeRecycleRequest, executant *user.User) events.ItemPurged {
	root, relativePath := router.ShiftPath(req.Key)
	if relativePath == "/" && !strings.HasSuffix(req.Key, "/") {
		relativePath = ""
	}
	if root == "" {
		// if there is no key this is about purging the whole trashbin
		root = req.Ref.GetResourceId().GetSpaceId()
	}

	ref := &provider.Reference{
		ResourceId: &provider.ResourceId{
			StorageId: req.Ref.GetResourceId().GetStorageId(),
			SpaceId:   req.Ref.GetResourceId().GetSpaceId(),
			OpaqueId:  root,
		},
		Path: relativePath,
	}

	return events.ItemPurged{
		Executant:         executant.GetId(),
		Ref:               ref,
		Timestamp:         utils.TSNow(),
		ImpersonatingUser: extractImpersonator(executant),
	}
}

// ItemRestored converts the response to an event
func ItemRestored(r *provider.RestoreRecycleItemResponse, req *provider.RestoreRecycleItemRequest, spaceOwner *user.UserId, executant *user.User) events.ItemRestored {
	ref := req.Ref
	if req.RestoreRef != nil {
		ref = req.RestoreRef
	}
	return events.ItemRestored{
		SpaceOwner:        spaceOwner,
		Executant:         executant.GetId(),
		Ref:               ref,
		OldReference:      req.Ref,
		Key:               req.Key,
		Timestamp:         utils.TSNow(),
		ImpersonatingUser: extractImpersonator(executant),
	}
}

// FileVersionRestored converts the response to an event
func FileVersionRestored(r *provider.RestoreFileVersionResponse, req *provider.RestoreFileVersionRequest, spaceOwner *user.UserId, executant *user.User) events.FileVersionRestored {
	return events.FileVersionRestored{
		SpaceOwner:        spaceOwner,
		Executant:         executant.GetId(),
		Ref:               req.Ref,
		Key:               req.Key,
		Timestamp:         utils.TSNow(),
		ImpersonatingUser: extractImpersonator(executant),
	}
}

// SpaceCreated converts the response to an event
func SpaceCreated(r *provider.CreateStorageSpaceResponse, executant *user.User) events.SpaceCreated {
	return events.SpaceCreated{
		Executant: executant.GetId(),
		ID:        r.StorageSpace.Id,
		Owner:     extractOwner(r.StorageSpace.Owner),
		Root:      r.StorageSpace.Root,
		Name:      r.StorageSpace.Name,
		Type:      r.StorageSpace.SpaceType,
		Quota:     r.StorageSpace.Quota,
		MTime:     r.StorageSpace.Mtime,
	}
}

// SpaceRenamed converts the response to an event
func SpaceRenamed(r *provider.UpdateStorageSpaceResponse, req *provider.UpdateStorageSpaceRequest, executant *user.User) events.SpaceRenamed {
	return events.SpaceRenamed{
		Executant: executant.GetId(),
		ID:        r.StorageSpace.Id,
		Owner:     extractOwner(r.StorageSpace.Owner),
		Name:      r.StorageSpace.Name,
		Timestamp: utils.TSNow(),
	}
}

// SpaceUpdated converts the response to an event
func SpaceUpdated(r *provider.UpdateStorageSpaceResponse, req *provider.UpdateStorageSpaceRequest, executant *user.User) events.SpaceUpdated {
	return events.SpaceUpdated{
		Executant: executant.GetId(),
		ID:        r.StorageSpace.Id,
		Space:     r.StorageSpace,
		Timestamp: utils.TSNow(),
	}
}

// SpaceEnabled converts the response to an event
func SpaceEnabled(r *provider.UpdateStorageSpaceResponse, req *provider.UpdateStorageSpaceRequest, executant *user.User) events.SpaceEnabled {
	return events.SpaceEnabled{
		Executant: executant.GetId(),
		ID:        r.StorageSpace.Id,
		Owner:     extractOwner(r.StorageSpace.Owner),
		Timestamp: utils.TSNow(),
	}
}

// SpaceShared converts the response to an event
// func SpaceShared(req *provider.AddGrantRequest, executant, sharer *user.UserId, grantee *provider.Grantee) events.SpaceShared {
func SpaceShared(r *provider.AddGrantResponse, req *provider.AddGrantRequest, executant *user.User) events.SpaceShared {
	id := storagespace.FormatStorageID(req.Ref.ResourceId.StorageId, req.Ref.ResourceId.SpaceId)
	return events.SpaceShared{
		Executant:      executant.GetId(),
		Creator:        req.Grant.Creator,
		GranteeUserID:  req.Grant.GetGrantee().GetUserId(),
		GranteeGroupID: req.Grant.GetGrantee().GetGroupId(),
		ID:             &provider.StorageSpaceId{OpaqueId: id},
		Timestamp:      time.Now(),
	}
}

// SpaceShareUpdated converts the response to an events
func SpaceShareUpdated(r *provider.UpdateGrantResponse, req *provider.UpdateGrantRequest, executant *user.User) events.SpaceShareUpdated {
	id := storagespace.FormatStorageID(req.Ref.ResourceId.StorageId, req.Ref.ResourceId.SpaceId)
	return events.SpaceShareUpdated{
		Executant:      executant.GetId(),
		GranteeUserID:  req.Grant.GetGrantee().GetUserId(),
		GranteeGroupID: req.Grant.GetGrantee().GetGroupId(),
		ID:             &provider.StorageSpaceId{OpaqueId: id},
		Timestamp:      time.Now(),
	}
}

// SpaceUnshared  converts the response to an event
func SpaceUnshared(r *provider.RemoveGrantResponse, req *provider.RemoveGrantRequest, executant *user.User) events.SpaceUnshared {
	id := storagespace.FormatStorageID(req.Ref.ResourceId.StorageId, req.Ref.ResourceId.SpaceId)
	return events.SpaceUnshared{
		Executant:      executant.GetId(),
		GranteeUserID:  req.Grant.GetGrantee().GetUserId(),
		GranteeGroupID: req.Grant.GetGrantee().GetGroupId(),
		ID:             &provider.StorageSpaceId{OpaqueId: id},
		Timestamp:      time.Now(),
	}
}

// SpaceDisabled converts the response to an event
func SpaceDisabled(r *provider.DeleteStorageSpaceResponse, req *provider.DeleteStorageSpaceRequest, executant *user.User) events.SpaceDisabled {
	return events.SpaceDisabled{
		Executant: executant.GetId(),
		ID:        req.Id,
		Timestamp: time.Now(),
	}
}

// SpaceDeleted converts the response to an event
func SpaceDeleted(r *provider.DeleteStorageSpaceResponse, req *provider.DeleteStorageSpaceRequest, executant *user.User) events.SpaceDeleted {
	var final map[string]provider.ResourcePermissions
	_ = utils.ReadJSONFromOpaque(r.GetOpaque(), "grants", &final)
	return events.SpaceDeleted{
		Executant:    executant.GetId(),
		ID:           req.Id,
		SpaceName:    utils.ReadPlainFromOpaque(r.GetOpaque(), "spacename"),
		FinalMembers: final,
		Timestamp:    time.Now(),
	}
}

func extractOwner(u *user.User) *user.UserId {
	if u != nil {
		return u.Id
	}
	return nil
}

func extractImpersonator(u *user.User) *user.User {
	var impersonator user.User
	if err := utils.ReadJSONFromOpaque(u.Opaque, "impersonating-user", &impersonator); err != nil {
		return nil
	}
	return &impersonator
}
