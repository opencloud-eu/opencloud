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

// Bulk executes a /_bulk request with the needed BulkReq
func (c Client) Bulk(ctx context.Context, req BulkReq) (*BulkResp, error) {
	var (
		data BulkResp
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

// BulkReq represents possible options for the /_bulk request
type BulkReq struct {
	Index  string
	Body   io.Reader
	Header http.Header
	Params BulkParams
}

// GetRequest returns the *http.Request that gets executed by the client
func (r BulkReq) GetRequest(method string) (*http.Request, error) {
	path, err := ospath.BulkPath{Index: r.Index}.Build()
	if err != nil {
		return nil, err
	}
	return build.Request(method, path, r.Body, r.Params.get(), r.Header)
}

// BulkResp represents the returned struct of the /_bulk response
type BulkResp struct {
	Took     int                       `json:"took"`
	Errors   bool                      `json:"errors"`
	Items    []map[string]BulkRespItem `json:"items"`
	response *opensearch.Response
}

// BulkRespItem represents an item of the BulkResp
type BulkRespItem struct {
	Index   string `json:"_index"`
	ID      string `json:"_id"`
	Version int    `json:"_version"`
	Type    string `json:"_type,omitempty"` // Deprecated: ES 6.0, removed in OS 2.0
	Result  string `json:"result"`
	Shards  struct {
		Total      int `json:"total"`
		Successful int `json:"successful"`
		Failed     int `json:"failed"`
	} `json:"_shards"`
	SeqNo       int `json:"_seq_no"`
	PrimaryTerm int `json:"_primary_term"`
	Status      int `json:"status"`
	Error       *struct {
		Type   string `json:"type"`
		Reason string `json:"reason"`
		Cause  struct {
			Type        string    `json:"type"`
			Reason      string    `json:"reason"`
			ScriptStack *[]string `json:"script_stack,omitempty"`
			Script      *string   `json:"script,omitempty"`
			Lang        *string   `json:"lang,omitempty"`
			Position    *struct {
				Offset int `json:"offset"`
				Start  int `json:"start"`
				End    int `json:"end"`
			} `json:"position,omitempty"`
			Cause *struct {
				Type   string  `json:"type"`
				Reason *string `json:"reason"`
			} `json:"caused_by"`
		} `json:"caused_by,omitzero"`
	} `json:"error,omitempty"`
}

// Inspect returns the Inspect type containing the raw *opensearch.Response
func (r BulkResp) Inspect() Inspect {
	return Inspect{Response: r.response}
}
