package unifiedrole_test

import (
	"slices"

	libregraph "github.com/opencloud-eu/libre-graph-api-go"
)

func getRoleActions(definitions ...*libregraph.UnifiedRoleDefinition) []string {
	var actions []string

	for _, definition := range definitions {
		for _, permission := range definition.GetRolePermissions() {
			for _, action := range permission.GetAllowedResourceActions() {
				if slices.Contains(actions, action) {
					continue
				}
				actions = append(actions, action)
			}
		}
	}

	return actions
}
