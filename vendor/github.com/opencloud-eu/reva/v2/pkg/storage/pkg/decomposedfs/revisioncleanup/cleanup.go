// Copyright 2026 OpenCloud GmbH <mail@opencloud.eu>
// SPDX-License-Identifier: Apache-2.0

// Package revisioncleanup applies a revisions.RevisionCleanupStrategy to the
// revisions of a node, independently of the storage driver that holds them.
//
// # How the storage drivers plug in
//
// All decomposedfs based drivers (decomposed, decomposeds3, posix) share the
// Decomposedfs struct, which implements the CS3 API, the upload sessions and
// RestoreRevision. What differs per driver is the on-disk layout, which each
// driver supplies as a node.Tree through aspects.Aspects:
//
//   - decomposed and decomposeds3 use decomposedfs/tree.Tree. A revision is an
//     empty node file next to the node, '<node path>.REV.<timestamp>', whose
//     metadata references the content as a blob in the blobstore (BlobIDAttr).
//     The two drivers differ only in the blobstore (local disk or S3).
//   - posix uses fs/posix/tree.Tree. A revision is a full copy of the content in
//     the '.oc-nodes' directory of the space (lookup.VersionPath), the posix
//     blobstore does not hold any separate blobs.
//
// In both cases the revision key is '<nodeid>.REV.<RFC3339Nano mtime>', with an
// optional '.<n>' suffix when CreateRevision had to rename a colliding revision.
//
// See pkg/storage/pkg/decomposedfs/README.md for a more detailed overview of the
// driver structure.
//
// The shared code only knows node.Tree. Listing revisions is driver specific
// (node.Tree.ListRevisionsUnchecked), but deleting one is not: Node.DeleteRevisionUnchecked handles
// both layouts, and node.Tree.DeleteBlob deletes the blob in whichever blobstore the
// driver uses. PurgeRevisionUnchecked combines the two. So both the decision about *what* to
// delete and the deletion itself are made here, once for all drivers.
// Neither ListRevisionsUnchecked nor PurgeRevisionUnchecked check permissions: this is system
// housekeeping, which runs e.g. in async postprocessing where there is no suitable
// user in the context.
//
// # When the cleanup runs
//
// Service.HandleRevisionCreated is called after a new revision has been created and has
// settled:
//
//   - after a successful synchronous upload (upload.DecomposedFsSession.FinishUploadDecomposed),
//   - after successful async postprocessing (Decomposedfs.Postprocessing),
//   - in the upload pipeline: when Decomposedfs.MarkProcessing clears the processing mark
//     of an upload,
//   - after Decomposedfs.RestoreRevision,
//   - in posix, when a change on disk turned the '.CURRENT' copy into a revision
//     (assimilation). Those revisions are created by a plain rename and never go
//     through CreateRevision or the upload code, hence the separate trigger.
//
// Besides, the revisions of whole spaces, folders or single files can be cleaned up on demand,
// e.g. by a command line tool, through the OnDemand interface implemented by the storage drivers
// (see Request and Run). That cleanup also reaches files that never get a new revision anymore.
//
// It deliberately does not run inside Tree.CreateRevision, because revisions are
// not settled at that point while uploads are in flight:
//
//   - An upload whose postprocessing is still pending writes its blob into a
//     revision node that a newer upload created for it in the meantime
//     (see DecomposedFsSession.Finalize, "uploading to revision node, that was
//     created for us by another upload").
//   - A failed upload rolls back by restoring the revision recorded in its
//     session's MetaData["versionID"].
//
// The 'auto' strategy keeps only one revision per 10 seconds in the last minute,
// so with rapid uploads a cleanup inside CreateRevision could delete a revision
// that one of those uploads still needs. Instead, Cleanup does nothing at all
// while the node is marked as processing.
//
// # Running in the background
//
// By default (option revisions_cleanup_async) HandleRevisionCreated only queues the node, and
// background workers run Cleaner.Cleanup, so that the cleanup does not slow down the
// upload, restore or change on disk that triggered it (see Service). This also
// shortens the time the node lock is held by those operations. The number of workers,
// the queue size and what happens when the queue is full (block or drop) are
// configurable (AsyncOptions). With revisions_cleanup_async disabled, HandleRevisionCreated runs the
// cleanup right away, in the caller's goroutine.
//
// # Metrics
//
// The cleanup exports Prometheus metrics (see metrics.go), all named
// reva_revision_cleanup_*: counters of deleted revisions, deleted bytes, failed deletions
// and requests that found the queue full, and gauges of the queue length and capacity, the
// workers, the running cleanups and whether the cleanup is enabled. The queue and worker
// gauges are summed over all Services of the process.
//
// Overlapping uploads: the processing status only tracks the latest upload of a node, so a
// cleanup triggered by a newer upload doesn't see an older one that is still in
// postprocessing. That is safe: if the cleanup deletes the revision the older upload writes
// its content into, the older upload creates it again when it finishes; and the revision
// the older upload would roll back to after failed postprocessing must not be restored once
// a newer upload has finished, as that would discard the newer upload's content.
package revisioncleanup

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"time"

	provider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"

	"github.com/opencloud-eu/reva/v2/pkg/appctx"
	"github.com/opencloud-eu/reva/v2/pkg/storage/pkg/decomposedfs/metadata"
	"github.com/opencloud-eu/reva/v2/pkg/storage/pkg/decomposedfs/metadata/prefixes"
	"github.com/opencloud-eu/reva/v2/pkg/storage/pkg/decomposedfs/node"
	"github.com/opencloud-eu/reva/v2/pkg/storage/revisions"
)

// BlobDeleter is the part of node.Tree needed to purge a revision.
type BlobDeleter interface {
	DeleteBlob(n *node.Node) error
}

// RevisionStore is the part of node.Tree needed to clean up revisions.
// It is a separate interface so that the posix tree can pass itself from within
// its own package without the cleanup depending on the full node.Tree.
type RevisionStore interface {
	ListRevisionsUnchecked(ctx context.Context, n *node.Node) ([]*provider.FileVersion, error)
	BlobDeleter
}

// SpaceScoper is implemented by trees whose files may only be accessible to the group of their
// space, i.e. posix with use_space_groups. The cleanup runs on behalf of the system, e.g. in
// background workers, so nothing has switched to that group for it, unlike for user requests,
// where the storage driver's middleware does.
type SpaceScoper interface {
	// ScopeToSpace switches the current goroutine to the group of the space, and returns a
	// context carrying the group, for the goroutines the tree starts itself, e.g. to list a
	// folder, and a function that switches back. The goroutine stays on its OS thread until then.
	ScopeToSpace(ctx context.Context, spaceID string) (context.Context, func() error, error)
}

// scopeToSpace switches to the group of the space of n if rs is a SpaceScoper, see there. It
// returns ctx and a no-op otherwise.
func scopeToSpace(ctx context.Context, rs any, spaceID string) (context.Context, func() error, error) {
	scoper, ok := rs.(SpaceScoper)
	if !ok {
		return ctx, func() error { return nil }, nil
	}
	return scoper.ScopeToSpace(ctx, spaceID)
}

// PurgeRevisionUnchecked physically removes the revision of n with the given key, including its
// blob, metadata and lockfile, without checking permissions. Callers that act on behalf
// of a user must check permissions themselves.
//
// It works for all decomposedfs based drivers:
//   - decomposed, decomposeds3: a revision is the empty node file '<node path>.REV.<timestamp>'
//     with its metadata and lockfile, plus the blob holding the content, referenced by
//     BlobIDAttr in the revision's metadata.
//   - posix: a revision is a full copy of the content in the '.oc-nodes' directory of the
//     space, with its metadata and lockfile. Revisions created from changes on disk by
//     assimilation have the same layout. The posix blobstore does not hold separate blobs,
//     its Delete is a no-op.
//
// Node.DeleteRevisionUnchecked removes everything but the blob, under the node's metadata lock
// (which is reentrant, callers may hold it), and returns the blob id, so that the blob can be
// deleted here. It is a no-op for a revision that does not exist (anymore), e.g. when it was purged
// concurrently.
//
// The key is validated first (node.ValidateRevisionKey), as it is turned into the version that
// Node.DeleteRevisionUnchecked expects, and an empty version would make
// Node.DeleteRevisionUnchecked revert or even purge the node itself instead of deleting a revision.
//
// The blob of a revision is never shared with the current version or other revisions:
// every upload gets a new blob id, CreateRevision moves the reference of the previous
// blob to the revision, and RestoreRevision moves it back while removing the revision
// without deleting its blob. So deleting it here cannot affect anything else.
//
// Decomposedfs.deleteRevisionFile, used by the DeleteFileVersions API and the DeleteRevision
// event, does the same without the key validation. It is left as it is to keep that code
// identical to upstream.
func PurgeRevisionUnchecked(ctx context.Context, bd BlobDeleter, n *node.Node, revisionKey string) error {
	blobID, err := removeRevisionUnchecked(ctx, n, revisionKey)
	if err != nil {
		return err
	}
	return deleteBlob(bd, n.SpaceID, blobID)
}

// removeRevisionUnchecked removes the revision of n with the given key, but not its blob, and
// returns the blob id, see PurgeRevisionUnchecked. It checks no permissions.
func removeRevisionUnchecked(ctx context.Context, n *node.Node, revisionKey string) (string, error) {
	if err := node.ValidateRevisionKey(n, revisionKey); err != nil {
		return "", err
	}
	return n.DeleteRevisionUnchecked(ctx, strings.TrimPrefix(revisionKey, n.ID+node.RevisionIDDelimiter))
}

// deleteBlob deletes the blob of a removed revision. An empty blob id means there is no blob:
// a zero byte revision, or no such revision.
func deleteBlob(bd BlobDeleter, spaceID, blobID string) error {
	if blobID == "" {
		return nil
	}
	// DeleteBlob only needs the blob coordinates, not a full node
	return bd.DeleteBlob(&node.Node{BaseNode: node.BaseNode{SpaceID: spaceID}, BlobID: blobID})
}

// Cleaner deletes old revisions according to a cleanup strategy.
// A disabled Cleaner, created by New without a strategy, never deletes anything; see
// Enabled. A nil *Cleaner behaves the same, as a safeguard, but New never returns one.
//
// It is stateless apart from the parsed strategy and safe for concurrent use.
// The storage drivers use it through a Service, created from their options by
// NewServiceFromOptions and passed on through aspects.Aspects.RevisionsCleanup.
type Cleaner struct {
	// nil if the cleanup is disabled
	strategy revisions.RevisionCleanupStrategy
	// the specification the strategy was parsed from, logged so that support can tell
	// which configuration was in effect from the log output alone
	config string
}

// New creates a Cleaner from a strategy specification as understood by
// revisions.NewRevisionCleanupStrategy. An empty or "disabled" specification
// returns a disabled Cleaner.
//
// A disabled Cleaner, rather than nil, lets the call sites call Cleanup unconditionally,
// without the risk of a nil pointer, while it still skips all the work, including taking
// the node lock, when the feature is off, which is the default.
func New(config string) (*Cleaner, error) {
	config = strings.TrimSpace(config)
	if config == "" || config == "disabled" {
		return &Cleaner{config: "disabled"}, nil
	}
	strategy, err := revisions.NewRevisionCleanupStrategy(config)
	if err != nil {
		return nil, err
	}
	return &Cleaner{strategy: strategy, config: config}, nil
}

// Enabled tells whether c cleans up anything, i.e. whether it was created with a strategy.
// It is false for a disabled Cleaner, and for a nil one.
func (c *Cleaner) Enabled() bool {
	return c != nil && c.strategy != nil
}

// keptOnRequest is the reason for revisions that the operation triggering the cleanup asked
// to keep, see Cleanup.
const keptOnRequest = "protected: kept on request of the operation that triggered the cleanup (e.g. the state from before a restore)"

// Cleanup deletes the revisions of n that the strategy does not want to keep.
// Revisions with a key in keep are never deleted.
//
// It locks the metadata of n (the lock is reentrant, callers may already hold it)
// and does nothing while an upload to n is still being processed, as that upload
// may still need existing revisions to write its blob into or to roll back to.
// See the package documentation for the details. The blobs of the deleted revisions are
// deleted after the lock is released. If rs is a SpaceScoper, it switches to the group of the
// space of n first.
//
// Errors are logged, and reported in the Result, but not returned: a failed cleanup must
// not fail the operation that triggered it. Callers that don't report on the cleanup can
// ignore the Result.
//
// It checks no permissions: it is housekeeping on behalf of the system, e.g. after async
// postprocessing, where there is no suitable user in the context. Callers decide which nodes
// to clean up.
func (c *Cleaner) Cleanup(ctx context.Context, lu node.PathLookup, rs RevisionStore, n *node.Node, keep ...string) Result {
	return c.run(ctx, lu, rs, n, false, keep)
}

// DryRun decides like Cleanup, but deletes nothing: the Result tells which revisions would
// be deleted, kept or protected, and why. It neither counts metrics nor logs purges.
//
// Like Cleanup, it checks no permissions.
func (c *Cleaner) DryRun(ctx context.Context, lu node.PathLookup, rs RevisionStore, n *node.Node, keep ...string) Result {
	return c.run(ctx, lu, rs, n, true, keep)
}

// The result is named, so that the deletion of the blobs, which runs deferred, can report its
// errors in it.
func (c *Cleaner) run(ctx context.Context, lu node.PathLookup, rs RevisionStore, n *node.Node, dryRun bool, keep []string) (result Result) {
	if !c.Enabled() || n == nil {
		return Result{DryRun: dryRun, Skipped: "the revision cleanup is disabled"}
	}
	result = Result{SpaceID: n.SpaceID, NodeID: n.ID, DryRun: dryRun}
	log := appctx.GetLogger(ctx).With().Str("spaceid", n.SpaceID).Str("nodeid", n.ID).Logger()

	ctx, unscope, err := scopeToSpace(ctx, rs, n.SpaceID)
	if err != nil {
		log.Error().Err(err).Msg("revision cleanup: could not switch to the group of the space")
		result.Err = fmt.Errorf("could not switch to the group of the space: %w", err)
		return result
	}
	defer func() { _ = unscope() }()

	// The blobs of the purged revisions are deleted once the node lock is released (this defer
	// runs after the one releasing it): their revisions are gone by then, so nothing refers to
	// them anymore, and deleting a blob can be slow, e.g. one request per blob with
	// decomposeds3, which would block uploads to the node all along.
	type pendingBlob struct {
		key    string // of the revision the blob belonged to
		blobID string
		size   uint64 // of the revision
	}
	var blobs []pendingBlob
	defer func() {
		for _, b := range blobs {
			if err := deleteBlob(rs, n.SpaceID, b.blobID); err != nil {
				// the revision is gone, but not its blob: that's a failure, not a deletion
				PurgeFailures.Inc()
				log.Error().Err(err).Str("revision", b.key).Str("blobid", b.blobID).Msg("revision cleanup: could not delete the blob of a purged revision")
				for i := range result.Revisions {
					if result.Revisions[i].Key == b.key {
						result.Revisions[i].Err = fmt.Errorf("the revision was deleted, but not its blob %q: %w", b.blobID, err)
					}
				}
				continue
			}
			RevisionsDeleted.Inc()
			BytesDeleted.Add(float64(b.size))
		}
	}()

	// Holding the node lock serializes the cleanup with uploads to the same node:
	// those create revisions and mark the node as processing while holding it, so
	// the processing check below cannot race with a new upload starting.
	// Revisions have their own lockfiles, purging them while holding the node lock
	// does not deadlock. A dry run takes the lock as well, to decide on the same
	// consistent state a real cleanup would see.
	unlock, err := lu.MetadataBackend().Lock(n)
	if err != nil {
		log.Error().Err(err).Msg("revision cleanup: could not lock node")
		result.Err = fmt.Errorf("could not lock node: %w", err)
		return result
	}
	defer func() { _ = unlock() }()

	// Read the status from the backend and not via n.IsProcessing(): n caches its
	// attributes, and after an upload the processing flag is removed by
	// session.Cleanup through a different *node.Node instance, so the cache of n
	// would still say "processing" and the cleanup would never run.
	status, err := lu.MetadataBackend().Get(ctx, n, prefixes.StatusPrefix)
	switch {
	case err != nil && !metadata.IsAttrUnset(err):
		log.Error().Err(err).Msg("revision cleanup: could not read node status")
		result.Err = fmt.Errorf("could not read node status: %w", err)
		return result
	case strings.HasPrefix(string(status), node.ProcessingStatus):
		log.Debug().Msg("revision cleanup: node is processing, skipping")
		result.Skipped = "an upload to the file is still being processed"
		return result
	}

	versions, err := rs.ListRevisionsUnchecked(ctx, n)
	if err != nil {
		log.Error().Err(err).Msg("revision cleanup: could not list revisions")
		result.Err = fmt.Errorf("could not list revisions: %w", err)
		return result
	}
	if len(versions) == 0 {
		return result
	}

	revs := make([]revisions.Revision, len(versions))
	for i, v := range versions {
		revs[i] = fileVersionRevision{version: v, mtime: revisionTime(v)}
	}

	// All revisions, including the ones in keep, are handed to the strategy, as
	// count based strategies ("keep the n newest") need to see the complete set to
	// decide correctly. keep is only applied to the strategy's deletion plan.
	purgeErrs := map[string]error{}
	purged := 0
	var purgedBytes uint64
	plan, err := revisions.ExecuteCleanupStrategy(time.Now(), c.strategy, revs, ctx, &log, func(rev revisions.Revision, reason string) error {
		if dryRun || slices.Contains(keep, rev.Key()) {
			return nil
		}
		blobID, err := removeRevisionUnchecked(ctx, n, rev.Key())
		if err != nil {
			PurgeFailures.Inc()
			purgeErrs[rev.Key()] = err
			return fmt.Errorf("could not purge revision %q: %w", rev.Key(), err)
		}
		blobs = append(blobs, pendingBlob{key: rev.Key(), blobID: blobID, size: rev.Size()})
		purged++
		purgedBytes += rev.Size()
		// the reason explains which rule deleted the revision, e.g. which slot of the 'auto'
		// strategy, see revisions.Plan
		log.Debug().Str("revision", rev.Key()).Uint64("size", rev.Size()).Str("reason", reason).Msg("revision cleanup: purged revision")
		return nil
	})
	if plan == nil {
		// the strategy itself failed
		log.Error().Err(err).Msg("revision cleanup: failed")
		result.Err = err
		return result
	}
	result.Revisions = revisionResults(revs, plan, keep, purgeErrs)

	// One summary per node at info level, so that admins can see the cleanup at work
	// without the per revision debug logging. Nothing is logged when nothing was purged,
	// which is the common case for nodes with few revisions.
	if purged > 0 {
		log.Info().Str("config", c.config).Int("purged", purged).Uint64("purgedbytes", purgedBytes).
			Int("revisions", len(versions)).Msg("revision cleanup: purged old revisions")
	}
	if err != nil {
		// a revision that could not be purged is still listed and will be retried
		// by the next cleanup of this node
		log.Error().Err(err).Msg("revision cleanup: failed")
	}
	return result
}

// revisionResults turns the plan of the strategy into one RevisionResult per revision, in the
// order of revs (newest first).
func revisionResults(revs []revisions.Revision, plan revisions.Plan, keep []string, purgeErrs map[string]error) []RevisionResult {
	decisions := map[string]Decision{}
	for _, rev := range plan.Deleted() {
		decisions[rev.Key()] = Deleted
	}
	for _, rev := range plan.Retained() {
		decisions[rev.Key()] = Kept
	}
	for _, rev := range plan.Safeguarded() {
		decisions[rev.Key()] = Protected
	}

	results := make([]RevisionResult, 0, len(revs))
	sorted := slices.Clone(revs)
	revisions.SortRevisionsCanonically(sorted)
	for _, rev := range sorted {
		r := RevisionResult{
			Key:      rev.Key(),
			Time:     rev.LastModificationTime(),
			Size:     rev.Size(),
			Decision: decisions[rev.Key()],
			Reason:   plan.Reason(rev.Key()),
			Err:      purgeErrs[rev.Key()],
		}
		if r.Decision == Deleted && slices.Contains(keep, rev.Key()) {
			r.Decision = Protected
			r.Reason = keptOnRequest
		}
		results = append(results, r)
	}
	return results
}

// revisionTime determines when a revision was last modified from the timestamp in its key,
// '<nodeid>.REV.<RFC3339Nano>' with an optional '.<n>' suffix for colliding revisions.
// The mtime of the revision file is used as a fallback.
//
// This is the age the strategies work with, and it is the age of the revision's *content*:
// revisions are named after the mtime of the content they preserve (see
// DecomposedFsStore.updateExistingNode, the posix assimilation and RestoreRevision), not after
// the time the revision was created, i.e. when that content was replaced. The time a revision
// was created is not recorded anywhere. For a file that was unchanged for two months and is
// changed today, the new revision therefore counts as two months old, and a strategy deleting
// by duration, e.g. "30d", deletes it right away. See "How the age of a revision is measured"
// in README.md.
// For changes on disk with posix, the name is the time the content was copied to '.CURRENT',
// when that change was picked up, normally right after the content was written (see the posix
// assimilation).
// Whether the age should rather be the time a revision was created, i.e. when its content was
// replaced, is discussed in https://github.com/opencloud-eu/opencloud/issues/3735.
//
// The key is preferred over the file mtime because the file mtime is not reliable:
// the upload code only sets it to the mtime of the old content (os.Chtimes in
// DecomposedFsStore.updateExistingNode) after CreateRevision has returned, and the
// FileVersion.Mtime from ListRevisionsUnchecked has only a precision of seconds.
// The '.<n>' suffix is appended by CreateRevision when a revision with the same
// timestamp already exists, it is not part of the timestamp and is stripped.
func revisionTime(v *provider.FileVersion) time.Time {
	if _, ts, ok := strings.Cut(v.Key, node.RevisionIDDelimiter); ok {
		if t, err := time.Parse(time.RFC3339Nano, ts); err == nil {
			return t
		}
		if i := strings.LastIndex(ts, "."); i > 0 {
			if t, err := time.Parse(time.RFC3339Nano, ts[:i]); err == nil {
				return t
			}
		}
	}
	return time.Unix(int64(v.Mtime), 0)
}

// fileVersionRevision adapts a CS3 FileVersion, as returned by
// node.Tree.ListRevisionsUnchecked, to the revisions.Revision interface the
// cleanup strategies work on.
type fileVersionRevision struct {
	version *provider.FileVersion
	mtime   time.Time
}

var _ revisions.Revision = fileVersionRevision{}

func (f fileVersionRevision) Key() string                     { return f.version.Key }
func (f fileVersionRevision) LastModificationTime() time.Time { return f.mtime }
func (f fileVersionRevision) Size() uint64                    { return f.version.Size }
