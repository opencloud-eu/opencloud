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

type countRetentionRevisionCleanupStrategy struct {
	versionsToSafeguard int
}

var _ RevisionCleanupStrategy = &countRetentionRevisionCleanupStrategy{}

func newCountRetentionRevisionCleanupStrategy(maxVersionsPerFile int) (RevisionCleanupStrategy, error) {
	return &countRetentionRevisionCleanupStrategy{
		versionsToSafeguard: maxVersionsPerFile,
	}, nil
}

func newCountRetentionRevisionCleanupStrategyFromString(config string) (RevisionCleanupStrategy, error) {
	if match, maxVersionsPerFile, err := countFromConfig(config); !match {
		return nil, nil
	} else if err != nil {
		return nil, err
	} else {
		return newCountRetentionRevisionCleanupStrategy(maxVersionsPerFile)
	}
}

func (s *countRetentionRevisionCleanupStrategy) Cleanup(now time.Time, revs []Revision, ctx context.Context, logger *zerolog.Logger) (Plan, error) {
	plan := NewSimplePlan()
	n := min(s.versionsToSafeguard, len(revs))
	spec := strconv.Itoa(s.versionsToSafeguard)
	plan.Safeguard(fmt.Sprintf("protected by %q: one of the %d newest", spec, s.versionsToSafeguard), revs[:n]...)
	plan.Retain(fmt.Sprintf("not protected by %q: not one of the %d newest", spec, s.versionsToSafeguard), revs[n:]...)
	return plan, nil
}
