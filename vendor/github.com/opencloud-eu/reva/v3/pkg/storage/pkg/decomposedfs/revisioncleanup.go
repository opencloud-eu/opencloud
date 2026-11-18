// Copyright 2026 OpenCloud GmbH <mail@opencloud.eu>
// SPDX-License-Identifier: Apache-2.0

package decomposedfs

import (
	"context"
	"fmt"
	"maps"
	"math/rand/v2"
	"os"
	"slices"
	"strings"

	provider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	"github.com/pkg/errors"

	"github.com/opencloud-eu/reva/v3/pkg/appctx"
	"github.com/opencloud-eu/reva/v3/pkg/errtypes"
	"github.com/opencloud-eu/reva/v3/pkg/storage"
	"github.com/opencloud-eu/reva/v3/pkg/storage/pkg/decomposedfs/metadata"
	"github.com/opencloud-eu/reva/v3/pkg/storage/pkg/decomposedfs/node"
	"github.com/opencloud-eu/reva/v3/pkg/storage/pkg/decomposedfs/revisioncleanup"
)

var _ revisioncleanup.OnDemand = (*Decomposedfs)(nil)

var _ storage.UploadFinishedHandler = (*Decomposedfs)(nil)

// HandleUploadFinished applies the revision cleanup strategy (see package revisioncleanup) to
// the file of an upload that has finished, if the upload created a revision, see
// storage.UploadFinishedHandler. Uploads that created the file, and rolled back ones, which
// never get here, leave nothing to clean up. It never fails the upload.
//
// The revision a newer upload kept for an older one only becomes a version when the older
// upload writes its content into it, see CommitUpload, which triggers the cleanup for that case
// itself: versionCreated of the older upload doesn't say anything about that revision.
func (fs *Decomposedfs) HandleUploadFinished(ctx context.Context, ref *provider.Reference, _ string, versionCreated bool) {
	if !versionCreated {
		return
	}
	// The cleanup takes the file by its ids, without resolving a path. The coordinator always
	// passes them: it sets the space id and the node id of a session when the upload is
	// initiated, also for a new file, whose node id it chooses itself. A reference without
	// them, e.g. one with only a path, would come from another caller and be a bug there:
	// it is logged, and the file is left as it is, as a missed cleanup only keeps revisions
	// a little longer, until the next one.
	id := ref.GetResourceId()
	if id.GetSpaceId() == "" || id.GetOpaqueId() == "" {
		appctx.GetLogger(ctx).Error().Interface("ref", ref).Msg("revision cleanup: a finished upload without the ids of its file, not cleaning up")
		return
	}
	fs.revisionsCleanup.HandleRevisionCreatedByID(ctx, id.GetSpaceId(), id.GetOpaqueId())
}

// CleanupRevisionsUnchecked cleans up the revisions of the files of whole spaces, or of a file or
// folder within a space, on demand, see revisioncleanup.OnDemand. It is meant for command line
// tools: it checks no permissions and needs no user in the context.
//
// With Request.All, a space that can't be cleaned up doesn't stop the others: a disabled space
// (in the trash) is reported as skipped, any other one as failed. A space requested by its id
// fails the request instead.
//
// Paths on disk (Request.FSPath) are not supported: the decomposed layout stores files under
// their ids. The posix driver resolves them before it calls this.
//
// The strategy and the AsyncOptions default to the configured ones (the
// revisions_cleanup_* options) and can be overridden in the request; without AsyncOptions in
// the request, the run waits for room in a full queue as long as it takes, see
// revisioncleanup.Request.
func (fs *Decomposedfs) CleanupRevisionsUnchecked(ctx context.Context, req revisioncleanup.Request, report func(revisioncleanup.Result)) error {
	if err := req.Validate(); err != nil {
		return err
	}
	if req.FSPath != "" {
		return errtypes.NotSupported("cleaning up revisions by a path on disk, only the posix driver supports it")
	}
	if req.DryRun {
		// a dry run changes nothing on disk, not even the lock files of what it reads
		ctx = metadata.WithReadOnly(ctx)
	}
	w, ok := fs.tp.(revisioncleanup.Walker)
	if !ok {
		return errtypes.NotSupported("cleaning up revisions on demand, the tree can't list folders for maintenance")
	}
	if report == nil {
		report = func(revisioncleanup.Result) {}
	}

	strategy := req.Strategy
	if strategy == "" {
		strategy = fs.o.RevisionsCleanupStrategy
	}
	cleaner, err := revisioncleanup.New(strategy)
	if err != nil {
		return errors.Wrap(err, "invalid revisions cleanup strategy")
	}
	if !cleaner.Enabled() {
		return errors.New("no revisions cleanup strategy configured or given")
	}
	o := revisioncleanup.RunOptions{Async: revisioncleanup.AsyncOptionsFromOptions(fs.o), DryRun: req.DryRun, Shuffle: req.Shuffle}
	// a run queues the files as fast as it finds them: dropping them, or giving up after
	// the timeout meant for uploads, would skip most of them
	o.Async.QueueFull, o.Async.QueueFullTimeout = revisioncleanup.QueueFullBlock, 0
	if req.Async != nil {
		o.Async = *req.Async
	}

	roots, err := fs.revisionCleanupRoots(ctx, req, report)
	if err != nil {
		return err
	}
	return revisioncleanup.Run(ctx, fs.lu, w, roots, cleaner, o, report, fs.log)
}

// errSpaceDisabled is the error for a space that is disabled, i.e. in the trash
var errSpaceDisabled = errors.New("the space is disabled")

// revisionCleanupRoots resolves the scope of the request to the nodes to clean up below. With
// Request.All, it reports the spaces that can't be cleaned up and leaves them out.
func (fs *Decomposedfs) revisionCleanupRoots(ctx context.Context, req revisioncleanup.Request, report func(revisioncleanup.Result)) ([]*node.Node, error) {
	// a space given more than once is cleaned up once
	spaceIDs := uniqueInOrder(req.SpaceIDs)
	if req.All {
		ids, err := fs.allSpaceIDs()
		if err != nil {
			return nil, err
		}
		spaceIDs = ids
	}
	// Request.Shuffle randomizes the order in two places: here, the order of the spaces, and in
	// revisioncleanup.Run, the order of the files and folders within each folder it walks
	// through (RunOptions.Shuffle).
	if req.Shuffle {
		spaceIDs = slices.Clone(spaceIDs)
		rand.Shuffle(len(spaceIDs), func(i, j int) { spaceIDs[i], spaceIDs[j] = spaceIDs[j], spaceIDs[i] })
	}

	roots := make([]*node.Node, 0, len(spaceIDs))
	for _, spaceID := range spaceIDs {
		n, err := fs.revisionCleanupRootScoped(ctx, spaceID, req.Path)
		// Stopped, before or during the lookup: the run ends here. A lookup that honours the
		// context, e.g. of the posix id cache, fails then, but the space hasn't failed.
		if ctxErr := ctx.Err(); ctxErr != nil {
			return nil, ctxErr
		}
		switch {
		case err == nil:
			roots = append(roots, n)
		case !req.All:
			return nil, err
		case errors.Is(err, errSpaceDisabled):
			report(revisioncleanup.Result{SpaceID: spaceID, NodeID: spaceID, DryRun: req.DryRun, Skipped: errSpaceDisabled.Error()})
		default:
			// one space that can't be cleaned up doesn't stop the others
			report(revisioncleanup.Result{SpaceID: spaceID, NodeID: spaceID, DryRun: req.DryRun, Err: err})
		}
	}
	return roots, nil
}

// revisionCleanupRootScoped resolves the root, see revisionCleanupRoot, switched to the group of
// its space if the tree is a revisioncleanup.SpaceScoper: with posix use_space_groups, the
// folders on the way to the file or folder at path may only be accessible to that group, like
// the folders the cleanup walks through below it.
func (fs *Decomposedfs) revisionCleanupRootScoped(ctx context.Context, spaceID, path string) (*node.Node, error) {
	scoper, ok := fs.tp.(revisioncleanup.SpaceScoper)
	if !ok {
		return fs.revisionCleanupRoot(ctx, spaceID, path)
	}
	scoped, unscope, err := scoper.ScopeToSpace(ctx, spaceID)
	if err != nil {
		return nil, errors.Wrapf(err, "could not switch to the group of space %q", spaceID)
	}
	defer func() { _ = unscope() }()
	return fs.revisionCleanupRoot(scoped, spaceID, path)
}

// revisionCleanupRoot resolves the space, or the file or folder at path within it.
func (fs *Decomposedfs) revisionCleanupRoot(ctx context.Context, spaceID, path string) (*node.Node, error) {
	var n *node.Node
	var err error
	if path == "" {
		n, err = fs.lu.NodeFromSpaceID(ctx, spaceID)
	} else {
		n, err = fs.lu.NodeFromResource(ctx, &provider.Reference{
			ResourceId: &provider.ResourceId{SpaceId: spaceID, OpaqueId: spaceID},
			Path:       "./" + strings.TrimLeft(path, "/"),
		})
	}
	if err == nil && n != nil && n.Exists {
		return n, nil
	}
	// the lookups don't find disabled spaces, tell them apart from missing ones
	if root, rerr := node.ReadNode(ctx, fs.lu, spaceID, spaceID, "", true, nil, false); rerr == nil && root.Exists && root.IsDisabled(ctx) {
		return nil, fmt.Errorf("space %q: %w", spaceID, errSpaceDisabled)
	}
	switch {
	case err != nil:
		return nil, errors.Wrapf(err, "could not find space %q", spaceID)
	case path != "":
		return nil, fmt.Errorf("no file or folder %q in space %q", path, spaceID)
	default:
		return nil, fmt.Errorf("no space %q", spaceID)
	}
}

// allSpaceIDs lists the ids of all spaces from the space type index, of all space types,
// without permission checks.
func (fs *Decomposedfs) allSpaceIDs() ([]string, error) {
	spaceTypes, err := fs.spaceTypeIndex.Indexes()
	if err != nil {
		return nil, errors.Wrap(err, "could not list the space types")
	}
	// a space can be in the index of more than one type, collect each once
	unique := map[string]struct{}{}
	for _, spaceType := range spaceTypes {
		spaces, err := fs.spaceTypeIndex.Load(spaceType)
		if err != nil {
			if os.IsNotExist(err) {
				continue // no spaces of this type
			}
			return nil, errors.Wrapf(err, "could not read the index of %q spaces", spaceType)
		}
		for spaceID := range spaces {
			unique[spaceID] = struct{}{}
		}
	}
	return slices.Sorted(maps.Keys(unique)), nil
}

// uniqueInOrder returns ids without duplicates, in the order of their first occurrence.
func uniqueInOrder(ids []string) []string {
	seen := make(map[string]struct{}, len(ids))
	unique := make([]string, 0, len(ids))
	for _, id := range ids {
		if _, ok := seen[id]; !ok {
			seen[id] = struct{}{}
			unique = append(unique, id)
		}
	}
	return unique
}
