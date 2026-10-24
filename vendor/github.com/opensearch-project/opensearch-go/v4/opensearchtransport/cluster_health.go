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

package opensearchtransport

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"time"
)

const clusterStatusRed = "red"

// NodeStatsResponse represents the response from GET /_nodes/_local/stats/jvm,breaker,thread_pool.
// Only the "nodes" map is used; the top-level "_nodes" and "cluster_name" fields are ignored.
// All fields are present in OpenSearch 1.3.0+.
type NodeStatsResponse struct {
	Nodes map[string]NodeStats `json:"nodes"`
}

// NodeStats represents per-node statistics used for load shedding and
// congestion window updates.
type NodeStats struct {
	JVM         JVMStats                   `json:"jvm"`
	Breakers    map[string]BreakerStats    `json:"breakers"`
	ThreadPools map[string]ThreadPoolStats `json:"thread_pool,omitempty"`
}

// JVMStats contains JVM-level statistics.
type JVMStats struct {
	Mem JVMMemStats `json:"mem"`
}

// JVMMemStats contains JVM memory statistics.
type JVMMemStats struct {
	HeapUsedPercent int `json:"heap_used_percent"`
}

// BreakerStats contains circuit breaker statistics for a single breaker.
// OpenSearch exposes breakers for: fielddata, request, in_flight_requests, accounting, parent.
type BreakerStats struct {
	LimitSizeInBytes     int64 `json:"limit_size_in_bytes"`
	EstimatedSizeInBytes int64 `json:"estimated_size_in_bytes"`
	Tripped              int64 `json:"tripped"`
}

// scheduleNodeStats starts a background ticker that periodically polls node stats
// for load shedding. When nodeStatsIntervalAuto is true, the interval is recalculated
// on each tick based on cluster size:
//
//	interval = clamp(liveNodes * clientsPerServer / healthCheckRate, 5s, 30s)
//
// Cancelled by c.ctx.
func (c *Client) scheduleNodeStats() {
	go func() {
		interval := c.nodeStatsInterval
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-c.ctx.Done():
				return
			case <-ticker.C:
				c.pollNodeStats()

				if c.nodeStatsIntervalAuto {
					newInterval := c.calculateNodeStatsInterval()
					if newInterval != interval {
						interval = newInterval
						ticker.Reset(interval)
					}
				}
			}
		}
	}()
}

// calculateNodeStatsInterval computes the polling interval based on the current
// cluster size and the configured health check rate.
//
//	interval = clamp(liveNodes * clientsPerServer / healthCheckRate, 5s, 30s)
func (c *Client) calculateNodeStatsInterval() time.Duration {
	liveNodes := c.countReadyNodes()
	if liveNodes <= 0 {
		liveNodes = 1
	}

	c.mu.RLock()
	clientsPerServer := c.clientsPerServer
	healthCheckRate := c.healthCheckRate
	c.mu.RUnlock()

	intervalSec := float64(liveNodes) * clientsPerServer / healthCheckRate
	interval := min(
		max(
			time.Duration(intervalSec*float64(time.Second)), defaultNodeStatsIntervalMin), defaultNodeStatsIntervalMax)

	return interval
}

// pollNodeStats polls node stats for all connections in the connection pool,
// regardless of pool type. Updates per-pool AIMD congestion windows and
// evaluates node-level overload state.
//
// Connection pointers are shared across all policies (see guides/routing.md
// "Shared Connections Across Policies"). Polling a connection updates the
// authoritative atomic state on the Connection itself, which is immediately
// visible to every policy.
func (c *Client) pollNodeStats() {
	c.mu.RLock()
	cp := c.mu.connectionPool
	c.mu.RUnlock()

	switch pool := cp.(type) {
	case *multiServerPool:
		// Snapshot connections to evaluate.
		pool.RLock()
		snapshot := make([]*Connection, 0, len(pool.mu.ready)+len(pool.mu.dead))
		snapshot = append(snapshot, pool.mu.ready...)
		// Include overload-demoted dead connections so we can promote them if recovered.
		for _, conn := range pool.mu.dead {
			if conn.loadConnState().lifecycle().has(lcOverloaded) {
				snapshot = append(snapshot, conn)
			}
		}
		pool.RUnlock()

		samples := make([]nodeSearchSample, 0, len(snapshot))
		for _, conn := range snapshot {
			if sample, ok := c.fetchAndEvaluateNodeStats(conn, pool); ok {
				samples = append(samples, sample)
			}
		}
		if len(samples) > 0 {
			c.clusterSearch.update(samples)
		}

	case *singleServerPool:
		// Single-node pool: poll the connection for AIMD updates.
		// Overload demotion is not applicable (no pool to demote within).
		conn := pool.connection
		if conn != nil {
			if sample, ok := c.fetchAndEvaluateNodeStats(conn, nil); ok {
				c.clusterSearch.update([]nodeSearchSample{sample})
			}
		}
	}
}

// fetchAndEvaluateNodeStats polls a single node's stats, updates per-pool
// congestion windows (AIMD), and evaluates node-level overload state.
//
// Two data sources are combined for the overload decision:
//  1. Node-level stats from GET /_nodes/_local/stats/jvm,breaker,thread_pool (fetched here)
//  2. Cluster health from conn.mu.clusterHealth (already populated by clusterHealthCheck)
//
// Thread pool stats drive per-pool AIMD congestion control via [updatePoolCongestion].
// This avoids redundant HTTP calls since the cluster health check already collects
// cluster health data during normal health check cycles.
//
// Returns the search thread pool sample for cluster-wide aggregation (see
// [clusterSearchAIMD]). ok is false when the poll fails or the node has no
// search pool data.
func (c *Client) fetchAndEvaluateNodeStats(conn *Connection, pool *multiServerPool) (nodeSearchSample, bool) {
	var sample nodeSearchSample

	ctx, cancel := context.WithTimeout(c.ctx, c.healthCheckTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "/_nodes/_local/stats/jvm,breaker,thread_pool", nil)
	if err != nil {
		return sample, false
	}

	c.setReqURL(conn.URL, req)
	c.setReqAuth(conn.URL, req)
	c.setReqUserAgent(req)

	if c.healthCheckRequestModifier != nil {
		c.healthCheckRequestModifier(req)
	}

	res, err := c.transport.RoundTrip(req)
	if err != nil {
		// Can't reach node -- if overload-demoted, clear overloaded flag so the normal
		// resurrection scheduler can take over (the node may actually be down, not just overloaded).
		if conn.loadConnState().lifecycle().has(lcOverloaded) {
			conn.mu.Lock()
			//nolint:errcheck // lock held; only errLifecycleNoop possible
			conn.casLifecycle(
				conn.loadConnState(), 0,
				lcDead|lcNeedsWarmup,
				lcReady|lcActive|lcStandby,
			)
			conn.mu.Unlock()
			if dl := loadDebugLogger(); dl != nil {
				dl.Logf("Stats poll failed for %q, clearing overloaded flag (resurrection scheduler will handle): %v\n", conn.URL, err)
			}
		}
		return sample, false
	}
	if res.Body != nil {
		// Drain on close so http.Transport can reuse the connection. This is a
		// raw RoundTrip path with no Perform buffering safety net, so closing a
		// partially-read body (e.g. on the non-200 early return below) would
		// otherwise defeat keep-alive. On the success path io.ReadAll has
		// already reached EOF, so the drain is a cheap no-op.
		defer func() {
			//nolint:errcheck // best-effort drain before close on a cleanup path
			io.Copy(io.Discard, res.Body)
			res.Body.Close()
		}()
	}

	if res.StatusCode != http.StatusOK || res.Body == nil {
		return sample, false
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return sample, false
	}

	var stats NodeStatsResponse
	if err := json.Unmarshal(body, &stats); err != nil {
		return sample, false
	}

	// The response contains a map with a single node entry (since we used _local)
	var nodeStats *NodeStats
	for _, ns := range stats.Nodes {
		nodeStats = &ns
		break
	}
	if nodeStats == nil {
		return sample, false
	}

	// Update per-pool congestion windows (AIMD) from thread pool stats.
	updatePoolCongestion(conn, nodeStats.ThreadPools)

	// Extract search pool stats for cluster-wide MCSR aggregation.
	var ok bool
	if searchTP, found := nodeStats.ThreadPools[poolSearch]; found {
		var searchMaxCwnd int32
		if searchPC := conn.pools.get(poolSearch); searchPC != nil {
			searchPC.mu.Lock()
			searchMaxCwnd = searchPC.mu.maxCwnd
			searchPC.mu.Unlock()
		}
		sample = nodeSearchSample{
			conn:    conn,
			stats:   searchTP,
			maxCwnd: searchMaxCwnd,
		}
		ok = true
	}

	overloaded := c.evaluateOverload(conn, nodeStats)

	// Overload demotion/promotion requires a multiServerPool (single-node pools
	// pass nil since there is nowhere to demote to).
	if pool == nil {
		return sample, ok
	}

	wasOverloaded := conn.loadConnState().lifecycle().has(lcOverloaded)

	switch {
	case overloaded && !wasOverloaded:
		pool.demoteOverloaded(conn)
	case !overloaded && wasOverloaded:
		pool.promoteFromOverloaded(conn)
	}

	return sample, ok
}

// evaluateOverload checks both node-level stats and cluster health against overload thresholds.
// Returns true if any overload condition is met.
//
// # Node-level checks (from /_nodes/_local/stats/jvm,breaker)
//
//   - JVM heap_used_percent >= overloadedHeapThreshold (default 85%)
//   - Any circuit breaker's estimated_size / limit_size >= overloadedBreakerRatio (default 0.90)
//   - Any circuit breaker's tripped count increased since the last poll (delta detection)
//
// # Cluster health checks (from conn.mu.clusterHealth, populated by clusterHealthCheck)
//
//   - Cluster status is "red" (data loss or primary shards unassigned)
//
// The cluster health data is a free signal -- it's already collected during periodic health
// checks via /_cluster/health?local=true without any additional HTTP calls.
//
// Updates conn.mu.lastBreakerTripped for delta detection on next poll.
func (c *Client) evaluateOverload(conn *Connection, stats *NodeStats) bool {
	overloaded := false

	// --- Cluster health checks (reuse data from clusterHealthCheck) ---

	conn.mu.RLock()
	health := conn.mu.clusterHealth
	conn.mu.RUnlock()

	if health != nil && health.Status == clusterStatusRed {
		if dl := loadDebugLogger(); dl != nil {
			dl.Logf("Node %q overloaded: cluster status is red\n", conn.URL)
		}
		overloaded = true
	}

	// --- Node-level stats checks ---

	// JVM heap usage
	if stats.JVM.Mem.HeapUsedPercent >= c.overloadedHeapThreshold {
		if dl := loadDebugLogger(); dl != nil {
			dl.Logf("Node %q overloaded: heap_used_percent=%d >= threshold=%d\n",
				conn.URL, stats.JVM.Mem.HeapUsedPercent, c.overloadedHeapThreshold)
		}
		overloaded = true
	}

	// Circuit breakers
	conn.mu.Lock()
	if conn.mu.lastBreakerTripped == nil {
		conn.mu.lastBreakerTripped = make(map[string]int64, len(stats.Breakers))
	}

	for name, breaker := range stats.Breakers {
		// Size ratio check (instantaneous)
		if breaker.LimitSizeInBytes > 0 {
			ratio := float64(breaker.EstimatedSizeInBytes) / float64(breaker.LimitSizeInBytes)
			if ratio >= c.overloadedBreakerRatio {
				if dl := loadDebugLogger(); dl != nil {
					dl.Logf("Node %q overloaded: breaker %q size ratio=%.3f >= threshold=%.3f\n",
						conn.URL, name, ratio, c.overloadedBreakerRatio)
				}
				overloaded = true
			}
		}

		// Trip delta check (cumulative)
		prevTripped, existed := conn.mu.lastBreakerTripped[name]
		conn.mu.lastBreakerTripped[name] = breaker.Tripped

		if existed && breaker.Tripped > prevTripped {
			if dl := loadDebugLogger(); dl != nil {
				dl.Logf("Node %q overloaded: breaker %q tripped %d times since last poll (prev=%d, now=%d)\n",
					conn.URL, name, breaker.Tripped-prevTripped, prevTripped, breaker.Tripped)
			}
			overloaded = true
		}
	}
	conn.mu.Unlock()

	return overloaded
}

// scheduleClusterHealthRefresh starts a background goroutine that periodically refreshes
// /_cluster/health?local=true data on ready connections that support cluster health.
// The refresh interval is dynamically calculated using:
//
//	refreshInterval = clamp(liveNodes * clientsPerServer / healthCheckRate, 5s, 5min)
//
// Single-node clusters skip refresh entirely since health data cannot influence routing.
// Cancelled by c.ctx.
func (c *Client) scheduleClusterHealthRefresh() {
	go func() {
		interval := c.calculateClusterHealthRefreshInterval()
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-c.ctx.Done():
				return
			case <-ticker.C:
				c.pollClusterHealth()

				// Recalculate interval after each poll (node count may have changed)
				newInterval := c.calculateClusterHealthRefreshInterval()
				if newInterval != interval {
					interval = newInterval
					ticker.Reset(interval)
				}
			}
		}
	}()
}

// calculateClusterHealthRefreshInterval computes the polling interval based on
// the current cluster size and the configured health check rate.
//
//	interval = clamp(liveNodes * clientsPerServer / healthCheckRate, 5s, 5min)
func (c *Client) calculateClusterHealthRefreshInterval() time.Duration {
	liveNodes := c.countReadyNodes()
	if liveNodes <= 0 {
		liveNodes = 1 // Prevent zero interval; will be short-circuited by single-node check in pollClusterHealth
	}

	c.mu.RLock()
	clientsPerServer := c.clientsPerServer
	healthCheckRate := c.healthCheckRate
	c.mu.RUnlock()

	intervalSec := float64(liveNodes) * clientsPerServer / healthCheckRate
	interval := min(
		// Clamp to [min, max]
		max(

			time.Duration(intervalSec*float64(time.Second)), defaultClusterHealthRefreshMin), defaultClusterHealthRefreshMax)

	return interval
}

// countReadyNodes returns the number of ready connections in the current pool.
func (c *Client) countReadyNodes() int {
	c.mu.RLock()
	pool := c.mu.connectionPool
	c.mu.RUnlock()

	switch p := pool.(type) {
	case *singleServerPool:
		return 1
	case *multiServerPool:
		p.mu.RLock()
		count := len(p.mu.ready)
		p.mu.RUnlock()
		return count
	default:
		return 0
	}
}

// pollClusterHealth refreshes /_cluster/health?local=true on all ready connections that
// have HasClusterHealth(). Skips single-node clusters and connections without cluster health.
func (c *Client) pollClusterHealth() {
	c.mu.RLock()
	pool := c.mu.connectionPool
	c.mu.RUnlock()

	// Skip single-node clusters: no value in refreshing health when we cannot route away.
	switch p := pool.(type) {
	case *singleServerPool:
		return
	case *multiServerPool:
		p.mu.RLock()
		totalNodes := len(p.mu.ready) + len(p.mu.dead)
		p.mu.RUnlock()
		if totalNodes <= 1 {
			return
		}
	default:
		return
	}

	conns := c.snapshotClusterHealthConnections()
	if len(conns) == 0 {
		return
	}

	for _, conn := range conns {
		c.refreshClusterHealth(conn)
	}
}

// snapshotClusterHealthConnections returns ready connections that have HasClusterHealth()
// from the current connection pool.
func (c *Client) snapshotClusterHealthConnections() []*Connection {
	c.mu.RLock()
	pool := c.mu.connectionPool
	c.mu.RUnlock()

	if pool == nil {
		return nil
	}

	p, ok := pool.(*multiServerPool)
	if !ok {
		return nil
	}

	p.mu.RLock()
	snapshot := make([]*Connection, 0, len(p.mu.ready))
	for _, conn := range p.mu.ready {
		if conn.hasClusterHealth() {
			snapshot = append(snapshot, conn)
		}
	}
	p.mu.RUnlock()

	return snapshot
}

// refreshClusterHealth performs a single /_cluster/health?local=true request against
// the given connection and updates conn.mu.clusterHealth.
//
// Error handling:
//   - 200: Parse response, store ClusterHealthLocal on connection, update timestamp.
//   - 401/403: Permission revoked at runtime. Reset cluster health lifecycle bits to pending,
//     zero out clusterHealth. The connection will fall back to baseline GET / health checks.
//   - Transient errors (network, 5xx): Log and skip. The next poll cycle will retry.

// ClusterHealthLocal represents the response from GET /_cluster/health?local=true.
// The local=true parameter causes the request to be served from the connected node's
// local cluster state cache rather than requiring a round-trip to the cluster-manager node,
// making it suitable for fast, lightweight health probes.
//
// This endpoint requires the cluster:monitor/health action privilege. If the OpenSearch
// Security plugin is installed, requests without valid credentials receive 401 Unauthorized,
// and authenticated users who lack the privilege receive 403 Forbidden. To grant the
// minimum required permission, create a role with:
//
//	health_check_role:
//	  cluster_permissions:
//	    - "cluster:monitor/health"
//
// The client automatically detects whether this permission is available and falls back to
// GET / when it is not. See [Client.DefaultHealthCheck] for the capability detection lifecycle.
//
// All fields are present in OpenSearch 1.3.0+.
type ClusterHealthLocal struct {
	ClusterName                 string  `json:"cluster_name"`
	Status                      string  `json:"status"` // "green", "yellow", "red"
	TimedOut                    bool    `json:"timed_out"`
	NumberOfNodes               int     `json:"number_of_nodes"`
	NumberOfDataNodes           int     `json:"number_of_data_nodes"`
	ActivePrimaryShards         int     `json:"active_primary_shards"`
	ActiveShards                int     `json:"active_shards"`
	RelocatingShards            int     `json:"relocating_shards"`
	InitializingShards          int     `json:"initializing_shards"`
	UnassignedShards            int     `json:"unassigned_shards"`
	DelayedUnassignedShards     int     `json:"delayed_unassigned_shards"`
	NumberOfPendingTasks        int     `json:"number_of_pending_tasks"`
	NumberOfInFlightFetch       int     `json:"number_of_in_flight_fetch"`
	TaskMaxWaitingInQueueMillis int     `json:"task_max_waiting_in_queue_millis"`
	ActiveShardsPercentAsNumber float64 `json:"active_shards_percent_as_number"`
	// Added in OpenSearch 2.4.0
	DiscoveredClusterManager *bool `json:"discovered_cluster_manager,omitempty"`
}

// healthCheckInfo represents the minimal structure needed to extract version from health check response
type healthCheckInfo struct {
	Version struct {
		Number string `json:"number"`
	} `json:"version"`
}

// ClusterHealth returns the most recent cluster health snapshot for this connection, or nil
// if cluster health has not been probed or is unavailable.
func (c *Connection) ClusterHealth() *ClusterHealthLocal {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.mu.clusterHealth
}

func (c *Client) refreshClusterHealth(conn *Connection) {
	applyModifier := c.healthCheckRequestModifier

	health, statusCode, err := c.fetchClusterHealth(c.ctx, conn.URL, applyModifier)
	if err != nil {
		if dl := loadDebugLogger(); dl != nil {
			dl.Logf("Cluster health refresh failed for %q: %v\n", conn.URL, err)
		}
		return
	}

	switch {
	case statusCode == http.StatusOK && health != nil:
		conn.mu.Lock()
		storeClusterHealth(conn, health)
		conn.setLifecycleBit(lcClusterHealthProbed | lcClusterHealthAvailable) //nolint:errcheck // noop is fine
		conn.mu.Unlock()

	case statusCode == http.StatusUnauthorized || statusCode == http.StatusForbidden:
		resetClusterHealth(conn)

		if dl := loadDebugLogger(); dl != nil {
			dl.Logf("Cluster health refresh got %d for %q, resetting to pending\n",
				statusCode, conn.URL)
		}

	default:
		// Unexpected status code; skip update and retry on next poll cycle.
	}
}
