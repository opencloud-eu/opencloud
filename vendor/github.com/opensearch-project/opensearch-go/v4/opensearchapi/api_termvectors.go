// SPDX-License-Identifier: Apache-2.0
//
// The OpenSearch Contributors require contributions made to
// this file be licensed under the Apache-2.0 license or a
// compatible open source license.

package opensearchapi

import (
	"context"
	"encoding/json"
	"io"
	"net/http"

	"github.com/opensearch-project/opensearch-go/v4"
	"github.com/opensearch-project/opensearch-go/v4/internal/build"
	ospath "github.com/opensearch-project/opensearch-go/v4/internal/path"
)

// Termvectors executes a /_termvectors request with the required TermvectorsReq
func (c Client) Termvectors(ctx context.Context, req TermvectorsReq) (*TermvectorsResp, error) {
	var (
		data TermvectorsResp
		err  error
	)
	if data.response, err = do(ctx, &c, http.MethodPost, req, &data); err != nil {
		return &data, err
	}

	return &data, nil
}

// TermvectorsReq represents possible options for the /_termvectors request
type TermvectorsReq struct {
	Index      string
	DocumentID string

	Body io.Reader

	Header http.Header
	Params TermvectorsParams
}

// GetRequest returns the *http.Request that gets executed by the client
func (r TermvectorsReq) GetRequest(method string) (*http.Request, error) {
	path, err := ospath.TermvectorsPath{
		ID:    r.DocumentID,
		Index: r.Index,
	}.Build()
	if err != nil {
		return nil, err
	}
	return build.Request(method, path, r.Body, r.Params.get(), r.Header)
}

// TermvectorsResp represents the returned struct of the /_termvectors response
type TermvectorsResp struct {
	Index       string          `json:"_index"`
	ID          string          `json:"_id"`
	Version     int             `json:"_version"`
	Found       bool            `json:"found"`
	Took        int             `json:"took"`
	Type        string          `json:"_type,omitempty"` // Deprecated: ES 6.0, removed in OS 2.0
	TermVectors json.RawMessage `json:"term_vectors"`
	response    *opensearch.Response
}

// Inspect returns the Inspect type containing the raw *opensearch.Response
func (r TermvectorsResp) Inspect() Inspect {
	return Inspect{Response: r.response}
}
