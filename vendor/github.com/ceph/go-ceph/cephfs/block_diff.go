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

// struct ceph_file_blockdiff_result lacks a definition, so create an alias and
// treat it as opaque.
typedef struct _ceph_file_blockdiff_result _ceph_file_blockdiff_result;

typedef struct
{
  struct ceph_mount_info* cmount;
  struct _ceph_file_blockdiff_result* blockp;
} _ceph_file_blockdiff_info;

typedef struct
{
  uint64_t offset;
  uint64_t len;
} _cblock;

typedef struct
{
  uint64_t num_blocks;
  struct _cblock *b;
} _ceph_file_blockdiff_changedblocks;

// ceph_file_blockdiff_init_fn matches the ceph_file_blockdiff_init function signature.
typedef int(*ceph_file_blockdiff_init_fn)(struct ceph_mount_info* cmount,
                                  const char* root_path,
                                  const char* rel_path,
                                  const char* snap1,
                                  const char* snap2,
                                  _ceph_file_blockdiff_info* out_info);

// ceph_file_blockdiff_init_dlsym take *fn as ceph_file_blockdiff_init and calls the dynamically loaded
// ceph_file_blockdiff_init function passed as 1st argument.
static inline int ceph_file_blockdiff_init_dlsym(void *fn,
                                  struct ceph_mount_info* cmount,
                                  const char* root_path,
                                  const char* rel_path,
                                  const char* snap1,
                                  const char* snap2,
                                  _ceph_file_blockdiff_info* out_info) {
	// cast function pointer fn to ceph_file_blockdiff_init and call the function
	return ((ceph_file_blockdiff_init_fn) fn)(cmount, root_path, rel_path, snap1, snap2, out_info);
}

// ceph_file_blockdiff_fn matches the ceph_file_blockdiff function signature.
typedef int(*ceph_file_blockdiff_fn)(_ceph_file_blockdiff_info* info,
                                  _ceph_file_blockdiff_changedblocks* blocks);

// ceph_file_blockdiff_dlsym take *fn as ceph_file_blockdiff and calls the dynamically loaded
// ceph_file_blockdiff function passed as 1st argument.
static inline int ceph_file_blockdiff_dlsym(void *fn,
                                  _ceph_file_blockdiff_info* info,
                                  _ceph_file_blockdiff_changedblocks* blocks) {
	// cast function pointer fn to ceph_file_blockdiff and call the function
	return ((ceph_file_blockdiff_fn) fn)(info, blocks);
}

// ceph_free_file_blockdiff_buffer_fn matches the ceph_free_file_blockdiff_buffer function signature.
typedef void(*ceph_free_file_blockdiff_buffer_fn)(_ceph_file_blockdiff_changedblocks* blocks);

// ceph_free_file_blockdiff_buffer_dlsym take *fn as ceph_free_file_blockdiff_buffer and calls the dynamically loaded
// ceph_free_file_blockdiff_buffer function passed as 1st argument.
static inline void ceph_free_file_blockdiff_buffer_dlsym(void *fn,
                                  _ceph_file_blockdiff_changedblocks* blocks) {
	// cast function pointer fn to ceph_free_file_blockdiff_buffer and call the function
	((ceph_free_file_blockdiff_buffer_fn) fn)(blocks);
}

// ceph_file_blockdiff_finish_fn matches the ceph_file_blockdiff_finish function signature.
typedef int(*ceph_file_blockdiff_finish_fn)(_ceph_file_blockdiff_info* info);

// ceph_file_blockdiff_finish_dlsym take *fn as ceph_file_blockdiff_finish and calls the dynamically loaded
// ceph_file_blockdiff_finish function passed as 1st argument.
static inline int ceph_file_blockdiff_finish_dlsym(void *fn,
                                  _ceph_file_blockdiff_info* info) {
	// cast function pointer fn to ceph_file_blockdiff_finish and call the function
	return ((ceph_file_blockdiff_finish_fn) fn)(info);
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
	cephFileBlockDiffInitOnce       sync.Once
	cephFileBlockDiffOnce           sync.Once
	cephFreeFileBlockDiffBufferOnce sync.Once
	cephFileBlockDiffFinishOnce     sync.Once

	cephFileBlockDiffInitErr       error
	cephFileBlockDiffErr           error
	cephFreeFileBlockDiffBufferErr error
	cephFileBlockDiffFinishErr     error

	cephFileBlockDiffInit       unsafe.Pointer
	cephFileBlockDiff           unsafe.Pointer
	cephFreeFileBlockDiffBuffer unsafe.Pointer
	cephFileBlockDiffFinish     unsafe.Pointer
)

// FileBlockDiffInfo is a struct that holds the block diff stream handle.
type FileBlockDiffInfo struct {
	cMount                  *MountInfo
	cephFileBlockDiffResult *C._ceph_file_blockdiff_result
	more                    bool
}

// ChangedBlock is a struct that holds the offset and length of a block.
type ChangedBlock struct {
	Offset uint64
	Len    uint64
}

// FileBlockDiffChangedBlocks is a struct that holds the number of blocks
// and list of Changed Blocks.
type FileBlockDiffChangedBlocks struct {
	NumBlocks     uint64
	ChangedBlocks []ChangedBlock
}

// FileBlockDiffInit initializes the block diff stream to get file block deltas.
// It takes the mount handle, root path, relative path, snapshot names and returns
// a FileBlockDiffInfo struct that contains the block diff stream handle.
//
// Implements:
//
//	int ceph_file_blockdiff_init(struct ceph_mount_info* cmount,
//				     const char* root_path,
//				     const char* rel_path,
//				     const char* snap1,
//				     const char* snap2,
//				     struct ceph_file_blockdiff_info* out_info);
func FileBlockDiffInit(mount *MountInfo, rootPath, relPath, snap1, snap2 string) (*FileBlockDiffInfo, error) {
	if mount == nil || rootPath == "" || relPath == "" ||
		snap1 == "" || snap2 == "" {
		return nil, errInvalid
	}
	cRootPath := C.CString(rootPath)
	cRelPath := C.CString(relPath)
	cSnap1 := C.CString(snap1)
	cSnap2 := C.CString(snap2)
	defer func() {
		C.free(unsafe.Pointer(cRootPath))
		C.free(unsafe.Pointer(cRelPath))
		C.free(unsafe.Pointer(cSnap1))
		C.free(unsafe.Pointer(cSnap2))
	}()

	// Load the ceph_file_blockdiff_init function from the shared library.
	cephFileBlockDiffInitOnce.Do(func() {
		cephFileBlockDiffInit, cephFileBlockDiffInitErr = dlsym.LookupSymbol("ceph_file_blockdiff_init")
	})
	if cephFileBlockDiffInitErr != nil {
		return nil, fmt.Errorf("%w: %w", ErrNotImplemented, cephFileBlockDiffInitErr)
	}

	rawCephBlockDiffInfo := &C._ceph_file_blockdiff_info{}

	// Call the ceph_file_blockdiff_init function with the provided arguments.
	ret := C.ceph_file_blockdiff_init_dlsym(cephFileBlockDiffInit,
		mount.mount,
		cRootPath,
		cRelPath,
		cSnap1,
		cSnap2,
		rawCephBlockDiffInfo,
	)
	if ret != 0 {
		return nil, getError(ret)
	}

	mountInfo := &MountInfo{
		mount: rawCephBlockDiffInfo.cmount,
	}

	cephFileBlockDiffInfo := &FileBlockDiffInfo{
		cMount:                  mountInfo,
		cephFileBlockDiffResult: rawCephBlockDiffInfo.blockp,
		more:                    true,
	}

	return cephFileBlockDiffInfo, nil
}

// validate checks if the FileBlockDiffInfo struct is valid.
func (info *FileBlockDiffInfo) validate() error {
	if info.cMount == nil || info.cephFileBlockDiffResult == nil {
		return errInvalid
	}

	return nil
}

// Read retrieves the next set of changed blocks for a file.
//
// Implements:
//
//	int ceph_file_blockdiff(struct ceph_file_blockdiff_info* info,
//				struct ceph_file_blockdiff_changedblocks* blocks);
func (info *FileBlockDiffInfo) Read() (*FileBlockDiffChangedBlocks, error) {
	err := info.validate()
	if err != nil {
		return nil, err
	}

	// Load the ceph_file_blockdiff function from the shared library.
	cephFileBlockDiffOnce.Do(func() {
		cephFileBlockDiff, cephFileBlockDiffErr = dlsym.LookupSymbol("ceph_file_blockdiff")
	})
	if cephFileBlockDiffErr != nil {
		return nil, fmt.Errorf("%w: %w", ErrNotImplemented, cephFileBlockDiffErr)
	}
	cephFreeFileBlockDiffBufferOnce.Do(func() {
		cephFreeFileBlockDiffBuffer, cephFreeFileBlockDiffBufferErr = dlsym.LookupSymbol("ceph_free_file_blockdiff_buffer")
	})
	if cephFreeFileBlockDiffBufferErr != nil {
		return nil, fmt.Errorf("%w: %w", ErrNotImplemented, cephFreeFileBlockDiffBufferErr)
	}

	rawCephBlockDiffChangedBlocks := &C._ceph_file_blockdiff_changedblocks{}
	rawCephFileBlockDiffInfo := &C._ceph_file_blockdiff_info{
		cmount: info.cMount.mount,
		blockp: info.cephFileBlockDiffResult,
	}

	// Call the ceph_file_blockdiff function with the provided arguments.
	ret := C.ceph_file_blockdiff_dlsym(cephFileBlockDiff,
		rawCephFileBlockDiffInfo,
		rawCephBlockDiffChangedBlocks,
	)
	if ret < 0 {
		return nil, getError(ret)
	}
	// ret > 0 indicates that there are more entries after this call.
	info.more = (ret > 0)

	// Free the memory allocated for the blocks by ceph_file_blockdiff.
	defer C.ceph_free_file_blockdiff_buffer_dlsym(cephFreeFileBlockDiffBuffer,
		rawCephBlockDiffChangedBlocks)

	cNumBlocks := uint64(rawCephBlockDiffChangedBlocks.num_blocks)

	// Convert the C struct to Go struct.
	cBlocks := make([]ChangedBlock, int(cNumBlocks))
	if cNumBlocks == 0 {
		return &FileBlockDiffChangedBlocks{
			NumBlocks:     0,
			ChangedBlocks: cBlocks,
		}, nil
	}

	currentCBlock := (*C._cblock)(unsafe.Pointer(rawCephBlockDiffChangedBlocks.b))
	for i := uint64(0); i < cNumBlocks; i++ {
		cBlocks[i] = ChangedBlock{
			Offset: uint64(currentCBlock.offset),
			Len:    uint64(currentCBlock.len),
		}
		currentCBlock = (*C._cblock)(unsafe.Pointer(uintptr(unsafe.Pointer(currentCBlock)) + unsafe.Sizeof(C._cblock{})))
	}

	return &FileBlockDiffChangedBlocks{
		NumBlocks:     cNumBlocks,
		ChangedBlocks: cBlocks,
	}, nil
}

// More returns true if there are more entries to read in the block diff stream.
func (info *FileBlockDiffInfo) More() bool {
	return info.more
}

// Close closes the block diff stream.
//
// Implements:
//
//	int ceph_file_blockdiff_finish(struct ceph_file_blockdiff_info* info);
func (info *FileBlockDiffInfo) Close() error {
	err := info.validate()
	if err != nil {
		return err
	}

	// Load the ceph_file_blockdiff_finish function from the shared library.
	cephFileBlockDiffFinishOnce.Do(func() {
		cephFileBlockDiffFinish, cephFileBlockDiffFinishErr = dlsym.LookupSymbol("ceph_file_blockdiff_finish")
	})
	if cephFileBlockDiffFinishErr != nil {
		return fmt.Errorf("%w: %w", ErrNotImplemented, cephFileBlockDiffFinishErr)
	}

	rawCephFileBlockDiffInfo := &C._ceph_file_blockdiff_info{
		cmount: info.cMount.mount,
		blockp: info.cephFileBlockDiffResult,
	}

	// Call the ceph_file_blockdiff_finish function with the provided arguments.
	ret := C.ceph_file_blockdiff_finish_dlsym(
		cephFileBlockDiffFinish,
		rawCephFileBlockDiffInfo,
	)
	if ret != 0 {
		return getError(ret)
	}

	return nil
}
