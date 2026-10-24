// Copyright 2026 OpenCloud GmbH <mail@opencloud.eu>
// SPDX-License-Identifier: Apache-2.0

//go:build !linux && !freebsd && !darwin

package disk

import "os"

func Fdatasync(f *os.File) error {
	return nil
}
