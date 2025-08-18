package fsutil

import (
	"io/fs"

	"github.com/gookit/goutil/comdef"
	"github.com/gookit/goutil/strutil"
)

const (
	// MimeSniffLen sniff Length, use for detect file mime type
	MimeSniffLen = 512
)

// NameMatchFunc name matches func, alias of comdef.StringMatchFunc
type NameMatchFunc = comdef.StringMatchFunc

// PathMatchFunc path matches func. alias of comdef.StringMatchFunc
type PathMatchFunc = comdef.StringMatchFunc

// Entry extends fs.DirEntry, add some useful methods
type Entry interface {
	fs.DirEntry
	// Path gets file/dir full path. eg: "/path/to/file.go"
	Path() string
	// Info get file info. like fs.DirEntry.Info(), but will cache result.
	Info() (fs.FileInfo, error)
}

type entry struct {
	fs.DirEntry
	path string
	stat fs.FileInfo
	sErr error
}

// NewEntry create a new Entry instance
func NewEntry(fPath string, ent fs.DirEntry) Entry {
	return &entry{
		path:     fPath,
		DirEntry: ent,
	}
}

// Path gets full file/dir path. eg: "/path/to/file.go"
func (e *entry) Path() string {
	return e.path
}

// Info gets file info, will cache result
func (e *entry) Info() (fs.FileInfo, error) {
	if e.stat == nil {
		e.stat, e.sErr = e.DirEntry.Info()
	}
	return e.stat, e.sErr
}

// String get string representation
func (e *entry) String() string {
	return strutil.OrCond(e.IsDir(), "dir: ", "file: ") + e.Path()
}

// FileInfo extends fs.FileInfo, add some useful methods
type FileInfo interface {
	fs.FileInfo
	// Path gets file full path. eg: "/path/to/file.go"
	Path() string
}

type fileInfo struct {
	fs.FileInfo
	fullPath string
}

// NewFileInfo create a new FileInfo instance
func NewFileInfo(fPath string, info fs.FileInfo) FileInfo {
	return &fileInfo{
		fullPath: fPath,
		FileInfo: info,
	}
}

// Path gets file full path. eg: "/path/to/file.go"
func (fi *fileInfo) Path() string {
	return fi.fullPath
}

// FileInfos type for FileInfo slice
//
// implements sort.Interface:
//
//	sorts by oldest time modified in the file info.
//	eg: [old_220211, old_220212, old_220213]
type FileInfos []FileInfo

// Len get length
func (fis FileInfos) Len() int {
	return len(fis)
}

// Swap swap values
func (fis FileInfos) Swap(i, j int) {
	fis[i], fis[j] = fis[j], fis[i]
}

// Less check by mod time
func (fis FileInfos) Less(i, j int) bool {
	return fis[j].ModTime().After(fis[i].ModTime())
}
