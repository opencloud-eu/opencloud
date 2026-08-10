package mimetype_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/opencloud-eu/opencloud/pkg/ast"
	"github.com/opencloud-eu/opencloud/services/search/pkg/query/mimetype"
)

// This is the single place the mediatype -> MimeType mapping is tested. The
// query pipeline consumes Expand via query.Normalize and must NOT re-test it.

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

var _ = Describe("Expand", func() {
	It("only triggers on the mediatype key", func() {
		Expect(mimetype.Expand("Name", "document")).To(BeNil())
		Expect(mimetype.Expand("MimeType", "file")).To(BeNil()) // the real field name is not the trigger
		Expect(mimetype.Expand("Tags", "file")).To(BeNil())
	})

	It("matches the key case-insensitively", func() {
		Expect(mimetype.Expand("MediaType", "file")).ToNot(BeNil())
	})

	It("matches the value case-insensitively", func() {
		// a category matches regardless of case
		Expect(mimetype.Expand("mediatype", "Folder")).To(Equal([]ast.Node{
			&ast.StringNode{Key: "MimeType", Value: "httpd/unix-directory"},
		}))
		// a literal MIME type is lowercased too (MIME types are case-insensitive)
		Expect(mimetype.Expand("mediatype", "Image/SVG+XML")).To(Equal([]ast.Node{
			&ast.StringNode{Key: "MimeType", Value: "image/svg+xml"},
		}))
	})

	// A non-category value is a literal MIME type and targets the MimeType field.
	It("passes literal values through to MimeType", func() {
		Expect(mimetype.Expand("mediatype", "application/pdf")).To(Equal([]ast.Node{
			&ast.StringNode{Key: "MimeType", Value: "application/pdf"},
		}))
		Expect(mimetype.Expand("mediatype", "image/jpeg")).To(Equal([]ast.Node{
			&ast.StringNode{Key: "MimeType", Value: "image/jpeg"},
		}))
	})

	It("expands file to not-a-folder", func() {
		// grouped so the negation stays atomic next to an operator.
		Expect(mimetype.Expand("mediatype", "file")).To(Equal([]ast.Node{
			&ast.GroupNode{Nodes: []ast.Node{
				&ast.OperatorNode{Value: "NOT"},
				&ast.StringNode{Key: "MimeType", Value: "httpd/unix-directory"},
			}},
		}))
	})

	It("expands folder to a single term", func() {
		Expect(mimetype.Expand("mediatype", "folder")).To(Equal([]ast.Node{
			&ast.StringNode{Key: "MimeType", Value: "httpd/unix-directory"},
		}))
	})

	It("expands wildcard categories", func() {
		for value, mime := range map[string]string{
			"image": "image/*", "video": "video/*", "audio": "audio/*", "pdf": "application/pdf",
		} {
			Expect(mimetype.Expand("mediatype", value)).To(Equal([]ast.Node{
				&ast.StringNode{Key: "MimeType", Value: mime},
			}), value)
		}
	})

	It("expands the document group", func() {
		got := mimetype.Expand("mediatype", "document")
		Expect(got).To(HaveLen(1))
		group, ok := got[0].(*ast.GroupNode)
		Expect(ok).To(BeTrue())
		Expect(mimeValues(group)).To(Equal([]string{
			"application/msword",
			"application/vnd.openxmlformats-officedocument.wordprocessingml.document",
			"application/vnd.openxmlformats-officedocument.wordprocessingml.form",
			"application/vnd.oasis.opendocument.text",
			"text/plain",
			"text/markdown",
			"application/rtf",
			"application/vnd.apple.pages",
		}))
	})

	// spreadsheet asserts the exact MIME set, in order, with no duplicate entry.
	It("expands the spreadsheet group", func() {
		group := mimetype.Expand("mediatype", "spreadsheet")[0].(*ast.GroupNode)
		Expect(mimeValues(group)).To(Equal([]string{
			"application/vnd.ms-excel",
			"application/vnd.oasis.opendocument.spreadsheet",
			"text/csv",
			"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
			"application/vnd.apple.numbers",
		}))
	})
})
