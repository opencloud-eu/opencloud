package cephfs

/*
#cgo LDFLAGS: -lcephfs
#cgo CPPFLAGS: -D_FILE_OFFSET_BITS=64
#include <stdlib.h>
#include <cephfs/libcephfs.h>

int _go_ceph_chown(struct ceph_mount_info *cmount, const char *path, uid_t uid, gid_t gid) {
	return ceph_chown(cmount, path, uid, gid);
}

int _go_ceph_lchown(struct ceph_mount_info *cmount, const char *path, uid_t uid, gid_t gid) {
	return ceph_lchown(cmount, path, uid, gid);
}
*/
import "C"

import (
	"unsafe"
)

// Chmod changes the mode bits (permissions) of a file/directory.
func (mount *MountInfo) Chmod(path string, mode uint32) error {
	cPath := C.CString(path)
	defer C.free(unsafe.Pointer(cPath))

	ret := C.ceph_chmod(mount.mount, cPath, C.mode_t(mode))
	return getError(ret)
}

// Chown changes the ownership of a file/directory.
func (mount *MountInfo) Chown(path string, user uint32, group uint32) error {
	cPath := C.CString(path)
	defer C.free(unsafe.Pointer(cPath))

	ret := C._go_ceph_chown(mount.mount, cPath, C.uid_t(user), C.gid_t(group))
	return getError(ret)
}

// Lchown changes the ownership of a file/directory/etc without following symbolic links
func (mount *MountInfo) Lchown(path string, user uint32, group uint32) error {
	cPath := C.CString(path)
	defer C.free(unsafe.Pointer(cPath))

	ret := C._go_ceph_lchown(mount.mount, cPath, C.uid_t(user), C.gid_t(group))
	return getError(ret)
}
