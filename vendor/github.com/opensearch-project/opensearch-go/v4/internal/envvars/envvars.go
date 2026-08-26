// SPDX-License-Identifier: Apache-2.0
//
// The OpenSearch Contributors require contributions made to
// this file be licensed under the Apache-2.0 license or a
// compatible open source license.

// Package envvars holds the canonical names of OPENSEARCH_GO_* environment
// variables consumed by both the opensearch and opensearchtransport packages.
// Centralizing the names here avoids two-way duplication across the import
// boundary (opensearch depends on opensearchtransport, but the transport
// must not import opensearch).
package envvars

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"
)

// OpenSearchURL is the comma-separated seed-URL list used by [opensearch.NewClient]
// when no Addresses are configured programmatically.
const OpenSearchURL = "OPENSEARCH_URL"

// Router controls whether the DefaultRouter is created automatically when
// no programmatic Config.Router is set, and whether opensearch.NewClient
// inherits on-start node discovery. Parsed as strconv.ParseBool.
const Router = "OPENSEARCH_GO_ROUTER"

// ShardCost configures shard cost multipliers for connection scoring.
// See [opensearchtransport.WithShardCosts] for the format.
const ShardCost = "OPENSEARCH_GO_SHARD_COST"

// RoutingConfig is a comma-separated bitfield/key-value spec controlling
// request-time routing features (e.g. -shard_exact, -adaptive_mcsr).
const RoutingConfig = "OPENSEARCH_GO_ROUTING_CONFIG"

// DiscoveryConfig is a comma-separated bitfield spec controlling which
// discovery API calls run (e.g. -cat_shards, -node_stats).
const DiscoveryConfig = "OPENSEARCH_GO_DISCOVERY_CONFIG"

// Fallback toggles seed-URL fallback when all router pools are exhausted.
const Fallback = "OPENSEARCH_GO_FALLBACK"

// ShardRequests configures adaptive max_concurrent_shard_requests bounds.
// Format: "true"/"false" or "min:max" (e.g. "10:512").
const ShardRequests = "OPENSEARCH_GO_SHARD_REQUESTS"

// RequestTimeout overrides the per-attempt HTTP round-trip timeout.
const RequestTimeout = "OPENSEARCH_GO_REQUEST_TIMEOUT"

// NodeStatsInterval overrides the node stats polling interval.
const NodeStatsInterval = "OPENSEARCH_GO_NODE_STATS_INTERVAL"

// OverloadedHeapThreshold overrides the JVM heap percentage threshold for
// marking a node overloaded.
const OverloadedHeapThreshold = "OPENSEARCH_GO_OVERLOADED_HEAP_THRESHOLD"

// OverloadedBreakerRatio overrides the circuit-breaker ratio threshold for
// marking a node overloaded.
const OverloadedBreakerRatio = "OPENSEARCH_GO_OVERLOADED_BREAKER_RATIO"

// ActiveListCap overrides the per-pool active-connection cap.
const ActiveListCap = "OPENSEARCH_GO_ACTIVE_LIST_CAP"

// VerifyDeadAfter is the duration a dead connection can be blindly resurrected
// for use as a zombie connection when no other healthy connections are
// available. Past this window the discovery loop stops treating the node as a
// zombie candidate until it health-checks clean again. Parsed by
// [ParseVerifyDeadAfter]: a boolean true selects [VerifyDeadAfterDefault];
// false disables the expiry (a proven connection stays a zombie candidate
// indefinitely); any other value is a time.ParseDuration string (e.g. "10m").
const VerifyDeadAfter = "OPENSEARCH_GO_VERIFY_DEAD_AFTER"

// VerifyDeadAfterDefault is the zombie-resurrection window used when
// VerifyDeadAfter is unset or set to a truthy boolean. 15m matches the AWS
// Lambda max timeout, so a proven connection is not force-expired mid-invocation.
const VerifyDeadAfterDefault = 15 * time.Minute

// StandbyRotationInterval overrides the interval between standby rotation cycles.
const StandbyRotationInterval = "OPENSEARCH_GO_STANDBY_ROTATION_INTERVAL"

// StandbyRotationCount overrides the number of standby rotations performed
// per cycle.
const StandbyRotationCount = "OPENSEARCH_GO_STANDBY_ROTATION_COUNT"

// StandbyPromotionChecks overrides the number of consecutive successful
// health checks required to promote a standby connection to active.
const StandbyPromotionChecks = "OPENSEARCH_GO_STANDBY_PROMOTION_CHECKS"

// Debug enables verbose internal logging via [strconv.ParseBool].
const Debug = "OPENSEARCH_GO_DEBUG"

// ErrorMask is a comma-separated bitfield spec controlling which categories
// of partial-failure errors API methods mask (ignore) instead of returning
// as typed Go errors. Tokens are the lowercase snake_case wrapper-schema
// names (`bulk_items`, `search_shards`, `write_shards`, `broadcast_shards`,
// `node_failures`, `bulk_by_scroll_failures`, `task_failures`,
// `multi_search_items`, `multi_doc_items`, `snapshot_create_shard_failures`,
// `snapshot_get_shard_failures`, `simulate_doc_failures`,
// `rank_eval_failures`, `ingestion_shard_failures`, `pit_node_failures`)
// plus `all` and `none`/`empty`/`unknown` (aliases for zero). Each token
// may be prefixed with `+` (set/mask) or `-` (clear/unmask); bare tokens
// are treated as `+`. Tokens are applied left-to-right starting from the
// programmatic Config.Errors value. Unrecognized tokens are silently
// dropped (forward-compatible) and reported via the debug logger when
// OPENSEARCH_GO_DEBUG=true.
const ErrorMask = "OPENSEARCH_GO_ERROR_MASK"

// Truthy reports whether the named environment variable is set to a
// strconv.ParseBool-truthy value. Empty, unset, unparseable, or falsy
// values all return false.
func Truthy(name string) bool {
	val, ok := os.LookupEnv(name)
	if !ok || val == "" {
		return false
	}
	b, err := strconv.ParseBool(val)
	return err == nil && b
}

// Falsy reports whether the named environment variable is set to a
// strconv.ParseBool-falsy value. Empty, unset, unparseable, or truthy
// values all return false. Use this when "explicitly opted out" needs
// to be distinguished from "unset" (where Truthy alone collapses both
// into false).
func Falsy(name string) bool {
	val, ok := os.LookupEnv(name)
	if !ok || val == "" {
		return false
	}
	b, err := strconv.ParseBool(val)
	return err == nil && !b
}

// ErrInvalidVerifyDeadAfter is returned by ParseVerifyDeadAfter (wrapped, match
// with errors.Is) when the value is neither a boolean nor a parseable duration.
var ErrInvalidVerifyDeadAfter = errors.New("envvars: invalid verify-dead-after value")

// ParseVerifyDeadAfter interprets a VerifyDeadAfter value. A boolean (per
// strconv.ParseBool) is handled first: true selects VerifyDeadAfterDefault,
// false disables the expiry. Otherwise the value is parsed as a duration by
// time.ParseDuration ("10m", "90s") or as a bare number of seconds ("600").
// A zero or negative result disables the expiry. The returned duration is the
// resolved window, or 0 when disabled. The error is non-nil (wrapping
// ErrInvalidVerifyDeadAfter) only when the value cannot be parsed at all; an
// empty value is treated as "disabled" with a nil error.
func ParseVerifyDeadAfter(val string) (time.Duration, error) {
	if val == "" {
		return 0, nil
	}
	if b, err := strconv.ParseBool(val); err == nil {
		if b {
			return VerifyDeadAfterDefault, nil
		}
		return 0, nil
	}
	if d, err := time.ParseDuration(val); err == nil {
		if d <= 0 {
			return 0, nil
		}
		return d, nil
	}
	if secs, err := strconv.ParseFloat(val, 64); err == nil {
		d := time.Duration(secs * float64(time.Second))
		if d <= 0 {
			return 0, nil
		}
		return d, nil
	}
	return 0, fmt.Errorf("%w: %q", ErrInvalidVerifyDeadAfter, val)
}
