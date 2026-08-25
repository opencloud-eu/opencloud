// SPDX-License-Identifier: Apache-2.0
//
// The OpenSearch Contributors require contributions made to
// this file be licensed under the Apache-2.0 license or a
// compatible open source license.

package opensearchapi

import (
	"context"
	"encoding/json"
	"io"
	"net/http"

	"github.com/opensearch-project/opensearch-go/v4"
	"github.com/opensearch-project/opensearch-go/v4/internal/build"
	ospath "github.com/opensearch-project/opensearch-go/v4/internal/path"
)

type mappingClient struct {
	apiClient *Client
}

// Get executes a get mapping request with the required MappingGetReq
func (c mappingClient) Get(ctx context.Context, req *MappingGetReq) (*MappingGetResp, error) {
	if req == nil {
		req = &MappingGetReq{}
	}

	var (
		data MappingGetResp
		err  error
	)
	if data.response, err = do(ctx, c.apiClient, http.MethodGet, req, &data); err != nil {
		return &data, err
	}

	return &data, nil
}

// Put executes a put mapping request with the required MappingPutReq
func (c mappingClient) Put(ctx context.Context, req MappingPutReq) (*MappingPutResp, error) {
	var (
		data MappingPutResp
		err  error
	)
	if data.response, err = do(ctx, c.apiClient, http.MethodPut, req, &data); err != nil {
		return &data, err
	}

	return &data, nil
}

// Field executes a field mapping request with the optional MappingFieldReq
func (c mappingClient) Field(ctx context.Context, req *MappingFieldReq) (*MappingFieldResp, error) {
	if req == nil {
		req = &MappingFieldReq{}
	}

	var (
		data MappingFieldResp
		err  error
	)
	if data.response, err = do(ctx, c.apiClient, http.MethodGet, req, &data); err != nil {
		return &data, err
	}

	return &data, nil
}

// MappingGetReq represents possible options for the mapping get request
type MappingGetReq struct {
	Indices []string

	Header http.Header
	Params MappingGetParams
}

// GetRequest returns the *http.Request that gets executed by the client
func (r MappingGetReq) GetRequest(method string) (*http.Request, error) {
	path, err := ospath.IndicesGetMappingPath{Index: r.Indices}.Build()
	if err != nil {
		return nil, err
	}
	return build.Request(method, path, nil, r.Params.get(), r.Header)
}

// MappingGetResp represents the returned struct of the mapping get response
type MappingGetResp struct {
	response *opensearch.Response

	// Direct mapping of index names to their mappings as top-level keys
	raw map[string]MappingGetRespIndex
}

// MappingGetRespIndex represents the structure of each index in the mapping response
type MappingGetRespIndex struct {
	Mappings json.RawMessage `json:"mappings"` // Available since OpenSearch 1.0.0
}

// GetIndices returns the map of index names to their mappings
func (r *MappingGetResp) GetIndices() map[string]MappingGetRespIndex {
	return r.raw
}

// UnmarshalJSON custom unmarshaling to handle dynamic index names as top-level keys
func (r *MappingGetResp) UnmarshalJSON(data []byte) error {
	// Unmarshal into a map to capture all dynamic index names
	r.raw = make(map[string]MappingGetRespIndex)
	return json.Unmarshal(data, &r.raw)
}

// Inspect returns the Inspect type containing the raw *opensearch.Response
func (r MappingGetResp) Inspect() Inspect {
	return Inspect{Response: r.response}
}

// MappingPutReq represents possible options for the mapping put request
type MappingPutReq struct {
	Indices []string

	Body io.Reader

	Header http.Header
	Params MappingPutParams
}

// GetRequest returns the *http.Request that gets executed by the client
func (r MappingPutReq) GetRequest(method string) (*http.Request, error) {
	path, err := ospath.IndicesPutMappingPath{Index: r.Indices}.Build()
	if err != nil {
		return nil, err
	}

	return build.Request(method, path, r.Body, r.Params.get(), r.Header)
}

// MappingPutResp represents the returned struct of the mapping put response
type MappingPutResp struct {
	Acknowledged bool `json:"acknowledged"`
	response     *opensearch.Response
}

// Inspect returns the Inspect type containing the raw *opensearch.Response
func (r MappingPutResp) Inspect() Inspect {
	return Inspect{Response: r.response}
}

// MappingFieldReq represents possible options for the mapping field request
type MappingFieldReq struct {
	Indices []string
	Fields  []string

	Header http.Header
	Params MappingPutParams
}

// GetRequest returns the *http.Request that gets executed by the client
func (r MappingFieldReq) GetRequest(method string) (*http.Request, error) {
	path, err := ospath.IndicesGetFieldMappingPath{
		Fields: r.Fields,
		Index:  r.Indices,
	}.Build()
	if err != nil {
		return nil, err
	}
	return build.Request(method, path, nil, r.Params.get(), r.Header)
}

// MappingFieldResp represents the returned struct of the mapping field response
type MappingFieldResp struct {
	response *opensearch.Response

	// Direct mapping of index names to their field mappings as top-level keys
	raw map[string]MappingFieldRespIndex
}

// MappingFieldRespIndex represents the structure of each index in the field mapping response
type MappingFieldRespIndex struct {
	Mappings json.RawMessage `json:"mappings"` // Available since OpenSearch 1.0.0
}

// GetIndices returns the map of index names to their field mappings
func (r *MappingFieldResp) GetIndices() map[string]MappingFieldRespIndex {
	return r.raw
}

// UnmarshalJSON custom unmarshaling to handle dynamic index names as top-level keys
func (r *MappingFieldResp) UnmarshalJSON(data []byte) error {
	// Unmarshal into a map to capture all dynamic index names
	r.raw = make(map[string]MappingFieldRespIndex)
	return json.Unmarshal(data, &r.raw)
}

// Inspect returns the Inspect type containing the raw *opensearch.Response
func (r MappingFieldResp) Inspect() Inspect {
	return Inspect{Response: r.response}
}
