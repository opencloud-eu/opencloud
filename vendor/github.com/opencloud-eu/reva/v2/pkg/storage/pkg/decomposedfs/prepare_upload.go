// Copyright 2026 OpenCloud GmbH <mail@opencloud.eu>
// SPDX-License-Identifier: Apache-2.0

package decomposedfs

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	provider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	"github.com/pkg/errors"

	"github.com/opencloud-eu/reva/v2/pkg/appctx"
	ctxpkg "github.com/opencloud-eu/reva/v2/pkg/ctx"
	"github.com/opencloud-eu/reva/v2/pkg/errtypes"
	"github.com/opencloud-eu/reva/v2/pkg/storage"
	"github.com/opencloud-eu/reva/v2/pkg/storage/pkg/decomposedfs/metadata"
	"github.com/opencloud-eu/reva/v2/pkg/storage/pkg/decomposedfs/metadata/prefixes"
	"github.com/opencloud-eu/reva/v2/pkg/storage/pkg/decomposedfs/node"
	"github.com/opencloud-eu/reva/v2/pkg/storagespace"
)

// PrepareCreatesNode reports that PrepareUpload creates a new file's node.
func (fs *Decomposedfs) PrepareCreatesNode() bool {
	return true
}

// PrepareUpload finalizes node metadata after bytes are received, before postprocessing,
// and marks the node as processing for sessionID. A new file's node is created here, under
// the id the ref carries. CommitUpload is called after postprocessing completes.
//
// The previous version is snapshotted through the tree's CreateRevision, like the legacy
// upload path does: posix copies the current content, decomposedfs only the metadata. When
// another upload is still processing the node, it is snapshotted even with versioning
// disabled, together with that upload's processing mark: if this upload is rolled back, the
// other one is restored and can still finish.
func (fs *Decomposedfs) PrepareUpload(ctx context.Context, ref *provider.Reference, sessionID string, info storage.UploadInfo) (*storage.PrepareUploadResult, error) {
	ctx, span := tracer.Start(ctx, "PrepareUpload")
	defer span.End()

	n, err := fs.lu.NodeFromResource(ctx, ref)
	if err != nil {
		appctx.GetLogger(ctx).Error().Err(err).Msg("PrepareUpload: unexpected NodeFromResource failure")
		return nil, errtypes.InternalError("PrepareUpload: node lookup failed unexpectedly")
	}
	// a new file nobody has created yet: PrepareUpload creates it
	create := !n.Exists && !info.NodeExisted && info.ParentID != "" && info.Name != ""
	if !n.Exists && !create {
		return nil, errtypes.NotFound(ref.String())
	}
	if create {
		if err := fs.checkNewFileParent(ctx, n.SpaceID, info); err != nil {
			return nil, err
		}
		n.ParentID = info.ParentID
		n.Name = info.Name
	}

	var (
		committed bool
		// only a node this upload created is purged, one it is creating only once
		// the name is its own: Purge removes the name
		purge = !info.NodeExisted && !create
	)
	// registered before the lock is taken, so it runs once the lock is released.
	// Purge, not trash: the file never had content, and Purge needs no Delete permission
	defer func() {
		if committed || !purge {
			return
		}
		if err := n.Purge(ctx); err != nil {
			appctx.GetLogger(ctx).Error().Err(err).Str("nodeid", n.ID).Msg("could not purge new node after failed prepare")
		}
	}()

	n.SpaceRoot, err = node.ReadNode(ctx, fs.lu, n.SpaceID, n.SpaceID, "", false, nil, false)
	if err != nil {
		return nil, err
	}

	if create {
		// a node still to be created holds no lock, so any lock id is a mismatch
		if lockID, _ := ctxpkg.ContextGetLockID(ctx); lockID != "" {
			return nil, errtypes.Aborted("not locked")
		}
	} else if err := n.CheckLock(ctx); err != nil {
		return nil, err
	}

	// scope to space owner GID for posix deployments; no-op with NullMapper
	if spaceGID, ok := ctx.Value(CtxKeySpaceGID).(uint32); ok {
		unscope, err := fs.um.ScopeUserByIds(-1, int(spaceGID))
		if err != nil {
			return nil, errors.Wrap(err, "failed to scope user")
		}
		if unscope != nil {
			defer func() { _ = unscope() }()
		}
	}

	var unlock metadata.UnlockFunc
	if create {
		// InitNewNode refuses a name already taken, checks the quota, and on failure
		// removes what it made and releases its lock. It returns the node lock.
		unlock, err = fs.tp.InitNewNode(ctx, n, uint64(info.Size))
		if err != nil {
			return nil, err
		}
		purge = true
	} else {
		unlock, err = fs.lu.MetadataBackend().Lock(n)
		if err != nil {
			return nil, err
		}
	}
	defer func() {
		if err := unlock(); err != nil {
			appctx.GetLogger(ctx).Error().Err(err).Str("nodeid", n.ID).Msg("could not close lock")
		}
	}()

	var (
		sizeDiff       int64
		versionCreated bool
		versionNode    *node.BaseNode
		oldAttrs       node.Attributes
		oldMtime       time.Time
	)

	defer func() {
		if committed {
			return
		}
		if versionCreated {
			removeRevision(ctx, fs.lu, versionNode)
		}
		if info.NodeExisted && oldAttrs != nil {
			// mtime goes in the same batch: we still hold the metadata lock and
			// SetMTime would try to retake it
			if err := fs.lu.TimeManager().OverrideMtime(ctx, n, &oldAttrs, oldMtime); err != nil {
				appctx.GetLogger(ctx).Error().Err(err).Str("nodeid", n.ID).Msg("could not restore node mtime during rollback")
			}
			if err := n.SetXattrsWithContext(ctx, oldAttrs); err != nil {
				appctx.GetLogger(ctx).Error().Err(err).Str("nodeid", n.ID).Msg("could not restore node xattrs during rollback")
			}
			// restoring only rewrites the old keys, so the mark this upload added would stay
			if _, ok := oldAttrs[prefixes.StatusPrefix]; !ok {
				if err := n.RemoveXattr(ctx, prefixes.StatusPrefix); err != nil && !metadata.IsAttrUnset(err) {
					appctx.GetLogger(ctx).Error().Err(err).Str("nodeid", n.ID).Msg("could not remove processing mark during rollback")
				}
			}
		}
	}()

	var (
		old         *node.Node
		overwrite   bool
		oldBlobsize int64
	)
	if info.NodeExisted {
		old, err = node.ReadNode(ctx, fs.lu, n.SpaceID, n.ID, "", false, nil, false)
		if err != nil {
			return nil, errors.Wrap(err, "PrepareUpload: failed to read existing node")
		}
		overwrite = old.BlobID != ""
		oldBlobsize = old.Blobsize
	}

	// also for new files: the coordinator's check fails open without GetQuota
	// permission, and CheckQuota guards disk space too. InitNewNode has checked a
	// node it created.
	if !create {
		if _, err := node.CheckQuota(ctx, n.SpaceRoot, overwrite, uint64(oldBlobsize), uint64(info.Size)); err != nil {
			return nil, err
		}
	}

	if info.NodeExisted {
		oldMtime, err = old.GetMTime(ctx)
		if err != nil {
			return nil, err
		}
		oldEtag, err := node.CalculateEtag(old.ID, oldMtime)
		if err != nil {
			return nil, err
		}

		if info.IfMatch != "" && info.IfMatch != oldEtag {
			return nil, errtypes.Aborted("etag mismatch")
		}
		if info.IfNoneMatch != "" {
			if info.IfNoneMatch == "*" {
				return nil, errtypes.Aborted("etag mismatch, resource exists")
			}
			for _, tag := range strings.Split(info.IfNoneMatch, ",") {
				if tag == oldEtag {
					return nil, errtypes.Aborted("etag mismatch")
				}
			}
		}
		if !info.IfUnmodifiedSince.IsZero() && oldMtime.After(info.IfUnmodifiedSince) {
			return nil, errtypes.Aborted("if-unmodified-since mismatch")
		}

		// capture full node xattrs for rollback before any write
		oldAttrs, err = fs.lu.MetadataBackend().All(ctx, n)
		if err != nil {
			return nil, err
		}

		// another upload is still processing the node: keep its state, so a rollback
		// of this upload can hand the node back to it
		supersededMark, superseding := oldAttrs[prefixes.StatusPrefix]
		if !fs.o.DisableVersioning || superseding {
			timestamp := oldMtime.UTC().Format(time.RFC3339Nano)
			versionPath, err := fs.tp.CreateRevision(ctx, n, timestamp)
			if err != nil {
				return nil, err
			}
			versionNode = node.NewBaseNode(n.SpaceID, n.ID+node.RevisionIDDelimiter+timestamp, fs.lu)
			versionCreated = true
			if err := os.Chtimes(versionPath, oldMtime, oldMtime); err != nil {
				return nil, errtypes.InternalError(fmt.Sprintf("failed to set mtime on version node: %s", err))
			}
			if superseding {
				revisionAttrs := map[string][]byte{prefixes.StatusPrefix: supersededMark}
				if fs.o.DisableVersioning {
					// not a version, only kept for a rollback: a live commit removes it
					revisionAttrs[uploadSnapshotAttr] = []byte("1")
				}
				if err := fs.lu.MetadataBackend().SetMultiple(ctx, versionNode, revisionAttrs); err != nil {
					return nil, err
				}
			}
		}

		sizeDiff = info.Size - old.Blobsize
	} else {
		// InitNewNode has cached a node it created
		if c, ok := fs.lu.(node.IDCacher); ok && !create {
			if err := c.CacheID(ctx, n.SpaceID, n.ID, filepath.Join(n.ParentPath(), n.Name)); err != nil {
				appctx.GetLogger(ctx).Error().Err(err).Msg("failed to cache id")
			}
		}
		sizeDiff = info.Size
	}

	attrs := node.Attributes{}
	attrs.SetString(prefixes.IDAttr, n.ID)
	attrs.SetInt64(prefixes.TypeAttr, int64(provider.ResourceType_RESOURCE_TYPE_FILE))
	attrs.SetString(prefixes.ParentidAttr, n.ParentID)
	attrs.SetString(prefixes.NameAttr, n.Name)
	attrs.SetString(prefixes.BlobIDAttr, sessionID)
	attrs.SetInt64(prefixes.BlobsizeAttr, info.Size)
	attrs[prefixes.ChecksumPrefix+"sha1"] = info.Checksums.SHA1
	attrs[prefixes.ChecksumPrefix+"md5"] = info.Checksums.MD5
	attrs[prefixes.ChecksumPrefix+"adler32"] = info.Checksums.Adler32
	// in the same batch, so marking the node costs no write of its own
	attrs.SetString(prefixes.StatusPrefix, node.ProcessingStatus+sessionID)

	mtime := time.Now()
	if !info.MTime.IsZero() {
		mtime = info.MTime
	}
	if err := fs.lu.TimeManager().OverrideMtime(ctx, n, &attrs, mtime); err != nil {
		return nil, errors.Wrap(err, "failed to set mtime")
	}

	if err := n.SetXattrsWithContext(ctx, attrs); err != nil {
		return nil, errors.Wrap(err, "could not write metadata")
	}

	if err := fs.tp.Propagate(ctx, n, sizeDiff); err != nil {
		return nil, errors.Wrap(err, "could not propagate size change")
	}
	committed = true

	result := &storage.PrepareUploadResult{VersionCreated: versionCreated, SizeDiff: sizeDiff}
	if !info.NodeExisted {
		// read off the disk, not through ListGrants: an uploader who cannot see the
		// space root still needs the manager of a project space reported
		result.SpaceOwner = n.SpaceOwnerOrManager(ctx)
	}
	return result, nil
}

// checkNewFileParent re-checks, for a file PrepareUpload is about to create, what
// TouchFile would have checked: the parent still exists, and the user may still upload
// into it. Either may have changed while the bytes were in flight.
func (fs *Decomposedfs) checkNewFileParent(ctx context.Context, spaceID string, info storage.UploadInfo) error {
	parentID := &provider.ResourceId{SpaceId: spaceID, OpaqueId: info.ParentID}
	parent, err := fs.lu.NodeFromID(ctx, parentID)
	if err != nil {
		return errtypes.InternalError(err.Error())
	}
	f, _ := storagespace.FormatReference(&provider.Reference{ResourceId: parentID, Path: info.Name})
	if !parent.Exists {
		return errtypes.NotFound(f)
	}

	rp, err := fs.p.AssemblePermissions(ctx, parent)
	switch {
	case err != nil:
		return err
	case !rp.InitiateFileUpload:
		if rp.Stat {
			return errtypes.PermissionDenied(f)
		}
		return errtypes.NotFound(f)
	}
	return nil
}

// removeRevision deletes a revision PrepareUpload created, with its metadata, while the
// caller holds the node lock (Node.DeleteRevision would try to take it again). Its blob
// is kept: the revision shares it with the node, whose old metadata is being restored.
func removeRevision(ctx context.Context, lu node.PathLookup, revisionNode *node.BaseNode) {
	log := appctx.GetLogger(ctx).With().Str("revisionid", revisionNode.ID).Logger()
	if err := os.Remove(revisionNode.InternalPath()); err != nil && !errors.Is(err, os.ErrNotExist) {
		log.Error().Err(err).Msg("could not remove revision during rollback")
	}
	if err := lu.MetadataBackend().Purge(ctx, revisionNode); err != nil && !errors.Is(err, os.ErrNotExist) {
		log.Error().Err(err).Msg("could not purge revision metadata during rollback")
	}
	if err := os.Remove(lu.MetadataBackend().LockfilePath(revisionNode)); err != nil && !errors.Is(err, os.ErrNotExist) {
		log.Error().Err(err).Msg("could not remove revision lockfile during rollback")
	}
}

// RollbackUpload reverts the node state written by PrepareUpload after a failed or aborted
// postprocessing run. It restores the previous revision and reverts the optimistic size
// propagation. Unlike upstream it never purges an existing node: without a revision (versioning
// disabled) the previous metadata cannot be restored, so it returns an error instead. On posix
// this also keeps the file itself, which still holds the previous content until CommitUpload.
// When the node metadata has become unreadable the ids recorded in info are used to remove the
// node and release its quota.
func (fs *Decomposedfs) RollbackUpload(ctx context.Context, ref *provider.Reference, sessionID string, info storage.RollbackInfo) error {
	n, err := fs.lu.NodeFromResource(ctx, ref)
	if err != nil {
		// The node metadata is unreadable, so the upload can never finish and its
		// quota would stay consumed forever.
		return fs.rollbackOrphaned(ctx, ref, sessionID, info, err)
	}
	if !n.Exists {
		return nil // nothing was written yet
	}
	n.SpaceRoot, err = node.ReadNode(ctx, fs.lu, n.SpaceID, n.SpaceID, "", false, nil, false)
	if err != nil {
		return err
	}

	curProcessingID, err := n.ProcessingID(ctx)
	if err != nil {
		return fmt.Errorf("RollbackUpload: could not read processing ID: %w", err)
	}
	if curProcessingID != sessionID {
		// a newer upload took over the node. The revision it kept of this upload would
		// hand the node back to an upload that is gone.
		fs.retireUploadRevision(ctx, n, sessionID, info.NodeExisted)
		return nil
	}

	sizeDiff := info.SizeDiff
	if info.NodeExisted {
		restored, purge, err := fs.restoreLatestRevision(ctx, n)
		if err != nil {
			return err
		}
		if !restored {
			return errtypes.PreconditionFailed("RollbackUpload: no revision to restore for node " + n.ID)
		}
		if purge {
			// the file was created by an upload that was rolled back while this one
			// was processing, so nothing is left to keep
			if err := n.Purge(ctx); err != nil {
				return fmt.Errorf("RollbackUpload: could not purge node: %w", err)
			}
			sizeDiff += n.Blobsize
		}
	} else {
		// the upload created this node, so undo it. Purge, not trash: the file never
		// became visible, and Purge needs no Delete permission
		if err := n.Purge(ctx); err != nil {
			return fmt.Errorf("RollbackUpload: could not purge node: %w", err)
		}
	}

	if sizeDiff != 0 {
		if err := fs.tp.Propagate(ctx, n, -sizeDiff); err != nil {
			appctx.GetLogger(ctx).Error().Err(err).Msg("RollbackUpload: could not revert propagate")
		}
	}
	return nil
}

// restoreLatestRevision restores the node from the revision PrepareUpload created, its
// content included on posix, and removes the revision. If the revision kept the processing
// mark of an upload this one superseded, the mark is restored too, so that upload can still
// finish. It reports false if the node has no revision, and purge if the revision says the
// file should not exist anymore (see retireUploadRevision); n then describes the restored
// state, with the size the file accounts for.
func (fs *Decomposedfs) restoreLatestRevision(ctx context.Context, n *node.Node) (restored, purge bool, err error) {
	unlock, err := fs.lu.MetadataBackend().Lock(n)
	if err != nil {
		return false, false, err
	}
	defer func() {
		if err := unlock(); err != nil {
			appctx.GetLogger(ctx).Error().Err(err).Str("nodeid", n.ID).Msg("could not close lock")
		}
	}()

	key, err := n.LatestRevisionKey(ctx)
	if err != nil || key == "" {
		return false, false, err
	}
	revisionNode := node.NewBaseNode(n.SpaceID, n.ID+node.RevisionIDDelimiter+key, fs.lu)
	revisionAttrs, err := fs.lu.MetadataBackend().All(ctx, revisionNode)
	if err != nil {
		return false, false, err
	}

	var mtime time.Time
	if v, ok := revisionAttrs[prefixes.MTimeAttr]; ok {
		if mtime, err = time.Parse(time.RFC3339Nano, string(v)); err != nil {
			return false, false, err
		}
	}
	if err := fs.tp.RestoreRevision(ctx, revisionNode, n, mtime); err != nil {
		return false, false, err
	}
	if mark, ok := revisionAttrs[prefixes.StatusPrefix]; ok {
		if err := n.SetXattrsWithContext(ctx, node.Attributes{prefixes.StatusPrefix: mark}); err != nil {
			return false, false, err
		}
	}
	if v, ok := revisionAttrs[prefixes.BlobsizeAttr]; ok {
		n.Blobsize, _ = strconv.ParseInt(string(v), 10, 64)
	}
	_, purge = revisionAttrs[purgeOnRestoreAttr]

	removeRevision(ctx, fs.lu, revisionNode)
	return true, purge, nil
}

// purgeOnRestoreAttr marks a revision kept for an upload that created the file and was
// rolled back while a newer upload was processing: restoring it removes the file.
var purgeOnRestoreAttr = prefixes.OcPrefix + "purgeonrestore"

// uploadSnapshotAttr marks a revision PrepareUpload kept with versioning disabled, only so
// that a rollback can hand the node back to the upload it superseded. It is not a version.
var uploadSnapshotAttr = prefixes.OcPrefix + "uploadsnapshot"

// retireUploadRevision handles the revision a newer upload kept of the upload sessionID,
// which is being rolled back. If that upload overwrote the file, the revision is dropped:
// the one it kept itself describes the file before it. If it created the file, there is
// nothing before it, so the revision is marked to remove the file should the newer upload
// be rolled back too.
func (fs *Decomposedfs) retireUploadRevision(ctx context.Context, n *node.Node, sessionID string, nodeExisted bool) {
	log := appctx.GetLogger(ctx).With().Str("nodeid", n.ID).Str("sessionid", sessionID).Logger()
	unlock, err := fs.lu.MetadataBackend().Lock(n)
	if err != nil {
		log.Error().Err(err).Msg("could not lock node")
		return
	}
	defer func() {
		if err := unlock(); err != nil {
			log.Error().Err(err).Msg("could not close lock")
		}
	}()

	revision, revisionUnlock, err := fs.uploadRevision(ctx, n, sessionID)
	if err != nil {
		log.Error().Err(err).Msg("could not look up the upload's revision")
		return
	}
	if revision == nil {
		return
	}
	defer func() { _ = revisionUnlock() }()

	if nodeExisted {
		removeRevision(ctx, fs.lu, &revision.BaseNode)
		return
	}
	if err := revision.SetXattrsWithContext(ctx, node.Attributes{purgeOnRestoreAttr: []byte("1")}); err != nil {
		log.Error().Err(err).Msg("could not mark the upload's revision")
	}
	if err := revision.RemoveXattr(ctx, prefixes.StatusPrefix); err != nil && !metadata.IsAttrUnset(err) {
		log.Error().Err(err).Msg("could not remove the processing mark from the upload's revision")
	}
}

// rollbackOrphaned releases the quota of an upload whose node metadata is unreadable, using
// the node and parent ids the upload session recorded.
func (fs *Decomposedfs) rollbackOrphaned(ctx context.Context, ref *provider.Reference, sessionID string, info storage.RollbackInfo, lookupErr error) error {
	spaceID := ref.GetResourceId().GetSpaceId()
	if info.NodeID == "" || info.ParentID == "" || spaceID == "" {
		return fmt.Errorf("RollbackUpload: node lookup failed: %w", lookupErr)
	}
	log := appctx.GetLogger(ctx).With().Str("sessionid", sessionID).Str("nodeid", info.NodeID).Logger()
	log.Info().Err(lookupErr).Msg("node unreadable, rolling back orphaned upload")

	n := node.New(spaceID, info.NodeID, info.ParentID, info.Filename, info.Size, sessionID,
		provider.ResourceType_RESOURCE_TYPE_FILE, nil, fs.lu)
	spaceRoot, err := node.ReadNode(ctx, fs.lu, spaceID, spaceID, "", false, nil, false)
	if err != nil {
		return fmt.Errorf("RollbackUpload: space root lookup failed: %w", err)
	}
	n.SpaceRoot = spaceRoot

	if info.SizeDiff != 0 {
		// an error keeps the session, so the rollback can be retried
		if err := fs.tp.Propagate(ctx, n, -info.SizeDiff); err != nil {
			return fmt.Errorf("RollbackUpload: could not revert propagate: %w", err)
		}
	}

	// the orphaned node cannot be resolved by any other means, remove it together
	// with its metadata. A missing node is fine: it may never have been created.
	if err := fs.lu.PurgeNode(n); err != nil && !errors.Is(err, os.ErrNotExist) {
		log.Error().Err(err).Str("nodepath", n.InternalPath()).Msg("RollbackUpload: removing orphaned node failed")
	}
	if err := fs.lu.MetadataBackend().Purge(ctx, n); err != nil && !errors.Is(err, os.ErrNotExist) {
		log.Error().Err(err).Str("nodepath", n.InternalPath()).Msg("RollbackUpload: purging orphaned node metadata failed")
	}
	return nil
}
