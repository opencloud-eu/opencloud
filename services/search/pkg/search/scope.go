package search

import (
	"errors"
	"strings"

	"github.com/opencloud-eu/opencloud/pkg/ast"
	"github.com/opencloud-eu/opencloud/pkg/kql"
)

// ErrNestedScope is returned when the scope sits inside a group: it narrows
// the whole search to a folder and cannot take part in a boolean expression.
var ErrNestedScope = errors.New("scope: must be a top-level restriction")

// ExtractScope takes the scope restriction out of the query, together with
// the operator that joined it, and returns the remaining nodes and the scope.
func ExtractScope(nodes []ast.Node) ([]ast.Node, string, error) {
	for i, n := range nodes {
		node, ok := n.(*ast.StringNode)
		if !ok {
			if group, isGroup := n.(*ast.GroupNode); isGroup && hasKey(group.Nodes, "scope") {
				return nil, "", ErrNestedScope
			}
			continue
		}
		if !strings.EqualFold(node.Key, "scope") {
			continue
		}

		rest := append(append([]ast.Node{}, nodes[:i]...), nodes[i+1:]...)
		if i > 0 && isOperator(rest[i-1], kql.BoolNOT) {
			rest = append(rest[:i-1], rest[i:]...)
			i--
		}
		switch {
		case i < len(rest) && isBooleanOperator(rest[i]):
			rest = append(rest[:i], rest[i+1:]...)
		case i > 0 && isBooleanOperator(rest[i-1]):
			rest = append(rest[:i-1], rest[i:]...)
		}
		return rest, strings.TrimSpace(node.Value), nil
	}
	return nodes, "", nil
}

// ExpandFavorites turns every is:favorite into the restriction on the user's
// favorites field, wherever it sits in the query.
func ExpandFavorites(nodes []ast.Node, userID string) {
	for _, n := range nodes {
		switch node := n.(type) {
		case *ast.StringNode:
			if strings.EqualFold(node.Key, "is") && strings.EqualFold(node.Value, "favorite") {
				node.Key = "Favorites"
				node.Value = userID
			}
		case *ast.GroupNode:
			ExpandFavorites(node.Nodes, userID)
		}
	}
}

func hasKey(nodes []ast.Node, key string) bool {
	for _, n := range nodes {
		switch node := n.(type) {
		case *ast.StringNode:
			if strings.EqualFold(node.Key, key) {
				return true
			}
		case *ast.GroupNode:
			if hasKey(node.Nodes, key) {
				return true
			}
		}
	}
	return false
}

func isOperator(n ast.Node, value string) bool {
	op, ok := n.(*ast.OperatorNode)
	return ok && op.Value == value
}

func isBooleanOperator(n ast.Node) bool {
	return isOperator(n, kql.BoolAND) || isOperator(n, kql.BoolOR)
}
