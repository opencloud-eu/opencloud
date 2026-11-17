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
	"net/http"

	grouppb "github.com/cs3org/go-cs3apis/cs3/identity/group/v1beta1"
	rpc "github.com/cs3org/go-cs3apis/cs3/rpc/v1beta1"
	collaboration "github.com/cs3org/go-cs3apis/cs3/sharing/collaboration/v1beta1"
	provider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	types "github.com/cs3org/go-cs3apis/cs3/types/v1beta1"
	"github.com/opencloud-eu/reva/v3/pkg/conversions"

	"github.com/opencloud-eu/reva/v3/internal/http/services/owncloud/ocs/response"
)

func (h *Handler) createGroupShare(w http.ResponseWriter, r *http.Request, statInfo *provider.ResourceInfo, role *conversions.Role, roleVal []byte) (*collaboration.Share, *ocsError) {
	ctx := r.Context()
	c, err := h.getClient()
	if err != nil {
		return nil, &ocsError{
			Code:    response.MetaServerError.StatusCode,
			Message: "error getting grpc gateway client",
			Error:   err,
		}
	}

	shareWith := r.FormValue("shareWith")
	if shareWith == "" {
		return nil, &ocsError{
			Code:    response.MetaBadRequest.StatusCode,
			Message: "missing shareWith",
		}
	}

	groupRes, err := c.GetGroupByClaim(ctx, &grouppb.GetGroupByClaimRequest{
		Claim:               "group_name",
		Value:               shareWith,
		SkipFetchingMembers: true,
	})
	if err != nil {
		return nil, &ocsError{
			Code:    response.MetaServerError.StatusCode,
			Message: "error searching recipient",
			Error:   err,
		}
	}
	if groupRes.Status.Code != rpc.Code_CODE_OK {
		return nil, &ocsError{
			Code:    response.MetaNotFound.StatusCode,
			Message: "group not found",
			Error:   err,
		}
	}

	createShareReq := &collaboration.CreateShareRequest{
		Opaque: &types.Opaque{
			Map: map[string]*types.OpaqueEntry{
				"role": {
					Decoder: "json",
					Value:   roleVal,
				},
			},
		},
		ResourceInfo: statInfo,
		Grant: &collaboration.ShareGrant{
			Grantee: &provider.Grantee{
				Type: provider.GranteeType_GRANTEE_TYPE_GROUP,
				Id:   &provider.Grantee_GroupId{GroupId: groupRes.Group.GetId()},
			},
			Permissions: &collaboration.SharePermissions{
				Permissions: role.CS3ResourcePermissions(),
			},
		},
	}

	share, ocsErr := h.createCs3Share(ctx, w, r, c, createShareReq)
	if ocsErr != nil {
		return nil, ocsErr
	}

	return share, nil
}
