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

// ScrollDeleteReq represents possible options for the index create request
type ScrollDeleteReq struct {
	ScrollIDs []string

	Body io.Reader

	Header http.Header
	Params ScrollDeleteParams
}

// GetRequest returns the *http.Request that gets executed by the client
func (r ScrollDeleteReq) GetRequest(method string) (*http.Request, error) {
	path, err := ospath.ClearScrollPath{ScrollID: r.ScrollIDs}.Build()
	if err != nil {
		return nil, err
	}
	return build.Request(method, path, r.Body, r.Params.get(), r.Header)
}

// ScrollDeleteResp represents the returned struct of the index create response
type ScrollDeleteResp struct {
	NumFreed  int  `json:"num_freed"`
	Succeeded bool `json:"succeeded"`
	response  *opensearch.Response
}

// Inspect returns the Inspect type containing the raw *opensearch.Response
func (r ScrollDeleteResp) Inspect() Inspect {
	return Inspect{Response: r.response}
}
