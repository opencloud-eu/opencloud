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

// ComponentTemplateGetReq represents possible options for the _component_template get request
type ComponentTemplateGetReq struct {
	ComponentTemplate string

	Header http.Header
	Params ComponentTemplateGetParams
}

// GetRequest returns the *http.Request that gets executed by the client
func (r ComponentTemplateGetReq) GetRequest(method string) (*http.Request, error) {
	path, err := ospath.ClusterGetComponentTemplatePath{Name: r.ComponentTemplate}.Build()
	if err != nil {
		return nil, err
	}
	return build.Request(method, path, nil, r.Params.get(), r.Header)
}

// ComponentTemplateGetResp represents the returned struct of the index create response
type ComponentTemplateGetResp struct {
	ComponentTemplates []ComponentTemplateGetDetails `json:"component_templates"`
	response           *opensearch.Response
}

// Inspect returns the Inspect type containing the raw *opensearch.Response
func (r ComponentTemplateGetResp) Inspect() Inspect {
	return Inspect{Response: r.response}
}

// ComponentTemplateGetDetails is a sub type of ComponentTemplateGetResp containing information about component template
type ComponentTemplateGetDetails struct {
	Name              string `json:"name"`
	ComponentTemplate struct {
		Template struct {
			Mappings json.RawMessage `json:"mappings"`
			Settings json.RawMessage `json:"settings"`
			Aliases  json.RawMessage `json:"aliases"`
		} `json:"template"`
	} `json:"component_template"`
}
