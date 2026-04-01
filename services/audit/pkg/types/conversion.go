package types

import (
	"fmt"
	"strings"
	"time"

	"github.com/opencloud-eu/reva/v2/pkg/events"
	"github.com/opencloud-eu/reva/v2/pkg/storagespace"
	"github.com/opencloud-eu/reva/v2/pkg/utils"

	group "github.com/cs3org/go-cs3apis/cs3/identity/group/v1beta1"
	user "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"
	provider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	types "github.com/cs3org/go-cs3apis/cs3/types/v1beta1"

	sdk "github.com/opencloud-eu/reva/v2/pkg/sdk/common"
)

const _linktype = "link"

// BasicAuditEvent creates an AuditEvent from given values
func BasicAuditEvent(uid string, ctime string, msg string, action string) AuditEvent {
	return AuditEvent{
		User:    uid,
		Time:    ctime,
		App:     "admin_audit",
		Message: msg,
		Action:  action,
		Level:   1,

		// NOTE: those values are not in the events and can therefore not be filled at the moment
		RemoteAddr: "",
		URL:        "",
		Method:     "",
		UserAgent:  "",
		CLI:        false,
	}
}

// SharingAuditEvent creates an AuditEventSharing from given values
func SharingAuditEvent(shareid string, fileid string, uid string, base AuditEvent) AuditEventSharing {
	return AuditEventSharing{
		AuditEvent: base,
		FileID:     fileid,
		Owner:      uid,
		ShareID:    shareid,

		// NOTE: those values are not in the events and can therefore not be filled at the moment
		Path: "",
	}
}

// ShareCreated converts a ShareCreated Event to an AuditEventShareCreated
func ShareCreated(ev events.ShareCreated) AuditEventShareCreated {
	uid := ev.Sharer.OpaqueId
	grantee, typ := extractGrantee(ev.GranteeUserID, ev.GranteeGroupID)
	base := BasicAuditEvent(uid, formatTime(ev.CTime), MessageShareCreated(uid, ev.ItemID.OpaqueId, grantee), ActionShareCreated)
	return AuditEventShareCreated{
		AuditEventSharing: SharingAuditEvent("", ev.ItemID.OpaqueId, uid, base),
		ShareOwner:        uid,
		ShareWith:         grantee,
		ShareType:         typ,

		// NOTE: those values are not in the event and can therefore not be filled at the moment
		ItemType:       "",
		ExpirationDate: "",
		SharePass:      false,
		Permissions:    "",
		ShareToken:     "",
	}
}

// LinkCreated converts a ShareCreated Event to an AuditEventShareCreated
func LinkCreated(ev events.LinkCreated) AuditEventShareCreated {
	uid := ev.Sharer.OpaqueId
	with, typ := "", _linktype
	base := BasicAuditEvent(uid, formatTime(ev.CTime), MessageLinkCreated(uid, ev.ItemID.OpaqueId, ev.ShareID.OpaqueId), ActionShareCreated)
	return AuditEventShareCreated{
		AuditEventSharing: SharingAuditEvent("", ev.ItemID.OpaqueId, uid, base),
		ShareOwner:        uid,
		ShareWith:         with,
		ShareType:         typ,
		ExpirationDate:    formatTime(ev.Expiration),
		SharePass:         ev.PasswordProtected,
		Permissions:       normalizeString(ev.Permissions.GetPermissions().String()),
		ShareToken:        ev.Token,

		// NOTE: those values are not in the event and can therefore not be filled at the moment
		ItemType: "",
	}
}

// ShareUpdated converts a ShareUpdated event to an AuditEventShareUpdated
func ShareUpdated(ev events.ShareUpdated) AuditEventShareUpdated {
	uid := ev.Sharer.OpaqueId
	with, typ := extractGrantee(ev.GranteeUserID, ev.GranteeGroupID)
	base := BasicAuditEvent(uid, formatTime(ev.MTime), MessageShareUpdated(uid, ev.ShareID.OpaqueId, ev.Updated), updateType(ev.Updated))
	return AuditEventShareUpdated{
		AuditEventSharing: SharingAuditEvent(ev.ShareID.GetOpaqueId(), ev.ItemID.OpaqueId, uid, base),
		ShareOwner:        uid,
		ShareWith:         with,
		ShareType:         typ,
		Permissions:       normalizeString(ev.Permissions.GetPermissions().String()),

		// NOTE: those values are not in the event and can therefore not be filled at the moment
		ItemType:       "",
		ExpirationDate: "",
		SharePass:      false,
		ShareToken:     "",
	}
}

// LinkUpdated converts a LinkUpdated event to an AuditEventShareUpdated
func LinkUpdated(ev events.LinkUpdated) AuditEventShareUpdated {
	uid := ev.Sharer.OpaqueId
	with, typ := "", _linktype
	base := BasicAuditEvent(uid, formatTime(ev.MTime), MessageLinkUpdated(uid, ev.ShareID.GetOpaqueId(), ev.FieldUpdated), updateType(ev.FieldUpdated))
	return AuditEventShareUpdated{
		AuditEventSharing: SharingAuditEvent(ev.ShareID.GetOpaqueId(), ev.ItemID.OpaqueId, uid, base),
		ShareOwner:        uid,
		ShareWith:         with,
		ShareType:         typ,
		Permissions:       normalizeString(ev.Permissions.GetPermissions().String()),
		ExpirationDate:    formatTime(ev.Expiration),
		SharePass:         ev.PasswordProtected,
		ShareToken:        ev.Token,

		// NOTE: those values are not in the event and can therefore not be filled at the moment
		ItemType: "",
	}
}

// ShareRemoved converts a ShareRemoved event to an AuditEventShareRemoved
func ShareRemoved(ev events.ShareRemoved) AuditEventShareRemoved {
	sid, uid, iid, with, typ := "", "", "", "", ""
	if ev.ShareID != nil {
		sid = ev.ShareID.GetOpaqueId()
	}

	if ev.ShareKey != nil {
		uid = ev.ShareKey.GetOwner().GetOpaqueId()
		iid = ev.ShareKey.GetResourceId().GetOpaqueId()
		with, typ = extractGrantee(ev.ShareKey.GetGrantee().GetUserId(), ev.ShareKey.GetGrantee().GetGroupId())
	}
	base := BasicAuditEvent(uid, formatTime(utils.TimeToTS(ev.Timestamp)), MessageShareRemoved(uid, sid, iid), ActionShareRemoved)
	return AuditEventShareRemoved{
		AuditEventSharing: SharingAuditEvent(sid, iid, uid, base),
		ShareWith:         with,
		ShareType:         typ,

		// NOTE: those values are not in the event and can therefore not be filled at the moment
		ItemType: "",
	}
}

// LinkRemoved converts a LinkRemoved event to an AuditEventShareRemoved
func LinkRemoved(ev events.LinkRemoved) AuditEventShareRemoved {
	uid, sid, typ := ev.Executant.GetOpaqueId(), "", _linktype
	if ev.ShareID != nil {
		sid = ev.ShareID.GetOpaqueId()
	} else {
		sid = ev.ShareToken
	}

	base := BasicAuditEvent(uid, formatTime(ev.Timestamp), MessageLinkRemoved(uid, sid), ActionShareRemoved)
	return AuditEventShareRemoved{
		AuditEventSharing: SharingAuditEvent(sid, "", uid, base),
		ShareWith:         "",
		ShareType:         typ,

		// NOTE: those values are not in the event and can therefore not be filled at the moment
		ItemType: "",
	}
}

// ReceivedShareUpdated converts a ReceivedShareUpdated event to an AuditEventReceivedShareUpdated
func ReceivedShareUpdated(ev events.ReceivedShareUpdated) AuditEventReceivedShareUpdated {
	uid := ev.Sharer.GetOpaqueId()
	sid := ev.ShareID.GetOpaqueId()
	with, typ := extractGrantee(ev.GranteeUserID, ev.GranteeGroupID)
	itemID := ev.ItemID.GetOpaqueId()

	msg, utype := "", ""
	switch ev.State {
	case "SHARE_STATE_ACCEPTED":
		msg = MessageShareAccepted(with, sid, uid)
		utype = ActionShareAccepted
	case "SHARE_STATE_DECLINED":
		msg = MessageShareDeclined(with, sid, uid)
		utype = ActionShareDeclined
	}
	base := BasicAuditEvent(with, formatTime(ev.MTime), msg, utype)
	return AuditEventReceivedShareUpdated{
		AuditEventSharing: SharingAuditEvent(sid, itemID, uid, base),
		ShareType:         typ,
		ShareWith:         with,

		// NOTE: those values are not in the event and can therefore not be filled at the moment
		ItemType: "",
	}
}

// LinkAccessed converts a LinkAccessed event to an AuditEventLinkAccessed
func LinkAccessed(ev events.LinkAccessed) AuditEventLinkAccessed {
	uid := ev.Sharer.OpaqueId
	base := BasicAuditEvent(uid, formatTime(ev.CTime), MessageLinkAccessed(ev.Token, true), ActionLinkAccessed)
	return AuditEventLinkAccessed{
		AuditEventSharing: SharingAuditEvent(ev.ShareID.GetOpaqueId(), ev.ItemID.OpaqueId, uid, base),
		ShareToken:        ev.Token,
		Success:           true,

		// NOTE: those values are not in the event and can therefore not be filled at the moment
		ItemType: "",
	}
}

// LinkAccessFailed converts a LinkAccessFailed event to an AuditEventLinkAccessed
func LinkAccessFailed(ev events.LinkAccessFailed) AuditEventLinkAccessed {
	base := BasicAuditEvent("", formatTime(ev.Timestamp), MessageLinkAccessed(ev.Token, false), ActionLinkAccessed)
	return AuditEventLinkAccessed{
		AuditEventSharing: SharingAuditEvent(ev.ShareID.GetOpaqueId(), "", "", base),
		ShareToken:        ev.Token,
		Success:           false,

		// NOTE: those values are not in the event and can therefore not be filled at the moment
		ItemType: "",
	}
}

// FilesAuditEvent creates an AuditEventFiles from the given values
func FilesAuditEvent(base AuditEvent, itemid, owner, path string) AuditEventFiles {
	return AuditEventFiles{
		AuditEvent: base,
		FileID:     itemid,
		Owner:      owner,
		Path:       path,
	}
}

// ContainerCreated converts a ContainerCreated event to an AuditEventContainerCreated
func ContainerCreated(ev events.ContainerCreated) AuditEventContainerCreated {
	iid, path, uid := extractFileDetails(ev.Ref, ev.Owner)
	base := BasicAuditEvent(uid, formatTime(ev.Timestamp), MessageContainerCreated(ev.Executant.GetOpaqueId(), iid), ActionContainerCreated)
	return AuditEventContainerCreated{
		AuditEventFiles: FilesAuditEvent(base, iid, uid, path),
	}
}

// FileUploaded converts a FileUploaded event to an AuditEventFileCreated
func FileUploaded(ev events.FileUploaded) AuditEventFileCreated {
	iid, path, uid := extractFileDetails(ev.Ref, ev.Owner)
	base := BasicAuditEvent(uid, formatTime(ev.Timestamp), MessageFileCreated(ev.Executant.GetOpaqueId(), iid), ActionFileCreated)
	return AuditEventFileCreated{
		AuditEventFiles: FilesAuditEvent(base, iid, uid, path),
	}
}

// FileDownloaded converts a FileDownloaded event to an AuditEventFileRead
func FileDownloaded(ev events.FileDownloaded) AuditEventFileRead {
	iid, path, uid := extractFileDetails(ev.Ref, ev.Owner)
	base := BasicAuditEvent(uid, formatTime(ev.Timestamp), MessageFileRead(ev.Executant.GetOpaqueId(), iid), ActionFileRead)
	return AuditEventFileRead{
		AuditEventFiles: FilesAuditEvent(base, iid, uid, path),
	}
}

// FileViewed converts a FileViewed event to an AuditEventFileViewed
func FileViewed(ev events.FileViewed) AuditEventFileViewed {
	iid, path, uid := extractFileDetails(ev.Ref, ev.Owner)
	base := BasicAuditEvent(uid, formatTime(ev.Timestamp), MessageFileRead(ev.Executant.GetOpaqueId(), iid), "file_viewed")
	return AuditEventFileViewed{
		AuditEventFiles: FilesAuditEvent(base, iid, uid, path),
	}
}

// ItemMoved converts a ItemMoved event to an AuditEventFileRenamed
func ItemMoved(ev events.ItemMoved) AuditEventFileRenamed {
	iid, path, uid := extractFileDetails(ev.Ref, ev.Owner)

	oldpath := ""
	if ev.OldReference != nil {
		oldpath = ev.OldReference.GetPath()
	}

	base := BasicAuditEvent(uid, formatTime(ev.Timestamp), MessageFileRenamed(ev.Executant.GetOpaqueId(), iid, oldpath, path), ActionFileRenamed)
	return AuditEventFileRenamed{
		AuditEventFiles: FilesAuditEvent(base, iid, uid, path),
		OldPath:         oldpath,
	}
}

// ItemTrashed converts a ItemTrashed event to an AuditEventFileDeleted
func ItemTrashed(ev events.ItemTrashed) AuditEventFileDeleted {
	iid, path, uid := extractFileDetails(ev.Ref, ev.Owner)
	base := BasicAuditEvent(uid, formatTime(ev.Timestamp), MessageFileTrashed(ev.Executant.GetOpaqueId(), iid), ActionFileTrashed)
	return AuditEventFileDeleted{
		AuditEventFiles: FilesAuditEvent(base, iid, uid, path),
	}
}

// ItemPurged converts a ItemPurged event to an AuditEventFilePurged
func ItemPurged(ev events.ItemPurged) AuditEventFilePurged {
	iid, path, uid := extractFileDetails(ev.Ref, ev.Owner)
	base := BasicAuditEvent(uid, formatTime(ev.Timestamp), MessageFilePurged(ev.Executant.GetOpaqueId(), iid), ActionFilePurged)
	return AuditEventFilePurged{
		AuditEventFiles: FilesAuditEvent(base, iid, uid, path),
	}
}

// ItemRestored converts a ItemRestored event to an AuditEventFileRestored
func ItemRestored(ev events.ItemRestored) AuditEventFileRestored {
	iid, path, uid := extractFileDetails(ev.Ref, ev.Owner)

	oldpath := ""
	if ev.OldReference != nil {
		oldpath = ev.OldReference.GetPath()
	}

	base := BasicAuditEvent(uid, formatTime(ev.Timestamp), MessageFileRestored(ev.Executant.GetOpaqueId(), iid, path), ActionFileRestored)
	return AuditEventFileRestored{
		AuditEventFiles: FilesAuditEvent(base, iid, uid, path),
		OldPath:         oldpath,
	}
}

// FileVersionRestored converts a FileVersionRestored event to an AuditEventFileVersionRestored
func FileVersionRestored(ev events.FileVersionRestored) AuditEventFileVersionRestored {
	iid, path, uid := extractFileDetails(ev.Ref, ev.Owner)
	base := BasicAuditEvent(uid, formatTime(ev.Timestamp), MessageFileVersionRestored(ev.Executant.GetOpaqueId(), iid, ev.Key), ActionFileVersionRestored)
	return AuditEventFileVersionRestored{
		AuditEventFiles: FilesAuditEvent(base, iid, uid, path),
		Key:             ev.Key,
	}
}

// SpacesAuditEvent creates an AuditEventSpaces from the given values
func SpacesAuditEvent(base AuditEvent, spaceID string) AuditEventSpaces {
	return AuditEventSpaces{
		AuditEvent: base,
		SpaceID:    spaceID,
	}
}

// SpaceCreated converts a SpaceCreated event to an AuditEventSpaceCreated
func SpaceCreated(ev events.SpaceCreated) AuditEventSpaceCreated {
	sid := ev.ID.GetOpaqueId()
	iid, _, owner := extractFileDetails(&provider.Reference{ResourceId: ev.Root}, ev.Owner)
	base := BasicAuditEvent("", formatTime(ev.MTime), MessageSpaceCreated(ev.Executant.GetOpaqueId(), sid, ev.Name), ActionSpaceCreated)
	return AuditEventSpaceCreated{
		AuditEventSpaces: SpacesAuditEvent(base, sid),
		Owner:            owner,
		RootItem:         iid,
		Name:             ev.Name,
		Type:             ev.Type,
	}
}

// SpaceRenamed converts a SpaceRenamed event to an AuditEventSpaceRenamed
func SpaceRenamed(ev events.SpaceRenamed) AuditEventSpaceRenamed {
	sid := ev.ID.GetOpaqueId()
	base := BasicAuditEvent("", formatTime(ev.Timestamp), MessageSpaceRenamed(ev.Executant.GetOpaqueId(), sid, ev.Name), ActionSpaceRenamed)
	return AuditEventSpaceRenamed{
		AuditEventSpaces: SpacesAuditEvent(base, sid),
		NewName:          ev.Name,
	}
}

// SpaceDisabled converts a SpaceDisabled event to an AuditEventSpaceDisabled
func SpaceDisabled(ev events.SpaceDisabled) AuditEventSpaceDisabled {
	sid := ev.ID.GetOpaqueId()
	base := BasicAuditEvent("", formatTime(utils.TimeToTS(ev.Timestamp)), MessageSpaceDisabled(ev.Executant.GetOpaqueId(), sid), ActionSpaceDisabled)
	return AuditEventSpaceDisabled{
		AuditEventSpaces: SpacesAuditEvent(base, sid),
	}
}

// SpaceEnabled converts a SpaceEnabled event to an AuditEventSpaceEnabled
func SpaceEnabled(ev events.SpaceEnabled) AuditEventSpaceEnabled {
	sid := ev.ID.GetOpaqueId()
	base := BasicAuditEvent("", formatTime(ev.Timestamp), MessageSpaceEnabled(ev.Executant.GetOpaqueId(), sid), ActionSpaceEnabled)
	return AuditEventSpaceEnabled{
		AuditEventSpaces: SpacesAuditEvent(base, sid),
	}
}

// SpaceDeleted converts a SpaceDeleted event to an AuditEventSpaceDeleted
func SpaceDeleted(ev events.SpaceDeleted) AuditEventSpaceDeleted {
	sid := ev.ID.GetOpaqueId()
	base := BasicAuditEvent("", formatTime(utils.TimeToTS(ev.Timestamp)), MessageSpaceDeleted(ev.Executant.GetOpaqueId(), sid), ActionSpaceDeleted)
	return AuditEventSpaceDeleted{
		AuditEventSpaces: SpacesAuditEvent(base, sid),
	}
}

// SpaceShared converts a SpaceShared event to an AuditEventSpaceShared
func SpaceShared(ev events.SpaceShared) AuditEventSpaceShared {
	sse := AuditEventSpaceShared{}

	sid := ev.ID.GetOpaqueId()
	grantee := "N/A"
	if ev.GranteeUserID != nil {
		sse.GranteeUserID = ev.GranteeUserID.OpaqueId
		grantee = "user:" + ev.GranteeUserID.OpaqueId
	} else if ev.GranteeGroupID != nil {
		sse.GranteeGroupID = ev.GranteeGroupID.OpaqueId
		grantee = "group:" + ev.GranteeGroupID.OpaqueId
	}
	base := BasicAuditEvent("", "", MessageSpaceShared(ev.Executant.GetOpaqueId(), sid, grantee), ActionSpaceShared)
	sse.AuditEventSpaces = SpacesAuditEvent(base, sid)

	return sse
}

// SpaceUnshared converts a SpaceUnshared event to an AuditEventSpaceUnshared
func SpaceUnshared(ev events.SpaceUnshared) AuditEventSpaceUnshared {
	sue := AuditEventSpaceUnshared{}

	sid := ev.ID.GetOpaqueId()
	grantee := "N/A"
	if ev.GranteeUserID != nil {
		sue.GranteeUserID = ev.GranteeUserID.OpaqueId
		grantee = "user:" + ev.GranteeUserID.OpaqueId
	} else if ev.GranteeGroupID != nil {
		sue.GranteeGroupID = ev.GranteeGroupID.OpaqueId
		grantee = "group:" + ev.GranteeGroupID.OpaqueId
	}
	base := BasicAuditEvent("", formatTime(utils.TimeToTS(ev.Timestamp)), MessageSpaceUnshared(ev.Executant.GetOpaqueId(), sid, grantee), ActionSpaceUnshared)
	sue.AuditEventSpaces = SpacesAuditEvent(base, sid)

	return sue
}

// SpaceUpdated converts a SpaceUpdated event to an AuditEventSpaceUpdated
func SpaceUpdated(ev events.SpaceUpdated) AuditEventSpaceUpdated {
	sid := ev.ID.GetOpaqueId()
	opaqueMap := sdk.DecodeOpaqueMap(ev.Space.Opaque)
	sue := AuditEventSpaceUpdated{
		Name:   ev.Space.Name,
		Opaque: opaqueMap,
	}

	base := BasicAuditEvent("", formatTime(ev.Timestamp), MessageSpaceUpdated(ev.Executant.GetOpaqueId(), sid, ev.Space.Name, ev.Space.Quota.QuotaMaxBytes, opaqueMap), ActionSpaceUpdated)
	sue.AuditEventSpaces = SpacesAuditEvent(base, sid)

	return sue
}

// UserCreated converts a UserCreated event to an AuditEventUserCreated
func UserCreated(ev events.UserCreated) AuditEventUserCreated {
	base := BasicAuditEvent("", formatTime(ev.Timestamp), MessageUserCreated(ev.Executant.GetOpaqueId(), ev.UserID), ActionUserCreated)
	return AuditEventUserCreated{
		AuditEvent: base,
		UserID:     ev.UserID,
	}
}

// UserDeleted converts a UserDeleted event to an AuditEventUserDeleted
func UserDeleted(ev events.UserDeleted) AuditEventUserDeleted {
	base := BasicAuditEvent("", formatTime(ev.Timestamp), MessageUserDeleted(ev.Executant.GetOpaqueId(), ev.UserID), ActionUserDeleted)
	return AuditEventUserDeleted{
		AuditEvent: base,
		UserID:     ev.UserID,
	}
}

// UserFeatureChanged converts a UserFeatureChanged event to an AuditEventUserFeatureChanged
func UserFeatureChanged(ev events.UserFeatureChanged) AuditEventUserFeatureChanged {
	msg := MessageUserFeatureChanged(ev.Executant.GetOpaqueId(), ev.UserID, ev.Features)
	base := BasicAuditEvent("", formatTime(ev.Timestamp), msg, ActionUserFeatureChanged)
	return AuditEventUserFeatureChanged{
		AuditEvent: base,
		UserID:     ev.UserID,
		Features:   ev.Features,
	}
}

// GroupCreated converts a GroupCreated event to an AuditEventGroupCreated
func GroupCreated(ev events.GroupCreated) AuditEventGroupCreated {
	base := BasicAuditEvent("", formatTime(ev.Timestamp), MessageGroupCreated(ev.Executant.GetOpaqueId(), ev.GroupID), ActionGroupCreated)
	return AuditEventGroupCreated{
		AuditEvent: base,
		GroupID:    ev.GroupID,
	}
}

// GroupDeleted converts a GroupDeleted event to an AuditEventGroupDeleted
func GroupDeleted(ev events.GroupDeleted) AuditEventGroupDeleted {
	base := BasicAuditEvent("", formatTime(ev.Timestamp), MessageGroupDeleted(ev.Executant.GetOpaqueId(), ev.GroupID), ActionGroupDeleted)
	return AuditEventGroupDeleted{
		AuditEvent: base,
		GroupID:    ev.GroupID,
	}
}

// GroupMemberAdded converts a GroupMemberAdded event to an AuditEventGroupMemberAdded
func GroupMemberAdded(ev events.GroupMemberAdded) AuditEventGroupMemberAdded {
	msg := MessageGroupMemberAdded(ev.Executant.GetOpaqueId(), ev.GroupID, ev.UserID)
	base := BasicAuditEvent("", formatTime(ev.Timestamp), msg, ActionGroupMemberAdded)
	return AuditEventGroupMemberAdded{
		AuditEvent: base,
		GroupID:    ev.GroupID,
		UserID:     ev.UserID,
	}
}

// GroupMemberRemoved converts a GroupMemberRemoved event to an AuditEventGroupMemberRemove
func GroupMemberRemoved(ev events.GroupMemberRemoved) AuditEventGroupMemberRemoved {
	msg := MessageGroupMemberRemoved(ev.Executant.GetOpaqueId(), ev.GroupID, ev.UserID)
	base := BasicAuditEvent("", formatTime(ev.Timestamp), msg, ActionGroupMemberRemoved)
	return AuditEventGroupMemberRemoved{
		AuditEvent: base,
		GroupID:    ev.GroupID,
		UserID:     ev.UserID,
	}
}

// ScienceMeshInviteTokenGenerated converts a ScienceMeshInviteTokenGenerated event to an AuditEventScienceMeshInviteTokenGenerated
func ScienceMeshInviteTokenGenerated(ev events.ScienceMeshInviteTokenGenerated) AuditEventScienceMeshInviteTokenGenerated {
	msg := MessageScienceMeshInviteTokenGenerated(ev.Sharer.GetOpaqueId(), ev.Token)
	base := BasicAuditEvent(ev.Sharer.GetOpaqueId(), formatTime(ev.Timestamp), msg, ActionScienceMeshInviteTokenGenerated)
	return AuditEventScienceMeshInviteTokenGenerated{
		AuditEvent:    base,
		RecipientMail: ev.RecipientMail,
		Token:         ev.Token,
		Description:   ev.Description,
		Expiration:    ev.Expiration,
		InviteLink:    ev.InviteLink,
	}
}

func extractGrantee(uid *user.UserId, gid *group.GroupId) (string, string) {
	switch {
	case uid != nil && uid.OpaqueId != "":
		return uid.OpaqueId, "user"
	case gid != nil && gid.OpaqueId != "":
		return gid.OpaqueId, "group"
	}

	return "", ""
}

func extractFileDetails(ref *provider.Reference, owner *user.UserId) (string, string, string) {
	id, path := "", ""
	if ref != nil {
		path = ref.GetPath()
		id, _ = storagespace.FormatReference(ref)
	}

	uid := ""
	if owner != nil {
		uid = owner.GetOpaqueId()
	}
	return id, path, uid
}

func formatTime(t *types.Timestamp) string {
	if t == nil {
		return ""
	}
	return time.Unix(int64(t.Seconds), int64(t.Nanos)).UTC().Format(time.RFC3339)
}

func updateType(u string) string {
	switch u {
	case "permissions":
		return ActionSharePermissionUpdated
	case "displayname":
		return ActionShareDisplayNameUpdated
	case "TYPE_PERMISSIONS":
		return ActionSharePermissionUpdated
	case "TYPE_DISPLAYNAME":
		return ActionShareDisplayNameUpdated
	case "TYPE_PASSWORD":
		return ActionSharePasswordUpdated
	case "TYPE_EXPIRATION":
		return ActionShareExpirationUpdated
	default:
		fmt.Println("Unknown update type", u)
		return ""
	}
}

// normalizeString tries to create a somewhat stable string
// from a prototext string. The prototext strings are unstable
// on purpose an insert additional spaces randomly.
// See: https://protobuf.dev/reference/go/faq/#unstable-text
func normalizeString(str string) string {
	return strings.Join(strings.Fields(str), " ")
}
