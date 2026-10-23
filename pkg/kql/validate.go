package kql

import (
	"github.com/opencloud-eu/opencloud/pkg/ast"
)

func validateAst(a *ast.Ast) error {
	return validateOperators(a.Nodes)
}

func validateGroupNode(n *ast.GroupNode) error {
	if err := validateOperators(n.Nodes); err != nil {
		return err
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

func validateOperators(nodes []ast.Node) error {
	for i, node := range nodes {
		op, ok := node.(*ast.OperatorNode)
		if !ok {
			continue
		}

		switch op.Value {
		case BoolAND, BoolOR:
			if i == 0 {
				return &DanglingOperatorError{Node: op}
			}
			if _, prevIsOperator := nodes[i-1].(*ast.OperatorNode); prevIsOperator {
				return &DanglingOperatorError{Node: op}
			}
		}

		if i == len(nodes)-1 {
			return &DanglingOperatorError{Node: op}
		}
	}

	return nil
}
