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

package authprovider

import (
	"context"
	"fmt"
	"path/filepath"

	provider "github.com/cs3org/go-cs3apis/cs3/auth/provider/v1beta1"
	"github.com/mitchellh/mapstructure"
	"github.com/opencloud-eu/reva/v2/pkg/appctx"
	"github.com/opencloud-eu/reva/v2/pkg/auth"
	"github.com/opencloud-eu/reva/v2/pkg/auth/manager/registry"
	"github.com/opencloud-eu/reva/v2/pkg/errtypes"
	"github.com/opencloud-eu/reva/v2/pkg/plugin"
	"github.com/opencloud-eu/reva/v2/pkg/rgrpc"
	"github.com/opencloud-eu/reva/v2/pkg/rgrpc/status"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"google.golang.org/grpc"
)

func init() {
	rgrpc.Register("authprovider", New)
}

type config struct {
	AuthManager  string                            `mapstructure:"auth_manager"`
	AuthManagers map[string]map[string]interface{} `mapstructure:"auth_managers"`
}

func (c *config) init() {
	if c.AuthManager == "" {
		c.AuthManager = "json"
	}
}

type service struct {
	authmgr auth.Manager
	conf    *config
	plugin  *plugin.RevaPlugin
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

func getAuthManager(manager string, m map[string]map[string]any, logger *zerolog.Logger) (auth.Manager, *plugin.RevaPlugin, error) {
	if manager == "" {
		return nil, nil, errtypes.InternalError("authsvc: driver not configured for auth manager")
	}
	p, err := plugin.Load("authprovider", manager)
	if err == nil {
		authManager, ok := p.Plugin.(auth.Manager)
		if !ok {
			return nil, nil, fmt.Errorf("could not assert the loaded plugin")
		}
		pluginConfig := filepath.Base(manager)
		err = authManager.Configure(m[pluginConfig])
		if err != nil {
			return nil, nil, err
		}
		return authManager, p, nil
	} else if _, ok := err.(errtypes.NotFound); ok {
		if f, ok := registry.NewFuncs[manager]; ok {
			authmgr, err := f(m[manager], logger)
			return authmgr, nil, err
		}
	} else {
		return nil, nil, err
	}
	return nil, nil, errtypes.NotFound(fmt.Sprintf("authsvc: driver %s not found for auth manager", manager))
}

// New returns a new AuthProviderServiceServer.
func New(m map[string]interface{}, ss *grpc.Server, logger *zerolog.Logger) (rgrpc.Service, error) {
	c, err := parseConfig(m)
	if err != nil {
		return nil, err
	}

	authManager, plug, err := getAuthManager(c.AuthManager, c.AuthManagers, logger)
	if err != nil {
		return nil, err
	}

	svc := &service{
		conf:    c,
		authmgr: authManager,
		plugin:  plug,
	}

	return svc, nil
}

func (s *service) Close() error {
	if s.plugin != nil {
		s.plugin.Kill()
	}
	return nil
}

func (s *service) UnprotectedEndpoints() []string {
	return []string{"/cs3.auth.provider.v1beta1.ProviderAPI/Authenticate"}
}

func (s *service) Register(ss *grpc.Server) {
	provider.RegisterProviderAPIServer(ss, s)
}

func (s *service) Authenticate(ctx context.Context, req *provider.AuthenticateRequest) (*provider.AuthenticateResponse, error) {
	log := appctx.GetLogger(ctx)
	username := req.ClientId
	password := req.ClientSecret

	u, scope, err := s.authmgr.Authenticate(ctx, username, password)
	if err != nil {
		log.Debug().Str("client_id", username).Err(err).Msg("authsvc: error in Authenticate")
		return &provider.AuthenticateResponse{
			Status: status.NewStatusFromErrType(ctx, "authsvc: error in Authenticate", err),
		}, nil
	}
	log.Info().Msgf("user %s authenticated", u.Id)
	return &provider.AuthenticateResponse{
		Status:     status.NewOK(ctx),
		User:       u,
		TokenScope: scope,
	}, nil
}
