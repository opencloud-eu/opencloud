package query

import "github.com/opencloud-eu/opencloud/pkg/ast"

// ForceCaseSensitive marks every string restriction in the tree as an exact,
// case-sensitive match. It is applied to a decoded aggregation filter after
// Normalize: the filter values are exact bucket keys the server issued, so they
// must match the case-preserving base field, not the lowercased sibling.
func ForceCaseSensitive(a *ast.Ast) *ast.Ast {
	if a == nil {
		return a
	}
	forceCaseSensitiveNodes(a.Nodes)
	return a
}

func forceCaseSensitiveNodes(nodes []ast.Node) {
	for _, n := range nodes {
		switch node := n.(type) {
		case *ast.StringNode:
			node.Exact = true
			node.CaseInsensitive = false
		case *ast.GroupNode:
			forceCaseSensitiveNodes(node.Nodes)
		}
	}
}
