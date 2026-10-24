// SPDX-License-Identifier: Apache-2.0
//
// The OpenSearch Contributors require contributions made to
// this file be licensed under the Apache-2.0 license or a
// compatible open source license.

package opensearchapi

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/opensearch-project/opensearch-go/v4"
	"github.com/opensearch-project/opensearch-go/v4/internal/build"
	ospath "github.com/opensearch-project/opensearch-go/v4/internal/path"
)

// IndexTemplateSimulateReq represents possible options for the index create request
type IndexTemplateSimulateReq struct {
	IndexTemplate string

	Body io.Reader

	Header http.Header
	Params IndexTemplateSimulateParams
}

// GetRequest returns the *http.Request that gets executed by the client
func (r IndexTemplateSimulateReq) GetRequest(method string) (*http.Request, error) {
	path, err := ospath.IndicesSimulateTemplatePath{Name: r.IndexTemplate}.Build()
	if err != nil {
		return nil, err
	}

	return build.Request(method, path, r.Body, r.Params.get(), r.Header)
}

// IndexTemplateSimulateResp represents the returned struct of the index create response
type IndexTemplateSimulateResp struct {
	Template struct {
		Mappings json.RawMessage `json:"mappings"`
		Settings json.RawMessage `json:"settings"`
		Aliases  json.RawMessage `json:"aliases"`
	} `json:"template"`
	Overlapping []struct {
		Name          string   `json:"name"`
		IndexPatterns []string `json:"index_patterns"`
	} `json:"overlapping"`
	response *opensearch.Response
}

// Inspect returns the Inspect type containing the raw *opensearch.Response
func (r IndexTemplateSimulateResp) Inspect() Inspect {
	return Inspect{Response: r.response}
}
