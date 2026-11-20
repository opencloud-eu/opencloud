package command

import (
	"io"
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

// runMetadataCmd runs "opencloud decomposedfs metadata" with the given arguments
func runMetadataCmd(args ...string) error {
	cmd := metadataCmd(nil)
	cmd.SetArgs(args)
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	return cmd.Execute()
}

func TestMetadataCommands(t *testing.T) {
	// the arguments each subcommand needs besides --root and --node
	subcommands := map[string][]string{
		"dump": nil,
		"get":  {"--attribute", "user.*"},
		"set":  {"--attribute", "user.foo", "--value", "bar"},
	}

	for name, extra := range subcommands {
		t.Run(name, func(t *testing.T) {
			// The root is an empty directory, so the command can't read any metadata
			// there, but it only gets that far with the value of --root.
			root := t.TempDir()
			readRoot := `could not read the metadata in "` + root + `"`

			t.Run("takes --root and --node after the subcommand", func(t *testing.T) {
				args := append([]string{name, "--root", root, "--node", "some-node"}, extra...)
				err := runMetadataCmd(args...)
				require.Error(t, err)
				require.Contains(t, err.Error(), readRoot)
			})

			t.Run("takes --root and --node before the subcommand", func(t *testing.T) {
				args := append([]string{"--root", root, "--node", "some-node", name}, extra...)
				err := runMetadataCmd(args...)
				require.Error(t, err)
				require.Contains(t, err.Error(), readRoot)
			})

			t.Run("takes the short flags -r and -n", func(t *testing.T) {
				args := append([]string{name, "-r", root, "-n", "some-node"}, extra...)
				err := runMetadataCmd(args...)
				require.Error(t, err)
				require.Contains(t, err.Error(), readRoot)
			})

			t.Run("requires --root", func(t *testing.T) {
				args := append([]string{name, "--node", "some-node"}, extra...)
				err := runMetadataCmd(args...)
				require.EqualError(t, err, `required flag(s) "root" not set`)
			})

			t.Run("requires --node", func(t *testing.T) {
				args := append([]string{name, "--root", root}, extra...)
				err := runMetadataCmd(args...)
				require.EqualError(t, err, `required flag(s) "node" not set`)
			})
		})
	}
}
