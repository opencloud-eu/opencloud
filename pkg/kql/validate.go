package kql

import (
	"github.com/opencloud-eu/opencloud/pkg/ast"
)

func validateAst(a *ast.Ast) error {
	switch node := a.Nodes[0].(type) {
	case *ast.OperatorNode:
		switch node.Value {
		case BoolAND, BoolOR:
			return &StartsWithBinaryOperatorError{Node: node}
		}
	}
	return nil
}

func validateGroupNode(n *ast.GroupNode) error {
	switch node := n.Nodes[0].(type) {
	case *ast.OperatorNode:
		switch node.Value {
		case BoolAND, BoolOR:
			return &StartsWithBinaryOperatorError{Node: node}
		}
	}

	if n.Key != "" {
		for _, node := range n.Nodes {
			if ast.NodeKey(node) != "" {
				return &NamedGroupInvalidNodesError{Node: node}
			}
		}
	}

	return nil
}
