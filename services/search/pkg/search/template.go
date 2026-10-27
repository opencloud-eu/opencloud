package search

import (
	"errors"
	"fmt"
	"strings"

	provider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	"github.com/opencloud-eu/reva/v2/pkg/storagespace"

	"github.com/opencloud-eu/opencloud/pkg/ast"
	"github.com/opencloud-eu/opencloud/pkg/kql"
	"github.com/opencloud-eu/opencloud/services/search/pkg/config"
)

// QueryTemplate is the parsed SEARCH_QUERY_TEMPLATE: a KQL tree with marker
// nodes where the user's query and the space roots go.
type QueryTemplate struct {
	nodes    []ast.Node
	query    *ast.GroupNode
	personal *ast.StringNode
	project  *ast.StringNode
}

const (
	queryMarker        = "opencloudquerymarker"
	personalRootMarker = "opencloudpersonalrootmarker"
	projectRootMarker  = "opencloudprojectrootmarker"
)

// ParseQueryTemplate parses the template once. An empty template is allowed
// and expands to the bare query.
func ParseQueryTemplate(template string) (*QueryTemplate, error) {
	if template == "" {
		return nil, nil
	}

	if strings.Count(template, config.QueryPlaceholder) != 1 {
		return nil, errors.New("search query template: must contain " + config.QueryPlaceholder + " exactly once")
	}
	wrapped := "(" + config.QueryPlaceholder + ")"
	if !strings.Contains(template, wrapped) {
		return nil, errors.New("search query template: " + config.QueryPlaceholder + " must be wrapped as " + wrapped)
	}

	source := strings.NewReplacer(
		config.QueryPlaceholder, queryMarker,
		config.PersonalSpaceRootPlaceholder, `"`+personalRootMarker+`"`,
		config.ProjectSpaceRootPlaceholder, `"`+projectRootMarker+`"`,
	).Replace(template)
	if strings.Contains(source, "{{") || strings.Contains(source, "}}") {
		return nil, errors.New("search query template: unknown placeholder")
	}

	parsed, err := kql.Builder{}.Build(source)
	if err != nil {
		return nil, fmt.Errorf("search query template: %w", err)
	}

	t := &QueryTemplate{nodes: parsed.Nodes}
	t.findMarkers(parsed.Nodes)
	if t.query == nil {
		return nil, errors.New("search query template: " + config.QueryPlaceholder + " must be wrapped as " + wrapped)
	}
	return t, nil
}

func (t *QueryTemplate) findMarkers(nodes []ast.Node) {
	for _, n := range nodes {
		switch node := n.(type) {
		case *ast.GroupNode:
			if len(node.Nodes) == 1 {
				if s, ok := node.Nodes[0].(*ast.StringNode); ok && s.Key == "" && s.Value == queryMarker {
					t.query = node
					continue
				}
			}
			t.findMarkers(node.Nodes)
		case *ast.StringNode:
			switch node.Value {
			case personalRootMarker:
				t.personal = node
			case projectRootMarker:
				t.project = node
			}
		}
	}
}

// Expand builds the query the engine answers for one space: the template with
// the user's query and the space roots filled in. Every node is a copy, so the
// engine may rewrite the result and the inputs stay untouched.
func (t *QueryTemplate) Expand(query *ast.Ast, space *provider.StorageSpace) *ast.Ast {
	if t == nil {
		return &ast.Ast{Base: query.Base, Nodes: cloneNodes(query.Nodes, nil)}
	}

	rootID := func(spaceType string) string {
		if space.GetSpaceType() == spaceType {
			return storagespace.FormatResourceID(space.GetRoot())
		}
		return config.NoResourceID
	}

	return &ast.Ast{Nodes: cloneNodes(t.nodes, func(n ast.Node) ast.Node {
		switch n {
		case t.query:
			return &ast.GroupNode{Base: t.query.Base, Nodes: cloneNodes(query.Nodes, nil)}
		case t.personal:
			c := *t.personal
			c.Value = rootID(_spaceTypePersonal)
			return &c
		case t.project:
			c := *t.project
			c.Value = rootID(_spaceTypeProject)
			return &c
		}
		return nil
	})}
}

// cloneNodes deep-copies the nodes; replace may hand back a substitute for a
// node, which is taken as is.
func cloneNodes(nodes []ast.Node, replace func(ast.Node) ast.Node) []ast.Node {
	out := make([]ast.Node, 0, len(nodes))
	for _, n := range nodes {
		if replace != nil {
			if r := replace(n); r != nil {
				out = append(out, r)
				continue
			}
		}
		switch node := n.(type) {
		case *ast.StringNode:
			c := *node
			out = append(out, &c)
		case *ast.BooleanNode:
			c := *node
			out = append(out, &c)
		case *ast.DateTimeNode:
			c := *node
			c.Operator = cloneOperator(node.Operator)
			out = append(out, &c)
		case *ast.NumberNode:
			c := *node
			c.Operator = cloneOperator(node.Operator)
			out = append(out, &c)
		case *ast.OperatorNode:
			out = append(out, cloneOperator(node))
		case *ast.GroupNode:
			c := *node
			c.Nodes = cloneNodes(node.Nodes, replace)
			out = append(out, &c)
		default:
			out = append(out, n)
		}
	}
	return out
}

func cloneOperator(op *ast.OperatorNode) *ast.OperatorNode {
	if op == nil {
		return nil
	}
	c := *op
	return &c
}
