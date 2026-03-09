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

package users

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	cs3gateway "github.com/cs3org/go-cs3apis/cs3/gateway/v1beta1"
	cs3identity "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"
	cs3rpc "github.com/cs3org/go-cs3apis/cs3/rpc/v1beta1"
	cs3storage "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	"github.com/go-chi/chi/v5"
	"github.com/opencloud-eu/reva/v2/internal/http/services/owncloud/ocs/config"
	"github.com/opencloud-eu/reva/v2/internal/http/services/owncloud/ocs/response"
	"github.com/opencloud-eu/reva/v2/pkg/appctx"
	"github.com/opencloud-eu/reva/v2/pkg/conversions"
	ctxpkg "github.com/opencloud-eu/reva/v2/pkg/ctx"
	"github.com/opencloud-eu/reva/v2/pkg/rgrpc/todo/pool"
	"github.com/opencloud-eu/reva/v2/pkg/utils"
)

// Handler renders user data for the user id given in the url path
type Handler struct {
	gatewayAddr string
}

// Init initializes this and any contained handlers
func (h *Handler) Init(c *config.Config) {
	h.gatewayAddr = c.GatewaySvc
}

// GetGroups handles GET requests on /cloud/users/groups
// TODO: implement
func (h *Handler) GetGroups(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	user := chi.URLParam(r, "userid")
	// FIXME use ldap to fetch user info
	u, ok := ctxpkg.ContextGetUser(ctx)
	if !ok {
		response.WriteOCSError(w, r, response.MetaServerError.StatusCode, "missing user in context", fmt.Errorf("missing user in context"))
		return
	}
	if user != u.Username {
		// FIXME allow fetching other users info? only for admins
		response.WriteOCSError(w, r, http.StatusForbidden, "user id mismatch", fmt.Errorf("%s tried to access %s user info endpoint", u.Id.OpaqueId, user))
		return
	}

	response.WriteOCSSuccess(w, r, &Groups{Groups: u.Groups})
}

// Quota holds quota information
type Quota struct {
	Free       int64   `json:"free,omitempty" xml:"free,omitempty"`
	Used       int64   `json:"used,omitempty" xml:"used,omitempty"`
	Total      int64   `json:"total,omitempty" xml:"total,omitempty"`
	Relative   float32 `json:"relative,omitempty" xml:"relative,omitempty"`
	Definition string  `json:"definition,omitempty" xml:"definition,omitempty"`
}

// User holds user data
type User struct {
	Enabled     string `json:"enabled" xml:"enabled"`
	Quota       *Quota `json:"quota,omitempty" xml:"quota,omitempty"`
	Email       string `json:"email" xml:"email"`
	DisplayName string `json:"displayname" xml:"displayname"` // is used in ocs/v(1|2).php/cloud/users/{username} - yes this is different from the /user endpoint
	UserType    string `json:"user-type" xml:"user-type"`
	UIDNumber   int64  `json:"uidnumber,omitempty" xml:"uidnumber,omitempty"`
	GIDNumber   int64  `json:"gidnumber,omitempty" xml:"gidnumber,omitempty"`
	// FIXME home should never be exposed ... even in oc 10, well only the admin can call this endpoint ...
	// Home                 string `json:"home" xml:"home"`
	TwoFactorAuthEnabled bool  `json:"two_factor_auth_enabled" xml:"two_factor_auth_enabled"`
	LastLogin            int64 `json:"last_login" xml:"last_login"`
}

// Groups holds group data
type Groups struct {
	Groups []string `json:"groups" xml:"groups>element"`
}

// GetUsers handles GET requests on /cloud/users
// Only allow self-read currently. TODO: List Users and Get on other users (both require
// administrative privileges)
func (h *Handler) GetUsers(w http.ResponseWriter, r *http.Request) {
	userid := chi.URLParam(r, "userid")
	userid, err := url.PathUnescape(userid)
	if err != nil {
		response.WriteOCSError(w, r, response.MetaServerError.StatusCode, "could not unescape username", err)
		return
	}

	currentUser, ok := ctxpkg.ContextGetUser(r.Context())
	if !ok {
		response.WriteOCSError(w, r, response.MetaServerError.StatusCode, "missing user in context", fmt.Errorf("missing user in context"))
		return
	}

	var user *cs3identity.User
	switch userid {
	case "":
		response.WriteOCSError(w, r, response.MetaBadRequest.StatusCode, "missing username", fmt.Errorf("missing username"))
		return
	case currentUser.Username:
		user = currentUser
	default:
		// FIXME allow fetching other users info? only for admins
		response.WriteOCSError(w, r, http.StatusForbidden, "user id mismatch", fmt.Errorf("%s tried to access %s user info endpoint", currentUser.Id.OpaqueId, user))
		return
	}

	d := &User{
		Enabled:     "true", // TODO include in response only when admin?
		DisplayName: user.DisplayName,
		Email:       user.Mail,
		UserType:    conversions.UserTypeString(user.Id.Type),
		Quota:       &Quota{},
	}
	// TODO how do we fill lastlogin of a user when another user (with the necessary permissions) looks up the user?
	// TODO someone needs to fill last-login
	if lastLogin := utils.ReadPlainFromOpaque(user.Opaque, "last-login"); lastLogin != "" {
		d.LastLogin, _ = strconv.ParseInt(lastLogin, 10, 64)
	}

	// lightweight and federated users don't have access to their storage space
	if currentUser.Id.Type != cs3identity.UserType_USER_TYPE_LIGHTWEIGHT && currentUser.Id.Type != cs3identity.UserType_USER_TYPE_FEDERATED {
		h.fillPersonalQuota(r.Context(), d, user)
	}

	response.WriteOCSSuccess(w, r, d)
}

func (h Handler) fillPersonalQuota(ctx context.Context, d *User, u *cs3identity.User) {

	sublog := appctx.GetLogger(ctx)

	gc, err := pool.GetGatewayServiceClient(h.gatewayAddr)
	if err != nil {
		sublog.Error().Err(err).Msg("error getting gateway client")
		return
	}

	res, err := gc.ListStorageSpaces(ctx, &cs3storage.ListStorageSpacesRequest{
		Filters: []*cs3storage.ListStorageSpacesRequest_Filter{
			{
				Type: cs3storage.ListStorageSpacesRequest_Filter_TYPE_OWNER,
				Term: &cs3storage.ListStorageSpacesRequest_Filter_Owner{
					Owner: u.Id,
				},
			},
			{
				Type: cs3storage.ListStorageSpacesRequest_Filter_TYPE_SPACE_TYPE,
				Term: &cs3storage.ListStorageSpacesRequest_Filter_SpaceType{
					SpaceType: "personal",
				},
			},
		},
	})
	if err != nil {
		sublog.Error().Err(err).Msg("error calling ListStorageSpaces")
		return
	}
	if res.Status.Code != cs3rpc.Code_CODE_OK {
		return
	}

	if len(res.StorageSpaces) == 0 {
		sublog.Error().Err(err).Msg("list spaces returned empty list")
		return

	}

	getQuotaRes, err := gc.GetQuota(ctx, &cs3gateway.GetQuotaRequest{Ref: &cs3storage.Reference{
		ResourceId: res.StorageSpaces[0].Root,
		Path:       ".",
	}})
	if err != nil {
		sublog.Error().Err(err).Msg("error calling GetQuota")
		return
	}
	if res.Status.Code != cs3rpc.Code_CODE_OK {
		sublog.Debug().Interface("status", res.Status).Msg("GetQuota returned non OK result")
		return
	}

	total := getQuotaRes.TotalBytes
	used := getQuotaRes.UsedBytes

	d.Quota = &Quota{
		Used: int64(used),
		// TODO support negative values or flags for the quota to carry special meaning: -1 = uncalculated, -2 = unknown, -3 = unlimited
		// for now we can only report total and used
		Total: int64(total),
		// we cannot differentiate between `default` or a human readable `1 GB` definition.
		// The web UI can create a human readable string from the actual total if it is set. Otherwise it has to leave out relative and total anyway.
		// Definition: "default",
	}

	if raw := utils.ReadPlainFromOpaque(getQuotaRes.Opaque, "remaining"); raw != "" {
		d.Quota.Free, _ = strconv.ParseInt(raw, 10, 64)
	}

	// only calculate free and relative when total is available
	if total > 0 {
		d.Quota.Free = int64(total - used)
		d.Quota.Relative = float32((float64(used) / float64(total)) * 100.0)
	} else {
		d.Quota.Definition = "none" // this indicates no quota / unlimited to the ui
	}
}
