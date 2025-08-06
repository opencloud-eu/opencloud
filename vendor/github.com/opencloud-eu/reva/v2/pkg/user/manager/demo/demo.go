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

package demo

import (
	"context"
	"errors"
	"strconv"
	"strings"

	userpb "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"
	"github.com/opencloud-eu/reva/v2/pkg/errtypes"
	"github.com/opencloud-eu/reva/v2/pkg/user"
	"github.com/opencloud-eu/reva/v2/pkg/user/manager/registry"
	"google.golang.org/protobuf/proto"
)

func init() {
	registry.Register("demo", New)
}

type manager struct {
	catalog map[string]*userpb.User
}

// New returns a new user manager.
func New(m map[string]interface{}) (user.Manager, error) {
	mgr := &manager{}
	err := mgr.Configure(m)
	if err != nil {
		return nil, err
	}
	return mgr, err
}

func (m *manager) Configure(ml map[string]interface{}) error {
	cat := getUsers()
	m.catalog = cat
	return nil
}

func (m *manager) GetUser(ctx context.Context, uid *userpb.UserId, skipFetchingGroups bool) (*userpb.User, error) {
	if user, ok := m.catalog[uid.OpaqueId]; ok {
		if uid.Idp == "" || user.Id.Idp == uid.Idp {
			u := proto.Clone(user).(*userpb.User)
			if skipFetchingGroups {
				u.Groups = nil
			}
			return u, nil
		}
	}
	return nil, errtypes.NotFound(uid.OpaqueId)
}

func (m *manager) GetUserByClaim(ctx context.Context, claim, value string, skipFetchingGroups bool) (*userpb.User, error) {
	for _, u := range m.catalog {
		if userClaim, err := extractClaim(u, claim); err == nil && value == userClaim {
			user := proto.Clone(u).(*userpb.User)
			if skipFetchingGroups {
				user.Groups = nil
			}
			return user, nil
		}
	}
	return nil, errtypes.NotFound(value)
}

func extractClaim(u *userpb.User, claim string) (string, error) {
	switch claim {
	case "mail":
		return u.Mail, nil
	case "username":
		return u.Username, nil
	case "userid":
		return u.Id.OpaqueId, nil
	case "uid":
		if u.UidNumber != 0 {
			return strconv.FormatInt(u.UidNumber, 10), nil
		}
	}
	return "", errors.New("demo: invalid field")
}

// TODO(jfd) compare sub?
func userContains(u *userpb.User, query, tenantID string) bool {
	if tenantID != "" && u.Id.TenantId != tenantID {
		return false
	}
	return strings.Contains(u.Username, query) ||
		strings.Contains(u.DisplayName, query) ||
		strings.Contains(u.Mail, query) ||
		strings.Contains(u.Id.OpaqueId, query)
}

func (m *manager) FindUsers(ctx context.Context, query, tenantID string, skipFetchingGroups bool) ([]*userpb.User, error) {
	users := []*userpb.User{}
	for _, u := range m.catalog {
		if userContains(u, query, tenantID) {
			user := proto.Clone(u).(*userpb.User)
			if skipFetchingGroups {
				user.Groups = nil
			}
			users = append(users, user)
		}
	}
	return users, nil
}

func (m *manager) GetUserGroups(ctx context.Context, uid *userpb.UserId) ([]string, error) {
	user, err := m.GetUser(ctx, uid, false)
	if err != nil {
		return nil, err
	}
	return user.Groups, nil
}

func getUsers() map[string]*userpb.User {
	return map[string]*userpb.User{
		"4c510ada-c86b-4815-8820-42cdf82c3d51": {
			Id: &userpb.UserId{
				Idp:      "http://localhost:9998",
				OpaqueId: "4c510ada-c86b-4815-8820-42cdf82c3d51",
				Type:     userpb.UserType_USER_TYPE_PRIMARY,
				TenantId: "c239389d-c249-499d-ae80-07558429769a",
			},
			Username:    "einstein",
			Groups:      []string{"sailing-lovers", "violin-haters", "physics-lovers"},
			Mail:        "einstein@example.org",
			DisplayName: "Albert Einstein",
			UidNumber:   123,
			GidNumber:   987,
		},
		"f7fbf8c8-139b-4376-b307-cf0a8c2d0d9c": {
			Id: &userpb.UserId{
				Idp:      "http://localhost:9998",
				OpaqueId: "f7fbf8c8-139b-4376-b307-cf0a8c2d0d9c",
				Type:     userpb.UserType_USER_TYPE_PRIMARY,
				TenantId: "c239389d-c249-499d-ae80-07558429769a",
			},
			Username:    "marie",
			Groups:      []string{"radium-lovers", "polonium-lovers", "physics-lovers"},
			Mail:        "marie@example.org",
			DisplayName: "Marie Curie",
			UidNumber:   456,
			GidNumber:   987,
		},
		"932b4540-8d16-481e-8ef4-588e4b6b151c": {
			Id: &userpb.UserId{
				Idp:      "http://localhost:9998",
				OpaqueId: "932b4540-8d16-481e-8ef4-588e4b6b151c",
				Type:     userpb.UserType_USER_TYPE_PRIMARY,
				TenantId: "c239389d-c249-499d-ae80-07558429769a",
			},
			Username:    "richard",
			Groups:      []string{"quantum-lovers", "philosophy-haters", "physics-lovers"},
			Mail:        "richard@example.org",
			DisplayName: "Richard Feynman",
		},
		"bf2ee2f2-67bc-418f-ac4e-2f6427f9fb2f": {
			Id: &userpb.UserId{
				Idp:      "http://localhost:9998",
				OpaqueId: "bf2ee2f2-67bc-418f-ac4e-2f6427f9fb2f",
				Type:     userpb.UserType_USER_TYPE_PRIMARY,
				TenantId: "d375ba5e-1140-472a-b199-ca7d671a9fe1",
			},
			Username:    "jen",
			Groups:      []string{},
			Mail:        "jen@example.org",
			DisplayName: "Jen Barber",
		},
		"79a91ae3-13da-4bab-9f91-63c60295c7cf": {
			Id: &userpb.UserId{
				Idp:      "http://localhost:9998",
				OpaqueId: "79a91ae3-13da-4bab-9f91-63c60295c7cf",
				Type:     userpb.UserType_USER_TYPE_PRIMARY,
				TenantId: "d375ba5e-1140-472a-b199-ca7d671a9fe1",
			},
			Username:    "ralph",
			Groups:      []string{},
			Mail:        "ralphi@example.org",
			DisplayName: "Ralph Wiggum",
		},
	}
}
