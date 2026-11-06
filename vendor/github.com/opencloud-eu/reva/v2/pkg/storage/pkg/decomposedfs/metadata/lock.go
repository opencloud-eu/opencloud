// Copyright 2026 OpenCloud GmbH <mail@opencloud.eu>
// SPDX-License-Identifier: Apache-2.0

package metadata

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/rogpeppe/go-internal/lockedfile"
)

// lockFile opens and locks the lock file of a node like lockedfile.OpenFile, recording in the metrics how long
// it waits for the lock and, once the returned lock is closed, how long the lock has been held
func lockFile(n MetadataNode, path string, flag int, m *backendMetrics) (*heldLock, error) {
	lm := m.locks(n)

	m.locksWaiting.Inc()
	start := time.Now()
	f, err := lockedfile.OpenFile(path, flag, 0600)
	m.locksWaiting.Dec()
	if err != nil {
		return nil, err
	}
	locked := time.Now()
	lm.wait.Observe(locked.Sub(start).Seconds())
	return &heldLock{f: f, held: lm.held, since: locked}, nil
}

// heldLock is a locked lock file
type heldLock struct {
	f     *lockedfile.File
	held  prometheus.Observer
	since time.Time
}

// Close unlocks and closes the lock file
func (l *heldLock) Close() error {
	l.held.Observe(time.Since(l.since).Seconds())
	return l.f.Close()
}
