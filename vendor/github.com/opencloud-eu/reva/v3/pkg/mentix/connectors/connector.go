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
	"fmt"

	"github.com/rs/zerolog"

	"github.com/opencloud-eu/reva/v3/pkg/mentix/config"
	"github.com/opencloud-eu/reva/v3/pkg/mentix/entity"
	"github.com/opencloud-eu/reva/v3/pkg/mentix/meshdata"
)

// Connector is the interface that all connectors must implement.
type Connector interface {
	entity.Entity

	// RetrieveMeshData fetches new mesh data.
	RetrieveMeshData() (*meshdata.MeshData, error)
	// UpdateMeshData updates the provided mesh data on the target side. The provided data only contains the data that
	// should be updated, not the entire data set.
	UpdateMeshData(data *meshdata.MeshData) error
}

// BaseConnector implements basic connector functionality common to all connectors.
type BaseConnector struct {
	conf *config.Configuration
	log  *zerolog.Logger
}

// Activate activates the connector.
func (connector *BaseConnector) Activate(conf *config.Configuration, log *zerolog.Logger) error {
	if conf == nil {
		return fmt.Errorf("no configuration provided")
	}
	connector.conf = conf

	if log == nil {
		return fmt.Errorf("no logger provided")
	}
	connector.log = log

	return nil
}

// UpdateMeshData updates the provided mesh data on the target side. The provided data only contains the data that
// should be updated, not the entire data set.
func (connector *BaseConnector) UpdateMeshData(data *meshdata.MeshData) error {
	return fmt.Errorf("the connector doesn't support updating of mesh data")
}
