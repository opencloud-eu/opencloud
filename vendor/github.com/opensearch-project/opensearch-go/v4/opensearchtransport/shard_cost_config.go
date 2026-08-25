// SPDX-License-Identifier: Apache-2.0
//
// The OpenSearch Contributors require contributions made to
// this file be licensed under the Apache-2.0 license or a
// compatible open source license.

package opensearchtransport

import (
	"math"
	"slices"
	"strconv"
	"strings"

	"github.com/opensearch-project/opensearch-go/v4/internal/envvars"
)

// shardCostValidKeysHint is appended to "unknown key" messages to enumerate
// the parser's accepted key set in a consistent order.
const shardCostValidKeysHint = "valid keys: r:base, r:amplify, r:exponent, unknown, " +
	"relocating, initializing, replica, write_primary, write_replica"

// Shard cost config error message constants.
const (
	shardCostMessageMissing    = "missing value"
	shardCostMessageInvalid    = "invalid value"
	shardCostMessageUnknownKey = "unknown key"
	shardCostMessageNotFinite  = "value must be a finite non-negative number"
)

// Shard cost config key constants for the dynamic read curve.
const (
	shardCostKeyBase     = "r:base"
	shardCostKeyAmplify  = "r:amplify"
	shardCostKeyExponent = "r:exponent"
)

// Shard cost config key constants for static overrides.
const (
	shardCostKeyUnknown      = "unknown"
	shardCostKeyRelocating   = "relocating"
	shardCostKeyInitializing = "initializing"
	shardCostKeyReplica      = "replica"
	shardCostKeyWritePrimary = "write_primary"
	shardCostKeyWriteReplica = "write_replica"
)

// ShardCostConfigError is returned when [parseShardCostConfig] encounters
// an invalid shard cost configuration string. Use errors.As to recover the
// type; use Error() for the human-readable message and errors.Unwrap (or
// errors.Is) to reach the underlying parse error if any.
type ShardCostConfigError struct {
	// Key is the config key that caused the error, when applicable
	// (empty for bare-numeric specs and similar context-free failures).
	Key string
	// Message is the human-readable description of the failure.
	Message string
	// Err is the underlying error, if any (e.g. strconv.ErrSyntax).
	Err error
}

func (e *ShardCostConfigError) Error() string {
	msg := "shard cost config: " + e.Message
	if e.Key != "" {
		msg += " for key " + strconv.Quote(e.Key)
	}
	if e.Err != nil {
		msg += ": " + e.Err.Error()
	}
	return msg
}

func (e *ShardCostConfigError) Unwrap() error { return e.Err }

// envShardCost controls the dynamic shard cost curve parameters used for
// connection scoring. When a read operation targets a primary shard, the
// effective cost increases with the node's write-pool utilization:
//
//	effectivePrimaryCost = base + amplify * (writeUtil ^ exponent)
//
// where writeUtil = min(write_inFlight / write_cwnd, 1.0). On idle nodes
// the primary is slightly preferred (base < 1.0); as write load ramps up,
// reads shed progressively to replica-hosting nodes.
//
// Format: bare float (sets r:base) or comma-separated key=value items.
//
// Dynamic keys (read cost curve, prefixed with "r:"):
//
//	r:base      - primary read cost at write-idle (default 0.95)
//	r:amplify   - amplification factor (default 2.0)
//	r:exponent  - curve shape (default 2.0 = quadratic)
//
// Static keys (shard state costs):
//
//	unknown       - cost when no shard data available (both tables)
//	relocating    - cost for relocating shards (both tables)
//	initializing  - cost for initializing shards (both tables)
//	replica       - read-table replica cost
//	write_primary - write-table primary cost
//	write_replica - write-table replica cost
//
// Examples:
//
//	OPENSEARCH_GO_SHARD_COST=1.0                                     # r:base=1.0 (equal with replicas at idle)
//	OPENSEARCH_GO_SHARD_COST=r:base=0.9,r:amplify=2.5,r:exponent=1.5 # custom read curve
//	OPENSEARCH_GO_SHARD_COST=r:base=1.0                              # equal at idle
const envShardCost = envvars.ShardCost

const (
	// counterFloor is the minimum value used for the decay counter
	// in score calculations. Prevents division-by-zero-like effects when a
	// node has received no recent traffic.
	counterFloor = 1.0

	// Shard cost multipliers used in the [shardCostMultiplier] tables and
	// [warmupPenaltyMax]. Lower value = preferred node.
	//
	// Read and write tables use different progressions: reads have a dynamic
	// primary cost (see [newReadScoreFunc]) with a mild static fallback,
	// while writes penalize non-primary nodes heavily since writes must
	// always land on the primary shard.

	// Read cost constants.
	costReadReplica      = 1.0  // preferred -- lock-free Lucene snapshot
	costReadPrimary      = 2.0  // static fallback; overridden by dynamic scoring
	costReadRelocating   = 8.0  // shard moving, may require proxy hop
	costReadInitializing = 16.0 // shard not yet ready to serve

	// Write cost constants. Writes always target the primary shard;
	// any other node must coordinate to the primary, which is avoidable work.
	costWritePrimary      = 1.0  // preferred -- write lands directly on primary
	costWriteReplica      = 8.0  // must coordinate to primary -- unnecessary proxy
	costWriteRelocating   = 16.0 // shard moving, coordination target uncertain
	costWriteInitializing = 24.0 // shard not ready, coordination may fail

	// Shared across both tables.
	costUnknown = 32.0 // no shard data, heavily penalized
)

// shardCostIndex identifies a shard state position in a [shardCostMultiplier].
type shardCostIndex int

const (
	// shardCostUnknown is the zero value: no shard data available.
	// A zero-initialized [shardCostMultiplier] produces 0.0 for unknown,
	// so tables must be explicitly constructed.
	shardCostUnknown shardCostIndex = iota

	// shardCostReplica: node hosts only replica shards for this index.
	shardCostReplica

	// shardCostPrimary: node hosts only primary shards for this index.
	shardCostPrimary

	// shardCostInitializing: node has initializing shards (reserved for
	// future use; discovery currently filters to STARTED shards only).
	shardCostInitializing

	// shardCostRelocating: node has relocating shards (reserved for
	// future use; discovery currently filters to STARTED shards only).
	shardCostRelocating
)

// shardCostMultiplier holds per-shard-state score multipliers used in
// [calcConnDefaultScore] and [connScoreFunc] implementations. The appropriate
// table is selected at policy construction time based on whether the route
// handles reads or writes.
//
// Lower multiplier = preferred node. Index via [shardCostIndex] constants.
type shardCostMultiplier [5]float64

// shardCostForReads provides static shard costs for read operations.
// For primary shards, the [connScoreFunc] returned by [newReadScoreFunc]
// overrides this with a dynamic cost derived from the node's write-pool
// utilization. The static primary cost serves as a fallback when no
// scoring function is configured.
//
//nolint:gochecknoglobals // Package-level constant table.
var shardCostForReads = shardCostMultiplier{
	shardCostUnknown:      costUnknown,          // no data, heavily penalized
	shardCostReplica:      costReadReplica,      // preferred for reads
	shardCostPrimary:      costReadPrimary,      // static fallback; overridden by dynamic cost
	shardCostInitializing: costReadInitializing, // shard not yet ready
	shardCostRelocating:   costReadRelocating,   // shard moving, may proxy
}

// shardCostForWrites prefers primary-hosting nodes. Writes always go to
// the primary shard; routing to a replica-only node forces unnecessary
// coordinator proxy work. Non-primary costs are set high to strongly
// prefer direct-to-primary routing.
//
//nolint:gochecknoglobals // Package-level constant table.
var shardCostForWrites = shardCostMultiplier{
	shardCostUnknown:      costUnknown,           // no data, heavily penalized
	shardCostReplica:      costWriteReplica,      // must coordinate to primary
	shardCostPrimary:      costWritePrimary,      // preferred -- write lands directly
	shardCostInitializing: costWriteInitializing, // shard not ready, coordination may fail
	shardCostRelocating:   costWriteRelocating,   // shard moving, coordination target uncertain
}

// forNode returns the shard cost multiplier for a node based on its shard
// composition for the target index.
//
// The lookup is categorical: if the node hosts the preferred shard type
// (as encoded by the table), it gets the preferred cost. Mixed nodes that
// host both primaries and replicas get the best (lowest) cost since they
// can serve both reads and writes locally. Load-based differentiation
// between nodes is handled by the utilization ratio (inFlight+1)/cwnd, not by this
// multiplier.
func (m *shardCostMultiplier) forNode(node *shardNodeInfo) float64 {
	if node == nil {
		return m[shardCostUnknown]
	}
	total := node.Primaries + node.Replicas
	if total == 0 {
		return m[shardCostUnknown]
	}
	if node.Primaries == 0 {
		return m[shardCostReplica]
	}
	if node.Replicas == 0 {
		return m[shardCostPrimary]
	}
	// Mixed node: hosts both primaries and replicas. It can serve reads
	// from replicas and writes to primaries locally. Use the better cost;
	// the CPU counter differentiates actual load between mixed nodes.
	return min(m[shardCostReplica], m[shardCostPrimary])
}

// forShard returns the shard cost multiplier for a connection based on its
// role for a specific shard. Unlike [forNode] (which uses aggregate per-node
// counts across all shards of an index), this uses the per-shard placement
// data from [shardNodes] to determine the exact role --primary or replica
// --for the target shard resolved via murmur3 hashing.
//
// Used by the shard lookup path when the target shard number is known.
func (m *shardCostMultiplier) forShard(shard *shardNodes, connName string) float64 {
	if shard == nil {
		return m[shardCostUnknown]
	}
	if shard.Primary == connName {
		return m[shardCostPrimary]
	}
	if slices.Contains(shard.Replicas, connName) {
		return m[shardCostReplica]
	}
	return m[shardCostUnknown]
}

// calcShardPrimaryPct returns the primary percentage for a connection's
// role on a specific shard: 1.0 if it is the primary, 0.0 otherwise.
func calcShardPrimaryPct(shard *shardNodes, connName string) float64 {
	if shard != nil && shard.Primary == connName {
		return 1.0
	}
	return 0.0
}

// calcNodePrimaryPct returns the percentage of shards on this node that
// are in the primary state versus other shard states. A node with
// 3 primaries and 7 replicas returns 0.3. This drives proportional
// blending in the dynamic read cost curve: nodes with more primaries
// see more of the dynamic effect.
func calcNodePrimaryPct(node *shardNodeInfo) float64 {
	if node == nil {
		return 0.0
	}
	total := node.Primaries + node.Replicas
	if total == 0 {
		return 0.0
	}
	return float64(node.Primaries) / float64(total)
}

// Default read cost curve parameters.
const (
	defaultReadCurveBase     = 0.95 // slightly prefer primary at idle (< replica's 1.0)
	defaultReadCurveAmplify  = 2.0  // amplification factor
	defaultReadCurveExponent = 2.0  // quadratic curve
)

// connScoreFunc computes a routing score for a connection given its
// static shard cost and primary percentage. The primaryPct ranges
// from 0.0 (pure replica) to 1.0 (pure primary) and drives blending
// between static and dynamic cost. Implementations may incorporate
// cross-pool utilization, custom cost curves, or any other signal.
// Returns the composite score (lower = preferred).
//
// When nil in [connScoreSelect], scoring falls back to the static
// formula: rtt * (inFlight+1)/cwnd * shardCost.
type connScoreFunc func(conn *Connection, shardCost float64, primaryPct float64, poolName string, poolInfoReady bool) float64

// newReadScoreFunc returns a [connScoreFunc] that dynamically adjusts
// primary shard cost based on write-pool utilization. The effective cost
// is blended by primaryPct:
//
//	dynamicCost   = base + amplify * (writeUtil ^ exponent)
//	effectiveCost = (1 - primaryPct) * shardCost + primaryPct * dynamicCost
//
// where writeUtil = min(write_inFlight / write_cwnd, 1.0).
//
// For shard-exact routing (primaryPct = 0.0 or 1.0), this is a
// clean switch. For node-level routing with mixed shards
// (e.g., primaryPct = 0.6), the cost blends proportionally.
//
// At idle (writeUtil=0), a pure primary costs base (0.95 by default),
// which is less than the replica's static 1.0 -- the primary is preferred.
// As writes ramp up, the dynamic cost increases and reads progressively
// shift to replica-hosting nodes.
func newReadScoreFunc(base, amplify, exponent float64) connScoreFunc {
	return func(conn *Connection, shardCost float64, primaryPct float64, poolName string, poolInfoReady bool) float64 {
		rtt, utilization, overloaded := connBaseScore(conn, poolName, poolInfoReady)
		if overloaded {
			return math.MaxFloat64
		}
		cost := shardCost
		if primaryPct > 0 {
			writePC := conn.pools.getForScoring(poolWrite)
			writeCwnd := float64(writePC.cwnd.Load())
			if writeCwnd < 1 {
				writeCwnd = 1
			}
			writeUtil := float64(writePC.inFlight.Load()) / writeCwnd
			writeUtil = min(max(writeUtil, 0), 1.0)
			dynamicCost := base + amplify*math.Pow(writeUtil, exponent)
			cost = (1-primaryPct)*shardCost + primaryPct*dynamicCost
		}
		return rtt * utilization * cost
	}
}

// connBaseScore computes the common components of connection scoring:
// RTT bucket and utilization ratio. Returns overloaded=true when the
// target pool has been marked as overloaded (the caller should return
// [math.MaxFloat64]).
func connBaseScore(conn *Connection, poolName string, poolInfoReady bool) (float64, float64, bool) {
	if poolName != "" && conn.isPoolOverloaded(poolName) {
		return 0, 0, true
	}
	rtt := float64(conn.rttRing.medianBucket())
	cwnd := float64(conn.loadCwnd(poolName, poolInfoReady))
	inFlight := float64(conn.loadInFlight(poolName))
	utilization := (inFlight + 1.0) / cwnd
	return rtt, utilization, false
}

// calcConnDefaultScore scores a connection using the static shard cost
// formula: rtt × utilization × shardCost. This is the default scoring
// path used for write operations and as the fallback when no
// [connScoreFunc] is configured (scoreFunc == nil in [connScoreSelect]).
func calcConnDefaultScore(conn *Connection, shardCost float64, poolName string, poolInfoReady bool) float64 {
	rtt, utilization, overloaded := connBaseScore(conn, poolName, poolInfoReady)
	if overloaded {
		return math.MaxFloat64
	}
	return rtt * utilization * shardCost
}

// shardCostConfig holds the result of parsing a shard cost configuration
// string. It combines modified static cost tables with an optional
// dynamic scoring function for read operations.
type shardCostConfig struct {
	reads     shardCostMultiplier
	writes    shardCostMultiplier
	scoreFunc connScoreFunc // non-nil: dynamic read scoring via newReadScoreFunc
}

// parseShardCostConfig parses a shard cost configuration string into
// static cost tables and an optional dynamic scoring function.
//
// Format: bare float (sets r:base) or comma-separated key=value items.
//
// Dynamic keys control the read-primary cost curve (prefixed with "r:"):
//
//	r:base      - primary read cost at write-idle (default 0.95)
//	r:amplify   - amplification factor (default 2.0)
//	r:exponent  - curve shape (default 2.0 = quadratic)
//
// Static keys override shard state costs in the lookup tables:
//
//	unknown       - cost when no shard data available (both tables)
//	relocating    - cost for relocating shards (both tables)
//	initializing  - cost for initializing shards (both tables)
//	replica       - read-table replica cost (baseline for dynamic primary)
//	write_primary - write-table primary cost
//	write_replica - write-table replica cost
//
// Any static value <= 0 is replaced by the compile-time default for
// that slot.
func parseShardCostConfig(spec string) (*shardCostConfig, error) {
	curveBase := defaultReadCurveBase
	curveAmplify := defaultReadCurveAmplify
	curveExponent := defaultReadCurveExponent

	cfg := shardCostConfig{
		reads:  shardCostForReads,
		writes: shardCostForWrites,
	}

	spec = strings.TrimSpace(spec)
	if spec == "" {
		cfg.scoreFunc = newReadScoreFunc(curveBase, curveAmplify, curveExponent)
		return &cfg, nil
	}

	// Try bare numeric first: sets r:base (primary read cost at idle).
	if v, parseErr := strconv.ParseFloat(spec, 64); parseErr == nil {
		if v <= 0 || math.IsInf(v, 0) || math.IsNaN(v) {
			return nil, &ShardCostConfigError{Message: shardCostMessageNotFinite}
		}
		curveBase = v
		cfg.scoreFunc = newReadScoreFunc(curveBase, curveAmplify, curveExponent)
		return &cfg, nil
	}

	// Key=value parsing.
	for item := range strings.SplitSeq(spec, ",") {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}

		key, valStr, hasEq := strings.Cut(item, "=")
		if !hasEq {
			return nil, &ShardCostConfigError{Key: key, Message: shardCostMessageMissing}
		}

		v, parseErr := strconv.ParseFloat(valStr, 64)
		if parseErr != nil {
			return nil, &ShardCostConfigError{Key: key, Message: shardCostMessageInvalid, Err: parseErr}
		}

		switch key {
		case shardCostKeyBase, shardCostKeyAmplify, shardCostKeyExponent:
			// r:amplify=0 disables dynamic amplification (static-only mode).
			// r:exponent=0 holds dynamic cost at max regardless of utilization.
			// r:base must be strictly positive: r:base=0 makes primaries free at
			// idle, and the bare-numeric form (which sets r:base) rejects v<=0,
			// so the key=value form must match.
			if math.IsNaN(v) || math.IsInf(v, 0) || v < 0 || (key == shardCostKeyBase && v == 0) {
				return nil, &ShardCostConfigError{
					Key:     key,
					Message: shardCostMessageNotFinite,
				}
			}
			switch key {
			case shardCostKeyBase:
				curveBase = v
			case shardCostKeyAmplify:
				curveAmplify = v
			case shardCostKeyExponent:
				curveExponent = v
			}

		// Static costs: shared (both tables).
		case shardCostKeyUnknown:
			if math.IsNaN(v) || math.IsInf(v, 0) {
				return nil, &ShardCostConfigError{Key: key, Message: shardCostMessageNotFinite}
			}
			cfg.reads[shardCostUnknown] = v
			cfg.writes[shardCostUnknown] = v
		case shardCostKeyRelocating:
			if math.IsNaN(v) || math.IsInf(v, 0) {
				return nil, &ShardCostConfigError{Key: key, Message: shardCostMessageNotFinite}
			}
			cfg.reads[shardCostRelocating] = v
			cfg.writes[shardCostRelocating] = v
		case shardCostKeyInitializing:
			if math.IsNaN(v) || math.IsInf(v, 0) {
				return nil, &ShardCostConfigError{Key: key, Message: shardCostMessageNotFinite}
			}
			cfg.reads[shardCostInitializing] = v
			cfg.writes[shardCostInitializing] = v

		// Static costs: table-specific.
		case shardCostKeyReplica:
			if math.IsNaN(v) || math.IsInf(v, 0) {
				return nil, &ShardCostConfigError{Key: key, Message: shardCostMessageNotFinite}
			}
			cfg.reads[shardCostReplica] = v
		case shardCostKeyWritePrimary:
			if math.IsNaN(v) || math.IsInf(v, 0) {
				return nil, &ShardCostConfigError{Key: key, Message: shardCostMessageNotFinite}
			}
			cfg.writes[shardCostPrimary] = v
		case shardCostKeyWriteReplica:
			if math.IsNaN(v) || math.IsInf(v, 0) {
				return nil, &ShardCostConfigError{Key: key, Message: shardCostMessageNotFinite}
			}
			cfg.writes[shardCostReplica] = v

		default:
			return nil, &ShardCostConfigError{
				Key:     key,
				Message: shardCostMessageUnknownKey + ": " + shardCostValidKeysHint,
			}
		}
	}

	// Clamp: replace any static value <= 0 with the compile-time default.
	for i := range cfg.reads {
		if cfg.reads[i] <= 0 {
			cfg.reads[i] = shardCostForReads[i]
		}
	}
	for i := range cfg.writes {
		if cfg.writes[i] <= 0 {
			cfg.writes[i] = shardCostForWrites[i]
		}
	}

	cfg.scoreFunc = newReadScoreFunc(curveBase, curveAmplify, curveExponent)
	return &cfg, nil
}
