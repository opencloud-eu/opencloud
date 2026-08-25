// SPDX-License-Identifier: Apache-2.0
//
// The OpenSearch Contributors require contributions made to
// this file be licensed under the Apache-2.0 license or a
// compatible open source license.

package opensearchtransport

import (
	"net/url"
)

// singleServerPool is a trivial connection pool for single-node clusters.
// All operations are no-ops except Next(), which returns the single connection.
type singleServerPool struct {
	connection *Connection

	metrics *metrics
}

// newSingleServerPool constructs a singleServerPool and marks the connection
// as lcActive.
//
// singleServerPool's OnSuccess/OnFailure are no-ops, so a connection placed
// in this pool would otherwise stay at its initial lifecycle (typically
// lcUnknown / "dead") for the lifetime of the pool. Observers that read
// lifecycle state via buildConnectionMetric (test readiness gates, metric
// consumers, etc.) would then incorrectly classify the only available
// connection as not-ready.
//
// The transition mirrors what multiServerPool does for its initial conns:
// set lcActive, clear lcUnknown|lcStandby. Pool-demotion paths
// (multiServerPool -> singleServerPool) reuse a connection that may already
// be lcActive, in which case the CAS is a documented errLifecycleNoop and
// the pool is published with the conn's existing state intact. Caller holds
// conn.mu, so the masked readiness/position bits cannot be mutated
// concurrently and errLifecycleConflict is impossible.
func newSingleServerPool(conn *Connection, m *metrics) *singleServerPool {
	if conn != nil {
		conn.mu.Lock()
		// errLifecycleNoop is fine (CAS would not change state).
		// errLifecycleConflict is impossible: we hold conn.mu and
		// casLifecycle requires callers to hold it before mutating
		// readiness/position bits.
		conn.casLifecycle(conn.loadConnState(), 0, lcActive, lcUnknown|lcStandby) //nolint:errcheck // lock held; only errLifecycleNoop possible
		conn.mu.Unlock()
	}
	return &singleServerPool{
		connection: conn,
		metrics:    m,
	}
}

// Compile-time check that singleServerPool implements ConnectionPool.
var _ ConnectionPool = (*singleServerPool)(nil)

// Next returns the single connection, or ErrNoConnections when that
// connection has never proven reachable.
//
// A discovered node whose publish_address is unroutable from the client
// (e.g. a NAT'd or misconfigured cluster) is minted lcDead|lcNeedsHardware
// and, when it is the only node, ends up as this pool's sole connection.
// Returning it blindly would route every request at an unreachable address
// and -- because the failure surfaces as a dial error rather than
// ErrNoConnections -- bypass the seed-URL fallback entirely (see
// availableForRouting and PR #954, which added the same gate to
// multiServerPool but not here). Reporting ErrNoConnections instead lets
// the request cascade to performSeedFallback until the node health-checks
// clean and clears lcNeedsHardware. A user-supplied seed connection is
// always available (availableForRouting short-circuits on c.seed), so a
// genuine single-seed-node pool is unaffected.
func (cp *singleServerPool) Next() (*Connection, error) {
	if cp.connection == nil || !cp.connection.availableForRouting() {
		return nil, ErrNoConnections
	}
	return cp.connection, nil
}

// OnSuccess is a no-op for single connection pool.
func (cp *singleServerPool) OnSuccess(*Connection) {}

// OnFailure is a no-op for single connection pool.
func (cp *singleServerPool) OnFailure(*Connection) error { return nil }

// URLs returns the list of URLs of available connections.
func (cp *singleServerPool) URLs() []*url.URL { return []*url.URL{cp.connection.URL} }

func (cp *singleServerPool) connections() []*Connection { return []*Connection{cp.connection} }
