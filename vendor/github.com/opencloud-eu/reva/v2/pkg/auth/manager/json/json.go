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

package json

import (
	"context"
	"encoding/json"
	"os"

	authpb "github.com/cs3org/go-cs3apis/cs3/auth/provider/v1beta1"
	user "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"
	typespb "github.com/cs3org/go-cs3apis/cs3/types/v1beta1"
	"github.com/mitchellh/mapstructure"
	"github.com/opencloud-eu/reva/v2/pkg/auth"
	"github.com/opencloud-eu/reva/v2/pkg/auth/manager/registry"
	"github.com/opencloud-eu/reva/v2/pkg/auth/scope"
	"github.com/opencloud-eu/reva/v2/pkg/errtypes"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
)

func init() {
	registry.Register("json", New)
}

// Credentials holds a pair of secret and userid
type Credentials struct {
	ID           *user.UserId    `mapstructure:"id" json:"id"`
	Username     string          `mapstructure:"username" json:"username"`
	Mail         string          `mapstructure:"mail" json:"mail"`
	MailVerified bool            `mapstructure:"mail_verified" json:"mail_verified"`
	DisplayName  string          `mapstructure:"display_name" json:"display_name"`
	Secret       string          `mapstructure:"secret" json:"secret"`
	Groups       []string        `mapstructure:"groups" json:"groups"`
	UIDNumber    int64           `mapstructure:"uid_number" json:"uid_number"`
	GIDNumber    int64           `mapstructure:"gid_number" json:"gid_number"`
	Opaque       *typespb.Opaque `mapstructure:"opaque" json:"opaque"`
}

type manager struct {
	credentials map[string]*Credentials
}

type config struct {
	// Users holds a path to a file containing json conforming the Users struct
	Users string `mapstructure:"users"`
}

func (c *config) init() {
	if c.Users == "" {
		c.Users = "/etc/revad/users.json"
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

// New returns a new auth Manager.
func New(m map[string]any, _ *zerolog.Logger) (auth.Manager, error) {
	mgr := &manager{}
	err := mgr.Configure(m)
	if err != nil {
		return nil, err
	}
	return mgr, nil
}

func (m *manager) Configure(ml map[string]interface{}) error {
	c, err := parseConfig(ml)
	if err != nil {
		return err
	}

	m.credentials = map[string]*Credentials{}
	f, err := os.ReadFile(c.Users)
	if err != nil {
		return err
	}

	credentials := []*Credentials{}

	err = json.Unmarshal(f, &credentials)
	if err != nil {
		return err
	}

	for _, c := range credentials {
		m.credentials[c.Username] = c
	}
	return nil
}

func (m *manager) Authenticate(ctx context.Context, username string, secret string) (*user.User, map[string]*authpb.Scope, error) {
	if c, ok := m.credentials[username]; ok {
		if c.Secret == secret {
			var scopes map[string]*authpb.Scope
			var err error
			if c.ID != nil && (c.ID.Type == user.UserType_USER_TYPE_LIGHTWEIGHT || c.ID.Type == user.UserType_USER_TYPE_FEDERATED) {
				scopes, err = scope.AddLightweightAccountScope(authpb.Role_ROLE_OWNER, nil)
				if err != nil {
					return nil, nil, err
				}
			} else {
				scopes, err = scope.AddOwnerScope(nil)
				if err != nil {
					return nil, nil, err
				}
			}
			return &user.User{
				Id:           c.ID,
				Username:     c.Username,
				Mail:         c.Mail,
				MailVerified: c.MailVerified,
				DisplayName:  c.DisplayName,
				Groups:       c.Groups,
				UidNumber:    c.UIDNumber,
				GidNumber:    c.GIDNumber,
				Opaque:       c.Opaque,
				// TODO add arbitrary keys as opaque data
			}, scopes, nil
		}
	}
	return nil, nil, errtypes.InvalidCredentials(username)
}
