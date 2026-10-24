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
	"errors"
	"fmt"
	"io"
	"math"
	"math/rand/v2"
	"net"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"
)

// errDiscoveryEmpty is returned by getNodesInfo when the /_nodes response
// reports zero successful nodes. Callers should retry; the existing
// connection pool is left untouched.
var errDiscoveryEmpty = errors.New("discovery returned zero successful nodes")

// errDiscoveryInterrupted is returned to a waiting caller when the discovery
// it blocked on was aborted by the cancellation of the goroutine that ran it,
// rather than by the waiter's own context. The waiter's context is still
// healthy, so it must not inherit the runner's context.Canceled (which would
// look like the waiter's own cancellation and could suppress a legitimate
// retry); it receives this sentinel instead and may retry discovery.
var errDiscoveryInterrupted = errors.New("discovery interrupted by another caller's cancellation")

// Node role constants matching upstream OpenSearch server definitions.
const (
	// RoleData nodes store and retrieve data, perform indexing, searching, and
	// aggregating operations on local shards. Available since OpenSearch 1.0.
	// See: https://docs.opensearch.org/latest/install-and-configure/configuring-opensearch/configuration-system/
	RoleData = "data"

	// RoleIngest nodes pre-process data before storing via ingest pipelines.
	// Available since OpenSearch 1.0.
	// See: https://docs.opensearch.org/latest/install-and-configure/configuring-opensearch/configuration-system/
	RoleIngest = "ingest"

	// RoleClusterManager nodes manage overall cluster operations, cluster state,
	// index creation/deletion, node health checks, and shard allocation.
	// Available since OpenSearch 1.0.
	// See: https://docs.opensearch.org/latest/install-and-configure/configuring-opensearch/configuration-system/
	RoleClusterManager = "cluster_manager"

	// RoleRemoteClusterClient nodes can act as cross-cluster clients and connect
	// to remote clusters. Available since OpenSearch 1.0 (based on Elasticsearch 7.8.0).
	// See: https://docs.opensearch.org/latest/install-and-configure/configuring-opensearch/configuration-system/
	RoleRemoteClusterClient = "remote_cluster_client"

	// RoleSearch nodes are dedicated to hosting search replica shards, allowing
	// separation of search workloads from indexing workloads. Added in OpenSearch 3.0.0-beta1.
	//
	// IMPORTANT: This role cannot be combined with any other node role. This restriction
	// has been enforced since OpenSearch 3.0.0-beta1.
	//
	// For searchable snapshots, use RoleWarm instead (recommended in OpenSearch 3.0+).
	// See: https://docs.opensearch.org/latest/tuning-your-cluster/separate-index-and-search-workloads/
	RoleSearch = "search"

	// RoleWarm nodes provide access to warm indices and searchable snapshots.
	// Added in OpenSearch 2.4. In OpenSearch 3.0+, warm role replaces search role
	// for searchable snapshot functionality.
	// See: https://docs.opensearch.org/latest/tuning-your-cluster/index/
	RoleWarm = "warm"

	// RoleML nodes are dedicated to running machine learning tasks and models.
	// This is a dynamic role added by the ML Commons plugin, not a built-in server role.
	// Available when ML Commons plugin is installed (typically OpenSearch 1.3+).
	// See: https://docs.opensearch.org/latest/ml-commons-plugin/cluster-settings/
	RoleML = "ml"

	// RoleCoordinatingOnly represents nodes with no explicit roles (node.roles: []).
	// These nodes delegate client requests to shards on data nodes and aggregate results.
	// This is not a built-in role but a derived state when no roles are specified.
	// Available since OpenSearch 1.0 as a configuration pattern.
	// See: https://docs.opensearch.org/latest/install-and-configure/configuring-opensearch/configuration-system/
	RoleCoordinatingOnly = "coordinating_only"

	// RoleMaster is the deprecated role name for cluster manager nodes.
	//
	// Deprecated: Use RoleClusterManager instead.
	// See: https://docs.opensearch.org/latest/install-and-configure/configuring-opensearch/configuration-system/
	RoleMaster = "master"
)

// roleSet represents a set of node roles for efficient O(1) role lookups.
type roleSet map[string]struct{}

// workRoles defines node roles that perform actual work (data processing, storage, or search).
// Used to distinguish dedicated cluster managers from nodes that handle client requests.
// This matches the Java client's NodeSelector.SKIP_DEDICATED_CLUSTER_MASTERS logic.
//
//nolint:gochecknoglobals // This global constant defines the standard work roles
var workRoles = []string{
	RoleData,   // stores and retrieves data
	RoleIngest, // processes incoming data
	RoleWarm,   // handles warm/cold data storage
	RoleSearch, // dedicated search processing
	RoleML,     // machine learning tasks
}

// newRoleSet creates a roleSet from a slice of role names.
func newRoleSet(roles []string) roleSet {
	rs := make(roleSet, len(roles))
	for _, role := range roles {
		rs[role] = struct{}{}
		if role == RoleMaster {
			// Alias deprecated "master" role to "cluster_manager" for internal checks,
			// so only a single check is needed for "cluster_manager" elsewhere in the library.
			rs[RoleClusterManager] = struct{}{}
		}
	}
	return rs
}

// has reports whether the roleSet contains the named role.
func (rs roleSet) has(roleName string) bool {
	_, exists := rs[roleName]
	return exists
}

// toSlice returns the roles as a sorted string slice.
func (rs roleSet) toSlice() []string {
	if len(rs) == 0 {
		return nil
	}

	roles := make([]string, 0, len(rs))
	for role := range rs {
		// Skip the internal cluster_manager alias added for deprecated master role
		if role == RoleClusterManager {
			// Check if this was added as an alias for deprecated master role
			if _, hasMaster := rs[RoleMaster]; hasMaster {
				continue // Skip the alias, keep only the original master role
			}
		}
		roles = append(roles, role)
	}

	// Sort roles alphabetically for consistent ordering
	slices.Sort(roles)
	return roles
}

// isDedicatedClusterManager implements the logic from upstream Java client
// NodeSelector.SKIP_DEDICATED_CLUSTER_MASTERS to determine if a node should be skipped.
// It returns true for nodes that are cluster-manager eligible but have no "work" roles
// (i.e., roles that actually process/store data or handle requests).
// This matches OpenSearch server's SniffConnectionStrategy.DEFAULT_NODE_PREDICATE behavior.
func (rs roleSet) isDedicatedClusterManager() bool {
	// Must be cluster manager eligible first
	if !rs.has(RoleClusterManager) {
		return false
	}

	return !slices.ContainsFunc(workRoles, rs.has)
}

// Discoverable defines the interface for transports supporting node discovery.
type Discoverable interface {
	DiscoverNodes(ctx context.Context) error
}

// nodeInfoHTTP represents the HTTP configuration from node info.
type nodeInfoHTTP struct {
	PublishAddress string `json:"publish_address"` // Available since OpenSearch 1.0.0
}

// nodeInfoOS represents the OS configuration from node info.
type nodeInfoOS struct {
	AllocatedProcessors *int `json:"allocated_processors"` // Available since OpenSearch 1.0.0
}

// nodeInfoThreadPool represents the configuration for a single thread pool
// from the /_nodes/_local/http,os,thread_pool response.
type nodeInfoThreadPool struct {
	Type      string `json:"type"`                 // "fixed", "scaling", "resizable"
	Size      int    `json:"size,omitempty"`       // configured size (fixed/resizable)
	QueueSize int    `json:"queue_size"`           // queue capacity (-1 = unbounded)
	Core      int    `json:"core,omitempty"`       // min threads (scaling pools)
	Max       int    `json:"max,omitempty"`        // max threads (scaling/resizable)
	KeepAlive string `json:"keep_alive,omitempty"` // idle timeout
}

// nodeInfo represents a node discovered via the /_nodes API.
type nodeInfo struct {
	ID         string         `json:"id"`   // Available since OpenSearch 1.0.0
	Name       string         `json:"name"` // Available since OpenSearch 1.0.0
	url        *url.URL       // Client-side field, not from server
	Roles      []string       `json:"roles"`        // Available since OpenSearch 1.0.0
	Attributes map[string]any `json:"attributes"`   // Available since OpenSearch 1.0.0
	HTTP       nodeInfoHTTP   `json:"http"`         // Available since OpenSearch 1.0.0
	OS         *nodeInfoOS    `json:"os,omitempty"` // Present in /_nodes/_local/http,os responses; nil from /_nodes/http

	// ThreadPool maps pool name -> configuration. Present in
	// /_nodes/_local/http,os,thread_pool responses; nil from /_nodes/http.
	ThreadPool map[string]nodeInfoThreadPool `json:"thread_pool,omitempty"`

	// Internal fields (not part of JSON)
	roleSet roleSet
}

// _NodesMeta is the "_nodes" metadata envelope returned by all
// /_nodes/* endpoints. It reports how many nodes were contacted
// (Total), how many responded successfully (Successful), and how
// many timed out or errored (Failed).
//
// Example:
//
//	"_nodes": { "total": 3, "successful": 1, "failed": 2 }
type _NodesMeta struct {
	Total      int               `json:"total"`
	Successful int               `json:"successful"`
	Failed     int               `json:"failed"`
	Failures   []json.RawMessage `json:"failures,omitempty"`
}

// formatFailures returns a compact JSON representation of the _nodes failures
// array for inclusion in error messages. Returns "[]" when no failures are present.
func (m *_NodesMeta) formatFailures() string {
	if len(m.Failures) == 0 {
		return "[]"
	}
	b, err := json.Marshal(m.Failures)
	if err != nil {
		return fmt.Sprintf("[<%d failures, marshal error: %v>]", len(m.Failures), err)
	}
	return string(b)
}

// DiscoverNodes reloads the client connections by fetching information from
// the cluster. If another discovery is already in progress, DiscoverNodes
// blocks until that discovery completes (or ctx is cancelled) and returns
// its result.
func (c *Client) DiscoverNodes(ctx context.Context) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}

	c.discoverMu.Lock()

	if c.discoverMu.inProgress {
		// Another goroutine is running discovery. Wait for it using
		// sync.Cond + context.AfterFunc so that context cancellation
		// wakes us even though Cond.Wait is not context-aware.
		stopf := context.AfterFunc(ctx, func() {
			c.discoverMu.Lock()
			defer c.discoverMu.Unlock()
			c.discoverMu.cond.Broadcast()
		})
		defer stopf()

		for c.discoverMu.inProgress {
			c.discoverMu.cond.Wait()
			if ctx.Err() != nil {
				c.discoverMu.Unlock()
				return ctx.Err()
			}
		}
		err := c.discoverMu.lastErr
		c.discoverMu.Unlock()
		// If our own context was cancelled (it can race the final
		// Broadcast and skip the in-loop check), that takes precedence.
		if ctx.Err() != nil {
			return ctx.Err()
		}
		// Otherwise the discovery we waited on may have been aborted by the
		// runner's context, not ours. Don't inherit a cancellation we never
		// initiated -- report a retryable sentinel instead.
		if err != nil && (errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded)) {
			return errDiscoveryInterrupted
		}
		return err
	}

	// We won the race: start discovery.
	// Lock is held — doDiscoverNodes takes ownership and releases it.
	return c.doDiscoverNodes(ctx)
}

// tryDiscoverNodes attempts to start a discovery cycle. If discovery is
// already in progress it returns nil immediately without waiting.
//
// This is used by the internal discoveryLoop, which must never block on
// another discovery. It could be exported in the future if callers need
// fire-and-forget semantics on the public API.
func (c *Client) tryDiscoverNodes(ctx context.Context) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}

	c.discoverMu.Lock()
	if c.discoverMu.inProgress {
		c.discoverMu.Unlock()
		return nil
	}
	// Lock is held — doDiscoverNodes takes ownership and releases it.
	return c.doDiscoverNodes(ctx)
}

// doDiscoverNodes performs the discovery work.
//
// Called with c.discoverMu held. It sets inProgress = true, releases the
// lock for I/O, then re-acquires it on completion to clear inProgress,
// store the result, and wake any waiters.
func (c *Client) doDiscoverNodes(ctx context.Context) error {
	c.discoverMu.inProgress = true
	c.discoverMu.Unlock()

	var discoverErr error
	defer func() {
		c.discoverMu.Lock()
		c.discoverMu.inProgress = false
		c.discoverMu.lastErr = discoverErr
		c.discoverMu.cond.Broadcast()
		c.discoverMu.Unlock()
	}()

	discovered, err := c.getNodesInfo(ctx)
	if err != nil {
		if dl := loadDebugLogger(); dl != nil {
			dl.Logf("Error getting nodes info: %s\n", err)
		}
		discoverErr = fmt.Errorf("discovery: get nodes: %w", err)
		return discoverErr
	}

	c.mu.RLock()
	connPool := c.mu.connectionPool
	c.mu.RUnlock()
	isColdStart := connPool == nil

	if isColdStart {
		if err := c.nodeDiscoveryAsyncStart(ctx, discovered); err != nil {
			discoverErr = err
			return discoverErr
		}
	} else {
		if err := c.nodeDiscovery(ctx, discovered); err != nil {
			discoverErr = err
			return discoverErr
		}
	}

	// Fetch shard placement data for scored routing. This is non-blocking:
	// if /_cat/shards fails (e.g., missing permissions), scored routing
	// continues with minFanOut and no shard-aware partitioning.
	if c.router != nil {
		c.fetchAndUpdateShardPlacement(ctx)
	}

	// Sync dead lists in policy pools. The allConns pool's scheduleResurrect
	// handles health checking and sets lcActive, but policy pools have their
	// own dead lists and no independent resurrection timers. CheckDead gives
	// each policy pool a chance to notice that shared *Connections have been
	// resurrected (fast path: lifecycle-bit check, no I/O) or to perform its
	// own health checks (slow path: parallel HTTP probes).
	if c.router != nil {
		if err := c.router.CheckDead(ctx, c.DefaultHealthCheck); err != nil {
			if dl := loadDebugLogger(); dl != nil {
				dl.Logf("DiscoverNodes: router.CheckDead: %v\n", err)
			}
		}
	}

	// Expire the zombie-resurrection grace on connections that have been dead
	// too long, so a node that never comes back stops being blindly retried.
	c.resetDeadConnViability()

	// Rotate standby connections after discovery completes.
	// This piggybacks on the discovery interval rather than using a separate timer.
	// Each rotation health-checks one standby and, if healthy, swaps it with a random active.
	// Routes through the router when available so policy pools (which actually have
	// standby partitions) are rotated -- the allConns pool auto-scales its cap to
	// pool size, so it never has standby connections.
	if c.activeListCap > 0 && c.standbyRotationInterval >= 0 {
		c.rotateStandbyConnections(ctx)
	}

	return nil
}

// rotateStandbyConnections performs one standby rotation cycle, health-checking
// a standby connection and swapping it with a random active connection.
func (c *Client) rotateStandbyConnections(ctx context.Context) {
	if c.router != nil {
		if n, err := c.router.RotateStandby(ctx, c.standbyRotationCount); err != nil {
			if dl := loadDebugLogger(); dl != nil {
				dl.Logf("DiscoverNodes: router.RotateStandby rotated %d/%d: %v\n", n, c.standbyRotationCount, err)
			}
		}
		return
	}

	c.mu.RLock()
	pool, ok := c.mu.connectionPool.(*multiServerPool)
	c.mu.RUnlock()

	if ok && pool != nil {
		if n, err := pool.rotateStandby(ctx, c.standbyRotationCount); err != nil {
			if dl := loadDebugLogger(); dl != nil {
				dl.Logf("DiscoverNodes: pool.rotateStandby rotated %d/%d: %v\n", n, c.standbyRotationCount, err)
			}
		}
	}
}

// nodeDiscoveryAsyncStart handles discovery with asynchronous connection startup - prioritizes fast startup.
func (c *Client) nodeDiscoveryAsyncStart(ctx context.Context, discovered []nodeInfo) error {
	// Async start - assume all connections are ready for fast startup
	readyConnections := make([]*Connection, 0, len(discovered))

	for _, node := range discovered {
		conn := c.createConnection(node)
		readyConnections = append(readyConnections, conn)

		// Async health check - will be handled by normal pool mechanics
		go func(conn *Connection) {
			c.healthCheckWithRetries(ctx, conn, c.discoveryHealthCheckRetries)
		}(conn)
	}

	// Cold start -- no existing connections to compare timestamps against.
	return c.updateConnectionPool(ctx, time.Time{}, readyConnections, []*Connection{})
}

// nodeDiscovery handles discovery for running clusters -- reuses existing connections.
//
// This function resolves each discovered node against the current connection pool,
// establishing a POINTER-IDENTITY CONTRACT that updateConnectionPool relies on:
//
//   - Same pointer (reused): the node's identity (ID, Name, Roles) is unchanged.
//     The existing *Connection is passed through, preserving all accumulated state
//     (RTT ring, allocatedProcessors, warmup progress, load estimates, etc.).
//
//   - New pointer (replaced): the node is new or its identity changed (different ID,
//     Name, or Roles). A fresh *Connection is created via createConnection() and placed
//     on the dead list. The pool's scheduleResurrect -> performHealthCheck ->
//     resurrectWithLock flow handles health checking and promotion.
//
// updateConnectionPool uses this contract: oldConn == newConn means unchanged,
// oldConn != newConn means remove+add (regardless of role comparison).
func (c *Client) nodeDiscovery(ctx context.Context, discovered []nodeInfo) error {
	// Build lookup of existing connections by URL to resolve pointers.
	c.mu.RLock()
	currentPool := c.mu.connectionPool
	c.mu.RUnlock()

	currentByURL := make(map[string]*Connection)
	if currentPool != nil {
		for _, u := range currentPool.URLs() {
			urlStr := u.String()
			if conn := c.findConnectionByURL(currentPool, urlStr); conn != nil {
				currentByURL[urlStr] = conn
			}
		}
	}

	// Resolve connections: reuse existing pointer or create new (dead).
	readyConnections := make([]*Connection, 0, len(discovered))
	deadConnections := make([]*Connection, 0, len(discovered))

	for _, node := range discovered {
		node.roleSet = newRoleSet(node.Roles)

		existing, found := currentByURL[node.url.String()]
		if found && canReuseConnection(existing, node) {
			readyConnections = append(readyConnections, existing)
			continue
		}

		// New node or identity changed -- create as dead, pool resurrection handles the rest
		conn := c.createConnection(node)
		deadConnections = append(deadConnections, conn)
	}

	return c.updateConnectionPool(ctx, time.Time{}, readyConnections, deadConnections)
}

// canReuseConnection returns true if the existing connection matches the
// discovered node's identity (ID, Name, Roles). If any of these differ, the
// node was replaced or reconfigured and a new Connection should be created.
// URL is already matched by the caller's map lookup.
func canReuseConnection(conn *Connection, node nodeInfo) bool {
	if conn.ID != node.ID {
		return false
	}
	if conn.Name != node.Name {
		return false
	}
	return slices.Compare(conn.Roles.toSlice(), node.roleSet.toSlice()) == 0
}

// createConnection creates a Connection from nodeInfo with proper role processing.
// New connections start in lcDead state so that the allConns pool's partition logic
// can distinguish them from reused connections (which retain their policy-pool
// lifecycle -- lcActive, lcStandby, etc.). The caller is responsible for
// transitioning new connections to the appropriate lifecycle after health checking.
func (c *Client) createConnection(node nodeInfo) *Connection {
	// Build role set for efficient O(1) lookups
	node.roleSet = newRoleSet(node.Roles)

	initialState := lcDead | lcNeedsWarmup | lcNeedsHardware

	conn := &Connection{
		URL:        node.url,
		URLString:  node.url.String(),
		ID:         node.ID,
		Name:       node.Name,
		Roles:      node.roleSet,
		Attributes: node.Attributes,
	}
	conn.estLoad.clock = realClock{}
	conn.weight.Store(1)
	conn.rttRing = newRTTRing(c.rttRingSize)

	// Store allocated_processors if present (populated when nodeInfo was parsed
	// from a /_nodes/_local/http,os,thread_pool response; nil from /_nodes/http discovery).
	if node.OS != nil && node.OS.AllocatedProcessors != nil {
		conn.storeAllocatedProcessors(*node.OS.AllocatedProcessors)
		initialState &^= lcNeedsHardware // hardware info obtained
	}

	// Store thread pool sizes if present (same response as OS data).
	storeThreadPoolSizes(conn, node.ThreadPool)

	conn.state.Store(int64(newConnState(initialState)))
	return conn
}

// storeThreadPoolSizes iterates a node's thread pool configuration and
// stores each pool's effective size as the cwnd ceiling on the connection.
// Scaling pools report max instead of size; whichever is positive is used.
// Called from createConnection and hardwareInfoHealthCheck.
func storeThreadPoolSizes(conn *Connection, pools map[string]nodeInfoThreadPool) {
	for name, tp := range pools {
		size := tp.Size
		if size <= 0 && tp.Max > 0 {
			size = tp.Max // scaling pools report max, not size
		}
		if size > 0 {
			conn.storeMaxCwnd(name, size)
		}
	}
}

// updateConnectionPool atomically updates the connection pool with new connection information
// and notifies the router of cluster topology changes.
//
// healthCheckedAt is the timestamp before health checks began. When reusing an existing
// Connection object whose deadSince predates healthCheckedAt, the more recent health check
// result wins and the connection is resurrected. If deadSince is newer (set concurrently
// during the health check window), the dead state is preserved. Zero means no timestamp
// comparison (cold start -- no existing connections to compare against).
func (c *Client) updateConnectionPool(
	ctx context.Context, healthCheckedAt time.Time, readyConnections, deadConnections []*Connection,
) error {
	totalNodes := len(readyConnections) + len(deadConnections)
	allConnections := make([]*Connection, 0, totalNodes)
	allConnections = append(allConnections, readyConnections...)
	allConnections = append(allConnections, deadConnections...)

	// Get current connections with their role information for diff calculation
	c.mu.RLock()
	currentPool := c.mu.connectionPool
	c.mu.RUnlock()

	var currentConnectionsByURL map[string]*Connection

	if currentPool != nil {
		currentURLs := currentPool.URLs()

		// Pre-size map based on current pool size
		currentConnectionsByURL = make(map[string]*Connection, len(currentURLs))

		// Get actual connections from the pool to preserve old role information
		// Old connections are needed to detect role changes
		for _, urlPtr := range currentURLs {
			url := urlPtr.String()
			// Try to find this URL in the current pool
			if conn := c.findConnectionByURL(currentPool, url); conn != nil {
				currentConnectionsByURL[url] = conn
			}
		}
	} else {
		currentConnectionsByURL = make(map[string]*Connection)
	}

	// Build map of new connections by URL
	newConnectionsByURL := make(map[string]*Connection, totalNodes)
	for _, conn := range allConnections {
		newConnectionsByURL[conn.URL.String()] = conn
	}

	// Calculate diffs: added, removed, unchanged
	//
	// POINTER-IDENTITY CONTRACT (established by nodeDiscovery):
	//   oldConn == newConn  -> unchanged (same node, same identity)
	//   oldConn != newConn  -> remove+add (node replaced or reconfigured)
	//   newConn not in old  -> add (brand new node)
	//   oldConn not in new  -> remove (node departed)
	//
	// nodeDiscovery resolves discovered nodes against the current pool's
	// connections. When a node's identity (ID, Name, Roles) matches, it passes
	// the existing *Connection pointer. When identity differs, it creates a new
	// *Connection via createConnection(). This means pointer equality is a
	// reliable proxy for "nothing changed" -- no field-by-field comparison needed.
	added := make([]*Connection, 0, len(newConnectionsByURL))
	removed := make([]*Connection, 0, len(currentConnectionsByURL))
	unchanged := make([]*Connection, 0, len(currentConnectionsByURL))

	// Build final connection list (mix of reused old and new objects)
	finalConnectionsByURL := make(map[string]*Connection, len(newConnectionsByURL))

	for url, newConn := range newConnectionsByURL {
		oldConn, existed := currentConnectionsByURL[url]
		if !existed {
			// Brand new URL -- node joined the cluster
			added = append(added, newConn)
			finalConnectionsByURL[url] = newConn
			continue
		}

		if oldConn == newConn {
			// Same pointer -- nodeDiscovery confirmed identity match
			unchanged = append(unchanged, newConn)
			finalConnectionsByURL[url] = newConn
			continue
		}

		// Different pointer for same URL -- nodeDiscovery determined the node
		// was replaced (different ID, Name, or Roles) and created a new
		// *Connection. Treat as remove+add so policies re-evaluate membership.
		removed = append(removed, oldConn)
		added = append(added, newConn)
		finalConnectionsByURL[url] = newConn
	}

	// Find removed connections (existed before but not in new discovery)
	for url, oldConn := range currentConnectionsByURL {
		if _, exists := newConnectionsByURL[url]; !exists {
			removed = append(removed, oldConn)
			if dl := loadDebugLogger(); dl != nil {
				dl.Logf("Discovery: Connection %q removed from cluster (roles: %v)\n", url, oldConn.Roles.toSlice())
			}
		}
	}

	// Build final ready/dead lists from the connection objects chosen for use
	// (mix of reused old objects and new objects).
	//
	// When reusing an old Connection object that has deadSince set, compare it
	// against healthCheckedAt to determine which information is newer:
	//   - deadSince < healthCheckedAt -> dead state is stale, health check is newer -> resurrect
	//   - deadSince >= healthCheckedAt -> dead state set concurrently/after health check -> keep dead
	// When healthCheckedAt is zero (cold start), there are no old connections to reuse,
	// so the comparison is moot.
	readyURLs := make(map[string]struct{}, len(readyConnections))
	for _, conn := range readyConnections {
		readyURLs[conn.URL.String()] = struct{}{}
	}

	finalReady := make([]*Connection, 0, len(finalConnectionsByURL))
	finalDead := make([]*Connection, 0, len(finalConnectionsByURL))

	for url, conn := range finalConnectionsByURL {
		if _, isReady := readyURLs[url]; isReady {
			conn.mu.Lock()
			deadSince := conn.loadDeadSince()
			stale := !deadSince.IsZero() && !healthCheckedAt.IsZero() && deadSince.Before(healthCheckedAt)
			switch {
			case stale:
				// Dead state predates the health check -- resurrect.
				conn.storeDeadSince(time.Time{})
				conn.mu.Unlock()
				conn.failures.Store(0)
			case !deadSince.IsZero():
				// Dead state is concurrent or newer -- keep dead.
				conn.mu.Unlock()
				finalDead = append(finalDead, conn)
				continue
			default:
				conn.mu.Unlock()
			}
			finalReady = append(finalReady, conn)
		} else {
			finalDead = append(finalDead, conn)
		}
	}

	// Compute GCD-normalized weights from allocatedProcessors before connections
	// are added to pools. This must happen before the pool swap so that the
	// duplicate-pointer insertion (appendToReadyActiveWithLock) uses the correct
	// weight for each connection.
	allFinal := make([]*Connection, 0, len(finalReady)+len(finalDead))
	allFinal = append(allFinal, finalReady...)
	allFinal = append(allFinal, finalDead...)
	computeWeights(allFinal)

	// Dynamically recalculate capacity model from discovered hardware.
	// Use the minimum allocatedProcessors across all nodes with known values --
	// the smallest node is the bottleneck for per-server rate limits.
	c.recalculateCapacityModel(allFinal)

	// Atomically swap the connection pool under c.mu, then release the lock
	// before notifying the router and observers. This avoids holding c.mu (W)
	// while router policies acquire pool-level locks (the lock ordering that
	// caused the deadlock: c.mu(W) -> pool.mu(W) vs pool.mu(R) in the request path).
	c.mu.Lock()

	totalFinalNodes := len(finalReady) + len(finalDead)
	var newConnectionPool ConnectionPool
	if totalFinalNodes == 1 {
		newConnectionPool = c.createOrUpdateSingleNodePool(finalReady, finalDead)
	} else {
		newConnectionPool = c.createOrUpdateMultiNodePoolWithLock(finalReady, finalDead)
	}

	c.mu.connectionPool = newConnectionPool

	// Set up health check function and observer for pools that support it
	if pool, ok := c.mu.connectionPool.(*multiServerPool); ok {
		pool.mu.Lock()
		pool.mu.healthCheck = c.DefaultHealthCheck
		pool.mu.Unlock()
		if obs := c.observer.Load(); obs != nil {
			pool.observer.Store(obs)
		}
	}

	c.mu.Unlock()

	// Schedule resurrection for dead connections so they get health-checked
	// (which clears lcNeedsHardware and populates allocatedProcessors) and
	// promoted to active through the standard resurrection path.
	if pool, ok := newConnectionPool.(*multiServerPool); ok && len(finalDead) > 0 {
		for _, conn := range finalDead {
			pool.scheduleResurrect(ctx, conn)
		}
	}

	// Schedule RTT probes for reused connections that have no RTT measurements.
	// This handles connections reused by nodeDiscovery() that were never health-checked,
	// which would otherwise leave rttRing at rttBucketUnknown and break connection scoring.
	if pool, ok := newConnectionPool.(*multiServerPool); ok {
		pool.mu.RLock()
		hc := pool.mu.healthCheck
		pool.mu.RUnlock()
		for _, conn := range finalReady {
			if conn.rttRing != nil && conn.rttRing.medianBucket().IsUnknown() && hc != nil {
				go pool.scheduleRTTProbe(conn) //nolint:contextcheck // scheduleRTTProbe uses pool's long-lived context.
			}
		}
	}

	// Notify router outside c.mu -- router.DiscoveryUpdate may acquire pool-level
	// locks via RolePolicy.discoveryUpdateAdd/enforceActiveCapWithLock. Holding c.mu
	// here would create the lock inversion: c.mu(W) -> pool.mu(W).
	// c.router is immutable after construction, so no lock is needed to read it.
	if c.router != nil {
		// Pass calculated diffs to router so policies don't have to recalculate
		if err := c.router.DiscoveryUpdate(added, removed, unchanged); err != nil {
			// Continue - don't fail discovery due to router errors
			_ = err
		}
	}

	// Notify observer of discovery changes (observer is atomic, safe without c.mu)
	if obs := observerFromAtomic(&c.observer); obs != nil {
		// Compute lifecycle-bit counts from local slices (no pool lock held).
		var discoveryCounts lifecycleCounts
		for _, conn := range finalReady {
			lc := conn.loadConnState().lifecycle()
			switch {
			case lc.has(lcActive):
				discoveryCounts.active++
			case lc.has(lcStandby):
				discoveryCounts.standby++
			default:
				discoveryCounts.dead++
			}
		}
		discoveryCounts.dead += len(finalDead)

		for _, conn := range added {
			obs.OnDiscoveryAdd(newConnectionEvent("allConns", conn, discoveryCounts))
		}
		for _, conn := range removed {
			obs.OnDiscoveryRemove(newConnectionEvent("allConns", conn, discoveryCounts))
		}
		for _, conn := range unchanged {
			obs.OnDiscoveryUnchanged(newConnectionEvent("allConns", conn, discoveryCounts))
		}
	}

	return nil
}

// gcd returns the greatest common divisor of two positive integers using the
// Euclidean algorithm. Both arguments must be positive.
func gcd(a, b int) int {
	for b != 0 {
		a, b = b, a%b
	}
	return a
}

// computeWeights sets each connection's weight based on its allocatedProcessors
// value, normalized by the GCD of all known core counts. Connections with unknown
// core counts (allocatedProcessors == 0) get weight 1.
//
// Examples:
//
//	Cores [8, 16]          -> GCD=8  -> weights [1, 2]
//	Cores [8, 16, 24]      -> GCD=8  -> weights [1, 2, 3]
//	Cores [8, 16, 32, 40]  -> GCD=8  -> weights [1, 2, 4, 5]
//	Cores [24, 32, 40]     -> GCD=8  -> weights [3, 4, 5]
//	Cores [8, 8, 8]        -> GCD=8  -> weights [1, 1, 1]  (homogeneous)
//
// If no connections have known core counts, all weights remain at their current
// value (typically 1 from initialization).
func computeWeights(conns []*Connection) {
	// Collect all known core counts.
	d := 0
	for _, c := range conns {
		if ap := c.loadAllocatedProcessors(); ap > 0 {
			if d == 0 {
				d = ap
			} else {
				d = gcd(d, ap)
			}
		}
	}

	if d == 0 {
		// No connections have known core counts -- leave weights unchanged.
		return
	}

	for _, c := range conns {
		if ap := c.loadAllocatedProcessors(); ap > 0 {
			c.weight.Store(int32(min(ap/d, math.MaxInt32))) //nolint:gosec // core count ratio always fits
		} else {
			c.weight.Store(1)
		}
	}
}

// recalculateCapacityModel updates the client's internal capacity model fields
// based on discovered hardware info. Uses the minimum allocatedProcessors across
// all nodes (the bottleneck) to derive serverMaxNewConnsPerSec, clientsPerServer,
// and healthCheckRate. Skips recalculation if no nodes have known core counts.
func (c *Client) recalculateCapacityModel(conns []*Connection) {
	minCores := 0
	for _, conn := range conns {
		if ap := conn.loadAllocatedProcessors(); ap > 0 {
			if minCores == 0 || ap < minCores {
				minCores = ap
			}
		}
	}

	if minCores == 0 {
		// No nodes have known core counts -- keep current defaults.
		return
	}

	c.mu.Lock()
	c.serverMaxNewConnsPerSec = float64(minCores) * serverMaxNewConnsPerSecMultiplier
	c.clientsPerServer = float64(minCores)
	c.healthCheckRate = float64(minCores) * healthCheckRateMultiplier
	c.mu.Unlock()
}

// findConnectionByURL attempts to find a connection in the pool by URL.
// This helper extracts connections from different pool types to get their role information.
func (c *Client) findConnectionByURL(pool ConnectionPool, url string) *Connection {
	switch p := pool.(type) {
	case *singleServerPool:
		if p.connection != nil && p.connection.URL.String() == url {
			return p.connection
		}
	case *multiServerPool:
		// Check both ready and dead lists
		p.RLock()
		defer p.RUnlock()

		for _, conn := range p.mu.ready {
			if conn.URL.String() == url {
				return conn
			}
		}
		for _, conn := range p.mu.dead {
			if conn.URL.String() == url {
				return conn
			}
		}
	}

	return nil
}

// createOrUpdateSingleNodePool handles single-node connection pool creation/updates.
// Caller must hold c.mu.Lock().
func (c *Client) createOrUpdateSingleNodePool(readyConnections, deadConnections []*Connection) ConnectionPool {
	// Single node - check if demotion from multiServerPool is needed
	if _, isStatusPool := c.mu.connectionPool.(*multiServerPool); isStatusPool {
		// Demote from multi-node to single-node pool
		return c.demoteConnectionPoolWithLock()
	}

	// Create or update single connection pool
	var connection *Connection
	if len(readyConnections) == 1 {
		connection = readyConnections[0]
	} else if len(deadConnections) == 1 {
		connection = deadConnections[0]
	}

	// Preserve metrics from existing single connection pool
	var metrics *metrics
	if existingSinglePool, ok := c.mu.connectionPool.(*singleServerPool); ok {
		metrics = existingSinglePool.metrics
	}

	return newSingleServerPool(connection, metrics)
}

// createOrUpdateMultiNodePoolWithLock handles multi-node connection pool creation/updates.
// Caller must hold c.mu.Lock().
func (c *Client) createOrUpdateMultiNodePoolWithLock(readyConnections, deadConnections []*Connection) ConnectionPool {
	// Multi-node - check if promotion from singleServerPool is needed
	if _, isSinglePool := c.mu.connectionPool.(*singleServerPool); isSinglePool {
		// Promote from single-node to multi-node pool
		return c.promoteConnectionPoolWithLock(readyConnections, deadConnections)
	}

	// allConns is the bookkeeping inventory of every discovered connection,
	// including dedicated cluster managers. Discovery reuses connections by
	// scanning allConns (findConnectionByURL) and evicts them via the removed
	// diff computed against allConns. Query traffic is kept off dedicated
	// cluster managers at the routing-policy layer (see RoundRobinPolicy).
	allReadyConns := make([]*Connection, 0, len(readyConnections))
	allDeadConns := make([]*Connection, 0, len(deadConnections))

	allReadyConns = append(allReadyConns, readyConnections...)
	allDeadConns = append(allDeadConns, deadConnections...)

	// Shuffle connections for load distribution unless disabled
	if !c.skipConnectionShuffle && len(allReadyConns) > 1 {
		rand.Shuffle(len(allReadyConns), func(i, j int) {
			allReadyConns[i], allReadyConns[j] = allReadyConns[j], allReadyConns[i]
		})
	}

	// Reuse the existing multiServerPool when possible. Updating in-place
	// preserves the pool pointer that resurrection goroutines captured via
	// scheduleResurrect's `cp` parameter. If we created a new pool, those
	// goroutines would find their connection absent from the old pool's
	// dead list (stillInPool check) and exit, orphaning the connection
	// with no health check goroutine.
	var allConnsPool *multiServerPool
	if existingMultiPool, ok := c.mu.connectionPool.(*multiServerPool); ok {
		allConnsPool = existingMultiPool

		allConnsPool.mu.Lock()
		allConnsPool.mu.ready = allReadyConns
		allConnsPool.mu.dead = nil
		allConnsPool.mu.members = make(map[*Connection]struct{}, max(len(allReadyConns)+len(allDeadConns), defaultMembersCapacity))
		for _, conn := range allReadyConns {
			allConnsPool.mu.members[conn] = struct{}{}
		}
		for _, conn := range allDeadConns {
			allConnsPool.mu.members[conn] = struct{}{}
			allConnsPool.appendToDeadWithLock(conn)
		}
		allConnsPool.mu.Unlock()
	} else {
		allConnsPool = c.newMultiServerPoolFromClientWithLock("allConns", nil)
		allConnsPool.mu.ready = allReadyConns
		allConnsPool.mu.members = make(map[*Connection]struct{}, max(len(allReadyConns)+len(allDeadConns), defaultMembersCapacity))
		for _, conn := range allReadyConns {
			allConnsPool.mu.members[conn] = struct{}{}
		}
		for _, conn := range allDeadConns {
			allConnsPool.mu.members[conn] = struct{}{}
			allConnsPool.appendToDeadWithLock(conn)
		}
	}

	// Recalculate warmup parameters and partition the ready list under the pool
	// write lock. recalculateWarmupParamsWithLock writes mu.activeListCap/warmupRounds/
	// warmupSkipCount, getWarmupParamsWithLock reads them, and the final assignment sets
	// mu.activeCount -- all mu-guarded fields that resurrectWithLock also touches
	// under pool.mu. Holding the lock across the whole section serializes it
	// against resurrection (c.mu, held by the caller, only serializes it against
	// metrics.snapshot). Per-connection conn.mu is taken inside the loop, matching
	// the pool.mu -> conn.mu ordering used by deferredStandbyPromotion.
	allConnsPool.mu.Lock()

	// Recalculate activeListCap and warmup parameters for the allConns pool before
	// partitioning so startWarmup calls use the correctly-scaled values.
	allConnsPool.recalculateWarmupParamsWithLock(len(allReadyConns) + len(allDeadConns))

	// Partition ready connections by their current lifecycle state.
	// Reused connections (unchanged in discovery) may already be in standby
	// in policy pools via the shared Connection.state atomic. Forcing them all
	// to lcActive would clobber the state that policy pools depend on, causing
	// Next() to evict "externally-demoted" connections from the wrong partition.
	//
	// Instead, read each connection's current state and partition accordingly:
	//   - lcActive (or warming) -> active partition (no state mutation)
	//   - lcStandby (with or without lcNeedsWarmup) -> standby partition (no state mutation)
	//   - lcDead -> newly discovered connection (createConnection initializes to
	//     lcDead). Transition to lcActive with warmup so the connection ramps up
	//     traffic gradually.
	activeCount := 0
	for i, conn := range allReadyConns {
		lc := conn.loadConnState().lifecycle()
		switch {
		case lc.has(lcActive):
			// Already active (possibly warming) -- keep in active partition.
			if i != activeCount {
				allReadyConns[i], allReadyConns[activeCount] = allReadyConns[activeCount], allReadyConns[i]
			}
			activeCount++
		case lc.has(lcStandby):
			// Standby in a policy pool -- leave past activeCount boundary.
		default:
			// New connection (lcDead from createConnection) or unexpected state.
			// Transition to active with warmup for gradual traffic ramp-up.
			conn.mu.Lock()
			conn.casLifecycle(conn.loadConnState(), 0, lcActive, lcUnknown|lcStandby) //nolint:errcheck // lock held; only errLifecycleNoop possible
			conn.mu.Unlock()
			rounds, skip := allConnsPool.getWarmupParamsWithLock()
			conn.startWarmup(rounds, skip)
			if i != activeCount {
				allReadyConns[i], allReadyConns[activeCount] = allReadyConns[activeCount], allReadyConns[i]
			}
			activeCount++
		}
	}
	allConnsPool.mu.activeCount = activeCount
	allConnsPool.mu.Unlock()

	// NOTE: enforceActiveCapWithLock() is intentionally NOT called here.
	// The allConns pool is a transport-level container for discovery bookkeeping.
	// Cap enforcement is owned by the policy pools (RoundRobinPolicy, RolePolicy),
	// which manage their own active/standby partitions via DiscoveryUpdate.
	// Calling enforceActiveCapWithLock on the allConns pool would mutate the shared
	// Connection.state atomics (setting lcStandby + clearing warmup), clobbering
	// the warmup progress that policy pools are tracking. This caused connections
	// to never finish warmup when the discovery interval was shorter than the
	// warmup duration.

	return allConnsPool
}

func (c *Client) getNodesInfo(ctx context.Context) ([]nodeInfo, error) {
	scheme := c.urls[0].Scheme

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "/_nodes/http", nil)
	if err != nil {
		return nil, err
	}

	conn, err := getConnectionFromPool(c, req)
	if err != nil && errors.Is(err, ErrNoConnections) && c.seedFallbackPool != nil {
		conn, err = c.seedFallbackPool.Next()
	}
	if err != nil {
		return nil, err
	}

	c.setReqURL(conn.URL, req)
	c.setReqAuth(conn.URL, req)
	c.setReqUserAgent(req)

	res, err := c.transport.RoundTrip(req)
	if err != nil {
		// Report connection failure to the pool.
		c.mu.RLock()
		pool := c.mu.connectionPool
		c.mu.RUnlock()
		if pool != nil {
			if poolErr := pool.OnFailure(conn); poolErr != nil && loadDebugLogger() != nil {
				loadDebugLogger().Logf("Failed to mark connection as failed: %v\n", poolErr) //nolint:errcheck // debug log
			}
		}
		return nil, err
	}

	if res.Body == nil {
		return nil, fmt.Errorf("unexpected empty body")
	}
	// Drain on close: raw RoundTrip path, and the success path uses json.Decode
	// below, which stops at the end of the JSON value without consuming trailing
	// bytes; a plain Close would then defeat keep-alive.
	defer func() {
		//nolint:errcheck // best-effort drain before close on a cleanup path
		io.Copy(io.Discard, res.Body)
		res.Body.Close()
	}()

	if res.StatusCode > http.StatusOK {
		body, err := io.ReadAll(res.Body)
		if err != nil {
			return nil, fmt.Errorf("server error: %s: %w", res.Status, err)
		}
		return nil, fmt.Errorf("server error: %s: %s", res.Status, body)
	}

	var env map[string]json.RawMessage
	if err := json.NewDecoder(res.Body).Decode(&env); err != nil {
		return nil, err
	}

	// Parse the "_nodes" metadata header. Every /_nodes/* response includes
	// this envelope with total/successful/failed counts from the internal
	// transport fan-out. When nodes time out or error during the fan-out,
	// they appear in "failed" and their entries are absent from "nodes".
	//
	// The metadata is present in all OpenSearch versions (forked from ES 7.10)
	// and Elasticsearch >= 2.0. If absent (pre-2.0 ES or a stripping proxy),
	// skip the check and fall through to the nodes map.
	var nodesMeta _NodesMeta
	var hasNodesMeta bool
	if raw, ok := env["_nodes"]; ok {
		if err := json.Unmarshal(raw, &nodesMeta); err != nil {
			return nil, fmt.Errorf("discovery: parse _nodes metadata: %w", err)
		}
		hasNodesMeta = true
	}

	if hasNodesMeta && nodesMeta.Successful == 0 {
		failuresJSON := nodesMeta.formatFailures()
		if dl := loadDebugLogger(); dl != nil {
			dl.Logf("Discovery: _nodes reports total=%d successful=%d failed=%d failures=%s; retaining current pool\n",
				nodesMeta.Total, nodesMeta.Successful, nodesMeta.Failed, failuresJSON)
		}
		return nil, fmt.Errorf("discovery: total=%d successful=%d failed=%d failures=%s: %w",
			nodesMeta.Total, nodesMeta.Successful, nodesMeta.Failed, failuresJSON, errDiscoveryEmpty)
	}

	var nodes map[string]nodeInfo
	if err := json.Unmarshal(env["nodes"], &nodes); err != nil {
		return nil, err
	}

	// Fallback for servers that don't return _nodes metadata (ElasticSearch or OpenSearch < 2.0):
	// use the nodes map length as a last-resort empty-response guard.
	if !hasNodesMeta && len(nodes) == 0 {
		if dl := loadDebugLogger(); dl != nil {
			dl.Logf("Discovery: no _nodes metadata and zero nodes returned; retaining current pool\n")
		}
		return nil, fmt.Errorf("discovery: %w", errDiscoveryEmpty)
	}

	if hasNodesMeta && nodesMeta.Failed > 0 {
		failuresJSON := nodesMeta.formatFailures()
		if dl := loadDebugLogger(); dl != nil {
			dl.Logf("Discovery: partial failure — _nodes reports total=%d successful=%d failed=%d failures=%s\n",
				nodesMeta.Total, nodesMeta.Successful, nodesMeta.Failed, failuresJSON)
		}
	}

	out := make([]nodeInfo, len(nodes))
	idx := 0

	for id, node := range nodes {
		node.ID = id
		u := c.getNodeURL(node, scheme)
		node.url = u
		out[idx] = node
		idx++
	}

	// Report connection success to the pool.
	c.mu.RLock()
	pool := c.mu.connectionPool
	c.mu.RUnlock()
	if pool != nil {
		pool.OnSuccess(conn)
	}

	return out, nil
}

func (c *Client) getNodeURL(node nodeInfo, scheme string) *url.URL {
	var (
		host string
		port string
		err  error

		addrs = strings.Split(node.HTTP.PublishAddress, "/")
		ports = strings.Split(node.HTTP.PublishAddress, ":")
	)

	if len(addrs) > 1 {
		host = addrs[0]
	} else {
		host, _, err = net.SplitHostPort(addrs[0])
		if err != nil {
			host = strings.Split(addrs[0], ":")[0]
		}
	}
	port = ports[len(ports)-1]
	u := &url.URL{
		Scheme: scheme,
		Host:   net.JoinHostPort(host, port),
	}

	return u
}

// ---------------------------------------------------------------------------
// Discovery loop
// ---------------------------------------------------------------------------

// discoveryLoop is a single goroutine that owns both node discovery and
// catalog (/_cat/shards) refresh. It wakes at the sooner of:
//
//   - nextNodes: periodic full discovery (default discoverNodesInterval, 30s)
//   - nextCat:   catalog-only refresh (set by jittered follow-up after
//     discovery, or when the request path signals stale shard data)
//
// After each full node discovery, a jittered catalog follow-up is scheduled.
// The follow-up fires in [0, catRefreshTimeout], giving requests time to
// create index cache entries via getOrCreate so the refresh can merge shard
// placement into them. Jitter avoids thundering herd when many clients
// start simultaneously.
//
// Catalog entry lifecycle (see indexSlotCache):
//   - Created lazily by getOrCreate on first request per index.
//   - Shard data merged by updateFromDiscovery each refresh.
//   - Stale shard data cleared when an index disappears from
//     /_cat/shards (shardNodeCount zeroed, shardNodeNames nilled).
//   - Idle slots (decay counter < 1 for > idleEvictionTTL,
//     default 90m) are evicted entirely.
//   - sync.Map compacted when live count falls to 50% of the
//     high-water mark, reclaiming internal hash table memory.
//
// The request path signals stale catalog data via requestCatRefresh
// (atomic flag, zero contention). The loop picks up the flag on its
// next wake and schedules a cat refresh after minCatRefreshInterval.
//
// A "refresh now" request (requestDiscoveryNow) is also supported:
// it sets a separate atomic flag that causes the loop to run a full
// node + catalog discovery on its next wake, then resets the interval.
func (c *Client) discoveryLoop() {
	nextNodes := time.Now().Add(c.discoverNodesInterval)
	var nextCat time.Time // zero = nothing scheduled

	timer := time.NewTimer(c.discoverNodesInterval)
	defer timer.Stop()

	for {
		select {
		case <-c.ctx.Done():
			return
		case <-timer.C:
		}

		now := time.Now()

		// Pick up request-path signal: "shard catalog may be stale."
		if c.catRefreshNeeded.Swap(false) && nextCat.IsZero() {
			nextCat = now.Add(minCatRefreshInterval)
		}

		// Pick up "discover now" signal (e.g., large topology change).
		if c.discoveryNeeded.Swap(false) {
			nextNodes = now // run immediately on this iteration
		}

		switch {
		case !now.Before(nextNodes):
			// Full node + shard discovery.
			c.tryDiscoverNodes(c.ctx) //nolint:errcheck // errors logged inside

			nextNodes = time.Now().Add(c.discoverNodesInterval)
			nextCat = time.Time{}

			// Jittered follow-up for cold-start cache entries.
			if c.router != nil {
				//nolint:gosec // jitter; cryptographic randomness not needed
				jitter := time.Duration(rand.Int64N(int64(catRefreshTimeout)))
				nextCat = time.Now().Add(jitter)
			}

		case !nextCat.IsZero() && !now.Before(nextCat):
			// Cat-only refresh (no /_nodes call, no topology changes).
			if c.router != nil {
				ctx, cancel := context.WithTimeout(c.ctx, catRefreshTimeout)
				c.fetchAndUpdateShardPlacement(ctx)
				cancel()
			}
			nextCat = time.Time{}
		}

		// Compute next wake-up.
		wakeAt := nextNodes
		if !nextCat.IsZero() && nextCat.Before(wakeAt) {
			wakeAt = nextCat
		}

		// If another signal arrived while we were working, poll sooner.
		if c.catRefreshNeeded.Load() || c.discoveryNeeded.Load() {
			earliest := time.Now().Add(minCatRefreshInterval)
			if earliest.Before(wakeAt) {
				wakeAt = earliest
			}
		}

		delay := time.Until(wakeAt)
		if delay <= 0 {
			delay = time.Millisecond
		}
		timer.Reset(delay)
	}
}

// resetDeadConnViability clears the viability mark on any discovered connection
// that has been continuously dead longer than verifyDeadAfter, so a node that
// never recovers stops being blindly served as a last-resort zombie (see
// Connection.availableForRouting and the lcViable const). Seed connections are
// exempt -- they are always available regardless of this bit.
//
// No-op when verifyDeadAfter <= 0 (the feature is disabled). Enumerates the
// allConns pool; policy pools share the same *Connection pointers, so clearing
// the bit once via CAS is visible everywhere.
func (c *Client) resetDeadConnViability() {
	if c.verifyDeadAfter <= 0 {
		return
	}

	c.mu.RLock()
	pool, ok := c.mu.connectionPool.(*multiServerPool)
	c.mu.RUnlock()
	if !ok {
		return
	}

	_, dead := pool.connectionsByState()
	cutoff := time.Now().Add(-c.verifyDeadAfter)

	for _, conn := range dead {
		if conn.seed {
			continue
		}
		// Skip connections that aren't currently dead or whose viability mark is
		// already clear -- nothing to expire.
		if conn.deadSinceIsZero() || !conn.loadConnState().lifecycle().has(lcViable) {
			continue
		}
		if conn.loadDeadSince().After(cutoff) {
			continue // not dead long enough yet
		}
		conn.mu.Lock()
		if err := conn.casLifecycle(conn.loadConnState(), 0, 0, lcViable); err == nil {
			if dl := loadDebugLogger(); dl != nil {
				dl.Logf("resetDeadConnViability: cleared lcViable on %s (dead since %s)\n",
					conn.URL, conn.loadDeadSince())
			}
		}
		conn.mu.Unlock()
	}
}

// requestCatRefresh signals the discovery loop that shard placement data
// may be stale (e.g., a connection error suggests a node went down).
// Lock-free: sets an atomic flag consumed by discoveryLoop.
func (c *Client) requestCatRefresh() {
	c.catRefreshNeeded.Store(true)
}

// requestDiscoveryNow signals the discovery loop to run a full node +
// catalog discovery on its next wake-up, rather than waiting for the
// regular interval. Used after large topology changes.
// Lock-free: sets an atomic flag consumed by discoveryLoop.
func (c *Client) requestDiscoveryNow() {
	c.discoveryNeeded.Store(true)
}

const (
	// minCatRefreshInterval is the absolute floor for failure-triggered
	// shard catalog refreshes. Even if every connection is flagged, the
	// catalog timer won't fire more frequently than this.
	minCatRefreshInterval = 5 * time.Second
)

// catRefreshTimeout bounds the time spent on a standalone /_cat/shards
// refresh triggered by the follow-up timer. Prevents a stalled HTTP
// request from blocking subsequent refreshes indefinitely.
const catRefreshTimeout = 10 * time.Second

// clearAllNeedsCatUpdate clears the lcNeedsCatUpdate flag on all connections
// in the pool. Called after a successful /_cat/shards refresh confirms current
// shard placement.
func (c *Client) clearAllNeedsCatUpdate() {
	c.mu.RLock()
	pool := c.mu.connectionPool
	c.mu.RUnlock()

	if mp, ok := pool.(*multiServerPool); ok {
		mp.mu.RLock()
		for _, conn := range mp.mu.ready {
			conn.clearNeedsCatUpdate() //nolint:errcheck // pool lock held; only errLifecycleNoop possible
		}
		for _, conn := range mp.mu.dead {
			conn.clearNeedsCatUpdate() //nolint:errcheck // pool lock held; only errLifecycleNoop possible
		}
		mp.mu.RUnlock()
	}
}

// fetchAndUpdateShardPlacement fetches shard-to-node mappings from
// /_cat/shards and propagates the data to router policies in the router
// tree. Non-blocking: failures are logged and scored routing falls back
// to minFanOut without shard-aware partitioning.
//
// Gated by discoveryFeatures: when catShardsEnabled() returns false,
// the entire fetch is skipped.
func (c *Client) fetchAndUpdateShardPlacement(ctx context.Context) {
	if !c.discoveryFeatures.catShardsEnabled() {
		return
	}

	shardPlacement, err := c.getShardPlacement(ctx)
	if err != nil {
		if dl := loadDebugLogger(); dl != nil {
			dl.Logf("Discovery: shard placement fetch failed (continuing with minFanOut): %v\n", err)
		}
		return
	}

	// Enrich placement data with routing metadata (routing_num_shards and
	// number_of_shards) for indexes that don't have it cached yet. This is
	// a one-time fetch per index from _cluster/state/metadata. Both values
	// are needed for murmur3 shard-exact routing; number_of_shards from
	// metadata is the authoritative shard count (the _cat/shards-derived
	// count can be transiently wrong during shard relocation).
	if c.discoveryFeatures.routingNumShardsEnabled() {
		c.fetchRoutingNumShards(ctx, shardPlacement)
	}

	// Count active nodes for fan-out clamping.
	c.mu.RLock()
	pool := c.mu.connectionPool
	c.mu.RUnlock()

	activeNodeCount := 0
	if pool != nil {
		activeNodeCount = len(pool.URLs())
	}

	// Walk the router's policy tree and update any index slot caches.
	updateShardPlacementTree(c.router, shardPlacement, activeNodeCount)

	// Shard placement is now fresh -- clear needsCatUpdate on all connections
	// so they can re-enter scored routing candidate sets.
	c.clearAllNeedsCatUpdate()

	// Clear the atomic flag so the discovery loop doesn't schedule a
	// redundant cat-only refresh on its next wake-up.
	c.catRefreshNeeded.Store(false)
}

// Shard state constants from OpenSearch's IndexShardState.
// Only STARTED shards are considered healthy for routing.
const (
	// shardStateStarted indicates the shard is fully allocated and serving requests.
	shardStateStarted = "STARTED"

	// shardStateInitializing indicates the shard is being recovered or created.
	// The node is not ready to serve requests for this shard.
	shardStateInitializing = "INITIALIZING"

	// shardStateRelocating indicates the shard is being moved to another node.
	// The source node is draining; the target may not be ready yet.
	shardStateRelocating = "RELOCATING"

	// shardStateUnassigned indicates the shard has no node allocated.
	shardStateUnassigned = "UNASSIGNED"
)

// Shard type constants from the /_cat/shards prirep column.
const (
	shardTypePrimary = "p" // Primary shard copy
	shardTypeReplica = "r" // Replica shard copy
)

// catShardsEntry represents a single row from /_cat/shards?format=json.
type catShardsEntry struct {
	Index  string `json:"index"`  // Index name
	Shard  string `json:"shard"`  // Shard number (string from _cat API)
	PriRep string `json:"prirep"` // shardTypePrimary or shardTypeReplica
	State  string `json:"state"`  // shardState* constants
	Node   string `json:"node"`   // Node name (empty for UNASSIGNED)
}

// shardNodeInfo holds per-node shard placement details for an index.
type shardNodeInfo struct {
	Primaries int // Number of primary shards on this node
	Replicas  int // Number of replica shards on this node
}

// shardNodes tracks which nodes host a specific shard number.
// Used by murmur3 shard-exact routing to find the node(s) holding
// the target shard.
type shardNodes struct {
	Primary  string   // Node name hosting the primary copy ("" if unknown/unassigned)
	Replicas []string // Node names hosting replica copies
}

// indexShardPlacement holds the full shard placement data for a single index.
type indexShardPlacement struct {
	// Nodes maps node name -> shard counts on that node.
	// Keyed by the /_cat/shards "node" column, which returns node names.
	// Used for aggregate scoring (replica-preference cost multiplier).
	Nodes map[string]*shardNodeInfo

	// NumberOfPrimaryShards is the count of distinct primary shard numbers
	// for this index. Derived from /_cat/shards by counting distinct
	// entries where prirep == "p".
	NumberOfPrimaryShards int

	// RoutingNumShards is the index's routing_num_shards metadata value,
	// fetched from _cluster/state/metadata. Zero until populated.
	RoutingNumShards int

	// ShardToNodes maps shard number -> the nodes hosting that shard.
	// Used by murmur3 shard-exact routing to resolve the target shard
	// to specific node(s) for direct routing.
	ShardToNodes map[int]*shardNodes
}

// nodeNameSet returns the set of node names for use by rendezvousTopK's hard partition.
func (p *indexShardPlacement) nodeNameSet() map[string]struct{} {
	if p == nil || len(p.Nodes) == 0 {
		return nil
	}
	set := make(map[string]struct{}, len(p.Nodes))
	for name := range p.Nodes {
		set[name] = struct{}{}
	}
	return set
}

// getShardPlacement fetches /_cat/shards?format=json&h=index,shard,prirep,state,node
// and returns a map from index name to its shard placement data.
//
// Only shards in [shardStateStarted] are included -- INITIALIZING, RELOCATING, and
// UNASSIGNED shards are skipped because their nodes are not ready to serve
// requests for those shards.
//
// Requires the indices:monitor/stats cluster permission. If the client's
// credentials lack this privilege, the call returns an error and callers
// should fall back gracefully.
func (c *Client) getShardPlacement(ctx context.Context) (map[string]*indexShardPlacement, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "/_cat/shards", nil)
	if err != nil {
		return nil, fmt.Errorf("creating shard placement request: %w", err)
	}
	req.URL.RawQuery = "format=json&h=index,shard,prirep,state,node"

	conn, err := getConnectionFromPool(c, req)
	if err != nil && errors.Is(err, ErrNoConnections) && c.seedFallbackPool != nil {
		conn, err = c.seedFallbackPool.Next()
	}
	if err != nil {
		return nil, fmt.Errorf("getting connection for shard placement: %w", err)
	}

	c.setReqURL(conn.URL, req)
	c.setReqAuth(conn.URL, req)
	c.setReqUserAgent(req)

	res, err := c.transport.RoundTrip(req)
	if err != nil {
		return nil, fmt.Errorf("shard placement request failed: %w", err)
	}
	if res == nil {
		return nil, fmt.Errorf("nil response from /_cat/shards")
	}
	// Drain on close: this raw RoundTrip path has no Perform buffering, and
	// json.Decode below stops at the end of the JSON value without consuming any
	// trailing bytes, so a plain Close would defeat keep-alive on both the
	// non-200 return and the healthy path.
	defer func() {
		if res.Body != nil {
			//nolint:errcheck // best-effort drain before close on a cleanup path
			io.Copy(io.Discard, res.Body)
			res.Body.Close()
		}
	}()

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("/_cat/shards returned status %d", res.StatusCode)
	}

	if res.Body == nil {
		return nil, fmt.Errorf("nil response body from /_cat/shards")
	}

	var entries []catShardsEntry
	if err := json.NewDecoder(res.Body).Decode(&entries); err != nil {
		return nil, fmt.Errorf("parsing /_cat/shards response: %w", err)
	}

	// Build index -> node placement, filtering to healthy shards only.
	// allPrimaryShards tracks distinct primary shard numbers per index across
	// ALL states (STARTED, RELOCATING, INITIALIZING, etc.) so a shard in flight
	// still contributes to NumberOfPrimaryShards. Counting from ShardToNodes
	// alone would undercount during a relocation window.
	result := make(map[string]*indexShardPlacement)
	allPrimaryShards := make(map[string]map[int]struct{})
	for _, entry := range entries {
		if entry.Index == "" {
			continue
		}

		// Track distinct primary shard numbers regardless of state, so a
		// relocating shard (RELOCATING source row + INITIALIZING destination
		// row, neither STARTED) still appears in the count.
		if entry.PriRep == shardTypePrimary {
			if shardNum, shardErr := strconv.Atoi(entry.Shard); shardErr == nil {
				perIndex, ok := allPrimaryShards[entry.Index]
				if !ok {
					perIndex = make(map[int]struct{})
					allPrimaryShards[entry.Index] = perIndex
				}
				perIndex[shardNum] = struct{}{}
			}
		}

		if entry.Node == "" {
			continue // Unassigned shard (no node)
		}
		if entry.State != shardStateStarted {
			continue // Skip unhealthy states
		}

		placement, ok := result[entry.Index]
		if !ok {
			placement = &indexShardPlacement{
				Nodes:        make(map[string]*shardNodeInfo),
				ShardToNodes: make(map[int]*shardNodes),
			}
			result[entry.Index] = placement
		}

		// Update per-node aggregate counts (existing scoring).
		info, ok := placement.Nodes[entry.Node]
		if !ok {
			info = &shardNodeInfo{}
			placement.Nodes[entry.Node] = info
		}

		// Parse shard number for per-shard tracking.
		shardNum, shardErr := strconv.Atoi(entry.Shard)

		switch entry.PriRep {
		case shardTypePrimary:
			info.Primaries++
			if shardErr == nil {
				copies := placement.ShardToNodes[shardNum]
				if copies == nil {
					copies = &shardNodes{}
					placement.ShardToNodes[shardNum] = copies
				}
				copies.Primary = entry.Node
			}
		case shardTypeReplica:
			info.Replicas++
			if shardErr == nil {
				copies := placement.ShardToNodes[shardNum]
				if copies == nil {
					copies = &shardNodes{}
					placement.ShardToNodes[shardNum] = copies
				}
				copies.Replicas = append(copies.Replicas, entry.Node)
			}
		}
	}

	// Compute NumberOfPrimaryShards per index from the cross-state primary set,
	// not from ShardToNodes alone. ShardToNodes is filtered to STARTED entries
	// (RELOCATING/INITIALIZING rows are excluded), so a relocating shard would
	// be missing there but is still counted in allPrimaryShards.
	for indexName, placement := range result {
		if perIndex, ok := allPrimaryShards[indexName]; ok {
			placement.NumberOfPrimaryShards = len(perIndex)
			continue
		}
		// Fallback: nothing tracked across states (shouldn't happen for a
		// healthy index, but stay defensible). Count from ShardToNodes.
		primaryCount := 0
		for _, copies := range placement.ShardToNodes {
			if copies.Primary != "" {
				primaryCount++
			}
		}
		placement.NumberOfPrimaryShards = primaryCount
	}

	return result, nil
}

// fetchRoutingNumShards fetches routing metadata (routing_num_shards and
// number_of_shards) from the cluster state for indexes that don't have it
// cached yet. This is a one-time fetch per index --once populated, the values
// are stored in the indexShardMap and persist across subsequent discovery cycles.
//
// Both values are needed for murmur3 shard-exact routing:
//
//	shard = floorMod(hash, routingNumShards) / routingFactor
//
// where routingFactor = routingNumShards / numberOfShards.
//
// number_of_shards from index settings is the authoritative shard count.
// The _cat/shards-derived count can be transiently wrong when shards are
// RELOCATING (e.g., during cluster rebalancing triggered by security plugin
// audit index creation), leading to an incorrect routing_factor and wrong
// shard computations.
//
// Server reference:
//
//	OperationRouting.java:calculateScaledShardId
//	MetadataCreateIndexService.java:calculateNumRoutingShards
func (c *Client) fetchRoutingNumShards(ctx context.Context, shardPlacement map[string]*indexShardPlacement) {
	if c.router == nil || len(shardPlacement) == 0 {
		return
	}

	// Find which indexes need routing metadata by checking the cache.
	routerPolicy, ok := c.router.(Policy)
	if !ok {
		return
	}
	cache := findRouterCache(routerPolicy)
	var needFetch []string
	for indexName := range shardPlacement {
		if cache == nil {
			needFetch = append(needFetch, indexName)
			continue
		}

		slot := cache.slotFor(indexName)
		if slot == nil {
			needFetch = append(needFetch, indexName)
			continue
		}

		sm := slot.shardMap.Load()
		if sm == nil || sm.RoutingNumShards == 0 {
			needFetch = append(needFetch, indexName)
			continue
		}

		// Already cached -- carry forward into the placement data.
		shardPlacement[indexName].RoutingNumShards = sm.RoutingNumShards
		if sm.NumberOfPrimaryShards > 0 {
			shardPlacement[indexName].NumberOfPrimaryShards = sm.NumberOfPrimaryShards
		}
	}

	if len(needFetch) == 0 {
		return
	}

	rns, err := c.getRoutingMeta(ctx, needFetch)
	if err != nil {
		if dl := loadDebugLogger(); dl != nil {
			dl.Logf(
				"Discovery: routing metadata fetch failed for %d indexes (shard-exact routing unavailable): %v\n",
				len(needFetch), err)
		}
		return
	}

	for indexName, meta := range rns {
		if placement, ok := shardPlacement[indexName]; ok {
			placement.RoutingNumShards = meta.RoutingNumShards
			// Use the authoritative number_of_shards from index metadata
			// rather than the _cat/shards-derived count, which can be
			// transiently wrong when shards are RELOCATING.
			if meta.NumberOfShards > 0 {
				placement.NumberOfPrimaryShards = meta.NumberOfShards
			}
		}
	}
}

// indexRoutingMeta holds the authoritative routing metadata for an index,
// fetched from _cluster/state/metadata. These values are immutable for the
// lifetime of an index and are needed by murmur3 shard-exact routing.
type indexRoutingMeta struct {
	RoutingNumShards int // Hash modulus (e.g. 640 for a 5-shard index)
	NumberOfShards   int // Configured number_of_shards from index settings
}

// getRoutingMeta fetches routing metadata (routing_num_shards and
// number_of_shards) for the given indexes from _cluster/state/metadata.
//
// number_of_shards from index settings is the authoritative shard count,
// unlike the _cat/shards-derived count which can be transiently wrong when
// shards are RELOCATING or INITIALIZING. The server's shard routing formula
// uses the configured number_of_shards (via routingFactor), so the client
// must use the same value for correct murmur3 shard computation.
func (c *Client) getRoutingMeta(ctx context.Context, indexes []string) (map[string]indexRoutingMeta, error) {
	path := "/_cluster/state/metadata/" + strings.Join(indexes, ",")

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, fmt.Errorf("creating routing metadata request: %w", err)
	}
	req.URL.RawQuery = "filter_path=metadata.indices.*.routing_num_shards,metadata.indices.*.settings.index.number_of_shards"

	conn, err := getConnectionFromPool(c, req)
	if err != nil && errors.Is(err, ErrNoConnections) && c.seedFallbackPool != nil {
		conn, err = c.seedFallbackPool.Next()
	}
	if err != nil {
		return nil, fmt.Errorf("getting connection for routing metadata: %w", err)
	}

	c.setReqURL(conn.URL, req)
	c.setReqAuth(conn.URL, req)
	c.setReqUserAgent(req)

	res, err := c.transport.RoundTrip(req)
	if err != nil {
		return nil, fmt.Errorf("routing metadata request failed: %w", err)
	}
	if res == nil {
		return nil, fmt.Errorf("nil response from _cluster/state/metadata")
	}
	// Drain on close: raw RoundTrip path, and the json.Decode below leaves any
	// trailing bytes unread, so a plain Close would defeat keep-alive on both
	// the non-200 return and the healthy path.
	defer func() {
		if res.Body != nil {
			//nolint:errcheck // best-effort drain before close on a cleanup path
			io.Copy(io.Discard, res.Body)
			res.Body.Close()
		}
	}()

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("_cluster/state/metadata returned status %d", res.StatusCode)
	}

	if res.Body == nil {
		return nil, fmt.Errorf("nil response body from _cluster/state/metadata")
	}

	// Response shape:
	// {"metadata":{"indices":{"idx":{
	//   "routing_num_shards": 640,
	//   "settings": {"index": {"number_of_shards": "5"}}
	// }}}}
	var clusterState struct {
		Metadata struct {
			Indices map[string]struct {
				RoutingNumShards int `json:"routing_num_shards"`
				Settings         struct {
					Index struct {
						NumberOfShards string `json:"number_of_shards"`
					} `json:"index"`
				} `json:"settings"`
			} `json:"indices"`
		} `json:"metadata"`
	}
	if err := json.NewDecoder(res.Body).Decode(&clusterState); err != nil {
		return nil, fmt.Errorf("parsing _cluster/state/metadata response: %w", err)
	}

	result := make(map[string]indexRoutingMeta, len(clusterState.Metadata.Indices))
	for name, meta := range clusterState.Metadata.Indices {
		if meta.RoutingNumShards > 0 {
			rm := indexRoutingMeta{RoutingNumShards: meta.RoutingNumShards}
			if n, err := strconv.Atoi(meta.Settings.Index.NumberOfShards); err == nil && n > 0 {
				rm.NumberOfShards = n
			}
			result[name] = rm
		}
	}

	return result, nil
}
