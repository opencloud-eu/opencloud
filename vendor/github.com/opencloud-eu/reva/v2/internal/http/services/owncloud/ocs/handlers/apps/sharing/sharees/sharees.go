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

package sharees

import (
	"net/http"
	"strings"

	grouppb "github.com/cs3org/go-cs3apis/cs3/identity/group/v1beta1"
	userpb "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"
	invitepb "github.com/cs3org/go-cs3apis/cs3/ocm/invite/v1beta1"
	rpc "github.com/cs3org/go-cs3apis/cs3/rpc/v1beta1"
	"github.com/opencloud-eu/reva/v2/pkg/conversions"

	"github.com/opencloud-eu/reva/v2/internal/http/services/owncloud/ocs/config"
	"github.com/opencloud-eu/reva/v2/internal/http/services/owncloud/ocs/response"
	"github.com/opencloud-eu/reva/v2/pkg/appctx"
	"github.com/opencloud-eu/reva/v2/pkg/rgrpc/todo/pool"
	"github.com/opencloud-eu/reva/v2/pkg/storage/utils/templates"
)

// Handler implements the open collaboration service sharing API
type Handler struct {
	gatewayAddr             string
	additionalInfoAttribute string
	includeOCMSharees       bool
	showUserEmailInResults  bool
}

// Init initializes this and any contained handlers
func (h *Handler) Init(c *config.Config) {
	h.gatewayAddr = c.GatewaySvc
	h.additionalInfoAttribute = c.AdditionalInfoAttribute
	h.includeOCMSharees = c.IncludeOCMSharees
	h.showUserEmailInResults = c.ShowEmailInResults
}

// FindSharees implements the /apps/files_sharing/api/v1/sharees endpoint
func (h *Handler) FindSharees(w http.ResponseWriter, r *http.Request) {
	log := appctx.GetLogger(r.Context())
	term := r.URL.Query().Get("search")

	if term == "" {
		response.WriteOCSError(w, r, response.MetaBadRequest.StatusCode, "search must not be empty", nil)
		return
	}

	gwc, err := pool.GetGatewayServiceClient(h.gatewayAddr)
	if err != nil {
		response.WriteOCSError(w, r, response.MetaServerError.StatusCode, "error getting gateway grpc client", err)
		return
	}
	usersRes, err := gwc.FindUsers(r.Context(), &userpb.FindUsersRequest{Query: term, SkipFetchingUserGroups: true})
	if err != nil {
		response.WriteOCSError(w, r, response.MetaServerError.StatusCode, "error searching users", err)
		return
	}
	log.Debug().Int("count", len(usersRes.GetUsers())).Str("search", term).Msg("users found")

	userMatches := make([]*conversions.MatchData, 0, len(usersRes.GetUsers()))
	exactUserMatches := make([]*conversions.MatchData, 0)
	for _, user := range usersRes.GetUsers() {
		match := h.userAsMatch(user)
		log.Debug().Interface("user", user).Interface("match", match).Msg("mapped")
		if h.isExactMatch(match, term) {
			exactUserMatches = append(exactUserMatches, match)
		} else {
			userMatches = append(userMatches, match)
		}
	}

	if h.includeOCMSharees {
		remoteUsersRes, err := gwc.FindAcceptedUsers(r.Context(), &invitepb.FindAcceptedUsersRequest{Filter: term})
		if err != nil {
			response.WriteOCSError(w, r, response.MetaServerError.StatusCode, "error searching remote users", err)
			return
		}
		if remoteUsersRes.Status.Code != rpc.Code_CODE_OK {
			response.WriteOCSError(w, r, response.MetaServerError.StatusCode, "error searching remote users", nil)
			return
		}
		for _, user := range remoteUsersRes.GetAcceptedUsers() {
			match := h.userAsMatch(user)
			log.Debug().Interface("user", user).Interface("match", match).Msg("mapped")
			if h.isExactMatch(match, term) {
				exactUserMatches = append(exactUserMatches, match)
			} else {
				userMatches = append(userMatches, match)
			}
		}
	}

	groupsRes, err := gwc.FindGroups(r.Context(), &grouppb.FindGroupsRequest{Filter: term, SkipFetchingMembers: true})
	if err != nil {
		response.WriteOCSError(w, r, response.MetaServerError.StatusCode, "error searching groups", err)
		return
	}
	log.Debug().Int("count", len(groupsRes.GetGroups())).Str("search", term).Msg("groups found")

	groupMatches := make([]*conversions.MatchData, 0, len(groupsRes.GetGroups()))
	exactGroupMatches := make([]*conversions.MatchData, 0)
	for _, g := range groupsRes.GetGroups() {
		match := h.groupAsMatch(g)
		log.Debug().Interface("group", g).Interface("match", match).Msg("mapped")
		if h.isExactMatch(match, term) {
			exactGroupMatches = append(exactGroupMatches, match)
		} else {
			groupMatches = append(groupMatches, match)
		}
	}

	if !h.showUserEmailInResults {
		for _, m := range userMatches {
			m.Value.ShareWithAdditionalInfo = m.Value.ShareWith
		}
		for _, m := range exactUserMatches {
			m.Value.ShareWithAdditionalInfo = m.Value.ShareWith
		}
		for _, m := range groupMatches {
			m.Value.ShareWithAdditionalInfo = m.Value.ShareWith
		}
		for _, m := range exactGroupMatches {
			m.Value.ShareWithAdditionalInfo = m.Value.ShareWith
		}
	}

	response.WriteOCSSuccess(w, r, &conversions.ShareeData{
		Exact: &conversions.ExactMatchesData{
			Users:   exactUserMatches,
			Groups:  exactGroupMatches,
			Remotes: []*conversions.MatchData{},
		},
		Users:   userMatches,
		Groups:  groupMatches,
		Remotes: []*conversions.MatchData{},
	})
}

func (h *Handler) userAsMatch(u *userpb.User) *conversions.MatchData {
	data := &conversions.MatchValueData{
		ShareType: int(conversions.ShareTypeUser),
		// api compatibility with oc10: mark guest users in share invite dialogue
		UserType: 0,
		// api compatibility with oc10: always use the username
		ShareWith:               u.Username,
		ShareWithAdditionalInfo: h.getAdditionalInfoAttribute(u),
	}

	switch u.Id.Type {
	case userpb.UserType_USER_TYPE_GUEST, userpb.UserType_USER_TYPE_LIGHTWEIGHT:
		data.UserType = 1
	case userpb.UserType_USER_TYPE_FEDERATED:
		data.ShareType = int(conversions.ShareTypeFederatedCloudShare)
		data.ShareWith = u.Id.OpaqueId
		data.ShareWithProvider = u.Id.Idp
	}

	return &conversions.MatchData{
		Label: u.DisplayName,
		Value: data,
	}
}

func (h *Handler) groupAsMatch(g *grouppb.Group) *conversions.MatchData {
	return &conversions.MatchData{
		Label: g.DisplayName,
		Value: &conversions.MatchValueData{
			ShareType:               int(conversions.ShareTypeGroup),
			ShareWith:               g.GroupName,
			ShareWithAdditionalInfo: g.Mail,
		},
	}
}

func (h *Handler) getAdditionalInfoAttribute(u *userpb.User) string {
	return templates.WithUser(u, h.additionalInfoAttribute)
}

func (h *Handler) isExactMatch(match *conversions.MatchData, term string) bool {
	if match == nil || match.Value == nil {
		return false
	}
	return strings.EqualFold(match.Value.ShareWith, term) || strings.EqualFold(match.Value.ShareWithAdditionalInfo, term) ||
		strings.EqualFold(match.Label, term)
}
