package fsutil

import (
	"io/fs"
	"os"
	"path"
	"path/filepath"

	"github.com/gookit/goutil/arrutil"
	"github.com/gookit/goutil/internal/comfunc"
	"github.com/gookit/goutil/strutil"
)

// FilePathInDirs get full file path in dirs.
//
// Params:
//   - file: can be relative path, file name, full path.
//   - dirs: dir paths
func FilePathInDirs(file string, dirs ...string) string {
	file = comfunc.ExpandHome(file)
	if FileExists(file) {
		return file
	}

	for _, dirPath := range dirs {
		fPath := JoinSubPaths(dirPath, file)
		if FileExists(fPath) {
			return fPath
		}
	}
	return "" // not found
}

// FirstExists check multi paths and return first exists path.
func FirstExists(paths ...string) string {
	return MatchFirst(paths, PathExists, "")
}

// FirstExistsDir check multi paths and return first exists dir.
func FirstExistsDir(paths ...string) string {
	return MatchFirst(paths, IsDir, "")
}

// FirstExistsFile check multi paths and return first exists file.
func FirstExistsFile(paths ...string) string {
	return MatchFirst(paths, IsFile, "")
}

// MatchPaths given paths by custom mather func.
func MatchPaths(paths []string, matcher PathMatchFunc) []string {
	var ret []string
	for _, p := range paths {
		if matcher(p) {
			ret = append(ret, p)
		}
	}
	return ret
}

// MatchFirst filter paths by filter func and return first match path.
func MatchFirst(paths []string, matcher PathMatchFunc, defaultPath string) string {
	for _, p := range paths {
		if matcher(p) {
			return p
		}
	}
	return defaultPath
}

// SearchNameUp find file/dir name in dirPath or parent dirs,
// return the name of directory path
//
// Usage:
//
//	repoDir := fsutil.SearchNameUp("/path/to/dir", ".git")
func SearchNameUp(dirPath, name string) string {
	dir, _ := SearchNameUpx(dirPath, name)
	return dir
}

// SearchNameUpx find file/dir name in dirPath or parent dirs,
// return the name of directory path and dir is changed.
func SearchNameUpx(dirPath, name string) (string, bool) {
	var level int
	dirPath = ToAbsPath(dirPath)

	for {
		namePath := filepath.Join(dirPath, name)
		if PathExists(namePath) {
			return dirPath, level > 0
		}

		level++
		prevLn := len(dirPath)
		dirPath = filepath.Dir(dirPath)
		if prevLn == len(dirPath) {
			return "", false
		}
	}
}

// WalkDir walks the file tree rooted at root, calling fn for each file or
// directory in the tree, including root.
//
// TIP: will recursively find in sub dirs.
func WalkDir(dir string, fn fs.WalkDirFunc) error {
	return filepath.WalkDir(dir, fn)
}

// Glob find files by glob path pattern. alias of filepath.Glob()
// and support filter matched files by name.
//
// Usage:
//
//	files := fsutil.Glob("/path/to/dir/*.go")
func Glob(pattern string, fls ...NameMatchFunc) []string {
	files, _ := filepath.Glob(pattern)
	if len(fls) == 0 || len(files) == 0 {
		return files
	}

	var matched []string
	for _, file := range files {
		for _, fn := range fls {
			if fn(path.Base(file)) {
				matched = append(matched, file)
				break
			}
		}
	}
	return matched
}

// GlobWithFunc find files by glob path pattern, then handle matched file
//
// - TIP: will be not find in subdir.
func GlobWithFunc(pattern string, fn func(filePath string) error) (err error) {
	files, err := filepath.Glob(pattern)
	if err != nil {
		return err
	}

	for _, filePath := range files {
		err = fn(filePath)
		if err != nil {
			break
		}
	}
	return
}

type (
	// FilterFunc type for FindInDir
	//
	// - return False will skip handle the file.
	FilterFunc func(fPath string, ent fs.DirEntry) bool

	// HandleFunc type for FindInDir
	HandleFunc func(fPath string, ent fs.DirEntry) error
)

// OnlyFindDir on find
func OnlyFindDir(_ string, ent fs.DirEntry) bool {
	return ent.IsDir()
}

// OnlyFindFile on find
func OnlyFindFile(_ string, ent fs.DirEntry) bool {
	return !ent.IsDir()
}

// ExcludeNames on find
func ExcludeNames(names ...string) FilterFunc {
	return func(_ string, ent fs.DirEntry) bool {
		return !arrutil.StringsHas(names, ent.Name())
	}
}

// IncludeSuffix on find
func IncludeSuffix(ss ...string) FilterFunc {
	return func(_ string, ent fs.DirEntry) bool {
		return strutil.HasOneSuffix(ent.Name(), ss)
	}
}

// ExcludeDotFile on find
func ExcludeDotFile(_ string, ent fs.DirEntry) bool {
	return ent.Name()[0] != '.'
}

// ExcludeSuffix on find
func ExcludeSuffix(ss ...string) FilterFunc {
	return func(_ string, ent fs.DirEntry) bool {
		return !strutil.HasOneSuffix(ent.Name(), ss)
	}
}

// ApplyFilters handle
func ApplyFilters(fPath string, ent fs.DirEntry, filters []FilterFunc) bool {
	for _, filter := range filters {
		if !filter(fPath, ent) {
			return true
		}
	}
	return false
}

// FindInDir code refer the go pkg: path/filepath.glob()
//
// - TIP: will be not find in subdir.
//
// filters: return false will skip the file.
func FindInDir(dir string, handleFn HandleFunc, filters ...FilterFunc) (e error) {
	fi, err := os.Stat(dir)
	if err != nil || !fi.IsDir() {
		return // ignore I/O error
	}

	des, err := os.ReadDir(dir)
	if err != nil {
		return
	}

	// remove the last '/' char
	dirLn := len(dir)
	if dirLn > 1 && dir[dirLn-1] == '/' {
		dir = dir[:dirLn-1]
	}

	for _, ent := range des {
		filePath := dir + "/" + ent.Name()

		// apply filters
		if len(filters) > 0 && ApplyFilters(filePath, ent, filters) {
			continue
		}

		if err := handleFn(filePath, ent); err != nil {
			return err
		}
	}
	return nil
}
