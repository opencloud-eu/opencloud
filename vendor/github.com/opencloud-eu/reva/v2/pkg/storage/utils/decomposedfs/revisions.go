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
	"github.com/pkg/errors"
	"github.com/rogpeppe/go-internal/lockedfile"

	"github.com/opencloud-eu/reva/v2/pkg/appctx"
	"github.com/opencloud-eu/reva/v2/pkg/errtypes"
	"github.com/opencloud-eu/reva/v2/pkg/storage/utils/decomposedfs/metadata/prefixes"
	"github.com/opencloud-eu/reva/v2/pkg/storage/utils/decomposedfs/node"
	"github.com/opencloud-eu/reva/v2/pkg/storagespace"
	"github.com/opencloud-eu/reva/v2/pkg/utils"
)

// Revision entries are stored inside the node folder and start with the same uuid as the current version.
// The `.REV.` indicates it is a revision and what follows is a timestamp, so multiple versions
// can be kept in the same location as the current file content. This prevents new fileuploads
// to trigger cross storage moves when revisions accidentally are stored on another partition,
// because the admin mounted a different partition there.
// We can add a background process to move old revisions to a slower storage
// and replace the revision file with a symbolic link in the future, if necessary.

// ListRevisions lists the revisions of the given resource
func (fs *Decomposedfs) ListRevisions(ctx context.Context, ref *provider.Reference) (revisions []*provider.FileVersion, err error) {
	_, span := tracer.Start(ctx, "ListRevisions")
	defer span.End()
	var n *node.Node
	if n, err = fs.lu.NodeFromResource(ctx, ref); err != nil {
		return
	}
	if !n.Exists {
		err = errtypes.NotFound(filepath.Join(n.ParentID, n.Name))
		return
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

	revisions = []*provider.FileVersion{}
	np := n.InternalPath()
	if items, err := filepath.Glob(np + node.RevisionIDDelimiter + "*"); err == nil {
		for i := range items {
			if fs.lu.MetadataBackend().IsMetaFile(items[i]) || strings.HasSuffix(items[i], ".mlock") {
				continue
			}

			if fi, err := os.Stat(items[i]); err == nil {
				parts := strings.SplitN(fi.Name(), node.RevisionIDDelimiter, 2)
				if len(parts) != 2 {
					appctx.GetLogger(ctx).Error().Err(err).Str("name", fi.Name()).Msg("invalid revision name, skipping")
					continue
				}
				mtime := fi.ModTime()
				rev := &provider.FileVersion{
					Key:   n.ID + node.RevisionIDDelimiter + parts[1],
					Mtime: uint64(mtime.Unix()),
				}
				_, blobSize, err := fs.lu.ReadBlobIDAndSizeAttr(ctx, items[i], nil)
				if err != nil {
					appctx.GetLogger(ctx).Error().Err(err).Str("name", fi.Name()).Msg("error reading blobsize xattr, using 0")
				}
				rev.Size = uint64(blobSize)
				etag, err := node.CalculateEtag(n.ID, mtime)
				if err != nil {
					return nil, errors.Wrapf(err, "error calculating etag")
				}
				rev.Etag = etag
				revisions = append(revisions, rev)
			}
		}
	}
	// maybe we need to sort the list by key
	/*
		sort.Slice(revisions, func(i, j int) bool {
			return revisions[i].Key > revisions[j].Key
		})
	*/

	return
}

// DownloadRevision returns a reader for the specified revision
// FIXME the CS3 api should explicitly allow initiating revision and trash download, a related issue is https://github.com/cs3org/reva/issues/1813
func (fs *Decomposedfs) DownloadRevision(ctx context.Context, ref *provider.Reference, revisionKey string, openReaderFunc func(md *provider.ResourceInfo) bool) (*provider.ResourceInfo, io.ReadCloser, error) {
	_, span := tracer.Start(ctx, "DownloadRevision")
	defer span.End()
	log := appctx.GetLogger(ctx)

	// verify revision key format
	kp := strings.SplitN(revisionKey, node.RevisionIDDelimiter, 2)
	if len(kp) != 2 {
		log.Error().Str("revisionKey", revisionKey).Msg("malformed revisionKey")
		return nil, nil, errtypes.NotFound(revisionKey)
	}
	log.Debug().Str("revisionKey", revisionKey).Msg("DownloadRevision")

	spaceID := ref.ResourceId.SpaceId
	// check if the node is available and has not been deleted
	n, err := node.ReadNode(ctx, fs.lu, spaceID, kp[0], false, nil, false)
	if err != nil {
		return nil, nil, err
	}
	if !n.Exists {
		err = errtypes.NotFound(filepath.Join(n.ParentID, n.Name))
		return nil, nil, err
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

	contentPath := fs.lu.InternalPath(spaceID, revisionKey)

	blobid, blobsize, err := fs.lu.ReadBlobIDAndSizeAttr(ctx, contentPath, nil)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "Decomposedfs: could not read blob id and size for revision '%s' of node '%s'", kp[1], n.ID)
	}

	revisionNode := node.Node{SpaceID: spaceID, BlobID: blobid, Blobsize: blobsize} // blobsize is needed for the s3ng blobstore

	ri, err := n.AsResourceInfo(ctx, rp, nil, []string{"size", "mimetype", "etag"}, true)
	if err != nil {
		return nil, nil, err
	}

	// update resource info with revision data
	mtime, err := time.Parse(time.RFC3339Nano, kp[1])
	if err != nil {
		return nil, nil, errors.Wrapf(err, "Decomposedfs: could not parse mtime for revision '%s' of node '%s'", kp[1], n.ID)
	}
	ri.Size = uint64(blobsize)
	ri.Mtime = utils.TimeToTS(mtime)
	ri.Etag, err = node.CalculateEtag(n.ID, mtime)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "error calculating etag for revision '%s' of node '%s'", kp[1], n.ID)
	}

	var reader io.ReadCloser
	if openReaderFunc(ri) {
		reader, err = fs.tp.ReadBlob(&revisionNode)
		if err != nil {
			return nil, nil, errors.Wrapf(err, "Decomposedfs: could not download blob of revision '%s' for node '%s'", n.ID, revisionKey)
		}
	}
	return ri, reader, nil
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
	n, err := node.ReadNode(ctx, fs.lu, spaceID, kp[0], false, nil, false)
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
	f, err := lockedfile.OpenFile(fs.lu.MetadataBackend().LockfilePath(n.InternalPath()), os.O_RDWR|os.O_CREATE, 0600)
	if err != nil {
		return err
	}
	defer func() {
		_ = f.Close()
		_ = os.Remove(fs.lu.MetadataBackend().LockfilePath(n.InternalPath()))
	}()

	// move current version to new revision
	nodePath := fs.lu.InternalPath(spaceID, kp[0])
	mtime, err := n.GetMTime(ctx)
	if err != nil {
		log.Error().Err(err).Interface("ref", ref).Str("originalnode", kp[0]).Str("revisionKey", revisionKey).Msg("cannot read mtime")
		return err
	}

	// revisions are stored alongside the actual file, so a rename can be efficient and does not cross storage / partition boundaries
	newRevisionPath := fs.lu.InternalPath(spaceID, kp[0]+node.RevisionIDDelimiter+mtime.UTC().Format(time.RFC3339Nano))

	// touch new revision
	if _, err := os.Create(newRevisionPath); err != nil {
		return err
	}
	defer func() {
		if returnErr != nil {
			if err := os.Remove(newRevisionPath); err != nil {
				log.Error().Err(err).Str("revision", filepath.Base(newRevisionPath)).Msg("could not clean up revision node")
			}
			if err := fs.lu.MetadataBackend().Purge(ctx, newRevisionPath); err != nil {
				log.Error().Err(err).Str("revision", filepath.Base(newRevisionPath)).Msg("could not clean up revision node")
			}
		}
	}()

	// copy blob metadata from node to new revision node
	err = fs.lu.CopyMetadataWithSourceLock(ctx, nodePath, newRevisionPath, func(attributeName string, value []byte) (newValue []byte, copy bool) {
		return value, strings.HasPrefix(attributeName, prefixes.ChecksumPrefix) || // for checksums
			attributeName == prefixes.TypeAttr ||
			attributeName == prefixes.BlobIDAttr ||
			attributeName == prefixes.BlobsizeAttr ||
			attributeName == prefixes.MTimeAttr // FIXME somewhere I mix up the revision time and the mtime, causing the restore to overwrite the other existing revisien
	}, f, true)
	if err != nil {
		return errtypes.InternalError("failed to copy blob xattrs to version node: " + err.Error())
	}

	// remember mtime from node as new revision mtime
	if err = os.Chtimes(newRevisionPath, mtime, mtime); err != nil {
		return errtypes.InternalError("failed to change mtime of version node")
	}

	// update blob id in node

	// copy blob metadata from restored revision to node
	restoredRevisionPath := fs.lu.InternalPath(spaceID, revisionKey)
	err = fs.lu.CopyMetadata(ctx, restoredRevisionPath, nodePath, func(attributeName string, value []byte) (newValue []byte, copy bool) {
		return value, strings.HasPrefix(attributeName, prefixes.ChecksumPrefix) ||
			attributeName == prefixes.TypeAttr ||
			attributeName == prefixes.BlobIDAttr ||
			attributeName == prefixes.BlobsizeAttr
	}, false)
	if err != nil {
		return errtypes.InternalError("failed to copy blob xattrs to old revision to node: " + err.Error())
	}
	// always set the node mtime to the current time
	err = fs.lu.MetadataBackend().SetMultiple(ctx, nodePath,
		map[string][]byte{
			prefixes.MTimeAttr: []byte(time.Now().UTC().Format(time.RFC3339Nano)),
		},
		false)
	if err != nil {
		return errtypes.InternalError("failed to set mtime attribute on node: " + err.Error())
	}

	revisionSize, err := fs.lu.MetadataBackend().GetInt64(ctx, restoredRevisionPath, prefixes.BlobsizeAttr)
	if err != nil {
		return errtypes.InternalError("failed to read blob size xattr from old revision")
	}

	// drop old revision
	if err := os.Remove(restoredRevisionPath); err != nil {
		log.Warn().Err(err).Interface("ref", ref).Str("originalnode", kp[0]).Str("revisionKey", revisionKey).Msg("could not delete old revision, continuing")
	}
	if err := os.Remove(fs.lu.MetadataBackend().MetadataPath(restoredRevisionPath)); err != nil {
		log.Warn().Err(err).Interface("ref", ref).Str("originalnode", kp[0]).Str("revisionKey", revisionKey).Msg("could not delete old revision metadata, continuing")
	}
	if err := os.Remove(fs.lu.MetadataBackend().LockfilePath(restoredRevisionPath)); err != nil {
		log.Warn().Err(err).Interface("ref", ref).Str("originalnode", kp[0]).Str("revisionKey", revisionKey).Msg("could not delete old revision metadata lockfile, continuing")
	}
	if err := fs.lu.MetadataBackend().Purge(ctx, restoredRevisionPath); err != nil {
		log.Warn().Err(err).Interface("ref", ref).Str("originalnode", kp[0]).Str("revisionKey", revisionKey).Msg("could not purge old revision from cache, continuing")
	}

	// revision 5, current 10 (restore a smaller blob) -> 5-10 = -5
	// revision 10, current 5 (restore a bigger blob) -> 10-5 = +5
	sizeDiff := revisionSize - n.Blobsize

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

	return fs.deleteRevisionFile(ctx, n, revisionKey)
}

// deleteRevisionFile deletes the revision node identified by revisionKey
// (nodeID + RevisionIDDelimiter + timestamp) together with its metadata
// sidecars and its own blob. It is a no-op if the revision does not exist.
func (fs *Decomposedfs) deleteRevisionFile(ctx context.Context, n *node.Node, revisionKey string) error {
	kp := strings.SplitN(revisionKey, node.RevisionIDDelimiter, 2)
	if len(kp) != 2 {
		return errtypes.NotFound(revisionKey)
	}

	blobID, err := n.DeleteRevision(ctx, kp[1])
	if err != nil {
		return err
	}
	if blobID == "" {
		// no blob to delete (0-byte file or current-revision revert)
		return nil
	}

	return fs.tp.DeleteBlob(&node.Node{SpaceID: n.SpaceID, BlobID: blobID})
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
	n, err := node.ReadNode(ctx, fs.lu, spaceID, kp[0], false, nil, false)
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
