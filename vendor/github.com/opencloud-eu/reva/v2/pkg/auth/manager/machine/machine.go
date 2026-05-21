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

package machine

import (
	"context"
	"strings"

	authpb "github.com/cs3org/go-cs3apis/cs3/auth/provider/v1beta1"
	userpb "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"
	rpc "github.com/cs3org/go-cs3apis/cs3/rpc/v1beta1"
	"github.com/mitchellh/mapstructure"
	"github.com/opencloud-eu/reva/v2/pkg/auth"
	"github.com/opencloud-eu/reva/v2/pkg/auth/manager/registry"
	"github.com/opencloud-eu/reva/v2/pkg/auth/scope"
	"github.com/opencloud-eu/reva/v2/pkg/errtypes"
	"github.com/opencloud-eu/reva/v2/pkg/rgrpc/todo/pool"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
)

// 'machine' is an authentication method used to impersonate users.
// To impersonate the given user it's only needed an api-key, saved
// in a config file.

// supported claims
var claims = []string{"mail", "uid", "username", "gid", "userid"}

type manager struct {
	APIKey      string `mapstructure:"api_key"`
	GatewayAddr string `mapstructure:"gateway_addr"`
}

func init() {
	registry.Register("machine", New)
}

// Configure parses the map conf
func (m *manager) Configure(conf map[string]interface{}) error {
	err := mapstructure.Decode(conf, m)
	if err != nil {
		return errors.Wrap(err, "error decoding conf")
	}
	return nil
}

// New creates a new manager for the 'machine' authentication
func New(conf map[string]any, _ *zerolog.Logger) (auth.Manager, error) {
	m := &manager{}
	err := m.Configure(conf)
	if err != nil {
		return nil, err
	}
	return m, nil
}

// Authenticate impersonate an user if the provided secret is equal to the api-key
func (m *manager) Authenticate(ctx context.Context, user, secret string) (*userpb.User, map[string]*authpb.Scope, error) {
	if m.APIKey != secret {
		return nil, nil, errtypes.InvalidCredentials("")
	}

	gtw, err := pool.GetGatewayServiceClient(m.GatewayAddr)
	if err != nil {
		return nil, nil, err
	}

	// username could be either a normal username or a string <claim>:<value>
	// in the first case the claim is "username"
	claim, value := parseUser(user)

	userResponse, err := gtw.GetUserByClaim(ctx, &userpb.GetUserByClaimRequest{
		Claim: claim,
		Value: value,
	})

	switch {
	case err != nil:
		return nil, nil, err
	case userResponse.Status.Code == rpc.Code_CODE_NOT_FOUND:
		return nil, nil, errtypes.NotFound(userResponse.Status.Message)
	case userResponse.Status.Code != rpc.Code_CODE_OK:
		return nil, nil, errtypes.InternalError(userResponse.Status.Message)
	}

	scope, err := scope.AddOwnerScope(nil)
	if err != nil {
		return nil, nil, err
	}

	return userResponse.GetUser(), scope, nil

}

func contains(lst []string, s string) bool {
	for _, e := range lst {
		if e == s {
			return true
		}
	}
	return false
}

func parseUser(user string) (string, string) {
	s := strings.SplitN(user, ":", 2)
	if len(s) == 2 && contains(claims, s[0]) {
		return s[0], s[1]
	}
	return "username", user
}
