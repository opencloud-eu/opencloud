package userroles

import (
	"context"

	cs3 "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"
	"github.com/opencloud-eu/opencloud/pkg/middleware"
	settingssvc "github.com/opencloud-eu/opencloud/protogen/gen/opencloud/services/settings/v0"
	"github.com/opencloud-eu/opencloud/services/settings/pkg/store/defaults"
	"github.com/opencloud-eu/reva/v2/pkg/utils"
	"go-micro.dev/v4/metadata"
)

type defaultRoleAssigner struct {
	Options
}

// NewDefaultRoleAssigner returns an implementation of the UserRoleAssigner interface
func NewDefaultRoleAssigner(opts ...Option) UserRoleAssigner {
	opt := Options{}
	for _, o := range opts {
		o(&opt)
	}

	return defaultRoleAssigner{
		Options: opt,
	}
}

// UpdateUserRoleAssignment assigns the role "User" to the supplied user. Unless the user
// already has a different role assigned.
func (d defaultRoleAssigner) UpdateUserRoleAssignment(ctx context.Context, user *cs3.User, claims map[string]any) (*cs3.User, error) {
	var roleIDs []string
	if user.Id.Type != cs3.UserType_USER_TYPE_LIGHTWEIGHT {
		var err error
		roleIDs, err = loadRolesIDs(ctx, user.Id.OpaqueId, d.roleService)
		if err != nil {
			d.logger.Error().Err(err).Msg("Could not load roles")
			return nil, err
		}

		if len(roleIDs) == 0 {
			// This user doesn't have a role assignment yet. Assign a
			// default user role. At least until proper roles are provided. See
			// https://github.com/owncloud/ocis/issues/1825 for more context.
			if user.Id.Type == cs3.UserType_USER_TYPE_PRIMARY || user.Id.Type == cs3.UserType_USER_TYPE_GUEST {
				roleId := defaults.BundleUUIDRoleUser
				if user.Id.Type == cs3.UserType_USER_TYPE_GUEST {
					roleId = defaults.BundleUUIDRoleGuest
				}
				d.logger.Info().Str("userid", user.Id.OpaqueId).Msg("user has no role assigned, assigning default user role")
				ctx = metadata.Set(ctx, middleware.AccountID, user.Id.OpaqueId)
				_, err := d.roleService.AssignRoleToUser(ctx, &settingssvc.AssignRoleToUserRequest{
					AccountUuid: user.Id.OpaqueId,
					RoleId:      roleId,
				})
				if err != nil {
					d.logger.Error().Err(err).Msg("Could not add default role")
					return nil, err
				}
				roleIDs = append(roleIDs, roleId)
			}
		}
	}

	user.Opaque = utils.AppendJSONToOpaque(user.Opaque, "roles", roleIDs)
	return user, nil
}

// ApplyUserRole it looks up the user's role in the settings service and adds it
// user's opaque data
func (d defaultRoleAssigner) ApplyUserRole(ctx context.Context, user *cs3.User) (*cs3.User, error) {
	roleIDs, err := loadRolesIDs(ctx, user.Id.OpaqueId, d.roleService)
	if err != nil {
		d.logger.Error().Err(err).Msg("Could not load roles")
		return nil, err
	}

	user.Opaque = utils.AppendJSONToOpaque(user.Opaque, "roles", roleIDs)
	return user, nil
}
