// Copyright 2026 OpenCloud GmbH <mail@opencloud.eu>
// SPDX-License-Identifier: Apache-2.0

package metadata

import (
	"errors"
	"syscall"

	"github.com/pkg/xattr"
)

// maxListXattrRetries bounds the number of attempts made by listXattr when it
// keeps racing with a concurrent writer.
const maxListXattrRetries = 10

// listXattr lists the extended attribute names of the file at path, retrying on
// transient ERANGE/E2BIG errors.
//
// xattr.List is not atomic: internally it issues a listxattr syscall to learn
// the size of the extended-attribute name list, allocates a buffer of that
// size and issues a second listxattr to read the names into it. If another
// goroutine or process adds an extended attribute to the same inode between the
// two syscalls, the name list grows beyond the buffer and the kernel returns
// ERANGE ("numerical result out of range") (or E2BIG on some filesystems).
//
// This is a transient condition: a fresh size query on the next attempt
// reflects the new size, so we simply retry a bounded number of times. See the
// posix fs watcher, whose assimilation can read a node's xattrs while the node
// is still being created, for a concrete source of such concurrent writes.
func listXattr(path string) ([]string, error) {
	var (
		attrs []string
		err   error
	)
	for i := 0; i < maxListXattrRetries; i++ {
		attrs, err = xattr.List(path)
		if err == nil || !isRangeError(err) {
			return attrs, err
		}
	}
	return attrs, err
}

// isRangeError reports whether err is a transient listxattr "buffer too small"
// error (ERANGE or E2BIG) caused by the name list growing concurrently.
func isRangeError(err error) bool {
	var xerr *xattr.Error
	if errors.As(err, &xerr) {
		if serr, ok := xerr.Err.(syscall.Errno); ok {
			return serr == syscall.ERANGE || serr == syscall.E2BIG
		}
	}
	return false
}
