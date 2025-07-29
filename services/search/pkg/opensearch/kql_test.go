package opensearch_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/opencloud-eu/opencloud/pkg/ast"
	"github.com/opencloud-eu/opencloud/services/search/pkg/opensearch"
)

func TestKQL_Compile(t *testing.T) {
	tests := []tableTest[*ast.Ast, opensearch.Builder]{
		{
			name: "federated",
			got: &ast.Ast{
				Nodes: []ast.Node{
					&ast.StringNode{Value: "federated"},
				},
			},
			want: opensearch.NewRootQuery(opensearch.NewTermQuery[string]("Name").Value("federated")),
		},
		{
			name: "John Smith",
			got: &ast.Ast{
				Nodes: []ast.Node{
					&ast.StringNode{Value: "John Smith"},
				},
			},
			want: opensearch.NewRootQuery(opensearch.NewMatchPhraseQuery("Name").Query("John Smith")),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := opensearch.KQL{}.Compile(test.got)
			assert.NoError(t, err)

			gotJSON, err := got.MarshalJSON()
			assert.NoError(t, err)

			wantJSON, err := test.want.MarshalJSON()
			assert.NoError(t, err)

			assert.JSONEq(t, string(wantJSON), string(gotJSON))
		})
	}
}
