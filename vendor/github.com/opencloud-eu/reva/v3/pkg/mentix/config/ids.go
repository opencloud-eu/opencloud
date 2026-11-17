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

const (
	// ConnectorIDGOCDB is the connector identifier for GOCDB.
	ConnectorIDGOCDB = "gocdb"
)

const (
	// ExporterIDWebAPI is the identifier for the WebAPI exporter.
	ExporterIDWebAPI = "webapi"
	// ExporterIDCS3API is the identifier for the CS3API exporter.
	ExporterIDCS3API = "cs3api"
	// ExporterIDSiteLocations is the identifier for the Site Locations exporter.
	ExporterIDSiteLocations = "siteloc"
	// ExporterIDPrometheusSD is the identifier for the PrometheusSD exporter.
	ExporterIDPrometheusSD = "promsd"
	// ExporterIDMetrics is the identifier for the Metrics exporter.
	ExporterIDMetrics = "metrics"
)
