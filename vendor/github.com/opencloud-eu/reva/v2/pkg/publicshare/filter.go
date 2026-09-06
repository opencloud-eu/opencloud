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

package publicshare

import (
	"path"
	"strings"

	provider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
)

// FilterResourceInfo rewrites a resource info for a public link consumer: the
// path becomes share-root relative (the structure above must not leak), the
// permissions are cut to the link grant. Consumers that reach link content by
// id bypass the publicstorageprovider and apply it themselves.
func FilterResourceInfo(info, shareRoot *provider.ResourceInfo, grant *provider.ResourcePermissions) {
	if info == nil {
		return
	}

	var sharePath string
	if shareRoot.GetType() == provider.ResourceType_RESOURCE_TYPE_FILE {
		sharePath = path.Base(shareRoot.GetPath())
	} else {
		sharePath = strings.TrimPrefix(info.GetPath(), shareRoot.GetPath())
	}
	info.Path = path.Join("/", sharePath)

	if info.PermissionSet != nil {
		FilterPermissions(info.PermissionSet, grant)
	}
}

// FilterPermissions reduces l to what r also grants. A nil r clears l.
func FilterPermissions(l, r *provider.ResourcePermissions) {
	l.AddGrant = l.AddGrant && r.GetAddGrant()
	l.CreateContainer = l.CreateContainer && r.GetCreateContainer()
	l.Delete = l.Delete && r.GetDelete()
	l.DenyGrant = l.DenyGrant && r.GetDenyGrant()
	l.GetPath = l.GetPath && r.GetGetPath()
	l.GetQuota = l.GetQuota && r.GetGetQuota()
	l.InitiateFileDownload = l.InitiateFileDownload && r.GetInitiateFileDownload()
	l.InitiateFileUpload = l.InitiateFileUpload && r.GetInitiateFileUpload()
	l.ListContainer = l.ListContainer && r.GetListContainer()
	l.ListFileVersions = l.ListFileVersions && r.GetListFileVersions()
	l.ListGrants = l.ListGrants && r.GetListGrants()
	l.ListRecycle = l.ListRecycle && r.GetListRecycle()
	l.Move = l.Move && r.GetMove()
	l.PurgeRecycle = l.PurgeRecycle && r.GetPurgeRecycle()
	l.RemoveGrant = l.RemoveGrant && r.GetRemoveGrant()
	l.RestoreFileVersion = l.RestoreFileVersion && r.GetRestoreFileVersion()
	l.RestoreRecycleItem = l.RestoreRecycleItem && r.GetRestoreRecycleItem()
	l.Stat = l.Stat && r.GetStat()
	l.UpdateGrant = l.UpdateGrant && r.GetUpdateGrant()
}
