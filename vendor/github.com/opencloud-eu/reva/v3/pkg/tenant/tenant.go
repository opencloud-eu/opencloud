// Copyright 2018-2021 CERN
// Copyright 2026 OpenCloud
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

package tenant

import (
	"context"

	tenant "github.com/cs3org/go-cs3apis/cs3/identity/tenant/v1beta1"
)

// Manager is the interface to implement to manipulate users.
type Manager interface {
	// GetTenant returns the tenant metadata identified by an id.
	GetTenant(ctx context.Context, id string) (*tenant.Tenant, error)
	// GetUserByClaim returns the user identified by a specific value for a given claim.
	GetTenantByClaim(ctx context.Context, claim, value string) (*tenant.Tenant, error)
}
