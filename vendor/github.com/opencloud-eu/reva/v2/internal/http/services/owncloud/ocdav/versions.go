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
	"net/http"
	"path"

	"github.com/opencloud-eu/reva/v2/internal/http/services/owncloud/ocdav/config"
	"github.com/opencloud-eu/reva/v2/internal/http/services/owncloud/ocdav/errors"
	"github.com/opencloud-eu/reva/v2/internal/http/services/owncloud/ocdav/net"
	"github.com/opencloud-eu/reva/v2/internal/http/services/owncloud/ocdav/propfind"
	"github.com/opencloud-eu/reva/v2/pkg/storagespace"
	"github.com/opencloud-eu/reva/v2/pkg/utils"

	rpc "github.com/cs3org/go-cs3apis/cs3/rpc/v1beta1"
	provider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	types "github.com/cs3org/go-cs3apis/cs3/types/v1beta1"
	"github.com/opencloud-eu/reva/v2/pkg/appctx"
	"github.com/opencloud-eu/reva/v2/pkg/rhttp/router"
)

// VersionsHandler handles version requests
type VersionsHandler struct {
}

func (h *VersionsHandler) init(c *config.Config) error {
	return nil
}

// Handler handles requests
// versions can be listed with a PROPFIND to /remote.php/dav/meta/<fileid>/v
// a version is identified by a timestamp, eg. /remote.php/dav/meta/<fileid>/v/1561410426
func (h *VersionsHandler) Handler(s *svc, rid *provider.ResourceId) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		if rid == nil {
			http.Error(w, "404 Not Found", http.StatusNotFound)
			return
		}

		// baseURI is encoded as part of the response payload in href field
		baseURI := path.Join(ctx.Value(net.CtxKeyBaseURI).(string), storagespace.FormatResourceID(rid))
		ctx = context.WithValue(ctx, net.CtxKeyBaseURI, baseURI)
		r = r.WithContext(ctx)

		var key string
		key, r.URL.Path = router.ShiftPath(r.URL.Path)
		if r.Method == http.MethodOptions {
			s.handleOptions(w, r)
			return
		}
		if key == "" && r.Method == MethodPropfind {
			h.doListVersions(w, r, s, rid)
			return
		}
		if key != "" {
			switch r.Method {
			case MethodCopy:
				// TODO(jfd) cs3api has no delete file version call
				// TODO(jfd) restore version to given Destination, but cs3api has no destination
				h.doRestore(w, r, s, rid, key)
				return
			case http.MethodHead:
				log := appctx.GetLogger(ctx)
				ref := &provider.Reference{
					ResourceId: &provider.ResourceId{
						StorageId: rid.StorageId,
						SpaceId:   rid.SpaceId,
						OpaqueId:  key,
					},
					Path: utils.MakeRelativePath(r.URL.Path),
				}
				s.handleHead(ctx, w, r, ref, *log)
				return
			case http.MethodGet:
				log := appctx.GetLogger(ctx)
				ref := &provider.Reference{
					ResourceId: &provider.ResourceId{
						StorageId: rid.StorageId,
						SpaceId:   rid.SpaceId,
						OpaqueId:  key,
					},
					Path: utils.MakeRelativePath(r.URL.Path),
				}
				s.handleGet(ctx, w, r, ref, "spaces", *log)
				return
			}
		}

		http.Error(w, "501 Forbidden", http.StatusNotImplemented)
	})
}

func (h *VersionsHandler) doListVersions(w http.ResponseWriter, r *http.Request, s *svc, rid *provider.ResourceId) {
	ctx, span := appctx.GetTracerProvider(r.Context()).Tracer(tracerName).Start(r.Context(), "listVersions")
	defer span.End()

	sublog := appctx.GetLogger(ctx).With().Interface("resourceid", rid).Logger()

	pf, status, err := propfind.ReadPropfind(r.Body)
	if err != nil {
		sublog.Debug().Err(err).Msg("error reading propfind request")
		w.WriteHeader(status)
		return
	}

	client, err := s.gatewaySelector.Next()
	if err != nil {
		sublog.Error().Err(err).Msg("error selecting next gateway client")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	ref := &provider.Reference{ResourceId: rid}
	res, err := client.Stat(ctx, &provider.StatRequest{Ref: ref})
	if err != nil {
		sublog.Error().Err(err).Msg("error sending a grpc stat request")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if res.Status.Code != rpc.Code_CODE_OK {
		if res.Status.Code == rpc.Code_CODE_PERMISSION_DENIED || res.Status.Code == rpc.Code_CODE_NOT_FOUND {
			w.WriteHeader(http.StatusNotFound)
			b, err := errors.Marshal(http.StatusNotFound, "Resource not found", "", "")
			errors.HandleWebdavError(&sublog, w, b, err)
			return
		}
		errors.HandleErrorStatus(&sublog, w, res.Status)
		return
	}

	info := res.Info

	lvRes, err := client.ListFileVersions(ctx, &provider.ListFileVersionsRequest{Ref: ref})
	if err != nil {
		sublog.Error().Err(err).Msg("error sending list container grpc request")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if lvRes.Status.Code != rpc.Code_CODE_OK {
		if lvRes.Status.Code == rpc.Code_CODE_PERMISSION_DENIED {
			w.WriteHeader(http.StatusForbidden)
			b, err := errors.Marshal(http.StatusForbidden, "You have no permission to list file versions on this resource", "", "")
			errors.HandleWebdavError(&sublog, w, b, err)
			return
		}
		errors.HandleErrorStatus(&sublog, w, lvRes.Status)
		return
	}

	versions := lvRes.GetVersions()
	infos := make([]*provider.ResourceInfo, 0, len(versions)+1)
	// add version dir . entry, derived from file info
	infos = append(infos, &provider.ResourceInfo{
		Type: provider.ResourceType_RESOURCE_TYPE_CONTAINER,
	})

	for i := range versions {
		vi := &provider.ResourceInfo{
			// TODO(jfd) we cannot access version content, this will be a problem when trying to fetch version thumbnails
			// Opaque
			Type: provider.ResourceType_RESOURCE_TYPE_FILE,
			Id: &provider.ResourceId{
				StorageId: "versions",
				OpaqueId:  info.Id.OpaqueId + "@" + versions[i].GetKey(),
			},
			// Checksum
			Etag: versions[i].Etag,
			// MimeType
			Mtime: &types.Timestamp{
				Seconds: versions[i].Mtime,
				// TODO cs3apis FileVersion should use types.Timestamp instead of uint64
			},
			Path: path.Join("v", versions[i].Key),
			// PermissionSet
			Size:  versions[i].Size,
			Owner: info.Owner,
		}
		infos = append(infos, vi)
	}

	prefer := net.ParsePrefer(r.Header.Get("prefer"))
	returnMinimal := prefer[net.HeaderPreferReturn] == "minimal"

	propRes, err := propfind.MultistatusResponse(ctx, &pf, infos, s.c.PublicURL, "", nil, returnMinimal, nil)
	if err != nil {
		sublog.Error().Err(err).Msg("error formatting propfind")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.Header().Set(net.HeaderDav, "1, 3, extended-mkcol")
	w.Header().Set(net.HeaderContentType, "application/xml; charset=utf-8")
	w.Header().Set(net.HeaderVary, net.HeaderPrefer)
	if returnMinimal {
		w.Header().Set(net.HeaderPreferenceApplied, "return=minimal")
	}
	w.WriteHeader(http.StatusMultiStatus)
	_, err = w.Write(propRes)
	if err != nil {
		sublog.Error().Err(err).Msg("error writing body")
		return
	}

}

func (h *VersionsHandler) doRestore(w http.ResponseWriter, r *http.Request, s *svc, rid *provider.ResourceId, key string) {
	ctx, span := appctx.GetTracerProvider(r.Context()).Tracer(tracerName).Start(r.Context(), "restore")
	defer span.End()

	sublog := appctx.GetLogger(ctx).With().Interface("resourceid", rid).Str("key", key).Logger()

	req := &provider.RestoreFileVersionRequest{
		Ref: &provider.Reference{ResourceId: rid},
		Key: key,
	}

	client, err := s.gatewaySelector.Next()
	if err != nil {
		sublog.Error().Err(err).Msg("error selecting next gateway client")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	res, err := client.RestoreFileVersion(ctx, req)
	if err != nil {
		sublog.Error().Err(err).Msg("error sending a grpc restore version request")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if res.Status.Code != rpc.Code_CODE_OK {
		if res.Status.Code == rpc.Code_CODE_PERMISSION_DENIED {
			w.WriteHeader(http.StatusForbidden)
			b, err := errors.Marshal(http.StatusForbidden, "You have no permission to restore versions on this resource", "", "")
			errors.HandleWebdavError(&sublog, w, b, err)
			return
		}
		errors.HandleErrorStatus(&sublog, w, res.Status)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
