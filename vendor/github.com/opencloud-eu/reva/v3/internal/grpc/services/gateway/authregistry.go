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

	registry "github.com/cs3org/go-cs3apis/cs3/auth/registry/v1beta1"
	gateway "github.com/cs3org/go-cs3apis/cs3/gateway/v1beta1"
	rpc "github.com/cs3org/go-cs3apis/cs3/rpc/v1beta1"
	"github.com/opencloud-eu/reva/v3/pkg/rgrpc/status"
	"github.com/opencloud-eu/reva/v3/pkg/rgrpc/todo/pool"
)

func (s *svc) ListAuthProviders(ctx context.Context, req *registry.ListAuthProvidersRequest) (*gateway.ListAuthProvidersResponse, error) {
	c, err := pool.GetAuthRegistryServiceClient(s.c.AuthRegistryEndpoint)
	if err != nil {
		return &gateway.ListAuthProvidersResponse{
			Status: status.NewInternal(ctx, "gateway"),
		}, nil
	}

	res, err := c.ListAuthProviders(ctx, req)
	if err != nil {
		return &gateway.ListAuthProvidersResponse{
			Status: status.NewInternal(ctx, "gateway"),
		}, nil
	}

	if res.Status.Code != rpc.Code_CODE_OK {
		return &gateway.ListAuthProvidersResponse{
			Status: status.NewInternal(ctx, "gateway"),
		}, nil
	}

	types := make([]string, len(res.Providers))
	for i, p := range res.Providers {
		types[i] = p.ProviderType
	}

	gwRes := &gateway.ListAuthProvidersResponse{
		Status: res.Status,
		Opaque: res.Opaque,
		Types:  types,
	}

	return gwRes, nil
}
