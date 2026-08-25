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
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"sync/atomic"
)

// Compile-time interface compliance checks.
var (
	_ Policy             = (*MuxPolicy)(nil)
	_ policyConfigurable = (*MuxPolicy)(nil)
	_ policyTyped        = (*MuxPolicy)(nil)
	_ policyOverrider    = (*MuxPolicy)(nil)
)

// ErrNoRouteMatched is returned when no route matches the request pattern.
var ErrNoRouteMatched = errors.New("no route matched request")

// routeAttr is a bitfield of per-route attributes applied when the route matches.
// Attributes are set at route construction time and evaluated by [MuxPolicy.Eval].
type routeAttr uint32

const (
	// attrInjectAdaptiveMCSR marks a route as eligible for client-side adaptive
	// max_concurrent_shard_requests injection. Only endpoints that accept this
	// query parameter (_search and _msearch) should set this bit. Other
	// search-pool endpoints (_count, _delete_by_query, _validate/query, etc.)
	// are routed to the same pool for connection selection but do not accept
	// the parameter — OpenSearch returns HTTP 400 if it is present.
	attrInjectAdaptiveMCSR routeAttr = 1 << iota
)

//nolint:gochecknoglobals // Shared HTTP methods map for splitMuxPattern
var httpMethods = map[string]struct{}{
	http.MethodGet:     {},
	http.MethodHead:    {},
	http.MethodPost:    {},
	http.MethodPut:     {},
	http.MethodPatch:   {},
	http.MethodDelete:  {},
	http.MethodConnect: {},
	http.MethodOptions: {},
	http.MethodTrace:   {},
}

// Route represents a pattern-to-policy mapping for HTTP request routing.
type Route interface {
	// Policy returns the policy to use for matching requests.
	Policy() Policy
	// Attrs returns the per-route attribute bitfield.
	Attrs() routeAttr
	// PoolName returns the thread pool name for congestion tracking.
	// Empty string means use the default pool.
	PoolName() string
	// OpID returns the operation identifier for this route.
	OpID() OperationID
}

// RouteMux represents a route with an HTTP method+path pattern.
type RouteMux struct {
	Pattern     string      // HTTP pattern (e.g., "POST /_bulk", "GET /_search", "/")
	policy      Policy      // Policy to use for matching requests
	operationID OperationID // Operation identifier for classification
	attrs       routeAttr   // Per-route attributes applied when the route matches
	poolName    string      // Thread pool name for congestion tracking (e.g., "search", "write")
}

// NewRouteMux creates a new route with validation.
func NewRouteMux(pattern string, policy Policy) (Route, error) {
	return NewRouteMuxAttrs(pattern, policy, 0)
}

// NewRouteMuxAttrs creates a new route with attributes.
func NewRouteMuxAttrs(pattern string, policy Policy, attrs routeAttr) (Route, error) {
	if policy == nil {
		return nil, errors.New("policy cannot be nil")
	}

	if _, _, err := splitMuxPattern(pattern); err != nil {
		return nil, fmt.Errorf("invalid pattern: %w", err)
	}

	return &RouteMux{
		Pattern: pattern,
		policy:  policy,
		attrs:   attrs,
	}, nil
}

// mustNewRouteMux creates a new route, panicking on error.
// Used for internal route construction where patterns are known to be valid.
func mustNewRouteMux(pattern string, policy Policy) Route {
	return mustNewRouteMuxAttrs(pattern, policy, 0)
}

// mustNewRouteMuxAttrs creates a new route with attributes,
// panicking on error. Used for internal route construction.
func mustNewRouteMuxAttrs(pattern string, policy Policy, attrs routeAttr) Route {
	route, err := NewRouteMuxAttrs(pattern, policy, attrs)
	if err != nil {
		panic("invalid RouteMux: " + err.Error())
	}
	return route
}

// RouteBuilder constructs a RouteMux with a fluent API. Use [NewRoute] to
// create a builder, chain methods for attributes and pool name, then call
// [MustBuild] to produce the [Route].
type RouteBuilder struct {
	pattern     string
	policy      Policy
	operationID OperationID
	attrs       routeAttr
	poolName    string
}

// NewRoute creates a RouteBuilder for the given pattern and policy.
func NewRoute(pattern string, policy Policy) *RouteBuilder {
	return &RouteBuilder{pattern: pattern, policy: policy}
}

// Pool sets the thread pool name for congestion tracking.
func (b *RouteBuilder) Pool(name string) *RouteBuilder {
	b.poolName = name
	return b
}

// Op sets the operation identifier for this route.
func (b *RouteBuilder) Op(id OperationID) *RouteBuilder {
	b.operationID = id
	return b
}

// InjectAdaptiveMCSR marks this route for client-side adaptive
// max_concurrent_shard_requests injection. Only endpoints whose server-side
// REST action accepts the parameter should use this (currently _search and
// _msearch). See [attrInjectAdaptiveMCSR].
func (b *RouteBuilder) InjectAdaptiveMCSR() *RouteBuilder {
	b.attrs |= attrInjectAdaptiveMCSR
	return b
}

// MustBuild validates the pattern and returns a RouteMux, panicking on error.
func (b *RouteBuilder) MustBuild() Route {
	if b.policy == nil {
		panic("RouteBuilder: policy cannot be nil")
	}
	if _, _, err := splitMuxPattern(b.pattern); err != nil {
		panic("RouteBuilder: invalid pattern: " + err.Error())
	}
	return &RouteMux{
		Pattern:     b.pattern,
		policy:      b.policy,
		operationID: b.operationID,
		attrs:       b.attrs,
		poolName:    b.poolName,
	}
}

// Policy returns the underlying policy used by this RouteMux.
func (r *RouteMux) Policy() Policy { return r.policy }

// Attrs returns the per-route attribute bitfield.
func (r *RouteMux) Attrs() routeAttr { return r.attrs }

// PoolName returns the thread pool name for congestion tracking.
func (r *RouteMux) PoolName() string { return r.poolName }

// OpID returns the operation identifier for this route.
func (r *RouteMux) OpID() OperationID { return r.operationID }

// MuxPolicy is a connection policy multiplexer that routes requests to different policies
// based on HTTP patterns, using a trie for endpoints. Literal children always match
// before wildcards, so system and index endpoints coexist without ambiguity.
type MuxPolicy struct {
	pathTrie       routeTrie           // Trie for all endpoints (system and index)
	uniquePolicies map[Policy]struct{} // Set of unique policies for lifecycle management
	policyState    atomic.Int32        // Bitfield: psEnabled|psDisabled|psEnvEnabled|psEnvDisabled
}

func (p *MuxPolicy) policyTypeName() string      { return policyTypeNameMux }
func (p *MuxPolicy) setEnvOverride(enabled bool) { psSetEnvOverride(&p.policyState, enabled) }

// NewMuxPolicy creates a new policy multiplexer with the given routes.
//
// Routes are registered into a single trie. Literal path segments always take
// priority over the {index} wildcard, so system endpoints (/_search, /_snapshot, etc.)
// and index endpoints (/{index}/_search, etc.) coexist without conflict.
func NewMuxPolicy(routes []Route) Policy {
	p := &MuxPolicy{
		uniquePolicies: make(map[Policy]struct{}),
	}

	for _, route := range routes {
		p.uniquePolicies[route.Policy()] = struct{}{}

		switch r := route.(type) {
		case *RouteMux:
			method, path, err := splitMuxPattern(r.Pattern)
			if err != nil {
				panic(fmt.Sprintf("invalid pattern: %v", err))
			}

			p.pathTrie.add(
				[]string{method}, path,
				r.Policy(), r.Attrs(), r.PoolName(), r.OpID(),
			)

		default:
			panic(fmt.Sprintf("unsupported Route type: %T", r))
		}
	}

	return p
}

// splitMuxPattern splits "METHOD /path" into method and path, validating both.
func splitMuxPattern(pattern string) (string, string, error) {
	if pattern == "" {
		return "", "", errors.New("pattern cannot be empty")
	}

	routeFields := strings.Fields(pattern)
	if len(routeFields) != 2 {
		return "", "", fmt.Errorf("route pattern must include HTTP method: %q", pattern)
	}
	if _, found := httpMethods[routeFields[0]]; !found {
		return "", "", fmt.Errorf("route pattern method invalid: %q", pattern)
	}

	return routeFields[0], routeFields[1], nil
}

// DiscoveryUpdate updates all sub-policies and caches the enabled state.
func (p *MuxPolicy) DiscoveryUpdate(added, removed, unchanged []*Connection) error {
	var firstError error
	hasEnabledPolicy := false

	// Update all unique policies (no ordering dependency)
	for policy := range p.uniquePolicies {
		if err := policy.DiscoveryUpdate(added, removed, unchanged); err != nil && firstError == nil {
			firstError = err
		}

		// Cache if any policy is enabled
		if !hasEnabledPolicy && policy.IsEnabled() {
			hasEnabledPolicy = true
		}
	}

	psSetEnabled(&p.policyState, hasEnabledPolicy)
	return firstError
}

// IsEnabled uses cached state to quickly determine if this mux can route requests.
func (p *MuxPolicy) IsEnabled() bool {
	return psIsEnabled(p.policyState.Load())
}

// Eval routes the request based on HTTP patterns and delegates to the matching policy.
func (p *MuxPolicy) Eval(ctx context.Context, req *http.Request) (NextHop, error) {
	if p.policyState.Load()&psEnvDisabled != 0 || req == nil {
		return NextHop{}, nil
	}

	m, matched := p.pathTrie.match(req.Method, req.URL.Path)
	if !matched {
		return NextHop{}, nil
	}

	hop, err := m.policy.Eval(ctx, req)
	if hop.Conn != nil && err == nil {
		if hop.PoolName == "" && m.poolName != "" {
			hop.PoolName = m.poolName
		}
		// Only inject adaptive MCSR for routes that explicitly opt in.
		// The poolRouter computes the value for the entire "search" pool,
		// but only _search and _msearch accept the query parameter.
		if hop.MaxConcurrentShardRequests > 0 && m.attrs&attrInjectAdaptiveMCSR == 0 {
			hop.MaxConcurrentShardRequests = 0
		}
	}
	return hop, err
}

// CheckDead delegates to all unique sub-policies.
func (p *MuxPolicy) CheckDead(ctx context.Context, healthCheck HealthCheckFunc) error {
	var firstError error
	for policy := range p.uniquePolicies {
		if err := policy.CheckDead(ctx, healthCheck); err != nil && firstError == nil {
			firstError = err
		}
	}
	return firstError
}

// RotateStandby delegates to all unique sub-policies, summing successful rotations.
func (p *MuxPolicy) RotateStandby(ctx context.Context, count int) (int, error) {
	var (
		total int
		errs  []error
	)
	for policy := range p.uniquePolicies {
		n, err := policy.RotateStandby(ctx, count)
		total += n
		if err != nil {
			errs = append(errs, err)
		}
	}
	return total, errors.Join(errs...)
}

// configurePolicySettings configures pool settings for all unique sub-policies.
func (p *MuxPolicy) configurePolicySettings(config policyConfig) error {
	var firstError error

	// Configure all unique policies
	for policy := range p.uniquePolicies {
		if configurablePolicy, ok := policy.(policyConfigurable); ok {
			if err := configurablePolicy.configurePolicySettings(config); err != nil && firstError == nil {
				firstError = err
			}
		}
	}

	return firstError
}

// childPolicies returns the unique sub-policies for tree walking.
// Sorted by policySortKey for deterministic path assignment.
func (p *MuxPolicy) childPolicies() []Policy {
	policies := make([]Policy, 0, len(p.uniquePolicies))
	for policy := range p.uniquePolicies {
		policies = append(policies, policy)
	}
	slices.SortFunc(policies, func(a, b Policy) int {
		return strings.Compare(policySortKey(a), policySortKey(b))
	})
	return policies
}
