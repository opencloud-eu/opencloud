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

// IndexTemplateSimulateIndexReq represents possible options for the index create request
type IndexTemplateSimulateIndexReq struct {
	Index string

	Body io.Reader

	Header http.Header
	Params IndexTemplateSimulateIndexParams
}

// GetRequest returns the *http.Request that gets executed by the client
func (r IndexTemplateSimulateIndexReq) GetRequest(method string) (*http.Request, error) {
	path, err := ospath.IndicesSimulateIndexTemplatePath{Name: r.Index}.Build()
	if err != nil {
		return nil, err
	}

	return build.Request(method, path, r.Body, r.Params.get(), r.Header)
}

// IndexTemplateSimulateIndexResp represents the returned struct of the index create response
type IndexTemplateSimulateIndexResp struct {
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
func (r IndexTemplateSimulateIndexResp) Inspect() Inspect {
	return Inspect{Response: r.response}
}
