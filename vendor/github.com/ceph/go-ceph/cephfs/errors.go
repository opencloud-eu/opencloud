package cephfs

/*
#include <errno.h>
*/
import "C"

import (
	"errors"

	"github.com/ceph/go-ceph/internal/errutil"
)

func getError(e C.int) error {
	return errutil.GetError("cephfs", int(e))
}

// getErrorIfNegative converts a ceph return code to error if negative.
// This is useful for functions that return a usable positive value on
// success but a negative error number on error.
func getErrorIfNegative(ret C.int) error {
	if ret >= 0 {
		return nil
	}
	return getError(ret)
}

// Public go errors:

var (
	// ErrEmptyArgument may be returned if a function argument is passed
	// a zero-length slice or map.
	ErrEmptyArgument = errors.New("Argument must contain at least one item")

	// ErrNotConnected may be returned when client is not connected
	// to a cluster.
	ErrNotConnected = getError(-C.ENOTCONN)
	// ErrNotExist indicates a non-specific missing resource.
	ErrNotExist = getError(-C.ENOENT)
	// ErrOpNotSupported is returned in general for operations that are not
	// supported.
	ErrOpNotSupported = getError(-C.EOPNOTSUPP)
	// ErrNotImplemented indicates a function is not implemented in by libcephfs.
	ErrNotImplemented = getError(-C.ENOSYS)

	// Private errors:

	errInvalid     = getError(-C.EINVAL)
	errNameTooLong = getError(-C.ENAMETOOLONG)
	errRange       = getError(-C.ERANGE)
	errBadFile     = getError(-C.EBADF)
	errNotDir      = getError(-C.ENOTDIR)
)
