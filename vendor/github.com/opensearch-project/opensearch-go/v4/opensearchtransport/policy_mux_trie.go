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
	stdpath "path"
	"strings"
)

// trieLeaf stores the routing decision for a matched route.
type trieLeaf struct {
	policy      Policy
	operationID OperationID
	attrs       routeAttr
	poolName    string
}

// trieNode represents a node in the route matching trie.
// The trie is built once at init time and never mutated afterward.
type trieNode struct {
	// children maps literal path segments to child nodes.
	children map[string]*trieNode
	// wildcard matches any single path segment (replaces {param} patterns).
	wildcard *trieNode
	// wildName is the wildcard parameter name ("index", "id", etc.)
	// used for byte offset tracking in trieMatch.
	wildName string
	// methods maps HTTP method to the terminal routing leaf.
	methods map[string]trieLeaf
}

// trieMatch is the result of a trie route lookup. All fields are populated
// from the trie walk without allocation --just integer bookkeeping.
type trieMatch struct {
	policy      Policy
	operationID OperationID
	attrs       routeAttr
	poolName    string

	// Path segment byte offsets into the original path string.
	// Zero start and end means the segment was not present.
	indexStart, indexEnd int // Position of {index} segment
	docStart, docEnd     int // Position of {id} segment
	isSystem             bool
}

// routeTrie is a zero-allocation HTTP route matcher. It is built once from
// an immutable, compile-time-known route list and never mutated afterward.
type routeTrie struct {
	root trieNode
}

// add registers a route in the trie for each given HTTP method.
// The path must start with "/" and may contain {param} wildcard segments.
// Only called during construction; match() is the hot path.
func (t *routeTrie) add(methods []string, path string, policy Policy, attrs routeAttr, poolName string, opID OperationID) {
	// Walk/create trie nodes for each path segment.
	node := &t.root
	for path != "" {
		if path[0] == '/' {
			path = path[1:]
			continue
		}

		seg := path
		if idx := strings.IndexByte(path, '/'); idx >= 0 {
			seg = path[:idx]
			path = path[idx:]
		} else {
			path = ""
		}

		if seg == "" {
			continue
		}

		// Check for wildcard segment: {param}
		if len(seg) > 2 && seg[0] == '{' && seg[len(seg)-1] == '}' {
			paramName := seg[1 : len(seg)-1]
			if node.wildcard == nil {
				node.wildcard = &trieNode{wildName: paramName}
			}
			node = node.wildcard
		} else {
			if node.children == nil {
				node.children = make(map[string]*trieNode)
			}
			child, ok := node.children[seg]
			if !ok {
				child = &trieNode{}
				node.children[seg] = child
			}
			node = child
		}
	}

	// Register terminal leaf for each method.
	if node.methods == nil {
		node.methods = make(map[string]trieLeaf, len(methods))
	}
	leaf := trieLeaf{policy: policy, operationID: opID, attrs: attrs, poolName: poolName}
	for _, m := range methods {
		node.methods[m] = leaf
	}
}

// match performs a zero-allocation route lookup against the trie.
// It returns the routing decision and byte offsets for captured wildcard
// segments ({index}, {id}). The seg substrings share the path's backing
// array so map lookups do not allocate.
func (t *routeTrie) match(method, path string) (trieMatch, bool) {
	// Normalize the path the same way http.ServeMux does: collapse multi-slashes,
	// resolve dot segments. No-op (zero alloc) for already-clean paths.
	path = stdpath.Clean(path)

	var result trieMatch
	result.isSystem = len(path) >= 2 && path[0] == '/' && path[1] == '_'
	node := &t.root

	pos := 0
	for pos < len(path) {
		// Skip leading '/'
		if path[pos] == '/' {
			pos++
			continue
		}

		// Find segment boundaries.
		segStart := pos
		segEnd := strings.IndexByte(path[pos:], '/')
		if segEnd >= 0 {
			segEnd += pos
		} else {
			segEnd = len(path)
		}
		seg := path[segStart:segEnd]

		if seg == "" {
			pos = segEnd
			continue
		}

		// Try exact match first, then wildcard.
		if child, ok := node.children[seg]; ok {
			node = child
		} else if node.wildcard != nil {
			// Record byte offsets for known wildcard names.
			switch node.wildcard.wildName {
			case "index":
				result.indexStart = segStart
				result.indexEnd = segEnd
			case "id":
				result.docStart = segStart
				result.docEnd = segEnd
			}
			node = node.wildcard
		} else {
			return trieMatch{}, false
		}

		pos = segEnd
	}

	// Root route guard: only exact "/" matches root, not "", "//", "///", etc.
	if node == &t.root && path != "/" {
		return trieMatch{}, false
	}

	if leaf, ok := node.methods[method]; ok {
		result.policy = leaf.policy
		result.operationID = leaf.operationID
		result.attrs = leaf.attrs
		result.poolName = leaf.poolName
		return result, true
	}
	return trieMatch{}, false
}
