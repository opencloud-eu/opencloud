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
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/opencloud-eu/reva/v3/pkg/mentix/connectors"
	"github.com/opencloud-eu/reva/v3/pkg/mentix/exchangers"
	"github.com/opencloud-eu/reva/v3/pkg/mentix/meshdata"
)

// Importer is the interface that all importers must implement.
type Importer interface {
	exchangers.Exchanger

	// Process is called periodically to perform the actual import; if data has been imported, true is returned.
	Process(*connectors.Collection) (bool, error)
}

// BaseImporter implements basic importer functionality common to all importers.
type BaseImporter struct {
	exchangers.BaseExchanger

	meshDataUpdates meshdata.Vector

	updatesLocker sync.RWMutex
}

// Process is called periodically to perform the actual import; if data has been imported, true is returned.
func (importer *BaseImporter) Process(connectors *connectors.Collection) (bool, error) {
	if importer.meshDataUpdates == nil { // No data present for updating, so nothing to process
		return false, nil
	}

	var processErrs []string

	// Data is read, so lock it for writing during the loop
	importer.updatesLocker.RLock()
	for _, connector := range connectors.Connectors {
		if !importer.IsConnectorEnabled(connector.GetID()) {
			continue
		}

		if err := importer.processMeshDataUpdates(connector); err != nil {
			processErrs = append(processErrs, fmt.Sprintf("unable to process imported mesh data for connector '%v': %v", connector.GetName(), err))
		}
	}
	importer.updatesLocker.RUnlock()

	importer.setMeshDataUpdates(nil)

	var err error
	if len(processErrs) != 0 {
		err = errors.New(strings.Join(processErrs, "; "))
	}
	return true, err
}

func (importer *BaseImporter) processMeshDataUpdates(connector connectors.Connector) error {
	for _, meshData := range importer.meshDataUpdates {
		if err := connector.UpdateMeshData(meshData); err != nil {
			return fmt.Errorf("error while updating mesh data: %v", err)
		}
	}

	return nil
}

func (importer *BaseImporter) setMeshDataUpdates(meshDataUpdates meshdata.Vector) {
	importer.updatesLocker.Lock()
	defer importer.updatesLocker.Unlock()

	importer.meshDataUpdates = meshDataUpdates
}
