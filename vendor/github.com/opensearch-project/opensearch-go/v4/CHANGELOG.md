# CHANGELOG

Inspired from [Keep a Changelog](https://keepachangelog.com/en/1.0.0/)

## [4.7.3]

### Changed

- Point Dependabot at the `cmd/osgen` module as well as the repository root. The `gomod` entry in `.github/dependabot.yml` used the singular `directory: "/"` key and so only ever read the root manifest, leaving the nested generator module unwatched since it was introduced. It now uses `directories` with both paths listed explicitly, which is what the singular key cannot express. Modules are listed one by one rather than globbed so that linter testdata fixtures, which pin old dependency versions deliberately, stay out of scope. Any nested module added later needs its own entry ([#1019](https://github.com/opensearch-project/opensearch-go/pull/1019))

### Fixed

- Fix `opensearchutil.BulkIndexer` retaining a worker's peak batch memory after a traffic burst subsides. `(*worker).flush` released a completed batch with `w.items = w.items[:0]`, which keeps the slice's backing array -- and every `BulkIndexerItem` it holds, including each item's `Body` (an `io.ReadSeeker` over the caller's document bytes) and its `OnSuccess`/`OnFailure` closures -- reachable until a later batch of equal or greater size overwrites the slots. A worker that peaked at N items during a backlog replay stayed pinned at ~N items' worth of document bodies and closures indefinitely, even after traffic dropped. `flush` now `clear`s the item slice before truncating, dropping those references so the GC can reclaim them ([#912](https://github.com/opensearch-project/opensearch-go/issues/912))
- Fix an unbounded connection/heap leak in node discovery when the cluster has a dedicated cluster manager (`cluster_manager` role with no work roles). The node was filtered out of the `allConns` inventory while the router received the unfiltered added/removed diffs, so `findConnectionByURL` never matched it: a new `*Connection` was created every discovery cycle and the stale one was never evicted, accumulating without bound in the round-robin fallback pool whose `checkDead` health checks repopulated a per-connection `poolRegistry` `sync.Map` each cycle (leak rate scaled with discovery frequency). `allConns` is now the full connection inventory so discovery reuses and evicts symmetrically, and dedicated cluster managers are excluded at request-routing selection instead: `RoundRobinPolicy` skips them in its `DiscoveryUpdate` add path and `multiServerPool.Next()` skips them during selection (including the no-router fallback), both gated on `IncludeDedicatedClusterManagers`. Discovery still bootstraps against a dedicated cluster manager seed via the seed-fallback pool ([#1003](https://github.com/opensearch-project/opensearch-go/pull/1003))

### Dependencies

- Bump `github.com/getkin/kin-openapi` from 0.142.0 to 0.144.0 and `golang.org/x/text` from 0.14.0 to 0.40.0 in `cmd/osgen` in order to resolve CVE-2026-56852. Details in the Pull Request ([#1019](https://github.com/opensearch-project/opensearch-go/pull/1019))

## [4.7.2]

### Fixed

- Fix data races on `multiServerPool` lock-guarded fields: `activeListCap`, `warmupRounds`, `warmupSkipCount`, and `healthCheck` were de-facto guarded by `cp.mu` but declared at the top level of the struct, allowing accesses from `snapshot()`, `createOrUpdateMultiNodePoolWithLock`, and `updateConnectionPool` to escape the lock without looking wrong. All four fields are now nested inside the `mu` embedded struct so every access is spelled `cp.mu.<field>` and the guard is structural. `snapshot()` reads `activeListCap` under the read lock; `createOrUpdateMultiNodePoolWithLock` runs fully under `allConnsPool.mu` with per-connection `conn.mu` taken inside the loop; `updateConnectionPool`'s RTT-probe scheduling loop reads `healthCheck` under the read lock. `recalculateWarmupParams`/`getWarmupParams` renamed `*WithLock` to document that all callers now hold the pool lock ([#995](https://github.com/opensearch-project/opensearch-go/pull/995))
- Fix removed `opensearch.BuildRequest` reference in v4 upgrade guide ([#978](https://github.com/opensearch-project/opensearch-go/pull/978))

### Dependencies

- Bump `actions/setup-go` to 7.0.0 and `actions/setup-java` to 5.6.0 ([#993](https://github.com/opensearch-project/opensearch-go/pull/993))

## [4.7.1]

### Changed

- Add a first-class container-provider abstraction to the test harness `Makefile`. `CONTAINER_PROVIDER` is auto-detected by CLI presence in the order Colima -> Rancher Desktop (`rdctl`) -> Docker, and overridable with `CONTAINER_PROVIDER=colima|rancher|docker`. Selecting a provider pins the docker context (`colima` / `rancher-desktop`; the Docker provider leaves the active context alone, and a pre-set `DOCKER_CONTEXT` in the environment is respected), resolves the CLI runtime `$(CTR)` (now `docker` by default for every provider, with `CONTAINER_RUNTIME=nerdctl` as an advanced override), ensures the backing VM/daemon is running via the new `cluster.provider.ensure` target (wired into `cluster.start`), and sets `vm.max_map_count` through the provider's VM (`colima ssh` / `rdctl shell`) or a privileged helper container. Previously `$(CTR)` preferred `nerdctl` whenever it was on `PATH`, so a Rancher-installed `nerdctl` could hijack a Colima session. `make cluster.runtime` now reports the detected provider, docker context, and runtime ([#968](https://github.com/opensearch-project/opensearch-go/pull/968))

### Fixed

- Fix two gaps in the seed-fallback routing path left by [#952](https://github.com/opensearch-project/opensearch-go/pull/952)/[#954](https://github.com/opensearch-project/opensearch-go/pull/954)/[#956](https://github.com/opensearch-project/opensearch-go/pull/956): `RoundRobinPolicy.Eval` went straight to `pool.Next()` without checking its own `psEnabled` bit (unlike `CoordinatorPolicy.Eval` and `RolePolicy.Eval`), so a dead but unverified discovered node was returned as a zombie producing a transport error — not `ErrNoConnections` — and the seed fallback never fired; `PolicyChain.Eval` also did not gate on `IsEnabled()` even though `PolicyChain.Route` did, leaving the nested chain inside `IfEnabledPolicy.Eval` unguarded. Both gaps are now closed: `RoundRobinPolicy.Eval` returns no connection when its enabled bit is clear, and `PolicyChain.Eval` skips not-enabled sub-policies, matching the behavior of every other leaf policy and `Route` ([#966](https://github.com/opensearch-project/opensearch-go/pull/966))
- Fix node discovery serving unverified discovered nodes as last-resort zombies and bypassing the seed-URL fallback when `publish_address` is unroutable from the client (NAT'd clusters, Kubernetes stack clusters in CI). `multiServerPool.nextFallbackWithLock` previously returned any dead connection unconditionally; a never-health-checked discovered node was handed out and failed with a transport error (`connection reset by peer` / `no route to host`), which is not `ErrNoConnections` so the seed fallback never fired. Introduce `lcViable`, a monotonic lifecycle bit meaning "proven directly reachable at least once" — seeds are born viable; discovered nodes earn it on their first successful health check or request. `availableForRouting()` now gates on `lcViable` so a dead list holding only never-verified discovered nodes yields `ErrNoConnections` and the request cascades to the seed-URL fallback ([#973](https://github.com/opensearch-project/opensearch-go/pull/973))

## [4.7.0]

### Added

- Add `Close()` to `opensearch.Client` and `opensearchapi.Client` to release background goroutines (node discovery, health/stats pollers, DNS refresh) and idle connections; `opensearchutil.NewBulkIndexer` now closes the client it implicitly creates. Backport of #926 without the default-client cache ([#928](https://github.com/opensearch-project/opensearch-go/issues/928), [#893](https://github.com/opensearch-project/opensearch-go/issues/893))
- Add `VerifyDeadAfter` (`opensearch.Config` / `opensearchtransport.Config`, env override `OPENSEARCH_GO_VERIFY_DEAD_AFTER`): bounds how long a connection proven reachable may still be blindly resurrected as a last-resort "zombie" while dead. Each discovery cycle clears the viability mark on any non-seed connection that has been dead longer than the window, so a node that never recovers stops absorbing requests and must health-check clean again before it is routed to; seed connections are exempt. The env var accepts a boolean (`true` selects the 15m default, `false` disables the expiry) or a `time.ParseDuration` string; the `Config` field follows the `0` = default, `<0` = disabled, `>0` = explicit convention. See `guides/routing.md` (Zombie Connection Resurrection and Connection Viability)
- Add `cmd/osgen` code generator for typed path builders and API consumer files from the OpenAPI spec
- v5preview/opensearchapi: `NewClient` and `NewDefaultClient` now inject `opensearchtransport.NewDefaultRouter` when `config.Client.Router` is nil, opting every v5preview client into intelligent request routing by default. The `OPENSEARCH_GO_ROUTER` env var preserves its v4 semantics end-to-end: `=true`/`=1` enables auto-discovery (via `DiscoverNodesOnStart`); `=false`/`=0` suppresses both Router injection and auto-discovery; unset injects the Router without auto-discovery. v4's `opensearchapi.NewClient` is unchanged. ([#816](https://github.com/opensearch-project/opensearch-go/issues/816))
- Add `envvars.Falsy(name)` helper that distinguishes "explicitly opted out" from "unset" (Truthy collapses both into false). Used by v5preview's router injection rule.
- Add `v5preview/opensearchapi/` package: regenerated v5-track API surface produced by `cmd/osgen` from the OpenAPI spec. Fully typed Req/Resp/Params structs, sub-clients matching OpenSearch namespaces (`client.Cat`, `client.Cluster`, `client.Indices`, etc.), and a `plugins/` subtree for ML/k-NN/security/ISM/etc. Coexists with `opensearchapi/` during the v4 -> v5 transition; see `v5preview/opensearchapi/README.md` for usage and `UPGRADING.md` for migration guidance ([#650](https://github.com/opensearch-project/opensearch-go/issues/650))
- Add `primary_terms_map` and `split_shards_metadata` fields to ClusterState index metadata for OpenSearch >=3.6.0 compatibility
- Add generic `opensearch.Do[T]()` function for compile-time pointer enforcement on response types, preventing a class of bugs where non-pointer values are silently passed to `Client.Do()` and fail at runtime during JSON unmarshaling. Includes `opensearch.NoBody` marker type for calls that expect no response body, unifying all internal dispatch through a single generic path ([#809](https://github.com/opensearch-project/opensearch-go/pull/809))
- Add dynamic read cost scoring: primary shard cost scales with write-pool utilization via `connScoreFunc`, preferring primaries at idle and shedding reads to replicas under write load
- Add `OPENSEARCH_GO_SHARD_COST` environment variable and `WithShardCosts()` router option with `r:base`/`r:amplify`/`r:exponent` curve keys and static cost overrides
- Add `ShardCostConfig` field to `Config` struct for programmatic shard cost override passthrough
- Add `OPENSEARCH_GO_ROUTER` environment variable to enable the DefaultRouter without code changes; set to `true` to opt in (off by default in v4, on by default in v5, removed in v6) ([#815](https://github.com/opensearch-project/opensearch-go/pull/815))
- Add client-side metrics guide covering Metrics API, ConnectionMetric, PolicySnapshot, and RouterSnapshot ([#812](https://github.com/opensearch-project/opensearch-go/pull/812))
- Add `InsecureSkipVerify` config option to disable TLS certificate verification without constructing a custom `http.Transport`, preserving `DefaultTransport` connection pooling, HTTP/2, and timeout defaults ([#786](https://github.com/opensearch-project/opensearch-go/issues/786))
- Add `(*opensearchtransport.Client).Stream(*http.Request) (*http.Response, error)` and a `(*opensearch.Client).Stream` passthrough for raw byte forwarding (proxy and streaming use cases). Stream returns the unbuffered response body from `RoundTrip`; the caller owns reading and closing `res.Body`. Pairs with `opensearch.Do[T]` for typed, decoded results (the SDK owns the body). Stream is exposed only on the concrete `*Client` in v4; v5 will add it to `opensearchtransport.Interface` and remove the deprecated `Perform` ([#786](https://github.com/opensearch-project/opensearch-go/issues/786))
- Add per-attempt `RequestTimeout` to bound individual HTTP round-trips, preventing indefinite hangs on stalled connections ([#786](https://github.com/opensearch-project/opensearch-go/issues/786))
- Add `opensearchutil/shardhash` package with exported `Hash` and `ForRouting` functions for computing OpenSearch shard routing
- Enhanced cluster readiness checking for improved test reliability: `testutil.NewClient()` now includes readiness validation (health + cluster state + nodes info)
- Add `Status` field (`json.RawMessage`) to `TasksGetResp`, `TasksListTask`, and `TaskCancelInfo` for polymorphic task status data; add typed status structs matching the OpenSearch API specification: `BulkByScrollTaskStatus`, `ReplicationTaskStatus`, `ResyncTaskStatus`, `PersistentTaskStatus`; add `Parse*` helpers and `BulkByScrollTaskStatusOrException` for sliced task status ([#788](https://github.com/opensearch-project/opensearch-go/issues/788))
- Test parallelization support via TEST_PARALLEL environment variable (default: CPU cores - 1, minimum 1)
- Add `cmd/osgen/emit.TestPerOpErrorTypeName_CatalogConsistency` to pin the catalog <-> switch coupling between `emit.PerOpErrorTypeName` and `errwrap.OperationWrappers`. Asserts three directions: every group naming a per-op aggregator type has 2+ wrappers in the catalog, every catalog entry with 2+ wrappers names a per-op aggregator type, and every group named by the switch is present in the catalog. Does not pin the runtime `emittableWrappers`/`resolveErrorWrappers` paths; today those sets coincide for the only 2+-wrapper groups (`msearch` / `msearch_template`) ([#857](https://github.com/opensearch-project/opensearch-go/pull/857))
- opensearchapi/testutil package with test suite, client helpers, and JSON comparison utilities
- Add typed path builders in `internal/path/` generated from the OpenAPI spec via `cmd/osgen` for compile-time URL construction safety ([#617](https://github.com/opensearch-project/opensearch-go/issues/617), [#650](https://github.com/opensearch-project/opensearch-go/issues/650))
  - `sync.Pool`-backed `[]byte` buffers eliminate per-request allocation churn; buffers over 4 KiB are discarded to bound pool growth
- opensearchtransport/testutil package with PollUntil helper for eventual consistency testing (ISM policies, index readiness, cluster state changes)
- Configuration option `IncludeDedicatedClusterManagers` for controlling cluster manager node routing ([#765](https://github.com/opensearch-project/opensearch-go/issues/765))
- Policy-based routing system for improved request routing and service availability ([#771](https://github.com/opensearch-project/opensearch-go/pull/771))
  - `Policy` interface for composable routing strategies with lifecycle management
  - `Router` interface with `Route()` method for request-based connection selection
  - `NewPolicy()` implementing chain-of-responsibility pattern for composable routing strategies
  - `NewIfEnabledPolicy()` for conditional routing with runtime evaluation
  - `NewMuxPolicy()` for trie-based HTTP pattern matching with zero-allocation route lookup
  - `NewRolePolicy()` for role-based node selection
  - `NewRoundRobinRouter()` with coordinating node preference and round-robin fallback
  - `NewMuxRouter()` providing role-based request routing with graceful fallback
    - Automatic routing of bulk, streaming bulk, and reindex operations to ingest nodes
    - Automatic routing of search operations (search, msearch, count, by-query operations, scroll, PIT, validate, rank eval) to search/data nodes
    - Automatic routing of document retrieval operations (get, mget, source, explain, termvectors, mtermvectors) to search/data nodes for read locality
    - Automatic routing of template operations (search template, msearch template) and search shards to search/data nodes
    - Automatic routing of field capabilities to search/data nodes
    - Automatic routing of shard maintenance operations (refresh, flush, synced flush, forcemerge, cache clear, segments) to data nodes
    - Automatic routing of single-document writes (index, create, update, delete) to data nodes
    - Automatic routing of shard diagnostics (recovery, shard stores, stats) and rethrottle operations to data nodes
  - `NewDefaultRouter()` extending role-based routing with per-index node affinity (recommended for most users)
- Add consistent hash routing with per-index node affinity for cache locality and AZ-aware load distribution ([#786](https://github.com/opensearch-project/opensearch-go/pull/786))
  - Rendezvous hashing selects a stable subset of nodes per index, preserving OS page cache and query cache locality
  - RTT-bucketed scoring naturally prefers AZ-local nodes and overflows to remote AZs under load
  - Per-pool congestion window (cwnd) routing using TCP-style AIMD congestion control for capacity-aware connection scoring
  - Thread pool discovery via `/_nodes/_local/http,os,thread_pool` provides per-pool capacity ceiling (maxCwnd)
  - Thread pool stats polling via `/_nodes/_local/stats/jvm,breaker,thread_pool` drives AIMD window adjustments
  - Scoring formula: `RTTBucket * (InFlight + 1) / Cwnd * ShardCostMultiplier` -- nearest node with most headroom wins
  - In-flight request tracking per node per pool with atomic add/release bracketing each RoundTrip
  - AIMD slow start (double cwnd) transitions to congestion avoidance (additive increase) at ssthresh
  - Multiplicative decrease on congestion signals: `total_wait_time_in_nanos` for RESIZABLE pools, queue saturation fallback for others
  - Pool overload detection: `delta(rejected) > 0` or HTTP 429 marks pool overloaded, cleared only by stats poller
  - HTTP 429 handling: TryLock + set overloaded + halve cwnd + retry on different node
  - Quorum gating: pre-quorum uses `4 * defaultServerCoreCount` (= 32) as synthetic cwnd, post-discovery uses `4 * allocatedProcessors`, post-quorum uses real pool cwnd
  - Asymmetric scale-up/scale-down thresholds with hysteresis band for stable active pool sizing
  - Dynamic per-index fan-out driven by shard placement (`/_cat/shards`) and request rate
  - `RouterOption` functional options: `WithMinFanOut`, `WithMaxFanOut`, `WithIndexFanOut`, `WithIdleEvictionTTL`, `WithDecayFactor`, `WithFanOutPerRequest`
- Add environment variable escape hatches (`OPENSEARCH_GO_POLICY_*`) to disable specific routing policies at startup ([#786](https://github.com/opensearch-project/opensearch-go/pull/786))
- Add failure-triggered shard map invalidation for faster routing recovery ([#786](https://github.com/opensearch-project/opensearch-go/pull/786))
  - `lcNeedsCatUpdate` lifecycle bit excludes failed connections from routing candidate sets until `/_cat/shards` refresh
  - Connections remain available for general routing (round-robin, zombie tryouts) while excluded from scored routing
  - Dedicated `discoverCatTimer` schedules lightweight `/_cat/shards`-only refresh (no full node discovery)
  - Refresh urgency scales with cluster impact: `interval = discoverNodesInterval * (1 - flaggedFraction)`, clamped to 5s floor
  - `OnShardMapInvalidation` observer event for monitoring invalidation triggers
  - `NeedsCatUpdate` field in `ConnectionMetric` for observability
- Add routing observability: observer events, metrics snapshot, connection inspection ([#786](https://github.com/opensearch-project/opensearch-go/pull/786))
  - `OnRoute` observer event with full scoring breakdown (`RouteEvent`, `RouteCandidate`)
  - `RouterSnapshot` in `Client.Metrics()` exposes per-index cache state (fan-out, shard nodes, request rate, idle-since)
  - `Connection.RTTMedian()`, `Connection.RTTBucket()`, `Connection.EstLoad()` for per-connection inspection
  - `ConnectionMetric` enriched with `rtt_bucket`, `rtt_median`, `est_load` fields
- Add murmur3 shard-exact routing for `?routing=` and document ID requests ([#786](https://github.com/opensearch-project/opensearch-go/pull/786))
  - Client-side murmur3 x86 32-bit hash matching OpenSearch's `Murmur3HashFunction.hash(String)` for shard-exact targeting
  - Document-level requests (`_doc`, `_source`, `_update`, `_explain`, `_termvectors`) use doc ID as default routing value when no explicit `?routing=` present, matching OpenSearch's `OperationRouting.generateShardId()` behavior
  - Client-side murmur3 shard-exact candidate selection routes requests to nodes hosting the target shard
  - Per-shard-number placement data from `/_cat/shards` maps shard numbers to primary and replica node names
  - Graceful fallback to rendezvous hashing when shard map data is unavailable
  - `RoutingValue`, `EffectiveRoutingKey`, `TargetShard`, `ShardExactMatch` fields in `RouteEvent` for observability
- Add `OPENSEARCH_GO_ROUTING_CONFIG` and `OPENSEARCH_GO_DISCOVERY_CONFIG` environment variables for runtime feature control ([#786](https://github.com/opensearch-project/opensearch-go/pull/786))
  - `OPENSEARCH_GO_ROUTING_CONFIG`: toggle shard-exact routing (`-shard_exact`)
  - `OPENSEARCH_GO_DISCOVERY_CONFIG`: skip individual discovery server calls (`-cat_shards`, `-routing_num_shards`, `-cluster_health`, `-node_stats`)
  - Bitfield flags use `+`/`-` prefix convention for explicit opt-in/out; zero-initialized = all features enabled
  - `WithShardExactRouting(bool)` `RouterOption` for programmatic control (env var overrides)
  - Evaluated once at client init time; immutable after
  - Document environment variables in `guides/routing.md`
  - Document read-after-write visibility guarantees with operation-aware routing in `guides/routing.md`
- Add adaptive `max_concurrent_shard_requests` derived from cluster-wide AIMD congestion window ([#800](https://github.com/opensearch-project/opensearch-go/issues/800))
- Add partial failure error types (`PartialBulkError`, `PartialSearchError`, `ShardFailureError`, `MultiSearchItemError`) that surface HTTP 200 partial failures as typed Go errors, controlled by a per-category `errmask.ErrorMask` bitfield on `Config.Errors` ([#816](https://github.com/opensearch-project/opensearch-go/issues/816))
  - `PartialBulkError` returned from `Bulk` when `resp.Errors` is true, carries `FailedItems` and `SucceededCount`
  - `PartialSearchError` returned from `Search`, `MSearch`, `MSearchTemplate`, `SearchTemplate`, `Scroll.Get` when `_shards.failed > 0`
  - `ShardFailureError` returned from `Index`, `Document.Create`, `Document.Delete`, `Update` when replica shards fail
  - `MultiSearchItemError` returned from `MSearch`/`MSearchTemplate` for per-sub-response Error envelopes
  - `MSearchErrors` / `MSearchTemplateErrors` per-op containers (Go 1.20+ multi-error contract via `Unwrap() []error`) when 2+ wrapper categories fire on the same response
  - `PartialFailureError` marker interface with `IsPartial() bool` for type-switching across all partial-failure types
  - `opensearchapi.Errors(err) []error` package-level helper that flattens single- and multi-wrapper errors into a uniform slice; recommended call-site pattern is a `for`/`switch` over the result (not `errors.As` against a specific type)
  - Helper functions: `IsPartialFailure`, `ToleratePartialFailures`, `RequireSuccessRate` for threshold-based error tolerance
  - Operation constants: `OperationIndex`, `OperationCreate`, `OperationUpdate`, `OperationDelete`
  - Per-Resp helper methods (`BulkItemFailures`, `SearchShardFailures`, `WriteShardFailures`, `MultiSearchItemFailures`, `PartialFailures(mask)`) exist on the response types as engine machinery for the dispatch; new code should prefer a `for`/`switch` over `opensearchapi.Errors(err)` rather than the per-Resp helpers, for forward compatibility
  - `Config.Errors *errmask.ErrorMask` replaces a single boolean: each bit suppresses one wrapper category. v4 defaults to `errmask.All` (mask everything, preserves pre-bitfield behavior); v5+ defaults to `errmask.Empty` (report everything)
  - `OPENSEARCH_GO_ERROR_MASK` environment variable overrides `Config.Errors` at runtime via comma-separated `+`/`-` tokens (lowercase snake_case wrapper names; unknown tokens silently dropped, debug-logged)
  - Both `(resp, error)` are non-nil on partial failure -- response is fully populated
  - `v5preview/opensearchapi` ports the same model with spec-driven types (regenerated from the OpenAPI `x-error-responses` extension on every `cmd/osgen` run)
- Add `OperationClassifier` for zero-allocation HTTP method+path to `OperationID` mapping ([#816](https://github.com/opensearch-project/opensearch-go/issues/816))
  - Bit-packed `OperationID` (int64) encoding R/W flag, category, and minor operation
  - Masking helpers: `IsWrite`, `IsRead`, `Category`, `Minor`
  - `String()` returns Prometheus-friendly labels (e.g., `"search"`, `"bulk"`, `"doc_get"`)
  - Reuses existing `routeTrie` for O(path-segments) lookup, safe for concurrent use
  - Enables transparent metrics/tracing middleware at the `http.RoundTripper` layer
  - Transport automatically sets `max_concurrent_shard_requests` query parameter on search requests routed through a coordinator node
  - Value derived from a cluster-wide aggregate of all polled nodes' search pool wait-time and completion deltas, clamped to `[floor, cap]` (default: 5–256)
  - Cluster-wide signal correctly models data-node fan-out capacity: single hot nodes are diluted by healthy peers, and MCSR only drops when aggregate cluster pressure rises
  - Per-node AIMD for connection scoring remains unchanged (hot-node avoidance is handled by connection selection, not fan-out throttling)
  - Falls back to per-node cwnd before the first poll cycle completes
  - Respects explicit caller overrides: pre-existing `max_concurrent_shard_requests` query parameter is never clobbered
  - Not applied to shard-exact routed requests (coordinator fan-out is irrelevant)
  - `WithAdaptiveConcurrency(bool)` and `WithAdaptiveConcurrencyLimits(floor, cap)` `RouterOption` for programmatic control
  - `OPENSEARCH_GO_SHARD_REQUESTS` environment variable: `true`/`false` to enable/disable, or `min:max` to set floor and cap (e.g., `10:512`)
  - `OPENSEARCH_GO_ROUTING_CONFIG=-adaptive_mcsr` to disable via routing config bitfield
  - `MaxConcurrentShardRequests` field in `RouteEvent` for observability
- Add seed URL fallback as last-resort connection source when all router pools are exhausted ([#786](https://github.com/opensearch-project/opensearch-go/pull/786))
  - Builds a dedicated `multiServerPool` from fresh copies of the original seed URLs at client init
  - Fires after the entire retry loop when all router policies and connection pools return `ErrNoConnections`
  - On success: triggers immediate cluster rediscovery to repopulate router pools
  - `OPENSEARCH_GO_FALLBACK=false` disables seed fallback (enabled by default)
- Add consolidated environment variable reference in `guides/routing.md` and `USER_GUIDE.md` ([#786](https://github.com/opensearch-project/opensearch-go/pull/786))
- Add connection pool health probes with cluster-aware resurrection timing ([#786](https://github.com/opensearch-project/opensearch-go/pull/786))
  - Auto-discover server core count from `/_nodes/http,os,thread_pool` to derive all rate-limiting and congestion window parameters (default: 8 cores)
  - Weighted round-robin for heterogeneous clusters: nodes with more cores get proportionally more traffic via GCD-normalized duplicate pointers in the ready list
  - `lcNeedsHardware` lifecycle bit tracks connections needing hardware info; per-node fallback via `/_nodes/_local/http,os,thread_pool` during health checks
  - Capacity model dynamically recalculated on each discovery cycle from minimum `allocatedProcessors` across all nodes
  - TLS-aware rate limiting prevents overwhelming recovering servers during outages
  - Three-input timeout formula: `max(healthTimeout, rateLimitedTimeout, minimumFloor) + jitter`
  - Shuffle ready connection list on add/resurrect to prevent round-robin hot-spotting
  - Two-phase readiness health check: `GET /` then `GET /_cluster/health?local=true` with `initializing_shards` gate to prevent routing to recovering nodes
  - Store cluster health metrics (`ClusterHealthLocal`) on each connection for observability
  - Periodic cluster health refresh for ready connections keeps `ClusterHealthLocal` data current for load-shedding and routing decisions
    - Refresh interval scales with cluster size: `clamp(liveNodes * clientsPerServer / healthCheckRate, 5s, 5min)`
    - Single-node clusters skip refresh entirely (no routing benefit)
  - Node stats polling with load shedding via `NodeStatsInterval` configuration
    - Polls `GET /_nodes/_local/stats/jvm,breaker,thread_pool` to detect overloaded nodes and update congestion windows
    - Per-pool AIMD congestion control adjusts cwnd based on thread pool wait time and queue saturation
    - Overloaded nodes are demoted from the ready list to the standby partition
    - Overload detection: JVM heap threshold (`OverloadedHeapThreshold`, default 85%), circuit breaker size ratio (`OverloadedBreakerRatio`, default 0.90), breaker trip delta, and cluster status red
- Add heterogeneous Docker cluster targets for integration-testing weighted routing and role-based request routing
  - `cluster.heterogeneous.cpu.1` and `cluster.heterogeneous.cpu.2` set per-node CPU limits via Docker Compose overrides
  - `cluster.heterogeneous.roles` assigns distinct node roles (cluster_manager+ingest, data+ingest, data)
  - `cluster.homogeneous` removes all overrides to reset to default configuration
  - `cluster.status` now shows per-node roles and allocated processors via `_nodes/http,os`
- Add request routing guide (`guides/routing.md`) consolidating routing architecture, connection scoring, pool lifecycle, cost model, and configuration reference ([#786](https://github.com/opensearch-project/opensearch-go/pull/786))
- Add per-item `Error` field to `MGetResp`, `MTermvectorsResp`, and `MSearchResp` for detecting partial failures in multi-document operations ([#797](https://github.com/opensearch-project/opensearch-go/issues/797))
- Add `DocumentError` type for structured per-item error information in multi-document responses
- Add `BulkByScrollFailure` type for structured failure information in `_delete_by_query`, `_update_by_query`, and `_reindex` responses
- Add `Routing` and `Fields` to `MGetResp.Docs` to match the full OpenSearch `_mget` response format
- Add `ForcedRefresh` field to `IndexResp`, `DocumentDeleteResp`, and `UpdateResp` for consistency with `DocumentCreateResp`
- Add `Status` and `Primary` fields to `ResponseShardsFailure` for shard failure diagnostics

### Changed

- Connection dead/overloaded timestamps (`deadSince`/`overloadedAt`) are now lock-free `atomic.Int64` values. `buildConnectionMetric` no longer locks each connection to read them, removing the dominant metrics-snapshot lock contention against the per-request writers. No public API change; `EnableMetrics` behavior is unchanged. ([#892](https://github.com/opensearch-project/opensearch-go/issues/892))
- Per-request transport metrics (`requests`, `failures`, responses-by-status) are now always collected via lock-free atomics, independent of `EnableMetrics`. `EnableMetrics` now gates only the detailed-metrics snapshot (per-connection, per-policy, and router state returned by `Metrics()`). The responses-by-status counter moved from a mutex-guarded map to a lock-free atomic array. `Metrics()` no longer returns an error when metrics are disabled — it always returns the per-request counters (callers that branched on `if err != nil` for the disabled case should drop that check). ([#891](https://github.com/opensearch-project/opensearch-go/issues/891))
- **BREAKING**: `opensearch.Request` interface signature changed from `GetRequest() (*http.Request, error)` to `GetRequest(method string) (*http.Request, error)`. The HTTP method is now caller-provided rather than hardcoded per operation, enabling correct method selection for operations that support multiple HTTP methods (e.g. search supports both GET and POST). This only affects code that implements or calls `GetRequest` directly; standard usage through client methods (e.g. `client.Search(ctx, req)`) is unaffected ([#650](https://github.com/opensearch-project/opensearch-go/issues/650))
- Bump CI and developer guide OpenSearch versions: compatibility matrix to 2.19.5, default integration test version to 3.6.0 ([#810](https://github.com/opensearch-project/opensearch-go/pull/810))
- Include `_nodes.failures` detail in discovery error messages for diagnosing intermittent CI failures on older OpenSearch versions ([#823](https://github.com/opensearch-project/opensearch-go/pull/823))
- Test against Opensearch 3.6.0 ([#817](https://github.com/opensearch-project/opensearch-go/pull/817))
- Consolidate test utilities into two canonical packages: opensearchtransport/testutil (env helpers, polling, version comparison) and opensearchapi/testutil (client-dependent helpers, test suite, JSON comparison)
- Rename `singleConnectionPool` to `singleServerPool` and `statusConnectionPool` to `multiServerPool` for clarity ([#786](https://github.com/opensearch-project/opensearch-go/pull/786))
- Refactor Client struct to use embedded mutex pattern for improved thread safety ([#775](https://github.com/opensearch-project/opensearch-go/pull/775))
- Refactor metrics struct to use atomic counters for lock-free request/failure tracking ([#776](https://github.com/opensearch-project/opensearch-go/pull/776))
- Test against Opensearch 2.19.4, 3.1, 3.3, and 3.4 ([#782](https://github.com/opensearch-project/opensearch-go/pull/782))
- Migrate all test files to context-aware API calls for proper timeout and cancellation support
- Add cluster readiness validation and improve cluster error diagnostics
- Update Docker cluster management to add version-aware role detection (cluster_manager vs master)
- Generate unique document IDs in tests for parallel test execution and eliminate known test flakes
- Reduce integration test timeout from 1h to 10m per package with parallel execution support
- Refactor transport code for improved maintainability (rename ErrInvalidRole -> InvalidRoleError, add response body cleanup, simplify initialization)
- **BREAKING**: Change `CatTemplatesReq.Templates` and `IndexTemplateGetReq.IndexTemplates` from `[]string` to `string` to match the OpenSearch API specification, which types these path parameters as scalar name patterns (not comma-separated lists). This breakage will show up at compile time as a type mismatch and is easy to fix. Callers passing a single pattern only need to remove the slice literal (e.g. `[]string{"*"}` becomes `"*"`). Callers that relied on the old behavior of joining multiple patterns can use `strings.Join(patterns, ",")` to produce the comma-separated string themselves.
- **BREAKING**: Enhanced node discovery to match OpenSearch server behavior ([#765](https://github.com/opensearch-project/opensearch-go/issues/765))
  - Dedicated cluster manager nodes are now excluded from client request routing by default (best practice)
  - Node selection logic now matches Java client `NodeSelector.SKIP_DEDICATED_CLUSTER_MASTERS` behavior
- **BREAKING**: Add context support to discovery and client lifecycle management
  - `opensearchtransport.Discoverable` interface now requires `context.Context` parameter: `DiscoverNodes(ctx context.Context) error`
  - `opensearch.Client.DiscoverNodes()` and `opensearchtransport.Client.DiscoverNodes()` now require `context.Context` parameter
  - `opensearch.Config` and `opensearchtransport.Config` now accept optional `Context` and `CancelFunc` fields
  - `opensearchutil.BulkIndexerConfig` now accepts optional `Context` and `CancelFunc` fields
  - Enables proper context propagation for timeouts, cancellation, and graceful shutdown
  - Role compatibility validation prevents conflicting role assignments (master+cluster_manager, warm+search)
  - OpenSearch 3.0+ searchable snapshots now use `warm` role instead of deprecated `search` role
- **BREAKING**: Migrate `signer/aws` package from AWS SDK v1 to AWS SDK v2 due to AWS SDK v1 reaching end-of-support on July 31, 2025
  - Constructor now takes `aws.Config` instead of `session.Options`
  - See USER_GUIDE.md for details required to migrate
  - Users who need access to the existing `signer/awsv2` API can still use it, however they are encouraged to migrate to `signer/aws`
- **BREAKING**: Replace `[]json.RawMessage` with typed `[]BulkByScrollFailure` for `Failures` field in `DocumentDeleteByQueryResp`, `UpdateByQueryResp`, and `ReindexResp` ([#797](https://github.com/opensearch-project/opensearch-go/issues/797)). This is a compile-time change only — callers that were not accessing `.Failures` are unaffected, and callers that were manually unmarshaling `json.RawMessage` can now access typed fields directly.
- Replace inline `_shards` struct with `ResponseShards` in `IndexResp`, `DocumentCreateResp`, `DocumentDeleteResp`, `UpdateResp`, `IndicesRefreshResp`, and `IndicesCountResp` to expose shard `Failures` and `Skipped` fields ([#797](https://github.com/opensearch-project/opensearch-go/issues/797)). Code accessing `resp.Shards.Total`, `resp.Shards.Successful`, or `resp.Shards.Failed` compiles unchanged.
- Add `omitempty` to all deprecated `_type` JSON tags so empty values are omitted during marshaling
- Modernize tests to use Go 1.25's `WaitGroup.Go()` ([#834](https://github.com/opensearch-project/opensearch-go/pull/834))

### Deprecated

- Mark `opensearchtransport.Client.Perform` and the `opensearch.Client.Perform` passthrough as deprecated; both remain fully functional in v4 (still buffering the response body via `io.ReadAll` + `NopCloser`) and will be removed in v5. New code should call `opensearch.Do[T]` for typed, decoded results or `opensearchtransport.Client.Stream` / `opensearch.Client.Stream` for raw byte forwarding.
- Mark `Client.Do()` with a `Deprecated` doc annotation in favor of `opensearch.Do[T]()` for compile-time pointer safety; `Client.Do()` remains fully functional and will not be removed, but `staticcheck` SA1019 will nudge cross-package callers toward the safer generic alternative
- Mark `opensearch.ToPointer` and `opensearchapi.ToPointer` as deprecated; they remain fully functional but will be removed in v5. Once the module's go directive moves to 1.26, callers can drop the helper entirely in favor of native `new(value)` literal syntax (e.g. `new(false)`)

### Removed

### Fixed

- Fix node discovery hijacking the request stream with unverified, unreachable discovered nodes and masking the user-supplied seed-URL fallback. When discovered `publish_address` values are unroutable from the client (NAT'd or misconfigured clusters, e.g. a Kubernetes stack cluster in CI), a freshly discovered but never-health-checked node could be served to requests as a zombie -- failing every request with `no route to host` -- instead of returning `ErrNoConnections` and cascading to the reachable seed URL. Connections are now considered available for routing only when they are a user-supplied seed (assumed reachable) or a discovered node confirmed reachable, and every routing policy and pool (round-robin, role, coordinator, index/doc router, single-server, and multi-server pools) consistently honors that gate on both the enabled-bit and connection-selection paths, so the seed fallback serves requests until a discovered node health-checks clean ([#952](https://github.com/opensearch-project/opensearch-go/pull/952), [#954](https://github.com/opensearch-project/opensearch-go/pull/954), [#956](https://github.com/opensearch-project/opensearch-go/pull/956))
- Fix `BulkIndexerStats.NumAdded` overcounting items rejected by `Add()` when the caller's context is cancelled before the item could be enqueued: increment `NumAdded` only after the queue accepts the item, and add a new `BulkAddFailCount` counter for items dropped on the `<-ctx.Done()` branch. Migrate `bulkIndexerStats` fields to `sync/atomic.Uint64` typed values so future direct access is a compile-time error rather than a `-race`-only finding ([#783](https://github.com/opensearch-project/opensearch-go/issues/783))
- Fix `opensearchtransport.Client.setReqGlobalHeader` comparing the per-request header value against the global header name, so a request-level header never suppressed the matching global default and both were sent ([#859](https://github.com/opensearch-project/opensearch-go/pull/859))
- Fix gzip buffer-pool nil poisoning on compress error: `gzipCompressor.compress` returned `(nil, err)` while the caller's deferred `collectBuffer` still ran, putting a typed-nil `*bytes.Buffer` into the `sync.Pool` that panics on the next `Get().Reset()` ([#859](https://github.com/opensearch-project/opensearch-go/pull/859))
- Fix `opensearchtransport.Client.Perform` silently dropping `io.ReadAll` errors during response buffering via `:=` shadowing; the read error now propagates wrapped in the new `opensearchtransport.ErrResponseBodyRead` sentinel, and `opensearch.Client.Do` classifies the `(resp != nil, err != nil)` case via `errors.Is` so only genuine body-read failures are labeled `ErrReadBody` (an unrelated transport error returned alongside a response, such as context cancellation during retry backoff, is no longer misreported as a read failure). As a consequence, `opensearch.Client.Do` now returns a non-nil `*Response` alongside a non-nil error in this case where it previously returned `(nil, err)`; callers detecting a hard transport failure should check `resp == nil` rather than `err != nil` ([#859](https://github.com/opensearch-project/opensearch-go/pull/859))
- Fix error-response body not being closed in `opensearch.ParseError`. `ParseError` now closes the original body before re-wrapping the read bytes in a `NopCloser`. The v5preview `do()` no-decode error path no longer needs its own drain: `opensearch.Do` routes through the buffered `opensearchtransport.Client.Perform`, so the returned `resp.Body` is already an in-memory `NopCloser` over the full payload and stays readable for the caller ([#859](https://github.com/opensearch-project/opensearch-go/pull/859))
- Fix response-body lifecycle on the raw `RoundTrip` paths that lack `Perform`'s buffering safety net, where closing a partially-read body defeated HTTP keep-alive: the stats poller (`cluster_health.go`), discovery's `/_cat/shards`, `/_cluster/state/metadata`, and `/_nodes` paths, and the `fetchClusterHealth`/`baselineHealthCheck`/`hardwareInfoHealthCheck` pollers now drain to EOF (`io.Copy(io.Discard, ...)`) before close (covering both non-200 returns and `json.Decode` success paths that stop before EOF). The AWS v1 and v2 signers now close the request body on the read-error path in `hexEncodedSha256OfRequest` ([#859](https://github.com/opensearch-project/opensearch-go/pull/859))
- Add typed response-format defaults for `v5preview/opensearchapi/` cat, list, ppl, and sql operations: when the caller leaves `Format` unset, the SDK now emits the value the typed Resp struct expects (`json` for cat/list/explain, `jdbc` for ppl/sql query) instead of letting the server fall back to a default the JSON decoder cannot handle.
- Replace `WaitForAllNodesReady` inline `require.Eventually` loop with a layered readiness FSM (`internal/test/readiness`) that observes per-node progression through `LayerTCP -> LayerHTTP -> LayerClusterJoin -> LayerStatsReady`, records transitions including regressions, and emits a structured per-node diagnostic with the full last cat-nodes response on timeout. Per-layer budgets are tuned for CI pessimism (cold JVM startup is the long pole); total budget for `TargetClusterReady` is 6.5 minutes. ([#650](https://github.com/opensearch-project/opensearch-go/issues/650))
- Fix bulk indexer HTML-escaping `_id` and `routing` values containing `<`, `>`, or `&` characters, causing OpenSearch to store escaped values (e.g., `\u003croot_account\u003e` stored instead of `<root_account>`), leading to duplicate documents, unreachable data on read-by-ID paths, and potential shard routing mismatches. Present since the `json.Marshal` migration in 2021 (commit `3da59092`). Replace `json.Marshal` with `json.NewEncoder` + `SetEscapeHTML(false)` in `opensearchutil.worker.writeMeta` and `opensearchutil.JSONReader`; replace per-worker `aux []byte` with `sync.Pool`-backed `*bytes.Buffer`; add table-driven test coverage for `writeMeta` edge cases and refactor remaining `TestBulkIndexer` subtests to table-driven `require`-based style ([#824](https://github.com/opensearch-project/opensearch-go/pull/824))
- Fix pool replacement orphaning resurrection goroutines during node discovery, causing connections to become permanently dead with no active health checker ([#786](https://github.com/opensearch-project/opensearch-go/pull/786))
- Fix multi-to-single pool demotion leaking resurrection goroutines by giving each `multiServerPool` its own derived context and cancelling it on demotion ([#830](https://github.com/opensearch-project/opensearch-go/pull/830))
- Extract `newMultiServerPoolFromClientWithLock` as single source of truth for Client-to-pool settings propagation ([#786](https://github.com/opensearch-project/opensearch-go/pull/786))
- Skip shard routing integration tests on OpenSearch < 2.2.0 with security plugin due to server-side `OptionalDataException` from non-thread-safe User serialization (opensearch-project/security#1970)
- Fix URL path construction across 74 `GetRequest` methods where empty path segments produced a double-slash `//` that `http.NewRequest` misparsed as an RFC 3986 authority separator; replace manual `strings.Builder` paths with typed path builder structs that reject empty required segments ([#617](https://github.com/opensearch-project/opensearch-go/issues/617), [#650](https://github.com/opensearch-project/opensearch-go/issues/650))
- Eliminate per-request `url.Parse` overhead by constructing `*http.Request` directly with a coalesced struct; reduce per-request allocations from 8/2930B to 2/472B for typical operations ([#650](https://github.com/opensearch-project/opensearch-go/issues/650))
- Fix alias, mapping, settings, and block API URL path construction when Indices is empty, which caused `http.NewRequest` to misparse the double-slash as an authority separator ([#650](https://github.com/opensearch-project/opensearch-go/issues/650))
- Fix discovery pool wipe when all cluster nodes time out during `/_nodes/http` fan-out: parse `_nodes` metadata envelope and return `errDiscoveryEmpty` when `successful == 0`, preserving the existing connection pool for retry ([#821](https://github.com/opensearch-project/opensearch-go/pull/821))
- Skip shard routing integration tests on OpenSearch < 2.2.0 with security plugin due to server-side `OptionalDataException` from non-thread-safe User serialization (opensearch-project/security#1970)
- Fix flaky `TestDefaultHealthCheck_RetryAfterMaxRetry`: replace wall-clock `time.Sleep` + `atomic.Int64` synchronization with context cancellation (`ctx.Done()`), and widen `maxRetryClusterHealth` to 5s so the baseline HTTP round-trip cannot race past the retry interval ([#787](https://github.com/opensearch-project/opensearch-go/pull/787))
- Skip opensearchtransport integration tests on OpenSearch < 2.2.0 with security plugin due to server-side `OptionalDataException` from non-thread-safe User serialization (opensearch-project/security#1970)
- Skip shard routing integration tests on OpenSearch < 2.2.0 with security plugin due to server-side `OptionalDataException` from non-thread-safe User serialization (opensearch-project/security#1970)
- Fix connection lifecycle bug in multiServerPool.OnFailure where connections were scheduled for resurrection before being moved from ready to dead list, causing potential race conditions
- Fix flaky connection integration test by replacing arbitrary sleep times with proper server readiness polling
- Fix cluster readiness checks in integration tests to handle HTTPS cold start delays (increase timeout to 15s)
- Fix GitHub Actions workflow authentication for OpenSearch 2.12.0+ password changes (admin -> myStrongPassword123!)
- Fix Docker cluster management to properly handle version-specific configurations and clean stale images/volumes
- Fix OpenSearch 2.8.0+ Tasks API compatibility by adding cancellation_time_millis field to TasksListTask struct
- Fix OpenSearch 3.1.0+ API compatibility by adding phase_results_processors field to nodes API and time_in_execution fields to cluster pending tasks API
- Fix OpenSearch 3.2.0+ API compatibility by adding max_last_index_request_timestamp and startree query fields across nodes stats, indices stats, and cat APIs, plus settings field to security plugin health API
- Fix OpenSearch 3.3.0+ API compatibility by adding neural_search breaker, query_failed and startree_query_failed search fields, search pipeline system_generated fields across multiple APIs, plus ingestion_status field to cluster state API and jwks_uri field to security config API
- Fix OpenSearch 3.4.0+ API compatibility by adding warmer fields to merges section, parallelism field to thread pool, and status_counter field across multiple APIs
- Fix cat indices API field naming compatibility across OpenSearch versions by using forward-compatible field names (PrimarySearchStartreeQuery) that match the corrected 3.3.0+ naming, with fallback support for the temporary 3.2.0 field names
- Fix cat APIs data type compatibility by changing byte fields from int to string to properly handle values like "0b"
- Fix floating point precision loss in nodes stats concurrent_avg_slice_count field by changing from float32 to float64
- Fix ISM RefreshSearchAnalyzers missing leading slash in URL path, causing HTTP/2 request failures ([#686](https://github.com/opensearch-project/opensearch-go/pull/686))

### Security

### Dependencies

- Bump golangci-lint from v2.11.2 to v2.11.4
- Bump `golang.org/x/sync` from v0.19.0 to v0.20.0 ([#831](https://github.com/opensearch-project/opensearch-go/pull/831))
- Bump `golang.org/x/mod` from v0.33.0 to v0.35.0 ([#831](https://github.com/opensearch-project/opensearch-go/pull/831))
- Bump `github.com/wI2L/jsondiff` from v0.7.0 to v0.7.1 ([#831](https://github.com/opensearch-project/opensearch-go/pull/831))
- Bump `github.com/aws/aws-sdk-go-v2` from v1.41.1 to v1.41.7 ([#831](https://github.com/opensearch-project/opensearch-go/pull/831))
- Bump `github.com/aws/aws-sdk-go-v2/config` from v1.32.7 to v1.32.17 ([#831](https://github.com/opensearch-project/opensearch-go/pull/831))
- Bump `github.com/aws/aws-sdk-go-v2/credentials` from v1.19.7 to v1.19.16 ([#831](https://github.com/opensearch-project/opensearch-go/pull/831))
- Bump `github.com/aws/smithy-go` from v1.24.0 to v1.25.1 ([#831](https://github.com/opensearch-project/opensearch-go/pull/831))
- Bump `github.com/aws/aws-sdk-go-v2/config` from 1.32.6 to 1.32.7 ([#767](https://github.com/opensearch-project/opensearch-go/pull/767))

## [4.6.0]

### Dependencies

- Bump `github.com/aws/aws-sdk-go-v2/config` from 1.29.14 to 1.32.5 ([#707](https://github.com/opensearch-project/opensearch-go/pull/707), [#711](https://github.com/opensearch-project/opensearch-go/pull/711), [#719](https://github.com/opensearch-project/opensearch-go/pull/719), [#730](https://github.com/opensearch-project/opensearch-go/pull/730), [#737](https://github.com/opensearch-project/opensearch-go/pull/737), [#761](https://github.com/opensearch-project/opensearch-go/pull/761))
- Bump `github.com/aws/aws-sdk-go-v2` from 1.36.4 to 1.41.0 ([#710](https://github.com/opensearch-project/opensearch-go/pull/710), [#720](https://github.com/opensearch-project/opensearch-go/pull/720), [#759](https://github.com/opensearch-project/opensearch-go/pull/759))
- Bump `github.com/stretchr/testify` from 1.10.0 to 1.11.1 ([#728](https://github.com/opensearch-project/opensearch-go/pull/728))
- Bump `github.com/aws/aws-sdk-go` from 1.55.7 to 1.55.8 ([#716](https://github.com/opensearch-project/opensearch-go/pull/716))
- Bump go version from 1.24.0 to 1.25.9 in order to resolve certain CVEs. Details in the Pull Request ([#825](https://github.com/opensearch-project/opensearch-go/pull/825))

### Added

- Adds new fields for Opensearch 3.0 ([#702](https://github.com/opensearch-project/opensearch-go/pull/702))
- Allow users to override signing port ([#721](https://github.com/opensearch-project/opensearch-go/pull/721))
- Add `phase_took` features supported from OpenSearch 2.12 ([#722](https://github.com/opensearch-project/opensearch-go/pull/722))
- Adds the action to refresh the search analyzers to the ISM plugin ([#686](https://github.com/opensearch-project/opensearch-go/pull/686))

### Changed

- Test against Opensearch 3.0 ([#702](https://github.com/opensearch-project/opensearch-go/pull/702))
- Add more SuggestOptions to SearchResp ([#713](https://github.com/opensearch-project/opensearch-go/pull/713))
- Updates Go version to 1.24 ([#674](https://github.com/opensearch-project/opensearch-go/pull/674))
- Replace `golang.org/x/exp/slices` usage with built-in `slices` ([#674](https://github.com/opensearch-project/opensearch-go/pull/674))
- Update golangci-linter to 1.64.8 ([#740](https://github.com/opensearch-project/opensearch-go/pull/740))
- Change MaxScore to pointer ([#740](https://github.com/opensearch-project/opensearch-go/pull/740))
- Update workflow action ([#760](https://github.com/opensearch-project/opensearch-go/pull/760))
- Migrate to golangci-lint v2 ([#760](https://github.com/opensearch-project/opensearch-go/pull/760))

### Deprecated

### Removed

### Fixed

- Missing "caused by" information in StructError ([#752](https://github.com/opensearch-project/opensearch-go/pull/752))
- Add missing `ignore_unavailable`, `allow_no_indices`, and `expand_wildcards` params to MSearch ([#757](https://github.com/opensearch-project/opensearch-go/pull/757))
- Fix `UpdateResp` to correctly parse the `get` field when `_source` is requested in update operations. ([#739](https://github.com/opensearch-project/opensearch-go/pull/739))

### Security

## [4.5.0]

### Dependencies

- Bump `github.com/aws/aws-sdk-go-v2/config` from 1.29.6 to 1.29.14 ([#692](https://github.com/opensearch-project/opensearch-go/pull/692))
- Bump `github.com/aws/aws-sdk-go` from 1.55.6 to 1.55.7 ([#696](https://github.com/opensearch-project/opensearch-go/pull/696))
- Bump `github.com/wI2L/jsondiff` from 0.6.1 to 0.7.0 ([#700](https://github.com/opensearch-project/opensearch-go/pull/700))

### Added

- Adds DataStream field to IndicesGetResp struct ([#701](https://github.com/opensearch-project/opensearch-go/pull/701))
- Adds `InnerHits` field to `SearchResp` ([#672](https://github.com/opensearch-project/opensearch-go/pull/672))
- Adds `FilterPath` param ([#673](https://github.com/opensearch-project/opensearch-go/pull/673))
- Adds `Aggregations` field to `MSearchResp` ([#690](https://github.com/opensearch-project/opensearch-go/pull/690))

### Changed

- Bump golang version to 1.22 ([#691](https://github.com/opensearch-project/opensearch-go/pull/691))
- Change ChangeCatRecoveryItemResp Byte fields from int to string ([#691](https://github.com/opensearch-project/opensearch-go/pull/691))
- Changed log formatted examples code ([#694](https://github.com/opensearch-project/opensearch-go/pull/694))
- Improve the error reporting of invalid body response ([#699](https://github.com/opensearch-project/opensearch-go/pull/699))

### Deprecated

### Removed

### Fixed

### Security

## [4.4.0]

### Added

- Adds `Highlight` field to `SearchHit` ([#654](https://github.com/opensearch-project/opensearch-go/pull/654))
- Adds `MatchedQueries` field to `SearchHit` ([#663](https://github.com/opensearch-project/opensearch-go/pull/663))
- Adds support for Opensearch 2.19 ([#668](https://github.com/opensearch-project/opensearch-go/pull/668))

### Changed

### Deprecated

### Removed

### Fixed

### Security

### Dependencies

- Bump `github.com/aws/aws-sdk-go` from 1.55.5 to 1.55.6 ([#657](https://github.com/opensearch-project/opensearch-go/pull/657))
- Bump `github.com/wI2L/jsondiff` from 0.6.0 to 0.6.1 ([#643](https://github.com/opensearch-project/opensearch-go/pull/643))
- Bump `github.com/aws/aws-sdk-go-v2` from 1.32.2 to 1.36.1 ([#664](https://github.com/opensearch-project/opensearch-go/pull/664))
- Bump `github.com/stretchr/testify` from 1.9.0 to 1.10.0 ([#644](https://github.com/opensearch-project/opensearch-go/pull/644))
- Bump `github.com/aws/aws-sdk-go-v2/config` from 1.27.43 to 1.29.6 ([#665](https://github.com/opensearch-project/opensearch-go/pull/665))

## [4.3.0]

### Added

- Adds ISM Alias action ([#615](https://github.com/opensearch-project/opensearch-go/pull/615))
- Adds support for opensearch 2.17 ([#623](https://github.com/opensearch-project/opensearch-go/pull/623))

### Changed

### Deprecated

### Removed

### Fixed

- Fix ISM Transition to omitempty Conditions field ([#609](https://github.com/opensearch-project/opensearch-go/pull/609))
- Fix ISM Allocation field types ([#609](https://github.com/opensearch-project/opensearch-go/pull/609))
- Fix ISM Error Notification types ([#612](https://github.com/opensearch-project/opensearch-go/pull/612))
- Fix signer receiving drained body on retries ([#620](https://github.com/opensearch-project/opensearch-go/pull/620))
- Fix Bulk Index Items not executing failure callbacks on bulk request failure ([#626](https://github.com/opensearch-project/opensearch-go/issues/626))

### Security

### Dependencies

- Bump `github.com/aws/aws-sdk-go-v2/config` from 1.27.31 to 1.27.43 ([#611](https://github.com/opensearch-project/opensearch-go/pull/611), [#630](https://github.com/opensearch-project/opensearch-go/pull/630), [#632](https://github.com/opensearch-project/opensearch-go/pull/632))
- Bump `github.com/aws/aws-sdk-go-v2` from 1.32.1 to 1.32.2 ([#631](https://github.com/opensearch-project/opensearch-go/pull/631))

## [4.2.0]

### Dependencies

- Bump `github.com/aws/aws-sdk-go-v2/config` from 1.27.23 to 1.27.31 ([#584](https://github.com/opensearch-project/opensearch-go/pull/584), [#588](https://github.com/opensearch-project/opensearch-go/pull/588), [#593](https://github.com/opensearch-project/opensearch-go/pull/593), [#605](https://github.com/opensearch-project/opensearch-go/pull/605))
- Bump `github.com/aws/aws-sdk-go` from 1.54.12 to 1.55.5 ([#583](https://github.com/opensearch-project/opensearch-go/pull/583), [#590](https://github.com/opensearch-project/opensearch-go/pull/590), [#595](https://github.com/opensearch-project/opensearch-go/pull/595), [#596](https://github.com/opensearch-project/opensearch-go/pull/596))

### Added

- Adds `Suggest` to `SearchResp` ([#602](https://github.com/opensearch-project/opensearch-go/pull/602))
- Adds `MaxScore` to `ScrollGetResp` ([#607](https://github.com/opensearch-project/opensearch-go/pull/607))

### Changed

- Split SnapshotGetResp into sub structs ([#603](https://github.com/opensearch-project/opensearch-go/pull/603))

### Deprecated

### Removed

- Remove workflow tests against gotip ([#604](https://github.com/opensearch-project/opensearch-go/pull/604))

### Fixed

### Security

## [4.1.0]

### Added

- Adds the `Routing` field in SearchHit interface. ([#516](https://github.com/opensearch-project/opensearch-go/pull/516))
- Adds the `SearchPipelines` field to `SearchParams` ([#532](https://github.com/opensearch-project/opensearch-go/pull/532))
- Adds support for OpenSearch 2.14 ([#552](https://github.com/opensearch-project/opensearch-go/pull/552))
- Adds the `Caches` field to Node stats ([#572](https://github.com/opensearch-project/opensearch-go/pull/572))
- Adds the `SeqNo` and `PrimaryTerm` fields in `SearchHit` ([#574](https://github.com/opensearch-project/opensearch-go/pull/574))
- Adds guide on configuring the client with retry and backoff ([#540](https://github.com/opensearch-project/opensearch-go/pull/540))
- Adds OpenSearch 2.15 to compatibility workflow test ([#575](https://github.com/opensearch-project/opensearch-go/pull/575))

### Changed

- Security roles get response struct has its own sub structs without omitempty ([#572](https://github.com/opensearch-project/opensearch-go/pull/572))

### Deprecated

### Removed

### Fixed

- Fixes empty request body on retry with compression enabled ([#543](https://github.com/opensearch-project/opensearch-go/pull/543))
- Fixes `Conditions` in `PolicyStateTransition` of ISM plugin ([#556](https://github.com/opensearch-project/opensearch-go/pull/556))
- Fixes integration test response validation when response is null ([#572](https://github.com/opensearch-project/opensearch-go/pull/572))
- Adjust security Role struct for FLS from string to []string ([#572](https://github.com/opensearch-project/opensearch-go/pull/572))
- Fixes wrong response parsing for indices mapping and recovery ([#572](https://github.com/opensearch-project/opensearch-go/pull/572))
- Fixes wrong response parsing for security get requests ([#572](https://github.com/opensearch-project/opensearch-go/pull/572))
- Fixes opensearchtransport ignores request context cancellation when `retryBackoff` is configured ([#540](https://github.com/opensearch-project/opensearch-go/pull/540))
- Fixes opensearchtransport sleeps unexpectedly after the last retry ([#540](https://github.com/opensearch-project/opensearch-go/pull/540))
- Improves ParseError response when server response is an unknown json ([#592](https://github.com/opensearch-project/opensearch-go/pull/592))

### Security

### Dependencies

- Bump `github.com/aws/aws-sdk-go` from 1.51.21 to 1.54.12 ([#534](https://github.com/opensearch-project/opensearch-go/pull/534), [#537](https://github.com/opensearch-project/opensearch-go/pull/537), [#538](https://github.com/opensearch-project/opensearch-go/pull/538), [#545](https://github.com/opensearch-project/opensearch-go/pull/545), [#554](https://github.com/opensearch-project/opensearch-go/pull/554), [#557](https://github.com/opensearch-project/opensearch-go/pull/557), [#563](https://github.com/opensearch-project/opensearch-go/pull/563), [#564](https://github.com/opensearch-project/opensearch-go/pull/564), [#570](https://github.com/opensearch-project/opensearch-go/pull/570), [#579](https://github.com/opensearch-project/opensearch-go/pull/579))
- Bump `github.com/wI2L/jsondiff` from 0.5.1 to 0.6.0 ([#535](https://github.com/opensearch-project/opensearch-go/pull/535), [#566](https://github.com/opensearch-project/opensearch-go/pull/566))
- Bump `github.com/aws/aws-sdk-go-v2/config` from 1.27.11 to 1.27.23 ([#546](https://github.com/opensearch-project/opensearch-go/pull/546), [#553](https://github.com/opensearch-project/opensearch-go/pull/553), [#558](https://github.com/opensearch-project/opensearch-go/pull/558), [#562](https://github.com/opensearch-project/opensearch-go/pull/562), [#567](https://github.com/opensearch-project/opensearch-go/pull/567), [#571](https://github.com/opensearch-project/opensearch-go/pull/571), [#577](https://github.com/opensearch-project/opensearch-go/pull/577))
- Bump `github.com/aws/aws-sdk-go-v2` from 1.27.0 to 1.30.1 ([#559](https://github.com/opensearch-project/opensearch-go/pull/559), [#578](https://github.com/opensearch-project/opensearch-go/pull/578))

## [4.0.0]

### Added

- Adds GlobalIOUsage struct for nodes stats ([#506](https://github.com/opensearch-project/opensearch-go/pull/506))
- Adds the `Explanation` field containing the document explain details to the `SearchHit` struct. ([#504](https://github.com/opensearch-project/opensearch-go/pull/504))
- Adds new error types ([#512](https://github.com/opensearch-project/opensearch-go/pull/506))
- Adds handling of non json errors to ParseError ([#512](https://github.com/opensearch-project/opensearch-go/pull/506))
- Adds the `Failures` field to opensearchapi structs ([#510](https://github.com/opensearch-project/opensearch-go/pull/510))
- Adds the `Fields` field containing the document fields to the `SearchHit` struct. ([#508](https://github.com/opensearch-project/opensearch-go/pull/508))
- Adds security plugin ([#507](https://github.com/opensearch-project/opensearch-go/pull/507))
- Adds security settings to container for security testing ([#507](https://github.com/opensearch-project/opensearch-go/pull/507))
- Adds cluster.get-certs to copy admin certs out of the container ([#507](https://github.com/opensearch-project/opensearch-go/pull/507))
- Adds the `Fields` field containing stored fields to the `DocumentGetResp` struct ([#526](https://github.com/opensearch-project/opensearch-go/pull/526))
- Adds ism plugin ([#524](https://github.com/opensearch-project/opensearch-go/pull/524))

### Changed

- Uses docker compose v2 instead of v1 ([#506](https://github.com/opensearch-project/opensearch-go/pull/506))
- Updates go version to 1.21 ([#509](https://github.com/opensearch-project/opensearch-go/pull/509))
- Moves Error structs from opensearchapi to opensearch package ([#512](https://github.com/opensearch-project/opensearch-go/pull/506))
- Moves parseError function from opensearchapi to opensearch package as ParseError ([#512](https://github.com/opensearch-project/opensearch-go/pull/506))
- Changes ParseError function to do type assertion to determine error type ([#512](https://github.com/opensearch-project/opensearch-go/pull/506))
- Removes unused structs and functions from opensearch ([#517](https://github.com/opensearch-project/opensearch-go/pull/517))
- Adjusts and extent opensearch tests for better coverage ([#517](https://github.com/opensearch-project/opensearch-go/pull/517))
- Bumps codecov action version to v4 ([#517](https://github.com/opensearch-project/opensearch-go/pull/517))
- Changes bulk error/reason field and some cat response fields to pointer as they can be nil ([#510](https://github.com/opensearch-project/opensearch-go/pull/510))
- Adjust workflows to work with security plugin ([#507](https://github.com/opensearch-project/opensearch-go/pull/507))
- Updates USER_GUIDE.md and add samples ([#518](https://github.com/opensearch-project/opensearch-go/pull/518))
- Updates opensearchtransport.Client to use pooled gzip writer and buffer ([#521](https://github.com/opensearch-project/opensearch-go/pull/521))
- Use go:build tags for testing ([#52?](https://github.com/opensearch-project/opensearch-go/pull/52?))

### Deprecated

### Removed

### Fixed

- Fixes search request missing a slash when no indices are given ([#470](https://github.com/opensearch-project/opensearch-go/pull/469))
- Fixes opensearchtransport check for nil response body ([#517](https://github.com/opensearch-project/opensearch-go/pull/517))

### Security

### Dependencies

- Bumps `github.com/aws/aws-sdk-go-v2` from 1.25.3 to 1.26.1
- Bumps `github.com/wI2L/jsondiff` from 0.4.0 to 0.5.1
- Bumps `github.com/aws/aws-sdk-go` from 1.50.36 to 1.51.21
- Bumps `github.com/aws/aws-sdk-go-v2/config` from 1.27.7 to 1.27.11

## [3.1.0]

### Added

- Adds new struct fields introduced in OpenSearch 2.12 ([#482](https://github.com/opensearch-project/opensearch-go/pull/482))
- Adds initial admin password environment variable and CI changes to support 2.12.0 release ([#449](https://github.com/opensearch-project/opensearch-go/pull/449))
- Adds `merge_id` field for indices segment request ([#488](https://github.com/opensearch-project/opensearch-go/pull/488))

### Changed

- Updates workflow action versions ([#488](https://github.com/opensearch-project/opensearch-go/pull/488))
- Changes integration tests to work with secure and unsecure OpenSearch ([#488](https://github.com/opensearch-project/opensearch-go/pull/488))
- Moves functions from `opensearch/internal/test` to `opensearchutil/testutil` for shared test utilities ([#488](https://github.com/opensearch-project/opensearch-go/pull/488))
- Changes `custom_foldername` field to pointer as it can be `null` ([#488](https://github.com/opensearch-project/opensearch-go/pull/488))
- Changs cat indices Primary and Replica field to pointer as it can be `null` ([#488](https://github.com/opensearch-project/opensearch-go/pull/488))
- Replaces `ioutil` with `io` in examples and integration tests [#495](https://github.com/opensearch-project/opensearch-go/pull/495)

### Fixed

- Fix incorrect SigV4 `x-amz-content-sha256` with AWS SDK v1 requests without a body ([#496](https://github.com/opensearch-project/opensearch-go/pull/496))

### Dependencies

- Bumps `github.com/aws/aws-sdk-go` from 1.48.13 to 1.50.36
- Bumps `github.com/aws/aws-sdk-go-v2/config` from 1.25.11 to 1.27.7
- Bumps `github.com/stretchr/testify` from 1.8.4 to 1.9.0

## [3.0.0]

### Added

- Adds `Err()` function to Response for detailed errors ([#246](https://github.com/opensearch-project/opensearch-go/pull/246))
- Adds golangci-lint as code analysis tool ([#313](https://github.com/opensearch-project/opensearch-go/pull/313))
- Adds govulncheck to check for go vulnerablities ([#405](https://github.com/opensearch-project/opensearch-go/pull/405))
- Adds opensearchapi with new client and function structure ([#421](https://github.com/opensearch-project/opensearch-go/pull/421))
- Adds integration tests for all opensearchapi functions ([#421](https://github.com/opensearch-project/opensearch-go/pull/421))
- Adds guide on making raw JSON REST requests ([#399](https://github.com/opensearch-project/opensearch-go/pull/399))
- Adds IPV6 support in the DiscoverNodes method ([#458](https://github.com/opensearch-project/opensearch-go/issues/458))

### Changed

- Removes the need for double error checking ([#246](https://github.com/opensearch-project/opensearch-go/pull/246))
- Updates and adjusted golangci-lint, solve linting complains for signer ([#352](https://github.com/opensearch-project/opensearch-go/pull/352))
- Solves linting complains for opensearchtransport ([#353](https://github.com/opensearch-project/opensearch-go/pull/353))
- Updates Developer guide to include docker build instructions ([#385](https://github.com/opensearch-project/opensearch-go/pull/385))
- Tests against version 2.9.0, 2.10.0, run tests in all branches, changes integration tests to wait for OpenSearch to start ([#392](https://github.com/opensearch-project/opensearch-go/pull/392))
- Makefile: uses docker golangci-lint, run integration test on `.` folder, change coverage generation ([#392](https://github.com/opensearch-project/opensearch-go/pull/392))
- golangci-lint: updates rules and fail when issues are found ([#421](https://github.com/opensearch-project/opensearch-go/pull/421))
- go: updates to golang version 1.20 ([#421](https://github.com/opensearch-project/opensearch-go/pull/421))
- guids: updates to work for the new opensearchapi ([#421](https://github.com/opensearch-project/opensearch-go/pull/421))
- Adjusts tests to new opensearchapi functions and structs ([#421](https://github.com/opensearch-project/opensearch-go/pull/421))
- Changes codecov to comment code coverage to each PR ([#410](https://github.com/opensearch-project/opensearch-go/pull/410))
- Changes module version from v2 to v3 ([#444](https://github.com/opensearch-project/opensearch-go/pull/444))
- Handle unexpected non-json errors with the response body ([#523](https://github.com/opensearch-project/opensearch-go/pull/523))

### Deprecated

- Deprecates legacy API `/_template` ([#390](https://github.com/opensearch-project/opensearch-go/pull/390))

### Removed

- Removes all old opensearchapi functions ([#421](https://github.com/opensearch-project/opensearch-go/pull/421))
- Removes `/internal/build` code and folders ([#421](https://github.com/opensearch-project/opensearch-go/pull/421))

### Fixed

- Corrects AWSv4 signature on DataStream `Stats` with no index name specified ([#338](https://github.com/opensearch-project/opensearch-go/pull/338))
- Fixes GetSourceRequest `Source` field and deprecated the `Source` parameter ([#402](https://github.com/opensearch-project/opensearch-go/pull/402))
- Corrects developer guide summary with golang version 1.20 ([#434](https://github.com/opensearch-project/opensearch-go/pull/434))

### Dependencies

- Bumps `github.com/aws/aws-sdk-go` from 1.44.263 to 1.48.13
- Bumps `github.com/aws/aws-sdk-go-v2` from 1.18.0 to 1.23.5
- Bumps `github.com/aws/aws-sdk-go-v2/config` from 1.18.25 to 1.25.11
- Bumps `github.com/stretchr/testify` from 1.8.2 to 1.8.4
- Bumps `golang.org/x/net` from 0.7.0 to 0.17.0
- Bumps `github.com/golangci/golangci-lint-action` from 1.53.3 to 1.54.2

## [2.3.0]

### Added

- Adds implementation of Data Streams API ([#257](https://github.com/opensearch-project/opensearch-go/pull/257))
- Adds Point In Time API ([#253](https://github.com/opensearch-project/opensearch-go/pull/253))
- Adds InfoResp type ([#253](https://github.com/opensearch-project/opensearch-go/pull/253))
- Adds markdown linter ([#261](https://github.com/opensearch-project/opensearch-go/pull/261))
- Adds testcases to check upsert functionality ([#269](https://github.com/opensearch-project/opensearch-go/pull/269))
- Adds @Jakob3xD to co-maintainers ([#270](https://github.com/opensearch-project/opensearch-go/pull/270))
- Adds dynamic type to \_source field ([#285](https://github.com/opensearch-project/opensearch-go/pull/285))
- Adds testcases for Document API ([#285](https://github.com/opensearch-project/opensearch-go/pull/285))
- Adds `index_lifecycle` guide ([#287](https://github.com/opensearch-project/opensearch-go/pull/287))
- Adds `bulk` guide ([#292](https://github.com/opensearch-project/opensearch-go/pull/292))
- Adds `search` guide ([#291](https://github.com/opensearch-project/opensearch-go/pull/291))
- Adds `document_lifecycle` guide ([#290](https://github.com/opensearch-project/opensearch-go/pull/290))
- Adds `index_template` guide ([#289](https://github.com/opensearch-project/opensearch-go/pull/289))
- Adds `advanced_index_actions` guide ([#288](https://github.com/opensearch-project/opensearch-go/pull/288))
- Adds testcases to check UpdateByQuery functionality ([#304](https://github.com/opensearch-project/opensearch-go/pull/304))
- Adds additional timeout after cluster start ([#303](https://github.com/opensearch-project/opensearch-go/pull/303))
- Adds docker healthcheck to auto restart the container ([#315](https://github.com/opensearch-project/opensearch-go/pull/315))

### Changed

- Uses `[]string` instead of `string` in `SnapshotDeleteRequest` ([#237](https://github.com/opensearch-project/opensearch-go/pull/237))
- Updates workflows to reduce CI time, consolidate OpenSearch versions, update compatibility matrix ([#242](https://github.com/opensearch-project/opensearch-go/pull/242))
- Moves @svencowart to emeritus maintainers ([#270](https://github.com/opensearch-project/opensearch-go/pull/270))
- Reads, closes and replaces the http Response Body ([#300](https://github.com/opensearch-project/opensearch-go/pull/300))

### Fixed

- Corrects curl logging to emit the correct URL destination ([#101](https://github.com/opensearch-project/opensearch-go/pull/101))

### Dependencies

- Bumps `github.com/aws/aws-sdk-go` from 1.44.180 to 1.44.263
- Bumps `github.com/aws/aws-sdk-go-v2` from 1.17.4 to 1.18.0
- Bumps `github.com/aws/aws-sdk-go-v2/config` from 1.18.8 to 1.18.25
- Bumps `github.com/stretchr/testify` from 1.8.1 to 1.8.2

## [2.2.0]

### Added

- Adds Github workflow for changelog verification ([#172](https://github.com/opensearch-project/opensearch-go/pull/172))
- Adds Go Documentation link for the client ([#182](https://github.com/opensearch-project/opensearch-go/pull/182))
- Adds support for Amazon OpenSearch Serverless ([#216](https://github.com/opensearch-project/opensearch-go/pull/216))

### Removed

- Removes info call before performing every request ([#219](https://github.com/opensearch-project/opensearch-go/pull/219))

### Fixed

- Renames the sequence number struct tag to if_seq_no to fix optimistic concurrency control ([#166](https://github.com/opensearch-project/opensearch-go/pull/166))
- Fixes `RetryOnConflict` on bulk indexer ([#215](https://github.com/opensearch-project/opensearch-go/pull/215))

### Dependencies

- Bumps `github.com/aws/aws-sdk-go-v2` from 1.17.1 to 1.17.3
- Bumps `github.com/aws/aws-sdk-go-v2/config` from 1.17.10 to 1.18.8
- Bumps `github.com/aws/aws-sdk-go` from 1.44.176 to 1.44.180
- Bumps `github.com/aws/aws-sdk-go` from 1.44.132 to 1.44.180
- Bumps `github.com/stretchr/testify` from 1.8.0 to 1.8.1
- Bumps `github.com/aws/aws-sdk-go` from 1.44.45 to 1.44.132

[4.7.3]: https://github.com/opensearch-project/opensearch-go/compare/v4.7.2...v4.7.3
[4.7.2]: https://github.com/opensearch-project/opensearch-go/compare/v4.7.1...v4.7.2
[4.7.1]: https://github.com/opensearch-project/opensearch-go/compare/v4.7.0...v4.7.1
[4.7.0]: https://github.com/opensearch-project/opensearch-go/compare/v4.6.0...v4.7.0
[4.6.0]: https://github.com/opensearch-project/opensearch-go/compare/v4.5.0...v4.6.0
[4.5.0]: https://github.com/opensearch-project/opensearch-go/compare/v4.4.0...v4.5.0
[4.4.0]: https://github.com/opensearch-project/opensearch-go/compare/v4.3.0...v4.4.0
[4.3.0]: https://github.com/opensearch-project/opensearch-go/compare/v4.2.0...v4.3.0
[4.2.0]: https://github.com/opensearch-project/opensearch-go/compare/v4.1.0...v4.2.0
[4.1.0]: https://github.com/opensearch-project/opensearch-go/compare/v4.0.0...v4.1.0
[4.0.0]: https://github.com/opensearch-project/opensearch-go/compare/v3.1.0...v4.0.0
[3.1.0]: https://github.com/opensearch-project/opensearch-go/compare/v3.0.0...v3.1.0
[3.0.0]: https://github.com/opensearch-project/opensearch-go/compare/v2.3.0...v3.0.0
[2.3.0]: https://github.com/opensearch-project/opensearch-go/compare/v2.2.0...v2.3.0
[2.2.0]: https://github.com/opensearch-project/opensearch-go/compare/v2.1.0...v2.2.0
[2.1.0]: https://github.com/opensearch-project/opensearch-go/compare/v2.0.1...v2.1.0
[2.0.1]: https://github.com/opensearch-project/opensearch-go/compare/v2.0.0...v2.0.1
[2.0.0]: https://github.com/opensearch-project/opensearch-go/compare/v1.1.0...v2.0.0
[1.0.0]: https://github.com/opensearch-project/opensearch-go/compare/v1.0.0...v1.1.0
