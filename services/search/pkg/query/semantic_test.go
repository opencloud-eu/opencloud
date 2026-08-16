package query_test

import (
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/opencloud-eu/opencloud/pkg/ast"
	"github.com/opencloud-eu/opencloud/pkg/kql"
	"github.com/opencloud-eu/opencloud/services/search/pkg/query"
)

// render flattens a node sequence into comparable tokens.
func render(nodes []ast.Node) []string {
	out := make([]string, 0, len(nodes))
	for _, n := range nodes {
		switch node := n.(type) {
		case *ast.StringNode:
			out = append(out, node.Key+":"+node.Value)
		case *ast.OperatorNode:
			out = append(out, node.Value)
		case *ast.GroupNode:
			out = append(out, "("+node.Key, fmt.Sprint(render(node.Nodes)), ")")
		default:
			out = append(out, fmt.Sprintf("%T", n))
		}
	}
	return out
}

var _ = DescribeTable("ExtractSemantic",
	func(input, wantText string, wantRemaining []string) {
		parsed, err := kql.Builder{}.Build(input)
		Expect(err).ToNot(HaveOccurred())

		text := query.ExtractSemantic(parsed)

		Expect(text).To(Equal(wantText))
		Expect(render(parsed.Nodes)).To(Equal(wantRemaining))
	},
	Entry("no semantic part",
		`mediatype:image AND Tags:foo`,
		``,
		[]string{"mediatype:image", "AND", "Tags:foo"},
	),
	Entry("purely semantic",
		`semantic:"Hund am Strand"`,
		`Hund am Strand`,
		[]string{},
	),
	Entry("unquoted single word",
		`semantic:Kirche`,
		`Kirche`,
		[]string{},
	),
	Entry("case-insensitive key",
		`Semantic:"Meer"`,
		`Meer`,
		[]string{},
	),
	Entry("semantic combined with a filter",
		`semantic:"Kirche" AND Tags:foo`,
		`Kirche`,
		[]string{"Tags:foo"},
	),
	Entry("semantic in the middle of the query",
		`mediatype:image AND semantic:"Meer" AND Tags:urlaub`,
		`Meer`,
		[]string{"mediatype:image", "AND", "Tags:urlaub"},
	),
	Entry("group that only held the semantic clause collapses",
		`(semantic:"Berge") AND mediatype:image`,
		`Berge`,
		[]string{"mediatype:image"},
	),
	Entry("semantic inside a quoted value stays a literal (web name wrapping)",
		`(name:"*semantic:konzert*" OR content:"semantic:konzert")`,
		``,
		[]string{"(", `[name:*semantic:konzert* OR content:semantic:konzert]`, ")"},
	),
	Entry("semantic value containing a colon",
		`semantic:"Kirche: innen" AND Tags:foo`,
		`Kirche: innen`,
		[]string{"Tags:foo"},
	),
)
