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

package shareid

import "strings"

const (
	// IDDelimiter is used to separate the providerid, spaceid and shareid
	IDDelimiter = ":"
)

// Encode encodes a share id
func Encode(providerID, spaceID, shareID string) string {
	return providerID + IDDelimiter + spaceID + IDDelimiter + shareID
}

// Decode decodes an encoded shareid
// share ids are of the format <storageid>:<spaceid>:<shareid>
func Decode(id string) (string, string, string) {
	parts := strings.SplitN(id, IDDelimiter, 3)
	switch len(parts) {
	case 1:
		return "", "", parts[0]
	case 2:
		return parts[0], parts[1], ""
	default:
		return parts[0], parts[1], parts[2]
	}
}
