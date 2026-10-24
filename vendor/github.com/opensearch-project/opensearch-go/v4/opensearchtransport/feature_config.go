// SPDX-License-Identifier: Apache-2.0
//
// The OpenSearch Contributors require contributions made to
// this file be licensed under the Apache-2.0 license or a
// compatible open source license.

package opensearchtransport

import (
	"net/url"
	"strconv"
	"strings"

	"github.com/opensearch-project/opensearch-go/v4/internal/envvars"
)

// Environment variable names for feature configuration.
const (
	// envRoutingConfig controls request-time routing behavior.
	// Format: comma-separated items. +/- prefixed items toggle bitfield
	// flags; other items are key=value pairs (URL-encoded).
	//
	// Bitfield flags:
	//   -shard_exact   Disable murmur3 shard-exact routing
	//   +shard_exact   Explicitly re-enable (overrides programmatic disable)
	//
	// Example: OPENSEARCH_GO_ROUTING_CONFIG=-shard_exact
	envRoutingConfig = envvars.RoutingConfig

	// envFallbackConfig controls whether the client falls back to seed
	// URLs when all router policies and connection pools are exhausted.
	// Parsed as strconv.ParseBool. Default: true (fallback enabled).
	//
	// Example: OPENSEARCH_GO_FALLBACK=false
	envFallbackConfig = envvars.Fallback

	// envDiscoveryConfig controls which server calls are made during
	// the discovery cycle.
	//
	// Bitfield flags:
	//   -cat_shards          Skip GET /_cat/shards
	//   -routing_num_shards  Skip GET /_cluster/state/metadata
	//   -cluster_health      Skip GET /_cluster/health?local=true
	//   -node_stats          Skip GET /_nodes/_local/stats
	//
	// Example: OPENSEARCH_GO_DISCOVERY_CONFIG=-routing_num_shards,-node_stats
	envDiscoveryConfig = envvars.DiscoveryConfig

	// envShardRequests controls adaptive max_concurrent_shard_requests.
	// Format: bool | min:max
	//
	// Boolean values (parsed via strconv.ParseBool):
	//   "true"  / "1" -- enable with default min (5) and max (256)
	//   "false" / "0" -- disable entirely
	//
	// Numeric min:max pairs (either may be omitted for defaults):
	//   "10:512"  -- min=10, max=512
	//   "10:"     -- min=10, max=default (256)
	//   ":512"    -- min=default (5), max=512
	//
	// The env var takes precedence over programmatic RouterOption
	// configuration, following the same override convention as
	// OPENSEARCH_GO_ROUTING_CONFIG.
	//
	// Default (unset): enabled with min=5, max=256.
	//
	// Examples:
	//   OPENSEARCH_GO_SHARD_REQUESTS=false       # disable entirely
	//   OPENSEARCH_GO_SHARD_REQUESTS=10:512       # min=10, max=512
	//   OPENSEARCH_GO_SHARD_REQUESTS=:512         # default min, max=512
	envShardRequests = envvars.ShardRequests

	// envRouter controls whether the DefaultRouter is created automatically
	// when no programmatic Config.Router is set. Parsed as strconv.ParseBool.
	//
	// In v4 the router is off by default; set to "true" to enable scored,
	// role-aware request routing without code changes. Programmatic
	// Config.Router always takes precedence: if already set, this env var
	// is ignored.
	//
	// In v5 the default will flip: the router will be on by default, and
	// setting this to "false" will disable it.
	//
	// This variable is transitional: it will be removed in v6, where the
	// router is unconditionally on (disable individual policies via
	// OPENSEARCH_GO_POLICY_* instead).
	//
	// Example: OPENSEARCH_GO_ROUTER=true
	envRouter = envvars.Router
)

// routingFeatures is a bitfield where zero-value means all features are
// enabled. Each bit, when set, disables a specific feature.
type routingFeatures uint32

const (
	// routingSkipShardExact disables murmur3 shard-exact routing.
	// When set, calcSingleKeyCost and calcMultiKeyCost return nil and
	// shard-exact routing is bypassed.
	routingSkipShardExact routingFeatures = 1 << iota

	// routingSkipAdaptiveConcurrency disables adaptive
	// max_concurrent_shard_requests injection on search requests.
	// When set, the transport does not derive the shard fan-out limit
	// from the connection's search pool congestion window.
	routingSkipAdaptiveConcurrency
)

// shardExactEnabled returns true when murmur3 shard-exact routing is active.
func (f routingFeatures) shardExactEnabled() bool {
	return f&routingSkipShardExact == 0
}

// adaptiveConcurrencyEnabled returns true when adaptive
// max_concurrent_shard_requests injection is active.
func (f routingFeatures) adaptiveConcurrencyEnabled() bool {
	return f&routingSkipAdaptiveConcurrency == 0
}

// routingFlagNames maps flag name strings to their bit constants.
//
//nolint:gochecknoglobals // Package-level constant map for config parsing.
var routingFlagNames = map[string]routingFeatures{
	"shard_exact":   routingSkipShardExact,
	"adaptive_mcsr": routingSkipAdaptiveConcurrency,
}

// discoveryFeatures is a bitfield where zero-value means all discovery
// fetches are enabled. Each bit, when set, skips a specific server call.
type discoveryFeatures uint32

const (
	// discoverySkipCatShards disables GET /_cat/shards during discovery.
	// No shard placement data -> no shard-aware partitioning, no shard
	// cost multiplier, no shard-exact routing.
	discoverySkipCatShards discoveryFeatures = 1 << iota

	// discoverySkipRoutingNumShards disables
	// GET /_cluster/state/metadata/{indexes} during discovery.
	// No routing_num_shards -> shard-exact routing falls back to
	// rendezvous hashing.
	discoverySkipRoutingNumShards

	// discoverySkipClusterHealth disables
	// GET /_cluster/health?local=true probes.
	// Health checks fall back to baseline GET / only.
	discoverySkipClusterHealth

	// discoverySkipNodeStats disables
	// GET /_nodes/_local/stats/jvm,breaker,thread_pool polling.
	// No per-pool congestion tracking, no overload detection.
	discoverySkipNodeStats
)

// catShardsEnabled returns true when /_cat/shards fetching is active.
func (f discoveryFeatures) catShardsEnabled() bool {
	return f&discoverySkipCatShards == 0
}

// routingNumShardsEnabled returns true when /_cluster/state/metadata fetching is active.
func (f discoveryFeatures) routingNumShardsEnabled() bool {
	return f&discoverySkipRoutingNumShards == 0
}

// clusterHealthEnabled returns true when /_cluster/health probes are active.
func (f discoveryFeatures) clusterHealthEnabled() bool {
	return f&discoverySkipClusterHealth == 0
}

// nodeStatsEnabled returns true when /_nodes/_local/stats polling is active.
func (f discoveryFeatures) nodeStatsEnabled() bool {
	return f&discoverySkipNodeStats == 0
}

// discoveryFlagNames maps flag name strings to their bit constants.
//
//nolint:gochecknoglobals // Package-level constant map for config parsing.
var discoveryFlagNames = map[string]discoveryFeatures{
	"cat_shards":         discoverySkipCatShards,
	"routing_num_shards": discoverySkipRoutingNumShards,
	"cluster_health":     discoverySkipClusterHealth,
	"node_stats":         discoverySkipNodeStats,
}

// parseConfigItems splits a config string on comma, separating bitfield
// items (+/- prefixed) from key=value items. Bitfield items are returned
// as a map of flag name -> enabled. Key=value items are returned as
// url.Values to support multiple values per key.
//
// Items that start with '+' or '-' are bitfield toggles:
//
//	"-shard_exact" -> bits["shard_exact"] = false
//	"+shard_exact" -> bits["shard_exact"] = true
//
// All other items are treated as URL-encoded key=value pairs:
//
//	"key=value" -> kv["key"] = ["value"]
//
// Empty items (from trailing commas) are silently ignored.
func parseConfigItems(value string) (map[string]bool, url.Values) {
	if value == "" {
		return nil, nil
	}

	var bits map[string]bool
	var kv url.Values

	items := strings.SplitSeq(value, ",")
	for item := range items {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}

		// Bitfield toggle: +flag or -flag.
		if item[0] == '+' || item[0] == '-' {
			if bits == nil {
				bits = make(map[string]bool)
			}
			bits[item[1:]] = item[0] == '+'
			continue
		}

		// Key=value pair. Split on first '=' only.
		k, v, hasEq := strings.Cut(item, "=")
		if !hasEq {
			// Bare key without value --ignore (forward-compatible).
			continue
		}

		// URL-decode both key and value.
		decodedKey, err := url.QueryUnescape(k)
		if err != nil {
			decodedKey = k
		}
		decodedVal, err := url.QueryUnescape(v)
		if err != nil {
			decodedVal = v
		}

		if kv == nil {
			kv = make(url.Values)
		}
		kv.Add(decodedKey, decodedVal)
	}

	return bits, kv
}

// parseRoutingConfig applies parsed OPENSEARCH_GO_ROUTING_CONFIG items
// to the routing feature bitfield.
func parseRoutingConfig(value string) routingFeatures {
	bits, _ := parseConfigItems(value)

	var features routingFeatures

	// Apply bitfield flags.
	for name, enabled := range bits {
		bit, ok := routingFlagNames[name]
		if !ok {
			continue // Unknown flag --ignore for forward compatibility.
		}
		if enabled {
			features &^= bit // Clear the skip bit (re-enable).
		} else {
			features |= bit // Set the skip bit (disable).
		}
	}

	return features
}

// parseDiscoveryConfig applies parsed OPENSEARCH_GO_DISCOVERY_CONFIG items
// to the discovery feature bitfield.
func parseDiscoveryConfig(value string) discoveryFeatures {
	bits, _ := parseConfigItems(value)

	var features discoveryFeatures
	for name, enabled := range bits {
		bit, ok := discoveryFlagNames[name]
		if !ok {
			continue // Unknown flag --ignore for forward compatibility.
		}
		if enabled {
			features &^= bit // Clear the skip bit (re-enable).
		} else {
			features |= bit // Set the skip bit (disable).
		}
	}

	return features
}

// adaptiveConcurrencyConfig holds the min and max for adaptive
// max_concurrent_shard_requests. Zero values mean "use the compile-time
// default" (minVal=5, maxVal=256). Stored immutably after client init.
type adaptiveConcurrencyConfig struct {
	minVal int // 0 = use adaptiveConcurrencyMinDefault
	maxVal int // 0 = use adaptiveConcurrencyMaxDefault
}

const (
	// adaptiveConcurrencyMinDefault is the minimum value for adaptive
	// max_concurrent_shard_requests. Matches the OpenSearch server default
	// so the client never reduces concurrency below what a plain request
	// would use.
	adaptiveConcurrencyMinDefault = 5

	// adaptiveConcurrencyMaxDefault is the default absolute maximum for
	// adaptive max_concurrent_shard_requests. Can be overridden higher or
	// lower via WithAdaptiveConcurrencyLimits or OPENSEARCH_GO_SHARD_REQUESTS.
	adaptiveConcurrencyMaxDefault = 256
)

// effectiveMin returns the configured min, falling back to the default.
func (c adaptiveConcurrencyConfig) effectiveMin() int {
	if c.minVal > 0 {
		return c.minVal
	}
	return adaptiveConcurrencyMinDefault
}

// effectiveMax returns the configured max, falling back to the default.
func (c adaptiveConcurrencyConfig) effectiveMax() int {
	if c.maxVal > 0 {
		return c.maxVal
	}
	return adaptiveConcurrencyMaxDefault
}

// parseShardRequests parses the OPENSEARCH_GO_SHARD_REQUESTS env var value.
// Returns the parsed config and an updated routingFeatures bitfield. The
// features parameter is the caller's current bitfield; it is returned with
// routingSkipAdaptiveConcurrency set or cleared based on the parsed value.
//
// Format: bool | min:max (see envShardRequests doc).
func parseShardRequests(value string, features routingFeatures) (adaptiveConcurrencyConfig, routingFeatures) {
	value = strings.TrimSpace(value)
	if value == "" {
		return adaptiveConcurrencyConfig{}, features
	}

	// Try boolean first.
	if b, err := strconv.ParseBool(value); err == nil {
		if !b {
			features |= routingSkipAdaptiveConcurrency
		} else {
			features &^= routingSkipAdaptiveConcurrency
		}
		return adaptiveConcurrencyConfig{}, features
	}

	// Parse as min:max. Presence of a colon distinguishes from a bare
	// integer (which ParseBool would have already handled for "0"/"1").
	minStr, maxStr, hasColon := strings.Cut(value, ":")
	if !hasColon {
		// Bare integer without colon: treat as minVal only.
		if v, err := strconv.Atoi(value); err == nil && v > 0 {
			features &^= routingSkipAdaptiveConcurrency
			return adaptiveConcurrencyConfig{minVal: v}, features
		}
		// Unparseable: ignore (forward-compatible).
		return adaptiveConcurrencyConfig{}, features
	}

	var cfg adaptiveConcurrencyConfig
	if minStr != "" {
		if v, err := strconv.Atoi(minStr); err == nil && v > 0 {
			cfg.minVal = v
		}
	}
	if maxStr != "" {
		if v, err := strconv.Atoi(maxStr); err == nil && v > 0 {
			cfg.maxVal = v
		}
	}

	// Providing numeric limits implies enabling.
	features &^= routingSkipAdaptiveConcurrency

	return cfg, features
}
