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

package shares

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	userv1beta1 "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"
	rpc "github.com/cs3org/go-cs3apis/cs3/rpc/v1beta1"
	link "github.com/cs3org/go-cs3apis/cs3/sharing/link/v1beta1"
	provider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	types "github.com/cs3org/go-cs3apis/cs3/types/v1beta1"
	"github.com/huandu/xstrings"
	"github.com/opencloud-eu/reva/v3/pkg/conversions"
	"github.com/opencloud-eu/reva/v3/pkg/permission"
	"github.com/opencloud-eu/reva/v3/pkg/rgrpc/todo/pool"
	"github.com/opencloud-eu/reva/v3/pkg/utils"
	"github.com/rs/zerolog/log"

	"github.com/opencloud-eu/reva/v3/internal/http/services/owncloud/ocs/response"
	"github.com/opencloud-eu/reva/v3/pkg/appctx"
	ctxpkg "github.com/opencloud-eu/reva/v3/pkg/ctx"
	"github.com/opencloud-eu/reva/v3/pkg/publicshare"
	"github.com/pkg/errors"
)

var _defaultPublicLinkPermission = 1

func (h *Handler) createPublicLinkShare(w http.ResponseWriter, r *http.Request, statInfo *provider.ResourceInfo) (*link.PublicShare, *ocsError) {
	ctx := r.Context()
	log := appctx.GetLogger(ctx)

	c, err := h.getClient()
	if err != nil {
		return nil, &ocsError{
			Code:    response.MetaServerError.StatusCode,
			Message: "error getting grpc gateway client",
			Error:   err,
		}
	}

	permKey, err := permKeyFromRequest(r, h)
	if err != nil {
		return nil, &ocsError{
			Code:    response.MetaBadRequest.StatusCode,
			Message: "Could not read permission from request",
			Error:   err,
		}
	}

	// NOTE: one is allowed to create an internal link without the `Publink.Write` permission
	if permKey != nil && *permKey != 0 {
		ok, err := utils.CheckPermission(ctx, permission.WritePublicLink, c)
		if err != nil {
			return nil, &ocsError{
				Code:    response.MetaServerError.StatusCode,
				Message: "failed to check user permission",
				Error:   err,
			}
		}
		if !ok {
			return nil, &ocsError{
				Code:    response.MetaForbidden.StatusCode,
				Message: "user is not allowed to create a public link",
			}
		}
	}

	err = r.ParseForm()
	if err != nil {
		return nil, &ocsError{
			Code:    response.MetaBadRequest.StatusCode,
			Message: "Could not parse form from request",
			Error:   err,
		}
	}

	// check if a quicklink should be created
	quick, _ := strconv.ParseBool(r.FormValue("quicklink")) // no need to check the error - defaults to zero value!
	if quick {
		f := []*link.ListPublicSharesRequest_Filter{publicshare.ResourceIDFilter(statInfo.Id)}
		req := link.ListPublicSharesRequest{Filters: f}
		res, err := c.ListPublicShares(ctx, &req)
		if err != nil {
			return nil, &ocsError{
				Code:    response.MetaServerError.StatusCode,
				Message: "could not list public links",
				Error:   err,
			}
		}
		if res.Status.Code != rpc.Code_CODE_OK {
			return nil, &ocsError{
				Code:    int(res.Status.GetCode()),
				Message: "could not list public links",
			}
		}

		for _, l := range res.GetShare() {
			if l.Quicklink {
				return l, nil
			}
		}
	}

	// default perms: read-only
	// TODO: the default might change depending on allowed permissions and configs
	if permKey == nil {
		permKey = &_defaultPublicLinkPermission
	}
	permissions, err := ocPublicPermToCs3(permKey)
	if err != nil {
		return nil, &ocsError{
			Code:    response.MetaBadRequest.StatusCode,
			Message: "Could not create permission from permission key",
			Error:   err,
		}
	}

	password := r.FormValue("password")
	if h.enforcePassword(permKey) && len(password) == 0 {
		return nil, &ocsError{
			Code:    response.MetaBadRequest.StatusCode,
			Message: "missing required password",
			Error:   errors.New("missing required password"),
		}
	}
	if len(password) > 0 {
		if err := h.passwordValidator.Validate(password); err != nil {
			return nil, &ocsError{
				Code:    response.MetaBadRequest.StatusCode,
				Message: xstrings.FirstRuneToUpper(err.Error()),
				Error:   fmt.Errorf("password validation failed: %w", err),
			}
		}
	}

	if statInfo != nil && statInfo.Type == provider.ResourceType_RESOURCE_TYPE_FILE {
		// Single file shares should never have delete or create permissions
		role := conversions.RoleFromResourcePermissions(permissions, true)
		p := role.OCSPermissions()
		p &^= conversions.PermissionCreate
		p &^= conversions.PermissionDelete
		permissions = conversions.RoleFromOCSPermissions(p, statInfo).CS3ResourcePermissions()
	}

	if !sufficientPermissions(statInfo.PermissionSet, permissions, true) {
		response.WriteOCSError(w, r, http.StatusForbidden, "no share permission", nil)
		return nil, &ocsError{
			Code:    http.StatusForbidden,
			Message: "Cannot set the requested share permissions",
			Error:   errors.New("cannot set the requested share permissions"),
		}
	}

	req := link.CreatePublicShareRequest{
		ResourceInfo: statInfo,
		Grant: &link.Grant{
			Permissions: &link.PublicSharePermissions{
				Permissions: permissions,
			},
			Password: password,
		},
	}

	expireTimeString, ok := r.Form["expireDate"]
	if ok {
		if expireTimeString[0] != "" {
			expireTime, err := conversions.ParseTimestamp(expireTimeString[0])
			if err != nil {
				return nil, &ocsError{
					Code:    response.MetaBadRequest.StatusCode,
					Message: err.Error(),
					Error:   err,
				}
			}
			if expireTime != nil {
				req.Grant.Expiration = expireTime
			}
		}
	}

	// set displayname and password protected as arbitrary metadata
	req.ResourceInfo.ArbitraryMetadata = &provider.ArbitraryMetadata{
		Metadata: map[string]string{
			"name":      r.FormValue("name"),
			"quicklink": r.FormValue("quicklink"),
		},
	}

	createRes, err := c.CreatePublicShare(ctx, &req)
	if err != nil {
		log.Debug().Err(err).Str("createShare", "shares").Msgf("error creating a public share to resource id: %v", statInfo.GetId())
		return nil, &ocsError{
			Code:    response.MetaServerError.StatusCode,
			Message: "error creating public share",
			Error:   fmt.Errorf("error creating a public share to resource id: %v", statInfo.GetId()),
		}
	}

	if createRes.Status.Code != rpc.Code_CODE_OK {
		log.Debug().Err(errors.New("create public share failed")).Str("shares", "createShare").Msgf("create public share failed with status code: %v", createRes.Status.Code.String())
		return nil, &ocsError{
			Code:    response.MetaServerError.StatusCode,
			Message: "grpc create public share request failed",
			Error:   nil,
		}
	}
	return createRes.Share, nil
}

func (h *Handler) listPublicShares(r *http.Request, filters []*link.ListPublicSharesRequest_Filter) ([]*conversions.ShareData, *rpc.Status, error) {
	ctx := r.Context()
	log := appctx.GetLogger(ctx)

	ocsDataPayload := make([]*conversions.ShareData, 0)
	// TODO(refs) why is this guard needed? Are we moving towards a gateway only for service discovery? without a gateway this is dead code.
	if h.gatewayAddr != "" {
		client, err := pool.GetGatewayServiceClient(h.gatewayAddr)
		if err != nil {
			return ocsDataPayload, nil, err
		}

		req := link.ListPublicSharesRequest{
			Filters: filters,
		}

		res, err := client.ListPublicShares(ctx, &req)
		if err != nil {
			return ocsDataPayload, nil, err
		}
		if res.Status.Code != rpc.Code_CODE_OK {
			return ocsDataPayload, res.Status, nil
		}

		for _, share := range res.GetShare() {
			info, status, err := h.getResourceInfoByID(ctx, client, share.ResourceId)
			if err != nil || status.Code != rpc.Code_CODE_OK {
				log.Debug().Interface("share", share).Interface("status", status).Err(err).Msg("could not stat share, skipping")
				continue
			}

			sData := conversions.PublicShare2ShareData(share, r, h.publicURL)

			sData.Name = share.DisplayName

			h.addFileInfo(ctx, sData, info)
			h.mapUserIds(ctx, client, sData)

			log.Debug().Interface("share", share).Interface("info", info).Interface("shareData", share).Msg("mapped")

			ocsDataPayload = append(ocsDataPayload, sData)

		}

		return ocsDataPayload, nil, nil
	}

	return ocsDataPayload, nil, errors.New("bad request")
}

func (h *Handler) isPublicShare(r *http.Request, oid string) (*link.PublicShare, bool) {
	logger := appctx.GetLogger(r.Context())
	client, err := h.getClient()
	if err != nil {
		logger.Err(err)
		return nil, false
	}

	psRes, err := client.GetPublicShare(r.Context(), &link.GetPublicShareRequest{
		Ref: &link.PublicShareReference{
			Spec: &link.PublicShareReference_Id{
				Id: &link.PublicShareId{
					OpaqueId: oid,
				},
			},
		},
	})
	switch {
	case err != nil:
		log.Err(err).Send()
	case psRes.Status.Code == rpc.Code_CODE_OK:
		return psRes.GetShare(), psRes.GetShare() != nil
	case psRes.Status.Code == rpc.Code_CODE_INTERNAL:
		log.Error().Str("message", psRes.GetStatus().GetMessage()).Str("code", psRes.GetStatus().GetCode().String()).Msg("isPublicShare received internal error")
	default:
		log.Debug().Str("message", psRes.GetStatus().GetMessage()).Str("code", psRes.GetStatus().GetCode().String()).Msg("isPublicShare received unexpected status")
	}
	return nil, false
}

func (h *Handler) updatePublicShare(w http.ResponseWriter, r *http.Request, share *link.PublicShare) {
	updates := []*link.UpdatePublicShareRequest_Update{}
	logger := appctx.GetLogger(r.Context())

	gwC, err := h.getClient()
	if err != nil {
		log.Err(err).Str("shareID", share.GetId().GetOpaqueId()).Msg("updatePublicShare")
		response.WriteOCSError(w, r, response.MetaBadRequest.StatusCode, "error getting a connection to the gateway service", nil)
		return
	}

	ctx := r.Context()
	user := ctxpkg.ContextMustGetUser(ctx)

	permKey, err := permKeyFromRequest(r, h)
	if err != nil {
		response.WriteOCSError(w, r, response.MetaBadRequest.StatusCode, "invalid permissions", err)
		return
	}

	createdByUser := publicshare.IsCreatedByUser(share, user)

	// NOTE: you are allowed to update a link TO a public link without the `PublicLink.Write` permission if you created it yourself
	if (permKey != nil && *permKey != 0) || !createdByUser {
		ok, err := utils.CheckPermission(ctx, permission.WritePublicLink, gwC)
		if err != nil {
			response.WriteOCSError(w, r, response.MetaServerError.StatusCode, "failed to check user permission", err)
			return
		}
		if !ok {
			response.WriteOCSError(w, r, response.MetaForbidden.StatusCode, "user is not allowed to update the public link", nil)
			return
		}
	}

	if !createdByUser {
		sRes, err := gwC.Stat(r.Context(), &provider.StatRequest{Ref: &provider.Reference{ResourceId: share.ResourceId}})
		if err != nil {
			log.Err(err).Interface("resource_id", share.ResourceId).Msg("failed to stat shared resource")
			response.WriteOCSError(w, r, response.MetaServerError.StatusCode, "failed to get public share", nil)
			return
		}

		if sRes.Info == nil || !sRes.Info.GetPermissionSet().UpdateGrant {
			response.WriteOCSError(w, r, response.MetaUnauthorized.StatusCode, "missing permissions to update share", err)
			return
		}
	}

	err = r.ParseForm()
	if err != nil {
		response.WriteOCSError(w, r, response.MetaBadRequest.StatusCode, "Could not parse form from request", err)
		return
	}

	// indicates whether values to update were found,
	// to check if the request was valid,
	// not whether an actual update has been performed
	updatesFound := false

	newName, ok := r.Form["name"]
	if ok {
		updatesFound = true
		if newName[0] != share.DisplayName {
			updates = append(updates, &link.UpdatePublicShareRequest_Update{
				Type:        link.UpdatePublicShareRequest_Update_TYPE_DISPLAYNAME,
				DisplayName: newName[0],
			})
		}
	}

	// Permissions
	newPermissions, err := ocPublicPermToCs3(permKey)
	logger.Debug().Interface("newPermissions", newPermissions).Msg("Parsed permissions")
	if err != nil {
		response.WriteOCSError(w, r, response.MetaBadRequest.StatusCode, "invalid permissions", err)
		return
	}

	// update permissions if given
	if newPermissions != nil {
		updatesFound = true
		publicSharePermissions := &link.PublicSharePermissions{
			Permissions: newPermissions,
		}
		beforePerm, _ := json.Marshal(share.Permissions)
		afterPerm, _ := json.Marshal(publicSharePermissions)
		if string(beforePerm) != string(afterPerm) {
			logger.Info().Str("shares", "update").Msgf("updating permissions from %v to: %v", string(beforePerm), string(afterPerm))
			updates = append(updates, &link.UpdatePublicShareRequest_Update{
				Type: link.UpdatePublicShareRequest_Update_TYPE_PERMISSIONS,
				Grant: &link.Grant{
					Permissions: publicSharePermissions,
					Password:    r.FormValue("password"),
				},
			})
		}
	}

	statReq := provider.StatRequest{Ref: &provider.Reference{ResourceId: share.ResourceId}}
	statRes, err := gwC.Stat(r.Context(), &statReq)
	if err != nil {
		log.Debug().Err(err).Str("shares", "update public share").Msg("error during stat")
		response.WriteOCSError(w, r, response.MetaServerError.StatusCode, "missing resource information", fmt.Errorf("error getting resource information"))
		return
	}
	if statRes.GetStatus().GetCode() != rpc.Code_CODE_OK {
		if statRes.GetStatus().GetCode() == rpc.Code_CODE_NOT_FOUND {
			response.WriteOCSError(w, r, response.MetaNotFound.StatusCode, "update public share: resource not found", err)
			return
		}
		response.WriteOCSError(w, r, response.MetaServerError.StatusCode, "grpc stat request failed for stat when updating a public share", err)
		return
	}

	// empty permissions mean internal link here - NOT denial. Hence we need an extra check
	if newPermissions != nil {
		if !sufficientPermissions(statRes.GetInfo().GetPermissionSet(), newPermissions, true) {
			response.WriteOCSError(w, r, http.StatusForbidden, "no share permission", nil)
			return
		}
	} else {
		statRes.GetInfo().GetPermissionSet()
		p := decreasePermissionsIfNecessary(int(conversions.RoleFromResourcePermissions(statRes.GetInfo().GetPermissionSet(), false).OCSPermissions()))
		permKey = &p
	}

	// ExpireDate
	expireTimeString, ok := r.Form["expireDate"]
	// check if value is set and must be updated or cleared
	if ok {
		updatesFound = true
		var newExpiration *types.Timestamp
		if expireTimeString[0] != "" {
			newExpiration, err = conversions.ParseTimestamp(expireTimeString[0])
			if err != nil {
				response.WriteOCSError(w, r, response.MetaBadRequest.StatusCode, "invalid datetime format", err)
				return
			}
		}

		beforeExpiration, _ := json.Marshal(share.Expiration)
		afterExpiration, _ := json.Marshal(newExpiration)
		if string(afterExpiration) != string(beforeExpiration) {
			logger.Debug().Str("shares", "update").Msgf("updating expire date from %v to: %v", string(beforeExpiration), string(afterExpiration))
			updates = append(updates, &link.UpdatePublicShareRequest_Update{
				Type: link.UpdatePublicShareRequest_Update_TYPE_EXPIRATION,
				Grant: &link.Grant{
					Expiration: newExpiration,
				},
			})
		}
	}

	// Password
	newPassword, ok := r.Form["password"]
	// enforcePassword
	if h.enforcePassword(permKey) {
		p, err := conversions.NewPermissions(decreasePermissionsIfNecessary(*permKey))
		if err != nil {
			response.WriteOCSError(w, r, response.MetaServerError.StatusCode, "failed to check permissions from request", err)
			return
		}
		if !ok && !share.PasswordProtected || ok && len(newPassword[0]) == 0 {
			if h.checkPasswordEnforcement(ctx, user, p, w, r) != nil {
				return
			}
		}
	}

	// update or clear password
	if ok {
		// skip validation if the clear password scenario
		if len(newPassword[0]) > 0 {
			if err := h.passwordValidator.Validate(newPassword[0]); err != nil {
				response.WriteOCSError(w, r, response.MetaBadRequest.StatusCode, xstrings.FirstRuneToUpper(err.Error()), err)
				return
			}
		}
		updatesFound = true
		logger.Info().Str("shares", "update").Msg("password updated")
		updates = append(updates, &link.UpdatePublicShareRequest_Update{
			Type: link.UpdatePublicShareRequest_Update_TYPE_PASSWORD,
			Grant: &link.Grant{
				Password: newPassword[0],
			},
		})
	}

	// Updates are atomical. See: https://github.com/cs3org/cs3apis/pull/67#issuecomment-617651428 so in order to get the latest updated version
	if len(updates) > 0 {
		uRes := &link.UpdatePublicShareResponse{Share: share}
		for k := range updates {
			uRes, err = gwC.UpdatePublicShare(r.Context(), &link.UpdatePublicShareRequest{
				Ref: &link.PublicShareReference{
					Spec: &link.PublicShareReference_Id{
						Id: &link.PublicShareId{
							OpaqueId: share.Id.OpaqueId,
						},
					},
				},
				Update: updates[k],
			})
			if err != nil {
				log.Err(err).Str("shareID", share.Id.OpaqueId).Msg("sending update request to public link provider")
				response.WriteOCSError(w, r, response.MetaServerError.StatusCode, "Error sending update request to public link provider", err)
				return
			}
			if uRes.Status.Code != rpc.Code_CODE_OK {
				log.Debug().Str("shareID", share.Id.OpaqueId).Msgf("sending update request to public link provider failed: %s", uRes.Status.Message)
				response.WriteOCSError(w, r, response.MetaServerError.StatusCode, fmt.Sprintf("Error sending update request to public link provider: %s", uRes.Status.Message), nil)
				return
			}
		}
		share = uRes.Share
	} else if !updatesFound {
		response.WriteOCSError(w, r, response.MetaBadRequest.StatusCode, "No updates specified in request", nil)
		return
	}

	s := conversions.PublicShare2ShareData(share, r, h.publicURL)
	h.addFileInfo(r.Context(), s, statRes.Info)
	h.mapUserIds(r.Context(), gwC, s)

	response.WriteOCSSuccess(w, r, s)
}

func (h *Handler) removePublicShare(w http.ResponseWriter, r *http.Request, share *link.PublicShare) {
	ctx := r.Context()

	c, err := h.getClient()
	if err != nil {
		response.WriteOCSError(w, r, response.MetaServerError.StatusCode, "error getting grpc gateway client", err)
		return
	}

	u := ctxpkg.ContextMustGetUser(ctx)
	if !publicshare.IsCreatedByUser(share, u) {
		sRes, err := c.Stat(r.Context(), &provider.StatRequest{Ref: &provider.Reference{ResourceId: share.ResourceId}})
		if err != nil {
			log.Err(err).Interface("resource_id", share.ResourceId).Msg("failed to stat shared resource")
			response.WriteOCSError(w, r, response.MetaServerError.StatusCode, "failed to get public share", nil)
			return
		}

		if !sRes.Info.PermissionSet.RemoveGrant {
			response.WriteOCSError(w, r, response.MetaUnauthorized.StatusCode, "missing permissions to remove share", err)
			return
		}
	}

	req := &link.RemovePublicShareRequest{
		Ref: &link.PublicShareReference{
			Spec: &link.PublicShareReference_Id{
				Id: &link.PublicShareId{
					OpaqueId: share.GetId().GetOpaqueId(),
				},
			},
		},
	}

	res, err := c.RemovePublicShare(ctx, req)
	if err != nil {
		response.WriteOCSError(w, r, response.MetaServerError.StatusCode, "error sending a grpc delete share request", err)
		return
	}
	if res.Status.Code != rpc.Code_CODE_OK {
		if res.Status.Code == rpc.Code_CODE_NOT_FOUND {
			response.WriteOCSError(w, r, response.MetaNotFound.StatusCode, "not found", nil)
			return
		}
		response.WriteOCSError(w, r, response.MetaServerError.StatusCode, "grpc delete share request failed", err)
		return
	}

	response.WriteOCSSuccess(w, r, nil)
}

// enforcePassword validate Password enforce based on configuration
// read_only:           1
// read_write:          3 or 5
// read_write_delete:   15
// upload_only:         4
func (h *Handler) enforcePassword(pk *int) bool {
	if pk == nil {
		return false
	}
	p, err := conversions.NewPermissions(decreasePermissionsIfNecessary(*pk))
	if err != nil {
		return false
	}
	if h.publicPasswordEnforced.EnforcedForReadOnly &&
		p == conversions.PermissionRead {
		return true
	}
	if h.publicPasswordEnforced.EnforcedForReadWrite &&
		(p == conversions.PermissionRead|conversions.PermissionWrite ||
			p == conversions.PermissionRead|conversions.PermissionCreate) {
		return true
	}
	if h.publicPasswordEnforced.EnforcedForReadWriteDelete &&
		p == conversions.PermissionRead|conversions.PermissionWrite|conversions.PermissionCreate|conversions.PermissionDelete {
		return true
	}
	if h.publicPasswordEnforced.EnforcedForUploadOnly &&
		p == conversions.PermissionCreate {
		return true
	}
	return false
}

// for public links oc10 api decreases all permissions to read: stay compatible!
func decreasePermissionsIfNecessary(perm int) int {
	if perm == int(conversions.PermissionAll) {
		perm = int(conversions.PermissionRead)
	}
	return perm
}

func ocPublicPermToCs3(pk *int) (*provider.ResourcePermissions, error) {
	if pk == nil {
		return nil, nil
	}

	permKey := decreasePermissionsIfNecessary(*pk)

	// TODO refactor this ocPublicPermToRole[permKey] check into a conversions.NewPublicSharePermissions?
	// not all permissions are possible for public shares
	_, ok := ocPublicPermToRole[permKey]
	if !ok {
		log.Error().Str("ocPublicPermToCs3", "shares").Int("perm", permKey).Msg("invalid public share permission")
		return nil, fmt.Errorf("invalid public share permission: %d", permKey)
	}

	perm, err := conversions.NewPermissions(permKey)
	if err != nil && err != conversions.ErrZeroPermission { // we allow empty permissions for public links
		return nil, err
	}

	return conversions.RoleFromOCSPermissions(perm, nil).CS3ResourcePermissions(), nil
}

// pointer will be nil if no permission is set
func permKeyFromRequest(r *http.Request, h *Handler) (*int, error) {
	var err error
	// phoenix sends: {"permissions": 15}. See ocPublicPermToRole struct for mapping

	permKey := 1

	// note: "permissions" value has higher priority than "publicUpload"

	// handle legacy "publicUpload" arg that overrides permissions differently depending on the scenario
	// https://github.com/owncloud/core/blob/v10.4.0/apps/files_sharing/lib/Controller/Share20OcsController.php#L447
	publicUploadString := r.FormValue("publicUpload")
	if publicUploadString != "" {
		publicUploadFlag, err := strconv.ParseBool(publicUploadString)
		if err != nil {
			log.Error().Err(err).Str("publicUpload", publicUploadString).Msg("could not parse publicUpload argument")
			return nil, err
		}

		if publicUploadFlag {
			// all perms except reshare
			permKey = 15
		}
	} else {
		permissionsString := r.FormValue("permissions")
		if permissionsString == "" {
			// no permission values given
			return nil, nil
		}

		permKey, err = strconv.Atoi(permissionsString)
		if err != nil {
			log.Error().Str("permissionFromRequest", "shares").Msgf("invalid type: %T", permKey)
			return nil, fmt.Errorf("invalid type: %T", permKey)
		}
	}

	return &permKey, nil
}

// checkPasswordEnforcement checks if the password needs to be set for a link
// some users can opt out of the enforcement based on a user permission
func (h *Handler) checkPasswordEnforcement(ctx context.Context, user *userv1beta1.User, perm conversions.Permissions, w http.ResponseWriter, r *http.Request) error {
	// Non-read-only links
	if perm != conversions.PermissionRead {
		response.WriteOCSError(w, r, response.MetaBadRequest.StatusCode, "missing required password", nil)
		return errors.New("missing required password")
	}
	// Check if the user is allowed to opt out of the password enforcement
	// for read-only links
	gwC, err := h.getClient()
	if err != nil {
		response.WriteOCSError(w, r, response.MetaServerError.StatusCode, "could not check permission", err)
		return errors.New("could not check permission")
	}
	ok, err := utils.CheckPermission(ctx, permission.DeleteReadOnlyPassword, gwC)
	if err != nil {
		response.WriteOCSError(w, r, response.MetaServerError.StatusCode, "failed to check user permission", err)
		return errors.New("failed to check user permission")
	}
	if !ok {
		response.WriteOCSError(w, r, response.MetaForbidden.StatusCode, "user is not allowed to delete the password from the public link", nil)
		return errors.New("user is not allowed to delete the password from the public link")
	}
	return nil
}

// TODO: add mapping for user share permissions to role

// Maps oc10 public link permissions to roles
var ocPublicPermToRole = map[int]string{
	// Recipients can do nothing
	0: "none",
	// Recipients can view and download contents.
	1: "viewer",
	// Recipients can view, download and edit single files.
	3: "file-editor",
	// Recipients can view, download, edit, delete and upload contents
	15: "editor",
	// Recipients can upload but existing contents are not revealed
	4: "uploader",
	// Recipients can view, download and upload contents
	5: "contributor",
}
