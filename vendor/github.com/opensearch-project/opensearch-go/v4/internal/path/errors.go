// SPDX-License-Identifier: Apache-2.0
//
// The OpenSearch Contributors require contributions made to
// this file be licensed under the Apache-2.0 license or a
// compatible open source license.

package path

// ErrRequired is exported so that the root opensearch package can re-export it
// as ErrPathRequired for consumer use with errors.Is.
var ErrRequired = errRequired
