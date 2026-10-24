// SPDX-License-Identifier: Apache-2.0
//
// The OpenSearch Contributors require contributions made to
// this file be licensed under the Apache-2.0 license or a
// compatible open source license.

package opensearchtransport

// routerCacheProvider is implemented by policies that own an
// [indexSlotCache]. Used by [PolicyChain] to walk the policy tree
// and find the cache for a given index.
type routerCacheProvider interface {
	routerCache() *indexSlotCache
}

// findRouterCache walks a policy tree depth-first and returns the
// first [indexSlotCache] found. Returns nil if no scoring-capable
// policy exists in the tree.
func findRouterCache(p Policy) *indexSlotCache {
	if acp, ok := p.(routerCacheProvider); ok {
		return acp.routerCache()
	}
	if walker, ok := p.(policyTreeWalker); ok {
		for _, child := range walker.childPolicies() {
			if cache := findRouterCache(child); cache != nil {
				return cache
			}
		}
	}
	return nil
}
