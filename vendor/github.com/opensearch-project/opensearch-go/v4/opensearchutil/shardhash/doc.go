// SPDX-License-Identifier: Apache-2.0
//
// The OpenSearch Contributors require contributions made to
// this file be licensed under the Apache-2.0 license or a
// compatible open source license.

// Package shardhash implements OpenSearch's shard routing hash algorithm.
//
// OpenSearch uses MurmurHash3 (x86, 32-bit) on UTF-16 LE encoded routing
// strings to deterministically assign documents to shards. This package
// exposes that algorithm so external tools -- custom routers, data pipelines,
// test harnesses -- can compute the same shard placement as the server.
//
// The two primary functions are:
//
//   - [Hash]       -- returns the raw murmur3 hash of a routing string.
//   - [ForRouting] -- returns the shard number for a given routing value,
//     matching OperationRouting.calculateScaledShardId on the server.
package shardhash
