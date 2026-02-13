// Copyright 2018-2022 CERN
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
// In applying this license, CERN does not waive the privileges and immunities
// granted to it by virtue of its status as an Intergovernmental Organization
// or submit itself to any jurisdiction.

package net

import (
	"net/url"
	"strings"

	"github.com/pkg/errors"
)

var (
	// ErrInvalidHeaderValue defines an error which can occure when trying to parse a header value.
	ErrInvalidHeaderValue = errors.New("invalid value")
)

type ctxKey int

const (
	// CtxKeyBaseURI is the key of the base URI context field
	CtxKeyBaseURI ctxKey = iota

	// NsDav is the Dav ns
	NsDav = "DAV:"
	// NsOwncloud is the owncloud ns
	NsOwncloud = "http://owncloud.org/ns"
	// NsOCS is the OCS ns
	NsOCS = "http://open-collaboration-services.org/ns"

	// RFC1123 time that mimics oc10. time.RFC1123 would end in "UTC", see https://github.com/golang/go/issues/13781
	RFC1123 = "Mon, 02 Jan 2006 15:04:05 GMT"

	// PropQuotaUnknown is the quota unknown property
	PropQuotaUnknown = "-2"
	// PropOcFavorite is the favorite ns property
	PropOcFavorite = "http://owncloud.org/ns/favorite"
	// PropOcMetaPathForUser is the meta-path-for-user ns property
	PropOcMetaPathForUser = "http://owncloud.org/ns/meta-path-for-user"

	// DepthZero represents the webdav zero depth value
	DepthZero Depth = "0"
	// DepthOne represents the webdav one depth value
	DepthOne Depth = "1"
	// DepthInfinity represents the webdav infinity depth value
	DepthInfinity Depth = "infinity"
)

// Depth is a type representing the webdav depth header value
type Depth string

// String returns the string representation of the webdav depth value
func (d Depth) String() string {
	return string(d)
}

// EncodePath encodes the path of a url.
//
// slashes (/) are treated as path-separators.
func EncodePath(path string) string {
	return (&url.URL{Path: path}).EscapedPath()
}

// ParseDepth parses the depth header value defined in https://tools.ietf.org/html/rfc4918#section-9.1
// Valid values are "0", "1" and "infinity". An empty string will be parsed to "1".
// For all other values this method returns an error.
func ParseDepth(s string) (Depth, error) {
	if s == "" {
		return DepthOne, nil
	}

	switch strings.ToLower(s) {
	case DepthZero.String():
		return DepthZero, nil
	case DepthOne.String():
		return DepthOne, nil
	case DepthInfinity.String():
		return DepthInfinity, nil
	default:
		return "", errors.Wrapf(ErrInvalidHeaderValue, "invalid depth: %s", s)
	}
}

// ParseOverwrite parses the overwrite header value defined in https://datatracker.ietf.org/doc/html/rfc4918#section-10.6
// Valid values are "T" and "F". An empty string will be parse to true.
func ParseOverwrite(s string) (bool, error) {
	if s == "" {
		s = "T"
	}
	if s != "T" && s != "F" {
		return false, errors.Wrapf(ErrInvalidHeaderValue, "invalid overwrite: %s", s)
	}
	return s == "T", nil
}

// ParseDestination parses the destination header value defined in https://datatracker.ietf.org/doc/html/rfc4918#section-10.3
// The returned path will be relative to the given baseURI.
func ParseDestination(baseURI, s string) (string, error) {
	if s == "" {
		return "", errors.Wrap(ErrInvalidHeaderValue, "destination header is empty")
	}
	dstURL, err := url.ParseRequestURI(s)
	if err != nil {
		return "", errors.Wrap(ErrInvalidHeaderValue, err.Error())
	}

	// TODO check if path is on same storage, return 502 on problems, see https://tools.ietf.org/html/rfc4918#section-9.9.4
	// TODO make request.php optional in destination header
	// Strip the base URI from the destination. The destination might contain redirection prefixes which need to be handled
	urlSplit := strings.Split(dstURL.Path, baseURI)
	if len(urlSplit) != 2 {
		return "", errors.Wrap(ErrInvalidHeaderValue, "destination path does not contain base URI")
	}

	return urlSplit[1], nil
}

// ParsePrefer parses the prefer header value defined in https://datatracker.ietf.org/doc/html/rfc8144
func ParsePrefer(s string) map[string]string {
	parts := strings.Split(s, ",")
	m := make(map[string]string, len(parts))
	for _, part := range parts {
		kv := strings.SplitN(strings.ToLower(strings.Trim(part, " ")), "=", 2)
		if len(kv) == 2 {
			m[kv[0]] = kv[1]
		} else {
			m[kv[0]] = "1" // mark it as set
		}
	}
	return m
}
