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
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"sync/atomic"
	"time"
)

// Measurable defines the interface for transports supporting metrics.
type Measurable interface {
	Metrics() (Metrics, error)
}

// Response status codes are bounded to [statusMin, statusMax). Anything
// outside that range is folded into a single overflow bucket so a malformed
// upstream status is still counted rather than panicking on a bad index.
const (
	statusMin      = 100
	statusMax      = 600
	statusBuckets  = statusMax - statusMin // 500 in-range buckets
	statusOverflow = -1                    // map key for the overflow bucket
)

// Metrics represents the transport metrics.
type Metrics struct {
	Requests int `json:"requests"`
	Failures int `json:"failures"`
	// Responses counts responses by HTTP status code. Any status outside the
	// valid [100, 600) range is folded into a single overflow bucket keyed by
	// -1 (statusOverflow) rather than its literal code.
	Responses map[int]int `json:"responses"`

	// Connection pool state.
	// LiveConnections is the number of non-dead connections (active + standby).
	// Named 'Live' for JSON API compatibility; corresponds to the internal ready list.
	LiveConnections int `json:"live_connections"`
	DeadConnections int `json:"dead_connections"`

	// Connection lifecycle counters
	ConnectionsPromoted int `json:"connections_promoted"` // Dead -> Ready (resurrected successfully)
	ConnectionsDemoted  int `json:"connections_demoted"`  // Ready -> Dead (marked dead)
	ZombieConnections   int `json:"zombie_connections"`   // Taken from dead list and forcibly retried

	// Client capabilities and health
	HealthChecks        int `json:"health_checks"`         // Baseline GET / health checks performed
	ClusterHealthChecks int `json:"cluster_health_checks"` // GET /_cluster/health?local=true health checks performed
	HealthChecksSuccess int `json:"health_checks_success"` // Successful health check outcomes
	HealthChecksFailed  int `json:"health_checks_failed"`  // Failed health check outcomes
	OverloadedServers   int `json:"overloaded_servers"`    // Number of servers client thinks are overloaded

	// Standby pool state
	StandbyConnections int `json:"standby_connections"` // Current standby pool size
	StandbyPromotions  int `json:"standby_promotions"`  // Standby -> Active transitions
	StandbyDemotions   int `json:"standby_demotions"`   // Active -> Standby transitions

	Connections []fmt.Stringer `json:"connections"`

	// Per-policy breakdown. Part of the detailed-metrics path: populated only
	// when EnableMetrics is set and a router with policies is active; nil otherwise.
	Policies []PolicySnapshot `json:"policies,omitempty"`

	// Router cache state. Part of the detailed-metrics path: populated only
	// when EnableMetrics is set and scored routing is active; nil otherwise.
	Router *RouterSnapshot `json:"router,omitempty"`
}

// RouterSnapshot is a point-in-time summary of the routing cache.
type RouterSnapshot struct {
	Indexes []IndexRouterState   `json:"indexes,omitempty"`
	Config  RouterSnapshotConfig `json:"config"`
}

// IndexRouterState is per-index routing state from the index routing cache.
type IndexRouterState struct {
	Name        string     `json:"name"`
	FanOut      int        `json:"fan_out"`
	ShardNodes  int        `json:"shard_nodes"`
	RequestRate float64    `json:"request_rate"`
	IdleSince   *time.Time `json:"idle_since,omitempty"`
}

// RouterSnapshotConfig holds the effective configuration values for
// the index routing cache.
type RouterSnapshotConfig struct {
	MinFanOut       int     `json:"min_fan_out"`
	MaxFanOut       int     `json:"max_fan_out"`
	DecayFactor     float64 `json:"decay_factor"`
	FanOutPerReq    float64 `json:"fan_out_per_request"`
	IdleEvictionTTL string  `json:"idle_eviction_ttl"`
}

// sortIndexRouterStates sorts index states by name for deterministic output.
func sortIndexRouterStates(states []IndexRouterState) {
	slices.SortFunc(states, func(a, b IndexRouterState) int {
		return strings.Compare(a.Name, b.Name)
	})
}

// ConnectionMetric represents metric information for a connection.
type ConnectionMetric struct {
	URL              string     `json:"url"`
	Failures         int        `json:"failures,omitempty"`
	IsDead           bool       `json:"dead,omitempty"`
	IsStandby        bool       `json:"standby,omitempty"`
	IsOverloaded     bool       `json:"overloaded,omitempty"`
	NeedsCatUpdate   bool       `json:"needs_cat_update,omitempty"`
	IsWarmingUp      bool       `json:"warming_up,omitempty"`
	IsHealthChecking bool       `json:"health_checking,omitempty"`
	Weight           int        `json:"weight,omitempty"`
	DeadSince        *time.Time `json:"dead_since,omitempty"`
	OverloadedSince  *time.Time `json:"overloaded_since,omitempty"`
	State            ConnState  `json:"state"`

	// Router metrics (populated when RTT ring or load counter has data)
	RTTBucket *int64   `json:"rtt_bucket,omitempty"`
	RTTMedian *string  `json:"rtt_median,omitempty"`
	EstLoad   *float64 `json:"est_load,omitempty"`
	MCSR      *int     `json:"mcsr,omitempty"` // Current max_concurrent_shard_requests value (nil when adaptive MCSR disabled)

	Meta struct {
		ID    string   `json:"id"`
		Name  string   `json:"name"`
		Roles []string `json:"roles"`
	} `json:"meta"`
}

// PolicySnapshot is a point-in-time snapshot of one policy's pool partitions.
type PolicySnapshot struct {
	Name                string `json:"name"`
	Enabled             bool   `json:"enabled"`
	ActiveCount         int    `json:"active_count"`
	StandbyCount        int    `json:"standby_count"`
	DeadCount           int    `json:"dead_count"`
	ActiveListCap       int    `json:"active_list_cap"`
	WarmingCount        int    `json:"warming_count"`
	HealthCheckingCount int    `json:"health_checking_count"`

	// Per-pool request counters
	Requests      int64 `json:"requests"`       // Connections returned by Next()
	Successes     int64 `json:"successes"`      // Resurrections via OnSuccess()
	Failures      int64 `json:"failures"`       // Demotions via OnFailure()
	WarmupSkips   int64 `json:"warmup_skips"`   // Requests skipped during warmup
	WarmupAccepts int64 `json:"warmup_accepts"` // Requests accepted during warmup
}

// String returns the policy snapshot as a compact string.
func (ps PolicySnapshot) String() string {
	enabledStr := "on"
	if !ps.Enabled {
		enabledStr = "off"
	}
	return fmt.Sprintf("%q (%s, cap=%d): active=%d standby=%d dead=%d warming=%d checking=%d | req=%d ok=%d fail=%d skip=%d accept=%d",
		ps.Name, enabledStr, ps.ActiveListCap, ps.ActiveCount, ps.StandbyCount, ps.DeadCount, ps.WarmingCount, ps.HealthCheckingCount,
		ps.Requests, ps.Successes, ps.Failures, ps.WarmupSkips, ps.WarmupAccepts)
}

// ConnectionMetricCallback augments per-connection metrics at snapshot time.
// Receives parallel slices of connections and their metrics; the callback
// modifies cms elements in place (e.g., setting MCSR). Policies register
// callbacks during [policyConfigurable.configurePolicySettings]; the
// [Client.Metrics] method invokes them once per snapshot. Callbacks must be
// safe for concurrent use and should be cheap (atomic loads, no allocations).
type ConnectionMetricCallback func(conns []*Connection, cms []ConnectionMetric) error

// PolicyMetricCallback returns a point-in-time snapshot for one policy's
// connection pool. Leaf policies (RolePolicy, RoundRobinPolicy,
// CoordinatorPolicy) register a callback during configurePolicySettings.
type PolicyMetricCallback func() (PolicySnapshot, error)

// MetricsCallback augments the top-level [Metrics] struct at snapshot time.
// Used for cross-cutting concerns like the router cache snapshot. Callbacks
// should check for nil/zero before overwriting (idempotent registration).
type MetricsCallback func(m *Metrics) error

// metrics represents the inner state of metrics.
type metrics struct {
	requests atomic.Int64
	failures atomic.Int64

	// Connection lifecycle counters
	connectionsPromoted atomic.Int64 // Dead -> Ready (resurrected successfully)
	connectionsDemoted  atomic.Int64 // Ready -> Dead (marked dead)
	zombieConnections   atomic.Int64 // Taken from dead list and forcibly retried

	// Health check counters
	healthChecks        atomic.Int64 // Baseline GET / health checks performed
	clusterHealthChecks atomic.Int64 // GET /_cluster/health?local=true health checks performed
	healthChecksSuccess atomic.Int64 // Successful health check outcomes (from DefaultHealthCheck)
	healthChecksFailed  atomic.Int64 // Failed health check outcomes (from DefaultHealthCheck)

	// Standby pool lifecycle counters
	standbyPromotions atomic.Int64 // Standby -> Active
	standbyDemotions  atomic.Int64 // Active -> Standby

	// detailed gates the expensive detailed-metrics path: registration and
	// invocation of the per-connection / per-policy / per-snapshot callbacks and
	// the connection-state enumeration in Metrics. The cheap per-request counters
	// above are populated regardless of detailed. Set from Config.EnableMetrics.
	detailed bool

	// Metric callbacks registered by policies at init time.
	// Immutable after client construction; no synchronization needed.
	connMetricCallbacks []ConnectionMetricCallback // batch per-connection
	policyCallbacks     []PolicyMetricCallback     // per-policy snapshot
	snapshotCallbacks   []MetricsCallback          // per-snapshot augmentation

	// responses counts HTTP responses by status code, lock-free. Index i holds
	// the count for status code statusMin+i; responsesOverflow holds any code
	// outside [statusMin, statusMax). Snapshotted in responsesSnapshot.
	responses         [statusBuckets]atomic.Int64
	responsesOverflow atomic.Int64
}

// detailedEnabled reports whether the detailed-metrics path is active. It is nil-safe:
// a nil *metrics (no metrics struct wired) reports false, so callers can guard
// detailed-only work with config.metrics.detailedEnabled() without a separate nil check.
func (m *metrics) detailedEnabled() bool {
	return m != nil && m.detailed
}

// incrementResponse records one response with the given status code. It is
// lock-free: a single atomic add to the bucket for statusCode, or to the
// overflow bucket when statusCode is outside [statusMin, statusMax).
func (m *metrics) incrementResponse(statusCode int) {
	if statusCode < statusMin || statusCode >= statusMax {
		m.responsesOverflow.Add(1)
		return
	}
	m.responses[statusCode-statusMin].Add(1)
}

// responsesSnapshot returns a map of status code to count. Codes with a zero
// count are omitted; the overflow bucket, when non-zero, is keyed by
// statusOverflow.
func (m *metrics) responsesSnapshot() map[int]int {
	out := make(map[int]int)
	for i := range m.responses {
		if n := m.responses[i].Load(); n > 0 {
			out[statusMin+i] = int(n)
		}
	}
	if n := m.responsesOverflow.Load(); n > 0 {
		out[statusOverflow] = int(n)
	}
	return out
}

// Metrics returns the transport metrics. The detailed fields -- per-connection
// enumeration, per-policy snapshots, and the router snapshot -- are populated
// only when Config.EnableMetrics is set.
func (c *Client) Metrics() (Metrics, error) {
	if c.metrics == nil {
		// Defensive: a custom transport could embed *Client without the
		// standard constructor. Treat as no metrics available.
		return Metrics{}, errors.New("transport metrics not enabled")
	}

	m := Metrics{
		Requests:  int(c.metrics.requests.Load()),
		Failures:  int(c.metrics.failures.Load()),
		Responses: c.metrics.responsesSnapshot(),

		ConnectionsPromoted: int(c.metrics.connectionsPromoted.Load()),
		ConnectionsDemoted:  int(c.metrics.connectionsDemoted.Load()),
		ZombieConnections:   int(c.metrics.zombieConnections.Load()),

		HealthChecks:        int(c.metrics.healthChecks.Load()),
		ClusterHealthChecks: int(c.metrics.clusterHealthChecks.Load()),
		HealthChecksSuccess: int(c.metrics.healthChecksSuccess.Load()),
		HealthChecksFailed:  int(c.metrics.healthChecksFailed.Load()),

		StandbyPromotions: int(c.metrics.standbyPromotions.Load()),
		StandbyDemotions:  int(c.metrics.standbyDemotions.Load()),
	}

	// Detailed-metrics path: connection enumeration + callbacks. The detailed-only
	// fields (LiveConnections, DeadConnections, OverloadedServers,
	// StandbyConnections, Connections, Policies, Router) stay zero/nil when
	// the detailed path is off.
	if !c.metrics.detailed {
		return m, nil
	}

	// Get connections from current connection pool
	var ready, dead []*Connection
	var singleConns []*Connection
	c.mu.RLock()
	if c.mu.connectionPool != nil {
		switch pool := c.mu.connectionPool.(type) {
		case *multiServerPool:
			ready, dead = pool.connectionsByState()
		case *singleServerPool:
			singleConns = pool.connections()
		}
	}
	c.mu.RUnlock()

	m.LiveConnections = len(ready) + len(singleConns)
	m.DeadConnections = len(dead)

	// Build per-connection metrics. Each connection's connState atomic
	// determines isDead/isStandby/isOverloaded -- no positional tricks needed.
	//
	// Deduplicate connections: the same *Connection can appear in multiple
	// policy pools, so collect into a set and iterate the unique keys.
	overloadedCount := 0
	standbyCount := 0
	var callbackErrs []error

	connSet := make(map[*Connection]struct{}, len(singleConns)+len(ready)+len(dead))
	for _, conn := range singleConns {
		connSet[conn] = struct{}{}
	}
	for _, conn := range ready {
		connSet[conn] = struct{}{}
	}
	for _, conn := range dead {
		connSet[conn] = struct{}{}
	}

	allConns := make([]*Connection, 0, len(connSet))
	for conn := range connSet {
		allConns = append(allConns, conn)
	}

	cms := make([]ConnectionMetric, len(allConns))
	for i, conn := range allConns {
		cms[i] = buildConnectionMetric(conn)
		if cms[i].IsOverloaded {
			overloadedCount++
		}
		if cms[i].IsStandby {
			standbyCount++
		}
	}

	// Run batch connection metric callbacks (e.g., MCSR injection).
	for _, cb := range c.metrics.connMetricCallbacks {
		if err := cb(allConns, cms); err != nil {
			callbackErrs = append(callbackErrs, err)
		}
	}

	m.Connections = make([]fmt.Stringer, len(cms))
	for i := range cms {
		m.Connections[i] = cms[i]
	}
	m.OverloadedServers = overloadedCount
	m.StandbyConnections = standbyCount

	// Collect per-policy snapshots via registered callbacks.
	for _, cb := range c.metrics.policyCallbacks {
		snap, err := cb()
		if err != nil {
			callbackErrs = append(callbackErrs, err)
			continue
		}
		m.Policies = append(m.Policies, snap)
	}

	// Include the flat client pool snapshot (not a policy, always present).
	c.mu.RLock()
	if c.mu.connectionPool != nil {
		if pool, ok := c.mu.connectionPool.(*multiServerPool); ok {
			snap := pool.snapshot()
			snap.Enabled = true // flat/client pool is always enabled
			m.Policies = append(m.Policies, snap)
		}
	}
	c.mu.RUnlock()

	// Run snapshot-level callbacks (e.g., router cache snapshot).
	for _, cb := range c.metrics.snapshotCallbacks {
		if err := cb(&m); err != nil {
			callbackErrs = append(callbackErrs, err)
		}
	}

	return m, errors.Join(callbackErrs...)
}

// buildConnectionMetric creates a ConnectionMetric from a Connection.
// State flags (isDead, isStandby, isOverloaded) are derived from the connection's
// connState atomic -- no positional or parameter-based inference needed.
func buildConnectionMetric(c *Connection) ConnectionMetric {
	state := c.loadConnState()
	lc := state.lifecycle()

	// Read the dead/overloaded timestamps lock-free: they are written under c.mu
	// but safe to read without it, so the metrics snapshot avoids the dominant
	// per-connection lock contention against the per-request OnSuccess/OnFailure
	// writers.
	deadSince := c.loadDeadSince()
	overloadedAt := c.loadOverloadedAt()

	cm := ConnectionMetric{
		URL:              c.URL.String(),
		IsDead:           lc.has(lcUnknown) && lc&(lcActive|lcStandby) == 0,
		IsStandby:        lc.has(lcStandby),
		IsOverloaded:     lc.has(lcOverloaded),
		NeedsCatUpdate:   lc.has(lcNeedsCatUpdate),
		IsWarmingUp:      state.isWarmingUp(),
		IsHealthChecking: lc.has(lcHealthChecking),
		Failures:         int(c.failures.Load()),
		Weight:           c.effectiveWeight(),
		State:            ConnState{packed: int64(state)},
	}

	if !deadSince.IsZero() {
		deadSinceCopy := deadSince
		cm.DeadSince = &deadSinceCopy
	}

	if cm.IsOverloaded && !overloadedAt.IsZero() {
		overloadedAtCopy := overloadedAt
		cm.OverloadedSince = &overloadedAtCopy
	}

	if c.ID != "" {
		cm.Meta.ID = c.ID
	}

	if c.Name != "" {
		cm.Meta.Name = c.Name
	}

	if len(c.Roles) > 0 {
		cm.Meta.Roles = c.Roles.toSlice()
	}

	// Populate routing metrics when data is available.
	if bucket := c.RTTBucket(); bucket >= 0 {
		cm.RTTBucket = &bucket
		median := c.RTTMedian().String()
		cm.RTTMedian = &median
	}
	if load := c.EstLoad(); load > 0 {
		cm.EstLoad = &load
	}

	return cm
}

// String returns the metrics as a string.
func (m Metrics) String() string {
	var (
		i int
		b strings.Builder
	)
	b.WriteString("{")

	b.WriteString("Requests:")
	b.WriteString(strconv.Itoa(m.Requests))

	b.WriteString(" Failures:")
	b.WriteString(strconv.Itoa(m.Failures))

	b.WriteString(" HealthChecks:")
	b.WriteString(strconv.Itoa(m.HealthChecks))

	b.WriteString(" ClusterHealthChecks:")
	b.WriteString(strconv.Itoa(m.ClusterHealthChecks))

	b.WriteString(" HealthChecksSuccess:")
	b.WriteString(strconv.Itoa(m.HealthChecksSuccess))

	b.WriteString(" HealthChecksFailed:")
	b.WriteString(strconv.Itoa(m.HealthChecksFailed))

	if len(m.Responses) > 0 {
		b.WriteString(" Responses: ")
		b.WriteString("[")

		for code, num := range m.Responses {
			b.WriteString(strconv.Itoa(code))
			b.WriteString(":")
			b.WriteString(strconv.Itoa(num))
			if i+1 < len(m.Responses) {
				b.WriteString(", ")
			}
			i++
		}
		b.WriteString("]")
	}

	b.WriteString(" Connections: [")
	for i, c := range m.Connections {
		b.WriteString(c.String())
		if i+1 < len(m.Connections) {
			b.WriteString(", ")
		}
	}
	b.WriteString("]")

	if len(m.Policies) > 0 {
		b.WriteString(" Policies: [")
		for i, p := range m.Policies {
			b.WriteString(p.String())
			if i+1 < len(m.Policies) {
				b.WriteString(", ")
			}
		}
		b.WriteString("]")
	}

	b.WriteString("}")
	return b.String()
}

// String returns the connection information as a string.
func (cm ConnectionMetric) String() string {
	var b strings.Builder
	b.WriteString("{")
	b.WriteString(cm.URL)

	// Show lifecycle state
	fmt.Fprintf(&b, " state=%s", cm.State)

	// Show roles if known
	if len(cm.Meta.Roles) > 0 {
		fmt.Fprintf(&b, " roles=%v", cm.Meta.Roles)
	}

	// Show name if known
	if cm.Meta.Name != "" {
		fmt.Fprintf(&b, " name=%s", cm.Meta.Name)
	}

	if cm.Failures > 0 {
		fmt.Fprintf(&b, " failures=%d", cm.Failures)
	}
	if cm.DeadSince != nil {
		fmt.Fprintf(&b, " dead_since=%s", cm.DeadSince.Local().Format(time.Stamp))
	}
	if cm.OverloadedSince != nil {
		fmt.Fprintf(&b, " overloaded_since=%s", cm.OverloadedSince.Local().Format(time.Stamp))
	}
	b.WriteString("}")
	return b.String()
}
