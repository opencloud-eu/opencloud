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

package decomposedfs

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	provider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	"github.com/google/uuid"
	"github.com/opencloud-eu/reva/v2/pkg/storage/utils/decomposedfs/metadata/prefixes"
	"github.com/pkg/errors"
	"github.com/rogpeppe/go-internal/lockedfile"
	tusd "github.com/tus/tusd/v2/pkg/handler"

	"github.com/opencloud-eu/reva/v2/pkg/appctx"
	ctxpkg "github.com/opencloud-eu/reva/v2/pkg/ctx"
	"github.com/opencloud-eu/reva/v2/pkg/errtypes"
	"github.com/opencloud-eu/reva/v2/pkg/rhttp/datatx/metrics"
	"github.com/opencloud-eu/reva/v2/pkg/storage"
	"github.com/opencloud-eu/reva/v2/pkg/storage/utils/chunking"
	"github.com/opencloud-eu/reva/v2/pkg/storage/utils/decomposedfs/node"
	"github.com/opencloud-eu/reva/v2/pkg/storage/utils/decomposedfs/upload"
	"github.com/opencloud-eu/reva/v2/pkg/storagespace"
	"github.com/opencloud-eu/reva/v2/pkg/utils"
)

// Upload uploads data to the given resource
// TODO(OCISDEV-901): remove Upload once all drivers are migrated to CommitUpload and the coordinator (OCISDEV-900) is in place.
func (fs *Decomposedfs) Upload(ctx context.Context, req storage.UploadRequest, uff storage.UploadFinishedFunc) (*provider.ResourceInfo, error) {
	_, span := tracer.Start(ctx, "Upload")
	defer span.End()
	up, err := fs.GetUpload(ctx, req.Ref.GetPath())
	if err != nil {
		return &provider.ResourceInfo{}, errors.Wrap(err, "Decomposedfs: error retrieving upload")
	}

	session := up.(*upload.OcisSession)

	ctx = session.Context(ctx)

	if session.Chunk() != "" { // check chunking v1
		p, assembledFile, err := fs.chunkHandler.WriteChunk(session.Chunk(), req.Body)
		if err != nil {
			return &provider.ResourceInfo{}, err
		}
		if p == "" {
			if err = session.Terminate(ctx); err != nil {
				return &provider.ResourceInfo{}, errors.Wrap(err, "Decomposedfs: error removing auxiliary files")
			}
			return &provider.ResourceInfo{}, errtypes.PartialContent(req.Ref.String())
		}
		fd, err := os.Open(assembledFile)
		if err != nil {
			return &provider.ResourceInfo{}, errors.Wrap(err, "Decomposedfs: error opening assembled file")
		}
		defer fd.Close()
		defer os.RemoveAll(assembledFile)
		req.Body = fd

		size, err := session.WriteChunk(ctx, 0, req.Body)
		if err != nil {
			return &provider.ResourceInfo{}, errors.Wrap(err, "Decomposedfs: error writing to binary file")
		}
		session.SetSize(size)
	} else {
		size, err := session.WriteChunk(ctx, 0, req.Body)
		if err != nil {
			return &provider.ResourceInfo{}, errors.Wrap(err, "Decomposedfs: error writing to binary file")
		}
		if size != req.Length {
			return &provider.ResourceInfo{}, errtypes.PartialContent("Decomposedfs: unexpected end of stream")
		}
	}

	if err := session.FinishUploadDecomposed(ctx); err != nil {
		return &provider.ResourceInfo{}, err
	}

	if uff != nil {
		uploadRef := &provider.Reference{
			ResourceId: &provider.ResourceId{
				StorageId: session.ProviderID(),
				SpaceId:   session.SpaceID(),
				OpaqueId:  session.SpaceID(),
			},
			Path: utils.MakeRelativePath(filepath.Join(session.Dir(), session.Filename())),
		}
		executant := session.Executant()
		uff(session.SpaceOwner(), &executant, uploadRef)
	}

	ri := &provider.ResourceInfo{
		// fill with at least fileid, mtime and etag
		Id: &provider.ResourceId{
			StorageId: session.ProviderID(),
			SpaceId:   session.SpaceID(),
			OpaqueId:  session.NodeID(),
		},
	}

	// add etag to metadata
	ri.Etag, _ = node.CalculateEtag(session.NodeID(), session.MTime())

	if !session.MTime().IsZero() {
		ri.Mtime = utils.TimeToTS(session.MTime())
	}

	return ri, nil
}

// InitiateUpload returns upload ids corresponding to different protocols it supports
// TODO(OCISDEV-901): remove InitiateUpload once all drivers are migrated to CommitUpload and the coordinator (OCISDEV-900) is in place.
func (fs *Decomposedfs) InitiateUpload(ctx context.Context, ref *provider.Reference, uploadLength int64, metadata map[string]string) (map[string]string, error) {
	_, span := tracer.Start(ctx, "InitiateUpload")
	defer span.End()
	log := appctx.GetLogger(ctx)

	// remember the path from the reference
	refpath := ref.GetPath()
	var chunk *chunking.ChunkBLOBInfo
	var err error
	if chunking.IsChunked(refpath) { // check chunking v1
		chunk, err = chunking.GetChunkBLOBInfo(refpath)
		if err != nil {
			return nil, errtypes.BadRequest(err.Error())
		}
		ref.Path = chunk.Path
	}
	n, err := fs.lu.NodeFromResource(ctx, ref)
	switch err.(type) {
	case nil:
		// ok
	case errtypes.IsNotFound:
		return nil, errtypes.PreconditionFailed(err.Error())
	default:
		return nil, err
	}

	// permissions are checked in NewUpload below

	relative, err := fs.lu.Path(ctx, n, node.NoCheck)
	// TODO why do we need the path here?
	// jfd: it is used later when emitting the UploadReady event ...
	// AAAND refPath might be . when accessing with an id / relative reference ... which causes NodeName to become . But then dir will also always be .
	// That is why we still have to read the path here: so that the event we emit contains a relative reference with a path relative to the space root. WTF
	if err != nil {
		return nil, err
	}

	lockID, _ := ctxpkg.ContextGetLockID(ctx)

	session := fs.sessionStore.New(ctx)
	session.SetMetadata("filename", n.Name)
	session.SetStorageValue("NodeName", n.Name)
	if chunk != nil {
		session.SetStorageValue("Chunk", filepath.Base(refpath))
	}
	session.SetMetadata("dir", filepath.Dir(relative))
	session.SetStorageValue("Dir", filepath.Dir(relative))
	session.SetMetadata("lockid", lockID)

	session.SetSize(uploadLength)
	session.SetStorageValue("SpaceRoot", n.SpaceRoot.ID)                                     // TODO SpaceRoot -> SpaceID
	session.SetStorageValue("SpaceOwnerOrManager", n.SpaceOwnerOrManager(ctx).GetOpaqueId()) // TODO needed for what?

	spaceGID, ok := ctx.Value(CtxKeySpaceGID).(uint32)
	if ok {
		session.SetStorageValue("SpaceGid", fmt.Sprintf("%d", spaceGID))
	}

	iid, _ := ctxpkg.ContextGetInitiator(ctx)
	session.SetMetadata("initiatorid", iid)

	if metadata != nil {
		session.SetMetadata("providerID", metadata["providerID"])
		if mtime, ok := metadata["mtime"]; ok {
			if mtime != "null" {
				session.SetMetadata("mtime", metadata["mtime"])
			}
		}
		if expiration, ok := metadata["expires"]; ok {
			if expiration != "null" {
				session.SetMetadata("expires", metadata["expires"])
			}
		}
		if _, ok := metadata["sizedeferred"]; ok {
			session.SetSizeIsDeferred(true)
		}
		if checksum, ok := metadata["checksum"]; ok {
			parts := strings.SplitN(checksum, " ", 2)
			if len(parts) != 2 {
				return nil, errtypes.BadRequest("invalid checksum format. must be '[algorithm] [checksum]'")
			}
			switch parts[0] {
			case "sha1", "md5", "adler32":
				session.SetMetadata("checksum", checksum)
			default:
				return nil, errtypes.BadRequest("unsupported checksum algorithm: " + parts[0])
			}
		}

		// only check preconditions if they are not empty // TODO or is this a bad request?
		if metadata["if-match"] != "" {
			session.SetMetadata("if-match", metadata["if-match"])
		}
		if metadata["if-none-match"] != "" {
			session.SetMetadata("if-none-match", metadata["if-none-match"])
		}
		if metadata["if-unmodified-since"] != "" {
			session.SetMetadata("if-unmodified-since", metadata["if-unmodified-since"])
		}
	}

	if session.MTime().IsZero() {
		session.SetMetadata("mtime", utils.TimeToOCMtime(time.Now()))
	}

	log.Debug().Str("uploadid", session.ID()).Str("spaceid", n.SpaceID).Str("nodeid", n.ID).Interface("metadata", metadata).Msg("Decomposedfs: resolved filename")

	_, err = node.CheckQuota(ctx, n.SpaceRoot, n.Exists, uint64(n.Blobsize), uint64(session.Size()))
	if err != nil {
		return nil, err
	}

	if session.Filename() == "" {
		return nil, errors.New("Decomposedfs: missing filename in metadata")
	}
	if session.Dir() == "" {
		return nil, errors.New("Decomposedfs: missing dir in metadata")
	}

	// the parent owner will become the new owner
	parent, perr := n.Parent(ctx)
	if perr != nil {
		return nil, errors.Wrap(perr, "Decomposedfs: error getting parent "+n.ParentID)
	}

	// check permissions
	var (
		checkNode *node.Node
		path      string
	)
	if n.Exists {
		// check permissions of file to be overwritten
		checkNode = n
		path, _ = storagespace.FormatReference(&provider.Reference{ResourceId: &provider.ResourceId{
			SpaceId:  checkNode.SpaceID,
			OpaqueId: checkNode.ID,
		}})
	} else {
		// check permissions of parent
		checkNode = parent
		path, _ = storagespace.FormatReference(&provider.Reference{ResourceId: &provider.ResourceId{
			SpaceId:  checkNode.SpaceID,
			OpaqueId: checkNode.ID,
		}, Path: n.Name})
	}
	rp, err := fs.p.AssemblePermissions(ctx, checkNode)
	switch {
	case err != nil:
		return nil, err
	case !rp.InitiateFileUpload:
		return nil, errtypes.PermissionDenied(path)
	}

	// are we trying to overwriting a folder with a file?
	if n.Exists && n.IsDir(ctx) {
		return nil, errtypes.PreconditionFailed("resource is not a file")
	}

	// check lock
	if err := n.CheckLock(ctx); err != nil {
		return nil, err
	}

	usr := ctxpkg.ContextMustGetUser(ctx)

	// fill future node info
	if n.Exists {
		if session.HeaderIfNoneMatch() == "*" {
			return nil, errtypes.Aborted(fmt.Sprintf("parent %s already has a child %s, id %s", n.ParentID, n.Name, n.ID))
		}
		session.SetStorageValue("NodeId", n.ID)
		session.SetStorageValue("NodeExists", "true")
	} else {
		session.SetStorageValue("NodeId", uuid.New().String())
	}
	session.SetStorageValue("NodeParentId", n.ParentID)
	session.SetExecutant(usr)
	session.SetStorageValue("LogLevel", log.GetLevel().String())

	log.Debug().Interface("session", session).Msg("Decomposedfs: built session info")

	err = fs.um.RunInBaseScope(func() error {
		// Create binary file in the upload folder with no content
		// It will be used when determining the current offset of an upload
		err := session.TouchBin()
		if err != nil {
			return err
		}

		return session.Persist(ctx)
	})
	if err != nil {
		return nil, err
	}
	metrics.UploadSessionsInitiated.Inc()

	if uploadLength == 0 {
		// Directly finish this upload
		err = session.FinishUploadDecomposed(ctx)
		if err != nil {
			return nil, err
		}
	}

	return map[string]string{
		"simple": session.ID(),
		"tus":    session.ID(),
	}, nil
}

// MarkProcessing toggles a processing flag on the resource.
func (fs *Decomposedfs) MarkProcessing(ctx context.Context, ref *provider.Reference, processing bool, sessionID string) error {
	n, err := fs.lu.NodeFromResource(ctx, ref)
	if err != nil {
		return err
	}
	if !n.Exists {
		return errtypes.NotFound(ref.String())
	}

	// Early lock, so MarkProcessing is atomic.
	f, err := lockedfile.OpenFile(fs.lu.MetadataBackend().LockfilePath(n.InternalPath()), os.O_RDWR|os.O_CREATE, 0600)
	if err != nil {
		return err
	}
	defer func() {
		if cerr := f.Close(); cerr != nil {
			appctx.GetLogger(ctx).Error().Err(cerr).Str("nodeid", n.ID).Msg("could not close mark-processing lock")
		}
	}()

	// Evict the node's in-process xattr cache so IsProcessing reads from disk while we hold the lock.
	n.ResetXattrsCache()

	if !processing {
		if !n.IsProcessing(ctx) {
			return nil
		}
		id, _ := n.ProcessingID(ctx)
		if id != sessionID {
			return nil // owned by a different session, do not clear
		}
		return n.RemoveXattr(ctx, prefixes.StatusPrefix, false)
	}

	if n.IsProcessing(ctx) {
		return errtypes.ResourceProcessing(ref.String())
	}
	return n.SetXattrsWithContext(ctx, node.Attributes{
		prefixes.StatusPrefix: []byte(node.ProcessingStatus + sessionID),
	}, false) // acquireLock=false, because outer lock already held
}

// CommitUpload writes the staged bytes from source to the resource at ref.
// sessionID is used to identify the correct blob slot prepared before postprocessing.
// Caller owns source.Body and must close it after CommitUpload returns.
func (fs *Decomposedfs) CommitUpload(ctx context.Context, ref *provider.Reference, sessionID string, source storage.UploadSource) error {
	if source.Body == nil {
		return errtypes.BadRequest("Decomposedfs: source body is nil")
	}
	if sessionID == "" {
		return errtypes.BadRequest("Decomposedfs: sessionID is empty")
	}

	n, err := fs.lu.NodeFromResource(ctx, ref)
	if err != nil {
		appctx.GetLogger(ctx).Error().Err(err).Msg("CommitUpload: unexpected NodeFromResource failure")
		return errtypes.InternalError("CommitUpload: node lookup failed unexpectedly")
	}
	if !n.Exists {
		return errtypes.NotFound(ref.String())
	}

	blobNode := node.New(n.SpaceID, n.ID, "", "", source.Length, sessionID,
		provider.ResourceType_RESOURCE_TYPE_FILE, nil, fs.lu)

	if err := fs.tp.WriteBlobFromReader(blobNode, source.Body, source.Length); err != nil {
		if derr := fs.tp.DeleteBlob(blobNode); derr != nil {
			appctx.GetLogger(ctx).Error().Err(derr).Str("nodeid", n.ID).Str("blobid", sessionID).Msg("could not clean up orphaned blob after failed write")
		}
		return errors.Wrap(err, "Decomposedfs: failed to write blob")
	}

	now := time.Now()
	if p, err := n.Parent(ctx); err != nil {
		appctx.GetLogger(ctx).Error().Err(err).Str("nodeid", n.ID).Msg("could not read parent for etag propagation")
	} else {
		_ = p.SetTMTime(ctx, &now)
		if err := fs.tp.Propagate(ctx, p, 0); err != nil {
			appctx.GetLogger(ctx).Error().Err(err).Str("nodeid", n.ID).Msg("could not propagate etag change")
		}
	}

	return nil
}

// UseIn tells the tus upload middleware which extensions it supports.
func (fs *Decomposedfs) UseIn(composer *tusd.StoreComposer) {
	composer.UseCore(fs)
	composer.UseTerminater(fs)
	composer.UseConcater(fs)
	composer.UseLengthDeferrer(fs)
}

// To implement the core tus.io protocol as specified in https://tus.io/protocols/resumable-upload.html#core-protocol
// - the storage needs to implement NewUpload and GetUpload
// - the upload needs to implement the tusd.Upload interface: WriteChunk, GetInfo, GetReader and FinishUpload

// NewUpload returns a new tus Upload instance
func (fs *Decomposedfs) NewUpload(ctx context.Context, info tusd.FileInfo) (tusd.Upload, error) {
	return nil, fmt.Errorf("not implemented, use InitiateUpload on the CS3 API to start a new upload")
}

// GetUpload returns the Upload for the given upload id
func (fs *Decomposedfs) GetUpload(ctx context.Context, id string) (tusd.Upload, error) {
	var ul tusd.Upload
	var err error
	_ = fs.um.RunInBaseScope(func() error {
		ul, err = fs.sessionStore.Get(ctx, id)
		return nil
	})
	return ul, err
}

// ListUploadSessions returns the upload sessions for the given filter
func (fs *Decomposedfs) ListUploadSessions(ctx context.Context, filter storage.UploadSessionFilter) ([]storage.UploadSession, error) {
	var sessions []*upload.OcisSession
	if filter.ID != nil && *filter.ID != "" {
		session, err := fs.sessionStore.Get(ctx, *filter.ID)
		if err != nil {
			return nil, err
		}
		sessions = []*upload.OcisSession{session}
	} else {
		var err error
		sessions, err = fs.sessionStore.List(ctx)
		if err != nil {
			return nil, err
		}
	}
	filteredSessions := []storage.UploadSession{}
	now := time.Now()
	for _, session := range sessions {
		if filter.Processing != nil && *filter.Processing != session.IsProcessing() {
			continue
		}
		if filter.Expired != nil {
			if *filter.Expired {
				if now.Before(session.Expires()) {
					continue
				}
			} else {
				if now.After(session.Expires()) {
					continue
				}
			}
		}
		if filter.HasVirus != nil {
			sr, _ := session.ScanData()
			infected := sr != ""
			if *filter.HasVirus != infected {
				continue
			}
		}
		filteredSessions = append(filteredSessions, session)
	}
	return filteredSessions, nil
}

// AsTerminatableUpload returns a TerminatableUpload
// To implement the termination extension as specified in https://tus.io/protocols/resumable-upload.html#termination
// the storage needs to implement AsTerminatableUpload
func (fs *Decomposedfs) AsTerminatableUpload(up tusd.Upload) tusd.TerminatableUpload {
	return up.(*upload.OcisSession)
}

// AsLengthDeclarableUpload returns a LengthDeclarableUpload
// To implement the creation-defer-length extension as specified in https://tus.io/protocols/resumable-upload.html#creation
// the storage needs to implement AsLengthDeclarableUpload
func (fs *Decomposedfs) AsLengthDeclarableUpload(up tusd.Upload) tusd.LengthDeclarableUpload {
	return up.(*upload.OcisSession)
}

// AsConcatableUpload returns a ConcatableUpload
// To implement the concatenation extension as specified in https://tus.io/protocols/resumable-upload.html#concatenation
// the storage needs to implement AsConcatableUpload
func (fs *Decomposedfs) AsConcatableUpload(up tusd.Upload) tusd.ConcatableUpload {
	return up.(*upload.OcisSession)
}
