// Copyright 2018-2021 CERN
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

package auth

import (
	"testing"
)

func TestGetCredsForUserAgent(t *testing.T) {
	type test struct {
		userAgent            string
		userAgentMap         map[string]string
		availableCredentials []string
		expected             []string
	}

	tests := []*test{
		// no user agent we return all available credentials
		{
			userAgent:            "",
			userAgentMap:         map[string]string{},
			availableCredentials: []string{"basic"},
			expected:             []string{"basic"},
		},

		// map set but user agent not in map
		{
			userAgent:            "curl",
			userAgentMap:         map[string]string{"mirall": "basic"},
			availableCredentials: []string{"basic", "bearer"},
			expected:             []string{"basic", "bearer"},
		},

		// no user map we return all available credentials
		{
			userAgent:            "mirall",
			userAgentMap:         map[string]string{},
			availableCredentials: []string{"basic"},
			expected:             []string{"basic"},
		},

		// user agent set but no mapping set we return all credentials
		{
			userAgent:            "mirall",
			userAgentMap:         map[string]string{},
			availableCredentials: []string{"basic"},
			expected:             []string{"basic"},
		},

		// user mapping set to non available credential, we return all available
		{
			userAgent:            "mirall",
			userAgentMap:         map[string]string{"mirall": "notfound"},
			availableCredentials: []string{"basic", "bearer"},
			expected:             []string{"basic", "bearer"},
		},

		// user mapping set and we return only desired credential
		{
			userAgent:            "mirall",
			userAgentMap:         map[string]string{"mirall": "bearer"},
			availableCredentials: []string{"basic", "bearer"},
			expected:             []string{"bearer"},
		},
	}

	for _, test := range tests {
		got := getCredsForUserAgent(
			test.userAgent,
			test.userAgentMap,
			test.availableCredentials)

		if !match(got, test.expected) {
			fail(t, got, test.expected)
		}
	}
}

func match(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i, v := range a {
		if v != b[i] {
			return false
		}
	}
	return true
}

func fail(t *testing.T, got, expected []string) {
	t.Fatalf("got: %+v expected: %+v", got, expected)
}
