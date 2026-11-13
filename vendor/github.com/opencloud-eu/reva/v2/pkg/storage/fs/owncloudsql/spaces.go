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

package owncloudsql

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	userpb "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"
	provider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	types "github.com/cs3org/go-cs3apis/cs3/types/v1beta1"
	ctxpkg "github.com/opencloud-eu/reva/v2/pkg/ctx"
	"github.com/opencloud-eu/reva/v2/pkg/errtypes"
	"github.com/opencloud-eu/reva/v2/pkg/storage"
	"github.com/opencloud-eu/reva/v2/pkg/storage/fs/owncloudsql/filecache"
	"github.com/opencloud-eu/reva/v2/pkg/storagespace"
	"github.com/opencloud-eu/reva/v2/pkg/utils"
)

// ListStorageSpaces lists storage spaces according to the provided filters
func (fs *owncloudsqlfs) ListStorageSpaces(ctx context.Context, filter []*provider.ListStorageSpacesRequest_Filter, unrestricted bool) ([]*provider.StorageSpace, error) {
	var (
		spaceID = "*"
		// uid     *userpb.UserId
	)

	filteringUnsupportedSpaceTypes := false

	for i := range filter {
		switch filter[i].Type {
		case provider.ListStorageSpacesRequest_Filter_TYPE_SPACE_TYPE:
			t := filter[i].GetSpaceType()
			filteringUnsupportedSpaceTypes = (t != "personal" && !strings.HasPrefix(t, "+"))
		case provider.ListStorageSpacesRequest_Filter_TYPE_ID:
			_, spaceID, _, _ = storagespace.SplitID(filter[i].GetId().OpaqueId)
		case provider.ListStorageSpacesRequest_Filter_TYPE_USER:
			// uid = filter[i].GetUser()
		}
	}
	if filteringUnsupportedSpaceTypes {
		// owncloudsql only supports personal spaces, no need to search for something else
		return []*provider.StorageSpace{}, nil
	}

	spaces := []*provider.StorageSpace{}
	if spaceID == "*" {
		u, ok := ctxpkg.ContextGetUser(ctx)
		if !ok {
			return nil, errtypes.UserRequired("error getting user from context")
		}
		// if uid != nil && utils.UserIDEqual(uid, u.Id) {
		//       return nil, errtypes.PermissionDenied("cannot access space of other user?")
		// }
		space, err := fs.getPersonalSpace(ctx, u)
		if err != nil {
			return nil, err
		}
		spaces = append(spaces, space)
	} else {
		id, err := strconv.Atoi(spaceID)
		if err != nil {
			// non-numeric space id -> this request is not for us
			return []*provider.StorageSpace{}, nil
		}
		space, err := fs.getSpaceByNumericID(ctx, id)
		if err != nil {
			return nil, err
		}
		spaces = append(spaces, space)
	}

	for _, space := range spaces {
		space.HasTrashedItems = false // owncloudsql does not support checking for trashed items
	}

	return spaces, nil
}

// CreateStorageSpace creates a storage space
func (fs *owncloudsqlfs) CreateStorageSpace(ctx context.Context, req *provider.CreateStorageSpaceRequest) (*provider.CreateStorageSpaceResponse, error) {
	return nil, errtypes.NotSupported("unimplemented: CreateStorageSpace")
}

// UpdateStorageSpace updates a storage space
func (fs *owncloudsqlfs) UpdateStorageSpace(ctx context.Context, req *provider.UpdateStorageSpaceRequest) (*provider.UpdateStorageSpaceResponse, error) {
	return nil, errtypes.NotSupported("update storage space")
}

// DeleteStorageSpace deletes a storage space
func (fs *owncloudsqlfs) DeleteStorageSpace(ctx context.Context, req *provider.DeleteStorageSpaceRequest) (*storage.DeleteStorageSpaceResult, error) {
	return nil, errtypes.NotSupported("delete storage space")
}

//  Note: currently unused but will be used later
// func (fs *owncloudsqlfs) listAllPersonalSpaces(ctx context.Context) ([]*provider.StorageSpace, error) {
// 	storages, err := fs.filecache.ListStorages(true)
// 	if err != nil {
// 		return nil, err
// 	}
// 	spaces := []*provider.StorageSpace{}
// 	for _, storage := range storages {
// 		space, err := fs.storageToSpace(ctx, storage)
// 		if err != nil {
// 			return nil, err
// 		}
// 		spaces = append(spaces, space)
// 	}
// 	return spaces, nil
// }

func (fs *owncloudsqlfs) getPersonalSpace(ctx context.Context, owner *userpb.User) (*provider.StorageSpace, error) {
	storageID, err := fs.filecache.GetNumericStorageID(ctx, "home::"+owner.Username)
	if err != nil {
		return nil, err
	}
	storage, err := fs.filecache.GetStorage(ctx, storageID)
	if err != nil {
		return nil, err
	}
	root, err := fs.filecache.Get(ctx, storage.NumericID, "")
	if err != nil {
		return nil, err
	}

	space := &provider.StorageSpace{
		Id: &provider.StorageSpaceId{OpaqueId: strconv.Itoa(storage.NumericID)},
		Root: &provider.ResourceId{
			// return ownclouds numeric storage id as the space id!
			SpaceId:  strconv.Itoa(storage.NumericID),
			OpaqueId: strconv.Itoa(root.ID),
		},
		Name:      owner.Username,
		SpaceType: "personal",
		Mtime:     &types.Timestamp{Seconds: uint64(root.MTime)},
		Owner:     owner,
	}
	space.Opaque = utils.AppendPlainToOpaque(space.Opaque, "spaceAlias", "personal/"+owner.Username)
	space.Opaque = utils.AppendPlainToOpaque(space.Opaque, "etag", fmt.Sprintf(`"%s"`, root.Etag))
	return space, nil
}

func (fs *owncloudsqlfs) getSpaceByNumericID(ctx context.Context, spaceID int) (*provider.StorageSpace, error) {
	storage, err := fs.filecache.GetStorage(ctx, spaceID)
	if err != nil {
		return nil, err
	}
	if !strings.HasPrefix(storage.ID, "home::") {
		return nil, fmt.Errorf("only personal spaces are supported")
	}

	return fs.storageToSpace(ctx, storage)
}

func (fs *owncloudsqlfs) storageToSpace(ctx context.Context, storage *filecache.Storage) (*provider.StorageSpace, error) {
	root, err := fs.filecache.Get(ctx, storage.NumericID, "")
	if err != nil {
		return nil, err
	}
	ownerName := strings.TrimPrefix(storage.ID, "home::")
	owner, err := fs.getUser(ctx, ownerName)
	if err != nil {
		return nil, err
	}

	space := &provider.StorageSpace{
		Id: &provider.StorageSpaceId{OpaqueId: strconv.Itoa(storage.NumericID)},
		Root: &provider.ResourceId{
			// return ownclouds numeric storage id as the space id!
			SpaceId:  strconv.Itoa(storage.NumericID),
			OpaqueId: strconv.Itoa(root.ID),
		},
		Name:      owner.Username,
		SpaceType: "personal",
		Mtime:     &types.Timestamp{Seconds: uint64(root.MTime)},
		Owner:     owner,
	}
	space.Opaque = utils.AppendPlainToOpaque(space.Opaque, "spaceAlias", "personal/"+owner.Username)
	space.Opaque = utils.AppendPlainToOpaque(space.Opaque, "etag", fmt.Sprintf(`"%s"`, root.Etag))
	return space, nil
}
