package revaconfig

import (
	"errors"
	"testing"
	"time"

	"github.com/opencloud-eu/reva/v2/pkg/storage/pkg/decomposedfs/options"
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

func TestRevisionsCleanupReachesAllDrivers(t *testing.T) {
	drivers := func(t *testing.T) map[string]map[string]any {
		cfg := defaults.FullDefaultConfig()
		cfg.Commons = &shared.Commons{GRPCClientTLS: &shared.GRPCClientTLS{}}
		if err := envdecode.Decode(cfg); !errors.Is(err, envdecode.ErrNoTargetFieldsAreSet) {
			require.NoError(t, err)
		}
		return map[string]map[string]any{
			"posix":                  Posix(cfg, false, false),
			"decomposed":             Decomposed(cfg),
			"decomposed no events":   DecomposedNoEvents(cfg),
			"decomposeds3":           DecomposedS3(cfg),
			"decomposeds3 no events": DecomposedS3NoEvents(cfg),
		}
	}

	t.Run("defaults keep all revisions", func(t *testing.T) {
		for name, driver := range drivers(t) {
			o, err := options.New(driver)
			require.NoError(t, err, name)
			require.Equal(t, "disabled", o.RevisionsCleanupStrategy, name)
			require.True(t, o.RevisionsCleanupAsync, name)
			require.Equal(t, 0, o.RevisionsCleanupAsyncWorkers, name)
			require.Equal(t, 0, o.RevisionsCleanupAsyncQueueSize, name)
			require.Equal(t, "drop", o.RevisionsCleanupAsyncQueueFull, name)
			require.Equal(t, 5*time.Second, o.RevisionsCleanupAsyncQueueFullTimeout, name)
		}
	})

	t.Run("environment variables", func(t *testing.T) {
		t.Setenv("OC_REVISIONS_CLEANUP_STRATEGY", "7d, auto, 90d")
		t.Setenv("OC_REVISIONS_CLEANUP_ASYNC", "false")
		t.Setenv("OC_REVISIONS_CLEANUP_ASYNC_WORKERS", "3")
		t.Setenv("OC_REVISIONS_CLEANUP_ASYNC_QUEUE_SIZE", "42")
		t.Setenv("OC_REVISIONS_CLEANUP_ASYNC_QUEUE_FULL", "block")
		t.Setenv("OC_REVISIONS_CLEANUP_ASYNC_QUEUE_FULL_TIMEOUT", "1m")
		for name, driver := range drivers(t) {
			// decoded by reva, so that a key it does not know would show up as a default value
			o, err := options.New(driver)
			require.NoError(t, err, name)
			require.Equal(t, "7d, auto, 90d", o.RevisionsCleanupStrategy, name)
			require.False(t, o.RevisionsCleanupAsync, name)
			require.Equal(t, 3, o.RevisionsCleanupAsyncWorkers, name)
			require.Equal(t, 42, o.RevisionsCleanupAsyncQueueSize, name)
			require.Equal(t, "block", o.RevisionsCleanupAsyncQueueFull, name)
			require.Equal(t, time.Minute, o.RevisionsCleanupAsyncQueueFullTimeout, name)
		}
	})
}
