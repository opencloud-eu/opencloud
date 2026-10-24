package trash

import (
	"os"
	"testing"

	"github.com/test-go/testify/require"
)

func TestIsTrashRootDecomposed(t *testing.T) {
	storageRoot := "test_temp_" + t.Name()
	defer os.RemoveAll(storageRoot)

	require.True(t, isTrashRoot(storageRoot+"/spaces/id/id/trash", storageRoot, false))
	require.False(t, isTrashRoot(storageRoot+"/spaces/id/id/trash/s1/s2/s3/node", storageRoot, false))
	require.False(t, isTrashRoot(storageRoot+"/spaces/id/id/trash/s1/s2/trash/node", storageRoot, false))
}

func TestIsTrashRootPosix(t *testing.T) {
	storageRoot := "test_temp_" + t.Name()
	defer os.RemoveAll(storageRoot)

	require.True(t, isTrashRoot(storageRoot+"/users/alice/.Trash/files", storageRoot, true))
	require.False(t, isTrashRoot(storageRoot+"/users/alice/.Trash/files/item.trashitem", storageRoot, true))
	require.False(t, isTrashRoot(storageRoot+"/users/alice/.Trash/files/item.trashitem/files", storageRoot, true))
}

func TestRemoveEmptyFolderPosix(t *testing.T) {
	storageRoot := "test_temp_" + t.Name()
	base := storageRoot + "/users/alice/.Trash/files"
	defer os.RemoveAll(storageRoot)

	emptyChain := base + "/empty.trashitem/sub/subsub"
	require.NoError(t, os.MkdirAll(emptyChain, os.ModePerm))

	nonEmpty := base + "/keep.trashitem"
	require.NoError(t, os.MkdirAll(nonEmpty, os.ModePerm))
	require.NoError(t, os.WriteFile(nonEmpty+"/file.txt", []byte("some text"), os.ModePerm))

	require.NoError(t, removeEmptyFolder(emptyChain, false, true, storageRoot))

	assertNoDirExists(t, emptyChain)
	assertNoDirExists(t, base+"/empty.trashitem")
	assertDirExists(t, base)
	assertDirExists(t, nonEmpty)
}

func TestRemoveEmptyFolderPosixUserDirNamedFiles(t *testing.T) {
	storageRoot := "test_temp_" + t.Name()
	base := storageRoot + "/users/alice/.Trash/files"
	defer os.RemoveAll(storageRoot)

	nestedFilesDir := base + "/folder.trashitem/files/files"
	require.NoError(t, os.MkdirAll(nestedFilesDir, os.ModePerm))

	require.NoError(t, removeEmptyFolder(nestedFilesDir, false, true, storageRoot))

	assertNoDirExists(t, nestedFilesDir)
	assertNoDirExists(t, base+"/folder.trashitem/files")
	assertNoDirExists(t, base+"/folder.trashitem")
	assertDirExists(t, base)
}

func TestRemoveEmptyFolderDecomposed(t *testing.T) {
	storageRoot := "test_temp_" + t.Name()
	base := storageRoot + "/spaces/id/id/trash"
	defer os.RemoveAll(storageRoot)

	emptyChain := base + "/s1/s2/s3/node"
	require.NoError(t, os.MkdirAll(emptyChain, os.ModePerm))

	require.NoError(t, removeEmptyFolder(emptyChain, false, false, storageRoot))

	assertNoDirExists(t, emptyChain)
	assertNoDirExists(t, base+"/s1/s2/s3")
	assertNoDirExists(t, base+"/s1/s2")
	assertNoDirExists(t, base+"/s1")
	assertDirExists(t, base)
}

func TestRemoveEmptyFolderDecomposedUserDirNamedTrash(t *testing.T) {
	storageRoot := "test_temp_" + t.Name()
	base := storageRoot + "/spaces/id/id/trash"
	defer os.RemoveAll(storageRoot)

	nestedTrashDir := base + "/s1/trash/s2/node"
	require.NoError(t, os.MkdirAll(nestedTrashDir, os.ModePerm))

	require.NoError(t, removeEmptyFolder(nestedTrashDir, false, false, storageRoot))

	assertNoDirExists(t, nestedTrashDir)
	assertNoDirExists(t, base+"/s1/trash/s2")
	assertNoDirExists(t, base+"/s1/trash")
	assertNoDirExists(t, base+"/s1")
	assertDirExists(t, base)
}

func assertNoDirExists(t *testing.T, path string) {
	t.Helper()
	_, err := os.Stat(path)
	require.True(t, os.IsNotExist(err))
}

func assertDirExists(t *testing.T, path string) {
	t.Helper()
	fi, err := os.Stat(path)
	require.NoError(t, err)
	require.True(t, fi.IsDir())
}
