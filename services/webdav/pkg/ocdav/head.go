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
	"strconv"
	"strings"
	"time"

	"github.com/opencloud-eu/reva/v2/pkg/storagespace"

	rpc "github.com/cs3org/go-cs3apis/cs3/rpc/v1beta1"
	provider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	"github.com/opencloud-eu/opencloud/services/webdav/pkg/ocdav/errors"
	"github.com/opencloud-eu/opencloud/services/webdav/pkg/ocdav/net"
	"github.com/opencloud-eu/opencloud/services/webdav/pkg/ocdav/spacelookup"
	"github.com/opencloud-eu/opencloud/services/webdav/pkg/xs"
	"github.com/opencloud-eu/reva/v2/pkg/appctx"
	"github.com/opencloud-eu/reva/v2/pkg/utils"
	"github.com/rs/zerolog"
)

func (s *svc) handlePathHead(w http.ResponseWriter, r *http.Request, ns string) {
	ctx, span := appctx.GetTracerProvider(r.Context()).Tracer(tracerName).Start(r.Context(), "head")
	defer span.End()

	fn := path.Join(ns, r.URL.Path)

	sublog := appctx.GetLogger(ctx).With().Str("path", fn).Logger()

	space, status, err := spacelookup.LookUpStorageSpaceForPath(ctx, s.gatewaySelector, fn)
	if err != nil {
		sublog.Error().Err(err).Str("path", fn).Msg("failed to look up storage space")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if status.Code != rpc.Code_CODE_OK {
		errors.HandleErrorStatus(&sublog, w, status)
		return
	}

	s.handleHead(ctx, w, r, spacelookup.MakeRelativeReference(space, fn, false), sublog)
}

func (s *svc) handleHead(ctx context.Context, w http.ResponseWriter, r *http.Request, ref *provider.Reference, log zerolog.Logger) {
	client, err := s.gatewaySelector.Next()
	if err != nil {
		log.Error().Err(err).Msg("error selecting next client")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	req := &provider.StatRequest{Ref: ref}
	res, err := client.Stat(ctx, req)
	if err != nil {
		log.Error().Err(err).Msg("error sending grpc stat request")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if res.Status.Code != rpc.Code_CODE_OK {
		errors.HandleErrorStatus(&log, w, res.Status)
		return
	}

	info := res.Info
	w.Header().Set(net.HeaderContentType, info.MimeType)
	w.Header().Set(net.HeaderETag, info.Etag)
	w.Header().Set(net.HeaderOCFileID, storagespace.FormatResourceID(info.Id))
	w.Header().Set(net.HeaderOCETag, info.Etag)
	if info.Checksum != nil {
		w.Header().Set(net.HeaderOCChecksum, fmt.Sprintf("%s:%s", strings.ToUpper(string(xs.GRPC2PKGXS(info.Checksum.Type))), info.Checksum.Sum))
	}
	t := utils.TSToTime(info.Mtime).UTC()
	lastModifiedString := t.Format(time.RFC1123Z)
	w.Header().Set(net.HeaderLastModified, lastModifiedString)
	w.Header().Set(net.HeaderContentLength, strconv.FormatUint(info.Size, 10))
	if info.Type != provider.ResourceType_RESOURCE_TYPE_CONTAINER {
		w.Header().Set(net.HeaderAcceptRanges, "bytes")
	}
	if utils.ReadPlainFromOpaque(res.GetInfo().GetOpaque(), "status") == "processing" {
		w.WriteHeader(http.StatusTooEarly)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (s *svc) handleSpacesHead(w http.ResponseWriter, r *http.Request, spaceID string) {
	ctx, span := appctx.GetTracerProvider(r.Context()).Tracer(tracerName).Start(r.Context(), "spaces_head")
	defer span.End()

	sublog := appctx.GetLogger(ctx).With().Str("spaceid", spaceID).Str("path", r.URL.Path).Logger()

	ref, err := spacelookup.MakeStorageSpaceReference(spaceID, r.URL.Path)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	s.handleHead(ctx, w, r, &ref, sublog)
}
