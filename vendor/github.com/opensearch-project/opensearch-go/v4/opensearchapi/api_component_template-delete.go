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

// ComponentTemplateDeleteReq represents possible options for the _component_template delete request
type ComponentTemplateDeleteReq struct {
	ComponentTemplate string

	Header http.Header
	Params ComponentTemplateDeleteParams
}

// GetRequest returns the *http.Request that gets executed by the client
func (r ComponentTemplateDeleteReq) GetRequest(method string) (*http.Request, error) {
	path, err := ospath.ClusterDeleteComponentTemplatePath{Name: r.ComponentTemplate}.Build()
	if err != nil {
		return nil, err
	}

	return build.Request(method, path, nil, r.Params.get(), r.Header)
}

// ComponentTemplateDeleteResp represents the returned struct of the _component_template delete response
type ComponentTemplateDeleteResp struct {
	Acknowledged bool `json:"acknowledged"`
	response     *opensearch.Response
}

// Inspect returns the Inspect type containing the raw *opensearch.Response
func (r ComponentTemplateDeleteResp) Inspect() Inspect {
	return Inspect{Response: r.response}
}
