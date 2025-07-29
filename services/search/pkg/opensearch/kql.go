package opensearch

import (
	"errors"
	"strings"

	"github.com/opencloud-eu/opencloud/pkg/ast"
)

type KQL struct{}

func (k KQL) Compile(a *ast.Ast) (*RootQuery, error) {
	switch {
	case len(a.Nodes) == 0:
		return nil, errors.New("no nodes in AST")
	case len(a.Nodes) == 1:
		builder, err := k.getBuilder(a.Nodes[0])
		if err != nil {
			return nil, err
		}
		return NewRootQuery(builder), nil
	}

	return nil, nil
}

func (k KQL) getBuilder(someNode ast.Node) (Builder, error) {
	var query Builder
	switch node := someNode.(type) {
	case *ast.StringNode:
		field := k.mapField(node.Key)
		switch spaces := strings.Split(node.Value, " "); {
		case len(spaces) == 1:
			query = NewTermQuery[string](field).Value(node.Value)
		case len(spaces) > 1:
			query = NewMatchPhraseQuery(field).Query(node.Value)
		}
	}

	return query, nil
}

func (k KQL) mapField(field string) string {
	if field == "" {
		return "Name"
	}

	mappings := map[string]string{
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

	if mapped, ok := mappings[strings.ToLower(field)]; ok {
		return mapped
	}

	return field
}
