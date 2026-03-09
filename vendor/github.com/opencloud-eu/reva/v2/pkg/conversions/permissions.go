// Copyright 2020 CERN
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

package conversions

import (
	"errors"
	"fmt"
	"strconv"
)

// Permissions reflects the CRUD permissions used in the OCS sharing API
type Permissions uint

const (
	// PermissionInvalid represents an invalid permission
	PermissionInvalid Permissions = 0
	// PermissionRead grants read permissions on a resource
	PermissionRead Permissions = 1 << (iota - 1)
	// PermissionWrite grants write permissions on a resource
	PermissionWrite
	// PermissionCreate grants create permissions on a resource
	PermissionCreate
	// PermissionDelete grants delete permissions on a resource
	PermissionDelete
	// PermissionShare grants share permissions on a resource
	PermissionShare
	// PermissionAll grants all permissions on a resource
	PermissionAll Permissions = (1 << (iota - 1)) - 1
	// PermissionMaxInput is to be used within value range checks
	PermissionMaxInput = PermissionAll
	// PermissionMinInput is to be used within value range checks
	PermissionMinInput = PermissionRead
	// PermissionsNone is to be used to deny access on a resource
	PermissionsNone = 64
)

var (
	// ErrPermissionNotInRange defines a permission specific error.
	ErrPermissionNotInRange = fmt.Errorf("the provided permission is not between %d and %d", PermissionMinInput, PermissionMaxInput)
	// ErrZeroPermission defines a permission specific error
	ErrZeroPermission = errors.New("permission is zero")
)

// NewPermissions creates a new Permissions instance.
// The value must be in the valid range.
func NewPermissions(val int) (Permissions, error) {
	if val == int(PermissionInvalid) {
		return PermissionInvalid, ErrZeroPermission
	} else if val < int(PermissionInvalid) || int(PermissionMaxInput) < val {
		return PermissionInvalid, ErrPermissionNotInRange
	}
	return Permissions(val), nil
}

// Contain tests if the permissions contain another one.
func (p Permissions) Contain(other Permissions) bool {
	return p&other == other
}

func (p Permissions) String() string {
	return strconv.FormatUint(uint64(p), 10)
}
