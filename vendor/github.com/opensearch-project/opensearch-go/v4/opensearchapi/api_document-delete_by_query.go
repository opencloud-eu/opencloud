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

// DocumentDeleteByQueryReq represents possible options for the /<index>/_delete_by_query request
type DocumentDeleteByQueryReq struct {
	Indices []string

	Body io.Reader

	Header http.Header
	Params DocumentDeleteByQueryParams
}

// GetRequest returns the *http.Request that gets executed by the client
func (r DocumentDeleteByQueryReq) GetRequest(method string) (*http.Request, error) {
	path, err := ospath.DeleteByQueryPath{Index: r.Indices}.Build()
	if err != nil {
		return nil, err
	}

	return build.Request(method, path, r.Body, r.Params.get(), r.Header)
}

// DocumentDeleteByQueryResp represents the returned struct of the /<index>/_delete_by_query response
type DocumentDeleteByQueryResp struct {
	Took             int  `json:"took"`
	TimedOut         bool `json:"timed_out"`
	Total            int  `json:"total"`
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
	Task                 string                `json:"task,omitempty"` // Needed when wait_for_completion is set to false
	response             *opensearch.Response
}

// Inspect returns the Inspect type containing the raw *opensearch.Response
func (r DocumentDeleteByQueryResp) Inspect() Inspect {
	return Inspect{Response: r.response}
}
