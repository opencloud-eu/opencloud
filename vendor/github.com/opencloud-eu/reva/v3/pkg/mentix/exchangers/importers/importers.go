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

package importers

import (
	"github.com/rs/zerolog"

	"github.com/opencloud-eu/reva/v3/pkg/mentix/config"
	"github.com/opencloud-eu/reva/v3/pkg/mentix/entity"
	"github.com/opencloud-eu/reva/v3/pkg/mentix/exchangers"
)

// Collection represents a collection of importers.
type Collection struct {
	Importers []Importer
}

var (
	registeredImporters = entity.NewRegistry()
)

// Entities returns a vector of entities within the collection.
func (collection *Collection) Entities() []entity.Entity {
	return exchangers.AsEntityCollection(collection).Entities()
}

// Exchangers returns a vector of exchangers within the collection.
func (collection *Collection) Exchangers() []exchangers.Exchanger {
	exchngrs := make([]exchangers.Exchanger, 0, len(collection.Importers))
	for _, connector := range collection.Importers {
		exchngrs = append(exchngrs, connector)
	}
	return exchngrs
}

// ActivateAll activates all importers.
func (collection *Collection) ActivateAll(conf *config.Configuration, log *zerolog.Logger) error {
	return entity.ActivateEntities(collection, conf, log)
}

// StartAll starts all importers.
func (collection *Collection) StartAll() error {
	return exchangers.StartExchangers(collection)
}

// StopAll stops all importers.
func (collection *Collection) StopAll() {
	exchangers.StopExchangers(collection)
}

// GetRequestImporters returns all importers that implement the RequestExchanger interface.
func (collection *Collection) GetRequestImporters() []exchangers.RequestExchanger {
	return exchangers.GetRequestExchangers(collection)
}

// AvailableImporters returns a collection of all importers that are enabled in the configuration.
func AvailableImporters(conf *config.Configuration) (*Collection, error) {
	// Try to add all importers configured in the environment
	entities, err := registeredImporters.FindEntities(conf.EnabledImporters, true, false)
	if err != nil {
		return nil, err
	}

	importers := make([]Importer, 0, len(entities))
	for _, entry := range entities {
		importers = append(importers, entry.(Importer))
	}

	return &Collection{Importers: importers}, nil
}

// TODO: Uncomment once an importer is actually implemented
/*
func registerImporter(importer Importer) {
	registeredImporters.Register(importer)
}
*/
