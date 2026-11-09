package revaconfig

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/pkg/shared"
	"github.com/opencloud-eu/opencloud/services/frontend/pkg/config/defaults"
)

func TestContentSearchCapabilityFollowsFullTextSearch(t *testing.T) {
	for _, enabled := range []bool{true, false} {
		cfg := defaults.FullDefaultConfig()
		cfg.Commons = &shared.Commons{}
		cfg.FullTextSearch = enabled

		rcfg, err := FrontendConfigFromStruct(cfg, log.NopLogger())
		require.NoError(t, err)

		got := dig(t, rcfg, "http", "services", "ocs", "capabilities", "capabilities", "search", "property", "content")["enabled"]
		require.Equal(t, enabled, got)
	}
}

func dig(t *testing.T, m map[string]any, keys ...string) map[string]any {
	t.Helper()
	for _, k := range keys {
		next, ok := m[k].(map[string]any)
		require.True(t, ok, "no map at %q", k)
		m = next
	}
	return m
}
