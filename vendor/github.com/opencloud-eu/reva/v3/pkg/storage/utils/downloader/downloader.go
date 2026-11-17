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

package downloader

import (
	"context"
	"fmt"
	"io"
	"net/http"

	gateway "github.com/cs3org/go-cs3apis/cs3/gateway/v1beta1"
	rpc "github.com/cs3org/go-cs3apis/cs3/rpc/v1beta1"
	provider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	"github.com/opencloud-eu/reva/v3/internal/http/services/datagateway"
	"github.com/opencloud-eu/reva/v3/pkg/errtypes"
	"github.com/opencloud-eu/reva/v3/pkg/rgrpc/todo/pool"
	"github.com/opencloud-eu/reva/v3/pkg/rhttp"
)

// Downloader is the interface implemented by the objects that are able to
// download a path into a destination Writer
type Downloader interface {
	Download(context.Context, *provider.ResourceId, io.Writer) error
}

type revaDownloader struct {
	gatewaySelector pool.Selectable[gateway.GatewayAPIClient]
	httpClient      *http.Client
}

// NewDownloader creates a Downloader from the reva gateway
func NewDownloader(gatewaySelector pool.Selectable[gateway.GatewayAPIClient], options ...rhttp.Option) Downloader {
	return &revaDownloader{
		gatewaySelector: gatewaySelector,
		httpClient:      rhttp.GetHTTPClient(options...),
	}
}

func getDownloadProtocol(protocols []*gateway.FileDownloadProtocol, prot string) (*gateway.FileDownloadProtocol, error) {
	for _, p := range protocols {
		if p.Protocol == prot {
			return p, nil
		}
	}
	return nil, errtypes.InternalError(fmt.Sprintf("protocol %s not supported for downloading", prot))
}

// Download downloads a resource given the path to the dst Writer
func (r *revaDownloader) Download(ctx context.Context, id *provider.ResourceId, dst io.Writer) error {
	gatewayClient, err := r.gatewaySelector.Next()
	if err != nil {
		return err
	}
	downResp, err := gatewayClient.InitiateFileDownload(ctx, &provider.InitiateFileDownloadRequest{
		Ref: &provider.Reference{
			ResourceId: id,
			Path:       ".",
		},
	})

	switch {
	case err != nil:
		return err
	case downResp.Status.Code != rpc.Code_CODE_OK:
		return errtypes.InternalError(downResp.Status.Message)
	}

	p, err := getDownloadProtocol(downResp.Protocols, "simple")
	if err != nil {
		p, err = getDownloadProtocol(downResp.Protocols, "spaces")
		if err != nil {
			return err
		}
	}

	httpReq, err := rhttp.NewRequest(ctx, http.MethodGet, p.DownloadEndpoint, nil)
	if err != nil {
		return err
	}
	httpReq.Header.Set(datagateway.TokenTransportHeader, p.Token)

	httpRes, err := r.httpClient.Do(httpReq)
	if err != nil {
		return err
	}
	defer httpRes.Body.Close()

	if err := errtypes.NewErrtypeFromHTTPStatusCode(httpRes.StatusCode, id.String()); err != nil {
		return err
	}

	_, err = io.Copy(dst, httpRes.Body)
	return err
}
