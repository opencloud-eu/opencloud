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
	"path/filepath"
	"strings"

	provider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	"github.com/opencloud-eu/reva/v2/internal/grpc/services/storageprovider"
	"github.com/opencloud-eu/reva/v2/pkg/appctx"
	ctxpkg "github.com/opencloud-eu/reva/v2/pkg/ctx"
	"github.com/opencloud-eu/reva/v2/pkg/errtypes"
	"github.com/opencloud-eu/reva/v2/pkg/storage/pkg/decomposedfs/metadata"
	"github.com/opencloud-eu/reva/v2/pkg/storage/pkg/decomposedfs/metadata/prefixes"
	"github.com/opencloud-eu/reva/v2/pkg/storage/pkg/decomposedfs/node"
	"github.com/opencloud-eu/reva/v2/pkg/storage/utils/ace"
	"github.com/opencloud-eu/reva/v2/pkg/storagespace"
	"github.com/opencloud-eu/reva/v2/pkg/utils"
)

// DenyGrant denies access to a resource.
func (fs *Decomposedfs) DenyGrant(ctx context.Context, ref *provider.Reference, grantee *provider.Grantee) error {
	_, span := tracer.Start(ctx, "DenyGrant")
	defer span.End()
	log := appctx.GetLogger(ctx)

	log.Debug().Interface("ref", ref).Interface("grantee", grantee).Msg("DenyGrant()")

	grantNode, err := fs.lu.NodeFromResource(ctx, ref)
	if err != nil {
		return err
	}
	if !grantNode.Exists {
		return errtypes.NotFound(filepath.Join(grantNode.ParentID, grantNode.Name))
	}

	// set all permissions to false
	grant := &provider.Grant{
		Grantee:     grantee,
		Permissions: &provider.ResourcePermissions{},
	}

	// add acting user
	u := ctxpkg.ContextMustGetUser(ctx)
	grant.Creator = u.GetId()

	rp, err := fs.p.AssemblePermissions(ctx, grantNode)

	switch {
	case err != nil:
		return err
	case !rp.DenyGrant:
		return errtypes.PermissionDenied(filepath.Join(grantNode.ParentID, grantNode.Name))
	}

	return fs.storeGrant(ctx, grantNode, grant)
}

// AddGrant adds a grant to a resource
func (fs *Decomposedfs) AddGrant(ctx context.Context, ref *provider.Reference, g *provider.Grant) (err error) {
	_, span := tracer.Start(ctx, "AddGrant")
	defer span.End()
	log := appctx.GetLogger(ctx)
	log.Debug().Interface("ref", ref).Interface("grant", g).Msg("AddGrant()")
	grantNode, unlockFunc, grant, err := fs.loadGrant(ctx, ref, g)
	if err != nil {
		return err
	}
	defer func() {
		_ = unlockFunc()
	}()

	if grant != nil {
		return errtypes.AlreadyExists(filepath.Join(grantNode.ParentID, grantNode.Name))
	}

	owner := grantNode.Owner()
	grants, err := grantNode.ListGrants(ctx)
	if err != nil {
		return err
	}

	// If the owner is empty and there are no grantees then we are dealing with a just created project space.
	// In this case we don't need to check for permissions and just add the grant since this will be the project
	// manager.
	// When the owner is empty but grants are set then we do want to check the grants.
	// However, if we are trying to edit an existing grant we do not have to check for permission if the user owns the grant
	// TODO: find a better to check this
	if !(len(grants) == 0 && (owner == nil || owner.OpaqueId == "" || (owner.OpaqueId == grantNode.SpaceID && owner.Type == 8))) {
		rp, err := fs.p.AssemblePermissions(ctx, grantNode)
		switch {
		case err != nil:
			return err
		case !rp.AddGrant:
			f, _ := storagespace.FormatReference(ref)
			if rp.Stat {
				return errtypes.PermissionDenied(f)
			}
			return errtypes.NotFound(f)
		}
	}

	if fs.o.MultiTenantEnabled {
		spaceTenant, err := grantNode.SpaceRoot.XattrString(ctx, prefixes.SpaceTenantIDAttr)
		if err != nil {
			log.Error().Err(err).Msg("failed to read tenant id of space")
			return errtypes.InternalError("error validating tenantID")
		}
		if g.Grantee.Type == provider.GranteeType_GRANTEE_TYPE_USER {
			if g.Grantee.GetUserId().GetTenantId() != spaceTenant {
				log.Error().Str("spaceTenant", spaceTenant).Str("granteeTenant", g.Grantee.GetUserId().GetTenantId()).Msg("cannot add grant for user from different tenant")
				return errtypes.PermissionDenied("cannot add grant for user from different tenant")
			}
		}
	}

	return fs.storeGrant(ctx, grantNode, g)
}

// ListGrants lists the grants on the specified resource
func (fs *Decomposedfs) ListGrants(ctx context.Context, ref *provider.Reference) (grants []*provider.Grant, err error) {
	_, span := tracer.Start(ctx, "ListGrants")
	defer span.End()
	var grantNode *node.Node
	if grantNode, err = fs.lu.NodeFromResource(ctx, ref); err != nil {
		return
	}
	if !grantNode.Exists {
		err = errtypes.NotFound(filepath.Join(grantNode.ParentID, grantNode.Name))
		return
	}
	rp, err := fs.p.AssemblePermissions(ctx, grantNode)
	switch {
	case err != nil:
		return nil, err
	case !rp.ListGrants && !rp.Stat:
		f, _ := storagespace.FormatReference(ref)
		return nil, errtypes.NotFound(f)
	}
	log := appctx.GetLogger(ctx)
	var attrs node.Attributes
	if attrs, err = grantNode.Xattrs(ctx); err != nil {
		log.Error().Err(err).Msg("error listing attributes")
		return nil, err
	}

	aces := []*ace.ACE{}
	for k, v := range attrs {
		if strings.HasPrefix(k, prefixes.GrantPrefix) {
			var err error
			var e *ace.ACE
			principal := k[len(prefixes.GrantPrefix):]
			if e, err = ace.Unmarshal(principal, v); err != nil {
				log.Error().Err(err).Str("principal", principal).Str("attr", k).Msg("could not unmarshal ace")
				continue
			}
			aces = append(aces, e)
		}
	}

	uid := ctxpkg.ContextMustGetUser(ctx).GetId()
	grants = make([]*provider.Grant, 0, len(aces))
	for i := range aces {
		g := aces[i].Grant()

		// you may list your own grants even without listgrants permission
		if !rp.ListGrants && !utils.UserIDEqual(g.Creator, uid) && !utils.UserIDEqual(g.Grantee.GetUserId(), uid) {
			continue
		}

		grants = append(grants, g)
	}

	return grants, nil
}

// RemoveGrant removes a grant from resource
func (fs *Decomposedfs) RemoveGrant(ctx context.Context, ref *provider.Reference, g *provider.Grant) (err error) {
	_, span := tracer.Start(ctx, "RemoveGrant")
	defer span.End()
	grantNode, unlockFunc, grant, err := fs.loadGrant(ctx, ref, g)
	if err != nil {
		return err
	}
	defer func() {
		_ = unlockFunc()
	}()

	if grant == nil {
		return errtypes.NotFound("grant not found")
	}

	// you are allowed to remove grants if you created them yourself or have the proper permission
	if !utils.UserIDEqual(grant.Creator, ctxpkg.ContextMustGetUser(ctx).GetId()) {
		rp, err := fs.p.AssemblePermissions(ctx, grantNode)
		switch {
		case err != nil:
			return err
		case !rp.RemoveGrant:
			f, _ := storagespace.FormatReference(ref)
			if rp.Stat {
				return errtypes.PermissionDenied(f)
			}
			return errtypes.NotFound(f)
		}
	}

	if err := grantNode.DeleteGrant(ctx, g, false); err != nil {
		return err
	}

	if isShareGrant(ctx) {
		// do not invalidate by user or group indexes
		// FIXME we should invalidate the by-type index, but that requires reference counting
	} else {
		// invalidate space grant
		switch {
		case g.Grantee.Type == provider.GranteeType_GRANTEE_TYPE_USER:
			// remove from user index
			if err := fs.userSpaceIndex.Remove(g.Grantee.GetUserId().GetOpaqueId(), grantNode.SpaceID); err != nil {
				return err
			}
		case g.Grantee.Type == provider.GranteeType_GRANTEE_TYPE_GROUP:
			// remove from group index
			if err := fs.groupSpaceIndex.Remove(g.Grantee.GetGroupId().GetOpaqueId(), grantNode.SpaceID); err != nil {
				return err
			}
		}
	}

	return fs.tp.Propagate(ctx, grantNode, 0)
}

func isShareGrant(ctx context.Context) bool {
	_, ok := storageprovider.SpaceTypeFromContext(ctx)
	return !ok
}

// UpdateGrant updates a grant on a resource
// TODO remove AddGrant or UpdateGrant grant from CS3 api, redundant? tracked in https://github.com/cs3org/cs3apis/issues/92
func (fs *Decomposedfs) UpdateGrant(ctx context.Context, ref *provider.Reference, g *provider.Grant) error {
	_, span := tracer.Start(ctx, "UpdateGrant")
	defer span.End()
	log := appctx.GetLogger(ctx)
	log.Debug().Interface("ref", ref).Interface("grant", g).Msg("UpdateGrant()")

	grantNode, unlockFunc, grant, err := fs.loadGrant(ctx, ref, g)
	if err != nil {
		return err
	}
	defer func() {
		_ = unlockFunc()
	}()

	if grant == nil {
		// grant not found
		// TODO: fallback to AddGrant?
		return errtypes.NotFound(g.Grantee.GetUserId().GetOpaqueId())
	}

	// You may update a grant when you have the UpdateGrant permission or created the grant (regardless what your permissions are now)
	if !utils.UserIDEqual(grant.Creator, ctxpkg.ContextMustGetUser(ctx).GetId()) {
		rp, err := fs.p.AssemblePermissions(ctx, grantNode)
		switch {
		case err != nil:
			return err
		case !rp.UpdateGrant:
			f, _ := storagespace.FormatReference(ref)
			if rp.Stat {
				return errtypes.PermissionDenied(f)
			}
			return errtypes.NotFound(f)
		}
	}

	return fs.storeGrant(ctx, grantNode, g)
}

// checks if the given grant exists and returns it. Nil grant means it doesn't exist
func (fs *Decomposedfs) loadGrant(ctx context.Context, ref *provider.Reference, g *provider.Grant) (*node.Node, metadata.UnlockFunc, *provider.Grant, error) {
	_, span := tracer.Start(ctx, "loadGrant")
	defer span.End()
	n, err := fs.lu.NodeFromResource(ctx, ref)
	if err != nil {
		return nil, nil, nil, err
	}
	if !n.Exists {
		return nil, nil, nil, errtypes.NotFound(filepath.Join(n.ParentID, n.Name))
	}

	// lock the metadata file
	unlockFunc, err := fs.lu.MetadataBackend().Lock(n)
	if err != nil {
		return nil, nil, nil, err
	}

	grants, err := n.ListGrants(ctx)
	if err != nil {
		return nil, nil, nil, err
	}

	for _, grant := range grants {
		switch grant.Grantee.GetType() {
		case provider.GranteeType_GRANTEE_TYPE_USER:
			if g.Grantee.GetUserId().GetOpaqueId() == grant.Grantee.GetUserId().GetOpaqueId() {
				return n, unlockFunc, grant, nil
			}
		case provider.GranteeType_GRANTEE_TYPE_GROUP:
			if g.Grantee.GetGroupId().GetOpaqueId() == grant.Grantee.GetGroupId().GetOpaqueId() {
				return n, unlockFunc, grant, nil
			}
		}
	}

	return n, unlockFunc, nil, nil
}

func (fs *Decomposedfs) storeGrant(ctx context.Context, n *node.Node, g *provider.Grant) error {
	_, span := tracer.Start(ctx, "storeGrant")
	defer span.End()
	// if is a grant to a space root, the receiver needs the space type to update the indexes
	spaceType, ok := storageprovider.SpaceTypeFromContext(ctx)
	if !ok {
		// this is not a grant on a space root we are just adding a share
		spaceType = spaceTypeShare
	}

	// set the grant
	e := ace.FromGrant(g)
	principal, value := e.Marshal()
	attribs := node.Attributes{
		prefixes.GrantPrefix + principal: value,
	}
	if err := n.SetXattrsWithContext(ctx, attribs, false); err != nil {
		appctx.GetLogger(ctx).Error().Err(err).
			Str("principal", principal).Msg("Could not set grant for principal")
		return err
	}

	// update the indexes only after successfully setting the grant
	err := fs.updateIndexes(ctx, g.GetGrantee(), spaceType, n.SpaceID, n.ID)
	if err != nil {
		return err
	}

	return fs.tp.Propagate(ctx, n, 0)
}
