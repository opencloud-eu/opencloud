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

package decomposedfs

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"maps"

	userv1beta1 "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"
	v1beta11 "github.com/cs3org/go-cs3apis/cs3/rpc/v1beta1"
	provider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	types "github.com/cs3org/go-cs3apis/cs3/types/v1beta1"
	"github.com/opencloud-eu/reva/v2/internal/grpc/services/storageprovider"
	"github.com/opencloud-eu/reva/v2/pkg/appctx"
	ocsconv "github.com/opencloud-eu/reva/v2/pkg/conversions"
	ctxpkg "github.com/opencloud-eu/reva/v2/pkg/ctx"
	"github.com/opencloud-eu/reva/v2/pkg/errtypes"
	"github.com/opencloud-eu/reva/v2/pkg/rgrpc/status"
	"github.com/opencloud-eu/reva/v2/pkg/rgrpc/todo/pool"
	sdk "github.com/opencloud-eu/reva/v2/pkg/sdk/common"
	"github.com/opencloud-eu/reva/v2/pkg/storage/fs/posix/lookup"
	"github.com/opencloud-eu/reva/v2/pkg/storage/pkg/decomposedfs/metadata/prefixes"
	"github.com/opencloud-eu/reva/v2/pkg/storage/pkg/decomposedfs/node"
	"github.com/opencloud-eu/reva/v2/pkg/storage/pkg/decomposedfs/permissions"
	"github.com/opencloud-eu/reva/v2/pkg/storage/utils/templates"
	"github.com/opencloud-eu/reva/v2/pkg/storagespace"
	"github.com/opencloud-eu/reva/v2/pkg/utils"
	"github.com/pkg/errors"
	"github.com/vmihailenco/msgpack/v5"
	"golang.org/x/sync/errgroup"
)

const (
	_spaceTypePersonal = "personal"
	_spaceTypeProject  = "project"
	spaceTypeShare     = "share"
	spaceTypeAny       = "*"
	spaceIDAny         = "*"

	quotaUnrestricted = 0
)

// CreateStorageSpace creates a storage space
func (fs *Decomposedfs) CreateStorageSpace(ctx context.Context, req *provider.CreateStorageSpaceRequest) (*provider.CreateStorageSpaceResponse, error) {
	ctx = storageprovider.WithSpaceType(ctx, "")
	u := ctxpkg.ContextMustGetUser(ctx)

	// "everything is a resource" this is the unique ID for the Space resource.
	spaceID, err := fs.lu.GenerateSpaceID(req.Type, req.GetOwner())
	if err != nil {
		return nil, err
	}
	if reqSpaceID := utils.ReadPlainFromOpaque(req.Opaque, "spaceid"); reqSpaceID != "" {
		spaceID = reqSpaceID
	}

	// Check if space already exists
	rootPath := ""
	switch req.Type {
	case _spaceTypePersonal:
		if fs.o.PersonalSpacePathTemplate != "" {
			rootPath = filepath.Join(fs.o.Root, templates.WithUser(u, fs.o.PersonalSpacePathTemplate))
		}
	default:
		if fs.o.GeneralSpacePathTemplate != "" {
			rootPath = filepath.Join(fs.o.Root, templates.WithSpacePropertiesAndUser(u, req.Type, req.Name, spaceID, fs.o.GeneralSpacePathTemplate))
		}
	}
	if rootPath != "" {
		if _, err := os.Stat(rootPath); err == nil {
			return nil, errtypes.AlreadyExists("decomposedfs: spaces: space already exists")
		}
	}

	description := utils.ReadPlainFromOpaque(req.Opaque, "description")
	alias := utils.ReadPlainFromOpaque(req.Opaque, "spaceAlias")
	if alias == "" {
		alias = templates.WithSpacePropertiesAndUser(u, req.Type, req.Name, spaceID, fs.o.GeneralSpaceAliasTemplate)
	}
	if req.Type == _spaceTypePersonal {
		alias = templates.WithSpacePropertiesAndUser(u, req.Type, req.Name, spaceID, fs.o.PersonalSpaceAliasTemplate)
	}

	root, err := node.ReadNode(ctx, fs.lu, spaceID, spaceID, true, nil, false) // will fall into `Exists` case below
	switch {
	case err != nil:
		return nil, err
	case !fs.p.CreateSpace(ctx, spaceID):
		return nil, errtypes.PermissionDenied(spaceID)
	case root.Exists:
		return nil, errtypes.AlreadyExists("decomposedfs: spaces: space already exists")
	}

	// create a directory node
	root.SetType(provider.ResourceType_RESOURCE_TYPE_CONTAINER)
	if rootPath == "" {
		rootPath = root.InternalPath()
	}

	// set 755 permissions for the base dir
	if err := os.MkdirAll(filepath.Dir(rootPath), 0755); err != nil {
		return nil, errors.Wrap(err, fmt.Sprintf("Decomposedfs: error creating spaces base dir %s", filepath.Dir(rootPath)))
	}

	// 770 permissions for the space
	if err := os.Mkdir(rootPath, 0770); err != nil {
		if os.IsExist(err) {
			// Someone has created the space in the meantime. Abort.
			return nil, errtypes.AlreadyExists(spaceID)
		}
		return nil, errors.Wrap(err, fmt.Sprintf("Decomposedfs: error creating space %s", rootPath))
	}

	// Store id in cache
	if c, ok := fs.lu.(node.IDCacher); ok {
		if err := c.CacheID(ctx, spaceID, spaceID, rootPath); err != nil {
			return nil, err
		}
	}

	if req.GetOwner() != nil && req.GetOwner().GetId() != nil {
		root.SetOwner(req.GetOwner().GetId())
	} else {
		root.SetOwner(&userv1beta1.UserId{
			OpaqueId: spaceID,
			TenantId: u.GetId().GetTenantId(),
			Type:     userv1beta1.UserType_USER_TYPE_SPACE_OWNER,
		})
	}

	metadata := node.Attributes{}
	metadata.SetString(prefixes.IDAttr, spaceID)
	metadata.SetString(prefixes.SpaceIDAttr, spaceID)
	metadata.SetString(prefixes.OwnerIDAttr, root.Owner().GetOpaqueId())
	metadata.SetString(prefixes.OwnerIDPAttr, root.Owner().GetIdp())
	metadata.SetString(prefixes.OwnerTypeAttr, utils.UserTypeToString(root.Owner().GetType()))

	if root.Owner().GetTenantId() != "" {
		metadata.SetString(prefixes.SpaceTenantIDAttr, root.Owner().GetTenantId())
	}
	// always mark the space root node as the end of propagation
	metadata.SetString(prefixes.PropagationAttr, "1")
	metadata.SetString(prefixes.NameAttr, req.Name)
	metadata.SetString(prefixes.SpaceNameAttr, req.Name)

	// This space is empty so set initial treesize to 0
	metadata.SetUInt64(prefixes.TreesizeAttr, 0)

	if req.Type != "" {
		metadata.SetString(prefixes.SpaceTypeAttr, req.Type)
	}

	if q := req.GetQuota(); q != nil {
		// set default space quota
		if fs.o.MaxQuota != quotaUnrestricted && q.GetQuotaMaxBytes() > fs.o.MaxQuota {
			return nil, errtypes.BadRequest("decompsedFS: requested quota is higher than allowed")
		}
		metadata.SetInt64(prefixes.QuotaAttr, int64(q.QuotaMaxBytes))
	} else if fs.o.MaxQuota != quotaUnrestricted {
		// If no quota was requested but a max quota was set then the the storage space has a quota
		// of max quota.
		metadata.SetInt64(prefixes.QuotaAttr, int64(fs.o.MaxQuota))
	}

	if description != "" {
		metadata.SetString(prefixes.SpaceDescriptionAttr, description)
	}

	if alias != "" {
		metadata.SetString(prefixes.SpaceAliasAttr, alias)
	}

	if err := root.SetXattrsWithContext(ctx, metadata, true); err != nil {
		return nil, err
	}

	// Write index
	err = fs.updateIndexes(ctx, &provider.Grantee{
		Type: provider.GranteeType_GRANTEE_TYPE_USER,
		Id:   &provider.Grantee_UserId{UserId: req.GetOwner().GetId()},
	}, req.Type, root.GetID(), root.GetID())
	if err != nil {
		return nil, err
	}

	ctx = storageprovider.WithSpaceType(ctx, req.Type)

	if req.Type != _spaceTypePersonal {
		if err := fs.AddGrant(ctx, &provider.Reference{
			ResourceId: &provider.ResourceId{
				SpaceId:  spaceID,
				OpaqueId: spaceID,
			},
		}, &provider.Grant{
			Grantee: &provider.Grantee{
				Type: provider.GranteeType_GRANTEE_TYPE_USER,
				Id: &provider.Grantee_UserId{
					UserId: u.GetId(),
				},
			},
			Permissions: ocsconv.NewManagerRole().CS3ResourcePermissions(),
		}); err != nil {
			return nil, err
		}
	}

	space, err := fs.StorageSpaceFromNode(ctx, root, true)
	if err != nil {
		return nil, err
	}

	resp := &provider.CreateStorageSpaceResponse{
		Status: &v1beta11.Status{
			Code: v1beta11.Code_CODE_OK,
		},
		StorageSpace: space,
	}
	return resp, nil
}

// ListStorageSpaces returns a list of StorageSpaces.
// The list can be filtered by space type or space id.
// Spaces are persisted with symlinks in /spaces/<type>/<spaceid> pointing to ../../nodes/<nodeid>, the root node of the space
// The spaceid is a concatenation of storageid + "!" + nodeid
func (fs *Decomposedfs) ListStorageSpaces(ctx context.Context, filter []*provider.ListStorageSpacesRequest_Filter, unrestricted bool) ([]*provider.StorageSpace, error) {
	// TODO check filters

	// TODO when a space symlink is broken delete the space for cleanup
	// read permissions are deduced from the node?

	// TODO for absolute references this actually requires us to move all user homes into a subfolder of /nodes/root,
	// e.g. /nodes/root/<space type> otherwise storage space names might collide even though they are of different types
	// /nodes/root/personal/foo and /nodes/root/shares/foo might be two very different spaces, a /nodes/root/foo is not expressive enough
	// we would not need /nodes/root if access always happened via spaceid+relative path

	var (
		spaceID         = spaceIDAny
		entry           = spaceIDAny
		requestedUserID *userv1beta1.UserId
	)

	spaceTypes := map[string]struct{}{}

	for i := range filter {
		switch filter[i].Type {
		case provider.ListStorageSpacesRequest_Filter_TYPE_SPACE_TYPE:
			switch filter[i].GetSpaceType() {
			case "+mountpoint":
				// TODO include mount poits
			case "+grant":
				// TODO include grants
			default:
				spaceTypes[filter[i].GetSpaceType()] = struct{}{}
			}
		case provider.ListStorageSpacesRequest_Filter_TYPE_ID:
			_, spaceID, entry, _ = storagespace.SplitID(filter[i].GetId().OpaqueId)
			if strings.Contains(entry, "/") {
				return []*provider.StorageSpace{}, nil
			}
		case provider.ListStorageSpacesRequest_Filter_TYPE_USER:
			// TODO: refactor this to GetUserId() in cs3
			requestedUserID = filter[i].GetUser()
		case provider.ListStorageSpacesRequest_Filter_TYPE_OWNER:
			// TODO: improve further by not evaluating shares
			requestedUserID = filter[i].GetOwner()
		}
	}
	if len(spaceTypes) == 0 {
		spaceTypes[spaceTypeAny] = struct{}{}
	}

	authenticatedUser := ctxpkg.ContextMustGetUser(ctx)
	authenticatedUserID := authenticatedUser.GetId().GetOpaqueId()

	if !fs.p.ListSpacesOfUser(ctx, requestedUserID) {
		return nil, errtypes.PermissionDenied(fmt.Sprintf("user %s is not allowed to list spaces of other users", authenticatedUserID))
	}

	checkNodePermissions := fs.MustCheckNodePermissions(ctx, unrestricted)

	spaces := []*provider.StorageSpace{}
	// build the glob path, eg.
	// /path/to/root/spaces/{spaceType}/{spaceId}
	// /path/to/root/spaces/personal/nodeid
	// /path/to/root/spaces/shared/nodeid

	if spaceID != spaceIDAny && entry != spaceIDAny {
		// try directly reading the node
		n, err := node.ReadNode(ctx, fs.lu, spaceID, entry, true, nil, false) // permission to read disabled space is checked later
		if err != nil {
			appctx.GetLogger(ctx).Error().Err(err).Str("id", entry).Msg("could not read node")
			return nil, err
		}
		if !n.Exists {
			// return empty list
			return spaces, nil
		}
		space, err := fs.StorageSpaceFromNode(ctx, n, checkNodePermissions)
		if err != nil {
			return nil, err
		}
		// filter space types
		_, ok1 := spaceTypes[spaceTypeAny]
		_, ok2 := spaceTypes[space.SpaceType]
		if ok1 || ok2 {
			spaces = append(spaces, space)
		}
		// TODO: filter user id
		return spaces, nil
	}

	matches := map[string]string{}
	var allMatches map[string]string
	var err error

	if requestedUserID != nil {
		allMatches, err = fs.userSpaceIndex.Load(requestedUserID.GetOpaqueId())
		// do not return an error if the user has no spaces
		if err != nil && !os.IsNotExist(err) {
			return nil, errors.Wrap(err, "error reading user index")
		}

		if entry == spaceIDAny {
			maps.Copy(matches, allMatches)
		} else {
			matches[allMatches[entry]] = allMatches[entry]
		}

		// get Groups for userid
		user := ctxpkg.ContextMustGetUser(ctx)
		// TODO the user from context may not have groups populated
		if !utils.UserIDEqual(user.GetId(), requestedUserID) {
			user, err = fs.UserIDToUserAndGroups(ctx, requestedUserID)
			if err != nil {
				return nil, err // TODO log and continue?
			}
		}

		for _, group := range user.Groups {
			allMatches, err = fs.groupSpaceIndex.Load(group)
			if err != nil {
				if os.IsNotExist(err) {
					continue // no spaces for this group
				}
				return nil, errors.Wrap(err, "error reading group index")
			}

			if entry == spaceIDAny {
				maps.Copy(matches, allMatches)
			} else {
				matches[allMatches[entry]] = allMatches[entry]
			}
		}

	}

	if requestedUserID == nil {
		if _, ok := spaceTypes[spaceTypeAny]; ok {
			// TODO do not hardcode dirs
			spaceTypes = map[string]struct{}{
				"personal": {},
				"project":  {},
				"share":    {},
			}
		}

		for spaceType := range spaceTypes {
			allMatches, err = fs.spaceTypeIndex.Load(spaceType)
			if err != nil {
				if os.IsNotExist(err) {
					continue // no spaces for this space type
				}
				return nil, errors.Wrap(err, "error reading type index")
			}

			if entry == spaceIDAny {
				maps.Copy(matches, allMatches)
			} else {
				matches[allMatches[entry]] = allMatches[entry]
			}
		}
	}

	// FIXME if the space does not exist try a node as the space root.

	// But then the whole /spaces/{spaceType}/{spaceid} becomes obsolete
	// we can alway just look up by nodeid
	// -> no. The /spaces folder is used for efficient lookup by type, otherwise we would have
	//    to iterate over all nodes and read the type from extended attributes
	// -> but for lookup by id we can use the node directly.
	// But what about sharding nodes by space?
	// an efficient lookup would be possible if we received a spaceid&opaqueid in the request
	// the personal spaces must also use the nodeid and not the name
	numShares := atomic.Int64{}
	errg, ctx := errgroup.WithContext(ctx)
	work := make(chan []string, len(matches))
	results := make(chan *provider.StorageSpace, len(matches))

	// Distribute work
	errg.Go(func() error {
		defer close(work)
		for spaceID, entry := range matches {
			select {
			case work <- []string{spaceID, entry}:
			case <-ctx.Done():
				return ctx.Err()
			}
		}
		return nil
	})

	// Spawn workers that'll concurrently work the queue
	numWorkers := 20
	if len(matches) < numWorkers {
		numWorkers = len(matches)
	}
	for i := 0; i < numWorkers; i++ {
		errg.Go(func() error {
			for match := range work {
				spaceID, err := fs.tp.ResolveSpaceIDIndexEntry(match[1])
				if err != nil {
					appctx.GetLogger(ctx).Error().Err(err).Str("id", spaceID).Msg("resolve space id index entry, skipping")
					continue
				}

				n, err := node.ReadNode(ctx, fs.lu, spaceID, spaceID, true, nil, true)
				if err != nil {
					appctx.GetLogger(ctx).Error().Err(err).Str("id", spaceID).Msg("could not read node, skipping")
					continue
				}

				if !n.Exists {
					continue
				}

				space, err := fs.StorageSpaceFromNode(ctx, n, checkNodePermissions)
				if err != nil {
					switch err.(type) {
					case errtypes.IsPermissionDenied:
						// ok
					case errtypes.NotFound:
						// ok
					default:
						appctx.GetLogger(ctx).Error().Err(err).Str("id", spaceID).Msg("could not convert to storage space")
					}
					continue
				}

				// FIXME type share evolved to grant on the edge branch ... make it configurable if the driver should support them or not for now ... ignore type share
				if space.SpaceType == spaceTypeShare {
					numShares.Add(1)
					// do not list shares as spaces for the owner
					continue
				}

				// TODO apply more filters
				_, ok1 := spaceTypes[spaceTypeAny]
				_, ok2 := spaceTypes[space.SpaceType]
				if ok1 || ok2 {
					select {
					case results <- space:
					case <-ctx.Done():
						return ctx.Err()
					}
				}
			}
			return nil
		})
	}

	// Wait for things to settle down, then close results chan
	go func() {
		_ = errg.Wait() // error is checked later
		close(results)
	}()

	for r := range results {
		if authenticatedUser.GetId().GetTenantId() != r.GetOwner().GetId().GetTenantId() {
			appctx.GetLogger(ctx).Warn().Str("id", r.GetId().GetOpaqueId()).Msg("space belongs to a different tenant")
			continue
		}
		r.HasTrashedItems = true
		resourceID, err := storagespace.ParseID(r.GetId().GetOpaqueId())
		if err != nil {
			appctx.GetLogger(ctx).Error().Err(err).Str("id", r.GetId().GetOpaqueId()).Msg("could not parse space id")
			r.HasTrashedItems = false
			continue
		}
		r.HasTrashedItems = !fs.trashbin.IsEmpty(ctx, resourceID.GetSpaceId())
		spaces = append(spaces, r)
	}

	// if there are no matches (or they happened to be spaces for the owner) and the node is a child return a space
	if int64(len(matches)) <= numShares.Load() && entry != spaceID {
		// try node id
		n, err := node.ReadNode(ctx, fs.lu, spaceID, entry, true, nil, false) // permission to read disabled space is checked in storageSpaceFromNode
		if err != nil {
			return nil, err
		}
		if n.Exists {
			space, err := fs.StorageSpaceFromNode(ctx, n, checkNodePermissions)
			if err != nil {
				return nil, err
			}
			spaces = append(spaces, space)
		}
	}

	return spaces, nil
}

// UserIDToUserAndGroups converts a user ID to a user with groups
func (fs *Decomposedfs) UserIDToUserAndGroups(ctx context.Context, userid *userv1beta1.UserId) (*userv1beta1.User, error) {
	user, err := fs.UserCache.Get(userid.GetOpaqueId())
	if err == nil {
		return user.(*userv1beta1.User), nil
	}

	gwConn, err := pool.GetGatewayServiceClient(fs.o.GatewayAddr)
	if err != nil {
		return nil, err
	}
	getUserResponse, err := gwConn.GetUser(ctx, &userv1beta1.GetUserRequest{
		UserId:                 userid,
		SkipFetchingUserGroups: false,
	})
	if err != nil {
		return nil, err
	}
	if getUserResponse.Status.Code != v1beta11.Code_CODE_OK {
		return nil, status.NewErrorFromCode(getUserResponse.Status.Code, "gateway")
	}
	_ = fs.UserCache.Set(userid.GetOpaqueId(), getUserResponse.GetUser())
	return getUserResponse.GetUser(), nil
}

// MustCheckNodePermissions checks if permission checks are needed to be performed when user requests spaces
func (fs *Decomposedfs) MustCheckNodePermissions(ctx context.Context, unrestricted bool) bool {
	// canListAllSpaces indicates if the user has the permission from the global user role
	canListAllSpaces := fs.p.ListAllSpaces(ctx)
	// unrestricted is the param which indicates if the user wants to list all spaces or only the spaces he is part of
	// if a user lists all spaces unrestricted and doesn't have the permissions from the role, we need to check
	// the nodePermissions and this will return a spaces list where the user has access to
	// we can only skip the NodePermissions check if both values are true
	if canListAllSpaces && unrestricted {
		return false
	}
	return true
}

// UpdateStorageSpace updates a storage space
func (fs *Decomposedfs) UpdateStorageSpace(ctx context.Context, req *provider.UpdateStorageSpaceRequest) (*provider.UpdateStorageSpaceResponse, error) {
	var restore bool
	if req.Opaque != nil {
		_, restore = req.Opaque.Map["restore"]
	}

	space := req.StorageSpace
	_, spaceID, _, _ := storagespace.SplitID(space.GetId().GetOpaqueId())

	metadata := make(node.Attributes, 5)
	if space.Name != "" {
		metadata.SetString(prefixes.NameAttr, space.Name)
		metadata.SetString(prefixes.SpaceNameAttr, space.Name)
	}

	if space.Quota != nil {
		if fs.o.MaxQuota != quotaUnrestricted && fs.o.MaxQuota < space.Quota.QuotaMaxBytes {
			return &provider.UpdateStorageSpaceResponse{
				Status: &v1beta11.Status{Code: v1beta11.Code_CODE_INVALID_ARGUMENT, Message: "decompsedFS: requested quota is higher than allowed"},
			}, nil
		} else if fs.o.MaxQuota != quotaUnrestricted && space.Quota.QuotaMaxBytes == quotaUnrestricted {
			// If the caller wants to unrestrict the space we give it the maximum allowed quota.
			space.Quota.QuotaMaxBytes = fs.o.MaxQuota
		}
		metadata.SetInt64(prefixes.QuotaAttr, int64(space.Quota.QuotaMaxBytes))
	}

	// TODO also return values which are not in the request
	if space.Opaque != nil {
		if description, ok := space.Opaque.Map["description"]; ok {
			metadata[prefixes.SpaceDescriptionAttr] = description.Value
		}
		if alias := utils.ReadPlainFromOpaque(space.Opaque, "spaceAlias"); alias != "" {
			metadata.SetString(prefixes.SpaceAliasAttr, alias)
		}
		if image := utils.ReadPlainFromOpaque(space.Opaque, "image"); image != "" {
			imageID, err := storagespace.ParseID(image)
			if err != nil {
				return &provider.UpdateStorageSpaceResponse{
					Status: &v1beta11.Status{Code: v1beta11.Code_CODE_NOT_FOUND, Message: "decomposedFS: space image resource not found"},
				}, nil
			}
			metadata.SetString(prefixes.SpaceImageAttr, imageID.OpaqueId)
		}
		if readme := utils.ReadPlainFromOpaque(space.Opaque, "readme"); readme != "" {
			readmeID, err := storagespace.ParseID(readme)
			if err != nil {
				return &provider.UpdateStorageSpaceResponse{
					Status: &v1beta11.Status{Code: v1beta11.Code_CODE_NOT_FOUND, Message: "decomposedFS: space readme resource not found"},
				}, nil
			}
			metadata.SetString(prefixes.SpaceReadmeAttr, readmeID.OpaqueId)
		}
	}

	// check which permissions are needed
	spaceNode, err := node.ReadNode(ctx, fs.lu, spaceID, spaceID, true, nil, false)
	if err != nil {
		return nil, err
	}

	if !spaceNode.Exists {
		return &provider.UpdateStorageSpaceResponse{
			Status: &v1beta11.Status{Code: v1beta11.Code_CODE_NOT_FOUND},
		}, nil
	}

	sp, err := fs.p.AssemblePermissions(ctx, spaceNode)
	if err != nil {
		return &provider.UpdateStorageSpaceResponse{
			Status: status.NewStatusFromErrType(ctx, "assembling permissions failed", err),
		}, nil

	}

	if !restore && len(metadata) == 0 && !permissions.IsViewer(sp) {
		// you may land here when making an update request without changes
		// check if user has access to the drive before continuing
		return &provider.UpdateStorageSpaceResponse{
			Status: &v1beta11.Status{Code: v1beta11.Code_CODE_NOT_FOUND},
		}, nil
	}

	if !permissions.IsManager(sp) {
		// We are not a space manager. We need to check for additional permissions.
		k := []string{prefixes.NameAttr, prefixes.SpaceDescriptionAttr}
		if !permissions.IsEditor(sp) {
			k = append(k, prefixes.SpaceReadmeAttr, prefixes.SpaceAliasAttr, prefixes.SpaceImageAttr)
		}

		if mapHasKey(metadata, k...) && !fs.p.ManageSpaceProperties(ctx, spaceID) {
			return &provider.UpdateStorageSpaceResponse{
				Status: &v1beta11.Status{Code: v1beta11.Code_CODE_PERMISSION_DENIED},
			}, nil
		}

		if restore && !fs.p.SpaceAbility(ctx, spaceID) {
			return &provider.UpdateStorageSpaceResponse{
				Status: &v1beta11.Status{Code: v1beta11.Code_CODE_NOT_FOUND},
			}, nil
		}
	}

	if mapHasKey(metadata, prefixes.QuotaAttr) {
		typ, err := spaceNode.SpaceRoot.Xattr(ctx, prefixes.SpaceTypeAttr)
		if err != nil {
			return &provider.UpdateStorageSpaceResponse{
				Status: &v1beta11.Status{
					Code:    v1beta11.Code_CODE_INTERNAL,
					Message: "space has no type",
				},
			}, nil
		}

		if !fs.p.SetSpaceQuota(ctx, spaceID, string(typ)) {
			return &provider.UpdateStorageSpaceResponse{
				Status: &v1beta11.Status{Code: v1beta11.Code_CODE_PERMISSION_DENIED},
			}, nil
		}
	}
	metadata[prefixes.TreeMTimeAttr] = []byte(time.Now().UTC().Format(time.RFC3339Nano))

	err = spaceNode.SetXattrsWithContext(ctx, metadata, true)
	if err != nil {
		return nil, err
	}

	if restore {
		if err := spaceNode.SetDTime(ctx, nil); err != nil {
			return nil, err
		}
	}

	// send back the updated data from the storage
	updatedSpace, err := fs.StorageSpaceFromNode(ctx, spaceNode, false)
	if err != nil {
		return nil, err
	}

	return &provider.UpdateStorageSpaceResponse{
		Status:       &v1beta11.Status{Code: v1beta11.Code_CODE_OK},
		StorageSpace: updatedSpace,
	}, nil
}

// DeleteStorageSpace deletes a storage space
func (fs *Decomposedfs) DeleteStorageSpace(ctx context.Context, req *provider.DeleteStorageSpaceRequest) error {
	opaque := req.Opaque
	var purge bool
	if opaque != nil {
		_, purge = opaque.Map["purge"]
	}

	_, spaceID, _, err := storagespace.SplitID(req.GetId().GetOpaqueId())
	if err != nil {
		return err
	}

	n, err := node.ReadNode(ctx, fs.lu, spaceID, spaceID, true, nil, false) // permission to read disabled space is checked later
	if err != nil {
		return err
	}

	st, err := n.SpaceRoot.XattrString(ctx, prefixes.SpaceTypeAttr)
	if err != nil {
		return errtypes.InternalError(fmt.Sprintf("space %s does not have a spacetype, possible corrupt decompsedfs", n.GetID()))
	}

	if err := canDeleteSpace(ctx, spaceID, st, purge, n, fs.p); err != nil {
		return err
	}
	if purge {
		if !n.IsDisabled(ctx) {
			return errtypes.NewErrtypeFromStatus(status.NewInvalid(ctx, "can't purge enabled space"))
		}

		// TODO invalidate ALL indexes in msgpack, not only by type
		spaceType, err := n.XattrString(ctx, prefixes.SpaceTypeAttr)
		if err != nil {
			return err
		}
		if err := fs.spaceTypeIndex.Remove(spaceType, spaceID); err != nil {
			return err
		}

		// invalidate cache
		if err := fs.lu.MetadataBackend().Purge(ctx, n); err != nil {
			return err
		}

		root := n.InternalPath()

		// walkfn will delete the blob if the node has one
		walkfn := func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			if filepath.Ext(path) != ".mpk" {
				return nil
			}

			b, err := os.ReadFile(path)
			if err != nil {
				return err
			}

			m := map[string][]byte{}
			if err := msgpack.Unmarshal(b, &m); err != nil {
				return err
			}

			bid := m["user.oc.blobid"]
			if string(bid) == "" {
				return nil
			}

			if err := fs.tp.DeleteBlob(&node.Node{
				BaseNode: node.BaseNode{SpaceID: spaceID},
				BlobID:   string(bid),
			}); err != nil {
				return err
			}

			// remove .mpk file so subsequent attempts will not try to delete the blob again
			return os.Remove(path)
		}

		// This is deletes all blobs of the space
		// NOTE: This isn't needed when no s3 is used, but we can't differentiate that here...
		if err := filepath.Walk(root, walkfn); err != nil {
			return err
		}

		// remove space metadata
		spaceRoot := fs.lu.InternalSpaceRoot(spaceID)
		if spaceRoot != "" {
			if err := os.RemoveAll(fs.lu.InternalSpaceRoot(spaceID)); err != nil {
				return err
			}
		}

		// invalidate id in cache
		if l, ok := fs.lu.(*lookup.Lookup); ok {
			if err := l.IDCache.DeleteByPath(ctx, root); err != nil {
				return err
			}
		}

		// try removing the space root node
		// Note that this will fail when there are other spaceids starting with the same two digits.
		_ = os.Remove(filepath.Dir(root))

		return nil
	}

	// mark as disabled by writing a dtime attribute
	dtime := time.Now()
	return n.SetDTime(ctx, &dtime)
}

// the value of `target` depends on the implementation:
// - for decomposedfs/decomposeds3 it is the relative link to the space root
// - for the posixfs it is the node id
func (fs *Decomposedfs) updateIndexes(ctx context.Context, grantee *provider.Grantee, spaceType, spaceID, nodeID string) error {
	target := fs.tp.BuildSpaceIDIndexEntry(spaceID)
	err := fs.linkStorageSpaceType(ctx, spaceType, spaceID, target)
	if err != nil {
		return err
	}
	if isShareGrant(ctx) {
		// FIXME we should count the references for the by-type index currently removing the second share from the same
		// space cannot determine if the by-type should be deletet, which is why we never delete them ...
		return nil
	}

	// create space grant index
	switch {
	case grantee.Type == provider.GranteeType_GRANTEE_TYPE_USER:
		return fs.linkSpaceByUser(ctx, grantee.GetUserId().GetOpaqueId(), spaceID, target)
	case grantee.Type == provider.GranteeType_GRANTEE_TYPE_GROUP:
		return fs.linkSpaceByGroup(ctx, grantee.GetGroupId().GetOpaqueId(), spaceID, target)
	default:
		return errtypes.BadRequest("invalid grantee type: " + grantee.GetType().String())
	}
}

func (fs *Decomposedfs) linkSpaceByUser(ctx context.Context, userID, spaceID, target string) error {
	return fs.userSpaceIndex.Add(userID, spaceID, target)
}

func (fs *Decomposedfs) linkSpaceByGroup(ctx context.Context, groupID, spaceID, target string) error {
	return fs.groupSpaceIndex.Add(groupID, spaceID, target)
}

func (fs *Decomposedfs) linkStorageSpaceType(ctx context.Context, spaceType, spaceID, target string) error {
	return fs.spaceTypeIndex.Add(spaceType, spaceID, target)
}

func (fs *Decomposedfs) StorageSpaceFromNode(ctx context.Context, n *node.Node, checkPermissions bool) (*provider.StorageSpace, error) {
	user := ctxpkg.ContextMustGetUser(ctx)
	if checkPermissions {
		rp, err := fs.p.AssemblePermissions(ctx, n)
		switch {
		case err != nil:
			return nil, err
		case !rp.Stat:
			return nil, errtypes.NotFound(fmt.Sprintf("space %s not found", n.GetID()))
		}

		if n.SpaceRoot.IsDisabled(ctx) {
			rp, err := fs.p.AssemblePermissions(ctx, n)
			if err != nil || !permissions.IsManager(rp) {
				return nil, errtypes.PermissionDenied(fmt.Sprintf("user %s is not allowed to list deleted spaces %s", user.Username, n.GetID()))
			}
		}
	}

	sublog := appctx.GetLogger(ctx).With().Str("spaceid", n.GetSpaceID()).Logger()

	var err error
	// TODO apply more filters
	var sname string
	if sname, err = n.SpaceRoot.XattrString(ctx, prefixes.SpaceNameAttr); err != nil {
		// FIXME: Is that a severe problem?
		sublog.Debug().Err(err).Msg("space does not have a name attribute")
	}

	/*
		if err := n.FindStorageSpaceRoot(); err != nil {
			return nil, err
		}
	*/

	// read the grants from the current node, not the root
	grants, err := n.ListGrants(ctx)
	if err != nil {
		return nil, err
	}

	grantMap := make(map[string]*provider.ResourcePermissions, len(grants))
	grantExpiration := make(map[string]*types.Timestamp)
	groupMap := make(map[string]struct{})
	for _, g := range grants {
		var id string
		switch g.Grantee.Type {
		case provider.GranteeType_GRANTEE_TYPE_GROUP:
			id = g.Grantee.GetGroupId().OpaqueId
			groupMap[id] = struct{}{}
		case provider.GranteeType_GRANTEE_TYPE_USER:
			id = g.Grantee.GetUserId().OpaqueId
		default:
			continue
		}

		if g.Expiration != nil {
			// We are doing this check here because we want to remove expired grants "on access".
			// This way we don't have to have a cron job checking the grants in regular intervals.
			// The tradeof obviously is that this code is here.
			if isGrantExpired(g) {
				if err := n.DeleteGrant(ctx, g, true); err != nil {
					sublog.Error().Err(err).Str("grantee", id).
						Msg("failed to delete expired space grant")
				}
				if n.IsSpaceRoot(ctx) {
					// invalidate space grant
					switch {
					case g.Grantee.Type == provider.GranteeType_GRANTEE_TYPE_USER:
						// remove from user index
						if err := fs.userSpaceIndex.Remove(g.Grantee.GetUserId().GetOpaqueId(), n.GetSpaceID()); err != nil {
							sublog.Error().Err(err).Str("grantee", id).
								Msg("failed to delete expired user space index")
						}
					case g.Grantee.Type == provider.GranteeType_GRANTEE_TYPE_GROUP:
						// remove from group index
						if err := fs.groupSpaceIndex.Remove(g.Grantee.GetGroupId().GetOpaqueId(), n.GetSpaceID()); err != nil {
							sublog.Error().Err(err).Str("grantee", id).
								Msg("failed to delete expired group space index")
						}
					}
				}

				continue
			}
			grantExpiration[id] = g.Expiration
		}
		grantMap[id] = g.Permissions
	}

	grantMapJSON, err := json.Marshal(grantMap)
	if err != nil {
		return nil, err
	}

	grantExpirationMapJSON, err := json.Marshal(grantExpiration)
	if err != nil {
		return nil, err
	}

	groupMapJSON, err := json.Marshal(groupMap)
	if err != nil {
		return nil, err
	}

	ssID, err := storagespace.FormatReference(
		&provider.Reference{
			ResourceId: &provider.ResourceId{
				SpaceId:  n.SpaceRoot.GetSpaceID(),
				OpaqueId: n.SpaceRoot.GetID()},
		},
	)
	if err != nil {
		return nil, err
	}
	space := &provider.StorageSpace{
		Opaque: &types.Opaque{
			Map: map[string]*types.OpaqueEntry{
				"grants": {
					Decoder: "json",
					Value:   grantMapJSON,
				},
				"grants_expirations": {
					Decoder: "json",
					Value:   grantExpirationMapJSON,
				},
				"groups": {
					Decoder: "json",
					Value:   groupMapJSON,
				},
			},
		},
		Id: &provider.StorageSpaceId{OpaqueId: ssID},
		Root: &provider.ResourceId{
			SpaceId:  n.SpaceRoot.GetSpaceID(),
			OpaqueId: n.SpaceRoot.GetID(),
		},
		Name: sname,
		// SpaceType is read from xattr below
		// Mtime is set either as node.tmtime or as fi.mtime below
	}

	space.SpaceType, err = n.SpaceRoot.XattrString(ctx, prefixes.SpaceTypeAttr)
	if err != nil {
		appctx.GetLogger(ctx).Debug().Err(err).Msg("space does not have a type attribute")
	}

	if n.SpaceRoot.IsDisabled(ctx) {
		space.Opaque = utils.AppendPlainToOpaque(space.Opaque, "trashed", "trashed")
	}

	if n.Owner() != nil && n.Owner().OpaqueId != "" {
		space.Owner = &userv1beta1.User{ // FIXME only return a UserID, not a full blown user object
			Id: n.Owner(),
		}
	}

	// we set the space mtime to the root item mtime
	// override the stat mtime with a tmtime if it is present
	var tmtime time.Time
	if tmt, err := n.GetTMTime(ctx); err == nil {
		tmtime = tmt
		un := tmt.UnixNano()
		space.Mtime = &types.Timestamp{
			Seconds: uint64(un / 1000000000),
			Nanos:   uint32(un % 1000000000),
		}
	} else if fi, err := os.Stat(n.InternalPath()); err == nil {
		// fall back to stat mtime
		tmtime = fi.ModTime()
		un := fi.ModTime().UnixNano()
		space.Mtime = &types.Timestamp{
			Seconds: uint64(un / 1000000000),
			Nanos:   uint32(un % 1000000000),
		}
	}

	etag, err := node.CalculateEtag(n.GetID(), tmtime)
	if err != nil {
		return nil, err
	}
	space.Opaque.Map["etag"] = &types.OpaqueEntry{
		Decoder: "plain",
		Value:   []byte(etag),
	}

	spaceAttributes, err := n.SpaceRoot.Xattrs(ctx)
	if err != nil {
		return nil, err
	}

	// if quota is set try parsing it as int64, otherwise don't bother
	if q, err := spaceAttributes.Int64(prefixes.QuotaAttr); err == nil && q >= 0 {
		// make sure we have a proper signed int
		// we use the same magic numbers to indicate:
		// -1 = uncalculated
		// -2 = unknown
		// -3 = unlimited
		space.Quota = &provider.Quota{
			QuotaMaxBytes: uint64(q),
			QuotaMaxFiles: math.MaxUint64, // TODO MaxUInt64? = unlimited? why even max files? 0 = unlimited?
		}

	}
	if si := spaceAttributes.String(prefixes.SpaceImageAttr); si != "" {
		space.Opaque = utils.AppendPlainToOpaque(space.Opaque, "image", storagespace.FormatResourceID(
			&provider.ResourceId{StorageId: space.Root.StorageId, SpaceId: space.GetRoot().GetSpaceId(), OpaqueId: si},
		))
	}
	if sd := spaceAttributes.String(prefixes.SpaceDescriptionAttr); sd != "" {
		space.Opaque = utils.AppendPlainToOpaque(space.Opaque, "description", sd)
	}
	if sr := spaceAttributes.String(prefixes.SpaceReadmeAttr); sr != "" {
		space.Opaque = utils.AppendPlainToOpaque(space.Opaque, "readme", storagespace.FormatResourceID(
			&provider.ResourceId{StorageId: space.Root.StorageId, SpaceId: space.GetRoot().GetSpaceId(), OpaqueId: sr},
		))
	}
	if sa := spaceAttributes.String(prefixes.SpaceAliasAttr); sa != "" {
		space.Opaque = utils.AppendPlainToOpaque(space.Opaque, "spaceAlias", sa)
	}

	// add rootinfo
	ps, _ := n.SpaceRoot.PermissionSet(ctx)
	space.RootInfo, _ = n.SpaceRoot.AsResourceInfo(ctx, ps, []string{"quota"}, nil, false)

	// we cannot put free, used and remaining into the quota, as quota, when set would always imply a quota limit
	// for now we use opaque properties with a 'quota.' prefix
	quotaStr := node.QuotaUnknown
	if quotaInOpaque := sdk.DecodeOpaqueMap(space.RootInfo.Opaque)["quota"]; quotaInOpaque != "" {
		quotaStr = quotaInOpaque
	}

	total, used, remaining, err := fs.calculateTotalUsedRemaining(quotaStr, space.GetRootInfo().GetSize())
	if err != nil {
		return nil, err
	}
	space.Opaque = utils.AppendPlainToOpaque(space.Opaque, "quota.total", strconv.FormatUint(total, 10))
	space.Opaque = utils.AppendPlainToOpaque(space.Opaque, "quota.used", strconv.FormatUint(used, 10))
	space.Opaque = utils.AppendPlainToOpaque(space.Opaque, "quota.remaining", strconv.FormatUint(remaining, 10))

	return space, nil
}

func mapHasKey(checkMap map[string][]byte, keys ...string) bool {
	for _, key := range keys {
		if _, hasKey := checkMap[key]; hasKey {
			return true
		}
	}
	return false
}

func isGrantExpired(g *provider.Grant) bool {
	if g.Expiration == nil {
		return false
	}
	return time.Now().After(time.Unix(int64(g.Expiration.Seconds), int64(g.Expiration.Nanos)))
}

// Space deletion can be tricky as there are lots of different cases:
// - spaces of type personal can only be disabled and deleted by users with the "delete-all-home-spaces" permission
// - a user with the "delete-all-spaces" permission may delete but not enable/disable any project space
// - a user with the "Drive.ReadWriteEnabled" permission may enable/disable but not delete any project space
// - a project space can always be enabled/disabled/deleted by its manager (i.e. users have the "remove" grant)
func canDeleteSpace(ctx context.Context, spaceID string, typ string, purge bool, n *node.Node, p permissions.Permissions) error {
	// delete-all-home spaces allows to disable and delete a personal space
	if typ == "personal" {
		if p.DeleteAllHomeSpaces(ctx) {
			return nil
		}
		return errtypes.PermissionDenied("user is not allowed to delete a personal space")
	}

	// space managers are allowed to disable and delete their project spaces
	if rp, err := p.AssemblePermissions(ctx, n); err == nil && permissions.IsManager(rp) {
		return nil
	}

	// delete-all-spaces permissions allows to delete (purge, NOT disable) project spaces
	if purge && p.DeleteAllSpaces(ctx) {
		return nil
	}

	// Drive.ReadWriteEnabled allows to disable a space
	if !purge && p.SpaceAbility(ctx, spaceID) {
		return nil
	}

	return errtypes.PermissionDenied(fmt.Sprintf("user is not allowed to delete space %s", n.GetID()))
}
