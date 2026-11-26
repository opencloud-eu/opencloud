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

package entity

import "fmt"

// Registry represents a simple id->entity map.
type Registry struct {
	Entities map[string]Entity
}

// Register registers a new entity.
func (r *Registry) Register(entity Entity) {
	r.Entities[entity.GetID()] = entity
}

// FindEntities returns all entities matching the provided IDs.
// If an entity with a certain ID doesn't exist and mustExist is true, an error is returned.
func (r *Registry) FindEntities(ids []string, mustExist bool, anyRequired bool) ([]Entity, error) {
	var entities []Entity
	for _, id := range ids {
		if entity, ok := r.Entities[id]; ok {
			entities = append(entities, entity)
		} else if mustExist {
			return nil, fmt.Errorf("no entity with ID '%v' registered", id)
		}
	}

	if anyRequired && len(entities) == 0 { // At least one entity must be configured
		return nil, fmt.Errorf("no entities available")
	}

	return entities, nil
}

// NewRegistry returns a new entity registry.
func NewRegistry() *Registry {
	return &Registry{
		Entities: make(map[string]Entity),
	}
}
