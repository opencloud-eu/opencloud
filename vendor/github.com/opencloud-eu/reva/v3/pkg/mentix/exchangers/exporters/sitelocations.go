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

package exporters

import (
	"github.com/rs/zerolog"

	"github.com/opencloud-eu/reva/v3/pkg/mentix/config"
	"github.com/opencloud-eu/reva/v3/pkg/mentix/exchangers/exporters/siteloc"
)

// SiteLocationsExporter implements the Site Locations exporter to use with Grafana.
type SiteLocationsExporter struct {
	BaseRequestExporter
}

// Activate activates the exporter.
func (exporter *SiteLocationsExporter) Activate(conf *config.Configuration, log *zerolog.Logger) error {
	if err := exporter.BaseRequestExporter.Activate(conf, log); err != nil {
		return err
	}

	// Store SiteLocations specifics
	exporter.SetEndpoint(conf.Exporters.SiteLocations.Endpoint, conf.Exporters.SiteLocations.IsProtected)
	exporter.SetEnabledConnectors(conf.Exporters.SiteLocations.EnabledConnectors)

	exporter.RegisterActionHandler("", siteloc.HandleDefaultQuery)

	return nil
}

// GetID returns the ID of the exporter.
func (exporter *SiteLocationsExporter) GetID() string {
	return config.ExporterIDSiteLocations
}

// GetName returns the display name of the exporter.
func (exporter *SiteLocationsExporter) GetName() string {
	return "Site Locations"
}

func init() {
	registerExporter(&SiteLocationsExporter{})
}
