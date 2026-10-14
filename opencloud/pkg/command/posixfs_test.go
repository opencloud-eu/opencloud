package command

import (
	"os"
	"path/filepath"
	"testing"

	storageUsersConfig "github.com/opencloud-eu/opencloud/services/storage-users/pkg/config"
	"github.com/opencloud-eu/reva/v2/pkg/storage/fs/posix/ignore"
	"github.com/opencloud-eu/reva/v2/pkg/storage/fs/posix/options"
	"github.com/opencloud-eu/reva/v2/pkg/storage/pkg/decomposedfs/metadata/prefixes"
	"github.com/pkg/xattr"
	"github.com/rs/zerolog"
	"github.com/shamaton/msgpack/v2"
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

func newOptions(t *testing.T, root string) *options.Options {
	t.Helper()
	opts, err := options.New(map[string]any{"root": root})
	require.NoError(t, err)
	return opts
}

func TestProcessPosixFsResourcesFindsSpaces(t *testing.T) {
	root := t.TempDir()
	spaces := []string{
		"users/space1",                   // default layout
		"personal/alice-123",             // personal/{{.User.Username}}-{{.Id.OpaqueId}}
		"project/Team Space/456-spaceid", // {{.SpaceType}}/{{.SpaceName}}/{{.Id.OpaqueId}}-{{.SpaceId}}
	}
	// users/space1/indexes is a user directory without a node ID, not a nested storage
	for _, dir := range append([]string{"indexes", "users/space1/docs", "users/space1/indexes", "project/Team Space/plain"}, spaces...) {
		require.NoError(t, os.MkdirAll(filepath.Join(root, dir), 0o700))
	}
	for _, s := range spaces {
		require.NoError(t, xattr.Set(filepath.Join(root, s), prefixes.SpaceIDAttr, []byte(s)))
	}
	// a space ID below a space root belongs to that space and is not a space of its own
	require.NoError(t, xattr.Set(filepath.Join(root, "users/space1/docs"), prefixes.SpaceIDAttr, []byte("space1")))
	// the internal directories of the storage and a storage nested in it are not searched
	for _, dir := range []string{"indexes/by-type/x", "uploads/x", "backup/indexes", "backup/users/x"} {
		require.NoError(t, os.MkdirAll(filepath.Join(root, dir), 0o700))
		require.NoError(t, xattr.Set(filepath.Join(root, dir), prefixes.SpaceIDAttr, []byte(dir)))
	}
	require.NoError(t, os.RemoveAll(filepath.Join(root, "backup/indexes")))
	require.NoError(t, os.Mkdir(filepath.Join(root, "backup/indexes"), 0o700))
	// a sibling of an internal directory is searched
	require.NoError(t, os.MkdirAll(filepath.Join(root, "indexes-team/789"), 0o700))
	require.NoError(t, xattr.Set(filepath.Join(root, "indexes-team/789"), prefixes.SpaceIDAttr, []byte("789")))
	require.NoError(t, os.WriteFile(filepath.Join(root, "personal/file"), nil, 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(root, "project/.oc-nodes"), nil, 0o600))
	// a space root that lost its space ID still has its metadata directory
	require.NoError(t, os.MkdirAll(filepath.Join(root, "personal/bob-789/.oc-nodes"), 0o700))

	link := filepath.Join(t.TempDir(), "link")
	require.NoError(t, os.Symlink(root, link))
	for _, r := range []string{root, link} {
		found := []string{}
		errs := processPosixFsResources([]string{r}, internalDirs(newOptions(t, r)), true,
			func(path string) error { found = append(found, path); return nil },
			func(path string) error { t.Errorf("unexpected entity %s", path); return nil },
		)
		require.Len(t, errs, 1)
		require.Contains(t, errs[0].Error(), filepath.Join(r, "personal/bob-789"))
		want := []string{}
		for _, s := range []string{"indexes-team/789", "personal/alice-123", "project/Team Space/456-spaceid", "users/space1"} {
			want = append(want, filepath.Join(r, s))
		}
		require.Equal(t, want, found)
	}
}

func TestConsistencyIgnoresSpaceInternals(t *testing.T) {
	root := t.TempDir()
	space := filepath.Join(root, "personal/alice-123")
	for _, dir := range []string{".Trash/x", ".oc-nodes", ".oc-tmp", "docs/.Trash"} {
		require.NoError(t, os.MkdirAll(filepath.Join(space, dir), 0o700))
	}
	require.NoError(t, xattr.Set(space, prefixes.IDAttr, []byte("alice")))
	require.NoError(t, xattr.Set(space, prefixes.SpaceIDAttr, []byte("alice")))
	// a directory created through the API has the space ID as well
	require.NoError(t, xattr.Set(filepath.Join(space, "docs"), prefixes.IDAttr, []byte("docs")))
	require.NoError(t, xattr.Set(filepath.Join(space, "docs"), prefixes.SpaceIDAttr, []byte("alice")))
	log := zerolog.Nop()
	c := &consistencyChecker{ignorer: ignore.NewIgnorer(newOptions(t, root), &log)}

	for _, p := range []string{".Trash", ".Trash/x", ".oc-nodes", ".oc-tmp"} {
		require.True(t, c.isIgnored(filepath.Join(space, p)), p)
	}
	for _, p := range []string{filepath.Join(space, "docs"), filepath.Join(space, "docs/.Trash")} {
		require.False(t, c.isIgnored(p), p)
	}
}

func TestUpdateOwnerIndexFileStaysInIndexDir(t *testing.T) {
	root := t.TempDir()
	indexDir := filepath.Join(root, "indexes/by-user-id")
	space := filepath.Join(root, "users/alice")
	for _, dir := range []string{indexDir, space} {
		require.NoError(t, os.MkdirAll(dir, 0o700))
	}
	index, err := msgpack.Marshal(map[string]string{"old": "old", "keep": "keep"})
	require.NoError(t, err)
	for _, f := range []string{filepath.Join(indexDir, "alice.mpk"), filepath.Join(root, "outside.mpk")} {
		require.NoError(t, os.WriteFile(f, index, 0o600))
	}
	c := &consistencyChecker{cfg: &storageUsersConfig.Config{}}
	c.cfg.Drivers.Posix.Root = root

	require.NoError(t, xattr.Set(space, prefixes.OwnerIDAttr, []byte("alice")))
	require.NoError(t, c.updateOwnerIndexFile(space, []string{"old"}))
	rootDir, err := os.OpenRoot(root)
	require.NoError(t, err)
	defer rootDir.Close()
	data, err := rootDir.ReadFile("indexes/by-user-id/alice.mpk")
	require.NoError(t, err)
	var got map[string]string
	require.NoError(t, msgpack.Unmarshal(data, &got))
	require.Equal(t, map[string]string{"keep": "keep"}, got)

	// an owner ID that points out of the index directory is refused
	require.NoError(t, xattr.Set(space, prefixes.OwnerIDAttr, []byte("../../outside")))
	require.Error(t, c.updateOwnerIndexFile(space, []string{"old"}))
	data, err = rootDir.ReadFile("outside.mpk")
	require.NoError(t, err)
	require.Equal(t, index, data)
}
