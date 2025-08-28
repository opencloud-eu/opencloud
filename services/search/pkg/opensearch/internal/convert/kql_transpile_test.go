package convert_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/opencloud-eu/opencloud/pkg/ast"
	"github.com/opencloud-eu/opencloud/services/search/pkg/opensearch/internal/convert"
	"github.com/opencloud-eu/opencloud/services/search/pkg/opensearch/internal/osu"
	"github.com/opencloud-eu/opencloud/services/search/pkg/opensearch/internal/test"
)

func TestTranspileKQLToOpenSearch(t *testing.T) {
	tests := []opensearchtest.TableTest[*ast.Ast, osu.Builder]{
		// kql to os dsl - type tests
		{
			Name: "term query - string node",
			Got: &ast.Ast{
				Nodes: []ast.Node{
					&ast.StringNode{Key: "Name", Value: "openCloud"},
				},
			},
			Want: osu.NewTermQuery[string]("Name").Value("openCloud"),
		},
		{
			Name: "term query - boolean node - true",
			Got: &ast.Ast{
				Nodes: []ast.Node{
					&ast.BooleanNode{Key: "Deleted", Value: true},
				},
			},
			Want: osu.NewTermQuery[bool]("Deleted").Value(true),
		},
		{
			Name: "term query - boolean node - false",
			Got: &ast.Ast{
				Nodes: []ast.Node{
					&ast.BooleanNode{Key: "Deleted", Value: false},
				},
			},
			Want: osu.NewTermQuery[bool]("Deleted").Value(false),
		},
		{
			Name: "match-phrase query - string node",
			Got: &ast.Ast{
				Nodes: []ast.Node{
					&ast.StringNode{Key: "Name", Value: "open cloud"},
				},
			},
			Want: osu.NewMatchPhraseQuery("Name").Query(`open cloud`),
		},
		{
			Name: "wildcard query - string node",
			Got: &ast.Ast{
				Nodes: []ast.Node{
					&ast.StringNode{Key: "Name", Value: "open*"},
				},
			},
			Want: osu.NewWildcardQuery("Name").Value("open*"),
		},
		{
			Name: "bool query",
			Got: &ast.Ast{
				Nodes: []ast.Node{
					&ast.GroupNode{Nodes: []ast.Node{
						&ast.StringNode{Key: "Name", Value: "a"},
						&ast.StringNode{Key: "Name", Value: "b"},
					}},
				},
			},
			Want: osu.NewBoolQuery().Must(
				osu.NewTermQuery[string]("Name").Value("a"),
				osu.NewTermQuery[string]("Name").Value("b"),
			),
		},
		{
			Name: "no bool query for single term",
			Got: &ast.Ast{
				Nodes: []ast.Node{
					&ast.GroupNode{Nodes: []ast.Node{
						&ast.StringNode{Key: "Name", Value: "any"},
					}},
				},
			},
			Want: osu.NewTermQuery[string]("Name").Value("any"),
		},
		{
			Name: "range query >",
			Got: &ast.Ast{
				Nodes: []ast.Node{
					&ast.DateTimeNode{
						Key:      "Mtime",
						Operator: &ast.OperatorNode{Value: ">"},
						Value:    opensearchtest.TimeMustParse(t, "2023-09-05T08:42:11.23554+02:00"),
					},
				},
			},
			Want: osu.NewRangeQuery[time.Time]("Mtime").Gt(opensearchtest.TimeMustParse(t, "2023-09-05T08:42:11.23554+02:00")),
		},
		{
			Name: "range query >=",
			Got: &ast.Ast{
				Nodes: []ast.Node{
					&ast.DateTimeNode{
						Key:      "Mtime",
						Operator: &ast.OperatorNode{Value: ">="},
						Value:    opensearchtest.TimeMustParse(t, "2023-09-05T08:42:11.23554+02:00"),
					},
				},
			},
			Want: osu.NewRangeQuery[time.Time]("Mtime").Gte(opensearchtest.TimeMustParse(t, "2023-09-05T08:42:11.23554+02:00")),
		},
		{
			Name: "range query <",
			Got: &ast.Ast{
				Nodes: []ast.Node{
					&ast.DateTimeNode{
						Key:      "Mtime",
						Operator: &ast.OperatorNode{Value: "<"},
						Value:    opensearchtest.TimeMustParse(t, "2023-09-05T08:42:11.23554+02:00"),
					},
				},
			},
			Want: osu.NewRangeQuery[time.Time]("Mtime").Lt(opensearchtest.TimeMustParse(t, "2023-09-05T08:42:11.23554+02:00")),
		},
		{
			Name: "range query <=",
			Got: &ast.Ast{
				Nodes: []ast.Node{
					&ast.DateTimeNode{
						Key:      "Mtime",
						Operator: &ast.OperatorNode{Value: "<="},
						Value:    opensearchtest.TimeMustParse(t, "2023-09-05T08:42:11.23554+02:00"),
					},
				},
			},
			Want: osu.NewRangeQuery[time.Time]("Mtime").Lte(opensearchtest.TimeMustParse(t, "2023-09-05T08:42:11.23554+02:00")),
		},
		// kql to os dsl - structure tests
		{
			Name: "[*]",
			Got: &ast.Ast{
				Nodes: []ast.Node{
					&ast.StringNode{Key: "Name", Value: "openCloud"},
				},
			},
			Want: osu.NewTermQuery[string]("Name").Value("openCloud"),
		},
		{
			Name: "[* *]",
			Got: &ast.Ast{
				Nodes: []ast.Node{
					&ast.StringNode{Key: "Name", Value: "openCloud"},
					&ast.StringNode{Key: "age", Value: "32"},
				},
			},
			Want: osu.NewBoolQuery().
				Must(
					osu.NewTermQuery[string]("Name").Value("openCloud"),
					osu.NewTermQuery[string]("age").Value("32"),
				),
		},
		{
			Name: "[* AND *]",
			Got: &ast.Ast{
				Nodes: []ast.Node{
					&ast.StringNode{Key: "Name", Value: "openCloud"},
					&ast.OperatorNode{Value: "AND"},
					&ast.StringNode{Key: "age", Value: "32"},
				},
			},
			Want: osu.NewBoolQuery().
				Must(
					osu.NewTermQuery[string]("Name").Value("openCloud"),
					osu.NewTermQuery[string]("age").Value("32"),
				),
		},
		{
			Name: "[* OR *]",
			Got: &ast.Ast{
				Nodes: []ast.Node{
					&ast.StringNode{Key: "Name", Value: "openCloud"},
					&ast.OperatorNode{Value: "OR"},
					&ast.StringNode{Key: "age", Value: "32"},
				},
			},
			Want: osu.NewBoolQuery().
				Params(&osu.BoolQueryParams{MinimumShouldMatch: 1}).
				Should(
					osu.NewTermQuery[string]("Name").Value("openCloud"),
					osu.NewTermQuery[string]("age").Value("32"),
				),
		},
		{
			Name: "[NOT *]",
			Got: &ast.Ast{
				Nodes: []ast.Node{
					&ast.OperatorNode{Value: "NOT"},
					&ast.StringNode{Key: "age", Value: "32"},
				},
			},
			Want: osu.NewBoolQuery().
				MustNot(
					osu.NewTermQuery[string]("age").Value("32"),
				),
		},
		{
			Name: "[* NOT *]",
			Got: &ast.Ast{
				Nodes: []ast.Node{
					&ast.StringNode{Key: "Name", Value: "openCloud"},
					&ast.OperatorNode{Value: "NOT"},
					&ast.StringNode{Key: "age", Value: "32"},
				},
			},
			Want: osu.NewBoolQuery().
				Must(
					osu.NewTermQuery[string]("Name").Value("openCloud"),
				).
				MustNot(
					osu.NewTermQuery[string]("age").Value("32"),
				),
		},
		{
			Name: "[* OR * OR *]",
			Got: &ast.Ast{
				Nodes: []ast.Node{
					&ast.StringNode{Key: "Name", Value: "openCloud"},
					&ast.OperatorNode{Value: "OR"},
					&ast.StringNode{Key: "age", Value: "32"},
					&ast.OperatorNode{Value: "OR"},
					&ast.StringNode{Key: "age", Value: "44"},
				},
			},
			Want: osu.NewBoolQuery().
				Params(&osu.BoolQueryParams{MinimumShouldMatch: 1}).
				Should(
					osu.NewTermQuery[string]("Name").Value("openCloud"),
					osu.NewTermQuery[string]("age").Value("32"),
					osu.NewTermQuery[string]("age").Value("44"),
				),
		},
		{
			Name: "[* AND * OR *]",
			Got: &ast.Ast{
				Nodes: []ast.Node{
					&ast.StringNode{Key: "a", Value: "a"},
					&ast.OperatorNode{Value: "AND"},
					&ast.StringNode{Key: "b", Value: "b"},
					&ast.OperatorNode{Value: "OR"},
					&ast.StringNode{Key: "c", Value: "c"},
				},
			},
			Want: osu.NewBoolQuery().
				Params(&osu.BoolQueryParams{MinimumShouldMatch: 1}).
				Must(
					osu.NewTermQuery[string]("a").Value("a"),
				).
				Should(
					osu.NewTermQuery[string]("b").Value("b"),
					osu.NewTermQuery[string]("c").Value("c"),
				),
		},
		{
			Name: "[* OR * AND *]",
			Got: &ast.Ast{
				Nodes: []ast.Node{
					&ast.StringNode{Key: "a", Value: "a"},
					&ast.OperatorNode{Value: "OR"},
					&ast.StringNode{Key: "b", Value: "b"},
					&ast.OperatorNode{Value: "AND"},
					&ast.StringNode{Key: "c", Value: "c"},
				},
			},
			Want: osu.NewBoolQuery().
				Params(&osu.BoolQueryParams{MinimumShouldMatch: 1}).
				Must(
					osu.NewTermQuery[string]("b").Value("b"),
					osu.NewTermQuery[string]("c").Value("c"),
				).
				Should(
					osu.NewTermQuery[string]("a").Value("a"),
				),
		},
		{
			Name: "[* OR * AND *]",
			Got: &ast.Ast{
				Nodes: []ast.Node{
					&ast.StringNode{Key: "a", Value: "a"},
					&ast.OperatorNode{Value: "OR"},
					&ast.StringNode{Key: "b", Value: "b"},
					&ast.OperatorNode{Value: "AND"},
					&ast.StringNode{Key: "c", Value: "c"},
				},
			},
			Want: osu.NewBoolQuery().
				Params(&osu.BoolQueryParams{MinimumShouldMatch: 1}).
				Should(
					osu.NewTermQuery[string]("a").Value("a"),
				).
				Must(
					osu.NewTermQuery[string]("b").Value("b"),
					osu.NewTermQuery[string]("c").Value("c"),
				),
		},
		{
			Name: "[[* OR * OR *] AND *]",
			Got: &ast.Ast{
				Nodes: []ast.Node{
					&ast.GroupNode{Nodes: []ast.Node{
						&ast.StringNode{Key: "a", Value: "a"},
						&ast.OperatorNode{Value: "OR"},
						&ast.StringNode{Key: "b", Value: "b"},
						&ast.OperatorNode{Value: "OR"},
						&ast.StringNode{Key: "c", Value: "c"},
					}},
					&ast.OperatorNode{Value: "AND"},
					&ast.StringNode{Key: "d", Value: "d"},
				},
			},
			Want: osu.NewBoolQuery().
				Must(
					osu.NewBoolQuery().
						Params(&osu.BoolQueryParams{MinimumShouldMatch: 1}).
						Should(
							osu.NewTermQuery[string]("a").Value("a"),
							osu.NewTermQuery[string]("b").Value("b"),
							osu.NewTermQuery[string]("c").Value("c"),
						),
					osu.NewTermQuery[string]("d").Value("d"),
				),
		},
		{
			Name: "[[* OR * OR *] AND NOT *]",
			Got: &ast.Ast{
				Nodes: []ast.Node{
					&ast.GroupNode{Nodes: []ast.Node{
						&ast.StringNode{Key: "a", Value: "a"},
						&ast.OperatorNode{Value: "OR"},
						&ast.StringNode{Key: "b", Value: "b"},
						&ast.OperatorNode{Value: "OR"},
						&ast.StringNode{Key: "c", Value: "c"},
					}},
					&ast.OperatorNode{Value: "AND"},
					&ast.OperatorNode{Value: "NOT"},
					&ast.StringNode{Key: "d", Value: "d"},
				},
			},
			Want: osu.NewBoolQuery().
				Must(
					osu.NewBoolQuery().
						Params(&osu.BoolQueryParams{MinimumShouldMatch: 1}).
						Should(
							osu.NewTermQuery[string]("a").Value("a"),
							osu.NewTermQuery[string]("b").Value("b"),
							osu.NewTermQuery[string]("c").Value("c"),
						),
				).MustNot(
				osu.NewTermQuery[string]("d").Value("d"),
			),
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			if test.Skip {
				t.Skip("skipping test: " + test.Name)
			}

			dsl, err := convert.TranspileKQLToOpenSearch(test.Got.Nodes)
			assert.NoError(t, err)

			assert.JSONEq(t, opensearchtest.JSONMustMarshal(t, test.Want), opensearchtest.JSONMustMarshal(t, dsl))
		})
	}
}
