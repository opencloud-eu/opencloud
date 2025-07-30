package opensearch

import (
	"fmt"
	"strings"

	"github.com/opencloud-eu/opencloud/pkg/ast"
	"github.com/opencloud-eu/opencloud/pkg/kql"
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
	case *ast.StringNode:
		switch len(strings.Split(node.Value, " ")) {
		case 1:
			builder = NewTermQuery[string](k.getFieldName(node.Key)).Value(node.Value)
		default:
			builder = NewMatchPhraseQuery(k.getFieldName(node.Key)).Query(node.Value)
		}
	case *ast.GroupNode:
		group, err := k.compile(node.Nodes)
		if err != nil {
			return nil, fmt.Errorf("failed to build group: %w", err)
		}
		builder = group
	}

	return builder, nil
}

func (k *KQL) compile(nodes []ast.Node) (Builder, error) {
	boolQuery := NewBoolQuery()
	add := boolQuery.Must

	for i, node := range nodes {
		prevOp := k.getOperatorValueAt(nodes, i-1)
		nextOp := k.getOperatorValueAt(nodes, i+1)

		switch {
		case nextOp == kql.BoolOR || prevOp == kql.BoolOR:
			add = boolQuery.Should
		case nextOp == kql.BoolAND || prevOp == kql.BoolAND:
			add = boolQuery.Must
		}

		if _, ok := node.(*ast.OperatorNode); ok {
			// operatorNodes are not builders, so we skip them
			continue
		}

		builder, err := k.getBuilder(node)
		if err != nil {
			return nil, fmt.Errorf("failed to get builder for node %T: %w", node, err)
		}

		switch {
		case len(nodes) == 1:
			return builder, nil
		default:
			add(builder)
		}
	}

	return boolQuery, nil
}
