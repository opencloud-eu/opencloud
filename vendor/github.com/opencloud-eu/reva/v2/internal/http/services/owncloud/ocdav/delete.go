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

package ocdav

import (
	"context"
	"errors"
	"net/http"
	"path"

	rpc "github.com/cs3org/go-cs3apis/cs3/rpc/v1beta1"
	provider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	"github.com/opencloud-eu/reva/v2/internal/http/services/owncloud/ocdav/net"
	"github.com/opencloud-eu/reva/v2/internal/http/services/owncloud/ocdav/spacelookup"
	"github.com/opencloud-eu/reva/v2/pkg/appctx"
	"github.com/opencloud-eu/reva/v2/pkg/errtypes"
	rstatus "github.com/opencloud-eu/reva/v2/pkg/rgrpc/status"
	"github.com/opencloud-eu/reva/v2/pkg/utils"
)

func (s *svc) handlePathDelete(w http.ResponseWriter, r *http.Request, ns string) (status int, err error) {
	ctx := r.Context()
	ctx, span := appctx.GetTracerProvider(r.Context()).Tracer(tracerName).Start(ctx, "path_delete")
	defer span.End()

	if !isBodyEmpty(r) {
		return http.StatusUnsupportedMediaType, errors.New("body must be empty")
	}

	fn := path.Join(ns, r.URL.Path)

	space, rpcStatus, err := spacelookup.LookUpStorageSpaceForPath(ctx, s.gatewaySelector, fn)
	switch {
	case err != nil:
		span.RecordError(err)
		return http.StatusInternalServerError, err
	case rpcStatus.Code != rpc.Code_CODE_OK:
		return rstatus.HTTPStatusFromCode(rpcStatus.Code), errtypes.NewErrtypeFromStatus(rpcStatus)
	}

	return s.handleDelete(ctx, w, r, spacelookup.MakeRelativeReference(space, fn, false))
}

func (s *svc) handleDelete(ctx context.Context, w http.ResponseWriter, r *http.Request, ref *provider.Reference) (status int, err error) {
	ctx, span := appctx.GetTracerProvider(r.Context()).Tracer(tracerName).Start(ctx, "delete")
	defer span.End()

	req := &provider.DeleteRequest{
		Ref:    ref,
		LockId: requestLockToken(r),
	}

	// FIXME the lock token is part of the application level protocol, it should be part of the DeleteRequest message not the opaque
	ih, ok := parseIfHeader(r.Header.Get(net.HeaderIf))
	if ok {
		if len(ih.lists) == 1 && len(ih.lists[0].conditions) == 1 {
			req.Opaque = utils.AppendPlainToOpaque(req.Opaque, "lockid", ih.lists[0].conditions[0].Token)
		}
	} else if r.Header.Get(net.HeaderIf) != "" {
		return http.StatusBadRequest, errtypes.BadRequest("invalid if header")
	}

	client, err := s.gatewaySelector.Next()
	if err != nil {
		return http.StatusInternalServerError, errtypes.InternalError(err.Error())
	}

	res, err := client.Delete(ctx, req)
	switch {
	case err != nil:
		span.RecordError(err)
		return http.StatusInternalServerError, err
	case res.Status.Code == rpc.Code_CODE_OK:
		return http.StatusNoContent, nil
	case res.Status.Code == rpc.Code_CODE_NOT_FOUND:
		//lint:ignore ST1005 mimic the exact oc10 error message
		return http.StatusNotFound, errors.New("Resource not found")
	case res.Status.Code == rpc.Code_CODE_PERMISSION_DENIED:
		status = http.StatusForbidden
		if lockID := utils.ReadPlainFromOpaque(res.Opaque, "lockid"); lockID != "" {
			// http://www.webdav.org/specs/rfc4918.html#HEADER_Lock-Token says that the
			// Lock-Token value is a Coded-URL. We add angle brackets.
			w.Header().Set("Lock-Token", "<"+lockID+">")
			status = http.StatusLocked
		}
		// check if user has access to resource
		sRes, err := client.Stat(ctx, &provider.StatRequest{Ref: ref})
		if err != nil {
			span.RecordError(err)
			return http.StatusInternalServerError, err
		}
		if sRes.Status.Code != rpc.Code_CODE_OK {
			// return not found error so we do not leak existence of a file
			// TODO hide permission failed for users without access in every kind of request
			// TODO should this be done in the driver?
			//lint:ignore ST1005 mimic the exact oc10 error message
			return http.StatusNotFound, errors.New("Resource not found")
		}
		return status, errors.New("") // mimic the oc10 error messages which have an empty message in this case
	case res.Status.Code == rpc.Code_CODE_INTERNAL && res.Status.Message == "can't delete mount path":
		// 405 must generate an Allow header
		w.Header().Set("Allow", "PROPFIND, MOVE, COPY, POST, PROPPATCH, HEAD, GET, OPTIONS, LOCK, UNLOCK, REPORT, SEARCH, PUT")
		return http.StatusMethodNotAllowed, errtypes.PermissionDenied(res.Status.Message)
	}
	return rstatus.HTTPStatusFromCode(res.Status.Code), errtypes.NewErrtypeFromStatus(res.Status)
}

func (s *svc) handleSpacesDelete(w http.ResponseWriter, r *http.Request, spaceID string) (status int, err error) {
	ctx := r.Context()
	ctx, span := appctx.GetTracerProvider(r.Context()).Tracer(tracerName).Start(ctx, "spaces_delete")
	defer span.End()

	if !isBodyEmpty(r) {
		return http.StatusUnsupportedMediaType, errors.New("body must be empty")
	}

	ref, err := spacelookup.MakeStorageSpaceReference(spaceID, r.URL.Path)
	if err != nil {
		return http.StatusBadRequest, err
	}

	// do not allow deleting spaces via dav endpoint - use graph endpoint instead
	// we get a relative reference coming from the space root
	// so if the path is "empty" and no opaque id is present or the opaque id equals
	// the space id, we are referencing the space
	rid := ref.GetResourceId()
	if ref.GetPath() == "." &&
		(rid.GetOpaqueId() == "" || rid.GetOpaqueId() == rid.GetSpaceId()) {
		return http.StatusMethodNotAllowed, errors.New("deleting spaces via dav is not allowed")
	}

	return s.handleDelete(ctx, w, r, &ref)
}
