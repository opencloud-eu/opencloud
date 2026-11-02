// Copyright 2026 OpenCloud GmbH <mail@opencloud.eu>
// SPDX-License-Identifier: Apache-2.0

package registry

var GetConfig = getConfig

var RedactAddrs = redactAddrs

// ResetConfigure drops the configuration set by Configure
func ResetConfigure() {
	_configured = nil
}
