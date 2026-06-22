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
	"encoding/json"
	"io"
	"net/http"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	rpc "github.com/cs3org/go-cs3apis/cs3/rpc/v1beta1"
	link "github.com/cs3org/go-cs3apis/cs3/sharing/link/v1beta1"
	provider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	typespb "github.com/cs3org/go-cs3apis/cs3/types/v1beta1"
	"github.com/opencloud-eu/reva/v2/internal/http/services/owncloud/ocdav/errors"
	"github.com/opencloud-eu/reva/v2/internal/http/services/owncloud/ocdav/net"
	"github.com/opencloud-eu/reva/v2/internal/http/services/owncloud/ocdav/spacelookup"
	"github.com/opencloud-eu/reva/v2/pkg/appctx"
	"github.com/opencloud-eu/reva/v2/pkg/conversions"
	"github.com/opencloud-eu/reva/v2/pkg/rhttp"
	"github.com/opencloud-eu/reva/v2/pkg/storagespace"
	"github.com/opencloud-eu/reva/v2/pkg/utils"
	"github.com/rs/zerolog"
	tusd "github.com/tus/tusd/v2/pkg/handler"
	"go.opentelemetry.io/otel/propagation"
)

// Propagator ensures the importer module uses the same trace propagation strategy.
var Propagator = propagation.NewCompositeTextMapPropagator(
	propagation.Baggage{},
	propagation.TraceContext{},
)

func (s *svc) handlePathTusPost(w http.ResponseWriter, r *http.Request, ns string) {
	ctx, span := appctx.GetTracerProvider(r.Context()).Tracer(tracerName).Start(r.Context(), "tus-post")
	defer span.End()

	// read filename from metadata
	meta := tusd.ParseMetadataHeader(r.Header.Get(net.HeaderUploadMetadata))

	// append filename to current dir
	ref := &provider.Reference{
		// a path based request has no resource id, so we can only provide a path. The gateway has te figure out which provider is responsible
		Path: path.Join(ns, r.URL.Path, meta["filename"]),
	}

	sublog := appctx.GetLogger(ctx).With().Str("path", r.URL.Path).Str("filename", meta["filename"]).Logger()

	s.handleTusPost(ctx, w, r, meta, ref, sublog)
}

func (s *svc) handleSpacesTusPost(w http.ResponseWriter, r *http.Request, spaceID string) {
	ctx, span := appctx.GetTracerProvider(r.Context()).Tracer(tracerName).Start(r.Context(), "spaces-tus-post")
	defer span.End()

	// read filename from metadata
	meta := tusd.ParseMetadataHeader(r.Header.Get(net.HeaderUploadMetadata))

	ref, err := spacelookup.MakeStorageSpaceReference(spaceID, path.Join(r.URL.Path, meta["filename"]))
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	sublog := appctx.GetLogger(ctx).With().Str("spaceid", spaceID).Str("path", r.URL.Path).Str("filename", meta["filename"]).Logger()

	s.handleTusPost(ctx, w, r, meta, &ref, sublog)
}

func (s *svc) handleTusPost(ctx context.Context, w http.ResponseWriter, r *http.Request, meta map[string]string, ref *provider.Reference, log zerolog.Logger) {
	w.Header().Add(net.HeaderAccessControlAllowHeaders, strings.Join([]string{net.HeaderTusResumable, net.HeaderUploadLength, net.HeaderUploadMetadata, net.HeaderIfMatch}, ", "))
	w.Header().Add(net.HeaderAccessControlExposeHeaders, strings.Join([]string{net.HeaderTusResumable, net.HeaderUploadOffset, net.HeaderLocation}, ", "))
	w.Header().Set(net.HeaderTusExtension, "creation,creation-with-upload,checksum,expiration")

	w.Header().Set(net.HeaderTusResumable, "1.0.0")

	// Test if the version sent by the client is supported
	// GET methods are not checked since a browser may visit this URL and does
	// not include this header. This request is not part of the specification.
	if r.Header.Get(net.HeaderTusResumable) != "1.0.0" {
		w.WriteHeader(http.StatusPreconditionFailed)
		return
	}
	if r.Header.Get(net.HeaderUploadLength) == "" {
		w.WriteHeader(http.StatusPreconditionFailed)
		return
	}
	if err := ValidateName(filename(meta["filename"]), s.nameValidators); err != nil {
		w.WriteHeader(http.StatusPreconditionFailed)
		return
	}

	// Test if the target is a secret filedrop
	var isSecretFileDrop bool
	tokenStatInfo, ok := TokenStatInfoFromContext(ctx)
	// We assume that when the uploader can create containers, but is not allowed to list them, it is a secret file drop
	if ok && tokenStatInfo.GetPermissionSet().CreateContainer && !tokenStatInfo.GetPermissionSet().ListContainer {
		isSecretFileDrop = true
	}

	checksum := ""
	if cs, ok := meta["checksum"]; ok {
		cparts := strings.SplitN(cs, " ", 2)
		if len(cparts) != 2 {
			log.Debug().Str("upload-checksum", cs).Msg("invalid Upload-Checksum format, expected '[algorithm] [checksum]'")
			w.WriteHeader(http.StatusBadRequest)
			return
		} else {
			// we do not check the algorithm here, because it might depend on the storage
			checksum = strings.ToLower(cparts[0]) + " " + cparts[1]
		}
	}

	client, err := s.gatewaySelector.Next()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	sReq := &provider.StatRequest{
		Ref: ref,
	}
	sRes, err := client.Stat(ctx, sReq)
	if err != nil {
		log.Error().Err(err).Msg("error sending grpc stat request")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if sRes.Status.Code != rpc.Code_CODE_OK && sRes.Status.Code != rpc.Code_CODE_NOT_FOUND {
		errors.HandleErrorStatus(&log, w, sRes.Status)
		return
	}

	info := sRes.Info
	if info != nil && info.Type != provider.ResourceType_RESOURCE_TYPE_FILE {
		log.Warn().Msg("resource is not a file")
		w.WriteHeader(http.StatusConflict)
		return
	}

	if info != nil {
		clientETag := r.Header.Get(net.HeaderIfMatch)
		serverETag := info.Etag
		if clientETag != "" {
			if clientETag != serverETag {
				log.Warn().Str("client-etag", clientETag).Str("server-etag", serverETag).Msg("etags mismatch")
				w.WriteHeader(http.StatusPreconditionFailed)
				return
			}
		}
		if isSecretFileDrop {
			// find next filename
			newName, status, err := FindName(ctx, client, filepath.Base(ref.Path), sRes.GetInfo().GetParentId())
			if err != nil {
				log.Error().Err(err).Msg("error sending grpc stat request")
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			if status.GetCode() != rpc.Code_CODE_OK {
				log.Error().Interface("status", status).Msg("error listing file")
				errors.HandleErrorStatus(&log, w, status)
				return
			}
			ref.Path = filepath.Join(filepath.Dir(ref.GetPath()), newName)
			sRes.GetInfo().Name = newName
		}
	}

	uploadLength, err := strconv.ParseInt(r.Header.Get(net.HeaderUploadLength), 10, 64)
	if err != nil {
		log.Debug().Err(err).Msg("wrong request")
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if uploadLength == 0 {
		tfRes, err := client.TouchFile(ctx, &provider.TouchFileRequest{
			Ref: ref,
		})
		if err != nil {
			log.Error().Err(err).Msg("error sending grpc stat request")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		switch tfRes.Status.Code {
		case rpc.Code_CODE_OK:
			w.Header().Set(net.HeaderLocation, "")
			w.WriteHeader(http.StatusCreated)
			return
		case rpc.Code_CODE_ALREADY_EXISTS:
			// Fall through to the tus case
		default:
			log.Error().Interface("status", tfRes.Status).Msg("error touching file")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	}

	opaqueMap := map[string]*typespb.OpaqueEntry{
		net.HeaderUploadLength: {
			Decoder: "plain",
			Value:   []byte(r.Header.Get(net.HeaderUploadLength)),
		},
	}

	if checksum != "" {
		opaqueMap[net.HeaderUploadChecksum] = &typespb.OpaqueEntry{
			Decoder: "plain",
			Value:   []byte(checksum),
		}
	}
	mtime := meta["mtime"]
	if mtime != "" {
		opaqueMap[net.HeaderOCMtime] = &typespb.OpaqueEntry{
			Decoder: "plain",
			Value:   []byte(mtime),
		}
	}

	// initiateUpload
	uReq := &provider.InitiateFileUploadRequest{
		Ref: ref,
		Opaque: &typespb.Opaque{
			Map: opaqueMap,
		},
	}

	uRes, err := client.InitiateFileUpload(ctx, uReq)
	if err != nil {
		log.Error().Err(err).Msg("error initiating file upload")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if uRes.Status.Code != rpc.Code_CODE_OK {
		if r.ProtoMajor == 1 {
			// drain body to avoid `connection closed` errors
			_, _ = io.Copy(io.Discard, r.Body)
		}
		if uRes.Status.Code == rpc.Code_CODE_NOT_FOUND {
			w.WriteHeader(http.StatusPreconditionFailed)
			return
		}
		errors.HandleErrorStatus(&log, w, uRes.Status)
		return
	}

	var ep, token string
	for _, p := range uRes.Protocols {
		if p.Protocol == "tus" {
			ep, token = p.UploadEndpoint, p.Token
		}
	}

	// TUS clients don't understand the reva transfer token. We need to append it to the upload endpoint.
	// The DataGateway has to take care of pulling it back into the request header upon request arrival.
	if token != "" {
		if !strings.HasSuffix(ep, "/") {
			ep += "/"
		}
		ep += token
	}

	w.Header().Set(net.HeaderLocation, ep)

	// for creation-with-upload extension forward bytes to dataprovider
	// TODO check this really streams
	if r.Header.Get(net.HeaderContentType) == "application/offset+octet-stream" {
		finishUpload := true
		if uploadLength > 0 {
			var httpRes *http.Response

			httpReq, err := rhttp.NewRequest(ctx, http.MethodPatch, ep, r.Body)
			if err != nil {
				log.Debug().Err(err).Msg("wrong request")
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			Propagator.Inject(ctx, propagation.HeaderCarrier(httpReq.Header))

			httpReq.Header.Set(net.HeaderContentType, r.Header.Get(net.HeaderContentType))
			httpReq.Header.Set(net.HeaderContentLength, r.Header.Get(net.HeaderContentLength))
			if r.Header.Get(net.HeaderUploadOffset) != "" {
				httpReq.Header.Set(net.HeaderUploadOffset, r.Header.Get(net.HeaderUploadOffset))
			} else {
				httpReq.Header.Set(net.HeaderUploadOffset, "0")
			}
			httpReq.Header.Set(net.HeaderTusResumable, r.Header.Get(net.HeaderTusResumable))

			httpRes, err = s.client.Do(httpReq)
			if err != nil || httpRes == nil {
				log.Error().Err(err).Msg("error doing PATCH request to data gateway")
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			defer httpRes.Body.Close()

			if httpRes.StatusCode != http.StatusNoContent {
				w.WriteHeader(httpRes.StatusCode)
				return
			}

			w.Header().Set(net.HeaderUploadOffset, httpRes.Header.Get(net.HeaderUploadOffset))
			w.Header().Set(net.HeaderTusResumable, httpRes.Header.Get(net.HeaderTusResumable))
			w.Header().Set(net.HeaderTusUploadExpires, httpRes.Header.Get(net.HeaderTusUploadExpires))
			if httpRes.Header.Get(net.HeaderOCMtime) != "" {
				w.Header().Set(net.HeaderOCMtime, httpRes.Header.Get(net.HeaderOCMtime))
			}

			if strings.HasPrefix(uReq.GetRef().GetPath(), "/public") && uReq.GetRef().GetResourceId() == nil {
				// Use the path based request for the public link
				sReq.Ref.Path = uReq.Ref.GetPath()
				sReq.Ref.ResourceId = nil
			} else {
				if resid, err := storagespace.ParseID(httpRes.Header.Get(net.HeaderOCFileID)); err == nil {
					sReq.Ref = &provider.Reference{
						ResourceId: &resid,
					}
				}
			}
			finishUpload = httpRes.Header.Get(net.HeaderUploadOffset) == r.Header.Get(net.HeaderUploadLength)
		}

		// check if upload was fully completed
		if uploadLength == 0 || finishUpload {
			// get uploaded file metadata

			sRes, err := client.Stat(ctx, sReq)
			if err != nil {
				log.Error().Err(err).Msg("error sending grpc stat request")
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			if sRes.Status.Code != rpc.Code_CODE_OK && sRes.Status.Code != rpc.Code_CODE_NOT_FOUND {
				if sRes.Status.Code == rpc.Code_CODE_PERMISSION_DENIED {
					// the token expired during upload, so the stat failed
					// and we can't do anything about it.
					// the clients will handle this gracefully by doing a propfind on the file
					w.WriteHeader(http.StatusOK)
					return
				}

				errors.HandleErrorStatus(&log, w, sRes.Status)
				return
			}

			info := sRes.Info
			if info == nil {
				log.Error().Msg("No info found for uploaded file")
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			// get WebDav permissions for file
			isPublic := false
			if info.Opaque != nil && info.Opaque.Map != nil {
				if info.Opaque.Map["link-share"] != nil && info.Opaque.Map["link-share"].Decoder == "json" {
					ls := &link.PublicShare{}
					_ = json.Unmarshal(info.Opaque.Map["link-share"].Value, ls)
					isPublic = ls != nil
				}
			}
			isShared := !net.IsCurrentUserOwnerOrManager(ctx, info.Owner, info)
			role := conversions.RoleFromResourcePermissions(info.PermissionSet, isPublic)
			permissions := role.WebDAVPermissions(
				info.Type == provider.ResourceType_RESOURCE_TYPE_CONTAINER,
				isShared,
				false,
				isPublic,
			)

			w.Header().Set(net.HeaderContentType, info.MimeType)
			w.Header().Set(net.HeaderOCFileID, storagespace.FormatResourceID(info.Id))
			w.Header().Set(net.HeaderOCETag, info.Etag)
			w.Header().Set(net.HeaderETag, info.Etag)
			w.Header().Set(net.HeaderOCPermissions, permissions)

			t := utils.TSToTime(info.Mtime).UTC()
			lastModifiedString := t.Format(time.RFC1123Z)
			w.Header().Set(net.HeaderLastModified, lastModifiedString)
		}
	}

	w.WriteHeader(http.StatusCreated)
}
