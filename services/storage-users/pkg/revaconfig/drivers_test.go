package revaconfig

import (
	"bytes"
	"encoding/base64"
	"testing"

	"github.com/test-go/testify/require"

	"github.com/opencloud-eu/opencloud/pkg/config/envdecode"
	"github.com/opencloud-eu/opencloud/pkg/shared"
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

func TestSSECKeyReachesS3Drivers(t *testing.T) {
	key := base64.StdEncoding.EncodeToString(bytes.Repeat([]byte{42}, 32))
	t.Setenv("STORAGE_USERS_DECOMPOSEDS3_SSE_C_KEY", key)
	cfg := defaults.FullDefaultConfig()
	cfg.Commons = &shared.Commons{GRPCClientTLS: &shared.GRPCClientTLS{}}
	require.NoError(t, envdecode.Decode(cfg))
	require.Equal(t, key, DecomposedS3(cfg)["s3.sse_c_key"])
	require.Equal(t, key, DecomposedS3NoEvents(cfg)["s3.sse_c_key"])
}
