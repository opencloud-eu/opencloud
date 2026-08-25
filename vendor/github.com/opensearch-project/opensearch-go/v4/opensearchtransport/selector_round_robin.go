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
	"sync/atomic"
)

// roundRobinSelector implements round-robin connection selection.
type roundRobinSelector struct {
	curr atomic.Int64 // Index of the current connection
}

// NewRoundRobinSelector creates a new round-robin connection selector.
func NewRoundRobinSelector() *roundRobinSelector {
	s := &roundRobinSelector{}
	s.curr.Store(-1)
	return s
}

// Select returns the connection in a round-robin fashion.
func (s *roundRobinSelector) Select(conns []*Connection) (*Connection, error) {
	if len(conns) == 0 {
		return nil, ErrNoConnections
	}

	// Atomic increment with wrap-around
	next := s.curr.Add(1)
	index := int(next % int64(len(conns)))
	return conns[index], nil
}
