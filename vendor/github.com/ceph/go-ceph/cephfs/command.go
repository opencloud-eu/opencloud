package cephfs

/*
#cgo LDFLAGS: -lcephfs
#cgo CPPFLAGS: -D_FILE_OFFSET_BITS=64
#include <stdlib.h>
#include <cephfs/libcephfs.h>
*/
import "C"

import (
	"unsafe"

	"github.com/ceph/go-ceph/internal/cutil"
)

func cephBufferFree(p unsafe.Pointer) {
	C.ceph_buffer_free((*C.char)(p))
}

// MdsCommand sends commands to the specified MDS.
//
// The args parameter takes a slice of byte slices but typically a single
// slice element is sufficient. The use of two slices exists to best match
// the structure of the underlying C call which is often a legacy interface
// in Ceph.
func (mount *MountInfo) MdsCommand(mdsSpec string, args [][]byte) ([]byte, string, error) {
	return mount.mdsCommand(mdsSpec, args, nil)
}

// MdsCommandWithInputBuffer sends commands to the specified MDS, with an input
// buffer.
//
// The args parameter takes a slice of byte slices but typically a single
// slice element is sufficient. The use of two slices exists to best match
// the structure of the underlying C call which is often a legacy interface
// in Ceph.
func (mount *MountInfo) MdsCommandWithInputBuffer(mdsSpec string, args [][]byte, inputBuffer []byte) ([]byte, string, error) {
	return mount.mdsCommand(mdsSpec, args, inputBuffer)
}

// mdsCommand supports sending formatted commands to MDS.
//
// Implements:
//
//	int ceph_mds_command(struct ceph_mount_info *cmount,
//	    const char *mds_spec,
//	    const char **cmd,
//	    size_t cmdlen,
//	    const char *inbuf, size_t inbuflen,
//	    char **outbuf, size_t *outbuflen,
//	    char **outs, size_t *outslen);
func (mount *MountInfo) mdsCommand(mdsSpec string, args [][]byte, inputBuffer []byte) (buffer []byte, info string, err error) {
	spec := C.CString(mdsSpec)
	defer C.free(unsafe.Pointer(spec))
	ci := cutil.NewCommandInput(args, inputBuffer)
	defer ci.Free()
	co := cutil.NewCommandOutput().SetFreeFunc(cephBufferFree)
	defer co.Free()

	ret := C.ceph_mds_command(
		mount.mount, // cephfs mount ref
		spec,        // mds spec
		(**C.char)(ci.Cmd()),
		C.size_t(ci.CmdLen()),
		(*C.char)(ci.InBuf()),
		C.size_t(ci.InBufLen()),
		(**C.char)(co.OutBuf()),
		(*C.size_t)(co.OutBufLen()),
		(**C.char)(co.Outs()),
		(*C.size_t)(co.OutsLen()))
	buf, status := co.GoValues()
	return buf, status, getError(ret)
}
