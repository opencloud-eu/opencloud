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

package reader

/*
Reader is the interface that defines the metrics to read.
Any metrics data driver must implement this interface.
Each metric function should return the current/latest available metrics figure relevant to that function.
*/

import "github.com/opencloud-eu/reva/v3/pkg/metrics/config"

// Reader the Reader interface
type Reader interface {

	// Configure configures the reader according to the specified configuration
	Configure(c *config.Config) error

	// GetNumUsersView returns an OpenCensus stats view which records the
	// number of users registered in the mesh provider.
	// Metric name: cs3_org_sciencemesh_site_total_num_users
	GetNumUsers() int64

	// GetNumGroupsView returns an OpenCensus stats view which records the
	// number of user groups registered in the mesh provider.
	// Metric name: cs3_org_sciencemesh_site_total_num_groups
	GetNumGroups() int64

	// GetAmountStorageView returns an OpenCensus stats view which records the
	// amount of storage in the system.
	// Metric name: cs3_org_sciencemesh_site_total_amount_storage
	GetAmountStorage() int64
}
