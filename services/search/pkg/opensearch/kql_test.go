package opensearch_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/opencloud-eu/opencloud/pkg/ast"
	"github.com/opencloud-eu/opencloud/services/search/pkg/opensearch"
)

func TestKQL_Compile(t *testing.T) {
	tests := []tableTest[*ast.Ast, opensearch.Builder]{
		// field name tests
		{
			name: "Name is the default field",
			got: &ast.Ast{
				Nodes: []ast.Node{
					&ast.StringNode{Value: "moby di*"},
				},
			},
			want: opensearch.NewBoolQuery().
				Must(
					opensearch.NewTermQuery[string]("Name").Value("moby di*"),
				),
		},
		{
			name: "remaps known field names",
			got: &ast.Ast{
				Nodes: []ast.Node{
					&ast.StringNode{Key: "mediatype", Value: "application/gzip"},
				},
			},
			want: opensearch.NewBoolQuery().
				Must(
					opensearch.NewTermQuery[string]("MimeType").Value("application/gzip"),
				),
		},
		// kql to os dsl - type tests
		// kql to os dsl - structure tests
		{
			name: "[*]",
			got: &ast.Ast{
				Nodes: []ast.Node{
					&ast.StringNode{Key: "name", Value: "moby di*"},
				},
			},
			want: opensearch.NewBoolQuery().
				Must(
					opensearch.NewTermQuery[string]("Name").Value("moby di*"),
				),
		},
		{
			name: "[* *]",
			got: &ast.Ast{
				Nodes: []ast.Node{
					&ast.StringNode{Key: "name", Value: "moby di*"},
					&ast.StringNode{Key: "age", Value: "32"},
				},
			},
			want: opensearch.NewBoolQuery().
				Must(
					opensearch.NewTermQuery[string]("Name").Value("moby di*"),
					opensearch.NewTermQuery[string]("age").Value("32"),
				),
		},
		{
			name: "[* AND *]",
			got: &ast.Ast{
				Nodes: []ast.Node{
					&ast.StringNode{Key: "name", Value: "moby di*"},
					&ast.OperatorNode{Value: "AND"},
					&ast.StringNode{Key: "age", Value: "32"},
				},
			},
			want: opensearch.NewBoolQuery().
				Must(
					opensearch.NewTermQuery[string]("Name").Value("moby di*"),
					opensearch.NewTermQuery[string]("age").Value("32"),
				),
		},
		{
			name: "[* OR *]",
			got: &ast.Ast{
				Nodes: []ast.Node{
					&ast.StringNode{Key: "name", Value: "moby di*"},
					&ast.OperatorNode{Value: "OR"},
					&ast.StringNode{Key: "age", Value: "32"},
				},
			},
			want: opensearch.NewBoolQuery().
				Should(
					opensearch.NewTermQuery[string]("Name").Value("moby di*"),
					opensearch.NewTermQuery[string]("age").Value("32"),
				),
		},
		{
			name: "[* OR * OR *]",
			got: &ast.Ast{
				Nodes: []ast.Node{
					&ast.StringNode{Key: "name", Value: "moby di*"},
					&ast.OperatorNode{Value: "OR"},
					&ast.StringNode{Key: "age", Value: "32"},
					&ast.OperatorNode{Value: "OR"},
					&ast.StringNode{Key: "age", Value: "44"},
				},
			},
			want: opensearch.NewBoolQuery().
				Should(
					opensearch.NewTermQuery[string]("Name").Value("moby di*"),
					opensearch.NewTermQuery[string]("age").Value("32"),
					opensearch.NewTermQuery[string]("age").Value("44"),
				),
		},
		{
			name: "[* AND * OR *]",
			got: &ast.Ast{
				Nodes: []ast.Node{
					&ast.StringNode{Key: "a", Value: "a"},
					&ast.OperatorNode{Value: "AND"},
					&ast.StringNode{Key: "b", Value: "b"},
					&ast.OperatorNode{Value: "OR"},
					&ast.StringNode{Key: "c", Value: "c"},
				},
			},
			want: opensearch.NewBoolQuery().
				Must(
					opensearch.NewTermQuery[string]("a").Value("a"),
				).
				Should(
					opensearch.NewTermQuery[string]("b").Value("b"),
					opensearch.NewTermQuery[string]("c").Value("c"),
				),
		},
		{
			name: "[* OR * AND *]",
			got: &ast.Ast{
				Nodes: []ast.Node{
					&ast.StringNode{Key: "a", Value: "a"},
					&ast.OperatorNode{Value: "OR"},
					&ast.StringNode{Key: "b", Value: "b"},
					&ast.OperatorNode{Value: "AND"},
					&ast.StringNode{Key: "c", Value: "c"},
				},
			},
			want: opensearch.NewBoolQuery().
				Must(
					opensearch.NewTermQuery[string]("c").Value("c"),
				).
				Should(
					opensearch.NewTermQuery[string]("a").Value("a"),
					opensearch.NewTermQuery[string]("b").Value("b"),
				),
		},
		{
			name: "[[* OR * OR *] AND *]",
			got: &ast.Ast{
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
			want: opensearch.NewBoolQuery().
				Must(
					opensearch.NewBoolQuery().
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
		t.Run(test.name, func(t *testing.T) {
			compiler, err := opensearch.NewKQL()
			assert.NoError(t, err)

			got, err := compiler.Compile(test.got)
			assert.NoError(t, err)

			gotJSON, err := got.MarshalJSON()
			assert.NoError(t, err)

			wantJSON, err := test.want.MarshalJSON()
			assert.NoError(t, err)

			assert.JSONEq(t, string(wantJSON), string(gotJSON))
		})
	}
}
