package query

import (
	"github.com/opencloud-eu/opencloud/pkg/ast"
	"github.com/opencloud-eu/opencloud/pkg/kql"
)

// MergeFilters parses and normalizes the main query and each decoded aggregation
// filter, forces the filters to exact case-sensitive matches, and ANDs
// everything into one AST ready to compile. The main query and every filter are
// wrapped in their own group so the AND binds across the whole query rather than
// tangling with the query's own operator precedence. With no filters the
// normalized main AST is returned unchanged.
func MergeFilters(b Builder, qs string, filters []string) (*ast.Ast, error) {
	main, err := b.Build(qs)
	if err != nil {
		return nil, err
	}
	main = Normalize(main, ResolveField)
	if len(filters) == 0 {
		return main, nil
	}

	nodes := make([]ast.Node, 0, 2*len(filters)+1)
	if len(main.Nodes) > 0 {
		nodes = append(nodes, &ast.GroupNode{Base: &ast.Base{}, Nodes: main.Nodes})
	}
	for _, f := range filters {
		fa, err := b.Build(f)
		if err != nil {
			return nil, err
		}
		fa = Normalize(fa, ResolveField)
		ForceCaseSensitive(fa)
		if len(fa.Nodes) == 0 {
			continue
		}
		if len(nodes) > 0 {
			nodes = append(nodes, &ast.OperatorNode{Value: kql.BoolAND})
		}
		nodes = append(nodes, &ast.GroupNode{Base: &ast.Base{}, Nodes: fa.Nodes})
	}
	return &ast.Ast{Nodes: nodes}, nil
}
