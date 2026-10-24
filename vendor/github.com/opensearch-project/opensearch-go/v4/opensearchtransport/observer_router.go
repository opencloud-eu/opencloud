// SPDX-License-Identifier: Apache-2.0
//
// The OpenSearch Contributors require contributions made to
// this file be licensed under the Apache-2.0 license or a
// compatible open source license.

package opensearchtransport

import (
	"math"
	"time"
)

// RouteEvent is a point-in-time snapshot of a routing
// decision captured at the moment a request is routed to a node. All fields
// are safe to retain after the callback returns.
type RouteEvent struct {
	// IndexName is the target index extracted from the request path.
	IndexName string

	// Key is the full routing key used for rendezvous hashing.
	// For index-level routing this is the same as IndexName.
	// For document-level routing this is "{index}/{docID}".
	Key string

	// FanOut is the effective fan-out (K) used for this routing decision.
	FanOut int

	// TotalNodes is the number of active connections available for selection.
	TotalNodes int

	// CandidateCount is the number of nodes returned by rendezvous hashing
	// (may be less than FanOut if fewer nodes are available).
	CandidateCount int

	// Selected holds the score breakdown for the node that won.
	Selected RouteCandidate

	// Candidates holds the score breakdown for all K candidates, including
	// the selected node. Ordered by rendezvous hash rank (not by score).
	Candidates []RouteCandidate

	// ShardMapLoaded is true when shard placement data has been received
	// from /_cat/shards. When false, all candidates use the default penalty
	// multiplier (32.0) until the first shard catalog refresh completes.
	ShardMapLoaded bool

	// RoutingValue is the explicit ?routing=X value extracted from the
	// request query string. Empty when no routing parameter was present.
	RoutingValue string

	// EffectiveRoutingKey is the routing value actually used for murmur3
	// shard-exact computation. When RoutingValue is non-empty, this
	// equals RoutingValue. When RoutingValue is empty and the request
	// is a document-level operation, this equals the document ID
	// (OpenSearch default: _id is the routing value). Empty when
	// neither ?routing= nor a document ID is available.
	EffectiveRoutingKey string

	// TargetShard is the shard number computed via murmur3 hashing of
	// the effective routing value. -1 when no effective routing value
	// was present or shard map data was unavailable.
	TargetShard int

	// ShardExactMatch is true when the client successfully used murmur3
	// shard-exact routing to select a node known to host the target shard.
	// When false with a non-empty RoutingValue, the client fell back to
	// rendezvous hashing (e.g., shard map not yet loaded).
	ShardExactMatch bool

	// MaxConcurrentShardRequests is the adaptive shard fan-out limit derived
	// from the selected connection's search pool cwnd. Zero when adaptive
	// concurrency does not apply (shard-exact routing, non-search pool,
	// feature disabled, or pre-existing caller override).
	MaxConcurrentShardRequests int

	// Timestamp is when the routing decision was made.
	Timestamp time.Time
}

// RouteCandidate holds the score breakdown for one node evaluated
// during routing.
type RouteCandidate struct {
	// URL is the connection's address.
	URL string

	// ID is the node's unique identifier.
	ID string

	// Name is the node's human-readable name.
	Name string

	// RTTBucket is the median RTT bucket for this node.
	// -1 indicates no RTT data is available.
	RTTBucket int64

	// InFlight is the number of in-flight requests tracked by this client
	// for the named pool on this connection at the time of scoring.
	InFlight int32

	// Cwnd is the congestion window for the named pool on this connection.
	// Reflects the AIMD-adjusted ceiling from the stats poller.
	Cwnd int32

	// PoolName identifies the thread pool used for scoring (e.g., "search",
	// "write"). Empty string means the default synthetic pool.
	PoolName string

	// PoolOverloaded is true if the pool has been marked overloaded
	// (delta(rejected) > 0 or HTTP 429).
	PoolOverloaded bool

	// ShardCostMultiplier is the shard-role-based score multiplier.
	// Determined by the shard cost table for the request direction (read vs write)
	// and the node's shard composition for the target index.
	ShardCostMultiplier float64

	// WarmupPenalty is the warmup penalty multiplier for observability.
	// 1.0 for fully-warmed connections, up to warmupPenaltyMax at the
	// start of warmup. Reported for diagnostics only --not included in
	// the selection Score (warmup is handled by skip/accept gating).
	WarmupPenalty float64

	// Score is the composite routing score:
	//   RTTBucket * (InFlight + 1) / Cwnd * ShardCostMultiplier.
	// Lower score = preferred node. Warmup state is handled separately by
	// [connScoreSelect] via skip/accept gating, not included in the score.
	Score float64
}

// newRouteCandidate creates a RouteCandidate snapshot from a connection.
// When shard is non-nil (shard lookup path), uses [forShard] for per-shard
// primary/replica cost. When nil, uses [forNode] with per-node aggregate data.
//
// When scoreFunc is non-nil, it is used to compute the score (matching
// the dynamic scoring logic in [connScoreSelect]). When nil, the static
// formula is used via [calcConnDefaultScore].
func newRouteCandidate(
	conn *Connection,
	slot *indexSlot,
	shard *shardNodes,
	costs *shardCostMultiplier,
	poolName string,
	poolInfoReady bool,
	scoreFunc connScoreFunc,
) RouteCandidate {
	bucket := conn.rttRing.medianBucket()
	var scm float64
	var primaryPct float64
	switch {
	case shard != nil:
		scm = costs.forShard(shard, conn.Name)
		primaryPct = calcShardPrimaryPct(shard, conn.Name)
	case slot != nil:
		nodeInfo := slot.shardNodeInfoFor(conn.Name)
		scm = costs.forNode(nodeInfo)
		primaryPct = calcNodePrimaryPct(nodeInfo)
	default:
		scm = costs.forNode(nil) // cluster-level: no index slot
	}
	inFlight := conn.loadInFlight(poolName)
	cwnd := conn.loadCwnd(poolName, poolInfoReady)
	overloaded := conn.isPoolOverloaded(poolName)

	// Compute score using the same logic as connScoreSelect.
	var score float64
	switch {
	case overloaded:
		score = math.MaxFloat64
	case scoreFunc != nil:
		score = scoreFunc(conn, scm, primaryPct, poolName, poolInfoReady)
	default:
		score = calcConnDefaultScore(conn, scm, poolName, poolInfoReady)
	}

	// Compute warmup penalty for observability.
	wp := 1.0
	cs := conn.loadConnState()
	if cs.isWarmingUp() {
		wp = warmupPenalty(cs)
	}

	// Expose -1 for unknown RTT in the candidate, but use the raw
	// rttBucketUnknown value for the score calculation (matches calcConnScore).
	displayBucket := bucket.Int64()
	if bucket.IsUnknown() {
		displayBucket = -1
	}

	return RouteCandidate{
		URL:                 conn.URLString,
		ID:                  conn.ID,
		Name:                conn.Name,
		RTTBucket:           displayBucket,
		InFlight:            inFlight,
		Cwnd:                cwnd,
		PoolName:            poolName,
		PoolOverloaded:      overloaded,
		ShardCostMultiplier: scm,
		WarmupPenalty:       wp,
		Score:               score,
	}
}

// routeEventParams collects the inputs for [buildRouteEvent].
// Using a struct avoids a 16-parameter positional function signature.
type routeEventParams struct {
	indexName           string
	key                 string
	fanOut              int
	totalNodes          int
	candidates          []*Connection
	best                *Connection
	slot                *indexSlot
	shard               *shardNodes
	costs               *shardCostMultiplier
	poolName            string
	routingValue        string
	effectiveRoutingKey string
	targetShard         int
	shardExactMatch     bool
	poolInfoReady       bool
	adaptiveMCSR        int
	scoreFunc           connScoreFunc
}

// buildRouteEvent constructs a RouteEvent from a completed
// routing decision. Called after the best candidate is selected.
func buildRouteEvent(p routeEventParams) RouteEvent {
	cs := make([]RouteCandidate, len(p.candidates))
	var selected RouteCandidate
	for i, c := range p.candidates {
		cs[i] = newRouteCandidate(c, p.slot, p.shard, p.costs, p.poolName, p.poolInfoReady, p.scoreFunc)
		if c == p.best {
			selected = cs[i]
		}
	}

	return RouteEvent{
		IndexName:                  p.indexName,
		Key:                        p.key,
		FanOut:                     p.fanOut,
		TotalNodes:                 p.totalNodes,
		CandidateCount:             len(p.candidates),
		Selected:                   selected,
		Candidates:                 cs,
		ShardMapLoaded:             p.slot != nil && p.slot.shardNodeNames.Load() != nil,
		RoutingValue:               p.routingValue,
		EffectiveRoutingKey:        p.effectiveRoutingKey,
		TargetShard:                p.targetShard,
		ShardExactMatch:            p.shardExactMatch,
		MaxConcurrentShardRequests: p.adaptiveMCSR,
		Timestamp:                  time.Now().UTC(),
	}
}

// ShardMapInvalidationEvent is emitted when a routing failure flags a
// connection's shard placement as stale. The connection is excluded from
// routing candidate sets until a /_cat/shards refresh completes.
type ShardMapInvalidationEvent struct {
	// ConnURL is the URL of the connection that was flagged.
	ConnURL string

	// ConnName is the node name of the flagged connection.
	ConnName string

	// Reason describes what triggered the invalidation.
	Reason string // "transport_error" or "http_status_retry"

	// Timestamp is when the invalidation occurred.
	Timestamp time.Time
}
