// Copyright 2026 OpenCloud GmbH <mail@opencloud.eu>
// SPDX-License-Identifier: Apache-2.0

package metadata

import (
	"context"
	"errors"
	"syscall"

	"github.com/opencloud-eu/reva/v2/pkg/appctx"
	"github.com/pkg/xattr"
)

// maxListXattrRetries bounds the number of attempts made by listXattr when it
// keeps racing with a concurrent writer.
const maxListXattrRetries = 10

// listXattr lists the extended attribute names of the file at path, retrying on
// transient ERANGE errors.
//
// xattr.List is not atomic: internally it issues a listxattr syscall to learn
// the size of the extended-attribute name list, allocates a buffer of that
// size and issues a second listxattr to read the names into it. If another
// goroutine or process adds an extended attribute to the same inode between the
// two syscalls, the name list grows beyond the buffer and the kernel returns
// ERANGE ("numerical result out of range").
//
// This is a transient condition: a fresh size query on the next attempt
// reflects the new size, so we simply retry a bounded number of times. See the
// posix fs watcher, whose assimilation can read a node's xattrs while the node
// is still being created, for a concrete source of such concurrent writes.
//
// E2BIG is deliberately not retried: it means the name list is larger than the
// maximum size the kernel will hand out (the 64KiB VFS limit, e.g. on XFS),
// which is not a race and cannot be retried away. It is returned as-is.
func listXattr(ctx context.Context, path string) ([]string, error) {
	var (
		attrs []string
		err   error
	)
	for i := 0; i < maxListXattrRetries; i++ {
		attrs, err = xattr.List(path)
		if err == nil {
			if i > 0 {
				appctx.GetLogger(ctx).Debug().Str("path", path).Int("attempt", i+1).Msg("xattr.List ERANGE recovered after retry")
			}
			return attrs, nil
		}
		if !isRangeError(err) {
			return attrs, err
		}
		appctx.GetLogger(ctx).Debug().Err(err).Str("path", path).Int("attempt", i+1).Msg("xattr.List ERANGE, retrying")
	}
	appctx.GetLogger(ctx).Error().Err(err).Str("path", path).Msg("xattr.List still failing after ERANGE retries")
	return attrs, err
}

// isRangeError reports whether err is the transient listxattr "buffer too small"
// error (ERANGE) caused by the name list growing concurrently.
func isRangeError(err error) bool {
	var xerr *xattr.Error
	if errors.As(err, &xerr) {
		if serr, ok := xerr.Err.(syscall.Errno); ok {
			return serr == syscall.ERANGE
		}
	}
	return false
}

// isTooBigError reports whether err is the listxattr E2BIG error: the name list
// exceeds the maximum size the kernel will hand out (the 64KiB VFS limit, e.g.
// on XFS). It is not transient, so callers should fail fast instead of retrying.
func isTooBigError(err error) bool {
	var xerr *xattr.Error
	if errors.As(err, &xerr) {
		if serr, ok := xerr.Err.(syscall.Errno); ok {
			return serr == syscall.E2BIG
		}
	}
	return false
}
