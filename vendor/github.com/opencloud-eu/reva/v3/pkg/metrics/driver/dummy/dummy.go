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

package dummy

import (
	"math/rand"

	"github.com/opencloud-eu/reva/v3/pkg/metrics/config"
	"github.com/opencloud-eu/reva/v3/pkg/metrics/driver/registry"
)

func init() {
	driver := &MetricsDummyDriver{}
	registry.Register(driverName(), driver)
}

func driverName() string {
	return "dummy"
}

// MetricsDummyDriver the MetricsDummyDriver struct
type MetricsDummyDriver struct {
}

// Configure configures this driver
func (d *MetricsDummyDriver) Configure(c *config.Config) error {
	// no configuration necessary
	return nil
}

// GetNumUsers returns the number of site users; it's a random number
func (d *MetricsDummyDriver) GetNumUsers() int64 {
	return int64(rand.Intn(30000))
}

// GetNumGroups returns the number of site groups; it's a random number
func (d *MetricsDummyDriver) GetNumGroups() int64 {
	return int64(rand.Intn(200))
}

// GetAmountStorage returns the amount of site storage used; it's a random amount
func (d *MetricsDummyDriver) GetAmountStorage() int64 {
	return rand.Int63n(70000000000)
}
