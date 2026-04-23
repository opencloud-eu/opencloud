package theme

import (
	"errors"
	"path"

	"github.com/opencloud-eu/opencloud/pkg/capabilities"
	"github.com/opencloud-eu/opencloud/services/graph/pkg/unifiedrole"
)

var (
	_brandingRoot          = "_branding"
	_themeFileName         = "theme.json"
	ErrBuildingThemeFailed = errors.New("building theme failed")
)

// themeDefaults contains the default values for the theme.
// all rendered themes get the default values from here.
var themeDefaults = KV{
	"common": KV{
		"shareRoles": KV{
			unifiedrole.UnifiedRoleViewerID: KV{
				"name":     "UnifiedRoleViewer",
				"iconName": "eye",
			},
			unifiedrole.UnifiedRoleViewerListGrantsID: KV{
				"name":     "UnifiedRoleViewerListGrants",
				"iconName": "eye",
			},
			unifiedrole.UnifiedRoleSpaceViewerID: KV{
				"label":    "UnifiedRoleSpaceViewer",
				"iconName": "eye",
			},
			unifiedrole.UnifiedRoleFileEditorID: KV{
				"label":    "UnifiedRoleFileEditor",
				"iconName": "pencil",
			},
			unifiedrole.UnifiedRoleFileEditorListGrantsID: KV{
				"label":    "UnifiedRoleFileEditorListGrants",
				"iconName": "pencil",
			},
			unifiedrole.UnifiedRoleEditorID: KV{
				"label":    "UnifiedRoleEditor",
				"iconName": "pencil",
			},
			unifiedrole.UnifiedRoleEditorListGrantsID: KV{
				"label":    "UnifiedRoleEditorListGrants",
				"iconName": "pencil",
			},
			unifiedrole.UnifiedRoleSpaceEditorID: KV{
				"label":    "UnifiedRoleSpaceEditor",
				"iconName": "pencil",
			},
			unifiedrole.UnifiedRoleSpaceEditorWithoutVersionsID: KV{
				"label":    "UnifiedRoleSpaceEditorWithoutVersions",
				"iconName": "pencil",
			},
			unifiedrole.UnifiedRoleManagerID: KV{
				"label":    "UnifiedRoleManager",
				"iconName": "user-star",
			},
			unifiedrole.UnifiedRoleEditorLiteID: KV{
				"label":    "UnifiedRoleEditorLite",
				"iconName": "upload",
			},
			unifiedrole.UnifiedRoleSecureViewerID: KV{
				"label":    "UnifiedRoleSecureView",
				"iconName": "shield",
			},
			unifiedrole.UnifiedRoleDeniedID: KV{
				"label":    "UnifiedRoleFullDenial",
				"iconName": "stop-circle",
			},
		},
	},
}

// isFiletypePermitted checks if the given file extension is allowed.
func isFiletypePermitted(filename string, givenMime string) bool {
	// Check if we allow that extension and if the mediatype matches the extension
	extensionMime, ok := capabilities.Default().Theme.Logo.PermittedFileTypes[path.Ext(filename)]
	return ok && extensionMime == givenMime
}
