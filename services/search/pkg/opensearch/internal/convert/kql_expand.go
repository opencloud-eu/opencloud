package convert

import (
	"strings"

	"github.com/opencloud-eu/opencloud/pkg/ast"
	"github.com/opencloud-eu/opencloud/services/search/pkg/search"
)

// LowerValues folds restriction values for fields whose index analyzer
// lowercases (search.LowercaseValueFields, shared with bleve); case-preserved
// fields keep their casing. Runs after query.Normalize, so keys are resolved.
func LowerValues(nodes []ast.Node) []ast.Node {
	for _, n := range nodes {
		switch node := n.(type) {
		case *ast.StringNode:
			if _, ok := search.LowercaseValueFields()[node.Key]; ok {
				node.Value = strings.ToLower(node.Value)
			}
		case *ast.GroupNode:
			LowerValues(node.Nodes)
		}
	}
	return nodes
}
