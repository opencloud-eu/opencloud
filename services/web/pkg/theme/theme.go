package theme

import (
	"path"

	"github.com/opencloud-eu/opencloud/pkg/capabilities"
	"github.com/opencloud-eu/opencloud/services/graph/pkg/unifiedrole"
)

var (
	_brandingRoot  = "_branding"
	_themeFileName = "theme.json"
)

// themeDefaults contains the default values for the theme.
// all rendered themes get the default values from here.
var themeDefaults = KV{
	"common": KV{
		"shareRoles": KV{
			unifiedrole.UnifiedRoleViewerID: KV{
				"label":    unifiedrole.GetUnifiedRoleLabel(unifiedrole.UnifiedRoleViewerID),
				"iconName": "eye",
			},
			unifiedrole.UnifiedRoleViewerWithVersionsID: KV{
				"label":    unifiedrole.GetUnifiedRoleLabel(unifiedrole.UnifiedRoleViewerWithVersionsID),
				"iconName": "eye",
			},
			unifiedrole.UnifiedRoleViewerListGrantsID: KV{
				"label":    unifiedrole.GetUnifiedRoleLabel(unifiedrole.UnifiedRoleViewerListGrantsID),
				"iconName": "eye",
			},
			unifiedrole.UnifiedRoleSpaceViewerID: KV{
				"label":    unifiedrole.GetUnifiedRoleLabel(unifiedrole.UnifiedRoleSpaceViewerID),
				"iconName": "eye",
			},
			unifiedrole.UnifiedRoleSpaceViewerWithVersionsID: KV{
				"label":    unifiedrole.GetUnifiedRoleLabel(unifiedrole.UnifiedRoleSpaceViewerWithVersionsID),
				"iconName": "eye",
			},
			unifiedrole.UnifiedRoleFileEditorID: KV{
				"label":    unifiedrole.GetUnifiedRoleLabel(unifiedrole.UnifiedRoleFileEditorID),
				"iconName": "pencil",
			},
			unifiedrole.UnifiedRoleFileEditorWithVersionsID: KV{
				"label":    unifiedrole.GetUnifiedRoleLabel(unifiedrole.UnifiedRoleFileEditorWithVersionsID),
				"iconName": "pencil",
			},
			unifiedrole.UnifiedRoleFileEditorListGrantsID: KV{
				"label":    unifiedrole.GetUnifiedRoleLabel(unifiedrole.UnifiedRoleFileEditorListGrantsID),
				"iconName": "pencil",
			},
			unifiedrole.UnifiedRoleEditorID: KV{
				"label":    unifiedrole.GetUnifiedRoleLabel(unifiedrole.UnifiedRoleEditorID),
				"iconName": "pencil",
			},
			unifiedrole.UnifiedRoleEditorWithVersionsID: KV{
				"label":    unifiedrole.GetUnifiedRoleLabel(unifiedrole.UnifiedRoleEditorWithVersionsID),
				"iconName": "pencil",
			},
			unifiedrole.UnifiedRoleEditorListGrantsID: KV{
				"label":    unifiedrole.GetUnifiedRoleLabel(unifiedrole.UnifiedRoleEditorListGrantsID),
				"iconName": "pencil",
			},
			unifiedrole.UnifiedRoleSpaceEditorID: KV{
				"label":    unifiedrole.GetUnifiedRoleLabel(unifiedrole.UnifiedRoleSpaceEditorID),
				"iconName": "pencil",
			},
			unifiedrole.UnifiedRoleSpaceEditorWithoutVersionsID: KV{
				"label":    unifiedrole.GetUnifiedRoleLabel(unifiedrole.UnifiedRoleSpaceEditorWithoutVersionsID),
				"iconName": "pencil",
			},
			unifiedrole.UnifiedRoleManagerID: KV{
				"label":    unifiedrole.GetUnifiedRoleLabel(unifiedrole.UnifiedRoleManagerID),
				"iconName": "user-star",
			},
			unifiedrole.UnifiedRoleEditorLiteID: KV{
				"label":    unifiedrole.GetUnifiedRoleLabel(unifiedrole.UnifiedRoleEditorLiteID),
				"iconName": "upload",
			},
			unifiedrole.UnifiedRoleSecureViewerID: KV{
				"label":    unifiedrole.GetUnifiedRoleLabel(unifiedrole.UnifiedRoleSecureViewerID),
				"iconName": "shield",
			},
			unifiedrole.UnifiedRoleDeniedID: KV{
				"label":    unifiedrole.GetUnifiedRoleLabel(unifiedrole.UnifiedRoleDeniedID),
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
