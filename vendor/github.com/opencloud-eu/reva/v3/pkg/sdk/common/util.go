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

package common

import (
	"strings"
)

// FindString performs a case-sensitive string search in a string vector and returns its index or -1 if it couldn't be found.
func FindString(a []string, x string) int {
	for i, n := range a {
		if x == n {
			return i
		}
	}

	return -1
}

// FindStringNoCase performs a case-insensitive string search in a string vector and returns its index or -1 if it couldn't be found.
func FindStringNoCase(a []string, x string) int {
	for i, n := range a {
		if strings.EqualFold(x, n) {
			return i
		}
	}

	return -1
}
