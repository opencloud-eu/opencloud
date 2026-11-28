//go:build !linux

// Copyright 2025 OpenCloud GmbH <mail@opencloud.eu>
// SPDX-License-Identifier: Apache-2.0

package usermapper

import "github.com/opencloud-eu/reva/v3/pkg/errtypes"

// NewUnixMapper returns a new user mapper
func NewUnixMapper() (*NullMapper, error) {
	return &NullMapper{}, errtypes.NotSupported("inotify watcher is not supported on this platform")
}
