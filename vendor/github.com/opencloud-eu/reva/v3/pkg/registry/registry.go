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

package registry

import (
	mRegistry "go-micro.dev/v4/registry"
	"go-micro.dev/v4/selector"
)

var (
	// fixme: get rid of global registry
	gRegistry mRegistry.Registry
)

// Init prepares the service registry
func Init(nRegistry mRegistry.Registry) error {
	// first come first serves, the first service defines the registry type.
	if gRegistry == nil && nRegistry != nil {
		gRegistry = nRegistry
	}

	return nil
}

// GetRegistry exposes the registry
func GetRegistry() mRegistry.Registry {
	return gRegistry
}

// GetNodeAddress returns a random address from the service nodes
func GetNodeAddress(services []*mRegistry.Service) (string, error) {
	next := selector.Random(services)
	node, err := next()
	if err != nil {
		return "", err
	}

	return node.Address, nil
}
