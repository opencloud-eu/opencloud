package command

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/test-go/testify/require"
)

func newStorage(t *testing.T, root string) string {
	t.Helper()
	for _, dir := range []string{"indexes", "users/space1/docs"} {
		require.NoError(t, os.MkdirAll(filepath.Join(root, dir), 0o700))
	}
	return root
}

func TestCheckStorageRoot(t *testing.T) {
	configured := newStorage(t, t.TempDir())
	other := newStorage(t, t.TempDir())
	nested := newStorage(t, filepath.Join(configured, "users/space1/docs/nested"))
	link := filepath.Join(t.TempDir(), "link")
	require.NoError(t, os.Symlink(configured, link))

	tests := []struct {
		name       string
		configured string
		paths      []string
		wantErr    string
	}{
		{"path in the configured storage", configured, []string{filepath.Join(configured, "users/space1/docs")}, ""},
		{"the configured root itself", configured, []string{configured}, ""},
		{"path outside any storage", configured, []string{t.TempDir()}, ""},
		{"path in another storage", configured, []string{filepath.Join(other, "users/space1")}, "not under the configured posixfs root"},
		{"a later path in another storage", configured, []string{configured, filepath.Join(other, "users")}, other},
		{"path in a storage nested inside the configured one", configured, []string{filepath.Join(nested, "users")}, nested},
		// the ignore rules compare strings, so a symlink in either path must be refused
		{"configured root through a symlink", link, []string{filepath.Join(configured, "users/space1")}, "not under the configured posixfs root"},
		{"path through a symlink", configured, []string{filepath.Join(link, "users/space1")}, "not under the configured posixfs root"},
		{"relative path", configured, []string{"users/space1"}, "not an absolute path"},
		{"configured root that doesn't exist", filepath.Join(configured, "missing"), []string{configured}, "not accessible"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := checkStorageRoot(tc.configured, tc.paths)
			if tc.wantErr == "" {
				require.NoError(t, err)
				return
			}
			require.Error(t, err)
			require.Contains(t, err.Error(), tc.wantErr)
		})
	}
}
