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
	"fmt"
	"net/http"
	"time"

	groupv1beta1 "github.com/cs3org/go-cs3apis/cs3/identity/group/v1beta1"
	userpb "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"
	rpc "github.com/cs3org/go-cs3apis/cs3/rpc/v1beta1"
	collaborationv1beta1 "github.com/cs3org/go-cs3apis/cs3/sharing/collaboration/v1beta1"
	provider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	types "github.com/cs3org/go-cs3apis/cs3/types/v1beta1"
	"github.com/opencloud-eu/reva/v2/internal/http/services/owncloud/ocs/response"
	"github.com/opencloud-eu/reva/v2/pkg/appctx"
	"github.com/opencloud-eu/reva/v2/pkg/conversions"
	"github.com/opencloud-eu/reva/v2/pkg/storagespace"
	"github.com/opencloud-eu/reva/v2/pkg/utils"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
)

func (h *Handler) getGrantee(ctx context.Context, name string) (provider.Grantee, error) {
	log := appctx.GetLogger(ctx)
	client, err := h.getClient()
	if err != nil {
		return provider.Grantee{}, err
	}
	userRes, err := client.GetUserByClaim(ctx, &userpb.GetUserByClaimRequest{
		Claim: "username",
		Value: name,
	})
	if err == nil && userRes.Status.Code == rpc.Code_CODE_OK {
		return provider.Grantee{
			Type: provider.GranteeType_GRANTEE_TYPE_USER,
			Id:   &provider.Grantee_UserId{UserId: userRes.User.Id},
		}, nil
	}
	log.Debug().Str("name", name).Msg("no user found")

	groupRes, err := client.GetGroupByClaim(ctx, &groupv1beta1.GetGroupByClaimRequest{
		Claim:               "group_name",
		Value:               name,
		SkipFetchingMembers: true,
	})
	if err == nil && groupRes.Status.Code == rpc.Code_CODE_OK {
		return provider.Grantee{
			Type: provider.GranteeType_GRANTEE_TYPE_GROUP,
			Id:   &provider.Grantee_GroupId{GroupId: groupRes.Group.Id},
		}, nil
	}
	log.Debug().Str("name", name).Msg("no group found")

	return provider.Grantee{}, fmt.Errorf("no grantee found with name %s", name)
}

func (h *Handler) addSpaceMember(w http.ResponseWriter, r *http.Request, info *provider.ResourceInfo, role *conversions.Role, roleVal []byte) {
	ctx := r.Context()

	if info.Space.SpaceType == "personal" {
		response.WriteOCSError(w, r, response.MetaBadRequest.StatusCode, "can not add members to personal spaces", nil)
		return
	}

	shareWith := r.FormValue("shareWith")
	if shareWith == "" {
		response.WriteOCSError(w, r, response.MetaBadRequest.StatusCode, "missing shareWith", nil)
		return
	}

	grantee, err := h.getGrantee(ctx, shareWith)
	if err != nil {
		response.WriteOCSError(w, r, response.MetaNotFound.StatusCode, "error getting grantee", err)
		return
	}

	client, err := h.getClient()
	if err != nil {
		response.WriteOCSError(w, r, response.MetaNotFound.StatusCode, "error getting gateway client", err)
		return
	}

	permissions := role.CS3ResourcePermissions()
	// All members of a space should be able to list shares inside that space.
	// The viewer role doesn't have the ListGrants permission so we set it here.
	permissions.ListGrants = true

	fieldmask := []string{}
	expireDate := r.PostFormValue("expireDate")
	var expirationTs *types.Timestamp
	fieldmask = append(fieldmask, "expiration")
	if expireDate != "" {
		expiration, err := time.Parse(_iso8601, expireDate)
		if err != nil {
			// Web sends different formats when adding and when editing a space membership...
			// We need to fix this in a separate PR.
			expiration, err = time.Parse(time.RFC3339, expireDate)
			if err != nil {
				response.WriteOCSError(w, r, response.MetaBadRequest.StatusCode, "could not parse expireDate", err)
				return
			}
		}
		expirationTs = &types.Timestamp{
			Seconds: uint64(expiration.UnixNano() / int64(time.Second)),
			Nanos:   uint32(expiration.UnixNano() % int64(time.Second)),
		}
		fieldmask = append(fieldmask, "expiration")
	}

	lsRes, err := client.ListShares(ctx, &collaborationv1beta1.ListSharesRequest{
		Filters: []*collaborationv1beta1.Filter{
			{
				Type: collaborationv1beta1.Filter_TYPE_RESOURCE_ID,
				Term: &collaborationv1beta1.Filter_ResourceId{
					ResourceId: info.GetId(),
				},
			},
		},
	})
	if err != nil || lsRes.Status.Code != rpc.Code_CODE_OK {
		response.WriteOCSError(w, r, response.MetaServerError.StatusCode, "error listing space members", err)
		return
	}

	if !isSpaceManagerRemainingInShares(lsRes.Shares, &grantee) {
		response.WriteOCSError(w, r, http.StatusForbidden, "the space must have at least one manager", nil)
		return
	}

	if existingShare := findShareByGrantee(lsRes.Shares, &grantee); existingShare != nil {
		if permissions != nil {
			fieldmask = append(fieldmask, "permissions")
		}
		updateShareReq := &collaborationv1beta1.UpdateShareRequest{
			Opaque: utils.AppendPlainToOpaque(nil, "spacetype", info.GetSpace().GetSpaceType()),
			Share: &collaborationv1beta1.Share{
				Id: existingShare.Id,
				Permissions: &collaborationv1beta1.SharePermissions{
					Permissions: permissions,
				},
				Grantee:    &grantee,
				Expiration: expirationTs,
			},
			UpdateMask: &fieldmaskpb.FieldMask{
				Paths: fieldmask,
			},
		}
		updateShareRes, err := client.UpdateShare(ctx, updateShareReq)
		if err != nil || updateShareRes.Status.Code != rpc.Code_CODE_OK {
			response.WriteOCSError(w, r, response.MetaServerError.StatusCode, "could not update space member", err)
			return
		}
	} else {
		createShareRes, err := client.CreateShare(ctx, &collaborationv1beta1.CreateShareRequest{
			ResourceInfo: info,
			Grant: &collaborationv1beta1.ShareGrant{
				Permissions: &collaborationv1beta1.SharePermissions{
					Permissions: permissions,
				},
				Grantee:    &grantee,
				Expiration: expirationTs,
			},
		})
		if err != nil || createShareRes.Status.Code != rpc.Code_CODE_OK {
			response.WriteOCSError(w, r, response.MetaServerError.StatusCode, "could not add space member", err)
			return
		}
	}

	response.WriteOCSSuccess(w, r, nil)
}

func (h *Handler) isSpaceShare(r *http.Request, shareID string) bool {
	ref, err := storagespace.ParseReference(shareID)
	// NOTE: we ignore the 'Path' part of the reference here as we're just interested in the space root
	switch {
	case err != nil:
		return false
	case ref.GetResourceId().GetSpaceId() == "":
		return false
	case ref.GetResourceId().GetOpaqueId() == "" || ref.GetResourceId().GetSpaceId() == ref.GetResourceId().GetOpaqueId():
		return true
	default:
		return false
	}
}

func (h *Handler) removeSpaceMember(w http.ResponseWriter, r *http.Request, spaceID string) {
	ctx := r.Context()

	shareWith := r.URL.Query().Get("shareWith")
	if shareWith == "" {
		response.WriteOCSError(w, r, response.MetaBadRequest.StatusCode, "missing shareWith", nil)
		return
	}

	grantee, err := h.getGrantee(ctx, shareWith)
	if err != nil {
		response.WriteOCSError(w, r, response.MetaNotFound.StatusCode, "error getting grantee", err)
		return
	}

	ref, err := storagespace.ParseReference(spaceID)
	if err != nil {
		response.WriteOCSError(w, r, response.MetaBadRequest.StatusCode, "could not parse space id", err)
		return
	}

	if ref.ResourceId.OpaqueId == "" {
		ref.ResourceId.OpaqueId = ref.ResourceId.SpaceId
	}

	client, err := h.getClient()
	if err != nil {
		response.WriteOCSError(w, r, response.MetaNotFound.StatusCode, "error getting gateway client", err)
		return
	}

	lsRes, err := client.ListShares(ctx, &collaborationv1beta1.ListSharesRequest{
		Filters: []*collaborationv1beta1.Filter{
			{
				Type: collaborationv1beta1.Filter_TYPE_RESOURCE_ID,
				Term: &collaborationv1beta1.Filter_ResourceId{
					ResourceId: ref.ResourceId,
				},
			},
		},
	})
	if err != nil || lsRes.Status.Code != rpc.Code_CODE_OK {
		response.WriteOCSError(w, r, response.MetaServerError.StatusCode, "error listing space members", err)
		return
	}

	if len(lsRes.Shares) == 1 || !isSpaceManagerRemainingInShares(lsRes.Shares, &grantee) {
		response.WriteOCSError(w, r, http.StatusForbidden, "can't remove the last manager", nil)
		return
	}

	s := findShareByGrantee(lsRes.Shares, &grantee)
	if s == nil {
		response.WriteOCSError(w, r, response.MetaNotFound.StatusCode, "cannot find share", nil)
		return
	}

	removeShareRes, err := client.RemoveShare(ctx, &collaborationv1beta1.RemoveShareRequest{
		Ref: &collaborationv1beta1.ShareReference{
			Spec: &collaborationv1beta1.ShareReference_Id{
				Id: s.Id,
			},
		},
	})
	if err != nil {
		response.WriteOCSError(w, r, response.MetaServerError.StatusCode, "error removing space member", err)
		return
	}
	if removeShareRes.Status.Code != rpc.Code_CODE_OK {
		response.WriteOCSError(w, r, response.MetaServerError.StatusCode, "error removing space member", nil)
		return
	}

	response.WriteOCSSuccess(w, r, nil)
}

func isSpaceManagerRemainingInShares(shares []*collaborationv1beta1.Share, grantee *provider.Grantee) bool {
	for _, s := range shares {
		if s.GetPermissions().GetPermissions().GetRemoveGrant() && !isEqualGrantee(s.Grantee, grantee) {
			return true
		}
	}
	return false
}

func findShareByGrantee(shares []*collaborationv1beta1.Share, grantee *provider.Grantee) *collaborationv1beta1.Share {
	for _, s := range shares {
		if isEqualGrantee(s.Grantee, grantee) {
			return s
		}
	}
	return nil
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
		aID = a.GetGroupId().OpaqueId
		bID = b.GetGroupId().OpaqueId
	case provider.GranteeType_GRANTEE_TYPE_USER:
		aID = a.GetUserId().OpaqueId
		bID = b.GetUserId().OpaqueId
	}
	return aID == bID
}
