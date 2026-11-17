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

package exporters

import (
	"github.com/opencloud-eu/reva/v3/pkg/mentix/config"
	"github.com/opencloud-eu/reva/v3/pkg/mentix/exchangers/exporters/metrics"
	"github.com/opencloud-eu/reva/v3/pkg/mentix/meshdata"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
)

// MetricsExporter exposes various Prometheus metrics.
type MetricsExporter struct {
	BaseExporter

	metrics *metrics.Metrics
}

// Activate activates the exporter.
func (exporter *MetricsExporter) Activate(conf *config.Configuration, log *zerolog.Logger) error {
	if err := exporter.BaseExporter.Activate(conf, log); err != nil {
		return err
	}

	// Create the metrics handler
	m, err := metrics.New(conf, log)
	if err != nil {
		return errors.Wrap(err, "unable to create metrics")
	}
	exporter.metrics = m

	// Store Metrics specifics
	exporter.SetEnabledConnectors(conf.Exporters.Metrics.EnabledConnectors)

	return nil
}

// Update is called whenever the mesh data set has changed to reflect these changes.
func (exporter *MetricsExporter) Update(meshDataSet meshdata.Map) error {
	if err := exporter.BaseExporter.Update(meshDataSet); err != nil {
		return err
	}

	// Data is read, so acquire a read lock
	exporter.Locker().RLock()
	defer exporter.Locker().RUnlock()

	if err := exporter.metrics.Update(exporter.MeshData()); err != nil {
		return errors.Wrap(err, "error while updating the metrics")
	}

	return nil
}

// GetID returns the ID of the exporter.
func (exporter *MetricsExporter) GetID() string {
	return config.ExporterIDMetrics
}

// GetName returns the display name of the exporter.
func (exporter *MetricsExporter) GetName() string {
	return "Metrics"
}

func init() {
	registerExporter(&MetricsExporter{})
}
