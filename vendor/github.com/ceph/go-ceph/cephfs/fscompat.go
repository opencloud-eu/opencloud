//go:build ceph_preview

package cephfs

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"strings"
	"time"

	"github.com/ceph/go-ceph/internal/log"
)

var (
	errIsDir = errors.New("is a directory")
)

// MountWrapper provides a wrapper type that adapts a CephFS Mount into a
// io.FS compatible type.
type MountWrapper struct {
	mount       *MountInfo
	enableTrace bool
}

type fileWrapper struct {
	parent *MountWrapper
	file   *File
	name   string
}

type dirWrapper struct {
	parent    *MountWrapper
	directory *Directory
	name      string
}

type dentryWrapper struct {
	parent *MountWrapper
	de     *DirEntryPlus
}

type infoWrapper struct {
	parent *MountWrapper
	sx     *CephStatx
	name   string
}

// Wrap a CephFS Mount object into a new type that is compatible with Go's io.FS
// interface. CephFS Mounts are not compatible with io.FS directly because the
// go-ceph library predates the addition of io.FS to Go as well as the fact that
// go-ceph attempts to provide APIs that match the cephfs libraries first and
// foremost.
func Wrap(mount *MountInfo) *MountWrapper {
	wm := &MountWrapper{mount: mount}
	debugf(wm, "Wrap", "created")
	return wm
}

/* MountWrapper:
** Implements https://pkg.go.dev/io/fs#FS
** Wraps cephfs.MountInfo
 */

// SetTracing configures the MountWrapper and objects connected to it for debug
// tracing. True enables tracing and false disables it. A debug logging
// function must also be set using go-ceph's common.log.SetDebugf function.
func (mw *MountWrapper) SetTracing(enable bool) {
	mw.enableTrace = enable
}

// identify the MountWrapper object for logging purposes.
func (mw *MountWrapper) identify() string {
	return fmt.Sprintf("MountWrapper<%p>", mw)
}

// trace returns true if debug tracing is enabled.
func (mw *MountWrapper) trace() bool {
	return mw.enableTrace
}

// Open opens the named file. This may be either a regular file or a directory.
// Directories opened with this function will return object compatible with the
// io.ReadDirFile interface.
func (mw *MountWrapper) Open(name string) (fs.File, error) {
	debugf(mw, "Open", "(%v)", name)
	// there are a bunch of patterns that fsTetster/testfs looks for that seems
	// under-documented. They mainly seem to try and enforce "clean" paths.
	// look for them and reject them here because ceph libs won't reject on
	// its own
	if strings.HasPrefix(name, "/") ||
		strings.HasSuffix(name, "/.") ||
		strings.Contains(name, "//") ||
		strings.Contains(name, "/./") ||
		strings.Contains(name, "/../") {
		return nil, &fs.PathError{Op: "open", Path: name, Err: errInvalid}
	}

	d, err := mw.mount.OpenDir(name)
	if err == nil {
		debugf(mw, "Open", "(%v): dir ok", name)
		dw := &dirWrapper{parent: mw, directory: d, name: name}
		return dw, nil
	}
	if !errors.Is(err, errNotDir) {
		debugf(mw, "Open", "(%v): dir error: %v", name, err)
		return nil, &fs.PathError{Op: "open", Path: name, Err: err}
	}

	f, err := mw.mount.Open(name, os.O_RDONLY, 0)
	if err == nil {
		debugf(mw, "Open", "(%v): file ok", name)
		fw := &fileWrapper{parent: mw, file: f, name: name}
		return fw, nil
	}
	debugf(mw, "Open", "(%v): file error: %v", name, err)
	return nil, &fs.PathError{Op: "open", Path: name, Err: err}
}

/* fileWrapper:
** Implements https://pkg.go.dev/io/fs#FS
** Wraps cephfs.File
 */

func (fw *fileWrapper) Stat() (fs.FileInfo, error) {
	debugf(fw, "Stat", "()")
	sx, err := fw.file.Fstatx(StatxBasicStats, AtSymlinkNofollow)
	if err != nil {
		debugf(fw, "Stat", "() -> err:%v", err)
		return nil, &fs.PathError{Op: "stat", Path: fw.name, Err: err}
	}
	debugf(fw, "Stat", "() ok")
	return &infoWrapper{fw.parent, sx, path.Base(fw.name)}, nil
}

func (fw *fileWrapper) Read(b []byte) (int, error) {
	debugf(fw, "Read", "(...)")
	return fw.file.Read(b)
}

func (fw *fileWrapper) Close() error {
	debugf(fw, "Close", "()")
	return fw.file.Close()
}

func (fw *fileWrapper) identify() string {
	return fmt.Sprintf("fileWrapper<%p>[%v]", fw, fw.name)
}

func (fw *fileWrapper) trace() bool {
	return fw.parent.trace()
}

/* dirWrapper:
** Implements https://pkg.go.dev/io/fs#ReadDirFile
** Wraps cephfs.Directory
 */

func (dw *dirWrapper) Stat() (fs.FileInfo, error) {
	debugf(dw, "Stat", "()")
	sx, err := dw.parent.mount.Statx(dw.name, StatxBasicStats, AtSymlinkNofollow)
	if err != nil {
		debugf(dw, "Stat", "() -> err:%v", err)
		return nil, &fs.PathError{Op: "stat", Path: dw.name, Err: err}
	}
	debugf(dw, "Stat", "() ok")
	return &infoWrapper{dw.parent, sx, path.Base(dw.name)}, nil
}

func (dw *dirWrapper) Read(_ []byte) (int, error) {
	debugf(dw, "Read", "(...)")
	return 0, &fs.PathError{Op: "read", Path: dw.name, Err: errIsDir}
}

func (dw *dirWrapper) ReadDir(n int) ([]fs.DirEntry, error) {
	debugf(dw, "ReadDir", "(%v)", n)
	if n > 0 {
		return dw.readDirSome(n)
	}
	return dw.readDirAll()
}

const defaultDirReadCount = 256 // how many entries to read per loop

func (dw *dirWrapper) readDirAll() ([]fs.DirEntry, error) {
	debugf(dw, "readDirAll", "()")
	var (
		err     error
		egroup  []fs.DirEntry
		entries = make([]fs.DirEntry, 0)
		size    = defaultDirReadCount
	)
	for {
		egroup, err = dw.readDirSome(size)
		entries = append(entries, egroup...)
		if err == io.EOF {
			err = nil
			break
		}
		if err != nil {
			break
		}
	}
	debugf(dw, "readDirAll", "() -> len:%v, err:%v", len(entries), err)
	return entries, err
}

func (dw *dirWrapper) readDirSome(n int) ([]fs.DirEntry, error) {
	debugf(dw, "readDirSome", "(%v)", n)
	var (
		idx     int
		err     error
		entry   *DirEntryPlus
		entries = make([]fs.DirEntry, n)
	)
	for {
		entry, err = dw.directory.ReadDirPlus(StatxBasicStats, AtSymlinkNofollow)
		debugf(dw, "readDirSome", "(%v): got entry:%v, err:%v", n, entry, err)
		if err != nil || entry == nil {
			break
		}
		switch entry.Name() {
		case ".", "..":
			continue
		}
		entries[idx] = &dentryWrapper{dw.parent, entry}
		idx++
		if idx >= n {
			break
		}
	}
	if idx == 0 {
		debugf(dw, "readDirSome", "(%v): EOF", n)
		return nil, io.EOF
	}
	debugf(dw, "readDirSome", "(%v): got entry:%v, err:%v", n, entries[:idx], err)
	return entries[:idx], err
}

func (dw *dirWrapper) Close() error {
	debugf(dw, "Close", "()")
	return dw.directory.Close()
}

func (dw *dirWrapper) identify() string {
	return fmt.Sprintf("dirWrapper<%p>[%v]", dw, dw.name)
}

func (dw *dirWrapper) trace() bool {
	return dw.parent.trace()
}

/* dentryWrapper:
** Implements https://pkg.go.dev/io/fs#DirEntry
** Wraps cephfs.DirEntryPlus
 */

func (dew *dentryWrapper) Name() string {
	debugf(dew, "Name", "()")
	return dew.de.Name()
}

func (dew *dentryWrapper) IsDir() bool {
	v := dew.de.DType() == DTypeDir
	debugf(dew, "IsDir", "() -> %v", v)
	return v
}

func (dew *dentryWrapper) Type() fs.FileMode {
	m := dew.de.Statx().Mode
	v := cephModeToFileMode(m).Type()
	debugf(dew, "Type", "() -> %v", v)
	return v
}

func (dew *dentryWrapper) Info() (fs.FileInfo, error) {
	debugf(dew, "Info", "()")
	sx := dew.de.Statx()
	name := dew.de.Name()
	return &infoWrapper{dew.parent, sx, name}, nil
}

func (dew *dentryWrapper) identify() string {
	return fmt.Sprintf("dentryWrapper<%p>[%v]", dew, dew.de.Name())
}

func (dew *dentryWrapper) trace() bool {
	return dew.parent.trace()
}

/* infoWrapper:
** Implements https://pkg.go.dev/io/fs#FileInfo
** Wraps cephfs.CephStatx
 */

func (iw *infoWrapper) Name() string {
	debugf(iw, "Name", "()")
	return iw.name
}

func (iw *infoWrapper) Size() int64 {
	debugf(iw, "Size", "() -> %v", iw.sx.Size)
	return int64(iw.sx.Size)
}

func (iw *infoWrapper) Sys() any {
	debugf(iw, "Sys", "()")
	return iw.sx
}

func (iw *infoWrapper) Mode() fs.FileMode {
	v := cephModeToFileMode(iw.sx.Mode)
	debugf(iw, "Mode", "() -> %#o -> %#o/%v", iw.sx.Mode, uint32(v), v.Type())
	return v
}

func (iw *infoWrapper) IsDir() bool {
	v := iw.sx.Mode&modeIFMT == modeIFDIR
	debugf(iw, "IsDir", "() -> %v", v)
	return v
}

func (iw *infoWrapper) ModTime() time.Time {
	v := time.Unix(iw.sx.Mtime.Sec, iw.sx.Mtime.Nsec)
	debugf(iw, "ModTime", "() -> %v", v)
	return v
}

func (iw *infoWrapper) identify() string {
	return fmt.Sprintf("infoWrapper<%p>[%v]", iw, iw.name)
}

func (iw *infoWrapper) trace() bool {
	return iw.parent.trace()
}

/* copy and paste values from the linux headers. We always need to use
** the linux header values, regardless of the platform go-ceph is built
** for. Rather than jumping through header hoops, copy and paste is
** more consistent and reliable.
 */
const (
	/* file type mask */
	modeIFMT = uint16(0170000)
	/* file types */
	modeIFDIR  = uint16(0040000)
	modeIFCHR  = uint16(0020000)
	modeIFBLK  = uint16(0060000)
	modeIFREG  = uint16(0100000)
	modeIFIFO  = uint16(0010000)
	modeIFLNK  = uint16(0120000)
	modeIFSOCK = uint16(0140000)
	/* protection bits */
	modeISUID = uint16(0004000)
	modeISGID = uint16(0002000)
	modeISVTX = uint16(0001000)
)

// cephModeToFileMode takes a linux compatible cephfs mode value
// and returns a Go-compatiable os-agnostic FileMode value.
func cephModeToFileMode(m uint16) fs.FileMode {
	// start with permission bits
	mode := fs.FileMode(m & 0777)
	// file type - inspired by go's src/os/stat_linux.go
	switch m & modeIFMT {
	case modeIFBLK:
		mode |= fs.ModeDevice
	case modeIFCHR:
		mode |= fs.ModeDevice | fs.ModeCharDevice
	case modeIFDIR:
		mode |= fs.ModeDir
	case modeIFIFO:
		mode |= fs.ModeNamedPipe
	case modeIFLNK:
		mode |= fs.ModeSymlink
	case modeIFREG:
		// nothing to do
	case modeIFSOCK:
		mode |= fs.ModeSocket
	}
	// protection bits
	if m&modeISUID != 0 {
		mode |= fs.ModeSetuid
	}
	if m&modeISGID != 0 {
		mode |= fs.ModeSetgid
	}
	if m&modeISVTX != 0 {
		mode |= fs.ModeSticky
	}
	return mode
}

// wrapperObject helps identify an object to be logged.
type wrapperObject interface {
	identify() string
	trace() bool
}

// debugf formats info about a function and logs it.
func debugf(o wrapperObject, fname, format string, args ...any) {
	if o.trace() {
		log.Debugf(fmt.Sprintf("%v.%v: %s", o.identify(), fname, format), args...)
	}
}
