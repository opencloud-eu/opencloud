package validate

import (
	"context"
	"slices"

	"github.com/go-playground/validator/v10"
	libregraph "github.com/opencloud-eu/libre-graph-api-go"

	"github.com/opencloud-eu/opencloud/services/graph/pkg/unifiedrole"
)

type contextKey int

const (
	ContextKeyRoleIDsValueKey contextKey = iota
)

// initLibregraph initializes libregraph validation
func initLibregraph(v *validator.Validate) {
	for _, f := range []func(*validator.Validate){
		libregraphDriveItemInvite,
		libregraphDriveRecipient,
		libregraphPermission,
	} {
		f(v)
	}
}

// libregraphDriveItemInvite validates libregraph.DriveItemInvite
func libregraphDriveItemInvite(v *validator.Validate) {
	s := libregraph.DriveItemInvite{}

	v.RegisterStructValidationMapRules(map[string]string{
		"Recipients":         "len=1,dive",
		"Roles":              "max=1",
		"ExpirationDateTime": "omitnil,gt",
	}, s)

	v.RegisterStructValidationCtx(func(ctx context.Context, sl validator.StructLevel) {
		driveItemInvite := sl.Current().Interface().(libregraph.DriveItemInvite)
		rolesAndActions(ctx, sl, driveItemInvite.Roles, driveItemInvite.LibreGraphPermissionsActions, false)
	}, s)
}

// libregraphDriveRecipient validates libregraph.DriveRecipient
func libregraphDriveRecipient(v *validator.Validate) {
	v.RegisterStructValidationMapRules(map[string]string{
		"ObjectId":                "ne=",
		"LibreGraphRecipientType": "oneof=user group",
	}, libregraph.DriveRecipient{})
}

// libregraphPermission validates libregraph.Permission
func libregraphPermission(v *validator.Validate) {
	s := libregraph.Permission{}

	v.RegisterStructValidationMapRules(map[string]string{
		"Roles": "max=1",
	}, s)
	v.RegisterStructValidationCtx(func(ctx context.Context, sl validator.StructLevel) {
		permission := sl.Current().Interface().(libregraph.Permission)

		if _, ok := permission.GetIdOk(); ok {
			sl.ReportError(permission.Id, "Id", "Id", "readonly", "")
		}

		if _, ok := permission.GetHasPasswordOk(); ok {
			sl.ReportError(permission.HasPassword, "hasPassword", "HasPassword", "readonly", "")
		}

		rolesAndActions(ctx, sl, permission.Roles, permission.LibreGraphPermissionsActions, true)
	}, s)
}

func rolesAndActions(ctx context.Context, sl validator.StructLevel, roles, actions []string, allowEmpty bool) {
	totalRoles := len(roles)
	totalActions := len(actions)
	switch {
	case allowEmpty && totalRoles == 0 && totalActions == 0:
		break
	case totalRoles != 0 && totalActions != 0:
		fallthrough
	case totalRoles == totalActions:
		sl.ReportError(roles, "Roles", "Roles", "one_or_another", "")
		sl.ReportError(actions, "LibreGraphPermissionsActions", "LibreGraphPermissionsActions", "one_or_another", "")
	}

	var availableRoles []string
	var availableActions []string
	var definitions []*libregraph.UnifiedRoleDefinition

	switch roles, ok := ctx.Value(ContextKeyRoleIDsValueKey).([]string); {
	case ok:
		definitions = unifiedrole.GetRoles(unifiedrole.RoleFilterIDs(roles...))
	default:
		// it the ctx does not contain the allowed role IDs, we need to fall back to all roles
		definitions = unifiedrole.GetRoles(unifiedrole.RoleFilterAll())
	}

	for _, definition := range definitions {
		if slices.Contains(availableRoles, definition.GetId()) {
			continue
		}

		availableRoles = append(availableRoles, definition.GetId())

		for _, permission := range definition.GetRolePermissions() {
			for _, action := range permission.GetAllowedResourceActions() {
				if slices.Contains(availableActions, action) {
					continue
				}

				availableActions = append(availableActions, action)
			}
		}
	}

	for _, role := range roles {
		if slices.Contains(availableRoles, role) {
			continue
		}

		sl.ReportError(roles, "Roles", "Roles", "available_role", "")
	}

	for _, action := range actions {
		if slices.Contains(availableActions, action) {
			continue
		}

		sl.ReportError(actions, "LibreGraphPermissionsActions", "LibreGraphPermissionsActions", "available_action", "")
	}
}

// ContextWithAllowedRoleIDs returns a new context which includes the allowed role IDs.
func ContextWithAllowedRoleIDs(ctx context.Context, rolesIds []string) context.Context {
	return context.WithValue(ctx, ContextKeyRoleIDsValueKey, rolesIds)
}
