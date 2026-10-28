// Copyright 2026 OpenCloud GmbH <mail@opencloud.eu>
// SPDX-License-Identifier: Apache-2.0

package revisions

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// valuePattern matches a single value X: a number of versions, or a duration with a unit
// suffix, where durations in days or weeks with an uppercase unit (D, W) are aligned to midnight.
const valuePattern = `\d+[wdhmsDW]?`

var (
	configPatternAutoOnly               = regexp.MustCompile(`^(auto)$`)
	configPatternDeleteOnly             = regexp.MustCompile(`^(` + valuePattern + `)$`)
	configPatternSafeguardThenAuto      = regexp.MustCompile(`^(` + valuePattern + `)\s*,\s*(auto)$`)
	configPatternAutoThenDelete         = regexp.MustCompile(`^(auto)\s*,\s*(` + valuePattern + `)$`)
	configPatternSafeguardThenDelete    = regexp.MustCompile(`^(` + valuePattern + `)\s*,\s*(` + valuePattern + `)$`)
	configPatternSafeguardAutoAndDelete = regexp.MustCompile(`^(` + valuePattern + `)\s*,\s*(auto)\s*,\s*(` + valuePattern + `)$`)

	configPatternCount = regexp.MustCompile(`^\d+$`)
)

// This is the entry point for instantiating a strategy that may be composed of multiple ones.
//
// The config parameter is a string that denotes which strategy/ies to use, as a comma separated
// list of specifications.
//
// Each X is either a plain number, which is a number of revisions, or a number with a unit
// suffix (w for weeks, d for days, h for hours, m for minutes, s for seconds), which is a
// duration. A number of revisions always refers to the newest revisions.
//
// A duration in days or weeks with an uppercase unit (D, W) is aligned to midnight in the server's
// local time zone: "1d" means the last 24 hours, while "1D" means since yesterday 00:00, so
// today and yesterday; "0D" means since today 00:00, and "1W" since 00:00 seven days ago.
// Depending on the time of day, "ND" thus spans between N and N+1 days.
//
// Supported forms:
//   - "auto": Default, default clean-up pattern applies
//   - "X": Delete all versions beyond X: keep the X newest versions, or delete versions
//     that are older than X, without any further thinning
//     "0" deletes all versions: unlike disabling versioning, which only stops new versions
//     from being created, this also removes the versions that have accumulated so far.
//   - "X, auto": Keeps versions within X, apply default clean-up pattern to all other versions
//   - "auto, X": Delete all versions beyond X, apply the default clean-up pattern to the others
//   - "X1, X2": Keep versions within X1 and delete versions beyond X2. X1 and X2 must be of
//     different kinds, i.e. one number of versions and one duration, e.g. "5, 30d" (delete
//     versions older than 30 days, but always keep the 5 newest) or "7d, 10" (keep at most
//     10 versions, but never delete versions younger than 7 days).
//     Two values of the same kind are rejected, as one of them would have no effect:
//     "5, 10" is the same as "10", and "7d, 30d" is the same as "30d".
//   - "X1, auto, X2": Keep versions within X1, delete versions beyond X2, and apply the
//     default clean-up pattern to the versions in between, e.g. "7d, auto, 90d".
//     If X1 and X2 are of the same kind, X1 must be smaller than X2, otherwise the default
//     clean-up pattern would never apply.
//
// "disabled" and an empty specification are not strategies: the callers handle them
// without parsing a strategy (see revisioncleanup.New), as there is nothing to plan.
//
// A deletion by number of versions is absolute: in "7d, 10" the versions younger than 7
// days count towards the 10 versions to keep.
//
// In "auto, N" and "X1, auto, N", the number N applies before "auto", not after: "auto, 10"
// first deletes all but the 10 newest versions, and then thins out those 10, so it usually
// keeps fewer than 10, and possibly fewer than "auto" alone would. If the 10 newest versions
// were all saved within the same minute yesterday, "auto, 10" keeps a single version, while
// "auto" would keep one of them plus older ones. This is easily misread as "thin out first,
// then keep at most 10", which is not what it means.
//
// The age of a version, which durations are compared with, is Revision.LastModificationTime:
// the time its content was written, i.e. the modification time the file had before the change
// that turned it into a version, not the time of that change (see revisioncleanup.revisionTime). A version created today from content that was unchanged for
// two months is two months old.
func NewRevisionCleanupStrategy(config string) (RevisionCleanupStrategy, error) {
	config = strings.TrimSpace(config)

	// "auto": Default, default clean-up pattern applies
	if m := configPatternAutoOnly.FindStringSubmatch(config); len(m) > 0 {
		auto := m[1]
		return newAutoRevisionCleanupStrategyFromString(auto)
	}

	// "X": Delete all versions beyond X, no thinning
	if m := configPatternDeleteOnly.FindStringSubmatch(config); len(m) > 0 {
		return parseDelete(m[1])
	}

	// "X, auto": Keeps versions within X, apply default clean-up pattern to all other versions
	if m := configPatternSafeguardThenAuto.FindStringSubmatch(config); len(m) > 0 {
		return parseChain(
			func() (RevisionCleanupStrategy, error) { return parseKeep(m[1]) },
			func() (RevisionCleanupStrategy, error) { return parseAuto(m[2]) },
		)
	}

	// "auto, X": Delete all versions beyond X, apply the default clean-up pattern to the others.
	// The deletion comes first in the chain, so a number of versions is applied before the
	// thinning, see the doc comment above.
	if m := configPatternAutoThenDelete.FindStringSubmatch(config); len(m) > 0 {
		return parseChain(
			func() (RevisionCleanupStrategy, error) { return parseDelete(m[2]) },
			func() (RevisionCleanupStrategy, error) { return parseAuto(m[1]) },
		)
	}

	// "X1, X2": Keep versions within X1 and delete versions beyond X2
	if m := configPatternSafeguardThenDelete.FindStringSubmatch(config); len(m) > 0 {
		keep, delete := m[1], m[2]
		if sameKind(keep, delete) {
			equivalent, err := sameKindEquivalent(keep, delete)
			if err != nil {
				return nil, err
			}
			return nil, fmt.Errorf("unsupported revision cleanup strategy specification %q: "+
				"%q and %q are both %s, so one of them has no effect and it is the same as %s; "+
				"combine a number of versions with a duration instead, e.g. \"5, 30d\" or \"7d, 10\"",
				config, keep, delete, kindName(keep), equivalent)
		}
		return parseChain(
			func() (RevisionCleanupStrategy, error) { return parseKeep(keep) },
			func() (RevisionCleanupStrategy, error) { return parseDelete(delete) },
		)
	}

	// "X1, auto, X2": Keep versions within X1, delete versions beyond X2, thin out the ones in between.
	// As in "auto, X", a number of versions X2 is applied before the thinning.
	if m := configPatternSafeguardAutoAndDelete.FindStringSubmatch(config); len(m) > 0 {
		keep, auto, delete := m[1], m[2], m[3]
		if sameKind(keep, delete) {
			smaller, err := alwaysSmaller(keep, delete)
			if err != nil {
				return nil, err
			}
			if !smaller {
				return nil, fmt.Errorf("unsupported revision cleanup strategy specification %q: "+
					"%q must be smaller than %q at any time of day, otherwise the default clean-up "+
					"pattern does not apply in between", config, keep, delete)
			}
		}
		return parseChain(
			func() (RevisionCleanupStrategy, error) { return parseKeep(keep) },
			func() (RevisionCleanupStrategy, error) { return parseDelete(delete) },
			func() (RevisionCleanupStrategy, error) { return parseAuto(auto) },
		)
	}

	return nil, fmt.Errorf("unsupported revision cleanup strategy specification %q", config)
}

// parseChain parses the given strategies and chains them in that order.
func parseChain(parsers ...func() (RevisionCleanupStrategy, error)) (RevisionCleanupStrategy, error) {
	chain := make([]RevisionCleanupStrategy, len(parsers))
	for i, parse := range parsers {
		strategy, err := parse()
		if err != nil {
			return nil, err
		}
		chain[i] = strategy
	}
	return newChainRevisionCleanupStrategy(chain)
}

// sameKind tells whether both values are numbers of versions, or both are durations.
func sameKind(a, b string) bool {
	return configPatternCount.MatchString(a) == configPatternCount.MatchString(b)
}

func kindName(value string) string {
	if configPatternCount.MatchString(value) {
		return "numbers of versions"
	}
	return "durations"
}

// valueRange returns the smallest and the largest extent a value can have, as a number of
// versions for counts and in nanoseconds for durations. They are the same, except for durations
// aligned to midnight: "ND" covers between N days and just under N+1 days, depending on the
// time of day.
//
// A day counts as 24 hours here, although the days on which daylight saving time begins or ends
// have 23 or 25 hours. The comparisons based on it can therefore be off by up to an hour on those
// days, e.g. "1D, auto, 2d" is accepted, although "1D" can reach slightly beyond "2d" in the
// night after the clocks were set back. That is deliberate: rejecting such strategies for the
// sake of one hour a year would be worse, and during that hour, the versions in question are
// only kept, never deleted, as protecting them comes first in the chain.
func valueRange(value string) (int64, int64, error) {
	if configPatternCount.MatchString(value) {
		n, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return 0, 0, fmt.Errorf("invalid number of versions %q: %w", value, err)
		}
		return n, n, nil
	}
	truncated := strings.HasSuffix(value, "D") || strings.HasSuffix(value, "W")
	d, err := parseDuration(strings.ToLower(value))
	if err != nil {
		return 0, 0, err
	}
	if truncated {
		return int64(d), int64(d + 24*time.Hour - 1), nil
	}
	return int64(d), int64(d), nil
}

// alwaysSmaller tells whether a is smaller than b at any time of day (but see valueRange about
// daylight saving time). Both must be of the same kind. Values that only reach each other at a
// single instant count as smaller, e.g. "1d" and "1D", which are equal only at midnight; values
// that are always equal, e.g. "1w" and "7d", don't.
func alwaysSmaller(a, b string) (bool, error) {
	aMin, aMax, err := valueRange(a)
	if err != nil {
		return false, err
	}
	bMin, bMax, err := valueRange(b)
	if err != nil {
		return false, err
	}
	return aMax < bMin || (aMax == bMin && (aMin < aMax || bMin < bMax)), nil
}

// sameKindEquivalent describes what an "X1, X2" pair of values of the same kind is equivalent to:
// the larger of the two, as the smaller one has no effect. If the order depends on the time of
// day, as with "1D, 30h", it is the larger of the two at any given moment.
func sameKindEquivalent(a, b string) (string, error) {
	aMin, aMax, err := valueRange(a)
	if err != nil {
		return "", err
	}
	bMin, bMax, err := valueRange(b)
	if err != nil {
		return "", err
	}
	// a value that is never larger than the other one has no effect: "1d" is never larger than
	// "1D", so "1d, 1D" and "1D, 1d" are both the same as "1D"
	switch {
	case aMax <= bMin:
		return fmt.Sprintf("%q", b), nil
	case bMax <= aMin, aMin == bMin && aMax == bMax:
		return fmt.Sprintf("%q", a), nil
	default:
		return fmt.Sprintf("%q or %q, depending on the time of day", a, b), nil
	}
}

func parseKeep(retain string) (RevisionCleanupStrategy, error) {
	strategy, err := newDurationRetentionRevisionCleanupStrategyFromString(retain)
	if err != nil {
		return nil, err
	}
	if strategy != nil {
		return strategy, nil
	}
	strategy, err = newCountRetentionRevisionCleanupStrategyFromString(retain)
	if err != nil {
		return nil, err
	}
	if strategy != nil {
		return strategy, nil
	}
	// doesn't match either
	return nil, fmt.Errorf("unsupported revision safeguarding specification %q", retain)
}

func parseDelete(delete string) (RevisionCleanupStrategy, error) {
	strategy, err := newDurationDeletionRevisionCleanupStrategyFromString(delete)
	if err != nil {
		return nil, err
	}
	if strategy != nil {
		return strategy, nil
	}
	strategy, err = newCountDeletionRevisionCleanupStrategyFromString(delete)
	if err != nil {
		return nil, err
	}
	if strategy != nil {
		return strategy, nil
	}
	// doesn't match either
	return nil, fmt.Errorf("unsupported revision deletion specification %q", delete)
}

func parseAuto(auto string) (RevisionCleanupStrategy, error) {
	if strategy, err := newAutoRevisionCleanupStrategyFromString(auto); err != nil {
		return nil, err
	} else if strategy != nil {
		return strategy, nil
	} else {
		return nil, fmt.Errorf("unsupported revision 'auto' specification %q", auto)
	}
}
