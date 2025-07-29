package opensearch

import (
	"strings"

	"github.com/opencloud-eu/opencloud/pkg/ast"
)

type KQL struct{}

func (k KQL) Compile(givenAst *ast.Ast) (Builder, error) {
	q, err := k.compile(givenAst)
	if err != nil {
		return nil, err
	}
	return q, nil
}

func (k KQL) compile(a *ast.Ast) (Builder, error) {
	q, _, err := k.walk(0, a.Nodes)
	if err != nil {
		return nil, err
	}

	return q, nil
}

func (k KQL) walk(offset int, nodes []ast.Node) (Builder, int, error) {
	var boolQuery = NewBoolQuery()
	for i := offset; i < len(nodes); i++ {
		switch n := nodes[i].(type) {
		case *ast.StringNode:
			field := k.getField(n.Key)

			switch spaces := strings.Split(n.Value, " "); {
			case len(spaces) == 1:
				boolQuery.Must(NewTermQuery[string](field).Value(n.Value))
			case len(spaces) > 1:
				boolQuery.Must(NewMatchPhraseQuery(field).Query(n.Value))
			default:
				continue
			}
		case *ast.OperatorNode:
		}

	}

	return boolQuery, 0, nil
}

func (k KQL) getField(name string) string {
	if name == "" {
		return "Name"
	}

	fields := map[string]string{
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

	if _, ok := fields[strings.ToLower(name)]; ok {
		return fields[strings.ToLower(name)]
	}

	return name
}
