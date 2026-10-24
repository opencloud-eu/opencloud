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

// CatSegmentsReq represent possible options for the /_cat/segments request
type CatSegmentsReq struct {
	Indices []string
	Header  http.Header
	Params  CatSegmentsParams
}

// GetRequest returns the *http.Request that gets executed by the client
func (r CatSegmentsReq) GetRequest(method string) (*http.Request, error) {
	path, err := ospath.CatSegmentsPath{Index: r.Indices}.Build()
	if err != nil {
		return nil, err
	}
	return build.Request(method, path, nil, r.Params.get(), r.Header)
}

// CatSegmentsResp represents the returned struct of the /_cat/segments response
type CatSegmentsResp struct {
	Segments []CatSegmentResp
	response *opensearch.Response
}

// CatSegmentResp represents one index of the CatSegmentsResp
type CatSegmentResp struct {
	Index       string `json:"index"`
	Shard       int    `json:"shard,string"`
	Prirep      string `json:"prirep"`
	IP          string `json:"ip"`
	ID          string `json:"id"`
	Segment     string `json:"segment"`
	Generation  int    `json:"generation,string"`
	DocsCount   int    `json:"docs.count,string"`
	DocsDeleted int    `json:"docs.deleted,string"`
	Size        string `json:"size"`
	SizeMemory  string `json:"size.memory"`
	Committed   bool   `json:"committed,string"`
	Searchable  bool   `json:"searchable,string"`
	Version     string `json:"version"`
	Compound    bool   `json:"compound,string"`
}

// Inspect returns the Inspect type containing the raw *opensearch.Response
func (r CatSegmentsResp) Inspect() Inspect {
	return Inspect{
		Response: r.response,
	}
}
