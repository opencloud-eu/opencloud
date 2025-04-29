package fsutil

import (
	"os"
	"path/filepath"

	"github.com/gookit/goutil/internal/comfunc"
)

// DirPath get dir path from filepath, without last name.
func DirPath(fpath string) string { return filepath.Dir(fpath) }

// Dir get dir path from filepath, without last name.
func Dir(fpath string) string { return filepath.Dir(fpath) }

// PathName get file/dir name from full path
func PathName(fpath string) string { return filepath.Base(fpath) }

// Name get file/dir name from full path.
//
// eg: path/to/main.go => main.go
func Name(fpath string) string {
	if fpath == "" {
		return ""
	}
	return filepath.Base(fpath)
}

// FileExt get filename ext. alias of filepath.Ext()
//
// eg: path/to/main.go => ".go"
func FileExt(fpath string) string { return filepath.Ext(fpath) }

// Extname get filename ext. alias of filepath.Ext()
//
// eg: path/to/main.go => "go"
func Extname(fpath string) string {
	if ext := filepath.Ext(fpath); len(ext) > 0 {
		return ext[1:]
	}
	return ""
}

// Suffix get filename ext. alias of filepath.Ext()
//
// eg: path/to/main.go => ".go"
func Suffix(fpath string) string { return filepath.Ext(fpath) }

// Expand will parse first `~` as user home dir path.
func Expand(pathStr string) string {
	return comfunc.ExpandHome(pathStr)
}

// ExpandPath will parse `~` as user home dir path.
func ExpandPath(pathStr string) string {
	return comfunc.ExpandHome(pathStr)
}

// ResolvePath will parse `~` and env var in path
func ResolvePath(pathStr string) string {
	pathStr = comfunc.ExpandHome(pathStr)
	// return comfunc.ParseEnvVar()
	return os.ExpandEnv(pathStr)
}

// SplitPath splits path immediately following the final Separator, separating it into a directory and file name component
func SplitPath(pathStr string) (dir, name string) {
	return filepath.Split(pathStr)
}
