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

package preferences

import (
	"context"
)

// Manager defines an interface for a preferences manager.
type Manager interface {
	// SetKey sets a key under a specified namespace.
	SetKey(ctx context.Context, key, namespace, value string) error
	// GetKey returns the value for a combination of key and namespace, if set.
	GetKey(ctx context.Context, key, namespace string) (string, error)
}
