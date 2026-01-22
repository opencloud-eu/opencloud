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
	"time"

	provider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	"github.com/rogpeppe/go-internal/lockedfile"

	"github.com/opencloud-eu/reva/v2/pkg/appctx"
	"github.com/opencloud-eu/reva/v2/pkg/errtypes"
	"github.com/opencloud-eu/reva/v2/pkg/storage/pkg/decomposedfs/metadata/prefixes"
	"github.com/opencloud-eu/reva/v2/pkg/storage/pkg/decomposedfs/node"
	"github.com/opencloud-eu/reva/v2/pkg/storagespace"
)

// Revision entries are stored inside the node folder and start with the same uuid as the current version.
// The `.REV.` indicates it is a revision and what follows is a timestamp, so multiple versions
// can be kept in the same location as the current file content. This prevents new fileuploads
// to trigger cross storage moves when revisions accidentally are stored on another partition,
// because the admin mounted a different partition there.
// We can add a background process to move old revisions to a slower storage
// and replace the revision file with a symbolic link in the future, if necessary.

func (fs *Decomposedfs) ListRevisions(ctx context.Context, ref *provider.Reference) (revisions []*provider.FileVersion, err error) {
	return fs.tp.ListRevisions(ctx, ref)
}

// DownloadRevision returns a reader for the specified revision
// FIXME the CS3 api should explicitly allow initiating revision and trash download, a related issue is https://github.com/cs3org/reva/issues/1813
func (fs *Decomposedfs) DownloadRevision(ctx context.Context, ref *provider.Reference, revisionKey string, openReaderFunc func(md *provider.ResourceInfo) bool) (*provider.ResourceInfo, io.ReadCloser, error) {
	return fs.tp.DownloadRevision(ctx, ref, revisionKey, openReaderFunc)
}

// RestoreRevision restores the specified revision of the resource
func (fs *Decomposedfs) RestoreRevision(ctx context.Context, ref *provider.Reference, revisionKey string) (returnErr error) {
	_, span := tracer.Start(ctx, "RestoreRevision")
	defer span.End()
	log := appctx.GetLogger(ctx)

	// verify revision key format
	kp := strings.SplitN(revisionKey, node.RevisionIDDelimiter, 2)
	if len(kp) != 2 {
		log.Error().Str("revisionKey", revisionKey).Msg("malformed revisionKey")
		return errtypes.NotFound(revisionKey)
	}

	spaceID := ref.ResourceId.SpaceId
	// check if the node is available and has not been deleted
	n, err := node.ReadNode(ctx, fs.lu, spaceID, kp[0], "", false, nil, false)
	if err != nil {
		return err
	}
	if !n.Exists {
		err = errtypes.NotFound(filepath.Join(n.ParentID, n.Name))
		return err
	}

	rp, err := fs.p.AssemblePermissions(ctx, n)
	switch {
	case err != nil:
		return err
	case !rp.RestoreFileVersion:
		f, _ := storagespace.FormatReference(ref)
		if rp.Stat {
			return errtypes.PermissionDenied(f)
		}
		return errtypes.NotFound(f)
	}

	// Set space owner in context
	storagespace.ContextSendSpaceOwnerID(ctx, n.SpaceOwnerOrManager(ctx))

	// check lock
	if err := n.CheckLock(ctx); err != nil {
		return err
	}

	// write lock node before copying metadata
	f, err := lockedfile.OpenFile(fs.lu.MetadataBackend().LockfilePath(n), os.O_RDWR|os.O_CREATE, 0600)
	if err != nil {
		return err
	}
	defer func() {
		_ = f.Close()
		_ = os.Remove(fs.lu.MetadataBackend().LockfilePath(n))
	}()

	// move current version to new revision
	mtime, err := n.GetMTime(ctx)
	if err != nil {
		log.Error().Err(err).Interface("ref", ref).Str("originalnode", kp[0]).Str("revisionKey", revisionKey).Msg("cannot read mtime")
		return err
	}

	// create a revision of the current node
	if _, err := fs.tp.CreateRevision(ctx, n, mtime.UTC().Format(time.RFC3339Nano), f); err != nil {
		return err
	}

	// revision 5, current 10 (restore a smaller blob) -> 5-10 = -5
	// revision 10, current 5 (restore a bigger blob) -> 10-5 = +5
	revisionNode := node.NewBaseNode(spaceID, revisionKey, fs.lu)
	revisionSize, err := fs.lu.MetadataBackend().GetInt64(ctx, revisionNode, prefixes.BlobsizeAttr)
	if err != nil {
		return errtypes.InternalError("failed to read blob size xattr from old revision")
	}
	sizeDiff := revisionSize - n.Blobsize

	// restore revision
	restoredRevisionPath := fs.lu.InternalPath(spaceID, revisionKey)
	if err := fs.tp.RestoreRevision(ctx, revisionNode, n, time.Now()); err != nil {
		return err
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

	return fs.tp.Propagate(ctx, n, sizeDiff)
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

	if err := os.RemoveAll(fs.lu.InternalPath(n.SpaceID, revisionKey)); err != nil {
		return err
	}

	return fs.tp.DeleteBlob(n)
}

func (fs *Decomposedfs) getRevisionNode(ctx context.Context, ref *provider.Reference, revisionKey string, hasPermission func(*provider.ResourcePermissions) bool) (*node.Node, error) {
	_, span := tracer.Start(ctx, "getRevisionNode")
	defer span.End()
	log := appctx.GetLogger(ctx)

	// verify revision key format
	kp := strings.SplitN(revisionKey, node.RevisionIDDelimiter, 2)
	if len(kp) != 2 {
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
	storagespace.ContextSendSpaceOwnerID(ctx, n.SpaceOwnerOrManager(ctx))

	return n, nil
}
