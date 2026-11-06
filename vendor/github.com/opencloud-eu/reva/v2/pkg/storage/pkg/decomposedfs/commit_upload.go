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
// granted to it by virtue of its status as an Intergovernmental Organization.

package decomposedfs

import (
	"context"
	"path/filepath"
	"strings"
	"time"

	provider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	"github.com/pkg/errors"

	"github.com/opencloud-eu/reva/v2/pkg/appctx"
	"github.com/opencloud-eu/reva/v2/pkg/errtypes"
	"github.com/opencloud-eu/reva/v2/pkg/storage"
	"github.com/opencloud-eu/reva/v2/pkg/storage/pkg/decomposedfs/metadata"
	"github.com/opencloud-eu/reva/v2/pkg/storage/pkg/decomposedfs/metadata/prefixes"
	"github.com/opencloud-eu/reva/v2/pkg/storage/pkg/decomposedfs/node"
)

// MarkProcessing toggles the processing marker on a resource for a single upload session.
func (fs *Decomposedfs) MarkProcessing(ctx context.Context, ref *provider.Reference, processing bool, sessionID string) error {
	resolved, err := fs.lu.NodeFromResource(ctx, ref)
	if err != nil {
		return err
	}
	if !resolved.Exists {
		return errtypes.NotFound(ref.String())
	}

	n, unlock, err := node.LockAndReadNode(ctx, fs.lu, resolved.SpaceID, resolved.ID, "", false, resolved.SpaceRoot, false)
	if err != nil {
		return err
	}
	defer func() {
		if err := unlock(); err != nil {
			appctx.GetLogger(ctx).Error().Err(err).Str("nodeid", n.ID).Msg("could not close mark-processing lock")
		}
	}()

	if !processing {
		if !n.IsProcessing(ctx) {
			return nil
		}
		id, _ := n.ProcessingID(ctx)
		if id != sessionID {
			return nil
		}
		return n.RemoveXattr(ctx, prefixes.StatusPrefix)
	}

	// a newer upload takes over the mark, the last writer wins
	return n.SetXattrsWithContext(ctx, node.Attributes{
		prefixes.StatusPrefix: []byte(node.ProcessingStatus + sessionID),
	})
}

// CommitUpload writes the staged bytes from source to the resource at ref.
// sessionID identifies the upload that PrepareUpload wrote onto the node.
// Caller owns source.Body and must close it after CommitUpload returns.
//
// The bytes are written against the real node, so drivers that keep the content
// in the node itself (posix) can replace it atomically. If a newer upload took
// over the node in the meantime, the bytes go to the revision that upload created
// for this session instead. Without such a revision (versioning disabled) they are
// discarded: nothing references them anymore.
func (fs *Decomposedfs) CommitUpload(ctx context.Context, ref *provider.Reference, sessionID string, source storage.UploadSource) error {
	if source.Body == nil && source.Path == "" {
		return errtypes.BadRequest("Decomposedfs: source has neither body nor path")
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
	n.SpaceRoot, err = node.ReadNode(ctx, fs.lu, n.SpaceID, n.SpaceID, "", false, nil, false)
	if err != nil {
		return err
	}

	unlock, err := fs.lu.MetadataBackend().Lock(n)
	if err != nil {
		return errors.Wrap(err, "Decomposedfs: could not lock node")
	}
	defer func() {
		_ = unlock()
	}()

	// re-read the blob id while holding the lock, a newer upload may have taken over.
	// read the attribute itself: the posix lookup's ReadBlobIDAndSizeAttr reports no blob id
	blobID, err := fs.lu.MetadataBackend().Get(ctx, n, prefixes.BlobIDAttr)
	if err != nil {
		return errors.Wrap(err, "Decomposedfs: could not read blob id")
	}
	n.BlobID = string(blobID)

	target := n
	if n.BlobID != sessionID {
		revision, revisionUnlock, err := fs.uploadRevision(ctx, n, sessionID)
		if err != nil {
			return err
		}
		if revision == nil {
			appctx.GetLogger(ctx).Info().Str("nodeid", n.ID).Str("sessionid", sessionID).Msg("upload was superseded and has no revision, discarding its bytes")
			return nil
		}
		defer func() {
			_ = revisionUnlock()
		}()
		target = revision
	}

	if source.Path != "" {
		err = fs.tp.WriteBlob(target, source.Path)
	} else {
		err = fs.tp.WriteBlobFromReader(target, source.Body, source.Length)
	}
	if err != nil {
		if derr := fs.tp.DeleteBlob(target); derr != nil {
			appctx.GetLogger(ctx).Error().Err(derr).Str("nodeid", n.ID).Str("blobid", sessionID).Msg("could not clean up orphaned blob after failed write")
		}
		return errors.Wrap(err, "Decomposedfs: failed to write blob")
	}

	// on the node that holds the content, not the session: the session is deleted when the
	// upload finishes
	if !source.ScanDate.IsZero() {
		if err := target.SetScanData(ctx, source.ScanResult, source.ScanDate); err != nil {
			appctx.GetLogger(ctx).Error().Err(err).Str("nodeid", target.ID).Msg("could not set scan results")
		}
	}

	if target != n {
		// the revision kept this upload's processing mark for a rollback of the newer
		// upload; this upload is done, so restoring it must not mark the node again
		if err := target.RemoveXattr(ctx, prefixes.StatusPrefix); err != nil && !metadata.IsAttrUnset(err) {
			appctx.GetLogger(ctx).Error().Err(err).Str("nodeid", target.ID).Msg("could not remove the processing mark from the revision")
		}
	} else {
		// revisions kept only for a rollback of this upload served their purpose; versions
		// stay, also the ones created before versioning was disabled
		fs.removeRevisions(ctx, n)
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

// removeRevisions deletes the revisions of n that were kept only for a rollback, with their
// blobs, while the caller holds the node lock: the ones of uploads that were rolled back,
// and the snapshots taken with versioning disabled.
func (fs *Decomposedfs) removeRevisions(ctx context.Context, n *node.Node) {
	revPrefix := fs.lu.VersionPath(n.SpaceID, n.ID, "")
	revisions, err := filepath.Glob(fs.lu.VersionPath(n.SpaceID, n.ID, "*"))
	if err != nil {
		appctx.GetLogger(ctx).Error().Err(err).Str("nodeid", n.ID).Msg("could not list revisions")
		return
	}
	for _, rev := range revisions {
		if strings.HasSuffix(rev, ".mpk") || strings.HasSuffix(rev, ".mlock") {
			continue
		}
		revisionNode := node.NewBaseNode(n.SpaceID, n.ID+node.RevisionIDDelimiter+strings.TrimPrefix(rev, revPrefix), fs.lu)
		if !fs.keptForRollback(ctx, revisionNode) {
			continue
		}
		blobID, err := fs.lu.MetadataBackend().Get(ctx, revisionNode, prefixes.BlobIDAttr)
		if err == nil && len(blobID) > 0 && string(blobID) != n.BlobID {
			blobNode := node.New(n.SpaceID, revisionNode.ID, "", "", 0, string(blobID), provider.ResourceType_RESOURCE_TYPE_FILE, nil, fs.lu)
			if err := fs.tp.DeleteBlob(blobNode); err != nil {
				appctx.GetLogger(ctx).Error().Err(err).Str("revisionid", revisionNode.ID).Msg("could not delete the blob of a superseded revision")
			}
		}
		removeRevision(ctx, fs.lu, revisionNode)
	}
}

// keptForRollback reports whether a revision exists only for a rollback, rather than as a
// version.
func (fs *Decomposedfs) keptForRollback(ctx context.Context, revision metadata.MetadataNode) bool {
	for _, attr := range []string{purgeOnRestoreAttr, uploadSnapshotAttr} {
		if _, err := fs.lu.MetadataBackend().Get(ctx, revision, attr); err == nil {
			return true
		}
	}
	return false
}

// uploadRevision returns the locked revision of n that holds the upload sessionID,
// or nil if there is none.
func (fs *Decomposedfs) uploadRevision(ctx context.Context, n *node.Node, sessionID string) (*node.Node, metadata.UnlockFunc, error) {
	revPrefix := fs.lu.VersionPath(n.SpaceID, n.ID, "")
	revisions, err := filepath.Glob(fs.lu.VersionPath(n.SpaceID, n.ID, "*"))
	if err != nil {
		return nil, nil, err
	}
	for _, rev := range revisions {
		if strings.HasSuffix(rev, ".mpk") || strings.HasSuffix(rev, ".mlock") {
			continue
		}
		revisionID := n.ID + node.RevisionIDDelimiter + strings.TrimPrefix(rev, revPrefix)
		blobID, err := fs.lu.MetadataBackend().Get(ctx, node.NewBaseNode(n.SpaceID, revisionID, fs.lu), prefixes.BlobIDAttr)
		if err != nil || string(blobID) != sessionID {
			continue
		}
		revision, unlock, err := node.LockAndReadNode(ctx, fs.lu, n.SpaceID, revisionID, "", false, n.SpaceRoot, false)
		if err != nil {
			return nil, nil, err
		}
		// ReadNode caches the attributes of the node, not of the revision. The posix
		// blobstore copies them onto the file it writes, so read the revision's own.
		revision.ResetXattrsCache()
		return revision, unlock, nil
	}
	return nil, nil, nil
}
