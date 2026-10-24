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

// ScriptDeleteReq represents possible options for the delete script request
type ScriptDeleteReq struct {
	ScriptID string

	Header http.Header
	Params ScriptDeleteParams
}

// GetRequest returns the *http.Request that gets executed by the client
func (r ScriptDeleteReq) GetRequest(method string) (*http.Request, error) {
	path, err := ospath.DeleteScriptPath{ID: r.ScriptID}.Build()
	if err != nil {
		return nil, err
	}

	return build.Request(method, path, nil, r.Params.get(), r.Header)
}

// ScriptDeleteResp represents the returned struct of the delete script response
type ScriptDeleteResp struct {
	Acknowledged bool `json:"acknowledged"`
	response     *opensearch.Response
}

// Inspect returns the Inspect type containing the raw *opensearch.Response
func (r ScriptDeleteResp) Inspect() Inspect {
	return Inspect{Response: r.response}
}
