package convert_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/opencloud-eu/opencloud/pkg/ast"
	"github.com/opencloud-eu/opencloud/services/search/pkg/opensearch/internal/convert"
	opensearchtest "github.com/opencloud-eu/opencloud/services/search/pkg/opensearch/internal/test"
)

// LowerValues runs after the shared query.Normalize pass, so it operates on
// already-resolved pointer nodes. Field resolution, media-type expansion and
// group-key defaulting are tested once at the query.Normalize level (see
// pkg/query normalize_test), not here. Only lowercase-analyzed fields get their
// value folded; case-preserved fields keep their casing.
func TestLowerValues(t *testing.T) {
	tests := []opensearchtest.TableTest[[]ast.Node, []ast.Node]{
		{
			Name: "lowercase-analyzed field: value is folded, recursing into groups",
			Got: []ast.Node{
				&ast.StringNode{Key: "Name", Value: "StringNode"},
				&ast.GroupNode{Key: "GroupNode", Nodes: []ast.Node{
					&ast.StringNode{Key: "Name", Value: "StringNode"},
				}},
			},
			Want: []ast.Node{
				&ast.StringNode{Key: "Name", Value: "stringnode"},
				&ast.GroupNode{Key: "GroupNode", Nodes: []ast.Node{
					&ast.StringNode{Key: "Name", Value: "stringnode"},
				}},
			},
		},
		{
			Name: "case-preserved field: value keeps its casing",
			Got: []ast.Node{
				&ast.StringNode{Key: "aBc", Value: "StringNode"},
				&ast.GroupNode{Key: "GroupNode", Nodes: []ast.Node{
					&ast.StringNode{Key: "aBc", Value: "StringNode"},
				}},
			},
			Want: []ast.Node{
				&ast.StringNode{Key: "aBc", Value: "StringNode"},
				&ast.GroupNode{Key: "GroupNode", Nodes: []ast.Node{
					&ast.StringNode{Key: "aBc", Value: "StringNode"},
				}},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			require.Equal(t, test.Want, convert.LowerValues(test.Got))
		})
	}
}
