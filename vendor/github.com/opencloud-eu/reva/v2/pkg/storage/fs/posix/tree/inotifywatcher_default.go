//go:build !linux

// Copyright 2025 OpenCloud GmbH <mail@opencloud.eu>
// SPDX-License-Identifier: Apache-2.0

package tree

import (
	"github.com/opencloud-eu/reva/v2/pkg/errtypes"
	"github.com/opencloud-eu/reva/v2/pkg/storage/fs/posix/options"
	"github.com/rs/zerolog"
)

// NullWatcher is a dummy watcher that does nothing
type NullWatcher struct{}

// Watch does nothing
func (*NullWatcher) Watch(path string) {}

// NewInotifyWatcher returns a new inotify watcher
func NewInotifyWatcher(_ *Tree, _ *options.Options, _ *zerolog.Logger) (*NullWatcher, error) {
	return nil, errtypes.NotSupported("inotify watcher is not supported on this platform")
}
