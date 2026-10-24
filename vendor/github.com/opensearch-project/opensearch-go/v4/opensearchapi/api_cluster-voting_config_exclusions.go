// SPDX-License-Identifier: Apache-2.0
//
// The OpenSearch Contributors require contributions made to
// this file be licensed under the Apache-2.0 license or a
// compatible open source license.

package opensearchapi

import (
	"net/http"

	"github.com/opensearch-project/opensearch-go/v4/internal/build"
)

// ClusterPostVotingConfigExclusionsReq represents possible options for the /_cluster/voting_config_exclusions request
type ClusterPostVotingConfigExclusionsReq struct {
	Header http.Header
	Params ClusterPostVotingConfigExclusionsParams
}

// GetRequest returns the *http.Request that gets executed by the client
func (r ClusterPostVotingConfigExclusionsReq) GetRequest(method string) (*http.Request, error) {
	return build.Request(
		method,
		"/_cluster/voting_config_exclusions",
		nil,
		r.Params.get(),
		r.Header,
	)
}

// ClusterDeleteVotingConfigExclusionsReq represents possible options for the /_cluster/voting_config_exclusions request
type ClusterDeleteVotingConfigExclusionsReq struct {
	Header http.Header
	Params ClusterDeleteVotingConfigExclusionsParams
}

// GetRequest returns the *http.Request that gets executed by the client
func (r ClusterDeleteVotingConfigExclusionsReq) GetRequest(method string) (*http.Request, error) {
	return build.Request(
		method,
		"/_cluster/voting_config_exclusions",
		nil,
		r.Params.get(),
		r.Header,
	)
}
