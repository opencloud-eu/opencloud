// SPDX-License-Identifier: Apache-2.0
//
// The OpenSearch Contributors require contributions made to
// this file be licensed under the Apache-2.0 license or a
// compatible open source license.

package opensearchapi

import (
	"net/http"

	"github.com/opensearch-project/opensearch-go/v4"
	"github.com/opensearch-project/opensearch-go/v4/internal/build"
	ospath "github.com/opensearch-project/opensearch-go/v4/internal/path"
)

// IndicesForcemergeReq represents possible options for the <index>/_forcemerge request
type IndicesForcemergeReq struct {
	Indices []string

	Header http.Header
	Params IndicesForcemergeParams
}

// GetRequest returns the *http.Request that gets executed by the client
func (r IndicesForcemergeReq) GetRequest(method string) (*http.Request, error) {
	path, err := ospath.IndicesForcemergePath{Index: r.Indices}.Build()
	if err != nil {
		return nil, err
	}
	return build.Request(method, path, nil, r.Params.get(), r.Header)
}

// IndicesForcemergeResp represents the returned struct of the flush indices response
type IndicesForcemergeResp struct {
	Shards struct {
		Total      int             `json:"total"`
		Successful int             `json:"successful"`
		Failed     int             `json:"failed"`
		Failures   []FailuresShard `json:"failures"`
	} `json:"_shards"`
	response *opensearch.Response
}

// Inspect returns the Inspect type containing the raw *opensearch.Response
func (r IndicesForcemergeResp) Inspect() Inspect {
	return Inspect{Response: r.response}
}
