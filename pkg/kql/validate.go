package kql

import (
	"github.com/opencloud-eu/opencloud/pkg/ast"
	"github.com/opencloud-eu/opencloud/services/search/pkg/query"
)

func validateAst(a *ast.Ast) error {
	switch node := a.Nodes[0].(type) {
	case *ast.OperatorNode:
		switch node.Value {
		case BoolAND, BoolOR:
			return &query.StartsWithBinaryOperatorError{Node: node}
		}
	}
	return nil
}

func validateGroupNode(n *ast.GroupNode) error {
	switch node := n.Nodes[0].(type) {
	case *ast.OperatorNode:
		switch node.Value {
		case BoolAND, BoolOR:
			return &query.StartsWithBinaryOperatorError{Node: node}
		}
	}

	if n.Key != "" {
		for _, node := range n.Nodes {
			if ast.NodeKey(node) != "" {
				return &query.NamedGroupInvalidNodesError{Node: node}
			}
		}
	}

	return nil
}
