package revaconfig

import (
	"testing"

	"github.com/test-go/testify/require"

	"github.com/opencloud-eu/opencloud/pkg/config/envdecode"
	"github.com/opencloud-eu/opencloud/services/storage-system/pkg/config/defaults"
)

func TestMetadataPrefixReachesMetadataDriver(t *testing.T) {
	t.Setenv("OC_METADATA_PREFIX", "user.foreign.")
	cfg := defaults.FullDefaultConfig()
	require.NoError(t, envdecode.Decode(cfg))

	require.Equal(t, "user.foreign.", metadataDrivers("", cfg)["decomposed"].(map[string]any)["metadata_prefix"])
}
