// Copyright 2018-2026 CERN
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
	"encoding/json"

	link "github.com/cs3org/go-cs3apis/cs3/sharing/link/v1beta1"
	provider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"

	"github.com/opencloud-eu/reva/v2/pkg/conversions"
)

// WebDAVPermissions derives the WebDAV permissions string (the OC-Perm header value) for a
// resource. It is shared by the ocdav gateway's creation-with-upload response and the
// dataprovider's chunked TUS finalize response, so the two report the same permissions.
func WebDAVPermissions(ctx context.Context, ri *provider.ResourceInfo) string {
	isPublic := false
	if o := ri.GetOpaque(); o != nil && o.Map != nil {
		if e := o.Map["link-share"]; e != nil && e.Decoder == "json" {
			ls := &link.PublicShare{}
			_ = json.Unmarshal(e.Value, ls)
			isPublic = ls != nil
		}
	}
	isShared := !IsCurrentUserOwnerOrManager(ctx, ri.GetOwner(), ri)
	role := conversions.RoleFromResourcePermissions(ri.GetPermissionSet(), isPublic)
	return role.WebDAVPermissions(
		ri.GetType() == provider.ResourceType_RESOURCE_TYPE_CONTAINER,
		isShared,
		false,
		isPublic,
	)
}
