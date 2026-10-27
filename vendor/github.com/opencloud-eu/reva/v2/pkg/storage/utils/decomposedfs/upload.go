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
	iofs "io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	provider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	"github.com/opencloud-eu/reva/v2/pkg/storage/utils/decomposedfs/metadata/prefixes"
	"github.com/pkg/errors"
	"github.com/rogpeppe/go-internal/lockedfile"

	"github.com/opencloud-eu/reva/v2/pkg/appctx"
	ctxpkg "github.com/opencloud-eu/reva/v2/pkg/ctx"
	"github.com/opencloud-eu/reva/v2/pkg/errtypes"
	"github.com/opencloud-eu/reva/v2/pkg/storage"
	"github.com/opencloud-eu/reva/v2/pkg/storage/utils/decomposedfs/node"
	"github.com/opencloud-eu/reva/v2/pkg/storagespace"
	"github.com/opencloud-eu/reva/v2/pkg/utils"
)

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

	// on the node, not the session: the session is deleted when the upload finishes
	if !source.ScanDate.IsZero() {
		if err := n.SetScanData(ctx, source.ScanResult, source.ScanDate); err != nil {
			appctx.GetLogger(ctx).Error().Err(err).Str("nodeid", n.ID).Msg("could not set scan results")
		}
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

// PrepareUpload finalizes node metadata after bytes are received, before postprocessing,
// and marks the node as processing for sessionID. CommitUpload is called after
// postprocessing completes.
func (fs *Decomposedfs) PrepareUpload(ctx context.Context, ref *provider.Reference, sessionID string, info storage.UploadInfo) (*storage.PrepareUploadResult, error) {
	ctx, span := tracer.Start(ctx, "PrepareUpload")
	defer span.End()

	n, err := fs.lu.NodeFromResource(ctx, ref)
	if err != nil {
		appctx.GetLogger(ctx).Error().Err(err).Msg("PrepareUpload: unexpected NodeFromResource failure")
		return nil, errtypes.InternalError("PrepareUpload: node lookup failed unexpectedly")
	}
	// A new file nobody has created yet: PrepareUpload creates it, under the id the
	// ref already carries.
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

	committed := false
	// Only a node this upload created is purged, and one it is creating only once the
	// name is its own: Purge removes the name link by name.
	purge := !info.NodeExisted && !create
	// This upload created the node, so a failure must not leave it behind as an
	// empty file. Purge, not trash: it never had content, and Purge needs no
	// Delete permission. Registered before the lock is taken, so it runs once the
	// lock is released. Purge leaves the lock file alone: InitNewNode's unlock
	// removes it, a plain lockedfile's does not.
	defer func() {
		if committed || !purge {
			return
		}
		if err := n.Purge(ctx); err != nil {
			appctx.GetLogger(ctx).Error().Err(err).Str("nodeid", n.ID).Msg("could not purge new node after failed prepare")
		}
	}()

	n.SpaceRoot, err = node.ReadNode(ctx, fs.lu, n.SpaceID, n.SpaceID, false, nil, false)
	if err != nil {
		return nil, err
	}

	if create {
		// A node still to be created holds no lock, so any lock id is a mismatch, the
		// answer CheckLock would give, here without creating the node first.
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

	var (
		f      *lockedfile.File
		unlock func() error
	)
	if create {
		// InitNewNode refuses a name already taken, checks the quota, and on failure
		// removes what it made and releases its lock. Its lock is the one PrepareUpload
		// would take, so it is kept rather than taken again.
		unlock, err = fs.tp.InitNewNode(ctx, n, uint64(info.Size))
		if err != nil {
			return nil, err
		}
		purge = true
	} else {
		f, err = lockedfile.OpenFile(fs.lu.MetadataBackend().LockfilePath(n.InternalPath()), os.O_RDWR|os.O_CREATE, 0600)
		if err != nil {
			return nil, err
		}
		unlock = func() error { return f.Close() }
	}
	targetPath := n.InternalPath()
	defer func() {
		if err := unlock(); err != nil {
			appctx.GetLogger(ctx).Error().Err(err).Str("nodeid", n.ID).Msg("could not close lock")
		}
	}()

	var (
		sizeDiff       int64
		versionCreated bool
		versionPath    string
		oldAttrs       node.Attributes
		oldMtime       time.Time
	)

	defer func() {
		if committed {
			return
		}
		if versionCreated {
			if err := os.Remove(versionPath); err != nil && !errors.Is(err, os.ErrNotExist) {
				appctx.GetLogger(ctx).Error().Err(err).Str("versionpath", versionPath).Msg("could not remove version file during rollback")
			}
		}
		if info.NodeExisted && oldAttrs != nil {
			// mtime goes in the same batch: we still hold the metadata lock and
			// SetMTime would deadlock trying to retake it
			if err := fs.lu.TimeManager().OverrideMtime(ctx, n, &oldAttrs, oldMtime); err != nil {
				appctx.GetLogger(ctx).Error().Err(err).Str("nodeid", n.ID).Msg("could not restore node mtime during rollback")
			}
			if err := n.SetXattrsWithContext(ctx, oldAttrs, false); err != nil {
				appctx.GetLogger(ctx).Error().Err(err).Str("nodeid", n.ID).Msg("could not restore node xattrs during rollback")
			}
			// Restoring only rewrites the old keys, so the mark this upload added would
			// stay, and with the session discarded nothing else would ever remove it.
			if _, ok := oldAttrs[prefixes.StatusPrefix]; !ok {
				if err := n.RemoveXattr(ctx, prefixes.StatusPrefix, false); err != nil {
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
		old, err = node.ReadNode(ctx, fs.lu, n.SpaceID, n.ID, false, nil, false)
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
		oldAttrs, err = fs.lu.MetadataBackend().All(ctx, targetPath)
		if err != nil {
			return nil, err
		}

		if !fs.o.DisableVersioning {
			versionPath = fs.lu.InternalPath(n.SpaceID, n.ID+node.RevisionIDDelimiter+oldMtime.UTC().Format(time.RFC3339Nano))
			revFile, err := os.OpenFile(versionPath, os.O_CREATE|os.O_EXCL, 0600)
			if err != nil {
				if !errors.Is(err, os.ErrExist) {
					return nil, err
				}
				// revision with this mtime already exists; verify blobs match then reclaim the slot
				if err := validateChecksums(ctx, fs.lu, n, versionPath); err != nil {
					return nil, err
				}
				blobID, _, err := fs.lu.ReadBlobIDAndSizeAttr(ctx, versionPath, nil)
				if err != nil {
					return nil, err
				}
				if err := fs.tp.DeleteBlob(&node.Node{BlobID: blobID, SpaceID: n.SpaceID}); err != nil {
					return nil, err
				}
				f2, err := os.Create(versionPath)
				if err != nil {
					return nil, err
				}
				f2.Close()
			} else {
				revFile.Close()
			}

			if err := fs.lu.CopyMetadataWithSourceLock(ctx, targetPath, versionPath, func(attributeName string, value []byte) ([]byte, bool) {
				return value, strings.HasPrefix(attributeName, prefixes.ChecksumPrefix) ||
					attributeName == prefixes.TypeAttr ||
					attributeName == prefixes.BlobIDAttr ||
					attributeName == prefixes.BlobsizeAttr ||
					attributeName == prefixes.MTimeAttr
			}, f, true); err != nil {
				return nil, err
			}
			if err := os.Chtimes(versionPath, oldMtime, oldMtime); err != nil {
				return nil, errtypes.InternalError(fmt.Sprintf("failed to set mtime on version node: %s", err))
			}
			versionCreated = true
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

	if err := n.SetXattrsWithContext(ctx, attrs, false); err != nil {
		return nil, errors.Wrap(err, "could not write metadata")
	}

	if err := fs.tp.Propagate(ctx, n, sizeDiff); err != nil {
		return nil, errors.Wrap(err, "could not propagate size change")
	}
	committed = true

	result := &storage.PrepareUploadResult{VersionCreated: versionCreated, SizeDiff: sizeDiff}
	if !info.NodeExisted {
		// Read off the disk, not through ListGrants: an uploader who cannot see the
		// space root still needs the manager of a project space reported.
		result.SpaceOwner = n.SpaceOwnerOrManager(ctx)
	}
	return result, nil
}

// checkNewFileParent re-checks, for a file PrepareUpload is about to create, what
// TouchFile checked: the parent still exists, and the user may still upload into it.
// Either may have changed while the bytes were in flight.
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

// RollbackUpload reverts the node state written by PrepareUpload after a failed or aborted
// postprocessing run. It restores the previous revision (or purges the node if versioning is
// disabled and no prior version exists) and reverts the optimistic size propagation.
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
	n.SpaceRoot, err = node.ReadNode(ctx, fs.lu, n.SpaceID, n.SpaceID, false, nil, false)
	if err != nil {
		return err
	}

	curProcessingID, err := n.ProcessingID(ctx)
	if err != nil {
		return fmt.Errorf("RollbackUpload: could not read processing ID: %w", err)
	}
	if curProcessingID != sessionID {
		return nil
	}

	if info.NodeExisted {
		if err := n.RevertCurrentRevision(ctx, false); err != nil {
			return err
		}
	} else {
		// the upload created this node, so undo it. Purge, not trash: the file never
		// became visible, and Purge needs no Delete permission
		if err := n.Purge(ctx); err != nil {
			return fmt.Errorf("RollbackUpload: could not purge node: %w", err)
		}
	}

	if info.SizeDiff != 0 {
		if err := fs.tp.Propagate(ctx, n, -info.SizeDiff); err != nil {
			appctx.GetLogger(ctx).Error().Err(err).Msg("RollbackUpload: could not revert propagate")
		}
	}
	return nil
}

// node metadata corrupt, use session ids to release quota
func (fs *Decomposedfs) rollbackOrphaned(ctx context.Context, ref *provider.Reference, sessionID string, info storage.RollbackInfo, lookupErr error) error {
	if info.NodeID == "" || info.ParentID == "" {
		return fmt.Errorf("RollbackUpload: node lookup failed: %w", lookupErr)
	}
	spaceID := ref.GetResourceId().GetSpaceId()
	if spaceID == "" {
		return fmt.Errorf("RollbackUpload: node lookup failed: %w", lookupErr)
	}
	appctx.GetLogger(ctx).Info().Err(lookupErr).Str("sessionid", sessionID).Str("nodeid", info.NodeID).
		Msg("node unreadable, rolling back orphaned upload")
	n := node.New(spaceID, info.NodeID, info.ParentID, info.Filename, info.Size, sessionID,
		provider.ResourceType_RESOURCE_TYPE_FILE, nil, fs.lu)
	spaceRoot, err := node.ReadNode(ctx, fs.lu, spaceID, spaceID, false, nil, false)
	if err != nil {
		return fmt.Errorf("RollbackUpload: space root lookup failed: %w", err)
	}
	n.SpaceRoot = spaceRoot
	if info.SizeDiff != 0 {
		// return error so caller keeps session for retry
		if err := fs.tp.Propagate(ctx, n, -info.SizeDiff); err != nil {
			return fmt.Errorf("RollbackUpload: could not revert propagate: %w", err)
		}
	}
	nodePath := n.InternalPath()
	if err := utils.RemoveItem(nodePath); err != nil && !errors.Is(err, iofs.ErrNotExist) {
		appctx.GetLogger(ctx).Error().Err(err).Str("nodepath", nodePath).Msg("RollbackUpload: removing orphaned node failed")
	}
	if err := fs.lu.MetadataBackend().Purge(ctx, nodePath); err != nil && !errors.Is(err, iofs.ErrNotExist) {
		appctx.GetLogger(ctx).Error().Err(err).Str("nodepath", nodePath).Msg("RollbackUpload: purging orphaned node metadata failed")
	}
	// parent holds a child entry pointing to the now-removed node
	childEntry := filepath.Join(n.ParentPath(), n.Name)
	if err := os.Remove(childEntry); err != nil && !errors.Is(err, iofs.ErrNotExist) {
		appctx.GetLogger(ctx).Error().Err(err).Str("path", childEntry).Msg("RollbackUpload: removing orphaned child entry failed")
	}
	return nil
}

func validateChecksums(ctx context.Context, lu node.PathLookup, n *node.Node, versionPath string) error {
	for _, t := range []string{"md5", "sha1", "adler32"} {
		key := prefixes.ChecksumPrefix + t
		checksum, err := n.Xattr(ctx, key)
		if err != nil {
			return err
		}
		revisionChecksum, err := lu.MetadataBackend().Get(ctx, versionPath, key)
		if err != nil {
			return err
		}
		if string(checksum) == "" || string(revisionChecksum) == "" {
			return errors.New("checksum not found")
		}
		if string(checksum) != string(revisionChecksum) {
			return errors.New("checksum mismatch")
		}
	}
	return nil
}

// IsOrphaned reports whether the referenced resource exists but its metadata is unreadable.
func (fs *Decomposedfs) IsOrphaned(ctx context.Context, ref *provider.Reference) bool {
	_, err := fs.lu.NodeFromResource(ctx, ref)
	return err != nil
}
