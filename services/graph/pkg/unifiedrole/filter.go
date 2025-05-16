package unifiedrole

import (
	"slices"

	libregraph "github.com/opencloud-eu/libre-graph-api-go"
)

type (
	// RoleFilter is used to filter role collections
	RoleFilter func(*libregraph.UnifiedRoleDefinition) bool
)

// RoleFilterInvert inverts the provided role filter
func RoleFilterInvert(f RoleFilter) RoleFilter {
	return func(r *libregraph.UnifiedRoleDefinition) bool {
		return !f(r)
	}
}

// RoleFilterAll returns a role filter that matches all roles
func RoleFilterAll() RoleFilter {
	return func(r *libregraph.UnifiedRoleDefinition) bool {
		return true
	}
}

// RoleFilterIDs returns a role filter that matches the provided ids
// the filter is always OR!
func RoleFilterIDs(ids ...string) RoleFilter {
	return func(r *libregraph.UnifiedRoleDefinition) bool {
		return slices.Contains(ids, r.GetId())
	}
}

// filterRoles filters the provided roles by the provided filter
func filterRoles(roles []*libregraph.UnifiedRoleDefinition, f RoleFilter) []*libregraph.UnifiedRoleDefinition {
	return slices.DeleteFunc(
		slices.Clone(roles),
		func(r *libregraph.UnifiedRoleDefinition) bool {
			return !f(r)
		},
	)
}
