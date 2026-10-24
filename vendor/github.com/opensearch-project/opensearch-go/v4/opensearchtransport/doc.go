// SPDX-License-Identifier: Apache-2.0
//
// The OpenSearch Contributors require contributions made to
// this file be licensed under the Apache-2.0 license or a
// compatible open source license.
//
// Modifications Copyright OpenSearch Contributors. See
// GitHub history for details.

// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

/*
Package opensearchtransport provides the transport layer for the OpenSearch client.

It is automatically included in the client provided by the github.com/opensearch-project/opensearch-go package
and is not intended for direct use: to configure the client, use the opensearch.Config struct.

The default HTTP transport of the client is http.Transport; use the Transport option to customize it;

The package will automatically retry requests on network-related errors, and on specific
response status codes (by default 502, 503, 504). Use the RetryOnStatus option to customize the list.
The transport will not retry a timeout network error, unless enabled by setting EnableRetryOnTimeout to true.

Use the MaxRetries option to configure the number of retries, and set DisableRetry to true
to disable the retry behavior altogether.

By default, the retry will be performed without any delay; to configure a backoff interval,
implement the RetryBackoff option function; see an example in the package unit tests for information.

When multiple addresses are passed in configuration, the package will use them in a round-robin fashion,
and will keep track of live and dead nodes. The status of dead nodes is checked periodically.

# Dead Connection Resurrection and Rate Limiting

When a connection is marked dead (health check failure or request error), a goroutine schedules
periodic resurrection attempts using exponential backoff scaled by cluster health.

The base timeout uses exponential backoff capped at ResurrectTimeoutMax:

	baseTimeout = min(ResurrectTimeoutInitial * 2^(failures-1), ResurrectTimeoutMax)

Three inputs compete via max() to determine the final retry interval:

 1. Health-ratio timeout: baseTimeout * (liveNodes / totalNodes).
    Healthy clusters wait longer (no rush); degraded clusters retry sooner.

 2. Rate-limited timeout: (liveNodes * clientsPerServer) / serverMaxNewConnsPerSec.
    Both clientsPerServer and serverMaxNewConnsPerSec are auto-derived from the
    node's core count (discovered from /_nodes/_local/http,os). This throttles health checks
    based on estimated TLS handshake pressure on recovering servers.
    When all nodes are dead, this term is zero (most aggressive -- we need capacity back).

 3. Minimum floor: MinimumResurrectTimeout (absolute lower bound, default 500ms).

The final timeout is max(healthTimeout, rateLimitedTimeout, minimum) + random jitter.

Configure rate limiting via the auto-discovered server core count. The client
discovers each node's allocated_processors from /_nodes/_local/http,os and derives:

  - clientsPerServer = coreCount (1 active client per core)

  - serverMaxNewConnsPerSec = coreCount * 4 (OS queue depth scaling)

  - healthCheckRate = coreCount * 0.10 (10% of core budget)

Example:

	transport, err := opensearchtransport.New(opensearchtransport.Config{
		URLs: urls,
		// Rate limiting is auto-derived from server hardware.
		// No manual tuning required.
	})

# Request-Based Connection Routing

To enable intelligent request routing that routes operations to appropriate node types,
provide a Router implementation in the configuration:

	// Enable routing (recommended for production clusters)
	router, err := opensearchtransport.NewDefaultRouter()
	if err != nil {
		log.Fatal(err)
	}
	transport, err := opensearchtransport.New(opensearchtransport.Config{
		URLs:   []*url.URL{{Scheme: "http", Host: "localhost:9200"}},
		Router: router,
	})

Available router constructors:

  - NewDefaultRouter(): Connection-scoring routing with per-index
    consistency (recommended). When the fan-out set spans multiple
    RTT tiers, tier-span equalization inflates cost attribution so
    traffic distributes equally across all active tiers.
  - NewMuxRouter(): HTTP pattern-based routing (bulk->ingest, search->data) without connection scoring
  - NewRoundRobinRouter(): Coordinator-only preference with round-robin fallback
  - NewRouter(policies...): Custom policy chain

The routing system uses a chain-of-responsibility pattern where policies are tried in sequence:

  - Router: Top-level coordinator that tries policies until one matches
  - Policy: Individual routing strategy that may match or pass to next policy
  - Fallthrough: Policies return (NextHop{}, nil) when they don't match

# Available Policies

Individual routing policies that can be composed into custom strategies:

	// Role-based policies
	NewRolePolicy("data", "ingest")           // Nodes with specific roles
	NewRolePolicy(RoleCoordinatingOnly)       // Coordinating-only nodes

	// Pattern-based routing
	NewMuxPolicy(routes)                      // HTTP pattern matching

	// Conditional routing
	NewIfEnabledPolicy(condition, true, false) // Runtime conditions

	// Basic policies
	NewRoundRobinPolicy()                     // Round-robin across all nodes
	NewNullPolicy()                           // Always returns no connections

	// Policy composition
	NewPolicy(policies...)                    // Chain multiple policies (returns PolicyChain)

# Request Routing Patterns

The router automatically handles these OpenSearch operation patterns:

  - Bulk operations (/_bulk, /_bulk/stream) -> Ingest nodes
  - Ingest pipelines (/_ingest/pipeline/*) -> Ingest nodes
  - Search operations (/_search, /_msearch, /_count) -> Search nodes (3.0+) -> Data nodes
  - Document retrieval (/_doc/*, /_mget, /_source/*) -> Search nodes (3.0+) -> Data nodes
  - Searchable snapshots (/_snapshot/{repo}/{snap}/_mount) -> Warm nodes (2.4+) -> Data nodes
  - Index settings (/{index}/_settings) -> Warm nodes (2.4+) -> Data nodes
  - All other operations -> Round-robin fallback

Each role-specific route falls through to the next tier when the preferred node type is
unavailable in the cluster. If coordinating-only nodes exist, all requests route to them
instead of using role-based routing.

# Node Discovery

Enable automatic cluster discovery to maintain current node information:

	router, _ := opensearchtransport.NewDefaultRouter()
	transport, err := opensearchtransport.New(opensearchtransport.Config{
		URLs: []*url.URL{{Scheme: "http", Host: "localhost:9200"}},
		DiscoverNodesInterval: 5 * time.Minute,
		Router: router,
	})

The discovery process respects node roles and can exclude dedicated cluster manager nodes
from request routing (controlled by IncludeDedicatedClusterManagers configuration).

When a request fails (transport error or retryable HTTP status), the failing connection is
marked with a needsCatUpdate flag that excludes it from scored routing candidate sets.
The node remains available for round-robin fallback and zombie tryout, but stays out of
index-level selection until a /_cat/shards refresh confirms current shard placement.
This flag survives resurrection: a node can pass health checks and return to the active
pool while still excluded from scored routing. A lightweight /_cat/shards-only refresh
is scheduled with urgency proportional to the fraction of affected connections.

# Legacy Connection Pool

To replace the connection pool entirely, provide a custom ConnectionPool implementation via
the ConnectionPoolFunc option. When no Router is specified, the transport falls back to
the traditional connection pool with Selector-based node selection:

	// Legacy selector-based routing
	selector := opensearchtransport.NewRoundRobinSelector()
	transport, err := opensearchtransport.New(opensearchtransport.Config{
		URLs: []*url.URL{{Scheme: "http", Host: "localhost:9200"}},
		Selector: selector,
	})

# Logging and Metrics

The package defines the Logger interface for logging information about request and response.
It comes with several bundled loggers for logging in text and JSON.

Use the EnableDebugLogger option to enable the debugging logger for connection management.
Alternatively, set the OPENSEARCH_GO_DEBUG environment variable to "true" to enable debug
logging globally without code changes. When enabled, debug output is written to stderr.

Use the EnableMetrics option to enable the detailed-metrics snapshot
(per-connection, per-policy, and router state). The per-request counters
(requests, failures, responses by status) are always collected and are
returned by Metrics regardless of this option.

# Enabling the Router via Environment Variable

In v4 the DefaultRouter is off by default. Set the OPENSEARCH_GO_ROUTER environment
variable to enable scored, role-aware request routing without code changes:

	OPENSEARCH_GO_ROUTER=true

The value is parsed with strconv.ParseBool (accepts "true", "1", "t", etc.). When
set to a truthy value and no programmatic Config.Router is provided, [New]
automatically creates a [NewDefaultRouter]. Programmatic Config.Router always
takes precedence.

The opensearch package observes the same variable independently: when
OPENSEARCH_GO_ROUTER is truthy and opensearch.Config.DiscoverNodesOnStart has
not been explicitly set, opensearch.NewClient enables on-start topology
discovery so the router starts with fresh node information. opensearchtransport
itself has no DiscoverNodesOnStart field; callers constructing the transport
directly must trigger discovery themselves if desired.

In v5 the default will flip: the router will be on by default, and
OPENSEARCH_GO_ROUTER=false will disable it. This variable is transitional and will be
removed in v6, where the router is unconditionally enabled. Use the OPENSEARCH_GO_POLICY_*
variables below to disable individual policies if needed.

# Policy Environment Variable Overrides

Operators can disable specific routing policies at startup via environment variables,
without code changes. This is useful for debugging, A/B testing routing behavior, and
emergency overrides in production.

Each policy type has a corresponding environment variable:

	OPENSEARCH_GO_POLICY_CHAIN
	OPENSEARCH_GO_POLICY_MUX
	OPENSEARCH_GO_POLICY_IFENABLED
	OPENSEARCH_GO_POLICY_ROUTER
	OPENSEARCH_GO_POLICY_ROLE
	OPENSEARCH_GO_POLICY_ROUNDROBIN
	OPENSEARCH_GO_POLICY_COORDINATOR
	OPENSEARCH_GO_POLICY_NULL
	OPENSEARCH_GO_POLICY_INDEX_ROUTER
	OPENSEARCH_GO_POLICY_DOCUMENT_ROUTER

Value parsing (evaluated in order):

 1. Bool: if the entire value parses as a bool (e.g., "false"), it applies to ALL
    instances of that policy type. "true" is a no-op (same as default behavior).

 2. Comma-separated path matchers: each item is "path=bool". The path portion is
    matched first as a regexp, then as a string prefix. Example:

    OPENSEARCH_GO_POLICY_ROLE=chain[0].mux[0].role[0]=false

Policy paths use dot-delimited notation with per-type sibling indices:

	chain[0].ifenabled[0].chain[0].mux[0].role[0]

Set OPENSEARCH_GO_DEBUG=true to see policy paths and override actions in stderr.

# Seed URL Fallback

When all router policies and connection pools are exhausted (every discovered node is
dead), the client falls back to the original seed URLs as a last resort. A dedicated
pool built from fresh copies of the seed URLs is tried once after the retry loop is
fully exhausted. On success, discoveryNeeded is set to trigger immediate rediscovery.

Set OPENSEARCH_GO_FALLBACK=false to disable this behavior.
*/
package opensearchtransport
