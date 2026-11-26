// Copyright 2026 OpenCloud GmbH <mail@opencloud.eu>
// SPDX-License-Identifier: Apache-2.0

package bleve

// SetMaxBytes lowers the data limit of b, so tests don't need megabytes of content
func SetMaxBytes(b *Batch, n uint64) {
	b.maxBytes = n
}

// LockOptimize holds the lock of a running optimization until the returned func is called
func LockOptimize(b *Backend) func() {
	b.optimizeMu.Lock()
	return b.optimizeMu.Unlock
}
