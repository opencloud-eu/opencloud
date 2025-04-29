package fsutil

import (
	"io"
	"os"

	"github.com/gookit/goutil/basefn"
)

// ************************************************************
//	temp file or dir
// ************************************************************

// OSTempFile create a temp file on os.TempDir()
//
// Usage:
//
//	fsutil.OSTempFile("example.*.txt")
func OSTempFile(pattern string) (*os.File, error) {
	return os.CreateTemp(os.TempDir(), pattern)
}

// TempFile is like os.CreateTemp, but can custom temp dir.
//
// Usage:
//
//	// create temp file on os.TempDir()
//	fsutil.TempFile("", "example.*.txt")
//	// create temp file on "testdata" dir
//	fsutil.TempFile("testdata", "example.*.txt")
func TempFile(dir, pattern string) (*os.File, error) {
	return os.CreateTemp(dir, pattern)
}

// OSTempDir creates a new temp dir on os.TempDir and return the temp dir path
//
// Usage:
//
//	fsutil.OSTempDir("example.*")
func OSTempDir(pattern string) (string, error) {
	return os.MkdirTemp(os.TempDir(), pattern)
}

// TempDir creates a new temp dir and return the temp dir path
//
// Usage:
//
//	fsutil.TempDir("", "example.*")
//	fsutil.TempDir("testdata", "example.*")
func TempDir(dir, pattern string) (string, error) {
	return os.MkdirTemp(dir, pattern)
}

// ************************************************************
//	write, copy files
// ************************************************************

// MustSave create file and write contents to file, panic on error.
//
//   - data type allow: string, []byte, io.Reader
//
// default option see NewOpenOption()
func MustSave(filePath string, data any, optFns ...OpenOptionFunc) {
	basefn.MustOK(SaveFile(filePath, data, optFns...))
}

// SaveFile create file and write contents to file. will auto create dir.
//
//   - data type allow: string, []byte, io.Reader
//
// default option see NewOpenOption()
func SaveFile(filePath string, data any, optFns ...OpenOptionFunc) error {
	opt := NewOpenOption(optFns...)
	return WriteFile(filePath, data, opt.Perm, opt.Flag)
}

// PutContents create file and write contents to file at once.
//
// data type allow: string, []byte, io.Reader. will auto create dir.
//
// Tip: file flag default is FsCWTFlags (override write)
//
// Usage:
//
//	fsutil.PutContents(filePath, contents, fsutil.FsCWAFlags) // append write
//	fsutil.Must2(fsutil.PutContents(filePath, contents)) // panic on error
func PutContents(filePath string, data any, fileFlag ...int) (int, error) {
	f, err := QuickOpenFile(filePath, basefn.FirstOr(fileFlag, FsCWTFlags))
	if err != nil {
		return 0, err
	}
	return WriteOSFile(f, data)
}

// WriteFile create file and write contents to file, can set perm for file.
//
// data type allow: string, []byte, io.Reader
//
// Tip: file flag default is FsCWTFlags (override write)
//
// Usage:
//
//	fsutil.WriteFile(filePath, contents, fsutil.DefaultFilePerm, fsutil.FsCWAFlags)
func WriteFile(filePath string, data any, perm os.FileMode, fileFlag ...int) error {
	flag := basefn.FirstOr(fileFlag, FsCWTFlags)
	f, err := OpenFile(filePath, flag, perm)
	if err != nil {
		return err
	}

	_, err = WriteOSFile(f, data)
	return err
}

// WriteOSFile write data to give os.File, then close file.
//
// data type allow: string, []byte, io.Reader
func WriteOSFile(f *os.File, data any) (n int, err error) {
	switch typData := data.(type) {
	case []byte:
		n, err = f.Write(typData)
	case string:
		n, err = f.WriteString(typData)
	case io.Reader: // eg: buffer
		var n64 int64
		n64, err = io.Copy(f, typData)
		n = int(n64)
	default:
		_ = f.Close()
		panic("WriteFile: data type only allow: []byte, string, io.Reader")
	}

	if err1 := f.Close(); err1 != nil && err == nil {
		err = err1
	}
	return n, err
}

// CopyFile copy a file to another file path.
func CopyFile(srcPath, dstPath string) error {
	srcFile, err := os.OpenFile(srcPath, FsRFlags, 0)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	// create and open file
	dstFile, err := QuickOpenFile(dstPath, FsCWTFlags)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, srcFile)
	return err
}

// MustCopyFile copy file to another path.
func MustCopyFile(srcPath, dstPath string) {
	err := CopyFile(srcPath, dstPath)
	if err != nil {
		panic(err)
	}
}

// UpdateContents read file contents, call handleFn(contents) handle, then write updated contents to file
func UpdateContents(filePath string, handleFn func(bs []byte) []byte) error {
	osFile, err := os.OpenFile(filePath, os.O_RDWR|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	defer osFile.Close()

	// read file contents
	if bs, err1 := io.ReadAll(osFile); err1 == nil {
		bs = handleFn(bs)
		_, err = osFile.Write(bs)
	} else {
		err = err1
	}
	return err
}
