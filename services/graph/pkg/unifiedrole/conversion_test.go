package unifiedrole_test

import (
	"testing"

	provider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/types"
	cs3Conversions "github.com/opencloud-eu/reva/v2/pkg/conversions"
	libregraph "github.com/opencloud-eu/libre-graph-api-go"

	"github.com/opencloud-eu/opencloud/pkg/conversions"
	"github.com/opencloud-eu/opencloud/services/graph/pkg/unifiedrole"
)

func TestPermissionsToCS3ResourcePermissions(t *testing.T) {
	tests := map[string]struct {
		cs3Role               *cs3Conversions.Role
		unifiedRoleDefinition *libregraph.UnifiedRoleDefinition
		match                 bool
	}{
		cs3Conversions.RoleViewer:               {cs3Conversions.NewViewerRole(), unifiedrole.RoleViewer, true},
		cs3Conversions.RoleViewerListGrants:     {cs3Conversions.NewViewerListGrantsRole(), unifiedrole.RoleViewerListGrants, true},
		cs3Conversions.RoleEditor:               {cs3Conversions.NewEditorRole(), unifiedrole.RoleEditor, true},
		cs3Conversions.RoleEditorListGrants:     {cs3Conversions.NewEditorListGrantsRole(), unifiedrole.RoleEditorListGrants, true},
		cs3Conversions.RoleFileEditor:           {cs3Conversions.NewFileEditorRole(), unifiedrole.RoleFileEditor, true},
		cs3Conversions.RoleFileEditorListGrants: {cs3Conversions.NewFileEditorListGrantsRole(), unifiedrole.RoleFileEditorListGrants, true},
		cs3Conversions.RoleManager:              {cs3Conversions.NewManagerRole(), unifiedrole.RoleManager, true},
		cs3Conversions.RoleSecureViewer:         {cs3Conversions.NewSecureViewerRole(), unifiedrole.RoleSecureViewer, true},
		cs3Conversions.RoleDenied:               {cs3Conversions.NewDeniedRole(), unifiedrole.RoleDenied, true},
		"no match":                              {cs3Conversions.NewFileEditorRole(), unifiedrole.RoleManager, false},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			g := NewWithT(t)
			permsFromCS3 := tc.cs3Role.CS3ResourcePermissions()
			permsFromUnifiedRole := unifiedrole.PermissionsToCS3ResourcePermissions(
				conversions.ToPointerSlice(tc.unifiedRoleDefinition.RolePermissions),
			)

			var matcher types.GomegaMatcher

			if tc.match {
				matcher = Equal(permsFromUnifiedRole)
			} else {
				matcher = Not(Equal(permsFromUnifiedRole))
			}

			g.Expect(permsFromCS3).To(matcher)
		})
	}
}

func TestCS3ResourcePermissionsToRole(t *testing.T) {
	tests := map[string]struct {
		cs3ResourcePermissions *provider.ResourcePermissions
		unifiedRoleDefinition  *libregraph.UnifiedRoleDefinition
		constraints            string
	}{
		cs3Conversions.RoleViewer + "1":       {cs3Conversions.NewViewerRole().CS3ResourcePermissions(), unifiedrole.RoleViewer, unifiedrole.UnifiedRoleConditionFile},
		cs3Conversions.RoleViewer + "2":       {cs3Conversions.NewViewerRole().CS3ResourcePermissions(), unifiedrole.RoleViewer, unifiedrole.UnifiedRoleConditionFolder},
		cs3Conversions.RoleEditor:             {cs3Conversions.NewEditorRole().CS3ResourcePermissions(), unifiedrole.RoleEditor, unifiedrole.UnifiedRoleConditionFolder},
		cs3Conversions.RoleFileEditor:         {cs3Conversions.NewFileEditorRole().CS3ResourcePermissions(), unifiedrole.RoleFileEditor, unifiedrole.UnifiedRoleConditionFile},
		cs3Conversions.RoleManager:            {cs3Conversions.NewManagerRole().CS3ResourcePermissions(), unifiedrole.RoleManager, unifiedrole.UnifiedRoleConditionDrive},
		cs3Conversions.RoleSpaceViewer:        {cs3Conversions.NewSpaceViewerRole().CS3ResourcePermissions(), unifiedrole.RoleSpaceViewer, unifiedrole.UnifiedRoleConditionDrive},
		cs3Conversions.RoleSpaceEditor:        {cs3Conversions.NewSpaceEditorRole().CS3ResourcePermissions(), unifiedrole.RoleSpaceEditor, unifiedrole.UnifiedRoleConditionDrive},
		cs3Conversions.RoleSecureViewer + "1": {cs3Conversions.NewSecureViewerRole().CS3ResourcePermissions(), unifiedrole.RoleSecureViewer, unifiedrole.UnifiedRoleConditionFile},
		cs3Conversions.RoleSecureViewer + "2": {cs3Conversions.NewSecureViewerRole().CS3ResourcePermissions(), unifiedrole.RoleSecureViewer, unifiedrole.UnifiedRoleConditionFolder},
		cs3Conversions.RoleDenied:             {cs3Conversions.NewDeniedRole().CS3ResourcePermissions(), unifiedrole.RoleDenied, unifiedrole.UnifiedRoleConditionFolder},
		"custom 1":                            {&provider.ResourcePermissions{GetPath: true}, nil, unifiedrole.UnifiedRoleConditionFolder},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			NewWithT(t).Expect(
				unifiedrole.CS3ResourcePermissionsToRole(unifiedrole.BuildInRoles, tc.cs3ResourcePermissions, tc.constraints, false),
			).To(Equal(tc.unifiedRoleDefinition))
		})
	}
}
