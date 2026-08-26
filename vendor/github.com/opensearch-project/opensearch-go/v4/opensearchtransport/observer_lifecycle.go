// SPDX-License-Identifier: Apache-2.0
//
// The OpenSearch Contributors require contributions made to
// this file be licensed under the Apache-2.0 license or a
// compatible open source license.

package opensearchtransport

import "time"

// ConnectionEvent is a point-in-time snapshot of a connection captured at the
// moment a lifecycle event occurs. All fields are safe to retain after the
// callback returns.
type ConnectionEvent struct {
	// URL is the connection's address (e.g. "http://127.0.0.1:9200").
	URL string

	// ID is the node's unique identifier (populated after discovery).
	ID string

	// Name is the node's human-readable name (populated after discovery).
	Name string

	// Roles lists the node's roles (e.g. ["data", "ingest"]).
	Roles []string

	// Version is the node's OpenSearch version string.
	Version string

	// PoolName identifies which pool dispatched this event (e.g. "roundrobin", "role:data").
	PoolName string

	// Failures is the cumulative failure count at the time of the event.
	Failures int64

	// ActiveCount is the number of active connections in the pool at the time of the event.
	ActiveCount int

	// DeadCount is the number of dead connections in the pool at the time of the event.
	DeadCount int

	// StandbyCount is the number of standby connections in the pool at the time of the event.
	StandbyCount int

	// State is the connection's packed state snapshot at the time of the event.
	// Use its methods to query lifecycle phase and warmup progress.
	State ConnState

	// Weight is the connection's round-robin weight at the time of the event.
	// Default 1; higher for nodes with more cores in heterogeneous clusters.
	Weight int

	// Error is non-nil only for OnHealthCheckFail events.
	Error error

	// Timestamp is when the event was created.
	Timestamp time.Time
}

// newConnectionEvent builds a ConnectionEvent snapshot from a Connection.
// This reads only immutable fields (URL, ID, Name, Roles, Version) and atomic
// counters (failures, state), so it is safe to call while holding locks.
//
// Pool counts are derived from lifecycle bits (via lifecycleCounts), not
// structural list positions. This ensures reported counts reflect actual
// connection state even when lazy cleanup has not yet reconciled list membership.
func newConnectionEvent(poolName string, c *Connection, counts lifecycleCounts) ConnectionEvent {
	event := ConnectionEvent{
		URL:          c.URL.String(),
		ID:           c.ID,
		Name:         c.Name,
		Version:      c.loadVersion(),
		PoolName:     poolName,
		Failures:     c.failures.Load(),
		State:        ConnState{packed: c.state.Load()},
		Weight:       c.effectiveWeight(),
		ActiveCount:  counts.active,
		DeadCount:    counts.dead,
		StandbyCount: counts.standby,
		Timestamp:    time.Now().UTC(),
	}
	if len(c.Roles) > 0 {
		event.Roles = c.Roles.toSlice()
	}
	return event
}
