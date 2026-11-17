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

package config

// Config holds the config options that need to be passed down to the metrics reader(driver)
type Config struct {
	MetricsDataDriverType string `mapstructure:"metrics_data_driver_type"`
	MetricsDataLocation   string `mapstructure:"metrics_data_location"`
	MetricsRecordInterval int    `mapstructure:"metrics_record_interval"`
	XcloudInstance        string `mapstructure:"xcloud_instance"`
	XcloudPullInterval    int    `mapstructure:"xcloud_pull_interval"`
	InsecureSkipVerify    bool   `mapstructure:"insecure_skip_verify"`
}

// Init sets sane defaults
func (c *Config) Init() {
	if c.MetricsDataDriverType == "json" {
		// default values
		if c.MetricsDataLocation == "" {
			c.MetricsDataLocation = "/var/tmp/reva/metrics/metricsdata.json"
		}
	}

	if c.MetricsRecordInterval == 0 {
		c.MetricsRecordInterval = 5000
	}

	if c.XcloudPullInterval == 0 {
		c.XcloudPullInterval = 5
	}
}
