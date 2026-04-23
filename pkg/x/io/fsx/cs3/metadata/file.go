package metadata

import (
	"bytes"
	"context"
	"io"
	"io/fs"
	"os"
)

type File struct {
	name     string
	fs       *Fs
	fileMode os.FileMode
	content  []byte
	resource io.ReadCloser
}

func newFile(name string, fs *Fs, fileMode os.FileMode, content []byte) (*File, error) {
	return &File{
		name:     name,
		fs:       fs,
		fileMode: fileMode,
		content:  content,
		resource: io.NopCloser(bytes.NewBuffer(content)),
	}, nil
}

func (f *File) Close() error {
	return f.resource.Close()
}

func (f *File) Read(p []byte) (n int, err error) {
	return f.resource.Read(p)
}

func (f *File) ReadAt(p []byte, off int64) (n int, err error) {
	readerAt, ok := f.resource.(io.ReaderAt)
	if !ok {
		return -1, &fs.PathError{Op: "ReadAt", Path: f.name, Err: ErrNotImplemented}
	}

	return readerAt.ReadAt(p, off)
}

func (f *File) Seek(offset int64, whence int) (int64, error) {
	seeker, ok := f.resource.(io.Seeker)
	if !ok {
		return -1, &fs.PathError{Op: "Seek", Path: f.name, Err: ErrNotImplemented}
	}

	return seeker.Seek(offset, whence)
}

func (f *File) Write(p []byte) (n int, err error) {
	return len(p), f.fs.storage.SimpleUpload(context.Background(), f.name, p)
}

func (f *File) WriteAt(_ []byte, _ int64) (n int, err error) {
	return -1, &fs.PathError{Op: "Write", Path: f.name, Err: ErrNotImplemented}
}

func (f *File) Name() string {
	return f.name
}

func (f *File) Readdir(_ int) ([]os.FileInfo, error) {
	return nil, &fs.PathError{Op: "Readdir", Path: f.name, Err: ErrNotImplemented}
}

func (f *File) Readdirnames(_ int) ([]string, error) {
	return nil, &fs.PathError{Op: "Readdirnames", Path: f.name, Err: ErrNotImplemented}
}

func (f *File) Sync() error {
	return nil
}

func (f *File) Truncate(_ int64) error {
	return &fs.PathError{Op: "Truncate", Path: f.name, Err: ErrNotImplemented}
}

func (f *File) WriteString(_ string) (ret int, err error) {
	return -1, &fs.PathError{Op: "WriteString", Path: f.name, Err: ErrNotImplemented}
}

func (f *File) Stat() (os.FileInfo, error) {
	return newFileInfo(f.name, f.fs, f.fileMode)
}
