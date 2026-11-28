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

package gocdb

import (
	"fmt"

	"github.com/opencloud-eu/reva/v3/pkg/mentix/utils/network"
)

// QueryGOCDB retrieves data from one of GOCDB's endpoints.
func QueryGOCDB(address string, method string, isPrivate bool, scope string, apiKey string, params network.URLParams) ([]byte, error) {
	// The method must always be specified
	params["method"] = method

	// If a scope or an API key were specified, pass them to the endpoint as well
	if len(scope) > 0 {
		params["scope"] = scope
	}

	if len(apiKey) > 0 {
		params["apikey"] = apiKey
	}

	// GOCDB's public API is located at <gocdb-host>/gocdbpi/public, the private one at <gocdb-host>/gocdbpi/private
	var path string
	if isPrivate {
		path = "/gocdbpi/private"
	} else {
		path = "/gocdbpi/public"
	}

	// Query the data from GOCDB
	endpointURL, err := network.GenerateURL(address, path, params)
	if err != nil {
		return nil, fmt.Errorf("unable to generate the GOCDB URL: %v", err)
	}

	data, err := network.ReadEndpoint(endpointURL, nil, true)
	if err != nil {
		return nil, fmt.Errorf("unable to read GOCDB endpoint: %v", err)
	}

	return data, nil
}
