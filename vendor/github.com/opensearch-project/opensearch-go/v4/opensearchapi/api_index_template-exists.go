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

// IndexTemplateExistsReq represents possible options for the index create request
type IndexTemplateExistsReq struct {
	IndexTemplate string

	Header http.Header
	Params IndexTemplateExistsParams
}

// GetRequest returns the *http.Request that gets executed by the client
func (r IndexTemplateExistsReq) GetRequest(method string) (*http.Request, error) {
	path, err := ospath.IndicesExistsIndexTemplatePath{Name: r.IndexTemplate}.Build()
	if err != nil {
		return nil, err
	}

	return build.Request(method, path, nil, r.Params.get(), r.Header)
}
