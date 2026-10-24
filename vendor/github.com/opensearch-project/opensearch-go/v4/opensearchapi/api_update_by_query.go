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

// UpdateByQuery executes a /_update_by_query request with the optional UpdateByQueryReq
func (c Client) UpdateByQuery(ctx context.Context, req UpdateByQueryReq) (*UpdateByQueryResp, error) {
	var (
		data UpdateByQueryResp
		err  error
	)
	if data.response, err = do(ctx, &c, http.MethodPost, req, &data); err != nil {
		return &data, err
	}

	return &data, nil
}

// UpdateByQueryReq represents possible options for the /_update_by_query request
type UpdateByQueryReq struct {
	Indices []string

	Body io.Reader

	Header http.Header
	Params UpdateByQueryParams
}

// GetRequest returns the *http.Request that gets executed by the client
func (r UpdateByQueryReq) GetRequest(method string) (*http.Request, error) {
	path, err := ospath.UpdateByQueryPath{Index: r.Indices}.Build()
	if err != nil {
		return nil, err
	}

	return build.Request(method, path, r.Body, r.Params.get(), r.Header)
}

// UpdateByQueryResp represents the returned struct of the /_update_by_query response
type UpdateByQueryResp struct {
	Took             int  `json:"took"`
	TimedOut         bool `json:"timed_out"`
	Total            int  `json:"total"`
	Updated          int  `json:"updated"`
	Deleted          int  `json:"deleted"`
	Batches          int  `json:"batches"`
	VersionConflicts int  `json:"version_conflicts"`
	Noops            int  `json:"noops"`
	Retries          struct {
		Bulk   int `json:"bulk"`
		Search int `json:"search"`
	} `json:"retries"`
	ThrottledMillis      int                   `json:"throttled_millis"`
	RequestsPerSecond    float32               `json:"requests_per_second"`
	ThrottledUntilMillis int                   `json:"throttled_until_millis"`
	Failures             []BulkByScrollFailure `json:"failures"`
	Type                 string                `json:"_type,omitempty"` // Deprecated: ES 6.0, removed in OS 2.0
	Task                 string                `json:"task"`
	response             *opensearch.Response
}

// Inspect returns the Inspect type containing the raw *opensearch.Response
func (r UpdateByQueryResp) Inspect() Inspect {
	return Inspect{Response: r.response}
}
