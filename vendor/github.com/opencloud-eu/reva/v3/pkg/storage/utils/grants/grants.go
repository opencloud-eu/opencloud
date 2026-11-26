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

package grants

import (
	"errors"
	"strings"

	provider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	"github.com/opencloud-eu/reva/v3/pkg/storage/utils/acl"
	"google.golang.org/protobuf/proto"
)

// GetACLPerm generates a string representation of CS3APIs' ResourcePermissions
// TODO(labkode): fine grained permission controls.
func GetACLPerm(set *provider.ResourcePermissions) (string, error) {
	// resource permission is denied
	if proto.Equal(&provider.ResourcePermissions{}, set) {
		return "!r!w!x!m!u!d", nil
	}

	var b strings.Builder

	if set.Stat || set.InitiateFileDownload {
		b.WriteString("r")
	}
	if set.CreateContainer || set.InitiateFileUpload || set.Delete || set.Move {
		b.WriteString("w")
	}
	if set.ListContainer || set.ListFileVersions {
		b.WriteString("x")
	}
	if set.AddGrant || set.ListGrants || set.RemoveGrant {
		b.WriteString("m")
	}
	if set.GetQuota {
		b.WriteString("q")
	}

	if set.Delete {
		b.WriteString("+d")
	} else {
		b.WriteString("!d")
	}

	return b.String(), nil
}

// GetGrantPermissionSet converts CS3APIs' ResourcePermissions from a string
// TODO(labkode): add more fine grained controls.
// EOS acls are a mix of ACLs and POSIX permissions. More details can be found in
// https://github.com/cern-eos/eos/blob/master/doc/configuration/permission.rst
func GetGrantPermissionSet(perm string) *provider.ResourcePermissions {
	var rp provider.ResourcePermissions // default to 0 == all denied

	if strings.Contains(perm, "r") && !strings.Contains(perm, "!r") {
		rp.GetPath = true
		rp.Stat = true
		rp.InitiateFileDownload = true
	}

	if strings.Contains(perm, "w") && !strings.Contains(perm, "!w") {
		rp.Move = true
		rp.Delete = true
		rp.PurgeRecycle = true
		rp.InitiateFileUpload = true
		rp.RestoreFileVersion = true
		rp.RestoreRecycleItem = true
		rp.CreateContainer = true
	}

	if strings.Contains(perm, "x") && !strings.Contains(perm, "!x") {
		rp.ListFileVersions = true
		rp.ListRecycle = true
		rp.ListContainer = true
	}

	if strings.Contains(perm, "!d") {
		rp.Delete = false
		rp.PurgeRecycle = false
	}

	if strings.Contains(perm, "m") && !strings.Contains(perm, "!m") {
		rp.AddGrant = true
		rp.ListGrants = true
		rp.RemoveGrant = true
	}

	if strings.Contains(perm, "q") && !strings.Contains(perm, "!q") {
		rp.GetQuota = true
	}

	return &rp
}

// GetACLType returns a char representation of the type of grantee
func GetACLType(gt provider.GranteeType) (string, error) {
	switch gt {
	case provider.GranteeType_GRANTEE_TYPE_USER:
		return acl.TypeUser, nil
	case provider.GranteeType_GRANTEE_TYPE_GROUP:
		return acl.TypeGroup, nil
	default:
		return "", errors.New("no eos acl for grantee type: " + gt.String())
	}
}

// GetGranteeType returns the grantee type from a char
func GetGranteeType(aclType string) provider.GranteeType {
	switch aclType {
	case acl.TypeUser:
		return provider.GranteeType_GRANTEE_TYPE_USER
	case acl.TypeGroup:
		return provider.GranteeType_GRANTEE_TYPE_GROUP
	default:
		return provider.GranteeType_GRANTEE_TYPE_INVALID
	}
}

// PermissionsEqual returns true if the permissions are equal
func PermissionsEqual(p1, p2 *provider.ResourcePermissions) bool {
	return p1 != nil && p2 != nil && proto.Equal(p1, p2)
}

// GranteeEqual returns true if the grantee are equal
func GranteeEqual(g1, g2 *provider.Grantee) bool {
	return g1 != nil && g2 != nil && proto.Equal(g1, g2)
}
