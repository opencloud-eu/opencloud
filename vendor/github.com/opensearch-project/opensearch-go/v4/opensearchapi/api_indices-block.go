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

// IndicesBlockReq represents possible options for the index create request
type IndicesBlockReq struct {
	Indices []string
	Block   string

	Header http.Header
	Params IndicesBlockParams
}

// GetRequest returns the *http.Request that gets executed by the client
func (r IndicesBlockReq) GetRequest(method string) (*http.Request, error) {
	path, err := ospath.IndicesAddBlockPath{
		Block: r.Block,
		Index: r.Indices,
	}.Build()
	if err != nil {
		return nil, err
	}

	return build.Request(method, path, nil, r.Params.get(), r.Header)
}

// IndicesBlockResp represents the returned struct of the index create response
type IndicesBlockResp struct {
	Acknowledged       bool `json:"acknowledged"`
	ShardsAcknowledged bool `json:"shards_acknowledged"`
	Indices            []struct {
		Name         string `json:"name"`
		Blocked      bool   `json:"blocked"`
		FailedShards []struct {
			ID       int             `json:"id"`
			Failures []FailuresShard `json:"failures"`
		} `json:"failed_shards"`
	} `json:"indices"`
	response *opensearch.Response
}

// Inspect returns the Inspect type containing the raw *opensearch.Response
func (r IndicesBlockResp) Inspect() Inspect {
	return Inspect{Response: r.response}
}
