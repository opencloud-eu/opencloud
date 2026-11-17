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
	"io"
	"net/http"
	"net/url"

	"github.com/rs/zerolog"

	"github.com/opencloud-eu/reva/v3/pkg/mentix/config"
	"github.com/opencloud-eu/reva/v3/pkg/mentix/exchangers"
	"github.com/opencloud-eu/reva/v3/pkg/mentix/meshdata"
)

// BaseRequestImporter implements basic importer functionality common to all request importers.
type BaseRequestImporter struct {
	BaseImporter
	exchangers.BaseRequestExchanger
}

// HandleRequest handles the actual HTTP request.
func (importer *BaseRequestImporter) HandleRequest(resp http.ResponseWriter, req *http.Request, conf *config.Configuration, log *zerolog.Logger) {
	body, _ := io.ReadAll(req.Body)
	meshDataSet, status, respData, err := importer.handleQuery(body, req.URL.Query(), conf, log)
	if err == nil {
		if len(meshDataSet) > 0 {
			importer.mergeImportedMeshDataSet(meshDataSet)
		}
	} else {
		respData = []byte(err.Error())
	}
	resp.WriteHeader(status)
	_, _ = resp.Write(respData)
}

func (importer *BaseRequestImporter) mergeImportedMeshDataSet(meshDataSet meshdata.Vector) {
	// Merge the newly imported data with any existing data stored in the importer
	if importer.meshDataUpdates != nil {
		// Need to manually lock the data for writing
		importer.updatesLocker.Lock()
		defer importer.updatesLocker.Unlock()

		importer.meshDataUpdates = append(importer.meshDataUpdates, meshDataSet...)
	} else {
		importer.setMeshDataUpdates(meshDataSet) // SetMeshData will do the locking itself
	}
}

func (importer *BaseRequestImporter) handleQuery(data []byte, params url.Values, conf *config.Configuration, log *zerolog.Logger) (meshdata.Vector, int, []byte, error) {
	// Data is read, so lock it for writing
	importer.Locker().RLock()
	defer importer.Locker().RUnlock()

	return importer.HandleAction(importer.MeshData(), data, params, true, conf, log)
}
