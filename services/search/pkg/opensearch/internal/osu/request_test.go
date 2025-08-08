package osu_test

import (
	"io"
	"testing"

	opensearchgoAPI "github.com/opensearch-project/opensearch-go/v4/opensearchapi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/opencloud-eu/opencloud/services/search/pkg/opensearch/internal/osu"
	"github.com/opencloud-eu/opencloud/services/search/pkg/opensearch/internal/test"
)

func TestRequestBody(t *testing.T) {
	tests := []opensearchtest.TableTest[osu.Builder, map[string]any]{
		{
			Name: "simple",
			Got:  osu.NewRequestBody[any](osu.NewTermQuery[string]("name").Value("tom")),
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
				req, _ := osu.BuildSearchReq(
					&opensearchgoAPI.SearchReq{},
					osu.NewTermQuery[string]("content").Value("content"),
					osu.SearchReqOptions{
						Highlight: &osu.HighlightOption{
							PreTags:  []string{"<b>"},
							PostTags: []string{"</b>"},
							Fields: map[string]osu.HighlightOption{
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
