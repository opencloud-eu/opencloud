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

// IndicesFieldCapsReq represents possible options for the index shrink request
type IndicesFieldCapsReq struct {
	Indices []string

	Body io.Reader

	Header http.Header
	Params IndicesFieldCapsParams
}

// GetRequest returns the *http.Request that gets executed by the client
func (r IndicesFieldCapsReq) GetRequest(method string) (*http.Request, error) {
	path, err := ospath.FieldCapsPath{Index: r.Indices}.Build()
	if err != nil {
		return nil, err
	}
	return build.Request(method, path, nil, r.Params.get(), r.Header)
}

// IndicesFieldCapsResp represents the returned struct of the index shrink response
type IndicesFieldCapsResp struct {
	Indices []string `json:"indices"`
	Fields  map[string]map[string]struct {
		Type         string   `json:"type"`
		Searchable   bool     `json:"searchable"`
		Aggregatable bool     `json:"aggregatable"`
		Indices      []string `json:"indices"`
	} `json:"fields"`
	response *opensearch.Response
}

// Inspect returns the Inspect type containing the raw *opensearch.Response
func (r IndicesFieldCapsResp) Inspect() Inspect {
	return Inspect{Response: r.response}
}
