// Package mimetype maps the "mediatype" KQL restriction (field name and value)
// to a concrete MimeType query.
package mimetype

import (
	"strings"

	"github.com/opencloud-eu/opencloud/pkg/ast"
	"github.com/opencloud-eu/opencloud/pkg/kql"
)

// field is the real index field a mediatype restriction targets.
const field = "MimeType"

// Expand turns mediatype:<value> into the MimeType query it stands for: category
// values (file/document/image/...) expand to their MIME set, anything else is a
// literal MimeType:<value>. Returns nil for non-mediatype keys.
func Expand(key, value string) []ast.Node {
	if strings.ToLower(key) != "mediatype" {
		return nil
	}
	switch value {
	case "file":
		return []ast.Node{
			&ast.OperatorNode{Value: kql.BoolNOT},
			&ast.StringNode{Key: field, Value: "httpd/unix-directory"},
		}
	case "folder":
		return term("httpd/unix-directory")
	case "document":
		return group(
			"application/msword",
			"application/vnd.openxmlformats-officedocument.wordprocessingml.document",
			"application/vnd.openxmlformats-officedocument.wordprocessingml.form",
			"application/vnd.oasis.opendocument.text",
			"text/plain",
			"text/markdown",
			"application/rtf",
			"application/vnd.apple.pages",
		)
	case "spreadsheet":
		return group(
			"application/vnd.ms-excel",
			"application/vnd.oasis.opendocument.spreadsheet",
			"text/csv",
			"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
			"application/vnd.apple.numbers",
		)
	case "presentation":
		return group(
			"application/vnd.openxmlformats-officedocument.presentationml.presentation",
			"application/vnd.oasis.opendocument.presentation",
			"application/vnd.ms-powerpoint",
			"application/vnd.apple.keynote",
		)
	case "pdf":
		return term("application/pdf")
	case "image":
		return term("image/*")
	case "video":
		return term("video/*")
	case "audio":
		return term("audio/*")
	case "archive":
		return group(
			"application/zip",
			"application/gzip",
			"application/x-gzip",
			"application/x-7z-compressed",
			"application/x-rar-compressed",
			"application/x-tar",
			"application/x-bzip2",
			"application/x-bzip",
			"application/x-tgz",
		)
	}
	// not a category: treat the value as a literal MIME type.
	return term(value)
}

// term is a single MimeType:value restriction.
func term(value string) []ast.Node {
	return []ast.Node{&ast.StringNode{Key: field, Value: value}}
}

// group is a single OR group of MimeType:value restrictions.
func group(values ...string) []ast.Node {
	nodes := make([]ast.Node, 0, len(values)*2-1)
	for i, v := range values {
		if i > 0 {
			nodes = append(nodes, &ast.OperatorNode{Value: kql.BoolOR})
		}
		nodes = append(nodes, &ast.StringNode{Key: field, Value: v})
	}
	return []ast.Node{&ast.GroupNode{Nodes: nodes}}
}
