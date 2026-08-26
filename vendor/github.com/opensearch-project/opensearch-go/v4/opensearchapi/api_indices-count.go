// SPDX-License-Identifier: Apache-2.0
//
// The OpenSearch Contributors require contributions made to
// this file be licensed under the Apache-2.0 license or a
// compatible open source license.

package opensearchapi

import (
	"io"
	"net/http"

	"github.com/opensearch-project/opensearch-go/v4"
	"github.com/opensearch-project/opensearch-go/v4/internal/build"
	ospath "github.com/opensearch-project/opensearch-go/v4/internal/path"
)

// IndicesCountReq represents possible options for the index shrink request
type IndicesCountReq struct {
	Indices []string

	Body io.Reader

	Header http.Header
	Params IndicesCountParams
}

// GetRequest returns the *http.Request that gets executed by the client
func (r IndicesCountReq) GetRequest(method string) (*http.Request, error) {
	path, err := ospath.CountPath{Index: r.Indices}.Build()
	if err != nil {
		return nil, err
	}
	return build.Request(method, path, r.Body, r.Params.get(), r.Header)
}

// IndicesCountResp represents the returned struct of the index shrink response
type IndicesCountResp struct {
	Shards   ResponseShards `json:"_shards"`
	Count    int            `json:"count"`
	response *opensearch.Response
}

// Inspect returns the Inspect type containing the raw *opensearch.Response
func (r IndicesCountResp) Inspect() Inspect {
	return Inspect{Response: r.response}
}
