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
	"net/http"
	"sync/atomic"
)

// Compile-time interface compliance checks
var (
	_ Policy             = (*IfEnabledPolicy)(nil)
	_ policyConfigurable = (*IfEnabledPolicy)(nil)
	_ policyTyped        = (*IfEnabledPolicy)(nil)
	_ policyOverrider    = (*IfEnabledPolicy)(nil)
)

// ConditionFunc defines a function that evaluates a condition based on request context.
type ConditionFunc func(context.Context, *http.Request) bool

// IfEnabledPolicy provides conditional connection pool selection based on policy availability.
// It evaluates a condition at runtime and uses either the true policy (if condition is true) or false policy (if condition is false).
type IfEnabledPolicy struct {
	condition   ConditionFunc // Condition to evaluate at runtime
	truePolicy  Policy        // Policy to use when condition is true
	falsePolicy Policy        // Policy to use when condition is false
	policyState atomic.Int32  // Bitfield: psEnabled|psDisabled|psEnvEnabled|psEnvDisabled
}

func (p *IfEnabledPolicy) policyTypeName() string      { return policyTypeNameIfEnabled }
func (p *IfEnabledPolicy) setEnvOverride(enabled bool) { psSetEnvOverride(&p.policyState, enabled) }

// NewIfEnabledPolicy creates a new conditional policy.
//
// Example usage:
//
//	policy := NewIfEnabledPolicy(
//	    func(ctx context.Context, req *http.Request) bool {
//	        // Custom condition logic here
//	        return someCondition
//	    },
//	    NewCoordinatorPolicy(),
//	    NewRoundRobinPolicy(),
//	)
func NewIfEnabledPolicy(condition ConditionFunc, truePolicy, falsePolicy Policy) Policy {
	p := &IfEnabledPolicy{
		condition:   condition,
		truePolicy:  truePolicy,
		falsePolicy: falsePolicy,
	}
	isConfigured := condition != nil && truePolicy != nil && falsePolicy != nil
	psSetEnabled(&p.policyState, isConfigured)
	return p
}

// DiscoveryUpdate calls DiscoveryUpdate on sub-policies.
func (p *IfEnabledPolicy) DiscoveryUpdate(added, removed, unchanged []*Connection) error {
	var firstError error

	// Update both sub-policies
	if err := p.truePolicy.DiscoveryUpdate(added, removed, unchanged); err != nil && firstError == nil {
		firstError = err
	}
	if err := p.falsePolicy.DiscoveryUpdate(added, removed, unchanged); err != nil && firstError == nil {
		firstError = err
	}

	return firstError
}

// configurePolicySettings configures pool settings for both sub-policies.
func (p *IfEnabledPolicy) configurePolicySettings(config policyConfig) error {
	var firstError error

	// Configure both sub-policies
	if configurableTrue, ok := p.truePolicy.(policyConfigurable); ok {
		if err := configurableTrue.configurePolicySettings(config); err != nil && firstError == nil {
			firstError = err
		}
	}
	if configurableFalse, ok := p.falsePolicy.(policyConfigurable); ok {
		if err := configurableFalse.configurePolicySettings(config); err != nil && firstError == nil {
			firstError = err
		}
	}

	return firstError
}

// IsEnabled returns true if the policy is properly configured with condition and both sub-policies.
func (p *IfEnabledPolicy) IsEnabled() bool {
	return psIsEnabled(p.policyState.Load())
}

// Eval evaluates the condition and delegates to the appropriate policy.
func (p *IfEnabledPolicy) Eval(ctx context.Context, req *http.Request) (NextHop, error) {
	if p.policyState.Load()&psEnvDisabled != 0 {
		return NextHop{}, nil
	}

	// Evaluate condition and choose policy
	if p.condition(ctx, req) {
		return p.truePolicy.Eval(ctx, req)
	}
	return p.falsePolicy.Eval(ctx, req)
}

// CheckDead delegates to sub-policies.
func (p *IfEnabledPolicy) CheckDead(ctx context.Context, healthCheck HealthCheckFunc) error {
	var firstError error

	if err := p.truePolicy.CheckDead(ctx, healthCheck); err != nil && firstError == nil {
		firstError = err
	}
	if err := p.falsePolicy.CheckDead(ctx, healthCheck); err != nil && firstError == nil {
		firstError = err
	}

	return firstError
}

// RotateStandby delegates to the enabled sub-policy only.
// Unlike CheckDead (which must sync both branches), rotation only applies
// to the branch that is actively serving traffic.
func (p *IfEnabledPolicy) RotateStandby(ctx context.Context, count int) (int, error) {
	if p.truePolicy.IsEnabled() {
		return p.truePolicy.RotateStandby(ctx, count)
	}
	return p.falsePolicy.RotateStandby(ctx, count)
}

// childPolicies returns the sub-policies for tree walking.
func (p *IfEnabledPolicy) childPolicies() []Policy {
	return []Policy{p.truePolicy, p.falsePolicy}
}
