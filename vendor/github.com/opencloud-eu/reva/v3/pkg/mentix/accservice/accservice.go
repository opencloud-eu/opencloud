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

package accservice

import (
	"encoding/json"
	"fmt"
	"net/url"
	"path"
	"strings"

	"github.com/pkg/errors"

	"github.com/opencloud-eu/reva/v3/pkg/mentix/config"
	"github.com/opencloud-eu/reva/v3/pkg/mentix/utils/network"
)

// RequestResponse holds the response of an accounts service query.
type RequestResponse struct {
	Success bool
	Error   string
	Data    interface{}
}

type accountsServiceSettings struct {
	URL      *url.URL
	User     string
	Password string
}

var settings accountsServiceSettings

// Query performs an account service query.
func Query(endpoint string, params network.URLParams) (*RequestResponse, error) {
	fullURL, err := network.GenerateURL(fmt.Sprintf("%v://%v", settings.URL.Scheme, settings.URL.Host), path.Join(settings.URL.Path, endpoint), params)
	if err != nil {
		return nil, errors.Wrap(err, "error while building the service accounts query URL")
	}

	data, err := network.ReadEndpoint(fullURL, &network.BasicAuth{User: settings.User, Password: settings.Password}, false)
	if err != nil {
		return nil, errors.Wrap(err, "unable to query the service accounts endpoint")
	}

	resp := &RequestResponse{}
	if err := json.Unmarshal(data, resp); err != nil {
		return nil, errors.Wrap(err, "unable to unmarshal response data")
	}
	return resp, nil
}

// GetResponseValue gets a value from an account service query using a dotted path notation.
func GetResponseValue(resp *RequestResponse, path string) interface{} {
	if data, ok := resp.Data.(map[string]interface{}); ok {
		tokens := strings.Split(path, ".")
		for i, name := range tokens {
			if i == len(tokens)-1 {
				if value, ok := data[name]; ok {
					return value
				}
			}

			if data, ok = data[name].(map[string]interface{}); !ok {
				break
			}
		}
	}

	return nil
}

// InitAccountsService initializes the global accounts service.
func InitAccountsService(conf *config.Configuration) error {
	URL, err := url.Parse(conf.AccountsService.URL)
	if err != nil {
		return errors.Wrap(err, "unable to parse the accounts service URL")
	}

	settings.URL = URL
	settings.User = conf.AccountsService.User
	settings.Password = conf.AccountsService.Password

	return nil
}
