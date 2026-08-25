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

// SnapshotRepositoryVerifyReq represents possible options for the index create request
type SnapshotRepositoryVerifyReq struct {
	Repo string

	Header http.Header
	Params SnapshotRepositoryVerifyParams
}

// GetRequest returns the *http.Request that gets executed by the client
func (r SnapshotRepositoryVerifyReq) GetRequest(method string) (*http.Request, error) {
	path, err := ospath.SnapshotVerifyRepositoryPath{Repository: r.Repo}.Build()
	if err != nil {
		return nil, err
	}

	return build.Request(method, path, nil, r.Params.get(), r.Header)
}

// SnapshotRepositoryVerifyResp represents the returned struct of the index create response
type SnapshotRepositoryVerifyResp struct {
	Nodes map[string]struct {
		Name string `json:"name"`
	} `json:"nodes"`
	response *opensearch.Response
}

// Inspect returns the Inspect type containing the raw *opensearch.Response
func (r SnapshotRepositoryVerifyResp) Inspect() Inspect {
	return Inspect{Response: r.response}
}
