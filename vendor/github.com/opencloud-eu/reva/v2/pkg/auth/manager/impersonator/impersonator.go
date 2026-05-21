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

package impersonator

import (
	"context"
	"strings"

	authpb "github.com/cs3org/go-cs3apis/cs3/auth/provider/v1beta1"
	user "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"
	"github.com/opencloud-eu/reva/v2/pkg/auth"
	"github.com/opencloud-eu/reva/v2/pkg/auth/manager/registry"
	"github.com/opencloud-eu/reva/v2/pkg/auth/scope"
	"github.com/rs/zerolog"
)

func init() {
	registry.Register("impersonator", New)
}

type mgr struct{}

// New returns an auth manager implementation that allows to authenticate with any credentials.
func New(c map[string]any, _ *zerolog.Logger) (auth.Manager, error) {
	return &mgr{}, nil
}

func (m *mgr) Configure(ml map[string]interface{}) error {
	return nil
}

func (m *mgr) Authenticate(ctx context.Context, clientID, clientSecret string) (*user.User, map[string]*authpb.Scope, error) {
	// allow passing in uid as <opaqueid>@<idp>
	at := strings.LastIndex(clientID, "@")
	uid := &user.UserId{Type: user.UserType_USER_TYPE_PRIMARY}
	if at < 0 {
		uid.OpaqueId = clientID
	} else {
		uid.OpaqueId = clientID[:at]
		uid.Idp = clientID[at+1:]
	}

	scope, err := scope.AddOwnerScope(nil)
	if err != nil {
		return nil, nil, err
	}

	return &user.User{
		Id: uid,
		// not much else to provide
	}, scope, nil
}
