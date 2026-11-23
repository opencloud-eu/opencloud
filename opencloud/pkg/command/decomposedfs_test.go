package command

import (
	"testing"

	"github.com/opencloud-eu/reva/v3/pkg/storage/pkg/decomposedfs/metadata"
	"github.com/spf13/cobra"
	"github.com/test-go/testify/require"
)

func TestBackend(t *testing.T) {
	t.Run("selects messagepack", func(t *testing.T) {
		b, err := backend("mpk")
		require.NoError(t, err)
		require.IsType(t, metadata.MessagePackBackend{}, b)
	})

	for _, bod := range []string{"xattrs", "ini", "unknown", ""} {
		t.Run("rejects the unsupported backend "+bod, func(t *testing.T) {
			b, err := backend(bod)
			require.EqualError(t, err, `unsupported metadata backend "`+bod+`"`)
			require.Nil(t, b)
		})
	}
}

func TestGetBackend(t *testing.T) {
	t.Run("fails when no supported metadata backend is found on disk", func(t *testing.T) {
		cmd := &cobra.Command{}
		cmd.Flags().String("root", t.TempDir(), "")

		lu, b, err := getBackend(cmd)
		require.Error(t, err)
		require.Contains(t, err.Error(), "unsupported metadata backend")
		require.Nil(t, lu)
		require.Nil(t, b)
	})
}
