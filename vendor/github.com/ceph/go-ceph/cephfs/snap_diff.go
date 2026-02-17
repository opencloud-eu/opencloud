//go:build ceph_preview

package cephfs

/*
#cgo LDFLAGS: -lcephfs
#cgo CPPFLAGS: -D_FILE_OFFSET_BITS=64
#include <stdlib.h>
#include <dirent.h>
#include <cephfs/libcephfs.h>

// Types and constants are copied from libcephfs.h with added "_" as prefix. This
// prevents redefinition of the types on libcephfs versions that have them
// already.

typedef struct {
  struct dirent dir_entry;
  uint64_t snapid;
} _ceph_snapdiff_entry_t;

typedef struct {
  struct ceph_mount_info* cmount;
  struct ceph_dir_result* dir1;
  struct ceph_dir_result* dir_aux;
} _ceph_snapdiff_info;

// open_snapdiff_fn matches the open_snapdiff function signature.
typedef int(*open_snapdiff_fn)(struct ceph_mount_info* cmount,
                                  const char* root_path,
                                  const char* rel_path,
                                  const char* snap1,
                                  const char* snap2,
                                  _ceph_snapdiff_info* out);

// open_snapdiff_dlsym take *fn as open_snapdiff_fn and calls the dynamically loaded
// open_snapdiff function passed as 1st argument.
static inline int open_snapdiff_dlsym(void *fn,
                                  struct ceph_mount_info* cmount,
                                  const char* root_path,
                                  const char* rel_path,
                                  const char* snap1,
                                  const char* snap2,
                                  _ceph_snapdiff_info* out) {
	// cast function pointer fn to open_snapdiff and call the function
	return ((open_snapdiff_fn) fn)(cmount, root_path, rel_path, snap1, snap2, out);
}

// readdir_snapdiff_fn matches the readdir_snapdiff function signature.
typedef int(*readdir_snapdiff_fn)(_ceph_snapdiff_info* snapdiff,
                                  _ceph_snapdiff_entry_t* out);

// readdir_snapdiff_dlsym take *fn as readdir_snapdiff_fn and calls the dynamically loaded
// readdir_snapdiff function passed as 1st argument.
static inline int readdir_snapdiff_dlsym(void *fn,
                                  _ceph_snapdiff_info* snapdiff,
                                  _ceph_snapdiff_entry_t* out) {
	// cast function pointer fn to readdir_snapdiff and call the function
	return ((readdir_snapdiff_fn) fn)(snapdiff, out);
}

// close_snapdiff_fn matches the close_snapdiff function signature.
typedef int(*close_snapdiff_fn)(_ceph_snapdiff_info* snapdiff);

// close_snapdiff_dlsym take *fn as close_snapdiff_fn and calls the dynamically loaded
// close_snapdiff function passed as 1st argument.
static inline int close_snapdiff_dlsym(void *fn,
                                  _ceph_snapdiff_info* snapdiff) {
	// cast function pointer fn to close_snapdiff and call the function
	return ((close_snapdiff_fn) fn)(snapdiff);
}
*/
import "C"

import (
	"fmt"
	"sync"
	"unsafe"

	"github.com/ceph/go-ceph/internal/dlsym"
)

var (
	cephOpenSnapDiffOnce    sync.Once
	cephReaddirSnapDiffOnce sync.Once
	cephCloseSnapDiffOnce   sync.Once
	cephOpenSnapDiff        unsafe.Pointer
	cephReaddirSnapDiff     unsafe.Pointer
	cephCloseSnapDiff       unsafe.Pointer
	cephOpenSnapDiffErr     error
	cephReaddirSnapDiffErr  error
	cephCloseSnapDiffErr    error
)

// SnapDiffInfo is a handle to a snapshot diff API.
type SnapDiffInfo struct {
	cMount *MountInfo
	dir1   *Directory
	dirAux *Directory
}

// SnapDiffEntry is a single entry in the snapshot diff.
// It contains a DirEntry and the ID of the snapshot to
// which the directory belongs.
type SnapDiffEntry struct {
	DirEntry *DirEntry
	SnapID   uint64
}

// SnapDiffConfig is used to define the parameters of a open_snapdiff call.
// Snapshot Delta is generated between the passed snapshots snap1 and snap2.
// All fields must be specified before passing to OpenSnapDiff().
type SnapDiffConfig struct {
	// CMount is the ceph mount handle that will be used for snap.diff retrieval.
	CMount *MountInfo
	// RootPath represents the root path for snapshots-in-question.
	RootPath string
	// RelPath is the subpath under the root to build delta for.
	RelPath string
	// Snap1 is the first snapshot name.
	Snap1 string
	// Snap2 is the second snapshot name.
	Snap2 string
}

// OpenSnapDiff opens a snapshot diff stream to get snapshots delta
// and returns a SnapDiffInfo struct containing the diff information.
//
// Implements:
//
//	int ceph_open_snapdiff(struct ceph_mount_info* cmount,
//	                       const char* root_path,
//	                       const char* rel_path,
//	                       const char* snap1,
//	                       const char* snap2,
//	                       struct ceph_snapdiff_info* out);
func OpenSnapDiff(config SnapDiffConfig) (*SnapDiffInfo, error) {
	if config.CMount == nil || config.RootPath == "" || config.RelPath == "" ||
		config.Snap1 == "" || config.Snap2 == "" {
		return nil, errInvalid
	}

	cephOpenSnapDiffOnce.Do(func() {
		cephOpenSnapDiff, cephOpenSnapDiffErr = dlsym.LookupSymbol("ceph_open_snapdiff")
	})

	if cephOpenSnapDiffErr != nil {
		return nil, fmt.Errorf("%w: %w", ErrNotImplemented, cephOpenSnapDiffErr)
	}

	rawCephSnapDiffInfo := &C._ceph_snapdiff_info{}

	ret := C.open_snapdiff_dlsym(
		cephOpenSnapDiff,
		config.CMount.mount,
		C.CString(config.RootPath),
		C.CString(config.RelPath),
		C.CString(config.Snap1),
		C.CString(config.Snap2),
		rawCephSnapDiffInfo)

	if ret != 0 {
		return nil, getError(ret)
	}

	mountInfo := &MountInfo{
		mount: rawCephSnapDiffInfo.cmount,
	}
	cephSnapDiffInfo := &SnapDiffInfo{
		cMount: mountInfo,
		dir1: &Directory{
			mount: mountInfo,
			dir:   rawCephSnapDiffInfo.dir1,
		},
		dirAux: &Directory{
			mount: mountInfo,
			dir:   rawCephSnapDiffInfo.dir_aux,
		},
	}

	return cephSnapDiffInfo, nil
}

// validate checks that the SnapDiffInfo struct is valid.
func (info *SnapDiffInfo) validate() error {
	if info.cMount == nil || info.dir1 == nil || info.dirAux == nil {
		return errInvalid
	}

	return nil
}

// Readdir returns the next entry in the snapshot diff stream.
// If there are no more entries, it returns (nil, nil).
//
// Implements:
//
//	int ceph_readdir_snapdiff(struct ceph_snapdiff_info* snapdiff,
//	                           struct ceph_snapdiff_entry_t* out);
func (info *SnapDiffInfo) Readdir() (*SnapDiffEntry, error) {
	if err := info.validate(); err != nil {
		return nil, err
	}

	cephReaddirSnapDiffOnce.Do(func() {
		cephReaddirSnapDiff, cephReaddirSnapDiffErr = dlsym.LookupSymbol("ceph_readdir_snapdiff")
	})
	if cephReaddirSnapDiffErr != nil {
		return nil, fmt.Errorf("%w: %w", ErrNotImplemented, cephReaddirSnapDiffErr)
	}

	rawSnapDiffEntry := &C._ceph_snapdiff_entry_t{}
	rawSnapDiffInfo := &C._ceph_snapdiff_info{
		cmount:  info.cMount.mount,
		dir1:    info.dir1.dir,
		dir_aux: info.dirAux.dir,
	}

	ret := C.readdir_snapdiff_dlsym(
		cephReaddirSnapDiff,
		rawSnapDiffInfo,
		rawSnapDiffEntry)
	if ret < 0 {
		return nil, getError(ret)
	}
	if ret == 0 {
		// return 0 indicates there is not more entries to return.
		return nil, nil
	}

	snapDiffEntry := &SnapDiffEntry{
		DirEntry: toDirEntry(&rawSnapDiffEntry.dir_entry),
		SnapID:   uint64(rawSnapDiffEntry.snapid),
	}
	return snapDiffEntry, nil
}

// Close closes the snapshot diff handle.
//
// Implements:
//
//	int ceph_close_snapdiff(struct ceph_snapdiff_info* snapdiff);
func (info *SnapDiffInfo) Close() error {
	if err := info.validate(); err != nil {
		return err
	}

	cephCloseSnapDiffOnce.Do(func() {
		cephCloseSnapDiff, cephCloseSnapDiffErr = dlsym.LookupSymbol("ceph_close_snapdiff")
	})
	if cephCloseSnapDiffErr != nil {
		return fmt.Errorf("%w: %w", ErrNotImplemented, cephCloseSnapDiffErr)
	}

	rawCephSnapDiffInfo := &C._ceph_snapdiff_info{
		cmount:  info.cMount.mount,
		dir1:    info.dir1.dir,
		dir_aux: info.dirAux.dir,
	}
	ret := C.close_snapdiff_dlsym(
		cephCloseSnapDiff,
		rawCephSnapDiffInfo)

	if ret != 0 {
		return getError(ret)
	}

	return nil
}
