package unifiedrole_test

import (
	"slices"
	"testing"

	. "github.com/onsi/gomega"
	libregraph "github.com/opencloud-eu/libre-graph-api-go"
	"google.golang.org/protobuf/proto"

	"github.com/opencloud-eu/opencloud/services/graph/pkg/unifiedrole"
)

func TestGetDefinition(t *testing.T) {
	tests := map[string]struct {
		ids                   []string
		unifiedRoleDefinition *libregraph.UnifiedRoleDefinition
		expectError           error
	}{
		"pass single": {
			ids:                   []string{unifiedrole.UnifiedRoleViewerID},
			unifiedRoleDefinition: unifiedrole.RoleViewer,
		},
		"pass many": {
			ids:                   []string{unifiedrole.UnifiedRoleViewerID, unifiedrole.UnifiedRoleEditorID},
			unifiedRoleDefinition: unifiedrole.RoleViewer,
		},
		"fail unknown": {
			ids:         []string{"unknown"},
			expectError: unifiedrole.ErrUnknownRole,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			g := NewWithT(t)
			definition, err := unifiedrole.GetRole(unifiedrole.RoleFilterIDs(tc.ids...))

			if tc.expectError != nil {
				g.Expect(err).To(MatchError(tc.expectError))
			} else {
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(definition).To(Equal(tc.unifiedRoleDefinition))
			}
		})
	}
}

func TestWeightDefinitions(t *testing.T) {
	tests := map[string]struct {
		unifiedRoleDefinition []*libregraph.UnifiedRoleDefinition
		constraint            string
		descending            bool
		expectedDefinitions   []*libregraph.UnifiedRoleDefinition
	}{
		"ascending": {
			[]*libregraph.UnifiedRoleDefinition{
				unifiedrole.RoleViewer,
				unifiedrole.RoleFileEditor,
			},
			unifiedrole.UnifiedRoleConditionFile,
			false,
			[]*libregraph.UnifiedRoleDefinition{
				unifiedrole.RoleViewer,
				unifiedrole.RoleFileEditor,
			},
		},
		"descending": {
			[]*libregraph.UnifiedRoleDefinition{
				unifiedrole.RoleViewer,
				unifiedrole.RoleFileEditor,
			},
			unifiedrole.UnifiedRoleConditionFile,
			true,
			[]*libregraph.UnifiedRoleDefinition{
				unifiedrole.RoleFileEditor,
				unifiedrole.RoleViewer,
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			g := NewWithT(t)
			for i, generatedDefinition := range unifiedrole.WeightDefinitions(tc.unifiedRoleDefinition, tc.constraint, tc.descending) {
				g.Expect(generatedDefinition.Id).To(Equal(tc.expectedDefinitions[i].Id))
			}
		})
	}
}

func TestGetRolesByPermissions(t *testing.T) {
	tests := map[string]struct {
		givenActions          []string
		constraints           string
		listFederatedRoles    bool
		unifiedRoleDefinition []*libregraph.UnifiedRoleDefinition
	}{
		"RoleViewer | folder": {
			givenActions: getRoleActions(unifiedrole.RoleViewer),
			constraints:  unifiedrole.UnifiedRoleConditionFolder,
			unifiedRoleDefinition: []*libregraph.UnifiedRoleDefinition{
				unifiedrole.RoleViewer,
				unifiedrole.RoleSecureViewer,
			},
		},
		"RoleViewer | file": {
			givenActions: getRoleActions(unifiedrole.RoleViewer),
			constraints:  unifiedrole.UnifiedRoleConditionFile,
			unifiedRoleDefinition: []*libregraph.UnifiedRoleDefinition{
				unifiedrole.RoleViewer,
				unifiedrole.RoleSecureViewer,
			},
		},
		"RoleViewer | file | federated": {
			givenActions:       getRoleActions(unifiedrole.RoleViewer),
			constraints:        unifiedrole.UnifiedRoleConditionFile,
			listFederatedRoles: true,
			unifiedRoleDefinition: []*libregraph.UnifiedRoleDefinition{
				unifiedrole.RoleViewer,
			},
		},
		"RoleFileEditor | file": {
			givenActions: getRoleActions(unifiedrole.RoleFileEditor),
			constraints:  unifiedrole.UnifiedRoleConditionFile,
			unifiedRoleDefinition: []*libregraph.UnifiedRoleDefinition{
				unifiedrole.RoleViewer,
				unifiedrole.RoleSecureViewer,
				unifiedrole.RoleFileEditor,
			},
		},
		"RoleEditor | folder": {
			givenActions: getRoleActions(unifiedrole.RoleEditor),
			constraints:  unifiedrole.UnifiedRoleConditionFolder,
			unifiedRoleDefinition: []*libregraph.UnifiedRoleDefinition{
				unifiedrole.RoleViewer,
				unifiedrole.RoleSecureViewer,
				unifiedrole.RoleEditorLite,
				unifiedrole.RoleEditor,
			},
		},
		"RoleEditor | folder | federated": {
			givenActions:       getRoleActions(unifiedrole.RoleEditor),
			constraints:        unifiedrole.UnifiedRoleConditionFolder,
			listFederatedRoles: true,
			unifiedRoleDefinition: []*libregraph.UnifiedRoleDefinition{
				unifiedrole.RoleViewer,
				unifiedrole.RoleEditor,
			},
		},
		"RoleEditor | file | federated": {
			givenActions:       getRoleActions(unifiedrole.RoleEditor),
			constraints:        unifiedrole.UnifiedRoleConditionFile,
			listFederatedRoles: true,
			unifiedRoleDefinition: []*libregraph.UnifiedRoleDefinition{
				unifiedrole.RoleViewer,
				unifiedrole.RoleFileEditor,
			},
		},
		"BuildInRoles | file": {
			givenActions: getRoleActions(unifiedrole.BuildInRoles...),
			constraints:  unifiedrole.UnifiedRoleConditionFile,
			unifiedRoleDefinition: []*libregraph.UnifiedRoleDefinition{
				unifiedrole.RoleViewer,
				unifiedrole.RoleSecureViewer,
				unifiedrole.RoleViewerListGrants,
				unifiedrole.RoleFileEditor,
				unifiedrole.RoleFileEditorListGrants,
			},
		},
		"BuildInRoles | folder": {
			givenActions: getRoleActions(unifiedrole.BuildInRoles...),
			constraints:  unifiedrole.UnifiedRoleConditionFolder,
			unifiedRoleDefinition: []*libregraph.UnifiedRoleDefinition{
				unifiedrole.RoleViewer,
				unifiedrole.RoleSecureViewer,
				unifiedrole.RoleViewerListGrants,
				unifiedrole.RoleEditorLite,
				unifiedrole.RoleEditor,
				unifiedrole.RoleEditorListGrants,
				unifiedrole.RoleDenied,
			},
		},
		"BuildInRoles | drive": {
			givenActions: getRoleActions(unifiedrole.BuildInRoles...),
			constraints:  unifiedrole.UnifiedRoleConditionDrive,
			unifiedRoleDefinition: []*libregraph.UnifiedRoleDefinition{
				unifiedrole.RoleSpaceViewer,
				unifiedrole.RoleSpaceEditorWithoutVersions,
				unifiedrole.RoleSpaceEditor,
				unifiedrole.RoleManager,
			},
		},
		"custom | file": {
			givenActions:          []string{unifiedrole.DriveItemQuotaRead},
			constraints:           unifiedrole.UnifiedRoleConditionFile,
			unifiedRoleDefinition: []*libregraph.UnifiedRoleDefinition{},
		},
		"RoleEditorLite and custom | folder": {
			givenActions: append(getRoleActions(unifiedrole.RoleEditorLite), unifiedrole.DriveItemQuotaRead),
			constraints:  unifiedrole.UnifiedRoleConditionFolder,
			unifiedRoleDefinition: []*libregraph.UnifiedRoleDefinition{
				unifiedrole.RoleSecureViewer,
				unifiedrole.RoleEditorLite,
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			g := NewWithT(t)
			generatedDefinitions := unifiedrole.GetRolesByPermissions(unifiedrole.BuildInRoles, tc.givenActions, tc.constraints, tc.listFederatedRoles, false)

			g.Expect(len(generatedDefinitions)).To(Equal(len(tc.unifiedRoleDefinition)))

			for i, generatedDefinition := range generatedDefinitions {
				g.Expect(generatedDefinition.Id).To(Equal(tc.unifiedRoleDefinition[i].Id))
				g.Expect(generatedDefinition.LibreGraphWeight).To(Equal(tc.unifiedRoleDefinition[i].LibreGraphWeight))
			}

			generatedActions := getRoleActions(generatedDefinitions...)

			g.Expect(len(tc.givenActions) >= len(generatedActions)).To(BeTrue())
			for _, generatedAction := range generatedActions {
				g.Expect(slices.Contains(tc.givenActions, generatedAction)).To(BeTrue())
			}
		})
	}
}

func TestGetAllowedResourceActions(t *testing.T) {
	tests := map[string]struct {
		unifiedRoleDefinition *libregraph.UnifiedRoleDefinition
		condition             string
		expectedActions       []string
	}{
		"no role": {
			expectedActions: []string{},
		},
		"no match": {
			unifiedRoleDefinition: &libregraph.UnifiedRoleDefinition{
				RolePermissions: []libregraph.UnifiedRolePermission{
					{Condition: proto.String(unifiedrole.UnifiedRoleConditionDrive), AllowedResourceActions: []string{unifiedrole.DriveItemPermissionsCreate}},
					{Condition: proto.String(unifiedrole.UnifiedRoleConditionFolder), AllowedResourceActions: []string{unifiedrole.DriveItemDeletedRead}},
				},
			},
			condition:       unifiedrole.UnifiedRoleConditionFile,
			expectedActions: []string{},
		},
		"match": {
			unifiedRoleDefinition: &libregraph.UnifiedRoleDefinition{
				RolePermissions: []libregraph.UnifiedRolePermission{
					{Condition: proto.String(unifiedrole.UnifiedRoleConditionDrive), AllowedResourceActions: []string{unifiedrole.DriveItemPermissionsCreate}},
					{Condition: proto.String(unifiedrole.UnifiedRoleConditionFolder), AllowedResourceActions: []string{unifiedrole.DriveItemDeletedRead}},
				},
			},
			condition:       unifiedrole.UnifiedRoleConditionFolder,
			expectedActions: []string{unifiedrole.DriveItemDeletedRead},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			NewWithT(t).
				Expect(unifiedrole.GetAllowedResourceActions(tc.unifiedRoleDefinition, tc.condition)).
				To(ContainElements(tc.expectedActions))
		})
	}
}
