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

package registry

import (
	"github.com/opencloud-eu/reva/v2/pkg/auth"
)

// NewCredentialFunc is the function that credential strategies
// should register at init time.
type NewCredentialFunc func(map[string]interface{}) (auth.CredentialStrategy, error)

// NewCredentialFuncs is a map containing all the registered auth strategies.
var NewCredentialFuncs = map[string]NewCredentialFunc{}

// Register registers a new auth strategy  new function.
// Not safe for concurrent use. Safe for use from package init.
func Register(name string, f NewCredentialFunc) {
	NewCredentialFuncs[name] = f
}
