// SPDX-License-Identifier: Apache-2.0
//
// The OpenSearch Contributors require contributions made to
// this file be licensed under the Apache-2.0 license or a
// compatible open source license.

package opensearchapi

import (
	"encoding/json"
	"net/http"

	"github.com/opensearch-project/opensearch-go/v4"
	"github.com/opensearch-project/opensearch-go/v4/internal/build"
	ospath "github.com/opensearch-project/opensearch-go/v4/internal/path"
)

// IndicesRecoveryReq represents possible options for the index shrink request
type IndicesRecoveryReq struct {
	Indices []string

	Header http.Header
	Params IndicesRecoveryParams
}

// GetRequest returns the *http.Request that gets executed by the client
func (r IndicesRecoveryReq) GetRequest(method string) (*http.Request, error) {
	path, err := ospath.IndicesRecoveryPath{Index: r.Indices}.Build()
	if err != nil {
		return nil, err
	}
	return build.Request(method, path, nil, r.Params.get(), r.Header)
}

// IndicesRecoveryResp represents the returned struct of the index recovery response
type IndicesRecoveryResp struct {
	response *opensearch.Response

	// Direct mapping of index names to their recovery details as top-level keys
	raw map[string]IndicesRecoveryRespIndex
}

// IndicesRecoveryRespIndex represents the recovery information for a specific index
type IndicesRecoveryRespIndex struct {
	Shards []IndicesRecoveryShard `json:"shards"` // Available since OpenSearch 1.0.0
}

// IndicesRecoveryShard represents recovery information for a specific shard
type IndicesRecoveryShard struct {
	ID                int                     `json:"id"`                   // Available since OpenSearch 1.0.0
	Type              string                  `json:"type"`                 // Available since OpenSearch 1.0.0
	Stage             string                  `json:"stage"`                // Available since OpenSearch 1.0.0
	Primary           bool                    `json:"primary"`              // Available since OpenSearch 1.0.0
	StartTimeInMillis int64                   `json:"start_time_in_millis"` // Available since OpenSearch 1.0.0
	StopTimeInMillis  int64                   `json:"stop_time_in_millis"`  // Available since OpenSearch 1.0.0
	TotalTimeInMillis int                     `json:"total_time_in_millis"` // Available since OpenSearch 1.0.0
	Source            IndicesRecoveryNodeInfo `json:"source"`               // Available since OpenSearch 1.0.0
	Target            IndicesRecoveryNodeInfo `json:"target"`               // Available since OpenSearch 1.0.0
	Index             struct {
		Size struct {
			TotalInBytes     int    `json:"total_in_bytes"`     // Available since OpenSearch 1.0.0
			ReusedInBytes    int    `json:"reused_in_bytes"`    // Available since OpenSearch 1.0.0
			RecoveredInBytes int    `json:"recovered_in_bytes"` // Available since OpenSearch 1.0.0
			Percent          string `json:"percent"`            // Available since OpenSearch 1.0.0
		} `json:"size"` // Available since OpenSearch 1.0.0
		Files struct {
			Total     int    `json:"total"`     // Available since OpenSearch 1.0.0
			Reused    int    `json:"reused"`    // Available since OpenSearch 1.0.0
			Recovered int    `json:"recovered"` // Available since OpenSearch 1.0.0
			Percent   string `json:"percent"`   // Available since OpenSearch 1.0.0
		} `json:"files"` // Available since OpenSearch 1.0.0
		TotalTimeInMillis          int `json:"total_time_in_millis"`           // Available since OpenSearch 1.0.0
		SourceThrottleTimeInMillis int `json:"source_throttle_time_in_millis"` // Available since OpenSearch 1.0.0
		TargetThrottleTimeInMillis int `json:"target_throttle_time_in_millis"` // Available since OpenSearch 1.0.0
	} `json:"index"` // Available since OpenSearch 1.0.0
	Translog struct {
		Recovered         int    `json:"recovered"`            // Available since OpenSearch 1.0.0
		Total             int    `json:"total"`                // Available since OpenSearch 1.0.0
		Percent           string `json:"percent"`              // Available since OpenSearch 1.0.0
		TotalOnStart      int    `json:"total_on_start"`       // Available since OpenSearch 1.0.0
		TotalTimeInMillis int    `json:"total_time_in_millis"` // Available since OpenSearch 1.0.0
	} `json:"translog"` // Available since OpenSearch 1.0.0
	VerifyIndex struct {
		CheckIndexTimeInMillis int `json:"check_index_time_in_millis"` // Available since OpenSearch 1.0.0
		TotalTimeInMillis      int `json:"total_time_in_millis"`       // Available since OpenSearch 1.0.0
	} `json:"verify_index"` // Available since OpenSearch 1.0.0
}

// GetIndices returns the map of index names to their recovery information
func (r *IndicesRecoveryResp) GetIndices() map[string]IndicesRecoveryRespIndex {
	return r.raw
}

// UnmarshalJSON custom unmarshaling to handle dynamic index names as top-level keys
func (r *IndicesRecoveryResp) UnmarshalJSON(data []byte) error {
	// Unmarshal into a map to capture all dynamic index names
	r.raw = make(map[string]IndicesRecoveryRespIndex)
	return json.Unmarshal(data, &r.raw)
}

// Inspect returns the Inspect type containing the raw *opensearch.Response
func (r IndicesRecoveryResp) Inspect() Inspect {
	return Inspect{Response: r.response}
}

// IndicesRecoveryNodeInfo is a sub type of IndicesRecoveryResp representing Node information
type IndicesRecoveryNodeInfo struct {
	ID               string `json:"id"`                // Available since OpenSearch 1.0.0
	Host             string `json:"host"`              // Available since OpenSearch 1.0.0
	TransportAddress string `json:"transport_address"` // Available since OpenSearch 1.0.0
	IP               string `json:"ip"`                // Available since OpenSearch 1.0.0
	Name             string `json:"name"`              // Available since OpenSearch 1.0.0
}
