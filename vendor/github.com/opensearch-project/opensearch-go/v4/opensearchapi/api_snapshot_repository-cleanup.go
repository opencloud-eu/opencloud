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

// SnapshotRepositoryCleanupReq represents possible options for the index create request
type SnapshotRepositoryCleanupReq struct {
	Repo string

	Header http.Header
	Params SnapshotRepositoryCleanupParams
}

// GetRequest returns the *http.Request that gets executed by the client
func (r SnapshotRepositoryCleanupReq) GetRequest(method string) (*http.Request, error) {
	path, err := ospath.SnapshotCleanupRepositoryPath{Repository: r.Repo}.Build()
	if err != nil {
		return nil, err
	}

	return build.Request(method, path, nil, r.Params.get(), r.Header)
}

// SnapshotRepositoryCleanupResp represents the returned struct of the index create response
type SnapshotRepositoryCleanupResp struct {
	Results struct {
		DeletedBytes int `json:"deleted_bytes"`
		DeletedBlobs int `json:"deleted_blobs"`
	} `json:"results"`
	response *opensearch.Response
}

// Inspect returns the Inspect type containing the raw *opensearch.Response
func (r SnapshotRepositoryCleanupResp) Inspect() Inspect {
	return Inspect{Response: r.response}
}
