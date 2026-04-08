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

package gateway

import (
	"context"
	"slices"

	gateway "github.com/cs3org/go-cs3apis/cs3/gateway/v1beta1"
	rpc "github.com/cs3org/go-cs3apis/cs3/rpc/v1beta1"
	collaboration "github.com/cs3org/go-cs3apis/cs3/sharing/collaboration/v1beta1"
	provider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	typesv1beta1 "github.com/cs3org/go-cs3apis/cs3/types/v1beta1"
	"github.com/opencloud-eu/reva/v2/pkg/appctx"
	ctxpkg "github.com/opencloud-eu/reva/v2/pkg/ctx"
	"github.com/opencloud-eu/reva/v2/pkg/errtypes"
	"github.com/opencloud-eu/reva/v2/pkg/rgrpc/status"
	"github.com/opencloud-eu/reva/v2/pkg/rgrpc/todo/pool"
	"github.com/opencloud-eu/reva/v2/pkg/storage/utils/grants"
	"github.com/opencloud-eu/reva/v2/pkg/utils"
	"github.com/pkg/errors"
)

// TODO(labkode): add multi-phase commit logic when commit share or commit ref is enabled.
func (s *svc) CreateShare(ctx context.Context, req *collaboration.CreateShareRequest) (*collaboration.CreateShareResponse, error) {
	// Don't use the share manager when sharing a space root
	if !s.c.UseCommonSpaceRootShareLogic && refIsSpaceRoot(req.ResourceInfo.Id) {
		return s.addSpaceShare(ctx, req)
	}
	return s.addShare(ctx, req)
}

func (s *svc) RemoveShare(ctx context.Context, req *collaboration.RemoveShareRequest) (*collaboration.RemoveShareResponse, error) {
	key := req.GetRef().GetKey()
	if !s.c.UseCommonSpaceRootShareLogic && shareIsSpaceRoot(key) {
		return s.removeSpaceShare(ctx, key.GetResourceId(), key.GetGrantee())
	}
	return s.removeShare(ctx, req)
}

func (s *svc) UpdateShare(ctx context.Context, req *collaboration.UpdateShareRequest) (*collaboration.UpdateShareResponse, error) {
	if !s.c.UseCommonSpaceRootShareLogic && refIsSpaceRoot(req.GetShare().GetResourceId()) {
		return s.updateSpaceShare(ctx, req)
	}
	return s.updateShare(ctx, req)
}

// TODO(labkode): we need to validate share state vs storage grant and storage ref
// If there are any inconsistencies, the share needs to be flag as invalid and a background process
// or active fix needs to be performed.
func (s *svc) GetShare(ctx context.Context, req *collaboration.GetShareRequest) (*collaboration.GetShareResponse, error) {
	return s.getShare(ctx, req)
}

func (s *svc) getShare(ctx context.Context, req *collaboration.GetShareRequest) (*collaboration.GetShareResponse, error) {
	c, err := pool.GetUserShareProviderClient(s.c.UserShareProviderEndpoint)
	if err != nil {
		appctx.GetLogger(ctx).
			Err(err).
			Msg("getShare: failed to get user share provider")
		return &collaboration.GetShareResponse{
			Status: status.NewInternal(ctx, "error getting user share provider client"),
		}, nil
	}

	res, err := c.GetShare(ctx, req)
	if err != nil {
		return nil, errors.Wrap(err, "gateway: error calling GetShare")
	}

	return res, nil
}

// TODO(labkode): read GetShare comment.
func (s *svc) ListShares(ctx context.Context, req *collaboration.ListSharesRequest) (*collaboration.ListSharesResponse, error) {
	c, err := pool.GetUserShareProviderClient(s.c.UserShareProviderEndpoint)
	if err != nil {
		appctx.GetLogger(ctx).
			Err(err).
			Msg("ListShares: failed to get user share provider")
		return &collaboration.ListSharesResponse{
			Status: status.NewInternal(ctx, "error getting user share provider client"),
		}, nil
	}

	res, err := c.ListShares(ctx, req)
	if err != nil {
		return nil, errors.Wrap(err, "gateway: error calling ListShares")
	}

	return res, nil
}

func (s *svc) ListExistingShares(_ context.Context, _ *collaboration.ListSharesRequest) (*gateway.ListExistingSharesResponse, error) {
	return nil, errtypes.NotSupported("method ListExistingShares not implemented")
}

func (s *svc) updateShare(ctx context.Context, req *collaboration.UpdateShareRequest) (*collaboration.UpdateShareResponse, error) {
	// TODO: update wopi server
	// FIXME This is a workaround that should prevent removing or changing the share permissions when the file is locked.
	// https://github.com/owncloud/ocis/issues/8474
	if status, err := s.checkLock(ctx, req.GetShare().GetId()); err != nil {
		return &collaboration.UpdateShareResponse{
			Status: status,
		}, nil
	}

	c, err := pool.GetUserShareProviderClient(s.c.UserShareProviderEndpoint)
	if err != nil {
		appctx.GetLogger(ctx).
			Err(err).
			Msg("UpdateShare: failed to get user share provider")
		return &collaboration.UpdateShareResponse{
			Status: status.NewInternal(ctx, "error getting share provider client"),
		}, nil
	}
	res, err := c.UpdateShare(ctx, req)
	if err != nil {
		return nil, errors.Wrap(err, "gateway: error calling UpdateShare")
	}
	if res.GetStatus().GetCode() != rpc.Code_CODE_OK {
		return res, nil
	}

	if s.c.CommitShareToStorageGrant {
		creator, ok := ctxpkg.ContextGetUser(ctx)
		if !ok {
			return nil, errors.New("user not found in context")
		}

		grant := &provider.Grant{
			Grantee:     res.GetShare().GetGrantee(),
			Permissions: res.GetShare().GetPermissions().GetPermissions(),
			Expiration:  res.GetShare().GetExpiration(),
			Creator:     creator.GetId(),
		}
		var opaque *typesv1beta1.Opaque
		if refIsSpaceRoot(res.GetShare().GetResourceId()) {
			opaque = utils.SpaceGrantOpaque()
			utils.AppendPlainToOpaque(opaque, "spacetype", utils.ReadPlainFromOpaque(req.GetOpaque(), "spacetype"))
		}
		updateGrantStatus, err := s.updateGrant(ctx, res.GetShare().GetResourceId(), grant, opaque)

		if err != nil {
			return nil, errors.Wrap(err, "gateway: error calling updateGrant")
		}

		if updateGrantStatus.Code != rpc.Code_CODE_OK {
			return &collaboration.UpdateShareResponse{
				Status: updateGrantStatus,
				Share:  res.GetShare(),
			}, nil
		}
	}

	return res, nil
}

func (s *svc) updateSpaceShare(ctx context.Context, req *collaboration.UpdateShareRequest) (*collaboration.UpdateShareResponse, error) {
	if req.GetShare().GetGrantee() == nil {
		return &collaboration.UpdateShareResponse{Status: status.NewInvalid(ctx, "updating requires a received grantee object")}, nil
	}
	// If the share is a denial we call  denyGrant instead.
	var st *rpc.Status
	var err error
	// TODO: change CS3 APIs
	opaque := utils.SpaceGrantOpaque()
	utils.AppendPlainToOpaque(opaque, "spacetype", utils.ReadPlainFromOpaque(req.Opaque, "spacetype"))

	if grants.PermissionsEqual(req.Share.GetPermissions().GetPermissions(), &provider.ResourcePermissions{}) {
		st, err = s.denyGrant(ctx, req.GetShare().GetResourceId(), req.GetShare().GetGrantee(), opaque)
		if err != nil {
			return nil, errors.Wrap(err, "gateway: error denying grant in storage")
		}
	} else {
		listGrantRes, err := s.listGrants(ctx, req.GetShare().GetResourceId())
		if err != nil {
			return nil, errors.Wrap(err, "gateway: error getting grant to remove from storage")
		}
		existsGrant := s.getGranteeGrant(listGrantRes.GetGrants(), req.GetShare().GetGrantee())

		if !slices.Contains(req.GetUpdateMask().GetPaths(), "permissions") {
			req.Share.Permissions = &collaboration.SharePermissions{Permissions: existsGrant.GetPermissions()}
		}

		if !slices.Contains(req.GetUpdateMask().GetPaths(), "expiration") {
			req.Share.Expiration = existsGrant.GetExpiration()
		}

		u, ok := ctxpkg.ContextGetUser(ctx)
		if !ok {
			return nil, errors.New("user not found in context")
		}

		grant := &provider.Grant{
			Grantee:     req.GetShare().GetGrantee(),
			Permissions: req.GetShare().GetPermissions().GetPermissions(),
			Expiration:  req.GetShare().GetExpiration(),
			Creator:     u.GetId(),
		}

		if grant.GetPermissions() == nil {
			return &collaboration.UpdateShareResponse{Status: status.NewInvalid(ctx, "updating requires a received permission object")}, nil
		}

		if !grant.GetPermissions().GetRemoveGrant() {
			// this request might remove Manager Permissions so we need to
			// check if there is at least one manager remaining of the
			// resource.
			if !isSpaceManagerRemaining(listGrantRes.GetGrants(), grant.GetGrantee()) {
				return &collaboration.UpdateShareResponse{
					Status: status.NewPermissionDenied(ctx, errtypes.PermissionDenied(""), "can't remove the last manager"),
				}, nil
			}

		}
		st, err = s.updateGrant(ctx, req.GetShare().GetResourceId(), grant, opaque)
		if err != nil {
			return nil, errors.Wrap(err, "gateway: error adding grant to storage")
		}
	}

	res := &collaboration.UpdateShareResponse{
		Status: st,
		Share:  req.Share,
	}

	if st.Code != rpc.Code_CODE_OK {
		return res, nil
	}

	s.providerCache.RemoveListStorageProviders(req.GetShare().GetResourceId())
	return res, nil
}

// TODO(labkode): listing received shares just goes to the user share manager and gets the list of
// received shares. The display name of the shares should be the a friendly name, like the basename
// of the original file.
func (s *svc) ListReceivedShares(ctx context.Context, req *collaboration.ListReceivedSharesRequest) (*collaboration.ListReceivedSharesResponse, error) {
	logger := appctx.GetLogger(ctx)
	c, err := pool.GetUserShareProviderClient(s.c.UserShareProviderEndpoint)
	if err != nil {
		logger.Error().
			Err(err).
			Msg("ListReceivedShares: failed to get user share provider")
		return &collaboration.ListReceivedSharesResponse{
			Status: status.NewInternal(ctx, "error getting share provider client"),
		}, nil
	}

	res, err := c.ListReceivedShares(ctx, req)
	if err != nil {
		return nil, errors.Wrap(err, "gateway: error calling ListReceivedShares")
	}
	return res, nil
}

func (s *svc) GetReceivedShare(ctx context.Context, req *collaboration.GetReceivedShareRequest) (*collaboration.GetReceivedShareResponse, error) {
	c, err := pool.GetUserShareProviderClient(s.c.UserShareProviderEndpoint)
	if err != nil {
		appctx.GetLogger(ctx).
			Err(err).
			Msg("GetReceivedShare: failed to get user share provider")
		return &collaboration.GetReceivedShareResponse{
			Status: status.NewInternal(ctx, "error getting received share"),
		}, nil
	}

	res, err := c.GetReceivedShare(ctx, req)
	if err != nil {
		return nil, errors.Wrap(err, "gateway: error calling GetReceivedShare")
	}

	return res, nil
}

// When updating a received share:
// if the update contains update for displayName:
//  1. if received share is mounted: we also do a rename in the storage
//  2. if received share is not mounted: we only rename in user share provider.
func (s *svc) UpdateReceivedShare(ctx context.Context, req *collaboration.UpdateReceivedShareRequest) (*collaboration.UpdateReceivedShareResponse, error) {
	ctx, span := appctx.GetTracerProvider(ctx).Tracer("gateway").Start(ctx, "Gateway.UpdateReceivedShare")
	defer span.End()

	// sanity checks
	switch {
	case req.GetShare() == nil:
		return &collaboration.UpdateReceivedShareResponse{
			Status: status.NewInvalid(ctx, "updating requires a received share object"),
		}, nil
	case req.GetShare().GetShare() == nil:
		return &collaboration.UpdateReceivedShareResponse{
			Status: status.NewInvalid(ctx, "share missing"),
		}, nil
	case req.GetShare().GetShare().GetId() == nil:
		return &collaboration.UpdateReceivedShareResponse{
			Status: status.NewInvalid(ctx, "share id missing"),
		}, nil
	case req.GetShare().GetShare().GetId().GetOpaqueId() == "":
		return &collaboration.UpdateReceivedShareResponse{
			Status: status.NewInvalid(ctx, "share id empty"),
		}, nil
	}

	c, err := pool.GetUserShareProviderClient(s.c.UserShareProviderEndpoint)
	if err != nil {
		appctx.GetLogger(ctx).
			Err(err).
			Msg("UpdateReceivedShare: failed to get user share provider")
		return &collaboration.UpdateReceivedShareResponse{
			Status: status.NewInternal(ctx, "error getting share provider client"),
		}, nil
	}

	return c.UpdateReceivedShare(ctx, req)
	/*
		    TODO: Leftover from master merge. Do we need this?
			if err != nil {
				appctx.GetLogger(ctx).
					Err(err).
					Msg("UpdateReceivedShare: failed to get user share provider")
				return &collaboration.UpdateReceivedShareResponse{
					Status: status.NewInternal(ctx, "error getting share provider client"),
				}, nil
			}
			// check if we have a resource id in the update response that we can use to update references
			if res.GetShare().GetShare().GetResourceId() == nil {
				log.Err(err).Msg("gateway: UpdateReceivedShare must return a ResourceId")
				return &collaboration.UpdateReceivedShareResponse{
					Status: &rpc.Status{
						Code: rpc.Code_CODE_INTERNAL,
					},
				}, nil
			}

			// properties are updated in the order they appear in the field mask
			// when an error occurs the request ends and no further fields are updated
			for i := range req.UpdateMask.Paths {
				switch req.UpdateMask.Paths[i] {
				case "state":
					switch req.GetShare().GetState() {
					case collaboration.ShareState_SHARE_STATE_ACCEPTED:
						rpcStatus := s.createReference(ctx, res.GetShare().GetShare().GetResourceId())
						if rpcStatus.Code != rpc.Code_CODE_OK {
							return &collaboration.UpdateReceivedShareResponse{Status: rpcStatus}, nil
						}
					case collaboration.ShareState_SHARE_STATE_REJECTED:
						rpcStatus := s.removeReference(ctx, res.GetShare().GetShare().ResourceId)
						if rpcStatus.Code != rpc.Code_CODE_OK && rpcStatus.Code != rpc.Code_CODE_NOT_FOUND {
							return &collaboration.UpdateReceivedShareResponse{Status: rpcStatus}, nil
						}
					}
				case "mount_point":
					// TODO(labkode): implementing updating mount point
					err = errtypes.NotSupported("gateway: update of mount point is not yet implemented")
					return &collaboration.UpdateReceivedShareResponse{
						Status: status.NewUnimplemented(ctx, err, "error updating received share"),
					}, nil
				default:
					return nil, errtypes.NotSupported("updating " + req.UpdateMask.Paths[i] + " is not supported")
				}
			}
			return res, nil
	*/
}

func (s *svc) ListExistingReceivedShares(_ context.Context, _ *collaboration.ListReceivedSharesRequest) (*gateway.ListExistingReceivedSharesResponse, error) {
	return nil, errtypes.NotSupported("Unimplemented")
}

func (s *svc) denyGrant(ctx context.Context, id *provider.ResourceId, g *provider.Grantee, opaque *typesv1beta1.Opaque) (*rpc.Status, error) {
	ref := &provider.Reference{
		ResourceId: id,
	}

	grantReq := &provider.DenyGrantRequest{
		Ref:     ref,
		Grantee: g,
		Opaque:  opaque,
		// TODO add creator
	}

	c, _, err := s.find(ctx, ref)
	if err != nil {
		appctx.GetLogger(ctx).
			Err(err).
			Interface("reference", ref).
			Msg("denyGrant: failed to get storage provider")
		if _, ok := err.(errtypes.IsNotFound); ok {
			return status.NewNotFound(ctx, "storage provider not found"), nil
		}
		return status.NewInternal(ctx, "error finding storage provider"), nil
	}

	grantRes, err := c.DenyGrant(ctx, grantReq)
	if err != nil {
		return nil, errors.Wrap(err, "gateway: error calling DenyGrant")
	}
	return grantRes.Status, nil
}

func (s *svc) addGrant(ctx context.Context, id *provider.ResourceId, g *provider.Grantee, p *provider.ResourcePermissions, expiration *typesv1beta1.Timestamp, opaque *typesv1beta1.Opaque) (*rpc.Status, error) {
	ref := &provider.Reference{
		ResourceId: id,
	}

	creator, ok := ctxpkg.ContextGetUser(ctx)
	if !ok {
		return nil, errors.New("user not found in context")
	}

	grantReq := &provider.AddGrantRequest{
		Ref: ref,
		Grant: &provider.Grant{
			Grantee:     g,
			Permissions: p,
			Creator:     creator.GetId(),
			Expiration:  expiration,
		},
		Opaque: opaque,
	}

	c, _, err := s.find(ctx, ref)
	if err != nil {
		appctx.GetLogger(ctx).
			Err(err).
			Interface("reference", ref).
			Msg("addGrant: failed to get storage provider")
		if _, ok := err.(errtypes.IsNotFound); ok {
			return status.NewNotFound(ctx, "storage provider not found"), nil
		}
		return status.NewInternal(ctx, "error finding storage provider"), nil
	}

	grantRes, err := c.AddGrant(ctx, grantReq)
	if err != nil {
		return nil, errors.Wrap(err, "gateway: error calling AddGrant")
	}
	return grantRes.Status, nil
}

func (s *svc) updateGrant(ctx context.Context, id *provider.ResourceId, grant *provider.Grant, opaque *typesv1beta1.Opaque) (*rpc.Status, error) {
	ref := &provider.Reference{
		ResourceId: id,
	}

	grantReq := &provider.UpdateGrantRequest{
		Opaque: opaque,
		Ref:    ref,
		Grant:  grant,
	}

	c, _, err := s.find(ctx, ref)
	if err != nil {
		appctx.GetLogger(ctx).
			Err(err).
			Interface("reference", ref).
			Msg("updateGrant: failed to get storage provider")
		if _, ok := err.(errtypes.IsNotFound); ok {
			return status.NewNotFound(ctx, "storage provider not found"), nil
		}
		return status.NewInternal(ctx, "error finding storage provider"), nil
	}

	grantRes, err := c.UpdateGrant(ctx, grantReq)
	if err != nil {
		return nil, errors.Wrap(err, "gateway: error calling UpdateGrant")
	}
	if grantRes.Status.Code != rpc.Code_CODE_OK {
		return status.NewInternal(ctx,
			"error committing share to storage grant"), nil
	}

	return status.NewOK(ctx), nil
}

func (s *svc) removeGrant(ctx context.Context, id *provider.ResourceId, g *provider.Grantee, p *provider.ResourcePermissions, opaque *typesv1beta1.Opaque) (*rpc.Status, error) {
	ref := &provider.Reference{
		ResourceId: id,
	}

	grantReq := &provider.RemoveGrantRequest{
		Ref: ref,
		Grant: &provider.Grant{
			Grantee:     g,
			Permissions: p,
		},
		Opaque: opaque,
	}

	c, _, err := s.find(ctx, ref)
	if err != nil {
		appctx.GetLogger(ctx).
			Err(err).
			Interface("reference", ref).
			Msg("removeGrant: failed to get storage provider")
		if _, ok := err.(errtypes.IsNotFound); ok {
			return status.NewNotFound(ctx, "storage provider not found"), nil
		}
		return status.NewInternal(ctx, "error finding storage provider"), nil
	}

	grantRes, err := c.RemoveGrant(ctx, grantReq)
	if err != nil {
		return nil, errors.Wrap(err, "gateway: error calling RemoveGrant")
	}
	if grantRes.Status.Code != rpc.Code_CODE_OK {
		return grantRes.GetStatus(), nil
	}

	return status.NewOK(ctx), nil
}

func (s *svc) listGrants(ctx context.Context, id *provider.ResourceId) (*provider.ListGrantsResponse, error) {
	ref := &provider.Reference{
		ResourceId: id,
	}

	grantReq := &provider.ListGrantsRequest{
		Ref: ref,
	}

	c, _, err := s.find(ctx, ref)
	if err != nil {
		appctx.GetLogger(ctx).
			Err(err).
			Interface("reference", ref).
			Msg("listGrants: failed to get storage provider")
		if _, ok := err.(errtypes.IsNotFound); ok {
			return &provider.ListGrantsResponse{
				Status: status.NewNotFound(ctx, "storage provider not found"),
			}, nil
		}
		return &provider.ListGrantsResponse{
			Status: status.NewInternal(ctx, "error finding storage provider"),
		}, nil
	}

	grantRes, err := c.ListGrants(ctx, grantReq)
	if err != nil {
		return nil, errors.Wrap(err, "gateway: error calling ListGrants")
	}
	if grantRes.Status.Code != rpc.Code_CODE_OK {
		return &provider.ListGrantsResponse{Status: status.NewInternal(ctx,
				"error listing storage grants"),
			},
			nil
	}
	return grantRes, nil
}

func (s *svc) getGranteeGrant(grants []*provider.Grant, grantee *provider.Grantee) *provider.Grant {
	for _, g := range grants {
		if isEqualGrantee(g.Grantee, grantee) {
			return g
		}
	}
	return nil
}

func (s *svc) addShare(ctx context.Context, req *collaboration.CreateShareRequest) (*collaboration.CreateShareResponse, error) {
	c, err := pool.GetUserShareProviderClient(s.c.UserShareProviderEndpoint)
	if err != nil {
		appctx.GetLogger(ctx).
			Err(err).
			Msg("CreateShare: failed to get user share provider")
		return &collaboration.CreateShareResponse{
			Status: status.NewInternal(ctx, "error getting user share provider client"),
		}, nil
	}
	// TODO the user share manager needs to be able to decide if the current user is allowed to create that share (and not eg. incerase permissions)
	// jfd: AFAICT this can only be determined by a storage driver - either the storage provider is queried first or the share manager needs to access the storage using a storage driver
	res, err := c.CreateShare(ctx, req)
	if err != nil {
		return nil, errors.Wrap(err, "gateway: error calling CreateShare")
	}
	if res.Status.Code != rpc.Code_CODE_OK {
		return res, nil
	}

	rollBackFn := func(status *rpc.Status) {
		rmvReq := &collaboration.RemoveShareRequest{
			Ref: &collaboration.ShareReference{
				Spec: &collaboration.ShareReference_Key{
					Key: &collaboration.ShareKey{
						ResourceId: req.ResourceInfo.Id,
						Grantee:    req.Grant.Grantee,
					},
				},
			},
		}
		appctx.GetLogger(ctx).Debug().Interface("status", status).Interface("req", req).Msg("rollback the CreateShare attempt")
		if resp, err := s.removeShare(ctx, rmvReq); err != nil {
			appctx.GetLogger(ctx).Debug().Interface("status", resp.GetStatus()).Interface("req", req).Msg(err.Error())
		}
	}

	if s.c.CommitShareToStorageGrant {
		// If the share is a denial we call denyGrant instead.
		var status *rpc.Status
		var opaque *typesv1beta1.Opaque
		if refIsSpaceRoot(req.ResourceInfo.Id) {
			opaque = utils.SpaceGrantOpaque()
			utils.AppendPlainToOpaque(opaque, "spacetype", req.ResourceInfo.GetSpace().GetSpaceType())
		}
		if grants.PermissionsEqual(req.Grant.Permissions.Permissions, &provider.ResourcePermissions{}) {
			status, err = s.denyGrant(ctx, req.ResourceInfo.Id, req.Grant.Grantee, opaque)
			if err != nil {
				return nil, errors.Wrap(err, "gateway: error denying grant in storage")
			}
		} else {
			status, err = s.addGrant(ctx, req.ResourceInfo.Id, req.Grant.Grantee, req.Grant.Permissions.Permissions, req.Grant.Expiration, opaque)
			if err != nil {
				appctx.GetLogger(ctx).Debug().Interface("status", status).Interface("req", req).Msg(err.Error())
				rollBackFn(status)
				return nil, errors.Wrap(err, "gateway: error adding grant to storage")
			}
		}

		switch status.Code {
		case rpc.Code_CODE_OK, rpc.Code_CODE_ALREADY_EXISTS:
			// ok
		case rpc.Code_CODE_UNIMPLEMENTED:
			appctx.GetLogger(ctx).Debug().Interface("status", status).Interface("req", req).Msg("storing grants not supported, ignoring")
			rollBackFn(status)
		default:
			appctx.GetLogger(ctx).Debug().Interface("status", status).Interface("req", req).Msg("storing grants is not successful")
			rollBackFn(status)
			return &collaboration.CreateShareResponse{
				Status: status,
			}, err
		}
	}
	return res, nil
}

func (s *svc) addSpaceShare(ctx context.Context, req *collaboration.CreateShareRequest) (*collaboration.CreateShareResponse, error) {
	if refIsSpaceRoot(req.GetResourceInfo().GetId()) &&
		(req.GetResourceInfo().GetSpace().GetSpaceType() == _spaceTypePersonal || req.GetResourceInfo().GetSpace().GetSpaceType() == _spaceTypeVirtual) {
		return &collaboration.CreateShareResponse{Status: status.NewInvalid(ctx, "space type is not eligible for sharing")}, nil
	}
	// If the share is a denial we call  denyGrant instead.
	var st *rpc.Status
	var err error
	// TODO: change CS3 APIs
	opaque := utils.SpaceGrantOpaque()
	utils.AppendPlainToOpaque(
		opaque,
		"spacetype",
		req.ResourceInfo.GetSpace().GetSpaceType(),
	)
	if grants.PermissionsEqual(req.Grant.Permissions.Permissions, &provider.ResourcePermissions{}) {
		st, err = s.denyGrant(ctx, req.ResourceInfo.Id, req.Grant.Grantee, opaque)
		if err != nil {
			return nil, errors.Wrap(err, "gateway: error denying grant in storage")
		}
	} else {
		st, err = s.addGrant(ctx, req.ResourceInfo.Id, req.Grant.Grantee, req.Grant.Permissions.Permissions, req.Grant.Expiration, opaque)
		if err != nil {
			return nil, errors.Wrap(err, "gateway: error adding grant to storage")
		}
	}

	switch st.Code {
	case rpc.Code_CODE_OK:
		s.providerCache.RemoveListStorageProviders(req.ResourceInfo.Id)
	case rpc.Code_CODE_UNIMPLEMENTED:
		appctx.GetLogger(ctx).Debug().Interface("status", st).Interface("req", req).Msg("storing grants not supported, ignoring")
	default:
		return &collaboration.CreateShareResponse{
			Status: st,
		}, err
	}

	return &collaboration.CreateShareResponse{
		Status: status.NewOK(ctx),
		Share: &collaboration.Share{
			ResourceId:  req.ResourceInfo.Id,
			Permissions: &collaboration.SharePermissions{Permissions: req.Grant.Permissions.GetPermissions()},
			Grantee:     req.Grant.Grantee,
		},
	}, nil
}

func (s *svc) removeShare(ctx context.Context, req *collaboration.RemoveShareRequest) (*collaboration.RemoveShareResponse, error) {
	c, err := pool.GetUserShareProviderClient(s.c.UserShareProviderEndpoint)
	if err != nil {
		appctx.GetLogger(ctx).
			Err(err).
			Msg("RemoveShare: failed to get user share provider")
		return &collaboration.RemoveShareResponse{
			Status: status.NewInternal(ctx, "error getting user share provider client"),
		}, nil
	}

	// if we need to commit the share, we need the resource it points to.
	var share *collaboration.Share
	// FIXME: I will cause a panic as share will be nil when I'm false
	if s.c.CommitShareToStorageGrant {
		getShareReq := &collaboration.GetShareRequest{
			Ref: req.Ref,
		}
		getShareRes, err := c.GetShare(ctx, getShareReq)
		if err != nil {
			return nil, errors.Wrap(err, "gateway: error calling GetShare")
		}

		if getShareRes.Status.Code != rpc.Code_CODE_OK {
			res := &collaboration.RemoveShareResponse{
				Status: status.NewInternal(ctx,
					"error getting share when committing to the storage"),
			}
			return res, nil
		}
		share = getShareRes.Share
	}

	// TODO: update wopi server
	// FIXME This is a workaround that should prevent removing or changing the share permissions when the file is locked.
	// https://github.com/owncloud/ocis/issues/8474
	if status, err := s.checkShareLock(ctx, share); err != nil {
		return &collaboration.RemoveShareResponse{
			Status: status,
		}, nil
	}

	res, err := c.RemoveShare(ctx, req)
	if err != nil {
		return nil, errors.Wrap(err, "gateway: error calling RemoveShare")
	}
	if res.Status.Code != rpc.Code_CODE_OK {
		return res, nil
	}

	if s.c.CommitShareToStorageGrant {
		var opaque *typesv1beta1.Opaque
		if refIsSpaceRoot(share.ResourceId) {
			opaque = utils.SpaceGrantOpaque()
		}
		removeGrantStatus, err := s.removeGrant(ctx, share.ResourceId, share.Grantee, share.Permissions.Permissions, opaque)
		if err != nil {
			return nil, errors.Wrap(err, "gateway: error removing grant from storage")
		}
		if removeGrantStatus.Code != rpc.Code_CODE_OK {
			return &collaboration.RemoveShareResponse{
				Status: removeGrantStatus,
			}, err
		}
	}

	return res, nil
}

func (s *svc) removeSpaceShare(ctx context.Context, ref *provider.ResourceId, grantee *provider.Grantee) (*collaboration.RemoveShareResponse, error) {
	listGrantRes, err := s.listGrants(ctx, ref)
	if err != nil {
		return nil, errors.Wrap(err, "gateway: error getting grant to remove from storage")
	}
	var permissions *provider.ResourcePermissions
	for _, g := range listGrantRes.Grants {
		if isEqualGrantee(g.Grantee, grantee) {
			permissions = g.Permissions
		}
	}
	if permissions == nil {
		return nil, errors.New("gateway: error getting grant to remove from storage")
	}

	if len(listGrantRes.Grants) == 1 || !isSpaceManagerRemaining(listGrantRes.Grants, grantee) {
		return &collaboration.RemoveShareResponse{
			Status: status.NewPermissionDenied(ctx, errtypes.PermissionDenied(""), "can't remove the last manager"),
		}, nil
	}

	// TODO: change CS3 APIs
	opaque := utils.SpaceGrantOpaque()
	removeGrantStatus, err := s.removeGrant(ctx, ref, grantee, permissions, opaque)
	if err != nil {
		return nil, errors.Wrap(err, "gateway: error removing grant from storage")
	}
	if removeGrantStatus.Code != rpc.Code_CODE_OK {
		return &collaboration.RemoveShareResponse{
			Status: removeGrantStatus,
		}, err
	}
	s.providerCache.RemoveListStorageProviders(ref)
	return &collaboration.RemoveShareResponse{Status: status.NewOK(ctx)}, nil
}

func isSpaceManagerRemaining(grants []*provider.Grant, grantee *provider.Grantee) bool {
	for _, g := range grants {
		// RemoveGrant is currently the way to check for the manager role
		// If it is not set than the current grant is not for a manager and
		// we can just continue with the next one.
		if g.Permissions.RemoveGrant && !isEqualGrantee(g.Grantee, grantee) {
			return true
		}
	}
	return false
}

func (s *svc) checkLock(ctx context.Context, shareId *collaboration.ShareId) (*rpc.Status, error) {
	logger := appctx.GetLogger(ctx)
	getShareRes, err := s.GetShare(ctx, &collaboration.GetShareRequest{
		Ref: &collaboration.ShareReference{
			Spec: &collaboration.ShareReference_Id{Id: shareId},
		},
	})
	if err != nil {
		msg := "gateway: error calling GetShare"
		logger.Err(err).Interface("share_id", shareId).Msg(msg)
		return status.NewInternal(ctx, msg), errors.Wrap(err, msg)
	}
	if getShareRes.GetStatus().GetCode() != rpc.Code_CODE_OK {
		msg := "can not get share stat " + getShareRes.GetStatus().GetMessage()
		logger.Debug().Interface("share", shareId).Msg(msg)
		if getShareRes.GetStatus().GetCode() != rpc.Code_CODE_NOT_FOUND {
			return status.NewNotFound(ctx, msg), errors.New(msg)
		}
		return status.NewInternal(ctx, msg), errors.New(msg)
	}
	return s.checkShareLock(ctx, getShareRes.Share)
}

func (s *svc) checkShareLock(ctx context.Context, share *collaboration.Share) (*rpc.Status, error) {
	logger := appctx.GetLogger(ctx)
	sRes, err := s.Stat(ctx, &provider.StatRequest{Ref: &provider.Reference{ResourceId: share.GetResourceId()},
		ArbitraryMetadataKeys: []string{"lockdiscovery"}})
	if err != nil {
		msg := "failed to stat shared resource"
		logger.Err(err).Interface("resource_id", share.GetResourceId()).Msg(msg)
		return status.NewInternal(ctx, msg), errors.Wrap(err, msg)
	}
	if sRes.GetStatus().GetCode() != rpc.Code_CODE_OK {
		msg := "can not get share stat " + sRes.GetStatus().GetMessage()
		logger.Debug().Interface("lock", sRes.GetInfo().GetLock()).Msg(msg)
		if sRes.GetStatus().GetCode() != rpc.Code_CODE_NOT_FOUND {
			return status.NewNotFound(ctx, msg), errors.New(msg)
		}
		return status.NewInternal(ctx, msg), errors.New(msg)
	}

	if sRes.GetInfo().GetLock() != nil {
		msg := "can not change grants, the shared resource is locked"
		logger.Debug().Interface("lock", sRes.GetInfo().GetLock()).Msg(msg)
		return status.NewLocked(ctx, msg), errors.New(msg)
	}
	return nil, nil
}

func refIsSpaceRoot(ref *provider.ResourceId) bool {
	if ref == nil {
		return false
	}
	if ref.SpaceId == "" || ref.OpaqueId == "" {
		return false
	}

	return ref.SpaceId == ref.OpaqueId
}

func shareIsSpaceRoot(key *collaboration.ShareKey) bool {
	if key == nil {
		return false
	}
	return refIsSpaceRoot(key.ResourceId)
}

func isEqualGrantee(a, b *provider.Grantee) bool {
	// Ideally we would want to use utils.GranteeEqual()
	// but the grants stored in the decomposedfs aren't complete (missing usertype and idp)
	// because of that the check would fail so we can only check the ... for now.
	if a.Type != b.Type {
		return false
	}

	var aID, bID string
	switch a.Type {
	case provider.GranteeType_GRANTEE_TYPE_GROUP:
		aID = a.GetGroupId().GetOpaqueId()
		bID = b.GetGroupId().GetOpaqueId()
	case provider.GranteeType_GRANTEE_TYPE_USER:
		aID = a.GetUserId().GetOpaqueId()
		bID = b.GetUserId().GetOpaqueId()
	}
	return aID == bID
}
