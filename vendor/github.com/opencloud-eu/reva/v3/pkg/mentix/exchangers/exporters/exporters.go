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
	"github.com/opencloud-eu/reva/v3/pkg/mentix/entity"
	"github.com/opencloud-eu/reva/v3/pkg/mentix/exchangers"
)

// Collection represents a collection of exporters.
type Collection struct {
	Exporters []Exporter
}

var (
	registeredExporters = entity.NewRegistry()
)

// Entities returns a vector of entities within the collection.
func (collection *Collection) Entities() []entity.Entity {
	return exchangers.AsEntityCollection(collection).Entities()
}

// Exchangers returns a vector of exchangers within the collection.
func (collection *Collection) Exchangers() []exchangers.Exchanger {
	exchngrs := make([]exchangers.Exchanger, 0, len(collection.Exporters))
	for _, connector := range collection.Exporters {
		exchngrs = append(exchngrs, connector)
	}
	return exchngrs
}

// ActivateAll activates all exporters.
func (collection *Collection) ActivateAll(conf *config.Configuration, log *zerolog.Logger) error {
	return entity.ActivateEntities(collection, conf, log)
}

// StartAll starts all exporters.
func (collection *Collection) StartAll() error {
	return exchangers.StartExchangers(collection)
}

// StopAll stops all exporters.
func (collection *Collection) StopAll() {
	exchangers.StopExchangers(collection)
}

// GetRequestExporters returns all exporters that implement the RequestExchanger interface.
func (collection *Collection) GetRequestExporters() []exchangers.RequestExchanger {
	return exchangers.GetRequestExchangers(collection)
}

// AvailableExporters returns a list of all exporters that are enabled in the configuration.
func AvailableExporters(conf *config.Configuration) (*Collection, error) {
	// Try to add all exporters configured in the environment
	entries, err := registeredExporters.FindEntities(conf.EnabledExporters, true, false)
	if err != nil {
		return nil, err
	}

	exporters := make([]Exporter, 0, len(entries))
	for _, entry := range entries {
		exporters = append(exporters, entry.(Exporter))
	}

	return &Collection{Exporters: exporters}, nil
}

func registerExporter(exporter Exporter) {
	registeredExporters.Register(exporter)
}
