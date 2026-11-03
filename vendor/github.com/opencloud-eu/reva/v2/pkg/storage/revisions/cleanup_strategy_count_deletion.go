// Copyright 2026 OpenCloud GmbH <mail@opencloud.eu>
// SPDX-License-Identifier: Apache-2.0

package revisions

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/rs/zerolog"
)

// A RevisionCleanupStrategy that keeps at most a given number of revisions, the newest
// ones, and deletes all older revisions.
//
// Within a chain the limit is absolute: revisions safeguarded by earlier strategies of
// the chain count towards it (see chainAwareStrategy).
type countDeletionRevisionCleanupStrategy struct {
	maxVersionsPerFile int
}

var _ RevisionCleanupStrategy = &countDeletionRevisionCleanupStrategy{}

var _ chainAwareStrategy = &countDeletionRevisionCleanupStrategy{}

func newCountDeletionRevisionCleanupStrategy(maxVersionsPerFile int) (RevisionCleanupStrategy, error) {
	return &countDeletionRevisionCleanupStrategy{
		maxVersionsPerFile: maxVersionsPerFile,
	}, nil
}

func newCountDeletionRevisionCleanupStrategyFromString(config string) (RevisionCleanupStrategy, error) {
	if match, maxVersionsPerFile, err := countFromConfig(config); !match {
		return nil, nil
	} else if err != nil {
		return nil, err
	} else {
		return newCountDeletionRevisionCleanupStrategy(maxVersionsPerFile)
	}
}

func (s *countDeletionRevisionCleanupStrategy) Cleanup(now time.Time, revs []Revision, ctx context.Context, logger *zerolog.Logger) (Plan, error) {
	return s.cleanupInChain(now, revs, 0, ctx, logger)
}

func (s *countDeletionRevisionCleanupStrategy) cleanupInChain(now time.Time, revs []Revision, safeguarded int, ctx context.Context, logger *zerolog.Logger) (Plan, error) {
	plan := NewSimplePlan()
	// the safeguarded revisions already use up part of the limit, and if there are more
	// of them than the limit allows, all remaining revisions have to go
	keep := max(s.maxVersionsPerFile-safeguarded, 0)
	spec := strconv.Itoa(s.maxVersionsPerFile)
	counted := ""
	if safeguarded > 0 {
		counted = fmt.Sprintf(", counting the %d protected ones", safeguarded)
	}
	retained := fmt.Sprintf("kept by %q: one of the %d newest%s", spec, s.maxVersionsPerFile, counted)
	if len(revs) <= keep {
		plan.Retain(retained, revs...)
		return plan, nil
	}
	// no need to sort revisions, they are guaranteed to be sorted from newest to oldest
	plan.Delete(fmt.Sprintf("deleted by %q: beyond the %d newest%s", spec, s.maxVersionsPerFile, counted), revs[keep:]...)
	plan.Retain(retained, revs[:keep]...)
	return plan, nil
}
