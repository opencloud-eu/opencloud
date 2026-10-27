// Copyright 2026 OpenCloud GmbH <mail@opencloud.eu>
// SPDX-License-Identifier: Apache-2.0

package upload

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	user "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"
	provider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
	tusd "github.com/tus/tusd/v2/pkg/handler"

	"github.com/opencloud-eu/reva/v2/pkg/appctx"
	ctxpkg "github.com/opencloud-eu/reva/v2/pkg/ctx"
	"github.com/opencloud-eu/reva/v2/pkg/errtypes"
	"github.com/opencloud-eu/reva/v2/pkg/events"
	"github.com/opencloud-eu/reva/v2/pkg/rhttp/datatx/metrics"
	"github.com/opencloud-eu/reva/v2/pkg/storage"
	"github.com/opencloud-eu/reva/v2/pkg/storage/utils/chunking"
	"github.com/opencloud-eu/reva/v2/pkg/utils"
)

// Request carries the metadata of a non-resumable (PUT) upload.
type Request struct {
	Ref    *provider.Reference
	Body   io.ReadCloser
	Length int64
}

// FinishedFunc is called once an upload has finished, so that the caller can
// publish a FileUploaded event through its own publisher.
type FinishedFunc func(spaceOwner, executant *user.UserId, ref *provider.Reference)

// Coordinator owns the upload lifecycle: initiation, data transfer and listing.
type Coordinator interface {
	// InitiateUpload returns the protocols and ids that bytes can be appended to.
	InitiateUpload(ctx context.Context, ref *provider.Reference, uploadLength int64, metadata map[string]string) (map[string]string, error)
	// GetUpload returns the session with the given id as a tusd upload.
	GetUpload(ctx context.Context, id string) (tusd.Upload, error)
	// UseIn registers the coordinator as the tusd data store.
	UseIn(composer *tusd.StoreComposer)
	// ListUploadSessions returns the upload sessions matching the given filter.
	ListUploadSessions(ctx context.Context, filter storage.UploadSessionFilter) ([]storage.UploadSession, error)
	// Upload writes the whole body of a non-resumable (PUT) upload and finishes it.
	Upload(ctx context.Context, req Request, uff FinishedFunc) (*provider.ResourceInfo, error)
	// StartPostprocessing subscribes to postprocessing results and enables async
	// uploads. Call once, before serving requests.
	StartPostprocessing(stream events.Consumer, ac AsyncConf) error
	// ResolveUpload stats the resource a finished upload produced, as the upload's executant.
	storage.UploadSessionResolver
}

const maxCommitRetryBackoff = 2 * time.Minute

// coordinator is the concrete implementation of Coordinator.
type coordinator struct {
	fs           storage.FS
	store        SessionStore
	chunkHandler *chunking.ChunkHandler
	pub          events.Publisher
	// async defers the commit to postprocessing. Only StartPostprocessing sets it.
	async bool
	// mountID is the storage this coordinator serves, used to drop postprocessing
	// events belonging to another one.
	mountID            string
	commitMaxRetries   int
	commitRetryBackoff time.Duration
}

// NewCoordinator constructs a coordinator backed by the given driver and store.
func NewCoordinator(fs storage.FS, store SessionStore, chunkFolder string, pub events.Publisher) *coordinator {
	c := &coordinator{fs: fs, store: store, pub: pub}
	if chunkFolder != "" {
		c.chunkHandler = chunking.NewChunkHandler(chunkFolder)
	}
	return c
}

// ResolveUpload stats the resource produced by a finished upload, reading it as the upload's
// executant. The session is reconstructed from the given tus FileInfo rather than the session
// store, so it still resolves after a synchronous upload has cleaned up its session. It returns
// the resource info and the executant context, so the caller can derive identity dependent
// values such as the WebDAV permissions. https://github.com/opencloud-eu/opencloud/issues/2409
func (c *coordinator) ResolveUpload(ctx context.Context, info tusd.FileInfo) (*provider.ResourceInfo, context.Context, error) {
	session := c.store.FromInfo(info)
	ctx = session.Context(ctx)
	ref := session.Reference()
	ri, err := c.fs.GetMD(ctx, &ref, nil, nil)
	return ri, ctx, err
}

// NewCoordinatorFromConfig sets up a coordinator and its file store. Pass
// withChunking=false when only initiating uploads: chunk assembly happens on the
// data path, so the storageprovider doesn't need it.
func NewCoordinatorFromConfig(uploadDir string, driverConf map[string]interface{}, fs storage.FS, pub events.Publisher, log *zerolog.Logger, withChunking bool) (Coordinator, error) {
	store := NewFileStoreFromConfig(uploadDir, driverConf, log)
	if store == nil {
		return nil, fmt.Errorf("cannot determine the upload directory, set upload_directory")
	}
	if err := store.Setup(); err != nil {
		return nil, fmt.Errorf("upload directory setup failed: %w", err)
	}
	chunkFolder := ""
	if withChunking {
		chunkFolder = store.UploadDir()
	}
	return NewCoordinator(fs, store, chunkFolder, pub), nil
}

// InitiateUpload resolves the target, then creates and persists the session that
// bytes are appended to.
func (c *coordinator) InitiateUpload(ctx context.Context, ref *provider.Reference, uploadLength int64, metadata map[string]string) (map[string]string, error) {
	t, err := c.resolveTarget(ctx, ref, uploadLength, metadata)
	if err != nil {
		return nil, err
	}

	session := c.store.New(ctx)
	if err := c.populateSession(ctx, session, t, uploadLength, metadata); err != nil {
		return nil, err
	}

	if err := session.TouchBin(); err != nil {
		return nil, fmt.Errorf("coordinator: could not create bin file: %w", err)
	}
	if err := session.Persist(ctx); err != nil {
		session.Cleanup(ctx, true, true)
		return nil, fmt.Errorf("coordinator: could not persist session: %w", err)
	}

	metrics.UploadSessionsInitiated.Inc()

	// A zero-length upload has no bytes to append, so finish it here.
	if uploadLength == 0 {
		if _, err := c.finishUpload(ctx, session); err != nil {
			return nil, err
		}
	}

	return map[string]string{
		"simple": session.ID(),
		"tus":    session.ID(),
	}, nil
}

// populateSession writes the resolved target and the request's metadata into a
// new session.
func (c *coordinator) populateSession(ctx context.Context, session Session, t *uploadTarget, uploadLength int64, metadata map[string]string) error {
	session.SetMetadata("filename", t.name)
	session.SetStorageValue("NodeName", t.name)
	session.SetMetadata("dir", t.dir)
	session.SetStorageValue("Dir", t.dir)
	session.SetStorageValue("SpaceRoot", t.spaceID)
	session.SetStorageValue("NodeParentId", t.parentID)

	if t.exists {
		session.SetStorageValue("NodeId", t.nodeID)
		session.SetStorageValue("NodeExists", "true")
	} else {
		// A new file has no id yet, and an empty one would resolve to the space root
		// (lookup.go:191), so mint a placeholder TouchFile replaces.
		session.SetStorageValue("NodeId", uuid.New().String())
	}

	if t.spaceOwner != nil {
		session.SetStorageValue("SpaceOwnerOrManager", t.spaceOwner.GetOpaqueId())
		session.SetStorageValue("SpaceOwnerIdp", t.spaceOwner.GetIdp())
		session.SetStorageValue("SpaceOwnerType", utils.UserTypeToString(t.spaceOwner.GetType()))
	}
	if t.chunkName != "" {
		session.SetStorageValue("Chunk", t.chunkName)
	}

	// The ctx is gone by the time the bytes arrive, so record the actor now.
	session.SetExecutant(ctxpkg.ContextMustGetUser(ctx))
	lockID, _ := ctxpkg.ContextGetLockID(ctx)
	session.SetMetadata("lockid", lockID)
	initiatorID, _ := ctxpkg.ContextGetInitiator(ctx)
	session.SetMetadata("initiatorid", initiatorID)

	session.SetSize(uploadLength)

	return c.applyRequestMetadata(session, metadata)
}

// applyRequestMetadata stores the client's upload metadata, rejecting a malformed
// checksum. Conditional headers are only recorded, and evaluated at finish.
func (c *coordinator) applyRequestMetadata(session Session, metadata map[string]string) error {
	mtime, ok := metadata["mtime"]
	if !ok || mtime == "null" {
		mtime = utils.TimeToOCMtime(time.Now())
	}
	session.SetMetadata("mtime", mtime)

	session.SetMetadata("providerID", metadata["providerID"])
	if v, ok := metadata["expires"]; ok && v != "null" {
		session.SetMetadata("expires", v)
	}
	if _, ok := metadata["sizedeferred"]; ok {
		session.SetSizeIsDeferred(true)
	}
	for _, key := range []string{"if-match", "if-none-match", "if-unmodified-since"} {
		if v := metadata[key]; v != "" {
			session.SetMetadata(key, v)
		}
	}

	checksum, ok := metadata["checksum"]
	if !ok {
		return nil
	}
	parts := strings.SplitN(checksum, " ", 2)
	if len(parts) != 2 {
		return errtypes.BadRequest("invalid checksum format. must be '[algorithm] [checksum]'")
	}
	switch parts[0] {
	case "sha1", "md5", "adler32":
		session.SetMetadata("checksum", checksum)
	default:
		return errtypes.BadRequest("unsupported checksum algorithm: " + parts[0])
	}
	return nil
}

// Upload writes the whole body of a non-resumable (PUT) upload and finishes it.
// req.Ref.Path carries the session id minted by InitiateUpload.
func (c *coordinator) Upload(ctx context.Context, req Request, uff FinishedFunc) (*provider.ResourceInfo, error) {
	// The request path arrives rooted, while session ids are stored unrooted.
	session, err := c.store.Get(ctx, strings.TrimPrefix(req.Ref.GetPath(), "/"))
	if err != nil {
		return nil, err
	}
	// Its bytes already arrived, and an async finish keeps the session for
	// postprocessing: a repeated PUT would append to them and finish it again.
	if session.Offset() > 0 {
		return nil, errtypes.Aborted("coordinator: upload " + session.ID() + " was already received")
	}

	// Behind the data gateway the request carries a transfer token, not the user.
	ctx = session.Context(ctx)

	if chunk := session.Chunk(); chunk != "" { // legacy chunking v1
		if c.chunkHandler == nil {
			return nil, errtypes.NotSupported("coordinator: chunked uploads require a chunk folder")
		}
		assembled, assembledSize, done, aErr := c.chunkHandler.Assemble(chunk, req.Body)
		if aErr != nil {
			return nil, aErr
		}
		if !done {
			// The bytes accumulate in the chunk folder, so this session holds nothing.
			session.Cleanup(ctx, true, true)
			return nil, errtypes.PartialContent(req.Ref.String())
		}
		defer assembled.Close()
		// The declared length covers only the final chunk.
		req.Body, req.Length = assembled, assembledSize
		session.SetSize(assembledSize)
	}

	size, err := session.WriteChunk(ctx, 0, req.Body)
	if err != nil {
		return nil, err
	}
	if size != req.Length {
		return nil, errtypes.PartialContent("coordinator: unexpected end of stream")
	}

	ri, err := c.finishUpload(ctx, session)
	if err != nil {
		return nil, err
	}

	if uff != nil {
		executant := session.Executant()
		uff(session.SpaceOwner(), &executant, c.uploadRef(session))
	}
	return ri, nil
}

// finishUpload creates the node, validates the staged bytes and commits them, or
// hands them to postprocessing.
func (c *coordinator) finishUpload(ctx context.Context, session Session) (*provider.ResourceInfo, error) {
	// a client that goes away must not leave the upload half finished
	ctx = context.WithoutCancel(ctx)
	// Only the staged bytes are read, so a mismatch has no node to undo.
	if err := verifyAndStoreChecksums(ctx, session); err != nil {
		session.Cleanup(ctx, true, true)
		return nil, err
	}
	// Read before the node exists, so bad request metadata has no node to undo either.
	info, err := uploadInfo(session)
	if err != nil {
		session.Cleanup(ctx, true, true)
		return nil, err
	}
	// A driver whose PrepareUpload creates a new file's node is told where it goes,
	// under the id minted at initiate. Any other is handed one TouchFile created.
	if !session.NodeExists() && c.prepareCreatesNode() {
		info.ParentID, info.Name = session.NodeParentID(), session.Filename()
	} else if err := c.touchNode(ctx, session); err != nil {
		return nil, err
	}

	metrics.UploadSessionsBytesReceived.Inc()

	if err := c.prepare(ctx, session, info); err != nil {
		return nil, err
	}

	// Without a publisher there is nothing to hand the bytes to, so commit inline.
	if c.async && c.pub != nil && session.Size() > 0 {
		if err := c.publishBytesReceived(ctx, session); err != nil {
			c.rollbackPrepared(ctx, session, session.SizeDiff())
			return nil, err
		}
		// The node stays flagged and the staged bytes are kept: postprocessing needs
		// both to finish later.
		return c.uploadedResourceInfo(ctx, session), nil
	}

	return c.finishSync(ctx, session)
}

// publishBytesReceived hands the staged upload to postprocessing, which downloads
// the bytes from a signed URL to scan them before they are committed.
func (c *coordinator) publishBytesReceived(ctx context.Context, session Session) error {
	url, err := session.URL(ctx)
	if err != nil {
		return err
	}

	return events.Publish(ctx, c.pub, events.BytesReceived{
		UploadID:      session.ID(),
		URL:           url,
		SpaceOwner:    session.SpaceOwner(),
		ExecutingUser: session.ExecutantUser(),
		ResourceID: &provider.ResourceId{
			StorageId: session.ProviderID(),
			SpaceId:   session.SpaceID(),
			OpaqueId:  session.NodeID(),
		},
		Filename:          session.Filename(),
		Filesize:          uint64(session.Size()),
		ImpersonatingUser: impersonatingUser(ctx),
	})
}

// prepareCreatesNode reports whether the driver's PrepareUpload creates a new
// file's node, sparing the extra metadata write a TouchFile costs.
func (c *coordinator) prepareCreatesNode() bool {
	nc, ok := c.fs.(storage.NodeCreator)
	return ok && nc.PrepareCreatesNode()
}

// touchNode creates the node a new file's upload writes to.
func (c *coordinator) touchNode(ctx context.Context, session Session) error {
	if session.NodeExists() {
		return nil
	}

	// The node has no id yet, so address it as a named child of its parent.
	pathRef := &provider.Reference{
		ResourceId: &provider.ResourceId{
			SpaceId:  session.SpaceID(),
			OpaqueId: session.NodeParentID(),
		},
		Path: session.Filename(),
	}
	// PrepareUpload marks and propagates the node, so TouchFile need not.
	err := c.fs.TouchFile(storage.ContextSkipTouchPropagation(ctx), pathRef, false, session.Metadata()["mtime"])
	if err == nil {
		// TouchFile does not report the node it created, read it back.
		var ri *provider.ResourceInfo
		ri, err = c.fs.GetMD(ctx, pathRef, nil, nil)
		if err == nil {
			// Replaces the placeholder minted at initiate.
			session.SetStorageValue("NodeId", ri.GetId().GetOpaqueId())
			session.SetStorageValue("SpaceRoot", ri.GetId().GetSpaceId())
			if ri.GetOwner() != nil {
				if owner := c.spaceOwnerOrManager(ctx, ri.GetOwner(), ri.GetId().GetSpaceId()); owner != nil {
					session.SetStorageValue("SpaceOwnerOrManager", owner.GetOpaqueId())
					session.SetStorageValue("SpaceOwnerIdp", owner.GetIdp())
					session.SetStorageValue("SpaceOwnerType", utils.UserTypeToString(owner.GetType()))
				}
			}
			return nil
		}
	}
	session.Cleanup(ctx, true, true)
	if _, ok := err.(errtypes.IsNotFound); ok {
		// The parent went away, or the share was revoked, while bytes were in flight.
		return errtypes.PreconditionFailed(err.Error())
	}
	return err
}

// prepare has the driver write the node metadata, mark it as processing so clients
// do not read it before its bytes are committed, and snapshot the previous version,
// ahead of the commit that writes the bytes.
func (c *coordinator) prepare(ctx context.Context, session Session, info storage.UploadInfo) error {
	ref := session.Reference()

	// A failed PrepareUpload has already undone its own writes, a new file's node
	// included, and marks the node only on success: there is nothing to unmark.
	prepared, err := c.fs.PrepareUpload(ctx, &ref, session.ID(), info)
	if err != nil {
		session.Cleanup(ctx, true, true)
		if _, ok := err.(errtypes.IsNotFound); ok && info.ParentID != "" {
			// The parent went away, or the share was revoked, while bytes were in flight.
			return errtypes.PreconditionFailed(err.Error())
		}
		return err
	}
	metrics.UploadProcessing.Inc()

	// Persisted, not just held: the commit may run in another process, which can
	// only learn these, and the real node id from touchNode, by reading them back.
	// Saved once, here: a failure now finds the node marked, so the rollback can
	// purge it.
	if prepared.SpaceOwner != nil {
		session.SetStorageValue("SpaceOwnerOrManager", prepared.SpaceOwner.GetOpaqueId())
		session.SetStorageValue("SpaceOwnerIdp", prepared.SpaceOwner.GetIdp())
		session.SetStorageValue("SpaceOwnerType", utils.UserTypeToString(prepared.SpaceOwner.GetType()))
	}
	session.SetSizeDiff(prepared.SizeDiff)
	session.SetVersionCreated(prepared.VersionCreated)
	session.SetStatus(SessionStatusProcessing, "")
	if err := session.Persist(ctx); err != nil {
		c.rollbackPrepared(ctx, session, prepared.SizeDiff)
		return err
	}
	return nil
}

// rollbackInfo describes the upload to undo. The ids come from the session, which
// is the only place they survive a node whose own metadata has become unreadable.
func rollbackInfo(session Session, sizeDiff int64) storage.RollbackInfo {
	return storage.RollbackInfo{
		NodeExisted: session.NodeExists(),
		SizeDiff:    sizeDiff,
		NodeID:      session.NodeID(),
		ParentID:    session.NodeParentID(),
		Filename:    session.Filename(),
		Size:        session.Size(),
	}
}

// rollbackPrepared undoes a finish that failed after PrepareUpload succeeded.
func (c *coordinator) rollbackPrepared(ctx context.Context, session Session, sizeDiff int64) {
	ref := session.Reference()
	// Before unmarking: RollbackUpload keys off the processing id to confirm the
	// node is still this upload's.
	if err := c.fs.RollbackUpload(ctx, &ref, session.ID(), rollbackInfo(session, sizeDiff)); err != nil {
		// The session is the last record of what to undo, so keep it for a retry
		// rather than leaving the quota consumed with nothing to reclaim it from.
		appctx.GetLogger(ctx).Error().Err(err).Str("uploadid", session.ID()).Msg("could not roll back upload, keeping session")
		return
	}
	c.unmarkProcessing(ctx, session, &ref)
	metrics.UploadProcessing.Dec()
	session.Cleanup(ctx, true, true)
}

// unmarkProcessing clears the processing flag, tolerating a node the rollback
// already purged.
func (c *coordinator) unmarkProcessing(ctx context.Context, session Session, ref *provider.Reference) {
	err := c.fs.MarkProcessing(ctx, ref, false, session.ID())
	log := appctx.GetLogger(ctx).With().Str("uploadid", session.ID()).Logger()
	switch err.(type) {
	case nil:
	case errtypes.IsNotFound:
		// RollbackUpload purged the node this upload created.
		log.Debug().Err(err).Msg("could not unmark processing")
	default:
		log.Error().Err(err).Msg("could not unmark processing")
	}
}

// uploadInfo collects what the driver needs to write the node metadata. The
// preconditions are re-checked there, because the resource may have changed while
// the bytes were being uploaded.
func uploadInfo(session Session) (storage.UploadInfo, error) {
	md := session.Metadata()
	info := storage.UploadInfo{
		NodeExisted: session.NodeExists(),
		Size:        session.Size(),
		Checksums:   session.Checksums(),
		IfMatch:     md["if-match"],
		IfNoneMatch: md["if-none-match"],
	}
	if v := md["mtime"]; v != "" {
		mtime, err := utils.MTimeToTime(v)
		if err != nil {
			return info, errtypes.BadRequest("coordinator: invalid mtime: " + v)
		}
		info.MTime = mtime
	}
	if v := md["if-unmodified-since"]; v != "" {
		ius, err := time.Parse(time.RFC3339Nano, v)
		if err != nil {
			return info, errtypes.BadRequest("coordinator: invalid if-unmodified-since: " + v)
		}
		info.IfUnmodifiedSince = ius
	}
	return info, nil
}

// verifyAndStoreChecksums hashes the staged bytes, checks them against the
// checksum the client announced, and stores them for the commit.
func verifyAndStoreChecksums(ctx context.Context, session Session) error {
	sha1h, md5h, adler32h, err := calculateChecksums(ctx, session.BinPath())
	if err != nil {
		return err
	}

	// The announced checksum is not in Metadata(), so read the raw info.
	info, err := session.GetInfo(ctx)
	if err != nil {
		return err
	}
	if checksum := info.MetaData["checksum"]; checksum != "" {
		parts := strings.SplitN(checksum, " ", 2)
		if len(parts) != 2 {
			return errtypes.BadRequest("invalid checksum format. must be '[algorithm] [checksum]'")
		}
		var checkErr error
		switch parts[0] {
		case "sha1":
			checkErr = checkHash(parts[1], sha1h)
		case "md5":
			checkErr = checkHash(parts[1], md5h)
		case "adler32":
			checkErr = checkHash(parts[1], adler32h)
		default:
			checkErr = errtypes.BadRequest("unsupported checksum algorithm: " + parts[0])
		}
		if checkErr != nil {
			return checkErr
		}
	}

	session.SetChecksums(sha1h.Sum(nil), md5h.Sum(nil), adler32h.Sum(nil))
	return nil
}

// finishSync commits the staged bytes for an inline upload, discarding the
// session if the commit fails: the client is still waiting and will retry, so
// keeping the bytes would only leak them.
func (c *coordinator) finishSync(ctx context.Context, session Session) (*provider.ResourceInfo, error) {
	ri, err := c.commit(ctx, session)
	if err != nil {
		// Keep the staged bytes: they are the only copy of the upload, and the
		// failure may be transient.
		c.rollbackNode(ctx, session, "commit failed: "+err.Error())
		metrics.UploadProcessing.Dec()
		return nil, err
	}
	return ri, nil
}

// finishAsync commits the staged bytes for an upload postprocessing has cleared,
// retrying transient failures with capped exponential backoff. On exhaustion
// the caller is responsible for rolling back the node and publishing the failure.
func (c *coordinator) finishAsync(ctx context.Context, session Session) error {
	maxAttempts := c.commitMaxRetries + 1
	backoff := c.commitRetryBackoff
	var err error
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		if _, err = c.commit(ctx, session); err == nil {
			return nil
		}
		if attempt == maxAttempts {
			break
		}
		backoff = min(backoff, maxCommitRetryBackoff)
		appctx.GetLogger(ctx).Warn().Err(err).
			Int("attempt", attempt).Int("maxAttempts", maxAttempts).
			Dur("backoff", backoff).
			Str("uploadid", session.ID()).
			Msg("blob commit failed, retrying after backoff")
		t := time.NewTimer(backoff)
		select {
		case <-ctx.Done():
			t.Stop()
			return ctx.Err()
		case <-t.C:
		}
		backoff *= 2
	}
	return err
}

// commit writes the staged bytes through the driver seam and, on success, unmarks
// the node and retires the session. It leaves a failure untouched for the caller
// to handle, the two finish paths wanting opposite things.
func (c *coordinator) commit(ctx context.Context, session Session) (*provider.ResourceInfo, error) {
	ref := session.Reference()

	f, err := os.Open(session.BinPath())
	if err != nil {
		return nil, err
	}
	// The verdict rides along with the bytes, so a committed blob always carries
	// the one it was cleared under. Empty on the inline path, which never scans.
	scanResult, scanDate := session.ScanData()

	// CommitUpload does not own the body; we opened it, so we close it. The path
	// lets a driver that can move the staged file into place skip the copy.
	err = c.fs.CommitUpload(ctx, &ref, session.ID(), storage.UploadSource{
		Body:       f,
		Path:       session.BinPath(),
		Length:     session.Size(),
		ScanResult: scanResult,
		ScanDate:   scanDate,
	})
	f.Close()
	if err != nil {
		return nil, err
	}

	// Not unmarkProcessing: the commit succeeded, so a missing node is unexpected.
	if err := c.fs.MarkProcessing(ctx, &ref, false, session.ID()); err != nil {
		appctx.GetLogger(ctx).Error().Err(err).Str("uploadid", session.ID()).Msg("could not unmark processing")
	}
	metrics.UploadProcessing.Dec()
	session.Cleanup(ctx, true, true)
	metrics.UploadSessionsFinalized.Inc()

	ri := c.uploadedResourceInfo(ctx, session)
	c.publishUploadReady(ctx, session, ri)
	return ri, nil
}

// uploadedResourceInfo describes the uploaded resource for the caller, which turns
// it into PUT response headers.
//
// TODO: the fallback carries no etag, which a write-only share used to get.
// Returning it through the seam is the proper fix.
func (c *coordinator) uploadedResourceInfo(ctx context.Context, session Session) *provider.ResourceInfo {
	ref := session.Reference()
	// The driver owns the resulting etag and mtime, so read them back.
	ri, err := c.fs.GetMD(ctx, &ref, nil, nil)
	if err == nil {
		// Drivers do not know their own mount id and the dataprovider does not stamp
		// it, so an unstamped id would not resolve for the client.
		if ri.GetId().GetStorageId() == "" {
			ri.Id.StorageId = session.ProviderID()
		}
		return ri
	}

	// GetMD is permission-gated, so a write-only share cannot stat what it just
	// uploaded. That must not fail the upload.
	appctx.GetLogger(ctx).Debug().Err(err).Str("uploadid", session.ID()).Msg("could not stat uploaded resource")
	fallback := &provider.ResourceInfo{Id: ref.GetResourceId(), Size: uint64(session.Size())}
	if mtime, mErr := utils.MTimeToTime(session.Metadata()["mtime"]); mErr == nil {
		fallback.Mtime = utils.TimeToTS(mtime)
	}
	return fallback
}

// publishUploadReady announces that the file is available. Consumers such as the
// search indexer only listen for UploadReady, so without it they would never learn
// about an upload the coordinator committed inline.
func (c *coordinator) publishUploadReady(ctx context.Context, session Session, ri *provider.ResourceInfo) {
	if c.pub == nil {
		return
	}
	if err := events.Publish(ctx, c.pub, events.UploadReady{
		UploadID:      session.ID(),
		Filename:      session.Filename(),
		SpaceOwner:    session.SpaceOwner(),
		ExecutingUser: session.ExecutantUser(),
		FileRef:       c.uploadRef(session),
		ResourceID:    ri.GetId(),
		ParentID: &provider.ResourceId{
			StorageId: session.ProviderID(),
			SpaceId:   session.SpaceID(),
			OpaqueId:  session.NodeParentID(),
		},
		Timestamp:         utils.TSNow(),
		IsVersion:         session.VersionCreated(),
		ImpersonatingUser: impersonatingUser(ctx),
	}); err != nil {
		appctx.GetLogger(ctx).Error().Err(err).Str("uploadid", session.ID()).Msg("failed to publish UploadReady event")
	}
}

// uploadTarget is what an upload needs to know about its destination
type uploadTarget struct {
	// exists distinguishes an overwrite from a new file.
	exists bool
	// chunkName is set for legacy chunking-v1 uploads only.
	chunkName string

	nodeID     string
	spaceID    string
	parentID   string
	dir        string
	name       string
	spaceOwner *user.UserId
}

// resolveTarget locates the file an upload writes to, and rejects an upload that
// may not proceed.
func (c *coordinator) resolveTarget(ctx context.Context, ref *provider.Reference, uploadLength int64, metadata map[string]string) (*uploadTarget, error) {
	t := &uploadTarget{}
	if chunking.IsChunked(ref.GetPath()) { // check legacy chunking v1
		var err error
		ref, t.chunkName, err = rewriteChunkedRef(ref)
		if err != nil {
			return nil, err
		}
	}

	// TODO: GetMD reports a file hidden by a deny-grant as NotFound, so the upload
	// fails late with 409 instead of 403. Needs a permission-free resolve.
	existing, err := c.fs.GetMD(ctx, ref, []string{}, []string{})
	switch err.(type) {
	case nil:
		t.exists = true
	case errtypes.IsNotFound:
		t.exists = false
	default:
		return nil, err
	}

	if err := c.checkQuota(ctx, ref, uploadLength, existing); err != nil {
		return nil, err
	}

	if t.exists {
		if err := c.describeExisting(ctx, ref, existing, metadata, t); err != nil {
			return nil, err
		}
	} else if err := c.describeNew(ctx, ref, t); err != nil {
		return nil, err
	}

	if t.name == "" {
		return nil, errtypes.BadRequest("coordinator: missing filename in ref")
	}
	if t.dir == "" {
		return nil, errtypes.BadRequest("coordinator: could not determine upload directory")
	}
	return t, nil
}

// checkQuota rejects an upload that would not fit in the space, counting only
// the bytes it adds.
func (c *coordinator) checkQuota(ctx context.Context, ref *provider.Reference, uploadLength int64, existing *provider.ResourceInfo) error {
	if uploadLength < 0 {
		return nil
	}
	spaceRef := &provider.Reference{ResourceId: &provider.ResourceId{
		StorageId: ref.GetResourceId().GetStorageId(),
		SpaceId:   ref.GetResourceId().GetSpaceId(),
	}}
	_, _, remaining, err := c.fs.GetQuota(ctx, spaceRef)
	if err != nil {
		return nil
	}

	netRequired := uint64(uploadLength)
	if existingSize := existing.GetSize(); existingSize < netRequired {
		netRequired -= existingSize
	} else {
		netRequired = 0
	}
	if remaining < netRequired {
		return errtypes.InsufficientStorage("quota exceeded")
	}
	return nil
}

// describeExisting fills in the target from the file being overwritten.
func (c *coordinator) describeExisting(ctx context.Context, ref *provider.Reference, existing *provider.ResourceInfo, metadata map[string]string, t *uploadTarget) error {
	if !existing.GetPermissionSet().GetInitiateFileUpload() {
		return errtypes.PermissionDenied(ref.GetPath())
	}
	if existing.GetType() == provider.ResourceType_RESOURCE_TYPE_CONTAINER {
		return errtypes.PreconditionFailed("resource is not a file")
	}

	t.nodeID = existing.GetId().GetOpaqueId()
	t.spaceID = existing.GetId().GetSpaceId()
	t.parentID = existing.GetParentId().GetOpaqueId()
	t.name = existing.GetName()
	t.spaceOwner = c.spaceOwnerOrManager(ctx, existing.GetOwner(), t.spaceID)

	// GetMD returns only the basename for id-based refs, so ask for the full path.
	relPath := existing.GetPath()
	if utils.IsRelativeReference(ref) {
		if full, err := c.fs.GetPathByID(ctx, existing.GetId()); err == nil {
			relPath = full
		}
	}
	t.dir = filepath.Dir(relPath)

	// Lock before precondition, the order main checks them in (upload.go:293-302).
	if err := c.checkLock(ctx, ref); err != nil {
		return err
	}
	if metadata["if-none-match"] == "*" {
		return errtypes.Aborted(fmt.Sprintf("parent %s already has a child %s, id %s", t.parentID, t.name, t.nodeID))
	}
	return nil
}

// spaceOwnerOrManager resolves the space owner, falling back to a manager.
func (c *coordinator) spaceOwnerOrManager(ctx context.Context, owner *user.UserId, spaceID string) *user.UserId {
	// A space with no personal owner stores a SPACE_OWNER placeholder (spaces.go:145).
	if owner != nil && owner.GetType() != user.UserType_USER_TYPE_SPACE_OWNER {
		return owner
	}

	grants, err := c.fs.ListGrants(ctx, &provider.Reference{ResourceId: &provider.ResourceId{
		SpaceId:  spaceID,
		OpaqueId: spaceID,
	}})
	if err != nil {
		return nil
	}

	// Grants come back in map order, so several managers resolve to an arbitrary one.
	for _, g := range grants {
		// Group grants carry no user id.
		uid := g.GetGrantee().GetUserId()
		if uid == nil {
			continue
		}
		p := g.GetPermissions()
		if p.GetStat() && p.GetListContainer() && p.GetInitiateFileDownload() {
			return uid
		}
	}
	return nil
}

// checkLock rejects an upload that conflicts with a lock held on the file.
func (c *coordinator) checkLock(ctx context.Context, ref *provider.Reference) error {
	// An unlocked file and a driver without locks both error, so only a lock we
	// actually read counts as one.
	diskLock, err := c.fs.GetLock(ctx, ref)
	if err != nil {
		diskLock = nil
	}
	contextLockID, _ := ctxpkg.ContextGetLockID(ctx)

	switch {
	case diskLock == nil && contextLockID != "":
		return errtypes.Aborted("not locked")
	case diskLock == nil:
		return nil
	case contextLockID == "":
		return errtypes.Locked(diskLock.LockId)
	case contextLockID != diskLock.LockId:
		return errtypes.Aborted("mismatching lock")
	}
	return nil
}

// describeNew fills in the target for a file that does not exist yet, from its
// parent directory.
func (c *coordinator) describeNew(ctx context.Context, ref *provider.Reference, t *uploadTarget) error {
	t.spaceID = ref.GetResourceId().GetSpaceId()
	t.dir = filepath.Dir(ref.GetPath())
	t.name = filepath.Base(ref.GetPath())

	parentRef := &provider.Reference{ResourceId: ref.GetResourceId(), Path: t.dir}
	parentMD, err := c.fs.GetMD(ctx, parentRef, []string{}, []string{})
	switch err.(type) {
	case nil:
	case errtypes.IsNotFound:
		return c.missingParentError(ctx, ref, t.dir, err)
	default:
		return err
	}

	if !parentMD.GetPermissionSet().GetInitiateFileUpload() {
		return errtypes.PermissionDenied(ref.GetPath())
	}
	t.parentID = parentMD.GetId().GetOpaqueId()
	t.spaceID = parentMD.GetId().GetSpaceId()

	// An id-based ref yields a relative dir ("."), so ask for the parent's full path.
	if utils.IsRelativeReference(ref) {
		if parentPath, pErr := c.fs.GetPathByID(ctx, parentMD.GetId()); pErr == nil {
			t.dir = parentPath
		}
	}
	return nil
}

// missingParentError tells a missing directory apart from one that is only
// invisible to the caller: a visible ancestor means it is missing.
func (c *coordinator) missingParentError(ctx context.Context, ref *provider.Reference, dir string, notFound error) error {
	for ancestor := dir; ancestor != "." && ancestor != "/"; {
		ancestor = filepath.Dir(ancestor)
		ancestorRef := &provider.Reference{ResourceId: ref.GetResourceId(), Path: ancestor}
		if _, err := c.fs.GetMD(ctx, ancestorRef, []string{}, []string{}); err == nil {
			return errtypes.PreconditionFailed(notFound.Error())
		}
	}
	return errtypes.PermissionDenied(ref.GetPath())
}

// rewriteChunkedRef turns a legacy chunking-v1 path into a reference to the real
// target file, plus the chunk name.
func rewriteChunkedRef(ref *provider.Reference) (*provider.Reference, string, error) {
	ci, err := chunking.GetChunkBLOBInfo(ref.GetPath())
	if err != nil {
		return nil, "", errtypes.BadRequest(err.Error())
	}
	return &provider.Reference{ResourceId: ref.ResourceId, Path: ci.Path}, filepath.Base(ref.GetPath()), nil
}

// ListUploadSessions returns the upload sessions matching the given filter.
func (c *coordinator) ListUploadSessions(ctx context.Context, filter storage.UploadSessionFilter) ([]storage.UploadSession, error) {
	// Only the driver can resolve a session's node, so the orphaned filter needs a
	// driver that answers the question. Assert once rather than per session.
	var orphanChecker storage.OrphanChecker
	if filter.Orphaned != nil {
		var ok bool
		if orphanChecker, ok = c.fs.(storage.OrphanChecker); !ok {
			return nil, errtypes.NotSupported("coordinator: the orphaned filter is not supported")
		}
	}

	var sessions []Session
	if filter.ID != nil && *filter.ID != "" {
		session, err := c.store.Get(ctx, *filter.ID)
		if err != nil {
			return nil, err
		}
		sessions = []Session{session}
	} else {
		var err error
		sessions, err = c.store.List(ctx)
		if err != nil {
			return nil, err
		}
	}

	filtered := []storage.UploadSession{}
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
		// evaluated last: unlike the other filters this reads the node metadata
		// from disk, so it is only done for sessions that passed all other filters
		if filter.Orphaned != nil {
			ref := session.Reference()
			if *filter.Orphaned != orphanChecker.IsOrphaned(ctx, &ref) {
				continue
			}
		}
		filtered = append(filtered, session)
	}
	return filtered, nil
}

// uploadRef builds the reference upload events carry: the space root, plus the
// file's path within it.
func (c *coordinator) uploadRef(session Session) *provider.Reference {
	return &provider.Reference{
		ResourceId: &provider.ResourceId{
			StorageId: session.ProviderID(),
			SpaceId:   session.SpaceID(),
			OpaqueId:  session.SpaceID(),
		},
		Path: utils.MakeRelativePath(filepath.Join(session.Dir(), session.Filename())),
	}
}

// impersonatingUser returns the real actor behind a borrowed identity, which
// public link and OCM auth record in the request user's opaque.
func impersonatingUser(ctx context.Context) *user.User {
	u, ok := ctxpkg.ContextGetUser(ctx)
	if !ok || !utils.ExistsInOpaque(u.GetOpaque(), "impersonating-user") {
		return nil
	}
	impersonating := &user.User{}
	if err := utils.ReadJSONFromOpaque(u.GetOpaque(), "impersonating-user", impersonating); err != nil {
		appctx.GetLogger(ctx).Error().Err(err).Msg("could not read impersonating user")
		return nil
	}
	return impersonating
}
