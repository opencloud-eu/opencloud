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

// IndicesFlushReq represents possible options for the flush indices request
type IndicesFlushReq struct {
	Indices []string

	Header http.Header
	Params IndicesFlushParams
}

// GetRequest returns the *http.Request that gets executed by the client
func (r IndicesFlushReq) GetRequest(method string) (*http.Request, error) {
	path, err := ospath.IndicesFlushPath{Index: r.Indices}.Build()
	if err != nil {
		return nil, err
	}
	return build.Request(method, path, nil, r.Params.get(), r.Header)
}

// IndicesFlushResp represents the returned struct of the flush indices response
type IndicesFlushResp struct {
	Shards struct {
		Total      int             `json:"total"`
		Successful int             `json:"successful"`
		Failed     int             `json:"failed"`
		Failures   []FailuresShard `json:"failures"`
	} `json:"_shards"`
	response *opensearch.Response
}

// Inspect returns the Inspect type containing the raw *opensearch.Response
func (r IndicesFlushResp) Inspect() Inspect {
	return Inspect{Response: r.response}
}
