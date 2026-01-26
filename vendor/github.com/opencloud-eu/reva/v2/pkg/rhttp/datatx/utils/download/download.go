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

// Package download provides a library to handle file download requests.
package download

import (
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"path"
	"strconv"
	"strings"

	provider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	"github.com/rs/zerolog"

	"github.com/opencloud-eu/reva/v2/internal/grpc/services/storageprovider"
	"github.com/opencloud-eu/reva/v2/internal/http/services/owncloud/ocdav/net"
	"github.com/opencloud-eu/reva/v2/pkg/appctx"
	"github.com/opencloud-eu/reva/v2/pkg/errtypes"
	"github.com/opencloud-eu/reva/v2/pkg/storage"
	"github.com/opencloud-eu/reva/v2/pkg/storagespace"
	"github.com/opencloud-eu/reva/v2/pkg/utils"
)

type contextKey struct{}

var etagKey = contextKey{}

// ContextWithEtag returns a new `context.Context` that holds an etag.
func ContextWithEtag(ctx context.Context, etag string) context.Context {
	return context.WithValue(ctx, etagKey, etag)
}

// EtagFromContext returns the etag previously associated with `ctx`, or
// `""` if no such etag could be found.
func EtagFromContext(ctx context.Context) string {
	val := ctx.Value(etagKey)
	if etag, ok := val.(string); ok {
		return etag
	}
	return ""
}

// GetOrHeadFile returns the requested file content
func GetOrHeadFile(w http.ResponseWriter, r *http.Request, fs storage.FS, spaceID string) {
	ctx := r.Context()
	sublog := appctx.GetLogger(ctx).With().Str("svc", "datatx").Str("handler", "download").Logger()

	var fn string
	files, ok := r.URL.Query()["filename"]
	if !ok || len(files[0]) < 1 {
		fn = r.URL.Path
	} else {
		fn = files[0]
	}

	var ref *provider.Reference
	if spaceID == "" {
		// ensure the absolute path starts with '/'
		ref = &provider.Reference{Path: path.Join("/", fn)}
	} else {
		// build a storage space reference
		rid, err := storagespace.ParseID(spaceID)
		if err != nil {
			handleError(w, &sublog, err, "parse ID")
		}
		ref = &provider.Reference{
			ResourceId: &rid,
			// ensure the relative path starts with '.'
			Path: utils.MakeRelativePath(fn),
		}
	}
	// TODO check preconditions like If-Range, If-Match ...

	var md *provider.ResourceInfo
	var content io.ReadCloser
	var err error
	var notModified bool

	// do a stat to set Content-Length and etag headers

	md, content, err = fs.Download(ctx, ref, func(md *provider.ResourceInfo) bool {
		// range requests always need to open the reader to check if it is seekable
		if r.Header.Get("Range") != "" {
			return true
		}
		// otherwise, HEAD requests do not need to open a reader
		if r.Method == "HEAD" {
			return false
		}

		// check etag, see https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/If-None-Match
		for _, etag := range r.Header.Values(net.HeaderIfNoneMatch) {
			if md.Etag == etag {
				// When the condition fails for GET and HEAD methods, then the server must return
				// HTTP status code 304 (Not Modified). [...] Note that the server generating a
				// 304 response MUST generate any of the following header fields that would have
				// been sent in a 200 (OK) response to the same request:
				// Cache-Control, Content-Location, Date, ETag, Expires, and Vary.
				notModified = true
				return false
			}
		}
		return true
	})
	if err != nil {
		handleError(w, &sublog, err, "download")
		return
	}
	if content != nil {
		defer content.Close()
	}
	if notModified {
		w.Header().Set(net.HeaderETag, md.Etag)
		w.WriteHeader(http.StatusNotModified)
		return
	}

	// fill in storage provider id if it is missing
	if spaceID != "" && md.GetId().GetStorageId() == "" {
		md.Id.StorageId = ref.ResourceId.StorageId
	}

	var ranges []HTTPRange

	if r.Header.Get("Range") != "" {
		ranges, err = ParseRange(r.Header.Get("Range"), int64(md.Size))
		if err != nil {
			if err == ErrNoOverlap {
				w.Header().Set("Content-Range", fmt.Sprintf("bytes */%d", md.Size))
			}
			sublog.Error().Err(err).Interface("md", md).Interface("ranges", ranges).Msg("range request not satisfiable")
			w.WriteHeader(http.StatusRequestedRangeNotSatisfiable)

			return
		}
		if SumRangesSize(ranges) > int64(md.Size) {
			// The total number of bytes in all the ranges
			// is larger than the size of the file by
			// itself, so this is probably an attack, or a
			// dumb client. Ignore the range request.
			ranges = nil
		}
	}

	code := http.StatusOK
	sendSize := int64(md.Size)
	var sendContent io.Reader = content

	var s io.Seeker
	if s, ok = content.(io.Seeker); ok {
		// tell clients they can send range requests
		w.Header().Set("Accept-Ranges", "bytes")
	}

	w.Header().Set(net.HeaderContentType, strings.Join([]string{md.MimeType, "charset=UTF-8"}, "; "))

	if len(ranges) > 0 {
		sublog.Debug().Int64("start", ranges[0].Start).Int64("length", ranges[0].Length).Msg("range request")
		if s == nil {
			sublog.Error().Int64("start", ranges[0].Start).Int64("length", ranges[0].Length).Msg("ReadCloser is not seekable")
			w.WriteHeader(http.StatusRequestedRangeNotSatisfiable)
			return
		}

		code = http.StatusPartialContent

		switch {
		case len(ranges) == 1:
			// RFC 7233, Section 4.1:
			// "If a single part is being transferred, the server
			// generating the 206 response MUST generate a
			// Content-Range header field, describing what range
			// of the selected representation is enclosed, and a
			// payload consisting of the range.
			// ...
			// A server MUST NOT generate a multipart response to
			// a request for a single range, since a client that
			// does not request multiple parts might not support
			// multipart responses."
			ra := ranges[0]
			if _, err := s.Seek(ra.Start, io.SeekStart); err != nil {
				sublog.Error().Err(err).Int64("start", ra.Start).Int64("length", ra.Length).Msg("content is not seekable")
				w.WriteHeader(http.StatusRequestedRangeNotSatisfiable)
				return
			}
			sendSize = ra.Length
			w.Header().Set("Content-Range", ra.ContentRange(int64(md.Size)))
		case len(ranges) > 1:
			sendSize = RangesMIMESize(ranges, md.MimeType, int64(md.Size))

			pr, pw := io.Pipe()
			mw := multipart.NewWriter(pw)
			w.Header().Set("Content-Type", "multipart/byteranges; boundary="+mw.Boundary())
			sendContent = pr
			defer pr.Close() // cause writing goroutine to fail and exit if CopyN doesn't finish.
			go func() {
				for _, ra := range ranges {
					part, err := mw.CreatePart(ra.MimeHeader(md.MimeType+"; charset=UTF-8", int64(md.Size)))
					if err != nil {
						_ = pw.CloseWithError(err) // CloseWithError always returns nil
						return
					}
					if _, err := s.Seek(ra.Start, io.SeekStart); err != nil {
						_ = pw.CloseWithError(err) // CloseWithError always returns nil
						return
					}
					if _, err := io.CopyN(part, content, ra.Length); err != nil {
						_ = pw.CloseWithError(err) // CloseWithError always returns nil
						return
					}
				}
				mw.Close()
				pw.Close()
			}()
		}
	}

	if w.Header().Get(net.HeaderContentEncoding) == "" {
		w.Header().Set(net.HeaderContentLength, strconv.FormatInt(sendSize, 10))
	}

	w.Header().Set(net.HeaderContentDisposistion, net.ContentDispositionAttachment(path.Base(md.Path)))
	w.Header().Set(net.HeaderETag, md.Etag)
	w.Header().Set(net.HeaderOCFileID, storagespace.FormatResourceID(md.Id))
	w.Header().Set(net.HeaderOCETag, md.Etag)
	w.Header().Set(net.HeaderLastModified, net.RFC1123Z(md.Mtime))

	if md.Checksum != nil {
		w.Header().Set(net.HeaderOCChecksum, fmt.Sprintf("%s:%s", strings.ToUpper(string(storageprovider.GRPC2PKGXS(md.Checksum.Type))), md.Checksum.Sum))
	}

	w.WriteHeader(code)

	if r.Method != "HEAD" {
		var c int64
		c, err = io.CopyN(w, sendContent, sendSize)
		if err != nil {
			sublog.Error().Err(err).Interface("resourceid", md.Id).Msg("error copying data to response")
			return
		}
		if c != sendSize {
			sublog.Error().Int64("copied", c).Int64("size", sendSize).Msg("copied vs size mismatch")
		}
	}
}

func handleError(w http.ResponseWriter, log *zerolog.Logger, err error, action string) {
	switch err.(type) {
	case errtypes.IsNotFound:
		log.Debug().Err(err).Str("action", action).Msg("file not found")
		w.WriteHeader(http.StatusNotFound)
	case errtypes.IsPermissionDenied:
		log.Debug().Err(err).Str("action", action).Msg("permission denied")
		w.WriteHeader(http.StatusForbidden)
	case errtypes.Aborted:
		log.Debug().Err(err).Str("action", action).Msg("etags do not match")
		w.WriteHeader(http.StatusPreconditionFailed)
	case errtypes.TooEarly:
		log.Debug().Err(err).Str("action", action).Msg("file is still being processed")
		w.WriteHeader(http.StatusTooEarly)
	default:
		log.Error().Err(err).Str("action", action).Msg("unexpected error")
		w.WriteHeader(http.StatusInternalServerError)
	}
}
