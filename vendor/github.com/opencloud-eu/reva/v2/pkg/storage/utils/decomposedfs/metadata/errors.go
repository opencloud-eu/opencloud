// Copyright 2018-2021 CERN
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
// In applying this license, CERN does not waive the privileges and immunities
// granted to it by virtue of its status as an Intergovernmental Organization
// or submit itself to any jurisdiction.

package metadata

import (
	"io/fs"
	"os"
	"syscall"

	"github.com/opencloud-eu/reva/v2/pkg/errtypes"
	"github.com/pkg/errors"
	"github.com/pkg/xattr"
)

// IsNotExist checks if there is a os not exists error buried inside the xattr error,
// as we cannot just use os.IsNotExist().
func IsNotExist(err error) bool {
	if _, ok := err.(errtypes.IsNotFound); ok {
		return true
	}
	if os.IsNotExist(errors.Cause(err)) {
		return true
	}
	if xerr, ok := errors.Cause(err).(*xattr.Error); ok {
		if serr, ok2 := xerr.Err.(syscall.Errno); ok2 {
			return serr == syscall.ENOENT
		}
	}
	return false
}

// IsErrRange checks for a syscall.ERANGE buried inside an xattr error. listxattr
// returns ERANGE when the set of attribute names grows between the size-probe call
// and the read call (a concurrent setxattr landed in between). This is transient and
// self-clearing once the competing writer finishes its batch.
func IsErrRange(err error) bool {
	if xerr, ok := errors.Cause(err).(*xattr.Error); ok {
		if serr, ok2 := xerr.Err.(syscall.Errno); ok2 {
			return serr == syscall.ERANGE
		}
	}
	return false
}

// IsErrTooBig checks for a syscall.E2BIG buried inside an xattr error. listxattr
// returns E2BIG when the attribute name list is larger than the maximum size the
// kernel will hand out (the 64KiB VFS limit, e.g. on XFS). Unlike ERANGE this is
// not a transient race: the list cannot be retrieved at all, so callers should
// fail fast instead of retrying.
func IsErrTooBig(err error) bool {
	if xerr, ok := errors.Cause(err).(*xattr.Error); ok {
		if serr, ok2 := xerr.Err.(syscall.Errno); ok2 {
			return serr == syscall.E2BIG
		}
	}
	return false
}

// IsAttrUnset checks the xattr.ENOATTR from the xattr package which redifines it as ENODATA on platforms that do not natively support it (eg. linux)
// see https://github.com/pkg/xattr/blob/8725d4ccc0fcef59c8d9f0eaf606b3c6f962467a/xattr_linux.go#L19-L22
func IsAttrUnset(err error) bool {
	if xerr, ok := errors.Cause(err).(*xattr.Error); ok {
		if serr, ok2 := xerr.Err.(syscall.Errno); ok2 {
			return serr == xattr.ENOATTR
		}
	}
	return false
}

// The os error is buried inside the fs.PathError error
func IsNotDir(err error) bool {
	if perr, ok := errors.Cause(err).(*fs.PathError); ok {
		if serr, ok2 := perr.Err.(syscall.Errno); ok2 {
			return serr == syscall.ENOTDIR
		}
	}
	if xerr, ok := errors.Cause(err).(*xattr.Error); ok {
		if serr, ok2 := xerr.Err.(syscall.Errno); ok2 {
			return serr == syscall.ENOTDIR
		}
	}
	return false
}
