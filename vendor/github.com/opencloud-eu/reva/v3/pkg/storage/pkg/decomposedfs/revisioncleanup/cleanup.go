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
// (node.Tree.ListRevisions), but deleting one is not: Node.DeleteRevisionUnchecked handles
// both layouts, and node.Tree.DeleteBlob deletes the blob in whichever blobstore the
// driver uses. PurgeRevisionUnchecked combines the two. So both the decision about *what* to
// delete and the deletion itself are made here, once for all drivers.
// Neither ListRevisions nor PurgeRevisionUnchecked check permissions: this is system
// housekeeping, which runs e.g. at the end of an upload, where there is no suitable
// user in the context.
//
// # When the cleanup runs
//
// Service.HandleRevisionCreated is called after a new revision has been created and has
// settled:
//
//   - after an upload that replaced a file, once the upload has finished: the upload
//     coordinator (pkg/upload) calls Decomposedfs.HandleUploadFinished for every successful
//     upload, whether it was committed right away or after its postprocessing. Uploads that
//     created the file, and uploads that were rolled back, don't trigger it,
//   - in Decomposedfs.CommitUpload, when an upload that a newer one superseded wrote its
//     content into the revision the newer upload created for it, which makes that revision
//     a version (see "Overlapping uploads" below),
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
//     (see Decomposedfs.CommitUpload).
//   - A failed upload rolls back by restoring the revision it created
//     (see Decomposedfs.RollbackUpload).
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
// Overlapping uploads: the processing mark of a node only tracks its latest upload, so a
// cleanup triggered by a newer upload runs while an older one may still be in
// postprocessing. The revision the newer upload created for the older one keeps the older
// upload's processing mark, and the older upload writes its content into it when it
// finishes. The cleanup never deletes such a revision, see Cleaner: the older upload would
// then discard its content. Once the older upload has written its content into it and
// removed the mark, the revision is a version, and CommitUpload runs the cleanup for it.
package revisioncleanup

import (
	"cmp"
	"context"
	"fmt"
	"math"
	"slices"
	"strconv"
	"strings"
	"time"

	provider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"

	"github.com/opencloud-eu/reva/v3/pkg/appctx"
	"github.com/opencloud-eu/reva/v3/pkg/storage/pkg/decomposedfs/metadata"
	"github.com/opencloud-eu/reva/v3/pkg/storage/pkg/decomposedfs/metadata/prefixes"
	"github.com/opencloud-eu/reva/v3/pkg/storage/pkg/decomposedfs/node"
	"github.com/opencloud-eu/reva/v3/pkg/storage/revisions"
)

// BlobDeleter is the part of node.Tree needed to purge a revision.
type BlobDeleter interface {
	DeleteBlob(n *node.Node) error
}

// RevisionStore is the part of node.Tree needed to clean up revisions.
// It is a separate interface so that the posix tree can pass itself from within
// its own package without the cleanup depending on the full node.Tree.
type RevisionStore interface {
	ListRevisions(ctx context.Context, n *node.Node) ([]*provider.FileVersion, error)
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
//
// A goroutine that switched to the group of the space already doesn't switch again: a worker
// switches before it reads the node, and the Cleaner then runs in the same goroutine. Switching
// costs a stat of the root of the space and two system calls. The context records the space the
// goroutine switched to; it must not be handed to another goroutine that should run in the
// group, which has to switch on its own.
func scopeToSpace(ctx context.Context, rs any, spaceID string) (context.Context, func() error, error) {
	noop := func() error { return nil }
	scoper, ok := rs.(SpaceScoper)
	if !ok {
		return ctx, noop, nil
	}
	if scoped, _ := ctx.Value(scopedToSpaceKey{}).(string); scoped == spaceID {
		return ctx, noop, nil
	}
	ctx, unscope, err := scoper.ScopeToSpace(ctx, spaceID)
	if err != nil {
		return ctx, nil, err
	}
	return context.WithValue(ctx, scopedToSpaceKey{}, spaceID), unscope, nil
}

// scopedToSpaceKey is the context key of the id of the space whose group the current goroutine
// switched to, see scopeToSpace.
type scopedToSpaceKey struct{}

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
// event, does the same without the key validation. It is left as it is for now, using one
// for the other is https://github.com/opencloud-eu/opencloud/issues/3734.
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

// keptAfterRestore is the reason for the revisions a restore keeps, see keptAfterRestoreKeys.
const keptAfterRestore = "protected: holds the content a restore replaced, as long as the file has the restored content"

// Cleanup deletes the revisions of n that the strategy does not want to keep.
// The revision a restore created of the content it replaced is never deleted as long as the
// file has the restored content, see node.KeepAfterRestoreAttr.
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
func (c *Cleaner) Cleanup(ctx context.Context, lu node.PathLookup, rs RevisionStore, n *node.Node) Result {
	return c.run(ctx, lu, rs, n, false, true)
}

// DryRun decides like Cleanup, but deletes nothing: the Result tells which revisions would
// be deleted, kept or protected, and why. It neither counts metrics nor logs purges.
//
// It changes nothing on disk: it doesn't lock n, and reads its metadata under
// metadata.WithReadOnly, so that no lock file is created. Its decisions are a snapshot, which
// an upload to n may change meanwhile.
//
// Like Cleanup, it checks no permissions.
func (c *Cleaner) DryRun(ctx context.Context, lu node.PathLookup, rs RevisionStore, n *node.Node) Result {
	return c.run(ctx, lu, rs, n, true, true)
}

// The result is named, so that the deletion of the blobs, which runs deferred, can report its
// errors in it. Without details, it has no Revisions: the background cleanup has no one to report
// to, and building them, with the reason for every revision, would be wasted on every upload.
func (c *Cleaner) run(ctx context.Context, lu node.PathLookup, rs RevisionStore, n *node.Node, dryRun, details bool) (result Result) {
	if !c.Enabled() || n == nil {
		return Result{DryRun: dryRun, Skipped: "the revision cleanup is disabled"}
	}
	result = Result{SpaceID: n.SpaceID, NodeID: n.ID, DryRun: dryRun}
	if dryRun {
		// also the backends that lock to read must not create lock files
		ctx = metadata.WithReadOnly(ctx)
	}
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
	// the blob of the node itself, read under the node lock below
	var currentBlobID string
	defer func() {
		for _, b := range blobs {
			if currentBlobID != "" && b.blobID == currentBlobID {
				// A revision normally has a blob of its own. Should one ever share the blob of
				// the node, deleting it would delete the current content of the file, so it is
				// left alone (as CommitUpload does for the revisions it removes).
				log.Warn().Str("revision", b.key).Str("blobid", b.blobID).Msg("revision cleanup: not deleting the blob of a purged revision, the file still uses it")
				// the revision is deleted all the same, but it freed no space
				RevisionsDeleted.Inc()
				continue
			}
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
	// does not deadlock.
	// A dry run doesn't take the lock: it must not change anything on disk, and locking
	// creates the lock file of the node when there is none yet. Its decisions are a
	// snapshot, which an upload to the node may change meanwhile; a real cleanup decides
	// again, under the lock.
	if !dryRun {
		unlock, err := lu.MetadataBackend().Lock(n)
		if err != nil {
			log.Error().Err(err).Msg("revision cleanup: could not lock node")
			result.Err = fmt.Errorf("could not lock node: %w", err)
			return result
		}
		defer func() { _ = unlock() }()
	}

	// Read the status and the blob id from storage, under the lock (but in a dry run),
	// and neither via n.IsProcessing() nor from the cache of the backend:
	//   - n caches its attributes, and after an upload the processing flag is removed
	//     through a different *node.Node instance (Decomposedfs.MarkProcessing), so the
	//     cache of n would still say "processing" and the cleanup would never run;
	//   - the cache of the backend may be local to the process, and an on-demand cleanup
	//     runs in a process of its own next to the server: when it cached the node
	//     earlier, e.g. while listing the folder, it would miss an upload the server
	//     started since then, and delete the revision the upload has to roll back to.
	attribs, err := lu.MetadataBackend().Reload(ctx, n)
	if err != nil {
		log.Error().Err(err).Msg("revision cleanup: could not read node status")
		result.Err = fmt.Errorf("could not read node status: %w", err)
		return result
	}
	if strings.HasPrefix(string(attribs[prefixes.StatusPrefix]), node.ProcessingStatus) {
		log.Debug().Msg("revision cleanup: node is processing, skipping")
		result.Skipped = "an upload to the file is still being processed"
		return result
	}
	if blobID, ok := attribs[prefixes.BlobIDAttr]; ok {
		currentBlobID = string(blobID)
	}
	now := time.Now()
	// when the node got its current content, for the estimate in revisionTimes
	fileMTime, err := node.Attributes(attribs).Time(prefixes.MTimeAttr)
	if err != nil {
		fileMTime = now
	}

	versions, err := rs.ListRevisions(ctx, n)
	if err != nil {
		log.Error().Err(err).Msg("revision cleanup: could not list revisions")
		result.Err = fmt.Errorf("could not list revisions: %w", err)
		return result
	}
	times := revisionTimes(ctx, lu, n, versions, fileMTime, now)
	// The strategy only sees versions: the revisions that are not are neither deleted nor
	// counted, see notAVersion.
	versions, skipped := splitOffNotVersions(ctx, lu, n, versions)
	for _, sk := range skipped {
		log.Debug().Str("revision", sk.version.Key).Str("reason", sk.reason).Msg("revision cleanup: leaving a revision alone that is not a version")
	}
	if len(versions) == 0 {
		if details {
			result.Revisions = revisionResults(nil, nil, nil, nil, skipped, times)
		}
		return result
	}
	keep := keptAfterRestoreKeys(ctx, lu, n, versions, currentBlobID, dryRun)

	revs := make([]revisions.Revision, len(versions))
	for i, v := range versions {
		revs[i] = fileVersionRevision{version: v, created: times[v.Key]}
	}

	// All revisions, including the ones in keep, are handed to the strategy, as
	// count based strategies ("keep the n newest") need to see the complete set to
	// decide correctly. keep is only applied to the strategy's deletion plan.
	purgeErrs := map[string]error{}
	purged := 0
	var purgedBytes uint64
	plan, err := revisions.ExecuteCleanupStrategy(now, c.strategy, revs, ctx, &log, func(rev revisions.Revision, reason string) error {
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
	if details {
		result.Revisions = revisionResults(revs, plan, keep, purgeErrs, skipped, times)
	}

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
// order of revs (newest first), together with the revisions that were skipped as they are not
// versions (reported as protected). plan is nil if the strategy didn't run. times are the times of
// the revisions, see revisionTimes.
func revisionResults(revs []revisions.Revision, plan revisions.Plan, keep []string, purgeErrs map[string]error, skipped []skippedRevision, times map[string]time.Time) []RevisionResult {
	decisions := map[string]Decision{}
	reasons := map[string]string{}
	if plan != nil {
		for _, rev := range plan.Deleted() {
			decisions[rev.Key()] = Deleted
		}
		for _, rev := range plan.Retained() {
			decisions[rev.Key()] = Kept
		}
		for _, rev := range plan.Safeguarded() {
			decisions[rev.Key()] = Protected
		}
		for _, rev := range revs {
			reasons[rev.Key()] = plan.Reason(rev.Key())
		}
	}

	sorted := slices.Clone(revs)
	for _, sk := range skipped {
		sorted = append(sorted, fileVersionRevision{version: sk.version, created: times[sk.version.Key]})
		decisions[sk.version.Key] = Protected
		reasons[sk.version.Key] = "protected: " + sk.reason
	}
	revisions.SortRevisionsCanonically(sorted)
	results := make([]RevisionResult, 0, len(sorted))
	for _, rev := range sorted {
		r := RevisionResult{
			Key:      rev.Key(),
			Time:     rev.CreationTime(),
			Size:     rev.Size(),
			Decision: decisions[rev.Key()],
			Reason:   reasons[rev.Key()],
			Err:      purgeErrs[rev.Key()],
		}
		if r.Decision == Deleted && slices.Contains(keep, rev.Key()) {
			r.Decision = Protected
			r.Reason = keptAfterRestore
		}
		results = append(results, r)
	}
	return results
}

// keptAfterRestoreKeys returns the keys of the versions of n that a restore keeps: the ones
// marked with node.KeepAfterRestoreAttr, as long as n still has the content the restore put in
// place, i.e. while its blob id is the value of the mark. The next upload or restore of n, or a
// change on disk with posix, gives n a new blob id: from then on such a revision is a version
// like any other, and the mark is removed, but in a dry run, which changes nothing.
//
// The mark is on disk rather than in the request for the cleanup, so that it holds also when
// the request is dropped or merged with another, for a cleanup by another instance of the
// driver in the same process, and for a cleanup on demand in a process of its own.
func keptAfterRestoreKeys(ctx context.Context, lu node.PathLookup, n *node.Node, versions []*provider.FileVersion, currentBlobID string, dryRun bool) []string {
	backend := lu.MetadataBackend()
	var keep []string
	for _, v := range versions {
		revision := node.NewBaseNode(n.SpaceID, v.Key, lu)
		restoredBlobID, err := backend.Get(ctx, revision, node.KeepAfterRestoreAttr)
		if err != nil {
			continue
		}
		if currentBlobID != "" && string(restoredBlobID) == currentBlobID {
			keep = append(keep, v.Key)
			continue
		}
		if dryRun {
			continue
		}
		if err := backend.Remove(ctx, revision, node.KeepAfterRestoreAttr); err != nil && !metadata.IsAttrUnset(err) {
			appctx.GetLogger(ctx).Warn().Err(err).Str("revision", v.Key).Msg("revision cleanup: could not remove the mark of a restore that the file no longer has")
		}
	}
	return keep
}

// skippedRevision is a revision the cleanup leaves alone, see notAVersion.
type skippedRevision struct {
	version *provider.FileVersion
	reason  string
}

// splitOffNotVersions splits the revisions of n that are not versions (see notAVersion) off
// the given ones.
func splitOffNotVersions(ctx context.Context, lu node.PathLookup, n *node.Node, all []*provider.FileVersion) ([]*provider.FileVersion, []skippedRevision) {
	versions := make([]*provider.FileVersion, 0, len(all))
	var skipped []skippedRevision
	for _, v := range all {
		if reason := notAVersion(ctx, lu.MetadataBackend(), node.NewBaseNode(n.SpaceID, v.Key, lu)); reason != "" {
			skipped = append(skipped, skippedRevision{version: v, reason: reason})
			continue
		}
		versions = append(versions, v)
	}
	return versions, skipped
}

// notAVersion explains why a revision is not a version, or returns "" if it is one.
//
// The upload coordinator (pkg/upload) keeps revisions for uploads in flight that are not
// versions:
//
//   - a revision that keeps the processing mark of an older upload, which a newer upload
//     superseded while the older one was still in postprocessing: the older upload writes
//     its content into it when it finishes (Decomposedfs.CommitUpload), and discards its
//     content if the revision is gone,
//   - a revision kept only to roll back an upload (node.KeptForRollback), which the upload
//     coordinator removes itself.
//
// The cleanup leaves them alone and doesn't count them. Usually the node itself is marked as
// processing while such revisions exist, and the cleanup skips it altogether, but not always:
// an upload that the coordinator finishes right away, e.g. of an empty file, clears the mark
// of the node while an older upload may still be in postprocessing.
func notAVersion(ctx context.Context, backend metadata.Backend, revision metadata.MetadataNode) string {
	if status, err := backend.Get(ctx, revision, prefixes.StatusPrefix); err == nil && strings.HasPrefix(string(status), node.ProcessingStatus) {
		return uploadInProgress
	}
	if node.KeptForRollback(ctx, backend, revision) {
		return keptForRollback
	}
	return ""
}

const (
	// the reasons for leaving a revision alone, see notAVersion
	uploadInProgress = "an upload into this revision is still being processed"
	keptForRollback  = "kept only to roll back an upload, it is not a version"
)

// revisionTimes returns the time each of the given revisions of n was created, by key: the time
// the cleanup orders them by and measures their age from.
//
// It is the time recorded in node.RevisionCreatedAttr, i.e. when the content the revision holds
// was replaced, by the clock of the server. The name of a revision is no good for this: it is
// the modification time of its content (see contentTime), which whoever writes the file can
// choose, e.g. with X-OC-Mtime. A revision dated just after an existing one would push that one
// out of its slot of "auto", and revisions dated years ahead would take the places of a number
// of revisions to keep, e.g. of "5, 30d", so that every new version would be deleted, for as
// long as they exist.
//
// Revisions created before the attribute was introduced don't have it. Their time is estimated
// as the time they were replaced: the content time of the next newer revision, or fileMTime,
// the modification time of the file, for the newest one, but never later than now. These come
// from the client as well, but they only apply to the revisions that exist already, and
// capping them at now keeps revisions dated in the future from taking the places of the ones
// created from now on.
//
// That the age of a revision is the time it was replaced, rather than the time its content was
// written, is what https://github.com/opencloud-eu/opencloud/issues/3735 is about.
func revisionTimes(ctx context.Context, lu node.PathLookup, n *node.Node, all []*provider.FileVersion, fileMTime, now time.Time) map[string]time.Time {
	backend := lu.MetadataBackend()
	byContent := slices.Clone(all)
	slices.SortStableFunc(byContent, compareContentTime)
	times := make(map[string]time.Time, len(all))
	for i, v := range byContent {
		if t, err := node.RevisionCreated(ctx, backend, node.NewBaseNode(n.SpaceID, v.Key, lu)); err == nil {
			times[v.Key] = t
			continue
		}
		estimate := fileMTime
		if i+1 < len(byContent) {
			estimate, _ = contentTime(byContent[i+1])
		}
		if estimate.After(now) {
			estimate = now
		}
		times[v.Key] = estimate
	}
	return times
}

// contentTime determines when the content of a revision was last modified from the timestamp
// in its key, '<nodeid>.REV.<RFC3339Nano>' with an optional '.<n>' suffix for colliding
// revisions, which is returned as well (0 without one). The mtime of the revision file is used
// as a fallback.
//
// Revisions are named after the mtime of the content they preserve (see
// Decomposedfs.PrepareUpload, the posix assimilation and RestoreRevision). For changes on disk
// with posix, the name is the time the content was copied to '.CURRENT', when that change was
// picked up, normally right after the content was written (see the posix assimilation).
//
// The key is preferred over the file mtime because the file mtime is not reliable:
// the upload code only sets it to the mtime of the old content (os.Chtimes in
// Decomposedfs.PrepareUpload) after CreateRevision has returned, and the
// FileVersion.Mtime from ListRevisions has only a precision of seconds.
func contentTime(v *provider.FileVersion) (time.Time, int) {
	if _, ts, ok := strings.Cut(v.Key, node.RevisionIDDelimiter); ok {
		if t, err := time.Parse(time.RFC3339Nano, ts); err == nil {
			return t, 0
		}
		if i := strings.LastIndex(ts, "."); i > 0 {
			if t, err := time.Parse(time.RFC3339Nano, ts[:i]); err == nil {
				suffix, _ := strconv.Atoi(ts[i+1:])
				return t, suffix
			}
		}
	}
	return time.Unix(int64(v.Mtime), 0), 0
}

// compareContentTime orders revisions by their content time, oldest first. Of colliding
// revisions, the one with the key without a suffix is the newest: CreateRevision renames the
// existing one to the next free suffix, and gives the new one the key without a suffix.
func compareContentTime(a, b *provider.FileVersion) int {
	ta, sa := contentTime(a)
	tb, sb := contentTime(b)
	if c := ta.Compare(tb); c != 0 {
		return c
	}
	if sa == 0 {
		sa = math.MaxInt
	}
	if sb == 0 {
		sb = math.MaxInt
	}
	return cmp.Compare(sa, sb)
}

// fileVersionRevision adapts a CS3 FileVersion, as returned by
// node.Tree.ListRevisions, to the revisions.Revision interface the
// cleanup strategies work on.
type fileVersionRevision struct {
	version *provider.FileVersion
	created time.Time // see revisionTimes
}

var _ revisions.Revision = fileVersionRevision{}

func (f fileVersionRevision) Key() string             { return f.version.Key }
func (f fileVersionRevision) CreationTime() time.Time { return f.created }
func (f fileVersionRevision) Size() uint64            { return f.version.Size }
