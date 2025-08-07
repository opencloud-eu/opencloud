package opensearch_test

import (
	"io"
	"testing"

	opensearchgoAPI "github.com/opensearch-project/opensearch-go/v4/opensearchapi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/opencloud-eu/opencloud/services/search/pkg/opensearch"
	"github.com/opencloud-eu/opencloud/services/search/pkg/opensearch/internal/test"
)

func TestRequestBody(t *testing.T) {
	tests := []opensearchtest.TableTest[opensearch.Builder, map[string]any]{
		{
			Name: "simple",
			Got:  opensearch.NewRequestBody[any](opensearch.NewTermQuery[string]("name").Value("tom")),
			Want: map[string]any{
				"query": map[string]any{
					"term": map[string]any{
						"name": map[string]any{
							"value": "tom",
						},
					},
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			assert.JSONEq(t, opensearchtest.JSONMustMarshal(t, test.Want), opensearchtest.JSONMustMarshal(t, test.Got))
		})
	}
}

func TestBuildSearchReq(t *testing.T) {
	tests := []opensearchtest.TableTest[io.Reader, map[string]any]{
		{
			Name: "highlight",
			Got: func() io.Reader {
				req, _ := opensearch.BuildSearchReq(
					&opensearchgoAPI.SearchReq{},
					opensearch.NewTermQuery[string]("content").Value("content"),
					opensearch.SearchReqOptions{
						Highlight: &opensearch.HighlightOption{
							PreTags:  []string{"<b>"},
							PostTags: []string{"</b>"},
							Fields: map[string]opensearch.HighlightOption{
								"content": {},
							},
						},
					},
				)

				return req.Body
			}(),
			Want: map[string]any{
				"query": map[string]any{
					"term": map[string]any{
						"content": map[string]any{
							"value": "content",
						},
					},
				},
				"highlight": map[string]any{
					"pre_tags":  []string{"<b>"},
					"post_tags": []string{"</b>"},
					"fields": map[string]any{
						"content": map[string]any{},
					},
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			body, err := io.ReadAll(test.Got)
			require.NoError(t, err)
			assert.JSONEq(t, opensearchtest.JSONMustMarshal(t, test.Want), string(body))
		})
	}
}
