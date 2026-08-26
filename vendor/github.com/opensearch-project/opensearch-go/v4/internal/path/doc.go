// SPDX-License-Identifier: Apache-2.0
//
// The OpenSearch Contributors require contributions made to
// this file be licensed under the Apache-2.0 license or a
// compatible open source license.

// Package path provides generated typed path builder structs for all OpenSearch
// API operations. Each struct corresponds to one x-operation-group from the
// OpenSearch API specification and builds the URL path used in HTTP requests.
//
//go:generate sh -c "cd ../../cmd/osgen && go run . paths -spec ../../opensearch-openapi.yaml -pkg path -o ../../internal/path/builders_gen.go -test-out ../../internal/path/builders_gen_test.go"
package path
