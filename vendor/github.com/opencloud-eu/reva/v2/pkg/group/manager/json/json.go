// Copyright 2018-2020 CERN
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

package json

import (
	"context"
	"encoding/json"
	"os"
	"strconv"
	"strings"

	grouppb "github.com/cs3org/go-cs3apis/cs3/identity/group/v1beta1"
	userpb "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"
	"github.com/mitchellh/mapstructure"
	"github.com/opencloud-eu/reva/v2/pkg/errtypes"
	"github.com/opencloud-eu/reva/v2/pkg/group"
	"github.com/opencloud-eu/reva/v2/pkg/group/manager/registry"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"google.golang.org/protobuf/proto"
)

func init() {
	registry.Register("json", New)
}

type manager struct {
	groups []*grouppb.Group
}

type config struct {
	// Groups holds a path to a file containing json conforming to the Groups struct
	Groups string `mapstructure:"groups"`
}

func (c *config) init() {
	if c.Groups == "" {
		c.Groups = "/etc/revad/groups.json"
	}
}

func parseConfig(m map[string]interface{}) (*config, error) {
	c := &config{}
	if err := mapstructure.Decode(m, c); err != nil {
		err = errors.Wrap(err, "error decoding conf")
		return nil, err
	}
	c.init()
	return c, nil
}

// New returns a group manager implementation that reads a json file to provide group metadata.
func New(m map[string]any, _ *zerolog.Logger) (group.Manager, error) {
	c, err := parseConfig(m)
	if err != nil {
		return nil, err
	}

	f, err := os.ReadFile(c.Groups)
	if err != nil {
		return nil, err
	}

	groups := []*grouppb.Group{}

	err = json.Unmarshal(f, &groups)
	if err != nil {
		return nil, err
	}

	return &manager{
		groups: groups,
	}, nil
}

func (m *manager) GetGroup(ctx context.Context, gid *grouppb.GroupId, skipFetchingMembers bool) (*grouppb.Group, error) {
	for _, g := range m.groups {
		if (g.Id.GetOpaqueId() == gid.OpaqueId || g.GroupName == gid.OpaqueId) && (gid.Idp == "" || gid.Idp == g.Id.GetIdp()) {
			group := proto.Clone(g).(*grouppb.Group)
			if skipFetchingMembers {
				group.Members = nil
			}
			return group, nil
		}
	}
	return nil, errtypes.NotFound(gid.OpaqueId)
}

func (m *manager) GetGroupByClaim(ctx context.Context, claim, value string, skipFetchingMembers bool) (*grouppb.Group, error) {
	for _, g := range m.groups {
		if groupClaim, err := extractClaim(g, claim); err == nil && value == groupClaim {
			group := proto.Clone(g).(*grouppb.Group)
			if skipFetchingMembers {
				group.Members = nil
			}
			return group, nil
		}
	}
	return nil, errtypes.NotFound(value)
}

func extractClaim(g *grouppb.Group, claim string) (string, error) {
	switch claim {
	case "group_name":
		return g.GroupName, nil
	case "gid_number":
		return strconv.FormatInt(g.GidNumber, 10), nil
	case "display_name":
		return g.DisplayName, nil
	case "mail":
		return g.Mail, nil
	}
	return "", errors.New("json: invalid field")
}

func (m *manager) FindGroups(ctx context.Context, query string, skipFetchingMembers bool) ([]*grouppb.Group, error) {
	groups := []*grouppb.Group{}
	for _, g := range m.groups {
		if groupContains(g, query) {
			group := proto.Clone(g).(*grouppb.Group)
			if skipFetchingMembers {
				group.Members = nil
			}
			groups = append(groups, group)
		}
	}
	return groups, nil
}

func groupContains(g *grouppb.Group, query string) bool {
	query = strings.ToLower(query)
	return strings.Contains(strings.ToLower(g.GroupName), query) || strings.Contains(strings.ToLower(g.DisplayName), query) ||
		strings.Contains(strings.ToLower(g.Mail), query) || strings.Contains(strings.ToLower(g.Id.OpaqueId), query)
}

func (m *manager) GetMembers(ctx context.Context, gid *grouppb.GroupId) ([]*userpb.UserId, error) {
	for _, g := range m.groups {
		if g.Id.GetOpaqueId() == gid.OpaqueId || g.GroupName == gid.OpaqueId {
			return g.Members, nil
		}
	}
	return nil, errtypes.NotFound(gid.OpaqueId)
}

func (m *manager) HasMember(ctx context.Context, gid *grouppb.GroupId, uid *userpb.UserId) (bool, error) {
	members, err := m.GetMembers(ctx, gid)
	if err != nil {
		return false, err
	}

	for _, u := range members {
		if u.OpaqueId == uid.OpaqueId && u.Idp == uid.Idp {
			return true, nil
		}
	}
	return false, nil
}
