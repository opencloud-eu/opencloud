// Copyright 2026 OpenCloud GmbH <mail@opencloud.eu>
// SPDX-License-Identifier: Apache-2.0

package upload

import (
	"context"
	"errors"
	iofs "io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/opencloud-eu/reva/v2/pkg/appctx"
	"github.com/opencloud-eu/reva/v2/pkg/storage/pkg/decomposedfs/metadata/prefixes"
	"github.com/opencloud-eu/reva/v2/pkg/storage/pkg/decomposedfs/node"
	"github.com/rs/zerolog"
)

// interruptedUploadAge is how old the session of a processing node has to be
// before the upload is taken for interrupted. Until then it might still be
// running on an instance that does not write blobs first yet.
const interruptedUploadAge = 10 * time.Minute

// RecoverInterruptedUpload finishes a synchronous upload that was interrupted
// after its node had been switched to the new blob, e.g. by a crash. Older
// versions marked the node as processing for that time, and as nothing
// postprocesses synchronous uploads, the marker stayed forever. It reports
// whether the node is no longer processing.
func (store DecomposedFsStore) RecoverInterruptedUpload(ctx context.Context, n *node.Node) bool {
	if !store.separateBlobs || store.async {
		// processing markers are expected here, postprocessing removes them
		return false
	}
	log := appctx.GetLogger(ctx).With().Str("spaceid", n.SpaceID).Str("nodeid", n.ID).Logger()

	unlock, err := store.lu.MetadataBackend().Lock(n)
	if err != nil {
		log.Error().Err(err).Msg("could not lock node to recover an interrupted upload")
		return false
	}
	done, recovered := store.finishInterruptedUpload(ctx, n, log)
	_ = unlock()

	// Let clients see the change, once the node is unlocked like finished
	// uploads do. The size difference might have been counted already, e.g.
	// if the upload started while the store was async, so the tree size is
	// left as it is.
	if recovered {
		if err := store.tp.Propagate(ctx, n, 0); err != nil {
			log.Error().Err(err).Msg("could not propagate a recovered upload")
		}
	}
	return done
}

// finishInterruptedUpload does the work of RecoverInterruptedUpload with the
// node locked. Besides whether the node is no longer processing, it reports
// whether it recovered the upload itself.
func (store DecomposedFsStore) finishInterruptedUpload(ctx context.Context, n *node.Node, log zerolog.Logger) (done, recovered bool) {
	// the node might have changed while waiting for the lock
	n.ResetXattrsCache()
	attrs, err := n.Xattrs(ctx)
	if err != nil {
		log.Error().Err(err).Msg("could not read node to recover an interrupted upload")
		return false, false
	}
	n.BlobID = attrs.String(prefixes.BlobIDAttr)
	n.Blobsize, _ = attrs.Int64(prefixes.BlobsizeAttr)
	status := attrs.String(prefixes.StatusPrefix)
	if !strings.HasPrefix(status, node.ProcessingStatus) {
		return true, false
	}
	// uploads point the node to a blob named after their session, other
	// markers are none of our business
	uploadID := strings.TrimPrefix(status, node.ProcessingStatus)
	if uploadID == "" || n.BlobID != uploadID {
		return false, false
	}
	log = log.With().Str("uploadid", uploadID).Logger()

	infoPath := sessionPath(store.root, uploadID)
	binPath := filepath.Join(store.root, "uploads", uploadID)
	if info, err := os.Stat(infoPath); err == nil && time.Since(info.ModTime()) < interruptedUploadAge {
		return false, false
	}

	if _, err := os.Stat(binPath); err == nil {
		// the upload stopped before its blob was stored
		if err := store.tp.WriteBlob(n, binPath); err != nil {
			log.Error().Err(err).Msg("could not store the blob of an interrupted upload")
			return false, false
		}
	}
	blob, err := store.tp.ReadBlob(n)
	if err != nil {
		log.Error().Err(err).Msg("the blob of an interrupted upload is missing, the node stays marked as processing")
		return false, false
	}
	_ = blob.Close()

	if err := n.RemoveXattr(ctx, prefixes.StatusPrefix); err != nil {
		log.Error().Err(err).Msg("could not remove the processing status of an interrupted upload")
		return false, false
	}
	for _, p := range []string{binPath, infoPath} {
		if err := os.Remove(p); err != nil && !errors.Is(err, iofs.ErrNotExist) {
			log.Error().Err(err).Str("path", p).Msg("could not remove a file of an interrupted upload")
		}
	}
	log.Warn().Msg("recovered interrupted upload")
	return true, true
}
