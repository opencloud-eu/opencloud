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

package memory

import (
	"context"
	"strconv"
	"strings"

	userpb "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"
	typespb "github.com/cs3org/go-cs3apis/cs3/types/v1beta1"
	"github.com/mitchellh/mapstructure"
	"github.com/opencloud-eu/reva/v2/pkg/errtypes"
	"github.com/opencloud-eu/reva/v2/pkg/user"
	"github.com/opencloud-eu/reva/v2/pkg/user/manager/registry"
	"github.com/pkg/errors"
)

func init() {
	registry.Register("memory", New)
}

type config struct {
	// Users holds a map with userid and user
	Users map[string]*User `mapstructure:"users"`
}

// User holds a user but uses _ in mapstructure names
type User struct {
	ID           *userpb.UserId  `mapstructure:"id" json:"id"`
	Username     string          `mapstructure:"username" json:"username"`
	Mail         string          `mapstructure:"mail" json:"mail"`
	MailVerified bool            `mapstructure:"mail_verified" json:"mail_verified"`
	DisplayName  string          `mapstructure:"display_name" json:"display_name"`
	Groups       []string        `mapstructure:"groups" json:"groups"`
	UIDNumber    int64           `mapstructure:"uid_number" json:"uid_number"`
	GIDNumber    int64           `mapstructure:"gid_number" json:"gid_number"`
	Opaque       *typespb.Opaque `mapstructure:"opaque" json:"opaque"`
}

func parseConfig(m map[string]interface{}) (*config, error) {
	c := &config{}
	if err := mapstructure.Decode(m, c); err != nil {
		err = errors.Wrap(err, "error decoding conf")
		return nil, err
	}
	return c, nil
}

type manager struct {
	catalog map[string]*User
}

// New returns a new user manager.
func New(m map[string]interface{}) (user.Manager, error) {
	mgr := &manager{}
	err := mgr.Configure(m)
	return mgr, err
}

func (m *manager) Configure(ml map[string]interface{}) error {
	c, err := parseConfig(ml)
	if err != nil {
		return err
	}
	m.catalog = c.Users
	return nil
}

func (m *manager) GetUser(ctx context.Context, uid *userpb.UserId, skipFetchingGroups bool) (*userpb.User, error) {
	if uid.GetTenantId() != "" {
		return nil, errtypes.NotSupported("tenant filter not supported in memory user manager")
	}
	if user, ok := m.catalog[uid.OpaqueId]; ok {
		if uid.Idp == "" || user.ID.Idp == uid.Idp {
			u := *user
			if skipFetchingGroups {
				u.Groups = nil
			}
			return &userpb.User{
				Id:           u.ID,
				Username:     u.Username,
				Mail:         u.Mail,
				DisplayName:  u.DisplayName,
				MailVerified: u.MailVerified,
				Groups:       u.Groups,
				Opaque:       u.Opaque,
				UidNumber:    u.UIDNumber,
				GidNumber:    u.GIDNumber,
			}, nil
		}
	}
	return nil, errtypes.NotFound(uid.OpaqueId)
}

func (m *manager) GetUserByClaim(ctx context.Context, claim, value, tenantID string, skipFetchingGroups bool) (*userpb.User, error) {
	if tenantID != "" {
		return nil, errtypes.NotSupported("tenant filter not supported in memory user manager")
	}

	for _, u := range m.catalog {
		if userClaim, err := extractClaim(u, claim); err == nil && value == userClaim {
			user := &userpb.User{
				Id:           u.ID,
				Username:     u.Username,
				Mail:         u.Mail,
				DisplayName:  u.DisplayName,
				MailVerified: u.MailVerified,
				Groups:       u.Groups,
				Opaque:       u.Opaque,
				UidNumber:    u.UIDNumber,
				GidNumber:    u.GIDNumber,
			}
			if skipFetchingGroups {
				user.Groups = nil
			}
			return user, nil
		}
	}
	return nil, errtypes.NotFound(value)
}

func extractClaim(u *User, claim string) (string, error) {
	switch claim {
	case "mail":
		return u.Mail, nil
	case "username":
		return u.Username, nil
	case "userid":
		return u.ID.OpaqueId, nil
	case "uid":
		if u.UIDNumber != 0 {
			return strconv.FormatInt(u.UIDNumber, 10), nil
		}
	}
	return "", errors.New("memory: invalid field")
}

// TODO(jfd) compare sub?
func userContains(u *User, query string) bool {
	return strings.Contains(u.Username, query) || strings.Contains(u.DisplayName, query) || strings.Contains(u.Mail, query) || strings.Contains(u.ID.OpaqueId, query)
}

func (m *manager) FindUsers(ctx context.Context, query, tenantID string, skipFetchingGroups bool) ([]*userpb.User, error) {
	if tenantID != "" {
		return nil, errtypes.NotSupported("tenant filter not supported in memory user manager")
	}

	users := []*userpb.User{}
	for _, u := range m.catalog {
		if userContains(u, query) {
			user := &userpb.User{
				Id:           u.ID,
				Username:     u.Username,
				Mail:         u.Mail,
				DisplayName:  u.DisplayName,
				MailVerified: u.MailVerified,
				Groups:       u.Groups,
				Opaque:       u.Opaque,
				UidNumber:    u.UIDNumber,
				GidNumber:    u.GIDNumber,
			}
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
