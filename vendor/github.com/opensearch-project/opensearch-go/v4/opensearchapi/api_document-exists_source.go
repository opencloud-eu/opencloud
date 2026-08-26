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

// DocumentExistsSourceReq represents possible options for the _source exists request
type DocumentExistsSourceReq struct {
	Index      string
	DocumentID string

	Header http.Header
	Params DocumentExistsSourceParams
}

// GetRequest returns the *http.Request that gets executed by the client
func (r DocumentExistsSourceReq) GetRequest(method string) (*http.Request, error) {
	path, err := ospath.ExistsSourcePath{
		ID:    r.DocumentID,
		Index: r.Index,
	}.Build()
	if err != nil {
		return nil, err
	}

	return build.Request(method, path, nil, r.Params.get(), r.Header)
}
