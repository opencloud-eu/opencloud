// SPDX-License-Identifier: Apache-2.0
//
// The OpenSearch Contributors require contributions made to
// this file be licensed under the Apache-2.0 license or a
// compatible open source license.

package opensearchtransport

import (
	"maps"
	"math"
	"sync"
	"sync/atomic"
	"time"
)

const (
	// defaultMinFanOut is the minimum number of nodes in an index slot.
	defaultMinFanOut = 1

	// defaultMaxFanOut caps the fan-out for any single index slot.
	// Prevents pathologically sharded indexes (e.g., 9000 shards across
	// 100 nodes) from inflating the candidate set to the entire cluster.
	// Well-designed indexes at 100-200GB/shard rarely exceed 30 shard-hosting
	// nodes even at multi-TB scale. Rate-driven fan-out (rateFanOut) is also
	// capped by this value. Override with WithMaxFanOut(0) to uncap.
	defaultMaxFanOut = 32

	// defaultIdleEvictionTTL is how long an index slot with zero requests
	// persists before being evicted from the cache.
	defaultIdleEvictionTTL = 90 * time.Minute

	// defaultFanOutPerRequest controls how many requests map to one
	// additional fan-out node. When the decay counter reaches this
	// threshold, fan-out grows by 1.
	// With decay=0.999, steady state at 100 req/s converges to ~100k.
	// Threshold of 500 means fan-out grows by 1 per ~500 concurrent-equivalent requests.
	defaultFanOutPerRequest float64 = 500
)

// indexSlotCache maps index names to their routing slots. Entries are
// created on first request and updated during discovery. The cache
// determines both routing decisions and the client's active connection set.
//
// Fan-out per index is driven by an exponential decay counter:
// high request volume -> larger fan-out -> more nodes in the slot.
// When traffic drops, the counter decays and fan-out contracts.
//
// The entries map uses an atomic pointer to [sync.Map] so the entire map
// can be swapped for a fresh instance when the live entry count drops to
// 50% of its high-water mark. Without this, [sync.Map]'s internal hash
// tables retain the memory footprint of the peak entry count even after
// entries are deleted -- problematic for time-series index patterns
// (e.g., logs-2024.01.15) where thousands of index names rotate daily.
type indexSlotCache struct {
	entries       atomic.Pointer[sync.Map] // map[string]*indexSlot
	highWaterMark atomic.Int64             // peak live entry count (post-eviction)

	// Configuration (immutable after init)
	minFanOut       int
	maxFanOut       int            // caps fan-out per index slot (default 32)
	overrides       map[string]int // per-index fan-out overrides
	idleEvictionTTL time.Duration
	decayFactor     float64
	fanOutPerReq    float64 // decay-counter-to-fan-out divisor

	// Feature configuration from OPENSEARCH_GO_ROUTING_CONFIG.
	// Evaluated once at client init time and immutable after.
	features routingFeatures // bitfield: zero = all enabled

	// Adaptive max_concurrent_shard_requests limits.
	adaptiveConcurrency adaptiveConcurrencyConfig
}

// indexSlot is the per-index routing state.
type indexSlot struct {
	// Request volume tracking. Incremented per request via decay,
	// used to derive fan-out.
	requestDecay decayCounter

	// Effective fan-out (number of nodes in this slot).
	// Updated on discovery or when decay counter crosses a threshold.
	fanOut atomic.Int32

	// Shard placement: number of distinct nodes hosting shards for this
	// index. Updated from /_cat/shards during discovery. Acts as a floor
	// for fan-out (can't usefully fan out to fewer nodes than hold data).
	shardNodeCount atomic.Int32

	// Per-node shard placement for this index: node name -> primary/replica
	// counts. Updated during discovery from /_cat/shards (which returns
	// node names, not transport IDs). The key set doubles as the shard
	// node name set for rendezvousTopK's hard partition; the values provide
	// primary/replica detail for replica-preference scoring.
	shardNodeNames atomic.Pointer[map[string]*shardNodeInfo]

	// Per-shard-number placement data for murmur3 shard-exact routing.
	// Maps shard number -> nodes hosting that shard (primary + replicas).
	// Updated from /_cat/shards during discovery. When non-nil and the
	// request has ?routing=X, the client can compute the exact target
	// shard via murmur3 and route directly to a node hosting it.
	shardMap atomic.Pointer[indexShardMap]

	// Smoothed max RTT bucket across the candidate set for this index.
	// Used for tier-span cost equalization: when the fan-out expands to
	// include remote nodes, local connections accrue inflated cost so
	// that traffic distributes evenly across all RTT tiers (not
	// proportional to the bucket ratio).
	//
	// Updated via MIAD (Multiplicative Increase, Additive Decrease):
	//   - MI: when observed max grows (demand spike), converge fast
	//     so remote tiers absorb load before local nodes brown out.
	//   - AD: when observed max drops (demand subsides), step down
	//     slowly to find steady-state without oscillation.
	smoothedMaxBucketBits atomic.Uint64 // float64 stored as bits
	smoothedMaxBucketNano atomic.Int64  // last update time (UnixNano); 0 = never updated

	// clock provides timestamps for the MIAD algorithm.
	// Set to realClock{} in production via getOrCreate; tests inject
	// a testClock for deterministic behavior.
	clock clock

	// Idle eviction tracking.
	// Set to the current time when requestDecay reaches ~0 during a
	// discovery update. Cleared when a new request arrives. If the slot
	// has been idle for longer than idleEvictionTTL, it is evicted.
	idleSince atomic.Int64 // UnixNano; 0 = active
}

// indexShardMap holds per-shard-number placement data for an index.
// Used by murmur3 shard-exact routing to resolve a computed shard number
// to the node(s) hosting that shard.
type indexShardMap struct {
	// NumberOfPrimaryShards is the count of distinct primary shards.
	NumberOfPrimaryShards int

	// RoutingNumShards is the index's routing_num_shards metadata value,
	// fetched once per index from _cluster/state/metadata. OpenSearch
	// uses this (not NumberOfPrimaryShards) as the hash modulus to
	// allow future index splitting:
	//
	//   shard = floorMod(hash, RoutingNumShards) / RoutingFactor
	//
	// where RoutingFactor = RoutingNumShards / NumberOfPrimaryShards.
	// For a newly created 5-shard index, RoutingNumShards is typically
	// 640 and RoutingFactor is 128.
	//
	// Zero means not yet fetched from the server; shard-exact routing
	// is unavailable until this is populated.
	//
	// Server references:
	//   OperationRouting.java:calculateScaledShardId     -- the routing formula
	//   MetadataCreateIndexService.java:calculateNumRoutingShards -- default computation
	//   IndexMetadata.java:getRoutingNumShards           -- stored per-index metadata
	RoutingNumShards int

	// Shards maps shard number -> nodes hosting that shard.
	Shards map[int]*shardNodes
}

// newIndexSlotCache creates a cache with the given configuration.
func newIndexSlotCache(cfg indexSlotCacheConfig) *indexSlotCache {
	c := &indexSlotCache{
		minFanOut:           cfg.minFanOut,
		maxFanOut:           cfg.maxFanOut,
		overrides:           cfg.overrides,
		idleEvictionTTL:     cfg.idleEvictionTTL,
		decayFactor:         cfg.decayFactor,
		fanOutPerReq:        cfg.fanOutPerReq,
		features:            cfg.features,
		adaptiveConcurrency: cfg.adaptiveConcurrency,
	}
	c.entries.Store(new(sync.Map))

	if c.minFanOut <= 0 {
		c.minFanOut = defaultMinFanOut
	}
	if c.maxFanOut <= 0 {
		c.maxFanOut = defaultMaxFanOut
	}
	if c.idleEvictionTTL <= 0 {
		c.idleEvictionTTL = defaultIdleEvictionTTL
	}
	if c.decayFactor <= 0 || c.decayFactor >= 1 {
		c.decayFactor = defaultDecayFactor
	}
	if c.fanOutPerReq <= 0 {
		c.fanOutPerReq = defaultFanOutPerRequest
	}

	return c
}

// indexSlotCacheConfig holds the configuration for an indexSlotCache.
type indexSlotCacheConfig struct {
	minFanOut       int
	maxFanOut       int
	overrides       map[string]int
	idleEvictionTTL time.Duration
	decayFactor     float64
	fanOutPerReq    float64

	// Feature configuration from OPENSEARCH_GO_ROUTING_CONFIG.
	features routingFeatures // bitfield: zero = all enabled

	// Adaptive max_concurrent_shard_requests limits.
	adaptiveConcurrency adaptiveConcurrencyConfig
}

// getOrCreate returns the slot for indexName, creating one if needed.
// Increments the request decay counter and clears idle state.
func (c *indexSlotCache) getOrCreate(indexName string) *indexSlot {
	m := c.entries.Load()

	if v, ok := m.Load(indexName); ok {
		slot := v.(*indexSlot)
		slot.requestDecay.increment(c.decayFactor)
		slot.idleSince.Store(0) // active
		return slot
	}

	slot := &indexSlot{clock: realClock{}}
	slot.fanOut.Store(int32(c.minFanOut))      //nolint:gosec // minFanOut is bounded by config (default 1, max 32).
	slot.requestDecay.increment(c.decayFactor) // first request

	if v, loaded := m.LoadOrStore(indexName, slot); loaded {
		// Another goroutine created it first -- use theirs.
		existing := v.(*indexSlot)
		existing.requestDecay.increment(c.decayFactor)
		existing.idleSince.Store(0)
		return existing
	}

	return slot
}

// effectiveFanOut returns the current fan-out for a slot, clamped to
// configuration bounds and available node count.
//
// Fan-out is the number of candidate nodes considered for a request, NOT the
// number of nodes that receive the request (always 1). It is driven by:
//
//   - shardFloor: number of shard-hosting nodes for this index. Ensures the
//     candidate set covers all nodes with data for well-designed indexes.
//     Capped by maxFanOut to prevent pathologically sharded indexes from
//     inflating the candidate set to the entire cluster.
//   - rateFanOut: request-volume-driven growth via the decay counter. Under
//     sustained load, fan-out grows beyond shardFloor to distribute coordinator
//     load. Also capped by maxFanOut.
//   - minFanOut: absolute floor (default 1).
//
// The server handles shard-level scatter/gather internally; the routing choice
// determines coordinator consistency, cache warmth, and RTT.
func (c *indexSlotCache) effectiveFanOut(slot *indexSlot, indexName string, activeNodeCount int) int {
	// Check for per-index override first.
	if override, ok := c.overrides[indexName]; ok {
		return clampFanOut(override, activeNodeCount)
	}

	// Derive fan-out from request volume.
	rateFanOut := int(slot.requestDecay.load()/c.fanOutPerReq) + 1

	// Floor from shard placement: ensures the candidate set covers all
	// shard-hosting nodes for well-designed indexes. For pathological indexes
	// (shards on every node), maxFanOut caps the damage.
	shardFloor := int(slot.shardNodeCount.Load())

	fanOut := max(c.minFanOut, shardFloor, rateFanOut)

	if c.maxFanOut > 0 && fanOut > c.maxFanOut {
		fanOut = c.maxFanOut
	}

	return clampFanOut(fanOut, activeNodeCount)
}

func clampFanOut(fanOut, activeNodeCount int) int {
	if activeNodeCount > 0 && fanOut > activeNodeCount {
		return activeNodeCount
	}
	if fanOut < 1 {
		return 1
	}
	return fanOut
}

// updateFromDiscovery refreshes all cache entries with new shard placement
// data and evicts idle entries. Called during the discovery cycle.
//
// shardPlacement maps index names to their shard placement data (node names,
// primary/replica counts). May be nil if /_cat/shards failed (existing data
// preserved).
//
// After the walk, if the live entry count has dropped to 50% of its
// high-water mark, the underlying [sync.Map] is replaced with a fresh
// instance containing only the surviving entries. This reclaims internal
// hash table memory that [sync.Map] retains after deletes.
func (c *indexSlotCache) updateFromDiscovery(shardPlacement map[string]*indexShardPlacement, activeNodeCount int, now time.Time) {
	nowNano := now.UnixNano()

	m := c.entries.Load()
	var liveCount int64

	m.Range(func(key, value any) bool {
		indexName := key.(string)
		slot := value.(*indexSlot)

		// Update shard placement if data is available.
		if shardPlacement != nil {
			if placement, ok := shardPlacement[indexName]; ok {
				nodes := placement.Nodes
				slot.shardNodeCount.Store(int32(len(nodes))) //nolint:gosec // Node count bounded by cluster size.
				slot.shardNodeNames.Store(&nodes)
				slot.mergeShardMap(placement)
			} else {
				// Index not in shard data -- may have been deleted or
				// the /_cat/shards response was truncated/stale. Clear
				// node counts (safe -- they only affect fan-out floor)
				// but preserve the shard map so that in-flight requests
				// can still use shard-exact routing. The idle eviction
				// below will eventually reclaim the entire slot if the
				// index is truly gone.
				slot.shardNodeCount.Store(0)
				slot.shardNodeNames.Store(nil)
			}
		}

		// Decay the request counter (one decay step per discovery cycle).
		slot.requestDecay.decay(c.decayFactor)

		// Recompute fan-out.
		newFanOut := c.effectiveFanOut(slot, indexName, activeNodeCount)
		slot.fanOut.Store(int32(newFanOut)) //nolint:gosec // Fan-out clamped by effectiveFanOut (max 32 default).

		// Idle eviction.
		counter := slot.requestDecay.load()
		if counter < 1.0 {
			// Effectively idle.
			idleSince := slot.idleSince.Load()
			if idleSince == 0 {
				// Mark as idle starting now.
				slot.idleSince.Store(nowNano)
			} else if nowNano-idleSince > c.idleEvictionTTL.Nanoseconds() {
				// Idle for too long -- evict.
				m.Delete(indexName)
				return true // continue Range; don't count as live
			}
		} else {
			// Still active -- clear idle marker.
			slot.idleSince.Store(0)
		}

		liveCount++
		return true
	})

	// Update high-water mark.
	hwm := c.highWaterMark.Load()
	if liveCount > hwm {
		c.highWaterMark.Store(liveCount)
	} else if hwm > 0 && liveCount <= hwm/2 {
		// Live count has contracted to 50% of peak. Replace the sync.Map
		// to reclaim internal hash table memory retained after deletes.
		c.compactEntries(m, liveCount)
	}
}

// mergeShardMap updates the cached per-shard placement map from a fresh
// /_cat/shards-derived placement.
//
// /_cat/shards rows for RELOCATING (source) and INITIALIZING (destination)
// shards are skipped during parsing, so a shard in flight between two nodes
// is absent from placement.ShardToNodes for the duration of the relocation.
// Replacing the cached map with a partial response would leave Shards[N] == nil
// for the in-flight shard and disable shard-exact routing for it.
//
// Carry forward entries the new response is missing: a shard in RELOCATING
// state still serves reads from the source node until the destination flips
// to STARTED, so the prior entry remains a valid routing target for the
// relocation window. New data wins where present; entries beyond the current
// primary count are dropped so a shrunk index doesn't retain unreachable
// high-numbered shards.
//
// The merge triggers whenever the prior map has any keys missing from the
// new placement, regardless of relative size. A naive `len(old) > len(new)`
// gate misses equal-length divergent maps (e.g. shard 3 RELOCATING out
// concurrent with shard 4 appearing after a split or shrink) and silently
// drops the in-flight entry.
//
// Empty placements are ignored to preserve the previous map across transient
// /_cat/shards failures (a stale map still routes correctly; a nil one
// disables shard-exact routing entirely).
//
// The CAS is retried on a bounded loop. The original implementation took the
// loser's data as "stale enough to discard" because both racers derive from
// near-simultaneous cat-shards responses, but a CAS loss can still drop a
// strictly fresher in-flight observation; cheap retries beat that risk.
func (slot *indexSlot) mergeShardMap(placement *indexShardPlacement) {
	if len(placement.ShardToNodes) == 0 {
		return
	}

	const maxRetries = 3
	for range maxRetries {
		old := slot.shardMap.Load()
		merged := placement.ShardToNodes

		if old != nil && carriesForward(old.Shards, placement.ShardToNodes) {
			merged = make(map[int]*shardNodes, len(old.Shards)+len(placement.ShardToNodes))
			maps.Copy(merged, old.Shards)
			maps.Copy(merged, placement.ShardToNodes)
		}

		// Prune entries beyond the current primary count only when the new
		// placement is complete, i.e. covers every primary shard. A partial
		// placement (relocation in flight excludes RELOCATING/INITIALIZING
		// rows from ShardToNodes) makes NumberOfPrimaryShards an undercount
		// and would prune the very entry the carry-forward just preserved.
		// On a real shrink, the new placement covers every shard at the new
		// (smaller) count, so the prune still fires when needed.
		isCompletePlacement := len(placement.ShardToNodes) >= placement.NumberOfPrimaryShards
		if placement.NumberOfPrimaryShards > 0 && isCompletePlacement && len(merged) > placement.NumberOfPrimaryShards {
			// Defensive copy when we're about to mutate a map we don't own
			// (i.e. placement.ShardToNodes was assigned directly and now needs
			// pruning after a shrink). Without this, the discovery code's
			// caller would see its placement mutated underfoot. The
			// carries-forward branch above produces a fresh map, so this
			// only matters when carriesForward returned false.
			if len(merged) == len(placement.ShardToNodes) {
				clone := make(map[int]*shardNodes, len(merged))
				maps.Copy(clone, merged)
				merged = clone
			}
			for k := range merged {
				if k >= placement.NumberOfPrimaryShards {
					delete(merged, k)
				}
			}
		}

		if slot.shardMap.CompareAndSwap(old, &indexShardMap{
			NumberOfPrimaryShards: placement.NumberOfPrimaryShards,
			RoutingNumShards:      placement.RoutingNumShards,
			Shards:                merged,
		}) {
			return
		}
	}
	if dl := loadDebugLogger(); dl != nil {
		dl.Logf("mergeShardMap: CAS exhausted after %d attempts (concurrent writer wins)\n", maxRetries)
	}
}

// carriesForward reports whether old contains any shard keys absent from
// next. When true, the merge must preserve those entries to keep
// shard-exact routing alive across relocation windows and same-length
// shard-set divergences (e.g. concurrent split/shrink).
func carriesForward(old, next map[int]*shardNodes) bool {
	for k := range old {
		if _, ok := next[k]; !ok {
			return true
		}
	}
	return false
}

// shardNodeNameSet returns the set of node names hosting shards for an index,
// or nil if unknown. Used by rendezvousTopK for shard-aware partitioning.
func (slot *indexSlot) shardNodeNameSet() map[string]struct{} {
	p := slot.shardNodeNames.Load()
	if p == nil {
		return nil
	}
	m := *p
	if len(m) == 0 {
		return nil
	}
	set := make(map[string]struct{}, len(m))
	for name := range m {
		set[name] = struct{}{}
	}
	return set
}

// shardNodeInfoFor returns the shard info for a specific node on this index,
// or nil if unknown. Used by calcConnScore for replica-preference scoring.
// The nodeName must be the node's human-readable name (matching /_cat/shards).
func (slot *indexSlot) shardNodeInfoFor(nodeName string) *shardNodeInfo {
	p := slot.shardNodeNames.Load()
	if p == nil {
		return nil
	}
	return (*p)[nodeName]
}

const (
	// miadMIHalfLife controls how fast the smoothed max bucket grows
	// toward a higher observed value. A 2-second half-life closes 50%
	// of the gap in 2s, 87.5% in 6s. Fast growth ensures remote tiers
	// absorb load before local nodes brown out during demand spikes.
	miadMIHalfLife = 2.0 // seconds

	// miadADRate controls how fast the smoothed max bucket decreases
	// when the observed value drops. A low rate provides gradual
	// cooldown that avoids oscillation around the steady-state tier.
	miadADRate = 0.03 // buckets per second
)

// now returns the current time via the injected clock.
func (slot *indexSlot) now() time.Time {
	return slot.clock.Now()
}

// updateSmoothedMaxBucket updates the smoothed max RTT bucket for this
// index slot using MIAD (Multiplicative Increase, Additive Decrease).
//
// When the observed max grows (fan-out expanded to a higher tier), the
// smoothed value converges quickly via exponential increase (MI). When
// the observed max drops (fan-out contracted), the smoothed value steps
// down linearly (AD) to find steady-state without oscillation.
//
// Returns the new smoothed value. Safe for concurrent use (CAS loop).
func (slot *indexSlot) updateSmoothedMaxBucket(observed float64) float64 {
	miLambda := math.Ln2 / miadMIHalfLife

	for {
		oldBits := slot.smoothedMaxBucketBits.Load()
		oldNano := slot.smoothedMaxBucketNano.Load()

		oldVal := math.Float64frombits(oldBits)
		now := slot.now().UnixNano()
		dt := float64(now-oldNano) / 1e9

		if dt < 0 || oldNano == 0 {
			// First observation: snap to observed value.
			newVal := math.Max(1.0, observed)
			if slot.smoothedMaxBucketBits.CompareAndSwap(oldBits, math.Float64bits(newVal)) {
				slot.smoothedMaxBucketNano.Store(now)
				return newVal
			}
			continue
		}

		var newVal float64
		if observed >= oldVal {
			// Multiplicative increase: exponential convergence toward observed.
			//   smoothed = observed - (observed - old) * e^(-lambda*dt)
			newVal = observed - (observed-oldVal)*math.Exp(-miLambda*dt)
		} else {
			// Additive decrease: linear step-down toward observed, clamped.
			newVal = math.Max(oldVal-miadADRate*dt, observed)
		}

		if newVal < 1.0 {
			newVal = 1.0
		}

		if slot.smoothedMaxBucketBits.CompareAndSwap(oldBits, math.Float64bits(newVal)) {
			slot.smoothedMaxBucketNano.Store(now)
			return newVal
		}
	}
}

// loadSmoothedMaxBucket returns the current smoothed max bucket value.
// The value is not time-decayed on read; decay happens via MIAD updates
// in updateSmoothedMaxBucket during each Eval() call.
func (slot *indexSlot) loadSmoothedMaxBucket() float64 {
	return math.Float64frombits(slot.smoothedMaxBucketBits.Load())
}

// slotFor returns the existing slot for indexName, or nil if the index
// has no slot in the cache. Unlike getOrCreate, this does not create a
// slot or increment the request counter.
func (c *indexSlotCache) slotFor(indexName string) *indexSlot {
	m := c.entries.Load()
	if v, ok := m.Load(indexName); ok {
		return v.(*indexSlot)
	}
	return nil
}

// compactEntries replaces the current [sync.Map] with a fresh instance
// containing only the surviving entries from old. This reclaims internal
// hash table memory that [sync.Map] retains after [sync.Map.Delete] calls.
//
// Concurrency: a concurrent [getOrCreate] that wrote to the old map after
// the swap will lose that write. The slot is recreated on the next request
// for that index -- a brief reset of its decay counter, acceptable given
// compaction runs at most once per discovery cycle.
func (c *indexSlotCache) compactEntries(old *sync.Map, liveCount int64) {
	fresh := new(sync.Map)
	old.Range(func(key, value any) bool {
		fresh.Store(key, value)
		return true
	})
	c.entries.Store(fresh)
	c.highWaterMark.Store(liveCount)
}

// snapshot returns a point-in-time snapshot of all index slots and the
// effective configuration. Used by Metrics() for observability.
func (c *indexSlotCache) snapshot() RouterSnapshot {
	var indexes []IndexRouterState

	c.entries.Load().Range(func(key, value any) bool {
		indexName := key.(string)
		slot := value.(*indexSlot)

		state := IndexRouterState{
			Name:        indexName,
			FanOut:      int(slot.fanOut.Load()),
			ShardNodes:  int(slot.shardNodeCount.Load()),
			RequestRate: slot.requestDecay.load(),
		}

		if idleNano := slot.idleSince.Load(); idleNano != 0 {
			t := time.Unix(0, idleNano)
			state.IdleSince = &t
		}

		indexes = append(indexes, state)
		return true
	})

	// Sort by name for deterministic output.
	sortIndexRouterStates(indexes)

	return RouterSnapshot{
		Indexes: indexes,
		Config: RouterSnapshotConfig{
			MinFanOut:       c.minFanOut,
			MaxFanOut:       c.maxFanOut,
			DecayFactor:     c.decayFactor,
			FanOutPerReq:    c.fanOutPerReq,
			IdleEvictionTTL: c.idleEvictionTTL.String(),
		},
	}
}
