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

package appauth

import (
	"context"

	appauthpb "github.com/cs3org/go-cs3apis/cs3/auth/applications/v1beta1"
	authpb "github.com/cs3org/go-cs3apis/cs3/auth/provider/v1beta1"
	user "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"
	rpcv1beta1 "github.com/cs3org/go-cs3apis/cs3/rpc/v1beta1"
	"github.com/mitchellh/mapstructure"
	"github.com/opencloud-eu/reva/v2/pkg/auth"
	"github.com/opencloud-eu/reva/v2/pkg/auth/manager/registry"
	"github.com/opencloud-eu/reva/v2/pkg/errtypes"
	"github.com/opencloud-eu/reva/v2/pkg/rgrpc/todo/pool"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
)

func init() {
	registry.Register("appauth", New)
}

type manager struct {
	GatewayAddr string `mapstructure:"gateway_addr"`
}

// New returns a new auth Manager.
func New(m map[string]any, logger *zerolog.Logger) (auth.Manager, error) {
	mgr := &manager{}
	err := mgr.Configure(m)
	if err != nil {
		return nil, err
	}
	return mgr, nil
}

func (m *manager) Configure(ml map[string]interface{}) error {
	err := mapstructure.Decode(ml, m)
	if err != nil {
		return errors.Wrap(err, "error decoding conf")
	}
	return nil
}

func (m *manager) Authenticate(ctx context.Context, username, password string) (*user.User, map[string]*authpb.Scope, error) {
	gtw, err := pool.GetGatewayServiceClient(m.GatewayAddr)
	if err != nil {
		return nil, nil, err
	}

	// get user info
	userResponse, err := gtw.GetUserByClaim(ctx, &user.GetUserByClaimRequest{
		Claim: "username",
		Value: username,
	})

	switch {
	case err != nil:
		return nil, nil, err
	case userResponse.Status.Code == rpcv1beta1.Code_CODE_NOT_FOUND:
		return nil, nil, errtypes.NotFound(userResponse.Status.Message)
	case userResponse.Status.Code != rpcv1beta1.Code_CODE_OK:
		return nil, nil, errtypes.InternalError(userResponse.Status.Message)
	}

	// get the app password associated with the user and password
	appAuthResponse, err := gtw.GetAppPassword(ctx, &appauthpb.GetAppPasswordRequest{
		User:     userResponse.GetUser().Id,
		Password: password,
	})

	switch {
	case err != nil:
		return nil, nil, err
	case appAuthResponse.Status.Code == rpcv1beta1.Code_CODE_NOT_FOUND:
		return nil, nil, errtypes.NotFound(appAuthResponse.Status.Message)
	case appAuthResponse.Status.Code != rpcv1beta1.Code_CODE_OK:
		return nil, nil, errtypes.InternalError(appAuthResponse.Status.Message)
	}

	return userResponse.GetUser(), appAuthResponse.GetAppPassword().TokenScope, nil
}
