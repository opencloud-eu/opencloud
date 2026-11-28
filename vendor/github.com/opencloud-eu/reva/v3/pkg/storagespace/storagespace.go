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

package storagespace

import (
	"path"
	"strings"

	provider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	"github.com/opencloud-eu/reva/v3/pkg/utils"
	"github.com/pkg/errors"
)

const (
	_idDelimiter        string = "!"
	_storageIDDelimiter string = "$"
)

// The types of the spaces that hold files, see provider.StorageSpace.SpaceType. Other types, e.g.
// "mountpoint" or "virtual", are views onto files in such spaces, which the providers of shares
// and public links make up.
const (
	// TypePersonal is the type of the personal space of a user.
	TypePersonal = "personal"
	// TypeProject is the type of a space that space managers create, e.g. for a team or a
	// project, and share with its members.
	TypeProject = "project"
)

var (
	// ErrInvalidSpaceReference signals that the space reference is invalid.
	ErrInvalidSpaceReference = errors.New("invalid storage space reference")
	// ErrInvalidSpaceID signals that the space ID is invalid.
	ErrInvalidSpaceID = errors.New("invalid storage space id")
)

// SplitID splits a storage space ID into a provider ID and a node ID.
// The accepted formats results of the storage space ID and respective results
// are:
// <storageid>$<spaceid>!<nodeid> 	-> <storageid>, <spaceid>, <nodeid>
// <storageid>$<spaceid>			-> <storageid>, <spaceid>, ""
// <spaceid>						-> "", <spaceid>, ""
func SplitID(ssid string) (storageid, spaceid, nodeid string, err error) {
	if ssid == "" {
		return "", "", "", errors.Wrap(ErrInvalidSpaceID, "can't split empty storage space ID")
	}

	storageid, snid := SplitStorageID(ssid)
	parts := strings.SplitN(snid, _idDelimiter, 2)
	if len(parts) == 1 || parts[1] == "" {
		return storageid, parts[0], "", nil
	}

	return storageid, parts[0], parts[1], nil
}

// SplitStorageID splits a storage ID into the storage ID and the spaceID.
// The accepted formats are:
// <storageid>$<spaceid>			-> <storageid>, <spaceid>
// <spaceid>						-> "", <spaceid>
func SplitStorageID(sid string) (storageID, spaceID string) {
	parts := strings.SplitN(sid, _storageIDDelimiter, 2)
	if len(parts) == 2 {
		return parts[0], parts[1]
	}
	return "", parts[0]
}

// FormatResourceID converts a ResourceId into the string format.
// The result format will look like:
// <storageid>$<spaceid>!<opaqueid>
func FormatResourceID(sid *provider.ResourceId) string {
	if sid.GetOpaqueId() == "" {
		return FormatStorageID(sid.GetStorageId(), sid.GetSpaceId())
	}
	return strings.Join([]string{FormatStorageID(sid.StorageId, sid.SpaceId), sid.OpaqueId}, _idDelimiter)
}

// FormatStorageID converts the provider ID and space ID into the string format.
// The result format will look like:
// <storageid>$<spaceid> or
// <spaceid> in case the provider ID is empty.
func FormatStorageID(storageID, spaceID string) string {
	if storageID == "" {
		return spaceID
	}
	return strings.Join([]string{storageID, spaceID}, _storageIDDelimiter)
}

// ParseID parses a storage space ID and returns a storageprovider ResourceId.
// The accepted formats are:
// <providerid>$<spaceid>!<nodeid> 	-> <providerid>$<spaceid>, <nodeid>
// <providerid>$<spaceid>			-> <providerid>$<spaceid>, <spaceid>
// <spaceid>						-> <spaceid>, <spaceid>
func ParseID(ssid string) (provider.ResourceId, error) {
	sid, spid, oid, err := SplitID(ssid)
	return provider.ResourceId{
		StorageId: sid,
		SpaceId:   spid,
		OpaqueId:  oid,
	}, err
}

// ParseReference parses a string into a spaces reference.
// The expected format is `<providerid>$<spaceid>!<nodeid>/<path>`.
func ParseReference(sRef string) (provider.Reference, error) {
	parts := strings.SplitN(sRef, "/", 2)

	rid, err := ParseID(parts[0])
	if err != nil {
		return provider.Reference{}, err
	}

	var path string
	if len(parts) == 2 {
		path = parts[1]
	}

	return provider.Reference{
		ResourceId: &rid,
		Path:       utils.MakeRelativePath(path),
	}, nil
}

// FormatReference will format a storage space reference into a string representation.
// If ref or ref.ResourceId are nil an error will be returned.
// The function doesn't check if all values are set.
// The resulting format can be:
//
// "storage_id!opaque_id"
// "storage_id!opaque_id/path"
// "storage_id/path"
// "storage_id"
func FormatReference(ref *provider.Reference) (string, error) {
	if ref == nil || ref.ResourceId == nil || ref.ResourceId.SpaceId == "" {
		return "", ErrInvalidSpaceReference
	}
	ssid := FormatResourceID(ref.ResourceId)
	return path.Join(ssid, ref.Path), nil
}

// UpdateLegacyResourceID checks if the given resource id contains a correct triple and will convert legacy ids without a spaceid
// by splitting the storageid.
func UpdateLegacyResourceID(id *provider.ResourceId) *provider.ResourceId {
	if storageid, spaceid := SplitStorageID(id.StorageId); storageid != "" && id.SpaceId == "" {
		return &provider.ResourceId{
			StorageId: storageid,
			SpaceId:   spaceid,
			OpaqueId:  id.OpaqueId,
		}
	}
	return id
}
