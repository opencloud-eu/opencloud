// SPDX-License-Identifier: Apache-2.0
//
// The OpenSearch Contributors require contributions made to
// this file be licensed under the Apache-2.0 license or a
// compatible open source license.

package opensearchtransport

import (
	"context"
	"math"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
)

// Compile-time interface compliance checks.
var (
	_ Policy             = (*IndexRouter)(nil)
	_ policyConfigurable = (*IndexRouter)(nil)
	_ policyTyped        = (*IndexRouter)(nil)
	_ policyOverrider    = (*IndexRouter)(nil)
)

// IndexRouter routes requests to a consistent subset of nodes based
// on the target index name. It uses rendezvous hashing for stable node
// selection and RTT-based scoring for AZ-aware load distribution.
//
// When a request targets an index (e.g., GET /my-index/_search), the policy:
//  1. Extracts the index name from the URL path
//  2. Looks up (or creates) a routing slot in the index cache
//  3. Selects top-K nodes via rendezvous hash (K = fan-out for that index)
//  4. Picks the best candidate by connection score (RTT * (inFlight+1)/cwnd * shardCost)
//  5. Returns a NextHop with the selected node
//
// For system endpoints (no index in path), the policy returns (nil, nil)
// to let the next policy in the chain handle the request.
type IndexRouter struct {
	cache      *indexSlotCache
	shardCosts *shardCostMultiplier // shard cost table (nil = shardCostForReads)
	scoreFunc  connScoreFunc        // non-nil: dynamic read scoring; nil: static cost only
	poolName   string               // server-side thread pool name for congestion tracking and NextHop.PoolName
	jitter     atomic.Int64         // counter-based rotation within K-slot set
	decay      float64              // exponential decay factor for request counters
	observer   atomic.Pointer[ConnectionObserver]

	mu struct {
		sync.RWMutex
		activeConns []*Connection // RTT-sorted active connections
	}

	policyState atomic.Int32 // Bitfield: psEnabled|psDisabled|psEnvEnabled|psEnvDisabled
	config      policyConfig
}

func (p *IndexRouter) policyTypeName() string      { return policyTypeNameIndexRouter }
func (p *IndexRouter) setEnvOverride(enabled bool) { psSetEnvOverride(&p.policyState, enabled) }

// NewIndexRouter creates a standalone IndexRouter configured by
// RouterOption(s). It routes a request to a node hosting the target index
// (rendezvous-hashed top-K, optionally narrowed by ?routing= shard-exact
// lookup) without performing document-level routing. The router creates
// its own index slot cache and parses any shard cost configuration from
// [WithShardCosts] (or [OPENSEARCH_GO_SHARD_COST], which takes precedence).
//
// Returns an error if the shard cost configuration cannot be parsed or any
// option records a configuration error.
func NewIndexRouter(opts ...RouterOption) (*IndexRouter, error) {
	cfg, costCfg, err := buildStandaloneRouterConfig(opts)
	if err != nil {
		return nil, err
	}
	cacheCfg := indexSlotCacheConfigFromRouter(cfg)
	r := newIndexRouter(cacheCfg)
	r.shardCosts = &costCfg.reads
	r.scoreFunc = costCfg.scoreFunc
	return r, nil
}

// newIndexRouter is the internal constructor used by tests and by the
// public [NewIndexRouter] after option parsing. Takes the unexported
// [indexSlotCacheConfig] directly.
func newIndexRouter(cacheCfg indexSlotCacheConfig) *IndexRouter {
	if cacheCfg.decayFactor <= 0 || cacheCfg.decayFactor >= 1 {
		cacheCfg.decayFactor = defaultDecayFactor
	}
	return &IndexRouter{
		cache:      newIndexSlotCache(cacheCfg),
		shardCosts: &shardCostForReads,
		decay:      cacheCfg.decayFactor,
	}
}

// configurePolicySettings stores the observer and pool readiness flag.
func (p *IndexRouter) configurePolicySettings(config policyConfig) error {
	p.config = config
	if config.observer != nil {
		p.observer.Store(config.observer)
	}
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
	return nil
}

// IsEnabled returns true if the policy has active connections.
func (p *IndexRouter) IsEnabled() bool {
	return psIsEnabled(p.policyState.Load())
}

// Eval extracts the index name from the request path and routes to
// a consistent node subset. Returns (NextHop{}, nil) for non-index requests.
func (p *IndexRouter) Eval(_ context.Context, req *http.Request) (NextHop, error) {
	if p.policyState.Load()&psEnvDisabled != 0 {
		return NextHop{}, nil
	}

	indexName := extractIndexFromPath(req.URL.Path)
	if indexName == "" {
		// Cluster lookup: no index in path, no shard data.
		// Score all active connections by RTT + congestion only.
		p.mu.RLock()
		conns := p.mu.activeConns
		p.mu.RUnlock()

		if len(conns) == 0 {
			return NextHop{}, nil
		}

		scores := acquireFloats(len(conns))
		best := connScoreSelect(conns, nil, nil, p.shardCosts, p.poolName,
			loadPoolInfoReady(p.config.poolInfoReady), scores.Slice(), p.scoreFunc, nil)
		scores.Release()
		if best == nil {
			return NextHop{}, nil
		}

		return NextHop{Conn: best, PoolName: p.poolName}, nil
	}

	p.mu.RLock()
	conns := p.mu.activeConns
	p.mu.RUnlock()

	if len(conns) == 0 {
		return NextHop{}, nil
	}

	slot := p.cache.getOrCreate(indexName)

	// Attempt shard-exact routing when ?routing=X is present.
	routingValue := extractRouting(req)

	var candidatesBuf pooledConns
	var shardNum int
	var shard *shardNodes
	var extraCost pooledFloats

	if !strings.Contains(routingValue, routingValueSeparator) {
		candidatesBuf, shardNum, shard = calcSingleKeyCost(p.cache.features, slot, routingValue, conns)
	} else {
		candidatesBuf, extraCost = calcMultiKeyCost(p.cache.features, slot, routingValue, conns)
		shardNum = -1
	}

	// Shard-exact path: score the shard-hosting candidates directly.
	if shardCandidates := candidatesBuf.Slice(); len(shardCandidates) > 0 {
		scores := acquireFloats(len(shardCandidates))
		best := connScoreSelect(shardCandidates, slot, shard, p.shardCosts, p.poolName,
			loadPoolInfoReady(p.config.poolInfoReady), scores.Slice(), p.scoreFunc, extraCost.Slice())
		scores.Release()
		extraCost.Release()

		if obs := observerFromAtomic(&p.observer); obs != nil {
			obs.OnRoute(buildRouteEvent(routeEventParams{
				indexName:           indexName,
				key:                 indexName,
				fanOut:              len(shardCandidates),
				totalNodes:          len(conns),
				candidates:          shardCandidates,
				best:                best,
				slot:                slot,
				shard:               shard,
				costs:               p.shardCosts,
				poolName:            p.poolName,
				routingValue:        routingValue,
				effectiveRoutingKey: routingValue,
				targetShard:         shardNum,
				shardExactMatch:     true,
				poolInfoReady:       loadPoolInfoReady(p.config.poolInfoReady),
				scoreFunc:           p.scoreFunc,
			}))
		}
		candidatesBuf.Release()

		if best == nil {
			return NextHop{}, nil
		}
		return NextHop{Conn: best, PoolName: p.poolName}, nil
	}

	// Rendezvous hash fallback: select top-K via fan-out.
	fanOut := p.cache.effectiveFanOut(slot, indexName, len(conns))
	shardNodes := slot.shardNodeNameSet()

	// Select top-K via rendezvous hash with jitter rotation.
	bp := getConnSlice(fanOut)
	candidates := rendezvousTopK(indexName, "", conns, fanOut, &p.jitter, shardNodes, bp)
	if len(candidates) == 0 {
		putConnSlice(bp)
		return NextHop{}, nil
	}

	// Update tier-span equalization: find the max RTT bucket across the
	// candidate set and feed it to the slot's MIAD tracker. This drives
	// cost inflation in recordCPUTime so traffic distributes evenly
	// across all RTT tiers in the fan-out set.
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
		loadPoolInfoReady(p.config.poolInfoReady), scores.Slice(), p.scoreFunc, nil)
	scores.Release()

	// Compute adaptive max_concurrent_shard_requests for search requests
	// routed through a coordinator (non-shard-exact). Mirrors poolRouter's
	// rendezvous-fallback path: prefer the cluster-wide search pool cwnd
	// when available, fall back to the selected connection's per-node cwnd.
	var adaptiveMCSR int
	if p.poolName == poolSearch && best != nil {
		cwnd := loadClusterSearchCwnd(p.config.clusterSearchCwnd)
		if cwnd <= 0 {
			cwnd = best.loadCwnd(p.poolName, loadPoolInfoReady(p.config.poolInfoReady))
		}
		adaptiveMCSR = computeAdaptiveConcurrency(cwnd, p.cache.adaptiveConcurrency, p.cache.features)
	}

	if obs := observerFromAtomic(&p.observer); obs != nil {
		obs.OnRoute(buildRouteEvent(routeEventParams{
			indexName:           indexName,
			key:                 indexName,
			fanOut:              fanOut,
			totalNodes:          len(conns),
			candidates:          candidates,
			best:                best,
			slot:                slot,
			costs:               p.shardCosts,
			poolName:            p.poolName,
			routingValue:        routingValue,
			effectiveRoutingKey: routingValue,
			targetShard:         shardNum,
			poolInfoReady:       loadPoolInfoReady(p.config.poolInfoReady),
			adaptiveMCSR:        adaptiveMCSR,
			scoreFunc:           p.scoreFunc,
		}))
	}

	putConnSlice(bp)

	if best == nil {
		return NextHop{}, nil
	}
	return NextHop{Conn: best, PoolName: p.poolName, MaxConcurrentShardRequests: adaptiveMCSR}, nil
}

// CheckDead is a no-op; lifecycle management is handled by the
// underlying pool that owns the connections.
func (p *IndexRouter) CheckDead(_ context.Context, _ HealthCheckFunc) error {
	return nil
}

// RotateStandby is a no-op. Lifecycle is managed by the underlying pool.
func (p *IndexRouter) RotateStandby(_ context.Context, _ int) (int, error) {
	return 0, nil
}

// routerCache implements [routerCacheProvider].
func (p *IndexRouter) routerCache() *indexSlotCache {
	return p.cache
}

// routerDiscoveryUpdate is the shared DiscoveryUpdate implementation for
// router policies (IndexRouter, DocRouter). It deduplicates connections, applies additions/removals,
// sorts by RTT, and updates the caller's connection list under the provided lock.
func routerDiscoveryUpdate(
	mu *sync.RWMutex,
	activeConns *[]*Connection,
	policyState *atomic.Int32,
	added, removed []*Connection,
) error {
	if policyState.Load()&psEnvDisabled != 0 {
		return nil
	}

	if added == nil && removed == nil {
		return nil
	}

	// Rebuild the full active connection list.
	// Collect all connections from the three lists, dedup by URL.
	seen := make(map[string]struct{})
	var all []*Connection

	mu.RLock()
	for _, c := range *activeConns {
		key := c.URLString
		if _, ok := seen[key]; !ok {
			seen[key] = struct{}{}
			all = append(all, c)
		}
	}
	mu.RUnlock()

	// Add new connections.
	for _, c := range added {
		key := c.URLString
		if _, ok := seen[key]; !ok {
			seen[key] = struct{}{}
			all = append(all, c)
		}
	}

	// Remove old connections.
	if removed != nil {
		removedSet := make(map[string]struct{}, len(removed))
		for _, c := range removed {
			removedSet[c.URLString] = struct{}{}
		}
		filtered := all[:0]
		for _, c := range all {
			if _, found := removedSet[c.URLString]; !found {
				filtered = append(filtered, c)
			}
		}
		all = filtered
	}

	// Sort by RTT bucket ascending (nearest first). This is the invariant
	// that rendezvousTopK depends on for tier-based slot filling.
	sortConnectionsByRTT(all)

	mu.Lock()
	*activeConns = all
	mu.Unlock()

	// A router policy is enabled only when it has a connection available for
	// routing -- a user-supplied seed, or a discovered connection confirmed
	// reachable. A never-verified discovered connection does not count, so the
	// request cascades to the seed fallback rather than being served as a zombie.
	hasAvailable := false
	for _, c := range all {
		if c.availableForRouting() {
			hasAvailable = true
			break
		}
	}
	psSetEnabled(policyState, hasAvailable)

	return nil
}

// DiscoveryUpdate rebuilds the active connection list from topology changes.
func (p *IndexRouter) DiscoveryUpdate(added, removed, _ []*Connection) error {
	return routerDiscoveryUpdate(&p.mu.RWMutex, &p.mu.activeConns, &p.policyState, added, removed)
}

// loadPoolInfoReady returns true if the pointer is non-nil and set.
// Nil-safe: returns false (pre-quorum fallback) when no Client is wired up,
// e.g. in unit tests that construct policies directly.
func loadPoolInfoReady(p *atomic.Bool) bool {
	return p != nil && p.Load()
}

// loadClusterSearchCwnd returns the cluster-wide search cwnd from the atomic
// pointer. Nil-safe: returns 0 when no Client is wired up (e.g., tests
// without a full Client). A return value <= 0 signals the cluster aggregate
// is not yet available.
func loadClusterSearchCwnd(p *atomic.Int32) int32 {
	if p == nil {
		return 0
	}
	return p.Load()
}

const (
	// warmupPenaltyMax is the worst-case multiplier applied at the start of
	// warmup (remaining == total). Matches costUnknown so a freshly-promoted
	// connection scores equivalently to a node with no shard data, then
	// linearly approaches 1.0 as warmup progresses.
	warmupPenaltyMax = costUnknown
)

// warmupPenalty returns a multiplier in [1.0, warmupPenaltyMax] based on
// how far through warmup the connection is. Fully warmed = 1.0 (no penalty).
// Used by the observer event for reporting; not included in the selection score.
func warmupPenalty(cs connState) float64 {
	lcMgr := cs.lifecycleManager()
	total := lcMgr.rounds()
	if total <= 0 {
		return 1.0
	}
	remaining := cs.roundManager().rounds()
	fraction := float64(remaining) / float64(total)
	return 1.0 + (warmupPenaltyMax-1.0)*fraction
}

// connScoreSelect picks the best connection from scored candidates with
// warmup-aware skip/accept. Candidates are tried in ascending score order:
//
//   - Not warming -> selected immediately.
//   - Warming -> [tryWarmupSkip]:
//   - warmupAccepted or warmupInactive -> selected (S-curve ramp-up).
//   - warmupSkipped -> try next candidate.
//   - All candidates skip -> starvation prevention: pick the one closest
//     to its next accept point (lowest remaining skip count).
//
// This avoids the circular dependency where a warmup penalty in the score
// prevents selection, which prevents warmup advancement. Warmup rounds
// advance naturally at a rate proportional to how often the connection
// would win scoring --good candidates warm up fast, poor ones warm up
// slowly via background health checks.
//
// The scores buffer must have len >= len(candidates). It is populated by
// this function and can be used by the caller for observer reporting.
//
// When shard is non-nil (shard lookup path), scoring uses [forShard] for
// per-shard primary/replica cost. When nil (index lookup or cluster lookup),
// scoring uses [forNode] with per-node aggregate data from the index slot.
//
// When scoreFunc is non-nil, it replaces the static scoring formula for
// all candidates. This allows operation-specific scoring strategies (e.g.,
// dynamic read cost that adjusts primary shard cost based on write-pool
// utilization). When nil, the static formula is used via [calcConnDefaultScore].
func connScoreSelect(
	candidates []*Connection,
	slot *indexSlot,
	shard *shardNodes,
	costs *shardCostMultiplier,
	poolName string,
	poolInfoReady bool,
	scores []float64,
	scoreFunc connScoreFunc,
	extraCost []float64,
) *Connection {
	n := len(candidates)
	if n == 0 {
		return nil
	}

	// Compute scores once upfront.
	for i, c := range candidates {
		var sc float64
		var primaryPct float64
		switch {
		case shard != nil:
			sc = costs.forShard(shard, c.Name)
			primaryPct = calcShardPrimaryPct(shard, c.Name)
		case slot != nil:
			nodeInfo := slot.shardNodeInfoFor(c.Name)
			sc = costs.forNode(nodeInfo)
			primaryPct = calcNodePrimaryPct(nodeInfo)
		default:
			sc = costUnknown
		}

		if scoreFunc != nil {
			scores[i] = scoreFunc(c, sc, primaryPct, poolName, poolInfoReady)
		} else {
			scores[i] = calcConnDefaultScore(c, sc, poolName, poolInfoReady)
		}

		if extraCost != nil {
			scores[i] += extraCost[i]
		}
	}

	// Try candidates in ascending score order (lowest = best).
	// For small fan-out sets (typically 1-3), repeated linear scans
	// are faster than sorting. Tried candidates are marked by setting
	// their score to +Inf, avoiding a bitmask that overflows at >64 candidates.
	var bestWarming *Connection
	bestWarmingSkip := int(^uint(0) >> 1) // max int sentinel

	for range n {
		// Find untried candidate with lowest score.
		bestIdx := -1
		bestScore := math.MaxFloat64
		for i := range n {
			if scores[i] < bestScore {
				bestScore = scores[i]
				bestIdx = i
			}
		}
		if bestIdx < 0 {
			break
		}
		scores[bestIdx] = math.Inf(1) // mark as tried
		c := candidates[bestIdx]

		if !c.loadConnState().isWarmingUp() {
			return c
		}

		switch c.tryWarmupSkip() {
		case warmupAccepted, warmupInactive:
			return c
		case warmupSkipped:
			remSkip := c.loadConnState().roundManager().skipCount()
			if remSkip <= bestWarmingSkip {
				bestWarmingSkip = remSkip
				bestWarming = c
			}
		}
	}

	// Starvation prevention: all candidates are warming and skipped.
	// Return the one closest to its next accept point.
	if bestWarming != nil {
		return bestWarming
	}
	return candidates[0] // Defensive; shouldn't reach here.
}

// sortConnectionsByRTT sorts connections by [rttBucket] ascending.
// Ties are broken by URL string for deterministic ordering.
func sortConnectionsByRTT(conns []*Connection) {
	// Use a simple insertion sort -- connection lists are typically small (3-50).
	for i := 1; i < len(conns); i++ {
		key := conns[i]
		keyBucket := key.rttRing.medianBucket()
		keyURL := key.URLString
		j := i - 1
		for j >= 0 {
			jBucket := conns[j].rttRing.medianBucket()
			if jBucket < keyBucket || (jBucket == keyBucket && conns[j].URLString <= keyURL) {
				break
			}
			conns[j+1] = conns[j]
			j--
		}
		conns[j+1] = key
	}
}

// extractIndexFromPath returns the index name from a request URL path.
// Returns "" for system endpoints (paths starting with /_) and root path.
//
// Examples:
//
//	"/my-index/_search"       -> "my-index"
//	"/my-index/_doc/123"      -> "my-index"
//	"/_cluster/health"        -> ""  (system endpoint)
//	"/"                       -> ""
//	"/my-index"               -> "my-index"
func extractIndexFromPath(path string) string {
	// Strip leading slash.
	if len(path) > 0 && path[0] == '/' {
		path = path[1:]
	}
	if path == "" {
		return ""
	}

	// System endpoints start with underscore after the leading slash.
	if path[0] == '_' {
		return ""
	}

	// Take the first path segment as the index name.
	if before, _, ok := strings.Cut(path, "/"); ok {
		return before
	}
	return path
}

// computeAdaptiveConcurrency derives max_concurrent_shard_requests from the
// selected connection's congestion window for the search thread pool.
//
// The cwnd reflects queue-pressure-informed capacity: AIMD halves cwnd when
// per-request queue wait-time exceeds 1ms, well before the thread pool
// rejects requests. Since max_concurrent_shard_requests controls the
// coordinator's fan-out across the cluster (shard requests land on many
// data nodes' thread pools), the cwnd of the coordinator node -- not any
// single data node's thread pool size -- is the right scaling signal.
//
// The returned value is clamped to [minVal, maxVal] from the config.
//
// Parameters:
//   - cwnd: the connection's current cwnd for the search pool
//   - cfg: adaptive concurrency config (min/max overrides)
//   - features: routing feature flags (checked for adaptiveConcurrencyEnabled)
//
// Returns 0 when the feature is disabled, meaning Perform() should not inject
// the query parameter.
func computeAdaptiveConcurrency(cwnd int32, cfg adaptiveConcurrencyConfig, features routingFeatures) int {
	if !features.adaptiveConcurrencyEnabled() {
		return 0
	}

	minVal := cfg.effectiveMin()
	maxVal := cfg.effectiveMax()

	value := int(cwnd)
	value = min(value, maxVal)
	value = max(value, minVal)

	return value
}
