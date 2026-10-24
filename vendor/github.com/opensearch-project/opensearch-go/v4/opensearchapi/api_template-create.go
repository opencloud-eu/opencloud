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

// TemplateCreateReq represents possible options for the index create request
type TemplateCreateReq struct {
	Template string

	Body io.Reader

	Header http.Header
	Params TemplateCreateParams
}

// GetRequest returns the *http.Request that gets executed by the client
func (r TemplateCreateReq) GetRequest(method string) (*http.Request, error) {
	path, err := ospath.IndicesPutTemplatePath{Name: r.Template}.Build()
	if err != nil {
		return nil, err
	}

	return build.Request(method, path, r.Body, r.Params.get(), r.Header)
}

// TemplateCreateResp represents the returned struct of the index create response
type TemplateCreateResp struct {
	Acknowledged bool `json:"acknowledged"`
	response     *opensearch.Response
}

// Inspect returns the Inspect type containing the raw *opensearch.Response
func (r TemplateCreateResp) Inspect() Inspect {
	return Inspect{Response: r.response}
}
