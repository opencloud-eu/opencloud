package filepathx

import (
	"fmt"
	"path/filepath"
	"strings"
)

// JailJoin joins any number of path elements into a single path,
// it protects against directory traversal by removing any "../" elements
// and ensuring that the path is always under the jail.
func JailJoin(jail string, elem ...string) string {
	return filepath.Join(jail, filepath.Join(append([]string{"/"}, elem...)...))
}

// Determines whether the file or directory 'child' is same as or underneath the directory 'parent'.
//
// Note that 'parent' is expected to be a directory.
func IsSameOrContainedBy(parent string, child string) (bool, error) {
	absParent, err := filepath.Abs(parent)
	if err != nil {
		return false, fmt.Errorf("failed to make parent directory absolute: %q: %w", parent, err)
	}

	absChild, err := filepath.Abs(child)
	if err != nil {
		return false, fmt.Errorf("failed to make child file/directory absolute: %q: %w", child, err)
	}

	rel, err := filepath.Rel(absParent, absChild)
	if err != nil {
		return false, fmt.Errorf("failed to determine the relative path between the parent directory %q and the child file/directory: %q: %w", absParent, absChild, err)
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return false, nil
	}
	return true, nil
}
