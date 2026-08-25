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

// Index executes a /_doc request with the given IndexReq
func (c Client) Index(ctx context.Context, req IndexReq) (*IndexResp, error) {
	var (
		data IndexResp
		err  error
	)
	method := http.MethodPost
	if req.DocumentID != "" {
		method = http.MethodPut
	}
	if data.response, err = do(ctx, &c, method, req, &data); err != nil {
		return &data, err
	}

	if errs := data.PartialFailures(c.errors); len(errs) > 0 {
		return &data, errs[0]
	}
	return &data, nil
}

// IndexReq represents possible options for the /_doc request
type IndexReq struct {
	Index      string
	DocumentID string
	Body       io.Reader
	Header     http.Header
	Params     IndexParams
}

// GetRequest returns the *http.Request that gets executed by the client
func (r IndexReq) GetRequest(method string) (*http.Request, error) {
	path, err := ospath.IndexPath{
		ID:    r.DocumentID,
		Index: r.Index,
	}.Build()
	if err != nil {
		return nil, err
	}

	return build.Request(method, path, r.Body, r.Params.get(), r.Header)
}

// IndexResp represents the returned struct of the /_doc response
type IndexResp struct {
	Index         string         `json:"_index"`
	ID            string         `json:"_id"`
	Version       int            `json:"_version"`
	Result        string         `json:"result"`
	ForcedRefresh bool           `json:"forced_refresh"`
	Shards        ResponseShards `json:"_shards"`
	SeqNo         int            `json:"_seq_no"`
	PrimaryTerm   int            `json:"_primary_term"`
	Type          string         `json:"_type,omitempty"` // Deprecated: ES 6.0, removed in OS 2.0
	response      *opensearch.Response
}

// Inspect returns the Inspect type containing the raw *opensearch.Response
func (r IndexResp) Inspect() Inspect {
	return Inspect{Response: r.response}
}
