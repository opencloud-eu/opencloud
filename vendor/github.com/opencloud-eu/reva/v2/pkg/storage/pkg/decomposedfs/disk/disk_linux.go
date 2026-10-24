// Copyright 2026 OpenCloud GmbH <mail@opencloud.eu>
// SPDX-License-Identifier: Apache-2.0

//go:build linux

package disk

import (
	"os"
	"syscall"
)

func Fdatasync(f *os.File) error {
	return syscall.Fdatasync(int(f.Fd()))
}
