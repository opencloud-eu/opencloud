// SPDX-License-Identifier: Apache-2.0
//
// The OpenSearch Contributors require contributions made to
// this file be licensed under the Apache-2.0 license or a
// compatible open source license.

package opensearchtransport

import (
	"context"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// Compile-time interface compliance checks.
var (
	_ Policy             = (*poolRouter)(nil)
	_ policyConfigurable = (*poolRouter)(nil)
	_ policyTyped        = (*poolRouter)(nil)
	_ policyOverrider    = (*poolRouter)(nil)
)

// poolRouter wraps any Policy and applies connection scoring
// to the pool returned by the inner policy's Eval().
//
// When the inner policy returns a multiServerPool (e.g., a role-based pool
// of data nodes), the wrapper:
//  1. Reads the pool's active connections
//  2. Extracts the index (and optionally document ID) from the request path
//  3. Selects top-K nodes via rendezvous hash within the role pool
//  4. Picks the best candidate by connection score (RTT * (inFlight+1)/cwnd * shardCost)
//  5. Returns a NextHop with the selected node
//
// For non-index requests or when the inner policy returns nil, the wrapper
// passes through transparently.
type poolRouter struct {
	inner             Policy
	cache             *indexSlotCache
	shardCosts        *shardCostMultiplier
	scoreFunc         connScoreFunc // non-nil for read routes (search, get); nil for write routes
	poolName          string        // Thread pool name for congestion tracking (e.g., "search", "write")
	jitter            atomic.Int64
	observer          atomic.Pointer[ConnectionObserver]
	poolInfoReady     *atomic.Bool  // nil-safe; true once thread pool quorum is reached
	clusterSearchCwnd *atomic.Int32 // nil-safe; cluster-wide search cwnd for MCSR
	policyState       atomic.Int32  // Bitfield: env override bits only (dynamic state from inner)

	// mu guards sortedConns -- a pre-sorted (by RTT) snapshot of active
	// connections from the inner policy's pool. Rebuilt on DiscoveryUpdate;
	// read (without copy) on every Eval.
	mu struct {
		sync.RWMutex
		sortedConns []*Connection
	}
}

func (p *poolRouter) policyTypeName() string { return policyTypeNameRouter }
func (p *poolRouter) setEnvOverride(enabled bool) {
	psSetEnvOverride(&p.policyState, enabled)
}

// wrapWithRouter wraps a policy with connection-scoring selection. The costs
// parameter selects the shard cost table: [shardCostForReads] for read
// routes or [shardCostForWrites] for write routes (e.g., bulk ingest).
// The poolName identifies the server-side thread pool for congestion
// tracking (e.g., "search", "write", "get"); empty string uses the
// default pool.
func wrapWithRouter(
	inner Policy, cache *indexSlotCache,
	costs *shardCostMultiplier, poolName string, scoreFunc connScoreFunc,
) Policy {
	return &poolRouter{
		inner:      inner,
		cache:      cache,
		shardCosts: costs,
		scoreFunc:  scoreFunc,
		poolName:   poolName,
	}
}

// Eval delegates to the inner policy, then applies connection scoring on
// the pre-sorted connection list maintained by DiscoveryUpdate.
func (p *poolRouter) Eval(ctx context.Context, req *http.Request) (NextHop, error) {
	if p.policyState.Load()&psEnvDisabled != 0 {
		return NextHop{}, nil
	}

	hop, err := p.inner.Eval(ctx, req)
	if hop.Conn == nil || err != nil {
		return hop, err
	}

	// Determine the routing key from the request path.
	// Document-level key ({index}/{docID}) takes priority over index-level.
	var keyA, keyB, indexName string
	if idx, docID := extractDocumentFromPath(req.URL.Path); idx != "" && docID != "" {
		keyA = idx
		keyB = docID
		indexName = idx
	} else {
		indexName = extractIndexFromPath(req.URL.Path)
		keyA = indexName
	}

	if keyA == "" {
		// Cluster lookup: no index in path, no shard data.
		// Score all active connections by RTT + congestion only.
		p.mu.RLock()
		conns := p.mu.sortedConns
		p.mu.RUnlock()

		if len(conns) == 0 {
			return hop, nil
		}

		scores := acquireFloats(len(conns))
		pir := loadPoolInfoReady(p.poolInfoReady)
		best := connScoreSelect(conns, nil, nil, p.shardCosts, p.poolName, pir, scores.Slice(), p.scoreFunc, nil)
		scores.Release()

		if best == nil || best.loadConnState().lifecycle()&(lcActive|lcStandby) == 0 {
			return hop, nil
		}

		if obs := observerFromAtomic(&p.observer); obs != nil {
			obs.OnRoute(buildRouteEvent(routeEventParams{
				totalNodes:    len(conns),
				fanOut:        len(conns),
				candidates:    conns,
				best:          best,
				costs:         p.shardCosts,
				poolName:      p.poolName,
				targetShard:   -1,
				poolInfoReady: pir,
				scoreFunc:     p.scoreFunc,
			}))
		}

		return NextHop{Conn: best, PoolName: p.poolName}, nil
	}

	// Read the pre-sorted connection snapshot (rebuilt on DiscoveryUpdate).
	p.mu.RLock()
	conns := p.mu.sortedConns
	p.mu.RUnlock()

	if len(conns) == 0 {
		return hop, nil
	}

	// Look up (or create) the index slot for fan-out and shard data.
	slot := p.cache.getOrCreate(indexName)

	// Attempt shard-exact routing. When ?routing=X is present, use it.
	// Otherwise, for document-level operations, use the docID as the
	// routing value (OpenSearch default: _id is the routing value).
	routingValue := extractRouting(req)
	effectiveRoutingKey := routingValue
	if effectiveRoutingKey == "" && keyB != "" {
		effectiveRoutingKey = keyB // OpenSearch default: _id is the routing value
	}

	var candidatesBuf pooledConns
	var shardNum int
	var shard *shardNodes
	var extraCost pooledFloats

	if !strings.Contains(routingValue, routingValueSeparator) {
		candidatesBuf, shardNum, shard = calcSingleKeyCost(p.cache.features, slot, effectiveRoutingKey, conns)
	} else {
		candidatesBuf, extraCost = calcMultiKeyCost(p.cache.features, slot, routingValue, conns)
		shardNum = -1
	}

	if shardCandidates := candidatesBuf.Slice(); len(shardCandidates) > 0 {
		scores := acquireFloats(len(shardCandidates))
		best := connScoreSelect(shardCandidates, slot, shard, p.shardCosts, p.poolName,
			loadPoolInfoReady(p.poolInfoReady), scores.Slice(), p.scoreFunc, extraCost.Slice())
		scores.Release()
		extraCost.Release()

		if obs := observerFromAtomic(&p.observer); obs != nil {
			key := keyA
			if keyB != "" {
				key = keyA + "/" + keyB
			}
			obs.OnRoute(buildRouteEvent(routeEventParams{
				indexName:           indexName,
				key:                 key,
				fanOut:              len(shardCandidates),
				totalNodes:          len(conns),
				candidates:          shardCandidates,
				best:                best,
				slot:                slot,
				shard:               shard,
				costs:               p.shardCosts,
				poolName:            p.poolName,
				routingValue:        routingValue,
				effectiveRoutingKey: effectiveRoutingKey,
				targetShard:         shardNum,
				shardExactMatch:     true,
				poolInfoReady:       loadPoolInfoReady(p.poolInfoReady),
				scoreFunc:           p.scoreFunc,
			}))
		}
		candidatesBuf.Release()

		// Verify the selected connection is still active.
		if best == nil || best.loadConnState().lifecycle()&(lcActive|lcStandby) == 0 {
			return hop, nil
		}

		return NextHop{Conn: best, PoolName: p.poolName}, nil
	}

	// Rendezvous hash fallback.
	fanOut := p.cache.effectiveFanOut(slot, indexName, len(conns))
	shardNodes := slot.shardNodeNameSet()

	// Select top-K via rendezvous hash with jitter rotation.
	bp := getConnSlice(fanOut)
	candidates := rendezvousTopK(keyA, keyB, conns, fanOut, &p.jitter, shardNodes, bp)
	if len(candidates) == 0 {
		putConnSlice(bp)
		return hop, nil
	}

	// Update tier-span equalization (see IndexRouter.Eval).
	var maxBucket rttBucket
	for _, c := range candidates {
		if b := c.rttRing.medianBucket(); b > maxBucket {
			maxBucket = b
		}
	}
	slot.updateSmoothedMaxBucket(float64(maxBucket))

	// Select best candidate with warmup-aware skip/accept.
	scores := acquireFloats(len(candidates))
	best := connScoreSelect(candidates, slot, nil, p.shardCosts, p.poolName,
		loadPoolInfoReady(p.poolInfoReady), scores.Slice(), p.scoreFunc, nil)
	scores.Release()

	// Compute adaptive max_concurrent_shard_requests for search requests
	// routed through a coordinator (non-shard-exact).
	//
	// Prefer the cluster-wide search pool cwnd (aggregated across all polled
	// data nodes) when available. Falls back to the selected connection's
	// per-node cwnd before the first poll cycle completes.
	var adaptiveMCSR int
	if p.poolName == poolSearch && best != nil {
		cwnd := loadClusterSearchCwnd(p.clusterSearchCwnd)
		if cwnd <= 0 {
			cwnd = best.loadCwnd(p.poolName, loadPoolInfoReady(p.poolInfoReady))
		}
		adaptiveMCSR = computeAdaptiveConcurrency(cwnd, p.cache.adaptiveConcurrency, p.cache.features)
	}

	if obs := observerFromAtomic(&p.observer); obs != nil {
		key := keyA
		if keyB != "" {
			key = keyA + "/" + keyB
		}
		obs.OnRoute(buildRouteEvent(routeEventParams{
			indexName:           indexName,
			key:                 key,
			fanOut:              fanOut,
			totalNodes:          len(conns),
			candidates:          candidates,
			best:                best,
			slot:                slot,
			costs:               p.shardCosts,
			poolName:            p.poolName,
			routingValue:        routingValue,
			effectiveRoutingKey: effectiveRoutingKey,
			targetShard:         shardNum,
			poolInfoReady:       loadPoolInfoReady(p.poolInfoReady),
			adaptiveMCSR:        adaptiveMCSR,
			scoreFunc:           p.scoreFunc,
		}))
	}

	putConnSlice(bp)

	// Verify the selected connection is still active (dirty read).
	// If it was demoted since the last DiscoveryUpdate, fall through
	// to the inner policy's result.
	if best == nil || best.loadConnState().lifecycle()&(lcActive|lcStandby) == 0 {
		return hop, nil
	}

	return NextHop{Conn: best, PoolName: p.poolName, MaxConcurrentShardRequests: adaptiveMCSR}, nil
}

// DiscoveryUpdate delegates to the inner policy, then rebuilds the
// pre-sorted connection snapshot from the inner policy's pool.
func (p *poolRouter) DiscoveryUpdate(added, removed, unchanged []*Connection) error {
	err := p.inner.DiscoveryUpdate(added, removed, unchanged)

	// Rebuild the sorted snapshot from the inner policy's pool.
	p.rebuildSortedConns()

	return err
}

// rebuildSortedConns reads active connections from the inner policy's pool
// and stores an RTT-sorted copy. Called after discovery updates and
// health check state changes.
func (p *poolRouter) rebuildSortedConns() {
	conns := extractActiveConnsFromPolicy(p.inner)

	if len(conns) > 0 {
		sortConnectionsByRTT(conns)
	}

	p.mu.Lock()
	p.mu.sortedConns = conns
	p.mu.Unlock()
}

// extractActiveConnsFromPolicy extracts active connections from a policy's
// underlying pool. Walks through IfEnabledPolicy and other wrappers to
// find the first pool with active connections.
func extractActiveConnsFromPolicy(policy Policy) []*Connection {
	switch p := policy.(type) {
	case *RolePolicy:
		if p.pool == nil {
			return nil
		}
		p.pool.mu.RLock()
		n := p.pool.mu.activeCount
		if n == 0 {
			p.pool.mu.RUnlock()
			return nil
		}
		conns := make([]*Connection, n)
		copy(conns, p.pool.mu.ready[:n])
		p.pool.mu.RUnlock()
		return conns
	case *RoundRobinPolicy:
		if p.pool == nil {
			return nil
		}
		p.pool.mu.RLock()
		n := p.pool.mu.activeCount
		if n == 0 {
			p.pool.mu.RUnlock()
			return nil
		}
		conns := make([]*Connection, n)
		copy(conns, p.pool.mu.ready[:n])
		p.pool.mu.RUnlock()
		return conns
	case *IfEnabledPolicy:
		// Try the true branch first, then false.
		if conns := extractActiveConnsFromPolicy(p.truePolicy); len(conns) > 0 {
			return conns
		}
		if p.falsePolicy != nil {
			return extractActiveConnsFromPolicy(p.falsePolicy)
		}
		return nil
	case *PolicyChain:
		for _, sub := range p.policies {
			if conns := extractActiveConnsFromPolicy(sub); len(conns) > 0 {
				return conns
			}
		}
		return nil
	default:
		return nil
	}
}

// CheckDead delegates to the inner policy.
func (p *poolRouter) CheckDead(ctx context.Context, healthCheck HealthCheckFunc) error {
	return p.inner.CheckDead(ctx, healthCheck)
}

// RotateStandby delegates to the inner policy.
func (p *poolRouter) RotateStandby(ctx context.Context, count int) (int, error) {
	return p.inner.RotateStandby(ctx, count)
}

// IsEnabled delegates to the inner policy unless env-overridden.
func (p *poolRouter) IsEnabled() bool {
	if p.policyState.Load()&psEnvDisabled != 0 {
		return false
	}
	return p.inner.IsEnabled()
}

// configurePolicySettings stores the observer and passes config through
// to the inner policy.
func (p *poolRouter) configurePolicySettings(config policyConfig) error {
	if config.observer != nil {
		p.observer.Store(config.observer)
	}
	p.poolInfoReady = config.poolInfoReady
	p.clusterSearchCwnd = config.clusterSearchCwnd

	// Register the MCSR metric callback for the search pool so that
	// ConnectionMetric snapshots include the current per-node value.
	if p.poolName == poolSearch && config.metrics.detailedEnabled() && p.cache.features.adaptiveConcurrencyEnabled() {
		cache := p.cache
		poolInfoReady := p.poolInfoReady
		config.metrics.connMetricCallbacks = append(config.metrics.connMetricCallbacks,
			func(conns []*Connection, cms []ConnectionMetric) error {
				for i, conn := range conns {
					cwnd := conn.loadCwnd(poolSearch, loadPoolInfoReady(poolInfoReady))
					mcsr := computeAdaptiveConcurrency(cwnd, cache.adaptiveConcurrency, cache.features)
					cms[i].MCSR = &mcsr
				}
				return nil
			})
	}

	// Register the router cache snapshot callback.
	if config.metrics.detailedEnabled() {
		cache := p.cache
		config.metrics.snapshotCallbacks = append(config.metrics.snapshotCallbacks,
			func(m *Metrics) error {
				if m.Router == nil {
					snap := cache.snapshot()
					m.Router = &snap
				}
				return nil
			})
	}

	if configurable, ok := p.inner.(policyConfigurable); ok {
		return configurable.configurePolicySettings(config)
	}
	return nil
}

// childPolicies returns the inner policy for tree walking.
func (p *poolRouter) childPolicies() []Policy {
	return []Policy{p.inner}
}

// routerCache implements [routerCacheProvider].
func (p *poolRouter) routerCache() *indexSlotCache {
	return p.cache
}

// updateShardPlacement updates the index slot cache with shard placement data
// from the discovery cycle. Called after /_cat/shards returns shard-to-node
// mappings.
func (p *poolRouter) updateShardPlacement(shardPlacement map[string]*indexShardPlacement, activeNodeCount int) {
	p.cache.updateFromDiscovery(shardPlacement, activeNodeCount, time.Now())
}

// shardPlacementUpdater is an optional interface implemented by policies that
// need shard placement data from the discovery cycle. The discovery flow walks
// the policy tree and calls updateShardPlacement on any policy that implements
// this interface.
type shardPlacementUpdater interface {
	updateShardPlacement(shardPlacement map[string]*indexShardPlacement, activeNodeCount int)
}

// updateShardPlacementTree walks a policy tree and calls updateShardPlacement
// on any node that implements shardPlacementUpdater.
func updateShardPlacementTree(v any, shardPlacement map[string]*indexShardPlacement, activeNodeCount int) {
	if updater, ok := v.(shardPlacementUpdater); ok {
		updater.updateShardPlacement(shardPlacement, activeNodeCount)
	}
	if collector, ok := v.(policyTreeWalker); ok {
		for _, child := range collector.childPolicies() {
			updateShardPlacementTree(child, shardPlacement, activeNodeCount)
		}
	}
}

// policyTreeWalker is implemented by policies that contain sub-policies.
// Used for recursive tree walks (shard placement updates, etc.).
type policyTreeWalker interface {
	childPolicies() []Policy
}
