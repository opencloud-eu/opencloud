package types

import (
	"fmt"
	"strings"

	"github.com/opencloud-eu/reva/v3/pkg/events"
	"github.com/opencloud-eu/reva/v3/pkg/storagespace"
)

// short identifiers for audit actions
const (
	// Sharing
	ActionShareCreated            = "file_shared"
	ActionSharePermissionUpdated  = "share_permission_updated"
	ActionShareDisplayNameUpdated = "share_name_updated"
	ActionSharePasswordUpdated    = "share_password_updated"
	ActionShareExpirationUpdated  = "share_expiration_updated"
	ActionShareRemoved            = "file_unshared"
	ActionShareAccepted           = "share_accepted"
	ActionShareDeclined           = "share_declined"
	ActionLinkAccessed            = "public_link_accessed"

	// Files
	ActionContainerCreated    = "container_create"
	ActionFileCreated         = "file_create"
	ActionFileRead            = "file_read"
	ActionFileTrashed         = "file_delete"
	ActionFileRenamed         = "file_rename"
	ActionFilePurged          = "file_trash_delete"
	ActionFileRestored        = "file_trash_restore"
	ActionFileVersionRestored = "file_version_restore"

	// Spaces
	ActionSpaceCreated  = "space_created"
	ActionSpaceRenamed  = "space_renamed"
	ActionSpaceDisabled = "space_disabled"
	ActionSpaceEnabled  = "space_enabled"
	ActionSpaceDeleted  = "space_deleted"
	ActionSpaceShared   = "space_shared"
	ActionSpaceUnshared = "space_unshared"
	ActionSpaceUpdated  = "space_updated"

	// Users
	ActionUserCreated        = "user_created"
	ActionUserDeleted        = "user_deleted"
	ActionUserFeatureChanged = "user_feature_changed"

	// Groups
	ActionGroupCreated       = "group_created"
	ActionGroupDeleted       = "group_deleted"
	ActionGroupMemberAdded   = "group_member_added"
	ActionGroupMemberRemoved = "group_member_removed"

	// ScienceMesh
	ActionScienceMeshInviteTokenGenerated = "science_mesh_invite_token_generated"
)

// MessageShareCreated returns the human-readable string that describes the action
func MessageShareCreated(sharer, item, grantee string) string {
	return fmt.Sprintf("user '%s' shared file '%s' with '%s'", sharer, item, grantee)
}

// MessageLinkCreated returns the human-readable string that describes the action
func MessageLinkCreated(sharer, item, shareID string) string {
	return fmt.Sprintf("user '%s' created a public link to file '%s' with id '%s'", sharer, item, shareID)
}

// MessageShareUpdated returns the human-readable string that describes the action
func MessageShareUpdated(sharer, shareID, fieldUpdated string) string {
	return fmt.Sprintf("user '%s' updated field '%s' of share '%s'", sharer, fieldUpdated, shareID)
}

// MessageLinkUpdated returns the human-readable string that describes the action
func MessageLinkUpdated(sharer, shareID, fieldUpdated string) string {
	return fmt.Sprintf("user '%s' updated field '%s' of public link '%s'", sharer, fieldUpdated, shareID)
}

// MessageShareRemoved returns the human-readable string that describes the action
func MessageShareRemoved(sharer, shareID, itemid string) string {
	return fmt.Sprintf("share id:'%s' uid:'%s' item-id:'%s' was removed", shareID, sharer, itemid)
}

// MessageLinkRemoved returns the human-readable string that describes the action
func MessageLinkRemoved(executant, shareID string) string {
	return fmt.Sprintf("user '%s' removed public link with id:'%s'", executant, shareID)
}

// MessageShareAccepted returns the human-readable string that describes the action
func MessageShareAccepted(userid, shareID, sharerID string) string {
	return fmt.Sprintf("user '%s' accepted share '%s' from user '%s'", userid, shareID, sharerID)
}

// MessageShareDeclined returns the human-readable string that describes the action
func MessageShareDeclined(userid, shareID, sharerID string) string {
	return fmt.Sprintf("user '%s' declined share '%s' from user '%s'", userid, shareID, sharerID)
}

// MessageLinkAccessed returns the human-readable string that describes the action
func MessageLinkAccessed(token string, success bool) string {
	return fmt.Sprintf("link with token '%s' was accessed. Success: %v", token, success)
}

// MessageContainerCreated returns the human-readable string that describes the action
func MessageContainerCreated(executant, item string) string {
	return fmt.Sprintf("user '%s' created folder '%s'", executant, item)
}

// MessageFileCreated returns the human-readable string that describes the action
func MessageFileCreated(executant, item string) string {
	return fmt.Sprintf("user '%s' created file '%s'", executant, item)
}

// MessageFileRead returns the human-readable string that describes the action
func MessageFileRead(executant, item string) string {
	return fmt.Sprintf("user '%s' read file '%s'", executant, item)
}

// MessageFileTrashed returns the human-readable string that describes the action
func MessageFileTrashed(executant, item string) string {
	return fmt.Sprintf("user '%s' trashed file '%s'", executant, item)
}

// MessageFileRenamed returns the human-readable string that describes the action
func MessageFileRenamed(executant, item, oldpath, newpath string) string {
	return fmt.Sprintf("user '%s' moved file '%s' from '%s' to '%s'", executant, item, oldpath, newpath)
}

// MessageFilePurged returns the human-readable string that describes the action
func MessageFilePurged(executant, item string) string {
	return fmt.Sprintf("user '%s' removed file '%s' from trashbin", executant, item)
}

// MessageFileRestored returns the human-readable string that describes the action
func MessageFileRestored(executant, item, path string) string {
	return fmt.Sprintf("user '%s' restored file '%s' from trashbin to '%s'", executant, item, path)
}

// MessageFileVersionRestored returns the human-readable string that describes the action
func MessageFileVersionRestored(executant, item, version string) string {
	return fmt.Sprintf("user '%s' restored file '%s' in version '%s'", executant, item, version)
}

// MessageSpaceCreated returns the human-readable string that describes the action
func MessageSpaceCreated(executant, spaceID, name string) string {
	storagID, spaceID := storagespace.SplitStorageID(spaceID)
	return fmt.Sprintf("user '%s' created a space '%s' with name '%s' (storage: '%s')", executant, spaceID, name, storagID)
}

// MessageSpaceRenamed returns the human-readable string that describes the action
func MessageSpaceRenamed(executant, spaceID, name string) string {
	storagID, spaceID := storagespace.SplitStorageID(spaceID)
	return fmt.Sprintf("user '%s' renamed space '%s' to '%s' (storage: '%s')", executant, spaceID, name, storagID)
}

// MessageSpaceDisabled returns the human-readable string that describes the action
func MessageSpaceDisabled(executant, spaceID string) string {
	storagID, spaceID := storagespace.SplitStorageID(spaceID)
	return fmt.Sprintf("user '%s' disabled the space '%s' (storage: '%s')", executant, spaceID, storagID)
}

// MessageSpaceEnabled returns the human-readable string that describes the action
func MessageSpaceEnabled(executant, spaceID string) string {
	storagID, spaceID := storagespace.SplitStorageID(spaceID)
	return fmt.Sprintf("user '%s' (re-) enabled the space '%s' (storage: '%s')", executant, spaceID, storagID)
}

// MessageSpaceDeleted returns the human-readable string that describes the action
func MessageSpaceDeleted(executant, spaceID, name string) string {
	storagID, spaceID := storagespace.SplitStorageID(spaceID)
	return fmt.Sprintf("user '%s' deleted the space '%s' with name '%s' (storage: '%s')", executant, spaceID, name, storagID)
}

// MessageSpaceShared returns the human-readable string that describes the action
func MessageSpaceShared(executant, spaceID, grantee string) string {
	storagID, spaceID := storagespace.SplitStorageID(spaceID)
	return fmt.Sprintf("user '%s' shared the space '%s' with '%s' (storage: '%s')", executant, spaceID, grantee, storagID)
}

// MessageSpaceUnshared returns the human-readable string that describes the action
func MessageSpaceUnshared(executant, spaceID, grantee string) string {
	storagID, spaceID := storagespace.SplitStorageID(spaceID)
	return fmt.Sprintf("user '%s' unshared the space '%s' with '%s' (storage: '%s')", executant, spaceID, grantee, storagID)
}

// MessageSpaceUpdated returns the human-readable string that describes the action
func MessageSpaceUpdated(executant, spaceID, name string, quota uint64, opaque map[string]string) string {
	storagID, spaceID := storagespace.SplitStorageID(spaceID)
	return fmt.Sprintf("user '%s' updated space '%s'. name: '%s', quota: '%d', opaque: '%s' (storage: '%s')",
		executant, spaceID, name, quota, opaque, storagID)
}

// MessageUserCreated returns the human-readable string that describes the action
func MessageUserCreated(executant, userID string) string {
	return fmt.Sprintf("user '%s' created the user '%s'", executant, userID)
}

// MessageUserDeleted returns the human-readable string that describes the action
func MessageUserDeleted(executant, userID string) string {
	return fmt.Sprintf("user '%s' deleted the user '%s'", executant, userID)
}

// MessageUserFeatureChanged returns the human-readable string that describes the action
func MessageUserFeatureChanged(executant, userID string, features []events.UserFeature) string {
	// Result is: "user '%executant%' changed user %username%'s features: %featurename%=%featurevalue% %featurename%=%featurevalue%"
	var sb strings.Builder
	sb.WriteString("user '")
	sb.WriteString(executant)
	sb.WriteString("' changed user ")
	sb.WriteString(userID)
	sb.WriteString("'s features:")
	for _, f := range features {
		sb.WriteString(f.Name)
		sb.WriteRune('=')
		sb.WriteString(f.Value)
		sb.WriteRune(' ')
	}
	return sb.String()
}

// MessageGroupCreated returns the human-readable string that describes the action
func MessageGroupCreated(executant, groupID string) string {
	return fmt.Sprintf("user '%s' created group '%s'", executant, groupID)
}

// MessageGroupDeleted returns the human-readable string that describes the action
func MessageGroupDeleted(executant, groupID string) string {
	return fmt.Sprintf("user '%s' deleted group '%s'", executant, groupID)
}

// MessageGroupMemberAdded returns the human-readable string that describes the action
func MessageGroupMemberAdded(executant, userID, groupID string) string {
	return fmt.Sprintf("user '%s' added user '%s' was added to group '%s'", executant, userID, groupID)
}

// MessageGroupMemberRemoved returns the human-readable string that describes the action
func MessageGroupMemberRemoved(executant, userID, groupID string) string {
	return fmt.Sprintf("user '%s' added user '%s' was removed from group '%s'", executant, userID, groupID)
}

// MessageScienceMeshInviteTokenGenerated returns the human-readable string that describes the action
func MessageScienceMeshInviteTokenGenerated(user, token string) string {
	return fmt.Sprintf("user '%s' generated a ScienceMesh invite with token '%s'", user, token)
}
