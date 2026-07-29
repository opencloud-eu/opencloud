package query

import (
	"reflect"

	"github.com/opencloud-eu/opencloud/pkg/ast"
	"github.com/opencloud-eu/opencloud/services/search/pkg/query/mimetype"
)

// Normalize is the shared KQL lowering pass between parse and compile: it
// resolves keys to real field names (via resolve) and expands media-type
// restrictions, so the backends compile a plain field:value AST.
func Normalize(a *ast.Ast, resolve func(string) string) *ast.Ast {
	a.Nodes = normalizeNodes(a.Nodes, resolve, "")
	return a
}

// normalizeNodes rewrites nodes in place. defaultKey is what a bare restriction
// inherits: its enclosing group's key, or "" at the top level.
func normalizeNodes(nodes []ast.Node, resolve func(string) string, defaultKey string) []ast.Node {
	resolveKey := func(key string) string {
		if key == "" && defaultKey != "" {
			return defaultKey // bare child inherits the group key
		}
		return resolve(key)
	}

	out := make([]ast.Node, 0, len(nodes))
	for _, n := range nodes {
		n = toPointer(n) // ensure a pointer so in-place key rewrites persist
		switch node := n.(type) {
		case *ast.StringNode:
			node.Key = resolveKey(node.Key)
			if exp := mimetype.Expand(node.Key, node.Value); exp != nil {
				out = append(out, normalizeNodes(exp, resolve, defaultKey)...)
				continue
			}
			out = append(out, node)
		case *ast.DateTimeNode:
			node.Key = resolveKey(node.Key)
			out = append(out, node)
		case *ast.BooleanNode:
			node.Key = resolveKey(node.Key)
			out = append(out, node)
		case *ast.GroupNode:
			groupKey := defaultKey
			if node.Key != "" {
				node.Key = resolve(node.Key)
				groupKey = node.Key
			}
			node.Nodes = normalizeNodes(node.Nodes, resolve, groupKey)
			out = append(out, node)
		default:
			out = append(out, n)
		}
	}
	return out
}

// toPointer returns n as a pointer; the parser emits some nodes by value and the
// in-place key rewrites would be lost on those.
func toPointer(n ast.Node) ast.Node {
	rv := reflect.ValueOf(n)
	if rv.Kind() == reflect.Ptr {
		return n
	}
	ptr := reflect.New(rv.Type())
	ptr.Elem().Set(rv)
	if pn, ok := ptr.Interface().(ast.Node); ok {
		return pn
	}
	return n
}
