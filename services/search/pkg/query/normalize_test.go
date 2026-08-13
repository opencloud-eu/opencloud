package query_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

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

var _ = Describe("ResolveField", func() {
	It("resolves keys to canonical field names", func() {
		Expect(query.ResolveField("")).To(Equal("Name"))                             // empty -> free-text default
		Expect(query.ResolveField("NAME")).To(Equal("Name"))                         // canonical, case-insensitive key match
		Expect(query.ResolveField("tag")).To(Equal("Tags"))                          // singular alias
		Expect(query.ResolveField("mimetype")).To(Equal("MimeType"))                 // real field
		Expect(query.ResolveField("photo.CAMERAMAKE")).To(Equal("photo.cameraMake")) // facet, case-insensitive key match
		Expect(query.ResolveField("unknown.field")).To(Equal("unknown.field"))       // unknown key: unchanged, becomes a dead query
	})
})

var _ = Describe("FieldIsCaseInsensitive", func() {
	It("reports the CaseInsensitive override fields", func() {
		// The three CaseInsensitive override fields (resolved canonical names).
		for _, f := range []string{"Name", "Tags", "Favorites"} {
			Expect(query.FieldIsCaseInsensitive(f)).To(BeTrue(), f)
		}
		// Case-preserved / non-keyword fields are not.
		for _, f := range []string{"MimeType", "ID", "Content", "Path", "unknown"} {
			Expect(query.FieldIsCaseInsensitive(f)).To(BeFalse(), f)
		}
	})
})

var _ = Describe("Normalize", func() {
	It("resolves fields and expands mediatype", func() {
		got := norm(
			&ast.StringNode{Key: "", Value: "free"},
			&ast.OperatorNode{Value: "AND"},
			&ast.StringNode{Key: "TAG", Value: "x"},
			&ast.OperatorNode{Value: "AND"},
			&ast.StringNode{Key: "photo.cameramake", Value: "canon"},
			&ast.OperatorNode{Value: "AND"},
			&ast.StringNode{Key: "mediatype", Value: "file"},
		)
		Expect(got).To(Equal([]ast.Node{
			&ast.StringNode{Key: "Name", Value: "free", CaseInsensitive: true},
			&ast.OperatorNode{Value: "AND"},
			&ast.StringNode{Key: "Tags", Value: "x", CaseInsensitive: true},
			&ast.OperatorNode{Value: "AND"},
			&ast.StringNode{Key: "photo.cameraMake", Value: "canon"},
			&ast.OperatorNode{Value: "AND"},
			&ast.OperatorNode{Value: "NOT"},
			&ast.StringNode{Key: "MimeType", Value: "httpd/unix-directory"},
		}))
	})

	It("resolves driveId to RootID and completes the root id", func() {
		got := norm(
			&ast.StringNode{Key: "driveId", Value: "1$2"},
			&ast.OperatorNode{Value: "AND"},
			&ast.StringNode{Key: "driveid", Value: "1$2!4"},
			&ast.OperatorNode{Value: "AND"},
			&ast.StringNode{Key: "RootID", Value: "1$2"},
		)
		Expect(got).To(Equal([]ast.Node{
			&ast.StringNode{Key: "RootID", Value: "1$2!2"},
			&ast.OperatorNode{Value: "AND"},
			&ast.StringNode{Key: "RootID", Value: "1$2!4"},
			&ast.OperatorNode{Value: "AND"},
			&ast.StringNode{Key: "RootID", Value: "1$2!2"},
		}))
	})

	// A bare restriction inside a named group inherits the group key; a keyed
	// child keeps its own key; a bare restriction in an unnamed group falls
	// back to Name.
	It("defaults group keys", func() {
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
		Expect(got).To(Equal([]ast.Node{
			&ast.GroupNode{Key: "author", Nodes: []ast.Node{
				&ast.StringNode{Key: "author", Value: "b"},
				&ast.OperatorNode{Value: "OR"},
				&ast.StringNode{Key: "Name", Value: "d", CaseInsensitive: true},
			}},
			&ast.OperatorNode{Value: "AND"},
			&ast.GroupNode{Nodes: []ast.Node{
				&ast.StringNode{Key: "Name", Value: "e", CaseInsensitive: true},
			}},
		}))
	})

	It("converts value nodes to pointers", func() {
		got := norm(
			ast.StringNode{Key: "name", Value: "x"},
			ast.OperatorNode{Value: "AND"},
			ast.DateTimeNode{Key: "mtime"},
		)
		Expect(got).To(Equal([]ast.Node{
			&ast.StringNode{Key: "Name", Value: "x", CaseInsensitive: true},
			&ast.OperatorNode{Value: "AND"},
			&ast.DateTimeNode{Key: "Mtime"},
		}))
	})
})
