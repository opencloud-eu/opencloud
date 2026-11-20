package search_test

import (
	"fmt"
	"strings"
	"time"

	sprovider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/opencloud-eu/opencloud/pkg/ast"
	"github.com/opencloud-eu/opencloud/pkg/kql"
	"github.com/opencloud-eu/opencloud/services/search/pkg/search"
)

func parse(query string) *ast.Ast {
	GinkgoHelper()
	a, err := kql.Builder{}.Build(query)
	Expect(err).ToNot(HaveOccurred(), query)
	return a
}

// render prints the tree without its source locations, so two parses of the
// same query compare equal
func render(nodes []ast.Node) string {
	parts := make([]string, 0, len(nodes))
	for _, n := range nodes {
		switch node := n.(type) {
		case *ast.StringNode:
			parts = append(parts, node.Key+":"+node.Value)
		case *ast.BooleanNode:
			parts = append(parts, fmt.Sprintf("%s:%t", node.Key, node.Value))
		case *ast.DateTimeNode:
			parts = append(parts, node.Key+node.Operator.Value+node.Value.Format(time.RFC3339Nano))
		case *ast.NumberNode:
			parts = append(parts, fmt.Sprintf("%s%s%v", node.Key, node.Operator.Value, node.Value))
		case *ast.OperatorNode:
			parts = append(parts, node.Value)
		case *ast.GroupNode:
			parts = append(parts, node.Key+"("+render(node.Nodes)+")")
		}
	}
	return strings.Join(parts, " ")
}

var _ = Describe("QueryTemplate", func() {
	DescribeTable("ParseQueryTemplate",
		func(template string, valid bool) {
			_, err := search.ParseQueryTemplate(template)
			if valid {
				Expect(err).ToNot(HaveOccurred())
			} else {
				Expect(err).To(HaveOccurred())
			}
		},
		Entry("empty", "", true),
		Entry("bare query", `({{.QUERY}})`, true),
		Entry("the default", `({{.QUERY}}) AND (NOT path:"./.space" AND NOT id:{{.SPACE_ROOT_PERSONAL}})`, true),
		Entry("an OR", `({{.QUERY}}) OR mediatype:document`, true),
		Entry("a parenthesis inside a value", `({{.QUERY}}) AND NOT name:"a(b"`, true),
		Entry("query without parentheses", `{{.QUERY}}`, false),
		Entry("query without parentheses in a conjunction", `{{.QUERY}} AND NOT mediatype:folder`, false),
		Entry("query twice", `({{.QUERY}}) OR ({{.QUERY}})`, false),
		Entry("query twice, once bare", `({{.QUERY}}) AND {{.QUERY}}`, false),
		Entry("no query", `NOT mediatype:folder`, false),
		Entry("unknown placeholder", `({{.QUERY}}) AND NOT id:{{.SPACE_ROOT_GRANT}}`, false),
		Entry("broken placeholder", `({{.QUERY}}) AND NOT id:{{.SPACE_ROOT_PERSONAL}`, false),
		Entry("unbalanced parentheses", `(({{.QUERY}})`, false),
		Entry("dangling operators", `({{.QUERY}}) AND AND`, false),
		Entry("trailing operator", `({{.QUERY}}) AND`, false),
	)

	DescribeTable("Expand fills the space roots",
		func(spaceType, expected string) {
			space := &sprovider.StorageSpace{
				SpaceType: spaceType,
				Root:      &sprovider.ResourceId{StorageId: "storageid", SpaceId: "spaceid", OpaqueId: "spaceid"},
			}
			t, err := search.ParseQueryTemplate(`({{.QUERY}}) AND NOT id:{{.SPACE_ROOT_PERSONAL}} AND NOT id:{{.SPACE_ROOT_PROJECT}}`)
			Expect(err).ToNot(HaveOccurred())

			Expect(render(t.Expand(parse("foo"), space).Nodes)).To(Equal(render(parse(expected).Nodes)))
		},
		Entry("a personal space fills the personal placeholder", "personal", `(foo) AND NOT id:"storageid$spaceid!spaceid" AND NOT id:"none$none!none"`),
		Entry("a project space fills the project placeholder", "project", `(foo) AND NOT id:"none$none!none" AND NOT id:"storageid$spaceid!spaceid"`),
		Entry("a share fills neither", "grant", `(foo) AND NOT id:"none$none!none" AND NOT id:"none$none!none"`),
	)

	It("expands a nested query marker", func() {
		t, err := search.ParseQueryTemplate(`(NOT mediatype:folder AND ({{.QUERY}})) OR tag:keep`)
		Expect(err).ToNot(HaveOccurred())

		expanded := t.Expand(parse(`name:x OR name:y`), &sprovider.StorageSpace{})
		Expect(render(expanded.Nodes)).To(Equal(render(parse(`(NOT mediatype:folder AND (name:x OR name:y)) OR tag:keep`).Nodes)))
	})

	It("fills a root placeholder that is used twice", func() {
		t, err := search.ParseQueryTemplate(`({{.QUERY}}) AND NOT id:{{.SPACE_ROOT_PERSONAL}} AND NOT parentid:{{.SPACE_ROOT_PERSONAL}}`)
		Expect(err).ToNot(HaveOccurred())
		space := &sprovider.StorageSpace{SpaceType: "personal", Root: &sprovider.ResourceId{StorageId: "1", SpaceId: "2", OpaqueId: "2"}}

		expanded := t.Expand(parse(`name:x`), space)
		Expect(render(expanded.Nodes)).To(Equal(render(parse(`(name:x) AND NOT id:"1$2!2" AND NOT parentid:"1$2!2"`).Nodes)))
	})

	It("leaves a marker in the user's query alone", func() {
		t, err := search.ParseQueryTemplate(`({{.QUERY}}) AND NOT id:{{.SPACE_ROOT_PERSONAL}}`)
		Expect(err).ToNot(HaveOccurred())

		expanded := t.Expand(parse(`name:opencloudpersonalrootmarker`), &sprovider.StorageSpace{})
		Expect(render(expanded.Nodes)).To(Equal(render(parse(`(name:opencloudpersonalrootmarker) AND NOT id:"none$none!none"`).Nodes)))
	})

	It("keeps the key of the query group", func() {
		t, err := search.ParseQueryTemplate(`name:({{.QUERY}})`)
		Expect(err).ToNot(HaveOccurred())

		expanded := t.Expand(parse(`x OR y`), &sprovider.StorageSpace{})
		Expect(render(expanded.Nodes)).To(Equal(render(parse(`name:(x OR y)`).Nodes)))
	})

	It("expands an empty template to the query itself", func() {
		t, err := search.ParseQueryTemplate("")
		Expect(err).ToNot(HaveOccurred())
		query := parse(`name:x AND mtime>=2024-01-02T03:04:05Z AND size>1 AND hidden:true`)

		Expect(render(t.Expand(query, &sprovider.StorageSpace{}).Nodes)).To(Equal(render(query.Nodes)))
	})

	It("hands out copies the engine may rewrite", func() {
		t, err := search.ParseQueryTemplate(`({{.QUERY}}) AND NOT id:{{.SPACE_ROOT_PERSONAL}}`)
		Expect(err).ToNot(HaveOccurred())
		query := parse(`name:x`)

		first := t.Expand(query, &sprovider.StorageSpace{})
		first.Nodes[0].(*ast.GroupNode).Nodes[0].(*ast.StringNode).Value = "rewritten"
		first.Nodes[3].(*ast.StringNode).Value = "rewritten"

		second := t.Expand(query, &sprovider.StorageSpace{})
		Expect(render(second.Nodes)).To(Equal(render(parse(`(name:x) AND NOT id:"none$none!none"`).Nodes)))
		Expect(render(query.Nodes)).To(Equal("name:x"))
	})
})
