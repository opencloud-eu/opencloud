// SPDX-License-Identifier: Apache-2.0
//
// The OpenSearch Contributors require contributions made to
// this file be licensed under the Apache-2.0 license or a
// compatible open source license.

package opensearchtransport

import "sync/atomic"

// ConnectionObserver receives notifications about connection lifecycle events
// and routing decisions.
//
// Implementations must be safe for concurrent use from multiple goroutines.
// Methods should return quickly to avoid blocking transport operations.
//
// Observer methods may be called while internal locks are held. Implementations
// must NOT call back into the Client, connection pool, or any method that
// acquires transport-internal locks. Violating this contract will cause a
// deadlock.
type ConnectionObserver interface { //nolint:interfacebloat // lifecycle + routing observer needs all 14 event methods
	// OnPromote is called when a dead connection becomes ready
	// (successful resurrection after health check).
	OnPromote(event ConnectionEvent)

	// OnDemote is called when a ready connection becomes dead
	// (request failure).
	OnDemote(event ConnectionEvent)

	// OnOverloadDetected is called when a ready connection is demoted
	// because the node's resource usage exceeds configured thresholds.
	OnOverloadDetected(event ConnectionEvent)

	// OnOverloadCleared is called when an overload-demoted connection
	// is promoted back to ready after resource usage returns to normal.
	OnOverloadCleared(event ConnectionEvent)

	// OnDiscoveryAdd is called when node discovery finds a new node
	// not previously in the connection pool.
	OnDiscoveryAdd(event ConnectionEvent)

	// OnDiscoveryRemove is called when node discovery determines a node
	// is no longer part of the cluster.
	OnDiscoveryRemove(event ConnectionEvent)

	// OnDiscoveryUnchanged is called for each existing node that remains
	// in the pool after a discovery update.
	OnDiscoveryUnchanged(event ConnectionEvent)

	// OnHealthCheckPass is called when a connection passes a health check.
	OnHealthCheckPass(event ConnectionEvent)

	// OnHealthCheckFail is called when a connection fails a health check.
	// The Error field of ConnectionEvent contains the failure reason.
	OnHealthCheckFail(event ConnectionEvent)

	// OnStandbyPromote is called when a standby connection is promoted
	// to the active partition after passing consecutive health checks.
	OnStandbyPromote(event ConnectionEvent)

	// OnStandbyDemote is called when an active connection is moved to the
	// standby partition (during rotation or discovery overflow).
	OnStandbyDemote(event ConnectionEvent)

	// OnWarmupRequest is called when a request is about to be served by a
	// connection going through warmup. The State field carries the connection's
	// packed state after the warmup round was advanced; use
	// State.IsWarmingUp() to detect whether warmup just completed (false)
	// or is still in progress (true), and State.WarmupRoundsRemaining() /
	// State.WarmupSkipRemaining() to inspect progress.
	OnWarmupRequest(event ConnectionEvent)

	// OnRoute is called after the router selects a node for a
	// request. The event contains the full scoring breakdown for all
	// candidates considered during the routing decision.
	OnRoute(event RouteEvent)

	// OnShardMapInvalidation is called when a routing failure flags a
	// connection's shard placement as stale. The connection is excluded
	// from routing candidates until a /_cat/shards refresh.
	OnShardMapInvalidation(event ShardMapInvalidationEvent)
}

// BaseConnectionObserver is an embeddable no-op implementation of
// ConnectionObserver. Embed it in your own struct and override only the
// methods you care about.
//
//	type myObserver struct {
//	    opensearchtransport.BaseConnectionObserver
//	}
//
//	func (o *myObserver) OnDemote(event opensearchtransport.ConnectionEvent) {
//	    log.Printf("connection dead: %s", event.URL)
//	}
type BaseConnectionObserver struct{}

// OnPromote implements ConnectionObserver (no-op).
func (BaseConnectionObserver) OnPromote(ConnectionEvent) {}

// OnDemote implements ConnectionObserver (no-op).
func (BaseConnectionObserver) OnDemote(ConnectionEvent) {}

// OnOverloadDetected implements ConnectionObserver (no-op).
func (BaseConnectionObserver) OnOverloadDetected(ConnectionEvent) {}

// OnOverloadCleared implements ConnectionObserver (no-op).
func (BaseConnectionObserver) OnOverloadCleared(ConnectionEvent) {}

// OnDiscoveryAdd implements ConnectionObserver (no-op).
func (BaseConnectionObserver) OnDiscoveryAdd(ConnectionEvent) {}

// OnDiscoveryRemove implements ConnectionObserver (no-op).
func (BaseConnectionObserver) OnDiscoveryRemove(ConnectionEvent) {}

// OnDiscoveryUnchanged implements ConnectionObserver (no-op).
func (BaseConnectionObserver) OnDiscoveryUnchanged(ConnectionEvent) {}

// OnHealthCheckPass implements ConnectionObserver (no-op).
func (BaseConnectionObserver) OnHealthCheckPass(ConnectionEvent) {}

// OnHealthCheckFail implements ConnectionObserver (no-op).
func (BaseConnectionObserver) OnHealthCheckFail(ConnectionEvent) {}

// OnStandbyPromote implements ConnectionObserver (no-op).
func (BaseConnectionObserver) OnStandbyPromote(ConnectionEvent) {}

// OnStandbyDemote implements ConnectionObserver (no-op).
func (BaseConnectionObserver) OnStandbyDemote(ConnectionEvent) {}

// OnWarmupRequest implements ConnectionObserver (no-op).
func (BaseConnectionObserver) OnWarmupRequest(ConnectionEvent) {}

// OnRoute implements ConnectionObserver (no-op).
func (BaseConnectionObserver) OnRoute(RouteEvent) {}

// OnShardMapInvalidation implements ConnectionObserver (no-op).
func (BaseConnectionObserver) OnShardMapInvalidation(ShardMapInvalidationEvent) {}

// Compile-time check that BaseConnectionObserver implements ConnectionObserver.
var _ ConnectionObserver = (*BaseConnectionObserver)(nil)

// observerFromAtomic loads a ConnectionObserver from an atomic pointer.
// Returns nil if the pointer is nil or stores nil.
func observerFromAtomic(p *atomic.Pointer[ConnectionObserver]) ConnectionObserver {
	if p == nil {
		return nil
	}
	ptr := p.Load()
	if ptr == nil {
		return nil
	}
	return *ptr
}
