// Copyright 2018-2022 CERN
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

package upload

import (
	"context"
	"encoding/hex"
	"fmt"
	"hash"
	"io"
	"io/fs"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	userpb "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"
	provider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	"github.com/golang-jwt/jwt/v5"
	"github.com/pkg/errors"
	tusd "github.com/tus/tusd/v2/pkg/handler"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"

	"github.com/opencloud-eu/reva/v2/pkg/appctx"
	ctxpkg "github.com/opencloud-eu/reva/v2/pkg/ctx"
	"github.com/opencloud-eu/reva/v2/pkg/errtypes"
	"github.com/opencloud-eu/reva/v2/pkg/events"
	"github.com/opencloud-eu/reva/v2/pkg/rhttp/datatx/metrics"
	"github.com/opencloud-eu/reva/v2/pkg/rhttp/datatx/utils/download"
	"github.com/opencloud-eu/reva/v2/pkg/storage/pkg/decomposedfs/metadata/prefixes"
	"github.com/opencloud-eu/reva/v2/pkg/storage/pkg/decomposedfs/node"
	"github.com/opencloud-eu/reva/v2/pkg/utils"
)

var (
	tracer          trace.Tracer
	defaultFilePerm = os.FileMode(0664)
)

func init() {
	tracer = otel.Tracer("github.com/cs3org/reva/pkg/storage/utils/decomposedfs/upload")
}

// WriteChunk writes the stream from the reader to the given offset of the upload
func (session *DecomposedFsSession) WriteChunk(ctx context.Context, _ int64, src io.Reader) (int64, error) {
	ctx, span := tracer.Start(session.Context(ctx), "WriteChunk")
	defer span.End()
	_, subspan := tracer.Start(ctx, "os.OpenFile")
	file, err := os.OpenFile(session.binPath(), os.O_WRONLY|os.O_APPEND, defaultFilePerm)
	subspan.End()
	if err != nil {
		return 0, err
	}
	defer func() {
		_ = file.Close()
	}()

	// calculate cheksum here? needed for the TUS checksum extension. https://tus.io/protocols/resumable-upload.html#checksum
	// TODO but how do we get the `Upload-Checksum`? WriteChunk() only has a context, offset and the reader ...
	// It is sent with the PATCH request, well or in the POST when the creation-with-upload extension is used
	// but the tus handler uses a context.Background() so we cannot really check the header and put it in the context ...
	_, subspan = tracer.Start(ctx, "io.Copy")
	n, err := io.Copy(file, src)
	subspan.End()

	// If the HTTP PATCH request gets interrupted in the middle (e.g. because
	// the user wants to pause the upload), Go's net/http returns an io.ErrUnexpectedEOF.
	// However, for the decompsedfs driver it's not important whether the stream has ended
	// on purpose or accidentally.
	if err != nil && err != io.ErrUnexpectedEOF {
		return n, err
	}

	// update upload.Session.Offset so subsequent code flow can use it.
	// No need to persist the session as the offset is determined by stating the blob in the GetUpload / ReadSession codepath.
	// The session offset is written to disk in FinishUpload
	session.info.Offset += n
	return n, nil
}

// GetInfo returns the FileInfo
func (session *DecomposedFsSession) GetInfo(_ context.Context) (tusd.FileInfo, error) {
	return session.ToFileInfo(), nil
}

// GetReader returns an io.Reader for the upload
func (session *DecomposedFsSession) GetReader(ctx context.Context) (io.ReadCloser, error) {
	_, span := tracer.Start(session.Context(ctx), "GetReader")
	defer span.End()
	return os.Open(session.binPath())
}

// FinishUpload finishes an upload and moves the file to the internal destination
// implements tusd.DataStore interface
// returns tusd errors
func (session *DecomposedFsSession) FinishUpload(ctx context.Context) error {
	err := session.FinishUploadDecomposed(ctx)

	//  we need to return a tusd error here to make the tusd handler return the correct status code
	switch err.(type) {
	case errtypes.AlreadyExists:
		return tusd.NewError("ERR_ALREADY_EXISTS", err.Error(), http.StatusConflict)
	case errtypes.Aborted:
		return tusd.NewError("ERR_PRECONDITION_FAILED", err.Error(), http.StatusPreconditionFailed)
	default:
		return err
	}
}

// FinishUploadDecomposed finishes an upload and moves the file to the internal destination
// retures errtypes errors
func (session *DecomposedFsSession) FinishUploadDecomposed(ctx context.Context) error {
	ctx, span := tracer.Start(session.Context(ctx), "FinishUpload")
	defer span.End()
	log := appctx.GetLogger(ctx)

	ctx = ctxpkg.ContextSetInitiator(ctx, session.InitiatorID())

	sha1h, md5h, adler32h, err := node.CalculateChecksums(ctx, session.binPath())
	if err != nil {
		return err
	}

	// compare if they match the sent checksum
	// TODO the tus checksum extension would do this on every chunk, but I currently don't see an easy way to pass in the requested checksum. for now we do it in FinishUpload which is also called for chunked uploads
	if session.info.MetaData["checksum"] != "" {
		var err error
		parts := strings.SplitN(session.info.MetaData["checksum"], " ", 2)
		if len(parts) != 2 {
			return errtypes.BadRequest("invalid checksum format. must be '[algorithm] [checksum]'")
		}
		switch parts[0] {
		case "sha1":
			err = checkHash(parts[1], sha1h)
		case "md5":
			err = checkHash(parts[1], md5h)
		case "adler32":
			err = checkHash(parts[1], adler32h)
		default:
			err = errtypes.BadRequest("unsupported checksum algorithm: " + parts[0])
		}
		if err != nil {
			session.store.Cleanup(ctx, session, true, false, false)
			return err
		}
	}

	// update checksums
	attrs := node.Attributes{
		prefixes.ChecksumPrefix + "sha1":    sha1h.Sum(nil),
		prefixes.ChecksumPrefix + "md5":     md5h.Sum(nil),
		prefixes.ChecksumPrefix + "adler32": adler32h.Sum(nil),
	}

	// At this point we scope by the space to create the final file in the final location
	if session.store.um != nil && session.info.Storage["SpaceGid"] != "" {
		gid, err := strconv.Atoi(session.info.Storage["SpaceGid"])
		if err != nil {
			return errors.Wrap(err, "failed to parse space gid")
		}

		unscope, err := session.store.um.ScopeUserByIds(-1, gid)
		if err != nil {
			return errors.Wrap(err, "failed to scope user")
		}
		if unscope != nil {
			defer func() { _ = unscope() }()
		}
	}

	n, err := session.store.CreateNodeForUpload(ctx, session, attrs)
	if err != nil {
		return err
	}
	// increase the processing counter for every started processing
	// will be decreased in Cleanup()
	metrics.UploadProcessing.Inc()
	metrics.UploadSessionsBytesReceived.Inc()

	if session.store.pub != nil && session.info.Size > 0 {
		u, _ := ctxpkg.ContextGetUser(ctx)
		s, err := session.URL(ctx)
		if err != nil {
			return err
		}

		var iu *userpb.User
		if utils.ExistsInOpaque(u.Opaque, "impersonating-user") {
			iu = &userpb.User{}
			if err := utils.ReadJSONFromOpaque(u.Opaque, "impersonating-user", iu); err != nil {
				return err
			}
		}

		if err := events.Publish(ctx, session.store.pub, events.BytesReceived{
			UploadID:          session.ID(),
			URL:               s,
			SpaceOwner:        n.SpaceOwnerOrManager(session.Context(ctx)),
			ExecutingUser:     u,
			ResourceID:        &provider.ResourceId{SpaceId: n.SpaceID, OpaqueId: n.ID},
			Filename:          session.Filename(),
			Filesize:          uint64(session.Size()),
			ImpersonatingUser: iu,
		}); err != nil {
			return err
		}
	}

	// if the upload is synchronous or the upload is empty, finalize it now
	// for 0-byte uploads we take a shortcut and finalize isn't called elsewhere
	if !session.store.async || session.info.Size == 0 {
		// handle postprocessing synchronously
		err = session.Finalize(ctx)
		session.store.Cleanup(ctx, session, err != nil, false, err == nil)
		if err != nil {
			log.Error().Err(err).Msg("failed to upload")
			return err
		}
		metrics.UploadSessionsFinalized.Inc()
	}

	return session.store.tp.Propagate(ctx, n, session.SizeDiff())
}

// Terminate terminates the upload
func (session *DecomposedFsSession) Terminate(_ context.Context) error {
	session.Cleanup(true, true, true)
	return nil
}

// DeclareLength updates the upload length information
func (session *DecomposedFsSession) DeclareLength(ctx context.Context, length int64) error {
	session.info.Size = length
	session.info.SizeIsDeferred = false
	return session.store.um.RunInBaseScope(func() error {
		return session.Persist(session.Context(ctx))
	})
}

// ConcatUploads concatenates multiple uploads
func (session *DecomposedFsSession) ConcatUploads(_ context.Context, uploads []tusd.Upload) (err error) {
	file, err := os.OpenFile(session.binPath(), os.O_WRONLY|os.O_APPEND, defaultFilePerm)
	if err != nil {
		return err
	}
	defer func() {
		_ = file.Close()
	}()

	for _, partialUpload := range uploads {
		fileUpload := partialUpload.(*DecomposedFsSession)

		src, err := os.Open(fileUpload.binPath())
		if err != nil {
			return err
		}
		defer func() {
			_ = src.Close()
		}()

		if _, err := io.Copy(file, src); err != nil {
			return err
		}
	}

	return
}

// Finalize finalizes the upload (eg moves the file to the internal destination)
func (session *DecomposedFsSession) Finalize(ctx context.Context) (err error) {
	ctx, span := tracer.Start(session.Context(ctx), "Finalize")
	defer span.End()

	revisionNode := node.New(session.SpaceID(), session.NodeID(), "", "", session.Size(), session.ID(),
		provider.ResourceType_RESOURCE_TYPE_FILE, session.SpaceOwner(), session.store.lu)

	// upload the data to the blobstore
	_, subspan := tracer.Start(ctx, "WriteBlob")
	err = session.store.tp.WriteBlob(revisionNode, session.binPath())
	subspan.End()
	if err != nil {
		return errors.Wrap(err, "failed to upload file to blobstore")
	}

	return nil
}

func checkHash(expected string, h hash.Hash) error {
	shash := hex.EncodeToString(h.Sum(nil))
	if expected != shash {
		return errtypes.ChecksumMismatch(fmt.Sprintf("invalid checksum: expected %s got %x", expected, shash))
	}
	return nil
}

func (session *DecomposedFsSession) removeNode(ctx context.Context) {
	n, err := session.Node(ctx)
	if err != nil {
		appctx.GetLogger(ctx).Error().Str("session", session.ID()).Err(err).Msg("getting node from session failed")
		return
	}
	if err := n.Purge(ctx); err != nil {
		appctx.GetLogger(ctx).Error().Str("nodepath", n.InternalPath()).Err(err).Msg("purging node failed")
	}
}

// cleanup cleans up after the upload is finished
func (session *DecomposedFsSession) Cleanup(revertNodeMetadata, cleanBin, cleanInfo bool) {
	ctx := session.Context(context.Background())

	if revertNodeMetadata {
		n, err := session.Node(ctx)
		if err != nil {
			appctx.GetLogger(ctx).Error().Err(err).Str("sessionid", session.ID()).Msg("reading node for session failed")
		} else {
			if session.NodeExists() && session.info.MetaData["versionID"] != "" {
				versionID := session.info.MetaData["versionID"]
				revisionNode := node.NewBaseNode(n.SpaceID, versionID, session.store.lu)

				if err := session.store.lu.CopyMetadata(ctx, revisionNode, n, func(attributeName string, value []byte) (newValue []byte, copy bool) {
					return value, strings.HasPrefix(attributeName, prefixes.ChecksumPrefix) ||
						attributeName == prefixes.TypeAttr ||
						attributeName == prefixes.BlobIDAttr ||
						attributeName == prefixes.BlobsizeAttr ||
						attributeName == prefixes.MTimeAttr
				}, true); err != nil {
					appctx.GetLogger(ctx).Info().Str("version", versionID).Str("nodepath", n.InternalPath()).Err(err).Msg("renaming version node failed")
				}

				if err := os.RemoveAll(revisionNode.InternalPath()); err != nil {
					appctx.GetLogger(ctx).Info().Str("version", versionID).Str("nodepath", n.InternalPath()).Err(err).Msg("error removing version")
				}

			} else {
				// if no other upload session is in progress (processing id != session id) or has finished (processing id == "")
				latestSession, err := n.ProcessingID(ctx)
				if err != nil {
					appctx.GetLogger(ctx).Error().Err(err).Str("spaceid", n.SpaceID).Str("nodeid", n.ID).Str("uploadid", session.ID()).Msg("reading processingid for session failed")
				}
				if latestSession == session.ID() {
					// actually delete the node
					session.removeNode(ctx)
				}
				// FIXME else if the upload has become a revision, delete the revision, or if it is the last one, delete the node
			}
		}
	}

	if cleanBin {
		if err := os.Remove(session.binPath()); err != nil && !errors.Is(err, fs.ErrNotExist) {
			appctx.GetLogger(ctx).Error().Str("path", session.binPath()).Err(err).Msg("removing upload failed")
		}
	}

	if cleanInfo {
		if err := session.Purge(ctx); err != nil && !errors.Is(err, fs.ErrNotExist) {
			appctx.GetLogger(ctx).Error().Err(err).Str("session", session.ID()).Msg("removing upload info failed")
		}
	}
}

// URL returns a url to download an upload
func (session *DecomposedFsSession) URL(_ context.Context) (string, error) {
	type transferClaims struct {
		jwt.RegisteredClaims
		Target string `json:"target"`
	}

	u := joinurl(session.store.tknopts.DownloadEndpoint, "tus/", session.ID())
	ttl := time.Duration(session.store.tknopts.TransferExpires) * time.Second
	claims := transferClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(ttl)),
			Audience:  jwt.ClaimStrings{"reva"},
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
		Target: u,
	}

	t := jwt.NewWithClaims(jwt.GetSigningMethod("HS256"), claims)

	tkn, err := t.SignedString([]byte(session.store.tknopts.TransferSharedSecret))
	if err != nil {
		return "", errors.Wrapf(err, "error signing token with claims %+v", claims)
	}

	return joinurl(session.store.tknopts.DataGatewayEndpoint, tkn), nil
}

// ServeContent serves the content of the upload and implements the http.ServeContent interface needed by tusd,
// it is used by the tusd handler to serve the content of the upload and supports range requests
func (session *DecomposedFsSession) ServeContent(ctx context.Context, w http.ResponseWriter, req *http.Request) error {
	_, span := tracer.Start(session.Context(ctx), "ServeContent")
	defer span.End()

	f, err := os.Open(session.binPath())
	if err != nil {
		return err
	}
	defer func() {
		_ = f.Close()
	}()

	info, err := f.Stat()
	if err != nil {
		return err
	}

	var r io.Reader = f
	if err := func() error {
		if req.Header.Get("Range") == "" {
			return nil
		}

		ranges, err := download.ParseRange(req.Header.Get("Range"), info.Size())
		switch {
		case len(ranges) == 0:
			fallthrough
		case errors.Is(err, download.ErrInvalidRange):
			// ignore invalid range and return the whole file
			return nil
		case err != nil:
			return err
		}

		r = io.NewSectionReader(f, ranges[0].Start, ranges[0].Length)
		w.WriteHeader(http.StatusPartialContent)
		w.Header().Set("Content-Range", ranges[0].ContentRange(info.Size()))
		return nil
	}(); err != nil {
		return err
	}

	if _, err := io.Copy(w, r); err != nil {
		return err
	}

	return nil
}

// replace with url.JoinPath after switching to go1.19
func joinurl(paths ...string) string {
	var s strings.Builder
	l := len(paths)
	for i, p := range paths {
		s.WriteString(p)
		if !strings.HasSuffix(p, "/") && i != l-1 {
			s.WriteString("/")
		}
	}

	return s.String()
}
