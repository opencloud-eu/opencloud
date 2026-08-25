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

// PointInTimeCreateReq represents possible options for the index create request
type PointInTimeCreateReq struct {
	Indices []string

	Header http.Header
	Params PointInTimeCreateParams
}

// GetRequest returns the *http.Request that gets executed by the client
func (r PointInTimeCreateReq) GetRequest(method string) (*http.Request, error) {
	path, err := ospath.CreatePITPath{Index: r.Indices}.Build()
	if err != nil {
		return nil, err
	}
	return build.Request(method, path, nil, r.Params.get(), r.Header)
}

// PointInTimeCreateResp represents the returned struct of the index create response
type PointInTimeCreateResp struct {
	PitID  string `json:"pit_id"`
	Shards struct {
		Total      int                     `json:"total"`
		Successful int                     `json:"successful"`
		Skipped    int                     `json:"skipped"`
		Failed     int                     `json:"failed"`
		Failures   []ResponseShardsFailure `json:"failures,omitempty"` // Only present when Failed > 0
	} `json:"_shards"`
	CreationTime int64 `json:"creation_time"`
	response     *opensearch.Response
}

// Inspect returns the Inspect type containing the raw *opensearch.Response
func (r PointInTimeCreateResp) Inspect() Inspect {
	return Inspect{Response: r.response}
}
