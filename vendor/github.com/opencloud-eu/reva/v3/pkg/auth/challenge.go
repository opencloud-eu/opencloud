// Copyright 2026 OpenCloud GmbH <mail@opencloud.eu>
// SPDX-License-Identifier: Apache-2.0

package auth

import "strings"

var quotedStringEscaper = strings.NewReplacer(`\`, `\\`, `"`, `\"`)

// Challenge builds a WWW-Authenticate challenge for the given scheme with the
// realm escaped as a quoted-string. Additional params are appended without escaping.
func Challenge(scheme, realm string, params ...string) string {
	return strings.Join(append([]string{scheme + ` realm="` + quotedStringEscaper.Replace(realm) + `"`}, params...), ", ")
}
