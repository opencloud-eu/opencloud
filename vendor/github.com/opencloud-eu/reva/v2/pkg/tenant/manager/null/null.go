// Copyright 2018-2020 CERN
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

package null

import (
	"context"

	tenantpb "github.com/cs3org/go-cs3apis/cs3/identity/tenant/v1beta1"
	"github.com/opencloud-eu/reva/v2/pkg/errtypes"
	"github.com/opencloud-eu/reva/v2/pkg/tenant"
	"github.com/opencloud-eu/reva/v2/pkg/tenant/manager/registry"
)

func init() {
	registry.Register("null", New)
}

type manager struct {
}

// New returns a tenant manager implementation that return NOT FOUND or empty result set for every call
func New(m map[string]interface{}) (tenant.Manager, error) {
	return &manager{}, nil
}

func (m *manager) GetTenant(ctx context.Context, id string) (*tenantpb.Tenant, error) {
	return nil, errtypes.NotFound(id)
}

func (m *manager) GetTenantByClaim(ctx context.Context, claim, value string) (*tenantpb.Tenant, error) {
	return nil, errtypes.NotFound(value)
}
