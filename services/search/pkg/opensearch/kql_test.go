package opensearch_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/opencloud-eu/opencloud/pkg/ast"
	"github.com/opencloud-eu/opencloud/services/search/pkg/opensearch"
	"github.com/opencloud-eu/opencloud/services/search/pkg/opensearch/internal/test"
)

func TestKQL_Compile(t *testing.T) {
	tests := []opensearchtest.TableTest[*ast.Ast, opensearch.Builder]{
		// field name tests
		{
			Name: "Name is the default field",
			Got: &ast.Ast{
				Nodes: []ast.Node{
					&ast.StringNode{Value: "openCloud"},
				},
			},
			Want: opensearch.NewTermQuery[string]("Name").Value("openCloud"),
		},
		{
			Name: "remaps known field names",
			Got: &ast.Ast{
				Nodes: []ast.Node{
					&ast.StringNode{Key: "mediatype", Value: "application/gzip"},
				},
			},
			Want: opensearch.NewTermQuery[string]("MimeType").Value("application/gzip"),
		},
		// kql to os dsl - type tests
		{
			Name: "term query",
			Got: &ast.Ast{
				Nodes: []ast.Node{
					&ast.StringNode{Key: "Name", Value: "openCloud"},
				},
			},
			Want: opensearch.NewTermQuery[string]("Name").Value("openCloud"),
		},
		{
			Name: "match-phrase query",
			Got: &ast.Ast{
				Nodes: []ast.Node{
					&ast.StringNode{Key: "Name", Value: "open cloud"},
				},
			},
			Want: opensearch.NewMatchPhraseQuery("Name").Query("open cloud"),
		},
		{
			Name: "wildcard query",
			Got: &ast.Ast{
				Nodes: []ast.Node{
					&ast.StringNode{Key: "Name", Value: "open*"},
				},
			},
			Want: opensearch.NewWildcardQuery("Name").Value("open*"),
		},
		{
			Name: "bool query",
			Got: &ast.Ast{
				Nodes: []ast.Node{
					&ast.GroupNode{Nodes: []ast.Node{
						&ast.StringNode{Value: "a"},
						&ast.StringNode{Value: "b"},
					}},
				},
			},
			Want: opensearch.NewBoolQuery().Must(
				opensearch.NewTermQuery[string]("Name").Value("a"),
				opensearch.NewTermQuery[string]("Name").Value("b"),
			),
		},
		{
			Name: "no bool query for single term",
			Got: &ast.Ast{
				Nodes: []ast.Node{
					&ast.GroupNode{Nodes: []ast.Node{
						&ast.StringNode{Value: "any"},
					}},
				},
			},
			Want: opensearch.NewTermQuery[string]("Name").Value("any"),
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
			Want: opensearch.NewRangeQuery[time.Time]("Mtime").Gt(opensearchtest.TimeMustParse(t, "2023-09-05T08:42:11.23554+02:00")),
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
			Want: opensearch.NewRangeQuery[time.Time]("Mtime").Gte(opensearchtest.TimeMustParse(t, "2023-09-05T08:42:11.23554+02:00")),
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
			Want: opensearch.NewRangeQuery[time.Time]("Mtime").Lt(opensearchtest.TimeMustParse(t, "2023-09-05T08:42:11.23554+02:00")),
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
			Want: opensearch.NewRangeQuery[time.Time]("Mtime").Lte(opensearchtest.TimeMustParse(t, "2023-09-05T08:42:11.23554+02:00")),
		},
		// kql to os dsl - structure tests
		{
			Name: "[*]",
			Got: &ast.Ast{
				Nodes: []ast.Node{
					&ast.StringNode{Key: "name", Value: "openCloud"},
				},
			},
			Want: opensearch.NewTermQuery[string]("Name").Value("openCloud"),
		},
		{
			Name: "[* *]",
			Got: &ast.Ast{
				Nodes: []ast.Node{
					&ast.StringNode{Key: "name", Value: "openCloud"},
					&ast.StringNode{Key: "age", Value: "32"},
				},
			},
			Want: opensearch.NewBoolQuery().
				Must(
					opensearch.NewTermQuery[string]("Name").Value("openCloud"),
					opensearch.NewTermQuery[string]("age").Value("32"),
				),
		},
		{
			Name: "[* AND *]",
			Got: &ast.Ast{
				Nodes: []ast.Node{
					&ast.StringNode{Key: "name", Value: "openCloud"},
					&ast.OperatorNode{Value: "AND"},
					&ast.StringNode{Key: "age", Value: "32"},
				},
			},
			Want: opensearch.NewBoolQuery().
				Must(
					opensearch.NewTermQuery[string]("Name").Value("openCloud"),
					opensearch.NewTermQuery[string]("age").Value("32"),
				),
		},
		{
			Name: "[* OR *]",
			Got: &ast.Ast{
				Nodes: []ast.Node{
					&ast.StringNode{Key: "name", Value: "openCloud"},
					&ast.OperatorNode{Value: "OR"},
					&ast.StringNode{Key: "age", Value: "32"},
				},
			},
			Want: opensearch.NewBoolQuery(opensearch.BoolQueryOptions{MinimumShouldMatch: 1}).
				Should(
					opensearch.NewTermQuery[string]("Name").Value("openCloud"),
					opensearch.NewTermQuery[string]("age").Value("32"),
				),
		},
		{
			Name: "[* OR * OR *]",
			Got: &ast.Ast{
				Nodes: []ast.Node{
					&ast.StringNode{Key: "name", Value: "openCloud"},
					&ast.OperatorNode{Value: "OR"},
					&ast.StringNode{Key: "age", Value: "32"},
					&ast.OperatorNode{Value: "OR"},
					&ast.StringNode{Key: "age", Value: "44"},
				},
			},
			Want: opensearch.NewBoolQuery(opensearch.BoolQueryOptions{MinimumShouldMatch: 1}).
				Should(
					opensearch.NewTermQuery[string]("Name").Value("openCloud"),
					opensearch.NewTermQuery[string]("age").Value("32"),
					opensearch.NewTermQuery[string]("age").Value("44"),
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
			Want: opensearch.NewBoolQuery(opensearch.BoolQueryOptions{MinimumShouldMatch: 1}).
				Must(
					opensearch.NewTermQuery[string]("a").Value("a"),
				).
				Should(
					opensearch.NewTermQuery[string]("b").Value("b"),
					opensearch.NewTermQuery[string]("c").Value("c"),
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
			Want: opensearch.NewBoolQuery(opensearch.BoolQueryOptions{MinimumShouldMatch: 1}).
				Must(
					opensearch.NewTermQuery[string]("b").Value("b"),
					opensearch.NewTermQuery[string]("c").Value("c"),
				).
				Should(
					opensearch.NewTermQuery[string]("a").Value("a"),
				),
		},
		{
			Name: "NEW[* OR * AND *]",
			Got: &ast.Ast{
				Nodes: []ast.Node{
					&ast.StringNode{Key: "a", Value: "a"},
					&ast.OperatorNode{Value: "OR"},
					&ast.StringNode{Key: "b", Value: "b"},
					&ast.OperatorNode{Value: "AND"},
					&ast.StringNode{Key: "c", Value: "c"},
				},
			},
			Want: opensearch.NewBoolQuery(opensearch.BoolQueryOptions{MinimumShouldMatch: 1}).
				Should(
					opensearch.NewTermQuery[string]("a").Value("a"),
				).
				Must(
					opensearch.NewTermQuery[string]("b").Value("b"),
					opensearch.NewTermQuery[string]("c").Value("c"),
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
			Want: opensearch.NewBoolQuery().
				Must(
					opensearch.NewBoolQuery(opensearch.BoolQueryOptions{MinimumShouldMatch: 1}).
						Should(
							opensearch.NewTermQuery[string]("a").Value("a"),
							opensearch.NewTermQuery[string]("b").Value("b"),
							opensearch.NewTermQuery[string]("c").Value("c"),
						),
					opensearch.NewTermQuery[string]("d").Value("d"),
				),
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			compiler, err := opensearch.NewKQL()
			assert.NoError(t, err)

			dsl, err := compiler.Compile(test.Got)
			assert.NoError(t, err)

			assert.JSONEq(t, opensearchtest.JSONMustMarshal(t, test.Want), opensearchtest.JSONMustMarshal(t, dsl))
		})
	}
}
