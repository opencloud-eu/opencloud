// Copyright 2018-2021 CERN
// Copyright 2026 OpenCloud GmbH
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

	tenant "github.com/cs3org/go-cs3apis/cs3/identity/tenant/v1beta1"
	"github.com/opencloud-eu/reva/v2/pkg/rgrpc/status"
	"github.com/opencloud-eu/reva/v2/pkg/rgrpc/todo/pool"
	"github.com/pkg/errors"
)

func (s *svc) GetTenant(ctx context.Context, req *tenant.GetTenantRequest) (*tenant.GetTenantResponse, error) {
	c, err := pool.GetTenantProviderServiceClient(s.c.TenantProviderEndpoint)
	if err != nil {
		return &tenant.GetTenantResponse{
			Status: status.NewInternal(ctx, "error getting tenant service client"),
		}, nil
	}

	res, err := c.GetTenant(ctx, req)
	if err != nil {
		return nil, errors.Wrap(err, "gateway: error calling GetTenant")
	}

	return res, nil
}

func (s *svc) GetTenantByClaim(ctx context.Context, req *tenant.GetTenantByClaimRequest) (*tenant.GetTenantByClaimResponse, error) {
	c, err := pool.GetTenantProviderServiceClient(s.c.TenantProviderEndpoint)
	if err != nil {
		return &tenant.GetTenantByClaimResponse{
			Status: status.NewInternal(ctx, "error getting tenant service client"),
		}, nil
	}

	res, err := c.GetTenantByClaim(ctx, req)
	if err != nil {
		return nil, errors.Wrap(err, "gateway: error calling GetTenantByClaim")
	}

	return res, nil
}
