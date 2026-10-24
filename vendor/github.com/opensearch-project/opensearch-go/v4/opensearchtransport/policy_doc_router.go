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
)

// Compile-time interface compliance checks.
var (
	_ Policy             = (*DocRouter)(nil)
	_ policyConfigurable = (*DocRouter)(nil)
	_ policyTyped        = (*DocRouter)(nil)
	_ policyOverrider    = (*DocRouter)(nil)
)

// DocRouter routes document-level requests to a consistent node
// based on the composite key {index}/{docID}. This provides finer-grained
// routing consistency than [IndexRouter] for workloads with hot documents.
//
// Matches requests to document endpoints:
//
//	/{index}/_doc/{id}
//	/{index}/_source/{id}
//	/{index}/_update/{id}
//	/{index}/_explain/{id}
//	/{index}/_termvectors/{id}
//
// For non-document requests, returns a zero-value NextHop to fall through.
type DocRouter struct {
	cache     *indexSlotCache // shared with IndexRouter
	scoreFunc connScoreFunc   // non-nil: dynamic read scoring; nil: static cost only
	poolName  string          // server-side thread pool name for congestion tracking and NextHop.PoolName
	jitter    atomic.Int64
	decay     float64
	observer  atomic.Pointer[ConnectionObserver]

	mu struct {
		sync.RWMutex
		activeConns []*Connection
	}

	policyState atomic.Int32 // Bitfield: psEnabled|psDisabled|psEnvEnabled|psEnvDisabled
	config      policyConfig
}

func (p *DocRouter) policyTypeName() string { return policyTypeNameDocumentRouter }
func (p *DocRouter) setEnvOverride(enabled bool) {
	psSetEnvOverride(&p.policyState, enabled)
}

// NewDocRouter creates a standalone DocRouter configured by RouterOption(s).
// It routes document-level requests ({index}/_doc/{id} and friends) to a
// node hosting the target document via murmur3 shard-exact selection,
// falling back to rendezvous hashing when shard placement is unavailable.
// The router creates its own index slot cache and parses any shard cost
// configuration from [WithShardCosts] (or [OPENSEARCH_GO_SHARD_COST],
// which takes precedence).
//
// Returns an error if the shard cost configuration cannot be parsed or any
// option records a configuration error.
func NewDocRouter(opts ...RouterOption) (*DocRouter, error) {
	cfg, costCfg, err := buildStandaloneRouterConfig(opts)
	if err != nil {
		return nil, err
	}
	cacheCfg := indexSlotCacheConfigFromRouter(cfg)
	r := newDocRouter(newIndexSlotCache(cacheCfg), cfg.decay)
	r.scoreFunc = costCfg.scoreFunc
	return r, nil
}

// newDocRouter is the internal constructor used by tests and by the
// public [NewDocRouter]. The shared cache should be the one owned by an
// [IndexRouter] when fan-out and shard placement need to stay consistent
// across both routers; pass a fresh cache for standalone use.
func newDocRouter(cache *indexSlotCache, decay float64) *DocRouter {
	if decay <= 0 || decay >= 1 {
		decay = defaultDecayFactor
	}
	return &DocRouter{
		cache: cache,
		decay: decay,
	}
}

// configurePolicySettings implements policyConfigurable.
func (p *DocRouter) configurePolicySettings(config policyConfig) error {
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
func (p *DocRouter) IsEnabled() bool {
	return psIsEnabled(p.policyState.Load())
}

// documentEndpoints lists the path segments that indicate a document-level
// operation (segment immediately after the index name).
//
//nolint:gochecknoglobals // Package-level constant map used by extractDocumentFromPath.
var documentEndpoints = map[string]struct{}{
	"_doc":         {},
	"_source":      {},
	"_update":      {},
	"_explain":     {},
	"_termvectors": {},
}

// Eval extracts {index}/{docID} from the request path and routes to a
// consistent node. Returns (NextHop{}, nil) for non-document requests.
func (p *DocRouter) Eval(_ context.Context, req *http.Request) (NextHop, error) {
	if p.policyState.Load()&psEnvDisabled != 0 {
		return NextHop{}, nil
	}

	indexName, docID := extractDocumentFromPath(req.URL.Path)
	if indexName == "" || docID == "" {
		return NextHop{}, nil
	}

	p.mu.RLock()
	conns := p.mu.activeConns
	p.mu.RUnlock()

	if len(conns) == 0 {
		return NextHop{}, nil
	}

	// Use the index slot for fan-out and shard data, but hash on index/docID.
	slot := p.cache.getOrCreate(indexName)

	// Determine the effective routing value for shard selection:
	//   - ?routing=X  -> use X  (explicit override)
	//   - no routing  -> use docID  (OpenSearch default: _id is the routing value)
	// This is the common case: most requests don't carry ?routing=, so we
	// hash the doc ID to find the shard.
	routingValue := extractRouting(req)
	effectiveRoutingKey := routingValue
	if effectiveRoutingKey == "" {
		effectiveRoutingKey = docID
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
		best := connScoreSelect(shardCandidates, slot, shard, &shardCostForReads, p.poolName,
			loadPoolInfoReady(p.config.poolInfoReady), scores.Slice(), p.scoreFunc, extraCost.Slice())
		scores.Release()
		extraCost.Release()

		if obs := observerFromAtomic(&p.observer); obs != nil {
			key := indexName + "/" + docID
			obs.OnRoute(buildRouteEvent(routeEventParams{
				indexName:           indexName,
				key:                 key,
				fanOut:              len(shardCandidates),
				totalNodes:          len(conns),
				candidates:          shardCandidates,
				best:                best,
				slot:                slot,
				shard:               shard,
				costs:               &shardCostForReads,
				poolName:            p.poolName,
				routingValue:        routingValue,
				effectiveRoutingKey: effectiveRoutingKey,
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

	// Rendezvous hash fallback.
	fanOut := p.cache.effectiveFanOut(slot, indexName, len(conns))
	shardNodes := slot.shardNodeNameSet()

	bp := getConnSlice(fanOut)
	candidates := rendezvousTopK(indexName, docID, conns, fanOut, &p.jitter, shardNodes, bp)
	if len(candidates) == 0 {
		putConnSlice(bp)
		return NextHop{}, nil
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
	best := connScoreSelect(candidates, slot, nil, &shardCostForReads, p.poolName,
		loadPoolInfoReady(p.config.poolInfoReady), scores.Slice(), p.scoreFunc, nil)
	scores.Release()

	if obs := observerFromAtomic(&p.observer); obs != nil {
		key := indexName + "/" + docID
		obs.OnRoute(buildRouteEvent(routeEventParams{
			indexName:           indexName,
			key:                 key,
			fanOut:              fanOut,
			totalNodes:          len(conns),
			candidates:          candidates,
			best:                best,
			slot:                slot,
			costs:               &shardCostForReads,
			poolName:            p.poolName,
			routingValue:        routingValue,
			effectiveRoutingKey: effectiveRoutingKey,
			targetShard:         shardNum,
			poolInfoReady:       loadPoolInfoReady(p.config.poolInfoReady),
			scoreFunc:           p.scoreFunc,
		}))
	}

	putConnSlice(bp)

	if best == nil {
		return NextHop{}, nil
	}
	return NextHop{Conn: best, PoolName: p.poolName}, nil
}

// CheckDead is a no-op. Lifecycle is managed by the underlying pool.
func (p *DocRouter) CheckDead(_ context.Context, _ HealthCheckFunc) error {
	return nil
}

// RotateStandby is a no-op. Lifecycle is managed by the underlying pool.
func (p *DocRouter) RotateStandby(_ context.Context, _ int) (int, error) {
	return 0, nil
}

// routerCache implements [routerCacheProvider].
func (p *DocRouter) routerCache() *indexSlotCache {
	return p.cache
}

// DiscoveryUpdate rebuilds the active connection list.
func (p *DocRouter) DiscoveryUpdate(added, removed, _ []*Connection) error {
	return routerDiscoveryUpdate(&p.mu.RWMutex, &p.mu.activeConns, &p.policyState, added, removed)
}

// extractDocumentFromPath returns the index name and document ID from a
// document-level request path. Returns ("", "") for non-document paths.
//
// Matches: /{index}/_doc/{id}, /{index}/_source/{id}, etc.
func extractDocumentFromPath(path string) (string, string) {
	// Strip leading slash.
	if len(path) > 0 && path[0] == '/' {
		path = path[1:]
	}
	if path == "" || path[0] == '_' {
		return "", ""
	}

	// Split: index / endpoint / id
	// Need at least 3 segments.
	before, after, ok := strings.Cut(path, "/") // after index
	if !ok {
		return "", ""
	}
	indexName := before
	rest := after

	before, after, ok = strings.Cut(rest, "/") // after endpoint
	if !ok {
		return "", ""
	}
	endpoint := before
	docID := after

	// Strip trailing slash or query from docID.
	if idx := strings.IndexAny(docID, "/?"); idx >= 0 {
		docID = docID[:idx]
	}

	if docID == "" {
		return "", ""
	}

	if _, ok := documentEndpoints[endpoint]; !ok {
		return "", ""
	}

	return indexName, docID
}
