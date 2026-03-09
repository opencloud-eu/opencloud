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
	"io"
	"net/http"
	"path"
	"path/filepath"
	"strconv"

	gateway "github.com/cs3org/go-cs3apis/cs3/gateway/v1beta1"
	rpc "github.com/cs3org/go-cs3apis/cs3/rpc/v1beta1"
	provider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	typespb "github.com/cs3org/go-cs3apis/cs3/types/v1beta1"
	"github.com/opencloud-eu/reva/v2/internal/http/services/datagateway"
	"github.com/opencloud-eu/reva/v2/internal/http/services/owncloud/ocdav/errors"
	"github.com/opencloud-eu/reva/v2/internal/http/services/owncloud/ocdav/net"
	"github.com/opencloud-eu/reva/v2/internal/http/services/owncloud/ocdav/spacelookup"
	"github.com/opencloud-eu/reva/v2/pkg/appctx"
	"github.com/opencloud-eu/reva/v2/pkg/errtypes"
	"github.com/opencloud-eu/reva/v2/pkg/rgrpc/todo/pool"
	"github.com/opencloud-eu/reva/v2/pkg/rhttp"
	"github.com/opencloud-eu/reva/v2/pkg/rhttp/router"
	"github.com/opencloud-eu/reva/v2/pkg/storagespace"
	"github.com/opencloud-eu/reva/v2/pkg/utils"
	"github.com/rs/zerolog"
)

type copy struct {
	source      *provider.Reference
	sourceInfo  *provider.ResourceInfo
	destination *provider.Reference
	depth       net.Depth
	successCode int
}

func (s *svc) handlePathCopy(w http.ResponseWriter, r *http.Request, ns string) {
	ctx, span := appctx.GetTracerProvider(r.Context()).Tracer(tracerName).Start(r.Context(), "copy")
	defer span.End()

	if !isBodyEmpty(r) {
		w.WriteHeader(http.StatusUnsupportedMediaType)
		b, err := errors.Marshal(http.StatusUnsupportedMediaType, "body must be empty", "", "")
		errors.HandleWebdavError(appctx.GetLogger(ctx), w, b, err)
		return
	}

	if s.c.EnableHTTPTpc {
		if r.Header.Get("Source") != "" {
			// HTTP Third-Party Copy Pull mode
			s.handleTPCPull(ctx, w, r, ns)
			return
		} else if r.Header.Get("Destination") != "" {
			// HTTP Third-Party Copy Push mode
			s.handleTPCPush(ctx, w, r, ns)
			return
		}
	}

	// Local copy: in this case Destination is mandatory
	src := path.Join(ns, r.URL.Path)

	dh := r.Header.Get(net.HeaderDestination)
	baseURI := r.Context().Value(net.CtxKeyBaseURI).(string)
	dst, err := net.ParseDestination(baseURI, dh)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		b, err := errors.Marshal(http.StatusBadRequest, "failed to extract destination", "", "")
		errors.HandleWebdavError(appctx.GetLogger(ctx), w, b, err)
		return
	}

	if err := ValidateName(filename(src), s.nameValidators); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		b, err := errors.Marshal(http.StatusBadRequest, "source failed naming rules", "", "")
		errors.HandleWebdavError(appctx.GetLogger(ctx), w, b, err)
		return
	}

	if err := ValidateDestination(filename(dst), s.nameValidators); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		b, err := errors.Marshal(http.StatusBadRequest, "destination failed naming rules", "", "")
		errors.HandleWebdavError(appctx.GetLogger(ctx), w, b, err)
		return
	}

	dst = path.Join(ns, dst)

	sublog := appctx.GetLogger(ctx).With().Str("src", src).Str("dst", dst).Logger()

	srcSpace, status, err := spacelookup.LookUpStorageSpaceForPath(ctx, s.gatewaySelector, src)
	if err != nil {
		sublog.Error().Err(err).Str("path", src).Msg("failed to look up storage space")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if status.Code != rpc.Code_CODE_OK {
		errors.HandleErrorStatus(&sublog, w, status)
		return
	}
	dstSpace, status, err := spacelookup.LookUpStorageSpaceForPath(ctx, s.gatewaySelector, dst)
	if err != nil {
		sublog.Error().Err(err).Str("path", dst).Msg("failed to look up storage space")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if status.Code != rpc.Code_CODE_OK {
		errors.HandleErrorStatus(&sublog, w, status)
		return
	}

	cp := s.prepareCopy(ctx, w, r, spacelookup.MakeRelativeReference(srcSpace, src, false), spacelookup.MakeRelativeReference(dstSpace, dst, false), &sublog, dstSpace.GetRoot().GetStorageId() == utils.ShareStorageProviderID)
	if cp == nil {
		return
	}

	if err := s.executePathCopy(ctx, s.gatewaySelector, w, r, cp); err != nil {
		sublog.Error().Err(err).Str("depth", cp.depth.String()).Msg("error executing path copy")
		w.WriteHeader(http.StatusInternalServerError)
	}
	w.WriteHeader(cp.successCode)
}

func (s *svc) executePathCopy(ctx context.Context, selector pool.Selectable[gateway.GatewayAPIClient], w http.ResponseWriter, r *http.Request, cp *copy) error {
	log := appctx.GetLogger(ctx)
	log.Debug().Str("src", cp.sourceInfo.Path).Str("dst", cp.destination.Path).Msg("descending")

	client, err := selector.Next()
	if err != nil {
		return err
	}

	var fileid string
	if cp.sourceInfo.Type == provider.ResourceType_RESOURCE_TYPE_CONTAINER {
		// create dir
		createReq := &provider.CreateContainerRequest{
			Ref: cp.destination,
		}
		createRes, err := client.CreateContainer(ctx, createReq)
		if err != nil {
			log.Error().Err(err).Msg("error performing create container grpc request")
			return err
		}
		if createRes.Status.Code != rpc.Code_CODE_OK {
			if createRes.Status.Code == rpc.Code_CODE_PERMISSION_DENIED {
				w.WriteHeader(http.StatusForbidden)
				m := fmt.Sprintf("Permission denied to create %v", createReq.Ref.Path)
				b, err := errors.Marshal(http.StatusForbidden, m, "", "")
				errors.HandleWebdavError(log, w, b, err)
			}
			return nil
		}

		// TODO: also copy properties: https://tools.ietf.org/html/rfc4918#section-9.8.2

		if cp.depth != net.DepthInfinity {
			return nil
		}

		// descend for children
		listReq := &provider.ListContainerRequest{
			Ref: cp.source,
		}
		res, err := client.ListContainer(ctx, listReq)
		if err != nil {
			return err
		}
		if res.Status.Code != rpc.Code_CODE_OK {
			w.WriteHeader(http.StatusInternalServerError)
			return nil
		}

		for i := range res.Infos {
			child := filepath.Base(res.Infos[i].Path)
			src := &provider.Reference{
				ResourceId: cp.source.ResourceId,
				Path:       utils.MakeRelativePath(filepath.Join(cp.source.Path, child)),
			}
			childDst := &provider.Reference{
				ResourceId: cp.destination.ResourceId,
				Path:       utils.MakeRelativePath(filepath.Join(cp.destination.Path, child)),
			}
			err := s.executePathCopy(ctx, selector, w, r, &copy{source: src, sourceInfo: res.Infos[i], destination: childDst, depth: cp.depth, successCode: cp.successCode})
			if err != nil {
				return err
			}
		}

		// we need to stat again to get the fileid
		r, err := client.Stat(ctx, &provider.StatRequest{Ref: cp.destination})
		if err != nil {
			return err
		}

		if r.GetStatus().GetCode() != rpc.Code_CODE_OK {
			return fmt.Errorf("status code %d", r.GetStatus().GetCode())
		}

		fileid = storagespace.FormatResourceID(r.GetInfo().GetId())
	} else {
		// copy file

		// 1. get download url

		dReq := &provider.InitiateFileDownloadRequest{
			Ref: cp.source,
		}

		dRes, err := client.InitiateFileDownload(ctx, dReq)
		if err != nil {
			return err
		}

		if dRes.Status.Code != rpc.Code_CODE_OK {
			return fmt.Errorf("status code %d", dRes.Status.Code)
		}

		var downloadEP, downloadToken string
		for _, p := range dRes.Protocols {
			if p.Protocol == "spaces" {
				downloadEP, downloadToken = p.DownloadEndpoint, p.Token
			}
		}

		// 2. get upload url

		uReq := &provider.InitiateFileUploadRequest{
			Ref: cp.destination,
			Opaque: &typespb.Opaque{
				Map: map[string]*typespb.OpaqueEntry{
					net.HeaderUploadLength: {
						Decoder: "plain",
						// TODO: handle case where size is not known in advance
						Value: []byte(strconv.FormatUint(cp.sourceInfo.GetSize(), 10)),
					},
					net.HeaderOCMtime: {
						Decoder: "plain",
						Value: []byte(
							utils.TimeToOCMtime(utils.TSToTime(cp.sourceInfo.GetMtime())),
						),
					},
				},
			},
		}

		uRes, err := client.InitiateFileUpload(ctx, uReq)
		if err != nil {
			return err
		}

		if uRes.Status.Code != rpc.Code_CODE_OK {
			if uRes.Status.Code == rpc.Code_CODE_PERMISSION_DENIED {
				w.WriteHeader(http.StatusForbidden)
				m := fmt.Sprintf("Permissions denied to create %v", uReq.Ref.Path)
				b, err := errors.Marshal(http.StatusForbidden, m, "", "")
				errors.HandleWebdavError(log, w, b, err)
				return nil
			}
			errors.HandleErrorStatus(log, w, uRes.Status)
			return nil
		}

		var uploadEP, uploadToken string
		for _, p := range uRes.Protocols {
			if p.Protocol == "tus" {
				uploadEP, uploadToken = p.UploadEndpoint, p.Token
			}
		}

		// 3. do download

		httpDownloadReq, err := rhttp.NewRequest(ctx, "GET", downloadEP, nil)
		if err != nil {
			return err
		}
		httpDownloadReq.Header.Set(datagateway.TokenTransportHeader, downloadToken)

		httpDownloadRes, err := s.client.Do(httpDownloadReq)
		if err != nil {
			return err
		}
		defer httpDownloadRes.Body.Close()
		if httpDownloadRes.StatusCode == http.StatusForbidden {
			w.WriteHeader(http.StatusForbidden)
			b, err := errors.Marshal(http.StatusForbidden, http.StatusText(http.StatusForbidden), "", strconv.Itoa(http.StatusForbidden))
			errors.HandleWebdavError(log, w, b, err)
			return nil
		}
		if httpDownloadRes.StatusCode != http.StatusOK {
			return fmt.Errorf("status code %d", httpDownloadRes.StatusCode)
		}

		// 4. do upload
		fileid, err = s.tusUpload(ctx, uploadEP, uploadToken, httpDownloadRes.Body, int64(cp.sourceInfo.GetSize()))
		if err != nil {
			return err
		}
	}

	w.Header().Set(net.HeaderOCFileID, fileid)
	return nil
}

func (s *svc) handleSpacesCopy(w http.ResponseWriter, r *http.Request, spaceID string) {
	ctx, span := appctx.GetTracerProvider(r.Context()).Tracer(tracerName).Start(r.Context(), "spaces_copy")
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

	sublog := appctx.GetLogger(ctx).With().Str("spaceid", spaceID).Str("path", r.URL.Path).Str("destination", dst).Logger()

	srcRef, err := spacelookup.MakeStorageSpaceReference(spaceID, r.URL.Path)
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

	cp := s.prepareCopy(ctx, w, r, &srcRef, &dstRef, &sublog, dstRef.GetResourceId().GetStorageId() == utils.ShareStorageProviderID)
	if cp == nil {
		return
	}

	err = s.executeSpacesCopy(ctx, w, s.gatewaySelector, cp)
	if err != nil {
		sublog.Error().Err(err).Str("depth", cp.depth.String()).Msg("error descending directory")
		w.WriteHeader(http.StatusInternalServerError)
	}
	w.WriteHeader(cp.successCode)
}

func (s *svc) executeSpacesCopy(ctx context.Context, w http.ResponseWriter, selector pool.Selectable[gateway.GatewayAPIClient], cp *copy) error {
	log := appctx.GetLogger(ctx)
	log.Debug().Interface("src", cp.sourceInfo).Interface("dst", cp.destination).Msg("descending")

	client, err := selector.Next()
	if err != nil {
		return err
	}

	var fileid string
	if cp.sourceInfo.Type == provider.ResourceType_RESOURCE_TYPE_CONTAINER {
		// create dir
		createReq := &provider.CreateContainerRequest{
			Ref: cp.destination,
		}
		createRes, err := client.CreateContainer(ctx, createReq)
		if err != nil {
			log.Error().Err(err).Msg("error performing create container grpc request")
			return err
		}
		if createRes.Status.Code != rpc.Code_CODE_OK {
			if createRes.Status.Code == rpc.Code_CODE_PERMISSION_DENIED {
				w.WriteHeader(http.StatusForbidden)
				// TODO path could be empty or relative...
				m := fmt.Sprintf("Permission denied to create %v", createReq.Ref.Path)
				b, err := errors.Marshal(http.StatusForbidden, m, "", "")
				errors.HandleWebdavError(log, w, b, err)
			}
			return nil
		}

		// TODO: also copy properties: https://tools.ietf.org/html/rfc4918#section-9.8.2

		if cp.depth != net.DepthInfinity {
			return nil
		}

		// descend for children
		listReq := &provider.ListContainerRequest{Ref: &provider.Reference{ResourceId: cp.sourceInfo.Id, Path: "."}}
		res, err := client.ListContainer(ctx, listReq)
		if err != nil {
			return err
		}
		if res.Status.Code != rpc.Code_CODE_OK {
			w.WriteHeader(http.StatusInternalServerError)
			return nil
		}

		for i := range res.Infos {
			childRef := &provider.Reference{
				ResourceId: cp.destination.ResourceId,
				Path:       utils.MakeRelativePath(path.Join(cp.destination.Path, res.Infos[i].Path)),
			}
			err := s.executeSpacesCopy(ctx, w, selector, &copy{sourceInfo: res.Infos[i], destination: childRef, depth: cp.depth, successCode: cp.successCode})
			if err != nil {
				return err
			}
		}

		// we need to stat again to get the fileid
		r, err := client.Stat(ctx, &provider.StatRequest{Ref: cp.destination})
		if err != nil {
			return err
		}

		if r.GetStatus().GetCode() != rpc.Code_CODE_OK {
			return fmt.Errorf("stat: status code %d", r.GetStatus().GetCode())
		}

		fileid = storagespace.FormatResourceID(r.GetInfo().GetId())
	} else {
		// copy file
		// 1. get download url
		dReq := &provider.InitiateFileDownloadRequest{Ref: &provider.Reference{ResourceId: cp.sourceInfo.Id, Path: "."}}
		dRes, err := client.InitiateFileDownload(ctx, dReq)
		if err != nil {
			return err
		}

		if dRes.Status.Code != rpc.Code_CODE_OK {
			return fmt.Errorf("status code %d", dRes.Status.Code)
		}

		var downloadEP, downloadToken string
		for _, p := range dRes.Protocols {
			if p.Protocol == "spaces" {
				downloadEP, downloadToken = p.DownloadEndpoint, p.Token
			}
		}
		// 2. get upload url
		uReq := &provider.InitiateFileUploadRequest{
			Ref: cp.destination,
			Opaque: &typespb.Opaque{
				Map: map[string]*typespb.OpaqueEntry{
					net.HeaderUploadLength: {
						Decoder: "plain",
						// TODO: handle case where size is not known in advance
						Value: []byte(strconv.FormatUint(cp.sourceInfo.GetSize(), 10)),
					},
					net.HeaderOCMtime: {
						Decoder: "plain",
						Value: []byte(
							utils.TimeToOCMtime(utils.TSToTime(cp.sourceInfo.GetMtime())),
						),
					},
				},
			},
		}

		uRes, err := client.InitiateFileUpload(ctx, uReq)
		if err != nil {
			return err
		}

		if uRes.Status.Code != rpc.Code_CODE_OK {
			if uRes.Status.Code == rpc.Code_CODE_PERMISSION_DENIED {
				w.WriteHeader(http.StatusForbidden)
				// TODO path can be empty or relative
				m := fmt.Sprintf("Permissions denied to create %v", uReq.Ref.Path)
				b, err := errors.Marshal(http.StatusForbidden, m, "", "")
				errors.HandleWebdavError(log, w, b, err)
				return nil
			}
			errors.HandleErrorStatus(log, w, uRes.Status)
			return nil
		}

		var uploadEP, uploadToken string
		for _, p := range uRes.Protocols {
			if p.Protocol == "tus" {
				uploadEP, uploadToken = p.UploadEndpoint, p.Token
			}
		}

		// 3. do download
		httpDownloadReq, err := rhttp.NewRequest(ctx, http.MethodGet, downloadEP, nil)
		if err != nil {
			return err
		}
		if downloadToken != "" {
			httpDownloadReq.Header.Set(datagateway.TokenTransportHeader, downloadToken)
		}

		httpDownloadRes, err := s.client.Do(httpDownloadReq)
		if err != nil {
			return err
		}
		defer httpDownloadRes.Body.Close()
		switch httpDownloadRes.StatusCode {
		case http.StatusForbidden, http.StatusTooEarly:
			w.WriteHeader(httpDownloadRes.StatusCode)
			b, err := errors.Marshal(http.StatusForbidden, http.StatusText(httpDownloadRes.StatusCode), "", strconv.Itoa(httpDownloadRes.StatusCode))
			errors.HandleWebdavError(log, w, b, err)
			return nil
		case http.StatusOK:
			// ok
		default:
			return fmt.Errorf("status code %d", httpDownloadRes.StatusCode)
		}

		// 4. do upload
		fileid, err = s.tusUpload(ctx, uploadEP, uploadToken, httpDownloadRes.Body, int64(cp.sourceInfo.GetSize()))
		if err != nil {
			return err
		}
	}

	w.Header().Set(net.HeaderOCFileID, fileid)
	return nil
}

func (s *svc) prepareCopy(ctx context.Context, w http.ResponseWriter, r *http.Request, srcRef, dstRef *provider.Reference, log *zerolog.Logger, destInShareJail bool) *copy {
	isChild, err := s.referenceIsChildOf(ctx, s.gatewaySelector, dstRef, srcRef)
	if err != nil {
		switch err.(type) {
		case errtypes.IsNotSupported:
			log.Error().Err(err).Msg("can not detect recursive copy operation. missing machine auth configuration?")
			w.WriteHeader(http.StatusForbidden)
		default:
			log.Error().Err(err).Msg("error while trying to detect recursive copy operation")
			w.WriteHeader(http.StatusInternalServerError)
		}
	}
	if isChild {
		w.WriteHeader(http.StatusConflict)
		b, err := errors.Marshal(http.StatusBadRequest, "can not copy a folder into one of its children", "", "")
		errors.HandleWebdavError(log, w, b, err)
		return nil
	}

	isParent, err := s.referenceIsChildOf(ctx, s.gatewaySelector, srcRef, dstRef)
	if err != nil {
		switch err.(type) {
		case errtypes.IsNotFound:
			isParent = false
		case errtypes.IsNotSupported:
			log.Error().Err(err).Msg("can not detect recursive copy operation. missing machine auth configuration?")
			w.WriteHeader(http.StatusForbidden)
			return nil
		default:
			log.Error().Err(err).Msg("error while trying to detect recursive copy operation")
			w.WriteHeader(http.StatusInternalServerError)
			return nil
		}
	}

	if isParent {
		w.WriteHeader(http.StatusConflict)
		b, err := errors.Marshal(http.StatusBadRequest, "can not copy a folder into its parent", "", "")
		errors.HandleWebdavError(log, w, b, err)
		return nil

	}

	if srcRef.Path == dstRef.Path && srcRef.ResourceId == dstRef.ResourceId {
		w.WriteHeader(http.StatusConflict)
		b, err := errors.Marshal(http.StatusBadRequest, "source and destination are the same", "", "")
		errors.HandleWebdavError(log, w, b, err)
		return nil
	}

	oh := r.Header.Get(net.HeaderOverwrite)
	overwrite, err := net.ParseOverwrite(oh)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		m := fmt.Sprintf("Overwrite header is set to incorrect value %v", overwrite)
		b, err := errors.Marshal(http.StatusBadRequest, m, "", "")
		errors.HandleWebdavError(log, w, b, err)
		return nil
	}
	dh := r.Header.Get(net.HeaderDepth)
	depth, err := net.ParseDepth(dh)

	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		m := fmt.Sprintf("Depth header is set to incorrect value %v", dh)
		b, err := errors.Marshal(http.StatusBadRequest, m, "", "")
		errors.HandleWebdavError(log, w, b, err)
		return nil
	}
	if dh == "" {
		// net.ParseDepth returns "1" for an empty value but copy expects "infinity"
		// so we overwrite it here
		depth = net.DepthInfinity
	}

	log.Debug().Bool("overwrite", overwrite).Str("depth", depth.String()).Msg("copy")

	client, err := s.gatewaySelector.Next()
	if err != nil {
		log.Error().Err(err).Msg("error selecting next client")
		w.WriteHeader(http.StatusInternalServerError)
		return nil
	}

	srcStatReq := &provider.StatRequest{Ref: srcRef}
	srcStatRes, err := client.Stat(ctx, srcStatReq)
	switch {
	case err != nil:
		log.Error().Err(err).Msg("error sending grpc stat request")
		w.WriteHeader(http.StatusInternalServerError)
		return nil
	case srcStatRes.Status.Code == rpc.Code_CODE_NOT_FOUND:
		errors.HandleErrorStatus(log, w, srcStatRes.Status)
		m := fmt.Sprintf("Resource %v not found", srcStatReq.Ref.Path)
		b, err := errors.Marshal(http.StatusNotFound, m, "", "")
		errors.HandleWebdavError(log, w, b, err)
		return nil
	case srcStatRes.Status.Code != rpc.Code_CODE_OK:
		errors.HandleErrorStatus(log, w, srcStatRes.Status)
		return nil
	}
	if utils.IsSpaceRoot(srcStatRes.GetInfo()) {
		log.Error().Msg("the source is disallowed")
		w.WriteHeader(http.StatusBadRequest)
		return nil
	}

	dstStatReq := &provider.StatRequest{Ref: dstRef}
	dstStatRes, err := client.Stat(ctx, dstStatReq)
	switch {
	case err != nil:
		log.Error().Err(err).Msg("error sending grpc stat request")
		w.WriteHeader(http.StatusInternalServerError)
		return nil
	case dstStatRes.Status.Code != rpc.Code_CODE_OK && dstStatRes.Status.Code != rpc.Code_CODE_NOT_FOUND:
		errors.HandleErrorStatus(log, w, dstStatRes.Status)
		return nil
	}

	successCode := http.StatusCreated // 201 if new resource was created, see https://tools.ietf.org/html/rfc4918#section-9.8.5
	if dstStatRes.Status.Code == rpc.Code_CODE_OK {
		successCode = http.StatusNoContent // 204 if target already existed, see https://tools.ietf.org/html/rfc4918#section-9.8.5

		if !overwrite {
			log.Warn().Bool("overwrite", overwrite).Msg("dst already exists")
			w.WriteHeader(http.StatusPreconditionFailed)
			m := fmt.Sprintf("Could not overwrite Resource %v", dstRef.Path)
			b, err := errors.Marshal(http.StatusPreconditionFailed, m, "", "")
			errors.HandleWebdavError(log, w, b, err) // 412, see https://tools.ietf.org/html/rfc4918#section-9.8.5
			return nil
		}

		if utils.IsSpaceRoot(dstStatRes.GetInfo()) {
			log.Error().Msg("overwriting is not allowed")
			w.WriteHeader(http.StatusBadRequest)
			return nil
		}

		// delete existing tree when overwriting a directory or replacing a file with a directory
		if dstStatRes.Info.Type == provider.ResourceType_RESOURCE_TYPE_CONTAINER ||
			(dstStatRes.Info.Type == provider.ResourceType_RESOURCE_TYPE_FILE &&
				srcStatRes.Info.Type == provider.ResourceType_RESOURCE_TYPE_CONTAINER) {

			// we must not allow to override mountpoints - so we check if we have access to the parent. If not this is a mountpoint
			if destInShareJail {
				res, err := client.GetPath(ctx, &provider.GetPathRequest{ResourceId: dstStatRes.GetInfo().GetId()})
				if err != nil || res.GetStatus().GetCode() != rpc.Code_CODE_OK {
					log.Error().Err(err).Msg("error sending grpc get path request")
					w.WriteHeader(http.StatusInternalServerError)
					return nil
				}

				dir, file := filepath.Split(filepath.Clean(res.GetPath()))
				if dir == "/" || dir == "" || file == "" {
					log.Error().Msg("must not overwrite mount points")
					w.WriteHeader(http.StatusBadRequest)
					_, _ = w.Write([]byte("must not overwrite mount points"))
					return nil
				}
			}

			delReq := &provider.DeleteRequest{Ref: dstRef}
			delRes, err := client.Delete(ctx, delReq)
			if err != nil {
				log.Error().Err(err).Msg("error sending grpc delete request")
				w.WriteHeader(http.StatusInternalServerError)
				return nil
			}

			if delRes.Status.Code != rpc.Code_CODE_OK && delRes.Status.Code != rpc.Code_CODE_NOT_FOUND {
				errors.HandleErrorStatus(log, w, delRes.Status)
				return nil
			}
		}
	} else if p := path.Dir(dstRef.Path); p != "" {
		// check if an intermediate path / the parent exists
		pRef := &provider.Reference{
			ResourceId: dstRef.ResourceId,
			Path:       utils.MakeRelativePath(p),
		}
		intStatReq := &provider.StatRequest{Ref: pRef}
		intStatRes, err := client.Stat(ctx, intStatReq)
		if err != nil {
			log.Error().Err(err).Msg("error sending grpc stat request")
			w.WriteHeader(http.StatusInternalServerError)
			return nil
		}
		if intStatRes.Status.Code != rpc.Code_CODE_OK {
			if intStatRes.Status.Code == rpc.Code_CODE_NOT_FOUND {
				// 409 if intermediate dir is missing, see https://tools.ietf.org/html/rfc4918#section-9.8.5
				log.Debug().Interface("parent", pRef).Interface("status", intStatRes.Status).Msg("conflict")
				w.WriteHeader(http.StatusConflict)
			} else {
				errors.HandleErrorStatus(log, w, intStatRes.Status)
			}
			return nil
		}
		// TODO what if intermediate is a file?
	}

	return &copy{source: srcRef, sourceInfo: srcStatRes.Info, depth: depth, successCode: successCode, destination: dstRef}
}

func (s *svc) tusUpload(ctx context.Context, uploadEP, uploadToken string, body io.Reader, size int64) (string, error) {
	chunkSize := int64(10000000)
	var offset int64
	var fileid string

	for offset < size {
		n := chunkSize
		if offset+n > size {
			n = size - offset
		}

		req, err := rhttp.NewRequest(ctx, http.MethodPatch, uploadEP, io.LimitReader(body, n))
		if err != nil {
			return "", err
		}

		req.Header.Set(datagateway.TokenTransportHeader, uploadToken)
		req.Header.Set(net.HeaderTusResumable, "1.0.0")
		req.Header.Set(net.HeaderUploadOffset, strconv.FormatInt(offset, 10))
		req.Header.Set(net.HeaderContentType, "application/offset+octet-stream")
		req.ContentLength = n

		res, err := s.client.Do(req)
		if err != nil {
			return "", err
		}

		if res.StatusCode != http.StatusNoContent && res.StatusCode != http.StatusOK && res.StatusCode != http.StatusCreated {
			res.Body.Close()
			return "", fmt.Errorf("unexpected status code during TUS upload: %d", res.StatusCode)
		}

		if id := res.Header.Get(net.HeaderOCFileID); id != "" {
			fileid = id
		}

		newOffsetStr := res.Header.Get(net.HeaderUploadOffset)
		res.Body.Close()

		if newOffsetStr != "" {
			newOffset, err := strconv.ParseInt(newOffsetStr, 10, 64)
			if err != nil {
				return "", fmt.Errorf("invalid Upload-Offset header: %v", err)
			}
			offset = newOffset
		} else {
			offset += n
		}
	}
	return fileid, nil
}
