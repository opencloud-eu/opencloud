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

package provider

import (
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"time"
)

var (
	farPastExpiryTime = time.Unix(0, 0)
)

func uniqueStrings(s []string) []string {
	var res []string
	m := make(map[string]bool)
	for _, s := range s {
		if _, ok := m[s]; ok {
			continue
		}
		m[s] = true
		res = append(res, s)
	}

	return res
}

// makePathSet returns a set of the provided paths, skipping empty ones. It is
// used for endpoints which are served under more than one path.
func makePathSet(path string, alias []string) map[string]bool {
	set := make(map[string]bool, len(alias)+1)
	for _, p := range append([]string{path}, alias...) {
		if p == "" {
			continue
		}
		set[p] = true
	}

	return set
}

func getRequestURL(req *http.Request, isTrustedSource bool) *url.URL {
	u, _ := url.Parse(req.URL.String())

	if isTrustedSource {
		// Look at proxy injected values to rewrite URLs if trusted.
		for {
			prefix := req.Header.Get("X-Forwarded-Prefix")
			if prefix != "" {
				u.Path = fmt.Sprintf("%s%s", prefix, u.Path)
				break
			}
			replaced := req.Header.Get("X-Replaced-Path")
			if replaced != "" {
				u.Path = replaced
				break
			}

			break
		}
	}

	return u
}

func addResponseHeaders(header http.Header) {
	header.Set("Cache-Control", "no-cache, no-store, must-revalidate")
	header.Set("Pragma", "no-cache")
	header.Set("X-Content-Type-Options", "nosniff")
	header.Set("Referrer-Policy", "origin")
}

// makeArrayFromBoolMap returns the keys with a true value of the provided map,
// sorted, so that the same set of scopes or claims always produces the same
// list in a response or a token.
func makeArrayFromBoolMap(m map[string]bool) []string {
	result := []string{}
	for k, v := range m {
		if v {
			result = append(result, k)
		}
	}
	slices.Sort(result)

	return result
}
