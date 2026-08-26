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

// DocumentDeleteReq represents possible options for the /<index>/_doc/<DocID> delete request
type DocumentDeleteReq struct {
	Index      string
	DocumentID string

	Header http.Header
	Params DocumentDeleteParams
}

// GetRequest returns the *http.Request that gets executed by the client
func (r DocumentDeleteReq) GetRequest(method string) (*http.Request, error) {
	path, err := ospath.DeletePath{
		ID:    r.DocumentID,
		Index: r.Index,
	}.Build()
	if err != nil {
		return nil, err
	}

	return build.Request(method, path, nil, r.Params.get(), r.Header)
}

// DocumentDeleteResp represents the returned struct of the /<index>/_doc/<DocID> response
type DocumentDeleteResp struct {
	Index         string         `json:"_index"`
	ID            string         `json:"_id"`
	Version       int            `json:"_version"`
	Result        string         `json:"result"`
	ForcedRefresh bool           `json:"forced_refresh"`
	Type          string         `json:"_type,omitempty"` // Deprecated: ES 6.0, removed in OS 2.0
	Shards        ResponseShards `json:"_shards"`
	SeqNo         int            `json:"_seq_no"`
	PrimaryTerm   int            `json:"_primary_term"`
	response      *opensearch.Response
}

// Inspect returns the Inspect type containing the raw *opensearch.Response
func (r DocumentDeleteResp) Inspect() Inspect {
	return Inspect{Response: r.response}
}
