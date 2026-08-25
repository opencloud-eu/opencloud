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

// IndicesResolveReq represents possible options for the get indices request
type IndicesResolveReq struct {
	Indices []string

	Header http.Header
	Params IndicesResolveParams
}

// GetRequest returns the *http.Request that gets executed by the client
func (r IndicesResolveReq) GetRequest(method string) (*http.Request, error) {
	path, err := ospath.IndicesResolveIndexPath{Name: r.Indices}.Build()
	if err != nil {
		return nil, err
	}

	return build.Request(method, path, nil, r.Params.get(), r.Header)
}

// IndicesResolveResp represents the returned struct of the get indices response
type IndicesResolveResp struct {
	Indices []struct {
		Name       string   `json:"name"`
		Attributes []string `json:"attributes"`
		Aliases    []string `json:"aliases"`
	} `json:"indices"`
	Aliases []struct {
		Name    string   `json:"name"`
		Indices []string `json:"indices"`
	} `json:"aliases"`
	DataStreams []struct {
		Name           string   `json:"name"`
		BackingIndices []string `json:"backing_indices"`
		TimestampField string   `json:"timestamp_field"`
	} `json:"data_streams"`
	response *opensearch.Response
}

// Inspect returns the Inspect type containing the raw *opensearch.Response
func (r IndicesResolveResp) Inspect() Inspect {
	return Inspect{Response: r.response}
}
