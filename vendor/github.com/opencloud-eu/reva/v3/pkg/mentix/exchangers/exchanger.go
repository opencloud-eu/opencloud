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
	"strings"
	"sync"

	"github.com/rs/zerolog"

	"github.com/opencloud-eu/reva/v3/pkg/mentix/config"
	"github.com/opencloud-eu/reva/v3/pkg/mentix/entity"
	"github.com/opencloud-eu/reva/v3/pkg/mentix/meshdata"
)

// Exchanger is the base interface for importers and exporters.
type Exchanger interface {
	entity.Entity

	// Start starts the exchanger; only exchangers which perform periodical background tasks should do something here.
	Start() error
	// Stop stops any running background activities of the exchanger.
	Stop()

	// MeshData returns the mesh data.
	MeshData() *meshdata.MeshData

	// Update is called whenever the mesh data set has changed to reflect these changes.
	Update(meshdata.Map) error
}

// BaseExchanger implements basic exchanger functionality common to all exchangers.
type BaseExchanger struct {
	Exchanger

	conf *config.Configuration
	log  *zerolog.Logger

	enabledConnectors []string

	meshData *meshdata.MeshData

	locker sync.RWMutex
}

// Activate activates the exchanger.
func (exchanger *BaseExchanger) Activate(conf *config.Configuration, log *zerolog.Logger) error {
	if conf == nil {
		return fmt.Errorf("no configuration provided")
	}
	exchanger.conf = conf

	if log == nil {
		return fmt.Errorf("no logger provided")
	}
	exchanger.log = log

	return nil
}

// Start starts the exchanger; only exchangers which perform periodical background tasks should do something here.
func (exchanger *BaseExchanger) Start() error {
	return nil
}

// Stop stops any running background activities of the exchanger.
func (exchanger *BaseExchanger) Stop() {
}

// IsConnectorEnabled checks if the given connector is enabled for the exchanger.
func (exchanger *BaseExchanger) IsConnectorEnabled(id string) bool {
	for _, connectorID := range exchanger.enabledConnectors {
		if connectorID == "*" || strings.EqualFold(connectorID, id) {
			return true
		}
	}
	return false
}

// Update is called whenever the mesh data set has changed to reflect these changes.
func (exchanger *BaseExchanger) Update(meshDataSet meshdata.Map) error {
	// Update the stored mesh data set
	if err := exchanger.storeMeshDataSet(meshDataSet); err != nil {
		return fmt.Errorf("unable to store the mesh data: %v", err)
	}

	return nil
}

func (exchanger *BaseExchanger) storeMeshDataSet(meshDataSet meshdata.Map) error {
	// Store the new mesh data set by cloning it and then merging the cloned data into one object
	meshDataSetCloned := make(meshdata.Map)
	for connectorID, meshData := range meshDataSet {
		if !exchanger.IsConnectorEnabled(connectorID) {
			continue
		}

		meshDataCloned := meshData.Clone()
		if meshDataCloned == nil {
			return fmt.Errorf("unable to clone the mesh data")
		}

		meshDataSetCloned[connectorID] = meshDataCloned
	}
	exchanger.setMeshData(meshdata.MergeMeshDataMap(meshDataSetCloned))

	return nil
}

func (exchanger *BaseExchanger) cloneMeshData() *meshdata.MeshData {
	exchanger.locker.RLock()
	meshDataClone := exchanger.meshData.Clone()
	exchanger.locker.RUnlock()

	return meshDataClone
}

// Config returns the configuration object.
func (exchanger *BaseExchanger) Config() *config.Configuration {
	return exchanger.conf
}

// Log returns the logger object.
func (exchanger *BaseExchanger) Log() *zerolog.Logger {
	return exchanger.log
}

// EnabledConnectors returns the list of all enabled connectors for the exchanger.
func (exchanger *BaseExchanger) EnabledConnectors() []string {
	return exchanger.enabledConnectors
}

// SetEnabledConnectors sets the list of all enabled connectors for the exchanger.
func (exchanger *BaseExchanger) SetEnabledConnectors(connectors []string) {
	exchanger.enabledConnectors = connectors
}

// MeshData returns the stored mesh data. The returned data is cloned to prevent accidental data changes.
// Unauthorized sites are also removed if this exchanger doesn't allow them.
func (exchanger *BaseExchanger) MeshData() *meshdata.MeshData {
	return exchanger.cloneMeshData()
}

func (exchanger *BaseExchanger) setMeshData(meshData *meshdata.MeshData) {
	exchanger.locker.Lock()
	defer exchanger.locker.Unlock()

	exchanger.meshData = meshData
}

// Locker returns the locking object.
func (exchanger *BaseExchanger) Locker() *sync.RWMutex {
	return &exchanger.locker
}
