// Package fsutil Filesystem util functions, quick create, read and write file. eg: file and dir check, operate
package fsutil

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/gookit/goutil/basefn"
	"github.com/gookit/goutil/internal/comfunc"
)

// PathSep alias of os.PathSeparator
const PathSep = os.PathSeparator

// JoinPaths elements, alias of filepath.Join()
func JoinPaths(elem ...string) string {
	return filepath.Join(elem...)
}

// JoinPaths3 elements, like the filepath.Join()
func JoinPaths3(basePath, secPath string, elems ...string) string {
	return comfunc.JoinPaths3(basePath, secPath, elems)
}

// JoinSubPaths elements, like the filepath.Join()
func JoinSubPaths(basePath string, elems ...string) string {
	return comfunc.JoinPaths2(basePath, elems)
}

// SlashPath alias of filepath.ToSlash
func SlashPath(path string) string {
	return filepath.ToSlash(path)
}

// UnixPath like of filepath.ToSlash, but always replace
func UnixPath(path string) string {
	if !strings.ContainsRune(path, '\\') {
		return path
	}
	return strings.ReplaceAll(path, "\\", "/")
}

// ToAbsPath convert path to absolute path.
// Will expand home dir, if empty will return current work dir
//
// TIP: will don't check path is really exists
func ToAbsPath(p string) string {
	// return current work dir
	if len(p) == 0 {
		wd, err := os.Getwd()
		if err != nil {
			return p
		}
		return wd
	}

	if IsAbsPath(p) {
		return p
	}

	// expand home dir
	if p[0] == '~' {
		return comfunc.ExpandHome(p)
	}

	wd, err := os.Getwd()
	if err != nil {
		return p
	}
	return filepath.Join(wd, p)
}

// Must2 ok for (any, error) result. if it has error, will panic
func Must2(_ any, err error) { basefn.MustOK(err) }
