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

// Package datatx provides a library to abstract the complexity
// of using various data transfer protocols.
package datatx

import (
	"context"
	"net/http"

	userv1beta1 "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"
	provider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	"github.com/opencloud-eu/reva/v2/pkg/events"
	"github.com/opencloud-eu/reva/v2/pkg/storage"
	"github.com/opencloud-eu/reva/v2/pkg/upload"
	"github.com/opencloud-eu/reva/v2/pkg/utils"
)

// DataTX provides an abstraction around various data transfer protocols.
type DataTX interface {
	// Handler serves the protocol's data path. Uploads go through coord, which
	// owns the upload lifecycle for every driver; downloads read from driver.
	Handler(coord upload.Coordinator, driver storage.FS) (http.Handler, error)
}

// EmitFileUploadedEvent is a helper function which publishes a FileUploaded event
func EmitFileUploadedEvent(spaceOwnerOrManager, executant *userv1beta1.UserId, ref *provider.Reference, publisher events.Publisher) error {
	if ref == nil || publisher == nil {
		return nil
	}

	uploadedEv := events.FileUploaded{
		SpaceOwner: spaceOwnerOrManager,
		Owner:      spaceOwnerOrManager,
		Executant:  executant,
		Ref:        ref,
		Timestamp:  utils.TSNow(),
	}

	return events.Publish(context.Background(), publisher, uploadedEv)
}
