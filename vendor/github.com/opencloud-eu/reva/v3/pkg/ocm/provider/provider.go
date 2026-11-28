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

package provider

import (
	"context"

	ocmprovider "github.com/cs3org/go-cs3apis/cs3/ocm/provider/v1beta1"
)

// Authorizer provides provisions to verify and add sync'n'share system providers.
type Authorizer interface {
	// GetInfoByDomain returns the information of the provider identified by a specific domain.
	GetInfoByDomain(ctx context.Context, domain string) (*ocmprovider.ProviderInfo, error)

	// IsProviderAllowed checks if a given system provider is integrated into the OCM or not.
	IsProviderAllowed(ctx context.Context, provider *ocmprovider.ProviderInfo) error

	// ListAllProviders returns the information of all the providers registered in the mesh.
	ListAllProviders(ctx context.Context) ([]*ocmprovider.ProviderInfo, error)
}
