package fsutil

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/gookit/goutil/internal/comfunc"
)

// DirPath get dir path from filepath, without a last name.
func DirPath(fPath string) string { return filepath.Dir(fPath) }

// Dir get dir path from filepath, without a last name.
func Dir(fPath string) string { return filepath.Dir(fPath) }

// PathName get file/dir name from a full path
func PathName(fPath string) string { return filepath.Base(fPath) }

// PathNoExt get path from full path, without ext.
//
// eg: path/to/main.go => "path/to/main"
func PathNoExt(fPath string) string {
	ext := filepath.Ext(fPath)
	if el := len(ext); el > 0 {
		return fPath[:len(fPath)-el]
	}
	return fPath
}

// Name get file/dir name from full path.
//
// eg: path/to/main.go => "main.go"
func Name(fPath string) string {
	if fPath == "" {
		return ""
	}
	return filepath.Base(fPath)
}

// NameNoExt get file name from a full path, without an ext.
//
// eg: path/to/main.go => "main"
func NameNoExt(fPath string) string {
	if fPath == "" {
		return ""
	}

	fName := filepath.Base(fPath)
	if pos := strings.LastIndexByte(fName, '.'); pos > 0 {
		return fName[:pos]
	}
	return fName
}

// FileExt get filename ext. alias of filepath.Ext()
//
// eg: path/to/main.go => ".go"
func FileExt(fPath string) string { return filepath.Ext(fPath) }

// Extname get filename ext. alias of filepath.Ext()
//
// eg: path/to/main.go => "go"
func Extname(fPath string) string {
	if ext := filepath.Ext(fPath); len(ext) > 0 {
		return ext[1:]
	}
	return ""
}

// Suffix get filename ext. alias of filepath.Ext()
//
// eg: path/to/main.go => ".go"
func Suffix(fPath string) string { return filepath.Ext(fPath) }

// Expand will parse first `~` to user home dir path.
func Expand(pathStr string) string {
	return comfunc.ExpandHome(pathStr)
}

// ExpandPath will parse `~` to user home dir path.
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
