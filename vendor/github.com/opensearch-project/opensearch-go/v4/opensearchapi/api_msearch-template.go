// SPDX-License-Identifier: Apache-2.0
//
// The OpenSearch Contributors require contributions made to
// this file be licensed under the Apache-2.0 license or a
// compatible open source license.

package opensearchapi

import (
	"context"
	"io"
	"net/http"

	"github.com/opensearch-project/opensearch-go/v4"
	"github.com/opensearch-project/opensearch-go/v4/internal/build"
	ospath "github.com/opensearch-project/opensearch-go/v4/internal/path"
)

// MSearchTemplate executes a /_msearch request with the optional MSearchTemplateReq.
//
// Partial-failure detection mirrors MSearch: see [MSearchErrors] for the
// runtime-collapse rule and caller patterns.
func (c Client) MSearchTemplate(ctx context.Context, req MSearchTemplateReq) (*MSearchTemplateResp, error) {
	var (
		data MSearchTemplateResp
		err  error
	)
	if data.response, err = do(ctx, &c, http.MethodPost, req, &data); err != nil {
		return &data, err
	}

	return &data, collapsePerOpErrors(data.PartialFailures(c.errors), func(errs []error) error {
		return &MSearchTemplateErrors{errs: errs}
	})
}

// MSearchTemplateReq represents possible options for the /_msearch request
type MSearchTemplateReq struct {
	Indices []string

	Body io.Reader

	Header http.Header
	Params MSearchTemplateParams
}

// GetRequest returns the *http.Request that gets executed by the client
func (r MSearchTemplateReq) GetRequest(method string) (*http.Request, error) {
	path, err := ospath.MsearchTemplatePath{Index: r.Indices}.Build()
	if err != nil {
		return nil, err
	}
	return build.Request(method, path, r.Body, r.Params.get(), r.Header)
}

// MSearchTemplateResp represents the returned struct of the /_msearch response
type MSearchTemplateResp struct {
	Took      int `json:"took"`
	Responses []struct {
		Took    int            `json:"took"`
		Timeout bool           `json:"timed_out"`
		Shards  ResponseShards `json:"_shards"`
		Hits    struct {
			Total struct {
				Value    int    `json:"value"`
				Relation string `json:"relation"`
			} `json:"total"`
			MaxScore *float32    `json:"max_score"`
			Hits     []SearchHit `json:"hits"`
		} `json:"hits"`
		Status int            `json:"status"`
		Error  *DocumentError `json:"error,omitempty"`
	} `json:"responses"`
	response *opensearch.Response
}

// Inspect returns the Inspect type containing the raw *opensearch.Response
func (r MSearchTemplateResp) Inspect() Inspect {
	return Inspect{Response: r.response}
}
