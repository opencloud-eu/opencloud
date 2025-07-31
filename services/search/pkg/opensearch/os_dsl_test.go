package opensearch_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/opencloud-eu/opencloud/services/search/pkg/opensearch"
	"github.com/opencloud-eu/opencloud/services/search/pkg/opensearch/internal/test"
)

func TestBuilderToBoolQuery(t *testing.T) {
	tests := []opensearchtest.TableTest[opensearch.Builder, *opensearch.BoolQuery]{
		{
			Name: "term-query",
			Got:  opensearch.NewTermQuery[string]("Name").Value("openCloud"),
			Want: opensearch.NewBoolQuery().Must(opensearch.NewTermQuery[string]("Name").Value("openCloud")),
		},
		{
			Name: "root-query",
			Got:  opensearch.NewRootQuery(opensearch.NewTermQuery[string]("Name").Value("openCloud")),
			Want: opensearch.NewBoolQuery().Must(opensearch.NewTermQuery[string]("Name").Value("openCloud")),
		},
		{
			Name: "bool-query",
			Got:  opensearch.NewBoolQuery().Must(opensearch.NewTermQuery[string]("Name").Value("openCloud")),
			Want: opensearch.NewBoolQuery().Must(opensearch.NewTermQuery[string]("Name").Value("openCloud")),
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			assert.JSONEq(t, opensearchtest.ToJSON(t, test.Want), opensearchtest.ToJSON(t, opensearch.BuilderToBoolQuery(test.Got)))
		})
	}
}
