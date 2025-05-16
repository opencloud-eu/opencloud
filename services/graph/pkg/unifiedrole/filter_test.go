package unifiedrole_test

import (
	"testing"

	. "github.com/onsi/gomega"
	libregraph "github.com/opencloud-eu/libre-graph-api-go"

	"github.com/opencloud-eu/opencloud/services/graph/pkg/unifiedrole"
)

func TestRoleFilterIDs(t *testing.T) {
	NewWithT(t).Expect(
		unifiedrole.RoleFilterIDs(
			unifiedrole.UnifiedRoleEditorLiteID,
			unifiedrole.UnifiedRoleSpaceEditorID,
		)(unifiedrole.RoleEditorLite),
	).To(BeTrue())
}

func TestRoleFilterInvert(t *testing.T) {
	NewWithT(t).Expect(
		unifiedrole.RoleFilterInvert(
			func(_ *libregraph.UnifiedRoleDefinition) bool {
				return true
			},
		)(unifiedrole.RoleEditorLite),
	).To(BeFalse())
}

func TestRoleFilterAll(t *testing.T) {
	NewWithT(t).Expect(
		unifiedrole.RoleFilterAll()(unifiedrole.RoleEditorLite),
	).To(BeTrue())
}
