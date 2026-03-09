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

package json

import (
	"encoding/json"
	"errors"
	"os"

	"github.com/opencloud-eu/reva/v2/pkg/metrics/driver/registry"

	"github.com/opencloud-eu/reva/v2/pkg/logger"
	"github.com/opencloud-eu/reva/v2/pkg/metrics/config"
	"github.com/rs/zerolog"
)

var log zerolog.Logger

func init() {
	log = logger.New().With().Int("pid", os.Getpid()).Logger()
	driver := &MetricsJSONDriver{}
	registry.Register(driverName(), driver)
}

func driverName() string {
	return "json"
}

// readJSON always returns a data object but logs the error in case reading the json fails.
func readJSON(driver *MetricsJSONDriver) *data {
	data := &data{}

	file, err := os.ReadFile(driver.metricsDataLocation)
	if err != nil {
		log.Error().Err(err).Str("location", driver.metricsDataLocation).Msg("Unable to read json file from location.")
	}
	err = json.Unmarshal(file, data)
	if err != nil {
		log.Error().Err(err).Msg("Unable to unmarshall json file.")
	}

	return data
}

type data struct {
	NumUsers      int64 `json:"cs3_org_sciencemesh_site_total_num_users"`
	NumGroups     int64 `json:"cs3_org_sciencemesh_site_total_num_groups"`
	AmountStorage int64 `json:"cs3_org_sciencemesh_site_total_amount_storage"`
}

// MetricsJSONDriver the JsonDriver struct
type MetricsJSONDriver struct {
	metricsDataLocation string
}

// Configure configures this driver
func (d *MetricsJSONDriver) Configure(c *config.Config) error {
	if c.MetricsDataLocation == "" {
		err := errors.New("unable to initialize a metrics data driver, has the data location (metrics_data_location) been configured?")
		return err
	}

	d.metricsDataLocation = c.MetricsDataLocation

	return nil
}

// GetNumUsers returns the number of site users
func (d *MetricsJSONDriver) GetNumUsers() int64 {
	return readJSON(d).NumUsers
}

// GetNumGroups returns the number of site groups
func (d *MetricsJSONDriver) GetNumGroups() int64 {
	return readJSON(d).NumGroups
}

// GetAmountStorage returns the amount of site storage used
func (d *MetricsJSONDriver) GetAmountStorage() int64 {
	return readJSON(d).AmountStorage
}
