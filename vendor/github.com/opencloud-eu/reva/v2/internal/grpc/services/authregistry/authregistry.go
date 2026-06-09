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

package authregistry

import (
	"context"

	registrypb "github.com/cs3org/go-cs3apis/cs3/auth/registry/v1beta1"
	"github.com/go-viper/mapstructure/v2"
	"github.com/opencloud-eu/reva/v2/pkg/auth"
	"github.com/opencloud-eu/reva/v2/pkg/auth/registry/registry"
	"github.com/opencloud-eu/reva/v2/pkg/errtypes"
	"github.com/opencloud-eu/reva/v2/pkg/rgrpc"
	"github.com/opencloud-eu/reva/v2/pkg/rgrpc/status"
	"github.com/rs/zerolog"
	"google.golang.org/grpc"
)

func init() {
	rgrpc.Register("authregistry", New)
}

type service struct {
	reg auth.Registry
}

func (s *service) Close() error {
	return nil
}

func (s *service) UnprotectedEndpoints() []string {
	return []string{
		"/cs3.auth.registry.v1beta1.RegistryAPI/GetAuthProvider",
		"/cs3.auth.registry.v1beta1.RegistryAPI/ListAuthProviders",
	}
}

func (s *service) Register(ss *grpc.Server) {
	registrypb.RegisterRegistryAPIServer(ss, s)
}

type config struct {
	Driver  string                            `mapstructure:"driver"`
	Drivers map[string]map[string]interface{} `mapstructure:"drivers"`
}

func (c *config) init() {
	if c.Driver == "" {
		c.Driver = "static"
	}
}

// New creates a new AuthRegistry
func New(m map[string]interface{}, ss *grpc.Server, _ *zerolog.Logger) (rgrpc.Service, error) {
	c, err := parseConfig(m)
	if err != nil {
		return nil, err
	}

	c.init()

	reg, err := getRegistry(c)
	if err != nil {
		return nil, err
	}

	service := &service{
		reg: reg,
	}

	return service, nil
}

func parseConfig(m map[string]interface{}) (*config, error) {
	c := &config{}
	if err := mapstructure.Decode(m, c); err != nil {
		return nil, err
	}
	return c, nil
}

func getRegistry(c *config) (auth.Registry, error) {
	if f, ok := registry.NewFuncs[c.Driver]; ok {
		return f(c.Drivers[c.Driver])
	}
	return nil, errtypes.NotFound("authregistrysvc: driver not found: " + c.Driver)
}

func (s *service) ListAuthProviders(ctx context.Context, req *registrypb.ListAuthProvidersRequest) (*registrypb.ListAuthProvidersResponse, error) {
	pinfos, err := s.reg.ListProviders(ctx)
	if err != nil {
		return &registrypb.ListAuthProvidersResponse{
			Status: status.NewInternal(ctx, "error getting list of auth providers"),
		}, nil
	}

	res := &registrypb.ListAuthProvidersResponse{
		Status:    status.NewOK(ctx),
		Providers: pinfos,
	}
	return res, nil
}

func (s *service) GetAuthProviders(ctx context.Context, req *registrypb.GetAuthProvidersRequest) (*registrypb.GetAuthProvidersResponse, error) {
	pinfo, err := s.reg.GetProvider(ctx, req.Type)
	if err != nil {
		return &registrypb.GetAuthProvidersResponse{
			Status: status.NewInternal(ctx, "error getting auth provider for type: "+req.Type),
		}, nil
	}

	res := &registrypb.GetAuthProvidersResponse{
		Status:    status.NewOK(ctx),
		Providers: []*registrypb.ProviderInfo{pinfo},
	}
	return res, nil
}
