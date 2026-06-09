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

package simple

import (
	"net/http"
	"time"

	userpb "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"
	provider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	"github.com/go-viper/mapstructure/v2"
	ctxpkg "github.com/opencloud-eu/reva/v2/pkg/ctx"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"

	"github.com/opencloud-eu/reva/v2/internal/http/services/owncloud/ocdav/net"
	"github.com/opencloud-eu/reva/v2/pkg/appctx"
	"github.com/opencloud-eu/reva/v2/pkg/errtypes"
	"github.com/opencloud-eu/reva/v2/pkg/events"
	"github.com/opencloud-eu/reva/v2/pkg/rhttp/datatx"
	"github.com/opencloud-eu/reva/v2/pkg/rhttp/datatx/manager/registry"
	"github.com/opencloud-eu/reva/v2/pkg/rhttp/datatx/metrics"
	"github.com/opencloud-eu/reva/v2/pkg/rhttp/datatx/utils/download"
	"github.com/opencloud-eu/reva/v2/pkg/storage"
	"github.com/opencloud-eu/reva/v2/pkg/storage/cache"
	"github.com/opencloud-eu/reva/v2/pkg/storagespace"
	"github.com/opencloud-eu/reva/v2/pkg/utils"
)

func init() {
	registry.Register("simple", New)
}

type manager struct {
	conf      *cache.Config
	publisher events.Publisher
	log       *zerolog.Logger
}

func parseConfig(m map[string]interface{}) (*cache.Config, error) {
	c := &cache.Config{}
	if err := mapstructure.Decode(m, c); err != nil {
		err = errors.Wrap(err, "error decoding conf")
		return nil, err
	}
	return c, nil
}

// New returns a datatx manager implementation that relies on HTTP PUT/GET.
func New(m map[string]interface{}, publisher events.Publisher, log *zerolog.Logger) (datatx.DataTX, error) {
	c, err := parseConfig(m)
	if err != nil {
		return nil, err
	}

	l := log.With().Str("datatx", "simple").Logger()

	return &manager{
		conf:      c,
		publisher: publisher,
		log:       &l,
	}, nil
}

func (m *manager) Handler(fs storage.FS) (http.Handler, error) {
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sublog := m.log.With().Str("path", r.URL.Path).Logger()
		r = r.WithContext(appctx.WithLogger(r.Context(), &sublog))
		ctx := r.Context()

		switch r.Method {
		case "GET", "HEAD":
			if r.Method == "GET" {
				metrics.DownloadsActive.Add(1)
				defer func() {
					metrics.DownloadsActive.Sub(1)
				}()
			}
			download.GetOrHeadFile(w, r, fs, "")
		case "PUT":
			metrics.UploadsActive.Add(1)
			defer func() {
				metrics.UploadsActive.Sub(1)
			}()

			if r.ContentLength == 0 {
				sublog.Info().Msg("received invalid 0-byte PUT request")
				w.WriteHeader(http.StatusBadRequest)
				return
			}

			fn := r.URL.Path
			defer r.Body.Close()

			ref := &provider.Reference{Path: fn}

			if lockID := r.Header.Get("X-Lock-Id"); lockID != "" {
				ctx = ctxpkg.ContextSetLockID(ctx, lockID)
			}

			info, err := fs.Upload(ctx, storage.UploadRequest{
				Ref:    ref,
				Body:   r.Body,
				Length: r.ContentLength,
			}, func(spaceOwner, owner *userpb.UserId, ref *provider.Reference) {
				if err := datatx.EmitFileUploadedEvent(spaceOwner, owner, ref, m.publisher); err != nil {
					sublog.Error().Err(err).Msg("failed to publish FileUploaded event")
				}
			})
			switch v := err.(type) {
			case nil:
				// set etag, mtime and file id
				w.Header().Set(net.HeaderETag, info.Etag)
				w.Header().Set(net.HeaderOCETag, info.Etag)
				if info.Id != nil {
					w.Header().Set(net.HeaderOCFileID, storagespace.FormatResourceID(info.Id))
				}
				if info.Mtime != nil {
					t := utils.TSToTime(info.Mtime).UTC()
					lastModifiedString := t.Format(time.RFC1123Z)
					w.Header().Set(net.HeaderLastModified, lastModifiedString)
				}
				w.WriteHeader(http.StatusOK)
			case errtypes.PartialContent:
				w.WriteHeader(http.StatusPartialContent)
			case errtypes.ChecksumMismatch:
				w.WriteHeader(errtypes.StatusChecksumMismatch)
			case errtypes.NotFound:
				w.WriteHeader(http.StatusNotFound)
			case errtypes.PermissionDenied:
				w.WriteHeader(http.StatusForbidden)
			case errtypes.InvalidCredentials:
				w.WriteHeader(http.StatusUnauthorized)
			case errtypes.InsufficientStorage:
				w.WriteHeader(http.StatusInsufficientStorage)
			case errtypes.PreconditionFailed, errtypes.Aborted, errtypes.AlreadyExists:
				w.WriteHeader(http.StatusPreconditionFailed)
			default:
				sublog.Error().Err(v).Msg("error uploading file")
				w.WriteHeader(http.StatusInternalServerError)
			}
			return
		default:
			w.WriteHeader(http.StatusNotImplemented)
		}
	})
	return h, nil
}
