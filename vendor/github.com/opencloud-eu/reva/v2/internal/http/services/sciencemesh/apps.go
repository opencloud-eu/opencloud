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

package sciencemesh

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	gateway "github.com/cs3org/go-cs3apis/cs3/gateway/v1beta1"
	rpcv1beta1 "github.com/cs3org/go-cs3apis/cs3/rpc/v1beta1"
	ocmpb "github.com/cs3org/go-cs3apis/cs3/sharing/ocm/v1beta1"
	"github.com/opencloud-eu/reva/v2/internal/http/services/reqres"
	"github.com/opencloud-eu/reva/v2/pkg/errtypes"
	"github.com/opencloud-eu/reva/v2/pkg/rgrpc/todo/pool"
	"github.com/opencloud-eu/reva/v2/pkg/rhttp/router"
)

type appsHandler struct {
	gatewaySelector *pool.Selector[gateway.GatewayAPIClient]
	ocmMountPoint   string
}

func (h *appsHandler) init(c *config) error {
	var err error
	h.gatewaySelector, err = pool.GatewaySelector(c.GatewaySvc)
	if err != nil {
		return err
	}
	h.ocmMountPoint = c.OCMMountPoint

	return nil
}

func (h *appsHandler) shareInfo(p string) (*ocmpb.ShareId, string) {
	p = strings.TrimPrefix(p, h.ocmMountPoint)
	shareID, rel := router.ShiftPath(p)
	if len(rel) > 0 {
		rel = rel[1:]
	}
	return &ocmpb.ShareId{OpaqueId: shareID}, rel
}

func (h *appsHandler) OpenInApp(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if err := r.ParseForm(); err != nil {
		reqres.WriteError(w, r, reqres.APIErrorInvalidParameter, "parameters could not be parsed", nil)
		return
	}

	path := r.Form.Get("file")
	if path == "" {
		reqres.WriteError(w, r, reqres.APIErrorInvalidParameter, "missing file", nil)
		return
	}

	shareID, rel := h.shareInfo(path)

	template, err := h.webappTemplate(ctx, shareID)
	if err != nil {
		var e errtypes.NotFound
		if errors.As(err, &e) {
			reqres.WriteError(w, r, reqres.APIErrorNotFound, e.Error(), nil)
		}
		reqres.WriteError(w, r, reqres.APIErrorServerError, err.Error(), err)
		return
	}

	url := resolveTemplate(template, rel)

	if err := json.NewEncoder(w).Encode(map[string]any{
		"app_url": url,
	}); err != nil {
		reqres.WriteError(w, r, reqres.APIErrorServerError, "error marshalling JSON response", err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
}

func (h *appsHandler) webappTemplate(ctx context.Context, id *ocmpb.ShareId) (string, error) {
	gc, err := h.gatewaySelector.Next()
	if err != nil {
		return "", err
	}
	res, err := gc.GetReceivedOCMShare(ctx, &ocmpb.GetReceivedOCMShareRequest{
		Ref: &ocmpb.ShareReference{
			Spec: &ocmpb.ShareReference_Id{
				Id: id,
			},
		},
	})
	if err != nil {
		return "", err
	}
	if res.Status.Code != rpcv1beta1.Code_CODE_OK {
		if res.Status.Code == rpcv1beta1.Code_CODE_NOT_FOUND {
			return "", errtypes.NotFound(res.Status.Message)
		}
		return "", errtypes.InternalError(res.Status.Message)
	}

	webapp, ok := getWebappProtocol(res.Share.Protocols)
	if !ok {
		return "", errtypes.BadRequest("share does not contain webapp protocol")
	}

	return webapp.Uri, nil
}

func getWebappProtocol(protocols []*ocmpb.Protocol) (*ocmpb.WebappProtocol, bool) {
	for _, p := range protocols {
		if t, ok := p.Term.(*ocmpb.Protocol_WebappOptions); ok {
			return t.WebappOptions, true
		}
	}
	return nil, false
}

func resolveTemplate(template string, rel string) string {
	// the template is of type "https://open-cloud-mesh.org/s/share-hash/{relative-path-to-shared-resource}"
	return strings.Replace(template, "{relative-path-to-shared-resource}", rel, 1)
}
