// Copyright 2026 OpenCloud GmbH <mail@opencloud.eu>
// SPDX-License-Identifier: Apache-2.0

package utils

import (
	"errors"
	"net/mail"
	"strings"

	"golang.org/x/net/idna"
)

// CanonicalMail returns the canonical form of an email address. The result
// is always plain ASCII, so it can't contain look-alike characters, and two
// addresses that mail delivery treats the same end up equal:
//   - only a plain address is accepted (no display name, no quoting)
//   - the local part must be ASCII
//   - international domains are converted to punycode
//   - the result is lowercased
func CanonicalMail(email string) (string, error) {
	addr, err := mail.ParseAddress(email)
	if err != nil || addr.Name != "" || addr.Address != email {
		return "", errors.New("invalid email address")
	}
	at := strings.LastIndex(email, "@")
	local, domain := email[:at], email[at+1:]
	if !IsASCII(local) {
		return "", errors.New("email addresses with non-ASCII characters before the @ are not supported")
	}
	if !IsASCII(domain) {
		if domain, err = idna.Lookup.ToASCII(domain); err != nil {
			return "", errors.New("invalid email domain")
		}
	}
	return strings.ToLower(local + "@" + domain), nil
}
