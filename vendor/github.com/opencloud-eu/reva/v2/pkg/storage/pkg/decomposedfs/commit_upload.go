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
	"os"
	"time"

	provider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	"github.com/google/uuid"
	"github.com/pkg/errors"

	"github.com/opencloud-eu/reva/v2/pkg/appctx"
	"github.com/opencloud-eu/reva/v2/pkg/errtypes"
	"github.com/opencloud-eu/reva/v2/pkg/storage"
	"github.com/opencloud-eu/reva/v2/pkg/storage/pkg/decomposedfs/metadata/prefixes"
	"github.com/opencloud-eu/reva/v2/pkg/storage/pkg/decomposedfs/node"
	"github.com/opencloud-eu/reva/v2/pkg/utils"
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

	if n.IsProcessing(ctx) {
		return errtypes.ResourceProcessing(ref.String())
	}
	return n.SetXattrsWithContext(ctx, node.Attributes{
		prefixes.StatusPrefix: []byte(node.ProcessingStatus + sessionID),
	})
}

// CommitUpload writes staged bytes and their supplied metadata to an existing node.
func (fs *Decomposedfs) CommitUpload(ctx context.Context, ref *provider.Reference, source storage.UploadSource) (*provider.ResourceInfo, error) {
	if source.Body == nil {
		return nil, errtypes.BadRequest("Decomposedfs: source body is nil")
	}
	defer source.Body.Close()

	if len(source.Checksums.SHA1) == 0 || len(source.Checksums.MD5) == 0 || len(source.Checksums.Adler32) == 0 {
		return nil, errtypes.BadRequest("Decomposedfs: pre-computed checksums missing from source")
	}

	resolved, err := fs.lu.NodeFromResource(ctx, ref)
	if err != nil {
		return nil, err
	}
	if !resolved.Exists {
		return nil, errtypes.NotFound(ref.String())
	}

	n, unlock, err := node.LockAndReadNode(ctx, fs.lu, resolved.SpaceID, resolved.ID, "", false, resolved.SpaceRoot, false)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := unlock(); err != nil {
			appctx.GetLogger(ctx).Error().Err(err).Str("nodeid", n.ID).Msg("could not close commit-upload lock")
		}
	}()

	if err := n.CheckLock(ctx); err != nil {
		return nil, err
	}

	mtime := time.Now()
	if mts := source.Metadata["mtime"]; mts != "" {
		parsed, err := utils.MTimeToTime(mts)
		if err != nil {
			return nil, errtypes.BadRequest("invalid mtime: " + mts)
		}
		mtime = parsed
	}

	if fs.um != nil {
		if gid, ok := ctx.Value(CtxKeySpaceGID).(uint32); ok {
			unscope, err := fs.um.ScopeUserByIds(-1, int(gid))
			if err != nil {
				return nil, errors.Wrap(err, "Decomposedfs: failed to scope user")
			}
			if unscope != nil {
				defer func() { _ = unscope() }()
			}
		}
	}

	if _, err := node.CheckQuota(ctx, n.SpaceRoot, n.BlobID != "", uint64(n.Blobsize), uint64(source.Length)); err != nil {
		return nil, err
	}

	oldMtime, err := n.GetMTime(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "Decomposedfs: failed to read old mtime")
	}
	if !fs.o.DisableVersioning && n.BlobID != "" {
		versionPath, err := fs.tp.CreateRevision(ctx, n, oldMtime.UTC().Format(time.RFC3339Nano))
		if err != nil {
			return nil, errors.Wrap(err, "Decomposedfs: failed to create revision")
		}
		if err := os.Chtimes(versionPath, oldMtime, oldMtime); err != nil {
			return nil, errors.Wrap(err, "Decomposedfs: failed to set revision mtime")
		}
	}

	oldSize := n.Blobsize
	n.BlobID = uuid.New().String()
	n.Blobsize = source.Length

	attrs := node.Attributes{
		prefixes.ChecksumPrefix + "sha1":    source.Checksums.SHA1,
		prefixes.ChecksumPrefix + "md5":     source.Checksums.MD5,
		prefixes.ChecksumPrefix + "adler32": source.Checksums.Adler32,
	}
	attrs.SetString(prefixes.IDAttr, n.ID)
	attrs.SetInt64(prefixes.TypeAttr, int64(provider.ResourceType_RESOURCE_TYPE_FILE))
	attrs.SetString(prefixes.ParentidAttr, n.ParentID)
	attrs.SetString(prefixes.NameAttr, n.Name)
	attrs.SetString(prefixes.BlobIDAttr, n.BlobID)
	attrs.SetInt64(prefixes.BlobsizeAttr, n.Blobsize)
	if err := fs.lu.TimeManager().OverrideMtime(ctx, n, &attrs, mtime); err != nil {
		return nil, errors.Wrap(err, "Decomposedfs: failed to set the mtime")
	}

	newNode := node.New(n.SpaceID, n.ID, n.ParentID, n.Name, n.Blobsize, n.BlobID,
		provider.ResourceType_RESOURCE_TYPE_FILE, nil, fs.lu)
	newNode.SpaceRoot = n.SpaceRoot
	if _, err := newNode.Xattrs(ctx); err != nil {
		return nil, errors.Wrap(err, "Decomposedfs: failed to read node metadata before writing blob")
	}
	newNode.UpdateXattrsCache(attrs)
	if err := fs.tp.WriteBlobFromReader(newNode, source.Body, source.Length); err != nil {
		return nil, errors.Wrap(err, "Decomposedfs: failed to write blob")
	}

	committed := false
	defer func() {
		if committed {
			return
		}
		if err := fs.tp.DeleteBlob(newNode); err != nil {
			appctx.GetLogger(ctx).Error().Err(err).Str("nodeid", n.ID).Str("blobid", n.BlobID).Msg("could not clean up blob after failed upload commit")
		}
	}()

	if err := n.SetXattrsWithContext(ctx, attrs); err != nil {
		return nil, errors.Wrap(err, "Decomposedfs: failed to write metadata")
	}
	committed = true

	if err := fs.tp.Propagate(ctx, n, source.Length-oldSize); err != nil {
		return nil, errors.Wrap(err, "Decomposedfs: failed to propagate")
	}

	etag, _ := node.CalculateEtag(n.ID, mtime)
	return &provider.ResourceInfo{
		Id: &provider.ResourceId{
			StorageId: source.Metadata["providerID"],
			SpaceId:   n.SpaceID,
			OpaqueId:  n.ID,
		},
		Etag:  etag,
		Mtime: utils.TimeToTS(mtime),
	}, nil
}
