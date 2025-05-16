package unifiedrole

import (
	"strings"

	provider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	"github.com/opencloud-eu/reva/v2/pkg/conversions"
	libregraph "github.com/opencloud-eu/libre-graph-api-go"
)

// PermissionsToCS3ResourcePermissions converts the provided libregraph UnifiedRolePermissions to a cs3 ResourcePermissions
func PermissionsToCS3ResourcePermissions(unifiedRolePermissions []*libregraph.UnifiedRolePermission) *provider.ResourcePermissions {
	p := &provider.ResourcePermissions{}

	for _, permission := range unifiedRolePermissions {
		for _, allowedResourceAction := range permission.AllowedResourceActions {
			switch allowedResourceAction {
			case DriveItemPermissionsCreate:
				p.AddGrant = true
			case DriveItemChildrenCreate:
				p.CreateContainer = true
			case DriveItemStandardDelete:
				p.Delete = true
			case DriveItemPathRead:
				p.GetPath = true
			case DriveItemQuotaRead:
				p.GetQuota = true
			case DriveItemContentRead:
				p.InitiateFileDownload = true
			case DriveItemUploadCreate:
				p.InitiateFileUpload = true
			case DriveItemPermissionsRead:
				p.ListGrants = true
			case DriveItemChildrenRead:
				p.ListContainer = true
			case DriveItemVersionsRead:
				p.ListFileVersions = true
			case DriveItemDeletedRead:
				p.ListRecycle = true
			case DriveItemPathUpdate:
				p.Move = true
			case DriveItemPermissionsDelete:
				p.RemoveGrant = true
			case DriveItemDeletedDelete:
				p.PurgeRecycle = true
			case DriveItemVersionsUpdate:
				p.RestoreFileVersion = true
			case DriveItemDeletedUpdate:
				p.RestoreRecycleItem = true
			case DriveItemBasicRead:
				p.Stat = true
			case DriveItemPermissionsUpdate:
				p.UpdateGrant = true
			case DriveItemPermissionsDeny:
				p.DenyGrant = true
			}
		}
	}

	return p
}

// CS3ResourcePermissionsToLibregraphActions converts the provided cs3 ResourcePermissions to a list of
// libregraph actions
func CS3ResourcePermissionsToLibregraphActions(p *provider.ResourcePermissions) []string {
	var actions []string

	if p.GetAddGrant() {
		actions = append(actions, DriveItemPermissionsCreate)
	}

	if p.GetCreateContainer() {
		actions = append(actions, DriveItemChildrenCreate)
	}

	if p.GetDelete() {
		actions = append(actions, DriveItemStandardDelete)
	}

	if p.GetGetPath() {
		actions = append(actions, DriveItemPathRead)
	}

	if p.GetGetQuota() {
		actions = append(actions, DriveItemQuotaRead)
	}

	if p.GetInitiateFileDownload() {
		actions = append(actions, DriveItemContentRead)
	}

	if p.GetInitiateFileUpload() {
		actions = append(actions, DriveItemUploadCreate)
	}

	if p.GetListGrants() {
		actions = append(actions, DriveItemPermissionsRead)
	}

	if p.GetListContainer() {
		actions = append(actions, DriveItemChildrenRead)
	}

	if p.GetListFileVersions() {
		actions = append(actions, DriveItemVersionsRead)
	}

	if p.GetListRecycle() {
		actions = append(actions, DriveItemDeletedRead)
	}

	if p.GetMove() {
		actions = append(actions, DriveItemPathUpdate)
	}

	if p.GetRemoveGrant() {
		actions = append(actions, DriveItemPermissionsDelete)
	}

	if p.GetPurgeRecycle() {
		actions = append(actions, DriveItemDeletedDelete)
	}

	if p.GetRestoreFileVersion() {
		actions = append(actions, DriveItemVersionsUpdate)
	}

	if p.GetRestoreRecycleItem() {
		actions = append(actions, DriveItemDeletedUpdate)
	}

	if p.GetStat() {
		actions = append(actions, DriveItemBasicRead)
	}

	if p.GetUpdateGrant() {
		actions = append(actions, DriveItemPermissionsUpdate)
	}

	if p.GetDenyGrant() {
		actions = append(actions, DriveItemPermissionsDeny)
	}

	return actions
}

// CS3ResourcePermissionsToRole converts the provided cs3 ResourcePermissions to a libregraph UnifiedRoleDefinition
func CS3ResourcePermissionsToRole(roleSet []*libregraph.UnifiedRoleDefinition, p *provider.ResourcePermissions, constraints string, listFederatedRoles bool) *libregraph.UnifiedRoleDefinition {
	actionSet := map[string]struct{}{}
	for _, action := range CS3ResourcePermissionsToLibregraphActions(p) {
		actionSet[action] = struct{}{}
	}

	var res *libregraph.UnifiedRoleDefinition
	for _, uRole := range roleSet {
		definitionMatch := false

		for _, permission := range uRole.GetRolePermissions() {
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

			// if the actions converted from the ResourcePermissions equal the action the defined for the role, we have match
			if resourceActionsEqual(actionSet, permission.GetAllowedResourceActions()) {
				definitionMatch = true
				break
			}
		}
		if definitionMatch {
			res = uRole
			break
		}
	}
	return res
}

// resourceActionsEqual checks if the provided actions are equal to the actions defined for a resource
func resourceActionsEqual(targetActionSet map[string]struct{}, actions []string) bool {
	if len(targetActionSet) != len(actions) {
		return false
	}

	for _, action := range actions {
		if _, ok := targetActionSet[action]; !ok {
			return false
		}
	}
	return true
}

// cs3RoleToDisplayName converts a CS3 role to a human-readable display name
func cs3RoleToDisplayName(role *conversions.Role) string {
	if role == nil {
		return ""
	}

	switch role.Name {
	case conversions.RoleViewer:
		return _viewerUnifiedRoleDisplayName
	case conversions.RoleViewerListGrants:
		return _viewerListGrantsUnifiedRoleDisplayName
	case conversions.RoleSpaceViewer:
		return _spaceViewerUnifiedRoleDisplayName
	case conversions.RoleEditor:
		return _editorUnifiedRoleDisplayName
	case conversions.RoleEditorListGrants:
		return _editorListGrantsUnifiedRoleDisplayName
	case conversions.RoleSpaceEditor:
		return _spaceEditorUnifiedRoleDisplayName
	case conversions.RoleSpaceEditorWithoutVersions:
		return _spaceEditorWithoutVersionsUnifiedRoleDisplayName
	case conversions.RoleFileEditor:
		return _fileEditorUnifiedRoleDisplayName
	case conversions.RoleFileEditorListGrants:
		return _fileEditorListGrantsUnifiedRoleDisplayName
	case conversions.RoleEditorLite:
		return _editorLiteUnifiedRoleDisplayName
	case conversions.RoleManager:
		return _managerUnifiedRoleDisplayName
	case conversions.RoleSecureViewer:
		return _secureViewerUnifiedRoleDisplayName
	case conversions.RoleDenied:
		return _deniedUnifiedRoleDisplayName
	default:
		return ""
	}
}
