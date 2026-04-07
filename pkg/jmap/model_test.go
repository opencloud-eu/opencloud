package jmap

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestObjectNames(t *testing.T) { //NOSONAR
	require := require.New(t)
	objectTypeNames, err := parseConsts("github.com/opencloud-eu/opencloud/pkg/jmap", "Name", "ObjectTypeName")
	require.NoError(err)
	for n, v := range objectTypeNames {
		require.True(strings.HasSuffix(n, "Name"))
		prefix := n[0 : len(n)-len("Name")]
		require.Equal(prefix, v)
	}
}
