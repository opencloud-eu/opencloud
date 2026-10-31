package auth_api

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseSecrets(t *testing.T) {
	require := require.New(t)

	{
		m, err := parseSecrets("")
		require.NoError(err)
		require.Empty(m)
	}
	{
		m, err := parseSecrets("app=123")
		require.NoError(err)
		require.Len(m, 1)
		require.Contains(m, "123")
		require.Equal(appId("app"), m["123"])
	}
	{
		m, err := parseSecrets("app1=123;app2=23456")
		require.NoError(err)
		require.Len(m, 2)
		require.Contains(m, "123")
		require.Equal(appId("app1"), m["123"])
		require.Contains(m, "23456")
		require.Equal(appId("app2"), m["23456"])
	}
	{
		m, err := parseSecrets("app=123=456")
		require.NoError(err)
		require.Len(m, 1)
		require.Contains(m, "123=456")
		require.Equal(appId("app"), m["123=456"])
	}
}
