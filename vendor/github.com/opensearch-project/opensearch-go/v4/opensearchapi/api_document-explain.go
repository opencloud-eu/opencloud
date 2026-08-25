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

// DocumentExplainReq represents possible options for the /<Index>/_explain/<DocumentID> request
type DocumentExplainReq struct {
	Index      string
	DocumentID string

	Body io.Reader

	Header http.Header
	Params DocumentExplainParams
}

// GetRequest returns the *http.Request that gets executed by the client
func (r DocumentExplainReq) GetRequest(method string) (*http.Request, error) {
	path, err := ospath.ExplainPath{
		ID:    r.DocumentID,
		Index: r.Index,
	}.Build()
	if err != nil {
		return nil, err
	}

	return build.Request(method, path, r.Body, r.Params.get(), r.Header)
}

// DocumentExplainResp represents the returned struct of the /<Index>/_explain/<DocumentID> response
type DocumentExplainResp struct {
	Index       string                 `json:"_index"`
	ID          string                 `json:"_id"`
	Type        string                 `json:"_type,omitempty"` // Deprecated: ES 6.0, removed in OS 2.0
	Matched     bool                   `json:"matched"`
	Explanation DocumentExplainDetails `json:"explanation"`
	response    *opensearch.Response
}

// Inspect returns the Inspect type containing the raw *opensearch.Response
func (r DocumentExplainResp) Inspect() Inspect {
	return Inspect{Response: r.response}
}

// DocumentExplainDetails is a sub type of DocumentExplainResp containing information about why a query does what it does
type DocumentExplainDetails struct {
	Value       float64                  `json:"value"`
	Description string                   `json:"description"`
	Details     []DocumentExplainDetails `json:"details"`
}
