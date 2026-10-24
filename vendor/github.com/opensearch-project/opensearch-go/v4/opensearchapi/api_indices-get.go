// SPDX-License-Identifier: Apache-2.0
//
// The OpenSearch Contributors require contributions made to
// this file be licensed under the Apache-2.0 license or a
// compatible open source license.

package opensearchapi

import (
	"encoding/json"
	"net/http"

	"github.com/opensearch-project/opensearch-go/v4"
	"github.com/opensearch-project/opensearch-go/v4/internal/build"
	ospath "github.com/opensearch-project/opensearch-go/v4/internal/path"
)

// IndicesGetReq represents possible options for the get indices request
type IndicesGetReq struct {
	Indices []string

	Header http.Header
	Params IndicesGetParams
}

// GetRequest returns the *http.Request that gets executed by the client
func (r IndicesGetReq) GetRequest(method string) (*http.Request, error) {
	path, err := ospath.IndicesGetPath{Index: r.Indices}.Build()
	if err != nil {
		return nil, err
	}

	return build.Request(method, path, nil, r.Params.get(), r.Header)
}

// IndicesGetResp represents the returned struct of the get indices response
// Since the JSON has index names as top-level keys, we use a map-based approach
type IndicesGetResp struct {
	*IndicesGetRespData
	response *opensearch.Response
}

// IndicesGetRespData holds the actual response data with dynamic index names as keys
type IndicesGetRespData map[string]IndicesGetRespIndex

// IndicesGetRespIndex represents the structure of each index in the response
type IndicesGetRespIndex struct {
	DataStream *string             `json:"data_stream,omitempty"` // Available in OpenSearch 1.0.0+ (data streams introduced)
	Aliases    map[string]struct{} `json:"aliases"`               // Available since OpenSearch 1.0.0
	Mappings   json.RawMessage     `json:"mappings"`              // Available since OpenSearch 1.0.0
	Settings   json.RawMessage     `json:"settings"`              // Available since OpenSearch 1.0.0
}

// Inspect returns the Inspect type containing the raw *opensearch.Response
func (r IndicesGetResp) Inspect() Inspect {
	return Inspect{Response: r.response}
}
