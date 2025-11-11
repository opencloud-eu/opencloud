package store

import (
	"testing"

	"github.com/google/uuid"
	settingsmsg "github.com/opencloud-eu/opencloud/protogen/gen/opencloud/messages/settings/v0"
	"github.com/stretchr/testify/require"
)

var bundleScenarios = []struct {
	name   string
	bundle *settingsmsg.Bundle
}{
	{
		name: "generic-test-file-resource",
		bundle: &settingsmsg.Bundle{
			Id:          bundle1,
			Type:        settingsmsg.Bundle_TYPE_DEFAULT,
			Extension:   extension1,
			DisplayName: "test1",
			Resource: &settingsmsg.Resource{
				Type: settingsmsg.Resource_TYPE_FILE,
				Id:   "beep",
			},
			Settings: []*settingsmsg.Setting{
				{
					Id:          setting1,
					Description: "test-desc-1",
					DisplayName: "test-displayname-1",
					Resource: &settingsmsg.Resource{
						Type: settingsmsg.Resource_TYPE_FILE,
						Id:   "bleep",
					},
					Value: &settingsmsg.Setting_IntValue{
						IntValue: &settingsmsg.Int{
							Min: 0,
							Max: 42,
						},
					},
				},
			},
		},
	},
	{
		name: "generic-test-system-resource",
		bundle: &settingsmsg.Bundle{
			Id:          bundle2,
			Type:        settingsmsg.Bundle_TYPE_DEFAULT,
			Extension:   extension2,
			DisplayName: "test1",
			Resource: &settingsmsg.Resource{
				Type: settingsmsg.Resource_TYPE_SYSTEM,
			},
			Settings: []*settingsmsg.Setting{
				{
					Id:          setting2,
					Description: "test-desc-2",
					DisplayName: "test-displayname-2",
					Resource: &settingsmsg.Resource{
						Type: settingsmsg.Resource_TYPE_SYSTEM,
					},
					Value: &settingsmsg.Setting_IntValue{
						IntValue: &settingsmsg.Int{
							Min: 0,
							Max: 42,
						},
					},
				},
			},
		},
	},
	{
		name: "generic-test-role-bundle",
		bundle: &settingsmsg.Bundle{
			Id:          bundle3,
			Type:        settingsmsg.Bundle_TYPE_ROLE,
			Extension:   extension1,
			DisplayName: "Role1",
			Resource: &settingsmsg.Resource{
				Type: settingsmsg.Resource_TYPE_SYSTEM,
			},
			Settings: []*settingsmsg.Setting{
				{
					Id:          setting3,
					Description: "test-desc-3",
					DisplayName: "test-displayname-3",
					Resource: &settingsmsg.Resource{
						Type: settingsmsg.Resource_TYPE_SETTING,
						Id:   setting1,
					},
					Value: &settingsmsg.Setting_PermissionValue{
						PermissionValue: &settingsmsg.Permission{
							Operation:  settingsmsg.Permission_OPERATION_READ,
							Constraint: settingsmsg.Permission_CONSTRAINT_OWN,
						},
					},
				},
			},
		},
	},
}

var (
	appendTestBundleID = uuid.NewString()

	appendTestSetting1 = &settingsmsg.Setting{
		Id:          "append-test-setting-1",
		Description: "test-desc-3",
		DisplayName: "test-displayname-3",
		Resource: &settingsmsg.Resource{
			Type: settingsmsg.Resource_TYPE_SETTING,
			Id:   setting1,
		},
		Value: &settingsmsg.Setting_PermissionValue{
			PermissionValue: &settingsmsg.Permission{
				Operation:  settingsmsg.Permission_OPERATION_READ,
				Constraint: settingsmsg.Permission_CONSTRAINT_OWN,
			},
		},
	}

	appendTestSetting2 = &settingsmsg.Setting{
		Id:          "append-test-setting-2",
		Description: "test-desc-3",
		DisplayName: "test-displayname-3",
		Resource: &settingsmsg.Resource{
			Type: settingsmsg.Resource_TYPE_SETTING,
			Id:   setting1,
		},
		Value: &settingsmsg.Setting_PermissionValue{
			PermissionValue: &settingsmsg.Permission{
				Operation:  settingsmsg.Permission_OPERATION_READ,
				Constraint: settingsmsg.Permission_CONSTRAINT_OWN,
			},
		},
	}
)

func TestBundles(t *testing.T) {
	s := initStore()
	for i := range bundleScenarios {
		b := bundleScenarios[i]
		t.Run(b.name, func(t *testing.T) {
			_, err := s.WriteBundle(b.bundle)
			require.NoError(t, err)
			bundle, err := s.ReadBundle(b.bundle.Id)
			require.NoError(t, err)
			require.Equal(t, b.bundle, bundle)
		})
	}

	// check that ListBundles only returns bundles with type DEFAULT
	bundles, err := s.ListBundles(settingsmsg.Bundle_TYPE_DEFAULT, []string{})
	require.NoError(t, err)
	for i := range bundles {
		require.Equal(t, settingsmsg.Bundle_TYPE_DEFAULT, bundles[i].Type)
	}

	// check that ListBundles filtered by an id only returns that bundle
	filteredBundles, err := s.ListBundles(settingsmsg.Bundle_TYPE_DEFAULT, []string{bundle2})
	require.NoError(t, err)
	require.Equal(t, 1, len(filteredBundles))
	if len(filteredBundles) == 1 {
		require.Equal(t, bundle2, filteredBundles[0].Id)
	}

	// check that ListRoles only returns bundles with type ROLE
	roles, err := s.ListBundles(settingsmsg.Bundle_TYPE_ROLE, []string{})
	require.NoError(t, err)
	for i := range roles {
		require.Equal(t, settingsmsg.Bundle_TYPE_ROLE, roles[i].Type)
	}

	// check that ReadSetting works
	setting, err := s.ReadSetting(setting1)
	require.NoError(t, err)
	require.Equal(t, "test-desc-1", setting.Description) // could be tested better ;)
}

func TestAppendSetting(t *testing.T) {
	s := initStore()
	setupRoles(s)

	// appending to non existing bundle creates new
	_, err := s.AddSettingToBundle(appendTestBundleID, appendTestSetting1)
	require.NoError(t, err)

	b, err := s.ReadBundle(appendTestBundleID)
	require.NoError(t, err)
	require.Len(t, b.Settings, 1)

	_, err = s.AddSettingToBundle(appendTestBundleID, appendTestSetting2)
	require.NoError(t, err)

	b, err = s.ReadBundle(appendTestBundleID)
	require.NoError(t, err)
	require.Len(t, b.Settings, 2)

}
