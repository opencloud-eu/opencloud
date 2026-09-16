package convert_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/opencloud-eu/opencloud/pkg/ast"
	"github.com/opencloud-eu/opencloud/services/search/internal/opensearchtest"
	"github.com/opencloud-eu/opencloud/services/search/pkg/opensearch/internal/convert"
	"github.com/opencloud-eu/opencloud/services/search/pkg/opensearch/internal/osu"
	"github.com/opencloud-eu/opencloud/services/search/pkg/query"
)

func TestTranspileOpenExtensions(t *testing.T) {
	oct := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	either := func(alternatives ...osu.Builder) osu.Builder {
		return osu.NewBoolQuery().Params(&osu.BoolQueryParams{MinimumShouldMatch: 1}).Should(alternatives...)
	}

	tests := []opensearchtest.TableTest[ast.Node, osu.Builder]{
		{
			Name: "a plain string asks the lowercased sibling",
			Got:  &ast.StringNode{Key: "extensions.com.example.project.state", Value: "Open"},
			Want: osu.NewTermQuery[string]("ext.com.example.project.state.@lower").Value("open"),
		},
		{
			Name: "= asks the keyword sibling as written",
			Got:  &ast.StringNode{Key: "extensions.com.example.project.state", Value: "Open", Exact: true},
			Want: osu.NewTermQuery[string]("ext.com.example.project.state.@keyword").Value("Open"),
		},
		{
			Name: "a numeric literal also asks the number sibling",
			Got:  &ast.StringNode{Key: "extensions.com.example.project.priority", Value: "3"},
			Want: either(
				osu.NewTermQuery[string]("ext.com.example.project.priority.@lower").Value("3"),
				osu.NewRangeQuery[float64]("ext.com.example.project.priority.@number").Gte(3).Lte(3),
			),
		},
		{
			Name: "a boolean literal also asks the bool sibling",
			Got:  &ast.StringNode{Key: "extensions.com.example.project.done", Value: "false"},
			Want: either(
				osu.NewTermQuery[string]("ext.com.example.project.done.@lower").Value("false"),
				osu.NewTermQuery[bool]("ext.com.example.project.done.@bool").Value(false),
			),
		},
		{
			Name: "a date-time literal also asks the date sibling",
			Got:  &ast.StringNode{Key: "extensions.com.example.project.due", Value: "2026-10-01T00:00:00Z"},
			Want: either(
				osu.NewTermQuery[string]("ext.com.example.project.due.@lower").Value("2026-10-01t00:00:00z"),
				osu.NewRangeQuery[time.Time]("ext.com.example.project.due.@date").Gte(oct).Lte(oct),
			),
		},
		{
			Name: "a wildcard runs on the lowercased sibling",
			Got:  &ast.StringNode{Key: "extensions.com.example.project.state", Value: "Op*"},
			Want: osu.NewWildcardQuery("ext.com.example.project.state.@lower").Value("op*"),
		},
		{
			Name: "a number range asks the number sibling",
			Got:  &ast.NumberNode{Key: "extensions.com.example.project.priority", Operator: &ast.OperatorNode{Value: ">"}, Value: 3},
			Want: osu.NewRangeQuery[float64]("ext.com.example.project.priority.@number").Gt(3),
		},
		{
			Name: "a date range asks the date sibling",
			Got:  &ast.DateTimeNode{Key: "extensions.com.example.project.due", Operator: &ast.OperatorNode{Value: "<="}, Value: oct},
			Want: osu.NewRangeQuery[time.Time]("ext.com.example.project.due.@date").Lte(oct),
		},
		{
			Name: "a boolean node asks the bool sibling",
			Got:  &ast.BooleanNode{Key: "extensions.com.example.project.done", Value: true},
			Want: osu.NewTermQuery[bool]("ext.com.example.project.done.@bool").Value(true),
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			normalized := query.Normalize(&ast.Ast{Nodes: []ast.Node{test.Got}}, query.ResolveField)
			dsl, err := convert.TranspileKQLToOpenSearch(normalized.Nodes)
			assert.NoError(t, err)
			assert.JSONEq(t, opensearchtest.JSONMustMarshal(t, test.Want), opensearchtest.JSONMustMarshal(t, dsl))
		})
	}
}
