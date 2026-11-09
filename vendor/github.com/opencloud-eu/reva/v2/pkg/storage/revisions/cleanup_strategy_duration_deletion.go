// Copyright 2026 OpenCloud GmbH <mail@opencloud.eu>
// SPDX-License-Identifier: Apache-2.0

package revisions

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/rs/zerolog"
)

// A RevisionCleanupStrategy that deletes files that are older than a given duration.
//
// With truncateToMidnight ("ND" or "NW" instead of "Nd" or "Nw"), the cutoff is aligned to
// local midnight instead of being exactly the duration before now, see computeCutoffFromRetention.
type durationDeletionRevisionCleanupStrategy struct {
	retention          time.Duration
	truncateToMidnight bool
	// spec is the duration as configured, e.g. "7d", for the reasons in the Plan; when empty,
	// it is derived from retention, which would turn "7d" into "1w"
	spec string
}

var _ RevisionCleanupStrategy = &durationDeletionRevisionCleanupStrategy{}

func newDurationDeletionRevisionCleanupStrategy(retention time.Duration, truncateToMidnight bool) (RevisionCleanupStrategy, error) {
	return &durationDeletionRevisionCleanupStrategy{
		retention:          retention,
		truncateToMidnight: truncateToMidnight,
	}, nil
}

func newDurationDeletionRevisionCleanupStrategyFromString(config string) (RevisionCleanupStrategy, error) {
	if matches, duration, truncateToMidnight, err := durationFromConfig(config); !matches {
		return nil, nil
	} else if err != nil {
		return nil, err
	} else {
		strategy, err := newDurationDeletionRevisionCleanupStrategy(duration, truncateToMidnight)
		if err != nil {
			return nil, err
		}
		strategy.(*durationDeletionRevisionCleanupStrategy).spec = strings.TrimSpace(config)
		return strategy, nil
	}
}

func (s *durationDeletionRevisionCleanupStrategy) Cleanup(now time.Time, revs []Revision, ctx context.Context, logger *zerolog.Logger) (Plan, error) {
	cutoff := computeCutoffFromRetention(now, -s.retention, s.truncateToMidnight)
	spec := s.spec
	if spec == "" {
		spec = formatDurationSpec(s.retention, s.truncateToMidnight)
	}
	deleted := fmt.Sprintf("deleted by %q: older than %s", spec, formatTime(cutoff))
	retained := fmt.Sprintf("kept by %q: not older than %s", spec, formatTime(cutoff))
	plan := NewSimplePlan()
	for _, rev := range revs {
		if rev.LastModificationTime().Before(cutoff) {
			plan.Delete(deleted, rev)
		} else {
			plan.Retain(retained, rev)
		}
	}
	return plan, nil
}
