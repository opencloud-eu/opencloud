package revaconfig

import (
	"testing"

	"github.com/test-go/testify/require"

	"github.com/opencloud-eu/opencloud/pkg/config/envdecode"
	"github.com/opencloud-eu/opencloud/pkg/shared"
	"github.com/opencloud-eu/opencloud/pkg/storage"
	"github.com/opencloud-eu/opencloud/services/storage-system/pkg/config/defaults"
)

func TestMetadataPrefixReachesMetadataDriver(t *testing.T) {
	t.Setenv("OC_METADATA_PREFIX", "user.foreign.")
	cfg := defaults.FullDefaultConfig()
	require.NoError(t, envdecode.Decode(cfg))

	require.Equal(t, "user.foreign.", metadataDrivers("", cfg)["decomposed"].(map[string]any)["metadata_prefix"])
}

func TestLegacyMetadataDriverAlias(t *testing.T) {
	cfg := defaults.FullDefaultConfig()
	cfg.Driver = "ocis"
	cfg.Drivers.Decomposed.MetadataPrefix = "user.other."
	cfg.Commons = &shared.Commons{GRPCClientTLS: &shared.GRPCClientTLS{}}
	cfg.GRPC.TLS = &shared.GRPCServiceTLS{}
	drivers := metadataDrivers("", cfg)
	// The alias uses the replacement driver's options with the legacy prefix,
	// even when a different prefix was configured; Reva no longer needs an ocis entry.
	require.Equal(t, storage.LegacyMetadataPrefix(), drivers["decomposed"].(map[string]any)["metadata_prefix"])
	_, present := drivers["ocis"]
	require.False(t, present)
	// Generating Reva's configuration must not modify the OpenCloud configuration.
	require.Equal(t, "user.other.", cfg.Drivers.Decomposed.MetadataPrefix)
	rcfg := StorageSystemFromStruct(cfg)
	// Both the gRPC storage provider and HTTP data provider must select the replacement.
	for _, service := range []struct{ protocol, name string }{{"grpc", "storageprovider"}, {"http", "dataprovider"}} {
		services := rcfg[service.protocol].(map[string]any)["services"].(map[string]any)
		require.Equal(t, "decomposed", services[service.name].(map[string]any)["driver"])
	}
}
