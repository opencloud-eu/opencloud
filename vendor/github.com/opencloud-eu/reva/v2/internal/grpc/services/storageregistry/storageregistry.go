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

package storageregistry

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"

	"github.com/BurntSushi/toml"
	provider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	registrypb "github.com/cs3org/go-cs3apis/cs3/storage/registry/v1beta1"
	typespb "github.com/cs3org/go-cs3apis/cs3/types/v1beta1"
	"github.com/go-viper/mapstructure/v2"
	"github.com/opencloud-eu/reva/v2/pkg/errtypes"
	"github.com/opencloud-eu/reva/v2/pkg/rgrpc"
	"github.com/opencloud-eu/reva/v2/pkg/rgrpc/status"
	sdk "github.com/opencloud-eu/reva/v2/pkg/sdk/common"
	"github.com/opencloud-eu/reva/v2/pkg/storage"
	"github.com/opencloud-eu/reva/v2/pkg/storage/registry/registry"
	"github.com/rs/zerolog"
	"google.golang.org/grpc"
)

func init() {
	rgrpc.Register("storageregistry", New)
}

type service struct {
	reg storage.Registry
}

func (s *service) Close() error {
	return nil
}

func (s *service) UnprotectedEndpoints() []string {
	return []string{}
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

// New creates a new StorageBrokerService
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

func getRegistry(c *config) (storage.Registry, error) {
	if f, ok := registry.NewFuncs[c.Driver]; ok {
		return f(c.Drivers[c.Driver])
	}
	return nil, errtypes.NotFound("driver not found: " + c.Driver)
}

func (s *service) ListStorageProviders(ctx context.Context, req *registrypb.ListStorageProvidersRequest) (*registrypb.ListStorageProvidersResponse, error) {
	pinfos, err := s.reg.ListProviders(ctx, sdk.DecodeOpaqueMap(req.Opaque))
	if err != nil {
		return &registrypb.ListStorageProvidersResponse{
			Status: status.NewInternal(ctx, "error getting list of storage providers"),
		}, nil
	}

	res := &registrypb.ListStorageProvidersResponse{
		Status:    status.NewOK(ctx),
		Providers: pinfos,
	}
	return res, nil
}

func (s *service) GetStorageProviders(ctx context.Context, req *registrypb.GetStorageProvidersRequest) (*registrypb.GetStorageProvidersResponse, error) {
	space, err := decodeSpace(req.Opaque)
	if err != nil {
		return &registrypb.GetStorageProvidersResponse{
			Status: status.NewInvalid(ctx, err.Error()),
		}, nil
	}
	p, err := s.reg.GetProvider(ctx, space)
	if err != nil {
		switch err.(type) {
		case errtypes.IsNotFound:
			return &registrypb.GetStorageProvidersResponse{
				Status: status.NewNotFound(ctx, err.Error()),
			}, nil
		default:
			return &registrypb.GetStorageProvidersResponse{
				Status: status.NewInternal(ctx, "error finding storage provider"),
			}, nil
		}
	}

	res := &registrypb.GetStorageProvidersResponse{
		Status:    status.NewOK(ctx),
		Providers: []*registrypb.ProviderInfo{p},
	}
	return res, nil
}

func decodeSpace(o *typespb.Opaque) (*provider.StorageSpace, error) {
	if entry, ok := o.Map["space"]; ok {
		var unmarshal func([]byte, interface{}) error
		switch entry.Decoder {
		case "json":
			unmarshal = json.Unmarshal
		case "toml":
			unmarshal = toml.Unmarshal
		case "xml":
			unmarshal = xml.Unmarshal
		}

		space := &provider.StorageSpace{}
		return space, unmarshal(entry.Value, space)
	}

	return nil, fmt.Errorf("missing space in opaque property")
}

func (s *service) GetHome(ctx context.Context, req *registrypb.GetHomeRequest) (*registrypb.GetHomeResponse, error) {
	res := &registrypb.GetHomeResponse{
		Status: status.NewUnimplemented(ctx, nil, "getHome is no longer used. use List"),
	}
	return res, nil

}
