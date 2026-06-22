package unifiedrole_test

import (
	"slices"
	"sync"
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

func TestLocalizeRole_English(t *testing.T) {
	g := NewWithT(t)
	original := unifiedrole.RoleViewer

	result := unifiedrole.LocalizeRole(original, "en")

	// Strings are unchanged for English
	g.Expect(result.GetDisplayName()).To(Equal(original.GetDisplayName()))
	g.Expect(result.GetDescription()).To(Equal(original.GetDescription()))

	// Result is an independent copy — mutating it must not touch the global
	translated := "mutated"
	result.DisplayName = &translated
	g.Expect(original.GetDisplayName()).NotTo(Equal("mutated"))
}

func TestLocalizeRole_German(t *testing.T) {
	g := NewWithT(t)
	original := unifiedrole.RoleViewer

	result := unifiedrole.LocalizeRole(original, "de")

	g.Expect(result.GetDisplayName()).To(Equal("Kann anzeigen"))
	g.Expect(result.GetDescription()).To(Equal("Ansehen und herunterladen."))

	// Global singleton must be untouched
	g.Expect(original.GetDisplayName()).NotTo(Equal("Kann anzeigen"))
	g.Expect(original.GetDescription()).NotTo(Equal("Ansehen und herunterladen."))
}

func TestLocalizeRole_EmptyLocale(t *testing.T) {
	g := NewWithT(t)
	original := unifiedrole.RoleViewer

	result := unifiedrole.LocalizeRole(original, "")

	// Empty locale falls back to source strings
	g.Expect(result.GetDisplayName()).To(Equal(original.GetDisplayName()))
	g.Expect(result.GetDescription()).To(Equal(original.GetDescription()))
}

func TestLocalizeRoles_German(t *testing.T) {
	g := NewWithT(t)
	roles := unifiedrole.BuildInRoles

	results := unifiedrole.LocalizeRoles(roles, "de")

	g.Expect(results).To(HaveLen(len(roles)))

	// Every result is a value (not a pointer)
	for i, r := range results {
		// Id is preserved
		g.Expect(r.GetId()).To(Equal(roles[i].GetId()))
		// Global singleton is not mutated
		g.Expect(roles[i].GetDisplayName()).NotTo(Equal(r.GetDisplayName()),
			"global displayName for role %s was mutated", r.GetId())
	}
}

func TestLocalizeRole_ConcurrentCallsDoNotRace(t *testing.T) {
	// Run with -race to detect data races on the global buildInRoles strings.
	const goroutines = 20
	var wg sync.WaitGroup
	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		locale := "de"
		if i%2 == 0 {
			locale = "fr"
		}
		go func(loc string) {
			defer wg.Done()
			_ = unifiedrole.LocalizeRoles(unifiedrole.BuildInRoles, loc)
		}(locale)
	}
	wg.Wait()

	// After all concurrent translations the globals must still hold English strings
	g := NewWithT(t)
	g.Expect(unifiedrole.RoleViewer.GetDisplayName()).To(Equal("Can view"))
}
