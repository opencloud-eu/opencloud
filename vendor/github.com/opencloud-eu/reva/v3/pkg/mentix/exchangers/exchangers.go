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

package exchangers

import (
	"fmt"

	"github.com/opencloud-eu/reva/v3/pkg/mentix/entity"
)

// Collection is an interface for exchanger collections.
type Collection interface {
	entity.Collection

	// Exchangers returns a vector of exchangers within the collection.
	Exchangers() []Exchanger
}

type entityCollectionWrapper struct {
	entities []entity.Entity
}

func (collection *entityCollectionWrapper) Entities() []entity.Entity {
	return collection.entities
}

// AsEntityCollection transforms an exchanger collection into an entity collection.
func AsEntityCollection(collection Collection) entity.Collection {
	wrapper := entityCollectionWrapper{}
	for _, exchanger := range collection.Exchangers() {
		wrapper.entities = append(wrapper.entities, exchanger)
	}
	return &wrapper
}

// StartExchangers starts the given exchangers.
func StartExchangers(collection Collection) error {
	for _, exchanger := range collection.Exchangers() {
		if err := exchanger.Start(); err != nil {
			return fmt.Errorf("unable to start exchanger '%v': %v", exchanger.GetName(), err)
		}
	}

	return nil
}

// StopExchangers stops the given exchangers.
func StopExchangers(collection Collection) {
	for _, exchanger := range collection.Exchangers() {
		exchanger.Stop()
	}
}

// GetRequestExchangers gets all exchangers from a vector that implement the RequestExchanger interface.
func GetRequestExchangers(collection Collection) []RequestExchanger {
	var reqExchangers []RequestExchanger
	for _, exporter := range collection.Exchangers() {
		if reqExchanger, ok := exporter.(RequestExchanger); ok {
			reqExchangers = append(reqExchangers, reqExchanger)
		}
	}
	return reqExchangers
}
