package convert

import (
	"fmt"
	"reflect"
	"slices"
	"strings"

	"github.com/opencloud-eu/opencloud/pkg/ast"
)

func ExpandKQL(nodes []ast.Node) ([]ast.Node, error) {
	return kqlExpander{}.expand(nodes, "")
}

type kqlExpander struct{}

func (e kqlExpander) expand(nodes []ast.Node, defaultKey string) ([]ast.Node, error) {
	for i, node := range nodes {
		rnode := reflect.ValueOf(node)

		// we need to ensure that the node is a pointer to an ast.Node in every case
		if rnode.Kind() != reflect.Ptr {
			ptr := reflect.New(rnode.Type())
			ptr.Elem().Set(rnode)
			rnode = ptr
			cnode, ok := rnode.Interface().(ast.Node)
			if !ok {
				return nil, fmt.Errorf("expected node to be of type ast.Node, got %T", rnode.Interface())
			}

			node = cnode    // Update the original node to the pointer
			nodes[i] = node // Update the original slice with the pointer
		}

		var unfoldedNodes []ast.Node
		switch cnode := node.(type) {
		case *ast.GroupNode:
			if cnode.Key != "" { // group nodes should not get a default key
				cnode.Key = e.remapKey(cnode.Key, defaultKey)
			}

			groupNodes, err := e.expand(cnode.Nodes, cnode.Key)
			if err != nil {
				return nil, err
			}
			cnode.Nodes = groupNodes
		case *ast.StringNode:
			cnode.Key = e.remapKey(cnode.Key, defaultKey)
			cnode.Value = e.lowerValue(cnode.Key, cnode.Value)
			unfoldedNodes = e.unfoldValue(cnode.Key, cnode.Value)
		case *ast.DateTimeNode:
			cnode.Key = e.remapKey(cnode.Key, defaultKey)
		case *ast.BooleanNode:
			cnode.Key = e.remapKey(cnode.Key, defaultKey)
		}

		if unfoldedNodes != nil {
			// Insert unfolded nodes at the current index
			nodes = append(nodes[:i], append(unfoldedNodes, nodes[i+1:]...)...)
			// Adjust index to account for new nodes
			i += len(unfoldedNodes) - 1
		}
	}

	return nodes, nil
}

func (_ kqlExpander) remapKey(current string, defaultKey string) string {
	if defaultKey == "" {
		defaultKey = "Name" // Set a default key if none is provided
	}

	key, ok := map[string]string{
		"":          defaultKey, // Default case if current is empty
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
	}[current]
	if !ok {
		return current // Return the original key if not found
	}

	return key
}

func (_ kqlExpander) lowerValue(key, value string) string {
	if slices.Contains([]string{"Hidden"}, key) {
		return value // ignore certain keys and return the original value
	}

	return strings.ToLower(value)
}

func (_ kqlExpander) unfoldValue(key, value string) []ast.Node {
	result, ok := map[string][]ast.Node{
		"MimeType:file": {
			&ast.OperatorNode{Value: "NOT"},
			&ast.StringNode{Key: key, Value: "httpd/unix-directory"},
		},
		"MimeType:folder": {
			&ast.StringNode{Key: key, Value: "httpd/unix-directory"},
		},
		"MimeType:document": {
			&ast.GroupNode{Nodes: []ast.Node{
				&ast.StringNode{Key: key, Value: "application/msword"},
				&ast.OperatorNode{Value: "OR"},
				&ast.StringNode{Key: key, Value: "application/vnd.openxmlformats-officedocument.wordprocessingml.document"},
				&ast.OperatorNode{Value: "OR"},
				&ast.StringNode{Key: key, Value: "application/vnd.openxmlformats-officedocument.wordprocessingml.form"},
				&ast.OperatorNode{Value: "OR"},
				&ast.StringNode{Key: key, Value: "application/vnd.oasis.opendocument.text"},
				&ast.OperatorNode{Value: "OR"},
				&ast.StringNode{Key: key, Value: "text/plain"},
				&ast.OperatorNode{Value: "OR"},
				&ast.StringNode{Key: key, Value: "text/markdown"},
				&ast.OperatorNode{Value: "OR"},
				&ast.StringNode{Key: key, Value: "application/rtf"},
				&ast.OperatorNode{Value: "OR"},
				&ast.StringNode{Key: key, Value: "application/vnd.apple.pages"},
			}},
		},
		"MimeType:spreadsheet": {
			&ast.GroupNode{Nodes: []ast.Node{
				&ast.StringNode{Key: key, Value: "application/vnd.ms-excel"},
				&ast.OperatorNode{Value: "OR"},
				&ast.StringNode{Key: key, Value: "application/vnd.oasis.opendocument.spreadsheet"},
				&ast.OperatorNode{Value: "OR"},
				&ast.StringNode{Key: key, Value: "text/csv"},
				&ast.OperatorNode{Value: "OR"},
				&ast.StringNode{Key: key, Value: "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"},
				&ast.OperatorNode{Value: "OR"},
				&ast.StringNode{Key: key, Value: "application/vnd.oasis.opendocument.spreadshee"},
				&ast.OperatorNode{Value: "OR"},
				&ast.StringNode{Key: key, Value: "application/vnd.apple.numbers"},
			}},
		},
		"MimeType:presentation": {
			&ast.GroupNode{Nodes: []ast.Node{
				&ast.StringNode{Key: key, Value: "application/vnd.openxmlformats-officedocument.presentationml.presentation"},
				&ast.OperatorNode{Value: "OR"},
				&ast.StringNode{Key: key, Value: "application/vnd.oasis.opendocument.presentation"},
				&ast.OperatorNode{Value: "OR"},
				&ast.StringNode{Key: key, Value: "application/vnd.ms-powerpoint"},
				&ast.OperatorNode{Value: "OR"},
				&ast.StringNode{Key: key, Value: "application/vnd.apple.keynote"},
			}},
		},
		"MimeType:pdf": {
			&ast.StringNode{Key: key, Value: "application/pdf"},
		},
		"MimeType:image": {
			&ast.StringNode{Key: key, Value: "image/*"},
		},
		"MimeType:video": {
			&ast.StringNode{Key: key, Value: "video/*"},
		},
		"MimeType:audio": {
			&ast.StringNode{Key: key, Value: "audio/*"},
		},
		"MimeType:archive": {
			&ast.GroupNode{Nodes: []ast.Node{
				&ast.StringNode{Key: key, Value: "application/zip"},
				&ast.OperatorNode{Value: "OR"},
				&ast.StringNode{Key: key, Value: "application/gzip"},
				&ast.OperatorNode{Value: "OR"},
				&ast.StringNode{Key: key, Value: "application/x-gzip"},
				&ast.OperatorNode{Value: "OR"},
				&ast.StringNode{Key: key, Value: "application/x-7z-compressed"},
				&ast.OperatorNode{Value: "OR"},
				&ast.StringNode{Key: key, Value: "application/x-rar-compressed"},
				&ast.OperatorNode{Value: "OR"},
				&ast.StringNode{Key: key, Value: "application/x-tar"},
				&ast.OperatorNode{Value: "OR"},
				&ast.StringNode{Key: key, Value: "application/x-bzip2"},
				&ast.OperatorNode{Value: "OR"},
				&ast.StringNode{Key: key, Value: "application/x-bzip"},
				&ast.OperatorNode{Value: "OR"},
				&ast.StringNode{Key: key, Value: "application/x-tgz"},
			}},
		},
	}[fmt.Sprintf("%s:%s", key, value)]
	if !ok {
		return nil
	}

	return result
}
