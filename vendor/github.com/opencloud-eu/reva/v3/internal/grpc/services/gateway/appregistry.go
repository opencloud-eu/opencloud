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

package gateway

import (
	"context"

	registry "github.com/cs3org/go-cs3apis/cs3/app/registry/v1beta1"
	"github.com/opencloud-eu/reva/v3/pkg/rgrpc/status"
	"github.com/opencloud-eu/reva/v3/pkg/rgrpc/todo/pool"
	"github.com/pkg/errors"
)

func (s *svc) GetAppProviders(ctx context.Context, req *registry.GetAppProvidersRequest) (*registry.GetAppProvidersResponse, error) {
	c, err := pool.GetAppRegistryClient(s.c.AppRegistryEndpoint)
	if err != nil {
		return &registry.GetAppProvidersResponse{
			Status: status.NewInternal(ctx, "error getting app registry client"),
		}, nil
	}

	res, err := c.GetAppProviders(ctx, req)
	if err != nil {
		return nil, errors.Wrap(err, "gateway: error calling GetAppProviders")
	}

	return res, nil
}

func (s *svc) AddAppProvider(ctx context.Context, req *registry.AddAppProviderRequest) (*registry.AddAppProviderResponse, error) {
	c, err := pool.GetAppRegistryClient(s.c.AppRegistryEndpoint)
	if err != nil {
		return &registry.AddAppProviderResponse{
			Status: status.NewInternal(ctx, "error getting app registry client"),
		}, nil
	}

	res, err := c.AddAppProvider(ctx, req)
	if err != nil {
		return nil, errors.Wrap(err, "gateway: error calling AddAppProvider")
	}

	return res, nil
}

func (s *svc) ListAppProviders(ctx context.Context, req *registry.ListAppProvidersRequest) (*registry.ListAppProvidersResponse, error) {
	c, err := pool.GetAppRegistryClient(s.c.AppRegistryEndpoint)
	if err != nil {
		return &registry.ListAppProvidersResponse{
			Status: status.NewInternal(ctx, "error getting app registry client"),
		}, nil
	}

	res, err := c.ListAppProviders(ctx, req)
	if err != nil {
		return nil, errors.Wrap(err, "gateway: error calling ListAppProviders")
	}

	return res, nil
}

func (s *svc) ListSupportedMimeTypes(ctx context.Context, req *registry.ListSupportedMimeTypesRequest) (*registry.ListSupportedMimeTypesResponse, error) {
	c, err := pool.GetAppRegistryClient(s.c.AppRegistryEndpoint)
	if err != nil {
		return &registry.ListSupportedMimeTypesResponse{
			Status: status.NewInternal(ctx, "error getting app registry client"),
		}, nil
	}

	res, err := c.ListSupportedMimeTypes(ctx, req)
	if err != nil {
		return nil, errors.Wrap(err, "gateway: error calling ListSupportedMimeTypes")
	}

	return res, nil
}

func (s *svc) GetDefaultAppProviderForMimeType(ctx context.Context, req *registry.GetDefaultAppProviderForMimeTypeRequest) (*registry.GetDefaultAppProviderForMimeTypeResponse, error) {
	c, err := pool.GetAppRegistryClient(s.c.AppRegistryEndpoint)
	if err != nil {
		return &registry.GetDefaultAppProviderForMimeTypeResponse{
			Status: status.NewInternal(ctx, "error getting app registry client"),
		}, nil
	}

	res, err := c.GetDefaultAppProviderForMimeType(ctx, req)
	if err != nil {
		return nil, errors.Wrap(err, "gateway: error calling GetDefaultAppProviderForMimeType")
	}

	return res, nil
}

func (s *svc) SetDefaultAppProviderForMimeType(ctx context.Context, req *registry.SetDefaultAppProviderForMimeTypeRequest) (*registry.SetDefaultAppProviderForMimeTypeResponse, error) {
	c, err := pool.GetAppRegistryClient(s.c.AppRegistryEndpoint)
	if err != nil {
		return &registry.SetDefaultAppProviderForMimeTypeResponse{
			Status: status.NewInternal(ctx, "error getting app registry client"),
		}, nil
	}

	res, err := c.SetDefaultAppProviderForMimeType(ctx, req)
	if err != nil {
		return nil, errors.Wrap(err, "gateway: error calling SetDefaultAppProviderForMimeType")
	}

	return res, nil
}
