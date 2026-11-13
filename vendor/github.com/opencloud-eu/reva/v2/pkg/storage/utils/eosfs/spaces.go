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

package eosfs

import (
	"context"
	"fmt"
	"path"
	"regexp"
	"strconv"
	"strings"
	"time"

	userpb "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"
	rpc "github.com/cs3org/go-cs3apis/cs3/rpc/v1beta1"
	provider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	types "github.com/cs3org/go-cs3apis/cs3/types/v1beta1"
	"github.com/opencloud-eu/reva/v2/pkg/appctx"
	"github.com/opencloud-eu/reva/v2/pkg/eosclient"
	"github.com/opencloud-eu/reva/v2/pkg/errtypes"
	"github.com/opencloud-eu/reva/v2/pkg/storage"
	"github.com/opencloud-eu/reva/v2/pkg/storage/utils/templates"
	"github.com/opencloud-eu/reva/v2/pkg/storagespace"
	"github.com/pkg/errors"
)

const (
	spaceTypePersonal = "personal"
	spaceTypeProject  = "project"
	// spaceTypeShare    = "share"
)

// SpacesConfig specifies the required configuration parameters needed
// to connect to the project spaces DB
type SpacesConfig struct {
	Enabled    bool   `mapstructure:"enabled"`
	DbUsername string `mapstructure:"db_username"`
	DbPassword string `mapstructure:"db_password"`
	DbHost     string `mapstructure:"db_host"`
	DbName     string `mapstructure:"db_name"`
	DbTable    string `mapstructure:"db_table"`
	DbPort     int    `mapstructure:"db_port"`
}

var (
	egroupRegex = regexp.MustCompile(`^cernbox-project-(?P<Name>.+)-(?P<Permissions>admins|writers|readers)\z`)
)

func (fs *eosfs) ListStorageSpaces(ctx context.Context, filter []*provider.ListStorageSpacesRequest_Filter, unrestricted bool) ([]*provider.StorageSpace, error) {
	u, err := getUser(ctx)
	if err != nil {
		err = errors.Wrap(err, "eosfs: wrap: no user in ctx")
		return nil, err
	}

	spaceID, spaceType, spacePath := "", "", ""

	for i := range filter {
		switch filter[i].Type {
		case provider.ListStorageSpacesRequest_Filter_TYPE_ID:
			_, spaceID, _, _ = storagespace.SplitID(filter[i].GetId().OpaqueId)
		case provider.ListStorageSpacesRequest_Filter_TYPE_PATH:
			spacePath = filter[i].GetPath()
		case provider.ListStorageSpacesRequest_Filter_TYPE_SPACE_TYPE:
			spaceType = filter[i].GetSpaceType()
		}
	}

	if spaceType != "" && spaceType != spaceTypePersonal && spaceType != spaceTypeProject {
		spaceType = ""
	}

	cachedSpaces, err := fs.fetchCachedSpaces(ctx, u, spaceType, spaceID, spacePath)
	if err == nil {
		return cachedSpaces, nil
	}

	spaces := []*provider.StorageSpace{}

	if !fs.conf.SpacesConfig.Enabled && (spaceType == "" || spaceType == spaceTypePersonal) {
		personalSpaces, err := fs.listPersonalStorageSpaces(ctx, u, spaceID, spacePath)
		if err != nil {
			return nil, err
		}
		spaces = append(spaces, personalSpaces...)
	}
	if fs.conf.SpacesConfig.Enabled && (spaceType == "" || spaceType == spaceTypeProject) {
		projectSpaces, err := fs.listProjectStorageSpaces(ctx, u, spaceID, spacePath)
		if err != nil {
			return nil, err
		}
		spaces = append(spaces, projectSpaces...)
	}

	fs.cacheSpaces(ctx, u, spaceType, spaceID, spacePath, spaces)
	return spaces, nil
}

func (fs *eosfs) listPersonalStorageSpaces(ctx context.Context, u *userpb.User, spaceID, spacePath string) ([]*provider.StorageSpace, error) {
	var eosFileInfo *eosclient.FileInfo
	// if no spaceID and spacePath are provided, we just return the user home
	switch {
	case spaceID == "" && spacePath == "":
		fn, err := fs.wrapUserHomeStorageSpaceID(ctx, u, "/")
		if err != nil {
			return nil, err
		}

		auth, err := fs.getUserAuth(ctx, u, fn)
		if err != nil {
			return nil, err
		}
		eosFileInfo, err = fs.c.GetFileInfoByPath(ctx, auth, fn)
		if err != nil {
			return nil, err
		}
	case spacePath == "":
		// else, we'll stat the resource by inode
		auth, err := fs.getUserAuth(ctx, u, "")
		if err != nil {
			return nil, err
		}

		inode, err := strconv.ParseUint(spaceID, 10, 64)
		if err != nil {
			return nil, err
		}

		eosFileInfo, err = fs.c.GetFileInfoByInode(ctx, auth, inode)
		if err != nil {
			return nil, err
		}
	default:
		fn := fs.wrap(ctx, spacePath)
		auth, err := fs.getUserAuth(ctx, u, fn)
		if err != nil {
			return nil, err
		}
		eosFileInfo, err = fs.c.GetFileInfoByPath(ctx, auth, fn)
		if err != nil {
			return nil, err
		}
	}

	md, err := fs.convertToResourceInfo(ctx, eosFileInfo)
	if err != nil {
		return nil, err
	}

	// If the request was for a relative ref, return just the base path
	if !strings.HasPrefix(spacePath, "/") {
		md.Path = path.Base(md.Path)
	}

	return []*provider.StorageSpace{{
		Id:        &provider.StorageSpaceId{OpaqueId: md.Id.OpaqueId},
		Name:      md.Owner.OpaqueId,
		SpaceType: "personal",
		Owner:     &userpb.User{Id: md.Owner},
		Root: &provider.ResourceId{
			StorageId: md.Id.OpaqueId,
			OpaqueId:  md.Id.OpaqueId,
		},
		Mtime: &types.Timestamp{
			Seconds: eosFileInfo.MTimeSec,
			Nanos:   eosFileInfo.MTimeNanos,
		},
		Quota: &provider.Quota{},
		Opaque: &types.Opaque{
			Map: map[string]*types.OpaqueEntry{
				"path": {
					Decoder: "plain",
					Value:   []byte(md.Path),
				},
			},
		},
	}}, nil
}

func (fs *eosfs) listProjectStorageSpaces(ctx context.Context, user *userpb.User, spaceID, spacePath string) ([]*provider.StorageSpace, error) {
	if !fs.conf.SpacesConfig.Enabled {
		return nil, errtypes.NotSupported("list storage spaces")
	}

	log := appctx.GetLogger(ctx)

	// Find all the project groups the user belongs to
	userProjectGroupsMap := make(map[string]bool)
	for _, group := range user.Groups {
		match := egroupRegex.FindStringSubmatch(group)
		if match != nil {
			userProjectGroupsMap[match[1]] = true
		}
	}

	if len(userProjectGroupsMap) == 0 {
		return nil, nil
	}

	query := "SELECT project_name, eos_relative_path FROM " + fs.conf.SpacesConfig.DbTable + " WHERE project_name in (?" + strings.Repeat(",?", len(userProjectGroupsMap)-1) + ")"
	params := []interface{}{}
	for k := range userProjectGroupsMap {
		params = append(params, k)
	}

	rows, err := fs.spacesDB.Query(query, params...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var dbProjects []*provider.StorageSpace
	for rows.Next() {
		var name, relPath string
		if err = rows.Scan(&name, &relPath); err == nil {
			info, err := fs.GetMD(ctx, &provider.Reference{Path: relPath}, []string{}, nil)
			if err == nil {
				if (spaceID == "" || spaceID == info.Id.OpaqueId) && (spacePath == "" || spacePath == relPath) {
					// If the request was for a relative ref, return just the base path
					if !strings.HasPrefix(spacePath, "/") {
						relPath = path.Base(relPath)
					}

					dbProjects = append(dbProjects, &provider.StorageSpace{
						Id:        &provider.StorageSpaceId{OpaqueId: name},
						Name:      name,
						SpaceType: "project",
						Owner: &userpb.User{
							Id: info.Owner,
						},
						Root:  &provider.ResourceId{StorageId: info.Id.OpaqueId, OpaqueId: info.Id.OpaqueId},
						Mtime: info.Mtime,
						Quota: &provider.Quota{},
						Opaque: &types.Opaque{
							Map: map[string]*types.OpaqueEntry{
								"path": {
									Decoder: "plain",
									Value:   []byte(relPath),
								},
							},
						},
					})
				}

			} else {
				log.Error().Err(err).Str("path", relPath).Msgf("eosfs: error statting storage space")
			}
		}
	}

	return dbProjects, nil
}

func (fs *eosfs) fetchCachedSpaces(ctx context.Context, user *userpb.User, spaceType, spaceID, spacePath string) ([]*provider.StorageSpace, error) {
	key := user.Id.OpaqueId + ":" + spaceType + ":" + spaceID + ":" + spacePath
	if spacesIf, err := fs.spacesCache.Get(key); err == nil {
		log := appctx.GetLogger(ctx)
		log.Info().Msgf("found cached spaces %s", key)
		return spacesIf.([]*provider.StorageSpace), nil
	}
	return nil, errtypes.NotFound("eosfs: spaces not found in cache")
}

func (fs *eosfs) cacheSpaces(ctx context.Context, user *userpb.User, spaceType, spaceID, spacePath string, spaces []*provider.StorageSpace) {
	key := user.Id.OpaqueId + ":" + spaceType + ":" + spaceID + ":" + spacePath
	_ = fs.spacesCache.SetWithExpire(key, spaces, time.Second*time.Duration(60))
}

func (fs *eosfs) wrapUserHomeStorageSpaceID(ctx context.Context, u *userpb.User, fn string) (string, error) {
	layout := templates.WithUser(u, fs.conf.UserLayout)
	internal := path.Join(fs.conf.Namespace, layout, fn)

	appctx.GetLogger(ctx).Debug().Msg("eosfs: wrap storage space id=" + fn + " internal=" + internal)
	return internal, nil
}

func (fs *eosfs) CreateStorageSpace(ctx context.Context, req *provider.CreateStorageSpaceRequest) (*provider.CreateStorageSpaceResponse, error) {
	// The request is to create a user home
	if req.Type == spaceTypePersonal {
		u, err := getUser(ctx)
		if err != nil {
			err = errors.Wrap(err, "eosfs: wrap: no user in ctx")
			return nil, err
		}

		// We need the unique path corresponding to the user. We assume that the username is the ID, and determine the path based on a specified template
		fn, err := fs.wrapUserHomeStorageSpaceID(ctx, u, "/")
		if err != nil {
			return nil, err
		}

		err = fs.createNominalHome(ctx, fn)
		if err != nil {
			return nil, err
		}

		auth, err := fs.getUserAuth(ctx, u, fn)
		if err != nil {
			return nil, err
		}
		eosFileInfo, err := fs.c.GetFileInfoByPath(ctx, auth, fn)
		if err != nil {
			return nil, err
		}
		sid := fmt.Sprintf("%d", eosFileInfo.Inode)

		space := &provider.StorageSpace{
			Id:        &provider.StorageSpaceId{OpaqueId: sid},
			Name:      u.Id.OpaqueId,
			SpaceType: "personal",
			Owner:     u,
			Root: &provider.ResourceId{
				StorageId: sid,
				OpaqueId:  sid,
			},
			Mtime: &types.Timestamp{
				Seconds: eosFileInfo.MTimeSec,
				Nanos:   eosFileInfo.MTimeNanos,
			},
			Quota: &provider.Quota{},
			Opaque: &types.Opaque{
				Map: map[string]*types.OpaqueEntry{
					"path": {
						Decoder: "plain",
						Value:   []byte(path.Base(fn)),
					},
				},
			},
		}

		return &provider.CreateStorageSpaceResponse{
			Status: &rpc.Status{
				Code: rpc.Code_CODE_OK,
			},
			StorageSpace: space,
		}, nil

	}

	// We don't support creating any other types of shares (projects or spaces)
	return nil, errtypes.NotSupported("eosfs: creating storage spaces of specified type is not supported")
}

func (fs *eosfs) UpdateStorageSpace(ctx context.Context, req *provider.UpdateStorageSpaceRequest) (*provider.UpdateStorageSpaceResponse, error) {
	return nil, errtypes.NotSupported("update storage space")
}

func (fs *eosfs) DeleteStorageSpace(ctx context.Context, req *provider.DeleteStorageSpaceRequest) (*storage.DeleteStorageSpaceResult, error) {
	return nil, errtypes.NotSupported("delete storage spaces")
}
