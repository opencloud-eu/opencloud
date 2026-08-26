// SPDX-License-Identifier: Apache-2.0
//
// The OpenSearch Contributors require contributions made to
// this file be licensed under the Apache-2.0 license or a
// compatible open source license.

package opensearchapi

import (
	"net/http"

	"github.com/opensearch-project/opensearch-go/v4/internal/build"
	ospath "github.com/opensearch-project/opensearch-go/v4/internal/path"
)

// IndicesExistsReq represents possible options for the index exists request
type IndicesExistsReq struct {
	Indices []string
	Header  http.Header
	Params  IndicesExistsParams
}

// GetRequest returns the *http.Request that gets executed by the client
func (r IndicesExistsReq) GetRequest(method string) (*http.Request, error) {
	path, err := ospath.IndicesExistsPath{Index: r.Indices}.Build()
	if err != nil {
		return nil, err
	}

	return build.Request(method, path, nil, r.Params.get(), r.Header)
}
