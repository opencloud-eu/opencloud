// Copyright 2026 OpenCloud GmbH <mail@opencloud.eu>
// SPDX-License-Identifier: Apache-2.0

package revisions

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"math"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/rs/zerolog"
)

// Compares two Revisions, newest first.
//
// Revisions with the same time are told apart by their keys, so that the order, and with it
// which of them a strategy keeps, never depends on the order they were listed in. When the
// posix driver creates a revision whose key is already taken, it renames the existing
// revision to "<key>.<N>", with N one higher than any existing suffix, and creates the new
// one under the plain key. The plain key is therefore the newest, followed by the suffixed
// ones from the highest N down.
func CompareNewestFirst[T Revision](a, b T) int {
	if c := b.LastModificationTime().Compare(a.LastModificationTime()); c != 0 {
		return c
	}
	if c := cmp.Compare(collisionRank(b.Key()), collisionRank(a.Key())); c != 0 {
		return c
	}
	return strings.Compare(a.Key(), b.Key())
}

// collisionRank ranks revisions with the same time, see CompareNewestFirst: the higher, the
// newer. A key ending in ".<N>" ranks N, any other key ranks above all of those. A timestamp
// ending in fractions of a second ("...05.123Z") is not mistaken for a suffix, as it ends in
// "Z" or a time zone offset.
func collisionRank(key string) int {
	i := strings.LastIndexByte(key, '.')
	if i < 0 {
		return math.MaxInt
	}
	n, err := strconv.Atoi(key[i+1:])
	if err != nil || n < 0 {
		return math.MaxInt
	}
	return n
}

// Sorts a slice of Revisions canonically: newest first.
func SortRevisionsCanonically[T Revision](revisions []T) {
	slices.SortFunc(revisions, CompareNewestFirst)
}

// CleanupStrategyFunc is an adapter to use an ordinary function as a RevisionCleanupStrategy,
// like http.HandlerFunc: CleanupStrategyFunc(f) is a RevisionCleanupStrategy that calls f.
type CleanupStrategyFunc func(now time.Time, revs []Revision, ctx context.Context, logger *zerolog.Logger) (Plan, error)

var _ RevisionCleanupStrategy = CleanupStrategyFunc(nil)

// Cleanup calls f(now, revs, ctx, logger).
func (f CleanupStrategyFunc) Cleanup(now time.Time, revs []Revision, ctx context.Context, logger *zerolog.Logger) (Plan, error) {
	return f(now, revs, ctx, logger)
}

// Takes a RevisionCleanupStrategy and acts as a glue to piece everything together,
// except the deletion of Revisions itself, that it delegates to the cleanup function
// that is passed as a parameter, together with the reason the strategy gave for deleting
// the revision (see Plan).
//
// It returns the Plan of the strategy, so that callers can report on all revisions, also
// the ones that were kept or protected, together with the errors of the cleanup function.
// The Plan is nil if the strategy failed.
//
// Note that this implementation is assuming that it is used in a single-threaded
// fashion as it does not perform any locking of anything.
func ExecuteCleanupStrategy(now time.Time, strategy RevisionCleanupStrategy, revs []Revision, ctx context.Context, logger *zerolog.Logger,
	cleanup func(rev Revision, reason string) error,
) (Plan, error) {
	revs = slices.Clone(revs)
	SortRevisionsCanonically(revs)

	plan, err := strategy.Cleanup(now, revs, ctx, logger)
	if err != nil {
		return nil, err
	}
	plan = withFutureDates(plan, revs, now, logger)

	deleted := plan.Deleted()

	errs := make([]error, len(deleted))
	for i, rev := range deleted {
		err := cleanup(rev, plan.Reason(rev.Key()))
		errs[i] = err
	}
	return plan, errors.Join(errs...)
}

var (
	durationParserRegex = regexp.MustCompile(`^(\d+)([wdhms])$`)
)

func parseDuration(s string) (time.Duration, error) {
	s = strings.TrimSpace(s)
	m := durationParserRegex.FindStringSubmatch(s)
	if len(m) < 1 {
		return 0, fmt.Errorf("invalid duration number: %q", s)
	}
	value := m[1]
	unit := m[2]

	n, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("invalid duration number: %w", err)
	}

	var u time.Duration
	switch unit {
	case "w":
		u = 7 * 24 * time.Hour
	case "d":
		u = 24 * time.Hour
	case "h":
		u = time.Hour
	case "m":
		u = time.Minute
	case "s":
		u = time.Second
	default:
		return 0, fmt.Errorf("unknown duration unit: %q", unit)
	}
	// A time.Duration holds at most about 292 years. Beyond that the multiplication would
	// silently overflow, e.g. into a negative duration, which would put the cutoff into the
	// future and delete all revisions instead of keeping them.
	if limit := int64(math.MaxInt64 / u); int64(n) > limit {
		return 0, fmt.Errorf("duration %q is too large, at most %d%s is supported", s, limit, unit)
	}
	return time.Duration(n) * u, nil
}

var (
	// A duration: a number with a unit suffix (w, d, h, m, s). Durations in days or weeks
	// with an uppercase unit (D, W) align the cutoff to midnight (see computeCutoffFromRetention).
	durationSpec = regexp.MustCompile(`^\d+[wdhmsDW]$`)
)

func durationFromConfig(config string) (bool, time.Duration, bool, error) {
	config = strings.TrimSpace(config)
	if !durationSpec.MatchString(config) {
		return false, 0, false, nil
	}

	truncateToMidnight := false
	if unit := config[len(config)-1]; unit == 'D' || unit == 'W' {
		config = strings.ToLower(config)
		truncateToMidnight = true
	}

	duration, err := parseDuration(config)
	if err != nil {
		return true, 0, false, err
	}

	return true, duration, truncateToMidnight, nil
}

// computeCutoffFromRetention computes the point in time that separates the revisions within a
// duration from the ones beyond it. d is the negative duration, e.g. -48h for "2d".
//
// With truncateToMidnight, the cutoff is aligned to midnight in the time zone of now (the server's
// local time zone in practice): it is today's midnight minus the given number of days, so "0D"
// means "since today 00:00", "1D" "since yesterday 00:00" and "1W" "since 00:00 seven days ago"
// (not since the start of the calendar week). Calendar arithmetic is used rather than subtracting
// d, so that the cutoff is exactly at midnight also across daylight saving time changes.
// Only durations in days or weeks can be truncated to midnight (see durationSpec), so d is always
// a multiple of 24h in that case.
func computeCutoffFromRetention(now time.Time, d time.Duration, truncateToMidnight bool) time.Time {
	if truncateToMidnight {
		year, month, day := now.Date()
		midnight := time.Date(year, month, day, 0, 0, 0, 0, now.Location())
		return midnight.AddDate(0, 0, int(d/(24*time.Hour)))
	}
	return now.Add(d)
}

var (
	countConfigFormat = regexp.MustCompile(`^(\d+)$`)
)

func countFromConfig(config string) (bool, int, error) {
	if m := countConfigFormat.FindStringSubmatch(config); len(m) > 1 {
		if maxVersionsPerFile, err := strconv.Atoi(m[1]); err != nil {
			return true, 0, err
		} else if maxVersionsPerFile < 0 {
			return true, 0, fmt.Errorf("invalid value for maxVersionsPerFile: %q", m[1])
		} else {
			return true, maxVersionsPerFile, nil
		}
	} else {
		return false, 0, nil
	}
}

const (
	day  = 24 * time.Hour
	week = 7 * day
)

// formatDurationSpec renders a duration in the syntax of the configuration, e.g. "7d", "2w" or
// "1D" when aligned to midnight, for the reasons in a Plan.
func formatDurationSpec(d time.Duration, truncateToMidnight bool) string {
	var spec string
	switch {
	case d == 0 && truncateToMidnight:
		spec = "0d"
	case d == 0:
		spec = "0s"
	case d%week == 0:
		spec = fmt.Sprintf("%dw", d/week)
	case d%day == 0:
		spec = fmt.Sprintf("%dd", d/day)
	case d%time.Hour == 0:
		spec = fmt.Sprintf("%dh", d/time.Hour)
	case d%time.Minute == 0:
		spec = fmt.Sprintf("%dm", d/time.Minute)
	default:
		spec = fmt.Sprintf("%ds", d/time.Second)
	}
	if truncateToMidnight {
		spec = strings.ToUpper(spec)
	}
	return spec
}

// humanDuration renders a duration in words, e.g. "1 minute" or "10 seconds", for the
// reasons in a Plan.
func humanDuration(d time.Duration) string {
	unit := func(n time.Duration, name string) string {
		if n == 1 {
			return "1 " + name
		}
		return fmt.Sprintf("%d %ss", n, name)
	}
	switch {
	case d >= week && d%week == 0:
		return unit(d/week, "week")
	case d >= day && d%day == 0:
		return unit(d/day, "day")
	case d >= time.Hour && d%time.Hour == 0:
		return unit(d/time.Hour, "hour")
	case d >= time.Minute && d%time.Minute == 0:
		return unit(d/time.Minute, "minute")
	case d >= time.Second && d%time.Second == 0:
		return unit(d/time.Second, "second")
	default:
		return d.String()
	}
}

// formatTime renders a point in time for the reasons in a Plan.
func formatTime(t time.Time) string {
	return t.Format(time.RFC3339)
}

// withFutureDates points out the revisions dated in the future in the reasons of plan, and
// logs a warning about each of them.
//
// A revision's time is the modification time of its content, which clients and changes on disk
// can set to any time, e.g. a client with a wrong clock. A revision dated in the future counts
// as the newest one: it takes the first places of a number of versions to keep, and no
// duration deletes it before its date has passed. That is kept as it is, but made visible, so
// that the admin can look into the client or the clock.
func withFutureDates(plan Plan, revs []Revision, now time.Time, logger *zerolog.Logger) Plan {
	future := map[string]time.Time{}
	for _, rev := range revs {
		if t := rev.LastModificationTime(); t.After(now) {
			future[rev.Key()] = t
			if logger != nil {
				logger.Warn().Str("revision", rev.Key()).Time("time", t).
					Msg("revision cleanup: revision dated in the future, it counts as the newest one")
			}
		}
	}
	if len(future) == 0 {
		return plan
	}
	return &futureDatesPlan{Plan: plan, future: future}
}

// futureDatesPlan adds a note to the reasons of the revisions dated in the future.
type futureDatesPlan struct {
	Plan
	future map[string]time.Time
}

func (p *futureDatesPlan) Reason(key string) string {
	reason := p.Plan.Reason(key)
	if t, ok := p.future[key]; ok && reason != "" {
		reason += fmt.Sprintf("; dated in the future (%s), so it counts as the newest", formatTime(t))
	}
	return reason
}
