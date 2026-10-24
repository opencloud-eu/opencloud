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

// DocumentGetReq represents possible options for the /<Index>/_doc/<DocumentID> get request
type DocumentGetReq struct {
	Index      string
	DocumentID string

	Header http.Header
	Params DocumentGetParams
}

// GetRequest returns the *http.Request that gets executed by the client
func (r DocumentGetReq) GetRequest(method string) (*http.Request, error) {
	path, err := ospath.GetPath{
		ID:    r.DocumentID,
		Index: r.Index,
	}.Build()
	if err != nil {
		return nil, err
	}

	return build.Request(method, path, nil, r.Params.get(), r.Header)
}

// DocumentGetResp represents the returned struct of the /<Index>/_doc/<DocumentID> get response
type DocumentGetResp struct {
	Index       string          `json:"_index"`
	ID          string          `json:"_id"`
	Version     int             `json:"_version"`
	SeqNo       int             `json:"_seq_no"`
	PrimaryTerm int             `json:"_primary_term"`
	Found       bool            `json:"found"`
	Type        string          `json:"_type,omitempty"` // Deprecated: ES 6.0, removed in OS 2.0
	Source      json.RawMessage `json:"_source"`
	Fields      json.RawMessage `json:"fields"`
	response    *opensearch.Response
}

// Inspect returns the Inspect type containing the raw *opensearch.Response
func (r DocumentGetResp) Inspect() Inspect {
	return Inspect{Response: r.response}
}
