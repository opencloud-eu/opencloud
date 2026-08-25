// SPDX-License-Identifier: Apache-2.0
//
// The OpenSearch Contributors require contributions made to
// this file be licensed under the Apache-2.0 license or a
// compatible open source license.

package opensearchapi

import (
	"net/http"

	"github.com/opensearch-project/opensearch-go/v4/internal/build"
	ospath "github.com/opensearch-project/opensearch-go/v4/internal/path"
)

// NodesHotThreadsReq represents possible options for the /_nodes request
type NodesHotThreadsReq struct {
	NodeID []string

	Header http.Header
	Params NodesHotThreadsParams
}

// GetRequest returns the *http.Request that gets executed by the client
func (r NodesHotThreadsReq) GetRequest(method string) (*http.Request, error) {
	path, err := ospath.NodesHotThreadsPath{NodeID: r.NodeID}.Build()
	if err != nil {
		return nil, err
	}
	return build.Request(method, path, nil, r.Params.get(), r.Header)
}
