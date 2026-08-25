// SPDX-License-Identifier: Apache-2.0
//
// The OpenSearch Contributors require contributions made to
// this file be licensed under the Apache-2.0 license or a
// compatible open source license.

package opensearchtransport

import "github.com/opensearch-project/opensearch-go/v4/opensearchutil/shardhash"

// opensearchShardHash computes the same hash as OpenSearch's
// Murmur3HashFunction.hash(String). Delegates to [shardhash.Hash].
func opensearchShardHash(routing string) int32 {
	return shardhash.Hash(routing)
}

// shardForRouting computes the shard number for a routing value, matching
// OpenSearch's OperationRouting.calculateScaledShardId. Delegates to
// [shardhash.ForRouting].
func shardForRouting(routing string, routingNumShards, numPrimaryShards int) int {
	return shardhash.ForRouting(routing, routingNumShards, numPrimaryShards)
}
