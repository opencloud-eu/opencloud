package unifiedrole

import (
	"cmp"
	"slices"
	"strings"

	libregraph "github.com/opencloud-eu/libre-graph-api-go"
	"google.golang.org/protobuf/proto"

	"github.com/opencloud-eu/reva/v2/pkg/conversions"

	"github.com/opencloud-eu/opencloud/pkg/l10n"
)

const (
	// UnifiedRoleViewerID Unified role viewer id.
	UnifiedRoleViewerID = "b1e2218d-eef8-4d4c-b82d-0f1a1b48f3b5"
	// UnifiedRoleViewerListGrantsID Unified role viewer id.
	UnifiedRoleViewerListGrantsID = "d5041006-ebb3-4b4a-b6a4-7c180ecfb17d"
	// UnifiedRoleSpaceViewerID Unified role space viewer id.
	UnifiedRoleSpaceViewerID = "a8d5fe5e-96e3-418d-825b-534dbdf22b99"
	// UnifiedRoleEditorID Unified role editor id.
	UnifiedRoleEditorID = "fb6c3e19-e378-47e5-b277-9732f9de6e21"
	// UnifiedRoleEditorListGrantsID Unified role editor id.
	UnifiedRoleEditorListGrantsID = "e8ea8b21-abd4-45d2-b893-8d1546378e9e"
	// UnifiedRoleSpaceEditorID Unified role space editor id.
	UnifiedRoleSpaceEditorID = "58c63c02-1d89-4572-916a-870abc5a1b7d"
	// UnifiedRoleSpaceEditorWithoutVersionsID Unified role space editor without list/restore versions id.
	UnifiedRoleSpaceEditorWithoutVersionsID = "3284f2d5-0070-4ad8-ac40-c247f7c1fb27"
	// UnifiedRoleFileEditorID Unified role file editor id.
	UnifiedRoleFileEditorID = "2d00ce52-1fc2-4dbc-8b95-a73b73395f5a"
	// UnifiedRoleFileEditorListGrantsID Unified role file editor id.
	UnifiedRoleFileEditorListGrantsID = "c1235aea-d106-42db-8458-7d5610fb0a67"
	// UnifiedRoleEditorLiteID Unified role editor-lite id.
	UnifiedRoleEditorLiteID = "1c996275-f1c9-4e71-abdf-a42f6495e960"
	// UnifiedRoleManagerID Unified role manager id.
	UnifiedRoleManagerID = "312c0871-5ef7-4b3a-85b6-0e4074c64049"
	// UnifiedRoleSecureViewerID Unified role secure viewer id.
	UnifiedRoleSecureViewerID = "aa97fe03-7980-45ac-9e50-b325749fd7e6"
	// UnifiedRoleDeniedID Unified role to deny all access.
	UnifiedRoleDeniedID = "63e64e19-8d43-42ec-a738-2b6af2610efa"

	// Wile the below conditions follow the SDDL syntax, they are not parsed anywhere. We use them as strings to
	// represent the constraints that a role definition applies to. For the actual syntax, see the SDDL documentation
	// at https://learn.microsoft.com/en-us/windows/win32/secauthz/security-descriptor-definition-language-for-conditional-aces-#conditional-expressions

	// Some roles apply to a specific type of resource, for example, a role that applies to a file or a folder.
	// @Resource is the placeholder for the resource that the role is applied to
	// .Root, .Folder and .File are facets of the driveItem resource that indicate the type of the resource if they are present.

	// UnifiedRoleConditionDrive defines constraint that matches a Driveroot/Spaceroot
	UnifiedRoleConditionDrive = "exists @Resource.Root"
	// UnifiedRoleConditionFolder defines constraints that matches a DriveItem representing a Folder
	UnifiedRoleConditionFolder = "exists @Resource.Folder"
	// UnifiedRoleConditionFile defines a constraint that matches a DriveItem representing a File
	UnifiedRoleConditionFile = "exists @Resource.File"

	// Some roles apply to a specific type of user, for example, a role that applies to a federated user.
	// @Subject is the placeholder for the subject that the role is applied to. For sharing roles this is the user that the resource is shared with.
	// .UserType is the type of the user: 'Member' for a member of the organization, 'Guest' for a guest user, 'Federated' for a federated user.

	// UnifiedRoleConditionFederatedUser defines a constraint that matches a federated user
	UnifiedRoleConditionFederatedUser = "@Subject.UserType==\"Federated\""

	// For federated sharing we need roles that combine the constraints for the resource and the user.
	// UnifiedRoleConditionFileFederatedUser defines a constraint that matches a File and a federated user
	UnifiedRoleConditionFileFederatedUser = UnifiedRoleConditionFile + " && " + UnifiedRoleConditionFederatedUser
	// UnifiedRoleConditionFolderFederatedUser defines a constraint that matches a Folder and a federated user
	UnifiedRoleConditionFolderFederatedUser = UnifiedRoleConditionFolder + " && " + UnifiedRoleConditionFederatedUser

	DriveItemPermissionsCreate = "libre.graph/driveItem/permissions/create"
	DriveItemChildrenCreate    = "libre.graph/driveItem/children/create"
	DriveItemStandardDelete    = "libre.graph/driveItem/standard/delete"
	DriveItemPathRead          = "libre.graph/driveItem/path/read"
	DriveItemQuotaRead         = "libre.graph/driveItem/quota/read"
	DriveItemContentRead       = "libre.graph/driveItem/content/read"
	DriveItemUploadCreate      = "libre.graph/driveItem/upload/create"
	DriveItemPermissionsRead   = "libre.graph/driveItem/permissions/read"
	DriveItemChildrenRead      = "libre.graph/driveItem/children/read"
	DriveItemVersionsRead      = "libre.graph/driveItem/versions/read"
	DriveItemDeletedRead       = "libre.graph/driveItem/deleted/read"
	DriveItemPathUpdate        = "libre.graph/driveItem/path/update"
	DriveItemPermissionsDelete = "libre.graph/driveItem/permissions/delete"
	DriveItemDeletedDelete     = "libre.graph/driveItem/deleted/delete"
	DriveItemVersionsUpdate    = "libre.graph/driveItem/versions/update"
	DriveItemDeletedUpdate     = "libre.graph/driveItem/deleted/update"
	DriveItemBasicRead         = "libre.graph/driveItem/basic/read"
	DriveItemPermissionsUpdate = "libre.graph/driveItem/permissions/update"
	DriveItemPermissionsDeny   = "libre.graph/driveItem/permissions/deny"
)

var (
	// UnifiedRole Viewer, Role Description (resolves directly)
	_viewerUnifiedRoleDescription = l10n.Template("View and download.")

	// UnifiedRole Viewer, Role DisplayName (resolves directly)
	_viewerUnifiedRoleDisplayName = l10n.Template("Can view")

	// UnifiedRole ViewerListGrants, Role Description (resolves directly)
	_viewerListGrantsUnifiedRoleDescription = l10n.Template("View, download and show all invited people.")

	// UnifiedRole Viewer, Role DisplayName (resolves directly)
	_viewerListGrantsUnifiedRoleDisplayName = l10n.Template("Can view")

	// UnifiedRole SpaceViewer, Role Description (resolves directly)
	_spaceViewerUnifiedRoleDescription = l10n.Template("View and download.")

	// UnifiedRole SpaseViewer, Role DisplayName (resolves directly)
	_spaceViewerUnifiedRoleDisplayName = l10n.Template("Can view")

	// UnifiedRole Editor, Role Description (resolves directly)
	_editorUnifiedRoleDescription = l10n.Template("View, download, upload, edit, add and delete.")

	// UnifiedRole Editor, Role DisplayName (resolves directly)
	_editorUnifiedRoleDisplayName = l10n.Template("Can edit")

	// UnifiedRoleListGrants Editor, Role Description (resolves directly)
	_editorListGrantsUnifiedRoleDescription = l10n.Template("View, download, upload, edit, add, delete and show all invited people.")

	// UnifiedRole EditorListGrants, Role DisplayName (resolves directly)
	_editorListGrantsUnifiedRoleDisplayName = l10n.Template("Can edit")

	// UnifiedRole SpaseEditor, Role Description (resolves directly)
	_spaceEditorUnifiedRoleDescription = l10n.Template("View, download, upload, edit, add, delete including the history.")

	// UnifiedRole SpaseEditor, Role DisplayName (resolves directly)
	_spaceEditorUnifiedRoleDisplayName = l10n.Template("Can edit")

	// UnifiedRole SpaseEditorWithoutVersions, Role Description (resolves directly)
	_spaceEditorWithoutVersionsUnifiedRoleDescription = l10n.Template("View, download, upload, edit, add and delete.")

	// UnifiedRole SpaseEditorWithoutVersions, Role DisplayName (resolves directly)
	_spaceEditorWithoutVersionsUnifiedRoleDisplayName = l10n.Template("Can edit without versions")

	// UnifiedRole FileEditor, Role Description (resolves directly)
	_fileEditorUnifiedRoleDescription = l10n.Template("View, download and edit.")

	// UnifiedRole FileEditor, Role DisplayName (resolves directly)
	_fileEditorUnifiedRoleDisplayName = l10n.Template("Can edit")

	// UnifiedRole FileEditorListGrants, Role Description (resolves directly)
	_fileEditorListGrantsUnifiedRoleDescription = l10n.Template("View, download, edit and show all invited people.")

	// UnifiedRole FileEditorListGrants, Role DisplayName (resolves directly)
	_fileEditorListGrantsUnifiedRoleDisplayName = l10n.Template("Can edit")

	// UnifiedRole EditorLite, Role Description (resolves directly)
	_editorLiteUnifiedRoleDescription = l10n.Template("View, download and upload.")

	// UnifiedRole EditorLite, Role DisplayName (resolves directly)
	_editorLiteUnifiedRoleDisplayName = l10n.Template("Can upload")

	// UnifiedRole Manager, Role Description (resolves directly)
	_managerUnifiedRoleDescription = l10n.Template("View, download, upload, edit, add, delete and manage members.")

	// UnifiedRole Manager, Role DisplayName (resolves directly)
	_managerUnifiedRoleDisplayName = l10n.Template("Can manage")

	// UnifiedRole SecureViewer, Role Description (resolves directly)
	_secureViewerUnifiedRoleDescription = l10n.Template("View only documents, images and PDFs. Watermarks will be applied.")

	// UnifiedRole SecureViewer, Role DisplayName (resolves directly)
	_secureViewerUnifiedRoleDisplayName = l10n.Template("Can view (secure)")

	// UnifiedRole FullDenial, Role Description (resolves directly)
	_deniedUnifiedRoleDescription = l10n.Template("Deny all access.")

	// UnifiedRole FullDenial, Role DisplayName (resolves directly)
	_deniedUnifiedRoleDisplayName = l10n.Template("Cannot access")

	// legacyNames contains the legacy role names.
	legacyNames = map[string]string{
		UnifiedRoleViewerID: conversions.RoleViewer,
		// one V1 api the "spaceviewer" role was call "viewer" and the "spaceeditor" was "editor",
		// we need to stay compatible with that
		UnifiedRoleSpaceViewerID:                "viewer",
		UnifiedRoleSpaceEditorID:                "editor",
		UnifiedRoleSpaceEditorWithoutVersionsID: conversions.RoleSpaceEditorWithoutVersions,
		UnifiedRoleEditorID:                     conversions.RoleEditor,
		UnifiedRoleFileEditorID:                 conversions.RoleFileEditor,
		UnifiedRoleEditorLiteID:                 conversions.RoleEditorLite,
		UnifiedRoleManagerID:                    conversions.RoleManager,
		UnifiedRoleSecureViewerID:               conversions.RoleSecureViewer,
	}

	// buildInRoles contains the built-in roles.
	buildInRoles = []*libregraph.UnifiedRoleDefinition{
		roleViewer,
		roleViewerListGrants,
		roleSpaceViewer,
		roleEditor,
		roleEditorListGrants,
		roleSpaceEditor,
		roleSpaceEditorWithoutVersions,
		roleFileEditor,
		roleFileEditorListGrants,
		roleEditorLite,
		roleManager,
		roleSecureViewer,
		roleDenied,
	}

	// roleViewer creates a viewer role.
	roleViewer = func() *libregraph.UnifiedRoleDefinition {
		r := conversions.NewViewerRole()
		return &libregraph.UnifiedRoleDefinition{
			Id:          proto.String(UnifiedRoleViewerID),
			Description: proto.String(_viewerUnifiedRoleDescription),
			DisplayName: proto.String(cs3RoleToDisplayName(r)),
			RolePermissions: []libregraph.UnifiedRolePermission{
				{
					AllowedResourceActions: CS3ResourcePermissionsToLibregraphActions(r.CS3ResourcePermissions()),
					Condition:              proto.String(UnifiedRoleConditionFile),
				},
				{
					AllowedResourceActions: CS3ResourcePermissionsToLibregraphActions(r.CS3ResourcePermissions()),
					Condition:              proto.String(UnifiedRoleConditionFolder),
				},
				{
					AllowedResourceActions: CS3ResourcePermissionsToLibregraphActions(r.CS3ResourcePermissions()),
					Condition:              proto.String(UnifiedRoleConditionFileFederatedUser),
				},
				{
					AllowedResourceActions: CS3ResourcePermissionsToLibregraphActions(r.CS3ResourcePermissions()),
					Condition:              proto.String(UnifiedRoleConditionFolderFederatedUser),
				},
			},
			LibreGraphWeight: proto.Int32(10),
		}
	}()

	// roleSecureViewer creates a secure viewer role
	roleSecureViewer = func() *libregraph.UnifiedRoleDefinition {
		r := conversions.NewSecureViewerRole()
		return &libregraph.UnifiedRoleDefinition{
			Id:          proto.String(UnifiedRoleSecureViewerID),
			Description: proto.String(_secureViewerUnifiedRoleDescription),
			DisplayName: proto.String(cs3RoleToDisplayName(r)),
			RolePermissions: []libregraph.UnifiedRolePermission{
				{
					AllowedResourceActions: CS3ResourcePermissionsToLibregraphActions(r.CS3ResourcePermissions()),
					Condition:              proto.String(UnifiedRoleConditionFile),
				},
				{
					AllowedResourceActions: CS3ResourcePermissionsToLibregraphActions(r.CS3ResourcePermissions()),
					Condition:              proto.String(UnifiedRoleConditionFolder),
				},
			},
			LibreGraphWeight: proto.Int32(20),
		}
	}()

	// roleViewerListGrants creates a viewer role.
	roleViewerListGrants = func() *libregraph.UnifiedRoleDefinition {
		r := conversions.NewViewerListGrantsRole()
		return &libregraph.UnifiedRoleDefinition{
			Id:          proto.String(UnifiedRoleViewerListGrantsID),
			Description: proto.String(_viewerListGrantsUnifiedRoleDescription),
			DisplayName: proto.String(cs3RoleToDisplayName(r)),
			RolePermissions: []libregraph.UnifiedRolePermission{
				{
					AllowedResourceActions: CS3ResourcePermissionsToLibregraphActions(r.CS3ResourcePermissions()),
					Condition:              proto.String(UnifiedRoleConditionFile),
				},
				{
					AllowedResourceActions: CS3ResourcePermissionsToLibregraphActions(r.CS3ResourcePermissions()),
					Condition:              proto.String(UnifiedRoleConditionFolder),
				},
				{
					AllowedResourceActions: CS3ResourcePermissionsToLibregraphActions(r.CS3ResourcePermissions()),
					Condition:              proto.String(UnifiedRoleConditionFileFederatedUser),
				},
				{
					AllowedResourceActions: CS3ResourcePermissionsToLibregraphActions(r.CS3ResourcePermissions()),
					Condition:              proto.String(UnifiedRoleConditionFolderFederatedUser),
				},
			},
			LibreGraphWeight: proto.Int32(30),
		}
	}()

	// roleSpaceViewer creates a spaceviewer role
	roleSpaceViewer = func() *libregraph.UnifiedRoleDefinition {
		r := conversions.NewSpaceViewerRole()
		return &libregraph.UnifiedRoleDefinition{
			Id:          proto.String(UnifiedRoleSpaceViewerID),
			Description: proto.String(_spaceViewerUnifiedRoleDescription),
			DisplayName: proto.String(cs3RoleToDisplayName(r)),
			RolePermissions: []libregraph.UnifiedRolePermission{
				{
					AllowedResourceActions: CS3ResourcePermissionsToLibregraphActions(r.CS3ResourcePermissions()),
					Condition:              proto.String(UnifiedRoleConditionDrive),
				},
			},
			LibreGraphWeight: proto.Int32(40),
		}
	}()

	// roleEditorLite creates an editor-lite role
	roleEditorLite = func() *libregraph.UnifiedRoleDefinition {
		r := conversions.NewEditorLiteRole()
		return &libregraph.UnifiedRoleDefinition{
			Id:          proto.String(UnifiedRoleEditorLiteID),
			Description: proto.String(_editorLiteUnifiedRoleDescription),
			DisplayName: proto.String(cs3RoleToDisplayName(r)),
			RolePermissions: []libregraph.UnifiedRolePermission{
				{
					AllowedResourceActions: CS3ResourcePermissionsToLibregraphActions(r.CS3ResourcePermissions()),
					Condition:              proto.String(UnifiedRoleConditionFolder),
				},
			},
			LibreGraphWeight: proto.Int32(50),
		}
	}()

	// roleEditor creates an editor role.
	roleEditor = func() *libregraph.UnifiedRoleDefinition {
		r := conversions.NewEditorRole()
		return &libregraph.UnifiedRoleDefinition{
			Id:          proto.String(UnifiedRoleEditorID),
			Description: proto.String(_editorUnifiedRoleDescription),
			DisplayName: proto.String(cs3RoleToDisplayName(r)),
			RolePermissions: []libregraph.UnifiedRolePermission{
				{
					AllowedResourceActions: CS3ResourcePermissionsToLibregraphActions(r.CS3ResourcePermissions()),
					Condition:              proto.String(UnifiedRoleConditionFolder),
				},
				{
					AllowedResourceActions: CS3ResourcePermissionsToLibregraphActions(r.CS3ResourcePermissions()),
					Condition:              proto.String(UnifiedRoleConditionFolderFederatedUser),
				},
			},
			LibreGraphWeight: proto.Int32(60),
		}
	}()

	// roleEditorListGrants creates an editor role.
	roleEditorListGrants = func() *libregraph.UnifiedRoleDefinition {
		r := conversions.NewEditorListGrantsRole()
		return &libregraph.UnifiedRoleDefinition{
			Id:          proto.String(UnifiedRoleEditorListGrantsID),
			Description: proto.String(_editorListGrantsUnifiedRoleDescription),
			DisplayName: proto.String(cs3RoleToDisplayName(r)),
			RolePermissions: []libregraph.UnifiedRolePermission{
				{
					AllowedResourceActions: CS3ResourcePermissionsToLibregraphActions(r.CS3ResourcePermissions()),
					Condition:              proto.String(UnifiedRoleConditionFolder),
				},
				{
					AllowedResourceActions: CS3ResourcePermissionsToLibregraphActions(r.CS3ResourcePermissions()),
					Condition:              proto.String(UnifiedRoleConditionFolderFederatedUser),
				},
			},
			LibreGraphWeight: proto.Int32(70),
		}
	}()

	// roleSpaceEditorWithoutVersions creates an editor without versions role
	roleSpaceEditorWithoutVersions = func() *libregraph.UnifiedRoleDefinition {
		r := conversions.NewSpaceEditorWithoutVersionsRole()
		return &libregraph.UnifiedRoleDefinition{
			Id:          proto.String(UnifiedRoleSpaceEditorWithoutVersionsID),
			Description: proto.String(_spaceEditorWithoutVersionsUnifiedRoleDescription),
			DisplayName: proto.String(cs3RoleToDisplayName(r)),
			RolePermissions: []libregraph.UnifiedRolePermission{
				{
					AllowedResourceActions: CS3ResourcePermissionsToLibregraphActions(r.CS3ResourcePermissions()),
					Condition:              proto.String(UnifiedRoleConditionDrive),
				},
			},
			LibreGraphWeight: proto.Int32(80),
		}
	}()

	// roleSpaceEditor creates an editor role
	roleSpaceEditor = func() *libregraph.UnifiedRoleDefinition {
		r := conversions.NewSpaceEditorRole()
		return &libregraph.UnifiedRoleDefinition{
			Id:          proto.String(UnifiedRoleSpaceEditorID),
			Description: proto.String(_spaceEditorUnifiedRoleDescription),
			DisplayName: proto.String(cs3RoleToDisplayName(r)),
			RolePermissions: []libregraph.UnifiedRolePermission{
				{
					AllowedResourceActions: CS3ResourcePermissionsToLibregraphActions(r.CS3ResourcePermissions()),
					Condition:              proto.String(UnifiedRoleConditionDrive),
				},
			},
			LibreGraphWeight: proto.Int32(90),
		}
	}()

	// roleFileEditor creates a file-editor role
	roleFileEditor = func() *libregraph.UnifiedRoleDefinition {
		r := conversions.NewFileEditorRole()
		return &libregraph.UnifiedRoleDefinition{
			Id:          proto.String(UnifiedRoleFileEditorID),
			Description: proto.String(_fileEditorUnifiedRoleDescription),
			DisplayName: proto.String(cs3RoleToDisplayName(r)),
			RolePermissions: []libregraph.UnifiedRolePermission{
				{
					AllowedResourceActions: CS3ResourcePermissionsToLibregraphActions(r.CS3ResourcePermissions()),
					Condition:              proto.String(UnifiedRoleConditionFile),
				},
				{
					AllowedResourceActions: CS3ResourcePermissionsToLibregraphActions(r.CS3ResourcePermissions()),
					Condition:              proto.String(UnifiedRoleConditionFileFederatedUser),
				},
			},
			LibreGraphWeight: proto.Int32(100),
		}
	}()

	// roleFileEditorListGrants creates a file-editor role
	roleFileEditorListGrants = func() *libregraph.UnifiedRoleDefinition {
		r := conversions.NewFileEditorListGrantsRole()
		return &libregraph.UnifiedRoleDefinition{
			Id:          proto.String(UnifiedRoleFileEditorListGrantsID),
			Description: proto.String(_fileEditorListGrantsUnifiedRoleDescription),
			DisplayName: proto.String(cs3RoleToDisplayName(r)),
			RolePermissions: []libregraph.UnifiedRolePermission{
				{
					AllowedResourceActions: CS3ResourcePermissionsToLibregraphActions(r.CS3ResourcePermissions()),
					Condition:              proto.String(UnifiedRoleConditionFile),
				},
				{
					AllowedResourceActions: CS3ResourcePermissionsToLibregraphActions(r.CS3ResourcePermissions()),
					Condition:              proto.String(UnifiedRoleConditionFileFederatedUser),
				},
			},
			LibreGraphWeight: proto.Int32(110),
		}
	}()

	// roleManager creates a manager role
	roleManager = func() *libregraph.UnifiedRoleDefinition {
		r := conversions.NewManagerRole()
		return &libregraph.UnifiedRoleDefinition{
			Id:          proto.String(UnifiedRoleManagerID),
			Description: proto.String(_managerUnifiedRoleDescription),
			DisplayName: proto.String(cs3RoleToDisplayName(r)),
			RolePermissions: []libregraph.UnifiedRolePermission{
				{
					AllowedResourceActions: CS3ResourcePermissionsToLibregraphActions(r.CS3ResourcePermissions()),
					Condition:              proto.String(UnifiedRoleConditionDrive),
				},
			},
			LibreGraphWeight: proto.Int32(120),
		}
	}()

	// roleDenied creates a secure viewer role
	roleDenied = func() *libregraph.UnifiedRoleDefinition {
		r := conversions.NewDeniedRole()
		return &libregraph.UnifiedRoleDefinition{
			Id:          proto.String(UnifiedRoleDeniedID),
			Description: proto.String(_deniedUnifiedRoleDescription),
			DisplayName: proto.String(cs3RoleToDisplayName(r)),
			RolePermissions: []libregraph.UnifiedRolePermission{
				{
					AllowedResourceActions: CS3ResourcePermissionsToLibregraphActions(r.CS3ResourcePermissions()),
					Condition:              proto.String(UnifiedRoleConditionFolder),
				},
			},
			LibreGraphWeight: proto.Int32(200),
		}
	}()
)

// GetRoles returns a role filter that matches the provided resources
func GetRoles(f RoleFilter) []*libregraph.UnifiedRoleDefinition {
	return filterRoles(buildInRoles, f)
}

// GetRole returns a role filter that matches the provided resources
func GetRole(f RoleFilter) (*libregraph.UnifiedRoleDefinition, error) {
	roles := filterRoles(buildInRoles, f)
	if len(roles) == 0 {
		return nil, ErrUnknownRole
	}

	return roles[0], nil
}

// GetRolesByPermissions returns a list of role definitions
// that match the provided actions and constraints
func GetRolesByPermissions(roleSet []*libregraph.UnifiedRoleDefinition, actions []string, constraints string, listFederatedRoles, descending bool) []*libregraph.UnifiedRoleDefinition {
	roles := make([]*libregraph.UnifiedRoleDefinition, 0, len(roleSet))

	for _, role := range roleSet {
		var match bool

		for _, permission := range role.GetRolePermissions() {
			// this is a dirty comparison because we are not really parsing the SDDL, but as long as we && the conditions we are good
			isFederatedRole := strings.Contains(permission.GetCondition(), UnifiedRoleConditionFederatedUser)
			switch {
			case !strings.Contains(permission.GetCondition(), constraints):
				continue
			case listFederatedRoles && !isFederatedRole:
				continue
			case !listFederatedRoles && isFederatedRole:
				continue
			}

			for i, action := range permission.GetAllowedResourceActions() {
				if !slices.Contains(actions, action) {
					break
				}
				if i == len(permission.GetAllowedResourceActions())-1 {
					match = true
				}
			}
			if role.GetId() == UnifiedRoleDeniedID && slices.Contains(actions, DriveItemPermissionsDeny) {
				match = true
			}
			if match {
				break
			}
		}

		if match {
			roles = append(roles, role)
		}

	}

	return weightRoles(roles, constraints, descending)
}

// GetLegacyRoleName returns the legacy role name for the provided role
func GetLegacyRoleName(role libregraph.UnifiedRoleDefinition) string {
	return legacyNames[role.GetId()]
}

// weightRoles sorts the provided role definitions by the number of LibreGraphWeight,
// descending - false - sorts the roles from least to most permissions
// descending - true - sorts the roles from most to least permissions
func weightRoles(roleSet []*libregraph.UnifiedRoleDefinition, constraints string, descending bool) []*libregraph.UnifiedRoleDefinition {
	slices.SortFunc(roleSet, func(a, b *libregraph.UnifiedRoleDefinition) int {
		if descending {
			return cmp.Compare(b.GetLibreGraphWeight(), a.GetLibreGraphWeight())
		}
		return cmp.Compare(a.GetLibreGraphWeight(), b.GetLibreGraphWeight())
	})

	return roleSet
}

// GetAllowedResourceActions returns the allowed resource actions for the provided role by condition
func GetAllowedResourceActions(role *libregraph.UnifiedRoleDefinition, condition string) []string {
	if role == nil {
		return []string{}
	}

	for _, p := range role.GetRolePermissions() {
		if p.GetCondition() == condition {
			return p.GetAllowedResourceActions()
		}
	}

	return []string{}
}
