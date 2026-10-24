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

// ComponentTemplateCreateReq represents possible options for the _component_template create request
type ComponentTemplateCreateReq struct {
	ComponentTemplate string

	Body io.Reader

	Header http.Header
	Params ComponentTemplateCreateParams
}

// GetRequest returns the *http.Request that gets executed by the client
func (r ComponentTemplateCreateReq) GetRequest(method string) (*http.Request, error) {
	path, err := ospath.ClusterPutComponentTemplatePath{Name: r.ComponentTemplate}.Build()
	if err != nil {
		return nil, err
	}

	return build.Request(method, path, r.Body, r.Params.get(), r.Header)
}

// ComponentTemplateCreateResp represents the returned struct of the index create response
type ComponentTemplateCreateResp struct {
	Acknowledged bool `json:"acknowledged"`
	response     *opensearch.Response
}

// Inspect returns the Inspect type containing the raw *opensearch.Response
func (r ComponentTemplateCreateResp) Inspect() Inspect {
	return Inspect{Response: r.response}
}
