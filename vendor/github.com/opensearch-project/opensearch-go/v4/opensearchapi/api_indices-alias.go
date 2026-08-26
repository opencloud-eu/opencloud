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

type aliasClient struct {
	apiClient *Client
}

// Delete executes a delete alias request with the required AliasDeleteReq
func (c aliasClient) Delete(ctx context.Context, req AliasDeleteReq) (*AliasDeleteResp, error) {
	var (
		data AliasDeleteResp
		err  error
	)
	if data.response, err = do(ctx, c.apiClient, http.MethodDelete, req, &data); err != nil {
		return &data, err
	}

	return &data, nil
}

// Get executes a get alias request with the required AliasGetReq
func (c aliasClient) Get(ctx context.Context, req AliasGetReq) (*AliasGetResp, error) {
	var (
		data AliasGetResp
		err  error
	)
	if data.response, err = do(ctx, c.apiClient, http.MethodGet, req, &data); err != nil {
		return &data, err
	}

	return &data, nil
}

// Put executes a put alias request with the required AliasPutReq
func (c aliasClient) Put(ctx context.Context, req AliasPutReq) (*AliasPutResp, error) {
	var (
		data AliasPutResp
		err  error
	)
	if data.response, err = do(ctx, c.apiClient, http.MethodPut, req, &data); err != nil {
		return &data, err
	}

	return &data, nil
}

// Exists executes an exists alias request with the required AliasExistsReq
func (c aliasClient) Exists(ctx context.Context, req AliasExistsReq) (*opensearch.Response, error) {
	return do(ctx, c.apiClient, http.MethodHead, req, noBody)
}

// AliasDeleteReq represents possible options for the alias delete request
type AliasDeleteReq struct {
	Indices []string
	Alias   []string

	Header http.Header
	Params AliasDeleteParams
}

// GetRequest returns the *http.Request that gets executed by the client
func (r AliasDeleteReq) GetRequest(method string) (*http.Request, error) {
	path, err := ospath.IndicesDeleteAliasPath{
		Index: r.Indices,
		Name:  r.Alias,
	}.Build()
	if err != nil {
		return nil, err
	}
	return build.Request(method, path, nil, r.Params.get(), r.Header)
}

// AliasDeleteResp represents the returned struct of the alias delete response
type AliasDeleteResp struct {
	Acknowledged bool `json:"acknowledged"`
	response     *opensearch.Response
}

// Inspect returns the Inspect type containing the raw *opensearch.Response
func (r AliasDeleteResp) Inspect() Inspect {
	return Inspect{Response: r.response}
}

// AliasGetReq represents possible options for the alias get request
type AliasGetReq struct {
	Indices []string
	Alias   []string

	Header http.Header
	Params AliasGetParams
}

// GetRequest returns the *http.Request that gets executed by the client
func (r AliasGetReq) GetRequest(method string) (*http.Request, error) {
	path, err := ospath.IndicesGetAliasPath{
		Index: r.Indices,
		Name:  r.Alias,
	}.Build()
	if err != nil {
		return nil, err
	}
	return build.Request(method, path, nil, r.Params.get(), r.Header)
}

// AliasGetResp represents the returned struct of the alias get response
type AliasGetResp struct {
	response *opensearch.Response

	// Direct mapping of index names to their alias details as top-level keys
	raw map[string]AliasGetRespIndex
}

// AliasGetRespIndex represents the alias information for a specific index
type AliasGetRespIndex struct {
	Aliases map[string]json.RawMessage `json:"aliases"` // Available since OpenSearch 1.0.0
}

// GetIndices returns the map of index names to their alias information
func (r *AliasGetResp) GetIndices() map[string]AliasGetRespIndex {
	return r.raw
}

// UnmarshalJSON custom unmarshaling to handle dynamic index names as top-level keys
func (r *AliasGetResp) UnmarshalJSON(data []byte) error {
	// Unmarshal into a map to capture all dynamic index names
	r.raw = make(map[string]AliasGetRespIndex)
	return json.Unmarshal(data, &r.raw)
}

// Inspect returns the Inspect type containing the raw *opensearch.Response
func (r AliasGetResp) Inspect() Inspect {
	return Inspect{Response: r.response}
}

// AliasPutReq represents possible options for the alias put request
type AliasPutReq struct {
	Indices []string
	Alias   string

	Header http.Header
	Params AliasPutParams
}

// GetRequest returns the *http.Request that gets executed by the client
func (r AliasPutReq) GetRequest(method string) (*http.Request, error) {
	path, err := ospath.IndicesPutAliasPath{
		Index: r.Indices,
		Name:  r.Alias,
	}.Build()
	if err != nil {
		return nil, err
	}
	return build.Request(method, path, nil, r.Params.get(), r.Header)
}

// AliasPutResp represents the returned struct of the alias put response
type AliasPutResp struct {
	Acknowledged bool `json:"acknowledged"`
	response     *opensearch.Response
}

// Inspect returns the Inspect type containing the raw *opensearch.Response
func (r AliasPutResp) Inspect() Inspect {
	return Inspect{Response: r.response}
}

// AliasExistsReq represents possible options for the alias exists request
type AliasExistsReq struct {
	Indices []string
	Alias   []string

	Header http.Header
	Params AliasExistsParams
}

// GetRequest returns the *http.Request that gets executed by the client
func (r AliasExistsReq) GetRequest(method string) (*http.Request, error) {
	path, err := ospath.IndicesExistsAliasPath{
		Index: r.Indices,
		Name:  r.Alias,
	}.Build()
	if err != nil {
		return nil, err
	}
	return build.Request(method, path, nil, r.Params.get(), r.Header)
}
