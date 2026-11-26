// Copyright 2018-2020 CERN
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

package meshdata

const (
	// EndpointRevad identifies the main Reva Daemon endpoint
	EndpointRevad = "REVAD"

	// EndpointGateway identifies the Gateway endpoint
	EndpointGateway = "GATEWAY"
	// EndpointMetrics identifies the Metrics endpoint
	EndpointMetrics = "METRICS"
	// EndpointWebdav identifies the Webdav endpoint
	EndpointWebdav = "WEBDAV"
	// EndpointOCM identifies the OCM endpoint
	EndpointOCM = "OCM"
	// EndpointMeshDir identifies the Mesh Directory endpoint
	EndpointMeshDir = "MESHDIR"
)

// GetServiceEndpoints returns an array of all service endpoint identifiers.
func GetServiceEndpoints() []string {
	return []string{
		EndpointRevad,

		EndpointGateway,
		EndpointMetrics,
		EndpointWebdav,
		EndpointOCM,
		EndpointMeshDir,
	}
}
