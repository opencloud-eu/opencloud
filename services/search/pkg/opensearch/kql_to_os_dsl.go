package opensearch

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/opencloud-eu/opencloud/pkg/ast"
	"github.com/opencloud-eu/opencloud/pkg/kql"
)

var (
	ErrUnsupportedNodeType = fmt.Errorf("unsupported node type")
)

type KQLToOsDSL struct{}

func NewKQLToOsDSL() (*KQLToOsDSL, error) {
	return &KQLToOsDSL{}, nil
}

func (k *KQLToOsDSL) Compile(tree *ast.Ast) (Builder, error) {
	q, err := k.transpile(tree.Nodes)
	if err != nil {
		return nil, err
	}

	return q, nil
}

func (k *KQLToOsDSL) transpile(nodes []ast.Node) (Builder, error) {
	if len(nodes) == 0 {
		return nil, fmt.Errorf("no nodes to compile")
	}

	expandedNodes, err := expandKQLASTNodes(nodes)
	if err != nil {
		return nil, fmt.Errorf("failed to expand KQL AST nodes: %w", err)
	}

	if len(expandedNodes) == 1 {
		builder, err := k.toBuilder(expandedNodes[0])
		if err != nil {
			return nil, fmt.Errorf("failed to get builder for single node: %w", err)
		}
		return builder, nil
	}

	boolQuery := NewBoolQuery()
	add := boolQuery.Must

	for i, node := range expandedNodes {
		nextOp := k.getOperatorValueAt(expandedNodes, i+1)
		prevOp := k.getOperatorValueAt(expandedNodes, i-1)

		switch {
		case nextOp == kql.BoolOR:
			add = boolQuery.Should
		case nextOp == kql.BoolAND:
			add = boolQuery.Must
		case prevOp == kql.BoolNOT:
			add = boolQuery.MustNot
		}

		builder, err := k.toBuilder(node)
		switch {
		// if the node is not known, we skip it, such as an operator node
		case errors.Is(err, ErrUnsupportedNodeType):
			continue
		case err != nil:
			return nil, fmt.Errorf("failed to get builder for node %T: %w", node, err)
		}

		if _, ok := node.(*ast.OperatorNode); ok {
			// operatorNodes are not builders, so we skip them
			continue
		}

		add(builder)
	}

	if len(boolQuery.should) != 0 {
		boolQuery.options.MinimumShouldMatch = 1
	}

	return boolQuery, nil
}

func (k *KQLToOsDSL) getOperatorValueAt(nodes []ast.Node, i int) string {
	if i < 0 || i >= len(nodes) {
		return ""
	}

	if opn, ok := nodes[i].(*ast.OperatorNode); ok {
		return opn.Value
	}

	return ""
}

func (k *KQLToOsDSL) toBuilder(node ast.Node) (Builder, error) {
	var builder Builder

	switch node := node.(type) {
	case *ast.BooleanNode:
		return NewTermQuery[bool](node.Key).Value(node.Value), nil
	case *ast.StringNode:
		isWildcard := strings.Contains(node.Value, "*")
		if isWildcard {
			return NewWildcardQuery(node.Key).Value(node.Value), nil
		}

		totalTerms := strings.Split(node.Value, " ")
		isSingleTerm := len(totalTerms) == 1
		isMultiTerm := len(totalTerms) >= 1
		switch {
		case isSingleTerm:
			return NewTermQuery[string](node.Key).Value(node.Value), nil
		case isMultiTerm:
			return NewMatchPhraseQuery(node.Key).Query(node.Value), nil
		}

		return nil, fmt.Errorf("unsupported string node value: %s", node.Value)
	case *ast.DateTimeNode:
		if node.Operator == nil {
			return builder, fmt.Errorf("date time node without operator: %w", ErrUnsupportedNodeType)
		}

		query := NewRangeQuery[time.Time](node.Key)

		switch node.Operator.Value {
		case ">":
			return query.Gt(node.Value), nil
		case ">=":
			return query.Gte(node.Value), nil
		case "<":
			return query.Lt(node.Value), nil
		case "<=":
			return query.Lte(node.Value), nil
		}

		return nil, fmt.Errorf("unsupported operator %s for date time node: %w", node.Operator.Value, ErrUnsupportedNodeType)
	case *ast.GroupNode:
		group, err := k.transpile(node.Nodes)
		if err != nil {
			return nil, fmt.Errorf("failed to build group: %w", err)
		}

		return group, nil
	}

	return nil, fmt.Errorf("%w: %T", ErrUnsupportedNodeType, node)
}
