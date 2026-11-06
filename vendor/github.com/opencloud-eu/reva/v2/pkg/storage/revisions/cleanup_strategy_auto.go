// Copyright 2026 OpenCloud GmbH <mail@opencloud.eu>
// SPDX-License-Identifier: Apache-2.0

package revisions

import (
	"context"
	"fmt"
	"time"

	"github.com/rs/zerolog"
)

const (
	autoStrategy = "auto"
)

type autoRevisionCleanupStrategy struct {
}

var _ RevisionCleanupStrategy = &autoRevisionCleanupStrategy{}

func newAutoRevisionCleanupStrategy() (RevisionCleanupStrategy, error) {
	return &autoRevisionCleanupStrategy{}, nil
}

func newAutoRevisionCleanupStrategyFromString(config string) (RevisionCleanupStrategy, error) {
	if config != autoStrategy {
		return nil, nil // we don't support this
	}
	return newAutoRevisionCleanupStrategy()
}

// bucketPolicy defines a time window and the resolution slot within that window.
type bucketPolicy struct {
	name        string        // value only matters for debugging, but must be unique
	description string        // the time window in words, for the reasons in the Plan
	ageLimit    time.Duration // Max age relative to 'now' to belong to this bucket
	interval    time.Duration // Slot duration (<=0 means keep all)
	maxKeep     int           // Max revisions to keep per slot (<=0 means keep all, >0 caps per slot)
}

// Standard GFS (Grandfather-Father-Son) style retention tiers.
var defaultPolicy = []bucketPolicy{
	{
		name:        "LastMinute",
		description: "last minute",
		ageLimit:    1 * time.Minute,  // last minute
		maxKeep:     1,                // one version
		interval:    10 * time.Second, // is kept every 10 seconds
	},
	{
		name:        "LastHour",
		description: "last hour",
		ageLimit:    1 * time.Hour,   // last hour
		maxKeep:     1,               // one version
		interval:    1 * time.Minute, // is kept every minute
	},
	{
		name:        "Last24H",
		description: "last 24 hours",
		ageLimit:    24 * time.Hour, // last 24 hours
		maxKeep:     1,              // one version
		interval:    1 * time.Hour,  // is kept every hour
	},
	{
		name:        "Last30D",
		description: "last 30 days",
		ageLimit:    30 * 24 * time.Hour, // last 30 days
		maxKeep:     1,                   // one version
		interval:    24 * time.Hour,      // is kept every day
	},
	{
		name:        "Older30D",
		description: "older than 30 days",
		ageLimit:    0,                  // older than 30 days (0 AgeLimit denotes the catch-all "older" tier)
		maxKeep:     1,                  // one version
		interval:    7 * 24 * time.Hour, // s kept every week
	},
}

// evaluateRetention thins revisions across dynamic time tiers based on interval slots.
// Every decision in the returned Plan comes with a reason that names the tier and, for tiers
// with slots, the slot, so that the outcome of this not quite trivial bucket system can be
// explained (e.g. in the debug log or in verbose output).
func evaluateRetention(revisions []Revision, policies []bucketPolicy, now time.Time) Plan {
	// First, iterate over all revisions to group/sort them into discrete,
	// mutually exclusive buckets (one revision can only ever be in one bucket).
	// One bucket (of revisions) corresponds to one bucketPolicy.
	bucket := make(map[string][]Revision)
	for _, rev := range revisions {
		t := rev.LastModificationTime()
		age := now.Sub(t)

		matched := false
		for _, policy := range policies {
			if policy.ageLimit > 0 && age <= policy.ageLimit {
				bucket[policy.name] = append(bucket[policy.name], rev)
				matched = true
				break
			}
		}

		// if we couldn't find a fitting bucket for this rev yet,
		// then put it into the last bucket, which is assumed to be the
		// "catch-all" one for "older than":
		if !matched && len(policies) > 0 {
			lastPolicy := policies[len(policies)-1]
			bucket[lastPolicy.name] = append(bucket[lastPolicy.name], rev)
		}
	}

	plan := NewSimplePlan()

	// Thin each bucket according to its interval resolution and MaxKeep limit
	for _, policy := range policies {
		items := bucket[policy.name]
		if len(items) == 0 {
			continue // let's stop right here, nothing to do
		}
		kept := fmt.Sprintf("kept by %q (%s): ", autoStrategy, policy.description)
		deleted := fmt.Sprintf("deleted by %q (%s): ", autoStrategy, policy.description)

		if policy.interval <= 0 {
			// if interval is not set (<=0), then we just keep maxKeep items across the whole bucket:
			// keep the first maxKeep, and delete everything else (keeping in mind that revisions
			// are sorted in newest-first order)
			switch {
			case policy.maxKeep <= 0:
				// if maxKeep is not set (<=0), then we retain *all* of the revisions in the bucket
				plan.Retain(kept+"all revisions are kept", items...)
			case policy.maxKeep < len(items):
				// we keep less revisions than we have in total in this bucket
				// => we need to do something: since they are sorted in newest-first order,
				// we can simply split the revisions slice in two:
				retain := items[:policy.maxKeep] // the first ones up to maxKeep are retained
				delete := items[policy.maxKeep:] // everything after maxKeep is deleted
				plan.Retain(kept+fmt.Sprintf("one of the %d newest", policy.maxKeep), retain...)
				plan.Delete(deleted+fmt.Sprintf("beyond the %d newest", policy.maxKeep), delete...)
			default:
				// maxKeep is larger than the number of items => retain everything
				plan.Retain(kept+fmt.Sprintf("one of the %d newest", policy.maxKeep), items...)
			}
		} else {
			// interval is set for this bucket, in which case we need to apply the slightly
			// more complex policy of putting revisions within this bucket into slots of
			// duration of ${interval}, and then count the number of revisions in each slot
			// to determine which to delete (or not)

			// we will use unix nano timestamps as the keys for the map of slots,
			// just as a performance improvement to avoid using time.Duration objects
			slotCounts := make(map[int64]int)

			for _, rev := range items {
				// we take the revision's mtime, but truncate it to the interval for this
				// bucket (e.g. 1m or 1h), in order to count how many revisions we have
				// in each slot within the bucket; note that we also transform the
				// time.Duration into an int64 to use that as the key in the slot map
				slotKey := rev.LastModificationTime().Truncate(policy.interval).UnixNano()
				slot := fmt.Sprintf("its slot of %s starting %s", humanDuration(policy.interval), formatTime(time.Unix(0, slotKey).UTC()))

				count := slotCounts[slotKey]
				switch {
				case policy.maxKeep <= 0:
					// if maxKeep is not set (<=0), then we keep them all anyways
					plan.Retain(kept+"all revisions are kept", rev)
				case count < policy.maxKeep:
					// if the current count of revisions in this slot is smaller than maxKeep,
					// then we also keep this revision
					slotCounts[slotKey] = count + 1
					if policy.maxKeep == 1 {
						plan.Retain(kept+"the newest in "+slot, rev)
					} else {
						plan.Retain(kept+fmt.Sprintf("one of the %d newest in %s", policy.maxKeep, slot), rev)
					}
				default:
					// if maxKeep is set (>0) and the count of revisions within its assigned
					// slot is higher than maxKeep, then the revision must be deleted
					if policy.maxKeep == 1 {
						plan.Delete(deleted+slot+" already keeps a newer revision", rev)
					} else {
						plan.Delete(deleted+fmt.Sprintf("%s already keeps %d newer revisions", slot, policy.maxKeep), rev)
					}
				}
			}
		}
	}

	return plan
}

func (s *autoRevisionCleanupStrategy) Cleanup(now time.Time, revs []Revision, ctx context.Context, logger *zerolog.Logger) (Plan, error) {
	// no need to sort revisions, they are guaranteed to be sorted from newest to oldest
	return evaluateRetention(revs, defaultPolicy, now), nil
}
