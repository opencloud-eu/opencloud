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

	authpb "github.com/cs3org/go-cs3apis/cs3/auth/provider/v1beta1"
	user "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"
	"github.com/opencloud-eu/reva/v2/pkg/auth"
	"github.com/opencloud-eu/reva/v2/pkg/auth/manager/registry"
	"github.com/opencloud-eu/reva/v2/pkg/auth/scope"
	"github.com/opencloud-eu/reva/v2/pkg/errtypes"
	"github.com/rs/zerolog"
)

func init() {
	registry.Register("demo", New)
}

type manager struct {
	credentials map[string]Credentials
}

// Credentials holds a pair of secret and userid
type Credentials struct {
	User   *user.User
	Secret string
}

// New returns a new auth Manager.
func New(m map[string]any, _ *zerolog.Logger) (auth.Manager, error) {
	// m not used
	mgr := &manager{}
	err := mgr.Configure(m)
	return mgr, err
}

func (m *manager) Configure(ml map[string]interface{}) error {
	creds := getCredentials()
	m.credentials = creds
	return nil
}

func (m *manager) Authenticate(ctx context.Context, clientID, clientSecret string) (*user.User, map[string]*authpb.Scope, error) {
	if c, ok := m.credentials[clientID]; ok {
		if c.Secret == clientSecret {
			var scopes map[string]*authpb.Scope
			var err error
			if c.User.Id != nil && (c.User.Id.Type == user.UserType_USER_TYPE_LIGHTWEIGHT || c.User.Id.Type == user.UserType_USER_TYPE_FEDERATED) {
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
			return c.User, scopes, nil
		}
	}
	return nil, nil, errtypes.InvalidCredentials(clientID)
}

func getCredentials() map[string]Credentials {
	return map[string]Credentials{
		"einstein": {
			Secret: "relativity",
			User: &user.User{
				Id: &user.UserId{
					Idp:      "http://localhost:9998",
					OpaqueId: "4c510ada-c86b-4815-8820-42cdf82c3d51",
					Type:     user.UserType_USER_TYPE_PRIMARY,
				},
				Username:    "einstein",
				Groups:      []string{"sailing-lovers", "violin-haters", "physics-lovers"},
				Mail:        "einstein@example.org",
				DisplayName: "Albert Einstein",
			},
		},
		"marie": {
			Secret: "radioactivity",
			User: &user.User{
				Id: &user.UserId{
					Idp:      "http://localhost:9998",
					OpaqueId: "f7fbf8c8-139b-4376-b307-cf0a8c2d0d9c",
					Type:     user.UserType_USER_TYPE_PRIMARY,
				},
				Username:    "marie",
				Groups:      []string{"radium-lovers", "polonium-lovers", "physics-lovers"},
				Mail:        "marie@example.org",
				DisplayName: "Marie Curie",
			},
		},
		"richard": {
			Secret: "superfluidity",
			User: &user.User{
				Id: &user.UserId{
					Idp:      "http://localhost:9998",
					OpaqueId: "932b4540-8d16-481e-8ef4-588e4b6b151c",
					Type:     user.UserType_USER_TYPE_PRIMARY,
				},
				Username:    "richard",
				Groups:      []string{"quantum-lovers", "philosophy-haters", "physics-lovers"},
				Mail:        "richard@example.org",
				DisplayName: "Richard Feynman",
			},
		},
	}
}
