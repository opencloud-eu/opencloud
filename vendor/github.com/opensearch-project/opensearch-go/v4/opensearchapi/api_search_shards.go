// SPDX-License-Identifier: Apache-2.0
//
// The OpenSearch Contributors require contributions made to
// this file be licensed under the Apache-2.0 license or a
// compatible open source license.

package opensearchapi

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/opensearch-project/opensearch-go/v4"
	"github.com/opensearch-project/opensearch-go/v4/internal/build"
	ospath "github.com/opensearch-project/opensearch-go/v4/internal/path"
)

// SearchShards executes a /_search request with the optional SearchShardsReq
func (c Client) SearchShards(ctx context.Context, req *SearchShardsReq) (*SearchShardsResp, error) {
	if req == nil {
		req = &SearchShardsReq{}
	}

	var (
		data SearchShardsResp
		err  error
	)
	if data.response, err = do(ctx, &c, http.MethodPost, req, &data); err != nil {
		return &data, err
	}

	return &data, nil
}

// SearchShardsReq represents possible options for the /_search request
type SearchShardsReq struct {
	Indices []string

	Header http.Header
	Params SearchShardsParams
}

// GetRequest returns the *http.Request that gets executed by the client
func (r SearchShardsReq) GetRequest(method string) (*http.Request, error) {
	path, err := ospath.SearchShardsPath{Index: r.Indices}.Build()
	if err != nil {
		return nil, err
	}
	return build.Request(method, path, nil, r.Params.get(), r.Header)
}

// SearchShardsResp represents the returned struct of the /_search response
type SearchShardsResp struct {
	Nodes map[string]struct {
		Name             string            `json:"name"`
		EphemeralID      string            `json:"ephemeral_id"`
		TransportAddress string            `json:"transport_address"`
		Attributes       map[string]string `json:"attributes"`
	} `json:"nodes"`
	Indices map[string]json.RawMessage `json:"indices"`
	Shards  [][]struct {
		State                    string  `json:"state"`
		Primary                  bool    `json:"primary"`
		SearchOnly               bool    `json:"searchOnly"`
		Node                     string  `json:"node"`
		RelocatingNode           *string `json:"relocating_node"`
		Shard                    int     `json:"shard"`
		Index                    string  `json:"index"`
		ExpectedShardSizeInBytes int     `json:"expected_shard_size_in_bytes"`
		RecoverySource           struct {
			Type                    string `json:"type"`
			BootstrapNewHistoryUUID *bool  `json:"bootstrap_new_history_uuid,omitempty"` // Present since OpenSearch 1.0.0
		} `json:"recovery_source"`
		UnassignedInfo struct {
			Reason           string  `json:"reason"`
			At               string  `json:"at"`
			Delayed          bool    `json:"delayed"`
			AllocationStatus string  `json:"allocation_status"`
			Details          *string `json:"details,omitempty"` // Available since OpenSearch 1.0.0
		} `json:"unassigned_info"`
		AllocationID struct {
			ID           string  `json:"id"`
			RelocationID *string `json:"relocation_id,omitempty"` // Present when shard is relocating (available since OpenSearch 1.0.0)
		} `json:"allocation_id"`
	} `json:"shards"`
	response *opensearch.Response
}

// Inspect returns the Inspect type containing the raw *opensearch.Response
func (r SearchShardsResp) Inspect() Inspect {
	return Inspect{Response: r.response}
}
