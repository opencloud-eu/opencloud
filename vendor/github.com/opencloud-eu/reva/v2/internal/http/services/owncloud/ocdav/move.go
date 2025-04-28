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
	"fmt"
	"net/http"
	"path"

	rpc "github.com/cs3org/go-cs3apis/cs3/rpc/v1beta1"
	provider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	"github.com/opencloud-eu/reva/v2/internal/http/services/owncloud/ocdav/errors"
	"github.com/opencloud-eu/reva/v2/internal/http/services/owncloud/ocdav/net"
	"github.com/opencloud-eu/reva/v2/internal/http/services/owncloud/ocdav/spacelookup"
	"github.com/opencloud-eu/reva/v2/pkg/appctx"
	"github.com/opencloud-eu/reva/v2/pkg/errtypes"
	rstatus "github.com/opencloud-eu/reva/v2/pkg/rgrpc/status"
	"github.com/opencloud-eu/reva/v2/pkg/rhttp/router"
	"github.com/opencloud-eu/reva/v2/pkg/storagespace"
	"github.com/opencloud-eu/reva/v2/pkg/utils"
	"github.com/rs/zerolog"
)

func (s *svc) handlePathMove(w http.ResponseWriter, r *http.Request, ns string) {
	ctx, span := appctx.GetTracerProvider(r.Context()).Tracer(tracerName).Start(r.Context(), "move")
	defer span.End()

	if !isBodyEmpty(r) {
		w.WriteHeader(http.StatusUnsupportedMediaType)
		b, err := errors.Marshal(http.StatusUnsupportedMediaType, "body must be empty", "", "")
		errors.HandleWebdavError(appctx.GetLogger(ctx), w, b, err)
		return
	}

	srcPath := path.Join(ns, r.URL.Path)
	dh := r.Header.Get(net.HeaderDestination)
	baseURI := r.Context().Value(net.CtxKeyBaseURI).(string)
	dstPath, err := net.ParseDestination(baseURI, dh)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		b, err := errors.Marshal(http.StatusBadRequest, "failed to extract destination", "", "")
		errors.HandleWebdavError(appctx.GetLogger(ctx), w, b, err)
		return
	}

	if err := ValidateName(filename(srcPath), s.nameValidators); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		b, err := errors.Marshal(http.StatusBadRequest, "source failed naming rules", "", "")
		errors.HandleWebdavError(appctx.GetLogger(ctx), w, b, err)
		return
	}

	if err := ValidateDestination(filename(dstPath), s.nameValidators); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		b, err := errors.Marshal(http.StatusBadRequest, "destination naming rules", "", "")
		errors.HandleWebdavError(appctx.GetLogger(ctx), w, b, err)
		return
	}

	dstPath = path.Join(ns, dstPath)

	sublog := appctx.GetLogger(ctx).With().Str("src", srcPath).Str("dst", dstPath).Logger()

	srcSpace, status, err := spacelookup.LookUpStorageSpaceForPath(ctx, s.gatewaySelector, srcPath)
	if err != nil {
		sublog.Error().Err(err).Str("path", srcPath).Msg("failed to look up source storage space")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if status.Code != rpc.Code_CODE_OK {
		errors.HandleErrorStatus(&sublog, w, status)
		return
	}
	dstSpace, status, err := spacelookup.LookUpStorageSpaceForPath(ctx, s.gatewaySelector, dstPath)
	if err != nil {
		sublog.Error().Err(err).Str("path", dstPath).Msg("failed to look up destination storage space")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if status.Code != rpc.Code_CODE_OK {
		errors.HandleErrorStatus(&sublog, w, status)
		return
	}

	s.handleMove(ctx, w, r, spacelookup.MakeRelativeReference(srcSpace, srcPath, false), spacelookup.MakeRelativeReference(dstSpace, dstPath, false), sublog)
}

func (s *svc) handleSpacesMove(w http.ResponseWriter, r *http.Request, srcSpaceID string) {
	ctx, span := appctx.GetTracerProvider(r.Context()).Tracer(tracerName).Start(r.Context(), "spaces_move")
	defer span.End()

	if !isBodyEmpty(r) {
		w.WriteHeader(http.StatusUnsupportedMediaType)
		b, err := errors.Marshal(http.StatusUnsupportedMediaType, "body must be empty", "", "")
		errors.HandleWebdavError(appctx.GetLogger(ctx), w, b, err)
		return
	}

	dh := r.Header.Get(net.HeaderDestination)
	baseURI := r.Context().Value(net.CtxKeyBaseURI).(string)
	dst, err := net.ParseDestination(baseURI, dh)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	sublog := appctx.GetLogger(ctx).With().Str("spaceid", srcSpaceID).Str("path", r.URL.Path).Logger()

	srcRef, err := spacelookup.MakeStorageSpaceReference(srcSpaceID, r.URL.Path)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	dstSpaceID, dstRelPath := router.ShiftPath(dst)

	dstRef, err := spacelookup.MakeStorageSpaceReference(dstSpaceID, dstRelPath)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	s.handleMove(ctx, w, r, &srcRef, &dstRef, sublog)
}

func (s *svc) handleMove(ctx context.Context, w http.ResponseWriter, r *http.Request, src, dst *provider.Reference, log zerolog.Logger) {
	isChild, err := s.referenceIsChildOf(ctx, s.gatewaySelector, dst, src)
	if err != nil {
		switch err.(type) {
		case errtypes.IsNotFound:
			w.WriteHeader(http.StatusNotFound)
		case errtypes.IsNotSupported:
			log.Error().Err(err).Msg("can not detect recursive move operation. missing machine auth configuration?")
			w.WriteHeader(http.StatusForbidden)
		default:
			log.Error().Err(err).Msg("error while trying to detect recursive move operation")
			w.WriteHeader(http.StatusInternalServerError)
		}
		return
	}
	if isChild {
		w.WriteHeader(http.StatusConflict)
		b, err := errors.Marshal(http.StatusBadRequest, "can not move a folder into one of its children", "", "")
		errors.HandleWebdavError(&log, w, b, err)
		return
	}

	isParent, err := s.referenceIsChildOf(ctx, s.gatewaySelector, src, dst)
	if err != nil {
		switch err.(type) {
		case errtypes.IsNotFound:
			isParent = false
		case errtypes.IsNotSupported:
			log.Error().Err(err).Msg("can not detect recursive move operation. missing machine auth configuration?")
			w.WriteHeader(http.StatusForbidden)
			return
		default:
			log.Error().Err(err).Msg("error while trying to detect recursive move operation")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	}
	if isParent {
		w.WriteHeader(http.StatusConflict)
		b, err := errors.Marshal(http.StatusBadRequest, "can not move a folder into its parent", "", "")
		errors.HandleWebdavError(&log, w, b, err)
		return

	}

	oh := r.Header.Get(net.HeaderOverwrite)
	log.Debug().Str("overwrite", oh).Msg("move")

	overwrite, err := net.ParseOverwrite(oh)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	client, err := s.gatewaySelector.Next()
	if err != nil {
		log.Error().Err(err).Msg("error selecting next client")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// check src exists
	srcStatReq := &provider.StatRequest{Ref: src}
	srcStatRes, err := client.Stat(ctx, srcStatReq)
	if err != nil {
		log.Error().Err(err).Msg("error sending grpc stat request")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if srcStatRes.Status.Code != rpc.Code_CODE_OK {
		if srcStatRes.Status.Code == rpc.Code_CODE_NOT_FOUND {
			w.WriteHeader(http.StatusNotFound)
			m := fmt.Sprintf("Resource %v not found", srcStatReq.Ref.Path)
			b, err := errors.Marshal(http.StatusNotFound, m, "", "")
			errors.HandleWebdavError(&log, w, b, err)
		}
		errors.HandleErrorStatus(&log, w, srcStatRes.Status)
		return
	}
	if utils.IsSpaceRoot(srcStatRes.GetInfo()) {
		log.Error().Msg("the source is disallowed")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// check dst exists
	dstStatReq := &provider.StatRequest{Ref: dst}
	dstStatRes, err := client.Stat(ctx, dstStatReq)
	if err != nil {
		log.Error().Err(err).Msg("error sending grpc stat request")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if dstStatRes.Status.Code != rpc.Code_CODE_OK && dstStatRes.Status.Code != rpc.Code_CODE_NOT_FOUND {
		errors.HandleErrorStatus(&log, w, dstStatRes.Status)
		return
	}

	successCode := http.StatusCreated // 201 if new resource was created, see https://tools.ietf.org/html/rfc4918#section-9.9.4
	if dstStatRes.Status.Code == rpc.Code_CODE_OK {
		successCode = http.StatusNoContent // 204 if target already existed, see https://tools.ietf.org/html/rfc4918#section-9.9.4

		if utils.IsSpaceRoot(dstStatRes.GetInfo()) {
			log.Error().Msg("overwriting is not allowed")
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if !overwrite {
			log.Warn().Bool("overwrite", overwrite).Msg("dst already exists")
			w.WriteHeader(http.StatusPreconditionFailed) // 412, see https://tools.ietf.org/html/rfc4918#section-9.9.4
			return
		}
		// delete existing tree
		delReq := &provider.DeleteRequest{Ref: dst}
		delRes, err := client.Delete(ctx, delReq)
		if err != nil {
			log.Error().Err(err).Msg("error sending grpc delete request")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		if delRes.Status.Code != rpc.Code_CODE_OK && delRes.Status.Code != rpc.Code_CODE_NOT_FOUND {
			errors.HandleErrorStatus(&log, w, delRes.Status)
			return
		}
	} else {
		// check if an intermediate path / the parent exists
		intStatReq := &provider.StatRequest{Ref: &provider.Reference{
			ResourceId: dst.ResourceId,
			Path:       utils.MakeRelativePath(path.Dir(dst.Path)),
		}}
		intStatRes, err := client.Stat(ctx, intStatReq)
		if err != nil {
			log.Error().Err(err).Msg("error sending grpc stat request")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		if intStatRes.Status.Code != rpc.Code_CODE_OK {
			if intStatRes.Status.Code == rpc.Code_CODE_NOT_FOUND {
				// 409 if intermediate dir is missing, see https://tools.ietf.org/html/rfc4918#section-9.8.5
				log.Debug().Interface("parent", dst).Interface("status", intStatRes.Status).Msg("conflict")
				w.WriteHeader(http.StatusConflict)
			} else {
				errors.HandleErrorStatus(&log, w, intStatRes.Status)
			}
			return
		}
		// TODO what if intermediate is a file?
	}
	// resolve the destination path
	if dst.Path == "." {
		dst.Path = utils.MakeRelativePath(dstStatRes.GetInfo().GetName())
		dst.ResourceId = dstStatRes.GetInfo().GetParentId()
	}
	mReq := &provider.MoveRequest{
		Source:      src,
		Destination: dst,
		LockId:      requestLockToken(r),
	}
	mRes, err := client.Move(ctx, mReq)
	if err != nil {
		log.Error().Err(err).Msg("error sending move grpc request")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if mRes.Status.Code != rpc.Code_CODE_OK {
		status := rstatus.HTTPStatusFromCode(mRes.Status.Code)
		m := mRes.Status.Message
		switch mRes.Status.Code {
		case rpc.Code_CODE_ABORTED:
			status = http.StatusPreconditionFailed
		case rpc.Code_CODE_PERMISSION_DENIED:
			status = http.StatusForbidden
		case rpc.Code_CODE_UNIMPLEMENTED:
			// We translate this into a Bad Gateway error as per https://www.rfc-editor.org/rfc/rfc4918#section-9.9.4
			// > 502 (Bad Gateway) - This may occur when the destination is on another
			// > server and the destination server refuses to accept the resource.
			// > This could also occur when the destination is on another sub-section
			// > of the same server namespace.
			status = http.StatusBadGateway
		}

		w.WriteHeader(status)

		b, err := errors.Marshal(status, m, "", "")
		errors.HandleWebdavError(&log, w, b, err)
		return
	}

	dstStatRes, err = client.Stat(ctx, dstStatReq)
	if err != nil {
		log.Error().Err(err).Msg("error sending grpc stat request")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if dstStatRes.Status.Code != rpc.Code_CODE_OK {
		errors.HandleErrorStatus(&log, w, dstStatRes.Status)
		return
	}

	info := dstStatRes.Info
	w.Header().Set(net.HeaderContentType, info.MimeType)
	w.Header().Set(net.HeaderETag, info.Etag)
	w.Header().Set(net.HeaderOCFileID, storagespace.FormatResourceID(info.Id))
	w.Header().Set(net.HeaderOCETag, info.Etag)
	w.WriteHeader(successCode)
}
