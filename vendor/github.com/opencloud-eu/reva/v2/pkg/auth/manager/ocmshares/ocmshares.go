// Copyright 2018-2023 CERN
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

package ocmshares

import (
	"context"

	provider "github.com/cs3org/go-cs3apis/cs3/app/provider/v1beta1"
	authpb "github.com/cs3org/go-cs3apis/cs3/auth/provider/v1beta1"
	gateway "github.com/cs3org/go-cs3apis/cs3/gateway/v1beta1"
	userpb "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"
	ocminvite "github.com/cs3org/go-cs3apis/cs3/ocm/invite/v1beta1"
	rpc "github.com/cs3org/go-cs3apis/cs3/rpc/v1beta1"
	ocm "github.com/cs3org/go-cs3apis/cs3/sharing/ocm/v1beta1"
	types "github.com/cs3org/go-cs3apis/cs3/types/v1beta1"
	"github.com/opencloud-eu/reva/v2/pkg/appctx"
	"github.com/opencloud-eu/reva/v2/pkg/auth"
	"github.com/opencloud-eu/reva/v2/pkg/auth/manager/registry"
	"github.com/opencloud-eu/reva/v2/pkg/auth/scope"
	"github.com/opencloud-eu/reva/v2/pkg/errtypes"
	"github.com/opencloud-eu/reva/v2/pkg/rgrpc/todo/pool"
	"github.com/opencloud-eu/reva/v2/pkg/sharedconf"
	"github.com/opencloud-eu/reva/v2/pkg/utils"
	"github.com/opencloud-eu/reva/v2/pkg/utils/cfg"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
)

func init() {
	registry.Register("ocmshares", New)
}

type manager struct {
	c  *config
	gw gateway.GatewayAPIClient
}

type config struct {
	GatewayAddr string `mapstructure:"gatewaysvc"`
}

func (c *config) ApplyDefaults() {
	c.GatewayAddr = sharedconf.GetGatewaySVC(c.GatewayAddr)
}

// New creates a new ocmshares authentication manager.
func New(m map[string]any, _ *zerolog.Logger) (auth.Manager, error) {
	var mgr manager
	if err := mgr.Configure(m); err != nil {
		return nil, err
	}
	gw, err := pool.GetGatewayServiceClient(mgr.c.GatewayAddr)
	if err != nil {
		return nil, err
	}
	mgr.gw = gw

	return &mgr, nil
}

func (m *manager) Configure(ml map[string]interface{}) error {
	var c config
	if err := cfg.Decode(ml, &c); err != nil {
		return errors.Wrap(err, "ocmshares: error decoding config")
	}
	m.c = &c
	return nil
}

func (m *manager) Authenticate(ctx context.Context, ocmshare, sharedSecret string) (*userpb.User, map[string]*authpb.Scope, error) {
	log := appctx.GetLogger(ctx).With().Str("ocmshare", ocmshare).Logger()
	// We need to use GetOCMShareByToken, as GetOCMShare would require a user in the context
	shareRes, err := m.gw.GetOCMShareByToken(ctx, &ocm.GetOCMShareByTokenRequest{
		Token: sharedSecret,
	})

	switch {
	case err != nil:
		log.Error().Err(err).Msg("error getting ocm share by token")
		return nil, nil, err
	case shareRes.Status.Code == rpc.Code_CODE_NOT_FOUND:
		log.Debug().Msg("ocm share not found")
		return nil, nil, errtypes.NotFound(shareRes.Status.Message)
	case shareRes.Status.Code == rpc.Code_CODE_PERMISSION_DENIED:
		log.Debug().Msg("permission denied")
		return nil, nil, errtypes.InvalidCredentials(shareRes.Status.Message)
	case shareRes.Status.Code != rpc.Code_CODE_OK:
		log.Error().Interface("status", shareRes.Status).Msg("got unexpected error in the grpc call to GetOCMShare")
		return nil, nil, errtypes.InternalError(shareRes.Status.Message)
	}

	// compare ocm share id
	if shareRes.GetShare().GetId().GetOpaqueId() != ocmshare {
		log.Error().Str("persisted", ocmshare).Str("requested", shareRes.GetShare().GetId().GetOpaqueId()).Msg("mismatching ocm share id for existing secret")
		return nil, nil, errtypes.InvalidCredentials("invalid shared secret")
	}

	// the user authenticated using the ocmshares authentication method
	// is the recipient of the share
	u := shareRes.Share.Grantee.GetUserId()

	d, err := utils.MarshalProtoV1ToJSON(shareRes.GetShare().Creator)
	if err != nil {
		return nil, nil, err
	}

	o := &types.Opaque{
		Map: map[string]*types.OpaqueEntry{
			"user-filter": {
				Decoder: "json",
				Value:   d,
			},
		},
	}

	userRes, err := m.gw.GetAcceptedUser(ctx, &ocminvite.GetAcceptedUserRequest{
		RemoteUserId: u,
		Opaque:       o,
	})

	switch {
	case err != nil:
		return nil, nil, err
	case userRes.Status.Code == rpc.Code_CODE_NOT_FOUND:
		return nil, nil, errtypes.NotFound(shareRes.Status.Message)
	case userRes.Status.Code != rpc.Code_CODE_OK:
		return nil, nil, errtypes.InternalError(userRes.Status.Message)
	}

	role, roleStr := getRole(shareRes.Share)

	scope, err := scope.AddOCMShareScope(shareRes.Share, role, nil)
	if err != nil {
		return nil, nil, err
	}

	user := userRes.RemoteUser
	user.Opaque = &types.Opaque{
		Map: map[string]*types.OpaqueEntry{
			"ocm-share-role": {
				Decoder: "plain",
				Value:   []byte(roleStr),
			},
		},
	}

	user.Opaque = utils.AppendJSONToOpaque(user.Opaque, "impersonating-user", userRes.RemoteUser)

	return user, scope, nil
}

func getRole(s *ocm.Share) (authpb.Role, string) {
	// TODO: consider to somehow merge the permissions from all the access methods?
	// it's not clear infact which should be the role when webdav is editor role while
	// webapp is only view mode for example
	// this implementation considers only the simple case in which when a client creates
	// a share with multiple access methods, the permissions are matching in all of them.
	for _, m := range s.AccessMethods {
		switch v := m.Term.(type) {
		case *ocm.AccessMethod_WebdavOptions:
			p := v.WebdavOptions.Permissions
			if p.InitiateFileUpload {
				return authpb.Role_ROLE_EDITOR, "editor"
			}
			if p.InitiateFileDownload {
				return authpb.Role_ROLE_VIEWER, "viewer"
			}
		case *ocm.AccessMethod_WebappOptions:
			viewMode := v.WebappOptions.ViewMode
			if viewMode == provider.ViewMode_VIEW_MODE_VIEW_ONLY ||
				viewMode == provider.ViewMode_VIEW_MODE_READ_ONLY ||
				viewMode == provider.ViewMode_VIEW_MODE_PREVIEW {
				return authpb.Role_ROLE_VIEWER, "viewer"
			}
			if viewMode == provider.ViewMode_VIEW_MODE_READ_WRITE {
				return authpb.Role_ROLE_EDITOR, "editor"
			}
		}
	}
	return authpb.Role_ROLE_INVALID, "invalid"
}
