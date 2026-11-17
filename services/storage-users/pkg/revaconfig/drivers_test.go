package revaconfig

import (
	"testing"

	"github.com/test-go/testify/require"

	"github.com/opencloud-eu/opencloud/pkg/config/envdecode"
	"github.com/opencloud-eu/opencloud/pkg/shared"
	"github.com/opencloud-eu/opencloud/pkg/storage"
	"github.com/opencloud-eu/opencloud/services/storage-users/pkg/config/defaults"
)

func TestMetadataPrefixReachesAllDrivers(t *testing.T) {
	t.Setenv("OC_METADATA_PREFIX", "user.foreign.")
	cfg := defaults.FullDefaultConfig()
	cfg.Commons = &shared.Commons{GRPCClientTLS: &shared.GRPCClientTLS{}}
	require.NoError(t, envdecode.Decode(cfg))

	for name, driver := range map[string]map[string]any{
		"posix":                  Posix(cfg, false, false),
		"decomposed":             Decomposed(cfg),
		"decomposed no events":   DecomposedNoEvents(cfg),
		"decomposeds3":           DecomposedS3(cfg),
		"decomposeds3 no events": DecomposedS3NoEvents(cfg),
	} {
		require.Equal(t, "user.foreign.", driver["metadata_prefix"], name)
	}
}

func TestLegacyDriverAliases(t *testing.T) {
	for _, tc := range []struct{ legacy, replacement string }{
		{"ocis", "decomposed"},
		{"s3ng", "decomposeds3"},
	} {
		t.Run(tc.legacy, func(t *testing.T) {
			cfg := defaults.FullDefaultConfig()
			cfg.Commons = &shared.Commons{GRPCClientTLS: &shared.GRPCClientTLS{}}
			cfg.GRPC.TLS = &shared.GRPCServiceTLS{}
			cfg.Driver = tc.legacy
			cfg.Drivers.Decomposed.MetadataPrefix = "user.other."
			cfg.Drivers.DecomposedS3.MetadataPrefix = "user.other."

			rcfg := StorageUsersConfigFromStruct(cfg)
			// Both Reva services must resolve the alias to the replacement driver and
			// use the legacy prefix instead of the configured one.
			for _, service := range []struct{ protocol, name string }{{"grpc", "storageprovider"}, {"http", "dataprovider"}} {
				services := rcfg[service.protocol].(map[string]any)["services"].(map[string]any)
				c := services[service.name].(map[string]any)
				require.Equal(t, tc.replacement, c["driver"])
				drivers := c["drivers"].(map[string]any)
				driver := drivers[tc.replacement].(map[string]any)
				require.Equal(t, storage.LegacyMetadataPrefix(), driver["metadata_prefix"])
				_, present := drivers[tc.legacy]
				require.False(t, present)
			}
			// Resolving an alias must not change the original OpenCloud config.
			require.Equal(t, "user.other.", cfg.Drivers.Decomposed.MetadataPrefix)
			require.Equal(t, "user.other.", cfg.Drivers.DecomposedS3.MetadataPrefix)
		})
	}
}
