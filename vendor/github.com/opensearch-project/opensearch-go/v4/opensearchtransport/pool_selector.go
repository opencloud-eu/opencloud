// SPDX-License-Identifier: Apache-2.0
//
// The OpenSearch Contributors require contributions made to
// this file be licensed under the Apache-2.0 license or a
// compatible open source license.

package opensearchtransport

// Partition adjustment signals returned by poolSelector.selectNext.
const (
	capShrink = -1 // Reduce the partition by one
	capRemain = 0  // No change
	capGrow   = 1  // Expand the partition by one
)

// poolSelector determines how a [multiServerPool] selects from its active
// partition. The pool handles everything else: warmup, eviction, standby
// fallback, resurrection, and observer notifications.
//
// Implementations must be safe for concurrent use and must not acquire
// any locks (called with the pool's read lock held).
type poolSelector interface {
	// selectNext picks a connection from ready[:activeCount].
	// activeCount is guaranteed > 0 by the caller.
	//
	// Returns:
	//   conn       -- the selected connection
	//   activeCap  -- capGrow/capRemain/capShrink for the active partition
	//   standbyCap -- capGrow/capRemain/capShrink for the standby partition
	//   err        -- non-nil to signal selection failure
	//
	// For round-robin: always returns (conn, capRemain, nil).
	// For congestion-aware selection: returns capGrow when all active
	// connections exceed a saturation threshold, signaling the pool to
	// promote from standby.
	selectNext(ready []*Connection, activeCount int) (conn *Connection, activeCap int, standbyCap int, err error)
}
