// Copyright 2026 OpenCloud GmbH <mail@opencloud.eu>
// SPDX-License-Identifier: Apache-2.0

package revisions

import (
	"context"
	"slices"
	"time"

	"github.com/rs/zerolog"
)

// A RevisionCleanupStrategy that applies a sequence of strategies one after the other.
//
// Every strategy in the chain only gets to see the revisions that the previous ones
// retained: revisions that were deleted or safeguarded by an earlier strategy are final.
type chainRevisionCleanupStrategy struct {
	chain []RevisionCleanupStrategy
}

// chainAwareStrategy is implemented by strategies whose decision depends on how many
// revisions the earlier strategies of a chain have already safeguarded.
//
// This is needed for count based deletion: in "7d, 10", "10" means "keep at most 10
// revisions in total", not "keep 10 more on top of the ones younger than 7 days". As
// the chain only passes on the remaining revisions, the strategy needs to be told how
// many revisions it has to count in addition. Safeguarded revisions are always newer
// than the remaining ones (both safeguarding strategies protect the newest revisions),
// so they are exactly the ones counting towards the limit first.
type chainAwareStrategy interface {
	cleanupInChain(now time.Time, revs []Revision, safeguarded int, ctx context.Context, logger *zerolog.Logger) (Plan, error)
}

var _ RevisionCleanupStrategy = &chainRevisionCleanupStrategy{}

func newChainRevisionCleanupStrategy(chain []RevisionCleanupStrategy) (RevisionCleanupStrategy, error) {
	return &chainRevisionCleanupStrategy{
		chain: chain,
	}, nil
}

func (s *chainRevisionCleanupStrategy) Cleanup(now time.Time, revs []Revision, ctx context.Context, logger *zerolog.Logger) (Plan, error) {
	masterPlan := NewSimplePlan()
	remaining := slices.Clone(revs)
	var last Plan
	for _, strategy := range s.chain {
		var plan Plan
		var err error
		if aware, ok := strategy.(chainAwareStrategy); ok {
			plan, err = aware.cleanupInChain(now, remaining, len(masterPlan.Safeguarded()), ctx, logger)
		} else {
			plan, err = strategy.Cleanup(now, remaining, ctx, logger)
		}
		if err != nil {
			return nil, err
		}

		// the decisions of this step are final, they keep the reasons this step gave
		for _, rev := range plan.Deleted() {
			masterPlan.Delete(plan.Reason(rev.Key()), rev)
		}
		for _, rev := range plan.Safeguarded() {
			masterPlan.Safeguard(plan.Reason(rev.Key()), rev)
		}
		remaining = slices.Clone(plan.Retained())
		last = plan
	}
	// the revisions no step decided about are kept, with the reason the last step gave
	for _, rev := range remaining {
		reason := "kept: not decided by any rule"
		if last != nil && last.Reason(rev.Key()) != "" {
			reason = last.Reason(rev.Key())
		}
		masterPlan.Retain(reason, rev)
	}
	return masterPlan, nil
}
