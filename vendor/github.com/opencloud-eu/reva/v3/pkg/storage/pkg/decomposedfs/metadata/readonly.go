// Copyright 2026 OpenCloud GmbH <mail@opencloud.eu>
// SPDX-License-Identifier: Apache-2.0

package metadata

import (
	"context"
	"errors"
	"os"
)

type readOnlyKey struct{}

// WithReadOnly returns a context for reads that must not change anything on disk, e.g. the
// dry run of a revision cleanup.
//
// Today it covers the lock files: to read the metadata of a node, the backends that lock to
// read (the hybrid backend, and the xattrs backend in its locked retry) lock the lock file of
// the node only if it exists, and read without a lock otherwise, instead of creating the lock
// file and its folder. A read without a lock can see a write to the node halfway through, so
// use it only where that is acceptable, never for the reads of the server; and what it read
// isn't cached, where it could replace what a writer cached meanwhile. Writes are not
// affected: they lock, and create the lock files, as before.
func WithReadOnly(ctx context.Context) context.Context {
	return context.WithValue(ctx, readOnlyKey{}, true)
}

// IsReadOnly tells whether ctx is meant for reads that must not change anything on disk, see
// WithReadOnly.
func IsReadOnly(ctx context.Context) bool {
	readOnly, _ := ctx.Value(readOnlyKey{}).(bool)
	return readOnly
}

// lockExisting locks the lock file at path like lockFile, but never creates it: locked is
// false, and there is no error, when there is no lock file to lock. It is how the backends
// lock to read under WithReadOnly.
func lockExisting(n MetadataNode, path string, flag int, m *backendMetrics) (l *heldLock, locked bool, err error) {
	l, err = lockFile(n, path, flag&^os.O_CREATE, m)
	switch {
	case errors.Is(err, os.ErrNotExist):
		return nil, false, nil
	case err != nil:
		return nil, false, err
	}
	return l, true, nil
}
