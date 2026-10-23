/*
 * Copyright 2017-2019 Kopano and its licensors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 */

package clients

import (
	"crypto"
	"net/url"
)

// Details hold detail information about clients identified by ID.
type Details struct {
	ID          string `json:"id"`
	DisplayName string `json:"display_name"`
	RedirectURI string `json:"redirect_uri"`
	Trusted     bool   `json:"trusted"`

	// Unverified is true for a client nobody vouches for, one which registered
	// itself or which is described by a metadata document at its client ID.
	// For the latter DisplayName is the host of that client ID, and
	// ClaimedName is the name the document gives.
	Unverified  bool   `json:"unverified"`
	ClaimedName string `json:"claimed_name,omitempty"`

	Registration *ClientRegistration `json:"-"`
}

// A Secured is a client records public key identified by ID.
type Secured struct {
	ID              string
	DisplayName     string
	ApplicationType string

	Kid       string
	PublicKey crypto.PublicKey

	TrustedScopes []string

	Registration *ClientRegistration
}

// IsLocalNativeHTTPURI returns true if the provided URI qualifies to be used
// as http redirect URI for a native client.
func IsLocalNativeHTTPURI(uri *url.URL) bool {
	if uri.Scheme != "http" {
		return false
	}
	return IsLocalNativeHostURI(uri)
}

// IsLocalNativeHostURI returns true if the provided URI hostname is considered
// as localhost for a native client.
func IsLocalNativeHostURI(uri *url.URL) bool {
	hostname := uri.Hostname()
	return hostname == "localhost" || hostname == "127.0.0.1" || hostname == "::1"
}
