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
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	provider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"

	"github.com/opencloud-eu/reva/v3/pkg/appctx"
	"github.com/opencloud-eu/reva/v3/pkg/errtypes"
	"github.com/opencloud-eu/reva/v3/pkg/storage"
	"github.com/opencloud-eu/reva/v3/pkg/storage/pkg/decomposedfs/metadata/prefixes"
	"github.com/opencloud-eu/reva/v3/pkg/storage/pkg/decomposedfs/node"
	"github.com/opencloud-eu/reva/v3/pkg/storagespace"
)

// Revision entries are stored inside the node folder and start with the same uuid as the current version.
// The `.REV.` indicates it is a revision and what follows is a timestamp, so multiple versions
// can be kept in the same location as the current file content. This prevents new fileuploads
// to trigger cross storage moves when revisions accidentally are stored on another partition,
// because the admin mounted a different partition there.
// We can add a background process to move old revisions to a slower storage
// and replace the revision file with a symbolic link in the future, if necessary.

func (fs *Decomposedfs) ListRevisions(ctx context.Context, ref *provider.Reference) (revisions []*provider.FileVersion, err error) {
	_, span := tracer.Start(ctx, "ListRevisions")
	defer span.End()
	n, err := fs.lu.NodeFromResource(ctx, ref)
	if err != nil {
		return nil, err
	}
	if !n.Exists {
		return nil, errtypes.NotFound(filepath.Join(n.ParentID, n.Name))
	}

	rp, err := fs.p.AssemblePermissions(ctx, n)
	switch {
	case err != nil:
		return nil, err
	case !rp.ListFileVersions:
		f, _ := storagespace.FormatReference(ref)
		if rp.Stat {
			return nil, errtypes.PermissionDenied(f)
		}
		return nil, errtypes.NotFound(f)
	}

	return fs.tp.ListRevisions(ctx, n)
}

// DownloadRevision returns a reader for the specified revision
// FIXME the CS3 api should explicitly allow initiating revision and trash download, a related issue is https://github.com/cs3org/reva/issues/1813
func (fs *Decomposedfs) DownloadRevision(ctx context.Context, ref *provider.Reference, revisionKey string, openReaderFunc func(md *provider.ResourceInfo) bool) (*provider.ResourceInfo, io.ReadCloser, error) {
	_, span := tracer.Start(ctx, "DownloadRevision")
	defer span.End()
	log := appctx.GetLogger(ctx)

	// verify revision key format
	kp := strings.SplitN(revisionKey, node.RevisionIDDelimiter, 2)
	if len(kp) != 2 || !node.IsValidID(revisionKey) {
		log.Error().Str("revisionKey", revisionKey).Msg("malformed revisionKey")
		return nil, nil, errtypes.NotFound(revisionKey)
	}
	log.Debug().Str("revisionKey", revisionKey).Msg("DownloadRevision")

	// check if the node is available and has not been deleted
	n, err := node.ReadNode(ctx, fs.lu, ref.ResourceId.SpaceId, kp[0], "", false, nil, false)
	if err != nil {
		return nil, nil, err
	}
	if !n.Exists {
		return nil, nil, errtypes.NotFound(filepath.Join(n.ParentID, n.Name))
	}

	rp, err := fs.p.AssemblePermissions(ctx, n)
	switch {
	case err != nil:
		return nil, nil, err
	case !rp.ListFileVersions || !rp.InitiateFileDownload: // TODO add explicit permission in the CS3 api?
		f, _ := storagespace.FormatReference(ref)
		if rp.Stat {
			return nil, nil, errtypes.PermissionDenied(f)
		}
		return nil, nil, errtypes.NotFound(f)
	}

	return fs.tp.DownloadRevision(ctx, n, revisionKey, rp, openReaderFunc)
}

// RestoreRevision restores the specified revision of the resource
func (fs *Decomposedfs) RestoreRevision(ctx context.Context, ref *provider.Reference, revisionKey string) (_ *storage.RestoreRevisionResult, returnErr error) {
	_, span := tracer.Start(ctx, "RestoreRevision")
	defer span.End()
	log := appctx.GetLogger(ctx)

	// verify revision key format
	kp := strings.SplitN(revisionKey, node.RevisionIDDelimiter, 2)
	if len(kp) != 2 || !node.IsValidID(revisionKey) {
		log.Error().Str("revisionKey", revisionKey).Msg("malformed revisionKey")
		return nil, errtypes.NotFound(revisionKey)
	}

	spaceID := ref.ResourceId.SpaceId
	// check if the node is available and has not been deleted
	n, err := node.ReadNode(ctx, fs.lu, spaceID, kp[0], "", false, nil, false)
	if err != nil {
		return nil, err
	}
	if !n.Exists {
		err = errtypes.NotFound(filepath.Join(n.ParentID, n.Name))
		return nil, err
	}
	setAuditItem(ctx, n)
	fs.setAuditPath(ctx, n, ref)

	rp, err := fs.p.AssemblePermissions(ctx, n)
	switch {
	case err != nil:
		return nil, err
	case !rp.RestoreFileVersion:
		f, _ := storagespace.FormatReference(ref)
		if rp.Stat {
			return nil, errtypes.PermissionDenied(f)
		}
		return nil, errtypes.NotFound(f)
	}

	spaceOwner := n.SpaceOwnerOrManager(ctx)

	// check lock
	if err := n.CheckLock(ctx); err != nil {
		return nil, err
	}
	if n.IsProcessing(ctx) {
		return nil, errtypes.ResourceProcessing(ref.String())
	}

	// write lock node before copying metadata
	unlock, err := fs.lu.MetadataBackend().Lock(n)
	if err != nil {
		return nil, err
	}
	// released before the revision cleanup at the end, or on any return before that
	unlockNode := sync.OnceValue(unlock)
	defer func() { _ = unlockNode() }()

	// The revision must still exist on disk. Its metadata may come from a stale cache
	// entry: another process, e.g. `opencloud storage-users revisions cleanup` with the
	// default in-memory metadata cache, may have deleted it. Restoring it anyway would
	// point the node to the revision's deleted blob, and it would leave a revision of the
	// current content behind. Checked under the node's lock, which deleting a revision
	// takes too (Node.DeleteRevisionUnchecked).
	revisionNode := node.NewBaseNode(spaceID, revisionKey, fs.lu)
	if _, err := os.Stat(revisionNode.InternalPath()); err != nil {
		if !os.IsNotExist(err) {
			return nil, err
		}
		// drop the stale cache entry, if there is one
		_ = fs.lu.MetadataBackend().Purge(ctx, revisionNode)
		return nil, errtypes.NotFound(revisionKey)
	}

	// move current version to new revision
	mtime, err := n.GetMTime(ctx)
	if err != nil {
		log.Error().Err(err).Interface("ref", ref).Str("originalnode", kp[0]).Str("revisionKey", revisionKey).Msg("cannot read mtime")
		return nil, err
	}

	// the blob id the node gets back, for the mark of the revision created below
	restoredBlobID, err := fs.lu.MetadataBackend().Get(ctx, revisionNode, prefixes.BlobIDAttr)
	if err != nil {
		log.Warn().Err(err).Interface("ref", ref).Str("revisionKey", revisionKey).Msg("could not read the blob id of the revision, the revision of the replaced content won't be protected from the revision cleanup")
	}

	// create a revision of the current node
	// It is named after the mtime of the content it preserves, not after the time it is
	// created; the revision cleanup goes by the time it is created, which CreateRevision
	// records (see node.RevisionCreatedAttr).
	newRevision := mtime.UTC().Format(time.RFC3339Nano)
	if _, err := fs.tp.CreateRevision(ctx, n, newRevision); err != nil {
		return nil, err
	}

	// revision 5, current 10 (restore a smaller blob) -> 5-10 = -5
	// revision 10, current 5 (restore a bigger blob) -> 10-5 = +5
	revisionSize, err := fs.lu.MetadataBackend().GetInt64(ctx, revisionNode, prefixes.BlobsizeAttr)
	if err != nil {
		return nil, errtypes.InternalError("failed to read blob size xattr from old revision")
	}
	sizeDiff := revisionSize - n.Blobsize

	// restore revision
	restoredRevisionPath := fs.lu.InternalPath(spaceID, revisionKey)
	if err := fs.tp.RestoreRevision(ctx, revisionNode, n, time.Now()); err != nil {
		return nil, err
	}

	// A restore is a deliberate user action: the revision cleanup must not drop the content the
	// user just replaced, e.g. because its revision is older than the strategy allows, or as the
	// 'auto' strategy thins out revisions that are close in time. Marked on disk, while the
	// node is still locked, so that every cleanup sees it, see node.KeepAfterRestoreAttr. Not
	// marking it doesn't fail the restore, which is done by now: the revision is only protected
	// less.
	if len(restoredBlobID) > 0 {
		newRevisionNode := node.NewBaseNode(spaceID, n.ID+node.RevisionIDDelimiter+newRevision, fs.lu)
		if err := fs.lu.MetadataBackend().Set(ctx, newRevisionNode, node.KeepAfterRestoreAttr, restoredBlobID); err != nil {
			log.Error().Err(err).Interface("ref", ref).Str("revision", newRevisionNode.ID).Msg("could not mark the revision of the replaced content, the revision cleanup may delete it")
		}
	}

	// drop old revision
	if err := os.Remove(restoredRevisionPath); err != nil {
		log.Warn().Err(err).Interface("ref", ref).Str("originalnode", kp[0]).Str("revisionKey", revisionKey).Msg("could not delete old revision, continuing")
	}
	if err := os.Remove(fs.lu.MetadataBackend().MetadataPath(revisionNode)); err != nil {
		log.Warn().Err(err).Interface("ref", ref).Str("originalnode", kp[0]).Str("revisionKey", revisionKey).Msg("could not delete old revision metadata, continuing")
	}
	if err := os.Remove(fs.lu.MetadataBackend().LockfilePath(revisionNode)); err != nil {
		log.Warn().Err(err).Interface("ref", ref).Str("originalnode", kp[0]).Str("revisionKey", revisionKey).Msg("could not delete old revision metadata lockfile, continuing")
	}
	if err := fs.lu.MetadataBackend().Purge(ctx, revisionNode); err != nil {
		log.Warn().Err(err).Interface("ref", ref).Str("originalnode", kp[0]).Str("revisionKey", revisionKey).Msg("could not purge old revision from cache, continuing")
	}

	if err := fs.tp.Propagate(ctx, n, sizeDiff); err != nil {
		return nil, err
	}

	// The restore created a new revision from the previous content, so apply the
	// revision cleanup strategy (see package revisioncleanup). It runs last, once the
	// restore is complete, and never fails the restore. It keeps the revision marked above.
	// The node lock is released first: the cleanup takes it itself, and when it runs in the
	// background, waiting for room in its queue while holding the lock could deadlock with
	// a worker waiting for the lock.
	_ = unlockNode()
	fs.revisionsCleanup.HandleRevisionCreated(ctx, n)
	return &storage.RestoreRevisionResult{SpaceOwner: spaceOwner, SpaceID: n.SpaceID}, nil
}

// DeleteRevision deletes the specified revision of the resource
func (fs *Decomposedfs) DeleteRevision(ctx context.Context, ref *provider.Reference, revisionKey string) error {
	_, span := tracer.Start(ctx, "DeleteRevision")
	defer span.End()
	n, err := fs.getRevisionNode(ctx, ref, revisionKey, func(rp *provider.ResourcePermissions) bool {
		return rp.RestoreFileVersion
	})
	if err != nil {
		return err
	}

	return fs.deleteRevisionFile(ctx, n, revisionKey)
}

// deleteRevisionFile deletes the revision node identified by revisionKey
// (nodeID + RevisionIDDelimiter + timestamp) together with its metadata
// sidecars and its own blob. It is a no-op if the revision does not exist.
func (fs *Decomposedfs) deleteRevisionFile(ctx context.Context, n *node.Node, revisionKey string) error {
	kp := strings.SplitN(revisionKey, node.RevisionIDDelimiter, 2)
	if len(kp) != 2 || !node.IsValidID(revisionKey) {
		return errtypes.NotFound(revisionKey)
	}

	blobID, err := n.DeleteRevisionUnchecked(ctx, kp[1])
	if err != nil {
		return err
	}
	if blobID == "" {
		// no blob to delete (0-byte file or current-revision revert)
		return nil
	}

	return fs.tp.DeleteBlob(&node.Node{BaseNode: node.BaseNode{SpaceID: n.SpaceID}, BlobID: blobID})
}

func (fs *Decomposedfs) getRevisionNode(ctx context.Context, ref *provider.Reference, revisionKey string, hasPermission func(*provider.ResourcePermissions) bool) (*node.Node, error) {
	_, span := tracer.Start(ctx, "getRevisionNode")
	defer span.End()
	log := appctx.GetLogger(ctx)

	// verify revision key format
	kp := strings.SplitN(revisionKey, node.RevisionIDDelimiter, 2)
	if len(kp) != 2 || !node.IsValidID(revisionKey) {
		log.Error().Str("revisionKey", revisionKey).Msg("malformed revisionKey")
		return nil, errtypes.NotFound(revisionKey)
	}
	log.Debug().Str("revisionKey", revisionKey).Msg("DownloadRevision")

	spaceID := ref.ResourceId.SpaceId
	// check if the node is available and has not been deleted
	n, err := node.ReadNode(ctx, fs.lu, spaceID, kp[0], "", false, nil, false)
	if err != nil {
		return nil, err
	}
	if !n.Exists {
		err = errtypes.NotFound(filepath.Join(n.ParentID, n.Name))
		return nil, err
	}

	p, err := fs.p.AssemblePermissions(ctx, n)
	switch {
	case err != nil:
		return nil, err
	case !hasPermission(p):
		return nil, errtypes.PermissionDenied(filepath.Join(n.ParentID, n.Name))
	}

	// Set space owner in context
	storagespace.ContextSetSpaceOwner(ctx, n.SpaceOwnerOrManager(ctx))

	return n, nil
}
