package query_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/opencloud-eu/opencloud/pkg/ast"
	"github.com/opencloud-eu/opencloud/services/search/pkg/query"
)

// This is the single place the shared KQL lowering pass is tested (field
// resolution, media-type expansion, group-key defaulting, pointer conversion).
// The backend query compilers consume its canonical output and must not re-test
// it.

func norm(nodes ...ast.Node) []ast.Node {
	return query.Normalize(&ast.Ast{Nodes: nodes}, query.ResolveField).Nodes
}

func TestResolveField(t *testing.T) {
	require.Equal(t, "Name", query.ResolveField(""))          // empty -> free-text default
	require.Equal(t, "Name", query.ResolveField("NAME"))      // case-insensitive
	require.Equal(t, "Tags", query.ResolveField("tag"))       // singular alias
	require.Equal(t, "MimeType", query.ResolveField("mimetype"))                 // real field, case-insensitive
	require.Equal(t, "photo.cameraMake", query.ResolveField("photo.CAMERAMAKE")) // facet, case-insensitive
	require.Equal(t, "unknown.field", query.ResolveField("unknown.field"))       // unknown key: unchanged, becomes a dead query
}

func TestNormalize_ResolvesFieldsAndExpandsMediatype(t *testing.T) {
	got := norm(
		&ast.StringNode{Key: "", Value: "free"},
		&ast.OperatorNode{Value: "AND"},
		&ast.StringNode{Key: "TAG", Value: "x"},
		&ast.OperatorNode{Value: "AND"},
		&ast.StringNode{Key: "photo.cameramake", Value: "canon"},
		&ast.OperatorNode{Value: "AND"},
		&ast.StringNode{Key: "mediatype", Value: "file"},
	)
	require.Equal(t, []ast.Node{
		&ast.StringNode{Key: "Name", Value: "free"},
		&ast.OperatorNode{Value: "AND"},
		&ast.StringNode{Key: "Tags", Value: "x"},
		&ast.OperatorNode{Value: "AND"},
		&ast.StringNode{Key: "photo.cameraMake", Value: "canon"},
		&ast.OperatorNode{Value: "AND"},
		&ast.OperatorNode{Value: "NOT"},
		&ast.StringNode{Key: "MimeType", Value: "httpd/unix-directory"},
	}, got)
}

// A bare restriction inside a named group inherits the group key; a keyed child
// keeps its own key; a bare restriction in an unnamed group falls back to Name.
func TestNormalize_GroupKeyDefaulting(t *testing.T) {
	got := norm(
		&ast.GroupNode{Key: "author", Nodes: []ast.Node{
			&ast.StringNode{Value: "b"},
			&ast.OperatorNode{Value: "OR"},
			&ast.StringNode{Key: "name", Value: "d"},
		}},
		&ast.OperatorNode{Value: "AND"},
		&ast.GroupNode{Nodes: []ast.Node{
			&ast.StringNode{Value: "e"},
		}},
	)
	require.Equal(t, []ast.Node{
		&ast.GroupNode{Key: "author", Nodes: []ast.Node{
			&ast.StringNode{Key: "author", Value: "b"},
			&ast.OperatorNode{Value: "OR"},
			&ast.StringNode{Key: "Name", Value: "d"},
		}},
		&ast.OperatorNode{Value: "AND"},
		&ast.GroupNode{Nodes: []ast.Node{
			&ast.StringNode{Key: "Name", Value: "e"},
		}},
	}, got)
}

func TestNormalize_ConvertsValueNodesToPointers(t *testing.T) {
	got := norm(
		ast.StringNode{Key: "name", Value: "x"},
		ast.OperatorNode{Value: "AND"},
		ast.DateTimeNode{Key: "mtime"},
	)
	require.Equal(t, []ast.Node{
		&ast.StringNode{Key: "Name", Value: "x"},
		&ast.OperatorNode{Value: "AND"},
		&ast.DateTimeNode{Key: "Mtime"},
	}, got)
}
