package store

import (
	"log"
	"sync"
	"testing"

	"github.com/google/uuid"
	olog "github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/pkg/shared"
	settingsmsg "github.com/opencloud-eu/opencloud/protogen/gen/opencloud/messages/settings/v0"
	"github.com/opencloud-eu/opencloud/services/settings/pkg/config/defaults"
	"github.com/stretchr/testify/require"
)

var (
	alan   = "00000000-0000-0000-0000-000000000001"
	mary   = "00000000-0000-0000-0000-000000000002"
        dennis = "00000000-0000-0000-0000-000000000003"

	role1 = "11111111-1111-1111-1111-111111111111"
	role2 = "22222222-2222-2222-2222-222222222222"

	logger = olog.NewLogger(
		olog.Color(true),
		olog.Pretty(true),
		olog.Level("info"),
	)

	bundles = []*settingsmsg.Bundle{
		{
			Id:          "f36db5e6-a03c-40df-8413-711c67e40b47",
			Type:        settingsmsg.Bundle_TYPE_ROLE,
			DisplayName: "test role - reads | update",
			Name:        "TEST_ROLE",
			Extension:   "opencloud-settings",
			Resource: &settingsmsg.Resource{
				Type: settingsmsg.Resource_TYPE_BUNDLE,
			},
			Settings: []*settingsmsg.Setting{
				{
					Id:   "updateID",
					Name: "update",
					Value: &settingsmsg.Setting_PermissionValue{
						PermissionValue: &settingsmsg.Permission{
							Operation: settingsmsg.Permission_OPERATION_UPDATE,
						},
					},
					Resource: &settingsmsg.Resource{
						Type: settingsmsg.Resource_TYPE_SETTING,
					},
				},
				{
					Id:   "readID",
					Name: "read",
					Value: &settingsmsg.Setting_PermissionValue{
						PermissionValue: &settingsmsg.Permission{
							Operation: settingsmsg.Permission_OPERATION_READ,
						},
					},
					Resource: &settingsmsg.Resource{
						Type: settingsmsg.Resource_TYPE_BUNDLE,
					},
				},
			},
		},
		{
			Id:          "44f1a664-0a7f-461a-b0be-5b59e46bbc7a",
			Type:        settingsmsg.Bundle_TYPE_ROLE,
			DisplayName: "another",
			Name:        "ANOTHER_TEST_ROLE",
			Extension:   "opencloud-settings",
			Resource: &settingsmsg.Resource{
				Type: settingsmsg.Resource_TYPE_BUNDLE,
			},
			Settings: []*settingsmsg.Setting{
				{
					Id:   "readID",
					Name: "read",
					Value: &settingsmsg.Setting_PermissionValue{
						PermissionValue: &settingsmsg.Permission{
							Operation: settingsmsg.Permission_OPERATION_READ,
						},
					},
				},
			},
		},
	}
)

func initStore() *Store {
	s := &Store{
		Logger: logger,
		l:      &sync.Mutex{},
		cfg:    defaults.DefaultConfig(),
	}
	s.cfg.Commons = &shared.Commons{
		AdminUserID: uuid.NewString(),
	}

	_ = NewMDC(s)
	return s
}

func setupRoles(s *Store) {
	for i := range bundles {
		if _, err := s.WriteBundle(bundles[i]); err != nil {
			log.Fatal("error initializing ", err)
		}
	}
}

func TestAssignmentUniqueness(t *testing.T) {
	var scenarios = []struct {
		name       string
		userID     string
		firstRole  string
		secondRole string
	}{
		{
			"roles assignments",
			alan,
			"f36db5e6-a03c-40df-8413-711c67e40b47",
			"44f1a664-0a7f-461a-b0be-5b59e46bbc7a",
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			s := initStore()
			setupRoles(s)
			firstAssignment, err := s.WriteRoleAssignment(scenario.userID, scenario.firstRole)
			require.NoError(t, err)
			require.Equal(t, firstAssignment.RoleId, scenario.firstRole)
			// TODO: check entry exists

			list, err := s.ListRoleAssignments(scenario.userID)
			require.NoError(t, err)
			require.Equal(t, 1, len(list))
			require.Equal(t, list[0].RoleId, scenario.firstRole)

			// creating another assignment shouldn't add another entry, as we support max one role per user.
			// assigning the second role should remove the old
			secondAssignment, err := s.WriteRoleAssignment(scenario.userID, scenario.secondRole)
			require.NoError(t, err)
			require.Equal(t, secondAssignment.RoleId, scenario.secondRole)

			list, err = s.ListRoleAssignments(scenario.userID)
			require.NoError(t, err)
			require.Equal(t, 1, len(list))
			require.Equal(t, list[0].RoleId, scenario.secondRole)
		})
	}
}

func TestListRoleAssignmentByRole(t *testing.T) {
	type assignment struct {
		userID string
		roleID string
	}

	var scenarios = []struct {
		name        string
		assignments []assignment
		queryRole   string
		numResults  int
	}{
		{
			name: "just 2 assignments",
			assignments: []assignment{
				{
					userID: alan,
					roleID: role1,
				}, {
					userID: mary,
					roleID: role1,
				},
			},
			queryRole:  role1,
			numResults: 2,
		},
		{
			name: "no assignments match",
			assignments: []assignment{
				{
					userID: alan,
					roleID: role1,
				}, {
					userID: mary,
					roleID: role1,
				},
			},
			queryRole:  role2,
			numResults: 0,
		},
		{
			name: "only one assignment matches",
			assignments: []assignment{
				{
					userID: alan,
					roleID: role1,
				}, {
					userID: mary,
					roleID: role1,
				}, {
					userID: dennis,
					roleID: role2,
				},
			},
			queryRole:  role2,
			numResults: 1,
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			s := initStore()
			setupRoles(s)
			for _, a := range scenario.assignments {
				ass, err := s.WriteRoleAssignment(a.userID, a.roleID)
				require.NoError(t, err)
				require.Equal(t, ass.RoleId, a.roleID)
			}

			list, err := s.ListRoleAssignmentsByRole(scenario.queryRole)
			require.NoError(t, err)
			require.Equal(t, scenario.numResults, len(list))
			for _, ass := range list {
				require.Equal(t, ass.RoleId, scenario.queryRole)
			}
		})
	}
}

func TestDeleteAssignment(t *testing.T) {
	var scenarios = []struct {
		name       string
		userID     string
		firstRole  string
		secondRole string
	}{
		{
			"roles assignments",
			alan,
			"f36db5e6-a03c-40df-8413-711c67e40b47",
			"44f1a664-0a7f-461a-b0be-5b59e46bbc7a",
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			s := initStore()
			setupRoles(s)
			assignment, err := s.WriteRoleAssignment(scenario.userID, scenario.firstRole)
			require.NoError(t, err)
			require.Equal(t, assignment.RoleId, scenario.firstRole)
			// TODO: uncomment
			// require.True(t, mdc.IDExists(assignment.RoleId))

			list, err := s.ListRoleAssignments(scenario.userID)
			require.NoError(t, err)
			require.Equal(t, 1, len(list))
			require.Equal(t, assignment.Id, list[0].Id)

			err = s.RemoveRoleAssignment(assignment.Id)
			require.NoError(t, err)
			// TODO: uncomment
			// require.False(t, mdc.IDExists(assignment.RoleId))

			list, err = s.ListRoleAssignments(scenario.userID)
			require.NoError(t, err)
			require.Equal(t, 0, len(list))

			err = s.RemoveRoleAssignment(assignment.Id)
			require.Error(t, err)
			// TODO: do we want a custom error message?
		})
	}
}
