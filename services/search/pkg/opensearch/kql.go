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

type KQL struct{}

func NewKQL() (*KQL, error) {
	return &KQL{}, nil
}

func (k *KQL) Compile(tree *ast.Ast) (Builder, error) {
	q, err := k.compile(tree.Nodes)
	if err != nil {
		return nil, err
	}

	return q, nil
}

func (k *KQL) compile(nodes []ast.Node) (Builder, error) {
	if len(nodes) == 0 {
		return nil, fmt.Errorf("no nodes to compile")
	}

	if len(nodes) == 1 {
		builder, err := k.getBuilder(nodes[0])
		if err != nil {
			return nil, fmt.Errorf("failed to get builder for single node: %w", err)
		}
		return builder, nil
	}

	boolQuery := NewBoolQuery()
	add := boolQuery.Must

	for i, node := range nodes {
		nextOp := k.getOperatorValueAt(nodes, i+1)
		prevOp := k.getOperatorValueAt(nodes, i-1)

		switch {
		case nextOp == kql.BoolOR:
			add = boolQuery.Should
		case nextOp == kql.BoolAND:
			add = boolQuery.Must
		case prevOp == kql.BoolNOT:
			add = boolQuery.MustNot
		}

		builder, err := k.getBuilder(node)
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

func (k *KQL) getFieldName(name string) string {
	if name == "" {
		return "Name"
	}

	var _fields = map[string]string{
		"rootid":    "RootID",
		"path":      "Path",
		"id":        "ID",
		"name":      "Name",
		"size":      "Size",
		"mtime":     "Mtime",
		"mediatype": "MimeType",
		"type":      "Type",
		"tag":       "Tags",
		"tags":      "Tags",
		"content":   "Content",
		"hidden":    "Hidden",
	}

	switch n, ok := _fields[strings.ToLower(name)]; {
	case ok:
		return n
	default:
		return name
	}
}

func (k *KQL) getOperatorValueAt(nodes []ast.Node, i int) string {
	if i < 0 || i >= len(nodes) {
		return ""
	}

	if opn, ok := nodes[i].(*ast.OperatorNode); ok {
		return opn.Value
	}

	return ""
}

func (k *KQL) getBuilder(node ast.Node) (Builder, error) {
	var builder Builder
	switch node := node.(type) {
	case *ast.BooleanNode:
		builder = NewTermQuery[bool](k.getFieldName(node.Key)).Value(node.Value)
	case *ast.StringNode:
		if strings.Contains(node.Value, "*") {
			builder = NewWildcardQuery(k.getFieldName(node.Key)).Value(node.Value)
			break
		}

		switch len(strings.Split(node.Value, " ")) {
		case 1:
			builder = NewTermQuery[string](k.getFieldName(node.Key)).Value(node.Value)
		default:
			builder = NewMatchPhraseQuery(k.getFieldName(node.Key)).Query(node.Value)
		}
	case *ast.DateTimeNode:
		if node.Operator == nil {
			return builder, fmt.Errorf("date time node without operator: %w", ErrUnsupportedNodeType)
		}

		q := NewRangeQuery[time.Time](k.getFieldName(node.Key))

		switch node.Operator.Value {
		case ">":
			q.Gt(node.Value)
		case ">=":
			q.Gte(node.Value)
		case "<":
			q.Lt(node.Value)
		case "<=":
			q.Lte(node.Value)
		default:
			return nil, fmt.Errorf("unsupported operator %s for date time node: %w", node.Operator.Value, ErrUnsupportedNodeType)
		}

		return q, nil
	case *ast.GroupNode:
		group, err := k.compile(node.Nodes)
		if err != nil {
			return nil, fmt.Errorf("failed to build group: %w", err)
		}
		builder = group
	default:
		return nil, fmt.Errorf("%w: %T", ErrUnsupportedNodeType, node)
	}

	return builder, nil
}
