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
	"io"
	"net/http"
	"path"
	"path/filepath"
	"strconv"
	"strings"

	rpc "github.com/cs3org/go-cs3apis/cs3/rpc/v1beta1"
	provider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	typespb "github.com/cs3org/go-cs3apis/cs3/types/v1beta1"
	"github.com/opencloud-eu/reva/v2/internal/http/services/datagateway"
	"github.com/opencloud-eu/reva/v2/internal/http/services/owncloud/ocdav/errors"
	"github.com/opencloud-eu/reva/v2/internal/http/services/owncloud/ocdav/net"
	"github.com/opencloud-eu/reva/v2/internal/http/services/owncloud/ocdav/spacelookup"
	"github.com/opencloud-eu/reva/v2/pkg/appctx"
	"github.com/opencloud-eu/reva/v2/pkg/errtypes"
	"github.com/opencloud-eu/reva/v2/pkg/rhttp"
	"github.com/opencloud-eu/reva/v2/pkg/storagespace"
	"github.com/opencloud-eu/reva/v2/pkg/utils"
	"github.com/rs/zerolog"
	"go.opentelemetry.io/otel/propagation"
)

func sufferMacOSFinder(r *http.Request) bool {
	return r.Header.Get(net.HeaderExpectedEntityLength) != ""
}

func handleMacOSFinder(w http.ResponseWriter, r *http.Request) error {
	/*
	   Many webservers will not cooperate well with Finder PUT requests,
	   because it uses 'Chunked' transfer encoding for the request body.
	   The symptom of this problem is that Finder sends files to the
	   server, but they arrive as 0-length files.
	   If we don't do anything, the user might think they are uploading
	   files successfully, but they end up empty on the server. Instead,
	   we throw back an error if we detect this.
	   The reason Finder uses Chunked, is because it thinks the files
	   might change as it's being uploaded, and therefore the
	   Content-Length can vary.
	   Instead it sends the X-Expected-Entity-Length header with the size
	   of the file at the very start of the request. If this header is set,
	   but we don't get a request body we will fail the request to
	   protect the end-user.
	*/

	log := appctx.GetLogger(r.Context())
	content := r.Header.Get(net.HeaderContentLength)
	expected := r.Header.Get(net.HeaderExpectedEntityLength)
	log.Warn().Str("content-length", content).Str("x-expected-entity-length", expected).Msg("Mac OS Finder corner-case detected")

	// The best mitigation to this problem is to tell users to not use crappy Finder.
	// Another possible mitigation is to change the use the value of X-Expected-Entity-Length header in the Content-Length header.
	expectedInt, err := strconv.ParseInt(expected, 10, 64)
	if err != nil {
		log.Error().Err(err).Msg("error parsing expected length")
		w.WriteHeader(http.StatusBadRequest)
		return err
	}
	r.Header.Set(net.HeaderContentLength, expected)
	r.ContentLength = expectedInt
	return nil
}

func isContentRange(r *http.Request) bool {
	/*
		   Content-Range is dangerous for PUT requests:  PUT per definition
		   stores a full resource.  draft-ietf-httpbis-p2-semantics-15 says
		   in section 7.6:
			 An origin server SHOULD reject any PUT request that contains a
			 Content-Range header field, since it might be misinterpreted as
			 partial content (or might be partial content that is being mistakenly
			 PUT as a full representation).  Partial content updates are possible
			 by targeting a separately identified resource with state that
			 overlaps a portion of the larger resource, or by using a different
			 method that has been specifically defined for partial updates (for
			 example, the PATCH method defined in [RFC5789]).
		   This clarifies RFC2616 section 9.6:
			 The recipient of the entity MUST NOT ignore any Content-*
			 (e.g. Content-Range) headers that it does not understand or implement
			 and MUST return a 501 (Not Implemented) response in such cases.
		   OTOH is a PUT request with a Content-Range currently the only way to
		   continue an aborted upload request and is supported by curl, mod_dav,
		   Tomcat and others.  Since some clients do use this feature which results
		   in unexpected behaviour (cf PEAR::HTTP_WebDAV_Client 1.0.1), we reject
		   all PUT requests with a Content-Range for now.
	*/
	return r.Header.Get(net.HeaderContentRange) != ""
}

func (s *svc) handlePathPut(w http.ResponseWriter, r *http.Request, ns string) {
	ctx, span := appctx.GetTracerProvider(r.Context()).Tracer(tracerName).Start(r.Context(), "put")
	defer span.End()

	fn := path.Join(ns, r.URL.Path)
	sublog := appctx.GetLogger(ctx).With().Str("path", fn).Logger()

	if err := ValidateName(filename(r.URL.Path), s.nameValidators); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		b, err := errors.Marshal(http.StatusBadRequest, err.Error(), "", "")
		errors.HandleWebdavError(&sublog, w, b, err)
		return
	}

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

	s.handlePut(ctx, w, r, spacelookup.MakeRelativeReference(space, fn, false), sublog)
}

func (s *svc) handlePut(ctx context.Context, w http.ResponseWriter, r *http.Request, ref *provider.Reference, log zerolog.Logger) {
	if !checkPreconditions(w, r, log) {
		// checkPreconditions handles error returns
		return
	}

	length, err := getContentLength(r)
	if err != nil {
		log.Error().Err(err).Msg("error getting the content length")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	client, err := s.gatewaySelector.Next()
	if err != nil {
		log.Error().Err(err).Msg("error selecting next gateway client")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// Test if the target is a secret filedrop
	tokenStatInfo, ok := TokenStatInfoFromContext(ctx)
	// We assume that when the uploader can create containers, but is not allowed to list them, it is a secret file drop
	if ok && tokenStatInfo.GetPermissionSet().CreateContainer && !tokenStatInfo.GetPermissionSet().ListContainer {
		// TODO we can skip this stat if the tokenStatInfo is the direct parent
		sReq := &provider.StatRequest{
			Ref: ref,
		}
		sRes, err := client.Stat(ctx, sReq)
		if err != nil {
			log.Error().Err(err).Msg("error sending grpc stat request")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		// We also need to continue if we are not allowed to stat a resource. We may not have stat permission. That still means it exists and we need to find a new filename.
		switch sRes.Status.Code {
		case rpc.Code_CODE_OK, rpc.Code_CODE_PERMISSION_DENIED:
			// find next filename
			newName, status, err := FindName(ctx, client, filepath.Base(ref.Path), sRes.GetInfo().GetParentId())
			if err != nil {
				log.Error().Err(err).Msg("error sending grpc stat request")
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			if status.Code != rpc.Code_CODE_OK {
				log.Error().Interface("status", status).Msg("error listing file")
				errors.HandleErrorStatus(&log, w, status)
				return
			}
			ref.Path = utils.MakeRelativePath(filepath.Join(filepath.Dir(ref.GetPath()), newName))
		case rpc.Code_CODE_NOT_FOUND:
			// just continue with normal upload
		default:
			log.Error().Interface("status", sRes.Status).Msg("error stating file")
			errors.HandleErrorStatus(&log, w, sRes.Status)
			return
		}
	}

	opaque := &typespb.Opaque{}
	if mtime := r.Header.Get(net.HeaderOCMtime); mtime != "" {
		utils.AppendPlainToOpaque(opaque, net.HeaderOCMtime, mtime)

		// TODO: find a way to check if the storage really accepted the value
		w.Header().Set(net.HeaderOCMtime, "accepted")
	}
	if length == 0 {
		tfRes, err := client.TouchFile(ctx, &provider.TouchFileRequest{
			Opaque: opaque,
			Ref:    ref,
		})
		if err != nil {
			log.Error().Err(err).Msg("error sending grpc touch file request")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		if tfRes.Status.Code == rpc.Code_CODE_OK {
			sRes, err := client.Stat(ctx, &provider.StatRequest{
				Ref: ref,
			})
			if err != nil {
				log.Error().Err(err).Msg("error sending grpc touch file request")
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			if sRes.Status.Code != rpc.Code_CODE_OK {
				log.Error().Interface("status", sRes.Status).Msg("error touching file")
				errors.HandleErrorStatus(&log, w, sRes.Status)
				return
			}

			w.Header().Set(net.HeaderETag, sRes.Info.Etag)
			w.Header().Set(net.HeaderOCETag, sRes.Info.Etag)
			w.Header().Set(net.HeaderOCFileID, storagespace.FormatResourceID(sRes.Info.Id))
			w.Header().Set(net.HeaderLastModified, net.RFC1123Z(sRes.Info.Mtime))

			w.WriteHeader(http.StatusCreated)
			return
		}

		if tfRes.Status.Code != rpc.Code_CODE_ALREADY_EXISTS {
			log.Error().Interface("status", tfRes.Status).Msg("error touching file")
			errors.HandleErrorStatus(&log, w, tfRes.Status)
			return
		}
	}

	utils.AppendPlainToOpaque(opaque, net.HeaderUploadLength, strconv.FormatInt(length, 10))

	// curl -X PUT https://demo.example.org/remote.php/webdav/testcs.bin -u demo:demo -d '123' -v -H 'OC-Checksum: SHA1:40bd001563085fc35165329ea1ff5c5ecbdbbeef'

	var cparts []string
	// TUS Upload-Checksum header takes precedence
	if checksum := r.Header.Get(net.HeaderUploadChecksum); checksum != "" {
		cparts = strings.SplitN(checksum, " ", 2)
		if len(cparts) != 2 {
			log.Debug().Str("upload-checksum", checksum).Msg("invalid Upload-Checksum format, expected '[algorithm] [checksum]'")
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		// Then try OpenCloud header
	} else if checksum := r.Header.Get(net.HeaderOCChecksum); checksum != "" {
		cparts = strings.SplitN(checksum, ":", 2)
		if len(cparts) != 2 {
			log.Debug().Str("oc-checksum", checksum).Msg("invalid OC-Checksum format, expected '[algorithm]:[checksum]'")
			w.WriteHeader(http.StatusBadRequest)
			return
		}
	}
	// we do not check the algorithm here, because it might depend on the storage
	if len(cparts) == 2 {
		// Translate into TUS style Upload-Checksum header
		// algorithm is always lowercase, checksum is separated by space
		utils.AppendPlainToOpaque(opaque, net.HeaderUploadChecksum, strings.ToLower(cparts[0])+" "+cparts[1])
	}

	uReq := &provider.InitiateFileUploadRequest{
		Ref:    ref,
		Opaque: opaque,
		LockId: requestLock(r),
	}
	if ifMatch := r.Header.Get(net.HeaderIfMatch); ifMatch != "" {
		uReq.Options = &provider.InitiateFileUploadRequest_IfMatch{IfMatch: ifMatch}
	}

	// where to upload the file?
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
		switch uRes.Status.Code {
		case rpc.Code_CODE_PERMISSION_DENIED:
			status := http.StatusForbidden
			m := uRes.Status.Message
			// check if user has access to parent
			sRes, err := client.Stat(ctx, &provider.StatRequest{Ref: &provider.Reference{
				ResourceId: ref.ResourceId,
				Path:       utils.MakeRelativePath(path.Dir(ref.Path)),
			}})
			if err != nil {
				log.Error().Err(err).Msg("error performing stat grpc request")
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			if sRes.Status.Code != rpc.Code_CODE_OK {
				// return not found error so we do not leak existence of a file
				// TODO hide permission failed for users without access in every kind of request
				// TODO should this be done in the driver?
				status = http.StatusNotFound
			}
			if status == http.StatusNotFound {
				m = "Resource not found" // mimic the oc10 error message
			}
			w.WriteHeader(status)
			b, err := errors.Marshal(status, m, "", "")
			errors.HandleWebdavError(&log, w, b, err)
		case rpc.Code_CODE_ABORTED:
			w.WriteHeader(http.StatusPreconditionFailed)
		case rpc.Code_CODE_FAILED_PRECONDITION:
			w.WriteHeader(http.StatusConflict)
		default:
			errors.HandleErrorStatus(&log, w, uRes.Status)
		}
		return
	}

	// ony send actual PUT request if file has bytes. Otherwise the initiate file upload request creates the file
	if length != 0 {
		var ep, token string
		for _, p := range uRes.Protocols {
			if p.Protocol == "simple" {
				ep, token = p.UploadEndpoint, p.Token
			}
		}

		httpReq, err := rhttp.NewRequest(ctx, http.MethodPut, ep, r.Body)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		Propagator.Inject(ctx, propagation.HeaderCarrier(httpReq.Header))
		httpReq.Header.Set(datagateway.TokenTransportHeader, token)
		httpReq.ContentLength = length

		httpRes, err := s.client.Do(httpReq)
		if err != nil {
			log.Error().Err(err).Msg("error doing PUT request to data service")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		defer httpRes.Body.Close()
		if httpRes.StatusCode != http.StatusOK {
			if httpRes.StatusCode == http.StatusPartialContent {
				w.WriteHeader(http.StatusPartialContent)
				return
			}
			if httpRes.StatusCode == errtypes.StatusChecksumMismatch {
				w.WriteHeader(http.StatusBadRequest)
				b, err := errors.Marshal(http.StatusBadRequest, "The computed checksum does not match the one received from the client.", "", "")
				errors.HandleWebdavError(&log, w, b, err)
				return
			}
			log.Error().Err(err).Msg("PUT request to data server failed")
			w.WriteHeader(httpRes.StatusCode)
			return
		}

		// copy headers if they are present
		if httpRes.Header.Get(net.HeaderETag) != "" {
			w.Header().Set(net.HeaderETag, httpRes.Header.Get(net.HeaderETag))
		}
		if httpRes.Header.Get(net.HeaderOCETag) != "" {
			w.Header().Set(net.HeaderOCETag, httpRes.Header.Get(net.HeaderOCETag))
		}
		if httpRes.Header.Get(net.HeaderOCFileID) != "" {
			w.Header().Set(net.HeaderOCFileID, httpRes.Header.Get(net.HeaderOCFileID))
		}
		if httpRes.Header.Get(net.HeaderLastModified) != "" {
			w.Header().Set(net.HeaderLastModified, httpRes.Header.Get(net.HeaderLastModified))
		}
	}

	// file was new
	// FIXME make created flag a property on the InitiateFileUploadResponse
	if created := utils.ReadPlainFromOpaque(uRes.Opaque, "created"); created == "true" {
		w.WriteHeader(http.StatusCreated)
		return
	}

	// overwrite
	w.WriteHeader(http.StatusNoContent)
}

func (s *svc) handleSpacesPut(w http.ResponseWriter, r *http.Request, spaceID string) {
	ctx, span := appctx.GetTracerProvider(r.Context()).Tracer(tracerName).Start(r.Context(), "spaces_put")
	defer span.End()

	sublog := appctx.GetLogger(ctx).With().Str("spaceid", spaceID).Str("path", r.URL.Path).Logger()

	ref, err := spacelookup.MakeStorageSpaceReference(spaceID, r.URL.Path)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if ref.GetResourceId().GetOpaqueId() != "" && ref.GetResourceId().GetSpaceId() != ref.GetResourceId().GetOpaqueId() && r.URL.Path == "/" {
		s.handlePut(ctx, w, r, &ref, sublog)
		return
	}

	if err := ValidateName(filename(ref.Path), s.nameValidators); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		b, err := errors.Marshal(http.StatusBadRequest, err.Error(), "", "")
		errors.HandleWebdavError(&sublog, w, b, err)
		return
	}

	s.handlePut(ctx, w, r, &ref, sublog)
}

func checkPreconditions(w http.ResponseWriter, r *http.Request, log zerolog.Logger) bool {
	if isContentRange(r) {
		log.Debug().Msg("Content-Range not supported for PUT")
		w.WriteHeader(http.StatusNotImplemented)
		return false
	}

	if sufferMacOSFinder(r) {
		err := handleMacOSFinder(w, r)
		if err != nil {
			log.Debug().Err(err).Msg("error handling Mac OS corner-case")
			w.WriteHeader(http.StatusInternalServerError)
			return false
		}
	}
	return true
}

func getContentLength(r *http.Request) (int64, error) {
	length, err := strconv.ParseInt(r.Header.Get(net.HeaderContentLength), 10, 64)
	if err != nil {
		// Fallback to Upload-Length
		length, err = strconv.ParseInt(r.Header.Get(net.HeaderUploadLength), 10, 64)
		if err != nil {
			return 0, err
		}
	}
	return length, nil
}
