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

// SnapshotRepositoryDeleteReq represents possible options for the index create request
type SnapshotRepositoryDeleteReq struct {
	Repos []string

	Header http.Header
	Params SnapshotRepositoryDeleteParams
}

// GetRequest returns the *http.Request that gets executed by the client
func (r SnapshotRepositoryDeleteReq) GetRequest(method string) (*http.Request, error) {
	path, err := ospath.SnapshotDeleteRepositoryPath{Repository: r.Repos}.Build()
	if err != nil {
		return nil, err
	}
	return build.Request(method, path, nil, r.Params.get(), r.Header)
}

// SnapshotRepositoryDeleteResp represents the returned struct of the index create response
type SnapshotRepositoryDeleteResp struct {
	Acknowledged bool `json:"acknowledged"`
	response     *opensearch.Response
}

// Inspect returns the Inspect type containing the raw *opensearch.Response
func (r SnapshotRepositoryDeleteResp) Inspect() Inspect {
	return Inspect{Response: r.response}
}
