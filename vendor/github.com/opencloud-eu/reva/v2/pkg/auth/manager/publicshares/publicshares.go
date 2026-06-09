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

package publicshares

import (
	"context"
	"strings"
	"time"

	authpb "github.com/cs3org/go-cs3apis/cs3/auth/provider/v1beta1"
	user "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"
	rpcv1beta1 "github.com/cs3org/go-cs3apis/cs3/rpc/v1beta1"
	link "github.com/cs3org/go-cs3apis/cs3/sharing/link/v1beta1"
	types "github.com/cs3org/go-cs3apis/cs3/types/v1beta1"
	"github.com/go-viper/mapstructure/v2"
	"github.com/opencloud-eu/reva/v2/pkg/auth"
	"github.com/opencloud-eu/reva/v2/pkg/auth/manager/registry"
	"github.com/opencloud-eu/reva/v2/pkg/auth/scope"
	"github.com/opencloud-eu/reva/v2/pkg/errtypes"
	"github.com/opencloud-eu/reva/v2/pkg/rgrpc/todo/pool"
	"github.com/opencloud-eu/reva/v2/pkg/utils"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
)

func init() {
	registry.Register("publicshares", New)
}

type manager struct {
	c *config
}

type config struct {
	GatewayAddr string `mapstructure:"gateway_addr"`
}

func parseConfig(m map[string]interface{}) (*config, error) {
	c := &config{}
	if err := mapstructure.Decode(m, c); err != nil {
		err = errors.Wrap(err, "error decoding conf")
		return nil, err
	}
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
	conf, err := parseConfig(ml)
	if err != nil {
		return err
	}
	m.c = conf
	return nil
}

func (m *manager) Authenticate(ctx context.Context, token, secret string) (*user.User, map[string]*authpb.Scope, error) {
	gwConn, err := pool.GetGatewayServiceClient(m.c.GatewayAddr)
	if err != nil {
		return nil, nil, err
	}

	var auth *link.PublicShareAuthentication
	if strings.HasPrefix(secret, "password|") {
		secret = strings.TrimPrefix(secret, "password|")
		auth = &link.PublicShareAuthentication{
			Spec: &link.PublicShareAuthentication_Password{
				Password: secret,
			},
		}
	} else if strings.HasPrefix(secret, "signature|") {
		secret = strings.TrimPrefix(secret, "signature|")
		parts := strings.Split(secret, "|")
		sig, expiration := parts[0], parts[1]
		exp, _ := time.Parse(time.RFC3339, expiration)

		auth = &link.PublicShareAuthentication{
			Spec: &link.PublicShareAuthentication_Signature{
				Signature: &link.ShareSignature{
					Signature: sig,
					SignatureExpiration: &types.Timestamp{
						Seconds: uint64(exp.UnixNano() / 1000000000),
						Nanos:   uint32(exp.UnixNano() % 1000000000),
					},
				},
			},
		}
	}

	publicShareResponse, err := gwConn.GetPublicShareByToken(ctx, &link.GetPublicShareByTokenRequest{
		Token:          token,
		Authentication: auth,
		Sign:           true,
	})
	switch {
	case err != nil:
		return nil, nil, err
	case publicShareResponse.Status.Code == rpcv1beta1.Code_CODE_NOT_FOUND:
		return nil, nil, errtypes.NotFound(publicShareResponse.Status.Message)
	case publicShareResponse.Status.Code == rpcv1beta1.Code_CODE_PERMISSION_DENIED:
		return nil, nil, errtypes.InvalidCredentials(publicShareResponse.Status.Message)
	case publicShareResponse.Status.Code != rpcv1beta1.Code_CODE_OK:
		return nil, nil, errtypes.InternalError(publicShareResponse.Status.Message)
	}

	var owner *user.User
	// FIXME use new user type SPACE_OWNER
	if publicShareResponse.GetShare().GetOwner().GetType() == 8 {
		owner = &user.User{Id: publicShareResponse.GetShare().GetOwner(), DisplayName: "Public", Username: "public"}
	} else {
		getUserResponse, err := gwConn.GetUser(ctx, &user.GetUserRequest{
			UserId: publicShareResponse.GetShare().GetCreator(),
		})
		switch {
		case err != nil:
			return nil, nil, err
		case getUserResponse.GetStatus().GetCode() == rpcv1beta1.Code_CODE_NOT_FOUND:
			return nil, nil, errtypes.NotFound(getUserResponse.GetStatus().GetMessage())
		case getUserResponse.GetStatus().GetCode() == rpcv1beta1.Code_CODE_PERMISSION_DENIED:
			return nil, nil, errtypes.InvalidCredentials(getUserResponse.GetStatus().GetMessage())
		case getUserResponse.GetStatus().GetCode() != rpcv1beta1.Code_CODE_OK:
			return nil, nil, errtypes.InternalError(getUserResponse.GetStatus().GetMessage())
		}
		owner = getUserResponse.GetUser()
	}

	share := publicShareResponse.GetShare()
	role := authpb.Role_ROLE_VIEWER
	roleStr := "viewer"
	if share.Permissions.Permissions.InitiateFileUpload && !share.Permissions.Permissions.InitiateFileDownload {
		role = authpb.Role_ROLE_UPLOADER
		roleStr = "uploader"
	} else if share.Permissions.Permissions.InitiateFileUpload {
		role = authpb.Role_ROLE_EDITOR
		roleStr = "editor"
	}

	scope, err := scope.AddPublicShareScope(share, role, nil)
	if err != nil {
		return nil, nil, err
	}

	owner.Opaque = &types.Opaque{
		Map: map[string]*types.OpaqueEntry{
			"public-share-role": {
				Decoder: "plain",
				Value:   []byte(roleStr),
			},
		},
	}

	u := &user.User{Id: &user.UserId{OpaqueId: token, Idp: "public", Type: user.UserType_USER_TYPE_GUEST}, DisplayName: "Public", Username: "public"}
	owner.Opaque = utils.AppendJSONToOpaque(owner.Opaque, "impersonating-user", u)

	return owner, scope, nil
}

// ErrPasswordNotProvided is returned when the public share is password protected, but there was no password on the request
var ErrPasswordNotProvided = errors.New("public share is password protected, but password was not provided")
