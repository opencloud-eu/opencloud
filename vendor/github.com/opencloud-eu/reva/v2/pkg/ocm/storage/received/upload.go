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

package ocm

import (
	"context"
	"net/http"

	"github.com/studio-b12/gowebdav"

	ocmpb "github.com/cs3org/go-cs3apis/cs3/sharing/ocm/v1beta1"
	provider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	ctxpkg "github.com/opencloud-eu/reva/v2/pkg/ctx"
	"github.com/opencloud-eu/reva/v2/pkg/errtypes"
	"github.com/opencloud-eu/reva/v2/pkg/storage"
	"github.com/opencloud-eu/reva/v2/pkg/utils"
)

// MarkProcessing is a no-op: the file lives on the remote instance, so there is no
// local node to flag while postprocessing runs.
func (d *driver) MarkProcessing(ctx context.Context, ref *provider.Reference, processing bool, sessionID string) error {
	return nil
}

// CommitUpload streams the staged bytes to the remote instance.
func (d *driver) CommitUpload(ctx context.Context, ref *provider.Reference, sessionID string, source storage.UploadSource) error {
	client, _, rel, err := d.serviceWebdavClient(ctx, ref)
	if err != nil {
		return err
	}

	client.SetInterceptor(func(method string, rq *http.Request) {
		// Set the content length on the request struct directly instead of the header.
		// The content-length header gets reset by the golang http library before
		// sending out the request, resulting in chunked encoding to be used which
		// breaks the quota checks in ocdav.
		if method == "PUT" {
			rq.ContentLength = source.Length
		}
	})

	// the coordinator owns source.Body and closes it after this returns
	lockToken, _ := ctxpkg.ContextGetLockID(ctx)
	return client.WriteStream(rel, source.Body, 0, lockToken)
}

// serviceWebdavClient builds a webdav client without needing a request auth token:
// it authenticates as the service account and looks up the share on behalf of the
// upload's executant.
func (d *driver) serviceWebdavClient(ctx context.Context, ref *provider.Reference) (*gowebdav.Client, *ocmpb.ReceivedShare, string, error) {
	serviceUserCtx, err := utils.GetServiceUserContextWithContext(context.Background(), d.gateway, d.c.ServiceAccountID, d.c.ServiceAccountSecret)
	if err != nil {
		return nil, nil, "", err
	}
	executant, ok := ctxpkg.ContextGetUser(ctx)
	if !ok {
		return nil, nil, "", errtypes.BadRequest("ocm: no executant in context")
	}
	return d.webdavClient(serviceUserCtx, executant.GetId(), ref)
}
