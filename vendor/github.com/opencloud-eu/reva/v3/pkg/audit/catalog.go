package audit

import (
	"fmt"
	"sort"
)

// OCSF 1.9.0 class UIDs used by the catalog.
const (
	OCSFDetectionFinding          = 2004
	OCSFAuthentication            = 3002
	OCSFEntityManagement          = 3004
	OCSFGroupManagement           = 3006
	OCSFUserManagement            = 3007
	OCSFRoleManagement            = 3008
	OCSFAPIActivity               = 6003
	OCSFWebResourceAccessActivity = 6004
	OCSFFileHostingActivity       = 6006
)

// ocsfOther is the OCSF activity "Other", used when a class has no matching activity.
const ocsfOther = 99

// Definition describes one event of the catalog. Every event is defined once
// here; the emitter validates against it and the catalog docs are generated
// from it.
type Definition struct {
	// Name is the stable event name, opencloud.<object>.<action>.
	Name string
	// Category groups events for the docs and the stream subject.
	Category string
	// TargetType is the type of object the event acts on.
	TargetType string
	// code is the stable base code, e.g. F012. Never change or reuse it.
	code string
	// Severity on success. Failures and denials raise it to at least warning.
	Severity Severity
	// OCSFClass and OCSFActivity map the event to OCSF 1.9.0.
	OCSFClass    int
	OCSFActivity int
	// OCSFDeniedActivity, when set, replaces OCSFActivity for failures and denials.
	OCSFDeniedActivity int
}

// Code returns the event code for the given outcome: the base code plus
// S (success), F (failure) or D (denied), e.g. F012S.
func (d *Definition) Code(o Outcome) string {
	switch o {
	case OutcomeFailure:
		return d.code + "F"
	case OutcomeDenied:
		return d.code + "D"
	default:
		return d.code + "S"
	}
}

func (d *Definition) activity(o Outcome) int {
	if o != OutcomeSuccess && d.OCSFDeniedActivity != 0 {
		return d.OCSFDeniedActivity
	}
	return d.OCSFActivity
}

func (d *Definition) severity(o Outcome) Severity {
	if o != OutcomeSuccess && d.Severity.rank() < SeverityWarning.rank() {
		return SeverityWarning
	}
	return d.Severity
}

// Categories
const (
	CategoryAuthentication = "authentication"
	CategoryCredentials    = "credentials"
	CategoryFiles          = "files"
	CategorySharing        = "sharing"
	CategorySpaces         = "spaces"
	CategoryIdentity       = "identity"
	CategoryAdministration = "administration"
	CategorySecurity       = "security"
)

var registry = map[string]*Definition{}

func define(name, category, targetType, code string, sev Severity, class, activity int) *Definition {
	d := &Definition{
		Name:         "opencloud." + name,
		Category:     category,
		TargetType:   targetType,
		code:         code,
		Severity:     sev,
		OCSFClass:    class,
		OCSFActivity: activity,
	}
	if _, ok := registry[d.Name]; ok {
		panic(fmt.Sprintf("audit: event %s defined twice", d.Name))
	}
	for _, other := range registry {
		if other.code == code {
			panic(fmt.Sprintf("audit: code %s used by %s and %s", code, other.Name, d.Name))
		}
	}
	registry[d.Name] = d
	return d
}

func deniedActivity(d *Definition, activity int) *Definition {
	d.OCSFDeniedActivity = activity
	return d
}

// Authentication
var (
	AuthLogin             = define("auth.login", CategoryAuthentication, "user", "A001", SeverityInfo, OCSFAuthentication, 1)
	AuthBackchannelLogout = define("auth.backchannel_logout", CategoryAuthentication, "user", "A002", SeverityInfo, OCSFAuthentication, 2)
)

// Credentials
var (
	AppTokenCreate     = define("app_token.create", CategoryCredentials, "app_token", "C001", SeverityNotice, OCSFUserManagement, 18)
	AppTokenDelete     = define("app_token.delete", CategoryCredentials, "app_token", "C002", SeverityNotice, OCSFUserManagement, 19)
	GuestTokenCreate   = define("guest_token.create", CategoryCredentials, "guest_token", "C003", SeverityNotice, OCSFUserManagement, 18)
	GuestTokenRenew    = define("guest_token.renew", CategoryCredentials, "guest_token", "C004", SeverityNotice, OCSFUserManagement, 18)
	UserPasswordChange = define("user.password.change", CategoryCredentials, "user", "C005", SeverityNotice, OCSFUserManagement, 8)
	UserPasswordReset  = define("user.password.reset", CategoryCredentials, "user", "C006", SeverityNotice, OCSFUserManagement, 9)
)

// Files
var (
	FileUpload         = define("file.upload", CategoryFiles, "file", "F001", SeverityInfo, OCSFFileHostingActivity, 1)
	FileDownload       = define("file.download", CategoryFiles, "file", "F002", SeverityInfo, OCSFFileHostingActivity, 2)
	FilePreview        = define("file.preview", CategoryFiles, "file", "F003", SeverityInfo, OCSFFileHostingActivity, 9)
	FileOpenInApp      = define("file.open_in_app", CategoryFiles, "file", "F004", SeverityInfo, OCSFFileHostingActivity, 14)
	FileRename         = define("file.rename", CategoryFiles, "file", "F005", SeverityInfo, OCSFFileHostingActivity, 5)
	FileMove           = define("file.move", CategoryFiles, "file", "F006", SeverityInfo, OCSFFileHostingActivity, 7)
	FileCopy           = define("file.copy", CategoryFiles, "file", "F007", SeverityInfo, OCSFFileHostingActivity, 6)
	FileDelete         = define("file.delete", CategoryFiles, "file", "F008", SeverityInfo, OCSFFileHostingActivity, 4)
	FileRestore        = define("file.restore", CategoryFiles, "file", "F009", SeverityInfo, OCSFFileHostingActivity, 8)
	FilePurge          = define("file.purge", CategoryFiles, "file", "F010", SeverityNotice, OCSFFileHostingActivity, 4)
	FileLock           = define("file.lock", CategoryFiles, "file", "F011", SeverityInfo, OCSFFileHostingActivity, 10)
	FileUnlock         = define("file.unlock", CategoryFiles, "file", "F012", SeverityInfo, OCSFFileHostingActivity, 11)
	FileVersionRestore = define("file.version.restore", CategoryFiles, "file", "F013", SeverityInfo, OCSFFileHostingActivity, 8)
	FileVersionDelete  = define("file.version.delete", CategoryFiles, "file", "F014", SeverityNotice, OCSFFileHostingActivity, 4)
	FileMalwareDetect  = define("file.malware.detected", CategoryFiles, "file", "F015", SeverityAlert, OCSFDetectionFinding, 1)
	FolderCreate       = define("folder.create", CategoryFiles, "folder", "F016", SeverityInfo, OCSFFileHostingActivity, ocsfOther)
	FolderDownload     = define("folder.download", CategoryFiles, "folder", "F017", SeverityInfo, OCSFFileHostingActivity, 2)
	FolderRename       = define("folder.rename", CategoryFiles, "folder", "F018", SeverityInfo, OCSFFileHostingActivity, 5)
	FolderMove         = define("folder.move", CategoryFiles, "folder", "F019", SeverityInfo, OCSFFileHostingActivity, 7)
	FolderCopy         = define("folder.copy", CategoryFiles, "folder", "F020", SeverityInfo, OCSFFileHostingActivity, 6)
	FolderDelete       = define("folder.delete", CategoryFiles, "folder", "F021", SeverityInfo, OCSFFileHostingActivity, 4)
	FolderRestore      = define("folder.restore", CategoryFiles, "folder", "F022", SeverityInfo, OCSFFileHostingActivity, 8)
	FolderPurge        = define("folder.purge", CategoryFiles, "folder", "F023", SeverityNotice, OCSFFileHostingActivity, 4)
	TrashPurgeAll      = define("trash.purge_all", CategoryFiles, "space", "F024", SeverityNotice, OCSFFileHostingActivity, 4)
)

// Sharing
var (
	ShareCreate      = define("share.create", CategorySharing, "share", "S001", SeverityInfo, OCSFFileHostingActivity, 12)
	ShareUpdate      = define("share.update", CategorySharing, "share", "S002", SeverityInfo, OCSFFileHostingActivity, 12)
	ShareDelete      = define("share.delete", CategorySharing, "share", "S003", SeverityInfo, OCSFFileHostingActivity, 13)
	ShareAccept      = define("share.accept", CategorySharing, "share", "S004", SeverityInfo, OCSFFileHostingActivity, ocsfOther)
	ShareDecline     = define("share.decline", CategorySharing, "share", "S005", SeverityInfo, OCSFFileHostingActivity, ocsfOther)
	PublicLinkCreate = define("public_link.create", CategorySharing, "public_link", "S006", SeverityNotice, OCSFFileHostingActivity, 12)
	PublicLinkUpdate = define("public_link.update", CategorySharing, "public_link", "S007", SeverityNotice, OCSFFileHostingActivity, 12)
	PublicLinkDelete = define("public_link.delete", CategorySharing, "public_link", "S008", SeverityNotice, OCSFFileHostingActivity, 13)
	PublicLinkAccess = deniedActivity(define("public_link.access", CategorySharing, "public_link", "S009", SeverityInfo, OCSFWebResourceAccessActivity, 1), 2)
	OCMShareCreate   = define("ocm.share.create", CategorySharing, "ocm_share", "S010", SeverityNotice, OCSFFileHostingActivity, 12)
	OCMShareUpdate   = define("ocm.share.update", CategorySharing, "ocm_share", "S011", SeverityNotice, OCSFFileHostingActivity, 12)
	OCMShareDelete   = define("ocm.share.delete", CategorySharing, "ocm_share", "S012", SeverityNotice, OCSFFileHostingActivity, 13)
	OCMInviteCreate  = define("ocm.invite.create", CategorySharing, "ocm_invite", "S013", SeverityNotice, OCSFFileHostingActivity, ocsfOther)
	OCMInviteAccept  = define("ocm.invite.accept", CategorySharing, "ocm_invite", "S014", SeverityNotice, OCSFFileHostingActivity, ocsfOther)
)

// Spaces
var (
	SpaceCreate       = define("space.create", CategorySpaces, "space", "P001", SeverityNotice, OCSFEntityManagement, 1)
	SpaceUpdate       = define("space.update", CategorySpaces, "space", "P002", SeverityInfo, OCSFEntityManagement, 3)
	SpaceEnable       = define("space.enable", CategorySpaces, "space", "P003", SeverityNotice, OCSFEntityManagement, 8)
	SpaceDisable      = define("space.disable", CategorySpaces, "space", "P004", SeverityNotice, OCSFEntityManagement, 9)
	SpaceDelete       = define("space.delete", CategorySpaces, "space", "P005", SeverityNotice, OCSFEntityManagement, 4)
	SpaceMemberAdd    = define("space.member.add", CategorySpaces, "space", "P006", SeverityNotice, OCSFFileHostingActivity, 12)
	SpaceMemberUpdate = define("space.member.update", CategorySpaces, "space", "P007", SeverityNotice, OCSFFileHostingActivity, 12)
	SpaceMemberRemove = define("space.member.remove", CategorySpaces, "space", "P008", SeverityNotice, OCSFFileHostingActivity, 13)
	SpaceQuotaChange  = define("space.quota.change", CategorySpaces, "space", "P009", SeverityNotice, OCSFEntityManagement, 3)
)

// Identity
var (
	UserCreate        = define("user.create", CategoryIdentity, "user", "I001", SeverityNotice, OCSFUserManagement, 1)
	UserUpdate        = define("user.update", CategoryIdentity, "user", "I002", SeverityNotice, OCSFUserManagement, 2)
	UserEnable        = define("user.enable", CategoryIdentity, "user", "I003", SeverityNotice, OCSFUserManagement, 4)
	UserDisable       = define("user.disable", CategoryIdentity, "user", "I004", SeverityNotice, OCSFUserManagement, 5)
	UserSoftDelete    = define("user.soft_delete", CategoryIdentity, "user", "I005", SeverityNotice, OCSFUserManagement, 5)
	UserDelete        = define("user.delete", CategoryIdentity, "user", "I006", SeverityNotice, OCSFUserManagement, 3)
	GroupCreate       = define("group.create", CategoryIdentity, "group", "I007", SeverityNotice, OCSFGroupManagement, 6)
	GroupUpdate       = define("group.update", CategoryIdentity, "group", "I008", SeverityNotice, OCSFGroupManagement, 9)
	GroupDelete       = define("group.delete", CategoryIdentity, "group", "I009", SeverityNotice, OCSFGroupManagement, 5)
	GroupMemberAdd    = define("group.member.add", CategoryIdentity, "group", "I010", SeverityNotice, OCSFGroupManagement, 3)
	GroupMemberRemove = define("group.member.remove", CategoryIdentity, "group", "I011", SeverityNotice, OCSFGroupManagement, 4)
	AppRoleAssign     = define("app_role.assign", CategoryIdentity, "user", "I012", SeverityNotice, OCSFUserManagement, 16)
	AppRoleUnassign   = define("app_role.unassign", CategoryIdentity, "user", "I013", SeverityNotice, OCSFUserManagement, 17)
)

// Administration
var (
	AppRoleDefinitionChange = define("app_role.definition.change", CategoryAdministration, "app_role", "M001", SeverityNotice, OCSFRoleManagement, 2)
	SharingRoleEnable       = define("sharing_role.enable", CategoryAdministration, "sharing_role", "M002", SeverityNotice, OCSFRoleManagement, 2)
	SharingRoleDisable      = define("sharing_role.disable", CategoryAdministration, "sharing_role", "M003", SeverityNotice, OCSFRoleManagement, 2)
	SettingsChange          = define("settings.change", CategoryAdministration, "settings", "M004", SeverityInfo, OCSFAPIActivity, 3)
	ConfigChange            = define("config.change", CategoryAdministration, "config", "M005", SeverityNotice, OCSFAPIActivity, 3)
	CLICommand              = define("cli.command", CategoryAdministration, "cli", "M006", SeverityNotice, OCSFAPIActivity, ocsfOther)
	PersonalDataExport      = define("personal_data.export", CategoryAdministration, "user", "M007", SeverityNotice, OCSFAPIActivity, 2)
)

// Security
var (
	AccessDenied = define("access.denied", CategorySecurity, "resource", "X001", SeverityWarning, OCSFWebResourceAccessActivity, 2)
)

// Lookup returns the definition of the event with the given name.
func Lookup(name string) (*Definition, bool) {
	d, ok := registry[name]
	return d, ok
}

// Catalog returns all definitions, sorted by name.
func Catalog() []*Definition {
	defs := make([]*Definition, 0, len(registry))
	for _, d := range registry {
		defs = append(defs, d)
	}
	sort.Slice(defs, func(i, j int) bool { return defs[i].Name < defs[j].Name })
	return defs
}
