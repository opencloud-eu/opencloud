package svc

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRegex(t *testing.T) {
	require := require.New(t)

	matches := authRegex.FindStringSubmatch("Basic abc")
	require.NotNil(matches)
	require.Len(matches, 3)
	require.Equal("Basic", matches[1])
	require.Equal("abc", matches[2])
}
