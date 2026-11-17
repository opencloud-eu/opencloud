// Copyright 2018-2023 CERN
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

package ocmd

import (
	"io"
	"mime"
	"net/http"

	"github.com/opencloud-eu/reva/v3/internal/http/services/reqres"
	"github.com/opencloud-eu/reva/v3/pkg/appctx"
)

// var validate = validator.New()

type notifHandler struct {
}

func (h *notifHandler) init(c *config) error {
	return nil
}

// Notifications dispatches any notifications received from remote OCM sites
// according to the specifications at:
// https://cs3org.github.io/OCM-API/docs.html?branch=v1.1.0&repo=OCM-API&user=cs3org#/paths/~1notifications/post
func (h *notifHandler) Notifications(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := appctx.GetLogger(ctx)
	req, err := getNotification(r)
	if err != nil {
		reqres.WriteError(w, r, reqres.APIErrorInvalidParameter, err.Error(), nil)
		return
	}

	// TODO(lopresti) this is all to be implemented. For now we just log what we got
	log.Debug().Msgf("Received OCM notification: %+v", req)

	// this is to please Nextcloud
	w.WriteHeader(http.StatusCreated)
}

func getNotification(r *http.Request) (string, error) {
	// var req notificationRequest
	contentType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err == nil && contentType == "application/json" {
		bytes, _ := io.ReadAll(r.Body)
		return string(bytes), nil
	}
	return "", nil
}
