//go:build !linux && (!darwin || !experimental_watchfs_darwin)

// Copyright 2025 OpenCloud GmbH <mail@opencloud.eu>
// SPDX-License-Identifier: Apache-2.0

package tree

import (
	"github.com/rs/zerolog"

	"github.com/opencloud-eu/reva/v2/pkg/storage/fs/posix/options"
)

// NewWatcher returns a NoopWatcher on unsupported platforms
func NewWatcher(_ *Tree, _ *options.Options, _ *zerolog.Logger) (*NoopWatcher, error) {
	return nil, ErrUnsupportedWatcher
}
