// SPDX-License-Identifier: Apache-2.0
//
// The OpenSearch Contributors require contributions made to
// this file be licensed under the Apache-2.0 license or a
// compatible open source license.

package opensearchapi

import (
	"context"
	"io"
	"net/http"

	"github.com/opensearch-project/opensearch-go/v4"
	"github.com/opensearch-project/opensearch-go/v4/internal/build"
	ospath "github.com/opensearch-project/opensearch-go/v4/internal/path"
)

// Update executes a /_update request with the optional UpdateReq
func (c Client) Update(ctx context.Context, req UpdateReq) (*UpdateResp, error) {
	var (
		data UpdateResp
		err  error
	)
	if data.response, err = do(ctx, &c, http.MethodPost, req, &data); err != nil {
		return &data, err
	}

	if errs := data.PartialFailures(c.errors); len(errs) > 0 {
		return &data, errs[0]
	}
	return &data, nil
}

// UpdateReq represents possible options for the /_update request
type UpdateReq struct {
	Index      string
	DocumentID string

	Body io.Reader

	Header http.Header
	Params UpdateParams
}

// GetRequest returns the *http.Request that gets executed by the client
func (r UpdateReq) GetRequest(method string) (*http.Request, error) {
	path, err := ospath.UpdatePath{
		ID:    r.DocumentID,
		Index: r.Index,
	}.Build()
	if err != nil {
		return nil, err
	}

	return build.Request(method, path, r.Body, r.Params.get(), r.Header)
}

// UpdateResp represents the returned struct of the /_update response
type UpdateResp struct {
	Index         string           `json:"_index"`
	ID            string           `json:"_id"`
	Version       int              `json:"_version"`
	Result        string           `json:"result"`
	ForcedRefresh bool             `json:"forced_refresh"`
	Shards        ResponseShards   `json:"_shards"`
	SeqNo         int              `json:"_seq_no"`
	PrimaryTerm   int              `json:"_primary_term"`
	Type          string           `json:"_type,omitempty"` // Deprecated: ES 6.0, removed in OS 2.0
	Get           *DocumentGetResp `json:"get,omitempty"`
	response      *opensearch.Response
}

// Inspect returns the Inspect type containing the raw *opensearch.Response
func (r UpdateResp) Inspect() Inspect {
	return Inspect{Response: r.response}
}
