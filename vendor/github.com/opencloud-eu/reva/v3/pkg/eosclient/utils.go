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

package eosclient

import (
	"fmt"

	"github.com/opencloud-eu/reva/v3/pkg/errtypes"
)

const (
	// SystemAttr is the system extended attribute.
	SystemAttr AttrType = iota
	// UserAttr is the user extended attribute.
	UserAttr
)

// AttrStringToType converts a string to an AttrType
func AttrStringToType(t string) (AttrType, error) {
	switch t {
	case "sys":
		return SystemAttr, nil
	case "user":
		return UserAttr, nil
	default:
		return 0, errtypes.InternalError("attr type not existing")
	}
}

// AttrTypeToString converts a type to a string representation.
func AttrTypeToString(at AttrType) string {
	switch at {
	case SystemAttr:
		return "sys"
	case UserAttr:
		return "user"
	default:
		return "invalid"
	}
}

// GetKey returns the key considering the type of attribute.
func (a *Attribute) GetKey() string {
	return fmt.Sprintf("%s.%s", AttrTypeToString(a.Type), a.Key)
}
