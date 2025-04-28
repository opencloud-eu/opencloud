package cephfs

/*
#cgo LDFLAGS: -lcephfs
#cgo CPPFLAGS: -D_FILE_OFFSET_BITS=64
#define _GNU_SOURCE
#include <stdlib.h>
#include <fcntl.h>
#include <cephfs/libcephfs.h>
*/
import "C"

import (
	"io"
	"unsafe"

	"github.com/ceph/go-ceph/internal/cutil"
)

const (
	// SeekSet is used with Seek to set the absolute position in the file.
	SeekSet = int(C.SEEK_SET)
	// SeekCur is used with Seek to position the file relative to the current
	// position.
	SeekCur = int(C.SEEK_CUR)
	// SeekEnd is used with Seek to position the file relative to the end.
	SeekEnd = int(C.SEEK_END)
)

// SyncChoice is used to control how metadata and/or data is sync'ed to
// the file system.
type SyncChoice int

const (
	// SyncAll will synchronize both data and metadata.
	SyncAll = SyncChoice(0)
	// SyncDataOnly will synchronize only data.
	SyncDataOnly = SyncChoice(1)
)

// File represents an open file descriptor in cephfs.
type File struct {
	mount *MountInfo
	fd    C.int
}

// Open a file at the given path. The flags are the same os flags as
// a local open call. Mode is the same mode bits as a local open call.
//
// Implements:
//
//	int ceph_open(struct ceph_mount_info *cmount, const char *path, int flags, mode_t mode);
func (mount *MountInfo) Open(path string, flags int, mode uint32) (*File, error) {
	if mount.mount == nil {
		return nil, ErrNotConnected
	}
	cPath := C.CString(path)
	defer C.free(unsafe.Pointer(cPath))
	ret := C.ceph_open(mount.mount, cPath, C.int(flags), C.mode_t(mode))
	if ret < 0 {
		return nil, getError(ret)
	}
	return &File{mount: mount, fd: ret}, nil
}

func (f *File) validate() error {
	if f.mount == nil {
		return ErrNotConnected
	}
	return nil
}

// Close the file.
//
// Implements:
//
//	int ceph_close(struct ceph_mount_info *cmount, int fd);
func (f *File) Close() error {
	if f.fd == -1 {
		// already closed
		return nil
	}
	if err := f.validate(); err != nil {
		return err
	}
	if err := getError(C.ceph_close(f.mount.mount, f.fd)); err != nil {
		return err
	}
	f.fd = -1
	return nil
}

// read directly wraps the ceph_read call. Because read is such a common
// operation we deviate from the ceph naming and expose Read and ReadAt
// wrappers for external callers of the library.
//
// Implements:
//
//	int ceph_read(struct ceph_mount_info *cmount, int fd, char *buf, int64_t size, int64_t offset);
func (f *File) read(buf []byte, offset int64) (int, error) {
	if err := f.validate(); err != nil {
		return 0, err
	}
	if len(buf) == 0 {
		return 0, nil
	}
	bufptr := (*C.char)(unsafe.Pointer(&buf[0]))
	ret := C.ceph_read(
		f.mount.mount, f.fd, bufptr, C.int64_t(len(buf)), C.int64_t(offset))
	switch {
	case ret < 0:
		return 0, getError(ret)
	case ret == 0:
		return 0, io.EOF
	}
	return int(ret), nil
}

// Read data from file. Up to len(buf) bytes will be read from the file.
// The number of bytes read will be returned.
// When nothing is left to read from the file, Read returns, 0, io.EOF.
func (f *File) Read(buf []byte) (int, error) {
	// to-consider: should we mimic Go's behavior of returning an
	// io.ErrShortWrite error if write length < buf size?
	return f.read(buf, -1)
}

// ReadAt will read data from the file starting at the given offset.
// Up to len(buf) bytes will be read from the file.
// The number of bytes read will be returned.
// When nothing is left to read from the file, ReadAt returns, 0, io.EOF.
func (f *File) ReadAt(buf []byte, offset int64) (int, error) {
	if offset < 0 {
		return 0, errInvalid
	}
	return f.read(buf, offset)
}

// Preadv will read data from the file, starting at the given offset,
// into the byte-slice data buffers sequentially.
// The number of bytes read will be returned.
// When nothing is left to read from the file the return values will be:
// 0, io.EOF.
//
// Implements:
//
//	int ceph_preadv(struct ceph_mount_info *cmount, int fd, const struct iovec *iov, int iovcnt,
//	                int64_t offset);
func (f *File) Preadv(data [][]byte, offset int64) (int, error) {
	if err := f.validate(); err != nil {
		return 0, err
	}
	iov := cutil.ByteSlicesToIovec(data)
	defer iov.Free()

	ret := C.ceph_preadv(
		f.mount.mount,
		f.fd,
		(*C.struct_iovec)(iov.Pointer()),
		C.int(iov.Len()),
		C.int64_t(offset))
	switch {
	case ret < 0:
		return 0, getError(ret)
	case ret == 0:
		return 0, io.EOF
	}
	iov.Sync()
	return int(ret), nil
}

// write directly wraps the ceph_write call. Because write is such a common
// operation we deviate from the ceph naming and expose Write and WriteAt
// wrappers for external callers of the library.
//
// Implements:
//
//	int ceph_write(struct ceph_mount_info *cmount, int fd, const char *buf,
//	               int64_t size, int64_t offset);
func (f *File) write(buf []byte, offset int64) (int, error) {
	if err := f.validate(); err != nil {
		return 0, err
	}
	if len(buf) == 0 {
		return 0, nil
	}
	bufptr := (*C.char)(unsafe.Pointer(&buf[0]))
	ret := C.ceph_write(
		f.mount.mount, f.fd, bufptr, C.int64_t(len(buf)), C.int64_t(offset))
	if ret < 0 {
		return 0, getError(ret)
	}
	return int(ret), nil
}

// Write data from buf to the file.
// The number of bytes written is returned.
func (f *File) Write(buf []byte) (int, error) {
	return f.write(buf, -1)
}

// WriteAt writes data from buf to the file at the specified offset.
// The number of bytes written is returned.
func (f *File) WriteAt(buf []byte, offset int64) (int, error) {
	if offset < 0 {
		return 0, errInvalid
	}
	return f.write(buf, offset)
}

// Pwritev writes data from the slice of byte-slice buffers to the file at the
// specified offset.
// The number of bytes written is returned.
//
// Implements:
//
//	int ceph_pwritev(struct ceph_mount_info *cmount, int fd, const struct iovec *iov, int iovcnt,
//	                 int64_t offset);
func (f *File) Pwritev(data [][]byte, offset int64) (int, error) {
	if err := f.validate(); err != nil {
		return 0, err
	}
	iov := cutil.ByteSlicesToIovec(data)
	defer iov.Free()

	ret := C.ceph_pwritev(
		f.mount.mount,
		f.fd,
		(*C.struct_iovec)(iov.Pointer()),
		C.int(iov.Len()),
		C.int64_t(offset))
	if ret < 0 {
		return 0, getError(ret)
	}
	return int(ret), nil
}

// Seek will reposition the file stream based on the given offset.
//
// Implements:
//
//	int64_t ceph_lseek(struct ceph_mount_info *cmount, int fd, int64_t offset, int whence);
func (f *File) Seek(offset int64, whence int) (int64, error) {
	if err := f.validate(); err != nil {
		return 0, err
	}
	// validate the seek whence value in case the caller skews
	// from the seek values we technically support from C as documented.
	// TODO: need to support seek-(hole|data) in mimic and later.
	switch whence {
	case SeekSet, SeekCur, SeekEnd:
	default:
		return 0, errInvalid
	}

	ret := C.ceph_lseek(f.mount.mount, f.fd, C.int64_t(offset), C.int(whence))
	if ret < 0 {
		return 0, getError(C.int(ret))
	}
	return int64(ret), nil
}

// Fchmod changes the mode bits (permissions) of a file.
//
// Implements:
//
//	int ceph_fchmod(struct ceph_mount_info *cmount, int fd, mode_t mode);
func (f *File) Fchmod(mode uint32) error {
	if err := f.validate(); err != nil {
		return err
	}

	ret := C.ceph_fchmod(f.mount.mount, f.fd, C.mode_t(mode))
	return getError(ret)
}

// Fchown changes the ownership of a file.
//
// Implements:
//
//	int ceph_fchown(struct ceph_mount_info *cmount, int fd, int uid, int gid);
func (f *File) Fchown(user uint32, group uint32) error {
	if err := f.validate(); err != nil {
		return err
	}

	ret := C.ceph_fchown(f.mount.mount, f.fd, C.int(user), C.int(group))
	return getError(ret)
}

// Fstatx returns information about an open file.
//
// Implements:
//
//	int ceph_fstatx(struct ceph_mount_info *cmount, int fd, struct ceph_statx *stx,
//	                unsigned int want, unsigned int flags);
func (f *File) Fstatx(want StatxMask, flags AtFlags) (*CephStatx, error) {
	if err := f.validate(); err != nil {
		return nil, err
	}

	var stx C.struct_ceph_statx
	ret := C.ceph_fstatx(
		f.mount.mount,
		f.fd,
		&stx,
		C.uint(want),
		C.uint(flags),
	)
	if err := getError(ret); err != nil {
		return nil, err
	}
	return cStructToCephStatx(stx), nil
}

// FallocFlags represent flags which determine the operation to be
// performed on the given range.
// CephFS supports only following two flags.
type FallocFlags int

const (
	// FallocNoFlag means default option.
	FallocNoFlag = FallocFlags(0)
	// FallocFlKeepSize specifies that the file size will not be changed.
	FallocFlKeepSize = FallocFlags(C.FALLOC_FL_KEEP_SIZE)
	// FallocFlPunchHole specifies that the operation is to deallocate
	// space and zero the byte range.
	FallocFlPunchHole = FallocFlags(C.FALLOC_FL_PUNCH_HOLE)
)

// Fallocate preallocates or releases disk space for the file for the
// given byte range, the flags determine the operation to be performed
// on the given range.
//
// Implements:
//
//	int ceph_fallocate(struct ceph_mount_info *cmount, int fd, int mode,
//								  int64_t offset, int64_t length);
func (f *File) Fallocate(mode FallocFlags, offset, length int64) error {
	if err := f.validate(); err != nil {
		return err
	}
	ret := C.ceph_fallocate(f.mount.mount, f.fd, C.int(mode), C.int64_t(offset), C.int64_t(length))
	return getError(ret)
}

// LockOp determines operations/type of locks which can be applied on a file.
type LockOp int

const (
	// LockSH places a shared lock.
	// More than one process may hold a shared lock for a given file at a given time.
	LockSH = LockOp(C.LOCK_SH)
	// LockEX places an exclusive lock.
	// Only one process may hold an exclusive lock for a given file at a given time.
	LockEX = LockOp(C.LOCK_EX)
	// LockUN removes an existing lock held by this process.
	LockUN = LockOp(C.LOCK_UN)
	// LockNB can be ORed with any of the above to make a nonblocking call.
	LockNB = LockOp(C.LOCK_NB)
)

// Flock applies or removes an advisory lock on an open file.
// Param owner is the user-supplied identifier for the owner of the
// lock, must be an arbitrary integer.
//
// Implements:
//
//	int ceph_flock(struct ceph_mount_info *cmount, int fd, int operation, uint64_t owner);
func (f *File) Flock(operation LockOp, owner uint64) error {
	if err := f.validate(); err != nil {
		return err
	}

	// validate the operation values before passing it on.
	switch operation &^ LockNB {
	case LockSH, LockEX, LockUN:
	default:
		return errInvalid
	}

	ret := C.ceph_flock(f.mount.mount, f.fd, C.int(operation), C.uint64_t(owner))
	return getError(ret)
}

// Fsync ensures the file content that may be cached is committed to stable
// storage.
// Pass SyncAll to have this call behave like standard fsync and synchronize
// all data and metadata.
// Pass SyncDataOnly to have this call behave more like fdatasync (on linux).
//
// Implements:
//
//	int ceph_fsync(struct ceph_mount_info *cmount, int fd, int syncdataonly);
func (f *File) Fsync(sync SyncChoice) error {
	if err := f.validate(); err != nil {
		return err
	}

	ret := C.ceph_fsync(
		f.mount.mount,
		f.fd,
		C.int(sync),
	)
	return getError(ret)
}

// Sync ensures the file content that may be cached is committed to stable
// storage.
// Sync behaves like Go's os package File.Sync function.
func (f *File) Sync() error {
	return f.Fsync(SyncAll)
}

// Truncate sets the size of the open file.
// NOTE: In some versions of ceph a bug exists where calling ftruncate on a
// file open for read-only is permitted. The go-ceph wrapper does no additional
// checking and will inherit the issue on affected versions of ceph.  Please
// refer to the following issue for details:
// https://tracker.ceph.com/issues/48202
//
// Implements:
//
//	int ceph_ftruncate(struct ceph_mount_info *cmount, int fd, int64_t size);
func (f *File) Truncate(size int64) error {
	if err := f.validate(); err != nil {
		return err
	}

	ret := C.ceph_ftruncate(
		f.mount.mount,
		f.fd,
		C.int64_t(size),
	)
	return getError(ret)
}
