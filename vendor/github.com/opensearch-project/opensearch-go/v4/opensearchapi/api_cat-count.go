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

// CatCountReq represent possible options for the /_cat/count request
type CatCountReq struct {
	Indices []string
	Header  http.Header
	Params  CatCountParams
}

// GetRequest returns the *http.Request that gets executed by the client
func (r CatCountReq) GetRequest(method string) (*http.Request, error) {
	path, err := ospath.CatCountPath{Index: r.Indices}.Build()
	if err != nil {
		return nil, err
	}
	return build.Request(method, path, nil, r.Params.get(), r.Header)
}

// CatCountsResp represents the returned struct of the /_cat/count response
type CatCountsResp struct {
	Counts   []CatCountResp
	response *opensearch.Response
}

// CatCountResp represents one index of the CatCountResp
type CatCountResp struct {
	Epoch     int    `json:"epoch,string"`
	Timestamp string `json:"timestamp"`
	Count     int    `json:"count,string"`
}

// Inspect returns the Inspect type containing the raw *opensearch.Response
func (r CatCountsResp) Inspect() Inspect {
	return Inspect{
		Response: r.response,
	}
}
