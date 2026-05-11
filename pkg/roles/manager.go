package roles

import (
	"context"
	"time"

	"github.com/opencloud-eu/opencloud/pkg/log"
	settingsmsg "github.com/opencloud-eu/opencloud/protogen/gen/opencloud/messages/settings/v0"
	settingssvc "github.com/opencloud-eu/opencloud/protogen/gen/opencloud/services/settings/v0"
	"github.com/opencloud-eu/reva/v2/pkg/store"
	microstore "go-micro.dev/v4/store"
	"google.golang.org/protobuf/encoding/protojson"
)

const (
	cacheDatabase  = "opencloud-pkg"
	cacheTableName = "roles"
	cacheTTL       = 24 * time.Hour
)

// Manager manages a cache of roles by fetching unknown roles from the settings.RoleService.
type Manager struct {
	logger      log.Logger
	roleCache   microstore.Store
	roleService settingssvc.RoleService
}

// NewManager returns a new instance of Manager.
func NewManager(o ...Option) Manager {
	opts := newOptions(o...)

	nStore := store.Create(opts.storeOptions...)
	return Manager{
		roleCache:   nStore,
		roleService: opts.roleService,
	}
}

// List returns all roles that match the given roleIDs.
func (m *Manager) List(ctx context.Context, roleIDs []string) []*settingsmsg.Bundle {
	// get from cache
	result := make([]*settingsmsg.Bundle, 0)
	lookup := make([]string, 0)
	for _, roleID := range roleIDs {
		if records, err := m.roleCache.Read(roleID, microstore.ReadFrom(cacheDatabase, cacheTableName)); err != nil {
			lookup = append(lookup, roleID)
		} else {
			role := &settingsmsg.Bundle{}
			found := false
			for _, record := range records {
				if record.Key == roleID {
					if err := protojson.Unmarshal(record.Value, role); err == nil {
						// if we can unmarshal the role, append it to the result
						// otherwise assume the role wasn't found (data was damaged and
						// we need to get the role again)
						result = append(result, role)
						found = true
						break
					}
				}
			}
			if !found {
				lookup = append(lookup, roleID)
			}
		}
	}

	// if there are roles missing, fetch them from the RoleService
	if len(lookup) > 0 {
		request := &settingssvc.ListBundlesRequest{
			BundleIds: lookup,
		}
		res, err := m.roleService.ListRoles(ctx, request)
		if err != nil {
			m.logger.Debug().Err(err).Msg("failed to fetch roles by roleIDs")
			return nil
		}
		for _, role := range res.Bundles {
			jsonbytes, _ := protojson.Marshal(role)
			record := &microstore.Record{
				Key:    role.Id,
				Value:  jsonbytes,
				Expiry: cacheTTL,
			}
			err := m.roleCache.Write(
				record,
				microstore.WriteTo(cacheDatabase, cacheTableName),
				microstore.WriteTTL(cacheTTL),
			)
			if err != nil {
				m.logger.Debug().Err(err).Msg("failed to cache roles")
			}
			result = append(result, role)
		}
	}

	return result
}

// FindPermissionByID searches for a permission-setting by the permissionID, but limited to the given roleIDs
func (m *Manager) FindPermissionByID(ctx context.Context, roleIDs []string, permissionID string) *settingsmsg.Setting {
	for _, role := range m.List(ctx, roleIDs) {
		for _, setting := range role.Settings {
			if setting.Id == permissionID {
				return setting
			}
		}
	}
	return nil
}

// FindRoleIdsForUser returns all roles that are assigned to the supplied userid
func (m *Manager) FindRoleIDsForUser(ctx context.Context, userID string) ([]string, error) {
	req := &settingssvc.ListRoleAssignmentsRequest{AccountUuid: userID}
	assignmentResponse, err := m.roleService.ListRoleAssignments(ctx, req)

	if err != nil {
		return nil, err
	}

	roleIDs := make([]string, 0, len(assignmentResponse.Assignments))

	for _, assignment := range assignmentResponse.Assignments {
		roleIDs = append(roleIDs, assignment.RoleId)
	}

	return roleIDs, nil
}
