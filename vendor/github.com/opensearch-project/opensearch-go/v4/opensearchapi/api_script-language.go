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
)

// ScriptLanguageReq represents possible options for the delete script request
type ScriptLanguageReq struct {
	Header http.Header
	Params ScriptLanguageParams
}

// GetRequest returns the *http.Request that gets executed by the client
func (r ScriptLanguageReq) GetRequest(method string) (*http.Request, error) {
	return build.Request(
		method,
		"/_script_language",
		nil,
		r.Params.get(),
		r.Header,
	)
}

// ScriptLanguageResp represents the returned struct of the delete script response
type ScriptLanguageResp struct {
	TypesAllowed     []string `json:"types_allowed"`
	LanguageContexts []struct {
		Language string   `json:"language"`
		Contexts []string `json:"contexts"`
	} `json:"language_contexts"`
	response *opensearch.Response
}

// Inspect returns the Inspect type containing the raw *opensearch.Response
func (r ScriptLanguageResp) Inspect() Inspect {
	return Inspect{Response: r.response}
}
