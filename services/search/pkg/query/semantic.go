package query

import (
	"strings"

	"github.com/opencloud-eu/opencloud/pkg/ast"
	"github.com/opencloud-eu/opencloud/pkg/kql"
)

// ExtractSemantic splits the semantic free-text clause (`semantic:"..."`) off
// a parsed KQL query. The clause is removed from the tree (together with the
// operator it hung on) and its text returned; the remaining query is the
// filter part. Working on the parsed tree means quoted values elsewhere are
// never touched: name:"*semantic:x*" is a name search for a literal string.
func ExtractSemantic(a *ast.Ast) (text string) {
	a.Nodes = extractSemanticNodes(a.Nodes, &text)
	return text
}

// extractSemanticNodes rewrites nodes, dropping the first semantic restriction
// found. Keyed groups (e.g. name:(...)) are skipped: their bare children are
// values of the group key, not restrictions of their own.
func extractSemanticNodes(nodes []ast.Node, text *string) []ast.Node {
	out := make([]ast.Node, 0, len(nodes))
	for _, n := range nodes {
		n = toPointer(n) // parser emits some nodes by value
		switch node := n.(type) {
		case *ast.StringNode:
			if *text == "" && strings.EqualFold(node.Key, "semantic") && node.Value != "" {
				*text = node.Value
				continue
			}
		case *ast.GroupNode:
			if node.Key == "" {
				node.Nodes = extractSemanticNodes(node.Nodes, text)
				if len(node.Nodes) == 0 {
					continue // the group only held the semantic clause
				}
			}
		}
		out = append(out, n)
	}
	return sanitizeOperators(out)
}

// sanitizeOperators repairs a node sequence after removals: operators must
// only stand between operands, so leading, trailing and stacked operators
// left behind by a removed operand are dropped.
func sanitizeOperators(nodes []ast.Node) []ast.Node {
	out := make([]ast.Node, 0, len(nodes))
	for _, n := range nodes {
		op, isOp := n.(*ast.OperatorNode)
		if !isOp {
			out = append(out, n)
			continue
		}
		if len(out) == 0 && op.Value != kql.BoolNOT {
			continue // binary operator without a left operand
		}
		if len(out) > 0 {
			if prev, prevIsOp := out[len(out)-1].(*ast.OperatorNode); prevIsOp && prev.Value != kql.BoolNOT {
				// two operators in a row: the operand between them was removed
				out[len(out)-1] = n
				continue
			}
		}
		out = append(out, n)
	}
	// drop a trailing operator (its right operand was removed)
	for len(out) > 0 {
		if _, isOp := out[len(out)-1].(*ast.OperatorNode); !isOp {
			break
		}
		out = out[:len(out)-1]
	}
	return out
}
