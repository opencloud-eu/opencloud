// Copyright 2026 OpenCloud GmbH <mail@opencloud.eu>
// SPDX-License-Identifier: Apache-2.0

package revisions

import (
	"context"
	"time"

	"github.com/rs/zerolog"
)

type Revision interface {
	Key() string
	LastModificationTime() time.Time
	Size() uint64
}

// Plan is the outcome of a RevisionCleanupStrategy: which revisions to delete, which to
// retain (keep, or hand on to the next strategy of a chain) and which to safeguard (keep
// for sure).
//
// Every decision comes with a human readable reason, e.g. `deleted by "30d": older than
// 2026-09-14T10:00:00Z`, so that it can be explained why a revision was deleted or kept,
// in the debug log of the cleanup or in verbose output. The reasons name the rule in the
// syntax of the configuration and the concrete boundary it was evaluated with.
type Plan interface {
	Delete(reason string, revs ...Revision)
	Retain(reason string, revs ...Revision)
	Safeguard(reason string, revs ...Revision)
	Deleted() []Revision
	Retained() []Revision
	Safeguarded() []Revision
	// Reason returns the reason for the decision about the revision with the given key,
	// "" if the plan has no decision about it.
	Reason(key string) string
}

type RevisionCleanupStrategy interface {
	Cleanup(now time.Time, revs []Revision, ctx context.Context, logger *zerolog.Logger) (Plan, error)
}
