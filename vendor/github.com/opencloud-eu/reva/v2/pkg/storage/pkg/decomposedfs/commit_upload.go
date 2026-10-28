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
	"time"

	provider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	"github.com/pkg/errors"

	"github.com/opencloud-eu/reva/v2/pkg/appctx"
	"github.com/opencloud-eu/reva/v2/pkg/errtypes"
	"github.com/opencloud-eu/reva/v2/pkg/storage"
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
	// released before the revision cleanup, or on any return before that
	unlocked := false
	unlockNode := func() {
		if unlocked {
			return
		}
		unlocked = true
		if err := unlock(); err != nil {
			appctx.GetLogger(ctx).Error().Err(err).Str("nodeid", n.ID).Msg("could not close mark-processing lock")
		}
	}
	defer unlockNode()

	if !processing {
		if !n.IsProcessing(ctx) {
			return nil
		}
		id, _ := n.ProcessingID(ctx)
		if id != sessionID {
			return nil
		}
		if err := n.RemoveXattr(ctx, prefixes.StatusPrefix); err != nil {
			return err
		}
		// The upload of this session is done: apply the revision cleanup strategy (see package
		// revisioncleanup), which skipped the node while it was marked. It never fails clearing
		// the mark. The node lock is released first, for the reasons given in RestoreRevision.
		unlockNode()
		fs.revisionsCleanup.HandleRevisionCreated(ctx, n)
		return nil
	}

	if n.IsProcessing(ctx) {
		return errtypes.ResourceProcessing(ref.String())
	}
	return n.SetXattrsWithContext(ctx, node.Attributes{
		prefixes.StatusPrefix: []byte(node.ProcessingStatus + sessionID),
	})
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
