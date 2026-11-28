// Copyright 2026 OpenCloud GmbH <mail@opencloud.eu>
// SPDX-License-Identifier: Apache-2.0

package revisioncleanup

import "time"

// Decision is what the cleanup decided about a revision.
type Decision string

const (
	// Deleted revisions are deleted by the strategy (or would be, in a dry run).
	Deleted Decision = "deleted"
	// Kept revisions are not deleted by the strategy, but not protected either: a later
	// cleanup may delete them as they get older.
	Kept Decision = "kept"
	// Protected revisions are safeguarded by the strategy, or were requested to be kept by
	// the operation that triggered the cleanup (e.g. the state from before a restore).
	Protected Decision = "protected"
)

// RevisionResult is the outcome of a cleanup for one revision.
type RevisionResult struct {
	Key string
	// Time is the time the revision was created, which the cleanup orders the revisions by and
	// takes their age from, see revisionTimes.
	Time time.Time
	Size uint64
	// Decision and Reason explain what the strategy decided and why, see revisions.Plan.
	Decision Decision
	Reason   string
	// Err is set when deleting a revision failed.
	Err error
}

// Result is the outcome of a cleanup for one node, with an entry for each of its revisions,
// newest first. It is used to report on a cleanup, e.g. in the verbose output of a command.
type Result struct {
	SpaceID string
	NodeID  string
	// Path is the path of the file (or folder) within its space, e.g. "/Documents/report.odt".
	// Only on-demand runs (Run) set it, from the folders they walk through; it is empty when
	// the path is not known, e.g. for a space that could not be cleaned up.
	Path string
	// DryRun is set when nothing was deleted, only decided.
	DryRun bool
	// Strategy is the strategy the node was cleaned up with, and StrategySource the setting of
	// the Policy it came from. Both are empty when the cleanup failed before the strategy was
	// chosen, or when it didn't run because of the node, e.g. as it no longer exists.
	Strategy       string
	StrategySource StrategySource
	// Skipped explains why the node was not cleaned up, e.g. because an upload to it is still
	// being processed. Revisions is empty then.
	Skipped string
	// Err is set when the cleanup failed as a whole, e.g. when the revisions could not be
	// listed. Errors of single revisions are in Revisions.
	Err       error
	Revisions []RevisionResult
}

// DeletedCount returns the number and the total size of the revisions that were deleted
// (or would be, in a dry run), not counting the ones whose deletion failed.
func (r Result) DeletedCount() (count int, bytes uint64) {
	for _, rev := range r.Revisions {
		if rev.Decision == Deleted && rev.Err == nil {
			count++
			bytes += rev.Size
		}
	}
	return count, bytes
}
