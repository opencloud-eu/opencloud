package mimetype_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/opencloud-eu/opencloud/pkg/ast"
	"github.com/opencloud-eu/opencloud/services/search/pkg/query/mimetype"
)

// This is the single place the mediatype -> MimeType mapping is tested. The
// query pipeline consumes Expand via query.Normalize and must NOT re-test it.

func TestExpand_onlyTriggersOnMediatype(t *testing.T) {
	require.Nil(t, mimetype.Expand("Name", "document"))
	require.Nil(t, mimetype.Expand("MimeType", "file")) // the real field name is not the trigger
	require.Nil(t, mimetype.Expand("Tags", "file"))
}

func TestExpand_keyIsCaseInsensitive(t *testing.T) {
	require.NotNil(t, mimetype.Expand("MediaType", "file"))
}

// A non-category value is a literal MIME type and targets the MimeType field.
func TestExpand_literalValuePassesThroughToMimeType(t *testing.T) {
	require.Equal(t, []ast.Node{
		&ast.StringNode{Key: "MimeType", Value: "application/pdf"},
	}, mimetype.Expand("mediatype", "application/pdf"))

	require.Equal(t, []ast.Node{
		&ast.StringNode{Key: "MimeType", Value: "image/jpeg"},
	}, mimetype.Expand("mediatype", "image/jpeg"))
}

func TestExpand_fileIsNotAFolder(t *testing.T) {
	require.Equal(t, []ast.Node{
		&ast.OperatorNode{Value: "NOT"},
		&ast.StringNode{Key: "MimeType", Value: "httpd/unix-directory"},
	}, mimetype.Expand("mediatype", "file"))
}

func TestExpand_folderIsASingleTerm(t *testing.T) {
	require.Equal(t, []ast.Node{
		&ast.StringNode{Key: "MimeType", Value: "httpd/unix-directory"},
	}, mimetype.Expand("mediatype", "folder"))
}

func TestExpand_wildcardCategories(t *testing.T) {
	for value, mime := range map[string]string{
		"image": "image/*", "video": "video/*", "audio": "audio/*", "pdf": "application/pdf",
	} {
		require.Equal(t, []ast.Node{
			&ast.StringNode{Key: "MimeType", Value: mime},
		}, mimetype.Expand("mediatype", value), value)
	}
}

func TestExpand_documentGroup(t *testing.T) {
	got := mimetype.Expand("mediatype", "document")
	require.Len(t, got, 1)
	group, ok := got[0].(*ast.GroupNode)
	require.True(t, ok)
	require.Equal(t, mimeValues(group), []string{
		"application/msword",
		"application/vnd.openxmlformats-officedocument.wordprocessingml.document",
		"application/vnd.openxmlformats-officedocument.wordprocessingml.form",
		"application/vnd.oasis.opendocument.text",
		"text/plain",
		"text/markdown",
		"application/rtf",
		"application/vnd.apple.pages",
	})
}

// spreadsheet asserts the exact MIME set, in order, with no duplicate entry.
func TestExpand_spreadsheet(t *testing.T) {
	group := mimetype.Expand("mediatype", "spreadsheet")[0].(*ast.GroupNode)
	require.Equal(t, mimeValues(group), []string{
		"application/vnd.ms-excel",
		"application/vnd.oasis.opendocument.spreadsheet",
		"text/csv",
		"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
		"application/vnd.apple.numbers",
	})
}

// mimeValues extracts the StringNode values from an OR group, dropping operators.
func mimeValues(group *ast.GroupNode) []string {
	var out []string
	for _, n := range group.Nodes {
		if s, ok := n.(*ast.StringNode); ok {
			out = append(out, s.Value)
		}
	}
	return out
}
