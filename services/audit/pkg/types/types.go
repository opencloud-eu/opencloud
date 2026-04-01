package types

import "github.com/opencloud-eu/reva/v2/pkg/events"

// AuditEvent is the basic audit event
type AuditEvent struct {
	RemoteAddr string // the remote client IP
	User       string // the UID of the user performing the action. Or "IP x.x.x.x.", "cron", "CLI", "unknown"
	URL        string // the process request URI
	Method     string // the HTTP request method
	UserAgent  string // the HTTP request user agent
	Time       string // the time of the event eg: 2018-05-08T08:26:00+00:00
	App        string // always 'admin_audit'
	Message    string // sentence explaining the action
	Action     string // unique action identifier eg: file_delete or public_link_created
	CLI        bool   // if the action was performed from the CLI
	Level      int    // the log level of the entry (usually 1 for audit events)
}

/*
   Sharing
*/

// AuditEventSharing is the basic audit event for shares
type AuditEventSharing struct {
	AuditEvent

	FileID  string // The file identifier for the item shared.
	Owner   string // The UID of the owner of the shared item.
	Path    string // The path to the shared item.
	ShareID string // The sharing identifier. (not available for public_link_accessed or when recipient un-shares)
}

// AuditEventShareCreated is the event logged when a share is created
type AuditEventShareCreated struct {
	AuditEventSharing

	ItemType       string // file or folder
	ExpirationDate string // The text expiration date in format 'yyyy-mm-dd'
	SharePass      bool   // If the share is password protected.
	Permissions    string // The permissions string eg: "READ"
	ShareType      string // group user or link
	ShareWith      string // The UID or GID of the share recipient. (not available for public link)
	ShareOwner     string // The UID of the share owner.
	ShareToken     string // For link shares the unique token, else null
}

// AuditEventShareUpdated is the event logged when a share is updated
type AuditEventShareUpdated struct {
	AuditEventSharing

	ItemType       string // file or folder
	ExpirationDate string // The text expiration date in format 'yyyy-mm-dd'
	SharePass      bool   // If the share is password protected.
	Permissions    string // The permissions string eg: "READ"
	ShareType      string // group user or link
	ShareWith      string // The UID or GID of the share recipient. (not available for public link)
	ShareOwner     string // The UID of the share owner.
	ShareToken     string // For link shares the unique token, else null
}

// AuditEventShareRemoved is the event logged when a share is removed
type AuditEventShareRemoved struct {
	AuditEventSharing
	ItemType  string // file or folder
	ShareType string // group user or link
	ShareWith string // The UID or GID of the share recipient.
}

// AuditEventReceivedShareUpdated is the event logged when a share is accepted or declined
type AuditEventReceivedShareUpdated struct {
	AuditEventSharing
	ItemType  string // file or folder
	ShareType string // group user or link
	ShareWith string // The UID or GID of the share recipient.
}

// AuditEventLinkAccessed is the event logged when a link is accessed
type AuditEventLinkAccessed struct {
	AuditEventSharing
	ShareToken string // The share token.
	Success    bool   // If the request was successful.
	ItemType   string // file or folder
}

/*
   Files
*/

// AuditEventFiles is the basic audit event for files
type AuditEventFiles struct {
	AuditEvent

	Path   string // The full path to the created file.
	Owner  string // The UID of the owner of the file.
	FileID string // The newly created files identifier.
}

// AuditEventContainerCreated is the event logged when a container is created
type AuditEventContainerCreated struct {
	AuditEventFiles
}

// AuditEventFileCreated is the event logged when a file is created
type AuditEventFileCreated struct {
	AuditEventFiles
}

// AuditEventFileRead is the event logged when a file is read (aka downloaded)
type AuditEventFileRead struct {
	AuditEventFiles
}

// AuditEventFileViewed is the event logged when a file is viewed (accessed without download intent)
type AuditEventFileViewed struct {
	AuditEventFiles
}

// AuditEventFileUpdated is the event logged when a file is updated
// TODO: How to differentiate between new uploads and new version uploads?
// FIXME: implement
type AuditEventFileUpdated struct {
	AuditEventFiles
}

// AuditEventFileDeleted is the event logged when a file is deleted (aka trashed)
type AuditEventFileDeleted struct {
	AuditEventFiles
}

// AuditEventFileCopied is the event logged when a file is copied
// TODO: copy is a download&upload for now. How to know it was a copy?
// FIXME: implement
type AuditEventFileCopied struct {
	AuditEventFiles
}

// AuditEventFileRenamed is the event logged when a file is renamed (moved)
type AuditEventFileRenamed struct {
	AuditEventFiles

	OldPath string
}

// AuditEventFilePurged is the event logged when a file is purged (deleted from trash-bin)
type AuditEventFilePurged struct {
	AuditEventFiles
}

// AuditEventFileRestored is the event logged when a file is restored (from trash-bin)
type AuditEventFileRestored struct {
	AuditEventFiles

	OldPath string
}

// AuditEventFileVersionRestored is the event logged when a file version is restored
type AuditEventFileVersionRestored struct {
	AuditEventFiles

	Key string
}

// AuditEventFileVersionDeleted is the event logged when a file version is deleted
// TODO: is this even possible?
type AuditEventFileVersionDeleted struct {
	AuditEventFiles
}

/*
   Spaces
*/

// AuditEventSpaces is the basic audit event for spaces
type AuditEventSpaces struct {
	AuditEvent

	SpaceID string
}

// AuditEventSpaceCreated is the event logged when a space is created
type AuditEventSpaceCreated struct {
	AuditEventSpaces

	Owner    string
	RootItem string
	Name     string
	Type     string
}

// AuditEventSpaceRenamed is the event logged when a space is renamed
type AuditEventSpaceRenamed struct {
	AuditEventSpaces

	NewName string
}

// AuditEventSpaceDisabled is the event logged when a space is disabled
type AuditEventSpaceDisabled struct {
	AuditEventSpaces
}

// AuditEventSpaceEnabled is the event logged when a space is (re-)enabled
type AuditEventSpaceEnabled struct {
	AuditEventSpaces
}

// AuditEventSpaceDeleted is the event logged when a space is deleted
type AuditEventSpaceDeleted struct {
	AuditEventSpaces
}

// AuditEventSpaceShared is the event logged when a space is shared
type AuditEventSpaceShared struct {
	AuditEventSpaces

	GranteeUserID  string
	GranteeGroupID string
}

// AuditEventSpaceUnshared is the event logged when a space is unshared
type AuditEventSpaceUnshared struct {
	AuditEventSpaces

	GranteeUserID  string
	GranteeGroupID string
}

// AuditEventSpaceUpdated is the event logged when a space is updated
type AuditEventSpaceUpdated struct {
	AuditEventSpaces

	Name          string
	Opaque        map[string]string
	QuotaMaxBytes uint64
}

// AuditEventUserCreated is the event logged when a user is created
type AuditEventUserCreated struct {
	AuditEvent
	UserID string
}

// AuditEventUserDeleted is the event logged when a user is deleted
type AuditEventUserDeleted struct {
	AuditEvent
	UserID string
}

// AuditEventUserFeatureChanged is the event logged when a user feature is changed
type AuditEventUserFeatureChanged struct {
	AuditEvent
	UserID   string
	Features []events.UserFeature
}

// AuditEventGroupCreated is the event logged when a group is created
type AuditEventGroupCreated struct {
	AuditEvent
	GroupID string
}

// AuditEventGroupDeleted is the event logged when a group is deleted
type AuditEventGroupDeleted struct {
	AuditEvent
	GroupID string
}

// AuditEventGroupMemberAdded is the event logged when a group member is added
type AuditEventGroupMemberAdded struct {
	AuditEvent
	GroupID string
	UserID  string
}

// AuditEventGroupMemberRemoved is the event logged when a group member is removed
type AuditEventGroupMemberRemoved struct {
	AuditEvent
	GroupID string
	UserID  string
}

// AuditEventScienceMeshInviteTokenGenerated is the event logged when a ScienceMesh invite token is generated
type AuditEventScienceMeshInviteTokenGenerated struct {
	AuditEvent
	RecipientMail string
	Token         string
	Description   string
	Expiration    uint64
	InviteLink    string
}
