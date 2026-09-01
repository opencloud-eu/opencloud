package trash

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

const (
	// _trashGlobPattern is the glob pattern to find all trash items
	_trashGlobPattern = "spaces/*/*/trash/*/*/*/*"
	// _trashRootPattern is the glob pattern of the trash container root
	_trashRootPattern = "spaces/*/*/trash"
	// _posixTrashGlobPattern is the glob pattern to find all trash items on posix
	_posixTrashGlobPattern = "*/*/.Trash/files/*"
	// _posixTrashRootPattern is the glob pattern of the trash container root on posix
	_posixTrashRootPattern = "*/*/.Trash/files"
)

// PurgeTrashEmptyPaths purges empty paths in the trash
func PurgeTrashEmptyPaths(p string, dryRun bool, posix bool) error {
	pattern := _trashGlobPattern
	if posix {
		pattern = _posixTrashGlobPattern
	}

	// we have all trash nodes in all spaces now
	dirs, err := filepath.Glob(filepath.Join(p, pattern))
	if err != nil {
		return err
	}

	if len(dirs) == 0 {
		return errors.New("no trash found. Double check storage path")
	}

	for _, d := range dirs {
		if err := removeEmptyFolder(d, dryRun, posix, p); err != nil {
			return err
		}
	}
	return nil
}

func removeEmptyFolder(path string, dryRun bool, posix bool, basePath string) error {

	if dryRun {
		if posix {
			// on posix the ".trashitem" entries are the actual data and can be
			// files, which are skipped, so we need to check here if the path
			// is the actual dir, the same check for real removal part
			fi, err := os.Stat(path)
			if err != nil || !fi.IsDir() {
				return nil
			}
		}

		f, err := os.ReadDir(path)
		if err != nil {
			return err
		}
		if len(f) < 1 {
			fmt.Println("would remove", path)
		}
		return nil
	}

	if posix {
		fi, err := os.Stat(path)
		if err != nil || !fi.IsDir() {
			return nil
		}
	}

	if err := os.Remove(path); err != nil {
		// we do not really care about the error here
		// if the folder is not empty we will get an error,
		// this is our signal to break out of the recursion
		return nil
	}
	nd := filepath.Dir(path)
	if isTrashRoot(nd, basePath, posix) {
		return nil
	}
	return removeEmptyFolder(nd, dryRun, posix, basePath)
}

func isTrashRoot(path, basePath string, posix bool) bool {
	rootPattern := _trashRootPattern
	if posix {
		rootPattern = _posixTrashRootPattern
	}
	matched, _ := filepath.Match(filepath.Join(basePath, rootPattern), path)
	return matched
}
