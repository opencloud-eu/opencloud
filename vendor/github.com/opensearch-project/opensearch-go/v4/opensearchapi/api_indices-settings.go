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

type settingsClient struct {
	apiClient *Client
}

// Get executes a get settings request with the required SettingsGetReq
func (c settingsClient) Get(ctx context.Context, req *SettingsGetReq) (*SettingsGetResp, error) {
	if req == nil {
		req = &SettingsGetReq{}
	}
	var (
		data SettingsGetResp
		err  error
	)
	if data.response, err = do(ctx, c.apiClient, http.MethodGet, req, &data); err != nil {
		return &data, err
	}

	return &data, nil
}

// Put executes a put settings request with the required SettingsPutReq
func (c settingsClient) Put(ctx context.Context, req SettingsPutReq) (*SettingsPutResp, error) {
	var (
		data SettingsPutResp
		err  error
	)
	if data.response, err = do(ctx, c.apiClient, http.MethodPut, req, &data); err != nil {
		return &data, err
	}

	return &data, nil
}

// SettingsGetReq represents possible options for the settings get request
type SettingsGetReq struct {
	Indices  []string
	Settings []string

	Header http.Header
	Params SettingsGetParams
}

// GetRequest returns the *http.Request that gets executed by the client
func (r SettingsGetReq) GetRequest(method string) (*http.Request, error) {
	path, err := ospath.IndicesGetSettingsPath{
		Index: r.Indices,
		Name:  r.Settings,
	}.Build()
	if err != nil {
		return nil, err
	}
	return build.Request(method, path, nil, r.Params.get(), r.Header)
}

// SettingsGetResp represents the returned struct of the settings get response
type SettingsGetResp struct {
	response *opensearch.Response

	// Direct mapping of index names to their settings as top-level keys
	raw map[string]SettingsGetRespIndex
}

// SettingsGetRespIndex represents the structure of each index in the settings response
type SettingsGetRespIndex struct {
	Settings json.RawMessage `json:"settings"` // Available since OpenSearch 1.0.0
}

// GetIndices returns the map of index names to their settings
func (r *SettingsGetResp) GetIndices() map[string]SettingsGetRespIndex {
	return r.raw
}

// UnmarshalJSON custom unmarshaling to handle dynamic index names as top-level keys
func (r *SettingsGetResp) UnmarshalJSON(data []byte) error {
	// Unmarshal into a map to capture all dynamic index names
	r.raw = make(map[string]SettingsGetRespIndex)
	return json.Unmarshal(data, &r.raw)
}

// Inspect returns the Inspect type containing the raw *opensearch.Response
func (r SettingsGetResp) Inspect() Inspect {
	return Inspect{Response: r.response}
}

// SettingsPutReq represents possible options for the settings put request
type SettingsPutReq struct {
	Indices []string

	Body io.Reader

	Header http.Header
	Params SettingsPutParams
}

// GetRequest returns the *http.Request that gets executed by the client
func (r SettingsPutReq) GetRequest(method string) (*http.Request, error) {
	path, err := ospath.IndicesPutSettingsPath{Index: r.Indices}.Build()
	if err != nil {
		return nil, err
	}

	return build.Request(method, path, r.Body, r.Params.get(), r.Header)
}

// SettingsPutResp represents the returned struct of the settings put response
type SettingsPutResp struct {
	Acknowledged bool `json:"acknowledged"`
	response     *opensearch.Response
}

// Inspect returns the Inspect type containing the raw *opensearch.Response
func (r SettingsPutResp) Inspect() Inspect {
	return Inspect{Response: r.response}
}
