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

// OperationClassifier maps HTTP method+path pairs to [OperationID] values.
// It reuses the same route trie used by [MuxPolicy], so classification is
// zero-allocation for well-formed paths.
//
// Build once with [NewOperationClassifier] and reuse across requests.
type OperationClassifier struct {
	trie routeTrie
}

// NewOperationClassifier builds a classifier from the default route table.
// The returned classifier is safe for concurrent use.
func NewOperationClassifier() *OperationClassifier {
	c := &OperationClassifier{}

	// Use a nil-safe null policy for all routes — we only need the
	// operationID from the leaf, not a working policy.
	p := NewNullPolicy()

	routes := buildClassifierRoutes(p)
	for _, r := range routes {
		rm := r.(*RouteMux)
		method, path, err := splitMuxPattern(rm.Pattern)
		if err != nil {
			continue
		}
		c.trie.add([]string{method}, path, rm.policy, rm.attrs, rm.poolName, rm.operationID)
	}

	return c
}

// Classify returns the [OperationID] for the given HTTP method and path.
// Returns [OpOther] for unrecognized method+path combinations.
func (c *OperationClassifier) Classify(method, path string) OperationID {
	m, ok := c.trie.match(method, path)
	if !ok {
		return OpOther
	}
	return m.operationID
}

// buildClassifierRoutes constructs the route table with OperationID tags
// but using a single shared null policy. This avoids creating real role-based
// policies (which need live connections) for pure classification use.
func buildClassifierRoutes(p Policy) []Route {
	r := roleRoutes{
		ingestWrite:    p,
		ingestMgmt:     p,
		searchRead:     p,
		getRead:        p,
		dataWrite:      p,
		dataRefresh:    p,
		dataFlush:      p,
		dataForceMerge: p,
		dataMgmt:       p,
		searchMgmt:     p,
		warmMgmt:       p,
	}
	return buildRoleRoutes(r)
}
