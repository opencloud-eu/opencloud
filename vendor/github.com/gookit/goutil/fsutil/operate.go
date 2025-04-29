package fsutil

import (
	"archive/zip"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/gookit/goutil/basefn"
)

// Mkdir alias of os.MkdirAll()
func Mkdir(dirPath string, perm os.FileMode) error {
	return os.MkdirAll(dirPath, perm)
}

// MkDirs batch make multi dirs at once
func MkDirs(perm os.FileMode, dirPaths ...string) error {
	for _, dirPath := range dirPaths {
		if err := os.MkdirAll(dirPath, perm); err != nil {
			return err
		}
	}
	return nil
}

// MkSubDirs batch make multi sub-dirs at once
func MkSubDirs(perm os.FileMode, parentDir string, subDirs ...string) error {
	for _, dirName := range subDirs {
		dirPath := parentDir + "/" + dirName
		if err := os.MkdirAll(dirPath, perm); err != nil {
			return err
		}
	}
	return nil
}

// MkParentDir quick create parent dir for given path.
func MkParentDir(fpath string) error {
	dirPath := filepath.Dir(fpath)
	if !IsDir(dirPath) {
		return os.MkdirAll(dirPath, 0775)
	}
	return nil
}

// ************************************************************
//	options for open file
// ************************************************************

// OpenOption for open file
type OpenOption struct {
	// file open flag. see FsCWTFlags
	Flag int
	// file perm. see DefaultFilePerm
	Perm os.FileMode
}

// OpenOptionFunc for open/write file
type OpenOptionFunc func(*OpenOption)

// NewOpenOption create a new OpenOption instance
//
// Defaults:
//   - open flags: FsCWTFlags (override write)
//   - file Perm: DefaultFilePerm
func NewOpenOption(optFns ...OpenOptionFunc) *OpenOption {
	opt := &OpenOption{
		Flag: FsCWTFlags,
		Perm: DefaultFilePerm,
	}

	for _, fn := range optFns {
		fn(opt)
	}
	return opt
}

// OpenOptOrNew create a new OpenOption instance if opt is nil
func OpenOptOrNew(opt *OpenOption) *OpenOption {
	if opt == nil {
		return NewOpenOption()
	}
	return opt
}

// WithFlag set file open flag
func WithFlag(flag int) OpenOptionFunc {
	return func(opt *OpenOption) {
		opt.Flag = flag
	}
}

// WithPerm set file perm
func WithPerm(perm os.FileMode) OpenOptionFunc {
	return func(opt *OpenOption) {
		opt.Perm = perm
	}
}

// ************************************************************
//	open/create files
// ************************************************************

// some commonly flag consts for open file.
const (
	FsCWAFlags = os.O_CREATE | os.O_WRONLY | os.O_APPEND // create, append write-only
	FsCWTFlags = os.O_CREATE | os.O_WRONLY | os.O_TRUNC  // create, override write-only
	FsCWFlags  = os.O_CREATE | os.O_WRONLY               // create, write-only
	FsRWFlags  = os.O_RDWR                               // read-write, dont create.
	FsRFlags   = os.O_RDONLY                             // read-only
)

// OpenFile like os.OpenFile, but will auto create dir.
//
// Usage:
//
//	file, err := OpenFile("path/to/file.txt", FsCWFlags, 0666)
func OpenFile(filePath string, flag int, perm os.FileMode) (*os.File, error) {
	fileDir := filepath.Dir(filePath)
	if err := os.MkdirAll(fileDir, DefaultDirPerm); err != nil {
		return nil, err
	}

	file, err := os.OpenFile(filePath, flag, perm)
	if err != nil {
		return nil, err
	}
	return file, nil
}

// MustOpenFile like os.OpenFile, but will auto create dir.
//
// Usage:
//
//	file := MustOpenFile("path/to/file.txt", FsCWFlags, 0666)
func MustOpenFile(filePath string, flag int, perm os.FileMode) *os.File {
	file, err := OpenFile(filePath, flag, perm)
	if err != nil {
		panic(err)
	}
	return file
}

// QuickOpenFile like os.OpenFile, open for append write. if not exists, will create it.
//
// Alias of OpenAppendFile()
func QuickOpenFile(filepath string, fileFlag ...int) (*os.File, error) {
	flag := basefn.FirstOr(fileFlag, FsCWAFlags)
	return OpenFile(filepath, flag, DefaultFilePerm)
}

// OpenAppendFile like os.OpenFile, open for append write. if not exists, will create it.
func OpenAppendFile(filepath string, filePerm ...os.FileMode) (*os.File, error) {
	perm := basefn.FirstOr(filePerm, DefaultFilePerm)
	return OpenFile(filepath, FsCWAFlags, perm)
}

// OpenTruncFile like os.OpenFile, open for override write. if not exists, will create it.
func OpenTruncFile(filepath string, filePerm ...os.FileMode) (*os.File, error) {
	perm := basefn.FirstOr(filePerm, DefaultFilePerm)
	return OpenFile(filepath, FsCWTFlags, perm)
}

// OpenReadFile like os.OpenFile, open file for read contents
func OpenReadFile(filepath string) (*os.File, error) {
	return os.OpenFile(filepath, FsRFlags, OnlyReadFilePerm)
}

// CreateFile create file if not exists
//
// Usage:
//
//	CreateFile("path/to/file.txt", 0664, 0666)
func CreateFile(fpath string, filePerm, dirPerm os.FileMode, fileFlag ...int) (*os.File, error) {
	dirPath := filepath.Dir(fpath)
	if !IsDir(dirPath) {
		err := os.MkdirAll(dirPath, dirPerm)
		if err != nil {
			return nil, err
		}
	}

	flag := basefn.FirstOr(fileFlag, FsCWAFlags)
	return os.OpenFile(fpath, flag, filePerm)
}

// MustCreateFile create file, will panic on error
func MustCreateFile(filePath string, filePerm, dirPerm os.FileMode) *os.File {
	file, err := CreateFile(filePath, filePerm, dirPerm)
	if err != nil {
		panic(err)
	}
	return file
}

// ************************************************************
//	remove files
// ************************************************************

// alias methods
var (
	// MustRm removes the named file or (empty) directory.
	MustRm = MustRemove
	// QuietRm removes the named file or (empty) directory.
	QuietRm = QuietRemove
)

// Remove removes the named file or (empty) directory.
func Remove(fPath string) error {
	return os.Remove(fPath)
}

// MustRemove removes the named file or (empty) directory.
// NOTICE: will panic on error
func MustRemove(fPath string) {
	if err := os.Remove(fPath); err != nil {
		panic(err)
	}
}

// QuietRemove removes the named file or (empty) directory.
//
// NOTICE: will ignore error
func QuietRemove(fPath string) { _ = os.Remove(fPath) }

// SafeRemoveAll removes path and any children it contains. will ignore error
func SafeRemoveAll(path string) {
	_ = os.RemoveAll(path)
}

// RmIfExist removes the named file or (empty) directory on exists.
func RmIfExist(fPath string) error { return DeleteIfExist(fPath) }

// DeleteIfExist removes the named file or (empty) directory on exists.
func DeleteIfExist(fPath string) error {
	if PathExists(fPath) {
		return os.Remove(fPath)
	}
	return nil
}

// RmFileIfExist removes the named file on exists.
func RmFileIfExist(fPath string) error { return DeleteIfFileExist(fPath) }

// DeleteIfFileExist removes the named file on exists.
func DeleteIfFileExist(fPath string) error {
	if IsFile(fPath) {
		return os.Remove(fPath)
	}
	return nil
}

// RemoveSub removes all sub files and dirs of dirPath, but not remove dirPath.
func RemoveSub(dirPath string, fns ...FilterFunc) error {
	return FindInDir(dirPath, func(fPath string, ent fs.DirEntry) error {
		if ent.IsDir() {
			if err := RemoveSub(fPath, fns...); err != nil {
				return err
			}

			// skip rm not empty subdir
			if !IsEmptyDir(fPath) {
				return nil
			}
		}
		return os.Remove(fPath)
	}, fns...)
}

// ************************************************************
//	other operates
// ************************************************************

// Unzip a zip archive
// from https://blog.csdn.net/wangshubo1989/article/details/71743374
func Unzip(archive, targetDir string) (err error) {
	reader, err := zip.OpenReader(archive)
	if err != nil {
		return err
	}

	if err = os.MkdirAll(targetDir, DefaultDirPerm); err != nil {
		return
	}

	for _, file := range reader.File {
		if strings.Contains(file.Name, "..") {
			return fmt.Errorf("illegal file path in zip: %v", file.Name)
		}

		fullPath := filepath.Join(targetDir, file.Name)

		if file.FileInfo().IsDir() {
			err = os.MkdirAll(fullPath, file.Mode())
			if err != nil {
				return err
			}
			continue
		}

		fileReader, err := file.Open()
		if err != nil {
			return err
		}

		targetFile, err := os.OpenFile(fullPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, file.Mode())
		if err != nil {
			_ = fileReader.Close()
			return err
		}

		_, err = io.Copy(targetFile, fileReader)

		// close all
		_ = fileReader.Close()
		targetFile.Close()

		if err != nil {
			return err
		}
	}

	return
}
