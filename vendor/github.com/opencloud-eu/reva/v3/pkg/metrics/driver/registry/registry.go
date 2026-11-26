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

package registry

import (
	"github.com/opencloud-eu/reva/v3/pkg/metrics/reader"
)

var drivers map[string]reader.Reader // map key is driver type name

// Register register a driver
func Register(driverName string, r reader.Reader) {
	if drivers == nil {
		drivers = make(map[string]reader.Reader)
	}
	drivers[driverName] = r
}

// GetDriver returns the registered driver for the specified driver name, or nil if it is not registered
func GetDriver(driverName string) reader.Reader {
	driver, found := drivers[driverName]
	if found {
		return driver
	}
	return nil
}
