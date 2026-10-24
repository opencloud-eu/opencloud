// Copyright 2026 OpenCloud GmbH <mail@opencloud.eu>
// SPDX-License-Identifier: Apache-2.0

//go:build freebsd || darwin

package disk

import "os"

func Fdatasync(file *os.File) error {
	return file.Sync()
}
