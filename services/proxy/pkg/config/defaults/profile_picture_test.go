package defaults

import (
	"testing"

	"github.com/opencloud-eu/opencloud/pkg/config/envdecode"
	"github.com/stretchr/testify/require"
)

func TestPictureClaimConfiguration(t *testing.T) {
	cfg := DefaultConfig()
	require.False(t, cfg.AutoprovisionAccounts)
	require.Equal(t, "picture", cfg.OIDC.PictureClaim)
	t.Setenv("PROXY_OIDC_PICTURE_CLAIM", "avatar")
	require.NoError(t, envdecode.Decode(cfg))
	require.Equal(t, "avatar", cfg.OIDC.PictureClaim)
	t.Setenv("PROXY_OIDC_PICTURE_CLAIM", "")
	require.NoError(t, envdecode.Decode(cfg))
	require.Empty(t, cfg.OIDC.PictureClaim)
}
