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

package connectors

import (
	"github.com/rs/zerolog"

	"github.com/opencloud-eu/reva/v3/pkg/mentix/config"
	"github.com/opencloud-eu/reva/v3/pkg/mentix/entity"
)

var (
	registeredConnectors = entity.NewRegistry()
)

// Collection represents a collection of connectors.
type Collection struct {
	Connectors []Connector
}

// Entities gets the entities in this collection.
func (collection *Collection) Entities() []entity.Entity {
	entities := make([]entity.Entity, 0, len(collection.Connectors))
	for _, connector := range collection.Connectors {
		entities = append(entities, connector)
	}
	return entities
}

// ActivateAll activates all entities in the collection.
func (collection *Collection) ActivateAll(conf *config.Configuration, log *zerolog.Logger) error {
	return entity.ActivateEntities(collection, conf, log)
}

// AvailableConnectors returns a collection of all connectors that are enabled in the configuration.
func AvailableConnectors(conf *config.Configuration) (*Collection, error) {
	entities, err := registeredConnectors.FindEntities(conf.EnabledConnectors, true, true)
	if err != nil {
		return nil, err
	}

	connectors := make([]Connector, 0, len(entities))
	for _, entry := range entities {
		connectors = append(connectors, entry.(Connector))
	}

	return &Collection{Connectors: connectors}, nil
}

func registerConnector(connector Connector) {
	registeredConnectors.Register(connector)
}
