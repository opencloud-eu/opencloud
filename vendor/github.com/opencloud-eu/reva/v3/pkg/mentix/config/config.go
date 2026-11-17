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

// Configuration holds the general Mentix configuration.
type Configuration struct {
	Prefix string `mapstructure:"prefix"`

	Connectors struct {
		GOCDB struct {
			Address string `mapstructure:"address"`
			Scope   string `mapstructure:"scope"`
			APIKey  string `mapstructure:"apikey"`
		} `mapstructure:"gocdb"`
	} `mapstructure:"connectors"`

	UpdateInterval string `mapstructure:"update_interval"`

	Services struct {
		CriticalTypes []string `mapstructure:"critical_types"`
	} `mapstructure:"services"`

	Exporters struct {
		WebAPI struct {
			Endpoint          string   `mapstructure:"endpoint"`
			EnabledConnectors []string `mapstructure:"enabled_connectors"`
			IsProtected       bool     `mapstructure:"is_protected"`
		} `mapstructure:"webapi"`

		CS3API struct {
			Endpoint             string   `mapstructure:"endpoint"`
			EnabledConnectors    []string `mapstructure:"enabled_connectors"`
			IsProtected          bool     `mapstructure:"is_protected"`
			ElevatedServiceTypes []string `mapstructure:"elevated_service_types"`
		} `mapstructure:"cs3api"`

		SiteLocations struct {
			Endpoint          string   `mapstructure:"endpoint"`
			EnabledConnectors []string `mapstructure:"enabled_connectors"`
			IsProtected       bool     `mapstructure:"is_protected"`
		} `mapstructure:"siteloc"`

		PrometheusSD struct {
			OutputPath        string   `mapstructure:"output_path"`
			EnabledConnectors []string `mapstructure:"enabled_connectors"`
		} `mapstructure:"promsd"`

		Metrics struct {
			EnabledConnectors []string `mapstructure:"enabled_connectors"`
		} `mapstructure:"metrics"`
	} `mapstructure:"exporters"`

	AccountsService struct {
		URL      string `mapstructure:"url"`
		User     string `mapstructure:"user"`
		Password string `mapstructure:"password"`
	} `mapstructure:"accounts"`

	// Internal settings
	EnabledConnectors []string `mapstructure:"-"`
	EnabledImporters  []string `mapstructure:"-"`
	EnabledExporters  []string `mapstructure:"-"`
}

// Init sets sane defaults.
func (c *Configuration) Init() {
	if c.Prefix == "" {
		c.Prefix = "mentix"
	}
	// TODO(daniel): add default that works out of the box
}
