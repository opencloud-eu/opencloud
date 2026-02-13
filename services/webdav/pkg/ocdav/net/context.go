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
	"context"

	userv1beta1 "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"
	provider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	ctxpkg "github.com/opencloud-eu/reva/v2/pkg/ctx"
)

// IsCurrentUserOwnerOrManager returns whether the context user is the given owner or not
func IsCurrentUserOwnerOrManager(ctx context.Context, owner *userv1beta1.UserId, md *provider.ResourceInfo) bool {
	contextUser, ok := ctxpkg.ContextGetUser(ctx)
	// personal spaces have owners
	if ok && contextUser.Id != nil && owner != nil &&
		contextUser.Id.Idp == owner.Idp &&
		contextUser.Id.OpaqueId == owner.OpaqueId {
		return true
	}
	// check if the user is space manager
	if md != nil && md.Owner != nil && md.Owner.GetType() == userv1beta1.UserType_USER_TYPE_SPACE_OWNER {
		return md.GetPermissionSet().AddGrant
	}
	return false
}
