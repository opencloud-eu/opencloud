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

// IndicesValidateQueryReq represents possible options for the index shrink request
type IndicesValidateQueryReq struct {
	Indices []string

	Body io.Reader

	Header http.Header
	Params IndicesValidateQueryParams
}

// GetRequest returns the *http.Request that gets executed by the client
func (r IndicesValidateQueryReq) GetRequest(method string) (*http.Request, error) {
	path, err := ospath.IndicesValidateQueryPath{Index: r.Indices}.Build()
	if err != nil {
		return nil, err
	}
	return build.Request(method, path, r.Body, r.Params.get(), r.Header)
}

// IndicesValidateQueryResp represents the returned struct of the index shrink response
type IndicesValidateQueryResp struct {
	Shards struct {
		Total      int             `json:"total"`
		Successful int             `json:"successful"`
		Failed     int             `json:"failed"`
		Failures   []FailuresShard `json:"failures"`
	} `json:"_shards"`
	Valid        bool    `json:"valid"`
	Error        *string `json:"error"`
	Explanations []struct {
		Index       string  `json:"index"`
		Shard       int     `json:"shard"`
		Valid       bool    `json:"valid"`
		Explanation *string `json:"explanation"`
		Error       *string `json:"error"`
	} `json:"explanations"`
	response *opensearch.Response
}

// Inspect returns the Inspect type containing the raw *opensearch.Response
func (r IndicesValidateQueryResp) Inspect() Inspect {
	return Inspect{Response: r.response}
}
